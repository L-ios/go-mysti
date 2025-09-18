#!/bin/bash


# 清理临时文件
rm -f "$CERT_DIR/redis.csr" "$CERT_DIR/ca.srl"

# 启动 Redis 服务器，启用 TLS
exec redis-server --appendonly yes --tls-port 6379 --port 0 \
  --tls-cert-file "$CERT_DIR/redis.crt" \
  --tls-key-file "$CERT_DIR/redis.key" \
  --tls-ca-cert-file "$CERT_DIR/ca.crt"