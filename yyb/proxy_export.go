package yyb

import "github.com/cdle/xdd/yyb/internal/protocol"

// ProbeTCPProxy 检测 51 代理是否仍可用
func ProbeTCPProxy(proxyURL string) bool {
	return protocol.ProbeTCPProxy(proxyURL)
}
