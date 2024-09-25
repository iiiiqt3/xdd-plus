package controllers

import (
	"encoding/json"
	"fmt"
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
}

func (c *LoginController) GetUserInfo() {
	if !models.Config.VIP {
		return
	}
	pin := c.GetString("pin")
	cookie, err := models.GetJdCookie(pin)
	ok := models.CookieOK(cookie)
	if err != nil {
		logs.Error(err)
		result := Result{
			Data:    "null",
			Code:    1,
			Message: "查无匹配的pin",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	} else if !ok {
		result := Result{
			Data:    "账号过期",
			Code:    0,
			Message: "账号过期",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	} else {
		result := Result{
			Data:    cookie.Query(),
			Code:    0,
			Message: "查询成功",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}
}

func (c *LoginController) GetUserPin() {
	if !models.Config.VIP {
		return
	}
	qq := c.GetString("QQ")
	if strings.EqualFold(qq, models.Config.QQID) {
		return
	}
	pins := models.GetPinList(qq)
	if pins == nil {
		result := Result{
			Data:    "null",
			Code:    1,
			Message: "查无匹配的pin",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	} else {
		result := Result{
			Data:    pins,
			Code:    0,
			Message: "查询成功",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}
}

type Cookie struct {
	ck string
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
		value := models.GetCache("AdminToken")
		if value != "" && pin == value {
			c.SetSession("token", value)
			logs.Info("登录成功:" + pin)
			c.Ctx.WriteString("登录")
		} else {
			c.Ctx.Redirect(302, "/")
			c.StopRun()
		}
	}
}

func (c *LoginController) CkLogin() {
	pin := c.GetString("pin")
	key := c.GetString("key")
	qq, _ := c.GetInt("qq")
	bz := c.GetString("bz")
	push := c.GetString("push")

	//c.Ctx.WriteString("添加成功")
	if key != "" && pin != "" {
		//ptKey := FetchJdCookieValue("pt_key", cookies)
		//ptPin := FetchJdCookieValue("pt_pin", cookies)
		ck := &models.JdCookie{
			PtKey:    key,
			PtPin:    pin,
			UserId:   strconv.Itoa(qq),
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
					//result.Data = ck.Query()
					jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
					if errs != nil {
						fmt.Println(errs.Error())
					}
					c.Ctx.WriteString(string(jsons))
				} else if !models.HasKey(key) {
					ck, _ := models.GetJdCookie(pin)
					//更新CK
					models.UpdateCookie(ck)
					result.Message = fmt.Sprintf("更新成功")
					//result.Data = ck.Query()
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

func (c *LoginController) SMSLogin() {
	cookie := c.GetString("ck")
	qq := c.GetString("qq")
	token := c.GetString("token")
	logs.Info(cookie)

	if token == models.Config.ApiToken || models.Config.ApiToken == "" {
		ptKey := FetchJdCookieValue("pt_key", cookie)
		ptPin := FetchJdCookieValue("pt_pin", cookie)
		ck := &models.JdCookie{
			PtKey:  ptKey,
			PtPin:  ptPin,
			UserId: qq,
		}

		if ptKey != "" && ptPin != "" {
			if models.CookieOK(ck) {
				if nck, err := models.GetJdCookie(ck.PtPin); err == nil {
					if qq != "" && len(qq) > 6 {
						ck.Updates(models.JdCookie{
							PtKey:     ptKey,
							PtPin:     ptPin,
							UserId:    qq,
							Available: "true",
							UpdateAt:  time.Now().Local().Format("2006-01-02"),
						})
					} else {
						ck.Updates(models.JdCookie{
							PtKey:     ptKey,
							PtPin:     ptPin,
							Available: "true",
							UpdateAt:  time.Now().Local().Format("2006-01-02"),
						})
					}
					msg := fmt.Sprintf("来自短信的更新,账号：%s,QQ: %v", nck.PtPin, qq)
					ck.Push(ck.Query())
					(&models.JdCookie{}).Push(msg)
				} else {
					models.NewJdCookie(ck)
					if qq != "" {
						msg := fmt.Sprintf("来自短信的添加,账号：%s,QQ: %v", ck.PtPin, qq)
						(&models.JdCookie{}).Push(msg)
					} else {
						msg := fmt.Sprintf("来自短信的添加,账号：%s", ck.PtPin)
						(&models.JdCookie{}).Push(msg)
					}
					ck.Push(ck.Query())
				}
				go func() {
					models.Save <- &models.JdCookie{}
				}()

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
				msg := fmt.Sprintf("传入过期CK，请小心攻击，账号：%s", ck.PtPin)
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
			msg := fmt.Sprintf("传入错误CK，请小心攻击，账号：%s", ck.PtPin)
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
