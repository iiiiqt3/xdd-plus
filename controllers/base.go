package controllers

import (
	"encoding/json"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/cdle/xdd/models"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
	zh_translations "gopkg.in/go-playground/validator.v9/translations/zh"
	"net/http"
	"strconv"
)

var validate *validator.Validate
var trans ut.Translator

func init() {
	var zhCh = zh.New()
	validate = validator.New()
	var uni = ut.New(zhCh)
	trans, _ = uni.GetTranslator("zh")
	zh_translations.RegisterDefaultTranslations(validate, trans)
}

// BaseController 基础控制器，所有控制器继承此结构体，提供通用功能
type BaseController struct {
	beego.Controller
	PtPin         string
	Master        bool
	PortalUserID  int
	PortalAccount *models.WebUserAccount
	ClientCtx     models.ClientContext
}

// NextPrepare 接口定义，子控制器实现此接口以执行自定义前置处理
type NextPrepare interface {
	NextPrepare()
}

// Prepare beego生命周期钩子，请求进入时自动调用，执行子控制器的前置处理
func (c *BaseController) Prepare() {
	if app, ok := c.AppController.(NextPrepare); ok {
		app.NextPrepare()
	}
}

// Response 统一JSON响应方法，支持可变参数：数据、信息、状态码
func (c *BaseController) Response(ps ...interface{}) {
	rsp := struct {
		Code int         `json:"code"`
		Data interface{} `json:"data"`
		Msg  string      `json:"msg"`
	}{}
	switch len(ps) {
	case 3:
		rsp.Code = ps[2].(int)
		fallthrough
	case 2:
		switch ps[1].(type) {
		case string:
			rsp.Msg = ps[1].(string)
		case error:
			rsp.Msg = ps[1].(error).Error()
		}
		fallthrough
	case 1:
		rsp.Data = ps[0]
	}
	c.Data["json"] = rsp
	c.ServeJSON()
	c.StopRun()
}

// ResponseError 响应错误信息，支持多种类型参数（int状态码、error错误、string描述）
func (c *BaseController) ResponseError(ps ...interface{}) *BaseController {
	if ps[0] == nil {
		return c
	}
	var text = ""

	for _, p := range ps {
		switch t := p.(type) {
		case int:
			break
		case error:
			text = t.Error()
			break
		case string:
			text = t
			break
		}
	}
	c.Response(nil, text, 1)
	return nil
}

// Logined 管理员登录验证，未登录则重定向到首页
func (c *BaseController) Logined() *BaseController {
	if v := c.GetSession("token"); v == nil {
		c.Ctx.Redirect(302, "/")
		c.StopRun()
	} else {
		c.PtPin = v.(string)
		c.Master = true
	}
	return c
}

// PortalLogined 门户用户登录验证，从Session中恢复用户账号信息
func (c *BaseController) PortalLogined() *BaseController {
	v := c.GetSession("portal_account_id")
	if v == nil {
		c.Ctx.Redirect(302, "/portal/login")
		c.StopRun()
		return c
	}

	var accountID int
	switch t := v.(type) {
	case int:
		accountID = t
	case int64:
		accountID = int(t)
	case float64:
		accountID = int(t)
	default:
		c.Ctx.Redirect(302, "/portal/login")
		c.StopRun()
		return c
	}

	account, err := models.GetWebUserAccountByID(accountID)
	if err != nil {
		if vNum := c.GetSession("portal_user_number"); vNum != nil {
			var userNumber int
			switch n := vNum.(type) {
			case int:
				userNumber = n
			case int64:
				userNumber = int(n)
			case float64:
				userNumber = int(n)
			}
			if userNumber > 0 {
				if fallback, ferr := models.GetWebUserAccountByUserNumber(userNumber); ferr == nil {
					c.SetSession("portal_account_id", fallback.ID)
					account = fallback
					err = nil
				}
			}
		}
	}
	if err != nil {
		c.DelSession("portal_account_id")
		c.DelSession("portal_user_number")
		c.Ctx.Redirect(302, "/portal/login")
		c.StopRun()
		return c
	}
	c.PortalUserID = account.UserNumber
	c.PortalAccount = account
	return c
}

// ResolveRequestClientContext 解析当前 HTTP 请求来源（Portal 通用；NextPrepare 已赋值时直接返回）
func (c *BaseController) ResolveRequestClientContext() models.ClientContext {
	if !c.ClientCtx.IsZero() {
		return c.ClientCtx
	}
	return models.ResolveClientContext(
		c.Ctx.Input.Header("X-Request-Source"),
		c.Ctx.Input.Header("User-Agent"),
		c.Ctx.Input.Header("X-Sign-DeviceID"),
		c.Ctx.Input.Header("X-Client-Platform"),
	)
}

// RecordPortalCoin Portal 积分变动（自动附带来源，新接口优先使用）
func (c *BaseController) RecordPortalCoin(userNumber, amount int, typ, detail string) {
	models.RecordCoinLogEx(userNumber, amount, typ, detail, c.ResolveRequestClientContext())
}

// RecordPortalEvent Portal 非积分操作来源记录（酷我、反馈等）
func (c *BaseController) RecordPortalEvent(eventType string) {
	models.RecordClientSourceEvent(c.PortalUserID, eventType, c.ResolveRequestClientContext())
}

// Validate 表单验证，将请求体JSON反序列化到指定结构体并进行校验
func (c *BaseController) Validate(ps interface{}) *BaseController {
	c.ResponseError(json.Unmarshal(c.Ctx.Input.CopyBody(10000000), ps), http.StatusBadRequest)
	if err := validate.Struct(ps); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			c.ResponseError(err.Translate(trans), http.StatusBadRequest)
		}
	}
	return c
}

// GetPathInt64 从URL路径参数中获取int64类型的值
func (c *BaseController) GetPathInt64(v string) int64 {
	r := c.Ctx.Input.Param(":" + v)
	if r == "" {
		return 0
	}
	i, err := strconv.Atoi(r)
	c.ResponseError(err)
	return int64(i)
}

// GetPathInt 从URL路径参数中获取int类型的值
func (c *BaseController) GetPathInt(v string) int {
	r := c.Ctx.Input.Param(":" + v)
	if r == "" {
		return 0
	}
	i, err := strconv.Atoi(r)
	c.ResponseError(err)
	return i
}

// GetPathInt32 从URL路径参数中获取int32类型的值
func (c *BaseController) GetPathInt32(v string) int32 {
	r := c.Ctx.Input.Param(":" + v)
	if r == "" {
		return 0
	}
	i, err := strconv.Atoi(r)
	c.ResponseError(err)
	return int32(i)
}

// GetQueryInt64 从URL查询参数中获取int64类型的值
func (c *BaseController) GetQueryInt64(v string) int64 {
	r := c.GetString(v)
	if r == "" {
		return 0
	}
	i, err := strconv.Atoi(r)
	c.ResponseError(err)
	return int64(i)
}

// GetQueryInt 从URL查询参数中获取int类型的值
func (c *BaseController) GetQueryInt(v string) int {
	r := c.GetString(v)
	if r == "" {
		return 0
	}
	i, err := strconv.Atoi(r)
	c.ResponseError(err)
	return i
}

// GetQueryInt32 从URL查询参数中获取int32类型的值
func (c *BaseController) GetQueryInt32(v string) int32 {
	r := c.GetString(v)
	if r == "" {
		return 0
	}
	i, err := strconv.Atoi(r)
	c.ResponseError(err)
	return int32(i)
}
