package models

import (
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/buger/jsonparser"
)

type PortalJdAccount struct {
	Index      int    `json:"index"`
	Pin        string `json:"pin"`
	Nickname   string `json:"nickname"`
	StatusText string `json:"statusText"`
	Valid      bool   `json:"valid"`
}

type PortalJdSmsVerifyResult struct {
	Message     string `json:"message"`
	QueryResult string `json:"queryResult,omitempty"`
	NeedIdVerify bool  `json:"needIdVerify,omitempty"`
}

type PortalJdWxDevice struct {
	Index      int    `json:"index"`
	Wxid       string `json:"wxid"`
	Nickname   string `json:"nickname"`
	Device     string `json:"device"`
	ServerType string `json:"serverType"`
}

type PortalJdWxRefreshResult struct {
	Success        int      `json:"success"`
	Fail           int      `json:"fail"`
	Details        []string `json:"details"`
	NeedRiskVerify bool     `json:"needRiskVerify,omitempty"`
	RiskURL        string   `json:"riskUrl,omitempty"`
	RiskMsg        string   `json:"riskMsg,omitempty"`
}

var portalSmsRisk = make(map[int]bool)
var portalSmsPhone = make(map[int]string)

type portalWxRiskInfo struct {
	wxid    string
	riskUrl string
	riskMsg string
}

var portalWxJdRisk = make(map[int]portalWxRiskInfo)

func getPortalJdCookies(userNumber int) []JdCookie {
	cks := []JdCookie{}
	db.Where("QQ = ?", userNumber).Order("priority desc, ID asc").Find(&cks)
	return cks
}

// resolveJdTaskAccountsByIndex 与 Portal/App 任务选择一致：0=所有有效账号，N=有效账号列表中第 N 个（1-based）
func resolveJdTaskAccountsByIndex(userNumber int, accountIndexes []int, logChan chan string) ([]JdCookie, error) {
	allCks := getPortalJdCookies(userNumber)
	validCks := make([]JdCookie, 0, len(allCks))
	for i := range allCks {
		if CookieOK(&allCks[i]) {
			validCks = append(validCks, allCks[i])
		}
	}

	if logChan != nil {
		safeLogSend(logChan, fmt.Sprintf("查询到 %d 个有效账号 (userId=%d)", len(validCks), userNumber))
	}
	if len(validCks) == 0 {
		if logChan != nil {
			safeLogSend(logChan, "错误: 没有找到有效的京东账号")
		}
		return nil, fmt.Errorf("没有找到有效的京东账号")
	}

	var selectedCks []JdCookie
	pickAll := false
	seen := make(map[int]bool)
	for _, idx := range accountIndexes {
		if idx == 0 {
			pickAll = true
			break
		}
		if idx < 1 || idx > len(validCks) || seen[idx] {
			continue
		}
		seen[idx] = true
		selectedCks = append(selectedCks, validCks[idx-1])
	}
	if pickAll {
		selectedCks = validCks
	}

	if logChan != nil {
		safeLogSend(logChan, fmt.Sprintf("筛选后 %d 个账号 (传入索引: %v)", len(selectedCks), accountIndexes))
		for i, ck := range selectedCks {
			pin, _ := url.QueryUnescape(ck.PtPin)
			safeLogSend(logChan, fmt.Sprintf("  [%d] %s (%s)", i+1, ck.Nickname, pin))
		}
	}
	if len(selectedCks) == 0 {
		if logChan != nil {
			safeLogSend(logChan, "错误: 没有选择有效的账号")
		}
		return nil, fmt.Errorf("没有选择有效的账号")
	}
	return selectedCks, nil
}

func GetPortalJdAccounts(userNumber int) []PortalJdAccount {
	cks := getPortalJdCookies(userNumber)
	result := make([]PortalJdAccount, 0, len(cks))
	for i, ck := range cks {
		statusText, valid := GetAccountStatusText(&ck)
		decodedPin, _ := url.QueryUnescape(ck.PtPin)
		result = append(result, PortalJdAccount{
			Index:      i + 1,
			Pin:        decodedPin,
			Nickname:   ck.Nickname,
			StatusText: statusText,
			Valid:      valid,
		})
	}
	return result
}

