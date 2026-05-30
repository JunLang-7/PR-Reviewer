## 1. 测试先行

- [x] 1.1 新增 `TestBuildRiskPrompt_WithSummary` — 验证 summary 传入后 prompt 包含摘要段落
- [x] 1.2 新增 `TestBuildRiskPrompt_EmptySummary` — 验证空 summary 不产生摘要段落

## 2. 核心实现

- [x] 2.1 `buildRiskPrompt` 新增 `summary string` 参数，非空时注入 `## PR 变更摘要` 段落（含限定性说明）
- [x] 2.2 `runRiskScan` 新增 `summary string` 参数，透传给 `buildRiskPrompt`
- [x] 2.3 `Run` 中将 summary 结果传入 `runRiskScan`

## 3. Prompt 更新

- [x] 3.1 `risk.txt` 末尾添加摘要上下文说明，告知 LLM summary 仅作参考

## 4. 验证

- [x] 4.1 `go test ./internal/analyzer/... -v` 全部通过
- [x] 4.2 `go build ./...` 编译通过
