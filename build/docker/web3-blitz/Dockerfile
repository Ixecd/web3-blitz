FROM golang:1.24-alpine AS builder

ARG SERVICE_NAME=web3-blitz

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/${SERVICE_NAME} ./cmd/${SERVICE_NAME}

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/web3-blitz /app/bin/

EXPOSE 8080

CMD ["/app/bin/web3-blitz"]