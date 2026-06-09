package models

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
)

type PortalWxStatus struct {
    Nickname    string `json:"nickname"`
    Wxid        string `json:"wxid"`
    Device      string `json:"device"`
    Status      string `json:"status"`
    Online      bool   `json:"online"`
    LoginTime   string `json:"loginTime"`
    RefreshTime string `json:"refreshTime"`
}

type PortalWxActionResult struct {
    Message   string          `json:"message"`
    Status    *PortalWxStatus `json:"status,omitempty"`
    QRBase64  string          `json:"qrBase64,omitempty"`
    UUID      string          `json:"uuid,omitempty"`
    Cost      int             `json:"cost,omitempty"`
    NeedPoll  bool            `json:"needPoll,omitempty"`
}

// portalWxDeviceInfo 设备信息
type portalWxDeviceInfo struct {
    Wxid        string `json:"wxid"`
    Avatar      string `json:"avatar"`
    Nickname    string `json:"nickname"`
    Device      string `json:"device"`
    Survival    int    `json:"survival"`
    LoginDate   int64  `json:"loginDate"`
    RefreshDate int64  `json:"refreshDate"`
}

type portalWxUserStatusResponse struct {
    Status bool                          `json:"status"`
    Data   map[string]portalWxDeviceInfo `json:"data"`
    Message string                        `json:"message"`
}

func GetPortalWxStatus(userNumber int) (*PortalWxStatus, error) {
    user, err := getPortalUserByNumber(userNumber)
    if err != nil {
        return nil, err
    }
    wxid := strings.TrimSpace(user.Wxid)
    if wxid == "" {
        return nil, fmt.Errorf("当前用户未绑定微信ID")
    }

    // 同时查询新旧两个地址，取在线的那个
    var bestInfo *portalWxDeviceInfo

    // 查询活跃地址
    raw, err := getWxUserStatusRaw()
    if err == nil && raw.Data != nil {
        if info, ok := raw.Data[wxid]; ok {
            bestInfo = &info
        }
    }

    // 如果新协议已启用，再查询旧地址
    if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
        rawOld, errOld := getWxUserStatusRawFromURL(getOldWxLoginBaseURL())
        if errOld == nil && rawOld.Data != nil {
            if info, ok := rawOld.Data[wxid]; ok {
                // 如果旧地址在线而新地址不在线，用旧地址的
                if bestInfo == nil || (info.Survival == 1 && bestInfo.Survival != 1) {
                    bestInfo = &info
                }
            }
        }
    }

    // 如果活跃地址不是新地址且新地址已配置，也检查新地址
    if Config.WxProtocol.NewLoginBaseURL != "" && getWxLoginBaseURL() != getNewWxLoginBaseURL() {
        rawNew, errNew := getWxUserStatusRawFromURL(getNewWxLoginBaseURL())
        if errNew == nil && rawNew.Data != nil {
            if info, ok := rawNew.Data[wxid]; ok {
                if bestInfo == nil || (info.Survival == 1 && bestInfo.Survival != 1) {
                    bestInfo = &info
                }
            }
        }
    }

    if bestInfo == nil {
        return &PortalWxStatus{
            Nickname: user.Nickname,
            Wxid:     wxid,
            Device:   "-",
            Status:   "🔴 离线",
            Online:   false,
        }, nil
    }

    // 如果该 wxid 最近被主动登出（60秒内），强制显示为离线
    // 解决 wechat08 API 延迟更新 survival 字段的问题
    online := bestInfo.Survival == 1
    if online && isRecentlyLoggedOut(wxid) {
        online = false
    }

    return &PortalWxStatus{
        Nickname:    defaultWxNickname(bestInfo.Nickname, user.Nickname),
        Wxid:        wxid,
        Device:      bestInfo.Device,
        Status:      wxStatusTextBool(online),
        Online:      online,
        LoginTime:   formatPortalWxUnix(bestInfo.LoginDate),
        RefreshTime: formatPortalWxUnix(bestInfo.RefreshDate),
    }, nil
}

