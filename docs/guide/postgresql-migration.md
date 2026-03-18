# PostgreSQL 迁移指南

> 从 SQLite 迁移到 PostgreSQL 的完整记录，包括设计决策和注意事项。

---

## 为什么迁移

SQLite 是单文件数据库，适合本地开发，但有几个生产环境的硬伤：

- 不支持并发写入（WAL 模式下有限支持，但性能差）
- 不支持网络访问，无法多副本共享
- `REAL` 类型存金额有浮点精度问题（0.1 + 0.2 ≠ 0.3）

PostgreSQL 解决了以上所有问题，并且是交易所行业标准。

---

## 关键变更

### 1. 金额字段类型

```sql
-- SQLite（有精度问题）
amount REAL NOT NULL

-- PostgreSQL（精确到小数点后8位）
amount NUMERIC(20,8) NOT NULL
```

`NUMERIC(20,8)` 是定点数，不会有浮点误差，适合存储 BTC/ETH 金额。

sqlc 将 `NUMERIC` 映射为 Go 的 `string` 类型，handler 层需要手动转换：

```go
// 写入 DB
Amount: fmt.Sprintf("%.8f", req.Amount)

// 读出展示
var f float64
fmt.Sscanf(w.Amount, "%f", &f)
```

### 2. 时间字段类型

```sql
-- SQLite
created_at DATETIME DEFAULT CURRENT_TIMESTAMP

-- PostgreSQL
created_at TIMESTAMPTZ DEFAULT NOW()
```

`TIMESTAMPTZ` 带时区信息，sqlc 映射为 `sql.NullTime`。

### 3. 自增主键

```sql
-- SQLite
id INTEGER PRIMARY KEY AUTOINCREMENT

-- PostgreSQL
id BIGSERIAL PRIMARY KEY
```

### 4. 去重插入

```sql
-- SQLite
INSERT OR IGNORE INTO deposit_addresses ...

-- PostgreSQL
INSERT INTO deposit_addresses ...
ON CONFLICT (address) DO NOTHING
```

### 5. 占位符

```sql
-- SQLite
WHERE user_id = ? AND chain = ?

-- PostgreSQL（sqlc @param 风格，自动转换为 $1 $2）
WHERE user_id = @user_id AND chain = @chain
```

### 6. sqlc.yaml engine

```yaml
# 改之前
engine: "sqlite"

# 改之后
engine: "postgresql"
```

---

## 数据库初始化

PostgreSQL 容器首次启动时会自动执行 `docker-entrypoint-initdb.d/` 下的 SQL 文件：

```yaml
volumes:
  - ./internal/db/schema.sql:/docker-entrypoint-initdb.d/schema.sql
```

手动执行：
```bash
docker exec -i postgres psql -U blitz -d blitz < internal/db/schema.sql
```

---

## 连接配置

通过 `DATABASE_URL` 环境变量注入：

```bash
DATABASE_URL=postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable
```

K8s 生产环境走 Secret：
```yaml
env:
  - name: DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: wallet-secret
        key: database-url
```

---

## 注意事项

**NUMERIC → string 的类型转换**

sqlc 把 `NUMERIC` 列生成为 Go `string`，`COALESCE(SUM(amount), 0)` 也返回 `string`。handler 里用 `interface{}` 接收时需要处理多种类型：

```go
toFloat := func(v interface{}) float64 {
    switch val := v.(type) {
    case float64:
        return val
    case string:
        var f float64
        fmt.Sscanf(val, "%f", &f)
        return f
    case []byte:
        var f float64
        fmt.Sscanf(string(val), "%f", &f)
        return f
    }
    return 0
}
```

**schema.sql 的 CHECK 约束**

```sql
chain TEXT NOT NULL CHECK(chain IN ('btc','eth'))
```

新增链时记得更新 CHECK 约束，否则写入会报错。
