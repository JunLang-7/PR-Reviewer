## ADDED Requirements

### Requirement: LLM 输出固定字段模板

LLM 生成的每条 risk 评论 MUST 严格遵循以下模板：

```
[relative/file/path.go:123](ref):
**问题**：<一句话描述>
**后果**：<具体触发条件和影响>
**建议**：<可操作的修复方向>
严重程度: <critical|warning|suggestion>
```

字段说明：
- `[file:line](ref):` — 相对文件路径 + 行号，英文冒号结尾
- `**问题**：` — 一行，描述当前代码的行为或缺陷
- `**后果**：` — 一行或多行，描述触发条件和具体影响
- `**建议**：` — 一行或多行，描述修复方向
- `严重程度:` — 取值 `critical` / `warning` / `suggestion`

#### Scenario: 标准 risk 正确解析

- **WHEN** LLM 返回严格按照模板格式的 risk 文本
- **THEN** 解析器正确提取 file、severity、description 各字段

#### Scenario: severity 行使用中文冒号

- **WHEN** severity 行为 `严重程度：critical`（中文冒号）
- **THEN** 解析器同样识别 severity 为 `critical`

#### Scenario: 多条 risk 有分隔空行

- **WHEN** 两条 risk 之间有一个或多个空行
- **THEN** 解析器将它们分开为独立的 Risk 对象

#### Scenario: body 为空则丢弃该条

- **WHEN** 文件头行之后没有 `**问题**` 等字段内容（body 为空）
- **THEN** 该条被丢弃，不出现在结果中

### Requirement: LLM 禁止输出额外结构

LLM MUST NOT 输出模板之外的结构化元素，包括但不限于：

- 分组标题（如"🔴 严重逻辑缺陷"、"⚠️ 显著问题"）
- 总结段落（如"本次变更引入了..."）
- 在 `[file:line](ref):` 前添加 markdown bold (`**`) 或列表符号 (`-`)
- 在末尾添加非 risk 的内容

#### Scenario: 包含分组标题的响应被正确解析

- **WHEN** LLM 在 risk 之间插入了分组标题行（如 `### 🔴 严重逻辑缺陷`）
- **THEN** 解析器将分组标题视为噪音丢弃，仍正确提取后续 risk

#### Scenario: 末尾有总结段落

- **WHEN** 最后一条 risk 之后有非模板格式的总结段落
- **THEN** 总结段落被忽略，不出现在结果中

### Requirement: 解析器按结构化字段提取

解析器 SHALL 按以下方式提取 risk：

- 按 `[*.ext:NN](ref):` 模式分割 risk 块
- 从 `**问题**：` / `**后果**：` / `**建议**：` 三个字段提取 body
- 从最后一行 `严重程度: xxx` 提取 severity
- 文件路径和行号从文件头行解析，丢弃 ref 链接格式差异
- 空 body 或无法解析的块被丢弃

#### Scenario: body 缺少某个字段

- **WHEN** 某条 risk 的 `**后果**：` 字段缺失（但 `**问题**：` 和 `**建议**：` 存在）
- **THEN** 解析器正常提取存在的字段，description 为已有内容拼接

### Requirement: Risk Scan prompt 包含 summary 上下文

Risk Scan 的 prompt SHALL 在 diff 内容之前包含 Stage 1 的摘要作为可选上下文。summary 段落 MUST 带有限定性说明（如"仅供参考，不得基于此缩小审查范围"）以对冲锚定偏差。若 summary 为空（Stage 1 失败），MUST 省略该段落而不影响其他内容。

#### Scenario: summary 正常传入

- **WHEN** Stage 1 成功返回摘要文本
- **THEN** Risk Scan prompt 包含 `## PR 变更摘要` 段落，后跟限定性说明和摘要内容

#### Scenario: summary 为空则省略

- **WHEN** Stage 1 失败（summary 为空字符串）
- **THEN** Risk Scan prompt 不包含 summary 段落，其余内容与当前一致

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
