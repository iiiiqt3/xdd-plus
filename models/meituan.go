package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/skip2/go-qrcode"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/cdle/xdd/vweb"
	"github.com/google/uuid"
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
	Auto       string  `gorm:"column:Auto;default:false" validate:"oneof=true false"`
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
	val, _ := jsonparser.GetInt(info, "error", "code")
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
			tuan.Updates(MeiTuan{UpdateAt: Date(), Available: True, Token: token, QQ: sender.UserID, WeiXin: sender.WxId})
			sender.Reply("美团账号更新成功")
		}
		return true
	}
}

func getMeiTuans() []MeiTuan {
	var ck []MeiTuan
	db.Find(&ck)
	return ck
}

func getMeiTuan(id string) (*MeiTuan, error) {
	ck := &MeiTuan{}
	return ck, db.Where("id = ?", id).First(ck).Error
}

// 获取美团账号通过用户名前缀
func GetMeiTuanByPrefix(prefix string, sender *Sender) []MeiTuan {
	switch sender.Type {
	case "qq", "qqg", "tg":
		return GetMTCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s = ? and nickname like ? ", QQ), sender.UserID, fmt.Sprintf("%s%%", prefix))
		})
	case "wx", "wxg":
		return GetMTCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s = ? and nickname like ? ", "WeiXin"), sender.WxId, fmt.Sprintf("%s%%", prefix))
		})
	default:
		return nil
	}
	return nil

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
	body, _ := io.ReadAll(res.Body)

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

// 检查是否已安装Node.js
func isNodeInstalled() bool {
	cmd := exec.Command("node", "-v")
	err := cmd.Run()
	return err == nil
}

func (ck *MeiTuan) RunTT(sender *Sender) {
	logs.Info("开始领")
	if !isNodeInstalled() {
		sender.Reply("环境缺失，请等待管理员修复")
		JdCookie{}.Push("node环境缺失,请注意")
		return
	}

	file1, _ := vweb.JsFs.ReadFile("js/meituan.js")

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
		{Name: "meituanCommonTask", Value: False},
		{Name: "meituanMrzqTask", Value: False},
		{Name: "meituanCyfTask", Value: False},
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
	if sender.WxId == "auto" {
		logs.Info("自动领卷成功")
	} else {
		sender.Reply(replexQuan(string(output), sender))
	}

}

func replexQuan(info string, sender *Sender) string {
	re := regexp.MustCompile(`账号\[\d+\].*?:\s*(.*?减\d+)`)
	matches := re.FindAllString(info, -1)
	msgs := []string{
		fmt.Sprintf("领卷完成共计领卷%d张,明细如下:", len(matches)),
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

func MeituanSelect(sender *Sender, msg chan string, typ int, meituans []MeiTuan) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}

		if n == "q" {
			meituanList[sender.UserID] = nil
			close(msg)
			return
		}

		num, err := strconv.Atoi(n)
		if err != nil {
			sender.Reply(fmt.Sprintf("转换失败:%s", err))
			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
			meituanList[sender.UserID] = nil
			return
		}
		//typ 1 美团50  2 美团领卷  3美团UUID绑定

		switch typ {
		case 1:
			//美团50
			meituans[num].RunCoin()
			meituanList[sender.UserID] = nil
		case 2:
			sender.Reply("已开始领卷")

			if meituans[num].CheckDownLine() {
				meituans[num].RunTT(sender)
			} else {
				sender.Reply("账号已失效，请重新登录")
			}
			meituanList[sender.UserID] = nil
		case 3:
			//UUID绑定
			s := sender.Contents[0]
			meituans[num].Updates(MeiTuan{UUID: s})
			sender.Reply(fmt.Sprintf("UUID绑定成功:%s", meituans[num].Nickname))
			meituanList[sender.UserID] = nil
		default:
			sender.Reply("暂无对应的渠道,已经退出流程请重新输入")
			meituanList[sender.UserID] = nil
			return
		}

	}
}

func CheckMTList() {
	tuans := getMeiTuans()
	for i, tuan := range tuans {
		if !tuan.CheckDownLine() {
			tuan.Updates(MeiTuan{
				Available: False,
				LoseAt:    Date(),
			})
			tuans[i] = tuan
			SendWxMsg(tuan.WeiXin, fmt.Sprintf("美团账号:%s已失效", tuan.Nickname))
		}
	}
}

