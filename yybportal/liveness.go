package yybportal

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cdle/xdd/models"
)

var (
	yybLivenessLastRunDate string
	yybLivenessMu          sync.Mutex
)

func yybLivenessSettings() models.YybConfig {
	cfg := models.Config.Yyb
	models.NormalizeYybConfig(&cfg)
	return cfg
}

func parseYybLivenessCheckTime(raw string) (hour, minute int, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 9, 0, true
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

// runYybLivenessScheduleTick 每分钟检查是否到达配置的每日检测时刻
func runYybLivenessScheduleTick() {
	cfg := yybLivenessSettings()
	if !models.YybLivenessCheckEnabled(cfg) {
		return
	}
	hour, minute, ok := parseYybLivenessCheckTime(cfg.LivenessCheckTime)
	if !ok {
		models.Yyb().Warnf("应用宝每日存活检测时间格式无效：%s", cfg.LivenessCheckTime)
		return
	}
	now := time.Now()
	if now.Hour() != hour || now.Minute() != minute {
		return
	}
	today := now.Format("2006-01-02")
	yybLivenessMu.Lock()
	if yybLivenessLastRunDate == today {
		yybLivenessMu.Unlock()
		return
	}
	yybLivenessLastRunDate = today
	yybLivenessMu.Unlock()
	go RunYybDailyLivenessCheck(false)
}

func yybRefreshCooldownKey(scope, ref string) string {
	return fmt.Sprintf("yyb_refresh_cd:%s:%s", scope, strings.TrimSpace(ref))
}

func yybManualRefreshCooldownMin() int {
	mins := yybLivenessSettings().LivenessManualCooldownMin
	if mins <= 0 {
		return 10
	}
	return mins
}

// checkYybManualRefreshAllowed 仅检查冷却，不写入
func checkYybManualRefreshAllowed(scope, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("缺少账号 ref")
	}
	key := yybRefreshCooldownKey(scope, ref)
	if strings.TrimSpace(models.GetCache(key)) != "" {
		return fmt.Errorf("刷新太频繁，请 %d 分钟后再试", yybManualRefreshCooldownMin())
	}
	return nil
}

// ensureYybManualRefreshAllowed 手动刷新存活冷却（门户/管理端）
func ensureYybManualRefreshAllowed(scope, ref string) error {
	if err := checkYybManualRefreshAllowed(scope, ref); err != nil {
		return err
	}
	markYybManualRefreshUsed(scope, ref)
	return nil
}

func markYybManualRefreshUsed(scope, ref string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return
	}
	key := yybRefreshCooldownKey(scope, ref)
	models.SaveCacheTTL(key, "1", yybManualRefreshCooldownMin()*60)
}

func isYybRefreshCooldownError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "刷新太频繁")
}

func isYybProxyOrNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"socks5", "05020001", "proxy", "proxyconnect",
		"i/o timeout", "timeout", "deadline exceeded",
		"connection refused", "connection reset", "connection timed out",
		"no such host", "tls handshake timeout", "eof",
		"51代理", "未提取到代理", "dial tcp",
		"http 500", "http 502", "http 503", "http 504",
		"代理/网络异常",
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
}

func yybAccountRecentlyChecked(openid string) bool {
	cfg := yybLivenessSettings()
	hours := cfg.LivenessSkipIfCheckedWithinHours
	if hours <= 0 {
		return false
	}
	acc, err := AccountPublic(openid)
	if err != nil || acc == nil || acc.LastCheckedAt == nil || *acc.LastCheckedAt <= 0 {
		return false
	}
	return time.Since(time.Unix(*acc.LastCheckedAt, 0)) < time.Duration(hours)*time.Hour
}

// RunYybDailyLivenessCheck 每日定时：真 refresh 存活 + 掉线推送
func RunYybDailyLivenessCheck(force bool) {
	RunYybDailyLivenessCheckWithChannels(models.DefaultProtocolOfflineNotifyChannels(), nil, force)
}

