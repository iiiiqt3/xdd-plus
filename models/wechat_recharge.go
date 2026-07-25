package models

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"gorm.io/gorm"
)

const (
	wechatRechargeConfigEnv   = "wechat_recharge_config"
	wechatTallybookAppID      = "wx7c86e0c731b9b8ef"
	wechatRechargeGrace       = 15 * time.Second
	wechatRechargeDir         = "conf/wechat_recharge"
)

var (
	ErrWechatRechargeBusy       = errors.New("当前充值人数已满，请稍后再试")
	ErrWechatRechargeDailyLimit = errors.New("今日微信充值次数已达上限")
	errWechatRechargeSessionExpired = errors.New("微信账单会话失效")

	wechatRechargeOnce     sync.Once
	wechatRechargeCreateMu sync.Mutex
	wechatRechargeCfgMu    sync.RWMutex
	wechatRechargeSessionCache struct {
		sync.RWMutex
		AccountKey string
		SessionKey string
	}
)

// WechatRechargeConfig 微信赞赏码充值后台配置（存 env 表 JSON）
type WechatRechargeConfig struct {
	Enabled             bool   `json:"enabled"`
	PortalEnabled       bool   `json:"portal_enabled"`
	BotEnabled          bool   `json:"bot_enabled"`
	BillAccountType     string `json:"bill_account_type"` // yyb | wx
	BillAccountRef      string `json:"bill_account_ref"`
	QRCodeMode          string `json:"qrcode_mode"` // url | upload
	QRCodeURL           string `json:"qrcode_url"`
	QRCodeFile          string `json:"qrcode_file"`
	FallbackURL         string `json:"fallback_url"`
	ExternalPurchaseURL string `json:"external_purchase_url"`
	TiersYuan           []int  `json:"tiers_yuan"`
	PointsPerYuan       int    `json:"points_per_yuan"`
	MaxConcurrent       int    `json:"max_concurrent"`
	DailyLimit          int    `json:"daily_limit"`
	OrderTimeoutMinutes int    `json:"order_timeout_minutes"`
	RandomFenMin        int    `json:"random_fen_min"`
	RandomFenMax        int    `json:"random_fen_max"`
	PollIntervalSec     int    `json:"poll_interval_sec"`
}

// WechatRechargeOrder 微信赞赏充值订单
type WechatRechargeOrder struct {
	ID           uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderNo      string     `gorm:"type:varchar(64);uniqueIndex" json:"order_no"`
	QQ           int        `gorm:"index;index:idx_wxrc_daily,priority:1" json:"qq"`
	Channel      string     `gorm:"type:varchar(16)" json:"channel"`
	RequestedFen int        `json:"requested_fen"`
	RandomFen    int        `json:"random_fen"`
	PaidFen      int        `json:"paid_fen"`
	Points       int        `json:"points"`
	Status       string     `gorm:"type:varchar(24);index" json:"status"`
	TransID      string     `gorm:"type:varchar(128);index" json:"trans_id,omitempty"`
	SessionKey   string     `gorm:"type:text" json:"-"`
	LastError    string     `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt    time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ExpiresAt    time.Time  `gorm:"index" json:"expires_at"`
	ActivatedAt  *time.Time `gorm:"index:idx_wxrc_daily,priority:2" json:"-"`
	PaidAt       *time.Time `json:"paid_at,omitempty"`
	LastPolledAt *time.Time `json:"last_polled_at,omitempty"`
}

// WechatRechargeReceipt 微信账单号防重
type WechatRechargeReceipt struct {
	TransID   string    `gorm:"type:varchar(128);primaryKey"`
	OrderNo   string    `gorm:"type:varchar(64);index"`
	CreatedAt time.Time `gorm:"index"`
}

// WechatRechargePublicOrder 返回给门户/App 的脱敏订单
type WechatRechargePublicOrder struct {
	OrderNo      string `json:"order_no"`
	RequestedFen int    `json:"requested_fen"`
	PaymentFen   int    `json:"payment_fen"`
	PaidFen      int    `json:"paid_fen"`
	Points       int    `json:"points"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	ExpiresAt    string `json:"expires_at"`
	RemainingSec int64  `json:"remaining_sec"`
	Coin         int    `json:"coin,omitempty"`
	QRCodeURL    string `json:"qrcode_url,omitempty"`
}

// WechatRechargeIncomeSummary 收益汇总
type WechatRechargeIncomeSummary struct {
	Today     float64 `json:"today"`
	SevenDay  float64 `json:"seven_day"`
	LastMonth float64 `json:"last_month"`
	Month     float64 `json:"month"`
	Total     float64 `json:"total"`
	PaidCount int64   `json:"paid_count"`
}

type wechatTallyResponse struct {
	RetCode          int             `json:"ret_code"`
	RetMsg           string          `json:"ret_msg"`
	CustomSessionKey string          `json:"custom_session_key"`
	RecordsInfo      json.RawMessage `json:"records_info"`
}

type wechatBillRecord struct {
	ID       string
	TransID  string
	BillTime int64
	BillType int
	Balance  int
	Remark   string
}

func defaultWechatRechargeConfig() WechatRechargeConfig {
	return WechatRechargeConfig{
		Enabled:             false,
		PortalEnabled:       true,
		BotEnabled:          true,
		BillAccountType:     "yyb",
		QRCodeMode:          "url",
		QRCodeFile:          "qrcode.jpg",
		FallbackURL:         "http://180.152.5.230:8005/",
		ExternalPurchaseURL: "http://180.152.5.230:8005/#/",
		TiersYuan:           []int{5, 10, 20, 30},
		PointsPerYuan:       100,
		MaxConcurrent:       5,
		DailyLimit:          10,
		OrderTimeoutMinutes: 3,
		RandomFenMin:        1,
		RandomFenMax:        5,
		PollIntervalSec:     5,
	}
}

