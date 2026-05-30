# PR Reviewer

基于 AI 的 GitHub Pull Request 自动评审工具。提交 PR 后自动分析代码变更，生成结构化 Review 评论。

## 工作流程

```
PR 提交 → Webhook → 获取变更上下文 → AI 分析 → 发布 Review 评论
```

- **Stage 1**: 变更总结（轻量模型，快速概览 PR 目的和影响范围）
- **Stage 2**: 风险识别（主力模型，扫描安全漏洞、逻辑错误、并发问题、破坏性变更）
- **Stage 3**: 优化建议（小 PR 自动触发，提供行级改进建议）

## 快速开始

### 1. 创建 GitHub App

在 GitHub 创建 App，配置：

| 权限 | 级别 |
|------|------|
| Pull Requests | Read & Write |
| Contents | Read |
| Metadata | Read（自动） |

订阅事件：**Pull request**、**Issue comment**

生成 Webhook Secret 时，使用高熵随机字符串：

```bash
openssl rand -hex 32
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置环境变量

> 注意：`.env` 已在 `.gitignore` 中，切勿将包含真实凭据的 `.env` 提交到版本控制。

```bash
cp .env.example .env
```

编辑 `.env`：

```env
GITHUB_APP_ID=你的App ID
GITHUB_APP_PRIVATE_KEY=私钥文件路径（推荐 .pem 文件，避免多行值解析问题）
GITHUB_WEBHOOK_SECRET=Webhook Secret 推荐使用 openssl rand -hex 32 生成的随机字符串
LLM_API_KEY=你的 API Key
LLM_BASE_URL=https://api.deepseek.com/anthropic   # DeepSeek 官方 Anthropic 兼容端点
LLM_MODEL_FAST=deepseek-v4-flash                  # Stage 1 模型
LLM_MODEL_POWERFUL=deepseek-v4-pro                 # Stage 2+3 模型
```

### 4. 启动

> 以下 ngrok 方式仅适用于本地开发测试。生产环境应使用正式反向代理（如 Nginx、Caddy）并启用 HTTPS。

```bash
# 终端 1：启动服务
go run ./cmd/server/

# 终端 2：暴露公网（仅本地开发）
ngrok http 8080
```

将 ngrok 的 Forwarding URL 填入 GitHub App 的 Webhook URL（注意以 `/webhook` 结尾）：

```
https://xxxx.ngrok-free.app/webhook
```

### 5. 安装 App 到仓库

在 GitHub App 设置页 → Install App → 选择目标仓库。

然后在仓库提一个 PR，等待几秒即可看到 AI Review 评论。

## 手动触发

除了 PR 创建和推送时自动触发，也可以在 PR 下发评论指令手动触发：

```
@prreviewer-app review
```

手动触发会强制开启 Stage 3 深度分析。

## 配置说明

| 环境变量 | 必须 | 默认值 | 说明 |
|----------|------|--------|------|
| `GITHUB_APP_ID` | ✓ | - | GitHub App ID |
| `GITHUB_APP_PRIVATE_KEY` | ✓ | - | 私钥文件路径（支持 `.pem` 文件，也支持直接填写单行格式密钥内容） |
| `GITHUB_WEBHOOK_SECRET` | ✓ | - | Webhook 签名密钥，建议 `openssl rand -hex 32` 生成 |
| `LLM_API_KEY` | ✓ | - | LLM API Key |
| `LLM_BASE_URL` | - | `https://api.anthropic.com` | LLM 接口地址，支持任意兼容 Anthropic API 的服务 |
| `LLM_MODEL_FAST` | - | `claude-haiku-4-5` | Stage 1 模型名称 |
| `LLM_MODEL_POWERFUL` | - | `claude-sonnet-4-6` | Stage 2+3 模型名称 |
| `PORT` | - | `8080` | 监听端口 |

模型名称需与所选 LLM 服务支持的模型对应：

| 服务 | 示例模型 |
|------|----------|
| Anthropic Claude | `claude-haiku-4-5`、`claude-sonnet-4-6` |
| DeepSeek | `deepseek-v4-flash`、`deepseek-v4-pro` |

## 项目结构

```
PRReviewer/
├── cmd/server/main.go          # 入口
├── internal/
│   ├── config/                 # 环境变量配置
│   ├── github/                 # Webhook 验证 + API 客户端
│   ├── context/                # PR 上下文获取（diff + 文件内容）
│   ├── analyzer/               # AI 分析流水线 + 响应解析
│   ├── comment/                # Review 评论格式化与发布
│   ├── server/                 # HTTP 路由与流程编排
│   └── logger/                 # 请求日志中间件
├── .env.example
└── openspec/                   # 设计文档
```

## 许可证

MIT