func PortalJdQuery(userNumber int, index int) (string, error) {
	cks := getPortalJdCookies(userNumber)
	if len(cks) == 0 {
		return "", fmt.Errorf("没有找到您的有效账号，请先登录京东账号")
	}
	if index == 0 {
		var sb strings.Builder
		for i, ck := range cks {
			if i > 0 {
				sb.WriteString("\n\n────────────────────\n\n")
			}
			sb.WriteString(ck.Query())
		}
		return sb.String(), nil
	}
	if index < 1 || index > len(cks) {
		return "", fmt.Errorf("无效的账号序号")
	}
	return cks[index-1].Query(), nil
}

func portalJdSmsProvider() string {
	switch GetEnv("短信登录") {
	case "兔子":
		return "Rabbit"
	default:
		return "Nolan"
	}
}

func PortalJdSmsSend(userNumber int, phone string) (string, error) {
	phoneRegex := regexp.MustCompile(`^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`)
	if !phoneRegex.MatchString(phone) {
		return "", fmt.Errorf("手机号格式错误")
	}
	portalSmsPhone[userNumber] = phone
	delete(portalSmsRisk, userNumber)

	provider := portalJdSmsProvider()
	switch provider {
	case "Nolan":
		if sysConfig.NolanUrl == "" || sysConfig.NolanToken == "" {
			return "", fmt.Errorf("配置错误：Nolan 接口信息不完整")
		}
		return portalNolanSendSMS(phone)
	case "Rabbit":
		if sysConfig.RabbitUrl == "" {
			return "", fmt.Errorf("配置错误：Rabbit 接口信息不完整")
		}
		return portalRabbitSendSMS(phone)
	default:
		return "", fmt.Errorf("不支持的短信平台")
	}
}

func portalNolanSendSMS(phone string) (string, error) {
	req := httplib.Post(sysConfig.NolanUrl + "/sms/SendSMS")
	req.Header("content-type", "application/json")
	payload := fmt.Sprintf(`{"phone":"%s","botApitoken":"%s"}`, phone, sysConfig.NolanToken)
	data, err := req.Body(payload).Bytes()
	if err != nil {
		return "", fmt.Errorf("验证码请求失败，请稍后重试")
	}
	success, _ := jsonparser.GetBoolean(data, "success")
	message, _ := jsonparser.GetString(data, "message")
	if success {
		return "验证码已发送，请输入验证码", nil
	}
	if message == "" {
		message = "发送失败，未知错误"
	}
	return "", fmt.Errorf("验证码发送失败：%s", message)
}

