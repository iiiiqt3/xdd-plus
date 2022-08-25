package main

import (
	"fmt"
	"github.com/beego/beego/v2/server/web"
	"github.com/jpillora/overseer"
	"github.com/jpillora/overseer/fetcher"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/cdle/xdd/controllers"
	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/qbot"
)

var theme = ""

var query = ""

// BuildID is compile-time variable
var BuildID = "0"

//convert your 'main()' into a 'prog(state)'
//'prog()' is run in a child process
func prog(state overseer.State) {
	go func() {
		models.Save <- &models.JdCookie{}
	}()

	web.Get("/count", func(ctx *context.Context) {
		ctx.WriteString(models.Count())
	})

	if models.Config.VIP {
		web.Get("/query", func(ctx *context.Context) {
			if query != "" {
				ctx.WriteString(query)
				return
			}
			logs.Info("下载最新网页查询版本")
			s, _ := httplib.Get("https://git.smxy.xyz/jia_yuan/xdd-html/raw/branch/xdd/version/index.html").String()
			if s != "" {
				query = s
				ctx.WriteString(s)
				return
			}
			logs.Warn("主题下载失败，请注意查看网络环境")
		})
	}

	web.Get("/", func(ctx *context.Context) {
		if models.Config.Theme == "" {
			models.Config.Theme = "http://xdd.smxy.xyz/admin.html"
		}
		if theme != "" {
			ctx.WriteString(theme)
			return
		}
		if strings.Contains(models.Config.Theme, "http") {
			logs.Info("下载最新主题")
			s, _ := httplib.Get(models.Config.Theme).String()
			if s != "" {
				theme = s
				ctx.WriteString(s)
				return
			}
			logs.Warn("主题下载失败，使用默认主题")
		}
		f, err := os.Open(models.Config.Theme)
		if err == nil {
			d, _ := ioutil.ReadAll(f)
			theme = string(d)
			ctx.WriteString(string(d))
			return
		}
	})

	web.Router("/api/login/qrcode", &controllers.LoginController{}, "get:GetQrcode")
	web.Router("/api/login/qrcode.png", &controllers.LoginController{}, "get:GetQrcode")
	web.Router("/api/login/qrcode1", &controllers.LoginController{}, "get:GetQrcode1")
	web.Router("/api/login/query", &controllers.LoginController{}, "get:Query")
	web.Router("/api/login/cookie", &controllers.LoginController{}, "get:Cookie")
	web.Router("/api/login/admin", &controllers.LoginController{}, "post:IsAdmin")
	web.Router("/api/login/cklogin", &controllers.LoginController{}, "post:CkLogin")
	web.Router("/api/login/smslogin", &controllers.LoginController{}, "post:SMSLogin")
	web.Router("/api/getUserInfo", &controllers.LoginController{}, "post:GetUserInfo")
	web.Router("/api/getUserInfo", &controllers.LoginController{}, "get:GetUserInfo")
	web.Router("/api/getUserPin", &controllers.LoginController{}, "post:GetUserPin")
	web.Router("/api/getUserPin", &controllers.LoginController{}, "get:GetUserPin")
	web.Router("/api/account", &controllers.AccountController{}, "get:List")
	web.Router("/api/account", &controllers.AccountController{}, "post:CreateOrUpdate")
	web.Router("/admin", &controllers.AccountController{}, "get:Admin")
	web.Router("/admin", &controllers.AccountController{}, "post:Admin")
	web.Router("/wx/receive", &controllers.WxController{}, "post:HandleMessage")

	if models.Config.Static == "" {
		models.Config.Static = "./static"
	}
	web.BConfig.WebConfig.StaticDir["/static"] = models.Config.Static
	web.BConfig.AppName = models.AppName
	web.BConfig.WebConfig.AutoRender = false
	web.BConfig.CopyRequestBody = true
	web.BConfig.WebConfig.Session.SessionOn = true
	web.BConfig.WebConfig.Session.SessionGCMaxLifetime = 3600
	web.BConfig.WebConfig.Session.SessionName = models.AppName
	web.InsertFilter("*", web.BeforeRouter, cors.Allow(&cors.Options{
		//允许访问所有源
		AllowAllOrigins: true,
		//可选参数"GET", "POST", "PUT", "DELETE", "OPTIONS" (*为所有)
		//其中Options跨域复杂请求预检
		AllowMethods: []string{"*"},
		//指的是允许的Header的种类
		AllowHeaders: []string{"*"},
		//公开的HTTP标头列表
		ExposeHeaders: []string{"Content-Length"},
		//如果设置，则允许共享身份验证凭据，例如cookie
		AllowCredentials: true,
	}))

	if models.Config.QQID != 0 && models.Config.OpenQQ == "" {
		go qbot.Main()
	} else {
		logs.Info("不启动QQ")
	}
	go func() {
		time.Sleep(time.Second * 4)
		logs.Info(fmt.Sprintf("小滴滴已启动，版本号:%s", models.Config.Version))
		logs.Info(runtime.GOOS)
		logs.Info(runtime.GOARCH)
		(&models.JdCookie{}).Push(fmt.Sprintf("小滴滴已启动，版本号:%s", models.Config.Version))

	}()
	web.Run()
}
func main() {
	overseer.Run(overseer.Config{
		Program: prog,
		Fetcher: &fetcher.HTTP{
			URL: "http://xdd.smxy.xyz/xdd/xdd-linux-amd64",
			//URL:      "http://xdd.smxy.xyz/xdd/xdd-" + runtime.GOOS + "-" + runtime.GOARCH,
			Interval: 100 * time.Second,
			//e.g.http://localhost:4000/binaries/app-linux-amd64
		},
		Debug: true, //display log of overseer actions
	})
}
