package controllers

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/cdle/xdd/models"
)

//go:embed /web/*
var WebFs embed.FS

type AccountController struct {
	BaseController
}

func (c *AccountController) NextPrepare() {
	c.Logined()
}

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

func (c *AccountController) Admin() {
	file, _ := WebFs.ReadFile("admin.html")
	c.Ctx.WriteString(string(file))

	//if models.Config.QQID == 764763903 {
	//	logs.Info("下载最新主题")
	//	s, _ := httplib.Get("http://update1.smxy.xyz/admin.html").String()
	//	if s != "" {
	//		c.Ctx.WriteString(s)
	//		return
	//	}
	//	logs.Warn("主题下载失败，使用默认主题")
	//
	//} else {
	//	c.Ctx.WriteString(models.Admin)
	//}
}

//func (c *AccountController) ListLoginSelect() {
//	var page = c.GetQueryInt("page")
//	var limit = c.GetQueryInt("limit")
//	var envs = models.ListLoginSelect()
//	var len = len(envs)
//	var total = []int{len}
//	if page == 0 {
//		page = 1
//	}
//	if limit == 0 {
//		limit = 1
//	}
//	var from = (page - 1) * limit
//	var to = page * limit
//	if from >= len-1 {
//		from = len - 1
//	}
//	if to >= len {
//		to = len
//	}
//	if from < 0 {
//		from = 0
//	}
//	var data = envs[from:to]
//	c.Data["json"] = map[string]interface{}{
//		"code":    200,
//		"data":    data,
//		"message": total,
//	}
//	c.ServeJSON()
//}

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