// 美团登录
func Meituan_getck(sender *Sender) {
	var png []byte
	png, _ = qrcode.Encode("https://passport.meituan.com/useraccount/ilogin?", qrcode.Medium, 256)
	sender.SendImg(png)
	sender.Reply("请微信识别或扫描二维码，登录之后点击微信右上角的 ... 点击下面投诉旁边的复制链接发送给机器人")
}

// 获取真实链接
func Meituan_getRealUrl(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		logs.Error("获取链接失败")
		return ""
	}
	defer resp.Body.Close()
	// 获取重定向的URL
	realURL := resp.Request.URL.String()
	return realURL
}

// 获取链接中的UUID
func Meituan_getUUID(realUrl string) string {
	//提取utm_term的值
	re := regexp.MustCompile(`utm_term=(.*?)&`)
	match := re.FindStringSubmatch(realUrl)
	str := match[1]
	if strings.Contains(realUrl, "android") {
		//安卓提取UUID
		re := regexp.MustCompile(`(000.*)`)
		match := re.FindStringSubmatch(str)
		if len(match) > 1 {
			result := match[1]
			if len(result) > 3 {
				return result[:len(result)-3]
			}
		}
	} else {
		// Iphone提取UUID
		re := regexp.MustCompile(`G(.*?)2024`)
		match := re.FindStringSubmatch(str)
		if len(match) > 1 {
			return match[1]
		}
	}

	return ""
}

// 通过姓名前缀绑定UUID
func Meituan_Bind(sender *Sender, prefix string, uuid string) bool {
	//获取前缀的美团账号
	cks := GetMeiTuanByPrefix(prefix, sender)
	if len(cks) == 0 {
		sender.Reply("未找到对应的美团账号,进入手动匹配模式")
		return false
	} else if len(cks) > 1 {
		sender.Reply("匹配到多个美团账号,进入手动匹配模式")
		return false

	}
	for _, ck := range cks {
		ck.Updates(MeiTuan{UUID: uuid})
		sender.Reply(fmt.Sprintf("绑定成功:%s", ck.Nickname))
	}
	return true
}