func normalizeWechatRechargeConfig(cfg WechatRechargeConfig) WechatRechargeConfig {
	def := defaultWechatRechargeConfig()
	if cfg.PointsPerYuan <= 0 {
		cfg.PointsPerYuan = def.PointsPerYuan
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = def.MaxConcurrent
	}
	if cfg.DailyLimit <= 0 {
		cfg.DailyLimit = def.DailyLimit
	}
	if cfg.OrderTimeoutMinutes <= 0 {
		cfg.OrderTimeoutMinutes = def.OrderTimeoutMinutes
	}
	if cfg.RandomFenMin <= 0 {
		cfg.RandomFenMin = def.RandomFenMin
	}
	if cfg.RandomFenMax < cfg.RandomFenMin {
		cfg.RandomFenMax = def.RandomFenMax
	}
	if cfg.PollIntervalSec <= 0 {
		cfg.PollIntervalSec = def.PollIntervalSec
	}
	if len(cfg.TiersYuan) == 0 {
		cfg.TiersYuan = def.TiersYuan
	}
	cfg.BillAccountType = strings.ToLower(strings.TrimSpace(cfg.BillAccountType))
	if cfg.BillAccountType != "wx" {
		cfg.BillAccountType = "yyb"
	}
	cfg.QRCodeMode = strings.ToLower(strings.TrimSpace(cfg.QRCodeMode))
	if cfg.QRCodeMode != "upload" {
		cfg.QRCodeMode = "url"
	}
	if strings.TrimSpace(cfg.FallbackURL) == "" {
		cfg.FallbackURL = def.FallbackURL
	}
	if strings.TrimSpace(cfg.ExternalPurchaseURL) == "" {
		cfg.ExternalPurchaseURL = def.ExternalPurchaseURL
	}
	if strings.TrimSpace(cfg.QRCodeFile) == "" {
		cfg.QRCodeFile = def.QRCodeFile
	}
	seen := make(map[int]struct{})
	tiers := make([]int, 0, len(cfg.TiersYuan))
	for _, y := range cfg.TiersYuan {
		if y <= 0 || y > 10000 {
			continue
		}
		if _, ok := seen[y]; ok {
			continue
		}
		seen[y] = struct{}{}
		tiers = append(tiers, y)
	}
	if len(tiers) == 0 {
		tiers = def.TiersYuan
	}
	cfg.TiersYuan = tiers
	return cfg
}

func GetWechatRechargeConfig() WechatRechargeConfig {
	wechatRechargeCfgMu.RLock()
	defer wechatRechargeCfgMu.RUnlock()
	raw := strings.TrimSpace(GetEnv(wechatRechargeConfigEnv))
	if raw == "" {
		return defaultWechatRechargeConfig()
	}
	var cfg WechatRechargeConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		Warn("[微信充值] 配置解析失败，使用默认值: %v", err)
		return defaultWechatRechargeConfig()
	}
	return normalizeWechatRechargeConfig(cfg)
}

