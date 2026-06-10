package models

import "strings"

// IsJdTaskProxyEnabled 京东任务动态代理是否启用（需开关打开且 API 地址非空）
func IsJdTaskProxyEnabled() bool {
	url := strings.TrimSpace(sysConfig.JdTaskProxyUrl)
	if url == "" {
		return false
	}
	return sysConfig.JdTaskProxyEnabled
}

// ApplyJdTaskProxyEnvs 向 Node 脚本环境变量注入 DY_PROXY（脚本要求的环境变量名）
func ApplyJdTaskProxyEnvs(envs map[string]string) {
	if envs == nil || !IsJdTaskProxyEnabled() {
		return
	}
	envs["DY_PROXY"] = strings.TrimSpace(sysConfig.JdTaskProxyUrl)
	renum := strings.TrimSpace(sysConfig.JdTaskProxyRenum)
	if renum == "" {
		renum = "10"
	}
	redelay := strings.TrimSpace(sysConfig.JdTaskProxyRedelay)
	if redelay == "" {
		redelay = "2"
	}
	envs["DY_PROXY_RENUM"] = renum
	envs["DY_PROXY_REDELAY"] = redelay
}

// ApplyJdProTaskProxyEnvs 部分脚本使用 PRO_API_PROXY_URL，与 DY_PROXY 共用同一 API 地址
func ApplyJdProTaskProxyEnvs(envs map[string]string) {
	ApplyJdTaskProxyEnvs(envs)
	if envs == nil || !IsJdTaskProxyEnabled() {
		return
	}
	envs["PRO_API_PROXY_URL"] = strings.TrimSpace(sysConfig.JdTaskProxyUrl)
	envs["PRO_PROXY_WHITELIST"] = "jd"
}
