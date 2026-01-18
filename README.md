# SFTP Service Starter Kit (Go + Vault + Web UI + Helm)

This repo provides a secure, Kubernetes-friendly SFTP service with:
- **Go SFTP server** (`services/sftp-server`) supporting multiple users and per-user jailed roots.
- **Admin API** (`services/admin-api`) to manage users and authorized keys stored in **HashiCorp Vault** KV.
- **Web UI** (`services/web-ui`) minimal Next.js admin portal.
- **Helm chart** (`helm/sftp-service`) to deploy into Kubernetes (multiple SFTP replicas).

## Key design points
- **Public key authentication** (recommended default).
- Each user is jailed to: `/data/<username>` (or `rootSubdir` if set in Vault record).
- User cannot traverse outside their jail; symlink operations are denied.
- Kubernetes SFTP replicas: Service load-balances **new TCP connections** across pods.

## Vault data model
Vault KV v2 is preferred. Default prefix: `kv/sftp/users`.

User record at: `<prefix>/<username>` (e.g. `kv/sftp/users/alice`):
```json
{
  "username": "alice",
  "disabled": false,
  "publicKeys": ["ssh-ed25519 AAAA... alice@laptop"],
  "rootSubdir": "alice"
}
```

## Local dev quick start
1) Run Vault dev:
```bash
vault server -dev -dev-root-token-id=root
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=root
```

2) Start admin-api:
```bash
cd services/admin-api
go run ./cmd/admin-api
```

3) Create user in Vault:
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","publicKeys":["ssh-ed25519 AAAA... alice@laptop"]}'
```

4) Generate a host key for SFTP server:
```bash
ssh-keygen -t ed25519 -f /tmp/ssh_host_ed25519_key -N ""
```

5) Start sftp-server:
```bash
cd ../sftp-server
mkdir -p /tmp/sftp-data
LISTEN_ADDR=0.0.0.0:2022 \
DATA_ROOT=/tmp/sftp-data \
HOST_KEY_PATH=/tmp/ssh_host_ed25519_key \
VAULT_ADDR=$VAULT_ADDR VAULT_TOKEN=$VAULT_TOKEN \
VAULT_USERS_PREFIX=kv/sftp/users \
go run ./cmd/sftp-server
```

6) Connect:
```bash
sftp -P 2022 alice@127.0.0.1
```

## Kubernetes deployment
Build and push images, then:
```bash
helm upgrade --install sftp ./helm/sftp-service \
  --set sftp.image.repository=yourrepo/sftp-server --set sftp.image.tag=0.1.0 \
  --set adminApi.image.repository=yourrepo/admin-api --set adminApi.image.tag=0.1.0 \
  --set webUi.image.repository=yourrepo/web-ui --set webUi.image.tag=0.1.0 \
  --set adminApi.vault.tokenSecretName=vault-admin-token \
  --set sftp.vault.addr=http://vault.vault.svc:8200 \
  --set sftp.vault.tokenSecretName=vault-sftp-token \
  --set sftp.hostKeySecret.name=sftp-hostkey
```

### Host key secret
Create a Secret containing the **OpenSSH private key**:
```bash
kubectl create secret generic sftp-hostkey \
  --from-file=ssh_host_ed25519_key=/path/to/ssh_host_ed25519_key
```

## Security hardening checklist
- Use Vault **Kubernetes auth + Vault Agent** in production (instead of long-lived tokens).
- Apply **NetworkPolicy** restricting access to TCP/2022 and admin-api.
- Use PSA restricted / Pod Security Standards.
- Enable Vault audit logging and least-privilege policies.