func SaveWechatRechargeConfig(cfg WechatRechargeConfig) error {
	prev := GetWechatRechargeConfig()
	if strings.TrimSpace(cfg.QRCodeFile) == "" {
		cfg.QRCodeFile = prev.QRCodeFile
	}
	cfg = normalizeWechatRechargeConfig(cfg)
	if cfg.Enabled {
		if strings.TrimSpace(cfg.BillAccountRef) == "" {
			return errors.New("请配置查账账号")
		}
		if cfg.QRCodeMode == "url" && strings.TrimSpace(cfg.QRCodeURL) == "" {
			return errors.New("请配置收款二维码链接")
		}
		if cfg.QRCodeMode == "upload" {
			if _, err := os.Stat(WechatRechargeQRCodeFilePath(cfg)); err != nil {
				return errors.New("请先上传收款二维码图片")
			}
		}
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	ExportEnv(&Env{Name: wechatRechargeConfigEnv, Value: string(raw), Note: "微信赞赏码充值配置"})
	wechatRechargeCfgMu.Lock()
	wechatRechargeCfgMu.Unlock()
	return nil
}

func GetWechatRechargePortalTiers() []map[string]interface{} {
	cfg := GetWechatRechargeConfig()
	out := make([]map[string]interface{}, 0, len(cfg.TiersYuan))
	for _, y := range cfg.TiersYuan {
		fen := y * 100
		out = append(out, map[string]interface{}{
			"yuan":   y,
			"fen":    fen,
			"points": wechatRechargePointsForFen(fen, cfg),
		})
	}
	return out
}

func wechatRechargePointsForFen(fen int, cfg WechatRechargeConfig) int {
	if fen <= 0 {
		return 0
	}
	return fen * cfg.PointsPerYuan / 100
}

func wechatRechargeTTL() time.Duration {
	m := GetWechatRechargeConfig().OrderTimeoutMinutes
	if m <= 0 {
		m = 3
	}
	return time.Duration(m) * time.Minute
}

func wechatRechargePollInterval() time.Duration {
	s := GetWechatRechargeConfig().PollIntervalSec
	if s <= 0 {
		s = 5
	}
	return time.Duration(s) * time.Second
}

func WechatRechargeQRCodeDir() string {
	return filepath.Join(ExecPath, wechatRechargeDir)
}

func WechatRechargeQRCodeFilePath(cfg WechatRechargeConfig) string {
	name := filepath.Base(strings.TrimSpace(cfg.QRCodeFile))
	if name == "" || name == "." {
		name = "qrcode.jpg"
	}
	return filepath.Join(WechatRechargeQRCodeDir(), name)
}

func WechatRechargePublicQRCodeURL() string {
	cfg := GetWechatRechargeConfig()
	if cfg.QRCodeMode == "upload" {
		return ""
	}
	return strings.TrimSpace(cfg.QRCodeURL)
}

func WechatRechargeFallbackURL() string {
	return strings.TrimSpace(GetWechatRechargeConfig().FallbackURL)
}

func WechatRechargeExternalPurchaseURL() string {
	return strings.TrimSpace(GetWechatRechargeConfig().ExternalPurchaseURL)
}

// WechatRechargeBillAccountStatus 检测查账账号是否在线（应用宝 / 微信协议）
func WechatRechargeBillAccountStatus(cfg WechatRechargeConfig) (online bool, reason string) {
	ref := strings.TrimSpace(cfg.BillAccountRef)
	if ref == "" {
		return false, "后台尚未配置查账账号"
	}
	if cfg.BillAccountType == "wx" {
		ok, err := checkWxDeviceOnline(ref)
		if err != nil {
			return false, "微信协议状态检测失败，请稍后重试"
		}
		if !ok {
			return false, "微信协议查账账号当前离线，暂无法自动到账"
		}
		return true, ""
	}
	route := ResolveProtocolRoute(ref)
	openid := strings.TrimSpace(route.OpenID)
	if openid == "" {
		openid = ref
	}
	if protocolYybAccountExistsFn != nil && !protocolYybAccountExistsFn(openid) {
		return false, "应用宝查账账号不存在，请检查后台配置"
	}
	if ProtocolYybAccountAlive(openid) {
		return true, ""
	}
	return false, "应用宝查账账号当前离线，暂无法自动到账"
}

// WechatRechargePortalPublicConfig 门户/App 统一充值配置视图
func WechatRechargePortalPublicConfig() map[string]interface{} {
	cfg := GetWechatRechargeConfig()
	enabled := cfg.Enabled && cfg.PortalEnabled
	online, offlineReason := WechatRechargeBillAccountStatus(cfg)
	qrReady := wechatRechargeEnsureQRReady(cfg) == nil
	canRecharge := enabled && online && qrReady
	blockReason := ""
	if !enabled {
		blockReason = "微信赞赏充值暂未开放"
	} else if !online {
		blockReason = offlineReason
	} else if !qrReady {
		blockReason = "收款码尚未配置完成"
	}
	return map[string]interface{}{
		"enabled":               enabled,
		"can_recharge":          canRecharge,
		"block_reason":          blockReason,
		"bill_account_online":   online,
		"bill_account_type":     cfg.BillAccountType,
		"bill_account_message":  offlineReason,
		"points_per_yuan":       cfg.PointsPerYuan,
		"tiers":                 GetWechatRechargePortalTiers(),
		"timeout_minutes":       cfg.OrderTimeoutMinutes,
		"fallback_url":          WechatRechargeFallbackURL(),
		"external_purchase_url": WechatRechargeExternalPurchaseURL(),
		"random_fen_min":        cfg.RandomFenMin,
		"random_fen_max":        cfg.RandomFenMax,
		"payment_notice": []string{
			"每笔订单会生成「精确应付金额」（比档位略低 0.01～0.05 元），用于区分同时充值的用户。",
			"请务必按弹窗显示的金额支付，例如档位 10 元可能需付 9.98 元，付 10 元整将无法自动到账。",
			"支付前请再次核对金额，建议复制金额后再去微信付款。",
		},
		"wrong_payment_help": []string{
			"若支付了错误金额（如应付 9.98 元却付了 10 元），系统无法自动匹配，积分不会到账。",
			"请勿对同一笔订单重复付款；订单超时后请重新发起充值，使用新订单的新金额。",
			"如已付错金额，请保存支付截图，并联系管理员处理（需提供订单号、QQ 编号、实付金额与时间）。",
			"也可改用「前往购买积分」外链或卡密兑换获取积分。",
		},
	}
}

// WechatRechargeAnalytics 后台图表统计数据
type WechatRechargeAnalytics struct {
	Summary   WechatRechargeIncomeSummary `json:"summary"`
	Daily     []WechatRechargeDailyStat   `json:"daily"`
	Monthly   []WechatRechargeMonthlyStat `json:"monthly"`
	ByStatus  []WechatRechargeGroupStat   `json:"by_status"`
	ByChannel []WechatRechargeGroupStat   `json:"by_channel"`
}

type WechatRechargeDailyStat struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type WechatRechargeMonthlyStat struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

type WechatRechargeGroupStat struct {
	Key    string  `json:"key"`
	Label  string  `json:"label"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"`
}

func GetWechatRechargeAnalytics(days int) WechatRechargeAnalytics {
	if db == nil {
		return WechatRechargeAnalytics{}
	}
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
	result := WechatRechargeAnalytics{Summary: GetWechatRechargeIncomeSummary()}

	type orderRow struct {
		Status    string
		Channel   string
		PaidFen   int
		PaidAt    *time.Time
		CreatedAt time.Time
	}
	var rows []orderRow
	_ = db.Model(&WechatRechargeOrder{}).
		Select("status", "channel", "paid_fen", "paid_at", "created_at").
		Where("created_at >= ?", start).
		Find(&rows).Error

	dailyMap := make(map[string]*WechatRechargeDailyStat)
	monthlyMap := make(map[string]*WechatRechargeMonthlyStat)
	statusMap := make(map[string]*WechatRechargeGroupStat)
	channelMap := make(map[string]*WechatRechargeGroupStat)

	for d := 0; d < days; d++ {
		day := start.AddDate(0, 0, d).Format("2006-01-02")
		dailyMap[day] = &WechatRechargeDailyStat{Date: day}
	}
	for i := 0; i < 12; i++ {
		m := now.AddDate(0, -i, 0).Format("2006-01")
		monthlyMap[m] = &WechatRechargeMonthlyStat{Month: m}
	}

	statusLabels := map[string]string{
		"paid": "已支付", "pending": "待支付", "expired": "已过期", "failed": "失败", "preparing": "准备中", "crediting": "入账中",
	}
	channelLabels := map[string]string{"portal": "网页", "bot": "机器人", "app": "App"}

	for _, row := range rows {
		st := strings.TrimSpace(row.Status)
		if st == "" {
			st = "unknown"
		}
		if statusMap[st] == nil {
			statusMap[st] = &WechatRechargeGroupStat{Key: st, Label: statusLabels[st]}
			if statusMap[st].Label == "" {
				statusMap[st].Label = st
			}
		}
		statusMap[st].Count++

		if st == "paid" && row.PaidFen > 0 && row.PaidAt != nil {
			amt := float64(row.PaidFen) / 100
			day := row.PaidAt.Local().Format("2006-01-02")
			month := row.PaidAt.Local().Format("2006-01")
			ch := strings.TrimSpace(row.Channel)
			if ch == "" {
				ch = "unknown"
			}
			if dailyMap[day] != nil {
				dailyMap[day].Amount += amt
				dailyMap[day].Count++
			}
			if monthlyMap[month] != nil {
				monthlyMap[month].Amount += amt
				monthlyMap[month].Count++
			}
			statusMap[st].Amount += amt
			if channelMap[ch] == nil {
				channelMap[ch] = &WechatRechargeGroupStat{Key: ch, Label: channelLabels[ch]}
				if channelMap[ch].Label == "" {
					channelMap[ch].Label = ch
				}
			}
			channelMap[ch].Count++
			channelMap[ch].Amount += amt
		}
	}

	result.Daily = make([]WechatRechargeDailyStat, 0, len(dailyMap))
	for d := 0; d < days; d++ {
		day := start.AddDate(0, 0, d).Format("2006-01-02")
		stat := dailyMap[day]
		stat.Amount = round2(stat.Amount)
		result.Daily = append(result.Daily, *stat)
	}
	result.Monthly = make([]WechatRechargeMonthlyStat, 0, 12)
	for i := 11; i >= 0; i-- {
		m := now.AddDate(0, -i, 0).Format("2006-01")
		stat := monthlyMap[m]
		stat.Amount = round2(stat.Amount)
		result.Monthly = append(result.Monthly, *stat)
	}
	for _, m := range statusMap {
		m.Amount = round2(m.Amount)
		result.ByStatus = append(result.ByStatus, *m)
	}
	for _, m := range channelMap {
		m.Amount = round2(m.Amount)
		result.ByChannel = append(result.ByChannel, *m)
	}
	return result
}

func wechatRechargeBillAccountKey(cfg WechatRechargeConfig) string {
	return cfg.BillAccountType + ":" + strings.TrimSpace(cfg.BillAccountRef)
}

func wechatRechargeTierAllowed(fen int) bool {
	cfg := GetWechatRechargeConfig()
	for _, y := range cfg.TiersYuan {
		if y*100 == fen {
			return true
		}
	}
	return false
}

func InitWechatRecharge() {
	_ = os.MkdirAll(WechatRechargeQRCodeDir(), 0755)
	StartWechatRechargeWorker()
}

func StartWechatRechargeWorker() {
	wechatRechargeOnce.Do(func() {
		go func() {
			processWechatRechargeOrders()
			ticker := time.NewTicker(wechatRechargePollInterval())
			defer ticker.Stop()
			for range ticker.C {
				processWechatRechargeOrders()
			}
		}()
	})
}

func HasActiveWechatRechargeOrder() bool {
	if db == nil {
		return false
	}
	cfg := GetWechatRechargeConfig()
	var count int64
	_ = db.Model(&WechatRechargeOrder{}).
		Where("status IN ? AND expires_at > ?", []string{"preparing", "pending", "crediting"}, time.Now().Add(-wechatRechargeGrace)).
		Count(&count).Error
	return int(count) >= cfg.MaxConcurrent
}

func GetWechatRechargeIncomeSummary() WechatRechargeIncomeSummary {
	if db == nil {
		return WechatRechargeIncomeSummary{}
	}
	type paidRow struct {
		PaidFen int
		PaidAt  *time.Time
	}
	var rows []paidRow
	if err := db.Model(&WechatRechargeOrder{}).Select("paid_fen", "paid_at").Where("status = ? AND paid_fen > 0", "paid").Find(&rows).Error; err != nil {
		return WechatRechargeIncomeSummary{}
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	sevenStart := todayStart.AddDate(0, 0, -6)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	nextMonthStart := monthStart.AddDate(0, 1, 0)
	lastMonthStart := monthStart.AddDate(0, -1, 0)
	result := WechatRechargeIncomeSummary{PaidCount: int64(len(rows))}
	for _, row := range rows {
		amount := float64(row.PaidFen) / 100
		result.Total += amount
		if row.PaidAt == nil {
			continue
		}
		paidAt := row.PaidAt.Local()
		if !paidAt.Before(todayStart) {
			result.Today += amount
		}
		if !paidAt.Before(sevenStart) {
			result.SevenDay += amount
		}
		if !paidAt.Before(monthStart) && paidAt.Before(nextMonthStart) {
			result.Month += amount
		}
		if !paidAt.Before(lastMonthStart) && paidAt.Before(monthStart) {
			result.LastMonth += amount
		}
	}
	result.Today = round2(result.Today)
	result.SevenDay = round2(result.SevenDay)
	result.LastMonth = round2(result.LastMonth)
	result.Month = round2(result.Month)
	result.Total = round2(result.Total)
	return result
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// CreateWechatRechargeOrder 创建充值订单（门户/机器人/App 共用）
func CreateWechatRechargeOrder(qq, requestedFen int, channel string) (WechatRechargeOrder, error) {
	cfg := GetWechatRechargeConfig()
	if !cfg.Enabled {
		return WechatRechargeOrder{}, errors.New("微信充值功能未开启")
	}
	if qq <= 0 {
		return WechatRechargeOrder{}, errors.New("用户信息无效")
	}
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = "portal"
	}
	if channel == "portal" && !cfg.PortalEnabled {
		return WechatRechargeOrder{}, errors.New("网页充值暂未开放")
	}
	if channel == "bot" && !cfg.BotEnabled {
		return WechatRechargeOrder{}, errors.New("机器人充值暂未开放")
	}
	if !wechatRechargeTierAllowed(requestedFen) {
		return WechatRechargeOrder{}, errors.New("充值档位无效")
	}
	if online, reason := WechatRechargeBillAccountStatus(cfg); !online {
		return WechatRechargeOrder{}, errors.New(reason)
	}
	if err := wechatRechargeEnsureQRReady(cfg); err != nil {
		return WechatRechargeOrder{}, err
	}

	now := time.Now()
	order := WechatRechargeOrder{
		OrderNo:      newWechatRechargeOrderNo(),
		QQ:           qq,
		Channel:      channel,
		RequestedFen: requestedFen,
		RandomFen:    newWechatRechargeRandomFen(cfg),
		Status:       "preparing",
		ExpiresAt:    now.Add(wechatRechargeTTL()),
	}

	wechatRechargeCreateMu.Lock()
	claimErr := db.Transaction(func(tx *gorm.DB) error {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		dayEnd := dayStart.AddDate(0, 0, 1)
		var dailyActivated int64
		if err := tx.Model(&WechatRechargeOrder{}).
			Where("qq = ? AND activated_at >= ? AND activated_at < ?", qq, dayStart, dayEnd).
			Count(&dailyActivated).Error; err != nil {
			return err
		}
		if dailyActivated >= int64(cfg.DailyLimit) {
			return ErrWechatRechargeDailyLimit
		}
		var active int64
		if err := tx.Model(&WechatRechargeOrder{}).
			Where("status IN ? AND expires_at > ?", []string{"preparing", "pending", "crediting"}, now.Add(-wechatRechargeGrace)).
			Count(&active).Error; err != nil {
			return err
		}
		if int(active) >= cfg.MaxConcurrent {
			return ErrWechatRechargeBusy
		}
		var user User
		if err := tx.Where("number = ?", qq).
			Attrs(User{Number: qq, Class: "qq", ActiveAt: now}).
			FirstOrCreate(&user).Error; err != nil {
			return err
		}
		return tx.Create(&order).Error
	})
	wechatRechargeCreateMu.Unlock()
	if claimErr != nil {
		return WechatRechargeOrder{}, claimErr
	}

	session, err := prepareWechatRechargeSession(cfg)
	if err != nil {
		_ = markWechatRechargeFailed(order.ID, err.Error())
		notifyWechatRechargeAdmin("微信充值预检失败", order, err.Error())
		return WechatRechargeOrder{}, fmt.Errorf("微信账单能力预检失败：%w", err)
	}
	order.SessionKey = session
	readyAt := time.Now()
	expiresAt := readyAt.Add(wechatRechargeTTL())
	result := db.Model(&WechatRechargeOrder{}).Where("id = ? AND status = ? AND expires_at > ?", order.ID, "preparing", readyAt).Updates(map[string]interface{}{
		"session_key":  session,
		"status":       "pending",
		"last_error":   "",
		"expires_at":   expiresAt,
		"activated_at": readyAt,
	})
	if result.Error != nil {
		return WechatRechargeOrder{}, result.Error
	}
	if result.RowsAffected != 1 {
		_ = markWechatRechargeFailed(order.ID, "微信充值预检超时")
		return WechatRechargeOrder{}, errors.New("微信充值订单已超时，请重新发起")
	}
	order.SessionKey = session
	order.Status = "pending"
	order.ExpiresAt = expiresAt
	order.ActivatedAt = &readyAt
	return order, nil
}

func markWechatRechargeFailed(id uint64, reason string) error {
	return db.Model(&WechatRechargeOrder{}).Where("id = ? AND status IN ?", id, []string{"preparing", "pending"}).Updates(map[string]interface{}{
		"status":      "failed",
		"session_key": "",
		"last_error":  truncateWechatRechargeError(reason),
	}).Error
}

func CancelWechatRechargeOrder(orderNo string, qq int, reason string) {
	_ = db.Model(&WechatRechargeOrder{}).
		Where("order_no = ? AND qq = ? AND status IN ?", strings.TrimSpace(orderNo), qq, []string{"preparing", "pending"}).
		Updates(map[string]interface{}{
			"status":       "failed",
			"session_key":  "",
			"last_error":   truncateWechatRechargeError(reason),
			"activated_at": nil,
		}).Error
}

func GetWechatRechargeOrderForUser(orderNo string, qq int) (WechatRechargePublicOrder, error) {
	var order WechatRechargeOrder
	if err := db.Where("order_no = ? AND qq = ?", strings.TrimSpace(orderNo), qq).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return WechatRechargePublicOrder{}, errors.New("充值订单不存在")
		}
		return WechatRechargePublicOrder{}, err
	}
	return publicWechatRechargeOrder(order), nil
}

