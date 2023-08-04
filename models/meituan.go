package models

import (
	"fmt"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"gorm.io/gorm"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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

func CheckDownLine(cookie *MeiTuan) bool {
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
	tb.Find(&cks)
	return cks
}

func GetMeiTuan(sender *Sender) string {
	switch sender.Type {
	case "qq", "qqg":
		cks := GetMTCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s == ?  ", QQ), sender.UserID)
		})
		if len(cks) > 0 {
			for _, meituan := range cks {
				sender.Reply(meituan.Query())
			}
		} else {
			return "查无美团账号"
		}
	case "wx", "wxg":
		cks := GetMTCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s == ?  ", "WeiXin"), sender.WxId)
		})
		if len(cks) > 0 {
			for _, meituan := range cks {
				sender.Reply(meituan.Query())
			}
		} else {
			return "查无美团账号"
		}

	default:
		return "暂不匹配该渠道"
	}
	return ""
}

func (ck *MeiTuan) Query() string {
	msgs := []string{
		fmt.Sprintf("账号昵称：%s", ck.Nickname),
	}
	if ck.Note != "" {
		msgs = append(msgs, fmt.Sprintf("账号备注：%s", ck.Note))
	}

	if CheckDownLine(ck) {
		if ck.UpdateAt != "" {
			parse1, _ := time.Parse("2006-01-02", ck.UpdateAt)
			logs.Info(parse1)
			msgs = append(msgs, fmt.Sprintf("最后更新时间：%s", parse1.Format("2006-01-02")))
		}
		msgs = append(msgs, fmt.Sprintf("赚金币余额:%f", ck.CashToken))
		msgs = append(msgs, fmt.Sprintf("赚金币金币:%s", ck.CoinToken))
		msgs = append(msgs, fmt.Sprintf("赚金币金币:%s", ck.CoinToken))
	} else {
		msgs = append(msgs, []string{
			"提醒：该账号已过期，请重新登录",
		}...)
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

func LoginMeituan(meituan *MeiTuan) {

	//url := "https://game.meituan.com/mgc/gamecenter/common/mtUser/player/login?gameType=10402&mtUserId=2671745339&mtToken=AgGZIhpANpaGF3nKv4BkpMODg1umic_4eOtonxB4JIls-7qoFli6J74rcupVqKgFRnescC_xeqHvywAAAACpGQAAxmqNqbPoVXhS31jUUgGMtq9h2JkVFjcF5vtkm7c4M-ZmdVRmdyBVhO5ghs9GJZMC&mtDeviceId=00000000000009889BF295E9143E4BF8009AC0FFAAB72A169037358192590584&nonceStr=9lp09r18e2i1018e&externalStr={"cityId":"110"}"
	url := fmt.Sprintf("\"https://game.meituan.com/mgc/gamecenter/common/mtUser/player/login?gameType=10402&mtUserId=2671745339&mtToken=AgGZIhpANpaGF3nKv4BkpMODg1umic_4eOtonxB4JIls-7qoFli6J74rcupVqKgFRnescC_xeqHvywAAAACpGQAAxmqNqbPoVXhS31jUUgGMtq9h2JkVFjcF5vtkm7c4M-ZmdVRmdyBVhO5ghs9GJZMC&mtDeviceId=00000000000009889BF295E9143E4BF8009AC0FFAAB72A169037358192590584&nonceStr=9lp09r18e2i1018e&externalStr={\"cityId\":\"110\"}\"\n")
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Host", "game.meituan.com")
	req.Header.Add("X-Titans-User", "")
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("Accept-Language", "zh-CN,zh-Hans;q=0.9")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", "https://awp.meituan.com")
	req.Header.Add("mToken", "undefined")
	req.Header.Add("User-Agent", getUA())
	req.Header.Add("Referer", "https://awp.meituan.com/")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("acToken", "undefined")
	cookie := fmt.Sprintf("utm_medium=android;uuid=%s;token=%s;mt_c_token=%s;", GetUUID(), meituan.Token, meituan.Token)
	req.Header.Add("Cookie", cookie)
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Println(res)
	fmt.Println(string(body))

}

//随机固定UUID
func GetUUID() string {
	uuidArray := [11]string{
		"00000000000005A71494E8805428A93A5EBB6E00282BDA167095351831885613",
		"0000000000000E757A5DF300944AF82C239A47442EB26A168979373388967298",
		"0000000000000FE39E282D85D4184826D7A3CB6BDEE71A168979301991229841",
		"000000000000026CD052477C44C26964557ABBE7B24C7A168980721534295742",
		"0000000000000E757A5DF300944AF82C239A47442EB26A168979373388967298",
		"0000000000000FE39E282D85D4184826D7A3CB6BDEE71A168979301991229841",
		"000000000000026CD052477C44C26964557ABBE7B24C7A168980721534295742",
		"00000000000005A71494E8805428A93A5EBB6E00282BDA167095351831885613",
		"0000000000000E757A5DF300944AF82C239A47442EB26A168979373388967298",
		"0000000000000FE39E282D85D4184826D7A3CB6BDEE71A168979301991229841",
		"000000000000026CD052477C44C26964557ABBE7B24C7A168980721534295742"}
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(uuidArray))
	return uuidArray[randomIndex]
}
