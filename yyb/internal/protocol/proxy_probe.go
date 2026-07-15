package protocol

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// NewHTTPClientWithTCPProxy 与 MMTLS 共用 SOCKS5 拨号
func NewHTTPClientWithTCPProxy(timeout time.Duration, proxyValue string, fallbackDirect bool) *http.Client {
	if strings.TrimSpace(proxyValue) == "" {
		return &http.Client{Timeout: timeout}
	}
	tr := &http.Transport{}
	tr.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, portText, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		port, err := strconv.Atoi(portText)
		if err != nil {
			return nil, err
		}
		return dialTCP(ctx, host, port, timeout, proxyValue, fallbackDirect)
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}

// ProbeTCPProxy 检测短效 SOCKS5 是否仍可访问微信 OAuth 域名
func ProbeTCPProxy(proxyValue string) bool {
	proxyValue = strings.TrimSpace(proxyValue)
	if proxyValue == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://open.weixin.qq.com/", nil)
	client := NewHTTPClientWithTCPProxy(8*time.Second, proxyValue, false)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	return resp.StatusCode > 0 && resp.StatusCode < 500
}
