package httpapi

import (
	"context"
	"net/http"

	"github.com/cdle/xdd/yyb/internal/store"
)

// QRCreateResult 创建扫码会话结果
type QRCreateResult struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	ImageURL  string `json:"image_url"`
	ImageB64  string `json:"image_base64,omitempty"`
}

// CreateQR 创建微信扫码会话
func (a *App) CreateQR(ctx context.Context, asBase64 bool) (*QRCreateResult, error) {
	w := &responseRecorder{}
	r, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/qr?as_base64="+boolStr(asBase64), nil)
	a.handleQRRoot(w, r)
	if w.status >= 400 {
		return nil, w.apiError()
	}
	var env apiEnvelope
	if err := w.decode(&env); err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]any)
	if !ok {
		return nil, errUnexpectedData
	}
	out := &QRCreateResult{
		SessionID: strVal(data["session_id"]),
		Status:    strVal(data["status"]),
		ImageURL:  strVal(data["image_url"]),
	}
	if v, ok := data["image_base64"].(string); ok && v != "" {
		out.ImageB64 = v
	}
	return out, nil
}

// PollQR 轮询扫码状态
func (a *App) PollQR(ctx context.Context, sessionID string) (map[string]any, error) {
	w := &responseRecorder{}
	r, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/qr/"+sessionID+"/poll", nil)
	a.handleQR(w, r)
	if w.status == 404 {
		return nil, errSessionNotFound
	}
	if w.status >= 400 {
		return nil, w.apiError()
	}
	var env apiEnvelope
	if err := w.decode(&env); err != nil {
		return nil, err
	}
	if m, ok := env.Data.(map[string]any); ok {
		return m, nil
	}
	return map[string]any{"status": env.Data}, nil
}

// ConfirmQR 确认扫码并保存账号
func (a *App) ConfirmQR(ctx context.Context, sessionID string) (*store.AccountPublic, error) {
	w := &responseRecorder{}
	r, _ := http.NewRequestWithContext(ctx, http.MethodPost, "/qr/"+sessionID+"/confirm", nil)
	a.handleQR(w, r)
	if w.status == 404 {
		return nil, errSessionNotFound
	}
	if w.status >= 400 {
		return nil, w.apiError()
	}
	var env apiEnvelope
	if err := w.decode(&env); err != nil {
		return nil, err
	}
	b, _ := jsonMarshal(env.Data)
	var acc store.AccountPublic
	if err := jsonUnmarshal(b, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

// ListAccounts 全部账号
func (a *App) ListAccounts(ctx context.Context) ([]store.AccountPublic, error) {
	accounts, err := a.db.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.AccountPublic, 0, len(accounts))
	for _, acc := range accounts {
		out = append(out, acc.Public())
	}
	return out, nil
}

// GetAccount 按 ref 查账号
func (a *App) GetAccount(ctx context.Context, ref string) (*store.WechatAccount, error) {
	return a.db.ResolveAccount(ctx, ref)
}

// DeleteAccount 删除账号
func (a *App) DeleteAccount(ctx context.Context, ref string) (*store.WechatAccount, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	if err := a.db.DeleteAccount(ctx, acc.ID); err != nil {
		return nil, err
	}
	return acc, nil
}

// RefreshAccount 刷新存活
func (a *App) RefreshAccount(ctx context.Context, ref string) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	status := a.refreshLiveness(ctx, acc)
	if updated, err := a.db.GetAccount(ctx, acc.ID); err == nil {
		acc = updated
	}
	a.ensureAccountUIN(ctx, acc)
	if updated, err := a.db.GetAccount(ctx, acc.ID); err == nil {
		acc = updated
	}
	return refreshOut(acc, status), nil
}

// ResyncAccount 同步资料
func (a *App) ResyncAccount(ctx context.Context, ref string) (*store.AccountPublic, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	updated, err := a.resyncProfile(ctx, acc)
	if err != nil {
		return nil, err
	}
	pub := updated.Public()
	return &pub, nil
}

// WxappGetCode 获取小程序 code
func (a *App) WxappGetCode(ctx context.Context, ref, appID string) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	return a.invokeWXApp(ctx, acc, appID, nil, a.invokeGetCode)
}

// WxappGetPhoneNumber 获取手机号
func (a *App) WxappGetPhoneNumber(ctx context.Context, ref, appID string) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	return a.invokeWXApp(ctx, acc, appID, nil, a.invokeGetPhoneNumber)
}

// WxappOperateWXData 云函数
func (a *App) WxappOperateWXData(ctx context.Context, ref, appID string, payload map[string]any) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	return a.invokeWXApp(ctx, acc, appID, payload, a.invokeOperateWXData)
}

// ServeAccountAvatar 输出头像到 ResponseWriter
func (a *App) ServeAccountAvatar(w http.ResponseWriter, r *http.Request, ref string) error {
	acc, err := a.db.ResolveAccount(r.Context(), ref)
	if err != nil {
		return err
	}
	a.serveAvatar(w, r, acc)
	return nil
}

// DB 暴露 store（仅 yybportal 绑定用）
func (a *App) DB() *store.DB {
	return a.db
}
