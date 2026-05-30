## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Risk Scan prompt 包含 summary 上下文

Risk Scan 的 prompt SHALL 在 diff 内容之前包含 Stage 1 的摘要作为可选上下文。summary 段落 MUST 带有限定性说明（如"仅供参考，不得基于此缩小审查范围"）以对冲锚定偏差。若 summary 为空（Stage 1 失败），MUST 省略该段落而不影响其他内容。

#### Scenario: summary 正常传入

- **WHEN** Stage 1 成功返回摘要文本
- **THEN** Risk Scan prompt 包含 `## PR 变更摘要` 段落，后跟限定性说明和摘要内容

#### Scenario: summary 为空则省略

- **WHEN** Stage 1 失败（summary 为空字符串）
- **THEN** Risk Scan prompt 不包含 summary 段落，其余内容与当前一致
