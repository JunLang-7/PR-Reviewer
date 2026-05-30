## Context

当前 Risk Scan 的输出由 LLM 根据 `risk.txt` 的松散指引生成，解析器 `parseRiskBlock` 再用启发式规则兜底。问题是 LLM 输出多变：文件头格式不统一（markdown bold vs 裸链接）、冒号中英文混用、自行插入分组标题、body 字段结构散乱。

核心矛盾：prompt 给了 LLM 太多自由度，解析器被迫覆盖各种边缘情况。

## Goals / Non-Goals

**Goals:**
- 让 LLM 输出格式确定且统一，不需要启发式解析
- 解析器简化为按固定字段提取，减少 Bug 面
- 保持 Chinese + Markdown 的用户可读性

**Non-Goals:**
- 不改 `Risk` 结构体
- 不改 risk 的发送/展示管道（comment publisher, GitHub API 等）
- 不修改非 risk 的 prompt（summary 等）

## Decisions

### 1. 模板格式选择

**方案 A (选定): 固定字段模板**

```
[file:line](ref):
**问题**：<描述>
**后果**：<描述>
**建议**：<描述>
严重程度: <critical|warning|suggestion>
```

**方案 B (未选): 分隔线标记**

每条 risk 用 `---RISK---` 分隔，内部自由格式。缺点是 break 了 Markdown 读者友好性，且 prompt 中要对 LLM 解释标记语法。

**理由**: 方案 A 对人和解析器都友好，Markdown 渲染后读者可直接阅读，解析器也只用按行匹配 `**问题**：` 等前缀。

### 2. 解析策略

**当前**: `splitByFileHeader` 按 `contains(".") && endsWith(":")` 检测文件头，再 `parseRiskBlock` 跳过 markdown headers / bold 等噪音行找文件路径。

**改为**: 废弃 `splitByFileHeader` 和 `isFileHeader`，直接在原始响应中按 `[file:line](ref):` 正则分割。`parseRiskBlock` 改为按 `**问题**：` / `**后果**：` / `**建议**：` 字段提取，最后一行匹配 `严重程度: xxx`。

### 3. Prompt 强制策略

在 `risk.txt` 中加入明确的禁止清单：
- 禁止分组标题（emoji + 粗体分类）
- 禁止结尾总结段落
- 禁止在 `[file:line](ref):` 前添加修饰

采用负面示例：给出一个"❌ 错误示例"展示 LLM 常犯的问题。

## Risks / Trade-offs

- **格式过严导致 LLM 输出被截断/丢弃**: LLM 可能不严格遵守模板。→ 解析器保留空 body 过滤（返回 nil），同时 prompt 中强调"必须遵循模板"。
- **已有测试用例需要更新**: `TestParseRiskResponse_Standard` 的预期输出基于旧格式。→ 先更新测试用例再改代码（TDD）。
- **中文/英文冒号仍可能混用**: `严重程度:` vs `严重程度：`。→ 解析器同时兼容两种。
