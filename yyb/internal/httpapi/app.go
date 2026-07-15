package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"gorm.io/gorm"

	"github.com/cdle/xdd/yyb/internal/protocol"
	"github.com/cdle/xdd/yyb/internal/qr"
	"github.com/cdle/xdd/yyb/internal/store"
)

type Config struct {
	ResourceRoot   string
	DBFilename     string
	GormDB         *gorm.DB
	TCPProxy       string
	Proxy51Enabled bool
	SessionTTL     time.Duration
	RequestTimeout time.Duration
	AvatarTimeout  time.Duration
	ScanTimeout    time.Duration
	QRSessionTTL   time.Duration
}

type App struct {
	cfg       Config
	resources resources
	db        *store.DB
	pool      *protocol.Pool
	qr        *qr.Client

	accountTCPProxy        AccountTCPProxyResolver
	accountProxyForceRefresh AccountProxyForceRefresher

	mu         sync.Mutex
	qrSessions map[string]*qr.Session
}

// AccountTCPProxyResolver 按账号 credentials 解析 SOCKS5（51 代理账号级续提）
type AccountTCPProxyResolver func(ctx context.Context, acc *store.WechatAccount) (proxy string, updateCred bool, cred map[string]any)

// AccountProxyForceRefresher 强制重提账号同地区短效代理
type AccountProxyForceRefresher func(credentials map[string]any) (proxy string, updated map[string]any, err error)

var swaggerDocsHandler = httpSwagger.Handler(
	httpSwagger.URL("/openapi.json"),
	httpSwagger.DocExpansion("list"),
	httpSwagger.DeepLinking(true),
	httpSwagger.DefaultModelsExpandDepth(httpSwagger.ShowModel),
)

func NewApp(cfg Config) (*App, error) {
	if cfg.ResourceRoot == "" {
		cfg.ResourceRoot = filepath.Join(".", "resource")
	}
	if cfg.DBFilename == "" {
		cfg.DBFilename = DefaultDBFilename
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 8 * time.Second
	}
	if cfg.AvatarTimeout == 0 {
		cfg.AvatarTimeout = 10 * time.Second
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 30 * time.Minute
	}
	if cfg.QRSessionTTL == 0 {
		cfg.QRSessionTTL = 5 * time.Minute
	}
	res, err := ensureResources(cfg.ResourceRoot)
	if err != nil {
		return nil, err
	}
	var db *store.DB
	if cfg.GormDB != nil {
		db, err = store.OpenGORM(cfg.GormDB)
	} else {
		dbPath, err := prepareDBPath(res.DB, cfg.DBFilename)
		if err != nil {
			return nil, err
		}
		db, err = store.Open(dbPath)
	}
	if err != nil {
		return nil, err
	}
	poolCfg := protocol.DefaultConfig()
	poolCfg.SessionTTL = cfg.SessionTTL
	poolCfg.ShortlinkTimeout = cfg.RequestTimeout
	poolCfg.TCPProxy = cfg.TCPProxy
	if cfg.Proxy51Enabled {
		poolCfg.TCPProxy = ""
		poolCfg.TCPProxyFallbackDirect = false
	}
	pool := protocol.NewPool(poolCfg, db)
	return &App{
		cfg:        cfg,
		resources:  res,
		db:         db,
		pool:       pool,
		qr:         qr.NewClient(cfg.RequestTimeout),
		qrSessions: map[string]*qr.Session{},
	}, nil
}

func (a *App) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// SetTCPProxy 运行时切换默认 TCP 代理（51 代理扫码等）
func (a *App) SetTCPProxy(proxy string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.TCPProxy = strings.TrimSpace(proxy)
	if a.pool != nil {
		a.pool.SetTCPProxy(proxy)
	}
}

