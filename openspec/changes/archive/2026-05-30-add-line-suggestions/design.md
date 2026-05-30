# Design: Stage 3 行级优化建议

## 涉及的代码

```
internal/analyzer/
├── pipeline.go        # parseSuggestionResponse 实现
├── types.go           # 已有 Suggestion struct，无需改动
└── diff_position.go   # 新增：diff position 计算
```

## A. 建议解析

### LLM 输出格式

Prompt 要求 LLM 按以下格式输出：

```
### <文件路径>
- **<行号>** <描述>
  ```<语言>
  <代码示例>
  ```
```

示例：

```
### main.go
- **42** 使用 strings.Builder 替代字符串拼接
  ```go
  var b strings.Builder
  b.WriteString("hello")
  ```
- **58** 错误处理可提取为统一中间件
  ```go
  func errorHandler(next http.Handler) http.Handler { ... }
  ```

### handler/auth.go
- **15** 密码比较应使用常量时间比较
  ```go
  subtle.ConstantTimeCompare([]byte(a), []byte(b))
  ```
```

### 解析器逻辑

```
parseSuggestionResponse(resp):
  1. 按 "### " 分割出各文件区块
  2. 每个区块内按 "- **行号**" 分割出各建议
  3. 提取：文件路径、行号、描述文本、代码块
  4. 返回 []Suggestion
```

### Prompt 更新

```go
const systemPromptSuggestion = "你是一个代码改进顾问。请对以下代码提供具体的优化建议。\n" +
    "重点关注：代码可读性、性能、测试覆盖、最佳实践。\n\n" +
    "请按以下格式回复，每个文件一个区块：\n\n" +
    "### <文件路径>\n" +
    "- **<行号>** <简短描述>\n" +
    "  \\`\\`\\`<语言>\n" +
    "  <改进后的代码示例>\n" +
    "  \\`\\`\\`\n\n" +
    "如果没有建议，回复'无'。"
```

## C. Diff Position 计算

### 格式

Unified diff 格式：
```
@@ -start_old,count_old +start_new,count_new @@ header
 context_line
+added_line
-deleted_line
 context_line
```

### 计算规则

- diff position 从 1 开始计数
- 跳过 hunk header (@@ ... @@)
- 每行（包括 + - 和 上下文行）占一个 position
- 根据文件行号找到对应的 new line（+ 和 上下文行），返回其 position

### 接口

```go
// FileLineToPosition converts a file line number in the new version
// to a diff position within a unified diff hunk.
// Returns 0 if the line cannot be found in the diff.
func FileLineToPosition(hunk string, newLine int) int
```

### 算法

```
FileLineToPosition(hunk, targetLine):
  lines = split(hunk by "\n")
  position = 0
  currentNewLine = 0  // track new file line number from @@ header
  
  for each line:
    if line starts with "@@":
      parse start_new (e.g., "+start_new,count_new")
      currentNewLine = start_new
      continue  // skip hunk header
    
    position++
    
    if not line starts with "-":  // + or context line
      currentNewLine++
      if currentNewLine == targetLine:
        return position
    // deleted lines don't increment currentNewLine
  
  return 0  // not found in this hunk
```
