# Go_SFTP (Docker Compose)

A small SFTP service you can run on a **single machine** using **Docker Compose**.

It starts these containers:
- **vault**: HashiCorp Vault (dev mode, for local/demo)
- **admin-api**: HTTP API to manage SFTP users/keys stored in Vault
- **web-ui**: minimal web admin portal (talks to admin-api)
- **sftp-server**: Go SFTP server (public-key auth)

> The included `docker-compose.yml` uses **Vault dev mode** with a fixed token (`root`).
> This is convenient for local use, but **not production-safe**.

---

## Prerequisites
- Docker + Docker Compose
- or Kubernetes (minikube)
---

## Quick start

For docker:
```
./rebuild.sh
```

For kubernetes:
```
./rebuild_k8s.sh
```

Connect via SFTP (host port **2022**):
```bash
sftp -P 2022 -i dev/alice alice@127.0.0.1
```
Use the matching private key from `./dev/` (for example: `./dev/alice`).

---

## Web UI and Admin API

- **Web UI:** http://localhost:3000
- **Admin API:** http://localhost:8080

Example: create/update a user via API:
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{
    "username": "bob",
    "disabled": false,
    "rootSubdir": "bob",
    "publicKeys": ["ssh-ed25519 AAAA... bob@laptop"]
  }'
```

---

## Data persistence

Docker volumes are used:
- `sftp-data`  -> user home directories under `/data/<username>` in the container
- `sftp-keys`  -> SFTP **host key** (generated once and reused)

To reset everything (including users and files):

For docker:
```bash
docker compose down -v
```

For Kubernetes:
kubectl delete namespace sftp

---

## Configuration (compose defaults)

Useful ports:
- `2022/tcp`  SFTP
- `9090/tcp`  SFTP server metrics
- `3000/tcp`  Web UI
- `8080/tcp`  Admin API
- `8200/tcp`  Vault (dev)

Most runtime settings are in `docker-compose.yml` under each service’s `environment:` section.
