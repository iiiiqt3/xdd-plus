package controllers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/Login"
	"wechatdll/srv"
	"wechatdll/srv/wxcore"

	"github.com/golang/protobuf/proto"
)

// WxApiController 对接xdd后端的微信协议接口控制器
// 提供与xdd wxcode.go / wx_portal.go 期望的完全一致的API
type WxApiController struct {
	BaseController
}

// ==================== 通用响应结构 ====================

type wxBaseResp struct {
	Status  bool        `json:"status"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type wxLoginCodeData struct {
	QrBase64 string `json:"qrbase64"`
	Uuid     string `json:"uuid"`
}

type wxLoginStatusData struct {
	Uuid        string `json:"uuid"`
	Status      int    `json:"status"`
	Nickname    string `json:"nickname"`
	Wxid        string `json:"wxid"`
	HeadImg     string `json:"headimgurl"`
	ExpiredTime int    `json:"expiredtime"`
}

type wxLoginStatusUser struct {
	Wxid     string `json:"wxid"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type wxLoginStatusResp struct {
	Status bool              `json:"status"`
	Code   int               `json:"code"`
	Msg    string            `json:"msg"`
	Data   wxLoginStatusData `json:"data"`
	User   wxLoginStatusUser `json:"user"`
}

type wxDeviceInfo struct {
	Wxid        string `json:"wxid"`
	Avatar      string `json:"avatar"`
	Nickname    string `json:"nickname"`
	Device      string `json:"device"`
	Survival    int    `json:"survival"`
	LoginDate   int64  `json:"loginDate"`
	RefreshDate int64  `json:"refreshDate"`
}

// ==================== 请求结构 ====================

type wxLoginCodeReq struct {
	DeviceID   string          `json:"DeviceID"`
	DeviceName string          `json:"DeviceName"`
	DeviceType string          `json:"DeviceType"`
	Proxy      models.ProxyInfo `json:"Proxy"`
}

type wxUuidReq struct {
	Uuid string `json:"uuid"`
}

type wxWxidReq struct {
	Wxid  string          `json:"wxid"`
	Proxy models.ProxyInfo `json:"Proxy"`
}

type wxDeleteReq struct {
	Wxids []string `json:"wxids"`
}

// ==================== 辅助函数 ====================

// getQRCarResult 调用GetQRCODECar并提取结果
func getQRCarResult(proxy models.ProxyInfo, deviceID, deviceName string) (qrBase64 string, uuid string, err error) {
	getQRReq := Login.GetQRReq{
		Proxy:      proxy,
		DeviceID:   deviceID,
		DeviceName: deviceName,
	}

	// 直接调用底层函数，手动处理返回
	D, _ := comm.GetLoginataByDevId(getQRReq.DeviceID)
	reqDataLogin := Login.DataLogin{
		UserName:   "",
		Data62:     "",
		DeviceName: getQRReq.DeviceName,
		DeviceId:   getQRReq.DeviceID,
		Proxy:      getQRReq.Proxy,
	}
	if D == nil || D.Wxid == "" || D.ClientVersion != Algorithm.CarVersion {
		D = Login.GenCarLoginData(reqDataLogin)
	} else {
		D = Login.UpdateCarLoginData(D, reqDataLogin)
	}

	httpclient, MmtlsClient, err := comm.MmtlsInitialize(getQRReq.Proxy, Algorithm.MmtlsShortHost)
	if err != nil {
		return "", "", fmt.Errorf("MMTLS初始化失败：%v", err)
	}
	D.Aeskey = []byte(baseutils.RandSeq(16))
	Login.FpInitAndRrefresh(D, httpclient)
	if D.DeviceToken == nil {
		D.DeviceToken = &mm.TrustResponse{}
	}

	req := &mm.GetLoginQRCodeRequest{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    []byte{},
			Uin:           proto.Uint32(0),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		RandomEncryKey: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(D.Aeskey))),
			Buffer: D.Aeskey,
		},
		Opcode:           proto.Uint32(0),
		MsgContextPubKey: nil,
	}

	reqdata, err := proto.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("序列化失败：%v", err)
	}

	hec := Login.InitHec(D)
	hypack := hec.HybridEcdhPackIosEn(502, 0, nil, reqdata)
	recvData, err := httpclient.MMtlsPost(D.ShortHost, "/cgi-bin/micromsg-bin/getloginqrcode", hypack, getQRReq.Proxy)
	if err != nil {
		return "", "", fmt.Errorf("请求失败：%v", err)
	}

	ph1 := hec.HybridEcdhPackIosUn(recvData)
	getloginQRRes := mm.GetLoginQRCodeResponse{}
	err = proto.Unmarshal(ph1.Data, &getloginQRRes)
	if err != nil {
		return "", "", fmt.Errorf("反序列化失败：%v", err)
	}

	if getloginQRRes.GetBaseResponse().GetRet() != 0 {
		return "", "", fmt.Errorf("获取二维码失败")
	}

	if getloginQRRes.Uuid == nil || *getloginQRRes.Uuid == "" {
		return "", "", fmt.Errorf("取码过于频繁")
	}

	D.Uuid = getloginQRRes.GetUuid()
	D.NotifyKey = getloginQRRes.GetNotifyKey().GetBuffer()
	D.Cooike = ph1.Cookies
	D.MmtlsKey = MmtlsClient
	comm.CreateLoginData(D, "", 300, nil)

	if getloginQRRes.GetQrcode() == nil || len(getloginQRRes.GetQrcode().GetBuffer()) == 0 {
		return "", "", fmt.Errorf("二维码数据为空")
	}
	qrBase64 = fmt.Sprintf("data:image/jpg;base64,%s",
		base64.StdEncoding.EncodeToString(getloginQRRes.GetQrcode().GetBuffer()))
	uuid = getloginQRRes.GetUuid()

	return qrBase64, uuid, nil
}

