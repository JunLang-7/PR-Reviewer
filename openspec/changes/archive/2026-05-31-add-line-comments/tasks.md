## 1. 解析器提取 FixSuggestion

- [x] 1.1 编写 `parseRiskBlock` FixSuggestion 提取的单元测试（先写测试，确认红）
- [x] 1.2 修改 `parseRiskBlock` 将 `**建议**：` 内容提取到 `Risk.FixSuggestion`，`Description` 仅保留 `**问题**：` 和 `**后果**：`
- [x] 1.3 运行 `go test ./internal/analyzer/...` 确认绿

## 2. Publisher 内联评论 + Suggested Changes

- [x] 2.1 为 `buildInlineComments` 新增函数编写单元测试
- [x] 2.2 实现 `buildInlineComments`：遍历 Risks，调用 `FileLineToPosition` 计算 position，构建 `DraftPullRequestReviewComment` 列表
- [x] 2.3 实现 suggested change 包装（` ```suggestion ` 语法）
- [x] 2.4 实现 fallback 逻辑（position=0 的 risk 追加到 body）
- [x] 2.5 修改 `Publish` 签名新增 `diffFiles` 参数，在 `CreateReview` 调用中传入 `Comments`
- [x] 2.6 更新 `publisher_test.go` 适配新签名和 mock
- [x] 2.7 运行 `go test ./internal/comment/...` 确认绿

## 3. Server 适配

- [x] 3.1 更新 `server_test.go` mock 适配 `Publish` 新签名
- [x] 3.2 修改 `processPR` 传入 `prCtx.DiffFiles` 给 Publisher
- [x] 3.3 运行 `go test ./internal/server/...` 确认绿

## 4. 验证

- [x] 4.1 运行 `go test ./...` 全部通过
- [x] 4.2 运行 `go test -race ./...` 无竞态
- [x] 4.3 运行 `go vet ./...` 无警告
