package docs

import (
	"net/http"

	"github.com/KOMKZ/go-yogan-framework/errcode"
)

const ModuleDocs = 27

var (
	ErrPathNotAllowed    = errcode.Register(errcode.New(ModuleDocs, 1001, "docs", "error.docs.path_not_allowed", "路径不允许访问", http.StatusForbidden))
	ErrFileNotFound      = errcode.Register(errcode.New(ModuleDocs, 1002, "docs", "error.docs.file_not_found", "文件不存在", http.StatusNotFound))
	ErrReadFailed        = errcode.Register(errcode.New(ModuleDocs, 1003, "docs", "error.docs.read_failed", "读取文件失败", http.StatusInternalServerError))
	ErrDirectoryNotFound = errcode.Register(errcode.New(ModuleDocs, 1004, "docs", "error.docs.directory_not_found", "目录不存在", http.StatusNotFound))
	ErrPathTraversal     = errcode.Register(errcode.New(ModuleDocs, 1005, "docs", "error.docs.path_traversal", "检测到路径遍历攻击", http.StatusForbidden))
)
