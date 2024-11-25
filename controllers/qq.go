package controllers

import (
	"encoding/json"
	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/xdd/models"
	"github.com/gorilla/websocket"
	"regexp"
)

var upgrader = websocket.Upgrader{}
var ws *websocket.Conn
var mt int

type QQController struct {
	BaseController
}

type LLMessage struct {
	SelfId      int         `json:"self_id"`
	UserId      json.Number `json:"user_id"`
	Time        int         `json:"time"`
	MessageId   int         `json:"message_id"`
	RealId      int         `json:"real_id"`
	MessageSeq  int         `json:"message_seq"`
	MessageType string      `json:"message_type"`
	RequestType string      `json:"request_type"`
	Sender      struct {
		UserId   string `json:"user_id"`
		Nickname string `json:"nickname"`
		Card     string `json:"card"`
		Role     string `json:"role"`
	} `json:"sender"`
	RawMessage    string `json:"raw_message"`
	Font          int    `json:"font"`
	SubType       string `json:"sub_type"`
	Message       string `json:"message"`
	MessageFormat string `json:"message_format"`
	PostType      string `json:"post_type"`
	GroupId       int    `json:"group_id"`
	Flag          string `json:"flag"`
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
			//logs.Info("read:", err)
			logs.Info("ws连接已断开")
			return
		}
		var msg LLMessage
		err = json.Unmarshal(message, &msg)
		if err != nil {
			logs.Info(string(message))
			logs.Info("change:", err)
			break
		}
		go HandleQQMessage(msg)

	}

}

func HandleQQMessage(msg LLMessage) {
	if msg.PostType == "message_sent" {
		logs.Info("接收到信息" + msg.RawMessage)
		if msg.MessageType == "private" {
			models.ListenQQPrivateMessage(string(msg.UserId), msg.RawMessage)
		} else if msg.MessageType == "group" {
			models.ListenQQGroupMessage(string(msg.UserId), msg.GroupId, msg.RawMessage)
			//撤回手机号码
			regular := `^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`
			reg := regexp.MustCompile(regular)
			if reg.MatchString(msg.RawMessage) {
				logs.Info("识别为手机号，进行撤回")
				models.DeleteQQMsg(msg.MessageId)
			}
		}
	} else if msg.PostType == "post_type" {
		if msg.RequestType == "friend" {
			logs.Info(msg.Flag)
			models.AutoAgreeFriedns(msg.Flag)
		}
	}
}
