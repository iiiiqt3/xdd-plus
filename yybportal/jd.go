package yybportal

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cdle/xdd/models"
	"gorm.io/gorm"
)

var (
	portalYybJdRisk        = make(map[int]portalYybRiskInfo)
	offlineNotifiedYybOIDs sync.Map
)

type portalYybRiskInfo struct {
	openid  string
	riskUrl string
	riskMsg string
}

// PortalJdYybAccount 门户京东应用宝账号
type PortalJdYybAccount struct {
	Index    int    `json:"index"`
	OpenID   string `json:"openid"`
	Nickname string `json:"nickname"`
	Status   string `json:"status"`
}

// PortalJdYybRefreshResult 门户应用宝京东刷新结果
type PortalJdYybRefreshResult struct {
	Success        int      `json:"success"`
	Fail           int      `json:"fail"`
	Details        []string `json:"details"`
	NeedRiskVerify bool     `json:"needRiskVerify,omitempty"`
	RiskURL        string   `json:"riskUrl,omitempty"`
	RiskMsg        string   `json:"riskMsg,omitempty"`
}

func findUserYybAccounts(userNumber int) ([]PortalAccountView, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	return ListUserAliveAccounts(userNumber)
}

func yybJdGetCode(openid string) (string, error) {
	data, err := InternalWxappGetCode(openid, models.WxJdAppID)
	if err != nil {
		return "", err
	}
	code, _ := data["code"].(string)
	if code == "" {
		return "", fmt.Errorf("获取应用宝 code 失败: %v", truncateMap(data, 300))
	}
	return code, nil
}

func yybJdGetEidToken(openid string) string {
	payload := map[string]any{
		"api_name":         "webapi_getuserinfo",
		"data":             map[string]string{"lang": "zh_CN"},
		"with_credentials": true,
	}
	data, err := InternalWxappOperate(openid, models.WxJdAppID, payload)
	if err != nil {
		return ""
	}
	return parseYybEidToken(data)
}

