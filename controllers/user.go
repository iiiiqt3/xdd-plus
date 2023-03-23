package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/xdd/models"
	"net/url"
	"strconv"
	"strings"
)

type UserController struct {
	BaseController
}

func (c *UserController) GetUserInfo() {

	pin := c.GetString("pin")
	pin = url.QueryEscape(pin)
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

func (c *UserController) GetUserPin() {
	qq := c.GetString("QQ")
	if strings.EqualFold(qq, strconv.Itoa(models.Config.QQID)) {
		result := Result{
			Data:    "null",
			Code:    1,
			Message: "禁止查询他人ID",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
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
