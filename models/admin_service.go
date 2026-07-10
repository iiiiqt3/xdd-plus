package models

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// ===================== 活动配置管理（供后台API使用） =====================

// ActivityConfigAdmin 后台展示用的活动配置
type ActivityConfigAdmin struct {
	Name            string                   `json:"name"`
	EnvKey          string                   `json:"envKey"`
	NeedCoin        int                      `json:"needCoin"`
	QingLongConfig  string                   `json:"qingLongConfig"`
	IsMonthlyDeduct bool                     `json:"isMonthlyDeduct"`
	MonthlyCoin     int                      `json:"monthlyCoin"`
	IsDailyDeduct   bool                     `json:"isDailyDeduct"`
	DailyCoin       int                      `json:"dailyCoin"`
	DisplayOrder    int                      `json:"displayOrder"`
	Enabled         bool                     `json:"enabled"`
	InputFields     []map[string]interface{} `json:"inputFields"`
	CKTemplate      string                   `json:"ckTemplate"`
	RemarksTemplate string                   `json:"remarksTemplate"`
	ScriptPaths     map[string]string        `json:"scriptPaths"`
}

func GetActivityConfigsForAdmin() interface{} {
	data, err := ioutil.ReadFile(ExecPath + "/conf/activities.yaml")
	if err != nil {
		return nil
	}
	// 直接返回 YAML 原始内容，前端用 textarea 编辑
	return map[string]interface{}{
		"raw": string(data),
	}
}

// ===================== 青龙容器配置（活动配置关联的 ql1/ql2/ql3） =====================

func GetQingLongConfigsForAdmin() []map[string]interface{} {
	qlManager.mu.RLock()
	defer qlManager.mu.RUnlock()

	var result []map[string]interface{}
	// 收集所有名称并排序，确保 ql1, ql2, ql3 顺序固定
	var names []string
	for name := range qlManager.Configs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cfg := qlManager.Configs[name]
		result = append(result, map[string]interface{}{
			"name":         name,
			"host":         cfg.Host,
			"clientId":     cfg.ClientID,
			"clientSecret": cfg.ClientSecret,
			"timeout":      cfg.Timeout,
		})
	}
	return result
}

// ===================== 京东容器配置（config.yaml中的containers） =====================

func GetJdContainersForAdmin() []map[string]interface{} {
	var result []map[string]interface{}
	for _, c := range Config.Containers {
		result = append(result, map[string]interface{}{
			"address":  c.Address,
			"cid":      c.Cid,
			"secret":   c.Secret,
			"weight":   c.Weigth,
			"mode":     c.Mode,
			"limit":    c.Limit,
			"resident": c.Resident,
			"type":     c.Type,
			"username": c.Name,
		})
	}
	return result
}

// JdConfigForAdmin 后台展示用的京东配置
func GetJdConfigForAdmin() map[string]interface{} {
	return map[string]interface{}{
		"mode":            Config.Mode,
		"wsToken":         Config.WsToken,
		"atTime":          Config.CTime,
		"isOldV4":         Config.IsOldV4,
		"later":           Config.Later,
		"isAddFriend":     Config.IsAddFriend,
		"lim":             Config.Lim,
		"note":            Config.Note,
		"invalid":         Config.Invalid,
		"query":           Config.Query,
		"query1":          Config.Query1,
		"tyt":             Config.Tyt,
		"pzz":             Config.Pzz,
		"jbzl":            Config.Jbzl,
		"apiToken":        Config.ApiToken,
		"master":          Config.Master,
		"database":        Config.Database,
		"qywxKey":         Config.QywxKey,
		"qqUid":           Config.QQID,
		"qqGid":           Config.QQGroupID,
		"wxGid":           GetEnv("WxGroupID"),
		"openQQ":          Config.OpenQQ,
		"qbotPublicMode":  Config.QbotPublicMode,
		"defaultPriority": Config.DefaultPriority,
		// 活动推送群配置1111
		"activityPushQQGroupId": Config.ActivityPushQQGroupID,

		
		"activityPushWXGroupId": Config.ActivityPushWXGroupID,
		// 微信配置
		"wxModel":        Config.Wx.Model,
		"wxUrl":          Config.Wx.Url,
		"wxRobotId":      Config.Wx.Robotid,
		"wxToken":        Config.Wx.Token,
		"wxLoginBaseURL": Config.WxProtocol.LoginBaseURL,
		"wxNewLoginBaseURL": Config.WxProtocol.NewLoginBaseURL,
		"wxActiveProtocol": Config.WxProtocol.ActiveProtocol,
		"wxScanLoginCost": Config.WxProtocol.ScanLoginCost,
		"wxDeviceName":   Config.WxProtocol.DeviceName,
		// 应用宝协议
		"yybEnabled":            Config.Yyb.Enabled,
		"yybResourceRoot":       Config.Yyb.ResourceRoot,
		"yybDBFilename":         Config.Yyb.DBFilename,
		"yybTCPProxy":           Config.Yyb.TCPProxy,
		"yybScanLoginCost":      yybScanLoginCostForAdmin(),
		"yybMaxAccountsPerUser": Config.Yyb.MaxAccountsPerUser,
		"yybAPIToken":           Config.Yyb.APIToken,
		"yybExposeInternalAPI":  Config.Yyb.ExposeInternalAPI,
		// 极光推送
		"jpushEnabled":      Config.Jpush.Enabled,
		"jpushAppKey":       Config.Jpush.AppKey,
		"jpushMasterSecret": Config.Jpush.MasterSecret,
		"jpushProduction":   Config.Jpush.Production,
		// 游戏配置
		"guessNumberCost":     Config.Game.GuessNumberCost,
		"guessNumberTimes":    Config.Game.GuessNumberTimes,
		"guessNumberReward":   Config.Game.GuessNumberReward,
		"guessNumberMineCost": Config.Game.GuessNumberMineCost,
		"rockPaperScissors":   Config.Game.RockPaperScissors,
		"bigSmallCost":        Config.Game.BigSmallCost,
		"bigSmallPlayers":     Config.Game.BigSmallPlayers,
		"bigSmallMaxGames":    Config.Game.BigSmallMaxGames,
		"duelDefaultBet":      Config.Game.DuelDefaultBet,
		"duelMaxRooms":        Config.Game.DuelMaxRooms,
	}
}

func yybScanLoginCostForAdmin() int {
	if Config.Yyb.ScanLoginCost != nil {
		return *Config.Yyb.ScanLoginCost
	}
	if Config.WxProtocol.ScanLoginCost > 0 {
		return Config.WxProtocol.ScanLoginCost
	}
	return 2000
}

// dedupeTopLevelYAMLKeys 去除重复的顶级 YAML 键（保留第一次出现，清理历史误追加的重复项）
func dedupeTopLevelYAMLKeys(lines []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			result = append(result, line)
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			result = append(result, line)
			continue
		}
		idx := strings.Index(trimmed, ":")
		if idx <= 0 {
			result = append(result, line)
			continue
		}
		key := trimmed[:idx]
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, line)
	}
	return result
}

