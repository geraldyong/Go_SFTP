package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	vault "github.com/hashicorp/vault/api"
	"golang.org/x/crypto/ssh"
)

func makePublicKeyAuthCallback(cfg config, vc *vault.Client, cache *userCache) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
	return func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		user := c.User()
		remote := c.RemoteAddr().String()

		ctx, cancel := context.WithTimeout(context.Background(), cfg.VaultTimeout)
		defer cancel()

		ur, err := cache.getOrLoad(ctx, vc, cfg.VaultUsersPrefix, user, cfg.UserCacheTTL)
		if err != nil {
			audit(user, remote, "auth_fail_user_load", "", "", 0, err)
			return nil, fmt.Errorf("permission denied")
		}
		if ur.Disabled {
			audit(user, remote, "auth_fail_disabled", "", "", 0, fmt.Errorf("disabled"))
			return nil, fmt.Errorf("permission denied")
		}

		ok := isKeyAllowed(key, ur.PublicKeys)
		if !ok {
			audit(user, remote, "auth_fail_key", "", "", 0, fmt.Errorf("key not allowed"))
			return nil, fmt.Errorf("permission denied")
		}

		// Embed a hint in permissions if you want; not strictly needed.
		perms := &ssh.Permissions{
			Extensions: map[string]string{
				"authed": "true",
			},
		}

		audit(user, remote, "auth_ok", "", "", 0, nil)
		return perms, nil
	}
}

func makePublicKeyAuthCallbackWithMetrics(cfg config, vc *vault.Client, cache *userCache) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
    inner := makePublicKeyAuthCallback(cfg, vc, cache)

    return func(c ssh.ConnMetadata, k ssh.PublicKey) (*ssh.Permissions, error) {
        start := time.Now()
        perm, err := inner(c, k)

        user := c.User()
        result := AuthOK
        if err != nil {
            // Map your existing error behavior to labels.
            // If your inner() returns specific sentinel errors, map them here.
            // Otherwise, keep it simple:
            result = AuthFailKey
        }

        ObserveAuth(user, result, time.Since(start))
        return perm, err
    }
}

func isKeyAllowed(presented ssh.PublicKey, allowed []string) bool {
	pb := presented.Marshal()

	for _, s := range allowed {
		parsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(s))
		if err != nil {
			continue
		}
		ab := parsed.Marshal()
		if len(ab) != len(pb) {
			continue
		}
		if subtle.ConstantTimeCompare(ab, pb) == 1 {
			return true
		}
	}
	return false
}

// (Optional helper if you later add time-based disabling, etc.)
func within(_ time.Time, _ time.Time) bool { return true }

