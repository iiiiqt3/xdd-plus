package controllers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/xdd/models"
)

type LoginController struct {
	BaseController
}

type Result struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Cookie  string      `json:"cookie"`
}

func FetchJdCookieValue(key string, cookies string) string {
	match := regexp.MustCompile(key + `=([^;]*);{0,1}`).FindStringSubmatch(cookies)
	if len(match) == 2 {
		return match[1]
	} else {
		return ""
	}
}

func (c *LoginController) IsAdmin() {
	pin := c.GetString("pin")
	if pin == "" {
		c.Ctx.Redirect(302, "/")
		c.StopRun()
	} else {
		if strings.EqualFold(models.Config.Master, pin) {
			c.SetSession("pin", pin)
			c.Ctx.WriteString("登录")
		}
	}
}

func (c *LoginController) CkLogin() {
	pin := c.GetString("pin")
	key := c.GetString("key")
	qq, _ := c.GetInt("qq")
	bz := c.GetString("bz")
	push := c.GetString("push")
	ck := c.GetString("ck")

	if key != "" && pin != "" {
		nolanLogin(key, pin, qq, bz, push, c)
	} else if ck != "" {

	} else {
		result := Result{
			Data:    "null",
			Code:    2,
			Message: "ck格式错误",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}

}

func nolanLogin(key string, pin string, qq int, bz string, push string, c *LoginController) {
	ck := &models.JdCookie{
		PtKey:    key,
		PtPin:    pin,
		Hack:     models.False,
		QQ:       qq,
		Note:     bz,
		PushPlus: push,
	}
	if key != "" && pin != "" {
		if models.CookieOK(ck) {
			query := ck.Query()
			result := Result{
				Data: query,
				Code: 0,
			}
			if !models.HasPin(pin) {
				models.NewJdCookie(ck)
				result.Message = fmt.Sprintf("添加成功")
				jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
				if errs != nil {
					fmt.Println(errs.Error())
				}
				c.Ctx.WriteString(string(jsons))
			} else if !models.HasKey(key) {
				ck, _ := models.GetJdCookie(pin)
				ck.InPool(key)
				result.Message = fmt.Sprintf("更新成功")
				jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
				if errs != nil {
					fmt.Println(errs.Error())
				}
				c.Ctx.WriteString(string(jsons))
			}
			result.Message = "登录成功"
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			c.Ctx.WriteString(string(jsons))
		} else {
			result := Result{
				Data:    "null",
				Code:    1,
				Message: "CK过期",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			c.Ctx.WriteString(string(jsons))
		}
	}
}

func (c *LoginController) SMSLogin() {
	cookie := c.GetString("ck")
	qq := c.GetString("qq")
	token := c.GetString("token")
	logs.Info(cookie)

	if token == models.Config.ApiToken || models.Config.ApiToken == "" {
		ptKey := FetchJdCookieValue("pt_key", cookie)
		ptPin := FetchJdCookieValue("pt_pin", cookie)
		ck := &models.JdCookie{
			PtKey: ptKey,
			PtPin: ptPin,
			Hack:  models.False,
			QQ:    0,
		}
		if qq != "" {
			ck.QQ, _ = strconv.Atoi(qq)
		}

		if ptKey != "" && ptPin != "" {
			if models.CookieOK(ck) {
				//(&models.JdCookie{}).Push(cookie)
				if nck, err := models.GetJdCookie(ck.PtPin); err == nil {
					nck.InPool(ptKey)
					if qq != "" && len(qq) > 6 {
						//ck.Update(models.QQ, qq)
						atoi, _ := strconv.Atoi(qq)
						ck.Updates(models.JdCookie{
							QQ:       atoi,
							UpdateAt: time.Now().Local().Format("2006-01-02"),
						})
					}
					msg := fmt.Sprintf("来自短信的更新,%s,QQ: %v", nck.PtPin, qq)
					ck.Push(ck.Query())
					(&models.JdCookie{}).Push(msg)

				} else {

					models.NewJdCookie(ck)
					if qq != "" {
						msg := fmt.Sprintf("来自短信的添加,：%s,QQ: %v", ck.PtPin, qq)
						(&models.JdCookie{}).Push(msg)
					} else {
						msg := fmt.Sprintf("来自短信的添加,：%s", ck.PtPin)
						(&models.JdCookie{}).Push(msg)
					}
					ck.Push(ck.Query())

				}

				result := Result{
					Data:    "null",
					Code:    200,
					Message: "添加成功",
				}
				jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
				if errs != nil {
					fmt.Println(errs.Error())
				}
				c.Ctx.WriteString(string(jsons))

			} else {
				result := Result{
					Data:    "null",
					Code:    300,
					Message: "CK过期",
				}
				jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
				if errs != nil {
					fmt.Println(errs.Error())
				}
				msg := fmt.Sprintf("传入过期CK，请小心攻击，：%s", ck.PtPin)
				(&models.JdCookie{}).Push(msg)
				c.Ctx.WriteString(string(jsons))
			}
		} else {
			result := Result{
				Data:    "null",
				Code:    300,
				Message: "CK错误",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			msg := fmt.Sprintf("传入错误CK，请小心攻击，：%s", ck.PtPin)
			(&models.JdCookie{}).Push(msg)
			c.Ctx.WriteString(string(jsons))
		}
	} else {
		result := Result{
			Data:    "null",
			Code:    300,
			Message: "Token错误",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		msg := fmt.Sprintf("传入错误Token，请小心攻击")
		(&models.JdCookie{}).Push(msg)
		c.Ctx.WriteString(string(jsons))
	}

}

func (c *LoginController) WskeyLogin() {
	cookie := string(c.Ctx.Input.RequestBody)
	cookie, _ = url.QueryUnescape(cookie)
	logs.Info(cookie)
	Wskey := FetchJdCookieValue("wskey", cookie)
	ptPin := FetchJdCookieValue("pin", cookie)
	ptPin = url.QueryEscape(ptPin)
	ck := &models.JdCookie{
		WsKey: Wskey,
		PtPin: ptPin,
		Hack:  models.False,
		QQ:    0,
	}
	if Wskey != "" && ptPin != "" {
		ok, s := models.CheckWskeyOK(ck)
		if ok {
			if nck, err := models.GetJdCookie(ck.PtPin); err == nil {
				msg := fmt.Sprintf("Wskey账号更新,账号：%s", nck.PtPin)
				(&models.JdCookie{}).Push(msg)
			} else {
				models.NewJdCookie(ck)
				msg := fmt.Sprintf("添加新的Wskey账号,账号：%s", ck.PtPin)
				(&models.JdCookie{}).Push(msg)
			}
			result := Result{
				Data:    "null",
				Code:    200,
				Message: "添加成功",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			c.Ctx.WriteString(string(jsons))
		} else {
			result := Result{
				Data:    "null",
				Code:    0,
				Message: s,
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			c.Ctx.WriteString(string(jsons))
		}
	} else {
		result := Result{
			Data:    "null",
			Code:    0,
			Message: "CK错误",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}
}
