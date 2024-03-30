package controllers

import (
	"encoding/json"
	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/xdd/models"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var ws *websocket.Conn
var mt int

type QQController struct {
	BaseController
}

type LLMessage struct {
	SelfId      int    `json:"self_id"`
	UserId      int    `json:"user_id"`
	Time        int    `json:"time"`
	MessageId   int    `json:"message_id"`
	RealId      string `json:"real_id"`
	MessageType string `json:"message_type"`
	Sender      struct {
		UserId   int    `json:"user_id"`
		Nickname string `json:"nickname"`
		Card     string `json:"card"`
	} `json:"sender"`
	RawMessage    string `json:"raw_message"`
	Font          int    `json:"font"`
	SubType       string `json:"sub_type"`
	Message       string `json:"message"`
	MessageFormat string `json:"message_format"`
	PostType      string `json:"post_type"`
	GroupId       int    `json:"group_id"`
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

func (c *QQController) Echo() {
	//服务升级，对于来到的http连接进行服务升级，升级到ws
	var err error
	ws, err = upgrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
	defer ws.Close()
	if err != nil {
		panic(err)
	}
	logs.Info("ws接入成功")

	models.WsInit(ws, 1)

	for {
		//messageType int, p []byte, err error
		nt, message, err := ws.ReadMessage()
		mt = nt
		if err != nil {
			logs.Info("read:", err)
			break
		}
		//val, _ := jsonparser.GetString(message, "echo")
		//if val == "user_id" {
		//	//忽略跳过
		//	return
		//}

		//logs.Info("recv: %s , %d", message, mt)

		var msg LLMessage
		err = json.Unmarshal(message, &msg)
		if err != nil {
			logs.Info("change:", err)
			break
		}
		go HandleQQMessage(msg)

	}
}

func HandleQQMessage(msg LLMessage) {
	if msg.PostType == "message" {
		logs.Info("接收到信息" + msg.RawMessage)
		if msg.MessageType == "private" {
			models.ListenQQPrivateMessage(msg.UserId, msg.RawMessage)
		} else if msg.MessageType == "group" {
			models.ListenQQGroupMessage(msg.UserId, msg.GroupId, msg.RawMessage)
		}
	}
}
