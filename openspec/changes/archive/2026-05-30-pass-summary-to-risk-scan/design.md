## Context

当前 `Run` 分别调用 `runSummary` 和 `runRiskScan`，二者共享 `PipelineInput`（diff + 文件内容）但 risk scan 不知道 Stage 1 的总结结果。变动范围仅限于 `pipeline.go` 和 `risk.txt`，不涉及外部依赖。

## Goals / Non-Goals

**Goals:**
- Stage 2 的 risk prompt 包含 Stage 1 的 summary 作为参考上下文
- Stage 1 失败时 Stage 2 不受影响（空 summary 时省略上下文段落）

**Non-Goals:**
- 不改变并行/串行调用方式（当前已串行，因为 `Run` 中 result.Summary 先赋值）
- 不改变数据模型或 API

## Decisions

### 1. 传递方式

`buildRiskPrompt` 接受可选的 `summary string` 参数。这是最小侵入的方案：不需要新类型，不需要修改 `PipelineInput`，不需要修改接口。

### 2. Prompt 结构

summary 上下文放在 risk prompt 的最前面，位于 diff 内容之前：

```
请分析以下 PR 变更中的潜在风险：

## PR 变更摘要（仅供参考，不得基于此缩小审查范围）
<summary 内容>

## 变更文件 Diff
...
```

关键措辞"仅供参考，不得基于此缩小审查范围"用于对冲锚定偏差。

### 3. 失败兜底

`Run` 中先执行 `runSummary`，无论成功与否都将 summary 文本传入 `runRiskScan`。如果 Stage 1 出错，summary 为空字符串，`buildRiskPrompt` 直接省略整个 summary 段落。

## Risks / Trade-offs

- **锚定偏差**：LLM 可能被 summary 误导，忽略 summary 未覆盖的风险。→ prompt 中强调"仅供参考，不得缩小审查范围"，且 summary 放在首位而非末尾（先看意图再看 diff 可减少 skimming）
- **流水线延迟**：Stage 1 → Stage 2 串行化。→ 当前实现已经是串行的（`Run` 中顺序调用），实际无新增延迟
