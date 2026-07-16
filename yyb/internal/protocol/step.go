package protocol

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	stepUploadURL  = "/cgi-bin/mmoc-bin/hardware/uploaddevicestep"
	stepUploadCmd  = 1261
	stepDeviceURL  = "/cgi-bin/micromsg-bin/getboundharddevices"
	stepDeviceCmd  = 539
	stepMaxDefault = 98000
)

// StepRequest describes one WeChat Sports step upload request.
// The YYB runtime must return the real CGI response and must not fake success.
type StepRequest struct {
	Number      int64          `json:"number"`
	DeviceID    string         `json:"device_id"`
	DeviceType  string         `json:"device_type"`
	AppName     string         `json:"app_name"`
	BundleID    string         `json:"bundle_id"`
	RawBase64   string         `json:"raw_base64"`
	HostAppID   string         `json:"host_app_id"`
	Payload     map[string]any `json:"payload"`
	SkipLookup  bool           `json:"skip_lookup"`
	SessionMode string         `json:"session_mode"`
}

// UpdateStep uploads WeChat Sports steps through real hardware CGI.
// Critical logic: lookup bound hard device first, then call uploaddevicestep.
func (p *Pool) UpdateStep(ctx context.Context, loginBuffer string, accountID int64, tcpProxy string, req StepRequest) (map[string]any, error) {
	return p.run(ctx, loginBuffer, accountID, tcpProxy, func(ctx context.Context, st WmpfSession) (map[string]any, error) {
		if req.Number <= 0 {
			return nil, fmt.Errorf("number/stepCount must be greater than 0")
		}
		if req.Number > stepMaxDefault {
			return nil, fmt.Errorf("number/stepCount exceeds %d", stepMaxDefault)
		}
		if strings.TrimSpace(req.RawBase64) != "" {
			return p.sendStepRaw(ctx, st, req, stepUploadURL, stepUploadCmd)
		}

		deviceID := strings.TrimSpace(req.DeviceID)
		deviceType := strings.TrimSpace(req.DeviceType)
		var lookup map[string]any
		if (deviceID == "" || deviceType == "") && !req.SkipLookup {
			var err error
			deviceID, deviceType, lookup, err = p.lookupStepDevice(ctx, st, req)
			if err != nil {
				return nil, err
			}
		}
		if deviceID == "" || deviceType == "" {
			return nil, fmt.Errorf("device_id/device_type is required; no bound hard device was found")
		}
		body := buildUploadDeviceStepBody(st.Session, req, deviceID, deviceType)
		out, err := p.sendStepCGI(ctx, st, req, stepUploadURL, stepUploadCmd, body)
		if err != nil {
			return nil, err
		}
		out["_step"] = map[string]any{
			"cgi_url":     stepUploadURL,
			"cmd_id":      stepUploadCmd,
			"device_id":   deviceID,
			"device_type": deviceType,
			"number":      req.Number,
			"lookup":      lookup,
		}
		return out, nil
	})
}

// ReportMotion keeps 861-compatible naming and reuses the same upload CGI.
func (p *Pool) ReportMotion(ctx context.Context, loginBuffer string, accountID int64, tcpProxy string, req StepRequest) (map[string]any, error) {
	return p.UpdateStep(ctx, loginBuffer, accountID, tcpProxy, req)
}

// GetBoundHardDevices calls the real getboundharddevices CGI for debugging and lookup.
func (p *Pool) GetBoundHardDevices(ctx context.Context, loginBuffer string, accountID int64, tcpProxy string, req StepRequest) (map[string]any, error) {
	return p.run(ctx, loginBuffer, accountID, tcpProxy, func(ctx context.Context, st WmpfSession) (map[string]any, error) {
		return p.getBoundHardDevicesWithSession(ctx, st, req)
	})
}

func (p *Pool) lookupStepDevice(ctx context.Context, st WmpfSession, req StepRequest) (string, string, map[string]any, error) {
	out, err := p.getBoundHardDevicesWithSession(ctx, st, req)
	if err != nil {
		return "", "", nil, fmt.Errorf("get bound hard devices failed: %w", err)
	}
	raw, _ := out["_raw_resp"].([]byte)
	deviceID, deviceType := firstHardDeviceFromResponse(raw)
	if deviceID == "" || deviceType == "" {
		return "", "", out, fmt.Errorf("no bound hard device in response")
	}
	return deviceID, deviceType, out, nil
}

