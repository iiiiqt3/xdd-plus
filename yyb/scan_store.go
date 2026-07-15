package yyb

import (
	"context"

	"github.com/cdle/xdd/yyb/internal/protocol"
)

// StoreScanAccount 门户扫码确认后写入账号（含 51 代理元数据）
func (s *Service) StoreScanAccount(ctx context.Context, loginBuffer string, creds protocol.LoginBufferCredentials, proxyMeta map[string]interface{}) (*AccountPublic, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	acc, err := s.app.StoreScanAccount(ctx, loginBuffer, creds, proxyMeta)
	if err != nil {
		return nil, err
	}
	out := toAccountPublic(*acc)
	return &out, nil
}
