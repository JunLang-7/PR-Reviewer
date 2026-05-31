## ADDED Requirements

### Requirement: 行级内联评论发布

Publisher SHALL 对每条有效 diff position 的 Risk 以行级内联评论（`DraftReviewComment`）形式发布，而非仅在 body 中以文本列表展示。

每条内联评论 MUST 包含：
- `path`: Risk.File，相对文件路径
- `position`: 通过 `FileLineToPosition` 从 diff patch 计算出的位置值
- `body`: 包含 `**<Title>**` + `Description` 的 Markdown 文本

Review body SHALL 仅包含摘要概览（Stage 1 结果）和 fallback 风险项。

#### Scenario: 有效 position 的 risk 发布为内联评论

- **WHEN** Risk.File 匹配到 `DiffFile` 且 `FileLineToPosition` 返回非零 position
- **THEN** 该 risk 以内联 `DraftReviewComment` 形式发布，不在 body 中重复出现

#### Scenario: position 为 0 的 risk fallback 到 body

- **WHEN** Risk.File 未匹配到任何 `DiffFile`，或 `FileLineToPosition` 返回 0
- **THEN** 该 risk 以格式化文本追加到 Review body 末尾，不丢失

#### Scenario: 混合发布

- **WHEN** 部分 risk 有有效 position、部分没有
- **THEN** 一次 `CreateReview` 调用同时包含 body（摘要 + fallback 文本）和 comments（有效 position 的 risk）

### Requirement: Suggested Changes 生成

当 `Risk.FixSuggestion` 非空时，内联评论 body SHALL 在其末尾追加 GitHub suggested change 代码块：

```
```suggestion
<FixSuggestion 内容>
```
```

Suggested change 代码块 SHALL 放置在 `Description` 之后，以空行分隔。

#### Scenario: FixSuggestion 非空时追加 suggestion 块

- **WHEN** Risk.FixSuggestion 非空且 risk 已发布为内联评论
- **THEN** 评论 body 末尾包含 ` ```suggestion ` 代码块，内容为 FixSuggestion

#### Scenario: FixSuggestion 为空时不追加

- **WHEN** Risk.FixSuggestion 为空字符串
- **THEN** 内联评论 body 不包含 suggestion 块
