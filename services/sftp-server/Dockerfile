# ---------------------
# Build (static binary)
# ---------------------
FROM golang:1.25.5-alpine AS build
WORKDIR /src

# Add CA certs so Go can reach sum.golang.org / GitHub over TLS
RUN apk add --no-cache --no-check-certificate ca-certificates git && update-ca-certificates

# Cache modules
ENV GOSUMDB=off
ENV GOINSECURE=github.com,golang.org,go.googlesource.com
COPY go.mod go.sum ./
RUN go env -w GOPROXY=https://proxy.golang.org,direct
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
  go build -trimpath -ldflags="-s -w" \
  -o /out/sftp-server ./cmd/sftp-server

# -------------------------
# Run (distroless, nonroot)
# -------------------------
FROM gcr.io/distroless/static:nonroot
WORKDIR /
USER nonroot:nonroot

COPY --from=build /out/sftp-server /sftp-server
EXPOSE 2022
ENTRYPOINT ["/sftp-server"]
