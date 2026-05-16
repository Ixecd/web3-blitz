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
    -o sweep ./cmd/sweep

# ====================== Runtime Stage ======================
FROM alpine:3.20

LABEL org.opencontainers.image.source="https://github.com/Ixecd/KubePivot" \
      org.opencontainers.image.version="v0.1.0" \
      org.opencontainers.image.vendor="blitz" \
      org.opencontainers.image.title="blitz-sweep"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/sweep /sweep

USER 65532:65532

ENTRYPOINT ["/sweep"]
