package models

import (
	"path/filepath"
	"strings"
)

// uploadsRootDir 返回 uploads 根目录绝对路径
func uploadsRootDir() string {
	return filepath.Clean(filepath.Join(ExecPath, "uploads"))
}

// ResolveUploadAbsPath 将 uploads 下的相对路径解析为绝对路径；禁止目录穿越
// relative 可为 "guide/foo.jpg" 或 "/uploads/guide/foo.jpg"
func ResolveUploadAbsPath(relative string) (string, bool) {
	rel := strings.TrimSpace(relative)
	rel = strings.TrimPrefix(rel, "/")
	rel = strings.TrimPrefix(rel, "uploads/")
	rel = strings.TrimPrefix(rel, "uploads\\")
	if rel == "" || strings.Contains(rel, "\x00") {
		return "", false
	}
	uploadsRoot := uploadsRootDir()
	absPath := filepath.Clean(filepath.Join(uploadsRoot, filepath.FromSlash(rel)))
	relPath, err := filepath.Rel(uploadsRoot, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") || relPath == ".." {
		return "", false
	}
	return absPath, true
}
