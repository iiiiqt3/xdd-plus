package models

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"gorm.io/gorm"
)

const (
	wxJdAppID     = "wx73247c7819d61796"
	wxJdClientVer = "2.0.2"
	wxJdJDAppID   = "599"
	wxJdSignGSALT = "sb2cwlYyaCSN1KUv5RHG3tmqxfEb8NKN"
	wxJdBizKey    = "bce044c839bb9eb811aad5af18a629e199da4e13"
	wxJdFingerTk  = "L64RTJ562VJEYNEQN67XMUWSR4UFLOIQHJYZ3MWERRIKJGP24SDSBDS4I4AMVU24Y3Y7A4UPDICN2"
	wxJdAlpha     = "23IL<N01c7KvwZO56RSTAfghiFyzWJqVabGH4PQdopUrsCuX*xeBjkltDEmn89.-/"
	wxJdReferer   = "https://servicewechat.com/" + wxJdAppID + "/864/page-frame.html"
	wxJdUA        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13) UnifiedPCWindowsWechat(0xf254186b) XWEB/19481"
)

var wxJdList = make(map[int]chan string)

func getWxJdServer() string {
	if Config.WxProtocol.JdServer != "" {
		return strings.TrimRight(Config.WxProtocol.JdServer, "/")
	}
	return strings.TrimRight(Config.WxProtocol.LoginBaseURL, "/")
}

