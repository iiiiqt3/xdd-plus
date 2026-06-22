package models

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	mrand "math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

var jdCipherDictionary = map[rune]rune{
	'A': 'K', 'B': 'L', 'C': 'M', 'D': 'N', 'E': 'O', 'F': 'P', 'G': 'Q', 'H': 'R', 'I': 'S', 'J': 'T', 'K': 'A', 'L': 'B', 'M': 'C', 'N': 'D', 'O': 'E', 'P': 'F', 'Q': 'G', 'R': 'H', 'S': 'I', 'T': 'J',
	'e': 'o', 'f': 'p', 'g': 'q', 'h': 'r', 'i': 's', 'j': 't', 'k': 'u', 'l': 'v', 'm': 'w', 'n': 'x', 'o': 'e', 'p': 'f', 'q': 'g', 'r': 'h', 's': 'i', 't': 'j', 'u': 'k', 'v': 'l', 'w': 'm', 'x': 'n',
}

const (
	jdH5stAESKey = "wm0!@w-s#ll1flo("
	jdH5stAESIv  = "0102030405060708"
	jdEy8AESKey  = "&d74&yWoV.EYbWbZ"
	jdSignKey    = "80306f4370b39fd5630ad0529f77adb6"
)

type JDLocalQuery struct {
	Cookie     string
	Pin        string
	UA         string
	H5st       *JDH5ST
	httpClient *http.Client
	proxyLog   *jdProxyLogTransport
}

type JDLocalQueryResult struct {
	NickName        string
	LevelName       string
	JingXiang       string
	BeanCount       string
	TodayIncomeBean int
	TodayOutcomeBean int
	YesterdayIncomeBean int
	YesterdayOutcomeBean int
	HfJifen         string
	ECardCount      string
	ECardTotal      string
	SuperBalance    string
	WangBeiUsable   string
	WangBeiTotal    string
	RedPackCount    string
	RedPackTotal    string
	PlantBeanGrowth string
	PlantBeanDesc   string
	PlantBeanLast   string
	PlantBeanLastGrowth string
	OldFarmName     string
	OldFarmProgress string
	OldFarmWater    string
	OldFarmDays     string
	FarmName        string
	FarmStage       string
	FarmProgress    string
	FarmWater       string
	TrialApplyCount string
	TrialWaitCount  string
	IsPlus          bool
	JdHealth        string
	WanYiWan        string
	ShengQianBi     string
	BeanExpire      []string
	FarmAwards      []string
}

type JDH5ST struct {
	UA         string
	Pin        string
	httpClient *http.Client
}

type jdAlgoResp struct {
	Data struct {
		Result struct {
			Tk   string `json:"tk"`
			Algo string `json:"algo"`
		} `json:"result"`
	} `json:"data"`
}

func NewJDLocalQuery(cookie string) *JDLocalQuery {
	jdSeedRand()
	pin := FetchJdCookieValue("pt_pin", cookie)
	if decoded, err := url.QueryUnescape(pin); err == nil {
		pin = decoded
	}
	ua := jdGenerateUserAgent()
	client, proxyLog := NewJDProxyHTTPClient()
	return &JDLocalQuery{
		Cookie:     cookie,
		Pin:        pin,
		UA:         ua,
		httpClient: client,
		proxyLog:   proxyLog,
		H5st:       &JDH5ST{UA: ua, Pin: pin, httpClient: client},
	}
}

func (q *JDLocalQuery) Query() JDLocalQueryResult {
	defer func() {
		if q.proxyLog == nil {
			return
		}
		n := q.proxyLog.RequestCount()
		if host := q.proxyLog.ProxyHost(); host != "" {
			logs.Info("[京东代理] [jd_query] 查询结束 pin=%s, 共 %d 次 HTTP 请求经代理 %s", q.Pin, n, host)
		} else {
			logs.Info("[京东代理] [jd_query] 查询结束 pin=%s, 共 %d 次 HTTP 直连", q.Pin, n)
		}
	}()
	result := JDLocalQueryResult{}
	if jingxiang := q.queryJingxiang(); jingxiang != nil {
		result.NickName = jingxiang["nickName"]
		result.LevelName = jingxiang["levelName"]
		result.JingXiang = jingxiang["jingxiang"]
		result.BeanCount = jingxiang["beanCount"]
	}
	result.TodayIncomeBean, result.TodayOutcomeBean, result.YesterdayIncomeBean, result.YesterdayOutcomeBean = q.queryBeanStatistics()
	result.HfJifen = q.queryHfJifen()
	result.ECardCount, result.ECardTotal = q.queryECard()
	result.SuperBalance = q.querySuperMarket()
	result.WangBeiUsable, result.WangBeiTotal = q.queryWangBei()
	result.RedPackCount, result.RedPackTotal = q.queryRedPack()
	result.BeanExpire = q.queryBeanExpiring()
	result.PlantBeanGrowth, result.PlantBeanDesc, result.PlantBeanLast, result.PlantBeanLastGrowth = q.queryPlantBean()
	result.OldFarmName, result.OldFarmProgress, result.OldFarmWater, result.OldFarmDays = q.queryOldFarm()
	result.FarmName, result.FarmStage, result.FarmProgress, result.FarmWater = q.queryFarmNew()
	result.FarmAwards = q.queryFarmNewAwards()
	result.WanYiWan = q.queryWanYiWan()
	result.ShengQianBi = q.queryShengQianBi()
	result.TrialApplyCount, result.TrialWaitCount = q.queryTrial()
	result.IsPlus = q.queryIsPlus()
	result.JdHealth = q.queryJdHealth()
	return result
}

