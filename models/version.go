package models

import (
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"os"
	"regexp"
)

var version = "v1.0"
var AppName = "xdd"
var pname = pname1()
var UpdateUrl = "https://update.smxy.xyz"
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
func initVersion() {
	Config.Version = version
	logs.Info("检查更新" + version)
	value := GetEnv("updateUrl")
	if value != "" {
		UpdateUrl = value
	}
	value, err := httplib.Get(UpdateUrl + "/version1").String()
	if err != nil {
		logs.Info("更新版本的失败")
	} else {
		// name := AppName + "_" + runtime.GOOS + "_" + runtime.GOARCH
		logs.Info(value)
		logs.Info(version)
		if value != version {
			logs.Info("小滴滴检测到新版本：" + value)
			(&JdCookie{}).Push("小滴滴检测到新版本：" + value)
		}
	}
}

func GetNewVersion() {
	if notify {
		Config.Version = version
		logs.Info("检查更新" + version)
		value := GetEnv("updateUrl")
		if value != "" {
			UpdateUrl = value
		}
		value, err := httplib.Get(UpdateUrl + "/version1").String()
		if err != nil {
			logs.Info("更新版本的失败")
		} else {
			if value != version {
				notify = false
				logs.Info("小滴滴检测到新版本：" + value)
				(&JdCookie{}).Push("小滴滴检测到新版本：" + value)
			}
		}
	}
}