// ==================== 接口实现 ====================

// WxLoginCode 获取登录二维码（Car协议）
// 对应 xdd 调用: POST /api/v1/wx/login/code
// @router /LoginCode [post]
func (c *WxApiController) WxLoginCode() {
	var req wxLoginCodeReq
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "请求格式错误: " + err.Error()}
		c.ServeJSON()
		return
	}

	qrBase64, uuid, err := getQRCarResult(req.Proxy, req.DeviceID, req.DeviceName)
	if err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: err.Error()}
		c.ServeJSON()
		return
	}

	c.Data["json"] = wxBaseResp{
		Status:  true,
		Success: true,
		Message: "成功",
		Data: wxLoginCodeData{
			QrBase64: qrBase64,
			Uuid:     uuid,
		},
	}
	c.ServeJSON()
}

// WxLoginStatus 检查扫码状态
// 对应 xdd 调用: POST /api/v1/wx/login/status
// @router /LoginStatus [post]
func (c *WxApiController) WxLoginStatus() {
	var req wxUuidReq
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "请求格式错误"}
		c.ServeJSON()
		return
	}

	if req.Uuid == "" {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "缺少uuid参数"}
		c.ServeJSON()
		return
	}

	result := checkUuidForXdd(req.Uuid)
	c.Data["json"] = result
	c.ServeJSON()
}

