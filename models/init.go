package models

import (
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func init() {
	killp()
	for _, arg := range os.Args {
		if arg == "-d" {
			Daemon()
		}
	}
	ExecPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	logs.Info("当前%s", ExecPath)
	initConfig()
	initDB()
	go initVersion()
	//go initUserAgent()
	initContainer()
	initHandle()
	initCron()
	go initTgBot()
	InitReplies()
	initTask()
	initNolan()
	//initRepos()
	InitSky()
}

func initNolan() {

	s, _ := httplib.Get(fmt.Sprintf("http://auth.smxy.xyz/user/auth3?qqNum=%s&version=%s", strconv.Itoa(Config.QQID), Config.Version)).String()
	contains := strings.Contains(s, "true")
	logs.Info(s)
	if contains {
		Config.VIP = true
		logs.Info("VIP验证成功")
	} else {
		logs.Info("VIP校验失败")
	}

}
