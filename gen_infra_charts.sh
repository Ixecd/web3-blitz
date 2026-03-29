#!/bin/bash
# 为 web3-blitz 生成 postgres 和 etcd 独立 chart
# 用法: bash gen_infra_charts.sh ~/web3-blitz

ROOT=${1:-~/web3-blitz}
PROJECT=web3-blitz
DEPLOY_DIR="$ROOT/deployments/$PROJECT"

# ── postgres chart ─────────────────────────────────────────────────────────────
PGDIR="$DEPLOY_DIR/web3-blitz-postgres"
mkdir -p "$PGDIR/templates"

cat > "$PGDIR/Chart.yaml" << 'EOF'
apiVersion: v2
name: web3-blitz-postgres
description: PostgreSQL StatefulSet for web3-blitz
type: application
version: 0.1.0
appVersion: "16-alpine"
dependencies: []
EOF

cat > "$PGDIR/values.yaml" << 'EOF'
storage: 1Gi
EOF

cat > "$PGDIR/templates/statefulset.yaml" << 'EOF'
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web3-blitz-postgres
  namespace: {{ .Release.Namespace }}
  labels:
    app: web3-blitz-postgres
spec:
  serviceName: web3-blitz-postgres
  replicas: 1
  selector:
    matchLabels:
      app: web3-blitz-postgres
  template:
    metadata:
      labels:
        app: web3-blitz-postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          ports:
            - containerPort: 5432
          env:
            - name: POSTGRES_USER
              value: blitz
            - name: POSTGRES_PASSWORD
              value: blitz
            - name: POSTGRES_DB
              value: blitz
            - name: PGDATA
              value: /var/lib/postgresql/data/pgdata
          readinessProbe:
            exec:
              command: ["pg_isready", "-U", "blitz", "-d", "blitz"]
            initialDelaySeconds: 5
            periodSeconds: 5
          livenessProbe:
            exec:
              command: ["pg_isready", "-U", "blitz", "-d", "blitz"]
            initialDelaySeconds: 15
            periodSeconds: 10
          volumeMounts:
            - name: postgres-data
              mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
    - metadata:
        name: postgres-data
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: {{ .Values.storage }}
EOF

cat > "$PGDIR/templates/service.yaml" << 'EOF'
apiVersion: v1
kind: Service
metadata:
  name: web3-blitz-postgres
  namespace: {{ .Release.Namespace }}
spec:
  selector:
    app: web3-blitz-postgres
  ports:
    - port: 5432
      targetPort: 5432
EOF

echo "✅ postgres chart: $PGDIR"

# ── etcd chart ─────────────────────────────────────────────────────────────────
ETCDDIR="$DEPLOY_DIR/web3-blitz-etcd"
mkdir -p "$ETCDDIR/templates"

cat > "$ETCDDIR/Chart.yaml" << 'EOF'
apiVersion: v2
name: web3-blitz-etcd
description: etcd Deployment for web3-blitz
type: application
version: 0.1.0
appVersion: "v3.5.14"
dependencies: []
EOF

cat > "$ETCDDIR/templates/deployment.yaml" << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web3-blitz-etcd
  namespace: {{ .Release.Namespace }}
  labels:
    app: web3-blitz-etcd
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web3-blitz-etcd
  template:
    metadata:
      labels:
        app: web3-blitz-etcd
    spec:
      containers:
        - name: etcd
          image: quay.io/coreos/etcd:v3.5.14
          command:
            - etcd
            - --listen-client-urls=http://0.0.0.0:2379
            - --advertise-client-urls=http://web3-blitz-etcd:2379
            - --listen-peer-urls=http://0.0.0.0:2380
            - --initial-advertise-peer-urls=http://0.0.0.0:2380
            - --initial-cluster=default=http://0.0.0.0:2380
            - --data-dir=/etcd-data
          ports:
            - name: client
              containerPort: 2379
            - name: peer
              containerPort: 2380
          readinessProbe:
            httpGet:
              path: /health
              port: 2379
            initialDelaySeconds: 5
            periodSeconds: 5
          livenessProbe:
            httpGet:
              path: /health
              port: 2379
            initialDelaySeconds: 10
            periodSeconds: 10
          volumeMounts:
            - name: etcd-data
              mountPath: /etcd-data
      volumes:
        - name: etcd-data
          emptyDir: {}
EOF

cat > "$ETCDDIR/templates/service.yaml" << 'EOF'
apiVersion: v1
kind: Service
metadata:
  name: web3-blitz-etcd
  namespace: {{ .Release.Namespace }}
spec:
  selector:
    app: web3-blitz-etcd
  ports:
    - name: client
      port: 2379
      targetPort: 2379
    - name: peer
      port: 2380
      targetPort: 2380
EOF

echo "✅ etcd chart: $ETCDDIR"

echo ""
echo "目录结构："
find "$DEPLOY_DIR/web3-blitz-postgres" "$DEPLOY_DIR/web3-blitz-etcd" -type f | sort

echo ""
echo "helm lint 验证："
helm lint "$PGDIR"
helm lint "$ETCDDIR"
