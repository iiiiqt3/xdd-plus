package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/cdle/xdd/models"
)

type ConfigController struct {
	BaseController
}

func (c *AccountController) ListConfig() {

	var config = models.ListConfig()
	marshal, _ := json.Marshal(config)

	c.Data["json"] = map[string]interface{}{
		"code": 200,
		"data": marshal,
	}
	c.ServeJSON()
}

func (c *AccountController) CreateOrUpdateConfig() {

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