func (q *JDLocalQuery) RenderSummary(detail bool) string {
	result := q.Query()
	msgs := []string{}
	hasAccount := result.NickName != "" || result.JingXiang != "" || result.BeanCount != ""
	if hasAccount {
		nick := result.NickName
		if nick == "" {
			nick = q.Pin
		}
		msgs = append(msgs, fmt.Sprintf("🧑‍💼 账号：%s (京享值:%s)", nick, emptyDefault(result.JingXiang, "0")))
		memberType := emptyDefault(result.LevelName, "普通会员")
		if result.IsPlus {
			memberType += "Plus"
		}
		msgs = append(msgs, fmt.Sprintf("💎 会员：%s", memberType))
		if result.BeanCount != "" {
			beanNum, _ := strconv.Atoi(result.BeanCount)
			msgs = append(msgs, fmt.Sprintf("🫘 京豆：%s个 (约%.2f元)", result.BeanCount, float64(beanNum)/100))
		}
	}
	todayLine := fmt.Sprintf("【今日京豆】收%d豆", result.TodayIncomeBean)
	if result.TodayOutcomeBean > 0 {
		todayLine += fmt.Sprintf(",支%d豆", result.TodayOutcomeBean)
	}
	msgs = append(msgs, todayLine)
	yesterdayLine := fmt.Sprintf("【昨日京豆】收%d豆", result.YesterdayIncomeBean)
	if result.YesterdayOutcomeBean > 0 {
		yesterdayLine += fmt.Sprintf(",支%d豆", result.YesterdayOutcomeBean)
	}
	msgs = append(msgs, yesterdayLine)
	if hasAccount {
		msgs = append(msgs, "")
	}
	msgs = append(msgs, "──────── 资产概览 ────────")
	appendIf(&msgs, result.HfJifen, "🪙 话费积分: %s 分")
	if result.ECardCount != "" || result.ECardTotal != "" {
		msgs = append(msgs, fmt.Sprintf("🎫 E卡: %s张, 合计%s元", emptyDefault(result.ECardCount, "0"), emptyDefault(result.ECardTotal, "0.00")))
	}
	appendIf(&msgs, result.SuperBalance, "🛒 超市卡: 余额%s元")
	if result.WangBeiUsable != "" || result.WangBeiTotal != "" {
		msgs = append(msgs, fmt.Sprintf("🐶 汪贝: 可用%s, 总计%s", emptyDefault(result.WangBeiUsable, "0"), emptyDefault(result.WangBeiTotal, "0")))
	}
	appendIf(&msgs, result.WanYiWan, "🎮 玩一玩: %s券")
	appendIf(&msgs, result.ShengQianBi, "💰 省钱币: %s币")
	appendIf(&msgs, result.JdHealth, "⚡ 健康能量: %s")
	if result.RedPackCount != "" || result.RedPackTotal != "" {
		msgs = append(msgs, fmt.Sprintf("🧧 红包: %s个, 总额%s元", emptyDefault(result.RedPackCount, "0"), emptyDefault(result.RedPackTotal, "0.00")))
	}
	showFarm := detail || result.PlantBeanGrowth != "" || result.PlantBeanDesc != "" || result.PlantBeanLast != "" || result.OldFarmName != "" || result.FarmName != "" || result.FarmStage != "" || result.FarmWater != ""
	if showFarm {
		msgs = append(msgs, "")
		msgs = append(msgs, "──────── 农场状态 ────────")
		if result.PlantBeanGrowth != "" || result.PlantBeanDesc != "" || result.PlantBeanLast != "" {
			desc := emptyDefault(result.PlantBeanDesc, "-")
			if !strings.Contains(desc, "月") && desc != "-" {
				desc = "上期 " + desc
			}
			msgs = append(msgs, fmt.Sprintf("🌱 种豆得豆: 成长值%s (%s), 上期%s豆", emptyDefault(result.PlantBeanGrowth, "0"), desc, emptyDefault(result.PlantBeanLast, "0")))
		}
		if result.OldFarmName != "" {
			line := fmt.Sprintf("🍎 东东农场: %s", result.OldFarmName)
			if result.OldFarmProgress != "" {
				line += fmt.Sprintf("(%s%%)", result.OldFarmProgress)
			}
			if result.OldFarmDays != "" {
				line += fmt.Sprintf(",%s天", result.OldFarmDays)
			}
			if result.OldFarmWater != "" {
				line += fmt.Sprintf(", 水滴%s", result.OldFarmWater)
			}
			msgs = append(msgs, line)
		}
		if result.FarmName != "" || result.FarmStage != "" || result.FarmProgress != "" || result.FarmWater != "" {
			msgs = append(msgs, fmt.Sprintf("🚜 新农场: %s %s/5 (%s%%), 水滴%s", emptyDefault(result.FarmName, "未种植"), emptyDefault(result.FarmStage, "0"), emptyDefault(result.FarmProgress, "0"), emptyDefault(result.FarmWater, "0")))
		}
		for _, award := range result.FarmAwards {
			msgs = append(msgs, "🏆 奖励: "+award)
		}
	}
	if detail {
		msgs = append(msgs, "")
		msgs = append(msgs, "──────── 其他 ────────")
		if result.TrialApplyCount != "" || result.TrialWaitCount != "" {
			msgs = append(msgs, fmt.Sprintf("🧪 试用: %s件申请中, %s件待领取", emptyDefault(result.TrialApplyCount, "0"), emptyDefault(result.TrialWaitCount, "0")))
		}
		if len(result.BeanExpire) > 0 {
			msgs = append(msgs, "")
			msgs = append(msgs, "──────── 临期京豆 ────────")
			msgs = append(msgs, result.BeanExpire...)
		}
	}
	if len(msgs) == 0 {
		msgs = append(msgs, "暂无可用查询数据")
	}
	return strings.Join(msgs, "\n")
}

func (q *JDLocalQuery) h5stRequest(functionID string, body interface{}, appID, appid string, extraHeaders map[string]string) (map[string]interface{}, error) {
	h5st, ts, err := q.H5st.Generate(functionID, appID, body, appid)
	if err != nil {
		return nil, err
	}
	postBody := url.Values{}
	postBody.Set("appid", appid)
	postBody.Set("screen", "407*859")
	postBody.Set("build", "98990")
	postBody.Set("osVersion", "15")
	postBody.Set("networkType", "UNKNOWN")
	postBody.Set("d_brand", "Redmi")
	postBody.Set("d_model", "Redmi K50 Ultra")
	postBody.Set("partner", "jingdong")
	postBody.Set("functionId", functionID)
	bodyJSON, _ := json.Marshal(body)
	postBody.Set("body", string(bodyJSON))
	postBody.Set("client", "android")
	postBody.Set("clientVersion", "12.2.0")
	postBody.Set("h5st", h5st)
	postBody.Set("x-api-eid-token", "")
	postBody.Set("timestamp", strconv.FormatInt(ts, 10))
	headers := map[string]string{
		"cookie":           q.Cookie,
		"user-agent":       q.UA,
		"content-type":     "application/x-www-form-urlencoded;charset=UTF-8",
		"x-requested-with": "com.jingdong.app.mall",
	}
	for k, v := range extraHeaders {
		headers[k] = v
	}
	data, err := q.requestBytes("POST", "https://api.m.jd.com/client.action", postBody.Encode(), headers)
	if err != nil {
		return nil, err
	}
	return decodeJSONMap(data)
}

func (q *JDLocalQuery) signRequest(functionID string, body interface{}) (map[string]interface{}, error) {
	signBody := jdSign(functionID, body)
	data, err := q.requestBytes("POST", "https://api.m.jd.com/client.action?functionId="+functionID, signBody+"&x-api-eid-token=", map[string]string{
		"cookie":           q.Cookie,
		"user-agent":       q.UA,
		"content-type":     "application/x-www-form-urlencoded;charset=UTF-8",
		"x-requested-with": "com.jingdong.app.mall",
	})
	if err != nil {
		return nil, err
	}
	return decodeJSONMap(data)
}

