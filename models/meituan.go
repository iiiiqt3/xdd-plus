package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/cdle/xdd/vweb"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type PacketInfo struct {
	Code int `json:"code"`
	Data struct {
		PlayerBaseModel struct {
			ActivityCycleInfo struct {
				CashToken  int   `json:"cashToken"`
				CycleId    int   `json:"cycleId"`
				CoinToken  int   `json:"coinToken"`
				ExpireTime int64 `json:"expireTime"`
			} `json:"activityCycleInfo"`
			WithdrawInfoList []struct {
				CashTokenCost int `json:"cashTokenCost"`
				Id            int `json:"id"`
				Times         int `json:"times"`
				CoinTokenCost int `json:"coinTokenCost"`
				Type          int `json:"type"`
			} `json:"withdrawInfoList"`
			LotteryInfo struct {
				LastRefreshTime        int64 `json:"lastRefreshTime"`
				DailyDrawTimes         int   `json:"dailyDrawTimes"`
				LeftLotteryTimesAmount int   `json:"leftLotteryTimesAmount"`
			} `json:"lotteryInfo"`
			RedPacketInfo struct {
				DailyOpenWithdrawRedPacketTimes int   `json:"dailyOpenWithdrawRedPacketTimes"`
				LastRefreshTime                 int64 `json:"lastRefreshTime"`
				TotalOpenRedPacketTimes         int   `json:"totalOpenRedPacketTimes"`
				DailyOpenNormalRedPacketTimes   int   `json:"dailyOpenNormalRedPacketTimes"`
				LeftNormalRedPacketAmount       int   `json:"leftNormalRedPacketAmount"`
			} `json:"redPacketInfo"`
		} `json:"playerBaseModel"`
		RewardModelList []struct {
			Amount                 int         `json:"amount"`
			Seq                    int         `json:"seq"`
			BigReward              bool        `json:"bigReward"`
			Rewarded               bool        `json:"rewarded"`
			RewardedTime           int64       `json:"rewardedTime"`
			ResourceId             int         `json:"resourceId"`
			MaxAvailableAmount     int         `json:"maxAvailableAmount"`
			ResourceType           int         `json:"resourceType"`
			RewardedResourceAmount int         `json:"rewardedResourceAmount"`
			RewardedCouponModel    interface{} `json:"rewardedCouponModel"`
		} `json:"rewardModelList"`
		XtbAmount int `json:"xtbAmount"`
	} `json:"data"`
	ProtocolId int    `json:"protocolId"`
	ServerTime int64  `json:"serverTime"`
	Desc       string `json:"desc"`
}

type SignInfo struct {
	Code int `json:"code"`
	Data struct {
		PlayerBaseModel struct {
			ActivityCycleInfo struct {
				CashToken  int   `json:"cashToken"`
				CycleId    int   `json:"cycleId"`
				CoinToken  int   `json:"coinToken"`
				ExpireTime int64 `json:"expireTime"`
			} `json:"activityCycleInfo"`
			WithdrawInfoList []struct {
				CashTokenCost int `json:"cashTokenCost"`
				Id            int `json:"id"`
				Times         int `json:"times"`
				CoinTokenCost int `json:"coinTokenCost"`
				Type          int `json:"type"`
			} `json:"withdrawInfoList"`
			LotteryInfo struct {
				LastRefreshTime        int64 `json:"lastRefreshTime"`
				DailyDrawTimes         int   `json:"dailyDrawTimes"`
				LeftLotteryTimesAmount int   `json:"leftLotteryTimesAmount"`
			} `json:"lotteryInfo"`
			RedPacketInfo struct {
				DailyOpenWithdrawRedPacketTimes int   `json:"dailyOpenWithdrawRedPacketTimes"`
				LastRefreshTime                 int64 `json:"lastRefreshTime"`
				TotalOpenRedPacketTimes         int   `json:"totalOpenRedPacketTimes"`
				DailyOpenNormalRedPacketTimes   int   `json:"dailyOpenNormalRedPacketTimes"`
				LeftNormalRedPacketAmount       int   `json:"leftNormalRedPacketAmount"`
			} `json:"redPacketInfo"`
		} `json:"playerBaseModel"`
		SignInPopModel struct {
			Title           string `json:"title"`
			RewardModelList []struct {
				StartTime       int64 `json:"startTime"`
				RewardModelList []struct {
					Amount                 int         `json:"amount"`
					Seq                    int         `json:"seq"`
					BigReward              bool        `json:"bigReward"`
					Rewarded               bool        `json:"rewarded"`
					RewardedTime           int64       `json:"rewardedTime"`
					ResourceId             int         `json:"resourceId"`
					MaxAvailableAmount     int         `json:"maxAvailableAmount"`
					ResourceType           int         `json:"resourceType"`
					RewardedResourceAmount int         `json:"rewardedResourceAmount"`
					RewardedCouponModel    interface{} `json:"rewardedCouponModel"`
				} `json:"rewardModelList"`
				State     int  `json:"state"`
				Current   bool `json:"current"`
				Ratio     int  `json:"ratio"`
				TargetDay int  `json:"targetDay"`
			} `json:"rewardModelList"`
		} `json:"signInPopModel"`
		RemitNotificationModelList []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			IconUrl string `json:"iconUrl"`
		} `json:"remitNotificationModelList"`
	} `json:"data"`
	ProtocolId int    `json:"protocolId"`
	ServerTime int64  `json:"serverTime"`
	Desc       string `json:"desc"`
}

type LotteryInfo struct {
	Code int `json:"code"`
	Data struct {
		PlayerBaseModel struct {
			ActivityCycleInfo struct {
				CashToken  int   `json:"cashToken"`
				CycleId    int   `json:"cycleId"`
				CoinToken  int   `json:"coinToken"`
				ExpireTime int64 `json:"expireTime"`
			} `json:"activityCycleInfo"`
			WithdrawInfoList []struct {
				CashTokenCost int `json:"cashTokenCost"`
				Id            int `json:"id"`
				Times         int `json:"times"`
				CoinTokenCost int `json:"coinTokenCost"`
				Type          int `json:"type"`
			} `json:"withdrawInfoList"`
			LotteryInfo struct {
				LastRefreshTime        int64 `json:"lastRefreshTime"`
				DailyDrawTimes         int   `json:"dailyDrawTimes"`
				LeftLotteryTimesAmount int   `json:"leftLotteryTimesAmount"`
			} `json:"lotteryInfo"`
			RedPacketInfo struct {
				DailyOpenWithdrawRedPacketTimes int   `json:"dailyOpenWithdrawRedPacketTimes"`
				LastRefreshTime                 int64 `json:"lastRefreshTime"`
				TotalOpenRedPacketTimes         int   `json:"totalOpenRedPacketTimes"`
				DailyOpenNormalRedPacketTimes   int   `json:"dailyOpenNormalRedPacketTimes"`
				LeftNormalRedPacketAmount       int   `json:"leftNormalRedPacketAmount"`
			} `json:"redPacketInfo"`
		} `json:"playerBaseModel"`
		CurrentRewardList []struct {
			Amount                 int         `json:"amount"`
			Seq                    int         `json:"seq"`
			BigReward              bool        `json:"bigReward"`
			Rewarded               bool        `json:"rewarded"`
			RewardedTime           int64       `json:"rewardedTime"`
			ResourceId             int         `json:"resourceId"`
			MaxAvailableAmount     int         `json:"maxAvailableAmount"`
			ResourceType           int         `json:"resourceType"`
			RewardedResourceAmount int         `json:"rewardedResourceAmount"`
			RewardedCouponModel    interface{} `json:"rewardedCouponModel"`
		} `json:"currentRewardList"`
		XtbAmount       int `json:"xtbAmount"`
		RewardModelList []struct {
			Amount                 int   `json:"amount"`
			Seq                    int   `json:"seq"`
			BigReward              bool  `json:"bigReward"`
			Rewarded               bool  `json:"rewarded"`
			RewardedTime           int64 `json:"rewardedTime"`
			ResourceId             int   `json:"resourceId"`
			MaxAvailableAmount     int   `json:"maxAvailableAmount"`
			ResourceType           int   `json:"resourceType"`
			RewardedResourceAmount int   `json:"rewardedResourceAmount"`
			RewardedCouponModel    *struct {
				UseRule           string      `json:"useRule"`
				Uiinfo            interface{} `json:"uiinfo"`
				TabId             interface{} `json:"tabId"`
				FaceValue         string      `json:"faceValue"`
				Desc              string      `json:"desc"`
				SkuId             int         `json:"skuId"`
				Price             interface{} `json:"price"`
				Icon              string      `json:"icon"`
				TotalStorage      string      `json:"totalStorage"`
				Storage           string      `json:"storage"`
				CycleSurplusStock interface{} `json:"cycleSurplusStock"`
				Name              string      `json:"name"`
				CycleStock        interface{} `json:"cycleStock"`
			} `json:"rewardedCouponModel"`
		} `json:"rewardModelList"`
	} `json:"data"`
	ProtocolId int    `json:"protocolId"`
	ServerTime int64  `json:"serverTime"`
	Desc       string `json:"desc"`
}