func parseYybEidToken(data map[string]any) string {
	if data == nil {
		return ""
	}
	for _, k := range []string{"eid_token", "eidToken", "eid"} {
		if v, ok := data[k].(string); ok && v != "" {
			return v
		}
	}
	b64Data, _ := data["data"].(string)
	if b64Data == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return ""
	}
	var parsed map[string]any
	if json.Unmarshal(decoded, &parsed) != nil {
		return ""
	}
	for _, k := range []string{"eid_token", "eidToken", "eid"} {
		if v, ok := parsed[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func yybJdRefreshCK(openid string) (string, string, error) {
	if !IsYybAccountAlive(openid) {
		if st, err := RefreshYybAccountLiveness(openid); err == nil {
			st = strings.ToLower(strings.TrimSpace(st))
			if st != "alive" && st != "online" {
				return "", "", fmt.Errorf("应用宝账号不可用或已掉线")
			}
		} else {
			return "", "", fmt.Errorf("应用宝账号不可用或已掉线")
		}
	}
	code, err := yybJdGetCode(openid)
	if err != nil {
		return "", "", fmt.Errorf("获取应用宝 code 失败: %v", err)
	}
	eidToken := yybJdGetEidToken(openid)
	if eidToken == "" {
		models.Yyb().Infof("应用宝 eid_token 为空，尝试 finger_tk")
		tk, err := models.WxJdGetFingerTk()
		if err == nil {
			eidToken = tk
		}
	}
	ptKey, ptPin, err := models.WxJdSilentAuthLogin(code, eidToken)
	if err != nil {
		return "", "", err
	}
	return ptKey, ptPin, nil
}

func saveYybJdCookie(userNumber int, openid, ptKey, ptPin string) (string, error) {
	newCK := &models.JdCookie{PtKey: ptKey, PtPin: url.QueryEscape(ptPin)}
	if !models.CookieOK(newCK) {
		return "", fmt.Errorf("刷新成功但 CK 验证无效")
	}
	nick := ptPin
	encodedPin := url.QueryEscape(ptPin)
	if existingCK, e := models.GetJdCookie(encodedPin); e == nil {
		existingCK.Updates(models.JdCookie{
			PtKey: ptKey, Available: models.True, YybOpenID: openid,
			WeiXin: openid, UpdateAt: models.Date(),
		})
		if existingCK.Nickname != "" {
			nick = existingCK.Nickname
		}
	} else {
		newCK.YybOpenID = openid
		newCK.WeiXin = openid
		newCK.QQ = userNumber
		newCK.Available = models.True
		models.NewJdCookie(newCK)
	}
	go func() { models.Save <- &models.JdCookie{} }()
	return nick, nil
}

// GetPortalJdYybAccounts 获取可用于京东登录的应用宝账号
func GetPortalJdYybAccounts(userNumber int) ([]PortalJdYybAccount, error) {
	accounts, err := findUserYybAccounts(userNumber)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("未检测到可用的应用宝账号，请先在「应用宝协议」页面扫码绑定")
	}
	result := make([]PortalJdYybAccount, 0, len(accounts))
	for i, acc := range accounts {
		nick := acc.Nickname
		if nick == "" {
			nick = acc.OpenID
		}
		result = append(result, PortalJdYybAccount{
			Index:    i + 1,
			OpenID:   acc.OpenID,
			Nickname: nick,
			Status:   acc.Status,
		})
	}
	return result, nil
}

// PortalJdYybRefresh 通过应用宝协议刷新京东 CK
func PortalJdYybRefresh(userNumber int, openid string, riskConfirmed bool) (*PortalJdYybRefreshResult, error) {
	accounts, err := findUserYybAccounts(userNumber)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("未检测到可用的应用宝账号")
	}

	targets := accounts
	if openid != "" && openid != "all" {
		found := false
		for _, a := range accounts {
			if strings.EqualFold(a.OpenID, openid) {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("账号不可用或不存在")
		}
		targets = nil
		for _, a := range accounts {
			if strings.EqualFold(a.OpenID, openid) {
				targets = append(targets, a)
			}
		}
	}

	result := &PortalJdYybRefreshResult{Details: []string{}}
	for i, acc := range targets {
		if i > 0 {
			time.Sleep(time.Duration(rand.Intn(2000)+3000) * time.Millisecond)
		}
		detail, needRisk, riskURL, riskMsg, ok := portalYybJdRefreshOne(userNumber, acc.OpenID, riskConfirmed)
		result.Details = append(result.Details, detail)
		if needRisk {
			result.NeedRiskVerify = true
			result.RiskURL = riskURL
			result.RiskMsg = riskMsg
			if _, exists := portalYybJdRisk[userNumber]; !exists {
				portalYybJdRisk[userNumber] = portalYybRiskInfo{openid: acc.OpenID, riskUrl: riskURL, riskMsg: riskMsg}
			}
			return result, nil
		}
		if ok {
			result.Success++
		} else {
			result.Fail++
		}
	}
	delete(portalYybJdRisk, userNumber)
	return result, nil
}

func portalYybJdRefreshOne(userNumber int, openid string, riskConfirmed bool) (detail string, needRisk bool, riskURL, riskMsg string, ok bool) {
	if pending, exists := portalYybJdRisk[userNumber]; exists && pending.openid == openid && !riskConfirmed {
		return "请先完成短信验证后再继续", true, pending.riskUrl, pending.riskMsg, false
	}

	ptKey, ptPin, err := yybJdRefreshCK(openid)
	if err != nil {
		var riskErr *models.RiskVerifyError
		if errors.As(err, &riskErr) {
			portalYybJdRisk[userNumber] = portalYybRiskInfo{openid: openid, riskUrl: riskErr.JmpURL, riskMsg: riskErr.ErrMsg}
			return "账号需要短信验证", true, riskErr.JmpURL, riskErr.ErrMsg, false
		}
		return fmt.Sprintf("❌ %s 刷新失败: %v", openid, err), false, "", "", false
	}
	nick, err := saveYybJdCookie(userNumber, openid, ptKey, ptPin)
	if err != nil {
		return fmt.Sprintf("❌ %s %v", openid, err), false, "", "", false
	}
	delete(portalYybJdRisk, userNumber)
	(&models.JdCookie{}).Push(fmt.Sprintf("应用宝刷新京东成功: %s (OpenID: %s)", nick, openid))
	return fmt.Sprintf("✅ %s CK 刷新成功", nick), false, "", "", true
}

// PortalJdYybContinueAfterRisk 风控验证完成后继续刷新
func PortalJdYybContinueAfterRisk(userNumber int) (*PortalJdYybRefreshResult, error) {
	info, ok := portalYybJdRisk[userNumber]
	if !ok || info.openid == "" {
		return nil, fmt.Errorf("没有待继续的验证任务")
	}
	return PortalJdYybRefresh(userNumber, info.openid, true)
}

// RefreshYybCKAuto 应用宝京东 CK 自动刷新（每 3 小时，不通知用户）
func RefreshYybCKAuto() {
	if !Ready() {
		return
	}
	models.Yyb().Infof("开始应用宝京东 CK 自动刷新")
	(&models.JdCookie{}).Push("开始应用宝京东 CK 自动刷新")
	cks := models.GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where(fmt.Sprintf("%s >= ? and %s = ? and %s != ''", models.Priority, models.Available, "YybOpenID"), 0, models.True)
	})

	refreshOK := 0
	refreshFail := 0
	for _, ck := range cks {
		if ck.Available != models.True || models.CookieOK(&ck) || ck.YybOpenID == "" {
			continue
		}
		if !IsYybAccountAlive(ck.YybOpenID) {
			continue
		}
		ptKey, ptPin, err := yybJdRefreshCK(ck.YybOpenID)
		if err == nil && ptKey != "" && ptPin != "" {
			encodedPin := url.QueryEscape(ptPin)
			newCK := &models.JdCookie{PtKey: ptKey, PtPin: encodedPin}
			if models.CookieOK(newCK) {
				ck.Updates(models.JdCookie{PtKey: ptKey, PtPin: encodedPin, Available: models.True, UpdateAt: models.Date()})
				refreshOK++
				(&models.JdCookie{}).Push(fmt.Sprintf("应用宝自动刷新京东成功: %s (OpenID: %s)", ck.PtPin, ck.YybOpenID))
			} else {
				refreshFail++
				ck.Updates(models.JdCookie{Available: models.False})
			}
		} else {
			refreshFail++
			ck.Updates(models.JdCookie{Available: models.False})
		}
	}
	if refreshOK > 0 || refreshFail > 0 {
		msg := fmt.Sprintf("应用宝京东 CK 自动刷新完成：成功%d，失败%d", refreshOK, refreshFail)
		models.Yyb().Infof(msg)
		(&models.JdCookie{}).Push(msg)
	}
	go func() { models.Save <- &models.JdCookie{} }()
}

