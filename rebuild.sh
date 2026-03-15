#!/bin/bash

# Create directories if not already created.
[[ -d services/web-ui/public ]] || mkdir services/web-ui/public

# Create temporary key if needed.
./create_key.sh

# Restart docker services.
docker compose down
docker compose build
docker compose up -d
docker compose ps
