package models

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/core/logs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
)

type Asset struct {
	Nickname string
	Bean     struct {
		Total         int
		TodayIn       int
		TodayOut      int
		YestodayIn    int
		YestodayOut   int
		XDTodayIn     int
		XDTodayOut    int
		XDYestodayIn  int
		XDYestodayOut int
		ToExpire      []int
	}
	RedPacket struct {
		Total      float64
		ToExpire   float64
		ToExpireJd float64
		ToExpireJx float64
		ToExpireJs float64
		ToExpireJk float64
		Jd         float64
		Jx         float64
		Js         float64
		Jk         float64
	}
	Other struct {
		JsCoin   float64
		NcStatus float64
		McStatus float64
	}
}

type TotalBean struct {
	Base struct {
		TipUrl       string `json:"TipUrl"`
		AccountType  int    `json:"accountType"`
		CurPin       string `json:"curPin"`
		HeadImageUrl string `json:"headImageUrl"`
		IsJTH        string `json:"isJTH"`
		JdNum        int    `json:"jdNum"`
		Jvalue       int    `json:"jvalue"`
		LevelName    string `json:"levelName"`
		Mobile       string `json:"mobile"`
		Nickname     string `json:"nickname"`
		UserLevel    int    `json:"userLevel"`
	} `json:"base"`
	DefinePin        int    `json:"definePin"`
	IsHitArea        int    `json:"isHitArea"`
	IsHomeWhite      int    `json:"isHomeWhite"`
	IsLongPwdActive  int    `json:"isLongPwdActive"`
	IsPlusVip        bool   `json:"isPlusVip"`
	IsRealNameAuth   bool   `json:"isRealNameAuth"`
	IsShortPwdActive int    `json:"isShortPwdActive"`
	Msg              string `json:"msg"`
	OrderFlag        int    `json:"orderFlag"`
	Retcode          int    `json:"retcode"`
	UserFlag         int    `json:"userFlag"`
}

func getToTalBean(cookie string, totalBean chan TotalBean) {

	a := TotalBean{}
	req := httplib.Post(`https://wq.jd.com/user/info/QueryJDUserInfo?sceneval=2`)
	req.Header("User-Agent", "jdapp;iPhone;9.4.4;14.3;network/4g;Mozilla/5.0 (iPhone; CPU iPhone OS 14_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148;supportJDSHWK/1")
	req.Header("Accept", "application/json,text/plain, */*")
	req.Header("Connection", "keep-alive")
	req.Header("Accept-Language", "zh-cn")
	req.Header("Accept-Encoding", "gzip, deflate, br")
	req.Header("Referer", "https://wqs.jd.com/my/jingdou/my.shtml?sceneval=2")
	req.Header("Cookie", cookie)
	data, _ := req.Bytes()
	//logs.Info(string(data))
	json.Unmarshal(data, &a)
	totalBean <- a
}

var Int = func(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

var Float64 = func(s string) float64 {
	i, _ := strconv.ParseFloat(s, 64)
	return i
}

func DailyAssetsPush() {
	for _, ck := range GetJdCookies() {
		if (ck.UserId != "" && Config.QQID != 0 && SendQQ != nil) || ck.PushPlus != "" {
			msg := ck.Query()

			if ck.UserId != "" && Config.QQID != 0 && SendQQ != nil {
				SendQQ(ck.UserId, msg)
			}
			if ck.PushPlus != "" {
				pushPlus(ck.PushPlus, msg)
			}
			time.Sleep(time.Second * 60)
		}
	}
}

func CompletePush() {
	for _, ck := range GetJdCookies() {
		if (ck.UserId != "" && Config.QQID != 0 && SendQQ != nil) || ck.PushPlus != "" {
			flag := false
			var msg1 []string
			var fruit = make(chan string)
			cookie := fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin)
			go initFarm(cookie, fruit)
			if strings.Contains(<-fruit, "已可领取") {
				flag = true
				msg1 = append(msg1, ck.Nickname+"您的农场无门槛红包已经成熟，请尽快领取\r\n 【东东农场】京东->我的->东东农场,完成是京东红包,可以用于京东app的任意商品")
			}
			time.Sleep(time.Second * 30)
			if flag {
				if ck.UserId != "" && Config.QQID != 0 && SendQQ != nil {
					SendQQ(ck.UserId, strings.Join(msg1, "\n"))
				}
				if ck.PushPlus != "" {
					pushPlus(ck.PushPlus, strings.Join(msg1, "\n"))
				}
			}
			time.Sleep(time.Second * 30)
		}
	}
}

