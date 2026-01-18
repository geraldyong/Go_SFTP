package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	vault "github.com/hashicorp/vault/api"
	"golang.org/x/crypto/ssh"
)

func main() {
	log.SetFlags(0)

	cfg, err := loadConfigFromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	StartMetricsServer(ctx, DefaultMetricsConfigFromEnv())

	hostKey, err := readHostKey(cfg.HostKeyPath)
	if err != nil {
		log.Fatalf("read host key %q failed: %v", cfg.HostKeyPath, err)
	}

	vc, err := newVaultClient(cfg)
	if err != nil {
		log.Fatalf("vault client error: %v", err)
	}

	cache := newUserCache(cfg.UserCacheTTL)

	sshCfg := &ssh.ServerConfig{
		ServerVersion: "SSH-2.0-sftp-service",
		// Public key auth only
		PublicKeyCallback: makePublicKeyAuthCallbackWithMetrics(cfg, vc, cache),
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Explicitly disable password auth
			audit(c.User(), c.RemoteAddr().String(), "auth_password_rejected", "", "", 0, fmt.Errorf("password auth disabled"))
			return nil, fmt.Errorf("password auth disabled")
		},
	}
	sshCfg.AddHostKey(hostKey)

	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("listen %s failed: %v", cfg.ListenAddr, err)
	}
	defer ln.Close()

	log.Printf("sftp-server listening on %s", cfg.ListenAddr)

	// Graceful shutdown
	stop := make(chan os.Signal, 2)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					errCh <- nil
					return
				}
				errCh <- err
				return
			}
			
			IncSessionActive(1)
			IncSessionTotal("started")

			go func() {
    				defer IncSessionActive(-1)
    				handleConn(cfg, vc, cache, sshCfg, conn)
			}()
		}
	}()

	select {
	case <-stop:
		cancel()
		_ = ln.Close()
		<-errCh
	case err := <-errCh:
		if err != nil {
			log.Printf("accept loop error: %v", err)
		}
	}
}

func handleConn(cfg config, vc *vault.Client, cache *userCache, sshCfg *ssh.ServerConfig, raw net.Conn) {
	defer raw.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(raw, sshCfg)
	if err != nil {
		// Most auth failures surface here
		return
	}
	defer sshConn.Close()

	user := sshConn.User()
	remote := sshConn.RemoteAddr().String()

	audit(user, remote, "session_start", "", "", 0, nil)

	// Discard global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "only session channels supported")
			continue
		}

		ch, inReqs, err := newCh.Accept()
		if err != nil {
			continue
		}

		go func() {
			defer ch.Close()

			for req := range inReqs {
				switch req.Type {
				case "subsystem":
					// payload contains subsystem name
					subsystem := parseSubsystem(req.Payload)
					if subsystem != "sftp" {
						_ = req.Reply(false, nil)
						audit(user, remote, "subsystem_rejected", subsystem, "", 0, fmt.Errorf("unsupported subsystem"))
						continue
					}
					_ = req.Reply(true, nil)

					// Load user again to get quotas & rootSubdir (cached)
					ctx, cancel := context.WithTimeout(context.Background(), cfg.VaultTimeout)
					ur, err := cache.getOrLoad(ctx, vc, cfg.VaultUsersPrefix, user, cfg.UserCacheTTL)
					cancel()
					if err != nil {
						audit(user, remote, "user_load_failed", "", "", 0, err)
						return
					}

					root := userRootPath(cfg.DataRoot, ur.RootSubdir, user)

					// Ensure user root exists (will fail if /data not writable)
					if err := os.MkdirAll(root, 0o750); err != nil {
						audit(user, remote, "user_root_mkdir_failed", root, "", 0, err)
						return
					}

					qb := ur.QuotaBytes
					if qb <= 0 {
						qb = cfg.DefaultQuotaBytes
					}
					qf := ur.QuotaFiles
					if qf <= 0 {
						qf = cfg.DefaultQuotaFiles
					}

					fs := jailedFS{
						root:       root,
						user:       user,
						remote:     remote,
						quotaBytes: qb,
						quotaFiles: qf,
					}

					// Serve SFTP on this channel
					serveSFTP(ch, fs)
					return

				default:
					// deny all other request types (no PTY, no shell)
					_ = req.Reply(false, nil)
				}
			}
		}()
	}

	audit(user, remote, "session_end", "", "", 0, nil)
}

func readHostKey(path string) (ssh.Signer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(b)
}

func parseSubsystem(payload []byte) string {
	// SSH subsystem request payload is a string (RFC 4254)
	// Format: uint32 len + bytes
	if len(payload) < 4 {
		return ""
	}
	n := int(payload[0])<<24 | int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3])
	if n < 0 || 4+n > len(payload) {
		return ""
	}
	return string(payload[4 : 4+n])
}

func userRootPath(dataRoot, rootSubdir, username string) string {
	if rootSubdir != "" {
		return joinClean(dataRoot, rootSubdir)
	}
	return joinClean(dataRoot, username)
}

func joinClean(base, sub string) string {
	// base is absolute mount; sub is tenant folder
	// keep it simple: no traversal
	return fmt.Sprintf("%s/%s", trimRightSlash(base), trimSlashes(sub))
}

func trimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func trimSlashes(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

