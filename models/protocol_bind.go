package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const YybOpenIDPrefix = "owNAX"

// PortalProtocolBinding 微信协议 wxid 与应用宝 openid 一一对应双绑
type PortalProtocolBinding struct {
	ID         int64          `gorm:"column:id;primaryKey" json:"id"`
	UserNumber int            `gorm:"column:user_number;index;not null" json:"userNumber"`
	WxWxid     string         `gorm:"column:wx_wxid;size:128;not null;uniqueIndex:idx_proto_bind_wx" json:"wxWxid"`
	YybOpenID  string         `gorm:"column:yyb_open_id;size:128;not null;uniqueIndex:idx_proto_bind_openid" json:"yybOpenId"`
	Nickname   string         `gorm:"column:nickname;size:128" json:"nickname"`
	CreatedAt  time.Time      `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt  time.Time      `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (PortalProtocolBinding) TableName() string { return "portal_protocol_bindings" }

// ProtocolRoute 协议路由结果
type ProtocolRoute struct {
	Backend    string // yyb | wechat
	OpenID     string // 应用宝 openid（Backend=yyb 时）
	WxWxid     string // 原始微信 id（Backend=wechat 或回退时）
	FromBind   bool
	DirectYYB  bool
	InputRef   string
}

func protocolRefShort(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	if len(s) <= 18 {
		return s
	}
	return s[:8] + "…" + s[len(s)-6:]
}

// LogSummary 供应用宝分类日志展示路由决策
func (r ProtocolRoute) LogSummary() string {
	if r.Backend == "yyb" {
		switch {
		case r.DirectYYB:
			return fmt.Sprintf("route=应用宝 mode=openid直连 openid=%s", protocolRefShort(r.OpenID))
		case r.FromBind:
			return fmt.Sprintf("route=应用宝 mode=双绑 wxid=%s openid=%s", protocolRefShort(r.WxWxid), protocolRefShort(r.OpenID))
		default:
			return fmt.Sprintf("route=应用宝 openid=%s", protocolRefShort(r.OpenID))
		}
	}
	return fmt.Sprintf("route=wechat08 wxid=%s", protocolRefShort(r.WxWxid))
}

func IsYybOpenIDRef(ref string) bool {
	return strings.HasPrefix(strings.TrimSpace(ref), YybOpenIDPrefix)
}

func NormalizeOpenID(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if IsYybOpenIDRef(ref) {
		return ref
	}
	return ref
}

func FindProtocolBindingByWx(wxid string) (*PortalProtocolBinding, error) {
	wxid = strings.TrimSpace(wxid)
	if wxid == "" {
		return nil, nil
	}
	var row PortalProtocolBinding
	err := db.Where("wx_wxid = ?", wxid).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func FindProtocolBindingByOpenID(openid string) (*PortalProtocolBinding, error) {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return nil, nil
	}
	var row PortalProtocolBinding
	err := db.Where("yyb_open_id = ?", openid).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func ListProtocolBindings(userNumber int) ([]PortalProtocolBinding, error) {
	var rows []PortalProtocolBinding
	err := db.Where("user_number = ?", userNumber).Order("id asc").Find(&rows).Error
	return rows, err
}

func CountProtocolBindings(userNumber int) (int64, error) {
	var n int64
	err := db.Model(&PortalProtocolBinding{}).Where("user_number = ?", userNumber).Count(&n).Error
	return n, err
}

// CountPortalYybBindings 用户库内已成功登录的应用宝数量（含掉线/失效，不含已删除）
func CountPortalYybBindings(userNumber int) int {
	var n int64
	if err := GormDB().Table("portal_yyb_bindings").Where("user_number = ?", userNumber).Count(&n).Error; err != nil {
		return 0
	}
	return int(n)
}

// YybFreeSlotsForNewLogin 新 OpenID 登录剩余免费名额（在线微信数 − 库内应用宝数）
func YybFreeSlotsForNewLogin(userNumber int) int {
	online := CountOnlineWxProtocolSlots(userNumber)
	used := CountPortalYybBindings(userNumber)
	free := online - used
	if free < 0 {
		return 0
	}
	return free
}

// CalcYybScanLoginCost 新 OpenID 扫码费用：库内应用宝数（含掉线）未达在线微信数则免费。
// 已有 OpenID 续登在确认阶段另判免费，此处 hint 会提示「续登免费 / 新增才扣」。
func CalcYybScanLoginCost(userNumber int) (cost int, free bool, hint string) {
	cost = getYybScanLoginCostConfigured()
	online := CountOnlineWxProtocolSlots(userNumber)
	yybInLibrary := CountPortalYybBindings(userNumber)
	if online <= 0 {
		if cost <= 0 {
			return 0, true, "本次扫码免费，不扣除积分"
		}
		if yybInLibrary > 0 {
			return cost, false, fmt.Sprintf("已有账号续登免费；新增账号将扣除 %d 积分", cost)
		}
		return cost, false, fmt.Sprintf("本次扫码将扣除 %d 积分", cost)
	}
	freeSlots := online - yybInLibrary
	if freeSlots > 0 {
		return 0, true, "本次扫码免费，不扣除积分"
	}
	if cost <= 0 {
		return 0, true, "本次扫码免费，不扣除积分"
	}
	if yybInLibrary > 0 {
		return cost, false, fmt.Sprintf("已有账号续登免费；新增账号将扣除 %d 积分", cost)
	}
	return cost, false, fmt.Sprintf("本次扫码将扣除 %d 积分", cost)
}

// CountOnlineWxProtocolSlots 在线微信协议名额（主绑定 + 额外设备，仅在线）
func CountOnlineWxProtocolSlots(userNumber int) int {
	sender := &Sender{UserID: userNumber}
	devices := findUserProtocolDevices(sender)
	return len(devices)
}

func getYybScanLoginCostConfigured() int {
	if Config.Yyb.ScanLoginCost != nil {
		return *Config.Yyb.ScanLoginCost
	}
	if Config.WxProtocol.ScanLoginCost > 0 {
		return Config.WxProtocol.ScanLoginCost
	}
	return 2000
}

// ResolveProtocolRoute 根据 ref 决定走应用宝还是微信协议
func ResolveProtocolRoute(ref string) ProtocolRoute {
	ref = strings.TrimSpace(ref)
	route := ProtocolRoute{InputRef: ref, Backend: "wechat", WxWxid: ref}
	if ref == "" {
		return route
	}
	if IsYybOpenIDRef(ref) {
		route.Backend = "yyb"
		route.OpenID = ref
		route.DirectYYB = true
		return route
	}
	if b, err := FindProtocolBindingByWx(ref); err == nil && b != nil {
		route.Backend = "yyb"
		route.OpenID = b.YybOpenID
		route.WxWxid = b.WxWxid
		route.FromBind = true
		return route
	}
	return route
}

// BindProtocolPair 建立双绑（1:1 全局唯一）
func BindProtocolPair(userNumber int, wxWxid, yybOpenID, nickname string) (*PortalProtocolBinding, error) {
	wxWxid = strings.TrimSpace(wxWxid)
	yybOpenID = strings.TrimSpace(yybOpenID)
	if wxWxid == "" || yybOpenID == "" {
		return nil, fmt.Errorf("微信ID和应用宝openid不能为空")
	}
	if IsYybOpenIDRef(wxWxid) {
		return nil, fmt.Errorf("请使用微信协议ID绑定，不要填写应用宝openid")
	}
	if !IsYybOpenIDRef(yybOpenID) {
		return nil, fmt.Errorf("应用宝openid必须以 %s 开头", YybOpenIDPrefix)
	}
	if b, _ := FindProtocolBindingByWx(wxWxid); b != nil && b.UserNumber != userNumber {
		return nil, fmt.Errorf("该微信ID已被其他用户绑定")
	}
	if b, _ := FindProtocolBindingByOpenID(yybOpenID); b != nil && b.UserNumber != userNumber {
		return nil, fmt.Errorf("该应用宝账号已被其他用户绑定")
	}
	var existing PortalProtocolBinding
	err := db.Where("user_number = ? AND wx_wxid = ?", userNumber, wxWxid).First(&existing).Error
	if err == nil {
		if existing.YybOpenID != yybOpenID {
			if other, _ := FindProtocolBindingByOpenID(yybOpenID); other != nil && other.ID != existing.ID {
				return nil, fmt.Errorf("目标应用宝账号已绑定其他微信")
			}
			existing.YybOpenID = yybOpenID
		}
		if strings.TrimSpace(nickname) != "" {
			existing.Nickname = strings.TrimSpace(nickname)
		}
		existing.UpdatedAt = time.Now()
		if err := db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if other, _ := FindProtocolBindingByOpenID(yybOpenID); other != nil {
		return nil, fmt.Errorf("该应用宝账号已绑定其他微信")
	}
	row := PortalProtocolBinding{
		UserNumber: userNumber,
		WxWxid:     wxWxid,
		YybOpenID:  yybOpenID,
		Nickname:   strings.TrimSpace(nickname),
	}
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// UnbindProtocolPair 解除双绑
func UnbindProtocolPair(userNumber int, wxWxid, yybOpenID string) error {
	wxWxid = strings.TrimSpace(wxWxid)
	yybOpenID = strings.TrimSpace(yybOpenID)
	q := db.Where("user_number = ?", userNumber)
	if wxWxid != "" {
		q = q.Where("wx_wxid = ?", wxWxid)
	}
	if yybOpenID != "" {
		q = q.Where("yyb_open_id = ?", yybOpenID)
	}
	if wxWxid == "" && yybOpenID == "" {
		return fmt.Errorf("请指定要解绑的微信ID或应用宝openid")
	}
	res := q.Delete(&PortalProtocolBinding{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("未找到绑定记录")
	}
	return nil
}
