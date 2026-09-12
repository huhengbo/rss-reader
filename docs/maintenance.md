# Maintenance Policy

本文件定义 RSS Reader 的长期维护方式。目标是降低升级成本，同时避免为了“同步上游”或“追版本”破坏已经稳定的行为。

## Upstream 关系

仓库来源关系：

```text
srcrs/rss-reader
  └── myarchiving/rss-reader
       └── huhengbo/rss-reader
```

本仓库已经完成独立的工程化重构，因此 upstream 仅作为功能、bugfix 和安全修复的信息来源，不再作为可直接整体合并的代码基线。

## 2026-09-12 Upstream 差异评估

本次维护检查比较了当前 `main` 与直接父仓库 `myarchiving/rss-reader:main`：

- 两边已经 `diverged`；
- 父仓库存在 10 个本仓库未包含的提交；
- 本仓库存在 16 个父仓库未包含的提交；
- 父仓库相对共同基线主要修改 `README.md`、`utils/feed.go`、`utils/match.go`。

评估结果：**不执行整体 merge**。

原因：

1. 本仓库已经移除 `globals / models / utils` 旧结构，改为 `cmd + internal` 明确职责边界；直接 merge 会重新引入旧架构冲突。
2. 父仓库关键词逻辑改为正则表达式，而本仓库当前公开语义是“正向包含词 + `-` 排除词”；这是行为变更，不应通过 upstream merge 隐式改变。
3. 父仓库 Feed 逻辑包含针对 `nodeloc.com` 的站点特例。此类 workaround 不具有通用性，没有独立 Issue 和测试时不移植。
4. 本仓库已加入显式 State/Archive、context、HTTP/WebSocket 生命周期、安全配置和 CI 门禁，直接同步旧实现会造成回退。

### 当前采用结论

| Upstream 类型 | 当前策略 |
| --- | --- |
| 安全修复 | 优先评估并按本仓库架构重新实现 / cherry-pick |
| 通用 bugfix | 有回归测试后选择性移植 |
| 行为新功能 | 单独 Issue 评估，明确兼容性 |
| 正则关键词语义 | 不采用；如需要，作为显式可选功能设计 |
| `nodeloc.com` 特例 | 不采用；除非有可复现通用问题 |
| README 变化 | 不同步；本仓库维护独立文档 |

## Upstream 检查流程

建议低频检查，例如每月或准备发布前：

```bash
git fetch upstream

git log --oneline main..upstream/main
git diff main...upstream/main
```

对每个有价值的提交记录以下结论：

- `adopt`：直接采用或 cherry-pick；
- `reimplement`：按本仓库结构重新实现；
- `skip`：不适用，并说明理由。

不要为了消除“ahead / behind”数字而合并。

## Go 与依赖策略

当前 Go 基线：Go 1.27.x。

原则：

- 保持在 Go 官方支持周期内；
- CI 固定 major/minor（`1.27.x`），允许自动取得最新 patch；
- 升级到新 Go major/minor 单独 PR，不随 Dependabot 小升级混入；
- Dependabot 每周检查 Go modules 与 GitHub Actions；
- security 和已知 bugfix 优先；
- 所有依赖升级必须通过 `go test`、`go test -race`、跨平台 build 与 `govulncheck`；
- `go mod tidy` 必须保持工作区无差异。

## CI 门禁

`main` 与 Pull Request 必须通过：

- module verify / tidy；
- gofmt；
- go vet；
- tests；
- race detector；
- build；
- govulncheck；
- PR 的 Linux / macOS / Windows build；
- PR 的 Docker build。

不要通过 `continue-on-error` 长期掩盖安全扫描失败。如果出现漏洞，应升级、修复，或在 Issue 中记录为什么不可达 / 暂时不能修复。

## Docker 与发布

正式版本使用 SemVer tag：

```text
vMAJOR.MINOR.PATCH
```

发布链路：

```text
Git tag
  -> GitHub Actions Release
  -> linux/amd64 + linux/arm64
  -> GHCR
  -> GitHub Release
```

发布要求：

1. `main` CI 全绿；
2. 更新文档中任何 breaking change；
3. 确认 `config.example.json` 不含真实凭据；
4. 创建并推送版本 tag；
5. 检查 GHCR manifest、provenance / SBOM 与 GitHub Release；
6. Docker tag 与 Git tag 必须可互相追溯。

镜像标签：

- `X.Y.Z`
- `X.Y`
- `X`
- `latest`

生产环境推荐固定 `X.Y.Z`。

## 安全维护

- `config.json`、`archives.txt` 不进入 Git；
- 通知凭据优先环境变量注入；
- 已经泄露到历史记录的凭据应在提供方轮换，而不是只依赖删除最新文件；
- 日志必须脱敏认证 Feed URL；
- 安全问题遵循 `SECURITY.md`。

## 避免过度设计

维护性优先不等于增加更多层级：

- 只有真实职责边界才拆包；
- 只有多实现 / 测试替换需求才增加接口；
- 不为一次性迁移保留永久脚本；
- 不为单站点 workaround 污染通用路径；
- 不引入没有当前使用场景的配置和扩展点。