func PortalWxScanLogin(userNumber int) (*PortalWxActionResult, error) {
    user, err := getPortalUserByNumber(userNumber)
    if err != nil {
        return nil, err
    }
    cost := getWxScanLoginCost()
    coin := GetCoin(userNumber)
    if coin < cost {
        return nil, fmt.Errorf("积分不足，微信扫码登录需要 %d 积分，当前积分 %d", cost, coin)
    }

    reqBody := map[string]interface{}{
        "DeviceID":   "device_" + fmt.Sprintf("%d", time.Now().UnixNano()),
        "DeviceName": getWxDeviceName(),
        "DeviceType": "car",
        "Proxy": map[string]string{
            "ProxyIp":       "",
            "ProxyPassword": "",
            "ProxyUser":     "",
        },
    }

    body, err := wxLoginRequest("/api/v1/wx/login/code", reqBody)
    if err != nil {
        return nil, err
    }
    var result WxLoginCodeResp
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("解析响应失败：%v", err)
    }
    if !result.Status || !result.Success {
        return nil, fmt.Errorf("获取二维码失败：%s", result.Message)
    }

    qrPayload := normalizePortalQrBase64(result.Data.QrBase64)
    if qrPayload == "" {
        return nil, fmt.Errorf("二维码数据为空")
    }

    _ = user
    return &PortalWxActionResult{
        Message:  fmt.Sprintf("二维码已生成，扫码成功后将扣除 %d 积分", cost),
        QRBase64: qrPayload,
        UUID:     result.Data.Uuid,
        Cost:     cost,
        NeedPoll: true,
    }, nil
}

func PortalWxRelogin(userNumber int, targetWxid ...string) (*PortalWxActionResult, error) {
	user, err := getPortalUserByNumber(userNumber)
	if err != nil {
		return nil, err
	}
	wxid := strings.TrimSpace(user.Wxid)
	if len(targetWxid) > 0 && strings.TrimSpace(targetWxid[0]) != "" {
		wxid = strings.TrimSpace(targetWxid[0])
	}
	if wxid == "" {
		return nil, fmt.Errorf("当前用户未绑定微信ID")
	}

	online, _ := checkWxDeviceOnline(wxid)

	// 判断是否为需要迁移的旧用户：wxid 在旧地址设备列表中能找到
	isMigration := false
	if isNewProtocolEnabled() && !IsWxWxidMigrated(wxid) {
		oldExists, _ := checkWxDeviceExistsOnURL(getOldWxLoginBaseURL(), wxid)
		if oldExists {
			isMigration = true
			online = true
		}
	}

	if !online {
		return nil, fmt.Errorf("设备尚未登录，请先执行微信扫码登录")
	}

	// 根据是否迁移选择不同的登录方式
	var body []byte
	var qrPayload string
	var uuid string
	var msg string

	if isMigration {
		// 旧用户迁移：在新地址上走全新扫码登录（不扣积分）
		scanReqBody := map[string]interface{}{
			"DeviceID":   "device_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			"DeviceName": getWxDeviceName(),
			"DeviceType": "car",
			"Proxy": map[string]string{
				"ProxyIp":       "",
				"ProxyPassword": "",
				"ProxyUser":     "",
			},
		}
		body, err = wxLoginRequestToURL(getNewWxLoginBaseURL(), "/api/v1/wx/login/code", scanReqBody)
		if err != nil {
			return nil, fmt.Errorf("迁移登录失败：%v", err)
		}
		var scanResult WxLoginCodeResp
		if err = json.Unmarshal(body, &scanResult); err != nil || !scanResult.Status || !scanResult.Success {
		 errMsg := "获取二维码失败"
		 if err == nil {
		 	errMsg = scanResult.Message
		 }
			return nil, fmt.Errorf("迁移登录失败：%s", errMsg)
		}
		qrPayload = normalizePortalQrBase64(scanResult.Data.QrBase64)
		uuid = scanResult.Data.Uuid
		msg = "检测到旧协议设备，正在迁移到新协议（不扣积分），请扫码确认"
	} else {
		// 普通用户：走二次登录
		reqBody := map[string]interface{}{
			"wxid": wxid,
			"Proxy": map[string]string{
				"ProxyIp":       "",
				"ProxyPassword": "",
				"ProxyUser":     "",
			},
		}
		body, err = wxLoginRequest("/api/v1/wx/login/again", reqBody)
		if err != nil {
			return nil, err
		}
		var result WxLoginAgainResp
		if err = json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("解析响应失败：%v", err)
		}
		if !result.Status {
			return nil, fmt.Errorf("重新登录失败：%s", result.Message)
		}
		qrPayload = normalizePortalQrBase64(result.Data.QrBase64)
		uuid = result.Data.Uuid
		msg = "重新登录二维码已生成，请用手机微信扫码确认"
	}
	return &PortalWxActionResult{
		Message:  msg,
		QRBase64: qrPayload,
		UUID:     uuid,
		NeedPoll: uuid != "",
	}, nil
}

