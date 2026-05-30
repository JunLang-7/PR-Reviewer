# Design: Review Request 手动触发

## 交互流程

```
PR 页面
  ┌──────────────────────────────────────────┐
  │  Reviewers                               │
  │  ┌────────────────────────────┐          │
  │  │ 🔍 搜索 reviewer           │          │
  │  │                            │          │
  │  │ ☐ junlang                  │          │
  │  │ ☐ pr-reviewer ← 选择 App  │          │
  │  │    (bot)                   │          │
  │  └────────────────────────────┘          │
  └──────────────────────────────────────────┘
              │
              ▼
    GitHub 发送 webhook
    pull_request.review_requested
              │
              ▼
    服务器判断 requested_reviewer
    login == pr-reviewer[bot] ?
              │
         ┌────┴────┐
         │   YES    │  NO → 忽略
         └────┬────┘
              ▼
    触发 AI Review pipeline
    (force Stage 3 depth)
              │
              ▼
    发布 Review 评论到 PR
```

## 事件处理改动

### 当前代码

```go
if info.Action != "opened" && info.Action != "synchronize" {
    w.WriteHeader(http.StatusOK)
    return
}
```

### 改为

```go
if info.Action != "opened" && info.Action != "synchronize" && info.Action != "review_requested" {
    w.WriteHeader(http.StatusOK)
    return
}
```

### review_requested 事件解析

`pull_request.review_requested` 事件的 webhook payload 包含 `requested_reviewer` 字段。需要从 payload 中提取被请求的 reviewer 信息，判断是否为 App 自身的 bot 账号。

GitHub App 的 bot 账号名格式为 `<app-slug>[bot]`，例如 `pr-reviewer[bot]`。

### 需要修改的文件

| 文件 | 改动 |
|------|------|
| `internal/github/webhook.go` | `PRInfo` 新增 `RequestedReviewerLogin` 字段 |
| `internal/server/server.go` | 新增 `review_requested` 事件判断逻辑 |
| `cmd/server/main.go` | 新增 `GITHUB_APP_SLUG` 环境变量 |

## Stage 3 强制深度分析

手动请求时，`Stage3Eligible` 强制设为 `true`，无论 PR 规模如何。Reviewer 主动请求意味着需要更详细的分析报告。

## 环境变量

```env
GITHUB_APP_SLUG=pr-reviewer   # App 的 slug 名称，用于判断 reviewer 身份
```

## 风险

- **App slug 可变更**：GitHub 不支持 API 查询自身的 slug，需要用户手动配置。如果 slug 改了没更新 env，`review_requested` 事件会被静默忽略。
- **重复触发**：如果 PR 同时被 open 和 review_requested（用户在创建 PR 时就添加了 App reviewer），会触发两次 Review。解法：对已存在的 comment 做去重，或限制频率。