// CheckYybOfflineAndNotify 每日检测应用宝账号掉线并通知用户
func CheckYybOfflineAndNotify() {
	if n := models.CleanupStaleOfflineNotifications(); n > 0 {
		models.Yyb().Infof("已自动清理 %d 条超过 %d 天的微信/应用宝掉线提醒通知", n, models.OfflineNotifyRetentionDays)
	}
	CheckYybOfflineAndNotifyWithChannels(models.NotifyChannels{Web: true, App: true, Robot: false}, false)
}

// CheckYybOfflineAndNotifyWithChannels 带渠道的应用宝掉线检测
func CheckYybOfflineAndNotifyWithChannels(channels models.NotifyChannels, force bool) {
	if !Ready() {
		models.Yyb().Infof("应用宝掉线检测：服务不可用，跳过")
		return
	}
	models.Yyb().Infof("开始执行应用宝掉线检测推送...")
	bindings, err := ListAllBindings()
	if err != nil || len(bindings) == 0 {
		models.Yyb().Infof("应用宝掉线检测：暂无绑定账号，跳过")
		return
	}

	offlineCount := 0
	notifiedCount := 0
	seenUser := make(map[int]bool)

	for _, b := range bindings {
		openid := strings.TrimSpace(b.OpenID)
		if openid == "" {
			continue
		}
		alive := IsYybAccountAlive(openid)
		if alive {
			if _, loaded := offlineNotifiedYybOIDs.LoadAndDelete(openid); loaded {
				models.Yyb().Infof("应用宝掉线检测：账号 %s 已恢复可用，清除通知记录", openid)
			}
			continue
		}
		offlineCount++
		if !force {
			if _, loaded := offlineNotifiedYybOIDs.LoadOrStore(openid, true); loaded {
				continue
			}
		} else {
			offlineNotifiedYybOIDs.Store(openid, true)
		}

		nick := b.Nickname
		if nick == "" {
			nick = openid
		}
		notifyMsg := fmt.Sprintf(
			"⚠️ 你的应用宝协议账号已掉线，将影响京东 CK 自动续期。\n\n"+
				"📋 账号信息：\n"+
				"👤 %s (🔴 不可用)\n"+
				"🆔 %s\n\n"+
				"💡 请前往用户中心 → 应用宝协议，重新扫码登录。\n"+
				"📌 本消息只发送一次，账号恢复后如再次掉线将重新通知。\n"+
				"📌 网页/App 通知「%s」将在 %d 天后自动删除。",
			nick, openid, models.NotifyTitleYybOffline, models.OfflineNotifyRetentionDays,
		)
		if !seenUser[b.UserNumber] {
			_ = models.ReplaceUserOfflineNotification(models.NotifyTitleYybOffline, notifyMsg, models.NotifyCategoryWx, models.NotifySourceYyb, b.UserNumber, channels)
			seenUser[b.UserNumber] = true
			notifiedCount++
		}
		if offlineCount >= 2 {
			time.Sleep(time.Duration(3+rand.Intn(3)) * time.Second)
		}
	}
	models.Yyb().Infof("应用宝掉线检测完成，共 %d 个绑定，%d 个掉线，通知 %d 个用户", len(bindings), offlineCount, notifiedCount)
}

func truncateMap(v map[string]any, maxLen int) string {
	b, _ := json.Marshal(v)
	s := string(b)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
