package models

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var sysConfig SystemConfig

type SystemConfig struct {
	ImageUserName    string `json:"ImageUserName"`
	ImagePassword    string `json:"ImagePassword"`
	ImageToken       string `json:"ImageToken"`
	RabbitUrl        string `json:"RabbitUrl"`
	RabbitApiToken   string `json:"RabbitApiToken"`
	RabbitToken      string `json:"RabbitToken"`
	NolanUrl         string `json:"NolanUrl"`
	NolanToken       string `json:"NolanToken"`
	BBKWxUrl         string `json:"BBKWxUrl"`
	BBKJdUrl         string `json:"BBKJdUrl"`
	BBKToken         string `json:"BBKToken"`
	Rabbit           string `json:"Rabbit"`
	RabbitName       string `json:"RabbitName"`
	RabbitNumber     string `json:"RabbitNumber"`
	RabbitSms        string `json:"RabbitSms"`
	RabbitSmsNumber  string `json:"RabbitSmsNumber"`
	FastRabbit       string `json:"FastRabbit"`
	FastRabbitName   string `json:"FastRabbitName"`
	FastRabbitNumber string `json:"FastRabbitNumber"`
	MadRabbit        string `json:"MadRabbit"`
	MadRabbitName    string `json:"MadRabbitName"`
	MadRabbitNumber  string `json:"MadRabbitNumber"`
	Pro              string `json:"Pro"`
	ProName          string `json:"ProName"`
	ProNumber        string `json:"ProNumber"`
	ProSms           string `json:"ProSms"`
	ProSmsName       string `json:"ProSmsName"`
	ProSmsNumber     string `json:"ProSmsNumber"`
	BBKJd            string `json:"BBKJd"`
	BBKJdName        string `json:"BBKJdName"`
	BBKJdNumber      string `json:"BBKJdNumber"`
	BBKWx            string `json:"BBKWx"`
	BBKWxName        string `json:"BBKWxName"`
	BBKWxNumber      string `json:"BBKWxNumber"`
	// ProxyUrl 已弃用；全局动态代理统一使用 JdTaskProxy*（京东查询/任务、门户订阅回退、酷我等共用）
	ProxyUrl string `json:"ProxyUrl"`
	// JdTaskProxy*：系统「全局代理」——开关开且 URL 非空时生效；应用宝 51 / 微信协议不读此项
	JdTaskProxyEnabled     bool   `json:"JdTaskProxyEnabled"`
	JdTaskProxyUrl         string `json:"JdTaskProxyUrl"`
	JdTaskProxyRenum       string `json:"JdTaskProxyRenum"`
	JdTaskProxyRedelay     string `json:"JdTaskProxyRedelay"`
	JdTaskProxyMonthlyCoin int    `json:"JdTaskProxyMonthlyCoin"`
	// PortalMinCoinForAccess 门户活动中心/通知中心可见所需最低积分（nil 未配置时默认 1000；0 表示不限制）
	PortalMinCoinForAccess *int `json:"PortalMinCoinForAccess"`
}

func initSysConfig() {
	ListConfig()
	updateUsers()
}

func tempToken() {
	u := uuid.New()
	s := u.String()
	SaveCache("AdminToken", s)
	Info("您的临时Token为:" + s)
}

func ListConfig() SystemConfig {
	env := GetEnv("sysconfig")
	var config SystemConfig
	if env != "" {
		err := json.Unmarshal([]byte(env), &config)
		if err != nil {
			fmt.Println("解析失败:", err)
		} else {
			sysConfig = config
		}

	} else {
		Info("缺少系统配置")
	}

	return config
}

func SaveSysConfig(config SystemConfig) string {
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		fmt.Println("转换失败:", err)
		return "转换失败"
	}

	jsonStr := string(jsonBytes)
	env1 := &Env{
		Name:  "sysconfig",
		Value: jsonStr,
	}
	if err := ExportEnv(env1); err != nil {
		return "保存失败: " + err.Error()
	}
	ListConfig()
	return "保存成功"
}

func updateUsers() {
	env := GetEnv("14.6")
	if env == "" {
		Info("开始更新")

		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s >= ? and %s = ? and %s != ?", Priority, Available, "WeiXin"), 0, True, "")
		})
		for _, ck := range cks {
			if ck.WeiXin == "" {
				//根据userid查找wxid
				var user User
				tx := db.Where("number = ?", ck.QQ).First(&user)
				if tx.Error != nil {
					Info("未找到用户")
					continue
				}
				ck.Update("WxId", user.Wxid)
			}
		}

		deleteDuplicateWxidUsers()
		env := &Env{}
		env.Name = "14.6"
		env.Value = "true"
		_ = ExportEnv(env)

		JdCookie{}.Push("升级成功，已完成用户结构改造")
	}
}