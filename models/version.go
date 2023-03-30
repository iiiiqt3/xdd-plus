package models

import (
	"errors"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"os"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var version = "v9.5"
var describe = "修复图片"
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

func Update(sender *Sender) error {
	logs.Info("检查更新" + version)
	sender.Reply("小滴滴开始检查更新")
	value, err := httplib.Get(UpdateUrl + "/version1").String()
	if err != nil {
		return errors.New("获取版本号失败")
	} else {
		if strings.Contains(version, value) {
			return errors.New("小滴滴已是最新版啦")
		} else {
			logs.Info("开始更新")
			sender.Reply("小滴滴开始更新程序")
			logs.Info(UpdateUrl + "/github.com/cdle/xdd-linux-" + runtime.GOARCH)
			req := httplib.Get(UpdateUrl + "/github.com/cdle/xdd-linux-" + runtime.GOARCH)
			req.SetTimeout(time.Minute*5, time.Minute*5)
			data, err := req.Bytes()

			filename := ExecPath + "/" + AppName
			logs.Info(filename)
			if err = os.RemoveAll(filename); err != nil {
				return errors.New("删除旧程序错误")
			}
			if f, err := os.OpenFile(filename, syscall.O_CREAT, 0777); err != nil {
				return errors.New("创建程序错误")
			} else {
				_, err := f.Write(data)
				f.Close()
				if err != nil {
					des := err.Error()
					if err = os.WriteFile(filename, data, 777); err != nil {
						return errors.New("写入程序错误" + des)
					}
				}
			}
			sender.Reply("更新完成，马上重启")
			logs.Info("更新成功")
		}
		return nil
	}
}
