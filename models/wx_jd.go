package models

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

)

const (
	WxJdAppID     = "wx73247c7819d61796"
	wxJdClientVer = "2.0.2"
	wxJdJDAppID   = "599"
	wxJdSignGSALT = "sb2cwlYyaCSN1KUv5RHG3tmqxfEb8NKN"
	wxJdBizKey    = "bce044c839bb9eb811aad5af18a629e199da4e13"
	wxJdFingerTk  = "L64RTJ562VJEYNEQN67XMUWSR4UFLOIQHJYZ3MWERRIKJGP24SDSBDS4I4AMVU24Y3Y7A4UPDICN2"
	wxJdAlpha     = "23IL<N01c7KvwZO56RSTAfghiFyzWJqVabGH4PQdopUrsCuX*xeBjkltDEmn89.-/"
	wxJdReferer   = "https://servicewechat.com/" + WxJdAppID + "/864/page-frame.html"
	wxJdUA        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13) UnifiedPCWindowsWechat(0xf254186b) XWEB/19481"
)

var wxJdList = make(map[int]chan string)

type RiskVerifyError struct {
	JmpURL string
	ErrMsg string
}

func (e *RiskVerifyError) Error() string {
	return e.ErrMsg
}

func getWxJdServer() string {
	if Config.WxProtocol.JdServer != "" {
		return strings.TrimRight(Config.WxProtocol.JdServer, "/")
	}
	// 使用活跃协议地址
	if Config.WxProtocol.ActiveProtocol == "new" && Config.WxProtocol.NewLoginBaseURL != "" {
		return strings.TrimRight(Config.WxProtocol.NewLoginBaseURL, "/")
	}
	return strings.TrimRight(Config.WxProtocol.LoginBaseURL, "/")
}

// getWxJdServerForDevice 根据设备所在的服务器地址返回对应的JdServer
func getWxJdServerForDevice(wxid string) string {
	if Config.WxProtocol.JdServer != "" {
		return strings.TrimRight(Config.WxProtocol.JdServer, "/")
	}
	
	// 检查设备在哪个地址上在线
	// 先检查活跃地址
	activeURL := getWxLoginBaseURL()
	if online, _ := checkWxDeviceOnlineFromURL(activeURL, wxid); online {
		return strings.TrimRight(activeURL, "/")
	}
	
	// 如果新协议已启用，检查旧地址
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		oldURL := getOldWxLoginBaseURL()
		if online, _ := checkWxDeviceOnlineFromURL(oldURL, wxid); online {
			return strings.TrimRight(oldURL, "/")
		}
	}
	
	// 如果活跃地址不是新地址，检查新地址
	if Config.WxProtocol.NewLoginBaseURL != "" && activeURL != getNewWxLoginBaseURL() {
		newURL := getNewWxLoginBaseURL()
		if online, _ := checkWxDeviceOnlineFromURL(newURL, wxid); online {
			return strings.TrimRight(newURL, "/")
		}
	}
	
	// 默认使用活跃地址
	return strings.TrimRight(activeURL, "/")
}

func wxJdPost(paths []string, body interface{}, timeout int) (map[string]interface{}, error) {
	base := getWxJdServer()
	return wxJdPostToURL(base, paths, body, timeout)
}

