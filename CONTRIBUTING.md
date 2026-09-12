# Contributing

感谢参与 RSS Reader 的维护。

## 开始之前

- Bug 请先确认当前 `main` 是否仍可复现。
- 安全问题、凭据泄露、私有 Feed URL 不要提交到公开 Issue，请按 `SECURITY.md` 处理。
- 功能改动优先解决明确使用场景，避免为未来假设增加抽象层。
- 大范围重构应拆成可独立验证的小 PR。

## 本地开发

要求 Go 1.27.x。

```bash
git clone https://github.com/huhengbo/rss-reader.git
cd rss-reader
cp config.example.json config.json
go run ./cmd/rss-reader
```

## 提交前检查

首次安装漏洞扫描工具：

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

提交前运行：

```bash
go mod tidy
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build ./...
govulncheck ./...
```

如果修改 Dockerfile：

```bash
docker build -t rss-reader:dev .
```

## 代码约定

- `cmd/rss-reader` 只负责应用启动和依赖组装。
- 新的业务逻辑放入职责明确的 `internal/*` 包。
- 避免 package-level mutable globals。
- 不创建职责模糊的 `utils` / `common` 万能包。
- 接口只在确实存在替换、测试或多实现边界时引入。
- 外部网络调用必须支持 context 和 timeout。
- 并发状态必须明确所有权并通过 race test。
- 对 bug 修复优先增加回归测试。
- 日志不得输出 token、webhook、带鉴权查询参数的 Feed URL。

## Commit 与 Pull Request

推荐使用清晰的 Conventional Commit 风格，例如：

```text
fix: avoid duplicate notification writes
feat: add notification provider
refactor: isolate feed runtime state
ci: validate Docker image build
build: publish multi-arch images
docs: clarify deployment configuration
```

一个 PR 尽量聚焦一个目标。PR 描述至少说明：

- 为什么修改；
- 修改了什么；
- 是否改变外部行为；
- 如何验证；
- 关联 Issue。

CI 必须通过后再合并。

## 依赖升级

Dependabot 每周检查 Go modules 和 GitHub Actions。

升级优先级：

1. security；
2. bugfix；
3. 当前 Go 支持周期；
4. 有明确收益的功能版本。

不为了追求“全部最新”而接受无收益的大范围升级。依赖 PR 必须通过 `test / race / build / govulncheck`。

## 发布

版本遵循 SemVer：`vMAJOR.MINOR.PATCH`。

只有维护者推送 `v*` tag 才触发正式发布。Git tag、GitHub Release 和 GHCR 镜像版本必须互相可追溯。

## Upstream

本仓库不直接整体 merge 上游。同步前请按 `docs/maintenance.md` 对提交分类：security / bugfix / feature / site-specific workaround，并说明采用或不采用的原因。
