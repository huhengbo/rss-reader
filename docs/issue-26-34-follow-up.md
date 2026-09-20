# #26 / #34：阅读交互补充与验收边界

## 本次代码范围

- 手机竖屏与短横屏使用原生 `details/summary` 收纳“显示与筛选”；桌面保持展开。搜索、清空、刷新和连接状态不隐藏。
- 桌面与手机共用同一组控件和状态；手机展开偏好在当前页面的断点往返中保留，不重新建立 WebSocket。
- 移动输入/选择控件使用 16px 字号，所有主要操作（包括 summary）保持 44px 项目触控目标；底部 safe-area 不重复累加。
- 订阅源重排保留原焦点；文章仍连接 DOM 但已被隐藏时，焦点回到该卡片的可见折叠按钮；整个来源移除时回到 main。
- 刷新使用 `aria-disabled`/`aria-busy` 和重复请求保护，不移除按钮的键盘焦点，也不在异步完成时抢回已经移走的焦点。
- “展开更多”通过 `aria-controls` 关联原生列表。

## 自动化覆盖

沿用既有 UI workflow、真实 Go fixture、Chromium/Firefox/WebKit、Pixel 5 和 iPhone 13 profiles、原生 Chromium tab zoom；不新增测试框架，不放宽资产预算。

新增/调整：原生 disclosure 的 Enter/Space/Tab、320px/字体/触控目标、断点往返和旋转后的状态、连接数不变、隐藏项和重排后的焦点、刷新请求去重与焦点、展开控件与列表关联。原生缩放继续验证 100–200%（不是 CSS zoom 或 deviceScaleFactor）。

具体运行结果以关联 PR 的 Actions 与附件为准；本文不是预先宣称测试通过的证明。

## 仍需人工执行，不由 CI 代替

#26：真实 NVDA/VoiceOver 等读屏的标题、列表、控件名称、折叠与状态播报；记录系统、浏览器、读屏版本及实测结果。原生 tab zoom 自动测试也不冒充通过浏览器 UI 操作的人工缩放验收。

#34：实体 iPhone 的刘海/Dynamic Island、Home Indicator、安全区、Safari 软键盘，以及实体 Android 的键盘、横竖屏、触摸滚动与连续操作。

保持这两项 Issue 的人工验收边界，不能仅凭自动化全绿勾选“真机/真实读屏通过”。执行清单继续见 [accessibility-checklist.md](accessibility-checklist.md) 与 [ui.md](ui.md)。
