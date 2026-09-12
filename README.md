# RSS Reader

[![CI](https://github.com/huhengbo/rss-reader/actions/workflows/ci.yml/badge.svg)](https://github.com/huhengbo/rss-reader/actions/workflows/ci.yml)
[![Release](https://github.com/huhengbo/rss-reader/actions/workflows/release.yml/badge.svg)](https://github.com/huhengbo/rss-reader/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/huhengbo/rss-reader)](LICENSE)

一个轻量的自托管 RSS 聚合页面，支持关键词匹配、WebSocket 自动刷新，以及飞书、Telegram、钉钉通知。

本仓库基于 `srcrs/rss-reader` 的后续 fork 演进而来，目前已经形成独立的工程结构和维护策略。上游关系与同步原则见 [维护策略](docs/maintenance.md)。

## 功能

- 聚合多个 RSS / Atom / JSON Feed 订阅源
- 响应式 Web 页面与系统深色模式
- WebSocket 推送页面更新
- 关键词包含 / 排除匹配
- 飞书、Telegram、钉钉通知
- 配置文件热更新
- 通知链接持久化去重
- `/healthz` 健康检查
- HTTP / WebSocket 超时与优雅退出
- Docker 非 root 运行
- GitHub Actions：测试、race、漏洞扫描、跨平台编译、Docker 构建
- Release 自动发布 `linux/amd64` 与 `linux/arm64` GHCR 镜像

## 预览

### 桌面端

![Desktop](pc.png)

### 深色模式

![Dark mode](pc_night.png)

### 移动端

![Mobile](mobile.png)

## 快速开始

### Docker Compose

要求 Docker Engine 与 Docker Compose v2。

```bash
git clone https://github.com/huhengbo/rss-reader.git
cd rss-reader
cp config.example.json config.json
```

编辑 `config.json` 后启动：

```bash
docker compose up -d
```

默认映射到宿主机 `9898` 端口：

```text
http://localhost:9898
```

Compose 默认使用：

```text
ghcr.io/huhengbo/rss-reader:latest
```

也可以指定固定版本，生产部署推荐使用版本标签而不是长期依赖 `latest`：

```bash
RSS_READER_IMAGE=ghcr.io/huhengbo/rss-reader:1.0.0 docker compose up -d
```

从源码构建本地镜像：

```bash
docker build -t rss-reader:local .
RSS_READER_IMAGE=rss-reader:local docker compose up -d
```

### 本地 Go 运行

要求 Go 1.27.x。

```bash
cp config.example.json config.json
go run ./cmd/rss-reader
```

默认访问：

```text
http://localhost:8080
```

## 配置

仓库只提交安全的 `config.example.json`。真实 `config.json`、运行时 `archives.txt` 和本地配置文件不会进入 Git。

基础配置示例：

```json
{
  "port": 8080,
  "values": [
    "https://example.com/feed.xml"
  ],
  "refresh": 5,
  "autoUpdatePush": 0,
  "listHeight": 600,
  "webTitle": "RSS Reader",
  "webDes": "My RSS dashboard",
  "keywords": [
    "example -ignore"
  ],
  "notify": {
    "feishu": {
      "api": ""
    },
    "dingtalk": {
      "webhook": "",
      "sign": ""
    },
    "telegram": {
      "api": "https://api.telegram.org/bot${token}/sendMessage",
      "chat_id": "",
      "token": ""
    }
  },
  "archives": "archives.txt"
}
```

| 字段 | 说明 | 默认值 |
| --- | --- | --- |
| `port` | HTTP 监听端口 | `8080` |
| `values` | RSS / Atom / JSON Feed 地址列表 | 空 |
| `refresh` | 后端拉取订阅源的间隔，单位分钟 | `5` |
| `autoUpdatePush` | WebSocket 页面推送间隔，`0` 表示只推送连接时的当前数据 | `0` |
| `listHeight` | 页面 Feed 列表高度 | `600` |
| `webTitle` | 页面标题 | 空 |
| `webDes` | 页面描述 | 空 |
| `keywords` | 通知关键词规则 | 空 |
| `notify` | 通知渠道配置 | 空 |
| `archives` | 已通知链接的去重文件 | `archives.txt` |

### 关键词规则

每个 `keywords` 元素是一组规则。普通词表示“包含任一正向词即可匹配”，以 `-` 开头的词表示排除条件；匹配不区分大小写。

例如：

```json
{
  "keywords": [
    "抽奖 -测评",
    "cloudcone racknerd -expired"
  ]
}
```

`"抽奖 -测评"` 表示标题包含“抽奖”且不包含“测评”时匹配。

## 敏感配置与环境变量

生产环境建议通过环境变量注入通知凭据，不要把 token / webhook / 私有订阅地址写入仓库。

| 环境变量 | 作用 |
| --- | --- |
| `RSS_READER_CONFIG` | 配置文件路径 |
| `RSS_READER_PORT` | HTTP 端口覆盖 |
| `RSS_READER_ARCHIVES` | archive 文件路径覆盖 |
| `RSS_READER_FEISHU_API` | 飞书 webhook |
| `RSS_READER_DINGTALK_WEBHOOK` | 钉钉 webhook |
| `RSS_READER_DINGTALK_SIGN` | 钉钉签名 secret |
| `RSS_READER_TELEGRAM_API` | Telegram API 模板 |
| `RSS_READER_TELEGRAM_CHAT_ID` | Telegram chat id |
| `RSS_READER_TELEGRAM_TOKEN` | Telegram bot token |

Telegram 的 `token` 与 `chat_id` 必须同时配置。

更多安全说明见 [SECURITY.md](SECURITY.md)。

## 项目结构

```text
.
├── cmd/
│   └── rss-reader/          # 可执行入口与依赖组装
├── internal/
│   ├── archive/             # 通知去重持久化
│   ├── config/              # 配置加载、默认值、校验
│   ├── domain/              # Feed 领域模型
│   ├── feed/                # RSS 拉取、匹配、配置监听
│   ├── notify/              # 飞书 / Telegram / 钉钉
│   ├── server/              # HTTP / WebSocket handler
│   ├── state/               # 线程安全运行时状态
│   └── web/                 # 嵌入式前端资源
├── .github/
│   ├── workflows/ci.yml     # 质量门禁
│   └── workflows/release.yml
├── config.example.json
├── docker-compose.yml
└── Dockerfile
```

核心原则是显式依赖、单向职责边界，避免 package-level mutable globals 和新的“万能 utils 包”。

## 开发与验证

提交前至少运行：

```bash
go mod tidy
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
govulncheck ./...
```

CI 会验证：

- `go mod verify` / `go mod tidy`
- `gofmt`
- `go vet`
- 单元 / 回归测试
- race detector
- `govulncheck`
- Linux、macOS、Windows 编译
- Docker image build

贡献规范见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 健康检查

```bash
curl http://localhost:8080/healthz
```

正常返回：

```text
ok
```

Docker 镜像已内置该健康检查。

## Nginx 反向代理

WebSocket 路径为 `/ws`：

```nginx
server {
    listen 443 ssl;
    server_name rss.example.com;

    ssl_certificate     /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:9898;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://127.0.0.1:9898/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 120s;
    }
}
```

WebSocket 默认执行同源 Origin 校验，因此反向代理需要保留正确的 `Host`。

## 发布

版本使用 SemVer 标签：

```text
vMAJOR.MINOR.PATCH
```

推送 `v*` Git tag 后，Release workflow 会：

1. 构建 `linux/amd64` 和 `linux/arm64` 镜像；
2. 推送到 `ghcr.io/huhengbo/rss-reader`；
3. 生成 SemVer 镜像标签与 `latest`；
4. 生成 provenance 和 SBOM；
5. 创建对应 GitHub Release 并自动生成 release notes。

## Upstream 与维护策略

本仓库是长期维护的二次开发版本，不执行无审查的 upstream 整体 merge。安全修复和明确 bugfix 优先选择性移植，站点特例或改变现有语义的功能需单独评估。

详细规则、当前 upstream 差异评估和依赖策略见 [docs/maintenance.md](docs/maintenance.md)。

## License

[MIT](LICENSE)
