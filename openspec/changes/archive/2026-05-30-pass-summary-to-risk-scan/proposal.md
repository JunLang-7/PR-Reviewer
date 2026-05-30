## Why

Stage 2 (risk scan) 目前独立运行，不知道 Stage 1 总结出的 PR 意图和影响范围。将 summary 作为上下文传入 Stage 2 可以：让风险扫描理解 PR 的整体目的，区分"故意的设计选择"和"无意的风险"，减少误报，聚焦真正值得关注的问题。

## What Changes

- `Run` 中将 summary 结果传入 `runRiskScan`
- `buildRiskPrompt` 接受可选的 summary 参数，注入到 risk scan prompt 中作为参考上下文
- prompt 措辞强调 summary 仅供参考，不得基于此缩小审查范围（对冲锚定偏差）
- 若 Stage 1 失败（summary 为空），Stage 2 仍独立运行，不阻塞

## Capabilities

### New Capabilities

_无_（行为变更，不新增独立能力模块）

### Modified Capabilities

- `risk-output-format`: prompt 模板新增可选的 summary 上下文段落，要求 LLM 将 summary 作为参考但不缩小审查范围

## Impact

| 区域 | 影响 |
|---|---|
| `internal/analyzer/pipeline.go` | `Run`、`runRiskScan`、`buildRiskPrompt` 签名调整 |
| `internal/analyzer/prompts/risk.txt` | 新增 summary 上下文占位说明 |
| `internal/analyzer/pipeline_test.go` | 更新 mock 和 prompt 构建测试 |

## Non-goals

- 不将 summary 作为强制依赖，Stage 1 失败时 Stage 2 仍独立运行
- 不改变 Stage 1 的 prompt 或逻辑
- 不改变 Risk 解析器
