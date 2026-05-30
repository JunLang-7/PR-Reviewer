# Tasks: Review Request 手动触发

## Phase 1: Webhook 事件扩展

- [x] **1.1** PRInfo 扩展 requested_reviewer 字段
  - 新增 `RequestedReviewerLogin` 字段
  - 解析 `pull_request.review_requested` 事件 payload 中的 `requested_reviewer.login`

- [x] **1.2** 新增 webhook 事件解析测试
  - 测试 `review_requested` 事件解析
  - 测试未匹配到 reviewer 的情况
  - 测试非 App bot 的 reviewer（应忽略）

## Phase 2: Server 触发逻辑

- [x] **2.1** 新增 review_requested 触发判断
  - server 中 `handleWebhook` 添加 `review_requested` action
  - 比对 `RequestedReviewerLogin` 与 `GITHUB_APP_SLUG` 拼接 `[bot]` 后缀
  - 不匹配时跳过处理

- [x] **2.2** 强制 Stage 3 深度分析
  - `review_requested` 触发时 `Stage3Eligible` 强制为 true
  - 可选：在评论中标明"手动请求的深度分析"

- [x] **2.3** Server 测试
  - 测试 `review_requested` 事件触发 pipeline
  - 测试非 bot reviewer 被忽略
  - 测试 Stage 3 强制开启

## Phase 3: 配置与集成

- [x] **3.1** 新增 GITHUB_APP_SLUG 环境变量
  - config 中添加 `GitHubAppSlug` 字段
  - `.env.example` 更新

- [x] **3.2** main.go 集成
  - 传入 App slug 到 server

- [ ] **3.3** 端到端测试
  - 在 PR 页面 Request App Review，验证 Review 出现
