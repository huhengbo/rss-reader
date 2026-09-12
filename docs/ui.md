# 阅读界面与皮肤维护

本页描述 UI 改造后的源码行为，不表示已发布的 `v1.0.0` 镜像包含这些变更。版本与发布状态以 Git tag / GitHub Release 为准。

## 使用

| 皮肤 | 视觉方向 |
| --- | --- |
| Slate（默认） | 冷灰背景、靛蓝强调、适中圆角，适合工作台式浏览 |
| Paper | 暖纸色、衬线标题、印刷版式分隔线 |
| Grove | 雾绿背景、自然绿强调、舒展间距与柔和圆角 |
| Terminal | 等宽排版、低饱和绿、方角和清晰边框；没有闪烁或辉光 |

每套皮肤都有浅色与深色。皮肤、明暗（跟随系统 / 浅色 / 深色）、密度（舒适 / 紧凑）彼此独立。

外观与折叠偏好保存在当前浏览器的 localStorage，不同步到服务器。存储不可用、配额不足或旧数据损坏时回退默认，仍可阅读和切换外观。不会将私有订阅 URL 或通知密钥存入浏览器偏好。更换订阅 URL 后源 ID 会变化，该源的折叠偏好随之重置。

搜索在当前缓存的文章标题与源名中进行，不是全文搜索；清空会同时恢复全部订阅源。搜索不会触发外部请求。默认每个卡片显示前 8 篇，支持展开更多；移动端使用文档滚动，桌面文章列表可在配置高度内滚动。

“刷新视图”只读取服务器最新缓存，不会主动抓取外部 RSS。后台抓取频率仍由 `refresh` 控制。非首次加载时，已有来源的新文章/内容变化先显示提示，由用户点击“应用更新”；源的删除、重排、首次加载结果与错误状态及时同步。

连接状态与订阅源状态分开显示。连接恢复不重新加载整个页面，已有列表和搜索保留。页面隐藏时暂停连接，回到前台时恢复；离线恢复不区分桌面和移动端。`autoUpdatePush=0` 是快照模式，不自动持续更新；需要点击刷新视图获取当前缓存。

文章链接新窗口打开，并使用 `noopener noreferrer`。上游文章 HTML 不插入 DOM；页面不是文章全文阅读器。禁用 JavaScript 后，服务端渲染仍提供可点击的静态文章列表。

## 状态和时间

所有配置源都有状态：`loading`、`ready`、`empty`、`error`、`stale`。首次抓取失败与没有文章不同；失败但有历史缓存时显示旧缓存，不清空文章。

- `lastAttemptAt`：实际抓取尝试时间。
- `lastSuccessAt`：最后一次成功解析时间，即使内容不变也更新。
- `contentChangedAt`：标题、链接或完整文章内容变化时间，不再只比较第一条链接。

时间以 UTC RFC3339 传输。页面显示相对时间，`time[datetime]` 保留机器可读值，完整本地时间与时区在 title 中。健康检查 `/healthz` 只说明进程服务正常，不表示所有订阅源同步成功。

## 模块边界

```text
internal/web/static/
  index.html       Go 共享 source/item 模板、静态首屏、转义后的初始 JSON
  preferences.js   首次绘制前应用外观；可选存储
  model.js         消息校验、链接校验、内容变化/新文章计数
  connection.js    单连接、单重连 timer、取消旧快照请求、生命周期清理
  app.js           keyed DOM、搜索/筛选/折叠/更新提示
  themes.css       语义色彩、字体、圆角、间距、阴影、动效变量
  app.css          组件布局、焦点、小屏和 reduced-motion
```

生产运行没有 Vue、Element Plus、CDN、外部字体或 npm 依赖。Node / Playwright 仅用于测试；Go embed 直接打包上述源码，不要求先执行 npm 构建。

## 新协议与兼容入口

`GET /api/v1/snapshot` 和 `GET /ws?v=1` 返回相同的完整 JSON 快照。`version=1`、`revision`、`sources` 有序数组构成消息边界；空 sources 是有效的权威空列表。revision 只用于判断相同快照，不是可排序的时间戳。未知协议版本被拒绝。

source ID 是配置 URL 的 SHA-256 前 128 位十六进制标识，不暴露原始订阅地址；相同 URL 的 ID 不依赖排序或进程重启。它不是加密凭据或访问控制机制，修改 URL（包括轮换 URL 内认证参数）会改变 ID。同站不同订阅 URL 具有不同 ID；完全相同的配置 URL 在新快照中只显示一次。