type MeiTuan struct {
	ID         int     `gorm:"column:ID;primaryKey"`
	CreateAt   string  `gorm:"column:CreateAt"`
	LoseAt     string  `gorm:"column:LoseAt"`
	UpdateAt   string  `gorm:"column:UpdateAt"`
	Token      string  `gorm:"column:Token"`
	Note       string  `gorm:"column:Note"`
	Available  string  `gorm:"column:Available;default:true" validate:"oneof=true false"`
	Nickname   string  `gorm:"column:Nickname"`
	UserId     string  `gorm:"column:UserId"`
	QQ         int     `gorm:"column:QQ"`
	WeiXin     string  `gorm:"column:WeiXin"`
	PushPlus   string  `gorm:"column:PushPlus"`
	WxPush     string  `gorm:"column:WxPush"`
	Telegram   int     `gorm:"column:Telegram"`
	CashToken  float64 `gorm:"column:CashToken"`
	CoinToken  string  `gorm:"column:CoinToken"`
	ExpireTime string  `gorm:"column:ExpireTime"`
	UUID       string  `gorm:"column:UUID"`
	AcToken    string  `gorm:"column:AcToken"`
}

type MBody struct {
	AcToken    string `json:"acToken"`
	RiskParams struct {
		Ip          string `json:"ip"`
		Fingerprint string `json:"fingerprint"`
		CityId      string `json:"cityId"`
		Platform    int    `json:"platform"`
		App         int    `json:"app"`
		Version     string `json:"version"`
		Uuid        string `json:"uuid"`
	} `json:"riskParams"`
	ProtocolId int    `json:"protocolId"`
	Data       string `json:"data"`
}

type TaskList struct {
	ProtocolId int `json:"protocolId"`
	Data       struct {
		GuideInfo struct {
			FirstRedPacketGuide  bool `json:"firstRedPacketGuide"`
			SecondRedPacketGuide bool `json:"secondRedPacketGuide"`
			CoinTokenGuide       bool `json:"coinTokenGuide"`
			XtbGuide             bool `json:"xtbGuide"`
		} `json:"guideInfo"`
		TaskInfoList []struct {
			Id               int `json:"id"`
			Status           int `json:"status"`
			Process          int `json:"process"`
			DailyFinishTimes int `json:"dailyFinishTimes"`
			DailyRewardTimes int `json:"dailyRewardTimes"`
			MgcTaskBaseInfo  struct {
				ViewTitle               string `json:"viewTitle"`
				ViewContent             string `json:"viewContent"`
				ViewProcessName         string `json:"viewProcessName"`
				ViewTips                string `json:"viewTips"`
				ViewJumpUrl             string `json:"viewJumpUrl"`
				MinLimit                int    `json:"minLimit"`
				MaxLimit                int    `json:"maxLimit"`
				ViewExtraJson           string `json:"viewExtraJson"`
				Order                   int    `json:"order"`
				CurPeriodMaxFinishTimes int    `json:"curPeriodMaxFinishTimes"`
			} `json:"mgcTaskBaseInfo"`
			ExtraContent     string `json:"extraContent"`
			TotalRewardTimes int    `json:"totalRewardTimes"`
			TotalInitTimes   int    `json:"totalInitTimes"`
			TotalFinishTimes int    `json:"totalFinishTimes"`
			TotalFailTimes   int    `json:"totalFailTimes"`
			MgcTaskExtraData struct {
				NextAvailableFinishTime int `json:"nextAvailableFinishTime"`
				CurPeriodParams         struct {
				} `json:"curPeriodParams"`
				AllPeriodParams struct {
				} `json:"allPeriodParams"`
			} `json:"mgcTaskExtraData"`
		} `json:"taskInfoList"`
		SwitchMsg []struct {
			Type   int `json:"type"`
			Status int `json:"status"`
		} `json:"switchMsg"`
		Nickname        string `json:"nickname"`
		Portrait        string `json:"portrait"`
		PlayerBaseModel struct {
			ActivityCycleInfo struct {
				CashToken            int           `json:"cashToken"`
				CoinToken            int           `json:"coinToken"`
				ExpireTime           int64         `json:"expireTime"`
				TokenExchangeTimes   int           `json:"tokenExchangeTimes"`
				RandomWithdrawChance []interface{} `json:"randomWithdrawChance"`
				LargeWithdrawChance  []interface{} `json:"largeWithdrawChance"`
				CycleId              int           `json:"cycleId"`
			} `json:"activityCycleInfo"`
			LotteryInfo struct {
				LeftLotteryTimesAmount int         `json:"leftLotteryTimesAmount"`
				DailyDrawTimes         int         `json:"dailyDrawTimes"`
				LastRefreshTime        int64       `json:"lastRefreshTime"`
				RecentlyLotteryResult  interface{} `json:"recentlyLotteryResult"`
			} `json:"lotteryInfo"`
			RedPacketInfo struct {
				LeftRedPacketAmount     int   `json:"leftRedPacketAmount"`
				DailyOpenRedPacketTimes int   `json:"dailyOpenRedPacketTimes"`
				LastRefreshTime         int64 `json:"lastRefreshTime"`
				TotalOpenRedPacketTimes int   `json:"totalOpenRedPacketTimes"`
			} `json:"redPacketInfo"`
		} `json:"playerBaseModel"`
		SignInPopModel struct {
			Title           string `json:"title"`
			RewardModelList []struct {
				TargetDay       int   `json:"targetDay"`
				Ratio           int   `json:"ratio"`
				State           int   `json:"state"`
				Current         bool  `json:"current"`
				StartTime       int64 `json:"startTime"`
				RewardModelList []struct {
					ResourceId             int         `json:"resourceId"`
					ResourceType           int         `json:"resourceType"`
					Rewarded               bool        `json:"rewarded"`
					RewardedTime           int         `json:"rewardedTime"`
					Amount                 int         `json:"amount"`
					RewardedResourceAmount int         `json:"rewardedResourceAmount"`
					RewardedCouponModel    interface{} `json:"rewardedCouponModel"`
					MaxAvailableAmount     int         `json:"maxAvailableAmount"`
					Seq                    int         `json:"seq"`
				} `json:"rewardModelList"`
			} `json:"rewardModelList"`
		} `json:"signInPopModel"`
		PopModels []struct {
			Id          int    `json:"id"`
			Position    int    `json:"position"`
			IconUrl     string `json:"iconUrl"`
			BottomTitle string `json:"bottomTitle"`
			TopTitle    string `json:"topTitle"`
			JumpUrl     string `json:"jumpUrl"`
		} `json:"popModels"`
		Rule                     string `json:"rule"`
		RedPacketGiveLotteryConf string `json:"redPacketGiveLotteryConf"`
		CreateTime               int64  `json:"createTime"`
		RedPacketNumInitToday    int    `json:"redPacketNumInitToday"`
		XtbAmount                int    `json:"xtbAmount"`
		UpdateVersion            bool   `json:"updateVersion"`
		Version                  int    `json:"version"`
	} `json:"data"`
	Code       int    `json:"code"`
	Desc       string `json:"desc"`
	ServerTime int64  `json:"serverTime"`
}

