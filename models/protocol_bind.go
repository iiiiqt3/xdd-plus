package models

import (
	"errors"
	"fmt"
	"sort"
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

// WxOfflineNotifySkippedByDualBind 已双绑应用宝，或同一用户已登录应用宝时，跳过微信掉线推送（由应用宝存活检测统一通知）
func WxOfflineNotifySkippedByDualBind(wxid string) bool {
	wxid = strings.TrimSpace(wxid)
	if wxid == "" {
		return false
	}
	if b, err := FindProtocolBindingByWx(wxid); err == nil && b != nil {
		return true
	}
	for _, userNumber := range listUserNumbersOwningWxid(wxid) {
		if CountPortalYybBindings(userNumber) > 0 {
			return true
		}
	}
	return false
}

func listUserNumbersOwningWxid(wxid string) []int {
	seen := map[int]bool{}
	out := make([]int, 0, 2)
	add := func(n int) {
		if n > 0 && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	var users []User
	db.Where("wxid = ?", wxid).Select("number").Find(&users)
	for _, u := range users {
		add(u.Number)
	}
	var devices []PortalWxDevice
	db.Where("wxid = ?", wxid).Select("user_number").Find(&devices)
	for _, d := range devices {
		add(d.UserNumber)
	}
	return out
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

// YybScanRegionHint 扫码登录地区提示（门户/App/网页共用）
func YybScanRegionHint() string {
	return "请务必选择与你实际所在地一致的省/市。地区正确时有效期约 30 天；若微信提示「异地登录」，有效期可能仅 1 天。"
}

// YybScanRegionQrHint 与积分说明并列展示的短提醒（扫码页/积分预览区）
func YybScanRegionQrHint() string {
	return "若微信提示「异地登录」，说明地区不匹配，有效期可能仅 1 天，请重新选择与你所在地一致的省/市。"
}

// YybScanRegionHintWithBypass 带免代理地区前缀的地区提示
func YybScanRegionHintWithBypass(bypassRegionName string) string {
	base := YybScanRegionHint()
	if name := strings.TrimSpace(bypassRegionName); name != "" {
		return fmt.Sprintf("「%s」等地区可免代理直连；%s", name, base)
	}
	return base
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
	purgeProtocolBindingGhosts(wxWxid, yybOpenID)
	if b, _ := FindProtocolBindingByWx(wxWxid); b != nil && b.UserNumber != userNumber {
		return nil, fmt.Errorf("该微信ID已被其他用户绑定")
	}
	if b, _ := FindProtocolBindingByOpenID(yybOpenID); b != nil && b.UserNumber != userNumber {
		return nil, fmt.Errorf("该应用宝账号已被其他用户绑定")
	}
	var existing PortalProtocolBinding
	err := db.Where("user_number = ? AND wx_wxid = ?", userNumber, wxWxid).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = db.Unscoped().Where("user_number = ? AND wx_wxid = ?", userNumber, wxWxid).First(&existing).Error
		if err == nil {
			existing.DeletedAt = gorm.DeletedAt{}
		}
	}
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
		if err := db.Unscoped().Save(&existing).Error; err != nil {
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

// UnbindProtocolPair 解除双绑（物理删除，避免软删占位导致唯一索引冲突）
func UnbindProtocolPair(userNumber int, wxWxid, yybOpenID string) error {
	wxWxid = strings.TrimSpace(wxWxid)
	yybOpenID = strings.TrimSpace(yybOpenID)
	q := db.Unscoped().Where("user_number = ?", userNumber)
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

// purgeProtocolBindingGhosts 清理已软删但仍占用唯一索引的双绑残留
func purgeProtocolBindingGhosts(wxWxid, yybOpenID string) {
	q := db.Unscoped().Model(&PortalProtocolBinding{}).Where("deleted_at IS NOT NULL")
	switch {
	case wxWxid != "" && yybOpenID != "":
		q = q.Where("wx_wxid = ? OR yyb_open_id = ?", wxWxid, yybOpenID)
	case wxWxid != "":
		q = q.Where("wx_wxid = ?", wxWxid)
	case yybOpenID != "":
		q = q.Where("yyb_open_id = ?", yybOpenID)
	default:
		return
	}
	_ = q.Delete(&PortalProtocolBinding{}).Error
}

// ProtocolYybAccountBrief 应用宝账号摘要（供协议选项合并）
type ProtocolYybAccountBrief struct {
	OpenID   string
	Nickname string
	Status   string
}

// ProtocolAccountOption 协议活动可选账号
type ProtocolAccountOption struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Nickname       string `json:"nickname"`
	Mode           string `json:"mode"`
	WxWxid         string `json:"wxid,omitempty"`
	OpenID         string `json:"openid,omitempty"`
	FillRef        string `json:"fillRef"`
	UsedInActivity bool   `json:"usedInActivity"`
	Selectable     bool   `json:"selectable"`
}

func isYybAccountAlive(status string) bool {
	st := strings.ToLower(strings.TrimSpace(status))
	return st == "alive" || st == "online"
}

func pickNickname(parts ...string) string {
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			return p
		}
	}
	return "未命名账号"
}

func buildProtocolOptionLabel(opt ProtocolAccountOption, used bool) string {
	tag := "微信"
	displayRef := opt.FillRef
	switch opt.Mode {
	case "dual":
		tag = "已双绑"
		if strings.TrimSpace(opt.WxWxid) != "" {
			displayRef = opt.WxWxid
		}
	case "yyb_only":
		tag = "应用宝"
	}
	label := fmt.Sprintf("%s · %s · %s", opt.Nickname, protocolRefShort(displayRef), tag)
	if used {
		label += " · 已上车"
	}
	return label
}

func identityMatchesRef(identityID, wxid, openid, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	if wxid != "" && ref == wxid {
		return true
	}
	if openid != "" && ref == openid {
		return true
	}
	switch {
	case strings.HasPrefix(identityID, "dual:"):
		idOpen := strings.TrimPrefix(identityID, "dual:")
		return ref == idOpen || (openid != "" && ref == openid) || (wxid != "" && ref == wxid)
	case strings.HasPrefix(identityID, "wx:"):
		return ref == strings.TrimPrefix(identityID, "wx:")
	case strings.HasPrefix(identityID, "yyb:"):
		return ref == strings.TrimPrefix(identityID, "yyb:")
	default:
		return false
	}
}

type protocolUsedState struct {
	ids     map[string]bool
	wxids   map[string]bool
	openids map[string]bool
}

func projectRemarkAlias(p ActivityProject) string {
	if strings.TrimSpace(p.RemarkAlias) != "" {
		return strings.TrimSpace(p.RemarkAlias)
	}
	remarks := strings.TrimSpace(p.Remarks)
	if idx := strings.Index(remarks, "/"); idx > 0 {
		return remarks[:idx]
	}
	return remarks
}

func extractProtocolRefsFromProject(template, envValue string, wxToOpen map[string]string) []string {
	seen := make(map[string]bool)
	var refs []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			return
		}
		seen[v] = true
		refs = append(refs, v)
	}
	for _, v := range ExtractProtocolRefsFromCK(template, envValue) {
		add(v)
	}
	for wxid, openid := range wxToOpen {
		if strings.Contains(envValue, wxid) {
			add(wxid)
		}
		if strings.Contains(envValue, openid) {
			add(openid)
		}
	}
	return refs
}

