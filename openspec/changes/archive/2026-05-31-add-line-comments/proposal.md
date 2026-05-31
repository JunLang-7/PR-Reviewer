## Why

当前 Review 评论以单条整体 PR Review 形式发布，所有风险项堆在一个评论 body 里。PR 作者无法直观看到每条风险对应代码的哪一行，也无法一键接受修复建议。改为行级内联评论 + suggested changes 后，风险定位更精准，修复成本更低。

## What Changes

- **Publisher 支持内联评论**：利用 GitHub PR Review API 的 `comments` 字段，在具体 diff 行上发布内联评论，取代当前纯 body 文本列表
- **Suggested Changes**：当 `Risk.FixSuggestion` 非空时，用 GitHub ` ```suggestion ` 语法包裹修复代码，作者可一键 commit
- **Fallback 机制**：diff position 计算失败的 risk 退回 body 内文本展示，不丢失任何风险项
- **Server 传递 DiffFiles**：`processPR` 将 `prCtx.DiffFiles` 传入 Publisher，用于计算 diff position
- **风险解析器提取 FixSuggestion**：`parseRiskBlock` 将 `**建议**：` 字段单独提取到 `Risk.FixSuggestion`（当前统一放入 `Description`）

## Capabilities

### New Capabilities

- `line-comments`: 在 PR diff 行上发布内联 Review 评论，支持 GitHub suggested changes 语法

### Modified Capabilities

- `risk-output-format`: `parseRiskBlock` 将 `**建议**：` 字段内容单独提取到 `Risk.FixSuggestion`，与 `Description` 分离

## Impact

| 层 | 改动 |
|---|---|
| `internal/comment/publisher.go` | 核心改造：构建 inline comments + suggested changes |
| `internal/analyzer/pipeline.go` | `parseRiskBlock` 提取 `FixSuggestion` 字段 |
| `internal/server/server.go` | `processPR` 传入 `DiffFiles` |
| `internal/server/server_test.go` | 补充 Publisher mock 验证 |
| `internal/comment/publisher_test.go` | 补充 inline comment 构建测试 |
| `internal/analyzer/pipeline_test.go` | 补充 FixSuggestion 提取测试 |
