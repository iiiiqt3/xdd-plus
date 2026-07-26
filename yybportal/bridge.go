package yybportal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cdle/xdd/yyb"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type pendingScan struct {
	UserNumber int
	SessionID  string
	Cost       int
	DeductCoin bool
	CreatedAt  time.Time
}

var (
	scanMu    sync.Mutex
	scanStore = map[string]*pendingScan{}
)

func putPendingScan(userNumber int, sessionID string, cost int, deduct bool) {
	scanMu.Lock()
	defer scanMu.Unlock()
	scanStore[sessionID] = &pendingScan{
		UserNumber: userNumber,
		SessionID:  sessionID,
		Cost:       cost,
		DeductCoin: deduct,
		CreatedAt:  time.Now(),
	}
}

func popPendingScan(sessionID string, userNumber int) (*pendingScan, error) {
	scanMu.Lock()
	defer scanMu.Unlock()
	p, ok := scanStore[sessionID]
	if !ok || p.UserNumber != userNumber {
		return nil, fmt.Errorf("扫码会话无效或已过期")
	}
	delete(scanStore, sessionID)
	return p, nil
}

func svc() (*yyb.Service, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝模块未就绪")
	}
	s := Service()
	if s == nil || !s.Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	return s, nil
}

const portalYybLoginValidDays = 30

func portalYybLoginTimes(b PortalYybBinding) (loginAt, expiresAt int64) {
	t := b.LoginAt
	if t.IsZero() {
		t = b.CreatedAt
	}
	if t.IsZero() {
		return 0, 0
	}
	loginAt = t.Unix()
	expiresAt = t.Add(portalYybLoginValidDays * 24 * time.Hour).Unix()
	return loginAt, expiresAt
}

func bindAccount(userNumber int, acc *yyb.AccountPublic, status string) (*PortalYybBinding, error) {
	if acc == nil {
		return nil, fmt.Errorf("账号数据为空")
	}
	openid := strings.TrimSpace(acc.OpenID)
	if openid == "" {
		return nil, fmt.Errorf("openid 为空")
	}
	nick := ""
	if acc.Nickname != nil {
		nick = *acc.Nickname
	}
	st := status
	if st == "" {
		st = "alive"
	}

	if existing, ok := findUserBindingByOpenID(userNumber, openid); ok {
		return saveBinding(existing, userNumber, acc.ID, openid, nick, st)
	}
	if existing, ok := findBindingIncludingDeleted(userNumber, openid); ok {
		return saveBinding(existing, userNumber, acc.ID, openid, nick, st)
	}
	if existing, ok := findBindingByYybIDUnscoped(userNumber, acc.ID); ok {
		return saveBinding(existing, userNumber, acc.ID, openid, nick, st)
	}
	if isOpenIDBoundToOther(userNumber, openid) {
		return nil, fmt.Errorf("该微信账号已被其他用户绑定")
	}
	var count int64
	db().Model(&PortalYybBinding{}).Where(&PortalYybBinding{UserNumber: userNumber}).Count(&count)
	if int(count) >= getMaxAccountsPerUser() {
		return nil, fmt.Errorf("已达账号上限（%d 个）", getMaxAccountsPerUser())
	}

	b := PortalYybBinding{
		UserNumber:   userNumber,
		YybAccountID: acc.ID,
		OpenID:       openid,
		Nickname:     nick,
		Status:       st,
		LoginAt:      time.Now(),
	}
	err := db().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_number"}, {Name: "open_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"yyb_account_id": acc.ID,
			"nickname":       nick,
			"status":         st,
			"login_at":       time.Now(),
			"deleted_at":     nil,
			"updated_at":     time.Now(),
		}),
	}).Create(&b).Error
	if err != nil {
		if isDuplicateKey(err) {
			if existing, ok := findBindingIncludingDeleted(userNumber, openid); ok {
				return saveBinding(existing, userNumber, acc.ID, openid, nick, st)
			}
		}
		return nil, err
	}
	if b.ID == 0 {
		if existing, ok := findBindingIncludingDeleted(userNumber, openid); ok {
			return existing, nil
		}
	}
	return &b, nil
}

func saveBinding(b *PortalYybBinding, userNumber int, yybAccountID int64, openid, nick, status string) (*PortalYybBinding, error) {
	b.UserNumber = userNumber
	b.YybAccountID = yybAccountID
	b.OpenID = openid
	b.Nickname = nick
	b.Status = status
	b.LoginAt = time.Now()
	b.DeletedAt = gorm.DeletedAt{}
	if err := db().Unscoped().Save(b).Error; err != nil {
		return nil, err
	}
	return b, nil
}

func findBindingIncludingDeleted(userNumber int, openid string) (*PortalYybBinding, bool) {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return nil, false
	}
	var rows []PortalYybBinding
	err := db().Unscoped().
		Where("user_number = ? AND LOWER(TRIM(open_id)) = LOWER(?)", userNumber, openid).
		Order("deleted_at asc, id desc").
		Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, false
	}
	for i := range rows {
		if !rows[i].DeletedAt.Valid {
			return &rows[i], true
		}
	}
	return &rows[0], true
}

