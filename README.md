# go-yogan-domain-docs

安全的文档文件读取领域包，支持 Markdown 标题解析和 Redis 缓存。

## 功能

- 列出目录文件
- 读取文件内容
- **文章标题解析**：从 Markdown 第一行（`# 标题`）解析标题
- **Redis 缓存**：可选的标题缓存（24 小时 TTL）
- 路径遍历攻击防护
- 文件大小限制

## 安全特性

1. **路径遍历防护**：检测并阻止 `..` 路径
2. **基础目录限制**：只允许访问配置的目录
3. **隐藏文件过滤**：自动跳过 `.` 开头的文件
4. **文件大小限制**：最大 10MB

## 使用

### 基本用法

```go
svc, err := docs.NewService("/path/to/docs")
if err != nil {
    log.Fatal(err)
}

// 列出文件（包含解析的标题）
files, err := svc.ListFiles("", docs.SortDesc)

// 读取文件
content, err := svc.ReadFile("example.md")
```

### 启用 Redis 缓存

```go
import (
    docs "github.com/KOMKZ/go-yogan-domain-docs"
    "time"
)

// 从内核 Redis 组件获取客户端
redisClient := redisManager.Client("main")

// 创建服务并启用缓存
svc, err := docs.NewService(
    "/path/to/docs",
    docs.WithRedisCache(redisClient, "docs:title:", 24*time.Hour),
)
```

### FileInfo 结构

```go
type FileInfo struct {
    Name    string    `json:"name"`              // 文件名
    Title   string    `json:"title,omitempty"`   // 文章标题（从第一行解析）
    Path    string    `json:"path"`              // 相对路径
    Size    int64     `json:"size"`              // 文件大小
    IsDir   bool      `json:"is_dir"`            // 是否为目录
    ModTime time.Time `json:"mod_time"`          // 修改时间
}
```

## 缓存管理

```go
// 清除单个文件的标题缓存
svc.InvalidateTitleCache("path/to/file.md")

// 清除所有标题缓存
svc.InvalidateAllTitleCache()
```
