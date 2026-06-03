package controllers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/Login"
	"wechatdll/srv"
	"wechatdll/srv/wxcore"

	"github.com/bitly/go-simplejson"
)

// 登录模块 支持二次 唤醒 62数据登录(注意：代理必须使用SOCKS)
type LoginController struct {
	BaseController
}

// @Summary 获取二维码(iPad)
// @Param	body		body 	Login.GetQRReq	true	"不使用代理请留空"
// @Success 200
// @router /LoginGetQR [post]
func (c *LoginController) LoginGetQR() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.GetQRCODE(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取二维码(iPad-绕过验证码)
// @Param	body		body 	Login.GetQRReq	true		"不使用代理请留空"
// @Success 200
// @router /LoginGetQRx [post]
func (c *LoginController) LoginGetQRx() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.GetQRCODEx(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取二维码(安卓Pad)
// @Param	body		body 	Login.GetQRReq	true		"不使用代理请留空"
// @Success 200
// @router /LoginGetQRPad [post]
func (c *LoginController) LoginGetQRPad() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.GetQRCODEPad(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取二维码(安卓Pad-绕过验证码)
// @Param	body		body 	Login.GetQRReq	true		"不使用代理请留空"
// @Success 200
// @router /LoginGetQRPadx [post]
func (c *LoginController) LoginGetQRPadx() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.GetQRCODEPadx(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取二维码(Windows)
// @Param	body		body 	Login.GetQRReq	true		"不使用代理请留空"
// @Success 200
// @router /LoginGetQRWin [post]
func (c *LoginController) LoginGetQRWin() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.GetQRCODEWin(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取二维码(WindowsUwp-绕过验证码)
// @Param	body		body 	Login.GetQRReq	true		"不使用代理请留空"
// @Success 200
// @router /LoginGetQRWinUwp [post]
func (c *LoginController) LoginGetQRWinUwp() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.GetQRCODEWinUwp(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取二维码(WinUnified-统一PC版)
// @Param	body		body 	Login.GetQRReq	true		"不使用代理请留空"
// @Success 200
// @router /LoginGetQRWinUnified [post]
func (c *LoginController) LoginGetQRWinUnified() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.GetQRCODEWinUnified(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取二维码(Car)
// @Param	body		body 	Login.GetQRReq	true		"不使用代理请留空"
// @Success 200
// @router /LoginGetQRCar [post]
func (c *LoginController) LoginGetQRCar() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := Login.GetQRCODECar(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取二维码(Mac)
// @Param	body		body 	Login.GetQRReq	true		"不使用代理请留空"
// @Success 200
// @router /LoginGetQRMac [post]
func (c *LoginController) LoginGetQRMac() {
	var GetQR Login.GetQRReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.GetQRCODEMac(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 检测二维码
// @Param	uuid		query 	string	true		"请输入取码时返回的UUID"
// @Success 200
// @router /LoginCheckQR [post]
func (c *LoginController) LoginCheckQR() {
	uuid := c.GetString("uuid")
	WXDATA := Login.CheckUuid(uuid)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 二次登录
// @Param	wxid			query 	string	true		"请输入登录成功的wxid"
// @Failure 200
// @router /LoginTwiceAutoAuth [post]
func (c *LoginController) LoginTwiceAutoAuth() {
	wxid := c.GetString("wxid")
	WXDATA, _ := Login.Secautoauth(wxid)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 62登录(账号或密码)
// @Param	body			body 	Login.Data62LoginReq	true		"不使用代理请留空"
// @Failure 200
// @router /Data62Login [post]
func (c *LoginController) Data62Login() {
	var reqdata Login.Data62LoginReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)

	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.Data62(reqdata, Algorithm.MmtlsShortHost)

	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 62登录(账号或密码), 并申请使用SMS验证
// @Param	body			body 	Login.Data62LoginReq	true		"不使用代理请留空"
// @Failure 200
// @router /Data62SMSApply [post]
func (c *LoginController) Data62SMSApply() {
	var reqdata Login.Data62LoginReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)

	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	// 生成62随机数据
	if reqdata.Data62 == "" || reqdata.Data62 == "string" {
		deviceId := baseutils.CreateDeviceId(reqdata.Data62)
		reqdata.Data62 = baseutils.Get62Data(deviceId)
	}
	if reqdata.DeviceName == "" || reqdata.DeviceName == "string" {
		reqdata.DeviceName = "iPad"
	}
	// 使用62数据登录并自动滑块
	WXDATA := Login.Data62(reqdata, Algorithm.MmtlsShortHost)

	// 记录62
	WXDATA.Data62 = reqdata.Data62

	// 二次验证使用短信验证
	message, transed := WXDATA.Data.(mm.UnifyAuthResponse)
	if transed && strings.Index(message.GetBaseResponse().GetErrMsg().GetString_(), "&ticket=") >= 0 {
		checkUrl, againUrl, setCookie := Login.WechatSMS1(message.GetBaseResponse().GetErrMsg().GetString_(), comm.GenDefaultIpadUA(), reqdata.Proxy)
		WXDATA = models.ResponseResult{
			Code:    0,
			Success: true,
			Message: "已申请短信验证",
			Data: &map[string]string{
				"CheckUrl": checkUrl,
				"AgainUrl": againUrl,
				"Cookie":   setCookie,
			},
			Data62: reqdata.Data62,
		}
	}

	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 62登录(账号或密码), 重发验证码
// @Param	body			body 	Login.Data62SMSAgainReq	true		"不使用代理请留空"
// @Failure 200
// @router /Data62SMSAgain [post]
func (c *LoginController) Data62SMSAgain() {
	var reqdata Login.Data62SMSAgainReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	// 重发短信
	headers := &map[string]string{
		"Cookie": reqdata.Cookie,
	}
	res := comm.HttpGet1(reqdata.Url, headers, comm.GenDefaultIpadUA(), reqdata.Proxy)
	resJson, err := simplejson.NewJson([]byte(res))
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	title, err := resJson.Get("resultData").Get("title").String()
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "",
		Data:    title,
		Data62:  "",
	}
	c.Data["json"] = &WXDATA
	c.ServeJSON()
	return
}

// @Summary 62登录(账号或密码), 短信验证
// @Param	body			body 	Login.Data62SMSVerifyReq	true		"不使用代理请留空"
// @Failure 200
// @router /Data62SMSVerify [post]
func (c *LoginController) Data62SMSVerify() {
	var reqdata Login.Data62SMSVerifyReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	// 验证短信
	verifyUrl := strings.Replace(reqdata.Url, "[[[verifycode]]]", reqdata.Sms, -1)
	verifyUrl = strings.Replace(verifyUrl, "[[[currentMilliseStamp]]]", string(time.Now().Unix()), -1)
	headers := &map[string]string{
		"Cookie": reqdata.Cookie,
	}
	res := comm.HttpGet1(verifyUrl, headers, comm.GenDefaultIpadUA(), reqdata.Proxy)
	resJson, err := simplejson.NewJson([]byte(res))
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	title, err := resJson.Get("resultData").Get("title").String()
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "",
		Data:    title,
		Data62:  "",
	}
	c.Data["json"] = &WXDATA
	c.ServeJSON()
	return
}

// @Summary 62登录(账号或密码), 并申请使用二维码验证
// @Param	body			body 	Login.Data62LoginReq	true		"不使用代理请留空"
// @Failure 200
// @router /Data62QRCodeApply [post]
func (c *LoginController) Data62QRCodeApply() {
	var reqdata Login.Data62LoginReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)

	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	// 生成62随机数据
	if reqdata.Data62 == "" || reqdata.Data62 == "string" {
		deviceId := baseutils.CreateDeviceId(reqdata.Data62)
		reqdata.Data62 = baseutils.Get62Data(deviceId)
	}
	if reqdata.DeviceName == "" || reqdata.DeviceName == "string" {
		reqdata.DeviceName = "iPad"
	}
	// 使用62数据登录并自动滑块
	WXDATA := Login.Data62(reqdata, Algorithm.MmtlsShortHost)

	// 记录62
	WXDATA.Data62 = reqdata.Data62

	// 二次验证使用短信验证
	message, transed := WXDATA.Data.(mm.UnifyAuthResponse)
	if transed && strings.Index(message.GetBaseResponse().GetErrMsg().GetString_(), "&ticket=") >= 0 {
		qrUrl, checkUrl := Login.WeChatQrCode1(message.GetBaseResponse().GetErrMsg().GetString_(), comm.GenDefaultIpadUA(), reqdata.Proxy)
		WXDATA = models.ResponseResult{
			Code:    0,
			Success: true,
			Message: "已申请短信验证",
			Data: &map[string]string{
				"QrUrl":    qrUrl,
				"CheckUrl": checkUrl,
			},
			Data62: reqdata.Data62,
		}
	}

	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 62登录(账号或密码), 二维码验证校验
// @Param	body			body 	Login.Data62SMSVerifyReq	true		"不使用代理请留空"
// @Failure 200
// @router /Data62QRCodeVerify [post]
func (c *LoginController) Data62QRCodeVerify() {
	var reqdata Login.Data62QRCodeVerifyReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	// 验证短信
	verifyUrl := reqdata.Url
	verifyUrl = strings.Replace(verifyUrl, "[[[currentMilliseStamp]]]", string(time.Now().Unix()), -1)
	res := comm.HttpGet1(verifyUrl, nil, comm.GenDefaultIpadUA(), reqdata.Proxy)
	WXDATA := models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "",
		Data:    res,
		Data62:  "",
	}
	c.Data["json"] = &WXDATA
	c.ServeJSON()
	return
}

// @Summary A16登录(账号或密码)
// @Param	body			body 	Login.A16LoginParam	true		"不使用代理请留空"
// @Failure 200
// @router /A16Data [post]
func (c *LoginController) A16Data() {
	var reqdata Login.A16LoginParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)

	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.AndroidA16Login(reqdata, Algorithm.MmtlsShortHost)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary A16登录(账号或密码) - android == 新版云函数
// @Param	body			body 	Login.A16LoginParam	true		"不使用代理请留空"
// @Failure 200
// @router /A16Data1 [post]
func (c *LoginController) A16Data1() {
	var reqdata Login.A16LoginParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)

	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.AndroidA16Login1(reqdata, Algorithm.MmtlsShortHost)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 心跳包
// @Param	wxid			query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /HeartBeat [post]
func (c *LoginController) HeartBeat() {
	wxid := c.GetString("wxid")
	WXDATA, _ := Login.HeartBeat(wxid)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 心跳包
// @Param	wxid			query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /HeartBeatLong [post]
func (c *LoginController) HeartBeatLong() {
	wxid := c.GetString("wxid")
	WXDATA, _ := Login.HeartBeatLong(wxid)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 初始化
// @Param	wxid			query 	string	true		"请输入登录成功的wxid"
// @Param	MaxSynckey		query 	string	false		"二次同步需要带入"
// @Param	CurrentSynckey	query 	string	false		"二次同步需要带入"
// @Success 200
// @router /Newinit [post]
func (c *LoginController) Newinit() {
	wxid := c.GetString("wxid")
	MaxSynckey := c.GetString("MaxSynckey")
	CurrentSynckey := c.GetString("CurrentSynckey")
	WXDATA := Login.Newinit(wxid, MaxSynckey, CurrentSynckey)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 唤醒登录(只限扫码登录)
// @Param	wxid		query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /LoginAwaken [post]
//func (c *LoginController) LoginAwaken() {
//	wxid := c.GetString("wxid")
//	WXDATA := Login.AwakenLogin(wxid)
//	c.Data["json"] = &WXDATA
//	c.ServeJSON()
//}

func (c *LoginController) LoginAwaken() {
	var GetQR Login.AwakenReq
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &GetQR)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := Login.AwakenLoginNew(GetQR)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 重新登录(扫码) - 复用已有设备ID生成二维码
// @Param	wxid		query 	string	true		"请输入wxid"
// @router /ReLoginQR [post]
func (c *LoginController) ReLoginQR() {
	wxid := c.GetString("wxid")
	platform := c.GetString("platform")
	proxyIp := c.GetString("proxy")

	// 从Redis获取已有的登录数据
	D, err := comm.GetLoginata(wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("未找到登录数据: %v", wxid),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	// 使用已有的设备ID生成二维码
	proxy := models.ProxyInfo{ProxyIp: proxyIp}
	getQRReq := Login.GetQRReq{
		Proxy:      proxy,
		DeviceID:   D.Deviceid_str,
		DeviceName: D.DeviceName,
	}

	// 根据平台选择对应的二维码生成函数
	var WXDATA models.ResponseResult2
	switch platform {
	case "Pad":
		WXDATA = Login.GetQRCODEPad(getQRReq)
	case "Win":
		WXDATA = Login.GetQRCODEWin(getQRReq)
	case "WinUwp":
		WXDATA = Login.GetQRCODEWinUwp(getQRReq)
	case "WinUnified":
		WXDATA = Login.GetQRCODEWinUnified(getQRReq)
	case "Mac":
		WXDATA = Login.GetQRCODEMac(getQRReq)
	case "Car":
		WXDATA = Login.GetQRCODECar(getQRReq)
	default:
		WXDATA = Login.GetQRCODE(getQRReq)
	}

	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取登录缓存信息
// @Param	wxid		query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /GetCacheInfo [post]
func (c *LoginController) GetCacheInfo() {
	wxid := c.GetString("wxid")
	WXDATA := Login.CacheInfo(wxid)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取62数据
// @Param	wxid		query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /Get62Data [post]
func (c *LoginController) Get62Data() {
	wxid := c.GetString("wxid")
	Data62 := Login.Get62Data(wxid)
	Result := models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    Data62,
	}
	c.Data["json"] = &Result
	c.ServeJSON()
	return
}

// @Summary 获取A16数据
// @Param	wxid		query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /GetA16Data [post]
func (c *LoginController) GetA16Data() {
	wxid := c.GetString("wxid")
	Data62 := Login.GetA16Data(wxid)
	Result := models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    Data62,
	}
	c.Data["json"] = &Result
	c.ServeJSON()
	return
}

// @Summary 退出登录
// @Param	wxid			query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /LogOut [post]
func (c *LoginController) LogOut() {
	wxid := c.GetString("wxid")
	WXDATA := Login.LogOut(wxid)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 新设备扫码登录
// @Param	body			body 	Login.ExtDeviceLoginConfirmParam	true		"URL == MAC iPad Windows 的微信二维码解析出来的url"
// @Success 200
// @router /ExtDeviceLoginConfirmGet [post]
func (c *LoginController) ExtDeviceLoginConfirmGet() {
	var reqdata Login.ExtDeviceLoginConfirmParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)

	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.ExtDeviceLoginConfirmGet(reqdata)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 新设备扫码确认登录
// @Param	body			body 	Login.ExtDeviceLoginConfirmParam	true		"URL == MAC iPad Windows 的微信二维码解析出来的url"
// @Success 200
// @router /ExtDeviceLoginConfirmOk [post]
func (c *LoginController) ExtDeviceLoginConfirmOk() {
	var reqdata Login.ExtDeviceLoginConfirmParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)

	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.ExtDeviceLoginConfirmOk(reqdata)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 提交登录验证码
// @Param	body			body 	Login.VerificationcodeParam	true	""
// @Success 200
// @router /YPayVerificationcode [post]
func (c *LoginController) YPayVerificationcode() {
	var reqdata Login.VerificationcodeParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &reqdata)

	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}

	WXDATA := Login.Verificationcode2(reqdata)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 开启自动心跳, 自动二次登录（linux 长连接，win 短链接）
// @Param	wxid			query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /AutoHeartBeat [post]
func (c *LoginController) AutoHeartBeat() {
	wxid := c.GetString("wxid")
	// 开启自动心跳
	D, err := comm.GetLoginata(wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("系统异常：%v [%v]", "未找到登录信息", wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("系统异常：%v", err.Error())
		}
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: errorMsg,
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
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
	err = wXConnect.SendHeartBeat()
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("发送心跳失败：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	Result := models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "发送心跳成功",
		Data:    nil,
	}
	c.Data["json"] = &Result
	c.ServeJSON()
}

// @Summary 关闭自动心跳、自动二次登录
// @Param	wxid			query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /CloseAutoHeartBeat [post]
func (c *LoginController) CloseAutoHeartBeat() {
	wxid := c.GetString("wxid")
	// 关闭自动心跳
	D, err := comm.GetLoginata(wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("系统异常：%v [%v]", "未找到登录信息", wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("系统异常：%v", err.Error())
		}
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: errorMsg,
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	wxConnectMgr := wxcore.GetWXConnectMgr()
	wXConnect := wxConnectMgr.GetWXConnectByWXID(wxid)
	if wXConnect != nil {
		wXConnect.Stop()
	}
	comm.AutoHeartBeatListClear(wxid)
	Result := models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "关闭心跳成功",
		Data:    nil,
	}
	c.Data["json"] = &Result
	c.ServeJSON()
}

// @Summary 自动心跳日志
// @Param	wxid			query 	string	true		"请输入登录成功的wxid"
// @Success 200
// @router /AutoHeartBeatLog [post]
func (c *LoginController) AutoHeartBeatLog() {
	wxid := c.GetString("wxid")
	// 清理首尾空格
	wxid = strings.TrimSpace(wxid)
	if wxid == "" || wxid == "string" || strings.Contains(wxid, "*") {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: "wxid不能为空",
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	logs := make([]string, 0)
	comm.GETObj("AutoHeartBeatList:"+wxid, &logs)
	Result := models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "获取成功, 只保留最新的 100 条心跳、二次登录日志",
		Data:    logs,
	}
	c.Data["json"] = &Result
	c.ServeJSON()
}
