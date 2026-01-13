package docs

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RedisClient Redis 客户端接口（用于解耦内核依赖）
type RedisClient interface {
	Get(ctx context.Context, key string) *goredis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *goredis.StatusCmd
	Del(ctx context.Context, keys ...string) *goredis.IntCmd
	Keys(ctx context.Context, pattern string) *goredis.StringSliceCmd
}

// SortOrder 排序方式
type SortOrder string

const (
	SortDesc SortOrder = "desc" // 倒序（默认，最新在前）
	SortAsc  SortOrder = "asc"  // 正序（最旧在前）
)

// FileInfo 文件信息
type FileInfo struct {
	Name    string    `json:"name"`
	Title   string    `json:"title,omitempty"` // 文章标题（从第一行解析）
	Path    string    `json:"path"`            // 相对路径
	Size    int64     `json:"size"`
	IsDir   bool      `json:"is_dir"`
	ModTime time.Time `json:"mod_time"`
}

// FileContent 文件内容
type FileContent struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Size    int64  `json:"size"`
}

// Service 文档服务
type Service struct {
	basePath    string        // 允许访问的基础目录
	redisClient RedisClient   // Redis 客户端（通过内核获取）
	cachePrefix string        // 缓存键前缀
	cacheTTL    time.Duration // 缓存过期时间
}

// ServiceOption 服务选项
type ServiceOption func(*Service)

// WithRedisCache 设置 Redis 缓存
// client: Redis 客户端（通过内核 redisManager.Client("main") 获取）
// prefix: 缓存键前缀
// ttl: 缓存过期时间
func WithRedisCache(client RedisClient, prefix string, ttl time.Duration) ServiceOption {
	return func(s *Service) {
		s.redisClient = client
		s.cachePrefix = prefix
		s.cacheTTL = ttl
	}
}

// NewService 创建文档服务
func NewService(basePath string, opts ...ServiceOption) (*Service, error) {
	// 规范化路径
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}

	// 检查目录是否存在
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, ErrDirectoryNotFound
	}
	if !info.IsDir() {
		return nil, ErrDirectoryNotFound
	}

	svc := &Service{
		basePath:    absPath,
		cachePrefix: "docs:title:",
		cacheTTL:    24 * time.Hour, // 默认 24 小时
	}

	// 应用选项
	for _, opt := range opts {
		opt(svc)
	}

	return svc, nil
}

// ListFiles 列出目录下的文件
// order: 排序方式，"desc" 倒序（默认，最新在前），"asc" 正序（最旧在前）
func (s *Service) ListFiles(relativePath string, order SortOrder) ([]FileInfo, error) {
	// 安全检查：构建并验证完整路径
	fullPath, err := s.safePath(relativePath)
	if err != nil {
		return nil, err
	}

	// 检查是否为目录
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDirectoryNotFound
		}
		return nil, ErrReadFailed
	}
	if !info.IsDir() {
		return nil, ErrDirectoryNotFound
	}

	// 读取目录
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, ErrReadFailed
	}

	var files []FileInfo
	for _, entry := range entries {
		// 跳过隐藏文件
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		relPath := relativePath
		if relPath == "" || relPath == "." {
			relPath = entry.Name()
		} else {
			relPath = filepath.Join(relativePath, entry.Name())
		}

		fileInfo := FileInfo{
			Name:    entry.Name(),
			Path:    relPath,
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		}

		// 解析 Markdown 文件标题
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			fileInfo.Title = s.getArticleTitle(filepath.Join(fullPath, entry.Name()), relPath)
		}

		files = append(files, fileInfo)
	}

	// 按修改时间排序
	if order == SortAsc {
		// 正序：最旧在前
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime.Before(files[j].ModTime)
		})
	} else {
		// 倒序（默认）：最新在前
		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime.After(files[j].ModTime)
		})
	}

	return files, nil
}

// ReadFile 读取文件内容
func (s *Service) ReadFile(relativePath string) (*FileContent, error) {
	// 安全检查：构建并验证完整路径
	fullPath, err := s.safePath(relativePath)
	if err != nil {
		return nil, err
	}

	// 检查文件是否存在
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, ErrReadFailed
	}

	// 不允许读取目录
	if info.IsDir() {
		return nil, ErrFileNotFound
	}

	// 限制文件大小（10MB）
	const maxSize = 10 * 1024 * 1024
	if info.Size() > maxSize {
		return nil, ErrReadFailed
	}

	// 读取文件内容
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, ErrReadFailed
	}

	return &FileContent{
		Name:    filepath.Base(relativePath),
		Path:    relativePath,
		Content: string(content),
		Size:    info.Size(),
	}, nil
}

