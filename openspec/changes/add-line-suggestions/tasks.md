# Tasks: Stage 3 行级优化建议

## Phase A: 建议解析器

- [x] **A.1** 更新 Stage 3 prompt，要求结构化输出
  - 更新 `systemPromptSuggestion` 和 `buildSuggestionPrompt`
  - 要求 LLM 按文件+行号格式输出建议

- [x] **A.2** 实现 `parseSuggestionResponse`
  - 解析 `### <文件>` 区块
  - 解析 `- **<行号>** <描述>` 和代码块
  - 产出 `[]Suggestion`

- [x] **A.3** 解析器测试
  - 测试标准格式解析
  - 测试空响应/无建议情况
  - 测试多文件多建议场景

## Phase C: Diff Position 计算

- [x] **C.1** 实现 `FileLineToPosition`
  - 解析 unified diff hunk 头部
  - 计算目标行号的 diff position
  - 不在此 hunk 时返回 0

- [x] **C.2** Diff position 测试
  - 测试标准 hunk 解析
  - 测试边界条件（第一行、最后一行、删除行）
  - 测试多 hunk diff

## Phase 集成

- [ ] **集成** 确保 formatter 正确展示建议
  - 每条建议显示文件:行号链接、描述、代码片段
  - 端到端测试验证 Stage 3 输出出现在 Review 中