func (ck *JdCookie) Query() string {

	msgs := []string{
		fmt.Sprintf("账号昵称：%s", ck.Nickname),
	}
	parse, err := time.Parse("2006-01-02", ck.CreateAt)
	if err != nil {
		parse, _ = time.Parse("2006/01/02", ck.CreateAt)
	}
	t, _ := time.Parse("2006-01-02", time.Now().Format("2006-01-02"))
	f := t.Sub(parse).Hours() / 24
	i, _ := strconv.Atoi(fmt.Sprintf("%1.0f", f))

	if ck.CreateAt != "" {
		msgs = append(msgs, fmt.Sprintf("您已挂机：%d天", i))
	} else {
		msgs = append(msgs, fmt.Sprintf("账号登录成功：%s", ck.PtPin))
		return strings.Join(msgs, "\n")
	}
	if ck.Note != "" {
		msgs = append(msgs, fmt.Sprintf("账号备注：%s", ck.Note))
	}
	asset := Asset{}
	if CookieOK(ck) {
		//msgs = append(msgs, fmt.Sprintf("优先级：%v", ck.Priority))
		//msgs = append(msgs, fmt.Sprintf("用户等级：%v", ck.UserLevel))
		//msgs = append(msgs, fmt.Sprintf("等级名称：%v", ck.LevelName))

		cookie := fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin)
		if ck.UpdateAt != "" {
			parse1, _ := time.Parse("2006-01-02", ck.UpdateAt)
			logs.Info(parse1)
			msgs = append(msgs, fmt.Sprintf("最后更新时间：%s", parse1.Format("2006-01-02")))
		}
		var rpc = make(chan []RedList)
		var fruit = make(chan string)
		var gold = make(chan int64)
		//		var zjb = make(chan int64)
		//		var jxzz = make(chan string)
		var totalbean = make(chan TotalBean)
		go getToTalBean(cookie, totalbean)
		go redPacket(cookie, rpc)
		go initFarm(cookie, fruit)
		go jsGold(cookie, gold)
		//		go jdzz(cookie, zjb)
		//		go jingxiangzhi(cookie, jxzz)
		today := time.Now().Local().Format("2006-01-02")
		yestoday := time.Now().Local().Add(-time.Hour * 24).Format("2006-01-02")
		page := 1
		end := false
		//to := <-totalbean
		//if to.IsPlusVip {
		//	msgs = append(msgs, fmt.Sprintf("账号信息：%s,京享值:%d", "Plus会员", to.Base.Jvalue))
		//} else {
		//	msgs = append(msgs, fmt.Sprintf("账号信息：%s,京享值:%d", "普通会员", to.Base.Jvalue))
		//}
		for {
			if end {
				msgs = append(msgs, []string{
					fmt.Sprintf("昨日收入：%d京豆", asset.Bean.YestodayIn),
					//fmt.Sprintf("昨日支出：%d京豆", asset.Bean.YestodayOut),
					fmt.Sprintf("今日收入：%d京豆", asset.Bean.TodayIn),
					//fmt.Sprintf("今日支出：%d京豆", asset.Bean.TodayOut),
				}...)
				break
			}
			bds := getJingBeanBalanceDetail(page, cookie)
			if bds == nil {
				end = true
				msgs = append(msgs, "京豆加载中，请耐心等待")
				break
			}
			for _, bd := range bds {
				amount := Int(bd.Amount)
				if strings.Contains(bd.Date, today) {
					if amount > 0 {
						asset.Bean.TodayIn += amount
					} else {
						asset.Bean.TodayOut += -amount
					}
				} else if strings.Contains(bd.Date, yestoday) {
					if amount > 0 {
						asset.Bean.YestodayIn += amount
					} else {
						asset.Bean.YestodayOut += -amount
					}
				} else {
					end = true
					break
				}
			}
			page++
		}
		ysd := int(time.Now().Add(24*time.Hour).Unix()) * 1000
		if rps := <-rpc; len(rps) != 0 {
			for _, rp := range rps {
				b := rp.Balance
				asset.RedPacket.Total += b
				if strings.Contains(rp.OrgLimitStr, "京喜") || strings.Contains(rp.OrgLimitStr, "特价") {
					asset.RedPacket.Jx += b
					if ysd >= rp.EndTime {
						asset.RedPacket.ToExpireJx += b
						asset.RedPacket.ToExpire += b
					}
				} else if strings.Contains(rp.OrgLimitStr, "特价版") {
					asset.RedPacket.Js += b
					if ysd >= rp.EndTime {
						asset.RedPacket.ToExpireJs += b
						asset.RedPacket.ToExpire += b
					}

				} else if strings.Contains(rp.OrgLimitStr, "京东健康") {
					asset.RedPacket.Jk += b
					if ysd >= rp.EndTime {
						asset.RedPacket.ToExpireJk += b
						asset.RedPacket.ToExpire += b
					}
				} else {
					asset.RedPacket.Jd += b
					if ysd >= rp.EndTime {
						asset.RedPacket.ToExpireJd += b
						asset.RedPacket.ToExpire += b
					}
				}
			}
			e := func(m float64) string {
				if m > 0 {
					return fmt.Sprintf(`(今日过期%.2f)`, m)
				}
				return ""
			}
			msgs = append(msgs, []string{
				fmt.Sprintf("所有红包：%.2f%s元", asset.RedPacket.Total, e(asset.RedPacket.ToExpire)),
				fmt.Sprintf("京喜红包：%.2f%s元", asset.RedPacket.Jx, e(asset.RedPacket.ToExpireJx)),
				fmt.Sprintf("京东红包：%.2f%s元", asset.RedPacket.Jd, e(asset.RedPacket.ToExpireJd)),
			}...)
		} else {
			msgs = append(msgs, "暂无红包数据🧧")
		}
		msgs = append(msgs, fmt.Sprintf("东东农场：%s", <-fruit))
		gn := <-gold
		msgs = append(msgs, fmt.Sprintf("极速金币：%d(≈%.2f元)💰", gn, float64(gn)/10000))
	} else {
		msgs = append(msgs, []string{
			"提醒：该账号已过期，请重新登录",
		}...)
	}
	ck.PtPin, _ = url.QueryUnescape(ck.PtPin)
	if Config.Query1 != "" {
		msgs = append(msgs, Config.Query1)
	}
	return strings.Join(msgs, "\n")
}