func (q *JDLocalQuery) queryJingxiang() map[string]string {
	data, err := q.h5stRequest("pg_channel_page_data", map[string]interface{}{
		"v": "16.3",
		"paramData": map[string]interface{}{"token": "a243ca12-6642-4754-bc5e-0ff012681710", "lid": "Gv8zAj0mnx9iiLgIWfwBEA==", "priceChannel": 2, "device": 0},
		"argMap":    map[string]interface{}{"channel": "APP", "upstreamChannel": "jxz"},
	}, "6d239", "vipChannelHome", map[string]string{"referer": "https://huiyuan.m.jd.com/", "origin": "https://huiyuan.m.jd.com"})
	if err != nil {
		return nil
	}
	floorInfoList, ok := nestedSlice(data, "data", "floorInfoList")
	if !ok || len(floorInfoList) < 5 {
		return nil
	}
	info := nestedMapFromSlice(floorInfoList, 1, "floorData", "userInfo")
	beanInfo := nestedMapFromSlice(floorInfoList, 4, "floorData", "shoppingBeansParam")
	if getMapString(beanInfo, "currentBeanNum") == "" {
		beanInfo = nestedMapFromSlice(floorInfoList, 5, "floorData", "shoppingBeansParam")
	}
	return map[string]string{
		"nickName":  getMapString(info, "showName"),
		"levelName": getMapString(info, "vipGradeName"),
		"jingxiang": getMapString(info, "score"),
		"beanCount": getMapString(beanInfo, "currentBeanNum"),
	}
}

func (q *JDLocalQuery) queryECard() (string, string) {
	data, err := q.h5stRequest("queryGiftCardCountStatusCom", map[string]interface{}{"queryList": "b,i,d,g,a"}, "42e80", "mygiftcard", map[string]string{"origin": "https://mygiftcard.jd.com", "referer": "https://mygiftcard.jd.com/"})
	if err != nil {
		return "", ""
	}
	if getMapString(data, "code") != "success" {
		return "", ""
	}
	inner, ok := data["data"].(map[string]interface{})
	if !ok {
		return "", ""
	}
	var count string
	if cards, ok := inner["g"].([]interface{}); ok {
		for _, item := range cards {
			if m, ok := item.(map[string]interface{}); ok && intValue(m["id"]) == 1 {
				count = stringify(m["num"])
				break
			}
		}
	}
	return count, stringify(inner["a"])
}

func (q *JDLocalQuery) querySuperMarket() string {
	data, err := q.h5stRequest("atop_channel_marketCard_cardInfo", map[string]interface{}{"babelChannel": "ttt9", "isJdApp": "1", "isWx": "0"}, "35fa0", "jd-super-market", map[string]string{"origin": "https://pro.m.jd.com", "referer": "https://pro.m.jd.com/"})
	if err != nil {
		return ""
	}
	if success, ok := data["success"].(bool); !ok || !success {
		return ""
	}
	items, ok := nestedSlice(data, "data", "floorData", "items")
	if !ok || len(items) == 0 {
		return ""
	}
	card := nestedMapFromInterface(items[0], "marketCardVO")
	return getMapString(card, "balance")
}

func (q *JDLocalQuery) queryWangBei() (string, string) {
	body := map[string]interface{}{"pageSize": 10, "currentPage": 1, "projectId": "1764671", "projectKey": "2nym8aW7jNKRbmxXLdbb75m3ebSH", "sourceCode": 2, "needExchangeRestScore": 1}
	wbSign(body)
	data, err := q.h5stRequest("arvr_queryInteractiveRewardInfo", body, "84692", "commonActivity", map[string]string{"origin": "https://pro.m.jd.com", "referer": "https://pro.m.jd.com/"})
	if err != nil {
		return "", ""
	}
	if getMapString(data, "msg") != "success" {
		return "", ""
	}
	scoreInfoMap := nestedMap(data, "scoreInfoMap")
	return getMapString(scoreInfoMap, "usable"), getMapString(scoreInfoMap, "total")
}

func (q *JDLocalQuery) queryHfJifen() string {
	t := time.Now().UnixMilli()
	sum := md5.Sum([]byte(fmt.Sprintf("%de9c398ffcb2d4824b4d0a703e38yffdd", t)))
	encStr := hex.EncodeToString(sum[:])
	body := url.Values{}
	body.Set("appid", "h5-sep")
	body.Set("body", fmt.Sprintf(`{"t":%d,"encStr":"%s"}`, t, encStr))
	body.Set("client", "m")
	body.Set("clientVersion", "6.0.0")
	data, err := q.requestBytes("POST", "https://api.m.jd.com/api?functionId=DATAWALLET_USER_SIGN_INFO", body.Encode(), map[string]string{"cookie": q.Cookie, "user-agent": q.UA, "content-type": "application/x-www-form-urlencoded", "referer": "https://prodev.m.jd.com/"})
	if err != nil {
		return ""
	}
	balance, _ := jsonFieldString(data, "data", "balanceNum")
	return balance
}

func (q *JDLocalQuery) queryRedPack() (string, string) {
	data, err := q.signRequest("myhongbao_getUsableHongBaoList", map[string]interface{}{"activityArea": "-1", "activityType": "1", "appId": "appHongBao", "appToken": "apphongbao_token", "country": "cn", "platform": "1", "platformId": "appHongBao", "platformToken": "apphongbao_token"})
	if err != nil {
		return "", ""
	}
	list, ok := data["hongBaoList"].([]interface{})
	if !ok {
		return "", ""
	}
	total := 0.0
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			total += floatValue(m["balance"])
		}
	}
	return stringify(data["count"]), fmt.Sprintf("%.2f", total)
}

func (q *JDLocalQuery) queryBeanExpiring() []string {
	data, err := q.signRequest("jingBeanDetail", map[string]interface{}{"pageSize": "20", "page": "1"})
	if err != nil {
		return nil
	}
	list, ok := nestedSlice(data, "others", "jingBeanExpiringInfo", "detailList")
	if !ok {
		return nil
	}
	items := []string{}
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		items = append(items, fmt.Sprintf("%s: %s豆", stringify(m["eventMassage"]), stringify(m["amount"])))
	}
	return items
}

