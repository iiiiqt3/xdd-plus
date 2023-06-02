package controllers

import (
	"encoding/json"
	"fmt"
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