func findBindingByYybIDUnscoped(userNumber int, yybAccountID int64) (*PortalYybBinding, bool) {
	var rows []PortalYybBinding
	err := db().Unscoped().
		Where(&PortalYybBinding{UserNumber: userNumber, YybAccountID: yybAccountID}).
		Order("deleted_at asc, id desc").
		Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, false
	}
	for i := range rows {
		if !rows[i].DeletedAt.Valid {
			return &rows[i], true
		}
	}
	return &rows[0], true
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate entry")
}

func findUserBindingByOpenID(userNumber int, openid string) (*PortalYybBinding, bool) {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return nil, false
	}
	var row PortalYybBinding
	err := db().Where("user_number = ? AND LOWER(TRIM(open_id)) = LOWER(?)", userNumber, openid).First(&row).Error
	if err != nil {
		return nil, false
	}
	return &row, true
}

func isUserBoundOpenID(userNumber int, openid string) bool {
	_, ok := findUserBindingByOpenID(userNumber, openid)
	return ok
}

// isKnownYybAccount 与 bindAccount 一致：库内已有 open_id 或 yyb_account_id 视为续登账号。
func isKnownYybAccount(userNumber int, openid string, yybAccountID int64) bool {
	openid = strings.TrimSpace(openid)
	if openid != "" {
		if _, ok := findBindingIncludingDeleted(userNumber, openid); ok {
			return true
		}
	}
	if yybAccountID > 0 {
		if _, ok := findBindingByYybIDUnscoped(userNumber, yybAccountID); ok {
			return true
		}
	}
	return false
}

func dedupeBindings(rows []PortalYybBinding) []PortalYybBinding {
	if len(rows) <= 1 {
		return rows
	}
	seen := make(map[string]int, len(rows))
	out := make([]PortalYybBinding, 0, len(rows))
	for _, b := range rows {
		key := strings.ToLower(strings.TrimSpace(b.OpenID))
		if key == "" {
			key = fmt.Sprintf("id:%d", b.ID)
		}
		if idx, ok := seen[key]; ok {
			if b.ID > out[idx].ID {
				out[idx] = b
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, b)
	}
	return out
}

func listBindings(userNumber int) ([]PortalYybBinding, error) {
	var rows []PortalYybBinding
	err := db().Where(&PortalYybBinding{UserNumber: userNumber}).Order("id asc").Find(&rows).Error
	return rows, err
}

func resolveBinding(userNumber int, ref string) (*PortalYybBinding, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("ref 不能为空")
	}
	var row PortalYybBinding
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		err = db().Where(&PortalYybBinding{UserNumber: userNumber}).Where("id = ? OR yyb_account_id = ?", id, id).First(&row).Error
		if err == nil {
			return &row, nil
		}
	}
	if err := db().Where(&PortalYybBinding{UserNumber: userNumber, OpenID: ref}).First(&row).Error; err == nil {
		return &row, nil
	}
	if err := db().Where("user_number = ? AND LOWER(open_id) = LOWER(?)", userNumber, ref).First(&row).Error; err == nil {
		return &row, nil
	}
	if a, err := svc(); err == nil {
		if acc, err := a.GetAccountPublic(context.Background(), ref); err == nil && acc != nil {
			if err := db().Where(&PortalYybBinding{UserNumber: userNumber, OpenID: acc.OpenID}).
				Or(&PortalYybBinding{UserNumber: userNumber, YybAccountID: acc.ID}).
				First(&row).Error; err == nil {
				return &row, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到绑定账号")
}

func isOpenIDBoundToOther(userNumber int, openid string) bool {
	var count int64
	db().Model(&PortalYybBinding{}).Where("LOWER(open_id) = LOWER(?) AND user_number <> ?", openid, userNumber).Count(&count)
	return count > 0
}

func toPortalView(ctx context.Context, b PortalYybBinding, s *yyb.Service) PortalAccountView {
	view := PortalAccountView{
		BindingID:    b.ID,
		YybAccountID: b.YybAccountID,
		OpenID:       b.OpenID,
		Nickname:     b.Nickname,
		Status:       b.Status,
		AvatarURL:    fmt.Sprintf("/api/portal/yyb/avatar?ref=%s", b.OpenID),
		CreatedAt:    b.CreatedAt.Unix(),
	}
	view.LoginAt, view.ExpiresAt = portalYybLoginTimes(b)
	if s == nil {
		return view
	}
	acc, err := s.GetAccountPublic(ctx, strconv.FormatInt(b.YybAccountID, 10))
	if err != nil || acc == nil {
		return view
	}
	if acc.Nickname != nil && *acc.Nickname != "" {
		view.Nickname = *acc.Nickname
	}
	if acc.UIN != nil {
		view.UIN = acc.UIN
	}
	if acc.Status != nil && *acc.Status != "" {
		view.Status = *acc.Status
	}
	view.LastChecked = acc.LastCheckedAt
	if code, name := s.AccountProxyRegion(ctx, strconv.FormatInt(b.YybAccountID, 10)); name != "" {
		view.ProxyRegionCode = code
		view.ProxyRegionName = name
	} else if b.OpenID != "" {
		if code, name := s.AccountProxyRegion(ctx, b.OpenID); name != "" {
			view.ProxyRegionCode = code
			view.ProxyRegionName = name
		}
	}
	return view
}