func (q *JDLocalQuery) queryPlantBean() (string, string, string, string) {
	data, err := q.h5stRequest("plantBeanIndex", map[string]interface{}{
		"channel": "wojinghd", "monitor_source": "plant_m_plant_index", "monitor_refer": "", "version": "9.2.4.5",
	}, "d246a", "signed_wh5", map[string]string{"referer": "https://plantearth.m.jd.com/"})
	if err != nil {
		return "", "", "", ""
	}
	if !jdCodeOK(data["code"]) {
		return "", "", "", ""
	}
	rounds, ok := nestedSlice(data, "data", "roundList")
	if !ok || len(rounds) == 0 {
		return "", "", "", ""
	}
	currIdx, lastIdx := 1, 0
	if len(rounds) == 1 {
		currIdx = 0
		lastIdx = -1
	}
	curr := mapFromIface(rounds[currIdx])
	growth := getMapString(curr, "growth")
	desc := getMapString(curr, "dateDesc")
	lastBeans, lastGrowth := "", ""
	if lastIdx >= 0 {
		last := mapFromIface(rounds[lastIdx])
		lastBeans = getMapString(last, "awardBeans")
		lastGrowth = getMapString(last, "growth")
	}
	return growth, desc, lastBeans, lastGrowth
}

func (q *JDLocalQuery) queryOldFarm() (string, string, string, string) {
	farmHeaders := map[string]string{
		"origin":         "https://carry.m.jd.com",
		"referer":        "https://carry.m.jd.com/",
		"x-referer-page": "https://carry.m.jd.com/babelDiy/Zeus/3KSjXqQabiTuD1cJ28QskrpWoBKT/index.html",
	}
	farmBody := map[string]interface{}{"babelChannel": "522", "version": 26, "channel": 1, "lat": "0", "lng": "0"}
	taskData, _ := q.h5stRequest("taskInitForFarm", farmBody, "fcb5a", "signed_wh5", farmHeaders)
	initData, err := q.h5stRequest("initForFarm", farmBody, "8a2af", "signed_wh5", farmHeaders)
	if err != nil || initData == nil {
		return "", "", "", ""
	}
	waterTaskTimes := 0
	if taskData != nil {
		totalWater := nestedMap(taskData, "totalWaterTaskInit")
		if len(totalWater) > 0 {
			waterTaskTimes = intValue(totalWater["totalWaterTaskTimes"])
		}
	}
	farmUserPro := nestedMap(initData, "farmUserPro")
	if len(farmUserPro) == 0 {
		return "", "", "", ""
	}
	name := getMapString(farmUserPro, "name")
	treeEnergy := floatValue(farmUserPro["treeEnergy"])
	treeTotalEnergy := floatValue(farmUserPro["treeTotalEnergy"])
	totalEnergy := getMapString(farmUserPro, "totalEnergy")
	progress := ""
	if treeTotalEnergy > 0 {
		progress = strconv.Itoa(int(math.Round(treeEnergy / treeTotalEnergy * 100)))
	}
	days := ""
	if waterTaskTimes > 0 && treeTotalEnergy > treeEnergy {
		waterTotalT := (treeTotalEnergy - treeEnergy - floatValue(farmUserPro["totalEnergy"])) / 10
		if waterTotalT > 0 {
			days = strconv.Itoa(int(math.Ceil(waterTotalT / float64(waterTaskTimes))))
		}
	}
	return name, progress, totalEnergy, days
}

func (q *JDLocalQuery) queryFarmNew() (string, string, string, string) {
	data, err := q.h5stRequest("farm_home", map[string]interface{}{"version": 7}, "c57f6", "signed_wh5", map[string]string{
		"x-referer-page": "https://h5.m.jd.com/pb/015686010/Bc9WX7MpCW7nW9QjZ5N3fFeJXMH/index.html",
		"origin": "https://h5.m.jd.com", "referer": "https://h5.m.jd.com/", "x-rp-client": "h5_1.0.0", "request-from": "native",
	})
	if err != nil {
		return "", "", "", ""
	}
	if !jdCodeOK(data["code"]) {
		return "", "", "", ""
	}
	result := nestedMap(data, "data", "result")
	if len(result) == 0 {
		return "", "", "", ""
	}
	name := getMapString(result, "skuName")
	if name == "" {
		if intValue(result["treeFullStage"]) == 0 {
			name = "水果未种植"
		} else {
			name = "种植中"
		}
	} else if intValue(result["treeCurrentState"]) == 0 {
		tips := getMapString(result, "waterTips")
		if tips != "" {
			name = name + tips
		}
	}
	return name, getMapString(result, "treeFullStage"), getMapString(result, "currentProcess"), getMapString(result, "bottleWater")
}

func (q *JDLocalQuery) queryFarmNewAwards() []string {
	data, err := q.h5stRequest("farm_award_detail", map[string]interface{}{"version": 3, "type": 1}, "c57f6", "signed_wh5", map[string]string{"referer": "https://h5.m.jd.com/"})
	if err != nil {
		return nil
	}
	if !jdCodeOK(data["code"]) {
		return nil
	}
	awards, ok := nestedSlice(data, "data", "result", "plantAwards")
	if !ok {
		return nil
	}
	res := []string{}
	for _, item := range awards {
		m := mapFromIface(item)
		if intValue(m["awardStatus"]) == 1 {
			res = append(res, fmt.Sprintf("%s - %s", stringify(m["skuName"]), stringify(m["plantCompleteTip"])))
		}
	}
	return res
}

func (q *JDLocalQuery) queryWanYiWan() string {
	data, err := q.h5stRequest("wanyiwan_exchange_page", map[string]interface{}{"showShortcut": false, "version": 7}, "afec7", "signed_wh5", map[string]string{"origin": "https://pro.m.jd.com", "referer": "https://pro.m.jd.com/"})
	if err != nil {
		return ""
	}
	if intValue(data["code"]) != 0 {
		return ""
	}
	result := nestedMap(data, "data", "result")
	return getMapString(result, "score")
}

func (q *JDLocalQuery) queryShengQianBi() string {
	data, err := q.h5stRequest("miniTask_hbChannelPage", map[string]interface{}{"source": "task", "businessSource": "cjs"}, "60d61", "hot_channel", map[string]string{"referer": "https://servicewechat.com/"})
	if err != nil {
		return ""
	}
	if intValue(data["subCode"]) != 0 {
		return ""
	}
	return getMapString(nestedMap(data, "data"), "point")
}

