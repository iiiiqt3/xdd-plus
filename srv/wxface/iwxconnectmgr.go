package wxface

// IWXConnectMgr 微信链接管理器
type IWXConnectMgr interface {
	Add(wxConnect IWXConnect)                  // 添加链接
	GetWXConnectByWXID(wxid string) IWXConnect // 根据WXID获取微信链接
	Remove(wxconn IWXConnect)                  // 删除连接
	ClearWXConn()                              // 删除并停止所有链接
	ShowConnectInfo() string                   // 打印链接数量
	InitAutoHeartBeat()                        // 初始化自动心跳
}
