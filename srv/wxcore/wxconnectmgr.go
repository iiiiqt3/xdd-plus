package wxcore

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"wechatdll/comm"
	"wechatdll/srv"
	"wechatdll/srv/wxface"
)

// WXConnectMgr 微信链接管理器
type WXConnectMgr struct {
	wxConnectMap map[string]wxface.IWXConnect //管理的连接信息
	wxConnLock   sync.RWMutex                 //读写连接的读写锁
}

// Add 添加链接
func (wm *WXConnectMgr) Add(wxConnect wxface.IWXConnect) {
	wm.wxConnLock.Lock()
	defer wm.wxConnLock.Unlock()
	wm.wxConnectMap[wxConnect.GetWXAccount().GetUserInfo().Wxid] = wxConnect
	// 打印链接数量
	go wm.ShowConnectInfo()
}

// GetWXConnectByWXID 根据WXID获取微信链接
func (wm *WXConnectMgr) GetWXConnectByWXID(wxid string) wxface.IWXConnect {
	//保护共享资源Map 加读锁
	wm.wxConnLock.RLock()
	defer wm.wxConnLock.RUnlock()
	//根据WXID获取微信链接
	for _, wxConn := range wm.wxConnectMap {
		tmpUserInfo := wxConn.GetWXAccount().GetUserInfo()
		if tmpUserInfo == nil || strings.Compare(tmpUserInfo.Wxid, wxid) != 0 {
			continue
		}
		return wxConn
	}
	return nil
}

func (wm *WXConnectMgr) ShowConnectInfo() string {
	totalNum := len(wm.wxConnectMap)
	showText := time.Now().Format("2006-01-02 15:04:05")
	showText = showText + " 总链接数量: " + strconv.Itoa(totalNum)
	fmt.Println(showText)
	return showText
}

// Remove 删除连接
func (wm *WXConnectMgr) Remove(wxconn wxface.IWXConnect) {
	wm.wxConnLock.Lock()
	//删除
	currentUserInfo := wxconn.GetWXAccount().GetUserInfo()
	delete(wm.wxConnectMap, currentUserInfo.Wxid)
	currentUserInfo = nil
	// 打印链接数量
	wm.wxConnLock.Unlock()
	wm.ShowConnectInfo()
}

// ClearWXConn 删除并停止所有链接
func (wm *WXConnectMgr) ClearWXConn() {
	//保护共享资源Map 加写锁
	wm.wxConnLock.Lock()

	//停止并删除全部的连接信息
	for uuid, wxConn := range wm.wxConnectMap {
		//停止
		wxConn.Stop()
		//删除
		delete(wm.wxConnectMap, uuid)
	}

	wm.wxConnLock.Unlock()
	// 打印链接数量
	wm.ShowConnectInfo()
}

var wxConnectMgr *WXConnectMgr = &WXConnectMgr{
	wxConnectMap: make(map[string]wxface.IWXConnect),
}

// 获取 WXConnectMgr 对象
func GetWXConnectMgr() *WXConnectMgr {
	if wxConnectMgr == nil {
		wxConnectMgr = &WXConnectMgr{
			wxConnectMap: make(map[string]wxface.IWXConnect),
		}
	}
	return wxConnectMgr
}

// 初始化自动心跳
func (wm *WXConnectMgr) InitAutoHeartBeat() {
	AutoHeartBeatList := comm.GetAutoHeartBeatList()
	// 记录已经判断的wxid
	var wxidList []string = make([]string, 0)
	for _, v := range AutoHeartBeatList {
		item := v[0]
		// ["[xyuh111],[果汁] 发送心跳成功，下次心跳时间：2025-01-11 17:12:34"]
		// 去第一条是否包含“发送心跳成功”
		if !strings.Contains(item, "下次刷新时间") {
			continue
		}
		// 获取 wxid [xyuh111]
		wxid := strings.Split(item, "[")[1]
		wxid = strings.Split(wxid, "]")[0]
		// 判断 wxid 是否存在
		if strings.Contains(strings.Join(wxidList, ","), wxid) {
			continue
		}
		wxidList = append(wxidList, wxid)
		wXConnect := wm.GetWXConnectByWXID(wxid)
		if wXConnect == nil {
			// redis 取
			userInfo, err := comm.GetLoginata(wxid, nil)
			if err != nil || userInfo == nil || userInfo.Wxid == "" {
				fmt.Println("初始化心跳失败 err:", wxid)
				continue
			}
			wxAccount := srv.NewWXAccount(userInfo)
			wXConnect = NewWXConnect(wxConnectMgr, wxAccount)
		}
		wXConnect.Start()
		go wXConnect.SendHeartBeat()
	}

	// 补充：从Redis索引中恢复通过xdd等外部接口登录的设备
	indexWxids, err := comm.GetAllLoggedInWxids()
	if err == nil {
		for _, wxid := range indexWxids {
			if strings.Contains(strings.Join(wxidList, ","), wxid) {
				continue // 已在心跳日志中恢复过
			}
			userInfo, err := comm.GetLoginata(wxid, nil)
			if err != nil || userInfo == nil || userInfo.Wxid == "" {
				continue
			}
			if len(userInfo.Sessionkey) == 0 {
				continue // 未完成登录
			}
			fmt.Println("[InitAutoHeartBeat] 从索引恢复心跳:", wxid)
			wXConnect := wm.GetWXConnectByWXID(wxid)
			if wXConnect == nil {
				wxAccount := srv.NewWXAccount(userInfo)
				wXConnect = NewWXConnect(wxConnectMgr, wxAccount)
			}
			wXConnect.Start()
			go wXConnect.SendHeartBeat()
		}
	}
}