func (q *JDLocalQuery) queryTrial() (string, string) {
	data, err := q.h5stRequest("try_MyTrials", map[string]interface{}{"page": 1, "selected": 1}, "6d63a", "newtry", map[string]string{"origin": "https://prodev.m.jd.com", "referer": "https://prodev.m.jd.com/"})
	if err != nil {
		return "", ""
	}
	if success, ok := data["success"].(bool); !ok || !success {
		return "", ""
	}
	list, ok := nestedSlice(data, "data", "list")
	if !ok {
		return "0", "0"
	}
	waitdraw := 0
	for _, item := range list {
		buttons, ok := nestedSlice(mapFromIface(item), "tryButtonList")
		if ok && len(buttons) == 2 {
			first := mapFromIface(buttons[0])
			if intValue(first["id"]) <= 2 {
				waitdraw++
			}
		}
	}
	return strconv.Itoa(len(list)), strconv.Itoa(waitdraw)
}

func (q *JDLocalQuery) queryJdHealth() string {
	body := url.Values{}
	body.Set("body", `{"appKey":"231282000001","appId":"1EFRYwg","channel":"jdapp","activityId":8542,"taskIdList":["520953","520954","520955","674815","841731","674816","674814"],"awardType":2,"imei":"JHNFCKDL"}`)
	data, err := q.requestBytes("POST", fmt.Sprintf("https://api.m.jd.com/api?appid=jdh-middle&functionId=jdh_bm_queryAwardAndScore&t=%d", time.Now().UnixMilli()), body.Encode(), map[string]string{"cookie": q.Cookie, "user-agent": q.UA, "content-type": "application/x-www-form-urlencoded;charset=UTF-8", "x-requested-with": "com.jingdong.app.mall", "referer": "https://jdhm.jd.com/", "origin": "https://jdhm.jd.com"})
	if err != nil {
		return ""
	}
	m, err := decodeJSONMap(data)
	if err != nil || intValue(m["code"]) != 0 {
		return ""
	}
	return getMapString(nestedMap(m, "data"), "energyValue")
}

func (q *JDLocalQuery) queryBeanStatistics() (int, int, int, int) {
	var todayArr, yesterdayArr []map[string]interface{}
	today := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02")
	yesterday := time.Now().In(time.FixedZone("CST", 8*3600)).Add(-24 * time.Hour).Format("2006-01-02")

	testResp, testErr := q.getJingBeanBalanceDetail1(1)
	useAPI1 := testErr == nil && testResp != nil && (jdCodeOK(testResp["code"]) || testResp["jingDetailList"] != nil)

	page := 1
	done := false
	for !done {
		var detailList []interface{}
		var resp map[string]interface{}
		var err error

		if useAPI1 {
			resp, err = q.getJingBeanBalanceDetail1(page)
			if err != nil || resp == nil {
				break
			}
			if stringify(resp["code"]) == "3" {
				break
			}
			if !jdCodeOK(resp["code"]) && resp["jingDetailList"] == nil {
				break
			}
			list, ok := resp["jingDetailList"].([]interface{})
			if !ok || len(list) == 0 {
				break
			}
			detailList = list
		} else {
			resp, err = q.getJingBeanBalanceDetail(page)
			if err != nil || resp == nil {
				break
			}
			if stringify(resp["code"]) == "3" {
				break
			}
			if !jdCodeOK(resp["code"]) {
				break
			}
			list, ok := resp["detailList"].([]interface{})
			if !ok || len(list) == 0 {
				break
			}
			detailList = list
		}
		page++

		for _, item := range detailList {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			dateStr := getMapString(m, "date")
			if dateStr == "" {
				dateStr = getMapString(m, "createDate")
			}
			eventMassage := getMapString(m, "eventMassage")
			if strings.Contains(eventMassage, "退还") || strings.Contains(eventMassage, "物流") || strings.Contains(eventMassage, "扣赠") {
				continue
			}
			if jdBeanDateContains(dateStr, today) {
				todayArr = append(todayArr, m)
				continue
			}
			if jdBeanDateContains(dateStr, yesterday) {
				yesterdayArr = append(yesterdayArr, m)
				continue
			}
			if dateMs, ok := parseJDBeanDateMs(dateStr); ok {
				tm := parseJDBeanDayStartMs(time.Now()).Add(-24 * time.Hour).UnixMilli()
				if dateMs < tm {
					done = true
					break
				}
			} else if !strings.Contains(dateStr, today) && !strings.Contains(dateStr, yesterday) {
				done = true
				break
			}
		}
	}

	todayIncome, todayOutcome := sumBeanRecords(todayArr)
	yesterdayIncome, yesterdayOutcome := sumBeanRecords(yesterdayArr)
	return todayIncome, todayOutcome, yesterdayIncome, yesterdayOutcome
}

func sumBeanRecords(records []map[string]interface{}) (income, outcome int) {
	for _, item := range records {
		amount := intValue(item["amount"])
		if amount > 0 {
			income += amount
		} else if amount < 0 {
			outcome += -amount
		}
	}
	return income, outcome
}

func jdBeanDateContains(dateStr, day string) bool {
	if dateStr == "" || day == "" {
		return false
	}
	normalized := strings.ReplaceAll(dateStr, "/", "-")
	return strings.Contains(normalized, day)
}

func parseJDBeanDayStartMs(now time.Time) time.Time {
	loc := time.FixedZone("CST", 8*3600)
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func parseJDBeanDateMs(dateStr string) (int64, bool) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return 0, false
	}
	loc := time.FixedZone("CST", 8*3600)
	normalized := strings.ReplaceAll(dateStr, "-", "/")
	formats := []string{
		"2006/01/02 15:04:05",
		"2006/01/02",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.ParseInLocation(format, normalized, loc); err == nil {
			return t.UnixMilli(), true
		}
	}
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.UnixMilli(), true
	}
	return 0, false
}

func jdCodeOK(code interface{}) bool {
	switch v := code.(type) {
	case nil:
		return false
	case float64:
		return v == 0
	case int:
		return v == 0
	case int64:
		return v == 0
	case string:
		return v == "0" || v == "success"
	default:
		return stringify(v) == "0"
	}
}

func (q *JDLocalQuery) getJingBeanBalanceDetail1(page int) (map[string]interface{}, error) {
	bodyJSON, _ := json.Marshal(map[string]string{
		"pageSize": "20",
		"page":     strconv.Itoa(page),
	})
	postBody := "body=" + url.QueryEscape(string(bodyJSON)) + "&appid=ld"
	data, err := q.requestBytes("POST", fmt.Sprintf("https://bean.m.jd.com/beanDetail/detail.json?page=%d", page), postBody, map[string]string{
		"cookie":       q.Cookie,
		"user-agent":   "Mozilla/5.0 (Linux; Android 12; SM-G9880) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Mobile Safari/537.36 EdgA/106.0.1370.47",
		"content-type": "application/x-www-form-urlencoded",
		"referer":      "https://bean.m.jd.com/",
	})
	if err != nil {
		return nil, err
	}
	return decodeJSONMap(data)
}

