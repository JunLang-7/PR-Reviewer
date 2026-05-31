# 设计思路

本文档说明系统在模型选择、上下文获取及未来扩展方向上的核心设计决策。

## 模型选择：双模型分层策略

核心思路是 **用合适的模型做合适的事**。

```
Stage 1 (Summary)  → modelFast  (轻量、快、成本低)
Stage 2 (Risk Scan) → modelPower (主力、深度分析)
```

### 为什么分两阶段

- **成本/延迟权衡**：Stage 1 只做"概括 PR 意图"，不需要深度推理，用 fast model 即可。Stage 2 需要识别安全漏洞、竞态条件等复杂问题，投入更贵的 power model
- **信息增益**：Stage 1 的摘要作为上下文传入 Stage 2，让风险扫描先理解 PR 整体意图再逐行审视，减少误报
- **容错设计**：Stage 1 失败不阻塞 Stage 2 — summary 为空时自动省略摘要段落，Stage 2 独立运行

### 模型可配置

`LLM_MODEL_FAST` / `LLM_MODEL_POWERFUL` 环境变量分别控制两个阶段的模型，支持按预算和场景灵活切换。

| 部署场景 | modelFast | modelPower |
|---|---|---|
| 低成本 | claude-haiku-4-5 | claude-sonnet-4-6 |
| 高质量 | claude-sonnet-4-6 | claude-opus-4-7 |
| 国产替代 | deepseek-v4-flash | deepseek-v4-pro |

### Provider 无关

`LLMClient` 只定义 `Chat(ctx, model, system, user) → string` 一个方法。当前实现基于 Anthropic SDK，但 `LLM_BASE_URL` 指向任何 Anthropic 兼容端点即可切换，无需改动 pipeline 代码。

```go
// LLMClient abstracts the Claude API for testing.
type LLMClient interface {
    Chat(ctx context.Context, model, systemPrompt, userMessage string) (string, error)
}
```

### Prompt 外置

System prompt 通过 `//go:embed prompts/*.txt` 编译时嵌入，运行时零文件依赖。修改 prompt 只需编辑 `.txt` 重新编译，不需要改 Go 代码。

## 上下文获取：分层截断 + 接口隔离

核心思路是 **拿足够的上下文让 AI 做出判断，但不超出 token 预算和延迟容忍度**。

### 数据流

```
GitHub Compare API (增量 diff)
    │
    ├─ L0: Diff 文本 → Risk Scan 截断至 5000 字符
    │                  Summary  截断至 3000 字符
    │
    └─ L1: 文件完整内容 → 最多 20 个文件
             │             跳过二进制文件
             │             跳过 >500KB 文件
             │
             └─ 超限文件标记 Skipped=true，不阻塞流程
```

### 关键设计决策

**增量 diff 而非全量**：`CompareCommits(base, head)` 只拿变更部分。对于常规 PR（几十个文件、几百行改动），token 效率远高于传整份 repo。

**`PipelineInput` 与来源解耦**：只有两个字段：

```go
type PipelineInput struct {
    Diff         string
    FileContents map[string]string
}
```

不依赖任何 GitHub 类型。这意味着：
- 接入 GitLab / Bitbucket 只需写一个新的 Builder，下游 pipeline 不变
- 本地 CLI 模式下，`os.ReadFile` 读 diff 文件即可直接跑 pipeline
- 测试时 mock LLM 即可，不需要真实 API

**Builder 可测试**：`RepositoryClient` 接口抽象了 GitHub API：

```go
type RepositoryClient interface {
    CompareCommits(...) (*github.CommitsComparison, *github.Response, error)
    GetContents(...) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)
}
```

**渐进降级**：文件太多→截断，文件太大→跳过，API 调用失败→`Error` 字段非阻塞、partial result 仍发布。宁可少报不漏报。

## 扩展性：接口层 + 阶段间数据流

核心思路是 **每个模块都可替换、每个阶段都可独立演进**。

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│ LLMClient   │    │ Pipeline     │    │ Formatter/  │
│ (接口)       │◄───│ (模型+prompt)│───►│ Publisher   │
│             │    │              │    │ (接口)       │
│ anthropic   │    │ Stage 1 ─┐   │    │             │
│ Client      │    │          │   │    │ GitHub PR   │
│             │    │ Stage 2 ◄┘   │    │ Review API  │
│ 未来:        │    │ (summary→   │     │             │
│ OpenAI/     │    │  risk)       │    │ 未来:       │
│ 本地模型     │    │              │    │ Slack/邮件   │
└─────────────┘    └──────────────┘    └─────────────┘
```

### 各层扩展点

| 维度 | 现状 | 扩展方向 | 改动范围 |
|---|---|---|---|
| LLM Provider | Anthropic 兼容 API | OpenAI、Ollama 等 | 新 `LLMClient` 实现 |
| Pipeline 阶段 | 2 阶段串行 | Stage 1.5（识别受影响模块）、并行多 scan 投票 | 修改 `Run` 编排逻辑 |
| 输出渠道 | GitHub PR Review Comment | Slack webhook、CLI stdout | 新 `PRReviewClient` 实现 |
| Context 来源 | GitHub Compare API | GitLab MR、本地 git diff | 新 `Builder` + `PipelineInput` 构造 |
| Prompt A/B | 固定 prompt 文件 | 按 `PRNumber % 2` 路由不同 prompt | 创建 `Pipeline{promptA, promptB}` |
| Stage 间通信 | summary → risk 单向字符串 | risk → 自动生成 fix commit | 扩展 `AnalysisResult`（已预留字段） |

### 行级内联评论与 Suggested Changes

Review 评论发布采用混合模式：有效的 risk 以行级内联评论形式精确标注在 diff 代码行上，position 计算失败的 risk 退回 body 文本展示，不丢失。

```
┌──────────────┐     ┌──────────────────┐     ┌─────────────────────┐
│ Risk (LLM)   │────►│ diff patch       │────►│ DraftReviewComment  │
│ .File .Line  │     │ → FileLineToPos  │     │ .Path .Position     │
│ .Description │     │ → position (1-   │     │ .Body               │
│ .FixSug      │     │   based index)   │     │ (incl. 修复建议)     │
└──────────────┘     └──────────────────┘     └─────────────────────┘
```

- **定位**：`FileLineToPosition(patch, line)` 将 LLM 输出的文件行号转为 GitHub diff position。该函数按 GitHub 定义计数 diff 中所有行（除 `@@` 头外），支持多 hunk
- **FixSuggestion**：`parseRiskBlock` 将 `**建议**：` 内容单独提取到 `Risk.FixSuggestion`，内联评论以 `**修复建议**：` 格式展示，body 回退以 blockquote 格式展示
- **Fallback**：position 为 0 时（文件不匹配或行号不在变更范围内）回退为 body 文本，不丢失风险信息
### 不做

- **多轮对话**：每次 Review 是一次性调用，PR 是静态快照，不需要维持对话历史，省去状态管理复杂度
- **Agent/tool-calling**：不让 LLM 自己调 GitHub API。上下文由代码预先获取，LLM 只做分析，避免幻觉导致额外 API 调用和延迟
- **实时流式展示**：streaming 仅用于绕过 API 10 分钟超时限制。结果完整收集后再一次性 publish，保证 Review 评论的原子性
