## ADDED Requirements

### Requirement: FixSuggestion 字段提取

解析器 SHALL 将 `**建议**：` 字段内容单独提取到 `Risk.FixSuggestion` 字段中，与 `Description`（包含 `**问题**：` 和 `**后果**：` 内容）分离。

`Description` SHALL 仍包含 `**建议**：` 之前的所有 body 内容（即 `**问题**：` 和 `**后果**：` 部分），不含 `**建议**：`。

#### Scenario: 标准格式包含三个字段

- **WHEN** LLM 返回包含 `**问题**：`、`**后果**：`、`**建议**：` 三个字段的 risk
- **THEN** `Risk.Description` 包含 `**问题**：` 和 `**后果**：` 内容，`Risk.FixSuggestion` 包含 `**建议**：` 内容（去除 `**建议**：` 前缀）

#### Scenario: body 不包含建议字段

- **WHEN** LLM 返回的 risk 不包含 `**建议**：` 字段
- **THEN** `Risk.FixSuggestion` 为空字符串，`Risk.Description` 包含全部 body 内容

#### Scenario: 仅包含建议字段

- **WHEN** risk body 仅包含 `**建议**：` 而不含 `**问题**：` 和 `**后果**：`
- **THEN** `Risk.FixSuggestion` 包含建议内容，`Risk.Description` 不为空（从其他非标签行构建）
