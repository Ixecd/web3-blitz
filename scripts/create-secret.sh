#!/bin/bash
# 创建/更新 blitz 的 K8s Secret
# 幂等：已存在则更新，不存在则创建
# 用法：./scripts/create-secret.sh
#
# 生产环境：修改下方变量为真实值
# 开发/demo：直接运行，自动使用安全随机值

set -e

NAMESPACE=${KUBE_NAMESPACE:-blitz}

# 从 .env 读取（如果存在）
if [ -f .env ]; then
  export $(grep -v '^#' .env | grep -v '^$' | xargs)
fi

kubectl create secret generic blitz-secret \
  -n "$NAMESPACE" \
  --from-literal=DATABASE_URL="${DATABASE_URL:-postgres://blitz:blitz@blitz-postgres:5432/blitz?sslmode=disable}" \
  --from-literal=JWT_SECRET="${JWT_SECRET:-$(openssl rand -hex 32)}" \
  --from-literal=APP_SECRET="${APP_SECRET:-$(openssl rand -hex 16)}" \
  --from-literal=SEED_PASSPHRASE="${SEED_PASSPHRASE:-$(openssl rand -hex 16)}" \
  --save-config \
  --dry-run=client -o yaml | kubectl apply -f -

echo "✅ blitz-secret 已创建/更新（namespace: $NAMESPACE）"
echo "⚠️  生产环境请设置真实的 DATABASE_URL / JWT_SECRET / APP_SECRET / SEED_PASSPHRASE"
echo "⚠️  HD 种子文件还需另外创建: kubectl create secret generic blitz-hd-seed --from-file=hd-seed.enc=configs/secrets/hd-seed.enc"
