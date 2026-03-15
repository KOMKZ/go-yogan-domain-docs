package docs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListFiles_Success(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "b.md"), []byte("# B"), 0644)

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	files, err := svc.ListFiles("", SortDesc)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestListFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	files, err := svc.ListFiles("", SortDesc)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestReadFile_Success(t *testing.T) {
	dir := t.TempDir()
	content := []byte("# Hello\n\nWorld")
	_ = os.WriteFile(filepath.Join(dir, "readme.md"), content, 0644)

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	fc, err := svc.ReadFile("readme.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if fc.Content != string(content) {
		t.Errorf("content mismatch: got %q", fc.Content)
	}
	if fc.Name != "readme.md" {
		t.Errorf("name: got %q", fc.Name)
	}
}

func TestReadFile_NotFound(t *testing.T) {
	dir := t.TempDir()

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, err = svc.ReadFile("nonexistent.md")
	if err != ErrFileNotFound {
		t.Errorf("expected ErrFileNotFound, got %v", err)
	}
}

func TestSafePath_NormalPath(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "ok.md"), []byte("x"), 0644)

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	fc, err := svc.ReadFile("ok.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if fc.Content != "x" {
		t.Errorf("unexpected content: %q", fc.Content)
	}
}

func TestSafePath_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, err = svc.ReadFile("../etc/passwd")
	if err != ErrPathTraversal {
		t.Errorf("expected ErrPathTraversal, got %v", err)
	}
}

func TestSafePath_AbsolutePath(t *testing.T) {
	dir := t.TempDir()

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, err = svc.ReadFile("/etc/passwd")
	if err != ErrPathNotAllowed {
		t.Errorf("expected ErrPathNotAllowed, got %v", err)
	}
}

func TestSafePath_DotDotInMiddle(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	_ = os.MkdirAll(sub, 0755)
	_ = os.WriteFile(filepath.Join(sub, "ok.md"), []byte("x"), 0644)

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	fc, err := svc.ReadFile("sub/../sub/ok.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if fc.Content != "x" {
		t.Errorf("unexpected content: %q", fc.Content)
	}
}

func TestWalkFiles_Success(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "a", "b"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "a", "b", "c.md"), []byte("# C"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "root.md"), []byte("# Root"), 0644)

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	files, err := svc.WalkFiles("", 3)
	if err != nil {
		t.Fatalf("WalkFiles: %v", err)
	}
	if len(files) < 2 {
		t.Errorf("expected at least 2 files, got %d", len(files))
	}
}

func TestGetArticleTitle_Success(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "article.md"), []byte("# My Article Title\n\nBody"), 0644)

	svc, err := NewService(dir)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	files, err := svc.ListFiles("", SortDesc)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Title != "My Article Title" {
		t.Errorf("expected title 'My Article Title', got %q", files[0].Title)
	}
}

func TestListAllFiles_SortByName(t *testing.T) {
	dir := t.TempDir()
	alpha := filepath.Join(dir, "alpha.md")
	zeta := filepath.Join(dir, "zeta.md")
	_ = os.WriteFile(alpha, []byte("# A"), 0644)
	_ = os.WriteFile(zeta, []byte("# Z"), 0644)

	now := time.Now()
	_ = os.Chtimes(alpha, now, now.Add(2*time.Hour))
	_ = os.Chtimes(zeta, now, now.Add(-2*time.Hour))

	svc, err := NewMultiDirService([]DirectoryConfig{{Name: "docs", Path: dir}}, []string{".md"})
	if err != nil {
		t.Fatalf("NewMultiDirService: %v", err)
	}

	ascFiles, err := svc.ListAllFiles(SortAsc, SortByName)
	if err != nil {
		t.Fatalf("ListAllFiles asc name: %v", err)
	}
	if len(ascFiles) != 2 {
		t.Fatalf("expected 2 files, got %d", len(ascFiles))
	}
	if ascFiles[0].Name != "alpha.md" {
		t.Fatalf("expected alpha.md first in asc name sort, got %s", ascFiles[0].Name)
	}

	descFiles, err := svc.ListAllFiles(SortDesc, SortByName)
	if err != nil {
		t.Fatalf("ListAllFiles desc name: %v", err)
	}
	if len(descFiles) != 2 {
		t.Fatalf("expected 2 files, got %d", len(descFiles))
	}
	if descFiles[0].Name != "zeta.md" {
		t.Fatalf("expected zeta.md first in desc name sort, got %s", descFiles[0].Name)
	}
}

func TestListAllFiles_SortByModTime(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "old.md")
	newFile := filepath.Join(dir, "new.md")
	_ = os.WriteFile(oldFile, []byte("# Old"), 0644)
	_ = os.WriteFile(newFile, []byte("# New"), 0644)

	base := time.Now().Add(-3 * time.Hour)
	_ = os.Chtimes(oldFile, base, base)
	_ = os.Chtimes(newFile, base.Add(2*time.Hour), base.Add(2*time.Hour))

	svc, err := NewMultiDirService([]DirectoryConfig{{Name: "docs", Path: dir}}, []string{".md"})
	if err != nil {
		t.Fatalf("NewMultiDirService: %v", err)
	}

	descFiles, err := svc.ListAllFiles(SortDesc, SortByModTime)
	if err != nil {
		t.Fatalf("ListAllFiles desc mod_time: %v", err)
	}
	if len(descFiles) != 2 {
		t.Fatalf("expected 2 files, got %d", len(descFiles))
	}
	if descFiles[0].Name != "new.md" {
		t.Fatalf("expected new.md first in desc mod_time sort, got %s", descFiles[0].Name)
	}

	ascFiles, err := svc.ListAllFiles(SortAsc, SortByModTime)
	if err != nil {
		t.Fatalf("ListAllFiles asc mod_time: %v", err)
	}
	if len(ascFiles) != 2 {
		t.Fatalf("expected 2 files, got %d", len(ascFiles))
	}
	if ascFiles[0].Name != "old.md" {
		t.Fatalf("expected old.md first in asc mod_time sort, got %s", ascFiles[0].Name)
	}
}
