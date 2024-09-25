package models

var ElmList = make(map[string]chan string)

func ELMSelect(sender *Sender, msg chan string, typ int) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}

		if n == "q" {
			ElmList[sender.UserID] = nil
			close(msg)
			return
		}

		switch typ {
		case 0:
			//进入充值序列
			sender.Reply("您已进入充值流程，请转账或回复'q'退出流程!")

		default:
			sender.Reply("暂无对应的渠道,已经退出流程请重新输入")
			ElmList[sender.UserID] = nil
			return
		}

	}
}