func ListWechatRechargeOrdersForUser(qq, page, limit int) ([]WechatRechargePublicOrder, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := db.Model(&WechatRechargeOrder{}).Where("qq = ?", qq)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var orders []WechatRechargeOrder
	if err := q.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	out := make([]WechatRechargePublicOrder, 0, len(orders))
	for _, o := range orders {
		out = append(out, publicWechatRechargeOrder(o))
	}
	return out, total, nil
}

func ListWechatRechargeOrdersAdmin(qq, page, limit int, status string) ([]WechatRechargeOrder, int64, WechatRechargeIncomeSummary, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	q := db.Model(&WechatRechargeOrder{})
	if qq > 0 {
		q = q.Where("qq = ?", qq)
	}
	if s := strings.TrimSpace(status); s != "" {
		q = q.Where("status = ?", s)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, WechatRechargeIncomeSummary{}, err
	}
	var orders []WechatRechargeOrder
	if err := q.Order("id DESC").Offset((page-1)*limit).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, WechatRechargeIncomeSummary{}, err
	}
	return orders, total, GetWechatRechargeIncomeSummary(), nil
}

func CanReadWechatRechargeQRCode(orderNo string, qq int) bool {
	var count int64
	db.Model(&WechatRechargeOrder{}).
		Where("order_no = ? AND qq = ? AND status = ? AND expires_at > ?", strings.TrimSpace(orderNo), qq, "pending", time.Now()).
		Count(&count)
	return count == 1
}

