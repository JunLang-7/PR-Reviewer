## Why

Risk Scan 输出格式因 prompt 约束过松而高度不一致：LLM 自由发挥导致文件头格式不统一（bold markdown vs 裸链接）、中文/英文冒号混用、自行插入分组标题（如"🔴 严重逻辑缺陷"），以及 body 内部字段结构散乱。这直接影响下游解析器的准确性和最终展示的专业性。现在修复 -- 刚发现此问题，趁 risk 模块还在早期迭代，格式还未扩散到其他地方。

## What Changes

- 重写 `risk.txt` prompt，提供严格输出模板并明确禁止行为
- 重写 `parseRiskBlock` 解析器，按结构化字段提取，移除启发式兜底逻辑
- 补充对应测试用例覆盖新格式

## Capabilities

### New Capabilities

- `risk-output-format`: Risk Scan 输出格式规范 -- 定义 LLM 生成 risk 评论的统一模板，以及解析器应遵循的解析契约

### Modified Capabilities

_无_（risk 模块没有已存在的 spec）

## Impact

| 区域 | 影响 |
|---|---|
| `internal/analyzer/prompts/risk.txt` | 重写 prompt 为严格模板 |
| `internal/analyzer/pipeline.go` | 重写 `isFileHeader` / `parseRiskBlock` |
| `internal/analyzer/pipeline_test.go` | 新增新格式测试用例、校准旧用例 |

## Non-goals

- 不改变 `Risk` 结构体定义
- 不改变 LLM 调用方式（streaming 改动已完成）
- 不改变 risk 的输出载体（comment publisher 等）
