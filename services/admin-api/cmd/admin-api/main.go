package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	hv "github.com/hashicorp/vault/api"
)

type User struct {
	Username   string   `json:"username"`
	Disabled   bool     `json:"disabled"`
	PublicKeys []string `json:"publicKeys"`
	RootSubdir string   `json:"rootSubdir"`
	UpdatedAt  string   `json:"updatedAt,omitempty"`
}

// PartialUser is used by PATCH endpoints.
// Fields are pointers so we can distinguish "unset" vs "set to zero value".
type PartialUser struct {
	Disabled   *bool     `json:"disabled,omitempty"`
	PublicKeys *[]string `json:"publicKeys,omitempty"`
	RootSubdir *string   `json:"rootSubdir,omitempty"`
}

type apiError struct {
	OK    bool `json:"ok"`
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
}

type apiOK struct {
	OK   bool `json:"ok"`
	Data any  `json:"data,omitempty"`
}

var (
	// Conservative username policy for filesystem + Vault path safety.
	// Adjust as needed, but keep it restrictive.
	usernameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,31}$`)
)

func env(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func main() {
	listen := env("LISTEN_ADDR", "0.0.0.0:8080")
	vaultAddr := env("VAULT_ADDR", "")
	usersPrefix := env("VAULT_USERS_PREFIX", "kv/sftp/users")
	token := strings.TrimSpace(os.Getenv("VAULT_TOKEN"))

	if vaultAddr == "" || token == "" {
		log.Fatal("VAULT_ADDR and VAULT_TOKEN must be set for admin-api")
	}

	cfg := hv.DefaultConfig()
	cfg.Address = vaultAddr
	c, err := hv.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	c.SetToken(token)

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Users collection
		r.Get("/users", func(w http.ResponseWriter, req *http.Request) {
			q := strings.TrimSpace(req.URL.Query().Get("q"))
			disabledQ := strings.TrimSpace(req.URL.Query().Get("disabled"))
			limit := parseLimit(req.URL.Query().Get("limit"), 200)

			var filterDisabled *bool
			if disabledQ != "" {
				b, err := strconv.ParseBool(disabledQ)
				if err != nil {
					writeAPIError(w, http.StatusBadRequest, "INVALID_QUERY", "invalid 'disabled' query param", map[string]any{"disabled": disabledQ})
					return
				}
				filterDisabled = &b
			}

			users, err := listUsers(req.Context(), c, usersPrefix, q, filterDisabled, limit)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "VAULT_ERROR", err.Error(), nil)
				return
			}
			writeJSON(w, http.StatusOK, apiOK{OK: true, Data: users})
		})

		r.Post("/users", func(w http.ResponseWriter, req *http.Request) {
			var u User
			if err := json.NewDecoder(req.Body).Decode(&u); err != nil {
				writeAPIError(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil)
				return
			}
			if err := normalizeAndValidateUser(&u, "", true); err != nil {
				writeAPIError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
				return
			}
			if err := writeUserKV2(req.Context(), c, usersPrefix, u); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "VAULT_ERROR", err.Error(), nil)
				return
			}
			writeJSON(w, http.StatusOK, apiOK{OK: true})
		})

		// User item
		r.Route("/users/{username}", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				username := chi.URLParam(req, "username")
				if !usernameRe.MatchString(username) {
					writeAPIError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid username", map[string]any{"username": username})
					return
				}
				u, err := readUserKV2(req.Context(), c, usersPrefix, username)
				if errors.Is(err, errNotFound) {
					writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "user not found", map[string]any{"username": username})
					return
				}
				if err != nil {
					writeAPIError(w, http.StatusInternalServerError, "VAULT_ERROR", err.Error(), nil)
					return
				}
				writeJSON(w, http.StatusOK, apiOK{OK: true, Data: u})
			})

			// PUT = replace
			r.Put("/", func(w http.ResponseWriter, req *http.Request) {
				username := chi.URLParam(req, "username")
				var u User
				if err := json.NewDecoder(req.Body).Decode(&u); err != nil {
					writeAPIError(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil)
					return
				}
				if err := normalizeAndValidateUser(&u, username, true); err != nil {
					writeAPIError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
					return
				}
				if err := writeUserKV2(req.Context(), c, usersPrefix, u); err != nil {
					writeAPIError(w, http.StatusInternalServerError, "VAULT_ERROR", err.Error(), nil)
					return
				}
				writeJSON(w, http.StatusOK, apiOK{OK: true})
			})

			// PATCH = partial update
			r.Patch("/", func(w http.ResponseWriter, req *http.Request) {
				username := chi.URLParam(req, "username")
				if !usernameRe.MatchString(username) {
					writeAPIError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid username", map[string]any{"username": username})
					return
				}

				var p PartialUser
				if err := json.NewDecoder(req.Body).Decode(&p); err != nil {
					writeAPIError(w, http.StatusBadRequest, "INVALID_JSON", err.Error(), nil)
					return
				}

				u, err := readUserKV2(req.Context(), c, usersPrefix, username)
				if errors.Is(err, errNotFound) {
					writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "user not found", map[string]any{"username": username})
					return
				}
				if err != nil {
					writeAPIError(w, http.StatusInternalServerError, "VAULT_ERROR", err.Error(), nil)
					return
				}

				if p.Disabled != nil {
					u.Disabled = *p.Disabled
				}
				if p.RootSubdir != nil {
					u.RootSubdir = *p.RootSubdir
				}
				if p.PublicKeys != nil {
					u.PublicKeys = *p.PublicKeys
				}

				if err := normalizeAndValidateUser(&u, username, true); err != nil {
					writeAPIError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error(), nil)
					return
				}
				if err := writeUserKV2(req.Context(), c, usersPrefix, u); err != nil {
					writeAPIError(w, http.StatusInternalServerError, "VAULT_ERROR", err.Error(), nil)
					return
				}
				writeJSON(w, http.StatusOK, apiOK{OK: true})
			})

			r.Delete("/", func(w http.ResponseWriter, req *http.Request) {
				username := chi.URLParam(req, "username")
				if !usernameRe.MatchString(username) {
					writeAPIError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid username", map[string]any{"username": username})
					return
				}
				if err := deleteUserKV2(req.Context(), c, usersPrefix, username); err != nil {
					if errors.Is(err, errNotFound) {
						writeAPIError(w, http.StatusNotFound, "NOT_FOUND", "user not found", map[string]any{"username": username})
						return
					}
					writeAPIError(w, http.StatusInternalServerError, "VAULT_ERROR", err.Error(), nil)
					return
				}
				writeJSON(w, http.StatusOK, apiOK{OK: true})
			})
		})
	})

	log.Printf("admin-api listening on %s", listen)
	srv := &http.Server{
		Addr:              listen,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	e := apiError{OK: false}
	e.Error.Code = code
	e.Error.Message = message
	e.Error.Details = details
	writeJSON(w, status, e)
}

func parseLimit(raw string, def int) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	if n > def {
		return def
	}
	return n
}

func normalizeAndValidateUser(u *User, usernameFromPath string, requireKeys bool) error {
	if usernameFromPath != "" {
		u.Username = usernameFromPath
	}
	u.Username = strings.TrimSpace(u.Username)
	u.RootSubdir = strings.TrimSpace(u.RootSubdir)

	if !usernameRe.MatchString(u.Username) {
		return fmt.Errorf("invalid username")
	}

	if u.RootSubdir == "" {
		u.RootSubdir = u.Username
	}
	// rootSubdir should be a simple folder name (or relative path if you loosen this later).
	if strings.Contains(u.RootSubdir, "..") || strings.HasPrefix(u.RootSubdir, "/") || strings.Contains(u.RootSubdir, "\\") {
		return fmt.Errorf("invalid rootSubdir")
	}

	// Normalize keys
	clean := make([]string, 0, len(u.PublicKeys))
	for _, k := range u.PublicKeys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		clean = append(clean, k)
	}
	u.PublicKeys = clean

	if requireKeys && len(u.PublicKeys) == 0 {
		return fmt.Errorf("publicKeys required")
	}
	// light sanity check (the SFTP server does the real parsing)
	for _, k := range u.PublicKeys {
		parts := strings.Fields(k)
		if len(parts) < 2 {
			return fmt.Errorf("invalid SSH public key format")
		}
	}

	return nil
}

var errNotFound = errors.New("not found")

// kv2Paths derives the KV v2 data and metadata paths from a prefix like "kv/sftp/users".
func kv2Paths(usersPrefix, username string) (dataPath string, metadataPath string, metadataBase string, err error) {
	base := strings.Trim(usersPrefix, "/")
	parts := strings.SplitN(base, "/", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid VAULT_USERS_PREFIX: %q", usersPrefix)
	}
	mount := parts[0]
	sub := strings.Trim(parts[1], "/")

	metadataBase = mount + "/metadata/" + sub
	if username == "" {
		return mount + "/data/" + sub, "", metadataBase, nil
	}
	return mount + "/data/" + sub + "/" + username, mount + "/metadata/" + sub + "/" + username, metadataBase, nil
}

func writeUserKV2(ctx context.Context, c *hv.Client, usersPrefix string, u User) error {
	dataPath, _, _, err := kv2Paths(usersPrefix, u.Username)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"data": map[string]any{
			"username":   u.Username,
			"disabled":   u.Disabled,
			"rootSubdir": u.RootSubdir,
			"publicKeys": u.PublicKeys,
			"updatedAt":  time.Now().UTC().Format(time.RFC3339),
		},
	}
	_, err = c.Logical().WriteWithContext(ctx, dataPath, payload)
	return err
}

func readUserKV2(ctx context.Context, c *hv.Client, usersPrefix, username string) (User, error) {
	dataPath, _, _, err := kv2Paths(usersPrefix, username)
	if err != nil {
		return User{}, err
	}

	sec, err := c.Logical().ReadWithContext(ctx, dataPath)
	if err != nil {
		return User{}, err
	}
	if sec == nil || sec.Data == nil {
		return User{}, errNotFound
	}

	raw, ok := sec.Data["data"]
	if !ok {
		return User{}, errNotFound
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return User{}, fmt.Errorf("unexpected vault payload")
	}

	u := User{Username: username}
	if v, ok := m["disabled"].(bool); ok {
		u.Disabled = v
	}
	if v, ok := m["rootSubdir"].(string); ok {
		u.RootSubdir = v
	}
	if v, ok := m["updatedAt"].(string); ok {
		u.UpdatedAt = v
	}
	// publicKeys may come back as []interface{}
	switch v := m["publicKeys"].(type) {
	case []string:
		u.PublicKeys = v
	case []any:
		keys := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				keys = append(keys, s)
			}
		}
		u.PublicKeys = keys
	}

	return u, nil
}

func deleteUserKV2(ctx context.Context, c *hv.Client, usersPrefix, username string) error {
	_, metadataPath, _, err := kv2Paths(usersPrefix, username)
	if err != nil {
		return err
	}

	// Best-effort check if exists
	_, err = readUserKV2(ctx, c, usersPrefix, username)
	if err != nil {
		return err
	}

	_, err = c.Logical().DeleteWithContext(ctx, metadataPath)
	return err
}

func listUsers(ctx context.Context, c *hv.Client, usersPrefix, q string, filterDisabled *bool, limit int) ([]map[string]any, error) {
	_, _, metadataBase, err := kv2Paths(usersPrefix, "")
	if err != nil {
		return nil, err
	}

	sec, err := c.Logical().ListWithContext(ctx, metadataBase)
	if err != nil {
		return nil, err
	}
	if sec == nil || sec.Data == nil {
		return []map[string]any{}, nil
	}

	rawKeys, ok := sec.Data["keys"]
	if !ok {
		return []map[string]any{}, nil
	}

	var keys []string
	switch v := rawKeys.(type) {
	case []string:
		keys = v
	case []any:
		for _, x := range v {
			if s, ok := x.(string); ok {
				keys = append(keys, s)
			}
		}
	default:
		return nil, fmt.Errorf("unexpected list response")
	}

	out := make([]map[string]any, 0, len(keys))
	q = strings.ToLower(strings.TrimSpace(q))

	for _, username := range keys {
		if username == "" {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(username), q) {
			continue
		}

		u, err := readUserKV2(ctx, c, usersPrefix, username)
		if errors.Is(err, errNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if filterDisabled != nil && u.Disabled != *filterDisabled {
			continue
		}

		out = append(out, map[string]any{
			"username":   u.Username,
			"disabled":   u.Disabled,
			"rootSubdir": u.RootSubdir,
			"keyCount":   len(u.PublicKeys),
			"updatedAt":  u.UpdatedAt,
		})

		if len(out) >= limit {
			break
		}
	}

	return out, nil
}
