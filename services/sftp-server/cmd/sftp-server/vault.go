package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
// - "kv/sftp/users"      (KV v2 mounted at "kv")
// - "secret/sftp/users"  (KV v2 mounted at "secret")
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
			return ur, fmt.Errorf("unexpected vault kv response: missing data field")
		}

		rawMap, ok := raw.(map[string]interface{})
		if !ok {
			// Fallback via JSON in case Vault returns map[string]any aliasing weirdly
			b, err := json.Marshal(raw)
			if err != nil {
				return ur, fmt.Errorf("marshal raw vault user failed: %w", err)
			}
			if err := json.Unmarshal(b, &rawMap); err != nil {
				return ur, fmt.Errorf("decode raw vault user failed: %w", err)
			}
		}

		parsed, err := parseUserRecord(rawMap, username)
		if err != nil {
			return ur, err
		}
		return parsed, nil
	}
}

func parseUserRecord(m map[string]interface{}, fallbackUsername string) (userRecord, error) {
	var ur userRecord

	// username
	if v, ok := m["username"]; ok {
		s, err := asString(v)
		if err != nil {
			return ur, fmt.Errorf("invalid username: %w", err)
		}
		ur.Username = s
	}
	if ur.Username == "" {
		ur.Username = fallbackUsername
	}

	// disabled
	if v, ok := m["disabled"]; ok {
		b, err := asBool(v)
		if err != nil {
			return ur, fmt.Errorf("invalid disabled: %w", err)
		}
		ur.Disabled = b
	}

	// rootSubdir
	if v, ok := m["rootSubdir"]; ok {
		s, err := asString(v)
		if err != nil {
			return ur, fmt.Errorf("invalid rootSubdir: %w", err)
		}
		ur.RootSubdir = s
	}
	if ur.RootSubdir == "" {
		ur.RootSubdir = ur.Username
	}

	// publicKeys
	if v, ok := m["publicKeys"]; ok {
		keys, err := asStringSlice(v)
		if err != nil {
			return ur, fmt.Errorf("invalid publicKeys: %w", err)
		}
		ur.PublicKeys = keys
	}
	if len(ur.PublicKeys) == 0 {
		return ur, fmt.Errorf("user not found")
	}

	// quotaBytes
	if v, ok := m["quotaBytes"]; ok {
		n, err := asInt64(v)
		if err != nil {
			return ur, fmt.Errorf("invalid quotaBytes: %w", err)
		}
		ur.QuotaBytes = n
	}

	// quotaFiles
	if v, ok := m["quotaFiles"]; ok {
		n, err := asInt64(v)
		if err != nil {
			return ur, fmt.Errorf("invalid quotaFiles: %w", err)
		}
		ur.QuotaFiles = n
	}

	return ur, nil
}

func asString(v interface{}) (string, error) {
	switch x := v.(type) {
	case string:
		return x, nil
	case []byte:
		return string(x), nil
	case json.Number:
		return x.String(), nil
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(x), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("expected string, got %T", v)
	}
}

func asBool(v interface{}) (bool, error) {
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(x))
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return false, err
		}
		return i != 0, nil
	case float64:
		return x != 0, nil
	case int:
		return x != 0, nil
	case int64:
		return x != 0, nil
	case nil:
		return false, nil
	default:
		return false, fmt.Errorf("expected bool, got %T", v)
	}
}

func asInt64(v interface{}) (int64, error) {
	switch x := v.(type) {
	case int64:
		return x, nil
	case int:
		return int64(x), nil
	case float64:
		return int64(x), nil
	case json.Number:
		return x.Int64()
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, nil
		}
		return strconv.ParseInt(s, 10, 64)
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("expected integer, got %T", v)
	}
}

func asStringSlice(v interface{}) ([]string, error) {
	switch x := v.(type) {
	case []string:
		return x, nil

	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s, err := asString(item)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out, nil

	case string:
		if strings.TrimSpace(x) == "" {
			return nil, nil
		}
		return []string{x}, nil

	case nil:
		return nil, nil

	default:
		return nil, fmt.Errorf("expected []string, []interface{}, or string, got %T", v)
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
