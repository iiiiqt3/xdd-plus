package controllers

import (
	"encoding/json"
	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/xdd/models"
)

type QQController struct {
	BaseController
}

type CqMessage struct {
	PostType    string `json:"post_type"`
	MessageType string `json:"message_type"`
	Time        int    `json:"time"`
	SelfID      int    `json:"self_id"`
	SubType     string `json:"sub_type"`
	Font        int    `json:"font"`
	Sender      struct {
		Age      int    `json:"age"`
		Nickname string `json:"nickname"`
		Sex      string `json:"sex"`
		UserID   int    `json:"user_id"`
	} `json:"sender"`
	MessageID  int    `json:"message_id"`
	UserID     int    `json:"user_id"`
	TargetID   int    `json:"target_id"`
	Message    string `json:"message"`
	RawMessage string `json:"raw_message"`
}

func (c *QQController) HandleQQMessage() {
	data := c.Ctx.Input.RequestBody
	logs.Info(string(data))
	var msg CqMessage
	err := json.Unmarshal(data, &msg)
	if err != nil {
		logs.Error(err)
	}
	if msg.PostType == "message" {
		logs.Info("接收到信息" + msg.RawMessage)
		models.ListenQQPrivateMessage(int64(msg.UserID), msg.RawMessage)
	}
}
