## 1. 测试先行

- [x] 1.1 更新 `TestParseRiskResponse_Standard` 的输入为严格模板格式，更新期望输出对齐新解析逻辑
- [x] 1.2 新增 `TestParseRiskResponse_WithChineseColon` — 验证中文冒号 `严重程度：` 正常解析
- [x] 1.3 新增 `TestParseRiskResponse_WithNoiseHeaders` — 验证分组标题和总结段落被正确丢弃
- [x] 1.4 新增 `TestParseRiskResponse_MissingField` — 验证 body 缺少某个字段时仍正常提取
- [x] 1.5 新增 `TestParseRiskResponse_EmptyBody` — 验证 body 为空时该条被丢弃

## 2. Prompt 重写

- [x] 2.1 重写 `internal/analyzer/prompts/risk.txt`，加入严格模板、字段说明、禁止事项和错误示例

## 3. 解析器重写

- [x] 3.1 废弃 `splitByFileHeader` + `isFileHeader`，改为按 `[path:line](ref):` 格式正则分割
- [x] 3.2 重写 `parseRiskBlock`，改为按 `**问题**：` / `**后果**：` / `**建议**：` 字段提取
- [x] 3.3 保留 `严重程度:` 中英文冒号兼容，保留空 body 丢弃逻辑

## 4. 验证

- [x] 4.1 运行 `go test ./internal/analyzer/... -v` 确保全部测试通过
- [x] 4.2 运行 `go build ./...` 确保编译通过
