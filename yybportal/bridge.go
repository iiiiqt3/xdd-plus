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
	var count int64
	db().Model(&PortalYybBinding{}).Where("user_number = ?", userNumber).Count(&count)
	if int(count) >= getMaxAccountsPerUser() {
		return nil, fmt.Errorf("已达账号上限（%d 个）", getMaxAccountsPerUser())
	}
	nick := ""
	if acc.Nickname != nil {
		nick = *acc.Nickname
	}
	st := status
	if st == "" {
		st = "alive"
	}
	var existing PortalYybBinding
	err := db().Where("user_number = ? AND openid = ?", userNumber, acc.OpenID).First(&existing).Error
	if err == nil {
		existing.YybAccountID = acc.ID
		existing.Nickname = nick
		existing.Status = st
		if e := db().Save(&existing).Error; e != nil {
			return nil, e
		}
		return &existing, nil
	}
	b := PortalYybBinding{
		UserNumber:   userNumber,
		YybAccountID: acc.ID,
		OpenID:       acc.OpenID,
		Nickname:     nick,
		Status:       st,
	}
	if err := db().Create(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
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
	q := db().Where("user_number = ?", userNumber)
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		q = q.Where("id = ? OR yyb_account_id = ?", id, id)
	} else {
		q = q.Where("openid = ?", ref)
	}
	if err := q.First(&row).Error; err != nil {
		return nil, fmt.Errorf("未找到绑定账号")
	}
	return &row, nil
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