func PortalWxWakeLogin(userNumber int, targetWxid ...string) (*PortalWxActionResult, error) {
	user, err := getPortalUserByNumber(userNumber)
	if err != nil {
		return nil, err
	}
	wxid := strings.TrimSpace(user.Wxid)
	if len(targetWxid) > 0 && strings.TrimSpace(targetWxid[0]) != "" {
		wxid = strings.TrimSpace(targetWxid[0])
	}
	if wxid == "" {
		return nil, fmt.Errorf("当前用户未绑定微信ID")
	}

	online, _ := checkWxDeviceOnline(wxid)

	// 判断是否为需要迁移的旧用户：wxid 在旧地址设备列表中能找到
	isMigration := false
	if isNewProtocolEnabled() && !IsWxWxidMigrated(wxid) {
		oldExists, _ := checkWxDeviceExistsOnURL(getOldWxLoginBaseURL(), wxid)
		if oldExists {
			isMigration = true
			online = true
		}
	}

	if !online {
		return nil, fmt.Errorf("设备尚未登录，请先执行微信扫码登录")
	}

	var qrPayload string
	var uuid string
	var msg string

	if isMigration {
		// 旧用户迁移：在新地址上走全新扫码登录（不扣积分）
		scanReqBody := map[string]interface{}{
			"DeviceID":   "device_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			"DeviceName": getWxDeviceName(),
			"DeviceType": "car",
			"Proxy": map[string]string{
				"ProxyIp":       "",
				"ProxyPassword": "",
				"ProxyUser":     "",
			},
		}
		body, err := wxLoginRequestToURL(getNewWxLoginBaseURL(), "/api/v1/wx/login/code", scanReqBody)
		if err != nil {
			return nil, fmt.Errorf("迁移登录失败：%v", err)
		}
		var scanResult WxLoginCodeResp
		if err = json.Unmarshal(body, &scanResult); err != nil || !scanResult.Status || !scanResult.Success {
			errMsg := "获取二维码失败"
			if err == nil {
				errMsg = scanResult.Message
			}
			return nil, fmt.Errorf("迁移登录失败：%s", errMsg)
		}
		qrPayload = normalizePortalQrBase64(scanResult.Data.QrBase64)
		uuid = scanResult.Data.Uuid
		msg = "检测到旧协议设备，正在迁移到新协议（不扣积分），请扫码确认"
	} else {
		// 普通用户：走唤醒+二次登录
		activeURL := getWxLoginBaseURL()

		awakeBody := map[string]string{"wxid": wxid}
		body, err := wxLoginRequestToURL(activeURL, "/api/v1/wx/login/awake", awakeBody)
		if err != nil {
			return nil, fmt.Errorf("唤醒失败：%v", err)
		}
		var awakeResult struct {
			Status  bool   `json:"status"`
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &awakeResult); err != nil {
			return nil, fmt.Errorf("解析唤醒响应失败：%v", err)
		}
		if !awakeResult.Status {
			return nil, fmt.Errorf("唤醒设备失败：%s", awakeResult.Message)
		}

		twiceBody := map[string]string{"wxid": wxid}
		body, err = wxLoginRequestToURL(activeURL, "/api/v1/wx/login/twice", twiceBody)
		if err != nil {
			return nil, fmt.Errorf("获取唤醒登录二维码失败：%v", err)
		}
		var twiceResult struct {
			Status  bool   `json:"status"`
			Success bool   `json:"success"`
			Data    struct {
				QrBase64 string `json:"qrbase64"`
				Uuid     string `json:"uuid"`
			} `json:"data"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &twiceResult); err != nil {
			return nil, fmt.Errorf("解析唤醒登录响应失败：%v", err)
		}
		if !twiceResult.Status {
			return nil, fmt.Errorf("唤醒登录失败：%s", twiceResult.Message)
		}
		qrPayload = normalizePortalQrBase64(twiceResult.Data.QrBase64)
		uuid = twiceResult.Data.Uuid
		msg = "设备已唤醒，请扫码完成登录"
	}

	return &PortalWxActionResult{
		Message:  msg,
		QRBase64: qrPayload,
		UUID:     uuid,
		NeedPoll: uuid != "",
	}, nil
}

func PortalWxLogout(userNumber int, targetWxid ...string) (*PortalWxActionResult, error) {
    user, err := getPortalUserByNumber(userNumber)
    if err != nil {
        return nil, err
    }
    wxid := strings.TrimSpace(user.Wxid)
    if len(targetWxid) > 0 && strings.TrimSpace(targetWxid[0]) != "" {
        wxid = strings.TrimSpace(targetWxid[0])
    }
    if wxid == "" {
        return nil, fmt.Errorf("当前用户未绑定微信ID")
    }

    online, err := checkWxDeviceOnline(wxid)
    if err != nil {
        return nil, err
    }
    if !online {
        return nil, fmt.Errorf("设备尚未登录，无需登出")
    }

    body, err := wxLoginRequest("/api/v1/wx/login/logout", map[string]string{"wxid": wxid})
    if err != nil {
        return nil, err
    }
    var result WxLogoutResp
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("解析响应失败：%v", err)
    }
    if !result.Status {
        return nil, fmt.Errorf("登出失败：%s", result.Message)
    }

    // 标记为最近登出，避免 wechat08 API 延迟更新 survival 时仍显示在线
    markRecentlyLoggedOut(wxid)

    status, _ := GetPortalWxStatus(userNumber)
    return &PortalWxActionResult{Message: "已成功登出", Status: status}, nil
}

func PortalWxDelete(userNumber int, targetWxid ...string) (*PortalWxActionResult, error) {
    user, err := getPortalUserByNumber(userNumber)
    if err != nil {
        return nil, err
    }
    wxid := strings.TrimSpace(user.Wxid)
    if len(targetWxid) > 0 && strings.TrimSpace(targetWxid[0]) != "" {
        wxid = strings.TrimSpace(targetWxid[0])
    }
    if wxid == "" {
        return nil, fmt.Errorf("当前用户未绑定微信ID")
    }

    statusRaw, err := getWxUserStatusRaw()
    if err != nil {
        return nil, err
    }

    matchedWxid := ""
    for key, info := range statusRaw.Data {
        if strings.TrimSpace(key) == wxid || strings.TrimSpace(info.Wxid) == wxid {
            matchedWxid = key
            if strings.TrimSpace(info.Wxid) != "" {
                matchedWxid = info.Wxid
            }
            break
        }
    }
    if matchedWxid == "" {
        matchedWxid = wxid
    }

    body, err := wxLoginRequest("/api/v1/wx/user/delete", map[string]interface{}{"wxids": []string{matchedWxid}})
    if err != nil {
        return nil, err
    }
    var result WxLogoutResp
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("解析响应失败：%v", err)
    }
    if !result.Status {
        return nil, fmt.Errorf("删除失败：%s", result.Message)
    }

    return &PortalWxActionResult{Message: fmt.Sprintf("设备数据已删除（%s），如需再次上线需重新扫码并扣积分", matchedWxid)}, nil
}

func PortalWxPollLogin(userNumber int, uuid string, deductCoin bool) (*PortalWxActionResult, error) {
    if strings.TrimSpace(uuid) == "" {
        return nil, fmt.Errorf("缺少登录UUID")
    }
    status, err := checkLoginStatus(uuid)
    if err != nil {
        return nil, err
    }

    switch status.Code {
    case 0:
        return &PortalWxActionResult{Message: "等待扫码中", NeedPoll: true, UUID: uuid}, nil
    case 1:
        return &PortalWxActionResult{Message: "已扫码，请在手机上确认登录", NeedPoll: true, UUID: uuid}, nil
    case 2:
        nickname := status.User.Nickname
        wxid := status.User.Wxid
        if nickname == "" {
            nickname = status.Data.Nickname
        }
        if wxid == "" {
            wxid = status.Data.Wxid
        }
        if nickname == "" {
            nickname = "微信用户"
        }
        // 登录成功，清除登出缓存标记
        clearRecentlyLoggedOut(wxid)
        user, err := getPortalUserByNumber(userNumber)
        if err == nil && wxid != "" {
            if strings.TrimSpace(user.Wxid) == "" {
                db.Model(&user).Update("wxid", wxid)
            } else if user.Wxid != wxid {
                var existingDevice PortalWxDevice
                if db.Where("user_number = ? AND wxid = ?", userNumber, wxid).First(&existingDevice).Error != nil {
                    var deviceCount int64
                    db.Model(&PortalWxDevice{}).Where("user_number = ?", userNumber).Count(&deviceCount)
                    primaryCount := int64(0)
                    if strings.TrimSpace(user.Wxid) != "" {
                        primaryCount = 1
                    }
                    if deviceCount+primaryCount < maxWxDevicesPerUser {
                        db.Create(&PortalWxDevice{
                            UserNumber: userNumber,
                            Wxid:       wxid,
                        })
                    }
                }
            }
        }

        msg := fmt.Sprintf("登录成功，昵称：%s，微信ID：%s", nickname, wxid)
        if deductCoin {
            cost := getWxScanLoginCost()
            currentCoin := GetCoin(userNumber)
            if currentCoin < cost {
                return nil, fmt.Errorf("登录成功但积分不足，无法扣除 %d 积分（当前积分：%d），请联系管理员处理", cost, currentCoin)
            }
            actualCoin := RemCoin(userNumber, cost)
            if actualCoin > currentCoin {
                return nil, fmt.Errorf("积分扣除异常，请联系管理员")
            }
            RecordCoinLog(userNumber, -cost, "微信登录", "微信扫码登录扣费")
            msg = fmt.Sprintf("登录成功，已扣除 %d 积分，剩余 %d 积分，昵称：%s，微信ID：%s", cost, GetCoin(userNumber), nickname, wxid)
        } else if isNewProtocolEnabled() && !IsWxWxidMigrated(wxid) {
            // 迁移登录成功，标记为已迁移
            MarkWxMigrated(userNumber, wxid)
            msg = fmt.Sprintf("迁移成功！已切换到新协议地址（未扣积分），昵称：%s，微信ID：%s", nickname, wxid)
        }
        wxStatus, _ := GetPortalWxStatus(userNumber)
        return &PortalWxActionResult{Message: msg, Status: wxStatus}, nil
    default:
        if status.Msg != "" {
            return nil, fmt.Errorf("登录异常：%s", status.Msg)
        }
        return &PortalWxActionResult{Message: "登录状态未知，请稍后重试", NeedPoll: true, UUID: uuid}, nil
    }
}

func getPortalUserByNumber(userNumber int) (*User, error) {
    var user User
    if err := db.Where("number = ?", userNumber).First(&user).Error; err != nil {
        return nil, fmt.Errorf("未找到绑定用户")
    }
    return &user, nil
}

func getWxUserStatusRaw() (*portalWxUserStatusResponse, error) {
    url := getWxLoginBaseURL() + "/api/v1/wx/user/status"
    client := &http.Client{Timeout: HTTPTimeout}
    resp, err := client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("请求设备列表失败：%s", err.Error())
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取设备列表失败：%s", err.Error())
    }

    var result portalWxUserStatusResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("解析设备列表失败：%s", err.Error())
    }
    if !result.Status {
        return nil, fmt.Errorf("获取设备列表失败：%s", result.Message)
    }
    return &result, nil
}

// getWxUserStatusRawFromURL 从指定地址获取设备状态
func getWxUserStatusRawFromURL(baseURL string) (*portalWxUserStatusResponse, error) {
    url := baseURL + "/api/v1/wx/user/status"
    client := &http.Client{Timeout: HTTPTimeout}
    resp, err := client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("请求设备列表失败：%s", err.Error())
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("读取设备列表失败：%s", err.Error())
    }

    var result portalWxUserStatusResponse
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("解析设备列表失败：%s", err.Error())
    }
    if !result.Status {
        return nil, fmt.Errorf("获取设备列表失败：%s", result.Message)
    }
    return &result, nil
}

func normalizePortalQrBase64(raw string) string {
    raw = strings.TrimSpace(raw)
    if raw == "" {
        return ""
    }
    if strings.HasPrefix(raw, "data:image") {
        return raw
    }
    cleaned := raw
    if idx := strings.Index(cleaned, ","); idx != -1 {
        cleaned = cleaned[idx+1:]
    }
    if _, err := base64.StdEncoding.DecodeString(cleaned); err == nil {
        return "data:image/png;base64," + cleaned
    }
    if _, err := base64.RawStdEncoding.DecodeString(cleaned); err == nil {
        return "data:image/png;base64," + cleaned
    }
    return raw
}

func wxStatusText(survival int) string {
    if survival == 1 {
        return "🟢 在线"
    }
    return "🔴 离线"
}

func wxStatusTextBool(online bool) string {
    if online {
        return "🟢 在线"
    }
    return "🔴 离线"
}

func formatPortalWxUnix(ts int64) string {
    if ts <= 0 {
        return "-"
    }
    return time.Unix(ts, 0).Format("2006-01-02 15:04")
}

func defaultWxNickname(primary, fallback string) string {
    if strings.TrimSpace(primary) != "" {
        return primary
    }
    if strings.TrimSpace(fallback) != "" {
        return fallback
    }
    return "未知昵称"
}
