# ====================== Builder Stage ======================
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app

RUN go env -w GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags '-s -w -extldflags "-static"' \
    -o blitz ./cmd/wallet-service

# ====================== Runtime Stage ======================
# scratch：零依赖零 OS 漏洞，适合纯静态 Go 二进制
FROM scratch AS runtime

# OCI 标准标签（企业镜像扫描和审计必备）
LABEL org.opencontainers.image.source="https://github.com/Ixecd/KubePivot" \
      org.opencontainers.image.version="v0.1.0" \
      org.opencontainers.image.vendor="blitz" \
      org.opencontainers.image.title="blitz"

# CA 证书（支持 HTTPS 出站调用）
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/blitz /blitz

# 数据库迁移文件
COPY --from=builder /app/internal/db/migrations /migrations

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/blitz"]
