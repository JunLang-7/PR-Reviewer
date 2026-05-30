# Proposal: 重写 Risk 解析器匹配新 Prompt 格式

## Summary

Prompt 改为 Copilot 风格后，旧解析器（按 `### Critical` / `- **标题**` 分段）不再匹配。重写 `parseRiskResponse` 以解析 `In file:` + `----` 分隔的新格式。

## Motivation

新 prompt 格式更简洁、更接近 Copilot 的输出风格，但解析器不兼容。

旧格式：
```
### Critical
- **标题** `文件:行号` 置信度: high
  描述...
```

新格式：
```
In file.go:
> 原始代码
评论内容...
----
```

## Scope

### 改解析器

1. 按 `In <file>:` 分段识别文件路径
2. 按 `----` 分隔各条风险
3. 从 `> ` 首行提取代码引用
4. 其余文本作为描述
5. 默认 severity = "warning"，缺失行号时为 0

### Non-goals

- 不改 prompt 文件
- 不改 formatter

## Key design decisions

| 决策 | 选择 | 理由 |
|------|------|------|
| 分段方式 | `----` 分隔符 | prompt 明确要求此格式 |
| 严重度 | 统一为 warning | 新格式不再有三段式分类 |
| 行号 | 从 `> @@` hunk 头或文本中提取 | 尽可能保留行级定位 |
