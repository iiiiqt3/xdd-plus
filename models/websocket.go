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

func WriteMsg(msg chan []byte) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}
		if ws != nil {
			err := ws.WriteMessage(mt, n)
			if err != nil {
				logs.Info("write:", err)
			}
		} else {
			time.Sleep(time.Second * time.Duration(6))
			logs.Info("等待ws连接")
		}
	}
}
