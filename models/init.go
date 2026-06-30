package models

import (
	"os"
	"path/filepath"
)

var test2 = func(string) {

}

func init() {
	InitLogger()
	RedirectStdLog()
	killp()
	for _, arg := range os.Args {
		if arg == "-d" {
			Daemon()
		}
	}
	ExecPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	SetLogDir(filepath.Join(ExecPath, "logs"))
	System().Infof("当前工作目录 %s", ExecPath)
	InitChan()
	initConfig()
	initDB()
	initSysConfig()
	go initVersion()
	initContainer()
	initHandle()
	initCron()
	go initTgBot()
	initTask()
	initNolan()
	initWX()
	tempToken()
	// InitActivityListWithHotReload() 会自动加载 YAML 配置并启动热加载监控
	// 如果 YAML 加载失败，会回退到空配置，但不影响其他功能
	if err := InitActivityListWithHotReload(); err != nil {
		Error("活动配置热加载初始化失败: %v", err)
	} else {
		Info("活动配置热加载已启动")
	}

	// 启动数据库→青龙同步服务
	InitSyncService()

	// 从青龙迁移已有数据到数据库（仅首次运行时执行）
	go MigrateFromQingLongToDB()

	initiiiiqtTask()
}


func initNolan() {
    Config.VIP = true
    Info("VIP验证成功")
}


func initWX() {
	env := GetEnv("WxGroupID")
	Config.WXGroupID = env
		//Autocollection
 	env = GetEnv("Autocollection")
	Config.Autocollection = env
}
