# RSS Reader

[![CI](https://github.com/huhengbo/rss-reader/actions/workflows/ci.yml/badge.svg)](https://github.com/huhengbo/rss-reader/actions/workflows/ci.yml)
[![UI](https://github.com/huhengbo/rss-reader/actions/workflows/ui.yml/badge.svg)](https://github.com/huhengbo/rss-reader/actions/workflows/ui.yml)
[![Release](https://github.com/huhengbo/rss-reader/actions/workflows/release.yml/badge.svg)](https://github.com/huhengbo/rss-reader/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/huhengbo/rss-reader)](LICENSE)

轻量的自托管 RSS 聚合页面，支持关键词通知、可靠的实时更新，以及四套独立于明暗模式的阅读皮肤。

本仓库基于 `srcrs/rss-reader` 的后续 fork 演进，维护独立的工程结构与测试。来源关系与同步原则见 [维护策略](docs/maintenance.md)。

> 本 README 描述当前源码。UI 改造没有自动发版；已发布的 `v1.0.0` 镜像不含本轮新界面。需要新界面时使用当前源码构建，正式镜像版本以 Release 为准。

## 功能

- 聚合 RSS / Atom / JSON Feed，飞书、Telegram、钉钉关键词通知。
- 完整有序快照、同站多源独立、加载 / 空源 / 失败 / 陈旧状态。
- 桌面与移动端均可无整页刷新地恢复连接，新文章先提示再应用。
- 本地标题搜索、源筛选、折叠、展开更多与舒适 / 紧凑密度。
- Slate / Paper / Grove / Terminal 四套皮肤，均支持浅色、深色、跟随系统。
- 中文语义化页面、键盘焦点、减少动态效果；无 JavaScript 时保留静态阅读。
- 配置热更新、通知链接持久化去重、`/healthz` 与优雅退出。
- Go embed 单二进制，无运行时前端框架、CDN、外部字体或 Node 依赖。
- Go / race / 漏洞扫描、三系统编译、三浏览器交互测试、Docker 构建。
- Release 发布 `linux/amd64` 与 `linux/arm64` 的非 root GHCR 镜像。

## 阅读皮肤

| 名称 | 风格 |
| --- | --- |
| **Slate**（默认） | 冷灰底、靛蓝强调、细边框，适合工作台式扫读 |
| **Paper** | 暖纸色、衬线标题、印刷版式分隔线 |
| **Grove** | 雾绿底、自然绿强调、柔和圆角与舒展间距 |
| **Terminal** | 等宽排版、低饱和绿、方角，无持续闪烁或辉光 |

四套皮肤共用组件与业务逻辑，不维护四份页面。皮肤、明暗、密度独立选择；偏好存储不可用时仍可正常阅读。截图检查、皮肤扩展、交互与协议说明见 [阅读界面维护文档](docs/ui.md)。

## 快速开始

### Docker Compose

要求 Docker Engine 与 Docker Compose v2。

```bash
git clone https://github.com/huhengbo/rss-reader.git
cd rss-reader
cp config.example.json config.json
```

Windows PowerShell 的复制命令为 `Copy-Item config.example.json config.json`。编辑实际订阅配置后启动：

```text
docker compose up -d
```

默认访问 `http://localhost:9898`。Compose 使用 `ghcr.io/huhengbo/rss-reader:latest`；这对应已发布镜像，不是每次 main 提交都会更新。

生产部署推荐固定版本，在 Compose 同目录 `.env` 中指定（macOS / Windows / Linux 通用）：

```dotenv
RSS_READER_IMAGE=ghcr.io/huhengbo/rss-reader:1.0.0
```

从当前源码构建包含新界面的镜像：

```text
docker build -t rss-reader:local .
```

将 `.env` 的 `RSS_READER_IMAGE` 改为 `rss-reader:local`，再执行 `docker compose up -d`。不要用空示例覆盖已有真实配置。

### 本地 Go

要求 Go 1.27.x，不需要安装 Node 或执行 npm build。

```text
go run ./cmd/rss-reader
```

运行前从 `config.example.json` 创建并编辑 `config.json`。默认访问 `http://localhost:8080`。

## 配置

仓库只提交安全示例。真实配置、运行时归档与本地凭据不要提交到 Git。

```json
{
  "port": 8080,
  "values": ["https://example.com/feed.xml"],
  "refresh": 5,
  "autoUpdatePush": 0,
  "listHeight": 600,
  "webTitle": "RSS Reader",
  "webDes": "My RSS dashboard",
  "keywords": ["example -ignore"],
  "notify": {
    "feishu": {"api": ""},
    "dingtalk": {"webhook": "", "sign": ""},
    "telegram": {"api": "https://api.telegram.org/bot${token}/sendMessage", "chat_id": "", "token": ""}
  },
  "archives": "archives.txt"
}
```

| 字段 | 说明 | 默认值 |
| --- | --- | --- |
| `port` | HTTP 端口；变更需要重启 | `8080` |
| `values` | RSS / Atom / JSON Feed 地址列表 | 空 |
| `refresh` | 后台外部抓取间隔，单位分钟 | `5` |
| `autoUpdatePush` | 新页面：`0` 为单次快照；正值开启实时更新。旧 `/ws` 仍按该值的分钟周期发送 | `0` |
| `listHeight` | 桌面文章列表最大高度；移动端使用文档滚动 | `600` |
| `webTitle` | 页面标题；空值时界面使用 RSS Reader | 空 |
| `webDes` | 页面描述 | 空 |
| `keywords` | 通知关键词规则 | 空 |
| `notify` | 通知渠道配置 | 空 |
| `archives` | 已通知链接去重文件 | `archives.txt` |

新协议 `/ws?v=1` 在实时模式下每秒检查缓存 revision，有变化才发送完整快照；**不增加外部 RSS 抓取频率**。`autoUpdatePush=0` 时点击“刷新视图”只读取缓存。连接正常不代表所有源抓取成功，源状态和最后成功同步时间单独显示。

### 关键词规则

每个 keywords 元素是一组规则。普通词表示“包含任一正向词”，以 `-` 开头表示排除，不区分大小写：

```json
{"keywords": ["抽奖 -测评", "cloudcone racknerd -expired"]}
```

例如 `"抽奖 -测评"` 匹配包含“抽奖”但不包含“测评”的标题。页面搜索独立于通知规则，不改变通知行为。

## 敏感配置

建议通过环境变量注入通知凭据；私有订阅地址仍由运行时配置管理，不放入示例或公开 Issue。

| 环境变量 | 作用 |
| --- | --- |
| `RSS_READER_CONFIG` | 配置路径 |
| `RSS_READER_PORT` | HTTP 端口覆盖 |
| `RSS_READER_ARCHIVES` | 归档文件路径覆盖 |
| `RSS_READER_FEISHU_API` | 飞书 webhook |
| `RSS_READER_DINGTALK_WEBHOOK` | 钉钉 webhook |
| `RSS_READER_DINGTALK_SIGN` | 钉钉签名 secret |
| `RSS_READER_TELEGRAM_API` | Telegram API 模板 |
| `RSS_READER_TELEGRAM_CHAT_ID` | Telegram chat id |
| `RSS_READER_TELEGRAM_TOKEN` | Telegram token |

Telegram token 与 chat_id 必须同时配置。新快照不返回原始配置 URL，但阅读内容本身可能是私有的；页面没有用户登录系统，公网部署必须在反向代理增加访问控制。详见 [SECURITY.md](SECURITY.md)。

## 项目结构

```text
cmd/rss-reader/                  应用入口与依赖组装
internal/
  archive/                      通知去重持久化
  config/                       加载、默认值与校验
  domain/                       Feed 与版本化快照
  feed/                         拉取、匹配与配置监听
  notify/                       通知渠道
  server/                       HTTP / WebSocket 与共享模板
    testdata/ui/                仅用于 UI 测试的本地数据服务
  state/                        线程安全状态、源身份、抓取状态
  web/static/                   原生 HTML / JS / CSS，直接 embed
tests/ui/                       Node 纯逻辑与生命周期回归
tests/browser/                  Playwright 真实浏览器回归
scripts/check-assets.js          运行时资源预算检查
.github/workflows/
  ci.yml                        Go、跨系统编译、Docker
  ui.yml                        Chromium / Firefox / WebKit
  release.yml                   多架构镜像发布
docs/                           维护、UI 与协议文档
```

避免全局可变状态、万能 utils 包和没有实际用途的抽象。前端模块按职责分离，但不引入运行时构建链。

## 开发与验证

Go 检查：

```text
go mod tidy
gofmt -w cmd internal
go vet ./...
go test ./...
go test -race ./...
go build ./...
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

UI 开发者另需 Node.js 22+（CI 使用 24）：

```text
npm ci --ignore-scripts
npm test
npm run check:assets
npx playwright install chromium firefox webkit
npm run test:e2e
```

测试自动启动 localhost 固定数据服务，不调用真实 RSS 或通知。Linux 可能需要 `npx playwright install --with-deps chromium firefox webkit`。只测 Chromium 可用 `npm run test:e2e -- --project=chromium`。

CI 保留浏览器报告、截图与失败 trace；资源检查区分原始和 gzip 大小，不能把压缩测量当作实际传输量。人工屏幕阅读器和浏览器 200% 缩放验收仍单独跟踪，自动测试不是完整可访问性认证。

详见 [CONTRIBUTING.md](CONTRIBUTING.md) 与 [docs/ui.md](docs/ui.md)。

## 健康检查与反向代理

```text
curl http://localhost:8080/healthz
```

返回 `ok`，镜像内置同一健康检查。它表示 HTTP 服务存活，不表示全部订阅源可用。

Nginx 示例（证书与域名替换为实际值，并按需增加认证）：

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

保留原始 Host 与 `/ws?v=1` 的查询参数。同源 Origin 校验默认开启。旧 `/feeds`、无版本 `/ws` 保留兼容，新增 `/api/v1/snapshot` 为完整快照接口。

## 发布与维护

SemVer：`vMAJOR.MINOR.PATCH`。推送版本 tag 才触发 Release workflow，构建 amd64/arm64，发布 GHCR SemVer 标签与 latest，附带 provenance / SBOM，并创建 GitHub Release。

Go、GitHub Actions、npm 测试依赖由 Dependabot 分组更新，必须通过相应 CI。上游只选择性移植经审查的安全修复和通用 bugfix，不为消除 ahead/behind 数字整体覆盖本仓库。详见 [维护策略](docs/maintenance.md)。

## License

[MIT](LICENSE)