func (a *App) Handler() http.Handler {
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.Any("/", gin.WrapF(a.handleIndex))
	router.Any("/scan", gin.WrapF(a.handleScan))
	router.Any("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/docs/index.html")
	})
	router.Any("/docs/*path", gin.WrapF(a.handleDocs))
	router.Any("/openapi.json", gin.WrapF(a.handleOpenAPI))
	router.Any("/health", func(c *gin.Context) {
		writeJSON(c.Writer, http.StatusOK, gin.H{"ok": true})
	})
	router.StaticFS("/static", http.Dir(a.resources.Static))
	router.Any("/qr", gin.WrapF(a.handleQRRoot))
	router.Any("/qr/*path", gin.WrapF(a.handleQR))
	router.Any("/accounts", gin.WrapF(a.handleAccountsRoot))
	router.Any("/accounts/avatar", gin.WrapF(a.handleAccountAvatar))
	router.Any("/accounts/refresh", gin.WrapF(a.handleAccountRefresh))
	router.Any("/accounts/resync", gin.WrapF(a.handleAccountResync))
	router.Any("/wxapp/getCode", gin.WrapF(a.handleGetCode))
	router.Any("/wxapp/getPhoneNumber", gin.WrapF(a.handleGetPhoneNumber))
	router.Any("/wxapp/operateWxData", gin.WrapF(a.handleOperateWXData))
	router.NoRoute(func(c *gin.Context) {
		writeError(c.Writer, http.StatusNotFound, "not found")
	})

	return router
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	serveFileOrText(w, r, filepath.Join(a.resources.Templates, "index.html"), fallbackIndexHTML)
}

func (a *App) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	serveFileOrText(w, r, filepath.Join(a.resources.Templates, "scan.html"), fallbackScanHTML)
}

func (a *App) handleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if r.URL.Path == "/docs/" {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
		return
	}
	swaggerDocsHandler.ServeHTTP(w, r)
}

func (a *App) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeRawJSON(w, http.StatusOK, openAPISpec)
}

func (a *App) handleQRRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/qr" {
		writeError(w, http.StatusNotFound, "qr session not found")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	a.pruneQR()
	ctx, cancel := context.WithTimeout(r.Context(), a.cfg.RequestTimeout+35*time.Second)
	defer cancel()
	img, err := a.qr.GetQRCodeImage(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	a.mu.Lock()
	a.qrSessions[img.Session.ID] = img.Session
	keep := make(map[string]bool, len(a.qrSessions))
	for sid := range a.qrSessions {
		keep[sid] = true
	}
	a.mu.Unlock()
	path := a.resources.qrPath(img.Session.ID)
	_ = os.WriteFile(path, img.ImageBytes, 0o644)
	a.cleanupQR(keep)
	out := map[string]any{
		"session_id": img.Session.ID,
		"status":     img.Session.Status,
		"image_url":  "/qr/" + img.Session.ID + "/image",
	}
	if r.URL.Query().Get("as_base64") == "true" {
		out["image_base64"] = qr.DataURIJPEG(img.ImageBytes)
	} else {
		out["image_base64"] = nil
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) handleQR(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/qr/"), "/")
	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "qr session not found")
		return
	}
	sessionID, action := parts[0], parts[1]
	switch action {
	case "image":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		path := a.resources.qrPath(sessionID)
		if _, err := os.Stat(path); err != nil {
			writeError(w, http.StatusNotFound, "qr session not found")
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		http.ServeFile(w, r, path)
	case "poll":
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		sess := a.getQRSession(sessionID)
		if sess == nil {
			writeError(w, http.StatusNotFound, "qr session not found")
			return
		}
		result, err := a.qr.PollQRCode(r.Context(), sess)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		if terminalQR(result.Status) {
			a.dropQRSession(sessionID)
		}
		writeJSON(w, http.StatusOK, result)
	case "confirm":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		sess := a.getQRSession(sessionID)
		if sess == nil {
			writeError(w, http.StatusNotFound, "qr session not found")
			return
		}
		result, err := a.qr.GetLoginBuffer(r.Context(), sess)
		if err != nil {
			writeError(w, http.StatusConflict, "buffer not ready: "+err.Error())
			return
		}
		var userInfo map[string]any
		if ui, err := a.qr.LoginBuffers().FetchUserInfo(r.Context(), result.Credentials); err == nil {
			userInfo = ui
		}
		acc, err := a.storeFromScan(r.Context(), result.LoginBuffer, result.Credentials, userInfo)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.ensureAccountUIN(r.Context(), acc)
		if updated, err := a.db.GetAccount(r.Context(), acc.ID); err == nil {
			acc = updated
		}
		a.dropQRSession(sessionID)
		writeJSON(w, http.StatusOK, acc.Public())
	default:
		writeError(w, http.StatusNotFound, "qr session not found")
	}
}