// checkUuidForXdd 检查扫码状态，返回xdd期望的格式
func checkUuidForXdd(uuid string) interface{} {
	D, err := comm.GetLoginata(uuid, nil)
	if err != nil || D == nil || D.Uuid == "" {
		return wxLoginStatusResp{
			Status: false,
			Code:   -1,
			Msg:    "未找到登录信息",
		}
	}

	timenow := uint32(time.Now().Unix())
	req := &mm.CheckLoginQRCodeRequest{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Aeskey,
			Uin:           proto.Uint32(0),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		RandomEncryKey: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(D.Aeskey))),
			Buffer: D.Aeskey,
		},
		Uuid:      &D.Uuid,
		TimeStamp: &timenow,
		Opcode:    proto.Uint32(0),
	}

	reqdata, err := proto.Marshal(req)
	if err != nil {
		return wxLoginStatusResp{Status: false, Code: -1, Msg: "序列化失败: " + err.Error()}
	}

	hec := &Algorithm.Client{}
	hec.Init("IOS")
	hecData := hec.HybridEcdhPackIosEn(503, 0, nil, reqdata)

	// 重新初始化Mmtls（因为D.MmtlsKey从Redis反序列化后内部加密状态丢失）
	httpclient, freshMmtlsKey, mmtlsErr := comm.MmtlsInitialize(D.Proxy, Algorithm.MmtlsShortHost)
	if mmtlsErr != nil {
		return wxLoginStatusResp{Status: false, Code: -1, Msg: "MMTLS初始化失败: " + mmtlsErr.Error()}
	}
	D.MmtlsKey = freshMmtlsKey
	comm.CreateLoginData(D, uuid, 300, nil)

	recvData, err := httpclient.MMtlsPost(Algorithm.MmtlsShortHost, "/cgi-bin/micromsg-bin/checkloginqrcode", hecData, D.Proxy)
	if err != nil {
		return wxLoginStatusResp{Status: false, Code: -1, Msg: "请求失败: " + err.Error()}
	}

	ph1 := hec.HybridEcdhPackIosUn(recvData)
	checkloginQRRes := mm.CheckLoginQRCodeResponse{}
	err = proto.Unmarshal(ph1.Data, &checkloginQRRes)
	if err != nil {
		return wxLoginStatusResp{Status: false, Code: -1, Msg: "反序列化失败: " + err.Error()}
	}

	if checkloginQRRes.GetBaseResponse().GetRet() != 0 {
		return wxLoginStatusResp{Status: false, Code: -1, Msg: "接口返回错误"}
	}

	if checkloginQRRes.GetNotifyPkg().GetNotifyData().GetBuffer() == nil {
		// 等待扫码
		return wxLoginStatusResp{
			Status: true,
			Code:   0,
			Data: wxLoginStatusData{
				Uuid:   uuid,
				Status: 0,
			},
		}
	}

	notifydata := Algorithm.AesDecrypt(checkloginQRRes.GetNotifyPkg().GetNotifyData().GetBuffer(), D.NotifyKey)
	notifydataRsp := mm.LoginQRCodeNotify{}
	if err := proto.Unmarshal(notifydata, &notifydataRsp); err != nil {
		return wxLoginStatusResp{Status: false, Code: -1, Msg: "解包异常"}
	}

	status := notifydataRsp.GetStatus()

	switch status {
	case 1:
		// 已扫码待确认
		return wxLoginStatusResp{
			Status: true,
			Code:   1,
			Data: wxLoginStatusData{
				Uuid:     uuid,
				Status:   1,
				Nickname: notifydataRsp.GetNickName(),
				Wxid:     notifydataRsp.GetUserName(),
			},
		}
	case 2:
		// 已确认，执行登录
		D.Wxid = notifydataRsp.GetUserName()
		D.Pwd = notifydataRsp.GetPwd()
		D.Cooike = ph1.Cookies
		D.HeadUrl = notifydataRsp.GetHeadImgUrl()

		loginResult := Login.CheckSecManualAuth(D, D.ShortHost)
		if loginResult.Success {
			wxid := D.Wxid
			nickname := D.NickName
			if nickname == "" {
				nickname = "微信用户"
			}
			if loginRes, ok := loginResult.Data.(*mm.UnifyAuthResponse); ok {
				wxid = loginRes.GetAcctSectResp().GetUserName()
				nickname = loginRes.GetAcctSectResp().GetNickName()
			}

			// 登录成功后自动开启心跳
			go startAutoHeartBeat(wxid)

			return wxLoginStatusResp{
				Status: true,
				Code:   2,
				User: wxLoginStatusUser{
					Wxid:     wxid,
					Nickname: nickname,
					Avatar:   D.HeadUrl,
				},
			}
		}
		return wxLoginStatusResp{Status: false, Code: -1, Msg: "登录失败: " + loginResult.Message}
	default:
		return wxLoginStatusResp{
			Status: true,
			Code:   0,
			Data: wxLoginStatusData{
				Uuid:   uuid,
				Status: int(status),
			},
		}
	}
}