func (q *JDLocalQuery) getJingBeanBalanceDetail(page int) (map[string]interface{}, error) {
	return q.signRequest("getJingBeanBalanceDetail", map[string]interface{}{
		"pageSize": "20",
		"page":     strconv.Itoa(page),
	})
}

func (q *JDLocalQuery) queryIsPlus() bool {
	data, err := q.h5stRequest("user_getUserInfo_v2", map[string]interface{}{
		"qids":        "6_2_5_18_1_7_9_11_12_14_16_17_25",
		"checkLevel":  1,
		"signType":    1003,
		"topicId":     176,
		"contentType": "1_2_3_4_5_8_9_11_12_16_18",
		"skuSourceId": 600008,
	}, "b63ff", "plus_business", map[string]string{"referer": "https://plus.m.jd.com/index", "origin": "https://plus.m.jd.com"})
	if err != nil {
		return false
	}
	if getMapString(data, "code") != "1711000" {
		return false
	}
	plusUserBaseInfo := nestedMap(data, "rs", "plusUserBaseInfo")
	return intValue(plusUserBaseInfo["endDays"]) > 0
}

func (h *JDH5ST) Generate(functionID, appID string, body interface{}, appid string) (string, int64, error) {
	fmtTime, ts := jdFmtTime()
	fp := jdGetFP()
	rdm := jdGetRdm()
	rbparamt := map[string]interface{}{
		"wc": 1, "wd": 0, "l": "zh-CN", "ls": "zh-CN,en-US", "ml": 0, "pl": 0,
		"av": jdAv(h.UA), "ua": h.UA, "sua": jdSUA(h.UA), "pp": map[string]interface{}{},
		"extend": map[string]interface{}{"wd": 0, "l": 0, "ls": 0, "wk": 0, "bu1": "0.1.5", "bu2": 0, "bu3": 14, "bu4": 0},
		"pp1": "", "w": 407, "h": 904, "ow": 407, "oh": 810, "pr": 3, "re": "",
		"random": rdm, "referer": "", "v": "h5_file_v4.3.3", "ai": appID, "fp": fp,
	}
	rbparam := jdAESCBCEncryptHex(rbparamt, jdH5stAESKey, jdH5stAESIv)
	rdb := map[string]interface{}{"version": "4.3", "fp": fp, "appId": appID, "timestamp": time.Now().UnixMilli(), "platform": "web", "expandParams": rbparam, "fv": "h5_file_v4.3.3"}
	algoRes, err := h.getRdAndTk(rdb)
	if err != nil {
		return "", 0, err
	}
	tk := algoRes.Data.Result.Tk
	algo := algoRes.Data.Result.Algo
	rd := jdExtractRd(algo)
	text1 := tk + fp + fmtTime + "22" + appID + rd
	ey1 := jdAlgoDigest(algo, text1, tk)
	bodyJSON, _ := json.Marshal(body)
	signBody := jdSha256Hex(string(bodyJSON))
	fts := time.Now().UnixMilli()
	text2 := fmt.Sprintf("appid:%s&body:%s&client:android&clientVersion:12.2.0&functionId:%s", appid, signBody, functionID)
	ey5 := hmacHexSHA256(text2, ey1)
	ey8Data, _ := json.MarshalIndent(map[string]interface{}{
		"sua": jdSUA(h.UA),
		"pp": map[string]interface{}{"p1": h.Pin},
		"extend": map[string]interface{}{"wd": 0, "l": 0, "ls": 0, "wk": 0, "bu1": "0.1.5", "bu2": -1, "bu3": 14, "bu4": 0},
		"random": rdm, "v": "h5_file_v4.3.3", "fp": fp,
	}, "", "  ")
	ey8 := jdAESCBCEncryptHex(string(ey8Data), jdEy8AESKey, jdH5stAESIv)
	return fmt.Sprintf("%s;%s;%s;%s;%s;4.3;%d;%s", fmtTime, fp, appID, tk, ey5, ts, ey8), fts, nil
}

func jdGenerateUserAgent() string {
	sv := fmt.Sprintf("%d.%d.%d", randomChoice([]int{12, 13, 14, 15, 16}), randomChoice([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}), randomChoice([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}))
	ep := map[string]interface{}{"ciphertype": 5, "cipher": map[string]interface{}{}, "ts": time.Now().Unix(), "hdid": "", "version": "1.2.0", "appname": "com.jingdong.app.mall", "ridx": -1}
	cipherMap := ep["cipher"].(map[string]interface{})
	cipherMap["sv"] = jdTranslate(base64.StdEncoding.EncodeToString([]byte(sv)))
	cipherMap["ud"] = jdTranslate(base64.StdEncoding.EncodeToString([]byte(randomHex(40))))
	ep["hdid"] = "JM9F1ywUPwflvMIpYPok0tt5k9kW4ArJEU3lfLhxBqw="
	epJSON, _ := json.Marshal(ep)
	return "jdapp;android;12.2.0;;;M/5.0;appBuild/98990;ef/1;ep/" + url.QueryEscape(string(epJSON)) + ";jdSupportDarkMode/0;Mozilla/5.0 (Linux; Android 13; 22081212C Build/TKQ1.220829.002; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/89.0.4389.72 MQQBrowser/6.2 TBS/046269 Mobile Safari/537.36"
}

func jdTranslate(str string) string {
	var out strings.Builder
	for _, ch := range str {
		if v, ok := jdCipherDictionary[ch]; ok {
			out.WriteRune(v)
		} else {
			out.WriteRune(ch)
		}
	}
	return out.String()
}

func jdFmtTime() (string, int64) {
	d := time.Now()
	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d%03d", d.Year(), d.Month(), d.Day(), d.Hour(), d.Minute(), d.Second(), d.Nanosecond()/1e6), d.UnixMilli()
}

func jdGetFP() string {
	x := "kl9i1uct6d"
	u := pickString(x, 3)
	et := mrand.Intn(10)
	j := removeChars(x, u)
	s := randomFromCharset(et, j) + u + randomFromCharset(12-et, j) + strconv.Itoa(et)
	z := strings.Split(s, "")
	tt := z[:10]
	v := z[10:]
	nt := []string{}
	for len(tt) > 0 {
		last := tt[len(tt)-1]
		tt = tt[:len(tt)-1]
		n, _ := strconv.ParseInt(last, 36, 64)
		nt = append(nt, strconv.FormatInt(35-n, 36))
	}
	return strings.Join(append(nt, v...), "")
}

func jdGetRdm() string {
	charset := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_-"
	return randomFromCharset(10, charset)
}