func (a *App) handleAccountsRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/accounts" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		accounts, err := a.db.ListAccounts(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]store.AccountPublic, 0, len(accounts))
		for _, acc := range accounts {
			out = append(out, acc.Public())
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodDelete:
		acc, ok := a.resolveAccountFromQuery(w, r)
		if !ok {
			return
		}
		if err := a.db.DeleteAccount(r.Context(), acc.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": acc.ID, "openid": acc.OpenID})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *App) handleAccountAvatar(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/accounts/avatar" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	acc, ok := a.resolveAccountFromQuery(w, r)
	if !ok {
		return
	}
	a.serveAvatar(w, r, acc)
}

func (a *App) handleAccountRefresh(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/accounts/refresh" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body accountRefIn
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Ref == "" {
		a.refreshAll(w, r)
		return
	}
	acc, ok := a.resolveAccountRef(w, r, body.Ref)
	if !ok {
		return
	}
	status := a.refreshLiveness(r.Context(), acc)
	writeJSON(w, http.StatusOK, refreshOut(acc, status))
}

func (a *App) handleAccountResync(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/accounts/resync" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body accountRefIn
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Ref == "" {
		a.resyncAll(w, r)
		return
	}
	acc, ok := a.resolveAccountRef(w, r, body.Ref)
	if !ok {
		return
	}
	updated, err := a.resyncProfile(r.Context(), acc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated.Public())
}

func (a *App) handleGetCode(w http.ResponseWriter, r *http.Request) {
	if !acceptWXAppRoute(w, r, "/wxapp/getCode") {
		return
	}
	a.callWXApp(w, r, false, a.invokeGetCode)
}

func (a *App) handleGetPhoneNumber(w http.ResponseWriter, r *http.Request) {
	if !acceptWXAppRoute(w, r, "/wxapp/getPhoneNumber") {
		return
	}
	a.callWXApp(w, r, false, a.invokeGetPhoneNumber)
}

func (a *App) handleOperateWXData(w http.ResponseWriter, r *http.Request) {
	if !acceptWXAppRoute(w, r, "/wxapp/operateWxData") {
		return
	}
	a.callWXApp(w, r, true, a.invokeOperateWXData)
}

func acceptWXAppRoute(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.URL.Path != path {
		writeError(w, http.StatusNotFound, "not found")
		return false
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return true
}

type accountRefIn struct {
	Ref string `json:"ref"`
}

type wxappRequest struct {
	Ref     string         `json:"ref"`
	AppID   string         `json:"app_id"`
	Payload map[string]any `json:"payload"`
}

type wxappCall func(ctx context.Context, acc *store.WechatAccount, appID string, payload map[string]any, tcpProxy string) (map[string]any, error)

func (a *App) callWXApp(w http.ResponseWriter, r *http.Request, requirePayload bool, call wxappCall) {
	var body wxappRequest
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if body.Ref == "" {
		writeError(w, http.StatusBadRequest, "ref is required")
		return
	}
	if body.AppID == "" {
		writeError(w, http.StatusBadRequest, "app_id is required")
		return
	}
	if requirePayload && body.Payload == nil {
		writeError(w, http.StatusBadRequest, "payload is required")
		return
	}
	acc, ok := a.resolveAccountRef(w, r, body.Ref)
	if !ok {
		return
	}
	result, err := a.invokeWXApp(r.Context(), acc, body.AppID, body.Payload, call)
	if err != nil {
		var expired accountExpiredError
		switch {
		case errors.As(err, &expired):
			writeError(w, http.StatusConflict, "account login_buffer expired (refresh failed); re-scan required")
		default:
			writeError(w, http.StatusBadGateway, "call failed: "+err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"openid": acc.OpenID, "result": result})
}

func decodeOptionalJSON(r *http.Request, dst any) error {
	err := json.NewDecoder(r.Body).Decode(dst)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func (a *App) resolveAccountFromQuery(w http.ResponseWriter, r *http.Request) (*store.WechatAccount, bool) {
	ref := strings.TrimSpace(r.URL.Query().Get("ref"))
	if ref == "" {
		writeError(w, http.StatusBadRequest, "ref query param is required")
		return nil, false
	}
	return a.resolveAccountRef(w, r, ref)
}

func (a *App) resolveAccountRef(w http.ResponseWriter, r *http.Request, ref string) (*store.WechatAccount, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		writeError(w, http.StatusBadRequest, "ref is required")
		return nil, false
	}
	acc, err := a.db.ResolveAccount(r.Context(), ref)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "account not found: "+ref)
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return nil, false
	}
	return acc, true
}

func (a *App) refreshAll(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.db.ListAccounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]map[string]any, 0, len(accounts))
	for _, acc := range accounts {
		out = append(out, refreshOut(acc, a.refreshLiveness(r.Context(), acc)))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) resyncAll(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.db.ListAccounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]store.AccountPublic, 0, len(accounts))
	for _, acc := range accounts {
		updated, err := a.resyncProfile(r.Context(), acc)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, updated.Public())
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *App) serveAvatar(w http.ResponseWriter, r *http.Request, acc *store.WechatAccount) {
	if acc.Avatar != nil && *acc.Avatar != "" {
		if _, err := os.Stat(*acc.Avatar); err == nil {
			w.Header().Set("Content-Type", "image/jpeg")
			http.ServeFile(w, r, *acc.Avatar)
			return
		}
		if strings.HasPrefix(*acc.Avatar, "http://") || strings.HasPrefix(*acc.Avatar, "https://") {
			http.Redirect(w, r, *acc.Avatar, http.StatusFound)
			return
		}
	}
	writeError(w, http.StatusNotFound, "no avatar")
}

