# syntax=docker/dockerfile:1

#FROM golang:1.24 AS build
FROM golang:1.25.5-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -ldflags="-s -w" -o /out/admin-api ./cmd/admin-api

FROM gcr.io/distroless/static:nonroot
WORKDIR /
USER nonroot:nonroot
COPY --from=build /out/admin-api /admin-api
EXPOSE 8080
ENTRYPOINT ["/admin-api"]
