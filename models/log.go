package models

import (
	"fmt"
	"net"
)

const (
	addr = "192.168.195.44:19730"
)

var flag = false
var conn = getConn()

func getConn() net.Conn {
	if flag {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			fmt.Println("连接服务端失败:", err.Error())
			return nil
		}
		fmt.Println("已连接服务器")
		flag = true
		return conn
	}

	return conn
}

func GetLog() string {
	n := getConn()
	n.Write([]byte("getLog"))
	buf := make([]byte, 1024)
	c, err := conn.Read(buf)
	if err != nil {
		fmt.Println("读取服务器数据异常:", err.Error())
	}
	fmt.Println(string(buf[0:c]))
	return string(buf[0:c])
}
