#!/bin/bash
# 修复 wallet-service chart 里的 service name
# 用法: bash fix_wallet_chart.sh ~/web3-blitz

ROOT=${1:-~/web3-blitz}
SVC_DIR="$ROOT/deployments/web3-blitz/wallet-service"

# 修 values.yaml
sed -i '' \
  's|postgres://blitz:blitz@postgres:|postgres://blitz:blitz@web3-blitz-postgres:|g' \
  "$SVC_DIR/values.yaml"
sed -i '' \
  's|postgres://user:pass@postgres:|postgres://blitz:blitz@web3-blitz-postgres:|g' \
  "$SVC_DIR/values.yaml"
sed -i '' \
  's|etcd:2379|web3-blitz-etcd:2379|g' \
  "$SVC_DIR/values.yaml"

# 修 deployment.yaml initContainers
sed -i '' \
  's|nc -z postgres 5432|nc -z web3-blitz-postgres 5432|g' \
  "$SVC_DIR/templates/deployment.yaml"
sed -i '' \
  's|nc -z etcd 2379|nc -z web3-blitz-etcd 2379|g' \
  "$SVC_DIR/templates/deployment.yaml"

echo "✅ 修复完成，验证结果："
echo ""
echo "--- values.yaml env ---"
grep -A2 "DATABASE_URL\|ETCD_ENDPOINTS" "$SVC_DIR/values.yaml"
echo ""
echo "--- deployment.yaml initContainers ---"
grep "nc -z" "$SVC_DIR/templates/deployment.yaml"
