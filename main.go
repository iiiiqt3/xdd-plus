package main

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/cdle/xdd/controllers"
	"github.com/cdle/xdd/models"
	"github.com/eatmoreapple/openwechat"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var theme = ""

var query = ""
var friedns openwechat.Friends

type Result struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}
type AuthResult struct {
	Flag    bool        `json:"flag"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func main() {

	logs.SetLogger(logs.AdapterFile, "{\"filename\":\"logs/xdd.log\", \"level\":6}")

	go func() {
		models.Save <- &models.JdCookie{}
	}()

	web.Get("/permisson", func(ctx *context.Context) {
		tel := ctx.Input.Query("phone")
		logs.Info(tel)
		auth := models.GetAuth(tel)
		if auth {
			result := AuthResult{
				Flag:    true,
				Data:    "null",
				Code:    20000,
				Message: "操作成功",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			ctx.WriteString(string(jsons))
		} else {
			result := AuthResult{
				Flag:    false,
				Data:    "null",
				Code:    51000,
				Message: "操作失败",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			ctx.WriteString(string(jsons))
		}
	})

	web.Get("/count", func(ctx *context.Context) {
		ctx.WriteString(models.Count())
	})

	web.Get("/announcement", func(ctx *context.Context) {
		result := Result{
			Data:    models.Config.Title,
			Code:    0,
			Message: "查询成功",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		ctx.WriteString(string(jsons))
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

	//vweb.Router("/api/login/cookie", &controllers.LoginController{}, "get:Cookie")
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
	if models.Config.VIP {
		web.Router("/wx/receive", &controllers.WxController{}, "post:HandleWxMessage")
		web.Router("/qq", &controllers.QQController{}, "get,post:Echo")
		web.Router("/api/envs", &controllers.AccountController{}, "get:ListEnvs")
		web.Router("/api/envs", &controllers.AccountController{}, "post:CreateOrUpdateEnv")
		web.Router("/api/config", &controllers.ConfigController{}, "get:ListConfig")
		web.Router("/api/config", &controllers.ConfigController{}, "post:CreateOrUpdateConfig")

		//vweb.Router("/api/loginselect", &controllers.AccountController{}, "get:ListLoginSelect")
		//vweb.Router("/api/loginselect", &controllers.AccountController{}, "post,delete:CreateOrUpdateLoginSelect")

	}

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
	go func() {
		time.Sleep(time.Second * 4)
		(&models.JdCookie{}).Push(fmt.Sprintf("小滴滴已启动，版本号:%s", models.Config.Version))
	}()

	//bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式
	//
	//// 注册消息处理函数
	//bot.MessageHandler = func(msg *openwechat.Message) {
	//	if msg.IsText() && msg.Content == "ping" {
	//
	//		msg.ReplyText("pong")
	//
	//		name := msg.FromUserName
	//		id := friedns.GetByUsername(name).User
	//		id.Detail()
	//
	//		//msg.ReplyText(id)
	//
	//	}
	//}

	//
	//// 注册登陆二维码回调
	//bot.UUIDCallback = openwechat.PrintlnQrcodeUrl
	//
	//// 登陆
	//reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")
	//defer reloadStorage.Close()
	//err := bot.PushLogin(reloadStorage, openwechat.NewRetryLoginOption())
	//
	//// 获取登陆的用户
	//self, err := bot.GetCurrentUser()
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//friedns, err = self.Friends()
	//
	//// 阻塞主goroutine, 直到发生异常或者用户主动退出
	//bot.Block()

	web.Run()

}
