# go-yogan-domain-docs

安全的文档文件读取领域包。

## 功能

- 列出目录文件
- 读取文件内容
- 路径遍历攻击防护
- 文件大小限制

## 安全特性

1. **路径遍历防护**：检测并阻止 `..` 路径
2. **基础目录限制**：只允许访问配置的目录
3. **隐藏文件过滤**：自动跳过 `.` 开头的文件
4. **文件大小限制**：最大 10MB

## 使用

```go
svc, err := docs.NewService("/path/to/docs")
if err != nil {
    log.Fatal(err)
}

// 列出文件
files, err := svc.ListFiles("")

// 读取文件
content, err := svc.ReadFile("example.md")
```
