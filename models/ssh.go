package models

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	
	"time"

	"github.com/beego/beego/v2/core/logs"
)

var (
	inSshMode   = false
	currentDir  = "" // 当前工作目录
	dirMutex    = sync.RWMutex{}
)

// normalizeSpaces 将连续空白字符（空格、制表符等）压缩为单个空格
func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// 高危命令关键词（必须使用单空格格式，不区分大小写）
// 所有条目将与“规范化后的命令”进行子串匹配
var dangerousCommands = []string{
	"rm -rf /",
	"rm -fr /",
	"rm -rf /*",
	"rm -rf */",
	"rm -rf ./*",
	"rm -rf ~",
	"rm -rf ~/",
	":(){:|:&};:",
	"dd if=",
	"mkfs",
	"fdisk",
	"parted",
	"shutdown",
	"reboot",
	"halt",
	"init 0",
	"poweroff",
	"chmod -R 777 /",
	"chown -R root:root /",
	"> /dev/",
	"echo > /",
	"find / -delete",
	"wget .* -O /",
	"curl .* > /",
	"/dev/null",
	"mv /",
	"format",
}

// 初始化当前目录为程序运行目录
func init() {
	if wd, err := os.Getwd(); err == nil {
		currentDir = wd
	} else {
		currentDir = "/"
	}

	codeSignals = append(codeSignals, CodeSignal{
		Command: []string{"ssh"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if !sender.IsAdmin {
				return nil
			}
			dirMutex.Lock()
			inSshMode = true
			dirMutex.Unlock()
			sender.Reply("🔐 进入 Shell 模式（支持 cd 切换目录）。\n💡 输入 `exit` 或 `quit` 退出。")
			return nil
		},
	})
}

// 安全解析 cd 命令的目标路径
func parseCdPath(current, input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" || input == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "/", nil
		}
		return home, nil
	}

	if strings.HasPrefix(input, "/") {
		return filepath.Clean(input), nil
	}

	newPath := filepath.Join(current, input)
	return filepath.Clean(newPath), nil
}

// 在指定目录执行 shell 命令
func execShellInDir(dir, command string) string {
	escapedDir := strings.ReplaceAll(dir, "'", "'\"'\"'")
	fullCmd := fmt.Sprintf("cd '%s' && %s", escapedDir, command)

	cmd := exec.Command("sh", "-c", fullCmd)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case <-done:
		output := strings.TrimSpace(stdout.String() + stderr.String())
		if output == "" {
			return "（无输出）"
		}
		if len(output) > 1800 {
			return output[:1797] + "...\n[输出过长，已截断]"
		}
		return output
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		return "❌ 命令超时（30秒），已终止。"
	}
}

// 主消息处理函数：判断是否应作为 SSH 命令处理
func TryHandleSshMessage(sender *Sender) bool {
	if !sender.IsAdmin {
		return false
	}

	dirMutex.RLock()
	mode := inSshMode
	dir := currentDir
	dirMutex.RUnlock()

	if !mode {
		return false
	}

	msg := strings.TrimSpace(sender.RawMessage)
	if msg == "" {
		return true
	}

	// 处理退出
	if msg == "exit" || msg == "quit" {
		dirMutex.Lock()
		inSshMode = false
		dirMutex.Unlock()
		sender.Reply("✅ 已退出 Shell 模式。")
		return true
	}

	// ===== 高危命令拦截（支持多空格、制表符等）=====
	normalizedMsg := normalizeSpaces(strings.ToLower(msg))
	for _, danger := range dangerousCommands {
		if strings.Contains(normalizedMsg, danger) {
			dirMutex.Lock()
			inSshMode = false
			dirMutex.Unlock()

			sender.Reply("⚠️ 检测到高危操作命令，已自动退出 Shell 模式！\n🔒 为安全起见，禁止执行此类命令。")
			logs.Warn("[SSH] 高危命令拦截: %s", msg)
			return true
		}
	}
	// =============================================

	// 处理 cd 命令
	if strings.HasPrefix(strings.ToLower(msg), "cd ") || msg == "cd" {
		target := strings.TrimSpace(msg[2:])
		newDir, err := parseCdPath(dir, target)
		if err != nil {
			sender.Reply(fmt.Sprintf("❌ cd 解析错误: %v", err))
			return true
		}

		if _, err := os.Stat(newDir); os.IsNotExist(err) {
			sender.Reply(fmt.Sprintf("❌ 目录不存在: %s", newDir))
			return true
		}

		dirMutex.Lock()
		currentDir = newDir
		dirMutex.Unlock()

		sender.Reply(fmt.Sprintf("$ %s\n（目录已切换）", msg))
		return true
	}

	// 执行普通命令
	output := execShellInDir(dir, msg)
	sender.Reply(fmt.Sprintf("$ %s\n%s", msg, output))
	return true
}