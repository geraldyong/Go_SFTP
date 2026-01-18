package main

import (
  "context"
  "encoding/json"
  "fmt"
  "log"
  "net/http"
  "os"
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
}

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
    r.Post("/users", func(w http.ResponseWriter, req *http.Request) {
      var u User
      if err := json.NewDecoder(req.Body).Decode(&u); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
      }
      if u.Username == "" || len(u.PublicKeys) == 0 {
        http.Error(w, "username and publicKeys required", http.StatusBadRequest)
        return
      }
      if err := writeUserKV2(req.Context(), c, usersPrefix, u); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
      }
      w.Header().Set("Content-Type", "application/json")
      _ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
    })
  })

  log.Printf("admin-api listening on %s", listen)
  srv := &http.Server{
    Addr:              listen,
    Handler:           r,
    ReadHeaderTimeout: 5 * time.Second,
  }
  log.Fatal(srv.ListenAndServe())
}

func writeUserKV2(ctx context.Context, c *hv.Client, usersPrefix string, u User) error {
  path := strings.TrimSuffix(usersPrefix, "/") + "/" + u.Username
  parts := strings.SplitN(path, "/", 2)
  if len(parts) != 2 {
    return fmt.Errorf("invalid usersPrefix")
  }
  v2path := parts[0] + "/data/" + parts[1]

  payload := map[string]any{
    "data": map[string]any{
      "username":   u.Username,
      "disabled":   u.Disabled,
      "rootSubdir": u.RootSubdir,
      "publicKeys": u.PublicKeys,
      "updatedAt":  time.Now().UTC().Format(time.RFC3339),
    },
  }
  _, err := c.Logical().WriteWithContext(ctx, v2path, payload)
  return err
}