func jdAESCBCEncryptHex(word interface{}, key, iv string) string {
	var plain []byte
	switch v := word.(type) {
	case string:
		plain = []byte(v)
	default:
		plain, _ = json.Marshal(v)
	}
	block, _ := aes.NewCipher([]byte(key))
	plain = pkcs7Pad(plain, block.BlockSize())
	cipherText := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, []byte(iv)).CryptBlocks(cipherText, plain)
	return hex.EncodeToString(cipherText)
}

func (h *JDH5ST) getRdAndTk(body interface{}) (*jdAlgoResp, error) {
	payload, _ := json.Marshal(body)
	data, err := h.requestBytes("POST", "https://cactus.jd.com/request_algo?g_ty=ajax", string(payload), map[string]string{"content-type": "application/json", "referer": "https://bnzf.jd.com/"})
	if err != nil {
		return nil, err
	}
	res := &jdAlgoResp{}
	if err := json.Unmarshal(data, res); err != nil {
		return nil, err
	}
	return res, nil
}

func (h *JDH5ST) requestBytes(method, target, body string, headers map[string]string) ([]byte, error) {
	return doRequestBytes(h.httpClient, method, target, body, headers)
}

func (q *JDLocalQuery) requestBytes(method, target, body string, headers map[string]string) ([]byte, error) {
	return doRequestBytes(q.httpClient, method, target, body, headers)
}

func doRequestBytes(client *http.Client, method, target, body string, headers map[string]string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequest(method, target, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func jdExtractRd(algo string) string {
	start := strings.Index(algo, "rd='")
	if start == -1 {
		return ""
	}
	start += 4
	end := strings.Index(algo[start:], "';")
	if end == -1 {
		return ""
	}
	return algo[start : start+end]
}

func jdAlgoDigest(algo, text, tk string) string {
	switch {
	case strings.Contains(algo, "HmacMD5(str,tk)"):
		return hmacHexMD5(text, tk)
	case strings.Contains(algo, "MD5(str)"):
		return jdMd5Hex(text)
	case strings.Contains(algo, "HmacSHA256(str,tk)"):
		return hmacHexSHA256(text, tk)
	case strings.Contains(algo, "HmacSHA512(str,tk)"):
		return hmacHexSHA512(text, tk)
	case strings.Contains(algo, "SHA256(str)"):
		return jdSha256Hex(text)
	case strings.Contains(algo, "SHA512(str)"):
		return jdSha512Hex(text)
	default:
		return hmacHexSHA512(text, tk)
	}
}

func jdSign(functionID string, body interface{}) string {
	bodyJSON, _ := json.Marshal(body)
	eid := "eidAaf8081218as20a2GM" + randomAlphaNum(20) + "7FnfQYOecyDYLcd0rfzm3Fy2ePY4UJJOeV0Ub840kG8C7lmIqt3DTlc11fB/s4qsAP8gtPTSoxu"
	ep, ts, jduuid, dBrand := jdGetEP()
	versions := [][2]int{{0, 2}, {1, 1}, {2, 0}}
	pick := versions[mrand.Intn(len(versions))]
	sv := fmt.Sprintf("1%d%d", pick[0], pick[1])
	allArg := fmt.Sprintf("functionId=%s&body=%s&uuid=%s&client=android&clientVersion=12.2.0&st=%d&sv=%s", functionID, string(bodyJSON), jduuid, ts, sv)
	by := stringToBytes(allArg)
	signVal := jdMd5Hex(bytesToString(jdSignCore(by)))
	ext := url.QueryEscape(`{"prstate":"0","pvcStu":"1"}`)
	partner := strings.ToLower(dBrand)
	return fmt.Sprintf("body=%s&clientVersion=12.2.0&build=98935&client=android&partner=%s&sdkVersion=31&lang=zh_CN&harmonyOs=0&networkType=wifi&ext=%s&oaid=%s&eid=%s&ef=1&ep=%s&st=%d&sign=%s&sv=%s", url.QueryEscape(string(bodyJSON)), partner, ext, jduuid, eid, url.QueryEscape(ep), ts, signVal, sv)
}

func jdSignCore(inarg []byte) []byte {
	key := stringToBytes(jdSignKey)
	mask := []byte{0x37, 0x92, 0x44, 0x68, 0xA5, 0x3D, 0xCC, 0x7F, 0xBB, 0x0F, 0xD9, 0x88, 0xEE, 0x9A, 0xE9, 0x5A}
	array := make([]byte, len(inarg))
	for i := 0; i < len(inarg); i++ {
		r0 := int(inarg[i])
		r2 := int(mask[i&0xf])
		r4 := int(key[i&7])
		r0 = r2 ^ r0
		r0 = r0 ^ r4
		r0 = r0 + r2
		r2 = r2 ^ r0
		r2 = r2 ^ int(key[i&7])
		array[i] = byte(r2 & 0xff)
	}
	return []byte(customBase64(array))
}

func jdGetEP() (string, int64, string, string) {
	jduuid := strings.ReplaceAll(randomUUID(), "-", "")[:16]
	ts := time.Now().UnixMilli()
	area := randomDigits(2) + "_" + randomDigits(4) + "_" + randomDigits(5) + "_" + randomDigits(4)
	dBrandModel := map[string][]string{"OPPO": {"PAFM00", "PDEM10"}, "Xiaomi": {"23078PND5G", "2211133C"}, "HUAWEI": {"LIO-AL00", "OCE-AN10"}}
	brands := []string{"OPPO", "Xiaomi", "HUAWEI"}
	dBrand := brands[mrand.Intn(len(brands))]
	models := dBrandModel[dBrand]
	dModel := models[mrand.Intn(len(models))]
	ep := map[string]interface{}{
		"hdid": "JM9F1ywUPwflvMIpYPok0tt5k9kW4ArJEU3lfLhxBqw=", "ts": ts, "ridx": -1,
		"cipher": map[string]interface{}{
			"area":     jdSignBase64Encode(area),
			"d_model":  jdSignBase64Encode(dModel),
			"wifiBssid": jdSignBase64Encode("TP_LINK_" + randomAlphaNum(6)),
			"osVersion": jdSignBase64Encode("12"),
			"d_brand":   jdSignBase64Encode(dBrand),
			"screen":    jdSignBase64Encode("1080x1920"),
			"uuid":      jdSignBase64Encode(jduuid),
			"aid":       jdSignBase64Encode(jduuid),
			"openudid":  jdSignBase64Encode(jduuid),
		},
		"ciphertype": 5, "version": "1.2.0", "appname": "com.jingdong.app.mall",
	}
	payload, _ := json.Marshal(ep)
	return string(payload), ts, jduuid, dBrand
}

func jdSignBase64Encode(str string) string {
	enc := base64.StdEncoding.EncodeToString([]byte(str))
	from := "KLMNOPQRSTABCDEFGHIJUVWXYZabcdopqrstuvwxefghijklmnyz0123456789+/"
	to := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var out strings.Builder
	for _, ch := range enc {
		idx := strings.IndexRune(from, ch)
		if idx >= 0 {
			out.WriteByte(to[idx])
		}
	}
	return out.String()
}

func wbSign(body map[string]interface{}) {
	signKey := "c4491f13dce9c71f"
	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, stringify(body[key]))
	}
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	body["timestamp"] = timestamp
	body["sign"] = jdMd5Hex(signKey + strings.Join(parts, "") + timestamp)
	body["signKey"] = signKey
}