// RunYybDailyLivenessCheckWithChannels 带推送渠道的每日存活检测；openIDs 非空时仅处理指定账号
func RunYybDailyLivenessCheckWithChannels(channels models.NotifyChannels, openIDs []string, force bool) {
	channels = models.ProtocolOfflineNotifyChannels(channels)
	if !Ready() {
		models.Yyb().Infof("应用宝每日存活检测：服务不可用，跳过")
		return
	}
	cfg := yybLivenessSettings()
	if !force && !models.YybLivenessCheckEnabled(cfg) {
		return
	}
	models.Yyb().Infof("开始应用宝每日存活检测（真 refresh）...")
	bindings, err := ListAllBindings()
	if err != nil || len(bindings) == 0 {
		models.Yyb().Infof("应用宝每日存活检测：暂无绑定账号，跳过")
		return
	}

	interval := time.Duration(cfg.LivenessCheckIntervalSec) * time.Second
	if interval <= 0 {
		interval = 3 * time.Second
	}
	probed, skipped, proxyFail := 0, 0, 0
	offlineOpenIDs := make([]string, 0)

	for i, b := range bindings {
		openid := strings.TrimSpace(b.OpenID)
		if openid == "" {
			continue
		}
		if len(openIDs) > 0 && !yybOpenIDInList(openid, openIDs) {
			continue
		}
		if !force && yybAccountRecentlyChecked(openid) {
			skipped++
			continue
		}
		status, err := RefreshYybAccountLiveness(openid)
		probed++
		if err != nil {
			if isYybProxyOrNetworkError(err) {
				proxyFail++
				models.Yyb().Warnf("应用宝每日检测代理/网络失败 openid=%s: %v", openid, err)
			} else {
				offlineOpenIDs = append(offlineOpenIDs, openid)
			}
		} else if status != "alive" && status != "online" {
			offlineOpenIDs = append(offlineOpenIDs, openid)
		}
		if i < len(bindings)-1 {
			time.Sleep(interval)
			if probed >= 2 && interval < 2*time.Second {
				time.Sleep(time.Duration(1+rand.Intn(2)) * time.Second)
			}
		}
	}

	models.Yyb().Infof("应用宝每日存活检测完成：绑定 %d，探测 %d，跳过 %d，代理失败 %d，掉线 %d",
		len(bindings), probed, skipped, proxyFail, len(offlineOpenIDs))

	if n := models.CleanupStaleOfflineNotifications(); n > 0 {
		models.Yyb().Infof("已自动清理 %d 条超过 %d 天的应用宝掉线提醒通知", n, models.OfflineNotifyRetentionDays)
	}
	notifyYybOfflineBindings(bindings, offlineOpenIDs, channels, force)
}

func notifyYybOfflineBindings(bindings []PortalYybBinding, offlineOpenIDs []string, channels models.NotifyChannels, force bool) {
	if len(offlineOpenIDs) == 0 {
		return
	}
	offlineSet := map[string]struct{}{}
	for _, openid := range offlineOpenIDs {
		offlineSet[openid] = struct{}{}
	}
	notifiedCount := 0
	for _, b := range bindings {
		openid := strings.TrimSpace(b.OpenID)
		if openid == "" {
			continue
		}
		if _, offline := offlineSet[openid]; !offline {
			if _, loaded := offlineNotifiedYybOIDs.LoadAndDelete(openid); loaded {
				models.Yyb().Infof("应用宝掉线检测：账号 %s 已恢复可用，清除通知记录", openid)
			}
			continue
		}
		if !force {
			if _, loaded := offlineNotifiedYybOIDs.LoadOrStore(openid, true); loaded {
				continue
			}
		} else {
			offlineNotifiedYybOIDs.Store(openid, true)
		}
		protoBind, _ := models.FindProtocolBindingByOpenID(openid)
		if protoBind != nil && models.WxOfflineAlreadyNotified(protoBind.WxWxid) {
			models.Yyb().Infof("应用宝掉线检测：openid=%s 对应微信 %s 已推送掉线，跳过重复通知", openid, protoBind.WxWxid)
			continue
		}
		nick := b.Nickname
		if nick == "" {
			nick = openid
		}
		notifyMsg := fmt.Sprintf(
			"⚠️ 应用宝协议账号已掉线，将影响协议项目获取 CK。\n\n"+
				"📋 账号信息\n"+
				"👤 %s（不可用）\n"+
				"🆔 %s\n\n"+
				"💡 请前往 用户中心 → 应用宝协议，重新扫码登录。\n"+
				"%s",
			nick, openid, models.ProtocolOfflineNotifyFooter(),
		)
		if b.UserNumber > 0 {
			models.PushProtocolOfflineNotification(models.NotifyTitleYybOffline, notifyMsg, models.NotifyCategoryWx, models.NotifySourceYyb, b.UserNumber, channels)
			if protoBind != nil {
				models.MarkWxOfflineNotified(protoBind.WxWxid)
			}
			notifiedCount++
			if notifiedCount >= 2 {
				time.Sleep(time.Duration(3+rand.Intn(3)) * time.Second)
			}
		}
	}
	models.Yyb().Infof("应用宝掉线推送完成，通知 %d 个账号", notifiedCount)
}

func yybOpenIDInList(openid string, refs []string) bool {
	openid = strings.TrimSpace(openid)
	for _, ref := range refs {
		if strings.EqualFold(strings.TrimSpace(ref), openid) {
			return true
		}
	}
	return false
}