func wxJdPost(paths []string, body interface{}, timeout int) (map[string]interface{}, error) {
	base := getWxJdServer()
	jsonData, _ := json.Marshal(body)
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	var lastErr error
	for _, p := range paths {
		req, err := http.NewRequest("POST", base+p, strings.NewReader(string(jsonData)))
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
	data, err := wxJdPost(
		[]string{"/api/v1/wx/app/get/code", "/wx/app/get/code"},
		map[string]string{"wxid": wxid, "appid": wxJdAppID},
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
	payload := map[string]interface{}{
		"api_name":       "webapi_getuserinfo",
		"data":           map[string]string{"lang": "zh_CN"},
		"with_credentials": true,
	}
	data, err := wxJdPost(
		[]string{"/api/v1/wx/app/call/function", "/wx/app/call/function"},
		map[string]interface{}{"wxid": wxid, "appid": wxJdAppID, "data": mustJSON(payload)},
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
		"vid": wxJdAppID, "bk": wxJdBizKey, "cliet": now, "fp": randHex(16),
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
		"wxappid":           wxJdAppID,
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
	if errCode != 0 || ptKey == "" || ptPin == "" {
		return "", "", fmt.Errorf("silentauthlogin失败: err_code=%.0f body=%s", errCode, truncateStr(string(body), 400))
	}
	return ptKey, ptPin, nil
}

func wxJdRefreshCK(wxid string) (string, string, error) {
	code, err := wxJdGetWxCode(wxid)
	if err != nil {
		return "", "", fmt.Errorf("获取wx code失败: %v", err)
	}
	eidToken := wxJdGetEidToken(wxid)
	if eidToken == "" {
		logs.Info("eid_token为空，尝试finger_tk")
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

func findWxProtocolIDForCK(ck *JdCookie) (string, bool) {
	if ck.WeiXin != "" {
		online, err := checkWxDeviceOnline(ck.WeiXin)
		if err == nil && online {
			return ck.WeiXin, true
		}
	}
	return "", false
}

func findUserProtocolDevices(sender *Sender) []string {
	var devices []string
	raw, err := getWxUserStatusRaw()
	if err != nil || raw == nil {
		return devices
	}
	if sender.WxId != "" {
		if info, ok := raw.Data[sender.WxId]; ok && info.Survival == 1 {
			devices = append(devices, sender.WxId)
		}
	}
	if len(devices) > 0 {
		return devices
	}
	var user User
	if db.Where("number = ?", sender.UserID).First(&user).Error == nil {
		if user.Wxid != "" {
			if info, ok := raw.Data[user.Wxid]; ok && info.Survival == 1 {
				devices = append(devices, user.Wxid)
			}
		}
		var portalDevices []PortalWxDevice
		db.Where("user_number = ?", user.Number).Find(&portalDevices)
		for _, d := range portalDevices {
			if d.Wxid == user.Wxid {
				continue
			}
			if info, ok := raw.Data[d.Wxid]; ok && info.Survival == 1 {
				devices = append(devices, d.Wxid)
			}
		}
	}
	return devices
}

func wxJdListUserCKs(sender *Sender) []JdCookie {
	var cks []JdCookie
	var user User
	if db.Where("number = ?", sender.UserID).First(&user).Error != nil {
		return cks
	}
	cks = GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where("QQ = ?", user.Number)
	})
	return cks
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

	cks := wxJdListUserCKs(sender)
	if len(cks) == 0 {
		sender.Reply("❌ 未找到你绑定的京东账号，请先提交CK或通过短信登录上车。")
		return
	}

	var menu strings.Builder
	menu.WriteString("📋 请选择要刷新的京东账号：\n\n")
	raw, _ := getWxUserStatusRaw()
	for i, ck := range cks {
		status := "✅"
		if ck.Available == False || !CookieOK(&ck) {
			status = "❌"
		}
		nick := ck.Nickname
		if nick == "" {
			nick = ck.PtPin
		}
		wxNick := "未绑定微信协议"
		if ck.WeiXin != "" {
			wxNick = ck.WeiXin
			if raw != nil {
				if info, ok := raw.Data[ck.WeiXin]; ok && info.Nickname != "" {
					wxNick = info.Nickname + "(" + ck.WeiXin + ")"
				}
			}
		}
		menu.WriteString(fmt.Sprintf("%d、%s %s → 微信:%s\n", i+1, status, nick, wxNick))
	}
	menu.WriteString("\n输入序号刷新对应账号，输入 0 刷新全部，输入 q 退出：")
	sender.Reply(menu.String())

	input, ok := wxJdWaitInput(sender, msg, 60)
	if !ok {
		return
	}
	if input == "q" || input == "Q" {
		sender.Reply("已退出登录流程")
		return
	}
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 0 || idx > len(cks) {
		sender.Reply("输入无效，已退出登录流程")
		return
	}

	wxid := devices[0]
	if len(devices) > 1 {
		var devMenu strings.Builder
		devMenu.WriteString("检测到多个微信协议设备，请选择：\n\n")
		devRaw, _ := getWxUserStatusRaw()
		for i, d := range devices {
			nick := d
			if devRaw != nil {
				if info, ok := devRaw.Data[d]; ok && info.Nickname != "" {
					nick = info.Nickname
				}
			}
			devMenu.WriteString(fmt.Sprintf("%d、%s (%s)\n", i+1, nick, d))
		}
		sender.Reply(devMenu.String())
		devInput, ok := wxJdWaitInput(sender, msg, 60)
		if !ok {
			return
		}
		devIdx, err := strconv.Atoi(devInput)
		if err != nil || devIdx < 1 || devIdx > len(devices) {
			sender.Reply("输入无效，已退出登录流程")
			return
		}
		wxid = devices[devIdx-1]
	}

	if idx == 0 {
		wxJdRefreshAllCK(sender, cks, wxid)
	} else {
		ck := cks[idx-1]
		wxJdRefreshSingleCK(sender, &ck, wxid)
	}
}

func wxJdRefreshSingleCK(sender *Sender, ck *JdCookie, wxid string) {
	nick := ck.Nickname
	if nick == "" {
		nick = ck.PtPin
	}
	sender.Reply(fmt.Sprintf("⏳ 正在为 [%s] 刷新CK，请稍候...", nick))
	ptKey, ptPin, err := wxJdRefreshCK(wxid)
	if err != nil {
		sender.Reply(fmt.Sprintf("❌ 刷新失败: %v", err))
		return
	}
	newCK := &JdCookie{PtKey: ptKey, PtPin: ptPin}
	if !CookieOK(newCK) {
		sender.Reply("❌ 刷新成功但CK验证无效，可能被风控，请稍后重试")
		return
	}
	if existingCK, err := GetJdCookie(ptPin); err == nil {
		existingCK.Updates(JdCookie{
			PtKey:     ptKey,
			Available: True,
			WeiXin:    wxid,
			UpdateAt:  Date(),
		})
	} else {
		newCK.WeiXin = wxid
		newCK.QQ = sender.UserID
		newCK.Available = True
		NewJdCookie(newCK)
	}
	sender.Reply(fmt.Sprintf("✅ [%s] CK刷新成功！", nick))
	(&JdCookie{}).Push(fmt.Sprintf("微信协议自动刷新成功: %s (wxid: %s)", nick, wxid))
}

func wxJdRefreshAllCK(sender *Sender, cks []JdCookie, defaultWxid string) {
	sender.Reply(fmt.Sprintf("⏳ 正在批量刷新 %d 个账号，请稍候...", len(cks)))
	success := 0
	fail := 0
	for i, ck := range cks {
		if i > 0 {
			time.Sleep(time.Duration(rand.Intn(2000)+3000) * time.Millisecond)
		}
		wxid := defaultWxid
		if ck.WeiXin != "" {
			online, err := checkWxDeviceOnline(ck.WeiXin)
			if err == nil && online {
				wxid = ck.WeiXin
			}
		}
		nick := ck.Nickname
		if nick == "" {
			nick = ck.PtPin
		}
		ptKey, ptPin, err := wxJdRefreshCK(wxid)
		if err != nil {
			logs.Error("批量刷新失败 %s: %v", nick, err)
			fail++
			continue
		}
		newCK := &JdCookie{PtKey: ptKey, PtPin: ptPin}
		if !CookieOK(newCK) {
			fail++
			continue
		}
		if existingCK, err := GetJdCookie(ptPin); err == nil {
			existingCK.Updates(JdCookie{
				PtKey:     ptKey,
				Available: True,
				WeiXin:    wxid,
				UpdateAt:  Date(),
			})
			success++
		}
	}
	sender.Reply(fmt.Sprintf("🔄 批量刷新完成\n✅ 成功: %d\n❌ 失败: %d", success, fail))
	(&JdCookie{}).Push(fmt.Sprintf("微信协议批量刷新完成: 成功%d 失败%d", success, fail))
}

func wxJdWaitInput(sender *Sender, msg chan string, timeoutSec int) (string, bool) {
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
