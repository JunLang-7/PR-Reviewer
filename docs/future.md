# 未来扩展

本文档记录待实现的扩展方向及其大致思路。所有方向均可在不破坏现有接口的前提下渐进实现。

## 短期（接口已预留，改动成本低）

### 行级精准评论 + Suggested Changes

当前 Review 评论作为整体 PR Review 发布。可扩展为在具体 diff 行上发布 inline comment，并附带 `suggested_change` 代码补丁，让作者一键接受修复。

`analyzer.Risk` 已有 `File`/`Line`/`FixSuggestion` 字段。Publisher 侧扩展现有 `PRReviewClient` 接口（或新增 `CreateReviewComment` 调用）即可，Pipeline 不变。

### 多平台适配（GitLab / Bitbucket）

`PipelineInput` 与来源完全解耦（只有 `Diff` + `FileContents`）。新写一个 GitLab MR 的 Builder 实现即可，下游 Pipeline 零改动。同理，输出侧实现一个 GitLab MR Note API 的 `PRReviewClient`。

### 仓库级配置（`.prreviewer.yml`）

允许目标仓库在根目录放置配置文件，自定义以下行为：

- 忽略特定文件/目录（如 `*.pb.go`、`vendor/`）
- 调整风险阈值（如 critical 以下不报）
- 自定义 focus area 开关（性能、安全、并发等）

### IM 通知（Slack / 飞书 / 钉钉）

`PRReviewClient` 接口仅对接 GitHub Review API。新增通知渠道实现，当扫描到 critical 级别风险时向 IM 群推送摘要。

---

## 中期（需要新增模块或 Pipeline 阶段）

### 测试覆盖率感知

PR 变更了业务逻辑但没有新增或修改测试文件时，在 Risk 中追加一条 suggestion 级别提醒。上下文获取阶段已能拿到变更文件列表，只需在 Pipeline 中增加判断逻辑。

### Prompt A/B 实验

按 `PRNumber % 2` 路由到不同 prompt 文件，对比两组 prompt 的 Review 质量（评论采纳率、风险检出率），数据驱动迭代 prompt。需要额外的指标采集管道。

### 依赖漏洞联动

当 PR 变更了 `go.mod`/`package.json`/`requirements.txt` 等依赖声明文件时，调用公有漏洞数据库 API（如 osv.dev），将已知漏洞信息注入 Stage 2 上下文，辅助风险判断。

---

## 长期（架构演进）

### 反馈闭环与自优化

跟踪 Review 评论的后续状态 — 被标记 resolved、被 replied、被忽略 — 将反馈数据用于调整 prompt 或 prompt routing，使 Review 质量随使用量增长而提升。

### 跨 PR 模式检测

维护轻量级的仓库级别知识库（按文件/模块聚合历史风险）。当同一模块反复出现同类问题时，Review 中追加提示，例如"该模块近 30 天已有 3 个并发相关 PR 被标记为 critical"。

### 自动修复 PR

对机械性、高置信度的 fix suggestion（如"缺少 nil 检查"、"错误未 wrap"），自动生成 fix commit 或修复 PR 供作者直接合并。这是架构设计文档中提到的 `risk → 自动生成 fix commit` 的具体落地。

---

## 明确不做

以下方向在设计文档中已有定论，不在扩展范围内：

- **多轮对话**：PR 是静态快照，不做对话式 Review，省去状态管理复杂度
- **Agent / Tool Calling**：不让 LLM 自行调用 GitHub API。上下文由代码预先获取，LLM 只做分析，避免幻觉导致额外延迟和不稳定
- **实时流式展示**：streaming 仅用于绕过 API 超时限制。结果完整收集后再一次性发布，保证 Review 评论的原子性
