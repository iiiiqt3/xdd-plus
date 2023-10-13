package models

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/core/logs"
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
	ProxyUrl         string `json:"ProxyUrl"`
}

func initSysConfig() {
	ListConfig()
	updateConfig()
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
		logs.Info("缺少系统配置")
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
	ExportEnv(env1)
	ListConfig()
	return "保存成功"
}

func updateConfig() {
	env := GetEnv("11.6")
	if env == "" {
		logs.Info("开始更新")

		var sys SystemConfig

		sys.RabbitUrl = GetEnv("RabbitUrl")
		sys.RabbitApiToken = GetEnv("RabbitApiToken")
		sys.RabbitToken = GetEnv("RabbitToken")

		sys.NolanUrl = GetEnv("NolanUrl")
		sys.NolanToken = GetEnv("NolanToken")

		sys.BBKToken = GetEnv("BBKToken")
		sys.BBKJdUrl = GetEnv("BBKJdUrl")
		sys.ProxyUrl = GetEnv("proxy")
		SaveSysConfig(sys)

		env := &Env{}
		env.Name = "11.6"
		env.Value = "true"
		ExportEnv(env)

		//UnExportEnv(&Env{Name: "RabbitUrl"})
		//UnExportEnv(&Env{Name: "RabbitApiToken"})
		//UnExportEnv(&Env{Name: "RabbitToken"})
		//UnExportEnv(&Env{Name: "NolanUrl"})
		//UnExportEnv(&Env{Name: "NolanToken"})
		//UnExportEnv(&Env{Name: "BBKToken"})
		//UnExportEnv(&Env{Name: "BBKJdUrl"})

		//todo UserId转换为WxId

		JdCookie{}.Push("升级成功，已将短信相关配置转移，后续请使用网页端配置，请及时打开网页配置登录渠道")
	}
}

func updateUsers() {
	env := GetEnv("13.1")
	if env == "" {
		logs.Info("开始更新")
		//todo
		JdCookie{}.Push("正在进行用户结构改造，请勿关闭程序")
		sql := "要执行的SQL"
		// 在Exec方法中在sql 后面可以使用多个参数作为占位的补充
		// 例如需要name=?，则写法可以使用util.Db.Exec(sql,"张三").Error
		err := db.Exec(sql).Error
		if err != nil {
			logs.Info(err.Error())
		}

		JdCookie{}.Push("升级成功，已将短信相关配置转移，后续请使用网页端配置，请及时打开网页配置登录渠道")
	}
}
