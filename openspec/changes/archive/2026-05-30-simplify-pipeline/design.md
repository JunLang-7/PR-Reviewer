# Design: 合并 Stage 2 + Stage 3

## 删除清单

| 删除 | 位置 | 说明 |
|------|------|------|
| `Suggestion` struct | `types.go` | 不再需要 |
| `SuggestionResult` struct | `types.go` | 不再需要 |
| `runSuggestions()` | `pipeline.go` | 不再需要 |
| `parseSuggestionResponse()` | `pipeline.go` | 不再需要 |
| `parseSuggestionItem()` | `pipeline.go` | 不再需要 |
| `buildSuggestionPrompt()` | `pipeline.go` | 不再需要 |
| `systemPromptSuggestion` | `pipeline.go` | 不再需要 |
| `Stage3Eligible` 字段 | `pipeline.go` + `context/builder.go` | 不再需要 |
| `stage3` 强制逻辑 | `server.go` | 不再需要 |
| Stage 3 测试 | `pipeline_test.go` | 不再需要 |
| Stage 3 日志 | `server.go` | 简化 |

## 保留

| 保留 | 说明 |
|------|------|
| `FileLineToPosition()` | 工具函数，后续 inline comments 可能用到 |

## Stage 2 Prompt 增强

### 当前

```
建议修复：使用参数化查询
```

### 改为

```
```diff
- query := "SELECT * FROM users WHERE id = " + userID
+ query := "SELECT * FROM users WHERE id = ?"
+ db.Query(query, userID)
```
```

Prompt 改动：在 `systemPromptRisk` 中增加一段：

```
修复建议请使用 diff 格式展示代码变更：
```
diff
- 原代码
+ 建议修改
```
```

## Pipeline 简化

### Before

```go
func (p *Pipeline) Run(ctx, input) (*AnalysisResult, error) {
    summary, _ := p.runSummary(ctx, input)      // Stage 1
    risks, _ := p.runRiskScan(ctx, input)       // Stage 2
    if input.Stage3Eligible {
        suggestions, _ := p.runSuggestions(...) // Stage 3 (移除)
    }
}
```

### After

```go
func (p *Pipeline) Run(ctx, input) (*AnalysisResult, error) {
    summary, _ := p.runSummary(ctx, input)      // Stage 1
    risks, _ := p.runRiskScan(ctx, input)       // Stage 2
}
```
