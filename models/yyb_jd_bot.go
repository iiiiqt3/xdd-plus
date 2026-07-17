package models

var (
	yybJdList              = make(map[int]chan string)
	yybJdLoginHandler      func(sender *Sender, msg chan string)
	yybOfflineNotifyHandler func()
)

// SetYybJdLoginHandler 注册应用宝京东机器人登录处理（由 yybportal 在启动时注入）
func SetYybJdLoginHandler(h func(sender *Sender, msg chan string)) {
	yybJdLoginHandler = h
}

func handleYybJdLogin(sender *Sender, msg chan string) {
	defer delete(yybJdList, sender.UserID)
	if yybJdLoginHandler == nil {
		sender.Reply("❌ 应用宝服务不可用，请稍后再试\n已退出登录流程。")
		return
	}
	yybJdLoginHandler(sender, msg)
}

// StartYybJdLogin 启动应用宝京东登录流程
func StartYybJdLogin(sender *Sender) {
	c2 := make(chan string)
	yybJdList[sender.UserID] = c2
	go handleYybJdLogin(sender, c2)
}

// SetYybOfflineNotifyHandler 注册应用宝掉线检测推送（由 yybportal 启动时注入）
func SetYybOfflineNotifyHandler(h func()) {
	yybOfflineNotifyHandler = h
}

// RunYybOfflineNotify 触发应用宝掉线检测与推送
func RunYybOfflineNotify() {
	if yybOfflineNotifyHandler != nil {
		yybOfflineNotifyHandler()
	}
}