func (a *App) SetAccountTCPProxyResolver(r AccountTCPProxyResolver) {
	a.accountTCPProxy = r
}

func (a *App) SetAccountProxyForceRefresher(r AccountProxyForceRefresher) {
	a.accountProxyForceRefresh = r
}

func (a *App) tcpProxyForAccount(ctx context.Context, acc *store.WechatAccount) string {
	if a.accountTCPProxy != nil && acc != nil {
		proxy, update, cred := a.accountTCPProxy(ctx, acc)
		if update && cred != nil {
			_ = a.db.SetAccountCredential(ctx, acc.ID, acc.LoginBuffer, cred)
			acc.Credentials = cred
			if strings.TrimSpace(proxy) != "" {
				_ = a.db.InvalidateSession(ctx, acc.ID, proxy)
			}
		}
		if a.cfg.Proxy51Enabled {
			return strings.TrimSpace(proxy)
		}
	}
	return a.cfg.TCPProxy
}

func (a *App) StoreScanAccount(ctx context.Context, loginBuffer string, creds protocol.LoginBufferCredentials, proxyMeta map[string]interface{}) (*store.AccountPublic, error) {
	var userInfo map[string]any
	if ui, err := a.qr.LoginBuffers().FetchUserInfo(ctx, creds); err == nil {
		userInfo = ui
	}
	acc, err := a.storeFromScanWithMeta(ctx, loginBuffer, creds, userInfo, proxyMeta)
	if err != nil {
		return nil, err
	}
	a.ensureAccountUIN(ctx, acc)
	if updated, err := a.db.GetAccount(ctx, acc.ID); err == nil {
		acc = updated
	}
	pub := acc.Public()
	return &pub, nil
}

func (a *App) storeFromScanWithMeta(ctx context.Context, loginBuffer string, creds protocol.LoginBufferCredentials, userInfo map[string]any, proxyMeta map[string]interface{}) (*store.WechatAccount, error) {
	openid := creds.OpenID
	nick := pickNickname(userInfo, creds.Nickname)
	avatar := a.resolveAvatar(ctx, openid, userInfo)
	status := "alive"
	credMap := creds.ToMap()
	applyProxyMeta(credMap, proxyMeta)
	return a.db.UpsertAccount(ctx, openid, loginBuffer, stringPtrMaybe(nick), stringPtrMaybe(nick), stringPtrMaybe(avatar), userInfo, credMap, &status)
}

