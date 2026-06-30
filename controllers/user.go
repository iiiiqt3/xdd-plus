package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/cdle/xdd/models"
	"net/url"
	"strconv"
	"strings"
)

// UserController 用户信息查询控制器
type UserController struct {
	BaseController
}

// GetUserInfo 根据pin获取京东用户Cookie信息
func (c *UserController) GetUserInfo() {

	pin := c.GetString("pin")
	pin = url.QueryEscape(pin)
	cookie, err := models.GetJdCookie(pin)
	ok := models.CookieOK(cookie)
	if err != nil {
		models.Error(err)
		result := Result{
			Data:    "null",
			Code:    1,
			Message: "查无匹配的pin",
		}
		jsons, errs := json.Marshal(result)
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
		jsons, errs := json.Marshal(result)
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
		jsons, errs := json.Marshal(result)
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}
}

// GetUserPin 根据QQ号获取关联的京东pin列表
func (c *UserController) GetUserPin() {
	qq := c.GetString("QQ")
	if strings.EqualFold(qq, strconv.Itoa(models.Config.QQID)) {
		result := Result{
			Data:    "null",
			Code:    1,
			Message: "禁止查询他人ID",
		}
		jsons, errs := json.Marshal(result)
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
		jsons, errs := json.Marshal(result)
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
		jsons, errs := json.Marshal(result)
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}
}
