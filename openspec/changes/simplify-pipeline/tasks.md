# Tasks: 合并 Stage 2 和 Stage 3

## Phase 1: 删除 Stage 3

- [x] **1.1** 移除 Stage 3 相关代码
  - 删除 `runSuggestions`、`parseSuggestionResponse`、`parseSuggestionItem`、`buildSuggestionPrompt`
  - 删除 `systemPromptSuggestion`
  - 删除 `Suggestion`、`SuggestionResult` 类型
  - 从 `AnalysisResult` 中移除 `Suggestions` 字段

- [x] **1.2** 移除 `Stage3Eligible`
  - 从 `PRContext` 删除字段
  - 从 context builder 删除计算逻辑
  - 从 `PipelineInput` 删除字段

- [x] **1.3** 简化 server
  - 移除强制 Stage 3 逻辑
  - 移除 Stage 3 相关日志

## Phase 2: 增强 Stage 2

- [x] **2.1** 更新 `systemPromptRisk`
  - 修复建议以 diff 格式呈现
  - 要求包含行号用于 diff 定位

- [x] **2.2** 更新 `parseRiskResponse`
  - 确保 diff 代码块被正确提取到 `FixSuggestion` 字段

## Phase 3: 清理测试

- [x] **3.1** 更新测试
  - 删除 Stage 3 相关测试
  - 更新 pipeline 测试
  - 更新 formatter 测试
  - 全量测试通过
