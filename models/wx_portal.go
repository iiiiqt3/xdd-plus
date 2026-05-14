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

type portalWxUserStatusResponse struct {
    Status bool `json:"status"`
    Data   map[string]struct {
        Wxid        string `json:"wxid"`
        Avatar      string `json:"avatar"`
        Nickname    string `json:"nickname"`
        Device      string `json:"device"`
        Survival    int    `json:"survival"`
        LoginDate   int64  `json:"loginDate"`
        RefreshDate int64  `json:"refreshDate"`
    } `json:"data"`
    Message string `json:"message"`
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

    raw, err := getWxUserStatusRaw()
    if err != nil {
        return nil, err
    }
    info, ok := raw.Data[wxid]
    if !ok {
        return &PortalWxStatus{
            Nickname: user.Nickname,
            Wxid:     wxid,
            Device:   "-",
            Status:   "🔴 离线",
            Online:   false,
        }, nil
    }

    return &PortalWxStatus{
        Nickname:    defaultWxNickname(info.Nickname, user.Nickname),
        Wxid:        wxid,
        Device:      info.Device,
        Status:      wxStatusText(info.Survival),
        Online:      info.Survival == 1,
        LoginTime:   formatPortalWxUnix(info.LoginDate),
        RefreshTime: formatPortalWxUnix(info.RefreshDate),
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

func PortalWxRelogin(userNumber int) (*PortalWxActionResult, error) {
    user, err := getPortalUserByNumber(userNumber)
    if err != nil {
        return nil, err
    }
    wxid := strings.TrimSpace(user.Wxid)
    if wxid == "" {
        return nil, fmt.Errorf("当前用户未绑定微信ID")
    }

    online, err := checkWxDeviceOnline(wxid)
    if err != nil {
        return nil, err
    }
    if !online {
        return nil, fmt.Errorf("你尚未扫码登录，请先执行微信扫码登录")
    }

    reqBody := map[string]interface{}{
        "wxid": wxid,
        "Proxy": map[string]string{
            "ProxyIp":       "",
            "ProxyPassword": "",
            "ProxyUser":     "",
        },
    }
    body, err := wxLoginRequest("/api/v1/wx/login/again", reqBody)
    if err != nil {
        return nil, err
    }
    var result WxLoginAgainResp
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("解析响应失败：%v", err)
    }
    if !result.Status {
        return nil, fmt.Errorf("重新登录失败：%s", result.Message)
    }

    qrPayload := normalizePortalQrBase64(result.Data.QrBase64)
    return &PortalWxActionResult{
        Message:  "重新登录二维码已生成，请扫码确认",
        QRBase64: qrPayload,
        UUID:     result.Data.Uuid,
        NeedPoll: result.Data.Uuid != "",
    }, nil
}

func PortalWxWakeLogin(userNumber int) (*PortalWxActionResult, error) {
    user, err := getPortalUserByNumber(userNumber)
    if err != nil {
        return nil, err
    }
    wxid := strings.TrimSpace(user.Wxid)
    if wxid == "" {
        return nil, fmt.Errorf("当前用户未绑定微信ID")
    }

    online, err := checkWxDeviceOnline(wxid)
    if err != nil {
        return nil, err
    }
    if !online {
        return nil, fmt.Errorf("你尚未扫码登录，请先执行微信扫码登录")
    }

    awakeBody := map[string]string{"wxid": wxid}
    body, err := wxLoginRequest("/api/v1/wx/login/awake", awakeBody)
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
    body, err = wxLoginRequest("/api/v1/wx/login/twice", twiceBody)
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

    return &PortalWxActionResult{
        Message:  "设备已唤醒，请扫码完成登录",
        QRBase64: normalizePortalQrBase64(twiceResult.Data.QrBase64),
        UUID:     twiceResult.Data.Uuid,
        NeedPoll: twiceResult.Data.Uuid != "",
    }, nil
}

func PortalWxLogout(userNumber int) (*PortalWxActionResult, error) {
    user, err := getPortalUserByNumber(userNumber)
    if err != nil {
        return nil, err
    }
    wxid := strings.TrimSpace(user.Wxid)
    if wxid == "" {
        return nil, fmt.Errorf("当前用户未绑定微信ID")
    }

    online, err := checkWxDeviceOnline(wxid)
    if err != nil {
        return nil, err
    }
    if !online {
        return nil, fmt.Errorf("你尚未扫码登录，无需登出")
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

    status, _ := GetPortalWxStatus(userNumber)
    return &PortalWxActionResult{Message: "已成功登出", Status: status}, nil
}

func PortalWxDelete(userNumber int) (*PortalWxActionResult, error) {
    user, err := getPortalUserByNumber(userNumber)
    if err != nil {
        return nil, err
    }
    wxid := strings.TrimSpace(user.Wxid)
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

    db.Model(&user).Update("wxid", "")
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
        user, err := getPortalUserByNumber(userNumber)
        if err == nil && wxid != "" && strings.TrimSpace(user.Wxid) == "" {
            db.Model(&user).Update("wxid", wxid)
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
            msg = fmt.Sprintf("登录成功，已扣除 %d 积分，剩余 %d 积分，昵称：%s，微信ID：%s", cost, GetCoin(userNumber), nickname, wxid)
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