type BeanDetail struct {
	Date         string `json:"date"`
	Amount       string `json:"amount"`
	EventMassage string `json:"eventMassage"`
}

func getJingBeanBalanceDetail(page int, cookie string) []BeanDetail {
	type AutoGenerated struct {
		Code       string       `json:"code"`
		DetailList []BeanDetail `json:"detailList"`
	}
	a := AutoGenerated{}
	req := httplib.Post(`https://api.m.jd.com/client.action?functionId=getJingBeanBalanceDetail`)
	req.Header("User-Agent", ua)
	req.Header("Host", "api.m.jd.com")
	req.Header("Content-Type", "application/x-www-form-urlencoded")
	req.Header("Cookie", cookie)
	req.Body(fmt.Sprintf(`body={"pageSize": "20", "page": "%d"}&appid=ld`, page))
	if sysConfig.ProxyUrl != "" {
		proxy := func(req *http.Request) (*url.URL, error) {
			u, _ := url.ParseRequestURI(sysConfig.ProxyUrl)
			return u, nil
		}
		req.SetProxy(proxy)
	}
	data, err := req.Bytes()
	if err != nil {
		return nil
	}
	json.Unmarshal(data, &a)
	return a.DetailList
}

type RedList struct {
	ActivityID        int           `json:"activityId"`
	ActivityName      string        `json:"activityName"`
	BackgroundLinkURL string        `json:"backgroundLinkUrl"`
	Balance           float64       `json:"balance"`
	BeginTime         int64         `json:"beginTime"`
	Channel           string        `json:"channel"`
	Count             int           `json:"count"`
	CreateTime        int64         `json:"createTime"`
	Delay             bool          `json:"delay"`
	Delayed           bool          `json:"delayed"`
	Discount          float64       `json:"discount"`
	EndTime           int           `json:"endTime"`
	HbFlag            string        `json:"hbFlag"`
	HbID              string        `json:"hbId"`
	HbState           int           `json:"hbState"`
	HbType            int           `json:"hbType"`
	HongbaoName       string        `json:"hongbaoName"`
	HourLabel         int           `json:"hourLabel"`
	OrderIDList       []interface{} `json:"orderIdList"`
	OrgFlag           string        `json:"orgFlag,omitempty"`
	OrgLimitStr       string        `json:"orgLimitStr"`
	Pin               string        `json:"pin"`
	SkuLimitStr       string        `json:"skuLimitStr"`
	StationLimitStr   string        `json:"stationLimitStr"`
	Yn                int           `json:"yn"`
	PlatformFlag      string        `json:"platformFlag,omitempty"`
	UseLinkURL        string        `json:"useLinkUrl,omitempty"`
}

