# syntax=docker/dockerfile:1.7

FROM golang:1.27.1-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/rss-reader \
    ./cmd/rss-reader

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 rss-reader \
    && adduser -S -D -H -u 10001 -G rss-reader rss-reader \
    && mkdir -p /app /data \
    && chown -R rss-reader:rss-reader /app /data

WORKDIR /app
COPY --from=builder --chown=rss-reader:rss-reader /out/rss-reader /app/rss-reader

ENV TZ=Asia/Shanghai
EXPOSE 8080

USER 10001:10001

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q -O- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/rss-reader"]
