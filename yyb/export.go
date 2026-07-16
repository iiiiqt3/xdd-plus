package yyb

import (
	"context"
	"net/http"

	"github.com/cdle/xdd/yyb/internal/protocol"
	"github.com/cdle/xdd/yyb/internal/store"
)

func toAccountPublic(in store.AccountPublic) AccountPublic {
	return AccountPublic{
		ID:            in.ID,
		OpenID:        in.OpenID,
		UIN:           in.UIN,
		Alias:         in.Alias,
		Nickname:      in.Nickname,
		Avatar:        in.Avatar,
		Status:        in.Status,
		LastCheckedAt: in.LastCheckedAt,
		CreatedAt:     in.CreatedAt,
		UpdatedAt:     in.UpdatedAt,
	}
}

func (s *Service) CreateQR(ctx context.Context, asBase64 bool) (*QRCreateResult, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	r, err := s.app.CreateQR(ctx, asBase64)
	if err != nil {
		return nil, err
	}
	return &QRCreateResult{
		SessionID: r.SessionID,
		Status:    r.Status,
		ImageURL:  r.ImageURL,
		ImageB64:  r.ImageB64,
	}, nil
}

func (s *Service) PollQR(ctx context.Context, sessionID string) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.PollQR(ctx, sessionID)
}

func (s *Service) ConfirmQR(ctx context.Context, sessionID string) (*AccountPublic, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	acc, err := s.app.ConfirmQR(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	out := toAccountPublic(*acc)
	return &out, nil
}

func (s *Service) ListAccounts(ctx context.Context) ([]AccountPublic, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	list, err := s.app.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AccountPublic, 0, len(list))
	for _, a := range list {
		out = append(out, toAccountPublic(a))
	}
	return out, nil
}

func (s *Service) GetAccountPublic(ctx context.Context, ref string) (*AccountPublic, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	acc, err := s.app.GetAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	out := acc.Public()
	return ptrAccount(toAccountPublic(out)), nil
}

func (s *Service) DeleteAccount(ctx context.Context, ref string) error {
	if s == nil || s.app == nil {
		return errNotReady
	}
	_, err := s.app.DeleteAccount(ctx, ref)
	return err
}

func (s *Service) RefreshAccount(ctx context.Context, ref string) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.RefreshAccount(ctx, ref)
}

func (s *Service) ResyncAccount(ctx context.Context, ref string) (*AccountPublic, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	acc, err := s.app.ResyncAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	return ptrAccount(toAccountPublic(*acc)), nil
}

func (s *Service) WxappGetCode(ctx context.Context, ref, appID string) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.WxappGetCode(ctx, ref, appID)
}

func (s *Service) WxappGetPhoneNumber(ctx context.Context, ref, appID string) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.WxappGetPhoneNumber(ctx, ref, appID)
}

func (s *Service) WxappOperateWXData(ctx context.Context, ref, appID string, payload map[string]any) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.WxappOperateWXData(ctx, ref, appID, payload)
}

func (s *Service) OfficialCGI(ctx context.Context, ref string, req protocol.OfficialCGIRequest) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.OfficialCGI(ctx, ref, req)
}

func (s *Service) TenPayCGI(ctx context.Context, ref string, req protocol.TenPayRequest) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.TenPayCGI(ctx, ref, req)
}

func (s *Service) RuntimeSession(ctx context.Context, ref, appID string, payload map[string]any) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.RuntimeSession(ctx, ref, appID, payload)
}

func (s *Service) UpdateStep(ctx context.Context, ref string, req protocol.StepRequest) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.UpdateStep(ctx, ref, req)
}

func (s *Service) ReportMotion(ctx context.Context, ref string, req protocol.StepRequest) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.ReportMotion(ctx, ref, req)
}

func (s *Service) GetBoundHardDevices(ctx context.Context, ref string, req protocol.StepRequest) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.GetBoundHardDevices(ctx, ref, req)
}

func (s *Service) GetWeRunData(ctx context.Context, ref, appID string) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.GetWeRunData(ctx, ref, appID)
}

func (s *Service) ServeAccountAvatar(w http.ResponseWriter, r *http.Request, ref string) error {
	if s == nil || s.app == nil {
		return errNotReady
	}
	return s.app.ServeAccountAvatar(w, r, ref)
}

func ptrAccount(a AccountPublic) *AccountPublic { return &a }
