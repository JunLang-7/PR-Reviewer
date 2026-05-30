# Proposal: Stage 3 行级优化建议

## Summary

实现 Stage 3 的 LLM 响应解析，提取文件行号级别的优化建议，并提供 diff position 计算能力。不再返回空占位。

## Motivation

当前 `parseSuggestionResponse` 是一个空占位函数（始终返回 `nil`），Stage 3 分析了但没有输出。Review 评论中缺少代码级的改进建议段落。

## Scope

### 实现

1. **A. 建议解析器**：解析 LLM 响应，提取每条建议的文件路径、行号、描述、代码片段
2. **C. Diff Position 计算**：从文件行号 + unified diff hunk 计算出 diff position

### Non-goals

- 不做 Inline Comments 发布（留到下一个 change）

## Key design decisions

| 决策 | 选择 | 理由 |
|------|------|------|
| 输出格式 | Review body 中的行级引用 + 链接 | 简单可靠，不依赖 diff position 计算精度 |
| Position 计算 | 独立工具函数 | 复用性强，后续 B 阶段直接使用 |
| 解析格式 | 类似 Risk 的结构化 Markdown | 复用已有解析模式，LLM 输出可控 |
