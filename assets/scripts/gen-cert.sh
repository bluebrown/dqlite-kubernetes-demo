#!/usr/bin/env sh

: "${CERTS_DIR:=./assets/certs}"

mkdir -p "$CERTS_DIR"

exec openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
    -keyout "$CERTS_DIR/tls.key" -out "$CERTS_DIR/tls.crt" \
    -subj "/CN=dqlite-cluster" -addext "subjectAltName=DNS:dqlite-cluster"
