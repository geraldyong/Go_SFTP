#!/bin/bash

# Create directories if not already created.
[[ -d dev ]] || mkdir dev
[[ -d services/web-ui/public ]] || mkdir services/web-ui/public

# Restart docker services.
docker compose down
docker compose build
docker compose up -d
docker compose ps
