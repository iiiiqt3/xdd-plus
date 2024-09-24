package models

import "strings"

var ListenUOSWXPrivateMessage = func(uid string, msg string) {
	rt := handleMessage(msg, "wx", uid)
	switch rt.(type) {
	case string:
		SendWxMsg(uid, rt.(string))
	}
}

var ListenUOSWXGroupMessage = func(uid string, gid string, msg string) {
	if strings.Contains(Config.WXGroupID, gid) || msg == "监听微信群" || msg == "取消监听" {
		rt := handleMessage(msg, "wxg", uid, gid)
		switch rt.(type) {
		case string:
			SendWxGroupMsg(uid, gid, rt.(string))
		}
	}
}