func wxJdPostToURL(baseURL string, paths []string, body interface{}, timeout int) (map[string]interface{}, error) {
	jsonData, _ := json.Marshal(body)
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	var lastErr error
	for _, p := range paths {
		req, err := http.NewRequest("POST", baseURL+p, strings.NewReader(string(jsonData)))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var result map[string]interface{}
		if json.Unmarshal(respBody, &result) == nil {
			return result, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("微信协议请求失败")
}

func wxJdGetWxCode(wxid string) (string, error) {
	server := getWxJdServerForDevice(wxid)
	return wxJdGetWxCodeFromURL(server, wxid, WxJdAppID)
}

func wxJdGetWxCodeFromURL(baseURL string, wxid string, appID string) (string, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = WxJdAppID
	}
	data, err := wxJdPostToURL(
		baseURL,
		[]string{"/api/v1/wx/app/get/code", "/wx/app/get/code"},
		map[string]string{"wxid": wxid, "appid": appID},
		15,
	)
	if err != nil {
		return "", err
	}
	code := ""
	if d, ok := data["Data"].(map[string]interface{}); ok {
		code, _ = d["code"].(string)
	}
	if code == "" {
		if d, ok := data["data"].(map[string]interface{}); ok {
			code, _ = d["code"].(string)
		}
	}
	if code == "" {
		return "", fmt.Errorf("获取wx code失败: %v", truncateObj(data, 300))
	}
	return code, nil
}

func wxJdGetEidToken(wxid string) string {
	server := getWxJdServerForDevice(wxid)
	return wxJdGetEidTokenFromURL(server, wxid)
}

func wxJdGetEidTokenFromURL(baseURL string, wxid string) string {
	payload := map[string]interface{}{
		"api_name":       "webapi_getuserinfo",
		"data":           map[string]string{"lang": "zh_CN"},
		"with_credentials": true,
	}
	data, err := wxJdPostToURL(
		baseURL,
		[]string{"/api/v1/wx/app/call/function", "/wx/app/call/function"},
		map[string]interface{}{"wxid": wxid, "appid": WxJdAppID, "data": mustJSON(payload)},
		15,
	)
	if err != nil {
		return ""
	}
	inner, _ := data["Data"].(map[string]interface{})
	if inner == nil {
		inner, _ = data["data"].(map[string]interface{})
	}
	if inner == nil {
		return ""
	}
	b64Data, _ := inner["data"].(string)
	if b64Data == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return ""
	}
	var parsed map[string]interface{}
	json.Unmarshal(decoded, &parsed)
	if parsed == nil {
		return ""
	}
	for _, k := range []string{"eid_token", "eidToken", "eid"} {
		if v, ok := parsed[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func wxJdGetFingerTk() (string, error) {
	now := time.Now().UnixMilli()
	env := map[string]interface{}{
		"sv": "1.0.3.4", "clist": now, "vlv": "3.16.0", "ve": "4.1.8.107",
		"fs": -1, "la": "zh_CN", "br": "microsoft", "mo": "microsoft",
		"pr": 1, "pl": "windows", "sh": 780, "sw": 414, "sbh": "",
		"sy": "Windows 10", "wh": 780, "ww": 414, "bl": "", "nt": "wifi",
		"vid": WxJdAppID, "bk": wxJdBizKey, "cliet": now, "fp": randHex(16),
	}
	encoded := wxJdFingerEncode(env)
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("POST",
		"https://we.jd.com/stone/1/"+wxJdFingerTk,
		strings.NewReader(encoded),
	)
	req.Header.Set("User-Agent", wxJdUA)
	req.Header.Set("Referer", wxJdReferer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("finger_tk请求失败: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	tk := ""
	if d, ok := result["data"].(map[string]interface{}); ok {
		tk, _ = d["tk"].(string)
	}
	if tk == "" {
		tk, _ = result["tk"].(string)
	}
	if tk == "" {
		return "", fmt.Errorf("finger_tk获取失败: status=%d body=%s", resp.Status, truncateStr(string(body), 300))
	}
	return tk, nil
}

func wxJdSignSilentAuth(data map[string]string) string {
	extra := map[string]string{"cmd": "52", "sub_cmd": "1", "gsalt": wxJdSignGSALT}
	order := []string{"appid", "wxappid", "client_ver", "ts", "cmd", "sub_cmd", "gsalt"}
	var parts []string
	for _, k := range order {
		if v, ok := data[k]; ok && v != "" {
			parts = append(parts, v)
		} else if v, ok := extra[k]; ok {
			parts = append(parts, v)
		} else {
			parts = append(parts, "")
		}
	}
	raw := strings.Join(parts, "")
	h := md5.Sum([]byte(raw))
	return hex.EncodeToString(h[:])
}

func wxJdSilentAuthLogin(code string, eidToken string) (string, string, error) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	data := map[string]string{
		"globalTokenSource": "",
		"code":              code,
		"token":             "",
		"salt":              "",
		"user_data":         "",
		"user_iv":           "",
		"eid_token":         eidToken,
		"goToLogin":         "true",
		"returnurl":         "/pages/login/web-view/web-view",
		"wxappid":           WxJdAppID,
		"appid":             wxJdJDAppID,
		"client_ver":        wxJdClientVer,
		"ts":                ts,
	}
	data["sign"] = wxJdSignSilentAuth(data)
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	req, _ := http.NewRequest("POST",
		"https://wxapplogin.m.jd.com/cgi-bin/jxpp/silentauthlogin",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("User-Agent", wxJdUA)
	req.Header.Set("Referer", wxJdReferer)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("cookie", "guid=; pt_pin=; pt_key=; pt_token=")
	req.Header.Set("Accept", "*/*")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("silentauthlogin请求失败: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("silentauthlogin失败: status=%d body=%s", resp.StatusCode, truncateStr(string(body), 400))
	}
	errCode, _ := result["err_code"].(float64)
	ptKey, _ := result["pt_key"].(string)
	ptPin, _ := result["pt_pin"].(string)
	if errCode == 128 {
		jmpURL, _ := result["jmp_url"].(string)
		errMsg, _ := result["err_msg"].(string)
		return "", "", &RiskVerifyError{
			JmpURL: jmpURL,
			ErrMsg: errMsg,
		}
	}
	if errCode != 0 || ptKey == "" || ptPin == "" {
		return "", "", fmt.Errorf("silentauthlogin失败: err_code=%.0f body=%s", errCode, truncateStr(string(body), 400))
	}
	return ptKey, ptPin, nil
}

// WxJdSilentAuthLogin 京东静默登录（供应用宝京东刷新复用）
func WxJdSilentAuthLogin(code, eidToken string) (string, string, error) {
	return wxJdSilentAuthLogin(code, eidToken)
}

// WxJdGetFingerTk 获取京东 finger token 备用
func WxJdGetFingerTk() (string, error) {
	return wxJdGetFingerTk()
}

func wxJdRefreshCK(wxid string) (string, string, error) {
	code, err := ProtocolGetWxAppCode(wxid, WxJdAppID)
	if err != nil {
		return "", "", fmt.Errorf("获取wx code失败: %v", err)
	}
	eidToken := ProtocolGetEidTokenForRef(wxid)
	if eidToken == "" {
		Info("eid_token为空，尝试finger_tk")
		tk, err := wxJdGetFingerTk()
		if err == nil {
			eidToken = tk
		}
	}
	ptKey, ptPin, err := wxJdSilentAuthLogin(code, eidToken)
	if err != nil {
		return "", "", err
	}
	return ptKey, ptPin, nil
}

func findUserProtocolDevices(sender *Sender) []string {
	var devices []string
	seen := make(map[string]bool)
	
	// 同时查询新旧两个地址
	raw, err := getWxUserStatusRaw()
	if err != nil || raw == nil {
		raw = &portalWxUserStatusResponse{Data: make(map[string]portalWxDeviceInfo)}
	}
	
	var rawOld *portalWxUserStatusResponse
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		rawOld, _ = getWxUserStatusRawFromURL(getOldWxLoginBaseURL())
	}
	
	var rawNew *portalWxUserStatusResponse
	if Config.WxProtocol.NewLoginBaseURL != "" && getWxLoginBaseURL() != getNewWxLoginBaseURL() {
		rawNew, _ = getWxUserStatusRawFromURL(getNewWxLoginBaseURL())
	}
	
	// 合并多个来源的数据（在线优先）
	mergedRaw := mergePortalWxRaw(raw, rawOld, rawNew)

	if sender.WxId != "" {
		if info, ok := mergedRaw.Data[sender.WxId]; ok && info.Survival == 1 {
			devices = append(devices, sender.WxId)
			seen[sender.WxId] = true
		}
	}

	var user User
	if db.Where("number = ?", sender.UserID).First(&user).Error == nil {
		if user.Wxid != "" && !seen[user.Wxid] {
			if info, ok := mergedRaw.Data[user.Wxid]; ok && info.Survival == 1 {
				devices = append(devices, user.Wxid)
				seen[user.Wxid] = true
			}
		}
		var portalDevices []PortalWxDevice
		db.Where("user_number = ?", user.Number).Find(&portalDevices)
		for _, d := range portalDevices {
			if seen[d.Wxid] {
				continue
			}
			if info, ok := mergedRaw.Data[d.Wxid]; ok && info.Survival == 1 {
				devices = append(devices, d.Wxid)
				seen[d.Wxid] = true
			}
		}
	}
	return devices
}

func handleWxJdLogin(sender *Sender, msg chan string) {
	defer func() {
		delete(wxJdList, sender.UserID)
	}()

	devices := findUserProtocolDevices(sender)
	if len(devices) == 0 {
		sender.Reply("❌ 未检测到在线的微信协议设备\n\n请先发送「微信扫码登录」注册设备，注册完成后再使用此功能。\n已退出登录流程。")
		return
	}

	var devMenu strings.Builder
	devMenu.WriteString("📱 请选择微信协议设备：\n\n")
	
	// 获取合并后的设备状态数据
	raw, _ := getWxUserStatusRaw()
	if raw == nil {
		raw = &portalWxUserStatusResponse{Data: make(map[string]portalWxDeviceInfo)}
	}
	var rawOld *portalWxUserStatusResponse
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		rawOld, _ = getWxUserStatusRawFromURL(getOldWxLoginBaseURL())
	}
	var rawNew *portalWxUserStatusResponse
	if Config.WxProtocol.NewLoginBaseURL != "" && getWxLoginBaseURL() != getNewWxLoginBaseURL() {
		rawNew, _ = getWxUserStatusRawFromURL(getNewWxLoginBaseURL())
	}
	mergedRaw := mergePortalWxRaw(raw, rawOld, rawNew)
	
	for i, d := range devices {
		nick := d
		deviceInfo := ""
		if info, ok := mergedRaw.Data[d]; ok {
			if info.Nickname != "" {
				nick = info.Nickname
			}
			if info.Device != "" {
				deviceInfo = info.Device
			}
		}
		// 显示设备所在的服务器地址类型
		serverURL := getWxJdServerForDevice(d)
		serverType := "旧"
		if serverURL == strings.TrimRight(getNewWxLoginBaseURL(), "/") {
			serverType = "新"
		} else if serverURL == strings.TrimRight(getOldWxLoginBaseURL(), "/") {
			serverType = "旧"
		}
		if deviceInfo != "" {
			devMenu.WriteString(fmt.Sprintf("%d、%s (%s) [%s设备 - %s]\n", i+1, nick, d, serverType, deviceInfo))
		} else {
			devMenu.WriteString(fmt.Sprintf("%d、%s (%s) [%s设备]\n", i+1, nick, d, serverType))
		}
	}
	devMenu.WriteString("\n输入序号刷新对应设备，输入 0 刷新全部，输入 q 退出：")
	sender.Reply(devMenu.String())

	devInput, ok := WaitJdBotInput(sender, msg, 60)
	if !ok {
		return
	}
	if devInput == "q" || devInput == "Q" {
		sender.Reply("已退出登录流程")
		return
	}
	devIdx, err := strconv.Atoi(devInput)
	if err != nil || devIdx < 0 || devIdx > len(devices) {
		sender.Reply("输入无效，已退出登录流程")
		return
	}

	if devIdx == 0 {
		sender.Reply(fmt.Sprintf("⏳ 正在刷新全部 %d 个设备，请稍候...", len(devices)))
		success := 0
		fail := 0
		for i, d := range devices {
			if i > 0 {
				time.Sleep(time.Duration(rand.Intn(2000)+3000) * time.Millisecond)
			}
			if wxJdRefreshByDevice(sender, d) {
				success++
			} else {
				fail++
			}
		}
		sender.Reply(fmt.Sprintf("🔄 批量刷新完成\n✅ 成功: %d\n❌ 失败: %d", success, fail))
	} else {
		sender.Reply("⏳ 正在刷新，请稍候...")
		wxJdRefreshByDevice(sender, devices[devIdx-1])
	}
}

func wxJdRefreshByDevice(sender *Sender, wxid string) bool {
	ptKey, ptPin, err := wxJdRefreshCK(wxid)
	if err != nil {
		var riskErr *RiskVerifyError
		if errors.As(err, &riskErr) {
			if WaitJdRiskVerify(sender, riskErr.JmpURL) {
				return wxJdRefreshByDevice(sender, wxid)
			}
			return false
		}
		sender.Reply(fmt.Sprintf("❌ 刷新失败: %v", err))
		return false
	}
	newCK := &JdCookie{PtKey: ptKey, PtPin: url.QueryEscape(ptPin)}
	if !CookieOK(newCK) {
		sender.Reply("❌ 刷新成功但CK验证无效，可能被风控，请稍后重试")
		return false
	}
	nick := ptPin
	encodedPin := url.QueryEscape(ptPin)
	if existingCK, e := GetJdCookie(encodedPin); e == nil {
		existingCK.Updates(JdCookie{PtKey: ptKey, Available: True, WeiXin: wxid, WxPid: wxid, UpdateAt: Date()})
		if existingCK.Nickname != "" {
			nick = existingCK.Nickname
		}
	} else {
		newCK.WeiXin = wxid
		newCK.WxPid = wxid
		newCK.QQ = sender.UserID
		newCK.Available = True
		NewJdCookie(newCK)
	}
	
	// 获取设备所在的服务器地址类型
	serverURL := getWxJdServerForDevice(wxid)
	serverType := "旧"
	if serverURL == strings.TrimRight(getNewWxLoginBaseURL(), "/") {
		serverType = "新"
	}
	
	sender.Reply(fmt.Sprintf("✅ [%s] CK刷新成功！(设备类型: %s)", nick, serverType))
	(&JdCookie{}).Push(fmt.Sprintf("微信协议刷新成功: %s (设备: %s, 类型: %s)", nick, wxid, serverType))
	go func() {
		Save <- &JdCookie{}
	}()
	return true
}

// WaitJdRiskVerify 京东风控短信验证后等待用户确认继续
func WaitJdRiskVerify(sender *Sender, riskURL string) bool {
	sender.Reply("⚠️ 账号需要短信验证\n\n🔗 请点击网址进行验证：\n" + riskURL + "\n\n验证完成后回复 y 继续执行，回复 q 退出")
	if smsList[sender.UserID] == nil {
		smsList[sender.UserID] = make(chan string)
	}
	defer delete(smsList, sender.UserID)
	timeout := time.After(200 * time.Second)
	for {
		select {
		case msg, ok := <-smsList[sender.UserID]:
			if !ok {
				sender.Reply("通道已关闭，退出验证流程")
				return false
			}
			if msg == "q" || msg == "Q" {
				sender.Reply("已退出验证流程")
				return false
			}
			if msg == "y" || msg == "Y" {
				sender.Reply("⏳ 验证通过，正在重新刷新...")
				return true
			}
			sender.Reply("无效输入，请回复 y 继续，或回复 q 退出")
		case <-timeout:
			sender.Reply("⏰ 操作超时，已退出验证流程")
			return false
		}
	}
}

// WaitJdBotInput 机器人交互等待用户输入
func WaitJdBotInput(sender *Sender, msg chan string, timeoutSec int) (string, bool) {
	timeout := time.After(time.Duration(timeoutSec) * time.Second)
	select {
	case input, ok := <-msg:
		if !ok {
			sender.Reply("通道已关闭，退出登录流程")
			return "", false
		}
		return input, true
	case <-timeout:
		sender.Reply("操作超时，已退出登录流程")
		return "", false
	}
}

func wxJdWaitInput(sender *Sender, msg chan string, timeoutSec int) (string, bool) {
	return WaitJdBotInput(sender, msg, timeoutSec)
}

func wxJdFingerEncode(obj map[string]interface{}) string {
	jsonBytes, _ := json.Marshal(obj)
	text := url.QueryEscape(string(jsonBytes))
	var out strings.Builder
	i := 0
	for i < len(text) {
		e := int(text[i])
		r := 0
		u := 0
		if i+1 < len(text) {
			r = int(text[i+1])
		} else {
			r = -1
		}
		if i+2 < len(text) {
			u = int(text[i+2])
		} else {
			u = -1
		}
		a := e >> 2
		c := (3 & e) << 4
		var s, f int
		if r < 0 {
			s = 64
			f = 64
		} else if u < 0 {
			c |= r >> 4
			s = (15 & r) << 2
			f = 64
		} else {
			c |= r >> 4
			s = (15&r)<<2 | u>>6
			f = 63 & u
		}
		out.WriteByte(wxJdAlpha[a])
		out.WriteByte(wxJdAlpha[c])
		out.WriteByte(wxJdAlpha[s])
		out.WriteByte(wxJdAlpha[f])
		i += 3
	}
	out.WriteByte('/')
	return out.String()
}

func randHex(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return fmt.Sprintf("%x", b)
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func truncateObj(v interface{}, maxLen int) string {
	b, _ := json.Marshal(v)
	return truncateStr(string(b), maxLen)
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
