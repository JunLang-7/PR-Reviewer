## Context

当前 `Publisher.Publish` 通过 `CreateReview` API 发布一条整体 PR Review，所有风险项以 Markdown 列表形式写在 `Body` 中。GitHub PR Review API 同时支持 `Comments` 字段，可在具体 diff 行上发布内联评论并附带 `suggested_change` 代码块。

`FileLineToPosition` 函数已存在，可将文件行号转换为 diff 内的 position 值。`DiffFile` 结构已包含 `Path` 和 `Patch`。

## Goals / Non-Goals

**Goals:**
- 有有效 diff position 的 risk 以行级内联评论形式发布，附带 suggested change（若 `FixSuggestion` 非空）
- diff position 计算失败的 risk 退回 body 文本展示，不丢失
- `parseRiskBlock` 将 `**建议**：` 字段单独提取到 `Risk.FixSuggestion`

**Non-Goals:**
- 不修改 LLM prompt
- 不修改 Pipeline 编排逻辑
- 不修改 Context Builder

## Decisions

### 决策 1：Publisher.Publish 签名新增 `diffFiles` 参数

`Publish(ctx, owner, repo, prNumber, result)` → `Publish(ctx, owner, repo, prNumber, result, diffFiles)`

- **理由**：Publisher 需要 `DiffFile.Patch` 来计算 diff position。
- **替代方案**：在 `AnalysisResult` 中附带 `DiffFiles`。不选，因为 `AnalysisResult` 应与来源无关。

### 决策 2：使用 legacy `position` 而非 comfort-fade `line`+`side`

最初尝试了 comfort-fade API（`side: "RIGHT"` + `line: N`），但 GitHub 对该 API 要求行号必须在 diff 可见范围内，LLM 输出的行号可能对应文件全文（非仅 diff 上下文），导致 "Line could not be resolved"。

改用 legacy `position`：通过 `FileLineToPosition` 将文件行号转为 diff 内 position 值。该值是 diff 文件中从第一个 `@@` 头开始计数的行索引，GitHub 能可靠解析。

### 决策 3：修复 `FileLineToPosition` 的空行计数

原实现跳过空行和 `\ No newline` 标记（`if line == "" || strings.HasPrefix(line, "\\") { continue }`），但 GitHub 将 diff 中每一行（除 `@@` 头外）都计入 position。移除 skip 逻辑后 position 与 GitHub 一致。

### 决策 4：内联评论 + body 评论混合发布

一次 `CreateReview` 调用同时包含 `Body`（摘要概览 + fallback risk）和 `Comments`（有效 position 的行级评论），保证原子性。

### 决策 5：Suggested change 用 ` ```suggestion ` 语法

将 `Risk.FixSuggestion` 包裹在 ` ```suggestion ` 代码块中，GitHub 原生渲染 "Commit suggestion" 按钮。

## Risks / Trade-offs

- **LLM 行号不准** → position 计算返回 0 时 fallback 到 body，不丢失信息
- **FixSuggestion 格式不匹配** → 描述性文字而非代码时 suggestion 按钮不渲染，但仍以代码块展示
