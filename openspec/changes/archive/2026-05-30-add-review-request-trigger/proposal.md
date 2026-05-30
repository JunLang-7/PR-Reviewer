# Proposal: Review Request 手动触发

## Summary

支持用户在 PR 页面通过 Review Request 请求 App 进行 AI Review，作为当前自动触发模式的补充。用户只需在 PR 右侧 Reviewer 面板搜索并选择 `@pr-reviewer`，即可手动触发一次 AI 分析。

## Motivation

当前系统只在 PR 创建 (`opened`) 和新提交推送 (`synchronize`) 时自动触发。以下场景需要手动触发：

1. **PR 已完成 Review，Author 修改后想重新评估** — 不想等待 Push 触发
2. **大 PR 自动跳过深度分析** — Reviewer 想手动请求完整分析
3. **Reviewer 想针对特定 PR 请求 AI 意见** — 而非每个 PR 都自动跑

Review Request 是 GitHub 原生的"请求某人/某 App Review"操作，和请求人类同事 Review 完全一样的体验，零学习成本。

## Scope

### What we're building

- 新增 `pull_request.review_requested` webhook 事件处理
- 判断被请求的 Reviewer 是否为 App 自身的 bot 账号
- 触发同等分析流程（与 opened/synchronize 行为一致）
- 强制开启 Stage 3 深度分析（因为 Reviewer 手动请求意味着需要深度意见）

### Non-goals

- 不在 PR 评论中解析 `/review` 等指令（comment trigger）
- 不做 label 触发模式
- 不改变现有自动触发逻辑

## Key design decisions

| 决策 | 选择 | 理由 |
|------|------|------|
| 触发方式 | Review Request | GitHub 原生交互，UX 最自然 |
| 身份判断 | 比对 App bot 用户名 | webhook 事件包含 requested_reviewer |
| 分析模式 | 强制 Stage 3 深度分析 | 手动请求意味着需要更详细的分析 |
| context 复用 | 现有 pipeline 无需改动 | 输入一致，只改触发条件 |
