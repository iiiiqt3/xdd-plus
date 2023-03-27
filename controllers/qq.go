package controllers

import (
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
	MessageSeq  int    `json:"message_seq"`
	UserID      int    `json:"user_id"`
	GroupID     int    `json:"group_id"`
	Message     string `json:"message"`
	RawMessage  string `json:"raw_message"`
	Sender      struct {
		Age      int    `json:"age"`
		Area     string `json:"area"`
		Card     string `json:"card"`
		Level    string `json:"level"`
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
		Sex      string `json:"sex"`
		Title    string `json:"title"`
		UserID   int    `json:"user_id"`
	} `json:"sender"`
	MessageID int         `json:"message_id"`
	Anonymous interface{} `json:"anonymous"`
	Font      int         `json:"font"`
}

func HandleQQMessage(msg CqMessage) {
	if msg.PostType == "message" {
		logs.Info("接收到信息" + msg.RawMessage)
		if msg.MessageType == "private" {
			models.ListenQQPrivateMessage(msg.UserID, msg.Message)
		} else if msg.MessageType == "group" {
			models.ListenQQGroupMessage(msg.UserID, msg.GroupID, msg.Message)
		}
	}

}
