package models

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

)

const (
	kuwoAESKey      = "eXNpVmtMSkhIbnZNV0NIcQ=="
	kuwoAESIV       = "aWNoWW9vWCtNYjFnUmV0UA=="
	kuwoUserAgent   = "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro Build/AP4A.250405.002; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/136.0.7103.60 Mobile Safari/537.36/ kuwopage"
	kuwoCapURL      = "http://www.kuwo.cn/api/common/captcha/getcode"
	kuwoOCRURL      = "https://ddddocr.linzixuan.work/classification"
	kuwoLoginURL    = "https://wapi.kuwo.cn/api/www/login/loginByKw"
	kuwoSmsURL      = "https://integralapi.kuwo.cn/api/v1/online/sign/v1/userBindPhone"
	kuwoWithdrawURL = "https://integralapi.kuwo.cn/api/v1/online/sign/v1/getWithdraw"

	kuwoSessionCacheTTL     = 4 * time.Minute // 与倒计时窗口一致，短信码有效期5分钟
	kuwoWithdrawRounds        = 1
	kuwoWithdrawRetryPerRound = 30
	kuwoWithdrawStaggerMs   = 30
	kuwoScheduledLeadMs       = 30 // 整点前提前触发（毫秒）
	kuwoWarmupBeforeSec       = 5
	kuwoSessionRefreshBefore  = 35 * time.Second
	kuwoLoginMaxRetry         = 3
	kuwoTaskRetainDuration    = 2 * time.Hour
	kuwoProxyPrepareBeforeSec = 8  // 抢兑前提前准备代理（秒）
	kuwoProxyTestTimeout      = 3 * time.Second
	kuwoProxyPrepareMaxRetry  = 5
	kuwoProxyMaxAge           = 25 * time.Second // 动态 IP 约 30s 有效，超龄换 IP
)

var kuwoHTTPClient = &http.Client{
	Timeout: 12 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        32,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	},
}

// kuwoWithdrawProxy 仅抢兑提现请求使用（与京东共用动态代理 API）
type kuwoWithdrawProxy struct {
	client    *http.Client
	proxyHost string
	fetchedAt time.Time
}

func (p *kuwoWithdrawProxy) host() string {
	if p == nil {
		return ""
	}
	return p.proxyHost
}

func kuwoFetchWithdrawProxy(caller string) *kuwoWithdrawProxy {
	if !IsJdTaskProxyEnabled() {
		return nil
	}
	proxyURL, err := fetchDynamicProxyURL(caller)
	if err != nil || proxyURL == nil {
		return nil
	}
	base := &http.Transport{
		MaxIdleConns:          32,
		MaxIdleConnsPerHost:   16,
		MaxConnsPerHost:       16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
		Proxy:                 http.ProxyURL(proxyURL),
		ForceAttemptHTTP2:     true,
	}
	return &kuwoWithdrawProxy{
		client:    &http.Client{Timeout: 8 * time.Second, Transport: base},
		proxyHost: proxyURL.Host,
		fetchedAt: time.Now(),
	}
}