func ReadWechatRechargeQRCodeBytes() ([]byte, string, error) {
	cfg := GetWechatRechargeConfig()
	if cfg.QRCodeMode == "url" {
		url := strings.TrimSpace(cfg.QRCodeURL)
		if url == "" {
			return nil, "", errors.New("未配置收款码链接")
		}
		client := &http.Client{Timeout: 20 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, "", fmt.Errorf("下载收款码失败 HTTP %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if err != nil {
			return nil, "", err
		}
		ct := resp.Header.Get("Content-Type")
		if ct == "" {
			ct = "image/jpeg"
		}
		return data, ct, nil
	}
	path := WechatRechargeQRCodeFilePath(cfg)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	ct := "image/jpeg"
	switch ext {
	case ".png":
		ct = "image/png"
	case ".webp":
		ct = "image/webp"
	case ".gif":
		ct = "image/gif"
	}
	return data, ct, nil
}

func SaveWechatRechargeQRCodeUpload(filename string, data []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		return "", errors.New("仅支持 jpg/png/webp/gif")
	}
	if len(data) == 0 || len(data) > 5<<20 {
		return "", errors.New("图片大小需在 5MB 以内")
	}
	_ = os.MkdirAll(WechatRechargeQRCodeDir(), 0755)
	name := "qrcode" + ext
	path := filepath.Join(WechatRechargeQRCodeDir(), name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	cfg := GetWechatRechargeConfig()
	cfg.QRCodeMode = "upload"
	cfg.QRCodeFile = name
	if err := SaveWechatRechargeConfig(cfg); err != nil {
		return "", err
	}
	return name, nil
}

func GetWechatRechargePublicOrder(order WechatRechargeOrder) WechatRechargePublicOrder {
	return publicWechatRechargeOrder(order)
}

func publicWechatRechargeOrder(order WechatRechargeOrder) WechatRechargePublicOrder {
	remaining := int64(time.Until(order.ExpiresAt).Seconds())
	if remaining < 0 {
		remaining = 0
	}
	pub := WechatRechargePublicOrder{
		OrderNo:      order.OrderNo,
		RequestedFen: order.RequestedFen,
		PaymentFen:   wechatRechargeTargetFen(order),
		PaidFen:      order.PaidFen,
		Points:       order.Points,
		Status:       order.Status,
		CreatedAt:    order.CreatedAt.Format("2006-01-02 15:04:05"),
		ExpiresAt:    order.ExpiresAt.Format("2006-01-02 15:04:05"),
		RemainingSec: remaining,
		Coin:         GetCoin(order.QQ),
	}
	if order.Status == "pending" {
		if url := WechatRechargePublicQRCodeURL(); url != "" {
			pub.QRCodeURL = url
		} else {
			pub.QRCodeURL = "/api/portal/wechat-recharge/qrcode/" + order.OrderNo
		}
	}
	if order.Status == "paid" && order.Points > 0 {
		pub.Points = order.Points
	} else if order.Status == "pending" {
		pub.Points = wechatRechargePointsForFen(order.RequestedFen, GetWechatRechargeConfig())
	}
	return pub
}

func wechatRechargeEnsureQRReady(cfg WechatRechargeConfig) error {
	if cfg.QRCodeMode == "url" {
		if strings.TrimSpace(cfg.QRCodeURL) == "" {
			return errors.New("后台尚未配置收款二维码链接")
		}
		return nil
	}
	info, err := os.Stat(WechatRechargeQRCodeFilePath(cfg))
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
		return errors.New("后台尚未上传收款二维码")
	}
	return nil
}