func applyProxyMeta(dst map[string]any, meta map[string]interface{}) {
	if dst == nil || meta == nil {
		return
	}
	for k, v := range meta {
		dst[k] = v
	}
}

func (a *App) storeFromScan(ctx context.Context, loginBuffer string, creds protocol.LoginBufferCredentials, userInfo map[string]any) (*store.WechatAccount, error) {
	return a.storeFromScanWithMeta(ctx, loginBuffer, creds, userInfo, nil)
}

func (a *App) ensureAccountUIN(ctx context.Context, acc *store.WechatAccount) {
	if acc == nil || strings.TrimSpace(acc.LoginBuffer) == "" {
		return
	}
	if acc.UIN != nil && *acc.UIN > 0 {
		return
	}
	loginCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_ = a.pool.EnsureSession(loginCtx, acc.LoginBuffer, acc.ID, a.tcpProxyForAccount(ctx, acc))
}

func (a *App) forceRefreshAccountProxy(ctx context.Context, acc *store.WechatAccount) (string, error) {
	if acc == nil || acc.Credentials == nil {
		return "", fmt.Errorf("应用宝账号代理信息为空")
	}
	if !a.cfg.Proxy51Enabled || !accountUsesSavedProxy(acc.Credentials) {
		return a.tcpProxyForAccount(ctx, acc), nil
	}
	if a.accountProxyForceRefresh == nil {
		return a.tcpProxyForAccount(ctx, acc), nil
	}
	oldProxy := strings.TrimSpace(stringFromAny(acc.Credentials["yyb_proxy_url"]))
	proxy, cred, err := a.accountProxyForceRefresh(acc.Credentials)
	if err != nil {
		return "", err
	}
	if cred != nil {
		_ = a.db.SetAccountCredential(ctx, acc.ID, acc.LoginBuffer, cred)
		acc.Credentials = cred
	}
	if oldProxy != "" {
		_ = a.db.InvalidateSession(ctx, acc.ID, oldProxy)
	}
	if strings.TrimSpace(proxy) != "" {
		_ = a.db.InvalidateSession(ctx, acc.ID, proxy)
	}
	return strings.TrimSpace(proxy), nil
}

func (a *App) recordLivenessCheck(ctx context.Context, acc *store.WechatAccount, kind, reason string) {
	if acc == nil {
		return
	}
	if acc.Credentials == nil {
		acc.Credentials = map[string]any{}
	}
	acc.Credentials["yyb_last_check_at"] = time.Now().Unix()
	acc.Credentials["yyb_last_check_type"] = strings.TrimSpace(kind)
	acc.Credentials["yyb_last_check_reason"] = truncateString(strings.TrimSpace(reason), 500)
	_ = a.db.SetAccountCredential(ctx, acc.ID, acc.LoginBuffer, acc.Credentials)
}