func (p *Pool) getBoundHardDevicesWithSession(ctx context.Context, st WmpfSession, req StepRequest) (map[string]any, error) {
	body := make([]byte, 0, 96)
	body = append(body, pbLen(1, buildStepBaseRequest(st.Session, req.SessionMode))...)
	body = append(body, pbVar(2, 0)...)
	out, err := p.sendStepCGI(ctx, st, req, stepDeviceURL, stepDeviceCmd, body)
	if err != nil {
		return nil, err
	}
	if raw, ok := out["_raw_resp"].([]byte); ok {
		deviceID, deviceType := firstHardDeviceFromResponse(raw)
		if deviceID != "" || deviceType != "" {
			out["_device"] = map[string]any{"device_id": deviceID, "device_type": deviceType}
		}
	}
	out["_step_device"] = map[string]any{"cgi_url": stepDeviceURL, "cmd_id": stepDeviceCmd}
	return out, nil
}

func (p *Pool) sendStepRaw(ctx context.Context, st WmpfSession, req StepRequest, cgiURL string, cmdID uint64) (map[string]any, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.RawBase64))
	if err != nil {
		return nil, fmt.Errorf("raw_base64 decode failed: %w", err)
	}
	return p.sendStepCGI(ctx, st, req, cgiURL, cmdID, raw)
}

func (p *Pool) sendStepCGI(ctx context.Context, st WmpfSession, req StepRequest, cgiURL string, cmdID uint64, body []byte) (map[string]any, error) {
	// Critical logic: reuse YYB WMPF wxaruntime_transfer to forward the real CGI.
	plain := buildJSAPIPlaintext(st.Session.UIN, "", []byte(cgiURL), cmdID, body, probeHostAppID(req.HostAppID, st.Session.HostAppID), nil)
	envelope, err := buildTransferPacket(st.Session, plain)
	if err != nil {
		return nil, err
	}
	code, resp, err := p.sendEnvelope(ctx, st, envelope)
	if err != nil {
		return nil, err
	}
	out := parseRawResponse(code, resp)
	out["_raw_code"] = append([]byte(nil), code...)
	out["_raw_resp"] = append([]byte(nil), resp...)
	out["_cgi"] = map[string]any{
		"cgi_url":  cgiURL,
		"cmd_id":   cmdID,
		"code_hex": hex.EncodeToString(code),
		"resp_hex": truncateHex(resp, 4096),
	}
	return out, nil
}

func buildUploadDeviceStepBody(sess AppSession, req StepRequest, deviceID, deviceType string) []byte {
	from, to := stepDayRange()
	body := make([]byte, 0, 180)
	body = append(body, pbLen(1, buildStepBaseRequest(sess, req.SessionMode))...)
	body = append(body, pbLen(2, []byte(deviceID))...)
	body = append(body, pbLen(3, []byte(deviceType))...)
	body = append(body, pbVar(4, uint64(from))...)
	body = append(body, pbVar(5, uint64(to))...)
	body = append(body, pbVar(6, uint64(req.Number))...)
	body = append(body, pbLen(7, []byte("8.00"))...)
	if strings.TrimSpace(req.BundleID) != "" {
		body = append(body, pbLen(8, []byte(strings.TrimSpace(req.BundleID)))...)
	}
	if strings.TrimSpace(req.AppName) != "" {
		body = append(body, pbLen(9, []byte(strings.TrimSpace(req.AppName)))...)
	}
	body = append(body, pbVar(10, uint64(req.Number))...)
	body = append(body, pbVar(13, 0)...)
	return body
}

func buildStepBaseRequest(sess AppSession, mode string) []byte {
	sessionKey := sess.Ticket
	if strings.EqualFold(mode, "literal") || len(sessionKey) == 0 {
		sessionKey = sessionKeyLiteral
	}
	body := make([]byte, 0, 96)
	body = append(body, pbLen(1, sessionKey)...)
	body = append(body, pbVar(2, uint64(uint32(sess.UIN)))...)
	body = append(body, pbLen(3, sess.DeviceID)...)
	body = append(body, pbVar(4, uint64(clientVersion))...)
	body = append(body, pbLen(5, unifiedPCWindows)...)
	body = append(body, pbVar(6, 0)...)
	return body
}

func stepDayRange() (int64, int64) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	return start, start + 86400 - 1
}

func firstHardDeviceFromResponse(resp []byte) (string, string) {
	if len(resp) == 0 {
		return "", ""
	}
	root := pbParse(resp)
	for _, raw := range root {
		if b, ok := raw.([]byte); ok {
			if id, typ := firstHardDeviceFromResponse(b); id != "" && typ != "" {
				return id, typ
			}
		}
	}
	if f5, ok := root[5].([]byte); ok {
		mod := pbParse(f5)
		if f1, ok := mod[1].([]byte); ok {
			hard := pbParse(f1)
			deviceType := strings.TrimSpace(string(safeBytes(hard[1])))
			deviceID := strings.TrimSpace(string(safeBytes(hard[2])))
			if deviceID != "" || deviceType != "" {
				return deviceID, deviceType
			}
		}
	}
	return "", ""
}
