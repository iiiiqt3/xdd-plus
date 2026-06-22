package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/vweb"
)

// AccountController 账号管理控制器，处理京东Cookie的增删改查
type AccountController struct {
	BaseController
}

// NextPrepare 前置处理，验证管理员登录状态
func (c *AccountController) NextPrepare() {
	c.Logined()
}

// List 分页获取京东Cookie列表
func (c *AccountController) List() {
	var page = c.GetQueryInt("page")
	var limit = c.GetQueryInt("limit")
	var cks = models.GetJdCookies()
	var len = len(cks)
	var total = []int{len}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 1
	}
	var from = (page - 1) * limit
	var to = page * limit
	if from >= len-1 {
		from = len - 1
	}
	if to >= len {
		to = len
	}
	if from < 0 {
		from = 0
	}
	var data = cks[from:to]
	c.Data["json"] = map[string]interface{}{
		"code":    200,
		"data":    data,
		"message": total,
	}
	c.ServeJSON()
}

// ListEnvs 分页获取环境变量列表
func (c *AccountController) ListEnvs() {
	var page = c.GetQueryInt("page")
	var limit = c.GetQueryInt("limit")
	var envs = models.GetEnvs()
	var len = len(envs)
	var total = []int{len}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 1
	}
	var from = (page - 1) * limit
	var to = page * limit
	if from >= len-1 {
		from = len - 1
	}
	if to >= len {
		to = len
	}
	if from < 0 {
		from = 0
	}
	var data = envs[from:to]

	c.Data["json"] = map[string]interface{}{
		"code":    200,
		"data":    data,
		"message": total,
	}
	c.ServeJSON()
}

// CreateOrUpdateEnv 创建或更新环境变量
func (c *AccountController) CreateOrUpdateEnv() {
	ps := &models.JdCookie{}
	c.Validate(ps)
	if ps.PtPin != "" {
		ps.Pool = ""
		ps.Updates(*ps)
	}
	go func() {
		models.Save <- &models.JdCookie{}
	}()
	c.Response(nil, "操作成功")
}

// CreateOrUpdate 创建或更新京东Cookie账号
func (c *AccountController) CreateOrUpdate() {
	ps := &models.JdCookie{}
	c.Validate(ps)
	if ps.PtPin != "" {
		ps.Pool = ""
		ps.Updates(*ps)
	}
	go func() {
		models.Save <- &models.JdCookie{}
	}()
	c.Response(nil, "操作成功")
}

// Admin 返回后台管理页面HTML
func (c *AccountController) Admin() {
	file, _ := vweb.ReadFile("html/admin.html")
	c.Ctx.WriteString(string(file))
}

// CreateOrUpdateLoginSelect 处理登录选择配置的创建或更新
func (c *AccountController) CreateOrUpdateLoginSelect() {

	var result map[string]interface{}
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &result)
	if err != nil {
		panic(err)
	}

	for key, value := range result {
		fmt.Printf("%s: %v\n", key, value)
	}

	c.Response(nil, "操作成功")

}
