package models

func (ck JdCookie) Push(msg string) {
	if ck.PtPin != "" {
		go SendQQ(ck.QQ, msg)
		//go SendWxMsg()
		go pushPlus(ck.PushPlus, msg)
		go SendTgMsg(ck.Telegram, msg)
	} else {
		go SendQQ(Config.QQID, msg)
		go qywxNotify(&QywxConfig{QywxKey: Config.QywxKey, Content: msg})
		go SendTgMsg(Config.TelegramUserID, msg)
	}
}