// safePath 安全路径检查，防止目录遍历攻击
func (s *Service) safePath(relativePath string) (string, error) {
	// 清理路径
	cleanPath := filepath.Clean(relativePath)

	// 检查是否包含 .. 或绝对路径
	if strings.Contains(cleanPath, "..") {
		return "", ErrPathTraversal
	}
	if filepath.IsAbs(cleanPath) {
		return "", ErrPathNotAllowed
	}

	// 构建完整路径
	fullPath := filepath.Join(s.basePath, cleanPath)

	// 再次验证：确保最终路径在 basePath 内
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", ErrPathNotAllowed
	}

	// 使用 filepath.Rel 检查路径关系
	rel, err := filepath.Rel(s.basePath, absPath)
	if err != nil {
		return "", ErrPathNotAllowed
	}

	// 如果相对路径以 .. 开头，说明在 basePath 外部
	if strings.HasPrefix(rel, "..") {
		return "", ErrPathTraversal
	}

	return absPath, nil
}

// GetBasePath 获取基础路径
func (s *Service) GetBasePath() string {
	return s.basePath
}

// WalkFiles 递归遍历目录（可选）
func (s *Service) WalkFiles(relativePath string, depth int) ([]FileInfo, error) {
	if depth <= 0 {
		depth = 1
	}
	if depth > 3 {
		depth = 3 // 限制最大深度
	}

	fullPath, err := s.safePath(relativePath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	currentDepth := 0

	err = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // 跳过错误
		}

		// 计算相对于 fullPath 的深度
		relToFull, _ := filepath.Rel(fullPath, path)
		pathDepth := len(strings.Split(relToFull, string(filepath.Separator)))
		if relToFull == "." {
			pathDepth = 0
		}

		if pathDepth > depth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过隐藏文件和根目录
		if strings.HasPrefix(d.Name(), ".") || path == fullPath {
			if d.IsDir() && path != fullPath && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if path == fullPath {
				return nil
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(s.basePath, path)

		files = append(files, FileInfo{
			Name:    d.Name(),
			Path:    relPath,
			Size:    info.Size(),
			IsDir:   d.IsDir(),
			ModTime: info.ModTime(),
		})

		currentDepth = pathDepth
		_ = currentDepth

		return nil
	})

	if err != nil {
		return nil, ErrReadFailed
	}

	return files, nil
}

// getArticleTitle 获取文章标题（优先从缓存读取）
func (s *Service) getArticleTitle(fullPath, relativePath string) string {
	ctx := context.Background()

	// 尝试从 Redis 缓存读取
	if s.redisClient != nil {
		cacheKey := s.cachePrefix + relativePath
		if title, err := s.redisClient.Get(ctx, cacheKey).Result(); err == nil && title != "" {
			return title
		}
	}

	// 解析文件第一行获取标题
	title := s.parseFirstLineTitle(fullPath)
	if title == "" {
		return ""
	}

	// 缓存到 Redis
	if s.redisClient != nil {
		cacheKey := s.cachePrefix + relativePath
		_ = s.redisClient.Set(ctx, cacheKey, title, s.cacheTTL).Err()
	}

	return title
}

// parseFirstLineTitle 解析文件第一行作为标题
// 只读取文件的前几行，不读取全部内容
func (s *Service) parseFirstLineTitle(fullPath string) string {
	file, err := os.Open(fullPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// 最多读取前 10 行，寻找标题
	for i := 0; i < 10 && scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行
		if line == "" {
			continue
		}

		// 跳过 YAML front matter 开始标记
		if line == "---" {
			// 继续读取直到遇到下一个 ---
			for scanner.Scan() {
				if strings.TrimSpace(scanner.Text()) == "---" {
					break
				}
			}
			continue
		}

		// 检查是否是 Markdown 标题（# 开头）
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}

		// 如果第一个非空行不是标题格式，返回空
		break
	}

	return ""
}

// InvalidateTitleCache 清除单个标题缓存
func (s *Service) InvalidateTitleCache(relativePath string) error {
	if s.redisClient == nil {
		return nil
	}
	ctx := context.Background()
	cacheKey := s.cachePrefix + relativePath
	_, err := s.redisClient.Del(ctx, cacheKey).Result()
	return err
}

// InvalidateAllTitleCache 清除所有标题缓存
func (s *Service) InvalidateAllTitleCache() error {
	if s.redisClient == nil {
		return nil
	}
	ctx := context.Background()
	pattern := s.cachePrefix + "*"
	keys, err := s.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}
	if len(keys) > 0 {
		_, err = s.redisClient.Del(ctx, keys...).Result()
		return err
	}
	return nil
}
