# Go_SFTP (Docker Compose & Kubernetes)

A small SFTP service implemented in Go that can run either:

-   **Locally via Docker Compose**
-   **On Kubernetes via Helm (tested with Minikube)**

The system integrates with **HashiCorp Vault** to store SFTP user
records and SSH public keys.

------------------------------------------------------------------------

# Architecture

The platform consists of four services:

-   **vault**\
    HashiCorp Vault used as the user/credential store.

-   **admin-api**\
    REST API that reads/writes SFTP user records in Vault.

-   **web-ui**\
    Simple web admin portal that calls the Admin API.

-   **sftp-server**\
    Custom Go SFTP server that authenticates users using SSH public keys
    stored in Vault.

------------------------------------------------------------------------

# Key Concepts

Two different SSH key types are used:

## 1. Server Host Key

Identifies the SFTP server itself.

Generated as:

    dev/sftp_host
    dev/sftp_host.pub

The **private key** is mounted into the SFTP container/pod.

This key is stored as a Kubernetes secret:

    sftp-hostkey

## 2. User Login Keys

Used for authenticating SFTP users.

Example:

    dev/alice
    dev/alice.pub

-   `alice` → client private key
-   `alice.pub` → stored in Vault

------------------------------------------------------------------------

# Prerequisites

-   Docker + Docker Compose
-   or Kubernetes (Minikube recommended)
-   Helm
-   kubectl

------------------------------------------------------------------------

# Key Management

The repository provides a helper script:

    create_key.sh

This generates the required keys:

    dev/sftp_host
    dev/sftp_host.pub
    dev/alice
    dev/alice.pub

Run once when setting up the environment.

------------------------------------------------------------------------

# Running with Docker Compose

Start the full stack locally:

    ./rebuild.sh

This will:

1.  Build the Docker images
2.  Start Vault in dev mode
3.  Start the SFTP server
4.  Start the Admin API
5.  Start the Web UI

Vault dev mode uses the fixed token:

    root

This is **for development only**.

------------------------------------------------------------------------

# Running on Kubernetes (Minikube)

Deploy via Helm:

    ./rebuild_k8s.sh

This script:

1.  Builds images inside the Minikube Docker environment
2.  Creates the SFTP host key secret
3.  Installs the Helm chart
4.  Seeds Vault with the initial user

Manual Helm equivalent:

    helm upgrade --install sftp ./sftp-service   -n sftp   --create-namespace   --set-string seed.alicePublicKey="$(cat dev/alice.pub)"

------------------------------------------------------------------------

# Connecting via SFTP

Forward the service port:

    kubectl -n sftp port-forward svc/sftp-go-sftp 2022:2022

Connect:

    sftp -P 2022 -i dev/alice alice@127.0.0.1

------------------------------------------------------------------------

# Web UI and Admin API

Web UI:

    http://localhost:3000

Admin API:

    http://localhost:8080

Example: create/update a user via API

    curl -X POST http://localhost:8080/api/v1/users   -H 'Content-Type: application/json'   -d '{
        "username": "bob",
        "disabled": false,
        "rootSubdir": "bob",
        "publicKeys": ["ssh-ed25519 AAAA... bob@laptop"]
      }'

------------------------------------------------------------------------

# Metrics

The SFTP server exports Prometheus metrics:

    http://localhost:9090/metrics

Metrics include:

-   authentication attempts
-   file uploads/downloads
-   quota usage
-   connection counts

------------------------------------------------------------------------

# Data Persistence

Docker volumes:

    sftp-data
    sftp-keys

Used for:

-   user home directories
-   SFTP host key persistence

------------------------------------------------------------------------

# Resetting the Environment

Docker:

    docker compose down -v

Kubernetes:

    kubectl delete namespace sftp

------------------------------------------------------------------------

# Ports

  Port   Service
  ------ --------------------
  2022   SFTP
  9090   Prometheus metrics
  3000   Web UI
  8080   Admin API
  8200   Vault (dev)

------------------------------------------------------------------------

# Security Notes

The default configuration uses:

-   Vault dev mode
-   fixed root token
-   local Docker networking

This setup is intended for:

-   development
-   testing
-   demos

For production deployments you should:

-   use a real Vault cluster
-   enable Vault authentication methods
-   remove the root token
-   use TLS everywhere
-   store host keys in a secure secret store

------------------------------------------------------------------------

# Project Structure

    services/
      sftp-server/
      admin-api/
      web-ui/

    helm/
      sftp-service/

    dev/
      ssh keys for testing

------------------------------------------------------------------------

# License

See LICENSE file for project license.

Vault Enterprise licensing (if used) is **not included** in this
repository.
