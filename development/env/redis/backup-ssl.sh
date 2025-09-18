#!/usr/bin/env bash


function generate_ssl() {
  # 证书存储目录
  readonly CERT_DIR="$1"
  mkdir -p "$CERT_DIR"

  # 生成 CA 密钥和证书
  openssl genrsa -out "$CERT_DIR/ca.key" 4096
  openssl req -x509 -new -nodes -key "$CERT_DIR/ca.key" -sha256 -days 3650 -out "$CERT_DIR/ca.crt" -subj "/CN=Redis CA"

  # 生成 Redis 服务器密钥和证书
  openssl genrsa -out "$CERT_DIR/redis.key" 2048
  openssl req -new -key "$CERT_DIR/redis.key" -out "$CERT_DIR/redis.csr" -subj "/CN=redis"
  openssl x509 -req -in "$CERT_DIR/redis.csr" -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial -out "$CERT_DIR/redis.crt" -days 365 -sha256
}

generate_ssl redis-ssl