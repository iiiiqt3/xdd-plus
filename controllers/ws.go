package controllers

import (
	"encoding/json"
	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var ws *websocket.Conn
var mt int

type WsController struct {
	BaseController
}

func (c *WsController) Echo() {
	//服务升级，对于来到的http连接进行服务升级，升级到ws
	var err error
	ws, err = upgrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
	//defer ws.Close()
	if err != nil {
		panic(err)
	}
	for {
		//messageType int, p []byte, err error
		nt, message, err := ws.ReadMessage()
		mt = nt
		if err != nil {
			logs.Info("read:", err)
			break
		}
		logs.Info("recv: %s , %d", message, mt)
		var msg CqMessage
		err = json.Unmarshal(message, &msg)
		if err != nil {
			logs.Error(err)
			break
		}
		HandleQQMessage(msg)

		//err = cn.WriteMessage(mt, message)
		//if err != nil {
		//	logs.Info("write:", err)
		//	break
		//}
	}
}

func WriteMsg(msg []byte) {
	err := ws.WriteMessage(mt, msg)
	if err != nil {
		logs.Info("write:", err)
	}
}
