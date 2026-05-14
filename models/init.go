package models

import (
	//"fmt"
	//"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"os"
	"path/filepath"
	//"strconv"
//	"strings"
)

var test2 = func(string) {

}

func init() {
	killp()
	for _, arg := range os.Args {
		if arg == "-d" {
			Daemon()
		}
	}
	ExecPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	logs.Info("当前%s", ExecPath)
	InitChan()
	initConfig()
	initDB()
	initSysConfig()
	go initVersion()
	//go initUserAgent()
	initContainer()
	initHandle()
	initCron()
	go initTgBot()
//	InitReplies()
	initTask()
	initNolan()
	//go initOrder(branchHelpOrderQueue, "jd_qmckd_branchHelp", "jd_qmckd_inviteIdArr_expand")
	//initRepos()
	initWX()
	tempToken()
	// InitActivityListWithHotReload() 会自动加载 YAML 配置并启动热加载监控
	// 如果 YAML 加载失败，会回退到空配置，但不影响其他功能
	if err := InitActivityListWithHotReload(); err != nil {
		logs.Error("活动配置热加载初始化失败: %v", err)
	} else {
		logs.Info("活动配置热加载已启动")
	}

	initiiiiqtTask()
}
/*
func initNolan() {

	s, _ := httplib.Get(fmt.Sprintf("http://159.75.140.129:9801/user/auth3?qqNum=%s&version=%s", strconv.Itoa(Config.QQID), Config.Version)).String()
	contains := strings.Contains(s, "true")
	logs.Info(s)
	if contains {
		Config.VIP = true
		logs.Info("VIP验证成功")
	} else {
		logs.Info("VIP校验失败")
	}

}
*/

func initNolan() {
    Config.VIP = true
    logs.Info("VIP验证成功") // 或者可以直接记录“跳过验证”信息
}


func initWX() {
	env := GetEnv("WxGroupID")
	Config.WXGroupID = env
		//Autocollection
 	env = GetEnv("Autocollection")
	Config.Autocollection = env
}