func kuwoTestWithdrawProxy(proxy *kuwoWithdrawProxy) bool {
	if proxy == nil || proxy.client == nil || proxy.proxyHost == "" {
		return false
	}
	req, err := http.NewRequest(http.MethodGet, kuwoWithdrawURL+"?proxy_test=1", nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	ctx, cancel := context.WithTimeout(context.Background(), kuwoProxyTestTimeout)
	defer cancel()
	resp, err := proxy.client.Do(req.WithContext(ctx))
	if err != nil {
		return false
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode > 0
}

var kuwoBJLocation *time.Location

func kuwoBeijingLocation() *time.Location {
	if kuwoBJLocation != nil {
		return kuwoBJLocation
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	kuwoBJLocation = loc
	return loc
}

// KuwoSession holds login credentials for a Kuwo account.
type KuwoSession struct {
	LoginUid       string
	LoginSid       string
	Phone          string
	EncryptedPhone string
}

// KuwoAccountInput 抢兑账号（含密码，到点可重新登录刷新 session）
type KuwoAccountInput struct {
	Phone    string
	Password string
}

// kuwoSessionCache 登录session缓存，避免到点时重新登录浪费时间
var kuwoSessionCache = struct {
	sync.RWMutex
	sessions map[string]*KuwoSession
	expiry   map[string]time.Time
}{sessions: make(map[string]*KuwoSession), expiry: make(map[string]time.Time)}

// KuwoCacheSession 缓存登录session，到点抢兑时直接用
func KuwoCacheSession(phone string, session *KuwoSession) {
	kuwoSessionCache.Lock()
	defer kuwoSessionCache.Unlock()
	kuwoSessionCache.sessions[phone] = session
	kuwoSessionCache.expiry[phone] = time.Now().Add(kuwoSessionCacheTTL)
}

// KuwoGetCachedSession 获取缓存的登录session
func KuwoGetCachedSession(phone string) *KuwoSession {
	kuwoSessionCache.RLock()
	defer kuwoSessionCache.RUnlock()
	session, ok := kuwoSessionCache.sessions[phone]
	if !ok || time.Now().After(kuwoSessionCache.expiry[phone]) {
		return nil
	}
	return session
}

// KuwoWithdrawResult is the result of a withdrawal attempt.
type KuwoWithdrawResult struct {
	UID     string
	Phone   string
	Success bool
	Message string
	Error   error
}

// KuwoQuotaMap maps withdrawal amounts to their quota IDs.
var KuwoQuotaMap = map[string]string{
	"1":  "60004",
	"2":  "30002",
	"10": "60001",
}

// ---------- helpers ----------

func kuwoRandomAppUID() string {
	const digits = "0123456789"
	b := make([]byte, 10)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		b[i] = digits[idx.Int64()]
	}
	return string(b)
}

// KuwoEncryptPhone encrypts a phone number with AES-128-CBC.
func KuwoEncryptPhone(phone string) string {
	key, _ := base64.StdEncoding.DecodeString(kuwoAESKey)
	iv, _ := base64.StdEncoding.DecodeString(kuwoAESIV)

	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	padLen := aes.BlockSize - len(phone)%aes.BlockSize
	padded := make([]byte, len(phone)+padLen)
	copy(padded, phone)
	for i := len(phone); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func kuwoDoRequest(req *http.Request) ([]byte, int, error) {
	resp, err := kuwoHTTPClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	return kuwoReadResponseBody(resp)
}

func kuwoDoWithdrawRequest(proxy *kuwoWithdrawProxy, req *http.Request) ([]byte, int, error) {
	client := kuwoHTTPClient
	if proxy != nil && proxy.client != nil && proxy.proxyHost != "" {
		client = proxy.client
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	return kuwoReadResponseBody(resp)
}

func kuwoReadResponseBody(resp *http.Response) ([]byte, int, error) {
	var body []byte
	var err error
	contentEncoding := resp.Header.Get("Content-Encoding")
	if strings.Contains(contentEncoding, "gzip") {
		reader, gzErr := gzip.NewReader(resp.Body)
		if gzErr != nil {
			body, err = io.ReadAll(resp.Body)
		} else {
			defer reader.Close()
			body, err = io.ReadAll(reader)
		}
	} else {
		body, err = io.ReadAll(resp.Body)
	}
	return body, resp.StatusCode, err
}

func kuwoWarmupConnections() {
	req, err := http.NewRequest("GET", kuwoWithdrawURL+"?warmup=1", nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	req.Header.Set("Connection", "keep-alive")
	_, _, _ = kuwoDoRequest(req)
}

func kuwoSleepUntil(target time.Time) {
	for {
		d := time.Until(target)
		if d <= 0 {
			return
		}
		if d > 50*time.Millisecond {
			time.Sleep(d - 10*time.Millisecond)
			continue
		}
		for time.Now().Before(target) {
			runtime.Gosched()
		}
		return
	}
}

func kuwoIsWithdrawSuccess(text string) bool {
	if text == "" {
		return false
	}
	successKeys := []string{"提现成功", "提现申请发起成功", "申请成功", "已提现"}
	for _, k := range successKeys {
		if strings.Contains(text, k) {
			return true
		}
	}
	return false
}

func kuwoIsFatalWithdrawError(text string) bool {
	fatalKeys := []string{"验证码", "验证失败", "已过期", "登录", "未绑定", "请先登录"}
	lower := strings.ToLower(text)
	for _, k := range fatalKeys {
		if strings.Contains(lower, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

func kuwoEnsureSessions(accounts []*KuwoAccountInput) []*KuwoSession {
	sessions := make([]*KuwoSession, 0, len(accounts))
	for _, acc := range accounts {
		if acc == nil || acc.Phone == "" {
			continue
		}
		if cached := KuwoGetCachedSession(acc.Phone); cached != nil && cached.LoginUid != "" {
			sessions = append(sessions, cached)
			continue
		}
		if acc.Password != "" {
			sess, err := kuwoLoginWithRetry(acc.Phone, acc.Password)
			if err != nil {
				Kuwo().Infof("[kuwo] 刷新登录失败 phone=%s: %v\n", acc.Phone, err)
				continue
			}
			KuwoCacheSession(acc.Phone, sess)
			sessions = append(sessions, sess)
			continue
		}
		Kuwo().Infof("[kuwo] 无法获取有效session phone=%s\n", acc.Phone)
	}
	return sessions
}

func kuwoLoginWithRetry(phone, password string) (*KuwoSession, error) {
	var lastErr error
	for attempt := 1; attempt <= kuwoLoginMaxRetry; attempt++ {
		sess, err := KuwoLogin(phone, password)
		if err == nil {
			return sess, nil
		}
		lastErr = err
		if attempt < kuwoLoginMaxRetry {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}
	return nil, lastErr
}

func kuwoAnySuccess(results []KuwoWithdrawResult) bool {
	for _, r := range results {
		if r.Success {
			return true
		}
	}
	return false
}

func kuwoBestResultsPerPhone(all []KuwoWithdrawResult) []KuwoWithdrawResult {
	best := make(map[string]KuwoWithdrawResult)
	order := make([]string, 0)
	for _, r := range all {
		key := r.Phone
		if key == "" {
			key = r.UID
		}
		if _, ok := best[key]; !ok {
			order = append(order, key)
		}
		prev, ok := best[key]
		if !ok || (!prev.Success && r.Success) {
			best[key] = r
		}
	}
	out := make([]KuwoWithdrawResult, 0, len(order))
	for _, k := range order {
		out = append(out, best[k])
	}
	return out
}

func kuwoResultsToJSON(results []KuwoWithdrawResult) []WithdrawResultJSON {
	out := make([]WithdrawResultJSON, 0, len(results))
	for _, r := range results {
		errMsg := ""
		if r.Error != nil {
			errMsg = r.Error.Error()
		}
		out = append(out, WithdrawResultJSON{
			Phone:   r.Phone,
			Success: r.Success,
			Message: r.Message,
			Error:   errMsg,
		})
	}
	return out
}

func kuwoGetCaptcha() (imgBase64 string, token string, err error) {
	reqURL := kuwoCapURL + "?reqId=" + kuwoRandomAppUID() + "&httpsStatus=1"
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.6261.95 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "http://www.kuwo.cn/")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	body, _, err := kuwoDoRequest(req)
	if err != nil {
		return "", "", err
	}

	var capResp struct {
		Data struct {
			Img   string `json:"img"`
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &capResp); err != nil {
		return "", "", fmt.Errorf("parse captcha response: %w", err)
	}
	if capResp.Data.Img == "" || capResp.Data.Token == "" {
		return "", "", fmt.Errorf("captcha response missing img or token")
	}

	imgStr := capResp.Data.Img
	if idx := strings.Index(imgStr, ","); idx >= 0 {
		imgStr = imgStr[idx+1:]
	}

	ocrPayload, _ := json.Marshal(map[string]string{"image": imgStr})
	ocrReq, err := http.NewRequest("POST", kuwoOCRURL, bytes.NewReader(ocrPayload))
	if err != nil {
		return "", "", err
	}
	ocrReq.Header.Set("Content-Type", "application/json")
	ocrReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	ocrBody, _, err := kuwoDoRequest(ocrReq)
	if err != nil {
		return "", "", err
	}

	var ocrResp struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(ocrBody, &ocrResp); err != nil {
		return strings.TrimSpace(string(ocrBody)), capResp.Data.Token, nil
	}
	return strings.TrimSpace(ocrResp.Result), capResp.Data.Token, nil
}

// KuwoLogin performs the full login flow: captcha -> OCR -> login（直连）
func KuwoLogin(phone, password string) (*KuwoSession, error) {
	encPhone := KuwoEncryptPhone(phone)

	for attempt := 0; attempt < 5; attempt++ {
		captchaCode, captchaToken, err := kuwoGetCaptcha()
		if err != nil {
			return nil, fmt.Errorf("get captcha failed: %w", err)
		}
		Kuwo().Infof("[kuwo] captcha attempt %d: code=%s token=%s\n", attempt+1, captchaCode, captchaToken)

		loginBody, _ := json.Marshal(map[string]string{
			"userIp":          "www.kuwo.cn",
			"uname":           phone,
			"password":        password,
			"verifyCode":      captchaCode,
			"img":             "",
			"verifyCodeToken": captchaToken,
		})

		req, err := http.NewRequest("POST", kuwoLoginURL+"?httpsStatus=1", bytes.NewReader(loginBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.6261.95 Safari/537.36")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://www.kuwo.cn")
		req.Header.Set("Referer", "http://www.kuwo.cn/")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

		body, _, err := kuwoDoRequest(req)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Cookies struct {
					Websid interface{} `json:"websid"`
					Userid interface{} `json:"userid"`
				} `json:"cookies"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse login response: %w", err)
		}

		Kuwo().Infof("[kuwo] login response: code=%d msg=%s\n", resp.Code, resp.Msg)

		uid := fmt.Sprintf("%.0f", resp.Data.Cookies.Userid)
		sid := fmt.Sprintf("%v", resp.Data.Cookies.Websid)
		if resp.Code == 200 && uid != "" && uid != "<nil>" && sid != "" && sid != "<nil>" {
			return &KuwoSession{
				LoginUid:       uid,
				LoginSid:       sid,
				Phone:          phone,
				EncryptedPhone: encPhone,
			}, nil
		}
		Kuwo().Infof("[kuwo] login attempt %d failed: %s\n", attempt+1, resp.Msg)
	}

	return nil, fmt.Errorf("login failed after 5 attempts")
}

// KuwoSendSms sends an SMS verification code for the logged-in session（直连）
func KuwoSendSms(session *KuwoSession) error {
	params := url.Values{}
	params.Set("loginUid", session.LoginUid)
	params.Set("loginSid", session.LoginSid)
	params.Set("mobile", session.EncryptedPhone)

	reqURL := kuwoSmsURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://h5app.kuwo.cn")
	req.Header.Set("Referer", "https://h5app.kuwo.cn/apps/earning-sign/cash_out.html")

	body, _, err := kuwoDoRequest(req)
	if err != nil {
		return err
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse sms response: %w", err)
	}
	if resp.Code != 200 {
		return fmt.Errorf("send sms failed: %s", resp.Msg)
	}
	return nil
}

// KuwoExecuteWithdraw executes a single withdrawal request.
func KuwoExecuteWithdraw(session *KuwoSession, quotaId, smsCode string, proxy *kuwoWithdrawProxy) (string, error) {
	if session == nil || session.LoginUid == "" || session.LoginSid == "" {
		return "", fmt.Errorf("withdraw failed: session无效")
	}

	params := url.Values{}
	params.Set("encry", "")
	params.Set("type", "")
	params.Set("quotaId", quotaId)
	params.Set("loginUid", session.LoginUid)
	params.Set("loginSid", session.LoginSid)
	params.Set("appuid", kuwoRandomAppUID())
	params.Set("source", "kwplayer_ar_12.1.4.0_40.apk")
	params.Set("version", "1")
	params.Set("phone", session.EncryptedPhone)
	params.Set("code", smsCode)

	reqURL := kuwoWithdrawURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://h5app.kuwo.cn")
	req.Header.Set("Referer", "https://h5app.kuwo.cn/apps/earning-sign/cash_out.html")
	req.Header.Set("Connection", "keep-alive")

	body, _, err := kuwoDoWithdrawRequest(proxy, req)
	if err != nil {
		return "", err
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse withdraw response: %w", err)
	}

	text := resp.Data.Text
	if text == "" {
		text = resp.Msg
	}

	if kuwoIsWithdrawSuccess(text) {
		return text, nil
	}
	return text, fmt.Errorf("withdraw failed: %s", text)
}

// KuwoConcurrentWithdrawRetry 单账号多路错峰抢兑，取首个成功
func KuwoConcurrentWithdrawRetry(sessions []*KuwoSession, quotaId, smsCode string, retryCount int, task *KuwoScheduledTask, round int, proxy *kuwoWithdrawProxy) []KuwoWithdrawResult {
	if retryCount < 1 {
		retryCount = 1
	}
	results := make([]KuwoWithdrawResult, len(sessions))
	var wg sync.WaitGroup
	var logMu sync.Mutex

	for i, sess := range sessions {
		wg.Add(1)
		go func(idx int, s *KuwoSession) {
			defer wg.Done()
			type attemptResult struct {
				msg   string
				err   error
				index int
			}
			ch := make(chan attemptResult, retryCount)
			var once sync.Once
			stopCh := make(chan struct{})
			phoneLabel := kuwoMaskPhone(s.Phone)

			for t := 0; t < retryCount; t++ {
				go func(attempt int) {
					if attempt > 0 {
						time.Sleep(time.Duration(attempt*kuwoWithdrawStaggerMs) * time.Millisecond)
					}
					select {
					case <-stopCh:
						ch <- attemptResult{err: fmt.Errorf("cancelled"), index: attempt}
						return
					default:
					}
					msg, err := KuwoExecuteWithdraw(s, quotaId, smsCode, proxy)
					ch <- attemptResult{msg: msg, err: err, index: attempt}
					if task != nil {
						attemptNo := attempt + 1
						if err == nil {
							task.AddLog("success", "第%d轮 第%d次 %s 成功：%s", round+1, attemptNo, phoneLabel, msg)
						} else if kuwoIsFatalWithdrawError(msg) {
							task.AddLog("error", "第%d轮 第%d次 %s 失败（终止重试）：%s", round+1, attemptNo, phoneLabel, msg)
						} else {
							task.AddLog("warn", "第%d轮 第%d次 %s 失败：%s", round+1, attemptNo, phoneLabel, msg)
						}
					}
					if err == nil {
						once.Do(func() { close(stopCh) })
					} else if kuwoIsFatalWithdrawError(msg) {
						once.Do(func() { close(stopCh) })
					}
				}(t)
			}

			var lastResult attemptResult
			successFound := false
			for t := 0; t < retryCount; t++ {
				r := <-ch
				if r.err != nil && r.err.Error() == "cancelled" {
					continue
				}
				logMu.Lock()
				if r.err == nil && !successFound {
					results[idx] = KuwoWithdrawResult{
						UID:     s.LoginUid,
						Phone:   s.Phone,
						Success: true,
						Message: r.msg,
					}
					Kuwo().Infof("[kuwo] withdraw uid=%s success (round %d attempt %d/%d): %s\n", s.LoginUid, round+1, r.index+1, retryCount, r.msg)
					successFound = true
				}
				lastResult = r
				logMu.Unlock()
				if successFound {
					break
				}
			}
			if !successFound {
				results[idx] = KuwoWithdrawResult{
					UID:     s.LoginUid,
					Phone:   s.Phone,
					Success: false,
					Message: lastResult.msg,
					Error:   lastResult.err,
				}
				Kuwo().Infof("[kuwo] withdraw uid=%s failed (round %d) after %d attempts: %v\n", s.LoginUid, round+1, retryCount, lastResult.err)
			}
		}(i, sess)
	}

	wg.Wait()
	return results
}

// KuwoBurstWithdraw 多轮爆发抢兑（倒计时抢兑用）
func KuwoBurstWithdraw(sessions []*KuwoSession, quotaID, smsCode string, task *KuwoScheduledTask) []KuwoWithdrawResult {
	if task != nil {
		task.AddLog("info", "开始爆发抢兑：共%d轮，每轮最多%d次，错峰%dms",
			kuwoWithdrawRounds, kuwoWithdrawRetryPerRound, kuwoWithdrawStaggerMs)
	}
	var allResults []KuwoWithdrawResult
	for round := 0; round < kuwoWithdrawRounds; round++ {
		if round > 0 {
			time.Sleep(80 * time.Millisecond)
		}
		if task != nil {
			task.AddLog("info", "──── 第 %d/%d 轮 ────", round+1, kuwoWithdrawRounds)
		}
		if task != nil && round > 0 && IsJdTaskProxyEnabled() {
			task.ensureWithdrawProxyFresh()
		}
		var withdrawProxy *kuwoWithdrawProxy
		if task != nil {
			withdrawProxy = task.withdrawProxy
		}
		roundResults := KuwoConcurrentWithdrawRetry(sessions, quotaID, smsCode, kuwoWithdrawRetryPerRound, task, round, withdrawProxy)
		allResults = append(allResults, roundResults...)
		if kuwoAnySuccess(roundResults) {
			if task != nil {
				task.AddLog("success", "第%d轮抢兑成功，提前结束", round+1)
			}
			break
		}
		if task != nil {
			task.AddLog("warn", "第%d轮未成功，%s", round+1, func() string {
				if round+1 < kuwoWithdrawRounds {
					return "继续下一轮"
				}
				return "已无更多轮次"
			}())
		}
	}
	return kuwoBestResultsPerPhone(allResults)
}

// KuwoSingleWithdraw 单次提现（立即/手动抢兑，可传代理）
func KuwoSingleWithdraw(sessions []*KuwoSession, quotaID, smsCode string, proxy *kuwoWithdrawProxy) []KuwoWithdrawResult {
	results := make([]KuwoWithdrawResult, len(sessions))
	for i, s := range sessions {
		msg, err := KuwoExecuteWithdraw(s, quotaID, smsCode, proxy)
		results[i] = KuwoWithdrawResult{
			UID:     s.LoginUid,
			Phone:   s.Phone,
			Success: err == nil,
			Message: msg,
			Error:   err,
		}
	}
	return results
}

// KuwoManualWithdraw 人工触发提现（按需取代理，不提前准备）
func KuwoManualWithdraw(sessions []*KuwoSession, quotaID, smsCode string) ([]KuwoWithdrawResult, string) {
	proxy := kuwoWithdrawProxyOnDemand("kuwo_withdraw")
	proxyHost := ""
	if proxy != nil {
		proxyHost = proxy.host()
	}
	return KuwoSingleWithdraw(sessions, quotaID, smsCode, proxy), proxyHost
}

// kuwoWithdrawProxyOnDemand 人工抢兑时按需取代理（单次获取+测试，不做提前准备）
func kuwoWithdrawProxyOnDemand(caller string) *kuwoWithdrawProxy {
	if !IsJdTaskProxyEnabled() {
		return nil
	}
	for i := 0; i < 2; i++ {
		proxy := kuwoFetchWithdrawProxy(caller)
		if proxy != nil && proxy.host() != "" && kuwoTestWithdrawProxy(proxy) {
			return proxy
		}
	}
	return nil
}

// kuwoPrepareWithdrawProxy 定时抢兑专用：提前获取并测试代理（含重试）
func kuwoPrepareWithdrawProxy(caller string) *kuwoWithdrawProxy {
	if !IsJdTaskProxyEnabled() {
		return nil
	}
	deadline := time.Now().Add(6 * time.Second)
	for attempt := 1; attempt <= kuwoProxyPrepareMaxRetry; attempt++ {
		if time.Now().After(deadline) {
			break
		}
		proxy := kuwoFetchWithdrawProxy(caller)
		if proxy == nil || proxy.host() == "" {
			continue
		}
		if kuwoTestWithdrawProxy(proxy) {
			return proxy
		}
	}
	return nil
}

// KuwoGetQuotaID returns the quota ID for the given amount string.
func KuwoGetQuotaID(amount string) (string, bool) {
	id, ok := KuwoQuotaMap[amount]
	return id, ok
}

// KuwoFormatQuotaDisplay returns a human-readable list of available quotas.
func KuwoFormatQuotaDisplay() string {
	var parts []string
	for amount, id := range KuwoQuotaMap {
		parts = append(parts, fmt.Sprintf("%s元(id=%s)", amount, id))
	}
	return strings.Join(parts, ", ")
}

// GetKuwoCredentials 从用户的KWYY活动项目中读取酷我账号密码
// KuwoAccountInfo 酷我账号信息（支持多账号）
type KuwoAccountInfo struct {
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// GetKuwoCredentials 获取用户所有酷我账号（支持多活动上车）
func GetKuwoCredentials(userNumber int) (phone, password string, err error) {
	accounts, err := GetAllKuwoCredentials(userNumber)
	if err != nil {
		return "", "", err
	}
	if len(accounts) == 0 {
		return "", "", fmt.Errorf("未找到酷我音乐活动配置")
	}
	return accounts[0].Phone, accounts[0].Password, nil
}

// GetAllKuwoCredentials 获取用户所有酷我账号列表
func GetAllKuwoCredentials(userNumber int) ([]KuwoAccountInfo, error) {
	projects, err := GetActivityProjectsByUserAndEnv(userNumber, "KWYY", "KWYY")
	if err != nil {
		return nil, err
	}
	var accounts []KuwoAccountInfo
	for _, p := range projects {
		if p.EnvValue == "" {
			continue
		}
		parts := strings.SplitN(p.EnvValue, "#", 2)
		if len(parts) == 2 {
			accounts = append(accounts, KuwoAccountInfo{Phone: parts[0], Password: parts[1]})
		} else {
			accounts = append(accounts, KuwoAccountInfo{Phone: p.EnvValue})
		}
	}
	return accounts, nil
}

// CheckKuwoAuth 检查当前用户是否有酷我音乐(KWYY)活动授权
func CheckKuwoAuth(userNumber int) (bool, string) {
	projects, err := GetActivityProjectsByUserAndEnv(userNumber, "KWYY", "KWYY")
	if err != nil || len(projects) == 0 {
		return false, "您还没有上车酷我音乐活动，请先前往「项目中心」上车"
	}

	now := time.Now()
	for _, p := range projects {
		if p.ExpireDate != "" {
			expireTime, err := time.Parse("2006-01-02", p.ExpireDate)
			if err == nil {
				expireThreshold := time.Date(expireTime.Year(), expireTime.Month(), expireTime.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
				if now.After(expireThreshold) {
					return false, "您的酷我音乐活动授权已到期，请前往「项目中心」续费"
				}
			}
		}
	}
	return true, ""
}

func init() {
	Kuwo().Infof("[kuwo] module loaded, available quotas: %s\n", KuwoFormatQuotaDisplay())
}

// ===================== 抢兑任务（定时 / 立即） =====================

// KuwoTaskLog 抢兑任务执行日志（供前端展示）
type KuwoTaskLog struct {
	Time      string `json:"time"`
	Level     string `json:"level"` // info / success / warn / error
	Message   string `json:"message"`
	ProxyHost string `json:"proxyHost,omitempty"`
}

// KuwoScheduledTask 抢兑任务
type KuwoScheduledTask struct {
	ID          string               `json:"id"`
	Phone       string               `json:"phone"`
	QuotaID     string               `json:"quotaID"`
	SmsCode     string               `json:"smsCode"`
	TargetHour  int                  `json:"targetHour"`
	Immediate   bool                 `json:"immediate"`
	ProxyHost     string               `json:"proxyHost,omitempty"`
	Accounts      []*KuwoAccountInput  `json:"-"`
	Sessions      []*KuwoSession       `json:"-"`
	withdrawProxy *kuwoWithdrawProxy   `json:"-"`
	CreatedAt     time.Time            `json:"createdAt"`
	ExecuteAt   time.Time            `json:"executeAt"`
	Status      string               `json:"status"` // pending / running / completed / failed
	Results     []KuwoWithdrawResult `json:"results,omitempty"`
	ResultsJSON []WithdrawResultJSON `json:"resultsDetail,omitempty"`
	Logs        []KuwoTaskLog        `json:"logs,omitempty"`
	logMu       sync.Mutex           `json:"-"`
}

func (t *KuwoScheduledTask) currentProxyHost() string {
	if t == nil {
		return ""
	}
	if t.withdrawProxy != nil && t.withdrawProxy.host() != "" {
		return t.withdrawProxy.host()
	}
	return t.ProxyHost
}

// prepareWithdrawProxy 定时倒计时抢兑专用：提前预取并测试代理
func (t *KuwoScheduledTask) prepareWithdrawProxy() {
	if t == nil {
		return
	}
	if !IsJdTaskProxyEnabled() {
		t.AddLog("info", "抢兑代理：未启用（直连）")
		return
	}
	deadline := time.Now().Add(6 * time.Second)
	for attempt := 1; attempt <= kuwoProxyPrepareMaxRetry; attempt++ {
		if time.Now().After(deadline) {
			t.AddLog("warn", "抢兑代理：准备超时，停止重试")
			break
		}
		t.AddLog("info", "抢兑代理：第%d次获取 IP…", attempt)
		proxy := kuwoFetchWithdrawProxy("kuwo_withdraw")
		if proxy == nil || proxy.host() == "" {
			t.AddLog("warn", "抢兑代理：获取 IP 失败")
			continue
		}
		t.AddLog("info", "抢兑代理：测试 %s …", proxy.host())
		if kuwoTestWithdrawProxy(proxy) {
			t.withdrawProxy = proxy
			t.ProxyHost = proxy.host()
			t.AddLog("success", "抢兑代理：%s 已就绪", proxy.host())
			return
		}
		t.AddLog("warn", "抢兑代理：%s 不可用，重新获取", proxy.host())
	}
	t.AddLog("error", "抢兑代理：准备失败，抢兑时将尝试直连")
}

// ensureWithdrawProxyFresh 多轮抢兑间检查代理是否过期或失效
func (t *KuwoScheduledTask) ensureWithdrawProxyFresh() {
	if t == nil || !IsJdTaskProxyEnabled() {
		return
	}
	needRefresh := t.withdrawProxy == nil || t.withdrawProxy.host() == ""
	if !needRefresh && t.withdrawProxy != nil {
		if time.Since(t.withdrawProxy.fetchedAt) > kuwoProxyMaxAge {
			needRefresh = true
			t.AddLog("info", "抢兑代理：IP 已超 %ds，换新 IP", int(kuwoProxyMaxAge.Seconds()))
		} else if !kuwoTestWithdrawProxy(t.withdrawProxy) {
			needRefresh = true
			t.AddLog("warn", "抢兑代理：%s 失效，换新 IP", t.withdrawProxy.host())
		}
	}
	if needRefresh {
		t.prepareWithdrawProxy()
	}
}

func (t *KuwoScheduledTask) AddLog(level, format string, args ...interface{}) {
	if t == nil {
		return
	}
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	logTime := time.Now().In(kuwoBeijingLocation()).Format("15:04:05.000")
	proxyHost := t.currentProxyHost()
	t.logMu.Lock()
	defer t.logMu.Unlock()
	t.Logs = append(t.Logs, KuwoTaskLog{
		Time:      logTime,
		Level:     level,
		Message:   msg,
		ProxyHost: proxyHost,
	})
	// 写入日志文件
	go t.writeLogToFile(logTime, level, msg, proxyHost)
}

// writeLogToFile 将日志追加写入文件
func (t *KuwoScheduledTask) writeLogToFile(logTime, level, msg, proxyHost string) {
	logDir := filepath.Join(ExecPath, "logs")
	os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, fmt.Sprintf("kuwo_%s.log", time.Now().In(kuwoBeijingLocation()).Format("2006-01-02")))
	line := fmt.Sprintf("[%s] [%s] %s", logTime, level, msg)
	if proxyHost != "" {
		line += fmt.Sprintf(" [代理:%s]", proxyHost)
	}
	line += "\n"
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)
}

func kuwoMaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) == 11 {
		return phone[:3] + "****" + phone[7:]
	}
	return phone
}

func kuwoQuotaLabel(quotaID string) string {
	for amount, id := range KuwoQuotaMap {
		if id == quotaID {
			return amount + "元"
		}
	}
	return quotaID
}

// WithdrawResultJSON 提现结果JSON格式
type WithdrawResultJSON struct {
	Phone   string `json:"phone"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

var kuwoScheduledTasks = struct {
	sync.RWMutex
	tasks map[string]*KuwoScheduledTask
}{tasks: make(map[string]*KuwoScheduledTask)}

// KuwoScheduleWithdraw 创建抢兑任务；immediate=true 时立即执行。
// 第二个返回值 reused=true 表示返回了同手机号已有的 pending/running 任务。
func KuwoScheduleWithdraw(accounts []*KuwoAccountInput, quotaID, smsCode string, targetHour int, immediate bool) (*KuwoScheduledTask, bool) {
	kuwoPruneOldTasks()

	phone := ""
	if len(accounts) > 0 && accounts[0] != nil {
		phone = strings.TrimSpace(accounts[0].Phone)
	}
	if phone != "" {
		if existing := kuwoFindActiveTaskByPhone(phone); existing != nil {
			Kuwo().Infof("[kuwo] 复用已有任务: id=%s phone=%s status=%s\n", existing.ID, kuwoMaskPhone(phone), existing.Status)
			return existing, true
		}
	}

	now := time.Now()
	bjLoc := kuwoBeijingLocation()
	bjTime := now.In(bjLoc)

	var executeAt time.Time
	if immediate {
		executeAt = now
	} else {
		executeAt = time.Date(bjTime.Year(), bjTime.Month(), bjTime.Day(), targetHour, 0, 0, 0, bjLoc)
		if executeAt.Before(bjTime) {
			executeAt = executeAt.Add(24 * time.Hour)
		}
	}

	taskID := fmt.Sprintf("kuwo_%d_%s", now.UnixMilli(), quotaID)
	task := &KuwoScheduledTask{
		ID:         taskID,
		Phone:      phone,
		QuotaID:    quotaID,
		SmsCode:    smsCode,
		TargetHour: targetHour,
		Immediate:  immediate,
		Accounts:   accounts,
		CreatedAt:  now,
		ExecuteAt:  executeAt,
		Status:     "pending",
	}

	kuwoScheduledTasks.Lock()
	kuwoScheduledTasks.tasks[taskID] = task
	kuwoScheduledTasks.Unlock()

	if immediate {
		Kuwo().Infof("[kuwo] 立即抢兑任务: id=%s\n", taskID)
		go kuwoRunImmediateTask(taskID)
	} else {
		task.AddLog("info", "定时抢兑任务已创建")
		task.AddLog("info", "账号：%s | 档位：%s | 目标时间：%s",
			kuwoMaskPhone(phone), kuwoQuotaLabel(quotaID), executeAt.Format("15:04:05"))
		task.AddLog("info", "任务ID：%s", taskID)
		Kuwo().Infof("[kuwo] 定时任务已创建: id=%s target=%s\n", taskID, executeAt.Format("2006-01-02 15:04:05"))
		go kuwoRunScheduledTask(taskID)
	}

	return task, false
}

func kuwoFindActiveTaskByPhone(phone string) *KuwoScheduledTask {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil
	}
	kuwoScheduledTasks.RLock()
	defer kuwoScheduledTasks.RUnlock()
	for _, t := range kuwoScheduledTasks.tasks {
		if t == nil {
			continue
		}
		if strings.TrimSpace(t.Phone) == phone && (t.Status == "pending" || t.Status == "running") {
			return t
		}
	}
	return nil
}

func kuwoPruneOldTasks() {
	cutoff := time.Now().Add(-kuwoTaskRetainDuration)
	kuwoScheduledTasks.Lock()
	defer kuwoScheduledTasks.Unlock()
	for id, t := range kuwoScheduledTasks.tasks {
		if t == nil {
			delete(kuwoScheduledTasks.tasks, id)
			continue
		}
		if (t.Status == "completed" || t.Status == "failed") && t.CreatedAt.Before(cutoff) {
			delete(kuwoScheduledTasks.tasks, id)
		}
	}
}

func kuwoRunImmediateTask(taskID string) {
	kuwoScheduledTasks.RLock()
	task, ok := kuwoScheduledTasks.tasks[taskID]
	kuwoScheduledTasks.RUnlock()
	if !ok {
		return
	}

	task.Status = "running"
	task.AddLog("info", "立即抢兑开始")

	sessions := kuwoEnsureSessions(task.Accounts)
	task.Sessions = sessions
	if len(sessions) == 0 {
		task.Status = "failed"
		task.AddLog("error", "登录失败，无法获取有效会话")
		task.ResultsJSON = []WithdrawResultJSON{{
			Phone: task.Phone, Success: false, Message: "登录失败，无法获取有效会话", Error: "session无效",
		}}
		return
	}

	var proxy *kuwoWithdrawProxy
	if IsJdTaskProxyEnabled() {
		proxy = kuwoWithdrawProxyOnDemand("kuwo_immediate")
		if proxy != nil {
			task.withdrawProxy = proxy
			task.ProxyHost = proxy.host()
			task.AddLog("info", "抢兑代理：%s", proxy.host())
		} else {
			task.AddLog("warn", "抢兑代理：获取失败，直连提现")
		}
	}

	results := KuwoSingleWithdraw(sessions, task.QuotaID, task.SmsCode, proxy)
	task.Results = results
	task.ResultsJSON = kuwoResultsToJSON(results)
	task.Status = "completed"
	kuwoLogTaskDone(task, results)
}

func kuwoRunScheduledTask(taskID string) {
	kuwoScheduledTasks.RLock()
	task, ok := kuwoScheduledTasks.tasks[taskID]
	kuwoScheduledTasks.RUnlock()
	if !ok {
		return
	}

	fireAt := task.ExecuteAt.Add(-time.Duration(kuwoScheduledLeadMs) * time.Millisecond)
	warmupAt := fireAt.Add(-time.Duration(kuwoWarmupBeforeSec) * time.Second)
	refreshAt := fireAt.Add(-kuwoSessionRefreshBefore)
	proxyPrepAt := fireAt.Add(-time.Duration(kuwoProxyPrepareBeforeSec) * time.Second)

	task.AddLog("info", "后台调度已启动，等待抢兑时刻…")

	// 代理准备与预热/登录并行，避免卡点提交时取 IP 占用到点后的抢兑时间
	var proxyWg sync.WaitGroup
	if IsJdTaskProxyEnabled() {
		proxyWg.Add(1)
		go func() {
			defer proxyWg.Done()
			if d := time.Until(proxyPrepAt); d > 0 {
				task.AddLog("info", "距抢兑代理准备还有 %s", d.Round(time.Millisecond))
				kuwoSleepUntil(proxyPrepAt)
			}
			task.prepareWithdrawProxy()
		}()
	} else {
		task.AddLog("info", "抢兑代理：未启用（直连）")
	}

	if wait := time.Until(warmupAt); wait > 0 {
		task.AddLog("info", "距预热还有 %s", wait.Round(time.Second))
	}

	if d := time.Until(warmupAt); d > 0 {
		time.Sleep(d)
	}
	task.AddLog("info", "开始预热酷我 API 连接")
	kuwoWarmupConnections()
	task.AddLog("info", "连接预热完成")

	if d := time.Until(refreshAt); d > 0 {
		task.AddLog("info", "距刷新登录还有 %s", d.Round(time.Second))
		time.Sleep(d)
	}
	task.AddLog("info", "到点前刷新登录会话…")
	sessions := kuwoEnsureSessions(task.Accounts)
	task.Sessions = sessions
	if len(sessions) == 0 {
		task.AddLog("warn", "首次登录失败，%d 秒后重试…", 2)
		time.Sleep(2 * time.Second)
		sessions = kuwoEnsureSessions(task.Accounts)
		task.Sessions = sessions
	}
	if len(sessions) == 0 {
		task.Status = "failed"
		task.AddLog("error", "到点前登录失败，无法获取有效会话")
		task.ResultsJSON = []WithdrawResultJSON{{
			Phone: task.Phone, Success: false, Message: "到点前登录失败", Error: "session无效",
		}}
		return
	}
	task.AddLog("success", "登录会话就绪（%d 个账号）", len(sessions))

	if IsJdTaskProxyEnabled() {
		proxyWg.Wait()
	}

	if d := time.Until(fireAt); d > 0 {
		task.AddLog("info", "精确等待抢兑触发点 %s（提前 %dms）", fireAt.Format("15:04:05.000"), kuwoScheduledLeadMs)
		kuwoSleepUntil(fireAt)
	}

	kuwoExecuteScheduledTask(taskID)
}

func kuwoExecuteScheduledTask(taskID string) {
	kuwoScheduledTasks.RLock()
	task, ok := kuwoScheduledTasks.tasks[taskID]
	kuwoScheduledTasks.RUnlock()
	if !ok {
		return
	}

	task.Status = "running"
	task.AddLog("info", "═══ 到达抢兑时刻，开始执行 ═══")
	Kuwo().Infof("[kuwo] 定时任务开始执行: id=%s at %s proxy=%s\n", taskID, time.Now().Format("15:04:05.000"), task.currentProxyHost())

	if len(task.Sessions) == 0 {
		task.Sessions = kuwoEnsureSessions(task.Accounts)
	}
	if len(task.Sessions) == 0 {
		task.Status = "failed"
		task.AddLog("error", "执行时无有效会话")
		task.ResultsJSON = []WithdrawResultJSON{{
			Phone: task.Phone, Success: false, Message: "执行时无有效会话", Error: "session无效",
		}}
		return
	}

	results := KuwoBurstWithdraw(task.Sessions, task.QuotaID, task.SmsCode, task)
	task.Results = results
	task.ResultsJSON = kuwoResultsToJSON(results)
	task.Status = "completed"
	kuwoLogTaskDone(task, results)
}

func kuwoLogTaskDone(task *KuwoScheduledTask, results []KuwoWithdrawResult) {
	if task == nil {
		return
	}
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	Kuwo().Infof("[kuwo] 任务执行完成: id=%s 成功%d/%d\n", task.ID, successCount, len(results))
	if successCount > 0 {
		task.AddLog("success", "════ 抢兑完成：成功 %d/%d 个账号 ════", successCount, len(results))
		for _, r := range results {
			if r.Success {
				task.AddLog("success", "最终结果 %s：%s", kuwoMaskPhone(r.Phone), r.Message)
			}
		}
	} else {
		task.AddLog("error", "════ 抢兑结束：全部失败（%d 个账号）════", len(results))
		for _, r := range results {
			msg := r.Message
			if msg == "" && r.Error != nil {
				msg = r.Error.Error()
			}
			task.AddLog("error", "最终结果 %s：%s", kuwoMaskPhone(r.Phone), msg)
		}
	}
}

func (t *KuwoScheduledTask) LogsSnapshot() []KuwoTaskLog {
	if t == nil {
		return nil
	}
	t.logMu.Lock()
	defer t.logMu.Unlock()
	out := make([]KuwoTaskLog, len(t.Logs))
	copy(out, t.Logs)
	return out
}

// KuwoGetScheduledTask 查询任务状态
func KuwoGetScheduledTask(taskID string) *KuwoScheduledTask {
	kuwoScheduledTasks.RLock()
	defer kuwoScheduledTasks.RUnlock()
	return kuwoScheduledTasks.tasks[taskID]
}

// KuwoGetActiveTaskByPhone 查询手机号下 pending/running 任务
func KuwoGetActiveTaskByPhone(phone string) *KuwoScheduledTask {
	return kuwoFindActiveTaskByPhone(phone)
}

// KuwoListScheduledTasks 查询所有任务
func KuwoListScheduledTasks() []*KuwoScheduledTask {
	kuwoScheduledTasks.RLock()
	defer kuwoScheduledTasks.RUnlock()
	tasks := make([]*KuwoScheduledTask, 0, len(kuwoScheduledTasks.tasks))
	for _, t := range kuwoScheduledTasks.tasks {
		tasks = append(tasks, t)
	}
	return tasks
}

// KuwoBuildSessionsFromRequest 从请求构建 session（优先缓存，否则直连登录）
func KuwoBuildSessionsFromRequest(accounts []*KuwoAccountInput) []*KuwoSession {
	return kuwoEnsureSessions(accounts)
}