func SaveJdConfigForAdmin(req map[string]interface{}) string {
	data, err := ioutil.ReadFile(ExecPath + "/conf/config.yaml")
	if err != nil {
		return "读取配置文件失败: " + err.Error()
	}

	lines := strings.Split(string(data), "\n")
	// 简单处理：直接重新写入关键配置
	// 实际使用 mapstructure 或 viper 更好，这里用简单替换

	configMap := make(map[string]string)
	if v, ok := req["mode"].(string); ok {
		configMap["mode"] = v
	}
	if v, ok := req["apiToken"].(string); ok {
		configMap["ApiToken"] = v
	}
	if v, ok := req["master"].(string); ok {
		configMap["master"] = v
	}
	if v, ok := req["note"].(string); ok {
		configMap["Note"] = v
	}
	if v, ok := req["jdcurl"].(string); ok {
		configMap["Jdcurl"] = v
	}
	if v, ok := req["invalid"].(string); ok {
		configMap["Invalid"] = v
	}
	if v, ok := req["query"].(string); ok {
		configMap["Query"] = v
	}
	if v, ok := req["query1"].(string); ok {
		configMap["Query1"] = v
	}
	if v, ok := req["qywxKey"].(string); ok {
		configMap["qywx_key"] = v
	}
	if v, ok := req["dailyAssetPushCron"].(string); ok {
		configMap["daily_asset_push_cron"] = v
	}

	if v, ok := req["gameOpen"].(bool); ok {
		configMap["GameOpen"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["isHelp"].(bool); ok {
		configMap["IsHelp"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["wskey"].(bool); ok {
		configMap["Wskey"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["isAddFriend"].(bool); ok {
		configMap["IsAddFriend"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["openQQ"].(bool); ok {
		configMap["OpenQQ"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["qbotPublicMode"].(bool); ok {
		configMap["qbot_public_mode"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["noGhproxy"].(bool); ok {
		configMap["no_ghproxy"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["isOldV4"].(bool); ok {
		configMap["IsOldV4"] = fmt.Sprintf("%v", v)
	}

	if v, ok := req["wsToken"].(float64); ok {
		configMap["WsToken"] = fmt.Sprintf("%d", int(v))
	}
	// 定时推送时间（支持数值和字符串）
	if v, ok := req["atTime"].(float64); ok {
		configMap["AtTime"] = fmt.Sprintf("%d", int(v))
	} else if v, ok := req["atTime"].(string); ok {
		configMap["AtTime"] = v
	}
	if v, ok := req["later"].(float64); ok {
		configMap["Later"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["lim"].(float64); ok {
		configMap["Lim"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["tyt"].(float64); ok {
		configMap["Tyt"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["pzz"].(float64); ok {
		configMap["Pzz"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["jbzl"].(float64); ok {
		configMap["Jbzl"] = fmt.Sprintf("%d", int(v))
	}
	// QQ UID（支持qquid和qqUid两种字段名）
	if v, ok := req["qquid"].(float64); ok {
		configMap["qquid"] = fmt.Sprintf("%d", int(v))
	} else if v, ok := req["qqUid"].(float64); ok {
		configMap["qquid"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["defaultPriority"].(float64); ok {
		configMap["default_priority"] = fmt.Sprintf("%d", int(v))
	}

	// 新增文本字段
	if v, ok := req["dailyCompletePush"].(string); ok {
		configMap["daily_complete_push"] = v
	}
	if v, ok := req["userAgent"].(string); ok {
		configMap["user_agent"] = v
	}
	if v, ok := req["wxModel"].(string); ok {
		configMap["model"] = v
	}
	if v, ok := req["wxUrl"].(string); ok {
		configMap["url"] = v
	}
	if v, ok := req["wxRobotId"].(string); ok {
		configMap["robotid"] = v
	}
	if v, ok := req["wxToken"].(string); ok {
		// 使用 wx_token 前缀，避免与文件顶层 token 冲突；落盘时写入 wx.token
		configMap["wx_token"] = fmt.Sprintf("%q", v)
	}
	if v, ok := req["wxLoginBaseURL"].(string); ok {
		configMap["wp_login_base_url"] = v
	}
	if v, ok := req["wxNewLoginBaseURL"].(string); ok {
		configMap["wp_new_login_base_url"] = v
	}
	if v, ok := req["wxActiveProtocol"].(string); ok {
		configMap["wp_active_protocol"] = v
	}
	if v, ok := req["wxScanLoginCost"].(float64); ok {
		configMap["wp_scan_login_cost"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["wxDeviceName"].(string); ok {
		configMap["wp_device_name"] = v
	}

	// 应用宝协议
	if v, ok := req["yybEnabled"].(bool); ok {
		configMap["yyb_enabled"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["yybResourceRoot"].(string); ok {
		configMap["yyb_resource_root"] = fmt.Sprintf("%q", v)
	}
	if v, ok := req["yybDBFilename"].(string); ok {
		configMap["yyb_db_filename"] = fmt.Sprintf("%q", v)
	}
	if v, ok := req["yybTCPProxy"].(string); ok {
		configMap["yyb_tcp_proxy"] = fmt.Sprintf("%q", v)
	}
	if v, ok := req["yybScanLoginCost"].(float64); ok {
		configMap["yyb_scan_login_cost"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["yybMaxAccountsPerUser"].(float64); ok {
		configMap["yyb_max_accounts_per_user"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["yybAPIToken"].(string); ok {
		configMap["yyb_api_token"] = fmt.Sprintf("%q", v)
	}
	if v, ok := req["yybExposeInternalAPI"].(bool); ok {
		configMap["yyb_expose_internal_api"] = fmt.Sprintf("%v", v)
	}

	// 极光推送
	if v, ok := req["jpushEnabled"].(bool); ok {
		configMap["jpush_enabled"] = fmt.Sprintf("%v", v)
	}
	if v, ok := req["jpushAppKey"].(string); ok {
		configMap["jpush_app_key"] = fmt.Sprintf("%q", v)
	}
	if v, ok := req["jpushMasterSecret"].(string); ok {
		configMap["jpush_master_secret"] = fmt.Sprintf("%q", v)
	}
	if v, ok := req["jpushProduction"].(bool); ok {
		configMap["jpush_production"] = fmt.Sprintf("%v", v)
	}

	// 游戏配置
	if v, ok := req["guessNumberCost"].(float64); ok {
		configMap["game.guess_number_cost"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["guessNumberTimes"].(float64); ok {
		configMap["game.guess_number_times"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["guessNumberReward"].(float64); ok {
		configMap["game.guess_number_reward"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["guessNumberMineCost"].(float64); ok {
		configMap["game.guess_number_mine_cost"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["rockPaperScissors"].(float64); ok {
		configMap["game.rock_paper_scissors"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["bigSmallCost"].(float64); ok {
		configMap["game.big_small_cost"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["bigSmallPlayers"].(float64); ok {
		configMap["game.big_small_players"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["bigSmallMaxGames"].(float64); ok {
		configMap["game.big_small_max_games"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["duelDefaultBet"].(float64); ok {
		configMap["game.duel_default_bet"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["duelMaxRooms"].(float64); ok {
		configMap["game.duel_max_rooms"] = fmt.Sprintf("%d", int(v))
	}

	// QQ群配置（支持qqgid和qqGid两种字段名）
	if v, ok := req["qqgid"].(string); ok {
		configMap["qqgid"] = v
	} else if v, ok := req["qqGid"].(string); ok {
		configMap["qqgid"] = v
	}
	// 微信群监听配置（支持wxgid和wxGid两种字段名）
	var saveWxgid string
	if v, ok := req["wxgid"].(string); ok {
		saveWxgid = v
	} else if v, ok := req["wxGid"].(string); ok {
		saveWxgid = v
	}

	// 活动推送群配置
	if v, ok := req["activityPushQQGroupId"].(string); ok {
		configMap["activity_push_qqgid"] = v
	}
	if v, ok := req["activityPushWXGroupId"].(string); ok {
		configMap["activity_push_wxgid"] = v
	}

	// 简单写入方式
	var newLines []string
	skipKeys := map[string]bool{"containers": true}
	inWxProtocol := false
	inYyb := false
	inJpush := false
	wxProtocolEndIdx := -1
	wxEndIdx := -1
	yybEndIdx := -1
	jpushEndIdx := -1

	wxYamlKeys := map[string]string{
		"model":    "model",
		"url":      "url",
		"robotid":  "robotid",
		"wx_token": "token",
	}
	wpYamlKeys := map[string]string{
		"wp_login_base_url":     "login_base_url",
		"wp_new_login_base_url": "new_login_base_url",
		"wp_active_protocol":    "active_protocol",
		"wp_scan_login_cost":    "scan_login_cost",
		"wp_device_name":        "device_name",
	}
	yybYamlKeys := map[string]string{
		"yyb_enabled":               "enabled",
		"yyb_resource_root":         "resource_root",
		"yyb_db_filename":           "db_filename",
		"yyb_tcp_proxy":             "tcp_proxy",
		"yyb_scan_login_cost":       "scan_login_cost",
		"yyb_max_accounts_per_user": "max_accounts_per_user",
		"yyb_api_token":             "api_token",
		"yyb_expose_internal_api":   "expose_internal_api",
	}
	jpushYamlKeys := map[string]string{
		"jpush_enabled":       "enabled",
		"jpush_app_key":         "app_key",
		"jpush_master_secret":   "master_secret",
		"jpush_production":      "production",
	}

	isIndentedLine := func(line string) bool {
		return strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")
	}
	isTopLevelKeyLine := func(line, trimmed string) bool {
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			return false
		}
		return !isIndentedLine(line) && strings.Contains(trimmed, ":")
	}

	inWx := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			newLines = append(newLines, line)
			continue
		}
		// 跳过容器配置块（由容器管理单独处理）
		if strings.HasPrefix(trimmed, "containers:") {
			skipKeys["containers"] = true
			newLines = append(newLines, line)
			continue
		}
		if strings.HasPrefix(trimmed, "wx:") {
			inWx = true
			inWxProtocol = false
			inYyb = false
			inJpush = false
			newLines = append(newLines, line)
			continue
		}
		if strings.HasPrefix(trimmed, "wx_protocol:") {
			inWxProtocol = true
			inYyb = false
			inJpush = false
			inWx = false
			newLines = append(newLines, line)
			continue
		}
		if strings.HasPrefix(trimmed, "yyb:") {
			inYyb = true
			inJpush = false
			inWxProtocol = false
			inWx = false
			newLines = append(newLines, line)
			continue
		}
		if strings.HasPrefix(trimmed, "jpush:") {
			inJpush = true
			inYyb = false
			inWxProtocol = false
			inWx = false
			newLines = append(newLines, line)
			continue
		}
		// wx_protocol 节遇到下一个顶级键时结束（注释/空行不算）
		if inWxProtocol && isTopLevelKeyLine(line, trimmed) && !strings.HasPrefix(trimmed, "wx_protocol:") {
			inWxProtocol = false
		}
		if inYyb && isTopLevelKeyLine(line, trimmed) && !strings.HasPrefix(trimmed, "yyb:") {
			inYyb = false
		}
		if inJpush && isTopLevelKeyLine(line, trimmed) && !strings.HasPrefix(trimmed, "jpush:") {
			inJpush = false
		}
		// wx 节遇到下一个顶级键时结束
		if inWx && isTopLevelKeyLine(line, trimmed) && !strings.HasPrefix(trimmed, "wx:") {
			inWx = false
		}

		// 处理 wx 嵌套配置（model/url/robotid/token，必须缩进且在 wx 节内）
		if inWx && isIndentedLine(line) {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			// 注释掉的 # token: 也视为可替换目标
			checkTrimmed := trimmed
			if strings.HasPrefix(checkTrimmed, "#") {
				checkTrimmed = strings.TrimSpace(strings.TrimPrefix(checkTrimmed, "#"))
			}
			for mapKey, yamlKey := range wxYamlKeys {
				if newVal, ok := configMap[mapKey]; ok && strings.HasPrefix(checkTrimmed, yamlKey+":") {
					newLines = append(newLines, fmt.Sprintf("%s%s: %s", indent, yamlKey, newVal))
					delete(configMap, mapKey)
					wxEndIdx = len(newLines)
					goto next
				}
			}
			wxEndIdx = len(newLines) + 1
		}
		if inWx {
			wxEndIdx = len(newLines) + 1
		}
		// 处理 wx_protocol 嵌套配置（仅缩进行）
		if inWxProtocol && isIndentedLine(line) && strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			for wpKey, yamlKey := range wpYamlKeys {
				if strings.HasPrefix(trimmed, yamlKey+":") {
					if newVal, ok := configMap[wpKey]; ok {
						newLines = append(newLines, fmt.Sprintf("%s%s: %s", indent, yamlKey, newVal))
						delete(configMap, wpKey)
						goto next
					}
				}
			}
			wxProtocolEndIdx = len(newLines) + 1
			newLines = append(newLines, line)
			goto next
		}
		if inWxProtocol {
			wxProtocolEndIdx = len(newLines) + 1
		}
		if inYyb && isIndentedLine(line) && strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			for yybKey, yamlKey := range yybYamlKeys {
				if strings.HasPrefix(trimmed, yamlKey+":") {
					if newVal, ok := configMap[yybKey]; ok {
						newLines = append(newLines, fmt.Sprintf("%s%s: %s", indent, yamlKey, newVal))
						delete(configMap, yybKey)
						goto next
					}
				}
			}
			yybEndIdx = len(newLines) + 1
			newLines = append(newLines, line)
			goto next
		}
		if inYyb {
			yybEndIdx = len(newLines) + 1
		}
		if inJpush && isIndentedLine(line) && strings.Contains(trimmed, ":") && !strings.HasPrefix(trimmed, "#") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			for jpKey, yamlKey := range jpushYamlKeys {
				if strings.HasPrefix(trimmed, yamlKey+":") {
					if newVal, ok := configMap[jpKey]; ok {
						newLines = append(newLines, fmt.Sprintf("%s%s: %s", indent, yamlKey, newVal))
						delete(configMap, jpKey)
						goto next
					}
				}
			}
			jpushEndIdx = len(newLines) + 1
			newLines = append(newLines, line)
			goto next
		}
		if inJpush {
			jpushEndIdx = len(newLines) + 1
		}

		// 处理顶级配置
		if isTopLevelKeyLine(line, trimmed) {
			for yamlKey, newVal := range configMap {
				if strings.HasPrefix(trimmed, yamlKey+":") {
					newLines = append(newLines, fmt.Sprintf("%s: %s", yamlKey, newVal))
					delete(configMap, yamlKey)
					goto next
				}
			}
		}

		newLines = append(newLines, line)
	next:
	}

	// 将未匹配的 wx 字段插入到 wx 节末尾（例如原先只有注释 # token 时）
	if wxEndIdx < 0 {
		wxEndIdx = len(newLines)
	}
	var wxInsertLines []string
	for mapKey, yamlKey := range wxYamlKeys {
		if newVal, ok := configMap[mapKey]; ok {
			wxInsertLines = append(wxInsertLines, fmt.Sprintf("  %s: %s", yamlKey, newVal))
			delete(configMap, mapKey)
		}
	}
	if len(wxInsertLines) > 0 {
		hasWxSection := false
		for _, line := range newLines {
			if strings.HasPrefix(strings.TrimSpace(line), "wx:") {
				hasWxSection = true
				break
			}
		}
		if hasWxSection {
			newLines = append(newLines[:wxEndIdx], append(wxInsertLines, newLines[wxEndIdx:]...)...)
		} else {
			newLines = append(newLines, "", "# ==================== 微信机器人配置 ====================")
			newLines = append(newLines, "wx:")
			newLines = append(newLines, wxInsertLines...)
		}
	}

	// 清理历史误写入的顶级 token:（应属于 wx.token）
	cleaned := make([]string, 0, len(newLines))
	for _, line := range newLines {
		trimmed := strings.TrimSpace(line)
		if isTopLevelKeyLine(line, trimmed) && strings.HasPrefix(trimmed, "token:") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	newLines = cleaned

	// 将未匹配的 wx_protocol 字段插入到 wx_protocol 节末尾
	if wxProtocolEndIdx < 0 {
		wxProtocolEndIdx = len(newLines)
	}
	var wpInsertLines []string
	for wpKey, yamlKey := range wpYamlKeys {
		if newVal, ok := configMap[wpKey]; ok {
			wpInsertLines = append(wpInsertLines, fmt.Sprintf("  %s: %s", yamlKey, newVal))
			delete(configMap, wpKey)
		}
	}
	if len(wpInsertLines) > 0 {
		newLines = append(newLines[:wxProtocolEndIdx], append(wpInsertLines, newLines[wxProtocolEndIdx:]...)...)
	}

	if yybEndIdx < 0 {
		yybEndIdx = len(newLines)
	}
	var yybInsertLines []string
	for yybKey, yamlKey := range yybYamlKeys {
		if newVal, ok := configMap[yybKey]; ok {
			yybInsertLines = append(yybInsertLines, fmt.Sprintf("  %s: %s", yamlKey, newVal))
			delete(configMap, yybKey)
		}
	}
	if len(yybInsertLines) > 0 {
		hasYybSection := false
		for _, line := range newLines {
			if strings.HasPrefix(strings.TrimSpace(line), "yyb:") {
				hasYybSection = true
				break
			}
		}
		if hasYybSection {
			newLines = append(newLines[:yybEndIdx], append(yybInsertLines, newLines[yybEndIdx:]...)...)
		} else {
			newLines = append(newLines, "", "# ==================== 应用宝协议配置 ====================")
			newLines = append(newLines, "yyb:")
			newLines = append(newLines, yybInsertLines...)
		}
	}

	if jpushEndIdx < 0 {
		jpushEndIdx = len(newLines)
	}
	var jpushInsertLines []string
	for jpKey, yamlKey := range jpushYamlKeys {
		if newVal, ok := configMap[jpKey]; ok {
			jpushInsertLines = append(jpushInsertLines, fmt.Sprintf("  %s: %s", yamlKey, newVal))
			delete(configMap, jpKey)
		}
	}
	if len(jpushInsertLines) > 0 {
		hasJpushSection := false
		for _, line := range newLines {
			if strings.HasPrefix(strings.TrimSpace(line), "jpush:") {
				hasJpushSection = true
				break
			}
		}
		if hasJpushSection {
			newLines = append(newLines[:jpushEndIdx], append(jpushInsertLines, newLines[jpushEndIdx:]...)...)
		} else {
			newLines = append(newLines, "", "# ==================== 极光推送（狗东 App Android） ====================")
			newLines = append(newLines, "jpush:")
			newLines = append(newLines, jpushInsertLines...)
		}
	}

	// 剩余未匹配项：更新已有同名顶级键，避免在文件末尾重复追加
	for k, v := range configMap {
		if strings.HasPrefix(k, "game.") {
			continue
		}
		updated := false
		prefix := k + ":"
		for i, line := range newLines {
			trimmed := strings.TrimSpace(line)
			if isTopLevelKeyLine(line, trimmed) && strings.HasPrefix(trimmed, prefix) {
				newLines[i] = fmt.Sprintf("%s: %s", k, v)
				updated = true
				break
			}
		}
		if !updated {
			newLines = append(newLines, fmt.Sprintf("%s: %s", k, v))
		}
		delete(configMap, k)
	}

	newLines = dedupeTopLevelYAMLKeys(newLines)

	newContent := strings.Join(newLines, "\n")
	err = ioutil.WriteFile(ExecPath+"/conf/config.yaml", []byte(newContent), 0644)
	if err != nil {
		return "写入失败: " + err.Error()
	}

	// 热更新配置（无需重启）
	if err := ReloadConfig(); err != nil {
		Warn("配置热更新失败: %v", err)
		return "保存成功，但热更新失败（重启后生效）"
	}

	if saveWxgid != "" {
		ExportEnv(&Env{Name: "WxGroupID", Value: saveWxgid})
		Config.WXGroupID = saveWxgid
	}

	return "保存成功，配置已实时生效"
}

// ===================== 游戏配置管理（支持热更新） =====================

func GetGameConfigForAdmin() map[string]interface{} {
	return map[string]interface{}{
		"gameOpen":              Config.Game.GameOpen,
		"guessNumberOpen":       Config.Game.GuessNumberOpen,
		"guessNumberCost":       Config.Game.GuessNumberCost,
		"guessNumberTimes":      Config.Game.GuessNumberTimes,
		"guessNumberReward":     Config.Game.GuessNumberReward,
		"guessNumberMineCost":   Config.Game.GuessNumberMineCost,
		"rockPaperScissorsOpen": Config.Game.RockPaperScissorsOpen,
		"rockPaperScissors":     Config.Game.RockPaperScissors,
		"bigSmallOpen":          Config.Game.BigSmallOpen,
		"bigSmallCost":          Config.Game.BigSmallCost,
		"bigSmallPlayers":       Config.Game.BigSmallPlayers,
		"bigSmallMaxGames":      Config.Game.BigSmallMaxGames,
		"duelOpen":              Config.Game.DuelOpen,
		"duelDefaultBet":        Config.Game.DuelDefaultBet,
		"duelMaxRooms":          Config.Game.DuelMaxRooms,
	}
}

func SaveGameConfigForAdmin(req map[string]interface{}) string {
	data, err := ioutil.ReadFile(ExecPath + "/conf/config.yaml")
	if err != nil {
		return "读取配置文件失败: " + err.Error()
	}

	lines := strings.Split(string(data), "\n")
	configMap := make(map[string]string)

	gameKeys := map[string]string{
		"game_open":                "game_open",
		"guess_number_open":        "guess_number_open",
		"guess_number_cost":        "guess_number_cost",
		"guess_number_times":       "guess_number_times",
		"guess_number_reward":      "guess_number_reward",
		"guess_number_mine_cost":   "guess_number_mine_cost",
		"rock_paper_scissors_open": "rock_paper_scissors_open",
		"rock_paper_scissors":      "rock_paper_scissors",
		"big_small_open":           "big_small_open",
		"big_small_cost":           "big_small_cost",
		"big_small_players":        "big_small_players",
		"big_small_max_games":      "big_small_max_games",
		"duel_open":                "duel_open",
		"duel_default_bet":         "duel_default_bet",
		"duel_max_rooms":           "duel_max_rooms",
	}

	for jsonKey, yamlKey := range gameKeys {
		if v, ok := req[jsonKey]; ok {
			switch val := v.(type) {
			case float64:
				configMap[yamlKey] = fmt.Sprintf("%d", int(val))
			case string:
				configMap[yamlKey] = val
			case bool:
				configMap[yamlKey] = fmt.Sprintf("%v", val)
			}
		}
	}

	inGame := false
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "game:") {
			inGame = true
			newLines = append(newLines, line)
			continue
		}
		if inGame && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, ":") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			for yamlKey, newVal := range configMap {
				if strings.HasPrefix(trimmed, yamlKey+":") {
					newLines = append(newLines, fmt.Sprintf("%s%s: %s", indent, yamlKey, newVal))
					delete(configMap, yamlKey)
					goto nextGameLine
				}
			}
			newLines = append(newLines, line)
		nextGameLine:
			continue
		}
		if inGame && (trimmed == "" || strings.HasPrefix(trimmed, "#")) {
			newLines = append(newLines, line)
			continue
		}
		if inGame && !strings.HasPrefix(trimmed, "#") && !strings.Contains(trimmed, ":") {
			inGame = false
		}
		newLines = append(newLines, line)
	}

	for yamlKey, newVal := range configMap {
		if strings.Contains(newVal, ",") || strings.Contains(newVal, "#") || strings.Contains(newVal, ":") {
			newVal = fmt.Sprintf("\"%s\"", newVal)
		}
		newLines = append(newLines, fmt.Sprintf("  %s: %s", yamlKey, newVal))
	}

	newContent := strings.Join(newLines, "\n")
	err = ioutil.WriteFile(ExecPath+"/conf/config.yaml", []byte(newContent), 0644)
	if err != nil {
		return "写入失败: " + err.Error()
	}

	if err := ReloadConfig(); err != nil {
		Warn("游戏配置热更新失败: %v", err)
		return "保存成功，但热更新失败（重启后生效）"
	}

	return "保存成功，游戏配置已实时生效"
}

func SendActivityToGroups(title, content string) string {
	return SendActivityToGroupsWithOptions(title, content, true, true)
}

func SendActivityToGroupsWithOptions(title, content string, toQQ, toWX bool) string {
	msg := strings.TrimSpace(title)
	if msg == "" {
		msg = strings.TrimSpace(content)
	}
	if msg == "" {
		return "推送内容为空，未发送"
	}

	textMsg, media := PrepareGroupPushMessage(title, content)

	var results []string

	if toQQ {
		if Config.ActivityPushQQGroupID != "" {
			gidStr := strings.TrimSpace(Config.ActivityPushQQGroupID)
			parts := strings.Split(gidStr, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				gid, err := strconv.Atoi(part)
				if err != nil {
					results = append(results, "QQ群号格式错误: "+part)
					continue
				}
				PushRichTextToQQGroup(gid, textMsg, media)
				results = append(results, "已推送到QQ群: "+part)
			}
		} else {
			results = append(results, "未配置活动推送QQ群号")
		}
	}

	if toWX {
		if Config.ActivityPushWXGroupID != "" {
			gidStr := strings.TrimSpace(Config.ActivityPushWXGroupID)
			parts := strings.Split(gidStr, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				PushRichTextToWxGroup(part, textMsg, media)
				results = append(results, "已推送到微信群: "+part)
			}
		} else {
			results = append(results, "未配置活动推送微信群号")
		}
	}

	return strings.Join(results, "; ")
}

// ===================== 环境变量管理 =====================

func GetEnvVarsAdmin(search string, page, limit int) ([]Env, int) {
	var envs []Env
	var total int64
	tx := db.Model(&Env{})
	if search != "" {
		tx = tx.Where("name LIKE ? OR value LIKE ? OR note LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	tx.Count(&total)

	offset := (page - 1) * limit
	tx.Order("id ASC").Offset(offset).Limit(limit).Find(&envs)
	return envs, int(total)
}

func CreateEnvVar(env *Env) error {
	return db.Create(env).Error
}

func UpdateEnvVar(env *Env) error {
	return db.Model(env).Where("id = ?", env.ID).Updates(map[string]interface{}{
		"name":  env.Name,
		"value": env.Value,
		"note":  env.Note,
	}).Error
}

func DeleteEnvVar(id int) error {
	return db.Where("id = ?", id).Delete(&Env{}).Error
}

// ===================== 用户管理 =====================

func GetUsersAdmin(search string, page, limit int) ([]User, int) {
	var users []User
	var total int64
	tx := db.Model(&User{})
	if search != "" {
		searchStr := "%" + search + "%"
		tx = tx.Where("nickname LIKE ? OR wxid LIKE ? OR qq LIKE ? OR CAST(number AS CHAR) LIKE ?",
			searchStr, searchStr, searchStr, searchStr)
	}
	tx.Count(&total)

	offset := (page - 1) * limit
	tx.Order("coin DESC, id ASC").Offset(offset).Limit(limit).Find(&users)
	return users, int(total)
}

func SetUserCoin(number int, coin int) error {
	return db.Model(&User{}).Where("number = ?", number).Update("coin", coin).Error
}

// ===================== 系统配置 =====================

func GetSystemConfigForAdmin() SystemConfig {
	return ListConfig()
}

func SaveSystemConfigForAdmin(req map[string]interface{}) string {
	var sys SystemConfig
	data, _ := json.Marshal(req)
	json.Unmarshal(data, &sys)
	return SaveSysConfig(sys)
}

// ===================== 京东CK管理 =====================

func GetJdCookiesAdmin(search string, page, limit int) ([]JdCookie, int) {
	var cks []JdCookie
	var total int64
	tx := db.Model(&JdCookie{})
	if search != "" {
		searchStr := "%" + search + "%"
		tx = tx.Where("PtPin LIKE ? OR Nickname LIKE ? OR Note LIKE ? OR CAST(QQ AS CHAR) LIKE ? OR WeiXin LIKE ? OR LevelName LIKE ? OR Account LIKE ? OR CAST(Priority AS CHAR) LIKE ? OR Available LIKE ? OR PtKey LIKE ? OR CAST(UserId AS CHAR) LIKE ?",
			searchStr, searchStr, searchStr, searchStr, searchStr, searchStr, searchStr, searchStr, searchStr, searchStr, searchStr)
	}
	tx.Count(&total)

	offset := (page - 1) * limit
	tx.Order("Priority DESC, ID ASC").Offset(offset).Limit(limit).Find(&cks)
	return cks, int(total)
}

func DeleteJdCookieById(id int) error {
	return db.Where("ID = ?", id).Delete(&JdCookie{}).Error
}

func UpdateJdCookieById(id int, ptKey string, priority int, available, note, wxid, wxPid, qq string) error {
	updates := map[string]interface{}{
		"Priority":  priority,
		"Available": available,
		"Note":      note,
		"WeiXin":    wxid,
	}
	if wxPid != "" {
		updates["WxPid"] = wxPid
	}
	if ptKey != "" {
		updates["PtKey"] = ptKey
	}
	if qq != "" {
		var qqInt int
		fmt.Sscanf(qq, "%d", &qqInt)
		if qqInt > 0 {
			updates["QQ"] = qqInt
		}
	}
	return db.Model(&JdCookie{}).Where("ID = ?", id).Updates(updates).Error
}

// ===================== 青龙环境变量管理 =====================

func GetQLEnvsAdmin(configName, searchValue string) ([]QLEnvItem, error) {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()

	if !exists {
		// 默认使用第一个
		for _, c := range qlManager.Configs {
			cfg = c
			break
		}
		if cfg == nil {
			return nil, fmt.Errorf("没有可用的青龙配置")
		}
	}

	client := NewQingLongClient(cfg)
	// 先获取所有环境变量（不传搜索词给青龙API），然后在本地搜索所有字段（包括备注）
	envs, err := client.QueryEnvs("")
	if err != nil {
		return nil, err
	}
	if searchValue == "" {
		return envs, nil
	}
	// 本地搜索：name、value、remarks 都可以搜索
	search := strings.ToLower(searchValue)
	var filtered []QLEnvItem
	for _, e := range envs {
		if strings.Contains(strings.ToLower(e.Name), search) ||
			strings.Contains(strings.ToLower(e.Value), search) ||
			strings.Contains(strings.ToLower(e.Remarks), search) {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

func DeleteQLEnvAdmin(configName string, envID int) error {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()

	if !exists {
		for _, c := range qlManager.Configs {
			cfg = c
			break
		}
		if cfg == nil {
			return fmt.Errorf("没有可用的青龙配置")
		}
	}

	client := NewQingLongClient(cfg)
	return client.DeleteEnv(envID)
}

func EnableQLEnvAdmin(configName string, envID int) error {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()

	if !exists {
		for _, c := range qlManager.Configs {
			cfg = c
			break
		}
		if cfg == nil {
			return fmt.Errorf("没有可用的青龙配置")
		}
	}

	client := NewQingLongClient(cfg)
	return client.EnableEnv(envID)
}

func DisableQLEnvAdmin(configName string, envID int) error {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()

	if !exists {
		for _, c := range qlManager.Configs {
			cfg = c
			break
		}
		if cfg == nil {
			return fmt.Errorf("没有可用的青龙配置")
		}
	}

	client := NewQingLongClient(cfg)
	return client.DisableEnvs([]int{envID})
}

// Suppress unused import
var _ = (*gorm.DB)(nil)

// UpdateQLEnvAdmin 更新青龙环境变量
func UpdateQLEnvAdmin(configName string, envID int, name, value, remarks string) error {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()

	if !exists {
		for _, c := range qlManager.Configs {
			cfg = c
			break
		}
		if cfg == nil {
			return fmt.Errorf("没有可用的青龙配置")
		}
	}

	client := NewQingLongClient(cfg)
	return client.UpdateEnv(envID, name, value, remarks)
}

// DeleteUserAdmin 删除用户
func DeleteUserAdmin(id int) error {
	return db.Where("id = ?", id).Delete(&User{}).Error
}

// TestQingLongConnection 测试青龙容器连通性
func TestQingLongConnection(host, clientID, clientSecret string) (bool, string) {
	url := strings.TrimSuffix(host, "/") + "/open/auth/token?client_id=" + clientID + "&client_secret=" + clientSecret
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false, "连接失败: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)
	if code, ok := result["code"].(float64); ok && code == 200 {
		return true, "连接正常"
	}
	msg := "连接失败"
	if m, ok := result["message"].(string); ok {
		msg = m
	}
	return false, msg
}

// ===================== 青龙 Cron 任务管理 =====================

// getQLClientForAdmin 根据配置名获取青龙客户端（复用逻辑）
func getQLClientForAdmin(configName string) (*QingLongClient, error) {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()

	if !exists {
		for _, c := range qlManager.Configs {
			cfg = c
			break
		}
		if cfg == nil {
			return nil, fmt.Errorf("没有可用的青龙配置")
		}
	}
	return NewQingLongClient(cfg), nil
}

// GetCronTasksAdmin 查询青龙定时任务列表
func GetCronTasksAdmin(configName, searchValue string) ([]QLCronTask, error) {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return nil, err
	}
	// 获取所有任务（不传搜索词给青龙API），本地搜索所有字段
	tasks, err := client.QueryCronTasks("")
	if err != nil {
		return nil, err
	}
	if searchValue != "" {
		search := strings.ToLower(searchValue)
		var filtered []QLCronTask
		for _, t := range tasks {
			if strings.Contains(strings.ToLower(t.Name), search) ||
				strings.Contains(strings.ToLower(t.Command), search) ||
				strings.Contains(strings.ToLower(t.Extra), search) ||
				strings.Contains(strings.ToLower(t.Cron), search) ||
				strings.Contains(strings.ToLower(t.Schedule), search) ||
				strings.Contains(fmt.Sprintf("%d", t.ID), search) {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	enrichCronTasksWithLatestRun(client, tasks)
	return tasks, nil
}

func enrichCronTasksWithLatestRun(client *QingLongClient, tasks []QLCronTask) {
	// 性能优化：优先使用青龙原始字段；仅为前面少量仍缺失的任务补一次最新日志时间，避免整表 N+1。
	filledByExtraQuery := 0
	const maxExtraFill = 12
	for i := range tasks {
		if hasCronTaskTime(tasks[i].LastRun) {
			continue
		}
		if hasCronTaskTime(tasks[i].LastRunAlt) {
			tasks[i].LastRun = tasks[i].LastRunAlt
			continue
		}
		if hasCronTaskTime(tasks[i].LastRunAlt2) {
			tasks[i].LastRun = tasks[i].LastRunAlt2
			continue
		}
		if hasCronTaskTime(tasks[i].LastRunAlt3) {
			tasks[i].LastRun = tasks[i].LastRunAlt3
			continue
		}
		if hasCronTaskTime(tasks[i].LastRunAlt4) {
			tasks[i].LastRun = tasks[i].LastRunAlt4
			continue
		}
		if tasks[i].LastRunMS > 946684800 {
			sec := tasks[i].LastRunMS
			if sec > 1e12 {
				sec = sec / 1000
			}
			formatted := time.Unix(sec, 0).Format("2006-01-02 15:04:05")
			if raw, mErr := json.Marshal(formatted); mErr == nil {
				tasks[i].LastRun = json.RawMessage(raw)
				continue
			}
		}
		if filledByExtraQuery >= maxExtraFill {
			continue
		}
		logs, _, err := client.GetCronTaskLogs(tasks[i].ID, 1, 1)
		if err == nil && len(logs) > 0 {
			normalizeCronTaskLog(&logs[0])
			if logs[0].StartTime != "" {
				if raw, mErr := json.Marshal(strings.ReplaceAll(logs[0].StartTime, "T", " ")); mErr == nil {
					tasks[i].LastRun = json.RawMessage(raw)
					filledByExtraQuery++
					continue
				}
			}
		}
	}
}

func hasCronTaskTime(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		s = strings.TrimSpace(s)
		if s == "" || s == "-" || s == "0" {
			return false
		}
		if matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}`, s); matched {
			return true
		}
		if matched, _ := regexp.MatchString(`^\d{10,13}$`, s); matched {
			return true
		}
		return false
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil {
		return n >= 946684800
	}
	return false
}

// GetCronTaskLogsAdmin 查询青龙任务日志列表
func GetCronTaskLogsAdmin(configName string, taskID, page, limit int) ([]QLCronTaskLog, int, error) {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return nil, 0, err
	}
	logs, total, err := client.GetCronTaskLogs(taskID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	// 规范化日志字段：兼容不同青龙版本的时间字段命名
	for i := range logs {
		normalizeCronTaskLog(&logs[i])
	}
	enrichCronTaskLogsWithFiles(client, taskID, logs)
	return logs, total, nil
}

func enrichCronTaskLogsWithFiles(client *QingLongClient, taskID int, logs []QLCronTaskLog) {
	files, err := client.GetTaskLogFiles(taskID)
	if err != nil || len(files) == 0 {
		return
	}
	logPath := fmt.Sprintf("/log/%d", taskID)
	for i := range logs {
		logs[i].LogPath = logPath
		startTime := logs[i].StartTime
		if startTime == "" {
			continue
		}
		prefix := buildCronLogPrefix(startTime)
		if prefix == "" {
			continue
		}
		for _, f := range files {
			if f.Type != "file" {
				continue
			}
			fName := f.Key
			if idx := strings.LastIndex(fName, "/"); idx >= 0 {
				fName = fName[idx+1:]
			}
			if strings.HasPrefix(fName, prefix) {
				logs[i].LogFile = fName
				break
			}
		}
	}
}

func buildCronLogPrefix(startTime string) string {
	startTime = strings.TrimSpace(startTime)
	if startTime == "" {
		return ""
	}
	startTime = strings.ReplaceAll(startTime, "T", " ")
	if len(startTime) >= 19 {
		return startTime[:10] + "-" + strings.ReplaceAll(startTime[11:19], ":", "-")
	}
	return ""
}

// normalizeCronTaskLog 规范化日志字段，确保 startTime/endTime 等字段有值
func normalizeCronTaskLog(log *QLCronTaskLog) {
	if log.StartTime == "" {
		// 优先级：start_time -> createdAt -> updatedAt
		if log.StartTimeAlt != "" {
			log.StartTime = log.StartTimeAlt
		} else if log.CreatedAt != "" {
			log.StartTime = log.CreatedAt
		}
	}
	if log.EndTime == "" {
		if log.EndTimeAlt != "" {
			log.EndTime = log.EndTimeAlt
		} else if log.UpdatedAt != "" && log.UpdatedAt != log.StartTime {
			log.EndTime = log.UpdatedAt
		}
	}
	if log.TaskName == "" && log.TaskNameAlt != "" {
		log.TaskName = log.TaskNameAlt
	}
	if log.TaskID == 0 && log.TaskIdAlt != 0 {
		log.TaskID = log.TaskIdAlt
	}
}

// GetCronTaskLogContentAdmin 获取任务日志内容（通过 taskId）
// logFile 参数可选，用于获取指定历史日志
// logFile 可能是精确文件名（如 "2024-01-15-10-30-45-123.log"）或时间前缀（如 "2024-01-15-10-30-45"）
func GetCronTaskLogContentAdmin(configName string, taskID int, logFile ...string) (string, error) {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return "", err
	}
	// 如果指定了日志文件名，尝试获取指定历史日志
	if len(logFile) > 0 && logFile[0] != "" {
		fileName := logFile[0]
		// 确保有 .log 后缀
		if !strings.HasSuffix(fileName, ".log") {
			fileName += ".log"
		}

		// 方式1：通过任务级 API（部分青龙版本支持 file 参数）
		content, err := client.GetCronTaskLogContent(taskID, fileName)
		if err == nil && content != "" {
			return content, nil
		}

		// 方式2：通过文件系统 API 精确匹配
		for _, logPath := range []string{fmt.Sprintf("/log/%d", taskID), fmt.Sprintf("log/%d", taskID)} {
			content, err = client.GetLogFileContent(logPath, fileName)
			if err == nil && content != "" {
				return content, nil
			}
		}

		// 方式3：如果文件名不含毫秒（如 "2024-01-15-10-30-45.log"），
		// 尝试通过文件系统 API 模糊匹配（文件名以该前缀开头）
		if !strings.Contains(fileName[0:len(fileName)-4], "-") || len(fileName) < 24 {
			// 获取任务日志文件列表，模糊匹配
			files, listErr := client.GetTaskLogFiles(taskID)
			if listErr == nil {
				prefix := fileName[:len(fileName)-4] // 去掉 .log
				for _, f := range files {
					if f.Type == "file" {
						fName := f.Key
						if idx := strings.LastIndex(fName, "/"); idx >= 0 {
							fName = fName[idx+1:]
						}
						if strings.HasPrefix(fName, prefix) {
							content, err = client.GetLogFileContent(fmt.Sprintf("/log/%d", taskID), fName)
							if err == nil && content != "" {
								return content, nil
							}
						}
					}
				}
			}
		}

		// 用户明确选择了历史日志，不再静默 fallback 到最新日志
		System().Infof("获取历史日志失败(file=%s)，不再fallback到最新日志", fileName)
		return "", fmt.Errorf("未找到指定历史日志: %s，可能已被清理或当前青龙版本不支持按文件读取", fileName)
	}
	return client.GetCronTaskLogContent(taskID)
}

// GetCronTaskScriptAdmin 获取任务脚本内容
func GetCronTaskScriptAdmin(configName string, taskID int) (string, error) {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return "", err
	}
	return client.GetCronTaskScript(taskID)
}

// UpdateCronTaskScriptAdmin 更新任务脚本
func UpdateCronTaskScriptAdmin(configName string, taskID int, command string) error {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return err
	}
	return client.UpdateCronTaskScript(taskID, command)
}

// GetCronTaskScriptFileAdmin 获取任务脚本文件内容（读取实际脚本文件而非 command 字段）
func GetCronTaskScriptFileAdmin(configName string, taskID int) (string, string, error) {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return "", "", err
	}
	command, err := client.GetCronTaskScript(taskID)
	if err != nil {
		return "", "", err
	}
	scriptPath := parseScriptPath(command)
	if scriptPath.Filename == "" {
		return "", command, nil
	}
	content, err := client.GetScriptFileContent(scriptPath.Path, scriptPath.Filename)
	if err != nil {
		return "", command, fmt.Errorf("获取脚本文件内容失败: %v（任务命令: %s，解析路径: path=%s file=%s）", err, command, scriptPath.Path, scriptPath.Filename)
	}
	if strings.TrimSpace(content) == "" {
		return "", command, fmt.Errorf("脚本文件内容为空，疑似未命中真实脚本路径（任务命令: %s，解析路径: path=%s file=%s）", command, scriptPath.Path, scriptPath.Filename)
	}
	return content, command, nil
}

// UpdateCronTaskScriptFileAdmin 更新任务脚本文件内容
func UpdateCronTaskScriptFileAdmin(configName string, taskID int, content string) error {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return err
	}
	command, err := client.GetCronTaskScript(taskID)
	if err != nil {
		return err
	}
	scriptPath := parseScriptPath(command)
	if scriptPath.Filename == "" {
		return fmt.Errorf("无法从任务命令解析脚本文件名: %s", command)
	}
	return client.UpdateScriptFileContent(scriptPath.Path, scriptPath.Filename, content)
}

// parseScriptPath 从任务命令中解析脚本路径
// 支持格式: "task xmyx.js", "task dir/a.js", "node xmyx.js", "python3 a/b.py"
func parseScriptPath(command string) QLScriptPath {
	command = strings.TrimSpace(command)
	if command == "" {
		return QLScriptPath{}
	}

	prefixes := []string{"task ", "node ", "node_modules ", "ql/node ", "python ", "python3 ", "bash ", "sh "}
	for _, p := range prefixes {
		if strings.HasPrefix(command, p) {
			command = strings.TrimPrefix(command, p)
			break
		}
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return QLScriptPath{}
	}

	fullPath := strings.TrimSpace(parts[0])
	fullPath = strings.Trim(fullPath, "\"'")
	fullPath = strings.ReplaceAll(fullPath, "\\", "/")

	validExts := map[string]bool{".js": true, ".py": true, ".sh": true, ".ts": true}
	ext := ""
	if idx := strings.LastIndex(fullPath, "."); idx >= 0 {
		ext = fullPath[idx:]
	}
	if !validExts[ext] {
		return QLScriptPath{}
	}

	result := QLScriptPath{FullPath: fullPath, Filename: fullPath}
	if idx := strings.LastIndex(fullPath, "/"); idx >= 0 {
		result.Path = fullPath[:idx]
		result.Filename = fullPath[idx+1:]
	}
	return result
}

// UpdateCronTaskScheduleAdmin 更新任务定时规则
func UpdateCronTaskScheduleAdmin(configName string, taskID int, cronExpr string) error {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return err
	}
	return client.UpdateCronTaskSchedule(taskID, cronExpr)
}

// EnableCronTaskAdmin 启用任务
func EnableCronTaskAdmin(configName string, taskIDs []int) error {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return err
	}
	return client.EnableCronTask(taskIDs)
}

// DisableCronTaskAdmin 禁用任务
func DisableCronTaskAdmin(configName string, taskIDs []int) error {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return err
	}
	return client.DisableCronTask(taskIDs)
}

// DeleteCronTaskAdmin 删除任务
func DeleteCronTaskAdmin(configName string, taskIDs []int) error {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return err
	}
	return client.DeleteCronTask(taskIDs)
}

// RunCronTaskAdmin 手动运行任务
func RunCronTaskAdmin(configName string, taskIDs []int) error {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return err
	}
	return client.RunCronTask(taskIDs)
}

// StopCronTaskAdmin 停止运行中的任务
func StopCronTaskAdmin(configName string, taskIDs []int) error {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return err
	}
	return client.StopCronTask(taskIDs)
}

// GetRunningCronTasksAdmin 获取正在运行的任务
func GetRunningCronTasksAdmin(configName string) ([]QLCronTaskDetail, error) {
	client, err := getQLClientForAdmin(configName)
	if err != nil {
		return nil, err
	}
	return client.GetRunningCronTasks()
}

// GetJdContainerNamesForAdmin 获取京东容器名称列表（用于前端容器选择）
func GetJdContainerNamesForAdmin() []map[string]interface{} {
	var result []map[string]interface{}
	for i, c := range Config.Containers {
		name := c.Name
		if name == "" {
			name = fmt.Sprintf("容器%d", i+1)
		}
		// 兼容：无 type 字段时默认视为 ql 类型（青龙容器）
		containerType := c.Type
		if containerType == "" {
			containerType = "ql"
		}
		result = append(result, map[string]interface{}{
			"name":      name,
			"address":   c.Address,
			"available": containerType == "ql",
			"index":     i,
		})
	}
	return result
}

// getJdContainerClientByIndex 根据容器序号获取青龙客户端
func getJdContainerClientByIndex(idx int) (*QingLongClient, string, error) {
	if idx < 0 || idx >= len(Config.Containers) {
		return nil, "", fmt.Errorf("容器序号 %d 超出范围", idx)
	}
	c := Config.Containers[idx]
	// 兼容：无 type 字段时默认视为 ql 类型
	containerType := c.Type
	if containerType == "" {
		containerType = "ql"
	}
	if containerType != "ql" {
		return nil, "", fmt.Errorf("容器 %d 不是有效的青龙容器", idx)
	}
	name := c.Name
	if name == "" {
		name = fmt.Sprintf("容器%d", idx+1)
	}
	cfg := &QingLongConfig{
		Name:         name,
		Host:         c.Address,
		ClientID:     c.Cid,
		ClientSecret: c.Secret,
		Timeout:      30,
	}
	return NewQingLongClient(cfg), name, nil
}

// ===================== 京东容器 Cron 任务管理 =====================

// GetJdCronTasksAdmin 获取京东容器定时任务列表
func GetJdCronTasksAdmin(containerIdx int, searchValue string) ([]QLCronTask, error) {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return nil, err
	}
	// 获取所有任务（不传搜索词给青龙API），本地搜索所有字段
	tasks, err := client.QueryCronTasks("")
	if err != nil {
		return nil, err
	}
	if searchValue != "" {
		search := strings.ToLower(searchValue)
		var filtered []QLCronTask
		for _, t := range tasks {
			if strings.Contains(strings.ToLower(t.Name), search) ||
				strings.Contains(strings.ToLower(t.Command), search) ||
				strings.Contains(strings.ToLower(t.Extra), search) ||
				strings.Contains(strings.ToLower(t.Cron), search) ||
				strings.Contains(strings.ToLower(t.Schedule), search) ||
				strings.Contains(fmt.Sprintf("%d", t.ID), search) {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	enrichCronTasksWithLatestRun(client, tasks)
	return tasks, nil
}

// GetJdCronTaskLogsAdmin 获取京东容器任务日志列表
func GetJdCronTaskLogsAdmin(containerIdx, taskID, page, limit int) ([]QLCronTaskLog, int, error) {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return nil, 0, err
	}
	logs, total, err := client.GetCronTaskLogs(taskID, page, limit)
	if err != nil {
		return nil, 0, err
	}
	for i := range logs {
		normalizeCronTaskLog(&logs[i])
	}
	enrichCronTaskLogsWithFiles(client, taskID, logs)
	return logs, total, nil
}

// GetJdCronTaskLogContentAdmin 获取京东容器任务日志内容（通过 taskId）
// logFile 参数可选，用于获取指定历史日志
func GetJdCronTaskLogContentAdmin(containerIdx, taskID int, logFile ...string) (string, error) {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return "", err
	}
	// 如果指定了日志文件名，复用青龙的历史日志获取逻辑
	if len(logFile) > 0 && logFile[0] != "" {
		fileName := logFile[0]
		if !strings.HasSuffix(fileName, ".log") {
			fileName += ".log"
		}
		// 方式1：通过任务级 API
		content, err := client.GetCronTaskLogContent(taskID, fileName)
		if err == nil && content != "" {
			return content, nil
		}
		// 方式2：通过文件系统 API
		for _, logPath := range []string{fmt.Sprintf("/log/%d", taskID), fmt.Sprintf("log/%d", taskID)} {
			content, err = client.GetLogFileContent(logPath, fileName)
			if err == nil && content != "" {
				return content, nil
			}
		}
		// 方式3：模糊匹配
		prefix := fileName[:len(fileName)-4]
		files, listErr := client.GetTaskLogFiles(taskID)
		if listErr == nil {
			for _, f := range files {
				if f.Type == "file" {
					fName := f.Key
					if idx := strings.LastIndex(fName, "/"); idx >= 0 {
						fName = fName[idx+1:]
					}
					if strings.HasPrefix(fName, prefix) {
						content, err = client.GetLogFileContent(fmt.Sprintf("/log/%d", taskID), fName)
						if err == nil && content != "" {
							return content, nil
						}
					}
				}
			}
		}
		System().Infof("JD容器获取历史日志失败(file=%s)，不再fallback到最新日志", fileName)
		return "", fmt.Errorf("未找到指定历史日志: %s，可能已被清理或当前青龙版本不支持按文件读取", fileName)
	}
	return client.GetCronTaskLogContent(taskID)
}

// GetJdCronTaskScriptAdmin 获取京东容器任务脚本
func GetJdCronTaskScriptAdmin(containerIdx, taskID int) (string, error) {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return "", err
	}
	return client.GetCronTaskScript(taskID)
}

// UpdateJdCronTaskScriptAdmin 更新京东容器任务脚本
func UpdateJdCronTaskScriptAdmin(containerIdx, taskID int, command string) error {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return err
	}
	return client.UpdateCronTaskScript(taskID, command)
}

// GetJdCronTaskScriptFileAdmin 获取京东容器任务脚本文件内容
func GetJdCronTaskScriptFileAdmin(containerIdx, taskID int) (string, string, error) {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return "", "", err
	}
	command, err := client.GetCronTaskScript(taskID)
	if err != nil {
		return "", "", err
	}
	scriptPath := parseScriptPath(command)
	if scriptPath.Filename == "" {
		return "", command, nil
	}
	content, err := client.GetScriptFileContent(scriptPath.Path, scriptPath.Filename)
	if err != nil {
		return "", command, fmt.Errorf("获取脚本文件内容失败: %v（任务命令: %s，解析路径: path=%s file=%s）", err, command, scriptPath.Path, scriptPath.Filename)
	}
	if strings.TrimSpace(content) == "" {
		return "", command, fmt.Errorf("脚本文件内容为空，疑似未命中真实脚本路径（任务命令: %s，解析路径: path=%s file=%s）", command, scriptPath.Path, scriptPath.Filename)
	}
	return content, command, nil
}

// UpdateJdCronTaskScriptFileAdmin 更新京东容器任务脚本文件内容
func UpdateJdCronTaskScriptFileAdmin(containerIdx, taskID int, content string) error {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return err
	}
	command, err := client.GetCronTaskScript(taskID)
	if err != nil {
		return err
	}
	scriptPath := parseScriptPath(command)
	if scriptPath.Filename == "" {
		return fmt.Errorf("无法从任务命令解析脚本文件名: %s", command)
	}
	return client.UpdateScriptFileContent(scriptPath.Path, scriptPath.Filename, content)
}

// UpdateJdCronTaskScheduleAdmin 更新京东容器任务定时规则
func UpdateJdCronTaskScheduleAdmin(containerIdx, taskID int, cronExpr string) error {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return err
	}
	return client.UpdateCronTaskSchedule(taskID, cronExpr)
}

// EnableJdCronTaskAdmin 启用京东容器任务
func EnableJdCronTaskAdmin(containerIdx int, taskIDs []int) error {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return err
	}
	return client.EnableCronTask(taskIDs)
}

// DisableJdCronTaskAdmin 禁用京东容器任务
func DisableJdCronTaskAdmin(containerIdx int, taskIDs []int) error {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return err
	}
	return client.DisableCronTask(taskIDs)
}

// DeleteJdCronTaskAdmin 删除京东容器任务
func DeleteJdCronTaskAdmin(containerIdx int, taskIDs []int) error {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return err
	}
	return client.DeleteCronTask(taskIDs)
}

// RunJdCronTaskAdmin 手动运行京东容器任务
func RunJdCronTaskAdmin(containerIdx int, taskIDs []int) error {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return err
	}
	return client.RunCronTask(taskIDs)
}

// StopJdCronTaskAdmin 停止京东容器任务
func StopJdCronTaskAdmin(containerIdx int, taskIDs []int) error {
	client, _, err := getJdContainerClientByIndex(containerIdx)
	if err != nil {
		return err
	}
	return client.StopCronTask(taskIDs)
}

// ===================== 京东容器 CRUD（config.yaml 操作） =====================

// parseJdContainersFromYAML 从 config.yaml 内容解析京东容器列表
func parseJdContainersFromYAML(content string) []Container {
	var containers []Container
	lines := strings.Split(content, "\n")
	inContainers := false
	var current *Container

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "containers:") {
			inContainers = true
			continue
		}
		if inContainers {
			if trimmed != "" && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "#") {
				if current != nil {
					containers = append(containers, *current)
					current = nil
				}
				inContainers = false
				continue
			}
			if strings.HasPrefix(trimmed, "- address:") {
				if current != nil {
					containers = append(containers, *current)
				}
				current = &Container{}
				addr := strings.TrimSpace(strings.TrimPrefix(trimmed, "- address:"))
				if idx := strings.Index(addr, "#"); idx > 0 {
					addr = strings.TrimSpace(addr[:idx])
				}
				current.Address = addr
			} else if current != nil {
				if strings.HasPrefix(trimmed, "cid:") {
					val := strings.TrimSpace(strings.TrimPrefix(trimmed, "cid:"))
					if idx := strings.Index(val, "#"); idx > 0 {
						val = strings.TrimSpace(val[:idx])
					}
					current.Cid = val
				} else if strings.HasPrefix(trimmed, "secret:") {
					val := strings.TrimSpace(strings.TrimPrefix(trimmed, "secret:"))
					if idx := strings.Index(val, "#"); idx > 0 {
						val = strings.TrimSpace(val[:idx])
					}
					current.Secret = val
				} else if strings.HasPrefix(trimmed, "weigth:") || strings.HasPrefix(trimmed, "weight:") {
					val := strings.TrimSpace(strings.TrimPrefix(trimmed, "weigth:"))
					if strings.HasPrefix(trimmed, "weight:") {
						val = strings.TrimSpace(strings.TrimPrefix(trimmed, "weight:"))
					}
					if idx := strings.Index(val, "#"); idx > 0 {
						val = strings.TrimSpace(val[:idx])
					}
					w, _ := strconv.Atoi(val)
					current.Weigth = w
				} else if strings.HasPrefix(trimmed, "mode:") {
					val := strings.TrimSpace(strings.TrimPrefix(trimmed, "mode:"))
					if idx := strings.Index(val, "#"); idx > 0 {
						val = strings.TrimSpace(val[:idx])
					}
					current.Mode = val
				} else if strings.HasPrefix(trimmed, "limit:") {
					val := strings.TrimSpace(strings.TrimPrefix(trimmed, "limit:"))
					if idx := strings.Index(val, "#"); idx > 0 {
						val = strings.TrimSpace(val[:idx])
					}
					l, _ := strconv.Atoi(val)
					current.Limit = l
				} else if strings.HasPrefix(trimmed, "resident:") {
					val := strings.TrimSpace(strings.TrimPrefix(trimmed, "resident:"))
					if idx := strings.Index(val, "#"); idx > 0 {
						val = strings.TrimSpace(val[:idx])
					}
					current.Resident = val
				}
			}
		}
	}
	if current != nil {
		containers = append(containers, *current)
	}
	return containers
}

// rebuildConfigYAML 重建 config.yaml 的 containers 段
func rebuildConfigYAML(beforeLines, afterLines []string, containers []Container) string {
	newContainersYAML := "containers:\n"
	for _, c := range containers {
		newContainersYAML += fmt.Sprintf("  - address: %s\n    cid: %s\n    secret: %s\n    weigth: %d\n    mode: %s\n    limit: %d",
			c.Address, c.Cid, c.Secret, c.Weigth, c.Mode, c.Limit)
		if c.Resident != "" {
			newContainersYAML += "\n    resident: " + c.Resident
		}
		newContainersYAML += "\n\n"
	}
	return strings.Join(beforeLines, "\n") + "\n" + newContainersYAML + strings.Join(afterLines, "\n")
}

// splitConfigAroundContainers 将 config.yaml 按 containers 段拆分为前后两部分
func splitConfigAroundContainers(content string) (beforeLines []string, afterLines []string) {
	lines := strings.Split(content, "\n")
	inContainers := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "containers:") && !inContainers {
			inContainers = true
			continue
		}
		if inContainers {
			if trimmed != "" && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "#") {
				inContainers = false
				afterLines = append(afterLines, line)
			}
			continue
		}
		beforeLines = append(beforeLines, line)
	}
	return
}

// AddJdContainer 添加京东容器到 config.yaml
func AddJdContainer(address, cid, secret string, weight, limit int, mode, resident string) error {
	data, err := ioutil.ReadFile(ExecPath + "/conf/config.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	newBlock := fmt.Sprintf("\n  - address: %s\n    cid: %s\n    secret: %s\n    weigth: %d\n    mode: %s\n    limit: %d", address, cid, secret, weight, mode, limit)
	if resident != "" {
		newBlock += "\n    resident: " + resident
	}

	content := string(data)
	lastIdx := strings.LastIndex(content, "  - address:")
	if lastIdx == -1 {
		containersIdx := strings.Index(content, "containers:")
		if containersIdx == -1 {
			return fmt.Errorf("找不到 containers 配置段")
		}
		nlIdx := strings.Index(content[containersIdx:], "\n")
		if nlIdx == -1 {
			return fmt.Errorf("配置文件格式错误")
		}
		insertPos := containersIdx + nlIdx + 1
		content = content[:insertPos] + newBlock + "\n" + content[insertPos:]
	} else {
		afterLast := content[lastIdx:]
		nextEntry := strings.Index(afterLast[1:], "\n  - ")
		nextTopLevel := -1
		for i, ch := range afterLast[1:] {
			if ch == '\n' && i+2 < len(afterLast[1:]) {
				rest := afterLast[1+i+1:]
				if len(rest) > 0 && rest[0] != ' ' && rest[0] != '#' && rest[0] != '\n' {
					nextTopLevel = i + 1
					break
				}
			}
		}
		var endPos int
		if nextEntry != -1 && (nextTopLevel == -1 || nextEntry < nextTopLevel) {
			endPos = lastIdx + 1 + nextEntry + 1
		} else if nextTopLevel != -1 {
			endPos = lastIdx + 1 + nextTopLevel + 1
		} else {
			endPos = len(content)
		}
		content = content[:endPos] + newBlock + "\n" + content[endPos:]
	}

	return ioutil.WriteFile(ExecPath+"/conf/config.yaml", []byte(content), 0644)
}

// UpdateJdContainer 更新指定索引的京东容器
func UpdateJdContainer(index int, address, cid, secret string, weight, limit int, mode, resident string) error {
	data, err := ioutil.ReadFile(ExecPath + "/conf/config.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	containers := parseJdContainersFromYAML(string(data))
	if index < 0 || index >= len(containers) {
		return fmt.Errorf("无效的容器索引: %d", index)
	}

	containers[index].Address = address
	containers[index].Cid = cid
	containers[index].Secret = secret
	containers[index].Weigth = weight
	containers[index].Mode = mode
	containers[index].Limit = limit
	containers[index].Resident = resident

	beforeLines, afterLines := splitConfigAroundContainers(string(data))
	newContent := rebuildConfigYAML(beforeLines, afterLines, containers)
	return ioutil.WriteFile(ExecPath+"/conf/config.yaml", []byte(newContent), 0644)
}

// DeleteJdContainer 删除指定索引的京东容器
func DeleteJdContainer(index int) error {
	data, err := ioutil.ReadFile(ExecPath + "/conf/config.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	containers := parseJdContainersFromYAML(string(data))
	if index < 0 || index >= len(containers) {
		return fmt.Errorf("无效的容器索引: %d", index)
	}

	containers = append(containers[:index], containers[index+1:]...)

	beforeLines, afterLines := splitConfigAroundContainers(string(data))
	newContent := rebuildConfigYAML(beforeLines, afterLines, containers)
	return ioutil.WriteFile(ExecPath+"/conf/config.yaml", []byte(newContent), 0644)
}

// ===================== 青龙活动容器 CRUD（activities.yaml 操作） =====================

// AddQLConfig 添加青龙配置到 activities.yaml
func AddQLConfig(name, host, clientID, clientSecret string, timeout int) error {
	data, err := ioutil.ReadFile(ExecPath + "/conf/activities.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	newBlock := fmt.Sprintf("  - 名称: \"%s\"\n    地址: \"%s\"\n    客户端ID: \"%s\"\n    客户端密钥: \"%s\"\n    超时秒数: %d\n", name, host, clientID, clientSecret, timeout)

	content := string(data)
	lines := strings.Split(content, "\n")
	lastIdx := -1
	inQLConfigs := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "qinglong_configs:") {
			inQLConfigs = true
			continue
		}
		if inQLConfigs && trimmed != "" && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "#") {
			break
		}
		if inQLConfigs && strings.HasPrefix(line, "  - 名称:") {
			lastIdx = i
		}
	}

	if lastIdx == -1 {
		qlIdx := strings.Index(content, "qinglong_configs:")
		if qlIdx == -1 {
			return fmt.Errorf("找不到 qinglong_configs 配置段")
		}
		nlIdx := strings.Index(content[qlIdx:], "\n")
		if nlIdx == -1 {
			return fmt.Errorf("配置文件格式错误")
		}
		insertPos := qlIdx + nlIdx + 1
		content = content[:insertPos] + newBlock + "\n" + content[insertPos:]
	} else {
		endIdx := lastIdx + 1
		for i := lastIdx + 1; i < len(lines); i++ {
			trimmed := strings.TrimSpace(lines[i])
			if trimmed == "" || strings.HasPrefix(lines[i], "    ") {
				endIdx = i + 1
			} else {
				break
			}
		}
		linePos := 0
		for i := 0; i < endIdx; i++ {
			linePos += len(lines[i]) + 1
		}
		content = content[:linePos] + newBlock + "\n" + content[linePos:]
	}

	err = ioutil.WriteFile(ExecPath+"/conf/activities.yaml", []byte(content), 0644)
	if err != nil {
		return err
	}
	ReloadActivities()
	InvalidateActivityAdminStatsCache()
	go RefreshActivityAdminStatsCache()
	return nil
}

// UpdateQLConfig 更新青龙配置（名称不可修改）
func UpdateQLConfig(oldName, host, clientID, clientSecret string, timeout int) error {
	data, err := ioutil.ReadFile(ExecPath + "/conf/activities.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	inQLConfigs := false
	found := false
	var newLines []string
	skipUntilNext := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "qinglong_configs:") {
			inQLConfigs = true
			newLines = append(newLines, line)
			continue
		}
		if inQLConfigs && trimmed != "" && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "#") {
			inQLConfigs = false
		}
		if skipUntilNext {
			if trimmed == "" || strings.HasPrefix(line, "  - ") {
				skipUntilNext = false
			} else {
				continue
			}
		}
		if inQLConfigs && strings.HasPrefix(trimmed, "- 名称:") {
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "- 名称:"))
			name = strings.Trim(name, "\"")
			if name == oldName {
				found = true
				newLines = append(newLines, fmt.Sprintf("  - 名称: \"%s\"", oldName))
				newLines = append(newLines, fmt.Sprintf("    地址: \"%s\"", host))
				newLines = append(newLines, fmt.Sprintf("    客户端ID: \"%s\"", clientID))
				newLines = append(newLines, fmt.Sprintf("    客户端密钥: \"%s\"", clientSecret))
				newLines = append(newLines, fmt.Sprintf("    超时秒数: %d", timeout))
				skipUntilNext = true
				continue
			}
		}
		newLines = append(newLines, line)
	}

	if !found {
		return fmt.Errorf("未找到名称为 %s 的青龙配置", oldName)
	}

	err = ioutil.WriteFile(ExecPath+"/conf/activities.yaml", []byte(strings.Join(newLines, "\n")), 0644)
	if err != nil {
		return err
	}
	ReloadActivities()
	return nil
}

// DeleteQLConfig 删除青龙配置
func DeleteQLConfig(name string) error {
	data, err := ioutil.ReadFile(ExecPath + "/conf/activities.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	inQLConfigs := false
	found := false
	var newLines []string
	skipEntry := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "qinglong_configs:") {
			inQLConfigs = true
			newLines = append(newLines, line)
			continue
		}
		if inQLConfigs && trimmed != "" && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "#") {
			inQLConfigs = false
		}
		if skipEntry {
			if trimmed == "" || strings.HasPrefix(line, "  - ") {
				skipEntry = false
			} else {
				continue
			}
		}
		if inQLConfigs && strings.HasPrefix(trimmed, "- 名称:") {
			entryName := strings.TrimSpace(strings.TrimPrefix(trimmed, "- 名称:"))
			entryName = strings.Trim(entryName, "\"")
			if entryName == name {
				found = true
				skipEntry = true
				continue
			}
		}
		newLines = append(newLines, line)
	}

	if !found {
		return fmt.Errorf("未找到名称为 %s 的青龙配置", name)
	}

	// 清理尾部多余空行
	for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
		newLines = newLines[:len(newLines)-1]
	}
	newLines = append(newLines, "")

	err = ioutil.WriteFile(ExecPath+"/conf/activities.yaml", []byte(strings.Join(newLines, "\n")), 0644)
	if err != nil {
		return err
	}
	ReloadActivities()
	return nil
}

// ===================== 用户管理扩展 =====================

// CreateUser 创建新用户
func CreateUser(wxid, qq string, coin int, isAdmin bool, nickname string) error {
	user := User{
		Wxid:     wxid,
		QQ:       qq,
		Coin:     coin,
		IsAdmin:  isAdmin,
		Nickname: nickname,
	}
	var maxNum int
	db.Model(&User{}).Select("COALESCE(MAX(number), 0)").Scan(&maxNum)
	user.Number = maxNum + 1
	return db.Create(&user).Error
}

// UpdateUser 更新用户信息（除 ID 外所有字段）
func UpdateUser(id int, wxid, qq string, coin int, class string, isAdmin bool, nickname string) error {
	updates := map[string]interface{}{
		"wxid":     wxid,
		"qq":       qq,
		"coin":     coin,
		"class":    class,
		"isAdmin":  isAdmin,
		"nickname": nickname,
	}
	return db.Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

// BatchDeleteUsers 批量删除用户
func BatchDeleteUsers(ids []int) error {
	return db.Where("id IN ?", ids).Delete(&User{}).Error
}

// BatchUpdateUserCoins 批量修改用户积分
func BatchUpdateUserCoins(numbers []int, coin int) error {
	return db.Model(&User{}).Where("number IN ?", numbers).Update("coin", coin).Error
}

// ===================== 批量操作（京东CK、环境变量、青龙变量） =====================

// BatchDeleteJdCookies 批量删除京东CK
func BatchDeleteJdCookies(ids []int) error {
	return db.Where("ID IN ?", ids).Delete(&JdCookie{}).Error
}

// BatchUpdateJdCookies 批量修改京东CK
func BatchUpdateJdCookies(ids []int, priority int, available, note string) error {
	updates := map[string]interface{}{}
	if priority >= 0 {
		updates["Priority"] = priority
	}
	if available != "" {
		updates["Available"] = available
	}
	if note != "" {
		updates["Note"] = note
	}
	if len(updates) == 0 {
		return fmt.Errorf("没有需要更新的字段")
	}
	return db.Model(&JdCookie{}).Where("ID IN ?", ids).Updates(updates).Error
}

// BatchDeleteEnvVars 批量删除xdd环境变量
func BatchDeleteEnvVars(ids []int) error {
	return db.Where("id IN ?", ids).Delete(&Env{}).Error
}

// CreateQLEnv 创建青龙环境变量
func CreateQLEnv(configName, name, value, remarks string) error {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()
	if !exists {
		return fmt.Errorf("没有可用的青龙配置: %s", configName)
	}
	client := NewQingLongClient(cfg)
	return client.SubmitEnv(name, value, remarks)
}

// BatchDeleteQLEnvs 批量删除青龙环境变量
func BatchDeleteQLEnvs(configName string, envIDs []int) error {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()
	if !exists {
		return fmt.Errorf("没有可用的青龙配置: %s", configName)
	}
	client := NewQingLongClient(cfg)
	var lastErr error
	for _, id := range envIDs {
		if err := client.DeleteEnv(id); err != nil {
			lastErr = SanitizeError(err)
		}
	}
	return lastErr
}

// BatchUpdateQLEnvs 批量修改青龙环境变量备注
func BatchUpdateQLEnvs(configName string, envIDs []int, remarks string) error {
	qlManager.mu.RLock()
	cfg, exists := qlManager.Configs[configName]
	qlManager.mu.RUnlock()
	if !exists {
		return fmt.Errorf("没有可用的青龙配置: %s", configName)
	}
	client := NewQingLongClient(cfg)
	var lastErr error
	for _, id := range envIDs {
		envs, err := client.QueryEnvs("")
		if err != nil {
			lastErr = SanitizeError(err)
			continue
		}
		for _, env := range envs {
			if env.ID == id {
				if err := client.UpdateEnv(id, env.Name, env.Value, remarks); err != nil {
					lastErr = SanitizeError(err)
				}
				break
			}
		}
	}
	return lastErr
}

// ===================== 活动人数统计 =====================

// ActivityStat 单个活动的统计信息
type ActivityStat struct {
	ActivityID      string `json:"activityId"`
	ActivityName    string `json:"activityName"`
	DisplayOrder    int    `json:"displayOrder"`
	QLConfig        string `json:"qlConfig"`
	Enabled         bool   `json:"enabled"`
	IsMonthlyDeduct bool   `json:"isMonthlyDeduct"`
	NeedCoin        int    `json:"needCoin"`
	MonthlyCoin     int    `json:"monthlyCoin"`
	IsDailyDeduct   bool   `json:"isDailyDeduct"`
	DailyCoin       int    `json:"dailyCoin"`
	Total           int    `json:"total"`
	Valid           int    `json:"valid"`
	ExpiringSoon    int    `json:"expiringSoon"`
	Expired         int    `json:"expired"`
	Disabled        int    `json:"disabled"`
}

func sortConfigsForAdminDisplay(configs []ActivityConfig) {
	sort.SliceStable(configs, func(i, j int) bool {
		if configs[i].Enabled != configs[j].Enabled {
			return configs[i].Enabled
		}
		if !configs[i].Enabled {
			return configs[i].Name < configs[j].Name
		}
		if configs[i].DisplayOrder != configs[j].DisplayOrder {
			return configs[i].DisplayOrder < configs[j].DisplayOrder
		}
		return configs[i].Name < configs[j].Name
	})
}

func GetActivityStats() ([]ActivityStat, int, int, int, int) {
	return buildActivityStats()
}

func InvalidateActivityAdminStatsCache() {}

func RefreshActivityAdminStatsCache(forceRefresh ...bool) {}

type ActivityStatsSnapshot struct {
	Stats       []ActivityStat `json:"list"`
	TotalAll    int            `json:"totalAll"`
	ValidAll    int            `json:"validAll"`
	ExpiringAll int            `json:"expiringAll"`
	ExpiredAll  int            `json:"expiredAll"`
	Cached      bool           `json:"cached"`
	Refreshing  bool           `json:"refreshing"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

func GetActivityStatsSnapshot(force bool) ActivityStatsSnapshot {
	stats, totalAll, validAll, expiringAll, expiredAll := buildActivityStats()
	return ActivityStatsSnapshot{
		Stats:       stats,
		TotalAll:    totalAll,
		ValidAll:    validAll,
		ExpiringAll: expiringAll,
		ExpiredAll:  expiredAll,
		Cached:      false,
		Refreshing:  false,
		UpdatedAt:   time.Now(),
	}
}

func RefreshSingleActivityAdminStats(activityID string) error {
	return nil
}

func buildActivityStats() ([]ActivityStat, int, int, int, int) {
	activityConfigsMu.RLock()
	configs := make([]ActivityConfig, 0, len(ActivityConfigs))
	for _, cfg := range ActivityConfigs {
		if cfg != nil {
			configs = append(configs, *cfg)
		}
	}
	activityConfigsMu.RUnlock()
	sortConfigsForAdminDisplay(configs)

	stats := make([]ActivityStat, len(configs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, cfg := range configs {
		wg.Add(1)
		go func(i int, cfg ActivityConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			stats[i] = buildActivityStat(cfg)
		}(i, cfg)
	}
	wg.Wait()
	totalAll, validAll, expiringAll, expiredAll := summarizeActivityStats(stats)
	return stats, totalAll, validAll, expiringAll, expiredAll
}

func summarizeActivityStats(stats []ActivityStat) (int, int, int, int) {
	totalAll := 0
	validAll := 0
	expiringAll := 0
	expiredAll := 0
	for _, stat := range stats {
		totalAll += stat.Total
		validAll += stat.Valid
		expiringAll += stat.ExpiringSoon
		expiredAll += stat.Expired
	}
	return totalAll, validAll, expiringAll, expiredAll
}

func buildActivityStat(cfg ActivityConfig) ActivityStat {
	stat := ActivityStat{
		ActivityID: cfg.ID, ActivityName: cfg.Name, DisplayOrder: cfg.DisplayOrder,
		QLConfig: cfg.QingLongConfigName, Enabled: cfg.Enabled,
		IsMonthlyDeduct: cfg.IsMonthlyDeduct, NeedCoin: cfg.NeedCoin, MonthlyCoin: cfg.MonthlyCoin,
		IsDailyDeduct: cfg.IsDailyDeduct, DailyCoin: cfg.DailyCoin,
	}

	projects, err := GetActivityProjectsByActivityID(cfg.ID)
	if err != nil || len(projects) == 0 {
		return stat
	}

	now := time.Now()
	for _, project := range projects {
		stat.Total++
		if project.Status == 0 {
			if (cfg.IsMonthlyDeduct || cfg.IsDailyDeduct) && project.ExpireDate != "" {
				expireDate, parseErr := time.Parse(DateLayout, project.ExpireDate)
				if parseErr == nil {
					expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
					remain := expireThreshold.Sub(now)
					if remain <= 0 {
						stat.Expired++
					} else if remain <= 7*24*time.Hour {
						stat.ExpiringSoon++
						stat.Valid++
					} else {
						stat.Valid++
					}
				} else {
					stat.Valid++
				}
			} else {
				stat.Valid++
			}
		} else if (cfg.IsMonthlyDeduct || cfg.IsDailyDeduct) && project.ExpireDate != "" {
			expireDate, parseErr := time.Parse(DateLayout, project.ExpireDate)
			if parseErr == nil {
				expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
				if now.After(expireThreshold) || now.Equal(expireThreshold) {
					stat.Expired++
				} else {
					stat.Disabled++
				}
			} else {
				stat.Disabled++
			}
		} else {
			stat.Disabled++
		}
	}
	return stat
}

// ===================== 青龙活动授权管理 =====================

// ActivityAuthItem 授权管理列表项
type ActivityAuthItem struct {
	ActivityID      string `json:"activityId"`
	ActivityName    string `json:"activityName"`
	EnvKey          string `json:"envKey"`
	QLConfig        string `json:"qlConfig"`
	DisplayOrder    int    `json:"displayOrder"`
	Enabled         bool   `json:"enabled"`
	NeedCoin        int    `json:"needCoin"`
	MonthlyCoin     int    `json:"monthlyCoin"`
	IsMonthlyDeduct bool   `json:"isMonthlyDeduct"`
	IsDailyDeduct   bool   `json:"isDailyDeduct"`
	DailyCoin       int    `json:"dailyCoin"`
	Total           int    `json:"total"`
	Valid           int    `json:"valid"`
	Expired         int    `json:"expired"`
}

type ActivityAuthAccountItem struct {
	// EnvID 实际为数据库 activity_project.id（前端勾选/批量操作用），不再使用青龙 EnvID
	EnvID        int    `json:"envId"`
	Remarks      string `json:"remarks"`
	AccountAlias string `json:"accountAlias"`
	UserNumber   int    `json:"userNumber"`
	ExpireDate   string `json:"expireDate"`
	RemainDays   int    `json:"remainDays"`
	RefundCoin   int    `json:"refundCoin"`
	NeedCoin     int    `json:"needCoin"`
	Status       int    `json:"status"`
	StatusText   string `json:"statusText"`
}

func GetActivityAuthListSnapshot(force bool) ([]ActivityAuthItem, bool, bool, time.Time) {
	list := buildActivityAuthList()
	return list, len(list) > 0, false, time.Now()
}

func GetActivityAuthList() []ActivityAuthItem {
	list, _, _, _ := GetActivityAuthListSnapshot(false)
	return list
}

// GetActivityAuthItemByID 获取单个活动的授权统计（删除后局部刷新用）
func GetActivityAuthItemByID(activityID string) (ActivityAuthItem, bool) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return ActivityAuthItem{}, false
	}
	return buildActivityAuthItem(*cfg), true
}

func buildActivityAuthList() []ActivityAuthItem {
	activityConfigsMu.RLock()
	configs := make([]ActivityConfig, 0, len(ActivityConfigs))
	for _, cfg := range ActivityConfigs {
		if cfg != nil {
			configs = append(configs, *cfg)
		}
	}
	activityConfigsMu.RUnlock()
	sortConfigsForAdminDisplay(configs)
	list := make([]ActivityAuthItem, len(configs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, cfg := range configs {
		wg.Add(1)
		go func(i int, cfg ActivityConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			list[i] = buildActivityAuthItem(cfg)
		}(i, cfg)
	}
	wg.Wait()
	return list
}

func buildActivityAuthItem(cfg ActivityConfig) ActivityAuthItem {
	item := ActivityAuthItem{
		ActivityID: cfg.ID, ActivityName: cfg.Name, EnvKey: cfg.EnvKey,
		QLConfig: cfg.QingLongConfigName, DisplayOrder: cfg.DisplayOrder, Enabled: cfg.Enabled,
		NeedCoin: cfg.NeedCoin, MonthlyCoin: cfg.MonthlyCoin, IsMonthlyDeduct: cfg.IsMonthlyDeduct,
		IsDailyDeduct: cfg.IsDailyDeduct, DailyCoin: cfg.DailyCoin,
	}

	projects, err := GetActivityProjectsByActivityID(cfg.ID)
	if err != nil || len(projects) == 0 {
		return item
	}

	now := time.Now()
	for _, project := range projects {
		item.Total++
		if project.Status == 0 {
			if (cfg.IsMonthlyDeduct || cfg.IsDailyDeduct) && project.ExpireDate != "" {
				expireDate, parseErr := time.Parse(DateLayout, project.ExpireDate)
				if parseErr == nil {
					expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
					if now.After(expireThreshold) || now.Equal(expireThreshold) {
						item.Expired++
					} else {
						item.Valid++
					}
				} else {
					item.Valid++
				}
			} else {
				item.Valid++
			}
		} else {
			item.Expired++
		}
	}
	return item
}

func GetActivityAuthAccounts(activityID string) ([]ActivityAuthAccountItem, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return nil, fmt.Errorf("活动不存在")
	}

	projects, err := GetActivityProjectsByActivityID(activityID)
	if err != nil {
		return nil, fmt.Errorf("查询数据库失败：%v", err)
	}

	items := make([]ActivityAuthAccountItem, 0, len(projects))
	for _, project := range projects {
		item := buildActivityAuthAccountItemByProject(cfg, &project)
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return items[i].Status < items[j].Status
		}
		return items[i].RemainDays > items[j].RemainDays
	})
	return items, nil
}

func buildActivityAuthAccountItemByProject(cfg *ActivityConfig, project *ActivityProject) ActivityAuthAccountItem {
	accountAlias := project.RemarkAlias
	if accountAlias == "" {
		accountAlias = project.Remarks
	}
	expireDate := project.ExpireDate
	remainDays := 0
	refundCoin := 0
	if expireDate != "" {
		if d, parseErr := time.Parse(DateLayout, expireDate); parseErr == nil {
			now := time.Now()
			expireThreshold := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
			remainDays = int(math.Ceil(expireThreshold.Sub(now).Hours() / 24))
			if remainDays < 0 {
				remainDays = 0
			}
			if remainDays > 0 {
				remainDays = remainDays - 1
			}
			paidDays := CalcPaidRemainingDays(project)
			if project.DailyCoin > 0 && project.NeedCoin == 0 {
				refundCoin = project.DailyCoin * paidDays
			} else if project.MonthlyCoin > 0 && project.NeedCoin == 0 {
				refundCoin = int(math.Round(float64(project.MonthlyCoin) * float64(paidDays) / 30))
			}
		}
	}
	statusText := "正常"
	if !cfg.IsMonthlyDeduct && !cfg.IsDailyDeduct && expireDate == "" {
		statusText = "一次性"
	} else if project.Status != 0 {
		statusText = "已禁用"
	} else if remainDays <= 0 && expireDate != "" {
		statusText = "已到期"
	}
	return ActivityAuthAccountItem{
		EnvID:        project.ID,
		Remarks:      project.Remarks,
		AccountAlias: accountAlias,
		UserNumber:   project.UserNumber,
		ExpireDate:   expireDate,
		RemainDays:   remainDays,
		RefundCoin:   refundCoin,
		NeedCoin:     project.NeedCoin,
		Status:       project.Status,
		StatusText:   statusText,
	}
}

func DeleteActivityAuthAccount(activityID string, envID int, reason string, channels NotifyChannels) (int, error) {
	deletedCount, refundCoin, err := DeleteActivityAuthAccounts(activityID, []int{envID}, reason, channels, nil)
	if err != nil {
		return 0, err
	}
	if deletedCount == 0 {
		return 0, fmt.Errorf("未删除任何账号")
	}
	return refundCoin, nil
}

func DeleteActivityAuthAccounts(activityID string, envIDs []int, reason string, channels NotifyChannels, manualRefundCoins map[int]int) (int, int, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return 0, 0, fmt.Errorf("活动不存在")
	}
	isMonthly := cfg.IsMonthlyDeduct || cfg.IsDailyDeduct
	cleanEnvIDs := make([]int, 0, len(envIDs))
	seen := map[int]bool{}
	for _, envID := range envIDs {
		if envID > 0 && !seen[envID] {
			cleanEnvIDs = append(cleanEnvIDs, envID)
			seen[envID] = true
		}
	}
	if len(cleanEnvIDs) == 0 {
		return 0, 0, fmt.Errorf("账号ID无效")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "管理员删除授权账号"
	}

	projects, err := GetActivityProjectsByActivityID(activityID)
	if err != nil {
		return 0, 0, fmt.Errorf("查询数据库失败：%v", err)
	}
	// 按数据库项目 ID 定位（前端 envId 已改为 project.ID）
	projectMapByID := make(map[int]ActivityProject, len(projects))
	for _, p := range projects {
		projectMapByID[p.ID] = p
	}

	selectedItems := make([]ActivityAuthAccountItem, 0, len(cleanEnvIDs))
	selectedDBIDs := make([]int, 0, len(cleanEnvIDs))
	for _, projectID := range cleanEnvIDs {
		target, ok := projectMapByID[projectID]
		if !ok {
			return 0, 0, fmt.Errorf("未找到要删除的授权账号：%d", projectID)
		}
		selectedItems = append(selectedItems, buildActivityAuthAccountItemByProject(cfg, &target))
		selectedDBIDs = append(selectedDBIDs, target.ID)
	}
	deletedCount := 0
	totalRefundCoin := 0
	notifyPayloads := make([]struct {
		userNumber int
		title      string
		msg        string
	}, 0, len(selectedItems))

	if isMonthly && len(manualRefundCoins) > 0 {
		for i := range selectedItems {
			if coin, ok := manualRefundCoins[selectedItems[i].EnvID]; ok {
				selectedItems[i].RefundCoin = coin
			}
		}
	}

	for i, item := range selectedItems {
		dbID := selectedDBIDs[i]
		refundCoin := 0
		if isMonthly {
			refundCoin = item.RefundCoin
		}
		if err := DeleteProjectWithQinglongSync(dbID); err != nil {
			return deletedCount, totalRefundCoin, fmt.Errorf("删除账号失败（ID=%d）：%v", dbID, err)
		}
		deletedCount++
		if isMonthly && item.UserNumber > 0 && refundCoin > 0 {
			totalRefundCoin += refundCoin
			AdddCoin(item.UserNumber, refundCoin)
			RecordCoinLog(item.UserNumber, refundCoin, "退还", fmt.Sprintf("管理员批量删除%s退还", cfg.Name), AdminContext())
		}
		if item.UserNumber > 0 {
			var msg string
			var title string
			if isMonthly {
				title = "授权账号删除与积分退还通知"
				msg = fmt.Sprintf("📢【授权账号删除与积分退还通知】\n活动：%s\n账号备注：%s\n原到期日：%s\n剩余天数：%d 天\n退还积分：%d\n删除原因：%s\n\n如有疑问请联系管理员。", cfg.Name, item.AccountAlias, item.ExpireDate, item.RemainDays, refundCoin, reason)
			} else {
				title = "授权账号删除通知"
				msg = fmt.Sprintf("📢【授权账号删除通知】\n活动：%s\n账号备注：%s\n说明：该活动为一次性扣费，删除不退还积分\n删除原因：%s\n\n如有疑问请联系管理员。", cfg.Name, item.AccountAlias, reason)
			}
			notifyPayloads = append(notifyPayloads, struct {
				userNumber int
				title      string
				msg        string
			}{userNumber: item.UserNumber, title: title, msg: msg})
		}
	}
	adminMsg := fmt.Sprintf("📢【活动授权账号删除完成】\n活动：%s\n删除账号：%d 个\n退还积分：%d\n原因：%s", cfg.Name, deletedCount, totalRefundCoin, reason)
	adminTitle := "活动授权账号删除完成"
	go func(payloads []struct {
		userNumber int
		title      string
		msg        string
	}, adminTitle, adminMsg string, channels NotifyChannels) {
		for _, p := range payloads {
			if channels.Robot {
				PushByQQ(strconv.Itoa(p.userNumber), p.msg)
			}
			CreateSystemWebNotification(p.title, p.msg, NotifyCategoryAuth, NotifySourceAuth, p.userNumber, channels)
		}
		CreateAdminOnlyWebNotification(adminTitle, adminMsg, NotifyCategoryAuth, NotifySourceAuth, channels)
	}(notifyPayloads, adminTitle, adminMsg, channels)
	InvalidateActivityAdminStatsCache()
	return deletedCount, totalRefundCoin, nil
}

// BatchUpdateActivityAuth 批量增减活动授权天数
// direction: "add" 增加, "sub" 减少
// 返回: updated 成功数, failed 失败数, err 错误
func BatchUpdateActivityAuth(activityID, direction string, days int, envIDs []int, channels NotifyChannels) (int, int, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return 0, 0, fmt.Errorf("活动不存在")
	}
	if !cfg.IsMonthlyDeduct && !cfg.IsDailyDeduct {
		return 0, 0, fmt.Errorf("该活动不是月扣费/按天计费活动，不支持授权管理")
	}

	cleanEnvIDs := make([]int, 0, len(envIDs))
	seen := map[int]bool{}
	for _, envID := range envIDs {
		if envID > 0 && !seen[envID] {
			cleanEnvIDs = append(cleanEnvIDs, envID)
			seen[envID] = true
		}
	}
	if len(cleanEnvIDs) == 0 {
		return 0, 0, fmt.Errorf("账号ID无效")
	}

	projects, err := GetActivityProjectsByActivityID(activityID)
	if err != nil {
		return 0, 0, fmt.Errorf("查询数据库失败：%v", err)
	}
	if len(projects) == 0 {
		return 0, 0, fmt.Errorf("该活动暂无用户数据")
	}

	// 按数据库项目 ID 定位（前端 envId 已改为 project.ID）
	projectMapByID := make(map[int]ActivityProject, len(projects))
	for _, p := range projects {
		projectMapByID[p.ID] = p
	}
	selectedProjects := make([]ActivityProject, 0, len(cleanEnvIDs))
	for _, projectID := range cleanEnvIDs {
		target, ok := projectMapByID[projectID]
		if !ok {
			return 0, 0, fmt.Errorf("未找到要调整的授权账号：%d", projectID)
		}
		selectedProjects = append(selectedProjects, target)
	}

	dirText := "增加"
	if direction == "sub" {
		dirText = "减少"
	}

	updated := 0
	failed := 0
	notifyCount := 0

	for _, project := range selectedProjects {
		if project.ExpireDate == "" {
			failed++
			continue
		}

		oldDate, err := time.Parse(DateLayout, project.ExpireDate)
		if err != nil {
			failed++
			continue
		}

		var newDate time.Time
		if direction == "add" {
			newDate = oldDate.AddDate(0, 0, days)
		} else {
			newDate = oldDate.AddDate(0, 0, -days)
		}

		newDateStr := newDate.Format(DateLayout)

		project.ExpireDate = newDateStr
		project.Remarks = BuildMonthDeductRemarks(project.Remarks, newDateStr)
		project.RemarkAlias = GetFirstRemarkParam(project.Remarks)

		now := time.Now()
		newExpireThreshold := time.Date(newDate.Year(), newDate.Month(), newDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
		if direction == "add" && now.Before(newExpireThreshold) && project.Status != 0 {
			project.Status = 0
			project.SyncStatus = "pending_enable"
		} else if direction == "sub" && (now.After(newExpireThreshold) || now.Equal(newExpireThreshold)) && project.Status == 0 {
			project.Status = 1
			project.SyncStatus = "pending_disable"
		} else {
			project.SyncStatus = "pending_update"
		}
		project.SyncError = ""
		if err := UpdateActivityProject(&project); err != nil {
			System().Infof("[批量改备注] 更新数据库失败，ID=%d，错误：%v", project.ID, err)
			failed++
			continue
		}
		if err := SyncProjectNow(project.ID); err != nil {
			System().Infof("[批量改备注] 青龙同步失败 ID=%d（保持 %s 待重试）: %v", project.ID, project.SyncStatus, err)
		}

		updated++

		accountAlias := project.RemarkAlias
		if accountAlias == "" {
			accountAlias = "未知账号"
		}
		userID := fmt.Sprintf("%d", project.UserNumber)

		if userID != "" {
			msg := fmt.Sprintf(
				"📢【授权时间调整通知】\n"+
					"活动：%s\n"+
					"账号备注：%s\n"+
					"原到期日：%s\n"+
					"新到期日：%s\n"+
					"操作：%s %d 天\n\n"+
					"如有疑问请联系管理员。",
				cfg.Name, accountAlias,
				oldDate.Format(DateLayout),
				newDateStr,
				dirText, days)
			if channels.Robot {
				PushByQQ(userID, msg)
				notifyCount++
				if notifyCount >= 2 {
					time.Sleep(time.Duration(3+rand.Intn(3)) * time.Second)
				}
			}
			if userNumber, err := strconv.Atoi(userID); err == nil {
				CreateSystemWebNotification("授权时间调整通知", msg, NotifyCategoryAuth, NotifySourceAuth, userNumber, channels)
			}
		}
	}

	adminMsg := fmt.Sprintf(
		"📢【活动授权批量操作完成】\n"+
			"活动：%s\n"+
			"操作：%s %d 天\n"+
			"成功：%d 人\n"+
			"失败：%d 人\n"+
			"总计处理：%d 人",
		cfg.Name, dirText, days, updated, failed, updated+failed)
	if channels.Robot {
		go (&JdCookie{}).Push(adminMsg)
	}
	CreateAdminOnlyWebNotification("活动授权批量操作完成", adminMsg, NotifyCategoryAuth, NotifySourceAuth, channels)

	InvalidateActivityAdminStatsCache()
	return updated, failed, nil
}

// ConvertActivityToMonthly 将一次性活动转为月扣费活动，可选同步迁移现有用户
func ConvertActivityToMonthly(activityID string, monthlyCoin int, syncUsers bool, grantDays int, channels NotifyChannels) (int, error) {
	if monthlyCoin <= 0 {
		return 0, fmt.Errorf("每月积分必须大于0")
	}
	if syncUsers && (grantDays < 1 || grantDays > 3650) {
		return 0, fmt.Errorf("授权天数需在1-3650之间")
	}

	cfg := getActivityByID(activityID)
	if cfg == nil {
		return 0, fmt.Errorf("活动不存在")
	}
	if cfg.IsMonthlyDeduct {
		return 0, fmt.Errorf("该活动已经是月扣费活动")
	}

	if err := updateActivityMonthlyFieldsInYaml(cfg.EnvKey, monthlyCoin); err != nil {
		return 0, err
	}
	ReloadActivities()

	migrated := 0
	if !syncUsers {
		InvalidateActivityAdminStatsCache()
		return migrated, nil
	}

	projects, err := GetActivityProjectsByActivityID(activityID)
	if err != nil {
		return migrated, fmt.Errorf("活动配置已更新，但查询用户失败：%v", err)
	}

	now := time.Now()
	for _, project := range projects {
		var newExpire time.Time
		if project.ExpireDate != "" {
			if oldDate, parseErr := time.Parse(DateLayout, project.ExpireDate); parseErr == nil {
				newExpire = oldDate.AddDate(0, 0, grantDays)
			}
		}
		if newExpire.IsZero() {
			newExpire = now.AddDate(0, 0, grantDays)
		}
		newExpireStr := newExpire.Format(DateLayout)
		newRemarks := BuildMonthDeductRemarks(project.Remarks, newExpireStr)

		project.IsMonthlyDeduct = true
		project.MonthlyCoin = monthlyCoin
		project.IsDailyDeduct = false
		project.DailyCoin = 0
		project.MinDays = nil
		project.ExpireDate = newExpireStr
		project.GrantExpireDate = newExpireStr
		project.Remarks = newRemarks
		project.RemarkAlias = GetFirstRemarkParam(newRemarks)
		project.Status = 0
		project.SyncStatus = "pending_update"
		project.SyncError = ""
		if err := UpdateActivityProject(&project); err != nil {
			System().Infof("[转月费] 更新用户项目失败 ID=%d: %v", project.ID, err)
			continue
		}
		go TriggerSync(project.ID)
		migrated++

		if project.UserNumber > 0 {
			accountAlias := project.RemarkAlias
			if accountAlias == "" {
				accountAlias = "未知账号"
			}
			msg := fmt.Sprintf(
				"📢【活动计费方式变更通知】\n"+
					"活动：%s\n"+
					"账号备注：%s\n"+
					"变更：一次性扣费 → 按月扣费（每月%d积分）\n"+
					"授权到期日：%s\n\n"+
					"如有疑问请联系管理员。",
				cfg.Name, accountAlias, monthlyCoin, newExpireStr)
			if channels.Robot {
				PushByQQ(strconv.Itoa(project.UserNumber), msg)
			}
			CreateSystemWebNotification("活动计费方式变更通知", msg, NotifyCategoryAuth, NotifySourceAuth, project.UserNumber, channels)
		}
	}

	adminMsg := fmt.Sprintf("📢【活动转月费完成】\n活动：%s\n每月积分：%d\n同步迁移用户：%d 人\n赠送授权：%d 天", cfg.Name, monthlyCoin, migrated, grantDays)
	CreateAdminOnlyWebNotification("活动转月费完成", adminMsg, NotifyCategoryAuth, NotifySourceAuth, channels)
	InvalidateActivityAdminStatsCache()
	return migrated, nil
}

func updateActivityMonthlyFieldsInYaml(envKey string, monthlyCoin int) error {
	data, err := ioutil.ReadFile(ExecPath + "/conf/activities.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	inActivities := false
	inTarget := false
	found := false
	var newLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "activities:") {
			inActivities = true
			newLines = append(newLines, line)
			continue
		}
		if inActivities && trimmed != "" && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "#") {
			inActivities = false
			inTarget = false
		}
		if inActivities && strings.HasPrefix(trimmed, "- 名称:") {
			inTarget = false
		}
		if inActivities && strings.Contains(line, "环境变量名:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "环境变量名:"))
			val = strings.Trim(val, "\"")
			if val == envKey {
				inTarget = true
				found = true
			}
		}
		if inTarget && strings.HasPrefix(trimmed, "是否按月扣费:") {
			newLines = append(newLines, "    是否按月扣费: true")
			continue
		}
		if inTarget && strings.HasPrefix(trimmed, "每月积分:") {
			newLines = append(newLines, fmt.Sprintf("    每月积分: %d", monthlyCoin))
			continue
		}
		newLines = append(newLines, line)
	}

	if !found {
		return fmt.Errorf("配置文件中未找到环境变量名为 %s 的活动", envKey)
	}

	if err := ioutil.WriteFile(ExecPath+"/conf/activities.yaml", []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	return nil
}

// ===================== 微信协议配置管理 =====================

// GetWxProtocolConfigForAdmin 获取微信协议配置
func GetWxProtocolConfigForAdmin() map[string]interface{} {
	oldTotal, oldOnline, oldOffline := 0, 0, 0
	newTotal, newOnline, newOffline := 0, 0, 0

	// 获取旧地址统计
	oldURL := Config.WxProtocol.LoginBaseURL
	if oldURL != "" {
		oldTotal, oldOnline, oldOffline = getWxDeviceStatsFromURL(oldURL)
	}

	// 获取新地址统计
	newURL := Config.WxProtocol.NewLoginBaseURL
	if newURL != "" {
		newTotal, newOnline, newOffline = getWxDeviceStatsFromURL(newURL)
	}

	// 获取迁移统计
	migrated, notMigrated := GetWxMigrationStats()

	return map[string]interface{}{
		"loginBaseURL":     Config.WxProtocol.LoginBaseURL,
		"newLoginBaseURL":  Config.WxProtocol.NewLoginBaseURL,
		"activeProtocol":   Config.WxProtocol.ActiveProtocol,
		"scanLoginCost":    Config.WxProtocol.ScanLoginCost,
		"deviceName":       Config.WxProtocol.DeviceName,
		"oldStats": map[string]interface{}{
			"total":   oldTotal,
			"online":  oldOnline,
			"offline": oldOffline,
		},
		"newStats": map[string]interface{}{
			"total":   newTotal,
			"online":  newOnline,
			"offline": newOffline,
		},
		"migrationStats": map[string]interface{}{
			"migrated":     migrated,
			"notMigrated":  notMigrated,
		},
	}
}

// SaveWxProtocolConfigForAdmin 保存微信协议配置
func SaveWxProtocolConfigForAdmin(req map[string]interface{}) string {
	data, err := ioutil.ReadFile(ExecPath + "/conf/config.yaml")
	if err != nil {
		return "读取配置文件失败: " + err.Error()
	}

	lines := strings.Split(string(data), "\n")
	configMap := make(map[string]string)

	if v, ok := req["loginBaseURL"].(string); ok {
		configMap["wp_login_base_url"] = v
	}
	if v, ok := req["newLoginBaseURL"].(string); ok {
		configMap["wp_new_login_base_url"] = v
	}
	if v, ok := req["activeProtocol"].(string); ok {
		if v != "old" && v != "new" {
			return "activeProtocol 只能是 old 或 new"
		}
		configMap["wp_active_protocol"] = v
	}
	if v, ok := req["scanLoginCost"].(float64); ok {
		configMap["wp_scan_login_cost"] = fmt.Sprintf("%d", int(v))
	}
	if v, ok := req["deviceName"].(string); ok {
		configMap["wp_device_name"] = v
	}

	wpYamlKeys := map[string]string{
		"wp_login_base_url":     "login_base_url",
		"wp_new_login_base_url": "new_login_base_url",
		"wp_active_protocol":    "active_protocol",
		"wp_scan_login_cost":    "scan_login_cost",
		"wp_device_name":        "device_name",
	}

	inWxProtocol := false
	wxProtocolEndIdx := -1
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "wx_protocol:") {
			inWxProtocol = true
			newLines = append(newLines, line)
			continue
		}
		if inWxProtocol && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, ":") {
			indent := line[:len(line)-len(strings.TrimLeft(line, " "))]
			for wpKey, yamlKey := range wpYamlKeys {
				if strings.HasPrefix(trimmed, yamlKey+":") {
					if newVal, ok := configMap[wpKey]; ok {
						newLines = append(newLines, fmt.Sprintf("%s%s: %s", indent, yamlKey, newVal))
						delete(configMap, wpKey)
						goto nextWxProtocolLine
					}
				}
			}
			newLines = append(newLines, line)
		nextWxProtocolLine:
			wxProtocolEndIdx = len(newLines)
			continue
		}
		// 遇到下一个顶级节或非缩进行，结束 wx_protocol 范围
		if inWxProtocol && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "") && len(trimmed) > 0 {
			inWxProtocol = false
		}
		// 如果在 wx_protocol 范围内遇到空行或注释，记录位置
		if inWxProtocol {
			wxProtocolEndIdx = len(newLines)
		}
		newLines = append(newLines, line)
	}

	// 将未匹配到的 wx_protocol 字段插入到 wx_protocol 节末尾
	if wxProtocolEndIdx < 0 {
		wxProtocolEndIdx = len(newLines)
	}
	var insertLines []string
	for wpKey, yamlKey := range wpYamlKeys {
		if newVal, ok := configMap[wpKey]; ok {
			insertLines = append(insertLines, fmt.Sprintf("  %s: %s", yamlKey, newVal))
			delete(configMap, wpKey)
		}
	}
	if len(insertLines) > 0 {
		// 在 wx_protocol 节末尾插入
		newLines = append(newLines[:wxProtocolEndIdx], append(insertLines, newLines[wxProtocolEndIdx:]...)...)
	}

	newContent := strings.Join(newLines, "\n")
	err = ioutil.WriteFile(ExecPath+"/conf/config.yaml", []byte(newContent), 0644)
	if err != nil {
		return "写入失败: " + err.Error()
	}

	if err := ReloadConfig(); err != nil {
		Warn("配置热更新失败: %v", err)
		return "保存成功，但热更新失败（重启后生效）"
	}

	return "保存成功，微信协议配置已实时生效"
}

// getWxDeviceStatsFromURL 从指定URL获取设备统计数据
func getWxDeviceStatsFromURL(baseURL string) (total int, online int, offline int) {
	url := baseURL + "/api/v1/wx/user/status"
	client := &http.Client{Timeout: HTTPTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, 0
	}

	var result struct {
		Status bool `json:"status"`
		Data   map[string]struct {
			Survival int `json:"survival"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Status {
		return 0, 0, 0
	}

	for _, info := range result.Data {
		total++
		if info.Survival == 1 {
			online++
		} else {
			offline++
		}
	}
	return
}

// ===================== 微信协议设备管理 =====================

// WxDeviceInfo 后台展示用的微信设备信息
type WxDeviceInfo struct {
	Wxid        string `json:"wxid"`
	Nickname    string `json:"nickname"`
	Avatar      string `json:"avatar"`
	Device      string `json:"device"`
	Survival    int    `json:"survival"` // 1=在线 0=掉线
	LoginDate   int64  `json:"loginDate"`
	RefreshDate int64  `json:"refreshDate"`
}

// wxDeviceRawItem 设备原始数据
type wxDeviceRawItem struct {
	Wxid        string `json:"wxid"`
	Avatar      string `json:"avatar"`
	Nickname    string `json:"nickname"`
	Device      string `json:"device"`
	Survival    int    `json:"survival"`
	LoginDate   int64  `json:"loginDate"`
	RefreshDate int64  `json:"refreshDate"`
}

// fetchWxDevicesFromURL 从指定地址获取设备列表
func fetchWxDevicesFromURL(baseURL string) map[string]wxDeviceRawItem {
	url := baseURL + "/api/v1/wx/user/status"
	client := &http.Client{Timeout: HTTPTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var result struct {
		Status bool                      `json:"status"`
		Data   map[string]wxDeviceRawItem `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Status {
		return nil
	}
	return result.Data
}

// mergeWxDevices 合并两个地址的设备列表，在线优先
func mergeWxDevices(oldData, newData map[string]wxDeviceRawItem) map[string]wxDeviceRawItem {
	merged := make(map[string]wxDeviceRawItem)
	// 先放入旧地址数据
	for wxid, info := range oldData {
		merged[wxid] = info
	}
	// 合并新地址数据，在线状态优先，或更新时间更新的优先
	for wxid, newInfo := range newData {
		if oldInfo, exists := merged[wxid]; exists {
			// 如果新地址在线而旧地址离线，用新的
			if newInfo.Survival == 1 && oldInfo.Survival != 1 {
				merged[wxid] = newInfo
			} else if newInfo.Survival == oldInfo.Survival && newInfo.RefreshDate > oldInfo.RefreshDate {
				merged[wxid] = newInfo
			}
		} else {
			merged[wxid] = newInfo
		}
	}
	return merged
}

// GetWxDeviceList 获取所有微信设备列表（含统计）—— 同时读取新旧地址
func GetWxDeviceList(search string) ([]WxDeviceInfo, int, int, int) {
	var list []WxDeviceInfo
	totalCount := 0
	onlineCount := 0
	offlineCount := 0

	// 获取旧地址数据
	oldData := fetchWxDevicesFromURL(getOldWxLoginBaseURL())

	// 获取新地址数据（如果启用了新地址）
	var newData map[string]wxDeviceRawItem
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		newData = fetchWxDevicesFromURL(getNewWxLoginBaseURL())
	}

	// 合并两个地址的数据
	merged := mergeWxDevices(oldData, newData)

	searchLower := strings.ToLower(search)
	for _, info := range merged {
		// 如果该 wxid 最近被主动登出（60秒内），强制显示为离线
		survival := info.Survival
		if survival == 1 && isRecentlyLoggedOut(info.Wxid) {
			survival = 0
		}
		totalCount++
		if survival == 1 {
			onlineCount++
		} else {
			offlineCount++
		}
		// 搜索过滤
		if search != "" &&
			!strings.Contains(strings.ToLower(info.Wxid), searchLower) &&
			!strings.Contains(strings.ToLower(info.Nickname), searchLower) &&
			!strings.Contains(strings.ToLower(info.Device), searchLower) {
			continue
		}
		list = append(list, WxDeviceInfo{
			Wxid:        info.Wxid,
			Nickname:    info.Nickname,
			Avatar:      info.Avatar,
			Device:      info.Device,
			Survival:    survival,
			LoginDate:   info.LoginDate,
			RefreshDate: info.RefreshDate,
		})
	}

	return list, totalCount, onlineCount, offlineCount
}

// Count 统计用户信息
func Count() string {
	zs := 0  // 总数
	yx := 0  // 有效数
	wx := 0  // 无效数
	ts := 0  // 今日更新
	tc := 0  // 今日新增
	mm := 0  // 密码用户
	app := 0  // app用户
	Hack := 0 //屏蔽任务用户
	Smsverify := 0  // 需要验证账号
	dt := Date()  // 当前日期
	cks := GetJdCookies()  // 获取京东Cookies列表
	for _, ck := range cks {
		zs++  // 总数加一
		if ck.Available == "true" {
			yx++  // 有效数加一
			if ck.Password != "" {
				mm++  // 密码不为空时加一
			}
			if ck.IsApp == "true" {
				app++  // 只有当密码为空时，才增加 app 用户计数
			}
			if ck.CreateAt == dt {
				tc++  // 创建日期与当前日期相等时加一
			}
		}

		// 不再依赖于 Available 进行计数
		if ck.Smsverify == "true" {
			Smsverify++
		}
		if ck.Hack == "true" {
			Hack++
		}
		if ck.UpdateAt == dt {
			ts++  // 更新日期与当前日期相等时加一
		}
	}

	// 无效用户数为总数减去有效用户数
	wx = zs - yx

	// 返回统计结果的字符串格式
	return fmt.Sprintf("当前用户总数：%d\n有效用户总数：%d\n有效密码用户：%d\nAPP用户总数：%d\n任务屏蔽总数：%d\n无效用户总数：%d\n密码验证用户：%d\n今日更新总数：%d\n今日新增总数：%d", zs, yx, mm, app, Hack, wx, Smsverify, ts, tc)
}

// GetWxDeviceListByURL 按指定地址获取设备列表
// addr: "old"=旧地址, "new"=新地址, "merged"=合并(默认)
func GetWxDeviceListByURL(addr, search string) ([]WxDeviceInfo, int, int, int) {
	var data map[string]wxDeviceRawItem

	switch addr {
	case "old":
		data = fetchWxDevicesFromURL(getOldWxLoginBaseURL())
	case "new":
		if Config.WxProtocol.NewLoginBaseURL != "" {
			data = fetchWxDevicesFromURL(getNewWxLoginBaseURL())
		}
	default: // merged
		oldData := fetchWxDevicesFromURL(getOldWxLoginBaseURL())
		var newData map[string]wxDeviceRawItem
		if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
			newData = fetchWxDevicesFromURL(getNewWxLoginBaseURL())
		}
		data = mergeWxDevices(oldData, newData)
	}

	if data == nil {
		return nil, 0, 0, 0
	}

	var list []WxDeviceInfo
	totalCount := 0
	onlineCount := 0
	offlineCount := 0
	searchLower := strings.ToLower(search)

	for _, info := range data {
		// 如果该 wxid 最近被主动登出（60秒内），强制显示为离线
		survival := info.Survival
		if survival == 1 && isRecentlyLoggedOut(info.Wxid) {
			survival = 0
		}
		totalCount++
		if survival == 1 {
			onlineCount++
		} else {
			offlineCount++
		}
		if search != "" &&
			!strings.Contains(strings.ToLower(info.Wxid), searchLower) &&
			!strings.Contains(strings.ToLower(info.Nickname), searchLower) &&
			!strings.Contains(strings.ToLower(info.Device), searchLower) {
			continue
		}
		list = append(list, WxDeviceInfo{
			Wxid:        info.Wxid,
			Nickname:    info.Nickname,
			Avatar:      info.Avatar,
			Device:      info.Device,
			Survival:    survival,
			LoginDate:   info.LoginDate,
			RefreshDate: info.RefreshDate,
		})
	}

	return list, totalCount, onlineCount, offlineCount
}

// GetWxDeviceStats 只获取微信设备统计数据（轻量级，用于仪表盘）—— 同时读取新旧地址
func GetWxDeviceStats() (total int, online int, offline int) {
	// 获取旧地址数据
	oldData := fetchWxDevicesFromURL(getOldWxLoginBaseURL())

	// 获取新地址数据（如果启用了新地址）
	var newData map[string]wxDeviceRawItem
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		newData = fetchWxDevicesFromURL(getNewWxLoginBaseURL())
	}

	// 合并两个地址的数据
	merged := mergeWxDevices(oldData, newData)

	for _, info := range merged {
		total++
		// 如果该 wxid 最近被主动登出（60秒内），强制显示为离线
		if info.Survival == 1 && isRecentlyLoggedOut(info.Wxid) {
			offline++
		} else if info.Survival == 1 {
			online++
		} else {
			offline++
		}
	}
	return
}

type UserOnlineStats struct {
	TotalAccounts int64 `json:"totalAccounts"`
	OnlineCount   int64 `json:"onlineCount"`
	TodayActive   int64 `json:"todayActive"` // 今日活跃用户数
}

// AdminDeleteWxDevices 管理员批量删除微信设备
// addr: "old"=仅旧地址, "new"=仅新地址, "merged"=按设备所在地址分别删除
func AdminDeleteWxDevices(wxids []string, addr string) (int, string) {
	if len(wxids) == 0 {
		return 0, "未选择任何设备"
	}

	oldURL := getOldWxLoginBaseURL()
	newURL := getNewWxLoginBaseURL()

	var oldWxids, newWxids []string

	switch addr {
	case "old":
		// 仅从旧地址删除
		oldWxids = wxids
	case "new":
		// 仅从新地址删除
		newWxids = wxids
	default: // merged
		// 遍历每个 wxid，判断它在哪台地址上
		oldData := fetchWxDevicesFromURL(oldURL)
		newData := fetchWxDevicesFromURL(newURL)
		for _, wxid := range wxids {
			if _, exists := oldData[wxid]; exists {
				oldWxids = append(oldWxids, wxid)
			}
			if _, exists := newData[wxid]; exists {
				newWxids = append(newWxids, wxid)
			}
		}
	}

	deletedCount := 0
	var errMsgs []string

	if len(oldWxids) > 0 {
		body, err := wxLoginRequestToURL(oldURL, "/api/v1/wx/user/delete", map[string]interface{}{"wxids": oldWxids})
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("旧地址删除失败: %v", err))
		} else {
			var result struct {
				Status  bool   `json:"status"`
				Message string `json:"message"`
			}
			if json.Unmarshal(body, &result) == nil && result.Status {
				deletedCount += len(oldWxids)
			} else {
				errMsgs = append(errMsgs, fmt.Sprintf("旧地址删除失败: %s", result.Message))
			}
		}
	}

	if len(newWxids) > 0 {
		body, err := wxLoginRequestToURL(newURL, "/api/v1/wx/user/delete", map[string]interface{}{"wxids": newWxids})
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("新地址删除失败: %v", err))
		} else {
			var result struct {
				Status  bool   `json:"status"`
				Message string `json:"message"`
			}
			if json.Unmarshal(body, &result) == nil && result.Status {
				deletedCount += len(newWxids)
			} else {
				errMsgs = append(errMsgs, fmt.Sprintf("新地址删除失败: %s", result.Message))
			}
		}
	}

	msg := fmt.Sprintf("成功删除 %d 个设备", deletedCount)
	if len(errMsgs) > 0 {
		msg += "，部分失败: " + strings.Join(errMsgs, "; ")
	}
	return deletedCount, msg
}

func GetUserOnlineStats() UserOnlineStats {
	var stats UserOnlineStats
	onlineThreshold := time.Now().Add(-15 * time.Minute)
	todayStart := time.Now().Format("2006-01-02")

	db.Model(&WebUserAccount{}).Count(&stats.TotalAccounts)

	var webOnlineUsers []int
	db.Model(&WebUserAccount{}).Where("last_login_at >= ?", onlineThreshold).Pluck("user_number", &webOnlineUsers)

	var appOnlineUsers []int
	db.Model(&User{}).Where("active_at >= ?", onlineThreshold).Pluck("number", &appOnlineUsers)

	onlineSet := make(map[int]bool)
	for _, n := range webOnlineUsers {
		onlineSet[n] = true
	}
	for _, n := range appOnlineUsers {
		onlineSet[n] = true
	}
	stats.OnlineCount = int64(len(onlineSet))

	// 统计今日活跃用户数（APP和网页登录）
	var todayAppUsers []int
	db.Model(&User{}).Where("DATE(active_at) = ?", todayStart).Pluck("number", &todayAppUsers)
	var todayWebUsers []int
	db.Model(&WebUserAccount{}).Where("DATE(last_login_at) = ?", todayStart).Pluck("user_number", &todayWebUsers)
	todayActiveSet := make(map[int]bool)
	for _, n := range todayAppUsers {
		todayActiveSet[n] = true
	}
	for _, n := range todayWebUsers {
		todayActiveSet[n] = true
	}
	stats.TodayActive = int64(len(todayActiveSet))

	return stats
}