func NewBody(acToken string, uuid string, data string, ProtocolId int) MBody {
	return MBody{
		AcToken: acToken,
		RiskParams: struct {
			Ip          string `json:"ip"`
			Fingerprint string `json:"fingerprint"`
			CityId      string `json:"cityId"`
			Platform    int    `json:"platform"`
			App         int    `json:"app"`
			Version     string `json:"version"`
			Uuid        string `json:"uuid"`
		}(struct {
			Ip          string
			Fingerprint string
			CityId      string
			Platform    int
			App         int
			Version     string
			Uuid        string
		}{
			Ip:          "",
			Fingerprint: "undefined",
			CityId:      "1",
			Platform:    4,
			App:         0,
			Version:     "12.9.209",
			Uuid:        uuid,
		}),
		ProtocolId: ProtocolId,
		Data:       data,
	}
}

func getUA() string {
	rand.Seed(time.Now().UnixNano())
	androidVersion := strconv.Itoa(rand.Intn(3)+10) + ".0"                                                // 随机生成 Android 版本号
	chromeVersion := strconv.Itoa(rand.Intn(11)+80) + ".0." + strconv.Itoa(rand.Intn(1001)+4000) + ".210" // 随机生成 Chrome 版本号
	uaString := fmt.Sprintf("Mozilla/5.0 (Linux; Android %s; %s) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/%s Mobile Safari/537.36 TitansX/12.9.1 KNB/1.2.0 android/%s mt/com.sankuai.meituan/12.9.209 App/10120/12.9.209 MeituanGroup/12.9.209",
		androidVersion, getModel(), chromeVersion, androidVersion)
	return uaString
}

func getModel() string {
	// 实现获取设备型号的逻辑
	// 返回设备型号字符串
	models := []string{"M2012K10C", "22041211AC", "ABR-AL80", "AGT-AN00", "M2011K2C"}
	rand.Seed(time.Now().UnixNano())
	return models[rand.Intn(len(models))]
}

func (cookie *MeiTuan) CheckDownLine() bool {
	if cookie.Available == False {
		return false
	}
	info := GetUserInfo(cookie.Token)
	val, _ := jsonparser.GetInt(info, "code")
	if val == 401 {
		cookie.Updates(MeiTuan{
			Available: False,
			LoseAt:    Date(),
		})
		return false
	}
	return true
}

func UpLine(token string, sender *Sender) bool {
	info := GetUserInfo(token)
	val, _ := jsonparser.GetInt(info, "error", "code")
	if val == 401 {
		sender.Reply("您的CK已失效，请重新提交")
		return false
	} else {
		date := Date()
		id, _ := jsonparser.GetInt(info, "user", "id")
		tuan, err := getMeiTuan(strconv.FormatInt(id, 10))
		if err != nil {
			Username, _ := jsonparser.GetString(info, "user", "auditUsername")
			ck := MeiTuan{
				ID:        int(id),
				CreateAt:  date,
				LoseAt:    "",
				UpdateAt:  date,
				Token:     token,
				Note:      "",
				Available: True,
				Nickname:  Username,
				UserId:    strconv.FormatInt(id, 10),
				QQ:        sender.UserID,
				WeiXin:    sender.WxId,
				PushPlus:  "",
				WxPush:    "",
				Telegram:  0,
			}
			tx := db.Begin()
			if err := tx.Create(ck).Error; err != nil {
				tx.Rollback()
				return true
			}
			tx.Commit()
			sender.Reply(fmt.Sprintf("美团账号新增成功:%s", Username))
		} else {
			tuan.Updates(MeiTuan{UpdateAt: Date(), Token: token, QQ: sender.UserID, WeiXin: sender.WxId})
			sender.Reply("美团账号更新成功")
		}
		return true
	}
}

func UpLine2(token string, sender *Sender) bool {
	info := GetUserInfo(token)
	val, _ := jsonparser.GetInt(info, "error", "code")
	if val == 401 {
		//sender.Reply("您的CK已失效")
		return false
	} else {
		return true
	}
}

func getMeiTuan(id string) (*MeiTuan, error) {
	ck := &MeiTuan{}
	return ck, db.Where("id = ?", id).First(ck).Error
}

func (ck *MeiTuan) Updates(values interface{}) {
	if ck.ID != 0 {
		db.Model(ck).Updates(values)
		return
	}
}

func (ck *MeiTuan) Update(column string, value interface{}) {
	if ck.ID != 0 {
		db.Model(ck).Update(column, value)
		return
	}
}

func GetMTCookies(sbs ...func(sb *gorm.DB) *gorm.DB) []MeiTuan {
	var cks []MeiTuan
	tb := db
	for _, sb := range sbs {
		tb = sb(tb)
	}
	tb.Order("ID asc").Find(&cks)
	return cks
}

func GetMeiTuan(sender *Sender) []MeiTuan {
	switch sender.Type {
	case "qq", "qqg", "tg":
		return GetMTCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s = ?  ", QQ), sender.UserID)
		})
	case "wx", "wxg":
		return GetMTCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s = ?  ", "WeiXin"), sender.WxId)
		})
	default:
		return nil
	}
	return nil
}

func (ck *MeiTuan) Query() string {
	msgs := []string{
		fmt.Sprintf("账号昵称：%s", ck.Nickname),
	}
	if ck.Note != "" {
		msgs = append(msgs, fmt.Sprintf("账号备注：%s", ck.Note))
	}

	if ck.CheckDownLine() {
		if ck.UpdateAt != "" {
			parse1, _ := time.Parse("2006-01-02", ck.UpdateAt)
			logs.Info(parse1)
			msgs = append(msgs, fmt.Sprintf("最后更新时间：%s", parse1.Format("2006-01-02")))
		}
		if ck.CoinToken == "" {
			msgs = append(msgs, "暂无赚金币数据")
		} else {
			msgs = append(msgs, fmt.Sprintf("赚金币余额:%f", ck.CashToken))
			msgs = append(msgs, fmt.Sprintf("赚金币金币:%s", ck.CoinToken))
			msgs = append(msgs, fmt.Sprintf("赚金币金币:%s", ck.CoinToken))
			msgs = append(msgs, fmt.Sprintf("赚金币过期时间:%s", ck.CoinToken))
		}
	} else {
		parse1, _ := time.Parse("2006-01-02", ck.LoseAt)
		msgs = append(msgs, fmt.Sprintf("提醒：该账号已过期，请重新登录,失效时间:%s", parse1.Format("2006-01-02")))
	}

	return strings.Join(msgs, "\n")

}

func GetUserInfo(token string) []byte {

	url := "https://open.meituan.com/user/v1/info/auditting?fields=auditAvatarUrl,auditUsername"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Host", "open.meituan.com")
	req.Header.Add("X-Titans-User", "")
	req.Header.Add("Origin", "https://mtaccount.meituan.com")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("User-Agent", getUA())
	req.Header.Add("Referer", "https://mtaccount.meituan.com/")
	req.Header.Add("token", token)
	req.Header.Add("Accept-Language", "zh-CN,zh-Hans;q=0.9")
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	fmt.Println(res)
	fmt.Println(string(body))
	return body

}

func (ck *MeiTuan) RunCoin() {
	ck.UUID = GetUUID()
	meituan := LoginMeituan(ck)
	if meituan != 0 {
		logs.Info("登录失败")
		return
	} else {
		taskList(ck)
	}
}

func (ck *MeiTuan) RunTT(sender *Sender, orderNum int) {
	logs.Info("开始领")
	  resp, err := http.Get("https://gitee.com/feiniao520/gr/raw/main/meituan.js")
    if err != nil {
        fmt.Println("获取脚本内容失败:", err)
        return
    }
    defer resp.Body.Close()

    script, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("读取脚本内容失败:", err)
        return
    }