func redPacket(cookie string, rpc chan []RedList) {
	type UseRedInfo struct {
		Count       int       `json:"count"`
		HongBaoList []RedList `json:"hongBaoList"`
		Message     string    `json:"message"`
		PageNum     int       `json:"pageNum"`
		PageSize    int       `json:"pageSize"`
		Pin         string    `json:"pin"`
		ResultCode  int       `json:"resultCode"`
		Success     bool      `json:"success"`
		Tid         int       `json:"tid"`
	}

	a := UseRedInfo{}
	req := httplib.Get(`https://api.m.jd.com/client.action?functionId=myhongbao_getUsableHongBaoList&body=%7B%22appId%22%3A%22appHongBao%22%2C%22appToken%22%3A%22apphongbao_token%22%2C%22platformId%22%3A%22appHongBao%22%2C%22platformToken%22%3A%22apphongbao_token%22%2C%22platform%22%3A%221%22%2C%22orgType%22%3A%222%22%2C%22country%22%3A%22cn%22%2C%22childActivityId%22%3A%22-1%22%2C%22childActiveName%22%3A%22-1%22%2C%22childActivityTime%22%3A%22-1%22%2C%22childActivityUrl%22%3A%22-1%22%2C%22openId%22%3A%22-1%22%2C%22activityArea%22%3A%22-1%22%2C%22applicantErp%22%3A%22-1%22%2C%22eid%22%3A%22-1%22%2C%22fp%22%3A%22-1%22%2C%22shshshfp%22%3A%22-1%22%2C%22shshshfpa%22%3A%22-1%22%2C%22shshshfpb%22%3A%22-1%22%2C%22jda%22%3A%22-1%22%2C%22activityType%22%3A%221%22%2C%22isRvc%22%3A%22-1%22%2C%22pageClickKey%22%3A%22-1%22%2C%22extend%22%3A%22-1%22%2C%22organization%22%3A%22JD%22%7D&appid=JDReactMyRedEnvelope&client=apple&clientVersion=7.0.0`)
	req.Header("User-Agent", "jdapp;iPhone;9.4.4;14.3;network/4g;Mozilla/5.0 (iPhone; CPU iPhone OS 14_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148;supportJDSHWK/1")
	req.Header("Host", "api.m.jd.com")
	req.Header("Accept", "*/*")
	req.Header("Connection", "keep-alive")
	req.Header("Accept-Language", "zh-cn")
	req.Header("Accept-Encoding", "gzip, deflate, br")
	req.Header("Referer", "https://h5.jd.com/")
	req.Header("Cookie", cookie)
	data, _ := req.Bytes()
	json.Unmarshal(data, &a)
	rpc <- a.HongBaoList
}

