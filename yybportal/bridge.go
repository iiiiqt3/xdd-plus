package yybportal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cdle/xdd/yyb"
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
		existing.YybAccountID = acc.ID
		existing.OpenID = openid
		existing.Nickname = nick
		existing.Status = st
		if e := db().Save(existing).Error; e != nil {
			return nil, e
		}
		return existing, nil
	}
	var byYyb PortalYybBinding
	if err := db().Where("user_number = ? AND yyb_account_id = ?", userNumber, acc.ID).First(&byYyb).Error; err == nil {
		byYyb.OpenID = openid
		byYyb.Nickname = nick
		byYyb.Status = st
		if e := db().Save(&byYyb).Error; e != nil {
			return nil, e
		}
		return &byYyb, nil
	}
	if isOpenIDBoundToOther(userNumber, openid) {
		return nil, fmt.Errorf("该微信账号已被其他用户绑定")
	}
	var count int64
	db().Model(&PortalYybBinding{}).Where("user_number = ?", userNumber).Count(&count)
	if int(count) >= getMaxAccountsPerUser() {
		return nil, fmt.Errorf("已达账号上限（%d 个）", getMaxAccountsPerUser())
	}
	b := PortalYybBinding{
		UserNumber:   userNumber,
		YybAccountID: acc.ID,
		OpenID:       openid,
		Nickname:     nick,
		Status:       st,
	}
	if err := db().Create(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func findUserBindingByOpenID(userNumber int, openid string) (*PortalYybBinding, bool) {
	openid = strings.TrimSpace(openid)
	if openid == "" {
		return nil, false
	}
	var row PortalYybBinding
	err := db().Where("user_number = ? AND LOWER(openid) = LOWER(?)", userNumber, openid).First(&row).Error
	if err != nil {
		return nil, false
	}
	return &row, true
}

func isUserBoundOpenID(userNumber int, openid string) bool {
	_, ok := findUserBindingByOpenID(userNumber, openid)
	return ok
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
	err := db().Where("user_number = ?", userNumber).Order("id asc").Find(&rows).Error
	return rows, err
}

func resolveBinding(userNumber int, ref string) (*PortalYybBinding, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("ref 不能为空")
	}
	var row PortalYybBinding
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		err = db().Where("user_number = ? AND (id = ? OR yyb_account_id = ?)", userNumber, id, id).First(&row).Error
		if err == nil {
			return &row, nil
		}
	}
	if err := db().Where("user_number = ? AND openid = ?", userNumber, ref).First(&row).Error; err == nil {
		return &row, nil
	}
	if err := db().Where("user_number = ? AND LOWER(openid) = LOWER(?)", userNumber, ref).First(&row).Error; err == nil {
		return &row, nil
	}
	if a, err := svc(); err == nil {
		if acc, err := a.GetAccountPublic(context.Background(), ref); err == nil && acc != nil {
			if err := db().Where("user_number = ? AND (openid = ? OR yyb_account_id = ?)",
				userNumber, acc.OpenID, acc.ID).First(&row).Error; err == nil {
				return &row, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到绑定账号")
}

func isOpenIDBoundToOther(userNumber int, openid string) bool {
	var count int64
	db().Model(&PortalYybBinding{}).Where("LOWER(openid) = LOWER(?) AND user_number <> ?", openid, userNumber).Count(&count)
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
	return view
}
