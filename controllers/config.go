package controllers

import (
	"encoding/json"
	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/xdd/models"
)

type ConfigController struct {
	BaseController
}

func (c *ConfigController) NextPrepare() {
	c.Logined()
}

func (c *ConfigController) ListConfig() {

	var config = models.ListConfig()
	//marshal, _ := json.Unmarshal(config)
	logs.Info(config)

	c.Data["json"] = map[string]interface{}{
		"code": 200,
		"data": config,
	}
	c.ServeJSON()
}

func (c *ConfigController) CreateOrUpdateConfig() {

	var sys models.SystemConfig
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &sys)
	if err != nil {
		c.Ctx.Output.SetStatus(400)
		c.Ctx.Output.Body([]byte("Invalid JSON data"))
		return
	}

	// 处理用户数据
	// ...

	msg := models.SaveSysConfig(sys)
	logs.Info(msg)

	c.Ctx.Output.SetStatus(200)
	c.Ctx.Output.Body([]byte("User data saved"))

}
