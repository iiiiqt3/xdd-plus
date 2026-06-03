package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/astaxie/beego"
)

// 程序启动时生成的会话密钥，重启后失效
var adminSessionSecret string

func init() {
	b := make([]byte, 16)
	rand.Read(b)
	adminSessionSecret = hex.EncodeToString(b)
}

// 管理后台模块
type AdminController struct {
	BaseController
}

// @Summary 管理后台登录验证
// @Param	username	query	string	true	"用户名"
// @Param	password	query	string	true	"密码"
// @Success 200
// @router /Login [post]
func (c *AdminController) Login() {
	username := c.GetString("username")
	password := c.GetString("password")

	// 从配置文件读取账号密码
	adminUser := beego.AppConfig.String("adminuser")
	adminPass := beego.AppConfig.String("adminpass")

	// 如果配置文件没有设置，默认 admin/admin123
	if adminUser == "" {
		adminUser = "admin"
	}
	if adminPass == "" {
		adminPass = "admin123"
	}

	if username == adminUser && password == adminPass {
		c.Data["json"] = map[string]interface{}{
			"Code":    0,
			"Success": true,
			"Message": "登录成功",
			"Secret":  adminSessionSecret,
		}
	} else {
		c.Data["json"] = map[string]interface{}{
			"Code":    -1,
			"Success": false,
			"Message": "用户名或密码错误",
		}
	}
	c.ServeJSON()
}

// @Summary 验证会话是否有效（程序重启后失效）
// @Param	secret	query	string	true	"会话密钥"
// @Success 200
// @router /CheckSession [post]
func (c *AdminController) CheckSession() {
	secret := c.GetString("secret")
	if secret == adminSessionSecret {
		c.Data["json"] = map[string]interface{}{
			"Code":    0,
			"Success": true,
			"Message": "会话有效",
		}
	} else {
		c.Data["json"] = map[string]interface{}{
			"Code":    -1,
			"Success": false,
			"Message": "会话已失效，请重新登录",
		}
	}
	c.ServeJSON()
}