func initFarm(cookie string, state chan string) {
	type RightUpResouces struct {
		AdvertID string `json:"advertId"`
		Name     string `json:"name"`
		AppImage string `json:"appImage"`
		AppLink  string `json:"appLink"`
		CxyImage string `json:"cxyImage"`
		CxyLink  string `json:"cxyLink"`
		Type     string `json:"type"`
		OpenLink bool   `json:"openLink"`
	}
	type TurntableInit struct {
		TimeState int `json:"timeState"`
	}
	type MengchongResouce struct {
		AdvertID string `json:"advertId"`
		Name     string `json:"name"`
		AppImage string `json:"appImage"`
		AppLink  string `json:"appLink"`
		CxyImage string `json:"cxyImage"`
		CxyLink  string `json:"cxyLink"`
		Type     string `json:"type"`
		OpenLink bool   `json:"openLink"`
	}
	type GUIDPopupTask struct {
		GUIDPopupTask string `json:"guidPopupTask"`
	}
	type IosConfigResouces struct {
		AdvertID string `json:"advertId"`
		Name     string `json:"name"`
		AppImage string `json:"appImage"`
		AppLink  string `json:"appLink"`
		CxyImage string `json:"cxyImage"`
		CxyLink  string `json:"cxyLink"`
		Type     string `json:"type"`
		OpenLink bool   `json:"openLink"`
	}
	type TodayGotWaterGoalTask struct {
		CanPop bool `json:"canPop"`
	}
	type LeftUpResouces struct {
		AdvertID string `json:"advertId"`
		Name     string `json:"name"`
		AppImage string `json:"appImage"`
		AppLink  string `json:"appLink"`
		CxyImage string `json:"cxyImage"`
		CxyLink  string `json:"cxyLink"`
		Type     string `json:"type"`
		OpenLink bool   `json:"openLink"`
	}
	type RightDownResouces struct {
		AdvertID string `json:"advertId"`
		Name     string `json:"name"`
		AppImage string `json:"appImage"`
		AppLink  string `json:"appLink"`
		CxyImage string `json:"cxyImage"`
		CxyLink  string `json:"cxyLink"`
		Type     string `json:"type"`
		OpenLink bool   `json:"openLink"`
	}
	type FarmUserPro struct {
		TotalEnergy     int    `json:"totalEnergy"`
		TreeState       int    `json:"treeState"`
		CreateTime      int64  `json:"createTime"`
		TreeEnergy      int    `json:"treeEnergy"`
		TreeTotalEnergy int    `json:"treeTotalEnergy"`
		ShareCode       string `json:"shareCode"`
		WinTimes        int    `json:"winTimes"`
		NickName        string `json:"nickName"`
		CouponKey       string `json:"couponKey"`
		CouponID        string `json:"couponId"`
		CouponEndTime   int64  `json:"couponEndTime"`
		Type            string `json:"type"`
		SimpleName      string `json:"simpleName"`
		Name            string `json:"name"`
		GoodsImage      string `json:"goodsImage"`
		SkuID           string `json:"skuId"`
		LastLoginDate   int64  `json:"lastLoginDate"`
		NewOldState     int    `json:"newOldState"`
		OldMarkComplete int    `json:"oldMarkComplete"`
		CommonState     int    `json:"commonState"`
		PrizeLevel      int    `json:"prizeLevel"`
	}
	type LeftDownResouces struct {
		AdvertID string `json:"advertId"`
		Name     string `json:"name"`
		AppImage string `json:"appImage"`
		AppLink  string `json:"appLink"`
		CxyImage string `json:"cxyImage"`
		CxyLink  string `json:"cxyLink"`
		Type     string `json:"type"`
		OpenLink bool   `json:"openLink"`
	}
	type LoadFriend struct {
		Code            string      `json:"code"`
		StatisticsTimes interface{} `json:"statisticsTimes"`
		SysTime         int64       `json:"sysTime"`
		Message         interface{} `json:"message"`
		FirstAddUser    bool        `json:"firstAddUser"`
	}
	type AutoGenerated struct {
		Code                  string                `json:"code"`
		RightUpResouces       RightUpResouces       `json:"rightUpResouces"`
		TurntableInit         TurntableInit         `json:"turntableInit"`
		IosShieldConfig       interface{}           `json:"iosShieldConfig"`
		MengchongResouce      MengchongResouce      `json:"mengchongResouce"`
		ClockInGotWater       bool                  `json:"clockInGotWater"`
		GUIDPopupTask         GUIDPopupTask         `json:"guidPopupTask"`
		ToFruitEnergy         int                   `json:"toFruitEnergy"`
		StatisticsTimes       interface{}           `json:"statisticsTimes"`
		SysTime               int64                 `json:"sysTime"`
		CanHongbaoContineUse  bool                  `json:"canHongbaoContineUse"`
		ToFlowTimes           int                   `json:"toFlowTimes"`
		IosConfigResouces     IosConfigResouces     `json:"iosConfigResouces"`
		TodayGotWaterGoalTask TodayGotWaterGoalTask `json:"todayGotWaterGoalTask"`
		LeftUpResouces        LeftUpResouces        `json:"leftUpResouces"`
		MinSupportAPPVersion  string                `json:"minSupportAPPVersion"`
		LowFreqStatus         int                   `json:"lowFreqStatus"`
		FunCollectionHasLimit bool                  `json:"funCollectionHasLimit"`
		Message               interface{}           `json:"message"`
		TreeState             int                   `json:"treeState"`
		RightDownResouces     RightDownResouces     `json:"rightDownResouces"`
		IconFirstPurchaseInit bool                  `json:"iconFirstPurchaseInit"`
		ToFlowEnergy          int                   `json:"toFlowEnergy"`
		FarmUserPro           FarmUserPro           `json:"farmUserPro"`
		RetainPopupLimit      int                   `json:"retainPopupLimit"`
		ToBeginEnergy         int                   `json:"toBeginEnergy"`
		LeftDownResouces      LeftDownResouces      `json:"leftDownResouces"`
		EnableSign            bool                  `json:"enableSign"`
		LoadFriend            LoadFriend            `json:"loadFriend"`
		HadCompleteXgTask     bool                  `json:"hadCompleteXgTask"`
		OldUserIntervalTimes  []int                 `json:"oldUserIntervalTimes"`
		ToFruitTimes          int                   `json:"toFruitTimes"`
		OldUserSendWater      []string              `json:"oldUserSendWater"`
	}
	a := AutoGenerated{}
	req := httplib.Post(`https://api.m.jd.com/client.action?functionId=initForFarm`)
	req.Header("accept", "*/*")
	req.Header("accept-encoding", "gzip, deflate, br")
	req.Header("accept-language", "zh-CN,zh;q=0.9")
	req.Header("cache-control", "no-cache")
	req.Header("cookie", cookie)
	req.Header("origin", "https://home.m.jd.com")
	req.Header("pragma", "no-cache")
	req.Header("referer", "https://home.m.jd.com/myJd/newhome.action")
	req.Header("sec-fetch-dest", "empty")
	req.Header("sec-fetch-mode", "cors")
	req.Header("sec-fetch-site", "same-site")
	req.Header("User-Agent", ua)
	req.Header("Content-Type", "application/x-www-form-urlencoded")
	req.Body(`body={"version":4}&appid=wh5&clientVersion=9.1.0`)
	if sysConfig.ProxyUrl != "" {
		proxy := func(req *http.Request) (*url.URL, error) {
			u, _ := url.ParseRequestURI(sysConfig.ProxyUrl)
			return u, nil
		}
		req.SetProxy(proxy)
	}
	data, _ := req.Bytes()
	json.Unmarshal(data, &a)

	rt := a.FarmUserPro.Name
	if rt == "" {
		rt = "数据加载中"
	} else {
		if a.TreeState == 2 || a.TreeState == 3 {
			rt += "已可领取⏰"
		} else if a.TreeState == 1 {
			rt += fmt.Sprintf("种植中，进度%.2f%%🍒", 100*float64(a.FarmUserPro.TreeEnergy)/float64(a.FarmUserPro.TreeTotalEnergy))
		} else if a.TreeState == 0 {
			rt = "您忘了种植新的水果⏰"
		}
	}
	state <- rt
}

