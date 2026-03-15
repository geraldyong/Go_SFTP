#!/bin/bash

# Create directories if not already created.
[[ -d dev ]] || mkdir dev
[[ -d services/web-ui/public ]] || mkdir services/web-ui/public

# Build containers.
eval $(minikube docker-env)
docker compose build

# Create temporary key if needed.
./create_key.sh

# Install helm charts.
cd helm
kubectl -n sftp delete secret sftp-hostkey --ignore-not-found
kubectl -n sftp create secret generic sftp-hostkey \
  --from-file=ssh_host_ed25519_key=../dev/sftp_host
helm uninstall sftp -n sftp || true
helm upgrade --install sftp ./sftp-service -n sftp --create-namespace --set-string seed.alicePublicKey="$(cat ../dev/alice.pub)"

# Install the Alice key into Vault.
cd ..
PUBKEY="$(cat dev/alice.pub)"
kubectl -n sftp exec -i deploy/sftp-go-sftp-vault -- sh -lc '
export VAULT_ADDR=http://127.0.0.1:8200 VAULT_TOKEN=root
cat >/tmp/alice.json
wget -qO- \
  --header="X-Vault-Token: $VAULT_TOKEN" \
  --header="Content-Type: application/json" \
  --post-file=/tmp/alice.json \
  "$VAULT_ADDR/v1/secret/data/sftp/users/alice"
' <<EOF
{
  "data": {
    "username": "alice",
    "disabled": false,
    "rootSubdir": "alice",
    "publicKeys": ["$PUBKEY"]
  }
}
EOF
