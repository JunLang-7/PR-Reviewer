# Testing Convention

## 核心原则

**TDD (Test-Driven Development)**：在编写实现代码之前，先编写测试。

## 工作流

```
1. 理解需求 → 明确要实现的函数/模块
2. 编写测试 → 测试定义期望行为
3. 运行测试 → 确认测试失败（红）
4. 编写实现 → 最小代码让测试通过
5. 运行测试 → 确认测试通过（绿）
6. 重构     → 优化代码，测试保持绿色
```

## 要求

### 每个模块必须有测试

| 模块              | 测试文件                | 覆盖重点                  |
|-------------------|------------------------|--------------------------|
| internal/config   | config_test.go         | 配置加载、缺失变量、默认值  |
| internal/github   | client_test.go         | 认证、API 调用 mock       |
|                   | webhook_test.go        | 签名验证、事件解析         |
| internal/context  | builder_test.go        | diff 获取、文件获取、组装   |
| internal/analyzer | pipeline_test.go       | 阶段编排、跳过条件         |
|                   | summary_test.go        | prompt 构建、结果解析      |
|                   | risk_test.go           | prompt 构建、结果解析      |
|                   | suggestion_test.go     | 触发条件、prompt 构建      |
| internal/comment  | publisher_test.go      | Markdown 格式化、边界情况  |

### 测试规范

- 使用 Go 标准 `testing` 包
- 外部依赖（GitHub API, Claude API）使用接口抽象 + mock
- 测试函数命名：`Test<FunctionName>_<Scenario>`
- 使用 table-driven tests 覆盖多个场景
- 每个任务（tasks.md 中的细分项）应先有测试再有实现

### 运行

```bash
go test ./...
go test -race ./...
go test -cover ./...
```