func (a *App) refreshLivenessCore(ctx context.Context, acc *store.WechatAccount) (*store.WechatAccount, error) {
	if acc == nil {
		return nil, fmt.Errorf("nil YYB account")
	}
	if acc.Credentials == nil {
		_ = a.db.SetAccountStatus(ctx, acc.ID, "unknown")
		a.recordLivenessCheck(ctx, acc, "unknown", "应用宝账号 credentials 缺失")
		return nil, fmt.Errorf("YYB account credentials missing")
	}
	creds := protocol.CredentialsFromMap(acc.Credentials)
	maxAttempts := 1
	if a.cfg.Proxy51Enabled && accountUsesSavedProxy(acc.Credentials) {
		maxAttempts = accountProxyRotateLimit
	}
	var result protocol.LoginBufferResult
	var err error
	proxy := a.tcpProxyForAccount(ctx, acc)
	fallbackDirect := strings.TrimSpace(proxy) == "" && !a.cfg.Proxy51Enabled
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client := protocol.NewLoginBufferClientWithProxy(a.cfg.RequestTimeout+25*time.Second, proxy, fallbackDirect)
		result, err = client.RefreshLoginBuffer(ctx, creds)
		if err == nil {
			break
		}
		if !isProxyOrNetworkError(err) || attempt >= maxAttempts {
			break
		}
		if freshProxy, proxyErr := a.forceRefreshAccountProxy(ctx, acc); proxyErr == nil && strings.TrimSpace(freshProxy) != "" {
			proxy = freshProxy
		} else if proxyErr != nil {
			err = fmt.Errorf("%v；重新提取代理失败：%v", err, proxyErr)
		}
	}
	if err != nil {
		if isProxyOrNetworkError(err) {
			a.recordLivenessCheck(ctx, acc, "proxy_error", fmt.Sprintf("代理/网络异常，已重试 %d 次：%v", maxAttempts, err))
			_ = a.db.SetAccountStatus(ctx, acc.ID, "unknown")
			return nil, fmt.Errorf("应用宝检测代理/网络异常，已重试 %d 次：%w", maxAttempts, err)
		}
		a.recordLivenessCheck(ctx, acc, "expired", fmt.Sprintf("登录态刷新失败：%v", err))
		_ = a.db.SetAccountStatus(ctx, acc.ID, "expired")
		return nil, err
	}
	credMap := result.Credentials.ToMap()
	applyProxyMeta(credMap, proxyMetaFromCredentials(acc.Credentials))
	credMap["yyb_last_check_at"] = time.Now().Unix()
	credMap["yyb_last_check_type"] = "success"
	credMap["yyb_last_check_reason"] = "刷新成功"
	if err := a.db.SetAccountCredential(ctx, acc.ID, result.LoginBuffer, credMap); err != nil {
		return nil, err
	}
	_ = a.db.SetAccountStatus(ctx, acc.ID, "alive")
	if avatar := a.resolveAvatar(ctx, acc.OpenID, acc.UserInfo); avatar != "" {
		_ = a.db.SetAccountProfile(ctx, acc.ID, acc.Nickname, &avatar, acc.UserInfo)
	}
	fresh, err := a.db.GetAccount(ctx, acc.ID)
	if err != nil {
		return nil, err
	}
	return fresh, nil
}

func (a *App) refreshLiveness(ctx context.Context, acc *store.WechatAccount) string {
	_, err := a.refreshLivenessCore(ctx, acc)
	if err != nil {
		if isProxyOrNetworkError(err) {
			return "unknown"
		}
		return "expired"
	}
	return "alive"
}

func (a *App) resyncProfile(ctx context.Context, acc *store.WechatAccount) (*store.WechatAccount, error) {
	if acc.Credentials != nil {
		proxy := a.tcpProxyForAccount(ctx, acc)
		fallbackDirect := strings.TrimSpace(proxy) == "" && !a.cfg.Proxy51Enabled
		client := protocol.NewLoginBufferClientWithProxy(a.cfg.RequestTimeout+25*time.Second, proxy, fallbackDirect)
		creds := protocol.CredentialsFromMap(acc.Credentials)
		if ui, err := client.FetchUserInfo(ctx, creds); err == nil && ui != nil {
			acc.UserInfo = ui
		}
	}
	nick := pickNickname(acc.UserInfo, deref(acc.Nickname))
	avatar := a.resolveAvatar(ctx, acc.OpenID, acc.UserInfo)
	if avatar == "" {
		avatar = deref(acc.Avatar)
	}
	if err := a.db.SetAccountProfile(ctx, acc.ID, stringPtrMaybe(nick), stringPtrMaybe(avatar), acc.UserInfo); err != nil {
		return nil, err
	}
	return a.db.GetAccount(ctx, acc.ID)
}

type accountExpiredError struct{ openid string }

func (e accountExpiredError) Error() string { return "account expired: " + e.openid }

func (a *App) businessUsesProxy(acc *store.WechatAccount) bool {
	return a.cfg.Proxy51Enabled && accountUsesSavedProxy(acc.Credentials)
}

func (a *App) businessProxyForAccount(ctx context.Context, acc *store.WechatAccount) string {
	if !a.businessUsesProxy(acc) {
		return ""
	}
	return a.tcpProxyForAccount(ctx, acc)
}

