#!/bin/bash
set -e

NAMESPACE=${KUBE_NAMESPACE:-web3-blitz}

# 从 .env 读取（本地开发用）或从环境变量读取（CI/CD 用）
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

kubectl create secret generic wallet-service-secret \
  -n "$NAMESPACE" \
  --from-literal=DATABASE_URL="postgres://blitz:blitz@web3-blitz-postgres:5432/blitz?sslmode=disable&search_path=public" \
  --from-literal=WALLET_HD_SEED="$WALLET_HD_SEED" \
  --from-literal=SMTP_PASS="$SMTP_PASS" \
  --from-literal=JWT_SECRET="${JWT_SECRET:-$(openssl rand -hex 32)}" \
  --save-config \
  --dry-run=client -o yaml | kubectl apply -f -

echo "✅ wallet-service-secret 已创建/更新"