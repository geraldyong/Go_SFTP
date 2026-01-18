package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
)

type userRecord struct {
	Username   string   `json:"username"`
	Disabled   bool     `json:"disabled"`
	RootSubdir string   `json:"rootSubdir"`
	PublicKeys []string `json:"publicKeys"`

	QuotaBytes int64 `json:"quotaBytes"`
	QuotaFiles int64 `json:"quotaFiles"`
}

func newVaultClient(cfg config) (*vault.Client, error) {
	vcfg := vault.DefaultConfig()
	vcfg.Address = cfg.VaultAddr

	c, err := vault.NewClient(vcfg)
	if err != nil {
		return nil, err
	}
	c.SetToken(cfg.VaultToken)
	return c, nil
}

// Supports VAULT_USERS_PREFIX like:
// - "kv/sftp/users"  (KV v2 mounted at "kv")
// - "secret/sftp/users" (KV v2 mounted at "secret")
// Internally reads: "<mount>/data/<path>/<username>"
func loadUserFromVault(ctx context.Context, vc *vault.Client, usersPrefix, username string) (userRecord, error) {
	var ur userRecord

	path := kvV2DataPath(usersPrefix, username)

	type result struct {
		sec *vault.Secret
		err error
	}
	ch := make(chan result, 1)
	go func() {
		sec, err := vc.Logical().Read(path)
		ch <- result{sec: sec, err: err}
	}()

	select {
	case <-ctx.Done():
		return ur, ctx.Err()
	case res := <-ch:
		if res.err != nil {
			return ur, res.err
		}
		if res.sec == nil || res.sec.Data == nil {
			return ur, fmt.Errorf("user not found")
		}

		// KV v2 wraps actual fields under "data"
		raw, ok := res.sec.Data["data"]
		if !ok {
			return ur, fmt.Errorf("unexpected vault kv response")
		}

		// Marshal/unmarshal to struct for convenience
		b, _ := json.Marshal(raw)
		if err := json.Unmarshal(b, &ur); err != nil {
			return ur, err
		}

		// Some minimal normalization
		if ur.Username == "" {
			ur.Username = username
		}
		return ur, nil
	}
}

func kvV2DataPath(usersPrefix, username string) string {
	p := strings.Trim(usersPrefix, "/")

	// If caller already included "/data/" assume it's correct.
	if strings.Contains(p, "/data/") {
		return fmt.Sprintf("%s/%s", p, username)
	}

	// Insert "data" after the mount segment.
	// e.g. "kv/sftp/users" -> "kv/data/sftp/users/<username>"
	parts := strings.SplitN(p, "/", 2)
	mount := parts[0]
	rest := ""
	if len(parts) == 2 {
		rest = parts[1]
	}
	if rest == "" {
		return fmt.Sprintf("%s/data/%s", mount, username)
	}
	return fmt.Sprintf("%s/data/%s/%s", mount, rest, username)
}

// --- small in-memory cache to reduce Vault calls ---
type cachedUser struct {
	u       userRecord
	expires time.Time
}

type userCache struct {
	ttl   time.Duration
	store map[string]cachedUser
	mu    chan struct{}
}

func newUserCache(ttl time.Duration) *userCache {
	return &userCache{
		ttl:   ttl,
		store: map[string]cachedUser{},
		mu:    make(chan struct{}, 1),
	}
}

func (c *userCache) lock()   { c.mu <- struct{}{} }
func (c *userCache) unlock() { <-c.mu }

func (c *userCache) getOrLoad(ctx context.Context, vc *vault.Client, prefix, username string, ttl time.Duration) (userRecord, error) {
	now := time.Now()

	c.lock()
	if cu, ok := c.store[username]; ok && now.Before(cu.expires) {
		u := cu.u
		c.unlock()
		return u, nil
	}
	c.unlock()

	u, err := loadUserFromVault(ctx, vc, prefix, username)
	if err != nil {
		return userRecord{}, err
	}

	c.lock()
	c.store[username] = cachedUser{u: u, expires: now.Add(ttl)}
	c.unlock()

	return u, nil
}