func collectProtocolUsedState(userNumber int, cfg *ActivityConfig, excludeRemarks string) (*protocolUsedState, error) {
	state := &protocolUsedState{
		ids:     make(map[string]bool),
		wxids:   make(map[string]bool),
		openids: make(map[string]bool),
	}
	if cfg == nil {
		return state, nil
	}
	projects, err := GetActivityProjectsByUserAndEnv(userNumber, cfg.ID, cfg.EnvKey)
	if err != nil {
		return nil, err
	}
	bindings, _ := ListProtocolBindings(userNumber)
	wxToOpen := make(map[string]string, len(bindings))
	openToWx := make(map[string]string, len(bindings))
	for _, b := range bindings {
		wxToOpen[b.WxWxid] = b.YybOpenID
		openToWx[b.YybOpenID] = b.WxWxid
	}
	excludeRemarks = strings.TrimSpace(excludeRemarks)
	for _, p := range projects {
		if excludeRemarks != "" && projectRemarkAlias(p) == excludeRemarks {
			continue
		}
		for _, ref := range extractProtocolRefsFromProject(cfg.CKTemplate, p.EnvValue, wxToOpen) {
			if id := protocolIdentityID(ref, wxToOpen, openToWx); id != "" {
				state.ids[id] = true
			}
			if IsYybOpenIDRef(ref) {
				state.openids[ref] = true
			} else {
				state.wxids[ref] = true
			}
		}
	}
	return state, nil
}

