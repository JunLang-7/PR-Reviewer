# Proposal: AI PR Reviewer

## Summary

构建一个 GitHub App，在 PR 提交后自动触发 AI Review，将分析结果以评论形式发布到 PR 上，帮助 Reviewer 快速理解变更、识别风险、获得改进建议。

## Motivation

代码评审中的核心痛点不是"看不懂代码"，而是：

1. **认知负荷高** — Reviewer 需要在有限时间内理解变更意图、发现隐蔽问题、判断跨文件影响
2. **重复性疲劳** — 风格检查、安全模式匹配等机械工作消耗注意力
3. **知识盲区** — 不熟悉的模块或技术栈导致 Review 深度不均匀

AI 可以在变更总结、风险模式识别、代码建议三个层面显著降低认知负荷，让 Reviewer 把注意力集中在需要人类判断的关键决策上。

## Scope

### What we're building

- GitHub App 接收 webhook（`pull_request.opened`、`pull_request.synchronize`）
- 自动获取 PR 增量 diff 和变更文件全文作为分析上下文
- 三阶段 AI 分析流水线：变更总结 → 风险识别 → Review 建议
- 结构化评论输出（Critical / Warning / Suggestion 分级，含代码定位和建议修复）
- PR 更新后追加新评论（保留历史）

### Non-goals (v1)

- 不建立反馈闭环（👍/👎 收集）
- 不做全量 PR diff 重分析（每次只分析增量 commit）
- 不做持久化存储（知识库、历史 Review 模式）
- 不接入多模型（仅 Claude API）
- 不做 Stage 3 自动触发（手动标签触发暂不做）

## Key design decisions

| 决策 | 选择 | 理由 |
|------|------|------|
| 部署方式 | 本地 + ngrok | demo 阶段零成本、易调试 |
| 语言 | Go | 团队熟悉，单二进制部署 |
| 上下文策略 | L0 (diff) + L1 (变更文件全文) | 平衡分析深度与 token 成本 |
| 触发范围 | opened + synchronize | 覆盖 PR 生命周期主要节点 |
| Review 范围 | 增量 commit diff | 避免重复分析已 Review 内容 |
| 模型路由 | Haiku(总结) + Sonnet(风险) | 按任务复杂度匹配模型，控制成本 |

## Risks

- **分析延迟** — Claude API 调用可能 30-120s，需流式或异步处理
- **token 成本** — 大 PR 可能消耗较多，需设置文件数/大小上限
- **误报** — AI 判断可能不准确，需置信度标注和严重度分级让 Reviewer 自行判断
