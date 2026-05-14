package models

func ELMSelect(sender *Sender, msg chan string, typ int) {
	for {
		n, ok := <-msg
		// 说明发送方关闭了channel
		if !ok {
			break
		}

		if n == "q" {
			sender.Reply("已退出流程")
			ElmList[sender.UserID] = nil
			close(msg)
			return
		} else {
			sender.Reply("已退出流程，请重新发送【饿了么充值】指令")
			ElmList[sender.UserID] = nil
			return
		}

		switch typ {
		case 0:
			// 进入充值序列
			sender.Reply("您已进入充值流程，请转账或回复'q'退出流程!")

		default:
			sender.Reply("暂无对应的渠道，请重新发送指令")
			ElmList[sender.UserID] = nil
			return
		}
	}
}