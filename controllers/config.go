package controllers

import (
	"encoding/json"
	"github.com/cdle/xdd/models"
)

// ConfigController 系统配置管理控制器
type ConfigController struct {
	BaseController
}

// NextPrepare 前置处理，验证管理员登录状态
func (c *ConfigController) NextPrepare() {
	c.Logined()
}

// ListConfig 获取系统配置列表
func (c *ConfigController) ListConfig() {

	var config = models.ListConfig()
	models.Admin().Infof("%v", config)

	c.Data["json"] = map[string]interface{}{
		"code": 200,
		"data": config,
	}
	c.ServeJSON()
}

// CreateOrUpdateConfig 创建或更新系统配置
func (c *ConfigController) CreateOrUpdateConfig() {

	var sys models.SystemConfig
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &sys)
	if err != nil {
		c.Ctx.Output.SetStatus(400)
		c.Ctx.Output.Body([]byte("Invalid JSON data"))
		return
	}

	msg := models.SaveSysConfig(sys)
	models.Admin().Infof("%s", msg)

	c.Ctx.Output.SetStatus(200)
	c.Ctx.Output.Body([]byte("User data saved"))

}
