package yybportal

import (
	"context"
	"strings"
)

// InternalOfficialCGIMap 内部公众号/通用 CGI
func InternalOfficialCGIMap(ref string, body map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.OfficialCGIMap(context.Background(), strings.TrimSpace(ref), body)
}

// InternalTenPayCGIMap 内部 TenPay CGI
func InternalTenPayCGIMap(ref string, body map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.TenPayCGIMap(context.Background(), strings.TrimSpace(ref), body)
}

// InternalRuntimeSession 内部 runtime session
func InternalRuntimeSession(ref, appID string, payload map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.RuntimeSessionMap(context.Background(), strings.TrimSpace(ref), strings.TrimSpace(appID), payload)
}

// InternalUpdateStepMap 内部刷步
func InternalUpdateStepMap(ref string, body map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.UpdateStepMap(context.Background(), strings.TrimSpace(ref), body)
}

// InternalReportMotionMap 内部上报运动
func InternalReportMotionMap(ref string, body map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.ReportMotionMap(context.Background(), strings.TrimSpace(ref), body)
}

// InternalGetBoundHardDevicesMap 内部查询绑定硬件
func InternalGetBoundHardDevicesMap(ref string, body map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.GetBoundHardDevicesMap(context.Background(), strings.TrimSpace(ref), body)
}

// InternalGetWeRunData 内部获取微信运动数据
func InternalGetWeRunData(ref, appID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.GetWeRunDataMap(context.Background(), strings.TrimSpace(ref), strings.TrimSpace(appID))
}