func portalRabbitSendSMS(phone string) (string, error) {
	body := fmt.Sprintf(`{"Phone":"%s"}`, phone)
	req := httplib.Post(fmt.Sprintf("%s/bot/mck/sendSMS?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	req.Header("Content-Type", "application/json; charset=utf-8")
	data, err := req.Body(body).Bytes()
	if err != nil {
		return "", fmt.Errorf("验证码请求失败，请稍后重试")
	}
	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	status, _ := jsonparser.GetInt(data, "data", "status")
	if message != "" && status != 666 {
		// 非成功状态先记录 message
	}
	if success {
		return "验证码已发送，请输入验证码", nil
	}
	for attempt := 1; attempt <= 5; attempt++ {
		time.Sleep(3 * time.Second)
		autoReq := httplib.Post(fmt.Sprintf("%s/bot/mck/AutoCaptcha?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
		autoReq.Header("Content-Type", "application/json; charset=utf-8")
		data, err = autoReq.Body(body).Bytes()
		if err != nil {
			continue
		}
		message, _ = jsonparser.GetString(data, "message")
		success, _ = jsonparser.GetBoolean(data, "success")
		status, _ = jsonparser.GetInt(data, "data", "status")
		if success {
			return "验证码已发送，请输入验证码", nil
		}
		if status != 666 && status != 505 {
			if message != "" {
				return "", fmt.Errorf("%s", message)
			}
			return "", fmt.Errorf("验证失败，请重新登录")
		}
	}
	return "", fmt.Errorf("滑块验证多次失败，请稍后重试")
}

func PortalJdSmsVerify(userNumber int, phone, code, idCard string) (*PortalJdSmsVerifyResult, error) {
	provider := portalJdSmsProvider()
	if phone == "" {
		phone = portalSmsPhone[userNumber]
	}
	if phone == "" {
		return nil, fmt.Errorf("请先发送验证码")
	}

	codeRegex := regexp.MustCompile(`^\d{5}(\d|X|x)$`)
	if portalSmsRisk[userNumber] {
		if !codeRegex.MatchString(strings.ToUpper(code)) && idCard == "" {
			return nil, fmt.Errorf("请输入身份证前两位和后四位")
		}
	} else if !codeRegex.MatchString(code) {
		return nil, fmt.Errorf("验证码格式错误")
	}

	switch provider {
	case "Nolan":
		if portalSmsRisk[userNumber] {
			return portalNolanVerifyCard(userNumber, phone, code, idCard)
		}
		return portalNolanVerifyCode(userNumber, phone, code)
	case "Rabbit":
		return portalRabbitVerifyCode(userNumber, phone, code)
	default:
		return nil, fmt.Errorf("不支持的短信平台")
	}
}

func portalSaveJdCookieFromSms(userNumber int, ptKey, ptPin string) (string, error) {
	encodedPin := url.QueryEscape(ptPin)
	ck := JdCookie{
		PtPin:     encodedPin,
		PtKey:     ptKey,
		Available: True,
		QQ:        userNumber,
	}
	if nck, err := GetJdCookie(ck.PtPin); err == nil {
		cookie := JdCookie{
			QQ:        userNumber,
			PtKey:     ptKey,
			Available: True,
		}
		if nck.Password == "" {
			cookie.Smsverify = ""
		} else {
			cookie.Smsverify = "false"
		}
		nck.Updates(cookie)
		go func() { Save <- &JdCookie{} }()
		(&JdCookie{}).Push(fmt.Sprintf("来自网页短信登录成功:%s", ptPin))
		return fmt.Sprintf("短信登录成功：%s", ptPin), nil
	}
	NewJdCookie(&ck)
	go func() { Save <- &JdCookie{} }()
	(&JdCookie{}).Push(fmt.Sprintf("来自网页短信添加账号，账号名:%s", ck.PtPin))
	queryText := ck.Query()
	return queryText, nil
}

func portalNolanVerifyCode(userNumber int, phone, code string) (*PortalJdSmsVerifyResult, error) {
	req := httplib.Post(fmt.Sprintf("%s/sms/VerifyCode", sysConfig.NolanUrl))
	req.Header("Content-Type", "application/json; charset=utf-8")
	payload := fmt.Sprintf(`{"phone":"%s","code":"%s","botApitoken":"%s"}`, phone, code, sysConfig.NolanToken)
	data, _ := req.Body(payload).Bytes()
	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	ck, _ := jsonparser.GetString(data, "data", "ck")
	state, _ := jsonparser.GetInt(data, "data", "status")
	if success {
		ptKey := FetchJdCookieValue("pt_key", ck)
		ptPin := FetchJdCookieValue("pt_pin", ck)
		queryText, err := portalSaveJdCookieFromSms(userNumber, ptKey, ptPin)
		if err != nil {
			return nil, err
		}
		delete(portalSmsRisk, userNumber)
		delete(portalSmsPhone, userNumber)
		return &PortalJdSmsVerifyResult{Message: "登录成功", QueryResult: queryText}, nil
	}
	if state == 555 {
		mode, _ := jsonparser.GetString(data, "data", "mode")
		if mode == "USER_ID" {
			portalSmsRisk[userNumber] = true
			portalSmsPhone[userNumber] = phone
			return &PortalJdSmsVerifyResult{
				Message:      "账号需要身份证验证",
				NeedIdVerify: true,
			}, nil
		}
		return nil, fmt.Errorf("请使用手机进行验证后重新登录")
	}
	if state == 404 && message == "验证码输入错误" {
		return nil, fmt.Errorf("验证码输入错误，请重新输入")
	}
	if message == "" {
		message = "验证失败"
	}
	return nil, fmt.Errorf("%s", message)
}

func portalNolanVerifyCard(userNumber int, phone, code, idCard string) (*PortalJdSmsVerifyResult, error) {
	verifyCode := code
	if idCard != "" {
		verifyCode = idCard
	}
	req := httplib.Post(fmt.Sprintf("%s/sms/VerifyCard", sysConfig.NolanUrl))
	req.Header("Content-Type", "application/json; charset=utf-8")
	payload := fmt.Sprintf(`{"phone":"%s","code":"%s","botApitoken":"%s"}`, phone, verifyCode, sysConfig.NolanToken)
	data, _ := req.Body(payload).Bytes()
	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	ck, _ := jsonparser.GetString(data, "data", "ck")
	if success {
		ptKey := FetchJdCookieValue("pt_key", ck)
		ptPin := FetchJdCookieValue("pt_pin", ck)
		queryText, err := portalSaveJdCookieFromSms(userNumber, ptKey, ptPin)
		if err != nil {
			return nil, err
		}
		delete(portalSmsRisk, userNumber)
		delete(portalSmsPhone, userNumber)
		return &PortalJdSmsVerifyResult{Message: "登录成功", QueryResult: queryText}, nil
	}
	if message == "" {
		message = "身份证验证失败"
	}
	return nil, fmt.Errorf("%s", message)
}

func portalRabbitVerifyCode(userNumber int, phone, code string) (*PortalJdSmsVerifyResult, error) {
	req := httplib.Post(fmt.Sprintf("%s/bot/mck/VerifyCode?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	req.Header("Content-Type", "application/json; charset=utf-8")
	data, _ := req.Body(fmt.Sprintf(`{"Phone":%s,"Code":"%s"}`, phone, code)).Bytes()
	message, _ := jsonparser.GetString(data, "message")
	pin, _ := jsonparser.GetString(data, "pin")
	state, _ := jsonparser.GetInt(data, "code")
	wskey, _ := jsonparser.GetString(data, "wskey")
	appck, _ := jsonparser.GetString(data, "ck")
	if state == 200 {
		ptKey := FetchJdCookieValue("pt_key", appck)
		ck := JdCookie{
			PtPin:     pin,
			PtKey:     ptKey,
			WsKey:     wskey,
			Available: True,
			QQ:        userNumber,
		}
		if nck, err := GetJdCookie(ck.PtPin); err == nil {
			nck.Updates(JdCookie{WsKey: wskey, QQ: userNumber, PtKey: ptKey, Available: True})
			go func() { Save <- &JdCookie{} }()
			(&JdCookie{}).Push(fmt.Sprintf("网页登录成功:%s", pin))
			return &PortalJdSmsVerifyResult{Message: fmt.Sprintf("登录成功：%s", pin)}, nil
		}
		NewJdCookie(&ck)
		go func() { Save <- &JdCookie{} }()
		(&JdCookie{}).Push(fmt.Sprintf("网页添加账号，账号名:%s", ck.PtPin))
		return &PortalJdSmsVerifyResult{Message: "登录成功", QueryResult: ck.Query()}, nil
	}
	if message == "" {
		message = "验证失败"
	}
	return nil, fmt.Errorf("%s", message)
}

func GetPortalJdWxDevices(userNumber int) ([]PortalJdWxDevice, error) {
	sender := &Sender{UserID: userNumber}
	devices := findUserProtocolDevices(sender)
	if len(devices) == 0 {
		return nil, fmt.Errorf("未检测到在线的微信协议设备，请先在「微信协议」页面扫码登录")
	}

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

	result := make([]PortalJdWxDevice, 0, len(devices))
	for i, wxid := range devices {
		nick := wxid
		deviceInfo := ""
		if info, ok := mergedRaw.Data[wxid]; ok {
			if info.Nickname != "" {
				nick = info.Nickname
			}
			deviceInfo = info.Device
		}
		serverURL := getWxJdServerForDevice(wxid)
		serverType := "旧"
		if serverURL == strings.TrimRight(getNewWxLoginBaseURL(), "/") {
			serverType = "新"
		}
		result = append(result, PortalJdWxDevice{
			Index:      i + 1,
			Wxid:       wxid,
			Nickname:   nick,
			Device:     deviceInfo,
			ServerType: serverType,
		})
	}
	return result, nil
}

func PortalJdWxRefresh(userNumber int, wxid string, riskConfirmed bool) (*PortalJdWxRefreshResult, error) {
	sender := &Sender{UserID: userNumber}
	devices := findUserProtocolDevices(sender)
	if len(devices) == 0 {
		return nil, fmt.Errorf("未检测到在线的微信协议设备")
	}

	targets := devices
	if wxid != "" && wxid != "all" {
		found := false
		for _, d := range devices {
			if d == wxid {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("设备不在线或不存在")
		}
		targets = []string{wxid}
	}

	result := &PortalJdWxRefreshResult{Details: []string{}}
	for i, d := range targets {
		if i > 0 {
			time.Sleep(time.Duration(rand.Intn(2000)+3000) * time.Millisecond)
		}
		detail, needRisk, riskURL, riskMsg, ok := portalWxJdRefreshOne(userNumber, d, riskConfirmed)
		result.Details = append(result.Details, detail)
		if needRisk {
			result.NeedRiskVerify = true
			result.RiskURL = riskURL
			result.RiskMsg = riskMsg
			if _, exists := portalWxJdRisk[userNumber]; !exists {
				portalWxJdRisk[userNumber] = portalWxRiskInfo{wxid: d, riskUrl: riskURL, riskMsg: riskMsg}
			}
			return result, nil
		}
		if ok {
			result.Success++
		} else {
			result.Fail++
		}
	}
	delete(portalWxJdRisk, userNumber)
	return result, nil
}

func portalWxJdRefreshOne(userNumber int, wxid string, riskConfirmed bool) (detail string, needRisk bool, riskURL, riskMsg string, ok bool) {
	if pending, exists := portalWxJdRisk[userNumber]; exists && pending.wxid == wxid && !riskConfirmed {
		return "请先完成短信验证后再继续", true, pending.riskUrl, pending.riskMsg, false
	}

	ptKey, ptPin, err := wxJdRefreshCK(wxid)
	if err != nil {
		var riskErr *RiskVerifyError
		if errors.As(err, &riskErr) {
			portalWxJdRisk[userNumber] = portalWxRiskInfo{wxid: wxid, riskUrl: riskErr.JmpURL, riskMsg: riskErr.ErrMsg}
			return fmt.Sprintf("账号需要短信验证"), true, riskErr.JmpURL, riskErr.ErrMsg, false
		}
		return fmt.Sprintf("❌ %s 刷新失败: %v", wxid, err), false, "", "", false
	}
	newCK := &JdCookie{PtKey: ptKey, PtPin: url.QueryEscape(ptPin)}
	if !CookieOK(newCK) {
		return fmt.Sprintf("❌ %s 刷新成功但CK验证无效", wxid), false, "", "", false
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
		newCK.QQ = userNumber
		newCK.Available = True
		NewJdCookie(newCK)
	}
	delete(portalWxJdRisk, userNumber)
	go func() { Save <- &JdCookie{} }()
	(&JdCookie{}).Push(fmt.Sprintf("微信协议刷新成功: %s (设备: %s)", nick, wxid))
	return fmt.Sprintf("✅ %s CK刷新成功", nick), false, "", "", true
}

func PortalJdWxContinueAfterRisk(userNumber int) (*PortalJdWxRefreshResult, error) {
	info, ok := portalWxJdRisk[userNumber]
	if !ok || info.wxid == "" {
		return nil, fmt.Errorf("没有待继续的验证任务")
	}
	return PortalJdWxRefresh(userNumber, info.wxid, true)
}