文章身份优先使用源 ID + GUID，其次文章链接，再次标题。没有稳定 GUID / 链接的来源，标题变化可能被识别成新文章。文章链接只允许 HTTP(S)，移除 userinfo 与已知敏感查询键；内容提供方仍可能把敏感信息放在任意标题/路径中，因此不要将私人阅读页面直接公开。

新 v1 stream：`autoUpdatePush > 0` 时每秒检查快照 revision，仅有变化才发送完整快照；`0` 时发送一次完整快照并正常关闭。**这与旧客户端的分钟推送周期不同**，目的是及时同步源状态和配置增删。后台外部抓取仍按 refresh 的分钟间隔执行。

旧 `GET /feeds`（只返回已有缓存的 Feed 数组）与无 v 参数的 `/ws`（逐条 Feed、按原分钟周期）保留兼容。旧接口不是新快照的隐私/字段收敛接口；部署仍需要按 SECURITY.md 控制访问。

服务端发送正常关闭帧后，会等待客户端关闭确认或有限超时再关闭 TCP，避免快速关闭与浏览器握手竞争。应用退出显式关闭已升级连接，因为 HTTP Shutdown 不负责它们。

## 增加第五套皮肤

1. 在 themes.css 中增加 `:root[data-skin="new-skin"]`，只覆盖语义变量与必要字体/圆角参数。
2. 增加对应 `[data-color="dark"]` 配色。不要复制业务 JS 或整份组件 CSS。
3. 在 preferences.js 的允许列表及 index.html 的选择器中登记名称。
4. 扩展浏览器皮肤矩阵和对比度检查，review 各状态截图。

正文、弱化文字、强调色、成功/失败/陈旧状态都应覆盖。不得把状态仅编码为颜色。正常文字对比度目标至少 4.5:1；控件键盘 focus 必须明显。

没有全局 transition: all。只对按钮背景/边框/文字与文章链接文字过渡，时间由 token 控制；reduced-motion 下关闭这些非必要过渡。新增动效不要引入动画库，不动画整个列表或持续闪烁。

## 本地验证（macOS / Windows / Linux）

安装 Go 1.27.x、Node.js 22 或更新版本，CI 使用 Node.js 24。以下测试命令不依赖 Bash 特有语法：

```text
npm ci --ignore-scripts
npm test
npm run check:assets
npx playwright install chromium firefox webkit
npm run test:e2e
```

Linux 首次测试可能需要 `npx playwright install --with-deps chromium firefox webkit` 安装系统依赖。

Playwright 会自动启动真实 Go HTTP server fixture，使用固定内存订阅数据，绑定 127.0.0.1:4173；没有真实 RSS/通知调用。Fixture 的 `/fixture/*` 写入口仅存在于 testdata 可执行程序，不存在于生产命令。

只运行一个浏览器：

```text
npm run test:e2e -- --project=chromium
```

手工检查示例数据页面：

```text
npm run test:ui-server
```

浏览器打开 http://127.0.0.1:4173/。这些是测试数据，不能将其误认为已连接用户实际订阅。

## CI 与资源预算

`.github/workflows/ui.yml` 对 PR 和 main 执行 Chromium / Firefox / WebKit；使用锁文件，不自动重试掩盖不稳定失败。测试覆盖连接恢复、消息校验、完整快照、更新确认、源删除、存储、键盘、长文本、无 JS，以及四皮肤 × 两明暗 × 320/390/1440 宽度。

Node 单元测试另外断言只有一个重连计时器、页面停止后移除监听器、取消旧 HTTP 请求等生命周期性质。Go 测试验证实际状态与协议，并继续执行 race。

每次 UI run 保存浏览器截图与报告，失败保存 trace。截图用于 review；目前没有自动像素差异金图门禁，不应把“截图生成成功”称为视觉回归全部验收。禁止不审查便更新视觉基线。

`check:assets` 记录每个 JS/CSS 的原始及独立 gzip 大小，当前预算为 128 KiB 原始 / 32 KiB gzip。输出 artifacts/asset-sizes.json。gzip 是静态压缩测量，并不表示 Go handler 已开启 gzip 传输；部署可在反向代理配置压缩。

测试工具升级由 npm Dependabot 负责，Go / Actions 继续使用原有更新策略。运行时没有第三方 JS vendor，因此无需维护不可复现的 min.js 来源。

## 验收边界

自动检查不等于 WCAG 合规认证。真实屏幕阅读器（如 NVDA / VoiceOver）与浏览器 UI 200% 缩放需要人工记录设备、版本、操作结果；这些工作在 #26 跟踪，未执行不得打勾。CSS zoom、deviceScaleFactor 或缩窄 viewport 都不能冒充真实浏览器放大操作。没有为本次改造自动创建新 Git tag 或发布镜像。