func (a *App) runBusinessWithProxyRetry(ctx context.Context, acc *store.WechatAccount, op func(*store.WechatAccount, string) (map[string]any, error)) (map[string]any, *store.WechatAccount, error) {
	if acc == nil {
		return nil, acc, fmt.Errorf("应用宝账号为空")
	}
	current := acc
	maxAttempts := 1
	if a.businessUsesProxy(current) {
		maxAttempts = accountProxyRotateLimit
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		tcpProxy := a.businessProxyForAccount(ctx, current)
		result, err := op(current, tcpProxy)
		if err == nil {
			return result, current, nil
		}
		lastErr = err
		if isProxyOrNetworkError(err) && a.businessUsesProxy(current) {
			if attempt < maxAttempts {
				if _, proxyErr := a.forceRefreshAccountProxy(ctx, current); proxyErr != nil {
					lastErr = fmt.Errorf("%v；重新提取代理失败：%v", err, proxyErr)
				}
				continue
			}
			break
		}
		fresh, refreshErr := a.refreshLivenessCore(ctx, current)
		if refreshErr != nil {
			lastErr = refreshErr
			if isProxyOrNetworkError(refreshErr) && a.businessUsesProxy(current) && attempt < maxAttempts {
				if _, proxyErr := a.forceRefreshAccountProxy(ctx, current); proxyErr != nil {
					lastErr = fmt.Errorf("%v；重新提取代理失败：%v", refreshErr, proxyErr)
				}
				continue
			}
			if !isProxyOrNetworkError(refreshErr) {
				return nil, current, accountExpiredError{openid: current.OpenID}
			}
			return nil, current, refreshErr
		}
		current = fresh
		tcpProxy = a.businessProxyForAccount(ctx, current)
		result, err = op(current, tcpProxy)
		if err == nil {
			return result, current, nil
		}
		lastErr = err
		if isProxyOrNetworkError(err) && a.businessUsesProxy(current) {
			if attempt < maxAttempts {
				if _, proxyErr := a.forceRefreshAccountProxy(ctx, current); proxyErr != nil {
					lastErr = fmt.Errorf("%v；重新提取代理失败：%v", err, proxyErr)
				}
				continue
			}
			break
		}
		return nil, current, lastErr
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("应用宝业务请求失败")
	}
	if isProxyOrNetworkError(lastErr) && a.businessUsesProxy(current) {
		return nil, current, fmt.Errorf("应用宝业务请求代理/网络异常，已重试 %d 次：%w", maxAttempts, lastErr)
	}
	return nil, current, lastErr
}