// WxLoginAgain 重新登录（复用已有设备ID生成新二维码）
// 对应 xdd 调用: POST /api/v1/wx/login/again
// @router /LoginAgain [post]
func (c *WxApiController) WxLoginAgain() {
	var req wxWxidReq
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "请求格式错误"}
		c.ServeJSON()
		return
	}

	if req.Wxid == "" {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "缺少wxid参数"}
		c.ServeJSON()
		return
	}

	D, err := comm.GetLoginata(req.Wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "未找到登录数据"}
		c.ServeJSON()
		return
	}

	qrBase64, uuid, err := getQRCarResult(req.Proxy, D.Deviceid_str, D.DeviceName)
	if err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: err.Error()}
		c.ServeJSON()
		return
	}

	c.Data["json"] = wxBaseResp{
		Status:  true,
		Success: true,
		Message: "成功",
		Data: wxLoginCodeData{
			QrBase64: qrBase64,
			Uuid:     uuid,
		},
	}
	c.ServeJSON()
}

// WxLoginAwake 唤醒登录
// 对应 xdd 调用: POST /api/v1/wx/login/awake
// @router /LoginAwake [post]
func (c *WxApiController) WxLoginAwake() {
	var req wxWxidReq
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "请求格式错误"}
		c.ServeJSON()
		return
	}

	if req.Wxid == "" {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "缺少wxid参数"}
		c.ServeJSON()
		return
	}

	awakenReq := Login.AwakenReq{
		Wxid:  req.Wxid,
		Proxy: req.Proxy,
	}

	result := Login.AwakenLoginNew(awakenReq)

	c.Data["json"] = wxBaseResp{
		Status:  result.Success,
		Success: result.Success,
		Message: result.Message,
	}
	c.ServeJSON()
}

// WxLoginTwice 唤醒后二次登录（获取唤醒登录二维码）
// 对应 xdd 调用: POST /api/v1/wx/login/twice
// @router /LoginTwice [post]
func (c *WxApiController) WxLoginTwice() {
	var req wxWxidReq
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "请求格式错误"}
		c.ServeJSON()
		return
	}

	if req.Wxid == "" {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "缺少wxid参数"}
		c.ServeJSON()
		return
	}

	// 执行唤醒
	awakenReq := Login.AwakenReq{
		Wxid:  req.Wxid,
		Proxy: req.Proxy,
	}
	awakenResult := Login.AwakenLoginNew(awakenReq)
	if !awakenResult.Success {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "唤醒失败: " + awakenResult.Message}
		c.ServeJSON()
		return
	}

	// 唤醒成功后，从登录数据中获取新的UUID
	D, err := comm.GetLoginata(req.Wxid, nil)
	if err != nil || D == nil || D.Uuid == "" {
		c.Data["json"] = wxBaseResp{Status: true, Success: true, Message: "唤醒成功，但未获取到UUID"}
		c.ServeJSON()
		return
	}

	// 唤醒登录不返回二维码图片，推送到已登录设备让用户确认
	// UUID用于轮询状态
	c.Data["json"] = wxBaseResp{
		Status:  true,
		Success: true,
		Message: "成功",
		Data: wxLoginCodeData{
			QrBase64: "",
			Uuid:     D.Uuid,
		},
	}
	c.ServeJSON()
}

// WxLoginLogout 登出
// 对应 xdd 调用: POST /api/v1/wx/login/logout
// @router /LoginLogout [post]
func (c *WxApiController) WxLoginLogout() {
	var req wxWxidReq
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "请求格式错误"}
		c.ServeJSON()
		return
	}

	if req.Wxid == "" {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "缺少wxid参数"}
		c.ServeJSON()
		return
	}

	// 停止自动心跳
	stopAutoHeartBeat(req.Wxid)

	result := Login.LogOut(req.Wxid)

	c.Data["json"] = wxBaseResp{
		Status:  result.Success,
		Success: result.Success,
		Message: result.Message,
	}
	c.ServeJSON()
}