//
//func Jd_fruit_watering(sender *Sender, msg chan string, cks []JdCookie) {
//	for {
//		n, ok := <-msg
//		//说明发送方关闭了channel
//		if !ok {
//			break
//		}
//
//		if n == "q" {
//			sender.Reply("退出登录流程")
//			ckList[sender.UserID] = nil
//			close(msg)
//			return
//		}
//
//		num, err := strconv.Atoi(n)
//		if err != nil {
//			//sender.Reply(fmt.Sprintf("转换失败:%s", err))
//			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
//			ckList[sender.UserID] = nil
//			return
//		}
//		regular := `^0$|^[1-9]\d*$`
//		reg := regexp.MustCompile(regular)
//		if reg.MatchString(n) {
//			//cks := GetJdCookie(sender)
//			if len(cks) <= num {
//				sender.Reply("输入序列号错误，已退出！")
//				ckList[sender.UserID] = nil
//				return
//			} else {
//				if sender.IsAdmin {
//					sender.Reply("开始农场浇水，预计5分钟左右请耐心等待回复~")
//				} else {
//					value := GetEnv("ncjs") // 变量设置美团扣的值，export ncjs 10
//					if value == "" {
//						sender.Reply("管理员未开启农场浇水")
//						ckList[sender.UserID] = nil
//						return
//					} else {
//						coin := GetCoin(sender.UserID)
//						jbcoin, _ := strconv.Atoi(value)
//						if coin < jbcoin {
//							sender.Reply(fmt.Sprintf("积分不足，农场浇水需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
//							ckList[sender.UserID] = nil
//							return
//						}
//						RemCoin(sender.UserID, jbcoin)
//						sender.Reply(fmt.Sprintf("开始农场浇水，预计5分钟左右请耐心等待回复，已扣除%d个积分，剩余积分%d\n提醒：农场红包每日限兑一次，每月限兑四次", jbcoin, GetCoin(sender.UserID)))
//					}
//				}
//
//			}
//		} else {
//			sender.Reply("输入序列号错误，已退出！！")
//			ckList[sender.UserID] = nil
//			return
//		}
//		cks[num].Watering(sender)
//		ckList[sender.UserID] = nil
//	}
//}
//
//func (ck *JdCookie) Watering(sender *Sender) {
//	logs.Info("开始农场浇水")
//	jsFilePath := ExecPath + "/scripts/jd_fruit.js"
//	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
//		logs.Error("JavaScript 文件不存在: %v", err)
//		return
//	}
//	cmd := exec.Command("node", jsFilePath)
//	envs := map[string]string{
//		"pins":               "&" + ck.PtPin,
//		"DO_TEN_WATER_AGAIN": "false",
//		"FRUIT_FAST_CARD":    "true",
//		"FRUIT_DELAY":        "6000",
//	}
//
//	for key, value := range envs {
//		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
//	}
//
//	// 输出环境变量以进行调试
//	for _, env := range cmd.Env {
//		logs.Info("环境变量: %s", env)
//	}
//
//	// 获取并输出脚本执行结果
//	output, err := cmd.CombinedOutput()
//	logs.Info("脚本输出: %s", string(output))
//
//	if err != nil && strings.TrimSpace(string(output)) == "" {
//		logs.Error("执行 JavaScript 脚本失败: %v", err)
//		return
//	}
//
//	sender.Reply(replexQuan_Watering(string(output), sender))
//}
//
//func replexQuan_Watering(info string, sender *Sender) string {
//	re1 := regexp.MustCompile(`(?m)^.*(【京东账号1🆔】.+?)$`)
//	re2 := regexp.MustCompile(`(?m)^.*(【水果名称】.+?)$`)
//	re3 := regexp.MustCompile(`(?m)^.*(【已兑换水果】.+?)$`)
//	re4 := regexp.MustCompile(`(?m)^.*(【今日共浇水】.+?)$`)
//	re5 := regexp.MustCompile(`(?m)^.*(【剩余水滴】.+?)$`)
//	re6 := regexp.MustCompile(`(?m)^.*(【水果进度】.+?)$`)
//	re7 := regexp.MustCompile(`(?m)^.*(【预测】.+?)$`)
//	re8 := regexp.MustCompile(`(?m)^.*(【数据异常】.+?)$`)
//
//	matches1 := re1.FindStringSubmatch(info)
//	matches2 := re2.FindStringSubmatch(info)
//	matches3 := re3.FindStringSubmatch(info)
//	matches4 := re4.FindStringSubmatch(info)
//	matches5 := re5.FindStringSubmatch(info)
//	matches6 := re6.FindStringSubmatch(info)
//	matches7 := re7.FindStringSubmatch(info)
//	matches8 := re8.FindStringSubmatch(info)
//
//	msgs := []string{
//		fmt.Sprintf("农场浇水任务已完成："),
//	}
//
//	if len(matches1) > 1 {
//		replaceText := strings.Replace(matches1[1], "【京东账号1🆔】", "【京东账号】", -1)
//		msgs = append(msgs, replaceText)
//	}
//	if len(matches2) > 1 {
//		msgs = append(msgs, matches2[1])
//	}
//	if len(matches3) > 1 {
//		msgs = append(msgs, matches3[1])
//	}
//	if len(matches4) > 1 {
//		msgs = append(msgs, matches4[1])
//	}
//	if len(matches5) > 1 {
//		if sender.Type == "wx" || sender.Type == "wxg" {
//			replaceText := strings.Replace(matches5[1], "💧", "💧", -1)
//			msgs = append(msgs, replaceText)
//		} else {
//			msgs = append(msgs, matches5[1])
//		}
//	}
//	if len(matches6) > 1 {
//		msgs = append(msgs, matches6[1])
//	}
//	if len(matches7) > 1 {
//		if (sender.Type == "wx" || sender.Type == "wxg") && strings.Contains(matches7[1], "🍉") {
//			replaceText := strings.Replace(matches7[1], "🍉", "[庆祝]", -1)
//			msgs = append(msgs, replaceText)
//		} else {
//			msgs = append(msgs, matches7[1])
//		}
//	}
//	if len(matches8) > 1 {
//		msgs = append(msgs, matches8[1])
//	}
//	msgs = append(msgs, "=================\n提示：农场兑红包，每月限兑4次数，次数可能变更，自测！\n=================")
//	return strings.Join(msgs, "\n")
//}
//
//func Jd_price(sender *Sender, msg chan string, cks []JdCookie) {
//	for {
//		n, ok := <-msg
//		//说明发送方关闭了channel
//		if !ok {
//			break
//		}
//
//		if n == "q" {
//			sender.Reply("退出登录流程")
//			ckList[sender.UserID] = nil
//			close(msg)
//			return
//		}
//
//		num, err := strconv.Atoi(n)
//		if err != nil {
//			//sender.Reply(fmt.Sprintf("转换失败:%s", err))
//			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
//			ckList[sender.UserID] = nil
//			return
//		}
//		regular := `^0$|^[1-9]\d*$`
//		reg := regexp.MustCompile(regular)
//		if reg.MatchString(n) {
//			//cks := GetJdCookie(sender)
//			if len(cks) <= num {
//				sender.Reply("输入序列号错误，已退出！")
//				ckList[sender.UserID] = nil
//				return
//			} else {
//				if sender.IsAdmin {
//					sender.Reply("开始京东保价")
//				} else {
//					value := GetEnv("baojia") // 变量设置美团扣的值，export baojia=15
//					if value == "" {
//						sender.Reply("管理员未开启保价功能")
//						ckList[sender.UserID] = nil
//						return
//					} else {
//						coin := GetCoin(sender.UserID)
//						jbcoin, _ := strconv.Atoi(value)
//						if coin < jbcoin {
//							sender.Reply(fmt.Sprintf("积分不足，保价功能需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
//							ckList[sender.UserID] = nil
//							return
//						}
//						RemCoin(sender.UserID, jbcoin)
//						sender.Reply(fmt.Sprintf("开始一键保价，预计1分钟左右请耐心等待回复，已扣除%d个积分，剩余积分%d\n", jbcoin, GetCoin(sender.UserID)))
//					}
//				}
//
//			}
//		} else {
//			sender.Reply("输入序列号错误，已退出！！！")
//			meituanList[sender.UserID] = nil
//			return
//		}
//		cks[num].Price(sender)
//		ckList[sender.UserID] = nil
//	}
//}
//
//func (ck *JdCookie) Price(sender *Sender) {
//	logs.Info("开始运行一键保价")
//
//	// 指定 JavaScript 脚本文件路径
//	jsFilePath := ExecPath + "/scripts/jd_OnceApply.js"
//
//	// 检查 JavaScript 文件是否存在
//	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
//		logs.Error("JavaScript 文件不存在: %v", err)
//		return
//	}
//
//	// 执行 JavaScript 脚本
//	cmd := exec.Command("node", jsFilePath)
//
//	envs := []Env{
//		{Name: "pins", Value: "&" + ck.PtPin},
//	}
//	for _, env := range envs {
//		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
//	}
//
//	output, err := cmd.CombinedOutput()
//	logs.Info("脚本输出: %s", string(output))
//
//	if err != nil && strings.TrimSpace(string(output)) == "" {
//		logs.Error("执行 JavaScript 脚本失败: %v", err)
//		return
//	}
//	sender.Reply(replexQuan_Price(string(output), sender))
//}
//
//func replexQuan_Price(info string, sender *Sender) string {
//
//	re1 := regexp.MustCompile(`保价失败：([^：]+)$`)
//	re2 := regexp.MustCompile(`价保成功：([^：]+)`)
//	re3 := regexp.MustCompile(`没有可保价的订单 😂`)
//
//	matches1 := re1.FindStringSubmatch(info)
//	matches2 := re2.FindStringSubmatch(info)
//	matches3 := re3.FindStringSubmatch(info)
//
//	msgs := []string{
//		fmt.Sprintf("保价任务已完成："),
//	}
//
//	if len(matches3) > 0 {
//		msgs = append(msgs, "没有可保价的订单 😂")
//	}
//	if len(matches1) > 1 {
//		msgs = append(msgs, "保价失败："+matches1[1])
//	}
//	if len(matches2) > 1 {
//		msgs = append(msgs, fmt.Sprintf("价保成功，回血%s元 🤑", matches2[1]))
//	}
//
//	return strings.Join(msgs, "\n")
//}
//
//func Jd_AutoEval(sender *Sender, msg chan string, cks []JdCookie) {
//	for {
//		n, ok := <-msg
//		if !ok {
//			break
//		}
//
//		if n == "q" {
//			sender.Reply("退出登录流程")
//			ckList[sender.UserID] = nil
//			close(msg)
//			return
//		}
//
//		num, err := strconv.Atoi(n)
//		if err != nil {
//			//sender.Reply(fmt.Sprintf("转换失败:%s", err))
//			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
//			ckList[sender.UserID] = nil
//			return
//		}
//		regular := `^0$|^[1-9]\d*$`
//		reg := regexp.MustCompile(regular)
//		if reg.MatchString(n) {
//			//cks := GetJdCookie(sender)
//			if len(cks) <= num {
//				sender.Reply("输入序列号错误，已退出！")
//				ckList[sender.UserID] = nil
//				return
//			} else {
//				if sender.IsAdmin {
//					sender.Reply("开始京东评价")
//				} else {
//					value := GetEnv("pingjia") // 变量设置美团扣的值，export pingjia=15
//					if value == "" {
//						sender.Reply("管理员未开启保价功能")
//						ckList[sender.UserID] = nil
//						return
//					} else {
//						coin := GetCoin(sender.UserID)
//						jbcoin, _ := strconv.Atoi(value)
//						if coin < jbcoin {
//							sender.Reply(fmt.Sprintf("积分不足，评价功能需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买）", jbcoin))
//							ckList[sender.UserID] = nil
//							return
//						}
//						RemCoin(sender.UserID, jbcoin)
//						sender.Reply(fmt.Sprintf("开始一键评价，请耐心等待回复，已扣除%d个积分，剩余积分%d\n", jbcoin, GetCoin(sender.UserID)))
//					}
//				}
//
//			}
//		} else {
//			sender.Reply("输入序列号错误，已退出！！！")
//			meituanList[sender.UserID] = nil
//			return
//		}
//		cks[num].AutoEval(sender)
//		ckList[sender.UserID] = nil
//	}
//}
//
//func (ck *JdCookie) AutoEval(sender *Sender) {
//	logs.Info("开始运行一键评价")
//	jsFilePath := ExecPath + "/scripts/jd_AutoEval.js"
//	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
//		logs.Error("JavaScript 文件不存在: %v", err)
//		return
//	}
//	cmd := exec.Command("node", jsFilePath)
//	envs := []Env{
//		{Name: "pins", Value: "&" + ck.PtPin},
//		{Name: "ONEVAL", Value: "true"}, //##开启评价
//
//	}
//	for _, env := range envs {
//		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", env.Name, env.Value))
//	}
//
//	output, err := cmd.CombinedOutput()
//	logs.Info("脚本输出: %s", string(output))
//
//	if err != nil && strings.TrimSpace(string(output)) == "" {
//		logs.Error("执行 JavaScript 脚本失败: %v", err)
//		return
//	}
//	sender.Reply(replexQuan_AutoEval(string(output), sender))
//}
//
//func replexQuan_AutoEval(info string, sender *Sender) string {
//	re1 := regexp.MustCompile(`(?m)^.*(开始【京东账号1】.+?)$`)
//	re2 := regexp.MustCompile(`当前.*?个商品`)
//
//	// 使用正则表达式匹配 info 字符串
//	matches1 := re1.FindStringSubmatch(info)
//	matches2 := re2.FindStringSubmatch(info)
//
//	msgs := []string{
//		"当前评价任务如下：",
//	}
//	if len(matches1) > 1 {
//		replaceText := strings.Replace(matches1[1], "开始【京东账号1】", "【京东账号】", -1)
//		msgs = append(msgs, replaceText)
//	}
//	if len(matches2) > 0 {
//		msgs = append(msgs, matches2[0])
//	}
//	msgs = append(msgs, "======评价任务已完成======")
//	return strings.Join(msgs, "\n")
//}
//
