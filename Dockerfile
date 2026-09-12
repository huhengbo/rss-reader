FROM golang:1.20.4-alpine3.18 AS builder

COPY . /src
WORKDIR /src

# 国内服务器可以取消以下注释
# RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN go build -ldflags "-s -w" -o /out/rss-reader ./cmd/rss-reader

FROM alpine

COPY --from=builder /out/rss-reader /app/rss-reader

WORKDIR /app

EXPOSE 8080

RUN apk add --no-cache tzdata
ENV TZ=Asia/Shanghai

ENTRYPOINT ["./rss-reader"]
