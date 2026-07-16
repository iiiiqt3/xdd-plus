package protocol

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
)

const (
	tenPayURL = "/cgi-bin/micromsg-bin/tenpay"
	tenPayCmd = 385
)

// TenPayRequest describes one real TenPay CGI request through YYB WMPF transfer.
// It is intentionally generic: callers must provide the real cgi_cmd/req_text and the real response is returned.
type TenPayRequest struct {
	CGICmd      uint64         `json:"cgi_cmd"`
	ReqText     string         `json:"req_text"`
	ReqTextWx   string         `json:"req_text_wx"`
	RawBase64   string         `json:"raw_base64"`
	HostAppID   string         `json:"host_app_id"`
	SessionMode string         `json:"session_mode"`
	Payload     map[string]any `json:"payload"`
}

// TenPayCGI calls /cgi-bin/micromsg-bin/tenpay with the current YYB login state.
// Critical logic: this never fakes payment success; unsupported/missing parameters return errors.
func (p *Pool) TenPayCGI(ctx context.Context, loginBuffer string, accountID int64, tcpProxy string, req TenPayRequest) (map[string]any, error) {
	return p.run(ctx, loginBuffer, accountID, tcpProxy, func(ctx context.Context, st WmpfSession) (map[string]any, error) {
		body, err := buildTenPayBody(st.Session, req)
		if err != nil {
			return nil, err
		}
		plain := buildJSAPIPlaintext(st.Session.UIN, "", []byte(tenPayURL), tenPayCmd, body, probeHostAppID(req.HostAppID, st.Session.HostAppID), nil)
		envelope, err := buildTransferPacket(st.Session, plain)
		if err != nil {
			return nil, err
		}
		code, resp, err := p.sendEnvelope(ctx, st, envelope)
		if err != nil {
			return nil, err
		}
		out := parseRawResponse(code, resp)
		out["_tenpay"] = map[string]any{
			"cgi_url":  tenPayURL,
			"cmd_id":   tenPayCmd,
			"cgi_cmd":  req.CGICmd,
			"code_hex": hex.EncodeToString(code),
			"resp_hex": truncateHex(resp, 4096),
		}
		return out, nil
	})
}

func buildTenPayBody(sess AppSession, req TenPayRequest) ([]byte, error) {
	if strings.TrimSpace(req.RawBase64) != "" {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.RawBase64))
		if err != nil {
			return nil, fmt.Errorf("raw_base64 decode failed: %w", err)
		}
		return raw, nil
	}
	cgiCmd := req.CGICmd
	if cgiCmd == 0 {
		cgiCmd = 85
	}
	reqText := strings.TrimSpace(req.ReqText)
	if reqText == "" {
		reqText = buildTenPayReqTextFromPayload(req.Payload)
	}
	if reqText == "" {
		return nil, fmt.Errorf("req_text is required for TenPay CGI")
	}
	body := make([]byte, 0, 128+len(reqText)+len(req.ReqTextWx))
	body = append(body, pbLen(1, buildOfficialBaseRequest(sess, req.SessionMode))...)
	body = append(body, pbVar(2, cgiCmd)...)
	body = append(body, pbVar(3, 1)...)
	body = append(body, pbLen(4, skBuiltinString(reqText))...)
	body = append(body, pbLen(5, skBuiltinString(req.ReqTextWx))...)
	return body, nil
}

func skBuiltinString(s string) []byte {
	body := make([]byte, 0, len(s)+8)
	body = append(body, pbVar(1, uint64(len(s)))...)
	body = append(body, pbLen(2, []byte(s))...)
	return body
}

func buildTenPayReqTextFromPayload(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	if raw := stringFromMapAny(payload, "req_text", "ReqText", "tenpay_url", "TenpayUrl"); raw != "" {
		return raw
	}
	// Best-effort merchant transfer confirmation compatibility. Some scripts pass only mch/app/package.
	// The business server decides whether these fields are sufficient; YYB returns the real TenPay response.
	pkg := stringFromMapAny(payload, "PackageInfo", "package", "package_info", "req_key")
	mchid := stringFromMapAny(payload, "Mchid", "mchid", "mch_id")
	mchAppid := stringFromMapAny(payload, "MchAppid", "mch_appid", "app_id", "appid")
	if pkg == "" && mchid == "" && mchAppid == "" {
		return ""
	}
	v := url.Values{}
	if mchid != "" {
		v.Set("mchid", mchid)
		v.Set("mch_id", mchid)
	}
	if mchAppid != "" {
		v.Set("mch_appid", mchAppid)
		v.Set("appid", mchAppid)
	}
	if pkg != "" {
		v.Set("package", pkg)
		v.Set("req_key", pkg)
	}
	v.Set("op", firstNonEmptyProtocol(stringFromMapAny(payload, "op", "Op"), "confirm"))
	return v.Encode()
}