func (s *protocolUsedState) optionUsed(opt ProtocolAccountOption, wxToOpen, openToWx map[string]string) bool {
	if s == nil {
		return false
	}
	if s.ids[opt.ID] {
		return true
	}
	if opt.WxWxid != "" {
		if s.wxids[opt.WxWxid] || s.ids["wx:"+opt.WxWxid] {
			return true
		}
		if openid, ok := wxToOpen[opt.WxWxid]; ok && openid != "" {
			if s.openids[openid] || s.ids["dual:"+openid] || s.ids["yyb:"+openid] {
				return true
			}
		}
	}
	if opt.OpenID != "" {
		if s.openids[opt.OpenID] || s.ids["dual:"+opt.OpenID] || s.ids["yyb:"+opt.OpenID] {
			return true
		}
		if wxid, ok := openToWx[opt.OpenID]; ok && wxid != "" {
			if s.wxids[wxid] || s.ids["wx:"+wxid] {
				return true
			}
		}
	}
	return false
}

func protocolIdentityID(ref string, wxToOpen, openToWx map[string]string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if IsYybOpenIDRef(ref) {
		if wx, ok := openToWx[ref]; ok && wx != "" {
			return "dual:" + ref
		}
		return "yyb:" + ref
	}
	if open, ok := wxToOpen[ref]; ok && open != "" {
		return "dual:" + open
	}
	return "wx:" + ref
}

// BuildProtocolAccountOptions 构建协议活动账号下拉选项
func BuildProtocolAccountOptions(userNumber int, activityID, excludeRemarks string, yybAccounts []ProtocolYybAccountBrief) ([]ProtocolAccountOption, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return nil, fmt.Errorf("活动不存在")
	}
	if !cfg.IsProtocolActivity {
		return []ProtocolAccountOption{}, nil
	}

	usedState, err := collectProtocolUsedState(userNumber, cfg, excludeRemarks)
	if err != nil {
		return nil, err
	}

	bindings, err := ListProtocolBindings(userNumber)
	if err != nil {
		return nil, err
	}
	wxToOpen := make(map[string]string, len(bindings))
	openToWx := make(map[string]string, len(bindings))
	for _, b := range bindings {
		wxToOpen[b.WxWxid] = b.YybOpenID
		openToWx[b.YybOpenID] = b.WxWxid
	}

	wxDevices, err := GetPortalWxDevices(userNumber)
	if err != nil {
		return nil, err
	}

	boundWx := make(map[string]PortalProtocolBinding, len(bindings))
	boundOpen := make(map[string]PortalProtocolBinding, len(bindings))
	for _, b := range bindings {
		boundWx[b.WxWxid] = b
		boundOpen[b.YybOpenID] = b
	}

	yybByOpen := make(map[string]ProtocolYybAccountBrief, len(yybAccounts))
	for _, a := range yybAccounts {
		openid := strings.TrimSpace(a.OpenID)
		if openid == "" {
			continue
		}
		yybByOpen[openid] = a
	}

	wxNick := make(map[string]string)
	for _, d := range wxDevices {
		wxNick[d.Wxid] = d.Nickname
	}

	options := make([]ProtocolAccountOption, 0)
	seenIdentity := make(map[string]bool)

	addOption := func(opt ProtocolAccountOption) {
		if opt.ID == "" || seenIdentity[opt.ID] {
			return
		}
		seenIdentity[opt.ID] = true
		opt.UsedInActivity = usedState.optionUsed(opt, wxToOpen, openToWx)
		opt.Selectable = !opt.UsedInActivity
		opt.Label = buildProtocolOptionLabel(opt, opt.UsedInActivity)
		options = append(options, opt)
	}

	for _, b := range bindings {
		yyb, ok := yybByOpen[b.YybOpenID]
		if !ok || !isYybAccountAlive(yyb.Status) {
			continue
		}
		nickname := pickNickname(b.Nickname, yyb.Nickname, wxNick[b.WxWxid])
		addOption(ProtocolAccountOption{
			ID:       "dual:" + b.YybOpenID,
			Nickname: nickname,
			Mode:     "dual",
			WxWxid:   b.WxWxid,
			OpenID:   b.YybOpenID,
			FillRef:  b.YybOpenID,
		})
	}

	for _, d := range wxDevices {
		wxid := strings.TrimSpace(d.Wxid)
		if wxid == "" || !d.Online {
			continue
		}
		if _, bound := boundWx[wxid]; bound {
			continue
		}
		nickname := pickNickname(d.Nickname)
		addOption(ProtocolAccountOption{
			ID:       "wx:" + wxid,
			Nickname: nickname,
			Mode:     "wx_only",
			WxWxid:   wxid,
			FillRef:  wxid,
		})
	}

	for _, a := range yybAccounts {
		openid := strings.TrimSpace(a.OpenID)
		if openid == "" || !isYybAccountAlive(a.Status) {
			continue
		}
		if _, bound := boundOpen[openid]; bound {
			continue
		}
		nickname := pickNickname(a.Nickname)
		addOption(ProtocolAccountOption{
			ID:       "yyb:" + openid,
			Nickname: nickname,
			Mode:     "yyb_only",
			OpenID:   openid,
			FillRef:  openid,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		if options[i].UsedInActivity != options[j].UsedInActivity {
			return !options[i].UsedInActivity
		}
		return options[i].Label < options[j].Label
	})
	return options, nil
}

