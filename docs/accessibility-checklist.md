# 可访问性与放大阅读验收

本清单配合 [阅读界面维护指南](ui.md) 与 Issue #26 使用。仅记录实际执行结果，不把自动化测试通过等同 WCAG 合规认证。UI 代码位于 main 不代表已发布镜像包含这些改动。

## 可重复执行的自动检查

沿用项目锁定的 Node / Playwright 测试依赖，不新增运行时依赖：

```text
npm ci --ignore-scripts
npm test
npm run check:assets
npx playwright install chromium firefox webkit
npm run test:e2e
```

Linux 可能需要使用 `npx playwright install --with-deps chromium firefox webkit` 安装浏览器系统依赖。

| 范围 | 入口 | 检查边界 |
| --- | --- | --- |
| 原生语义与键盘 | tests/browser/accessibility.spec.js | 中文语言、main/h1/h2/列表、控件可访问名称、Tab 顺序、删除当前焦点所在源后恢复焦点 |
| 模块下载失败 | tests/browser/fallback.spec.js | 分别阻断 app.js / model.js，确保第 9 篇之后的静态文章不被隐藏；不同于完全关闭 JavaScript |
| 连接、状态、皮肤与布局 | tests/browser/reader.spec.js、lifecycle.spec.js | 三浏览器、四皮肤、两种明暗、窄屏、搜索、展开、连接恢复与更新确认 |
| 实际标签页缩放 | tests/zoom/native-zoom.spec.js | 四皮肤 × 两种明暗 × 100/125/150/175/200%，以及 200% 下的关键阅读操作 |

### 原生缩放测试的机制与证据

```text
npm run test:e2e -- --project=chromium-zoom
```

该测试启动独立的临时 Chromium profile，通过测试专用扩展调用 `chrome.tabs.setZoom`，选择 automatic / per-tab 模式，由浏览器自己完成页面缩放；再通过 `chrome.tabs.getZoom` 回读倍率。没有使用 CSS zoom、deviceScaleFactor、pinch 缩放或更改 viewport 来冒充浏览器缩放。

同时检查浏览器外窗口宽度保持不变、CSS 视口宽度随倍率变化、devicePixelRatio 对应变化、visualViewport.scale 仍为 1、根元素没有 CSS zoom，防止只设置参数而实际未生效。40 组尺寸/倍率记录输出为 `native-zoom-metrics` 附件，完成标志必须为 true。200% 下保留八张皮肤截图、ARIA 结构记录，失败时保留独立 trace。

扩展只允许读取本地 127.0.0.1 的标签页信息，测试代码进一步限制为 `http://127.0.0.1:4173`。没有 content script、页面消息桥接、第三方网络调用或生产安装；不修改用户现有浏览器配置。测试结束关闭临时 profile。

CI 在 Chromium job 一起运行 `chromium` 与 `chromium-zoom` 项目，Firefox / WebKit 继续执行通用回归。这是自动化的真实 Chromium 页面缩放，不是 Firefox / Safari 缩放结论，也不是人工点击浏览器菜单的验收记录。

本地有桌面环境时可加 `--headed` 观察测试。也可以运行 `npm run test:ui-server`，打开固定示例数据页面人工验收；不要将 fixture 数据误认为用户的真实订阅。

## 人工验收记录模板

每次走查记录：代码 commit、操作系统、浏览器及版本、辅助技术及版本、窗口尺寸、系统显示倍率、浏览器缩放倍率、日期、执行人。截图使用固定示例数据，不包含私有订阅或密钥。

| 项目 | 实际步骤与预期 | 结果 / 证据 |
| --- | --- | --- |
| 浏览器菜单放大 | 通过浏览器菜单逐步放大到 200%；四皮肤浅深色均无内容截断、覆盖或意外横向滚动 | 未执行，执行后填写 |
| 长标题与控件 | 窄屏和 200% 下搜索、筛选、皮肤、明暗、密度、折叠、更多、更新确认都可见可用 | 未执行，执行后填写 |
| 标题与地标 | 用 NVDA / VoiceOver 等的地标和标题导航到主区及各源；顺序和名称正确 | 未执行，执行后填写 |
| 列表阅读 | 读出文章列表及链接；隐藏的折叠内容不进入浏览顺序，展开后可读 | 未执行，执行后填写 |
| 控件与焦点 | 仅用键盘完成搜索、清空、源筛选、换肤、折叠、展开；状态改变后焦点不丢失、不被遮挡 | 未执行，执行后填写 |
| 原文跳转 | 从文章链接打开原文后返回阅读页，阅读位置和焦点合理；不使用真实私有地址作为证据 | 未执行，执行后填写 |
| 状态播报 | 断线、恢复、无结果、有新文章时播报简短状态，不反复读整份文章列表，不抢走焦点 | 未执行，执行后填写 |
| 减少动态效果 | 系统开启减少动态效果后，交互仍清楚且无非必要位移或闪烁 | 未执行，执行后填写 |

在 #26 评论贴实际记录及未通过项。自动化 ARIA snapshot 只能提供语义结构证据，不能证明读屏的发音、浏览模式、live region 播报时机和实际使用体验。未完成的人工项保持未勾选，不为关闭 Issue 改写验收标准。

## 参考

- [WCAG 2.2 — Resize Text](https://www.w3.org/WAI/WCAG22/Understanding/resize-text.html)
- [Chrome Tabs API — Zoom settings / setZoom / getZoom](https://developer.chrome.com/docs/extensions/reference/api/tabs)
- [Playwright — Chrome extensions](https://playwright.dev/docs/chrome-extensions)
