# PR Reviewer

<p align="center">
  <img width="30%" src="docs/images/Icon.png" alt="PR Reviewer Logo">
</p>

基于 AI 的 GitHub Pull Request 自动评审工具。提交 PR 后自动分析代码变更，生成结构化 Review 评论。

[![Install](https://img.shields.io/badge/GitHub%20App-Install-blue)](https://github.com/apps/prreviewer-app)
[![Demo](https://img.shields.io/badge/Bilibili-演示视频-00A1D6)](https://www.bilibili.com/video/BV1eqVV6aEKL/)

## 使用教程

### 安装 App

点击上方 **Install** 按钮，选择要启用 AI Review 的仓库。

### 自动触发

安装后，每当仓库有 PR 创建或推送新 commit 时，App 会自动进行 AI Review 并发布评论。

### 手动触发

在 PR 下评论以下指令即可手动触发：

```
@prreviewer-app review
```

## 工作流程

```
PR 提交 → Webhook → 获取变更上下文 → AI 分析 → 发布 Review 评论
```

- **Stage 1**: 变更总结（轻量模型，快速概览 PR 目的和影响范围）
- **Stage 2**: 风险识别（主力模型，扫描安全漏洞、逻辑错误、并发问题、破坏性变更，每条风险评论包含问题描述、后果分析、修复建议和严重程度评级）

## 自部署

如果你想运行自己的实例，按以下步骤操作。

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
LLM_MODEL_POWERFUL=deepseek-v4-pro                 # Stage 2 模型
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

## 配置说明

| 环境变量 | 必须 | 默认值 | 说明 |
|----------|------|--------|------|
| `GITHUB_APP_ID` | ✓ | - | GitHub App ID |
| `GITHUB_APP_PRIVATE_KEY` | ✓ | - | 私钥文件路径（支持 `.pem` 文件，也支持直接填写单行格式密钥内容） |
| `GITHUB_WEBHOOK_SECRET` | ✓ | - | Webhook 签名密钥，建议 `openssl rand -hex 32` 生成 |
| `LLM_API_KEY` | ✓ | - | LLM API Key |
| `LLM_BASE_URL` | - | `https://api.anthropic.com` | LLM 接口地址，支持任意兼容 Anthropic API 的服务 |
| `LLM_MODEL_FAST` | - | `claude-haiku-4-5` | Stage 1 模型名称 |
| `LLM_MODEL_POWERFUL` | - | `claude-sonnet-4-6` | Stage 2 模型名称 |
| `PORT` | - | `8080` | 监听端口 |

模型名称需与所选 LLM 服务支持的模型对应：

| 服务 | 示例模型 |
|------|----------|
| Anthropic Claude | `claude-haiku-4-5`、`claude-sonnet-4-6` |
| DeepSeek | `deepseek-v4-flash`、`deepseek-v4-pro` |

## 设计思路

架构设计、模型选择、扩展性等设计思路详见 [设计文档](docs/design.md)。未来扩展方向详见 [扩展文档](docs/future.md)。

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
├── docs/design.md              # 架构设计思路
├── .env.example
└── openspec/                   # 变更规约与归档
```

## 许可证

MIT