// ValidateProtocolRefForActivity 校验协议引用是否可用于活动（上车/改 CK）
func ValidateProtocolRefForActivity(userNumber int, activityID, excludeRemarks, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("请选择协议账号")
	}
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return fmt.Errorf("活动不存在")
	}
	if !cfg.IsProtocolActivity {
		return nil
	}
	options, err := BuildProtocolAccountOptions(userNumber, activityID, excludeRemarks, listYybBriefsForUser(userNumber))
	if err != nil {
		return err
	}
	for _, opt := range options {
		if identityMatchesRef(opt.ID, opt.WxWxid, opt.OpenID, ref) {
			if !opt.Selectable {
				return fmt.Errorf("该协议账号已在本活动中使用")
			}
			return nil
		}
	}
	return fmt.Errorf("协议账号无效或已掉线，请重新选择")
}

var protocolListYybBriefsFn func(userNumber int) ([]ProtocolYybAccountBrief, error)

// SetProtocolListYybBriefsFn 注入应用宝账号列表（避免 models ↔ yybportal 循环依赖）
func SetProtocolListYybBriefsFn(fn func(userNumber int) ([]ProtocolYybAccountBrief, error)) {
	protocolListYybBriefsFn = fn
}

func listYybBriefsForUser(userNumber int) []ProtocolYybAccountBrief {
	if protocolListYybBriefsFn == nil {
		return nil
	}
	rows, err := protocolListYybBriefsFn(userNumber)
	if err != nil {
		return nil
	}
	return rows
}

// ListProtocolYybBriefs 获取用户应用宝账号摘要
func ListProtocolYybBriefs(userNumber int) []ProtocolYybAccountBrief {
	return listYybBriefsForUser(userNumber)
}

// ValidateProtocolActivityInputs 协议活动提交前校验 inputs 中的协议引用
func ValidateProtocolActivityInputs(userNumber int, cfg *ActivityConfig, excludeRemarks string, inputs map[string]string) error {
	if cfg == nil || !cfg.IsProtocolActivity {
		return nil
	}
	fields := GetCkTemplateFields(cfg.CKTemplate)
	if len(fields) == 0 {
		return fmt.Errorf("活动 CK 模板未配置字段")
	}
	ref := strings.TrimSpace(inputs[fields[0]])
	if ref == "" {
		for _, v := range inputs {
			v = strings.TrimSpace(v)
			if IsYybOpenIDRef(v) || strings.HasPrefix(v, "wxid") {
				ref = v
				break
			}
		}
	}
	return ValidateProtocolRefForActivity(userNumber, cfg.ID, excludeRemarks, ref)
}