func (a *App) invokeWXApp(ctx context.Context, acc *store.WechatAccount, appID string, payload map[string]any, call wxappCall) (map[string]any, error) {
	result, _, err := a.runBusinessWithProxyRetry(ctx, acc, func(current *store.WechatAccount, tcpProxy string) (map[string]any, error) {
		return call(ctx, current, appID, payload, tcpProxy)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *App) invokeGetCode(ctx context.Context, acc *store.WechatAccount, appID string, _ map[string]any, tcpProxy string) (map[string]any, error) {
	return a.pool.GetCode(ctx, acc.LoginBuffer, appID, acc.ID, tcpProxy)
}

func (a *App) invokeGetPhoneNumber(ctx context.Context, acc *store.WechatAccount, appID string, _ map[string]any, tcpProxy string) (map[string]any, error) {
	return a.pool.GetPhoneNumber(ctx, acc.LoginBuffer, appID, acc.ID, tcpProxy)
}

func (a *App) invokeOperateWXData(ctx context.Context, acc *store.WechatAccount, appID string, payload map[string]any, tcpProxy string) (map[string]any, error) {
	return a.pool.OperateWXData(ctx, acc.LoginBuffer, appID, payload, acc.ID, tcpProxy)
}

func refreshOut(acc *store.WechatAccount, status string) map[string]any {
	return map[string]any{"id": acc.ID, "openid": acc.OpenID, "uin": acc.UIN, "nickname": acc.Nickname, "status": status}
}

func pickNickname(userInfo map[string]any, fallback string) string {
	if s := stringFromAny(userInfo["nick_name"]); s != "" {
		return s
	}
	return fallback
}

func pickAvatarURL(userInfo map[string]any) string {
	for _, k := range []string{"head_img_url", "head_url", "headimgurl", "avatar"} {
		if s := stringFromAny(userInfo[k]); s != "" {
			return s
		}
	}
	return ""
}

func (a *App) resolveAvatar(ctx context.Context, openid string, userInfo map[string]any) string {
	u := pickAvatarURL(userInfo)
	if u == "" {
		return ""
	}
	dest := a.resources.avatarPath(openid)
	if downloadAvatar(ctx, u, dest, a.cfg.AvatarTimeout) {
		return dest
	}
	return u
}

func downloadAvatar(ctx context.Context, url, dest string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil || resp.StatusCode != 200 || !looksLikeImage(data) {
		return false
	}
	_ = os.MkdirAll(filepath.Dir(dest), 0o755)
	return os.WriteFile(dest, data, 0o644) == nil
}

func looksLikeImage(data []byte) bool {
	if len(data) < 64 {
		return false
	}
	magics := [][]byte{{0xff, 0xd8, 0xff}, {0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("GIF87a"), []byte("GIF89a")}
	for _, m := range magics {
		if strings.HasPrefix(string(data), string(m)) {
			return true
		}
	}
	return false
}

func (a *App) getQRSession(id string) *qr.Session {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.qrSessions[id]
}

func (a *App) dropQRSession(id string) {
	a.mu.Lock()
	delete(a.qrSessions, id)
	a.mu.Unlock()
	_ = os.Remove(a.resources.qrPath(id))
}

func (a *App) pruneQR() {
	a.mu.Lock()
	var drop []string
	for sid, sess := range a.qrSessions {
		if sess.Age() > a.cfg.QRSessionTTL {
			drop = append(drop, sid)
		}
	}
	for _, sid := range drop {
		delete(a.qrSessions, sid)
	}
	a.mu.Unlock()
	for _, sid := range drop {
		_ = os.Remove(a.resources.qrPath(sid))
	}
}

func (a *App) cleanupQR(keep map[string]bool) {
	files, _ := filepath.Glob(filepath.Join(a.resources.QR, "*.jpg"))
	for _, f := range files {
		sid := strings.TrimSuffix(filepath.Base(f), ".jpg")
		if !keep[sid] {
			_ = os.Remove(f)
		}
	}
}

func terminalQR(status string) bool {
	return status == "expired" || status == "cancelled" || status == "unknown"
}

type apiEnvelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	writeRawJSON(w, status, apiEnvelope{
		Code: 0,
		Msg:  "success",
		Data: v,
	})
}

func writeRawJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeRawJSON(w, status, apiEnvelope{
		Code: status,
		Msg:  detail,
		Data: nil,
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func serveFileOrText(w http.ResponseWriter, r *http.Request, path, fallback string) {
	if _, err := os.Stat(path); err == nil {
		http.ServeFile(w, r, path)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(fallback))
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func stringPtrMaybe(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

const accountProxyRotateLimit = 5

func accountUsesSavedProxy(cred map[string]any) bool {
	if cred == nil {
		return false
	}
	if v, ok := cred["yyb_proxy_bypass"].(bool); ok && v {
		return false
	}
	switch v := cred["yyb_proxy_enabled"].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
	default:
		return false
	}
}

func proxyMetaFromCredentials(cred map[string]any) map[string]interface{} {
	if cred == nil {
		return nil
	}
	keys := []string{
		"yyb_proxy_enabled", "yyb_proxy_bypass", "yyb_proxy_packid",
		"yyb_proxy_region_code", "yyb_proxy_region_name",
		"yyb_proxy_url", "yyb_proxy_ipport", "yyb_proxy_expire_at", "yyb_proxy_last_at",
	}
	out := map[string]interface{}{}
	for _, k := range keys {
		if v, ok := cred[k]; ok {
			out[k] = v
		}
	}
	return out
}

func isProxyOrNetworkError(err error) bool {
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
	}
	for _, n := range needles {
		if strings.Contains(msg, n) {
			return true
		}
	}
	return false
}

func sortedKeys[M ~map[string]V, V any](m M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
