# Design: 新解析器

## 输入格式

```
In server/tag.go:

> - app_key = value[0]
+    if value, ok := md["app_key"]; ok {
+        appKey = value[0]
+    }
读取 metadata 时未检查 value slice 长度，直接索引 value[0] 会 panic。
应添加 len(value) > 0 检查。
----

In cmd/api/main.go:

>  p := strings.TrimPrefix(r.URL.Path, "/swagger/")
  p = path.Join("proto", p)
用户输入未过滤 .. 路径段，可读取 proto 目录外任意文件。
应使用 filepath.Clean 并验证最终路径前缀。
```

## 解析算法

```
parseRiskResponse(resp):
  blocks = split(resp, "\n----\n")
  for each block:
    file, codeBlock, comment = parseBlock(block)
    if file != "":
      risks.append(Risk{
        File:        file
        Description: comment
        FixSuggestion: codeBlock
        Severity:    "warning"
        Confidence:  "medium"
      })

parseBlock(block):
  lines = split(block, "\n")
  // Line 0: "In <file>:"
  file = extractFrom(lines[0]) // "In " ... ":"
  skip blank line 1
  // Lines starting with "> " are code block
  for lines starting with "> ", " >", etc:
    codeBlock += line + "\n"
  // Rest is comment text
  comment = strings.TrimSpace(remaining)
  return file, codeBlock, comment
```

## 修改文件

| 文件 | 改动 |
|------|------|
| `internal/analyzer/pipeline.go` | 重写 `parseRiskResponse` 及相关函数 |
| `internal/analyzer/pipeline_test.go` | 更新测试用例匹配新格式 |
