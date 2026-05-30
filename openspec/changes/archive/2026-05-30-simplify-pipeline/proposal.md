# Proposal: 合并 Stage 2 和 Stage 3，简化流水线

## Summary

砍掉 Stage 3 独立 API 调用，将代码级优化建议融入 Stage 2 的风险项中。每次 Review 从 2 次 LLM 调用减少到 1 次。

## Motivation

1. Stage 3 几乎总是返回空（LLM 在 Stage 2 中已经充分表达了改进意见）
2. Stage 2 的 `建议修复` 已经是优化建议，再单独跑一次 Stage 3 是冗余
3. 40+ 文件的大 PR 省下一次 Sonnet API 调用，降低成本

## Scope

### 改动

1. **增强 Stage 2 prompt**：修复建议以 unified diff 格式呈现
2. **移除 Stage 3**：删除 `runSuggestions`、`parseSuggestionResponse`、`Suggestion` 类型
3. **移除 `Stage3Eligible`**：context builder 不再计算此字段
4. **简化 server**：移除强制 Stage 3 逻辑
5. **Diff position 保留**（工具函数本身有用）

### Non-goals

- 不改动 Risk 解析器
- 不改变 formatter 结构

## 预期效果

```
改动前: PR → Stage 1 (Haiku) → Stage 2 (Sonnet) → Stage 3 (Sonnet) → 评论
        3 次 API 调用

改动后: PR → Stage 1 (Haiku) → Stage 2 (Sonnet) → 评论
        2 次 API 调用，节省 1/3 成本
```

Risk 输出示例：

```
### Critical
- **SQL 注入** `db/user.go:42`
  直接拼接查询字符串
  ```diff
  - query := "SELECT * FROM users WHERE id = " + userID
  + query := "SELECT * FROM users WHERE id = ?"
  + db.Query(query, userID)
  ```
```