//	file1, _ := vweb.JsFs.ReadFile("js/meituan.js")

	// 创建一个临时文件来保存JavaScript脚本
	file, err := os.CreateTemp(ExecPath+"/scripts", "script.js")
	if err != nil {
		fmt.Println("创建临时文件失败:", err)
		return
	}
	defer os.Remove(file.Name())

	// 将JavaScript脚本写入临时文件
	_, err = file.Write(script)
	if err != nil {
		fmt.Println("写入临时文件失败:", err)
		return
	}

	// 执行JavaScript脚本
	cmd := exec.Command("node", file.Name())
envs := []Env{
		{Name: "meituanCookie", Value: ck.Token},
		{Name: "meituanCommonTask", Value: False},  //#集合任务
		{Name: "meituanMrzqTask", Value: False},  //#每日赚钱
		{Name: "meituanCyfTask", Value: True},		//#抽月符
		{Name: "meituanAutoWithdraw", Value: False},  //#随机提现
		{Name: "meituanXtbTask", Value: False},  //#小团比
		
	}
	for _, env := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("执行JavaScript脚本失败:", err)
		return
	}
	//输出脚本执行结果
	logs.Info(string(output))
	logs.Info(replexQuan(string(output), sender, orderNum))
	sender.Reply(replexQuan(string(output), sender, orderNum))

}

//#修改领取匹配

/*func replexQuan(info string, sender *Sender, orderNum int) string {
	re := regexp.MustCompile(`账号\[\d+\].*?\d+减\d+`)
	matches := re.FindAllString(info, -1)
	msgs := []string{
		fmt.Sprintf("订单类型:美团领券，订单编号：%d，领卷完成，券当天有效，共计领卷%d张,明细如下:", orderNum, len(matches)),
	}
	for _, match := range matches {
		parts := strings.SplitN(match, ":", 2)
		var replacedMsg string
		if sender.Type == "wx" || sender.Type == "wxg" {
			replacedMsg = "[红包]" + parts[1]
		} else {
			replacedMsg = "🧧" + parts[1]
		}
		msgs = append(msgs, replacedMsg)
	}
	return strings.Join(msgs, "\n")
}

*/


func replexQuan(info string, sender *Sender, orderNum int) string {
	// #修改正则表达式以匹配冒号后面的内容，并且含有"减"的内容
	re := regexp.MustCompile(`账号\[\d+\].*?:\s*(.*?减\d+)`)
	matches := re.FindAllStringSubmatch(info, -1)
	msgs := []string{
		fmt.Sprintf("订单类型:美团领券，订单编号：%d，领卷完成，共计领卷%d张,明细如下:", orderNum, len(matches)),
	}
	for _, match := range matches {
		// #match[1] 现在包含冒号后面的内容，且含有"减"的内容
		var replacedMsg string
		if sender.Type == "wx" || sender.Type == "wxg" {
			replacedMsg = "[红包]" + match[1]
		} else {
			replacedMsg = "🧧" + match[1]
		}
		msgs = append(msgs, replacedMsg)
	}
	return strings.Join(msgs, "\n")
}


func replexQuan_50(info string, sender *Sender, orderNum int) string {
	re1 := regexp.MustCompile(`(账号\[.*?\])(每日赚钱余额.*?(\d+金币))`)
	re2 := regexp.MustCompile(`(账号\[.*?\])(钱包余额.*?(\d+元))`)
	matches1 := re1.FindAllStringSubmatch(info, -1)
	matches2 := re2.FindAllStringSubmatch(info, -1)
	msgs := []string{
		fmt.Sprintf("订单类型:美团50，订单编号：%d，美团50-翻红包完成，", orderNum),
	}
	var replacedMsg string
	for _, match := range matches1 {
		if sender.Type == "wx" || sender.Type == "wxg" {
			replacedMsg = strings.Replace(match[0], match[1], "[庆祝]", 1)
		} else {
			replacedMsg = strings.Replace(match[0], match[1], "💸", 1)
		}
		msgs = append(msgs, replacedMsg)
	}

	for _, match := range matches2 {
		if sender.Type == "wx" || sender.Type == "wxg" {
			replacedMsg = strings.Replace(match[0], match[1], "[庆祝]", 1)
		} else {
			replacedMsg = strings.Replace(match[0], match[1], "💸", 1)
		}
		msgs = append(msgs, replacedMsg)
	}
	return strings.Join(msgs, "\n")
}

func (ck *MeiTuan) RunTT_50(sender *Sender, orderNum int) {
	logs.Info("开始领50")

	file1, _ := vweb.JsFs.ReadFile("js/meituan50.js")

	// 创建一个临时文件来保存JavaScript脚本
	file, err := os.CreateTemp(ExecPath+"/scripts", "script.js")
	if err != nil {
		fmt.Println("创建临时文件失败:", err)
		return
	}
	defer os.Remove(file.Name())

	// 将JavaScript脚本写入临时文件
	_, err = file.Write(file1)
	if err != nil {
		fmt.Println("写入临时文件失败:", err)
		return
	}

	// 执行JavaScript脚本
	cmd := exec.Command("node", file.Name())

	envs := []Env{
		{Name: "meituanCookie", Value: ck.Token},
		{Name: "meituanAutoWithdraw", Value: False}, //#关闭 APP每日赚钱，随机提现
		{Name: "meituanLjTask", Value: False},       //#关闭领券
		{Name: "meituanCyfTask", Value: False},		//#抽月符
		{Name: "meituanXtbTask", Value: False},  //#小团比
	}
	for _, env := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("执行JavaScript脚本失败:", err)
		return
	}
	// 输出脚本执行结果
	logs.Info(string(output))
	logs.Info(replexQuan_50(string(output), sender, orderNum))
    sender.Reply(replexQuan_50(string(output), sender, orderNum))

}

func LoginMeituan(meituan *MeiTuan) int64 {
	url := fmt.Sprintf("https://game.meituan.com/earn-daily/login/loginMgc?gameType=10402&mtUserId=%s&mtToken=%s&mtDeviceId=%s&nonceStr=%s&externalStr=%s", meituan.UserId, meituan.Token, meituan.UUID, gen16(), "{\"cityId\":\"1\"}")
	req := httplib.Get(url)
	setHeader(req)
	cookie := fmt.Sprintf("utm_medium=android;uuid=%s;token=%s;mt_c_token=%s;", meituan.UUID, meituan.Token, meituan.Token)
	req.Header("Cookie", cookie)
	body, _ := req.Bytes()
	val, _ := jsonparser.GetInt(body, "code")
	accessToken, _ := jsonparser.GetString(body, "response", "accessToken")
	meituan.AcToken = accessToken
	return val
}

func gen16() string {

	randomString, err := randStringBytesCrypto(16)
	if err != nil {
		panic(err)
	}
	return randomString

}