// WxUserStatus 查询所有微信设备状态
// 对应 xdd 调用: GET /api/v1/wx/user/status
// @router /UserStatus [get]
func (c *WxApiController) WxUserStatus() {
	devices := make(map[string]wxDeviceInfo)

	wxids, err := comm.GetAllLoggedInWxids()
	if err != nil || len(wxids) == 0 {
		c.Data["json"] = map[string]interface{}{
			"status":  true,
			"data":    devices,
			"message": "暂无设备",
		}
		c.ServeJSON()
		return
	}

	now := time.Now().Unix()
	for _, wxid := range wxids {
		D, err := comm.GetLoginata(wxid, nil)
		if err != nil || D == nil || D.Wxid == "" {
			continue
		}

		survival := 0
		if len(D.Sessionkey) > 0 && D.LoginDate > 0 {
			if D.RefreshTokenDate > 0 && (now-D.RefreshTokenDate) < 600 {
				// 心跳活跃（最近10分钟内有心跳）→ 在线
				survival = 1
			} else if D.RefreshTokenDate == 0 && (now-D.LoginDate) < 600 {
				// 刚登录还没有心跳记录（10分钟内）→ 在线
				survival = 1
			}
			// 其他情况：心跳超时或登录时间过久 → 离线
		}

		devices[wxid] = wxDeviceInfo{
			Wxid:        wxid,
			Avatar:      D.HeadUrl,
			Nickname:    D.NickName,
			Device:      D.DeviceName,
			Survival:    survival,
			LoginDate:   D.LoginDate,
			RefreshDate: D.RefreshTokenDate,
		}
	}

	c.Data["json"] = map[string]interface{}{
		"status":  true,
		"data":    devices,
		"message": "成功",
	}
	c.ServeJSON()
}

// WxUserDelete 删除微信设备数据
// 对应 xdd 调用: POST /api/v1/wx/user/delete
// @router /UserDelete [post]
func (c *WxApiController) WxUserDelete() {
	var req wxDeleteReq
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "请求格式错误"}
		c.ServeJSON()
		return
	}

	if len(req.Wxids) == 0 {
		c.Data["json"] = wxBaseResp{Status: false, Success: false, Message: "缺少wxids参数"}
		c.ServeJSON()
		return
	}

	for _, wxid := range req.Wxids {
		// 停止自动心跳（如果在运行）
		stopAutoHeartBeat(wxid)

		// 获取登录数据以清理 devId 映射
		D, _ := comm.GetLoginata(wxid, nil)
		if D != nil && D.Deviceid_str != "" {
			comm.RedisClient.Del("devId:" + D.Deviceid_str)
		}

		// 删除登录数据
		comm.DelLoginata(wxid)
		// 删除心跳日志
		comm.RedisClient.Del("AutoHeartBeatList:" + wxid)
		// 从设备索引移除
		comm.RemoveWxidFromIndex(wxid)
	}

	c.Data["json"] = wxBaseResp{Status: true, Success: true, Message: "删除成功"}
	c.ServeJSON()
}

// stopAutoHeartBeat 停止自动心跳
func stopAutoHeartBeat(wxid string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[WxApi] 停止心跳异常: wxid=%s, err=%v\n", wxid, r)
		}
	}()
	wxConnectMgr := wxcore.GetWXConnectMgr()
	wXConnect := wxConnectMgr.GetWXConnectByWXID(wxid)
	if wXConnect != nil {
		wXConnect.Stop()
	}
	comm.AutoHeartBeatListClear(wxid)
}

// startAutoHeartBeat 启动自动心跳（异步调用）
func startAutoHeartBeat(wxid string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[WxApi] 启动心跳异常: wxid=%s, err=%v\n", wxid, r)
		}
	}()

	D, err := comm.GetLoginata(wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		fmt.Printf("[WxApi] 启动心跳失败: 未找到登录数据 wxid=%s\n", wxid)
		return
	}

	wxConnectMgr := wxcore.GetWXConnectMgr()
	wXConnect := wxConnectMgr.GetWXConnectByWXID(wxid)
	if wXConnect == nil {
		wxAccount := srv.NewWXAccount(D)
		wXConnect = wxcore.NewWXConnect(wxConnectMgr, wxAccount)
		wxConnectMgr.Add(wXConnect)
	}
	wXConnect.Start()
	if err := wXConnect.SendHeartBeat(); err != nil {
		fmt.Printf("[WxApi] 发送心跳失败: wxid=%s, err=%v\n", wxid, err)
	} else {
		fmt.Printf("[WxApi] 心跳已启动: wxid=%s\n", wxid)
	}
}
