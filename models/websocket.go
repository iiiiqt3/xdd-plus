package models

import (
	"github.com/beego/beego/v2/core/logs"
	"github.com/gorilla/websocket"
	"time"
)

var ws *websocket.Conn
var mt int

func WsInit(wss *websocket.Conn, nt int) {
	ws = wss
	mt = nt
}

func WriteMsg(msg []byte) {

	for {
		if ws != nil {
			err := ws.WriteMessage(mt, msg)
			if err != nil {
				logs.Info("write:", err)
			}
			break
		} else {
			time.Sleep(time.Second * time.Duration(6))
			logs.Info("等待ws连接")
		}
	}
}
