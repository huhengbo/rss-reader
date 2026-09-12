# 更新日志

这里记录 RSS Reader 各正式版本的用户可见变更。GitHub Release 发布说明从对应版本章节自动生成。

## 发布说明模板

发布新版本前，复制下面结构并把标题改为实际 SemVer，例如 `## 1.1.0`。`### 📦 升级说明` 为必填章节；缺失时 Release workflow 会直接失败。

### ✨ 主要更新

- **功能名称**
  - 简要说明用户能获得什么。

### 🐛 问题修复

- 修复具体问题；没有可删除本章节。

### ⚙️ 优化调整

- 优化性能、交互、构建、兼容性或维护体验；没有可删除本章节。

### 📦 升级说明

- **可直接升级**：说明从哪些版本可以直接升级。
- **配置迁移**：说明 `config.json` / 环境变量是否有新增、删除、重命名或默认值变化。
- **数据迁移**：说明 `archives.txt` 或其他运行时数据是否需要处理；没有迁移也要明确写“无需数据迁移”。
- **Docker Compose**：说明是否只需拉取新镜像并重建容器，是否需要修改 `.env` 中固定版本。
- **默认行为**：说明升级是否改变现有默认行为，是否有需要用户主动开启的选项。
- **浏览器缓存**：前端有明显变化时说明是否需要强制刷新。

### ✅ 兼容性

- 说明本版本重点验证的平台、架构、浏览器或配置兼容性。

### 🐳 镜像

- 固定版本：`ghcr.io/huhengbo/rss-reader:X.Y.Z`
- 滚动版本：`ghcr.io/huhengbo/rss-reader:latest`
- 正式环境建议固定具体版本，不建议长期依赖 `latest`。

## 版本规则

RSS Reader 使用 SemVer：`MAJOR.MINOR.PATCH`，Git tag 使用 `v` 前缀，例如 `v1.1.0`。

- `MAJOR`：存在不兼容变更。
- `MINOR`：向后兼容的新功能。
- `PATCH`：向后兼容的问题修复。
- 已发布 tag 不移动、不覆盖；需要修正时发布新的 SemVer 版本。

## 1.0.0

> 本仓库完成独立工程化、可靠性加固与多架构发布后的首个正式版本。

### ✨ 主要更新

- 重构为清晰的 Go 项目结构，移除业务层全局可变状态。
- 增加配置热更新、通知去重持久化、健康检查与优雅退出。
- 建立 Go test / race / govulncheck / 三系统编译 / Docker 构建 CI。
- 发布 `linux/amd64` 与 `linux/arm64` GHCR 镜像，并附带 provenance / SBOM。

### 🔒 安全与稳定性

- 运行时敏感配置不再进入公开仓库示例。
- 外部网络调用增加超时与生命周期控制。
- Docker 镜像使用非 root 用户运行。

### 📦 升级说明

- **首个正式版本**：`v1.0.0` 是当前仓库独立维护后的首个正式 tag，没有更早的本仓库正式版本可作为直接升级基线。
- **旧部署迁移**：从未打 tag 的旧提交或上游版本迁移时，先备份现有 `config.json` 与 `archives.txt`，再按当前示例核对配置字段，不要用示例文件覆盖真实配置。
- **无需数据库迁移**：本版本不引入数据库迁移步骤；通知去重状态仍由归档文件维护。
- **Docker Compose**：使用固定版本时将 `RSS_READER_IMAGE` 指向 `ghcr.io/huhengbo/rss-reader:1.0.0`，然后执行 `docker compose pull && docker compose up -d`。
- **默认行为**：升级前请重点核对通知 webhook/token 与私有 Feed 地址仍由运行时配置或环境变量提供。

### ✅ 兼容性

- 正式镜像提供 `linux/amd64` 与 `linux/arm64`。
- Go CI 覆盖 Linux / macOS / Windows 构建；Docker 正式镜像面向 Linux 容器运行时。

### 🐳 镜像

- `ghcr.io/huhengbo/rss-reader:1.0.0`
- `ghcr.io/huhengbo/rss-reader:latest`