func randStringBytesCrypto(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func setHeader(req *httplib.BeegoHTTPRequest) {

	req.Header("Host", "game.meituan.com")
	req.Header("Sec-Fetch-Site", "same-site")
	req.Header("Accept", "application/json, text/plain, */*")
	req.Header("X-Requested-With", "XMLHttpRequest")
	req.Header("Accept-Language", "zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header("Content-Type", "application/json")
	req.Header("mToken", "undefined")
	req.Header("Origin", "https://awp.meituan.com")
	req.Header("Referer", "https://awp.meituan.com/")
	req.Header("Sec-Fetch-Mode", "cors")
	req.Header("Connection", "keep-alive")
	req.Header("acToken", "undefined")
	req.Header("Sec-Fetch-Dest", "empty")
	req.Header("content-type", "application/json")
	req.Header("User-Agent", getUA())

}

//utm_medium=android;uuid=%s;token=%s;mt_c_token=%s;

func taskList(meituan *MeiTuan) {
	//声明对象
	lbody := NewBody(meituan.AcToken, meituan.UUID, "{}", 1001)
	jsonData, err := json.Marshal(lbody)
	if err != nil {
		fmt.Println("JSON encoding error:", err)
		return
	}

	req := httplib.Post("https://game.meituan.com/earn-daily/msg/post")
	req.Body(jsonData)
	setHeader(req)
	cookie := fmt.Sprintf("utm_medium=android;uuid=%s;token=%s;mt_c_token=%s;", meituan.UUID, meituan.Token, meituan.Token)
	req.Header("Cookie", cookie)
	body, _ := req.Bytes()
	fmt.Println(string(body))

	//转换
	var list TaskList
	json.Unmarshal(body, &list)
	taskList := list.Data.TaskInfoList
	signState := list.Data.SignInPopModel.RewardModelList

	for _, s := range signState {
		if s.Current {
			if s.State == 2 {
				logs.Info("今日已签到打卡")
				break
			}
			// 签到
			SignIn(meituan)
		}
	}

	for _, s := range taskList {
		if s.Id == 15099 || s.Id == 15278 || s.Id == 780 {
			continue
		}
		times := s.DailyFinishTimes
		finishTimes := s.MgcTaskBaseInfo.CurPeriodMaxFinishTimes
		title := s.MgcTaskBaseInfo.ViewTitle
		if times == finishTimes {
			logs.Info(fmt.Sprintf("任务:%s,已完成", title))
			continue
		}
		num := finishTimes - times
		for i := 0; i < num; i++ {
			mBody := NewBody(meituan.AcToken, meituan.UUID, fmt.Sprintf("{\n    \"taskId\" : %d,\n    \"externalStr\" : \"{\\\"cityId\\\":-1}\"\n  }", s.Id), 1004)
			GoShoping(mBody, cookie)
			logs.Info(fmt.Sprintf("完成任务:%s", title))
			time.Sleep(time.Second * time.Duration(rand.Intn(6)+5))
			mBody1 := NewBody(meituan.AcToken, meituan.UUID, fmt.Sprintf("{\n    \"taskId\" : %d,\n    \"externalStr\" : \"{\\\"cityId\\\":-1}\"\n  }"), 1005)
			GoShoping(mBody1, cookie)
			logs.Info(fmt.Sprintf("领取%s任务奖励成功", title))
		}
	}
	time.Sleep(time.Second * 2)

	body, _ = req.Bytes()
	fmt.Println(body)
	json.Unmarshal(body, &list)
	model := list.Data.PlayerBaseModel
	packetAmount := model.RedPacketInfo.LeftRedPacketAmount
	if packetAmount == 0 {
		logs.Info("今天红包已经开完了")
	} else {
		logs.Info("今天红包个数:%d", packetAmount)
		for i := 0; i < packetAmount; i++ {
			OpenPacket(meituan)
		}
	}

	DrawInfo := list.Data.PlayerBaseModel.LotteryInfo.LeftLotteryTimesAmount
	if DrawInfo == 0 {
		logs.Info("抽奖次数为0")
	} else {
		logs.Info("默认关闭抽奖")
	}

}

func SignIn(meituan *MeiTuan) {
	lbody := NewBody(meituan.AcToken, meituan.UUID, "{}", 1007)
	jsonData, err := json.Marshal(lbody)
	if err != nil {
		fmt.Println("JSON encoding error:", err)
		return
	}

	req := httplib.Post("https://game.meituan.com/earn-daily/msg/post")
	req.Body(jsonData)
	setHeader(req)
	cookie := fmt.Sprintf("utm_medium=android;uuid=%s;token=%s;mt_c_token=%s;", meituan.UUID, meituan.Token, meituan.Token)
	req.Header("Cookie", cookie)
	body, _ := req.Bytes()

	val, _ := jsonparser.GetInt(body, "code")
	if val != 200 {
		logs.Info("签到失败")
		logs.Info(string(body))
	} else {
		var sign SignInfo
		json.Unmarshal(body, &sign)
		modelList := sign.Data.RemitNotificationModelList
		content := modelList[0].Content
		logs.Info(content)
	}
}

func OpenPacket(meituan *MeiTuan) {
	//声明对象
	lbody := NewBody(meituan.AcToken, meituan.UUID, "{}", 1008)
	jsonData, err := json.Marshal(lbody)
	if err != nil {
		fmt.Println("JSON encoding error:", err)
		return
	}
	req := httplib.Post("https://game.meituan.com/earn-daily/msg/post")
	req.Body(jsonData)
	setHeader(req)
	cookie := fmt.Sprintf("utm_medium=android;uuid=%s;token=%s;mt_c_token=%s;", meituan.UUID, meituan.Token, meituan.Token)
	req.Header("Cookie", cookie)
	body, _ := req.Bytes()
	fmt.Println(body)
	var red PacketInfo
	json.Unmarshal(body, &red)
	list := red.Data.RewardModelList
	activityCycleInfo := red.Data.PlayerBaseModel.ActivityCycleInfo
	cashToken := float64(activityCycleInfo.CashToken / 100.0)
	maxCashToken := 49.98
	for _, s := range list {
		if cashToken > maxCashToken {
			logs.Info("开红包获得:%d金币", s.Amount)
		} else {
			logs.Info("开红包获得:%d金额", s.Amount)
		}
	}

}

func GoShoping(body MBody, cookie string) string {
	jsonData, err := json.Marshal(body)
	if err != nil {
		fmt.Println("JSON encoding error:", err)
		return ""
	}
	req := httplib.Post("https://game.meituan.com/earn-daily/msg/post")
	req.Body(jsonData)
	setHeader(req)
	req.Header("Cookie", cookie)
	result, _ := req.Bytes()
	return string(result)

}

// 随机固定UUID
func GetUUID() string {
	rand.Seed(time.Now().UnixNano())
	uuid := fmt.Sprintf("0000000000000%sA%d%d", strings.ReplaceAll(strings.ToUpper(uuid.New().String()), "-", ""), time.Now().UnixMicro(), rand.Intn(89)+10)
	return uuid
}
var orderNum int
var orderNumber1 = 1 // case 1 的订单编号设定初始值为1
var orderNumber2 = 1 // case 2 的订单编号设定初始值为1
var orderNumber3 = 1 // case 3 的订单编号设定初始值为1

func MeituanSelect(sender *Sender, msg chan string, typ int, meituans []MeiTuan) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}

		if n == "q" {
			sender.Reply("退出登录流程")
			meituanList[sender.UserID] = nil
			close(msg)
			return
		}

		num, err := strconv.Atoi(n)
		if err != nil {
			//sender.Reply(fmt.Sprintf("转换失败:%s", err))
			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
			meituanList[sender.UserID] = nil
			return
		}
		//typ 1 美团50  2 美团领券  3美团抢卷

		switch typ {
		case 1:
			//美团50
			regular := `^0$|^[1-9]\d*$`
			reg := regexp.MustCompile(regular)
			if reg.MatchString(n) {
				meiTuans := GetMeiTuan(sender)
				if len(meiTuans) <= num {
					sender.Reply("输入序列号错误，已退出！")
					meituanList[sender.UserID] = nil
					return
				} else {
					orderNum = orderNumber1 // 获取case 1 的当前订单编号
					ck2 := meituans[num].Token
					result2 := UpLine2(ck2, sender)
					if result2 {
						if sender.IsAdmin {
							sender.Reply("开始美团50-翻红包")
						} else {
							value := GetEnv("mt50") // 变量设置美团扣的值，export mtre 10
							if value == "" {
								sender.Reply("管理员未开启美团50-翻红包")
								meituanList[sender.UserID] = nil
								return
							} else {
								coin := GetCoin(sender.UserID)
								jbcoin, _ := strconv.Atoi(value)
								if coin < jbcoin {
									sender.Reply(fmt.Sprintf("积分不足，美团美团50-翻红包需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
									meituanList[sender.UserID] = nil
									return
								}
								RemCoin(sender.UserID, jbcoin)
								sender.Reply(fmt.Sprintf("开始美团50-翻红包，已扣除%d个积分，剩余积分%d，订单编号：%d，", jbcoin, GetCoin(sender.UserID),orderNum,))
							}
						}
					} else {
						sender.Reply("选择序列号账号CK失效，已退出！")
						meituanList[sender.UserID] = nil
						return
					}
				}
			} else {
					sender.Reply("输入序列号错误，已退出！！！")
					meituanList[sender.UserID] = nil
					return
			}
		    orderNumber1++ // 订单编号自增+1
			meituans[num].RunTT_50(sender, orderNum)
			meituanList[sender.UserID] = nil
		case 2:
			//	sender.Reply("已开始领券")
			regular := `^0$|^[1-9]\d*$`
			reg := regexp.MustCompile(regular)
			if reg.MatchString(n) {
				meiTuans := GetMeiTuan(sender)
				if len(meiTuans) <= num {
					sender.Reply("输入序列号错误，已退出！")
					meituanList[sender.UserID] = nil
					return
				} else {
					orderNum = orderNumber2 // 获取case 2 的当前订单编号
					ck2 := meituans[num].Token
					result2 := UpLine2(ck2, sender)
					if result2 {
						if sender.IsAdmin {
							sender.Reply("开始美团领券")
						} else {
							value := GetEnv("mtlq") // 变量设置美团扣的值，export mtlq 10
							if value == "" {
								sender.Reply("管理员未开启美团领券")
								meituanList[sender.UserID] = nil
								return
							} else {
								coin := GetCoin(sender.UserID)
								jbcoin, _ := strconv.Atoi(value)
								if coin < jbcoin {
									sender.Reply(fmt.Sprintf("积分不足，美团领券需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
									meituanList[sender.UserID] = nil
									return
								}
								RemCoin(sender.UserID, jbcoin)
								sender.Reply(fmt.Sprintf("开始美团领券，已扣除%d个积分，剩余积分%d，订单编号：%d，", jbcoin, GetCoin(sender.UserID), orderNum))
							}
						}
					} else {
						sender.Reply("选择序列号账号CK失效，请重新发送：美团登录 命令，按要求提交账号")
						meituanList[sender.UserID] = nil
						return
					}

				}
			} else {
					sender.Reply("输入序列号错误，已退出！！！")
					meituanList[sender.UserID] = nil
					return
			}
		    orderNumber2++          // case 2 的订单编号自增+1
			meituans[num].RunTT(sender, orderNum)
			meituanList[sender.UserID] = nil
		case 3:
			//	sender.Reply("已小团币任务")
			regular := `^0$|^[1-9]\d*$`
			reg := regexp.MustCompile(regular)
			if reg.MatchString(n) {
				meiTuans := GetMeiTuan(sender)
				if len(meiTuans) <= num {
					sender.Reply("输入序列号错误，已退出！")
					meituanList[sender.UserID] = nil
					return
				} else {
					orderNum = orderNumber3 // 获取case 3 的当前订单编号
					ck2 := meituans[num].Token
					result2 := UpLine2(ck2, sender)
					if result2 {
						if sender.IsAdmin {
							sender.Reply("开始运行美团小团币任务,每日一次即可！任务时间较长预计10分钟，请耐心等待")
						} else {
							value := GetEnv("mttb") // 变量设置美团扣的值，export mttb 10
							if value == "" {
								sender.Reply("管理员未开启美团小团币任务")
								meituanList[sender.UserID] = nil
								return
							} else {
								coin := GetCoin(sender.UserID)
								jbcoin, _ := strconv.Atoi(value)
								if coin < jbcoin {
									sender.Reply(fmt.Sprintf("积分不足，小团币需要%d个积分,请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
									meituanList[sender.UserID] = nil
									return
								}
								RemCoin(sender.UserID, jbcoin)
								sender.Reply(fmt.Sprintf("小团币任务开始，每日一次即可！请耐心等待回执，期间发送任何口令，机器人都不会有回复，直到此任务结束。已扣除%d个积分，剩余积分%d，订单编号：%d，", jbcoin, GetCoin(sender.UserID), orderNum))
							}
						}
					} else {
						sender.Reply("选择序列号账号CK失效，请重新登录")
						meituanList[sender.UserID] = nil
						return
					}

				}
			} else {
					sender.Reply("输入序列号错误，已退出！！！")
					meituanList[sender.UserID] = nil
					return
			}
		    orderNumber3++          // case 3 的订单编号自增+1
			meituans[num].RunTT_xtb(sender, orderNum)
			meituanList[sender.UserID] = nil
		case 4:
			sender.Reply("开发中")
			meituanList[sender.UserID] = nil
			return	
		default:
			sender.Reply("暂无对应的渠道,已经退出流程请重新输入")
			meituanList[sender.UserID] = nil
			return
		}

	}
}

func getNodeVersion() {

	// 执行 `node -v` 命令
	cmd := exec.Command("node", "-v")
	output, err := cmd.Output()

	if err != nil {
		// 执行命令时发生错误，可能是因为未安装Node.js或命令不可用
		fmt.Println("未安装Node.js环境")
		return
	}

	// 检查输出是否包含Node.js的版本号
	nodeVersion := strings.TrimSpace(string(output))
	if strings.HasPrefix(nodeVersion, "v") {
		fmt.Println("已安装Node.js环境，版本号:", nodeVersion)
	} else {
		fmt.Println("已安装Node.js环境，但无法确定版本号")
	}
}

// 检查是否已安装Node.js
func isNodeInstalled() bool {
	cmd := exec.Command("node", "-v")
	err := cmd.Run()
	return err == nil
}

// 下载和安装Node.js
func installNode() error {
	// 根据操作系统选择合适的下载链接
	downloadURL := ""
	switch os := runtime.GOOS; os {
	case "darwin":
		downloadURL = "https://nodejs.org/dist/{version}/node-{version}-darwin-x64.tar.gz"
	case "linux":
		downloadURL = "https://nodejs.org/dist/{version}/node-{version}-linux-x64.tar.gz"
	case "windows":
		downloadURL = "https://nodejs.org/dist/{version}/node-{version}-win-x64.zip"
	default:
		return fmt.Errorf("不支持的操作系统：%s", os)
	}

	// 替换下载链接中的"{version}"占位符为实际的版本号
	downloadURL = strings.Replace(downloadURL, "{version}", "v16.8.0", 1)

	// 执行下载和安装命令
	cmd := exec.Command("curl", "-o", "node.tar.gz", downloadURL)
	err := cmd.Run()
	if err != nil {
		return err
	}

	// 解压缩下载的文件
	cmd = exec.Command("tar", "-xzf", "node.tar.gz")
	err = cmd.Run()
	if err != nil {
		return err
	}

	// 将解压后的Node.js二进制文件移动到适当的位置
	cmd = exec.Command("mv", "node-{version}", "/usr/local/node")
	err = cmd.Run()
	if err != nil {
		return err
	}

	// 添加Node.js二进制文件路径到环境变量
	cmd = exec.Command("export", "PATH=$PATH:/usr/local/node/bin")
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

// 美团登录
func Meituan_getck(sender *Sender) {
	var png []byte
	png, _ = qrcode.Encode("https://passport.meituan.com/useraccount/ilogin?", qrcode.Medium, 256)
	sender.SendImg(png)
	sender.Reply("请微信识别或扫描二维码，登录之后，先点击微信右上角的三个小点 ... 然后在点击下面投诉旁边的复制链接，随后把链接粘贴复制发送给机器人，完成CK提交")
	return
}

func Jd_fruit_watering(sender *Sender, msg chan string, cks []JdCookie) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}

		if n == "q" {
			sender.Reply("退出登录流程")
			ckList[sender.UserID] = nil
			close(msg)
			return
		}

		num, err := strconv.Atoi(n)
		if err != nil {
			//sender.Reply(fmt.Sprintf("转换失败:%s", err))
			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
			ckList[sender.UserID] = nil
			return
		}
		regular := `^0$|^[1-9]\d*$`
		reg := regexp.MustCompile(regular)
		if reg.MatchString(n) {
			//cks := GetJdCookie(sender)
			if len(cks) <= num {
				sender.Reply("输入序列号错误，已退出！")
				ckList[sender.UserID] = nil
				return
			} else {
				if sender.IsAdmin {
					sender.Reply("开始农场浇水，预计5分钟左右请耐心等待回复~")
				} else {
					value := GetEnv("ncjs") // 变量设置美团扣的值，export ncjs 10
					if value == "" {
						sender.Reply("管理员未开启农场浇水")
						ckList[sender.UserID] = nil
						return
					} else {
						coin := GetCoin(sender.UserID)
						jbcoin, _ := strconv.Atoi(value)
						if coin < jbcoin {
							sender.Reply(fmt.Sprintf("积分不足，农场浇水需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
							ckList[sender.UserID] = nil
							return
						}
						RemCoin(sender.UserID, jbcoin)
						sender.Reply(fmt.Sprintf("开始农场浇水，预计5分钟左右请耐心等待回复，已扣除%d个积分，剩余积分%d\n提醒：农场红包每日限兑一次，每月限兑四次", jbcoin, GetCoin(sender.UserID)))
					}
				}

			}
		} else {
					sender.Reply("输入序列号错误，已退出！！！")
					meituanList[sender.UserID] = nil
					return
			}
		cks[num].Watering(sender)
		ckList[sender.UserID] = nil
	}
}







func (ck *JdCookie) Watering(sender *Sender) {
	logs.Info("开始农场浇水")

	file1, _ := vweb.JsFs.ReadFile("js/jd_fruit.js")

	// 创建一个临时文件来保存JavaScript脚本
	file, err := os.CreateTemp(ExecPath+"/scripts", "jd_fruit.js")
	if err != nil {
		fmt.Println("创建临时文件失败:", err)
		return
	}
	defer os.Remove(file.Name())

	// 将JavaScript脚本写入临时文件
	_, err = file.Write(file1)
	if err != nil {
		fmt.Println("写入临时文件失败:", err)
		return
	}

	// 执行JavaScript脚本
	cmd := exec.Command("node", file.Name())

	envs := []Env{
		{Name: "pins", Value: "&" + ck.PtPin},
		{Name: "DO_TEN_WATER_AGAIN", Value: "false"}, //#攒水滴只交10次水，默认不攒水滴 false
		{Name: "FRUIT_FAST_CARD", Value: "true"},     //#使用快速浇水卡，水多可开启
		{Name: "FRUIT_DELAY", Value: "6000"},         //#设置等待时间(毫秒)，默认请求5次接口等待60秒（60000）
	//	{Name: "DY_PROXY", Value: "http://api2.xkdaili.com/tools/XApi.ashx?apikey=XK36A9AAF3BB521C9A16&qty=1&format=txt&split=0&iv=0&sign=f54f83c5fef6c6c7f3239c6732ae3128"},         //#农场代理
	}
	for _, env := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("执行JavaScript脚本失败:", err)
		return
	}
	// 输出脚本执行结果
	logs.Info(string(output))
	//logs.Info(replexQuan_Watering(string(output), sender))
	sender.Reply(replexQuan_Watering(string(output), sender))
}

func replexQuan_Watering(info string, sender *Sender) string {
	re1 := regexp.MustCompile(`(?m)^.*(【京东账号1🆔】.+?)$`)
	re2 := regexp.MustCompile(`(?m)^.*(【水果名称】.+?)$`)
	re3 := regexp.MustCompile(`(?m)^.*(【已兑换水果】.+?)$`)
	re4 := regexp.MustCompile(`(?m)^.*(【今日共浇水】.+?)$`)
	re5 := regexp.MustCompile(`(?m)^.*(【剩余水滴】.+?)$`)
	re6 := regexp.MustCompile(`(?m)^.*(【水果进度】.+?)$`)
	re7 := regexp.MustCompile(`(?m)^.*(【预测】.+?)$`)

	matches1 := re1.FindStringSubmatch(info)
	matches2 := re2.FindStringSubmatch(info)
	matches3 := re3.FindStringSubmatch(info)
	matches4 := re4.FindStringSubmatch(info)
	matches5 := re5.FindStringSubmatch(info)
	matches6 := re6.FindStringSubmatch(info)
	matches7 := re7.FindStringSubmatch(info)

	msgs := []string{
		fmt.Sprintf("农场浇水任务已完成："),
	}

	if len(matches1) > 1 {
		replaceText := strings.Replace(matches1[1], "【京东账号1🆔】", "【京东账号】", -1)
		msgs = append(msgs, replaceText)
	}
	if len(matches2) > 1 {
		msgs = append(msgs, matches2[1])
	}
	if len(matches3) > 1 {
		msgs = append(msgs, matches3[1])
	}
	if len(matches4) > 1 {
		msgs = append(msgs, matches4[1])
	}
	if len(matches5) > 1 {
		if sender.Type == "wx" || sender.Type == "wxg" {
			replaceText := strings.Replace(matches5[1], "💧", "💧", -1)
			msgs = append(msgs, replaceText)
		} else {
			msgs = append(msgs, matches5[1])
		}
	}
	if len(matches6) > 1 {
		msgs = append(msgs, matches6[1])
	}
	if len(matches7) > 1 {
		if (sender.Type == "wx" || sender.Type == "wxg") && strings.Contains(matches7[1], "🍉") {
			replaceText := strings.Replace(matches7[1], "🍉", "[庆祝]", -1)
			msgs = append(msgs, replaceText)
		} else {
			msgs = append(msgs, matches7[1])
		}
	}
	msgs = append(msgs, "========================================\n提示：农场兑红包，限兑次数，次数可能变更，自测！\n========================================")
	return strings.Join(msgs, "\n")
}













func Jd_price(sender *Sender, msg chan string, cks []JdCookie) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}

		if n == "q" {
			sender.Reply("退出登录流程")
			ckList[sender.UserID] = nil
			close(msg)
			return
		}

		num, err := strconv.Atoi(n)
		if err != nil {
			//sender.Reply(fmt.Sprintf("转换失败:%s", err))
			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
			ckList[sender.UserID] = nil
			return
		}
		regular := `^0$|^[1-9]\d*$`
		reg := regexp.MustCompile(regular)
		if reg.MatchString(n) {
			//cks := GetJdCookie(sender)
			if len(cks) <= num {
				sender.Reply("输入序列号错误，已退出！")
				ckList[sender.UserID] = nil
				return
			} else {
				if sender.IsAdmin {
					sender.Reply("开始京东保价")
				} else {
					value := GetEnv("baojia") // 变量设置美团扣的值，export ncjs 10
					if value == "" {
						sender.Reply("管理员未开启保价功能")
						ckList[sender.UserID] = nil
						return
					} else {
						coin := GetCoin(sender.UserID)
						jbcoin, _ := strconv.Atoi(value)
						if coin < jbcoin {
							sender.Reply(fmt.Sprintf("积分不足，保价功能需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
							ckList[sender.UserID] = nil
							return
						}
						RemCoin(sender.UserID, jbcoin)
						sender.Reply(fmt.Sprintf("开始一键保价，预计1分钟左右请耐心等待回复，已扣除%d个积分，剩余积分%d\n", jbcoin, GetCoin(sender.UserID)))
					}
				}

			}
		} else {
					sender.Reply("输入序列号错误，已退出！！！")
					meituanList[sender.UserID] = nil
					return
			}
		cks[num].Price(sender)
		ckList[sender.UserID] = nil
	}
}







func (ck *JdCookie) Price(sender *Sender) {
	logs.Info("开始运行一键保价")

	file1, _ := vweb.JsFs.ReadFile("js/jd_price.js")

	// 创建一个临时文件来保存JavaScript脚本
	file, err := os.CreateTemp(ExecPath+"/scripts", "jd_price.js")
	if err != nil {
		fmt.Println("创建临时文件失败:", err)
		return
	}
	defer os.Remove(file.Name())

	// 将JavaScript脚本写入临时文件
	_, err = file.Write(file1)
	if err != nil {
		fmt.Println("写入临时文件失败:", err)
		return
	}

	// 执行JavaScript脚本
	cmd := exec.Command("node", file.Name())

	envs := []Env{
		{Name: "pins", Value: "&" + ck.PtPin},

	}
	for _, env := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("执行JavaScript脚本失败:", err)
		return
	}
	// 输出脚本执行结果
	logs.Info(string(output))
	logs.Info(replexQuan_Price(string(output), sender))
	sender.Reply(replexQuan_Price(string(output), sender))
}

func replexQuan_Price(info string, sender *Sender) string {


    re1 := regexp.MustCompile(`保价失败：([^：]+)$`)
    re2 := regexp.MustCompile(`价保成功：([^：]+)`)
    re3 := regexp.MustCompile(`没有可保价的订单 😂`)

    matches1 := re1.FindStringSubmatch(info)
    matches2 := re2.FindStringSubmatch(info)
    matches3 := re3.FindStringSubmatch(info)


    msgs := []string{
        fmt.Sprintf("保价任务已完成："),
    }


    if len(matches3) > 0 {
        msgs = append(msgs, "没有可保价的订单 😂")
    }
	if len(matches1) > 1 {  
    msgs = append(msgs, "保价失败："+matches1[1])  
	}  
	if len(matches2) > 1 {  
    msgs = append(msgs, fmt.Sprintf("价保成功，回血%s元 🤑", matches2[1]))  
	}

    return strings.Join(msgs, "\n")
}





//##评价


func Jd_AutoEval(sender *Sender, msg chan string, cks []JdCookie) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}

		if n == "q" {
			sender.Reply("退出登录流程")
			ckList[sender.UserID] = nil
			close(msg)
			return
		}

		num, err := strconv.Atoi(n)
		if err != nil {
			//sender.Reply(fmt.Sprintf("转换失败:%s", err))
			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
			ckList[sender.UserID] = nil
			return
		}
		regular := `^0$|^[1-9]\d*$`
		reg := regexp.MustCompile(regular)
		if reg.MatchString(n) {
			//cks := GetJdCookie(sender)
			if len(cks) <= num {
				sender.Reply("输入序列号错误，已退出！")
				ckList[sender.UserID] = nil
				return
			} else {
				if sender.IsAdmin {
					sender.Reply("开始京东评价")
				} else {
					value := GetEnv("pingjia") // 变量设置美团扣的值，export ncjs 10
					if value == "" {
						sender.Reply("管理员未开启保价功能")
						ckList[sender.UserID] = nil
						return
					} else {
						coin := GetCoin(sender.UserID)
						jbcoin, _ := strconv.Atoi(value)
						if coin < jbcoin {
							sender.Reply(fmt.Sprintf("积分不足，评价功能需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
							ckList[sender.UserID] = nil
							return
						}
						RemCoin(sender.UserID, jbcoin)
						sender.Reply(fmt.Sprintf("开始一键评价，请耐心等待回复，已扣除%d个积分，剩余积分%d\n", jbcoin, GetCoin(sender.UserID)))
					}
				}

			}
		} else {
					sender.Reply("输入序列号错误，已退出！！！")
					meituanList[sender.UserID] = nil
					return
			}
		cks[num].AutoEval(sender)
		ckList[sender.UserID] = nil
	}
}







func (ck *JdCookie) AutoEval(sender *Sender) {
	logs.Info("开始运行一键评价")

	file1, _ := vweb.JsFs.ReadFile("js/jd_AutoEval.js")

	// 创建一个临时文件来保存JavaScript脚本
	file, err := os.CreateTemp(ExecPath+"/scripts", "jd_AutoEval.js")
	if err != nil {
		fmt.Println("创建临时文件失败:", err)
		return
	}
	defer os.Remove(file.Name())

	// 将JavaScript脚本写入临时文件
	_, err = file.Write(file1)
	if err != nil {
		fmt.Println("写入临时文件失败:", err)
		return
	}

	// 执行JavaScript脚本
	cmd := exec.Command("node", file.Name())

	envs := []Env{
		{Name: "pins", Value: "&" + ck.PtPin},
		{Name: "ONEVAL", Value: "true"},  //##开启评价
	

	}
	for _, env := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("执行JavaScript脚本失败:", err)
		return
	}
	// 输出脚本执行结果
	logs.Info(string(output))
	logs.Info(replexQuan_AutoEval(string(output), sender))
	sender.Reply(replexQuan_AutoEval(string(output), sender))
}