func decodeJSONMap(data []byte) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	if len(data) > 0 && data[0] != '{' && data[0] != '[' {
		data = bytes.TrimSpace(data)
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func nestedMap(m map[string]interface{}, path ...string) map[string]interface{} {
	cur := m
	for _, key := range path {
		v, ok := cur[key].(map[string]interface{})
		if !ok {
			return map[string]interface{}{}
		}
		cur = v
	}
	return cur
}

func nestedSlice(m map[string]interface{}, path ...string) ([]interface{}, bool) {
	var cur interface{} = m
	for _, key := range path {
		mm, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		cur = mm[key]
	}
	res, ok := cur.([]interface{})
	return res, ok
}

func nestedMapFromSlice(items []interface{}, idx int, path ...string) map[string]interface{} {
	if idx < 0 || idx >= len(items) {
		return map[string]interface{}{}
	}
	return nestedMap(mapFromIface(items[idx]), path...)
}

func nestedMapFromInterface(item interface{}, path ...string) map[string]interface{} {
	return nestedMap(mapFromIface(item), path...)
}

func mapFromIface(v interface{}) map[string]interface{} {
	m, _ := v.(map[string]interface{})
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}

func getMapString(m map[string]interface{}, key string) string {
	return stringify(m[key])
}

func stringify(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case float64:
		if math.Mod(val, 1) == 0 {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func intValue(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		return 0
	}
}

func floatValue(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func appendIf(msgs *[]string, v, format string) {
	if strings.TrimSpace(v) != "" {
		*msgs = append(*msgs, fmt.Sprintf(format, v))
	}
}

func emptyDefault(v, d string) string {
	if strings.TrimSpace(v) == "" {
		return d
	}
	return v
}

func jsonFieldString(data []byte, path ...string) (string, error) {
	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", err
	}
	cur := raw
	for _, key := range path {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("invalid path")
		}
		cur = m[key]
	}
	return stringify(cur), nil
}

func pkcs7Pad(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func jdMd5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func jdSha256Hex(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func jdSha512Hex(s string) string {
	h := sha512.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func hmacHexMD5(s, key string) string {
	h := hmac.New(md5.New, []byte(key))
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func hmacHexSHA256(s, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func hmacHexSHA512(s, key string) string {
	h := hmac.New(sha512.New, []byte(key))
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func stringToBytes(param string) []byte {
	bytesArr := []byte{}
	for _, c := range param {
		switch {
		case c == 0:
			bytesArr = append(bytesArr, 0xe3, 0x84, 0x80)
		case c < 0x80:
			bytesArr = append(bytesArr, byte(c))
		case c < 0x100:
			bytesArr = append(bytesArr, 0xc2, byte(c))
		case c < 0x800:
			bytesArr = append(bytesArr, byte(((c>>6)&0x1f)|0xc0), byte((c&0x3f)|0x80))
		case c < 0x10000:
			bytesArr = append(bytesArr, byte(((c>>12)&0x0f)|0xe0), byte(((c>>6)&0x3f)|0x80), byte((c&0x3f)|0x80))
		default:
			bytesArr = append(bytesArr, byte(((c>>18)&0x07)|0xf0), byte(((c>>12)&0x3f)|0x80), byte(((c>>6)&0x3f)|0x80), byte((c&0x3f)|0x80))
		}
	}
	return bytesArr
}

func bytesToString(params []byte) string {
	var result strings.Builder
	for i := 0; i < len(params); i++ {
		result.WriteByte(params[i])
	}
	return result.String()
}

func customBase64(params []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	if len(params) == 0 {
		return ""
	}
	var result []byte
	full := len(params) / 3 * 3
	for i := 0; i < full; i += 3 {
		bits := int(params[i])<<16 | int(params[i+1])<<8 | int(params[i+2])
		result = append(result, chars[(bits>>18)&0x3f], chars[(bits>>12)&0x3f], chars[(bits>>6)&0x3f], chars[bits&0x3f])
	}
	if len(params)%3 == 1 {
		bits := int(params[len(params)-1]) << 4
		result = append(result, chars[(bits>>6)&0x3f], chars[bits&0x3f], '=', '=')
	} else if len(params)%3 == 2 {
		bits := int(params[len(params)-2])<<10 | int(params[len(params)-1])<<2
		result = append(result, chars[(bits>>12)&0x3f], chars[(bits>>6)&0x3f], chars[bits&0x3f], '=')
	}
	return string(result)
}

func randomChoice(arr []int) int {
	return arr[mrand.Intn(len(arr))]
}

func randomHex(n int) string { return randomFromCharset(n, "0123456789abcdef") }
func randomDigits(n int) string { return randomFromCharset(n, "0123456789") }
func randomAlphaNum(n int) string { return randomFromCharset(n, "abcdefghijklmnopqrstuvwxyz0123456789") }

func randomFromCharset(n int, charset string) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = charset[mrand.Intn(len(charset))]
	}
	return string(buf)
}

func pickString(src string, num int) string {
	runes := []rune(src)
	picked := []rune{}
	pool := append([]rune{}, runes...)
	for i := 0; i < num && len(pool) > 0; i++ {
		r := mrand.Intn(len(pool))
		picked = append(picked, pool[r])
		pool = append(pool[:r], pool[r+1:]...)
	}
	for i := 0; i < len(picked); i++ {
		r := mrand.Intn(len(picked)-i) + i
		picked[i], picked[r] = picked[r], picked[i]
	}
	return string(picked)
}

func removeChars(src, chars string) string {
	for _, ch := range chars {
		src = strings.ReplaceAll(src, string(ch), "")
	}
	return src
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func jdSeedRand() {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		mrand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
		return
	}
	mrand.Seed(time.Now().UnixNano())
}

func jdAv(ua string) string {
	idx := strings.Index(ua, "M/5.0")
	if idx == -1 {
		return "M/5.0"
	}
	return ua[idx:]
}

func jdSUA(ua string) string {
	prefix := "Mozilla/5.0 ("
	start := strings.Index(ua, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := strings.Index(ua[start:], ")")
	if end == -1 {
		return ""
	}
	return ua[start : start+end]
}
