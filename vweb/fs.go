package vweb

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

// WebFs 嵌入前端HTML静态文件（保留用于未来可能的回退）
//
//go:embed html/*
var WebFs embed.FS

// JsFs 嵌入前端JS静态文件
//
//go:embed js/*
var JsFs embed.FS

// vwebDir 磁盘上 vweb 目录的绝对路径
var vwebDir string

func init() {
	// 优先：可执行文件同级目录下的 vweb
	exe, err := os.Executable()
	if err == nil {
		d := filepath.Join(filepath.Dir(exe), "vweb")
		if _, err := os.Stat(d); err == nil {
			vwebDir = d
		}
	}
	// 回退：当前工作目录下的 vweb
	if vwebDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			d := filepath.Join(cwd, "vweb")
			if _, err := os.Stat(d); err == nil {
				vwebDir = d
			}
		}
	}
	if vwebDir == "" {
		fmt.Println("[vweb] 警告: 未找到磁盘上的 vweb 目录，HTML文件将无法热更新")
	}
}

// ReadFile 从磁盘 vweb 目录读取文件，修改后刷新浏览器即可生效
func ReadFile(name string) ([]byte, error) {
	if vwebDir == "" {
		return nil, fmt.Errorf("vweb 目录未找到，无法读取 %s", name)
	}
	diskPath := filepath.Join(vwebDir, name)
	return os.ReadFile(diskPath)
}

// AssetDir 返回 vweb 下子目录的磁盘路径（用于静态资源路由）
func AssetDir(sub string) string {
	if vwebDir == "" {
		return ""
	}
	return filepath.Join(vwebDir, sub)
}
