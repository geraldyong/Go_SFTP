package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type config struct {
	ListenAddr       string
	DataRoot         string
	HostKeyPath      string
	VaultAddr        string
	VaultToken       string
	VaultUsersPrefix string

	// Defaults if user record omits quota fields
	DefaultQuotaBytes int64
	DefaultQuotaFiles int64

	// Operational
	VaultTimeout  time.Duration
	UserCacheTTL  time.Duration
	DisableCache  bool
	LogAuditJSON  bool
}

func loadConfigFromEnv() (config, error) {
	var c config

	c.ListenAddr = getenv("LISTEN_ADDR", "0.0.0.0:2022")
	c.DataRoot = getenv("DATA_ROOT", "/data")
	c.HostKeyPath = getenv("HOST_KEY_PATH", "/keys/ssh_host_ed25519_key")

	c.VaultAddr = getenv("VAULT_ADDR", "")
	c.VaultToken = getenv("VAULT_TOKEN", "")
	c.VaultUsersPrefix = getenv("VAULT_USERS_PREFIX", "kv/sftp/users")

	c.DefaultQuotaBytes = parseEnvInt64("DEFAULT_QUOTA_BYTES", 0) // 0 = unlimited by default
	c.DefaultQuotaFiles = parseEnvInt64("DEFAULT_QUOTA_FILES", 0) // 0 = unlimited by default

	c.VaultTimeout = parseEnvDuration("VAULT_TIMEOUT", 5*time.Second)
	c.UserCacheTTL = parseEnvDuration("USER_CACHE_TTL", 30*time.Second)

	c.DisableCache = parseEnvBool("DISABLE_USER_CACHE", false)
	c.LogAuditJSON = true // always JSON stdout in this starter kit

	if c.VaultAddr == "" {
		return c, fmt.Errorf("VAULT_ADDR is required")
	}
	if c.VaultToken == "" {
		return c, fmt.Errorf("VAULT_TOKEN is required (dev only; use K8s auth in prod)")
	}
	return c, nil
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func parseEnvInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func parseEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func parseEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

