# Tasks: 重写 Risk 解析器

## Phase 1: 新解析器

- [x] **1.1** 实现 `parseRiskResponse` 新逻辑
  - 按 `----` 分隔 block
  - 从 `In <file>:` 提取文件路径
  - 从 `> ` 行提取代码块
  - 剩余文本作为描述

- [x] **1.2** 删除旧解析函数
  - 删除 `extractSection`、`isEmpty`、`parseRiskItems`、`parseBulletRisk`
  - 删除 `extractTitle`、`extractConfidence`、`extractFile`、`extractLine`
  - 保留 `FileLineToPosition`（diff position 工具函数）

## Phase 2: 更新测试

- [x] **2.1** 更新 pipeline_test.go
  - 重写 `TestParseRiskResponse_*` 测试用例
  - 覆盖：空响应、单条评论、多条评论、含代码块