func replexQuan_AutoEval(info string, sender *Sender) string {
	// 定义两个正则表达式
	re1 := regexp.MustCompile(`(?m)^.*(开始【京东账号1】.+?)$`)
	re2 := regexp.MustCompile(`当前.*?个商品`)

	// 使用正则表达式匹配 info 字符串
	matches1 := re1.FindStringSubmatch(info)
	matches2 := re2.FindStringSubmatch(info)

	// 构建消息列表
	msgs := []string{
		"评价任务已完成：",
	}

	// 如果找到第一个正则表达式的匹配结果
	if len(matches1) > 1 {
		// 替换匹配结果中的文本，并添加到消息列表
		replaceText := strings.Replace(matches1[1], "开始【京东账号1】", "【京东账号】", -1)
		msgs = append(msgs, replaceText)
	}

	// 如果找到第二个正则表达式的匹配结果
	if len(matches2) > 0 {
		// 添加第二个匹配结果到消息列表
		msgs = append(msgs, matches2[0])
	}

	// 返回拼接后的消息
	return strings.Join(msgs, "\n")
}














func (ck *MeiTuan) RunTT_xtb(sender *Sender, orderNum int) {
	logs.Info("开始领取美团小团币")

	file1, _ := vweb.JsFs.ReadFile("js/xtb.py")

	// 创建一个临时文件来保存JavaScript脚本
	file, err := os.CreateTemp(ExecPath+"/scripts", "xtb.py")
	if err != nil {
		fmt.Println("创建临时文件失败:", err)
		return
	}
	defer os.Remove(file.Name())

	// 将JavaScript脚本写入临时文件
	_, err = file.Write(file1)
	if err != nil {
		fmt.Println("写入临时文件失败:", err)
		return
	}

		// 执行JavaScript脚本
	cmd := exec.Command("python3", file.Name())
	envs := []Env{
		{Name: "bd_mttoken", Value: ck.Token + "#" + GetEnv("uuid")},
		{Name: "bd_xtbkm", Value: GetEnv("bd_xtbkm")}, // #使用GetEnv函数获取bd_xtbkm的值  
	//	{Name: "bd_dlapi", Value: GetEnv("bd_dlapi")}, // #代理池ip  
	//	{Name: "bd_isdlt", Value: GetEnv("bd_isdlt")}, // #是否开启代理
 
	}
	for _, env := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
 	if envs[1].Value == "" {  
	 sender.Reply("请群主检查脚本卡密")			//#判断卡密是否异常
	return  
	 } 
//	 if envs[2].Value == "" {  
//	 sender.Reply("请设置代理api，,格式为 export bd_dlapi xxxxxxx")			//#代理api
//	return  
//	 }   
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("执行JavaScript脚本失败:", err)
		return
	}
	// 输出脚本执行结果
	logs.Info(string(output))
	logs.Info(replexQuan_xtb(string(output), sender, orderNum))
	sender.Reply(replexQuan_xtb(string(output), sender, orderNum))
	
}

func replexQuan_xtb(info string, sender *Sender, orderNum int) string {
	re1 := regexp.MustCompile(`(?m)^.*(运行后小团币:.+?)$`)
	re2 := regexp.MustCompile(`(?m)^.*(本次获得小团币:.+?)$`)
	re3 := regexp.MustCompile(`(?m)^.*(今日团币:.+?)$`)



	matches1 := re1.FindStringSubmatch(info)
	matches2 := re2.FindStringSubmatch(info)
	matches3 := re3.FindStringSubmatch(info)

	msgs := []string{
		fmt.Sprintf("订单类型:小团币，订单编号：%d，小团币任务已完成：", orderNum),
	}

	if len(matches1) > 1 {
		msgs = append(msgs, matches1[1])
	}
	if len(matches2) > 1 {
		msgs = append(msgs, matches2[1])
	}
	if len(matches3) > 1 {
		msgs = append(msgs, matches3[1])
	}
	return strings.Join(msgs, "\n")
}