func newWechatRechargeOrderNo() string {
	random := make([]byte, 5)
	if _, err := rand.Read(random); err != nil {
		return fmt.Sprintf("WR%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("WR%d%s", time.Now().Unix(), hex.EncodeToString(random))
}

func newWechatRechargeRandomFen(cfg WechatRechargeConfig) int {
	minF, maxF := cfg.RandomFenMin, cfg.RandomFenMax
	if minF <= 0 {
		minF = 1
	}
	if maxF < minF {
		maxF = minF
	}
	span := maxF - minF + 1
	n, err := rand.Int(rand.Reader, big.NewInt(int64(span)))
	if err != nil {
		return -(minF + int(time.Now().UnixNano()%int64(span)))
	}
	return -(minF + int(n.Int64()))
}

func wechatRechargeTargetFen(order WechatRechargeOrder) int {
	if order.RequestedFen <= 0 {
		return 0
	}
	return order.RequestedFen + order.RandomFen
}

func wechatRechargePointsForPayment(order WechatRechargeOrder, paidFen int) (int, bool) {
	cfg := GetWechatRechargeConfig()
	baseFen := paidFen - order.RandomFen
	if order.RandomFen == 0 {
		baseFen = paidFen
	}
	if order.RequestedFen > 0 && baseFen != order.RequestedFen {
		return 0, false
	}
	if !wechatRechargeTierAllowed(baseFen) {
		return 0, false
	}
	return wechatRechargePointsForFen(baseFen, cfg), true
}

func prepareWechatRechargeSession(cfg WechatRechargeConfig) (string, error) {
	accountKey := wechatRechargeBillAccountKey(cfg)
	if strings.TrimSpace(cfg.BillAccountRef) == "" {
		return "", errors.New("后台尚未配置查账账号")
	}
	if cached := getCachedWechatRechargeSession(accountKey); cached != "" {
		if _, renewed, err := queryWechatRechargeBills(cached); err == nil {
			if renewed != "" {
				cached = renewed
			}
			cacheWechatRechargeSession(accountKey, cached)
			return cached, nil
		} else if !isWechatRechargeSessionError(err) {
			return "", err
		}
		invalidateWechatRechargeSession(accountKey, cached)
	}
	return renewWechatRechargeSession(cfg)
}

func renewWechatRechargeSession(cfg WechatRechargeConfig) (string, error) {
	ref := strings.TrimSpace(cfg.BillAccountRef)
	if ref == "" {
		return "", errors.New("后台尚未配置查账账号")
	}
	wxcode, err := ProtocolGetWxAppCode(ref, wechatTallybookAppID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(wxcode) == "" {
		return "", errors.New("查账账号不在线或未返回 code")
	}
	entrance, err := callWechatTally("https://payapp.wechatpay.cn/tallybook/entrance", map[string]interface{}{
		"op_from": 0,
		"wxcode":  wxcode,
	})
	if err != nil {
		return "", err
	}
	if entrance.RetCode != 0 || strings.TrimSpace(entrance.CustomSessionKey) == "" {
		return "", fmt.Errorf("账单登录失败：%s", strings.TrimSpace(entrance.RetMsg))
	}
	session := entrance.CustomSessionKey
	if _, renewed, err := queryWechatRechargeBills(session); err != nil {
		return "", err
	} else if renewed != "" {
		session = renewed
	}
	cacheWechatRechargeSession(wechatRechargeBillAccountKey(cfg), session)
	return session, nil
}

func getCachedWechatRechargeSession(accountKey string) string {
	wechatRechargeSessionCache.RLock()
	defer wechatRechargeSessionCache.RUnlock()
	if wechatRechargeSessionCache.AccountKey != strings.TrimSpace(accountKey) {
		return ""
	}
	return strings.TrimSpace(wechatRechargeSessionCache.SessionKey)
}

func cacheWechatRechargeSession(accountKey, session string) {
	accountKey = strings.TrimSpace(accountKey)
	session = strings.TrimSpace(session)
	if accountKey == "" || session == "" {
		return
	}
	wechatRechargeSessionCache.Lock()
	wechatRechargeSessionCache.AccountKey = accountKey
	wechatRechargeSessionCache.SessionKey = session
	wechatRechargeSessionCache.Unlock()
}

func invalidateWechatRechargeSession(accountKey, session string) {
	wechatRechargeSessionCache.Lock()
	if wechatRechargeSessionCache.AccountKey == strings.TrimSpace(accountKey) &&
		wechatRechargeSessionCache.SessionKey == strings.TrimSpace(session) {
		wechatRechargeSessionCache.AccountKey = ""
		wechatRechargeSessionCache.SessionKey = ""
	}
	wechatRechargeSessionCache.Unlock()
}

func queryWechatRechargeBills(session string) ([]wechatBillRecord, string, error) {
	return queryWechatRechargeBillsAt(session, time.Now())
}

func queryWechatRechargeBillsAt(session string, now time.Time) ([]wechatBillRecord, string, error) {
	resp, err := callWechatTally("https://payapp.wechatpay.cn/tallybook/index", map[string]interface{}{
		"op_from":            0,
		"classification":     0,
		"year":               now.Year(),
		"month":              int(now.Month()),
		"limit":              50,
		"sort_type":          1,
		"is_get_more":        0,
		"skip_grant":         0,
		"use_new_sys":        1,
		"custom_session_key": session,
	})
	if err != nil {
		return nil, "", err
	}
	if resp.RetCode != 0 {
		errText := strings.TrimSpace(resp.RetMsg)
		if isWechatRechargeSessionErrorText(errText) {
			return nil, "", fmt.Errorf("%w：%s", errWechatRechargeSessionExpired, errText)
		}
		return nil, "", fmt.Errorf("账单查询失败：%s", errText)
	}
	var decoded interface{}
	if len(resp.RecordsInfo) > 0 {
		_ = json.Unmarshal(resp.RecordsInfo, &decoded)
	}
	records := make([]wechatBillRecord, 0)
	collectWechatBillRecords(decoded, &records)
	return records, strings.TrimSpace(resp.CustomSessionKey), nil
}

func callWechatTally(endpoint string, payload interface{}) (wechatTallyResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return wechatTallyResponse{}, err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return wechatTallyResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://servicewechat.com/"+wechatTallybookAppID+"/176/page-frame.html")
	req.Header.Set("User-Agent", "Mozilla/5.0 MicroMessenger MiniProgram")
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return wechatTallyResponse{}, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return wechatTallyResponse{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return wechatTallyResponse{}, fmt.Errorf("账单接口返回 HTTP %d", response.StatusCode)
	}
	var result wechatTallyResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return wechatTallyResponse{}, fmt.Errorf("账单响应解析失败：%w", err)
	}
	return result, nil
}

func isWechatRechargeSessionErrorText(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	lowerText := strings.ToLower(text)
	for _, keyword := range []string{"session", "invalid", "expired", "unauthorized", "登录", "会话", "失效", "过期"} {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}

func isWechatRechargeSessionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errWechatRechargeSessionExpired) {
		return true
	}
	text := strings.ToLower(err.Error())
	for _, keyword := range []string{"session", "登录失效", "会话失效", "会话过期", "invalid", "expired", "unauthorized", "未登录"} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func collectWechatBillRecords(value interface{}, records *[]wechatBillRecord) {
	switch current := value.(type) {
	case map[string]interface{}:
		if billType, ok := wechatRechargeInt(current["bill_type"]); ok {
			balance, _ := wechatRechargeInt(current["balance"])
			billTime, _ := wechatRechargeInt64(current["bill_time"])
			record := wechatBillRecord{
				ID:       strings.TrimSpace(fmt.Sprint(current["id"])),
				TransID:  strings.TrimSpace(fmt.Sprint(current["trans_id"])),
				BillTime: billTime,
				BillType: billType,
				Balance:  balance,
				Remark:   strings.TrimSpace(fmt.Sprint(current["remark"])),
			}
			if record.TransID == "" || record.TransID == "<nil>" {
				record.TransID = record.ID
			}
			if record.TransID == "<nil>" {
				record.TransID = ""
			}
			*records = append(*records, record)
			return
		}
		for _, child := range current {
			collectWechatBillRecords(child, records)
		}
	case []interface{}:
		for _, child := range current {
			collectWechatBillRecords(child, records)
		}
	}
}

func wechatRechargeInt(value interface{}) (int, bool) {
	switch current := value.(type) {
	case float64:
		return int(current), true
	case json.Number:
		n, err := current.Int64()
		return int(n), err == nil
	case int:
		return current, true
	case int64:
		return int(current), true
	case string:
		text := strings.TrimSpace(current)
		if text == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(text, 10, 64)
		return int(n), err == nil
	default:
		return 0, false
	}
}

func wechatRechargeInt64(value interface{}) (int64, bool) {
	var result int64
	var ok bool
	switch current := value.(type) {
	case float64:
		result, ok = int64(current), true
	case json.Number:
		n, err := current.Int64()
		result, ok = n, err == nil
	case int:
		result, ok = int64(current), true
	case int64:
		result, ok = current, true
	case string:
		text := strings.TrimSpace(current)
		if text == "" {
			return 0, false
		}
		n, err := strconv.ParseInt(text, 10, 64)
		result, ok = n, err == nil
	default:
		return 0, false
	}
	if !ok {
		return 0, false
	}
	if result >= 1000000000000 {
		result /= 1000
	}
	return result, true
}

func processWechatRechargeOrders() {
	if db == nil {
		return
	}
	now := time.Now()
	_ = db.Model(&WechatRechargeOrder{}).
		Where("status IN ? AND expires_at <= ?", []string{"preparing", "pending"}, now.Add(-wechatRechargeGrace)).
		Updates(map[string]interface{}{"status": "expired", "session_key": ""}).Error
	var orders []WechatRechargeOrder
	if err := db.Where("status = ? AND expires_at > ?", "pending", now.Add(-wechatRechargeGrace)).Order("id ASC").Find(&orders).Error; err != nil {
		logs.Warn("查询微信充值待支付订单失败：%v", err)
		return
	}
	cfg := GetWechatRechargeConfig()
	for i := range orders {
		processWechatRechargeOrder(&orders[i], cfg)
	}
}

func processWechatRechargeOrder(order *WechatRechargeOrder, cfg WechatRechargeConfig) {
	if order == nil || strings.TrimSpace(order.SessionKey) == "" {
		return
	}
	pollNow := time.Now()
	records, renewedSession, err := queryWechatRechargeBillsAt(order.SessionKey, pollNow)
	if err == nil && (order.CreatedAt.Year() != pollNow.Year() || order.CreatedAt.Month() != pollNow.Month()) {
		monthSession := order.SessionKey
		if renewedSession != "" {
			monthSession = renewedSession
		}
		previousRecords, previousSession, previousErr := queryWechatRechargeBillsAt(monthSession, order.CreatedAt)
		if previousErr == nil {
			records = append(records, previousRecords...)
			if previousSession != "" {
				renewedSession = previousSession
			}
		}
	}
	updates := map[string]interface{}{"last_polled_at": pollNow}
	accountKey := wechatRechargeBillAccountKey(cfg)
	if renewedSession != "" {
		updates["session_key"] = renewedSession
		order.SessionKey = renewedSession
		cacheWechatRechargeSession(accountKey, renewedSession)
	}
	if err != nil {
		lastError := err.Error()
		if isWechatRechargeSessionError(err) {
			invalidateWechatRechargeSession(accountKey, order.SessionKey)
			if refreshed, refreshErr := renewWechatRechargeSession(cfg); refreshErr == nil {
				updates["session_key"] = refreshed
				order.SessionKey = refreshed
				lastError = ""
			} else {
				lastError = "账单会话刷新失败：" + refreshErr.Error()
			}
		}
		updates["last_error"] = truncateWechatRechargeError(lastError)
		_ = db.Model(&WechatRechargeOrder{}).Where("id = ? AND status = ?", order.ID, "pending").Updates(updates).Error
		return
	}
	updates["last_error"] = ""
	_ = db.Model(&WechatRechargeOrder{}).Where("id = ? AND status = ?", order.ID, "pending").Updates(updates).Error
	for _, record := range records {
		if !wechatRechargeRecordMatches(*order, record) {
			continue
		}
		if wechatRechargeReceiptUsed(record.TransID) {
			continue
		}
		points, ok := wechatRechargePointsForPayment(*order, record.Balance)
		if !ok {
			continue
		}
		if err := creditWechatRechargeOrder(*order, record, points); err != nil {
			logs.Error("微信充值入账失败 order=%s: %v", order.OrderNo, err)
		}
		return
	}
}

func wechatRechargeRecordMatches(order WechatRechargeOrder, record wechatBillRecord) bool {
	if record.BillType != 2 || record.TransID == "" || record.Balance <= 0 {
		return false
	}
	if record.Balance != wechatRechargeTargetFen(order) {
		return false
	}
	if _, ok := wechatRechargePointsForPayment(order, record.Balance); !ok {
		return false
	}
	visibleAt := order.CreatedAt
	if order.ActivatedAt != nil {
		visibleAt = *order.ActivatedAt
	}
	if record.BillTime < visibleAt.Unix() || record.BillTime > order.ExpiresAt.Unix() {
		return false
	}
	if !strings.Contains(record.Remark, "二维码收款") && !strings.Contains(record.Remark, "赞赏") {
		return false
	}
	return true
}

func wechatRechargeReceiptUsed(transID string) bool {
	var count int64
	_ = db.Model(&WechatRechargeReceipt{}).Where("trans_id = ?", strings.TrimSpace(transID)).Count(&count).Error
	return count > 0
}

func creditWechatRechargeOrder(order WechatRechargeOrder, record wechatBillRecord, points int) error {
	paidAt := time.Now()
	credited := false
	err := db.Transaction(func(tx *gorm.DB) error {
		var current WechatRechargeOrder
		if err := tx.Where("id = ? AND status = ?", order.ID, "pending").First(&current).Error; err != nil {
			return err
		}
		claim := tx.Model(&WechatRechargeOrder{}).Where("id = ? AND status = ?", current.ID, "pending").Update("status", "crediting")
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return errors.New("充值订单已被其他轮询处理")
		}
		if err := tx.Create(&WechatRechargeReceipt{TransID: record.TransID, OrderNo: current.OrderNo}).Error; err != nil {
			return err
		}
		result := tx.Model(&User{}).Where("number = ?", current.QQ).Updates(map[string]interface{}{
			"coin":         gorm.Expr("coin + ?", points),
			"total_earned": gorm.Expr("total_earned + ?", points),
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("充值用户不存在")
		}
		result = tx.Model(&WechatRechargeOrder{}).Where("id = ? AND status = ?", current.ID, "crediting").Updates(map[string]interface{}{
			"status":      "paid",
			"paid_fen":    record.Balance,
			"points":      points,
			"trans_id":    record.TransID,
			"paid_at":     paidAt,
			"session_key": "",
			"last_error":  "",
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("充值订单状态更新失败")
		}
		credited = true
		return nil
	})
	if err != nil {
		notifyWechatRechargeAdmin("微信充值入账失败", order, err.Error())
		return err
	}
	if !credited {
		return nil
	}
	RecordCoinLog(order.QQ, points, "微信充值", fmt.Sprintf("微信赞赏充值 %.2f 元，订单 %s", float64(record.Balance)/100, order.OrderNo), WebContext())
	msg := fmt.Sprintf("微信充值成功！支付 %.2f 元，已增加 %d 积分，当前积分 %d。", float64(record.Balance)/100, points, GetCoin(order.QQ))
	SendQQ(order.QQ, msg)
	notifyWechatRechargeAdmin("微信充值成功", order, fmt.Sprintf("支付 %.2f 元，到账 %d 积分", float64(record.Balance)/100, points))
	return nil
}

func notifyWechatRechargeAdmin(title string, order WechatRechargeOrder, detail string) {
	channelName := map[string]string{"portal": "网页", "bot": "机器人", "app": "App"}[order.Channel]
	if channelName == "" {
		channelName = order.Channel
	}
	msg := fmt.Sprintf("%s\nQQ：%d\n来源：%s\n档位：%.2f 元\n订单号：%s", title, order.QQ, channelName, float64(order.RequestedFen)/100, order.OrderNo)
	if strings.TrimSpace(detail) != "" {
		msg += "\n详情：" + detail
	}
	(&JdCookie{}).Push(msg)
}

func truncateWechatRechargeError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 300 {
		value = value[:300]
	}
	return value
}

// HandleBotWechatRechargeTier 机器人选定档位后创建订单并发送二维码
func HandleBotWechatRechargeTier(sender *Sender, tierIndex int) {
	cfg := GetWechatRechargeConfig()
	if !cfg.Enabled || !cfg.BotEnabled {
		sender.Reply("请打开以下链接购买卡密充值：\n" + WechatRechargeFallbackURL())
		return
	}
	if tierIndex < 1 || tierIndex > len(cfg.TiersYuan) {
		sender.Reply("档位序号无效，请重新发送「充值」。")
		return
	}
	requestedFen := cfg.TiersYuan[tierIndex-1] * 100
	order, err := CreateWechatRechargeOrder(sender.UserID, requestedFen, "bot")
	if err != nil {
		if errors.Is(err, ErrWechatRechargeDailyLimit) {
			sender.Reply(err.Error() + "：\n" + WechatRechargeFallbackURL())
			return
		}
		if errors.Is(err, ErrWechatRechargeBusy) {
			sender.Reply(fmt.Sprintf("当前充值人数已满（最多 %d 人同时充值），请稍后再试。", cfg.MaxConcurrent))
			return
		}
		sender.Reply("微信充值服务暂不可用，请打开以下链接购买卡密充值：\n" + WechatRechargeFallbackURL())
		return
	}
	image, _, err := ReadWechatRechargeQRCodeBytes()
	if err != nil {
		CancelWechatRechargeOrder(order.OrderNo, sender.UserID, "读取收款码失败")
		sender.Reply("微信充值服务暂不可用，请打开以下链接购买卡密充值：\n" + WechatRechargeFallbackURL())
		return
	}
	points := wechatRechargePointsForFen(requestedFen, cfg)
	sender.Reply(fmt.Sprintf("【重要】必须支付精确金额：%.2f 元（不是 %d 元整）\n到账积分：%d\n\n付错金额无法自动到账！订单号：%s\n若付错请联系管理员并提供截图。\n\n下面是收款二维码：", float64(wechatRechargeTargetFen(order))/100, requestedFen/100, points, order.OrderNo))
	sender.SendImg(image)
}

// StartBotWechatRecharge 机器人充值档位选择会话
func StartBotWechatRecharge(sender *Sender) string {
	cfg := GetWechatRechargeConfig()
	if !cfg.Enabled || !cfg.BotEnabled {
		return "请打开以下链接购买卡密充值：\n" + WechatRechargeFallbackURL()
	}
	if online, reason := WechatRechargeBillAccountStatus(cfg); !online {
		return reason + "\n请稍后再试，或打开以下链接购买卡密充值：\n" + WechatRechargeExternalPurchaseURL()
	}
	if sender.Type != "qq" {
		return "请私聊 QQ 机器人发送「充值」。"
	}
	msg := make(chan string)
	if !chanSetIfAbsent(inputList, sender.UserID, msg) {
		return "您当前已有一个会话，请完成当前操作或输入 q 退出。"
	}
	tiers := cfg.TiersYuan
	go func() {
		defer chanDelIf(inputList, sender.UserID, msg)
		deadline := time.NewTimer(120 * time.Second)
		defer deadline.Stop()
		for {
			select {
			case input := <-msg:
				cfgNow := GetWechatRechargeConfig()
				if !cfgNow.Enabled || !cfgNow.BotEnabled {
					chanDelIf(inputList, sender.UserID, msg)
					sender.Reply("请打开以下链接购买卡密充值：\n" + WechatRechargeFallbackURL())
					return
				}
				input = strings.TrimSpace(input)
				if strings.EqualFold(input, "q") {
					chanDelIf(inputList, sender.UserID, msg)
					sender.Reply("已退出微信充值。")
					return
				}
				idx, err := strconv.Atoi(input)
				if err != nil || idx < 1 || idx > len(cfgNow.TiersYuan) {
					sender.Reply(fmt.Sprintf("请输入 1-%d 之间的档位序号，输入 q 退出。", len(cfgNow.TiersYuan)))
					continue
				}
				chanDelIf(inputList, sender.UserID, msg)
				sender.Reply("正在检测微信充值服务，请稍候...")
				HandleBotWechatRechargeTier(sender, idx)
				return
			case <-deadline.C:
				sender.Reply("微信充值已超时，请重新发送「充值」。")
				return
			}
		}
	}()
	var b strings.Builder
	fmt.Fprintf(&b, "微信赞赏码充值（1元=%d积分）\n请选择充值档位：\n", cfg.PointsPerYuan)
	for i, y := range tiers {
		fmt.Fprintf(&b, "%d. %d元 → %d积分\n", i+1, y, wechatRechargePointsForFen(y*100, cfg))
	}
	b.WriteString("\n请输入序号，输入 q 退出：")
	return b.String()
}
