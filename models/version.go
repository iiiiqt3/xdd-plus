package models

import (
	"errors"
	"github.com/beego/beego/v2/core/logs"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var version = "v14.0"
var describe = "修复新号不进入容器"
var AppName = "xdd"
var pname = pname1()
var GitRepo = "http://180.152.5.230:5699/feiniao/xdd.git"
var GitBranch = "master"
var GitUser = ""
var GitToken = ""
var notify = true

func pname1() string {
	var pname string
	executable, err := os.Executable()
	if err != nil {
		submatch := regexp.MustCompile(`[\\/]([^/\s]+)`).FindStringSubmatch(os.Args[0])
		if len(submatch) > 1 {
			pname = submatch[1]
		} else {
			pname = submatch[0]
		}
	} else {
		r, _ := regexp.Compile(`[\\/]`)
		split := r.Split(executable, -1)
		pname = split[len(split)-1]
	}
	return pname
}

func getGitRepo() string {
	value := GetEnv("gitRepo")
	if value != "" {
		return value
	}
	return GitRepo
}

func getGitBranch() string {
	value := GetEnv("gitBranch")
	if value != "" {
		return value
	}
	return GitBranch
}

func getGitUser() string {
	value := GetEnv("gitUser")
	if value != "" {
		return value
	}
	return GitUser
}

func getGitToken() string {
	value := GetEnv("gitToken")
	if value != "" {
		return value
	}
	return GitToken
}

func getAuthGitRepo() string {
	repo := getGitRepo()
	user := getGitUser()
	token := getGitToken()
	if user == "" && token == "" {
		return repo
	}
	cred := ""
	if token != "" && user != "" {
		cred = user + ":" + token
	} else if token != "" {
		cred = token
	} else {
		cred = user
	}
	if strings.HasPrefix(repo, "http://") {
		return "http://" + cred + "@" + repo[7:]
	}
	if strings.HasPrefix(repo, "https://") {
		return "https://" + cred + "@" + repo[8:]
	}
	return repo
}

func sanitizeGitOutput(output string) string {
	s := output
	s = regexp.MustCompile(`http[s]?://[^\s'"]+`).ReplaceAllString(s, "***")
	s = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`).ReplaceAllString(s, "***")
	return s
}

func isGitRepo() bool {
	_, err := os.Stat(ExecPath + "/.git")
	return err == nil
}

func ensureGitRepo() error {
	if isGitRepo() {
		return nil
	}
	logs.Info("目录不是git仓库，执行git init")
	cmd := exec.Command("git", "init")
	cmd.Dir = ExecPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		logs.Warn("git init失败: %s, %v", string(output), err)
		return errors.New("git init失败: " + string(output))
	}
	logs.Info("git init成功")
	return nil
}

func getRemoteHeadHash() (string, error) {
	cmd := exec.Command("git", "ls-remote", getAuthGitRepo(), "HEAD")
	cmd.Dir = ExecPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(sanitizeGitOutput(string(output)))
	}
	fields := strings.Fields(string(output))
	if len(fields) == 0 {
		return "", errors.New("无法解析远程HEAD")
	}
	return fields[0], nil
}

func getLocalHeadHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = ExecPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func checkGitUpdate() (bool, error) {
	remoteHash, err := getRemoteHeadHash()
	if err != nil {
		return false, err
	}
	localHash, err := getLocalHeadHash()
	if err != nil {
		return false, err
	}
	logs.Info("远程版本: %s, 本地版本: %s", remoteHash[:8], localHash[:8])
	return remoteHash != localHash, nil
}

func initVersion() {
	Config.Version = version
	logs.Info("检查更新 " + version)
	hasUpdate, err := checkGitUpdate()
	if err != nil {
		logs.Info("版本检查失败: %v", err)
		return
	}
	if hasUpdate {
		logs.Info("小滴滴检测到新版本")
		(&JdCookie{}).Push("小滴滴检测到新版本")
	}
}

func GetNewVersion() {
	if notify {
		Config.Version = version
		logs.Info("检查更新 " + version)
		hasUpdate, err := checkGitUpdate()
		if err != nil {
			logs.Info("版本检查失败: %v", err)
			return
		}
		if hasUpdate {
			notify = false
			logs.Info("小滴滴检测到新版本")
			(&JdCookie{}).Push("小滴滴检测到新版本")
		}
	}
}

func Update(sender *Sender) error {
	logs.Info("开始git更新检查")
	sender.Reply("小滴滴开始检查更新")

	hasUpdate, checkErr := checkGitUpdate()
	if checkErr != nil {
		logs.Warn("版本检查失败，跳过比对直接更新: %v", checkErr)
		sender.Reply("版本检查失败，直接尝试拉取更新...")
	} else if !hasUpdate {
		return errors.New("小滴滴已是最新版啦")
	}

	if err := ensureGitRepo(); err != nil {
		return errors.New("初始化git仓库失败: " + err.Error())
	}

	sender.Reply("正在拉取最新源码...")
	logs.Info("git fetch %s %s", getGitRepo(), getGitBranch())
	cmd := exec.Command("git", "fetch", getAuthGitRepo(), getGitBranch())
	cmd.Dir = ExecPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		logs.Warn("git fetch失败: %s, %v", string(output), err)
		return errors.New("拉取源码失败: " + sanitizeGitOutput(string(output)))
	}
	logs.Info("git fetch成功")

	cmd = exec.Command("git", "reset", "--hard", "FETCH_HEAD")
	cmd.Dir = ExecPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		logs.Warn("git reset失败: %s, %v", string(output), err)
		return errors.New("重置代码失败: " + sanitizeGitOutput(string(output)))
	}
	logs.Info("git reset成功: %s", string(output))

	sender.Reply("正在编译最新源码...")
	newBinary := ExecPath + "/" + AppName + "_new"
	cmd = exec.Command("go", "build", "-o", newBinary, "main.go")
	cmd.Dir = ExecPath
	output, err = cmd.CombinedOutput()
	if err != nil {
		logs.Warn("编译失败: %s, %v", string(output), err)
		return errors.New("编译失败: " + string(output))
	}
	logs.Info("编译成功")

	oldBinary := ExecPath + "/" + AppName
	if err = os.Remove(oldBinary); err != nil {
		logs.Warn("删除旧程序失败: %v", err)
	}
	if err = os.Rename(newBinary, oldBinary); err != nil {
		return errors.New("替换程序失败: " + err.Error())
	}

	sender.Reply("更新完成，马上重启")
	logs.Info("更新成功")
	return nil
}