func jsGold(cookie string, state chan int64) { //

	type BalanceVO struct {
		CashBalance       string `json:"cashBalance"`
		EstimatedAmount   string `json:"estimatedAmount"`
		ExchangeGold      string `json:"exchangeGold"`
		FormatGoldBalance string `json:"formatGoldBalance"`
		GoldBalance       int    `json:"goldBalance"`
	}
	type Gears struct {
		Amount         string `json:"amount"`
		ExchangeAmount string `json:"exchangeAmount"`
		Order          int    `json:"order"`
		Status         int    `json:"status"`
		Type           int    `json:"type"`
	}
	type Data struct {
		Advertise      string    `json:"advertise"`
		BalanceVO      BalanceVO `json:"balanceVO"`
		Gears          []Gears   `json:"gears"`
		IsGetCoupon    bool      `json:"isGetCoupon"`
		IsGetCouponEid bool      `json:"isGetCouponEid"`
		IsLogin        bool      `json:"isLogin"`
		NewPeople      bool      `json:"newPeople"`
	}
	type AutoGenerated struct {
		Code      int    `json:"code"`
		Data      Data   `json:"data"`
		IsSuccess bool   `json:"isSuccess"`
		Message   string `json:"message"`
		RequestID string `json:"requestId"`
	}
	a := AutoGenerated{}
	req := httplib.Post(`https://api.m.jd.com?functionId=MyAssetsService.execute&appid=market-task-h5`)
	req.Header("Accept", "application/json, text/plain, */*")
	req.Header("Accept-Encoding", "gzip, deflate, br")
	req.Header("Cookie", cookie)
	req.Header("Content-Type", "application/x-www-form-urlencoded")
	req.Header("Origin", "https://gold.jd.com")
	req.Header("Host", "api.m.jd.com")
	req.Header("Connection", "keep-alive")
	req.Header("User-Agent", ua)
	req.Header("Referer", "https://gold.jd.com/")
	req.Body(`functionId=MyAssetsService.execute&body={"method":"goldShopPage","data":{"channel":1}}&_t=` + fmt.Sprint(time.Now().Unix()) + `&appid=market-task-h5;`)
	data, _ := req.Bytes()
	json.Unmarshal(data, &a)
	state <- int64(a.Data.BalanceVO.GoldBalance)
}
