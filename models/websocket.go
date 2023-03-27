package models

import (
	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
)

var ws *websocket.Conn
var mt int

func WsInit(wss *websocket.Conn, nt int) {
	ws = wss
	mt = nt
}

func WriteMsg(msg []byte) {
	err := ws.WriteMessage(mt, msg)
	if err != nil {
		logs.Info("write:", err)
	}
}
