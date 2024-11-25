package models

import (
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"os"
	"path/filepath"
	"strings"
)

var msgchan = make(chan []byte)

func init() {
	killp()
	for _, arg := range os.Args {
		if arg == "-d" {
			Daemon()
		}
	}
	ExecPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	logs.Info("当前%s", ExecPath)
	go WriteMsg(msgchan)
	initConfig()
	initDB()
	initSysConfig()
	//go initVersion()
	initContainer()
	initCron()
	go initTgBot()
	InitReplies()
	initTask()
	initNolan()
	//go initOrder(branchHelpOrderQueue, "jd_qmckd_branchHelp", "jd_qmckd_inviteIdArr_expand")
	//initRepos()
	initWX()
	tempToken()
}

func initNolan() {

	s, _ := httplib.Get(fmt.Sprintf("http://auth.smxy.xyz/user/auth3?qqNum=%s&version=%s", Config.QQID, Config.Version)).String()
	contains := strings.Contains(s, "true")
	logs.Info(s)
	if contains {
		Config.VIP = true
		logs.Info("VIP验证成功")
	} else {
		logs.Info("VIP校验失败")
	}
}

func initWX() {
	env := GetEnv("WxGroupID")
	Config.WXGroupID = env
}
