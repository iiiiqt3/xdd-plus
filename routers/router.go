// @APIVersion
// @Title Wechat 08- 算法  ---  五端扫码版（ipad、安卓pad、win、mac、car、62 iphone、a16 安卓）
// @Description 可开启自动心跳 - 自动二次登录 - 长连接心跳  （新上线账号尽量 ipad、安卓 pad 加好友发圈。IOS 参数比较全）
package routers

import (
	"io/ioutil"
	"wechatdll/controllers"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/plugins/cors"
)

func init() {
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		AllowCredentials: true,
	}))

	ns := beego.NewNamespace("/api",
		beego.NSNamespace("/Admin",
			beego.NSInclude(
				&controllers.AdminController{},
			),
		),
		beego.NSNamespace("/Login",
			beego.NSInclude(
				&controllers.LoginController{},
			),
		),
		beego.NSNamespace("/Msg",
			beego.NSInclude(
				&controllers.MsgController{},
			),
		),
		beego.NSNamespace("/Friend",
			beego.NSInclude(
				&controllers.FriendController{},
			),
		),
		beego.NSNamespace("/Finder",
			beego.NSInclude(
				&controllers.FinderController{},
			),
		),
		beego.NSNamespace("/FriendCircle",
			beego.NSInclude(
				&controllers.FriendCircleController{},
			),
		),
		beego.NSNamespace("/Favor",
			beego.NSInclude(
				&controllers.FavorController{},
			),
		),
		beego.NSNamespace("/Group",
			beego.NSInclude(
				&controllers.GroupController{},
			),
		),
		beego.NSNamespace("/Label",
			beego.NSInclude(
				&controllers.LabelController{},
			),
		),
		beego.NSNamespace("/User",
			beego.NSInclude(
				&controllers.UserController{},
			),
		),
		beego.NSNamespace("/Wxapp",
			beego.NSInclude(
				&controllers.WxappController{},
			),
		),
		beego.NSNamespace("/QWContact",
			beego.NSInclude(
				&controllers.QWContactController{},
			),
		),
		beego.NSNamespace("/OfficialAccounts",
			beego.NSInclude(
				&controllers.OfficialAccountsController{},
			),
		),
		beego.NSNamespace("/SayHello",
			beego.NSInclude(
				&controllers.SayHelloController{},
			),
		),
		beego.NSNamespace("/Tools",
			beego.NSInclude(
				&controllers.ToolsController{},
			),
		),
		beego.NSNamespace("/TenPay",
			beego.NSInclude(
				&controllers.TenPayController{},
			),
		),
		// 对接xdd后端的微信协议接口（WxApiController）
		beego.NSRouter("/v1/wx/login/code", &controllers.WxApiController{}, "post:WxLoginCode"),
		beego.NSRouter("/v1/wx/login/status", &controllers.WxApiController{}, "post:WxLoginStatus"),
		beego.NSRouter("/v1/wx/login/again", &controllers.WxApiController{}, "post:WxLoginAgain"),
		beego.NSRouter("/v1/wx/login/awake", &controllers.WxApiController{}, "post:WxLoginAwake"),
		beego.NSRouter("/v1/wx/login/twice", &controllers.WxApiController{}, "post:WxLoginTwice"),
		beego.NSRouter("/v1/wx/login/logout", &controllers.WxApiController{}, "post:WxLoginLogout"),
		beego.NSRouter("/v1/wx/user/status", &controllers.WxApiController{}, "get:WxUserStatus"),
		beego.NSRouter("/v1/wx/user/delete", &controllers.WxApiController{}, "post:WxUserDelete"),
	)
	beego.AddNamespace(ns)

	// 根路径 -> 管理后台
	beego.Get("/", func(ctx *context.Context) {
		ctx.Redirect(302, "/admin.html")
	})

	// /admin -> /admin.html
	beego.Get("/admin", func(ctx *context.Context) {
		ctx.Redirect(302, "/admin.html")
	})

	// 确保 /admin.html 能正确访问
	beego.Get("/admin.html", func(ctx *context.Context) {
		content, err := ioutil.ReadFile("swagger/admin.html")
		if err != nil {
			ctx.Output.SetStatus(404)
			ctx.Output.Body([]byte("File not found"))
			return
		}
		ctx.Output.ContentType("text/html")
		ctx.Output.Body(content)
	})
}
