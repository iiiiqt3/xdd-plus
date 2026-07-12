package models

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/beego/beego/v2/client/httplib"
	"gopkg.in/yaml.v2"
)

type WxConfig struct {
	Model   string
	Url     string
	Robotid string
	Token   string
}

type WxProtocolConfig struct {
	LoginBaseURL    string `yaml:"login_base_url"`
	NewLoginBaseURL string `yaml:"new_login_base_url"`
	ActiveProtocol  string `yaml:"active_protocol"`
	ScanLoginCost   int    `yaml:"scan_login_cost"`
	DeviceName      string `yaml:"device_name"`
	JdServer        string `yaml:"jd_server"`
}

// YybConfig 应用宝模块配置（独立协议，不影响主服务）
type YybConfig struct {
	Enabled            bool   `yaml:"enabled"`
	ResourceRoot       string `yaml:"resource_root"`
	DBFilename         string `yaml:"db_filename"`
	TCPProxy           string `yaml:"tcp_proxy"`
	ScanLoginCost      *int   `yaml:"scan_login_cost"`
	MaxAccountsPerUser int    `yaml:"max_accounts_per_user"`
	APIToken           string `yaml:"api_token"`
	ExposeInternalAPI  bool   `yaml:"expose_internal_api"`
}

// GameConfig 游戏配置（支持热更新）
type GameConfig struct {
	GameOpen              bool `yaml:"game_open"`                // 游戏总开关
	GuessNumberOpen       bool `yaml:"guess_number_open"`        // 猜数字开关
	GuessNumberCost       int  `yaml:"guess_number_cost"`        // 猜数字每次积分
	GuessNumberTimes      int  `yaml:"guess_number_times"`       // 猜数字游戏次数
	GuessNumberReward     int  `yaml:"guess_number_reward"`      // 猜数字猜中奖励
	GuessNumberMineCost   int  `yaml:"guess_number_mine_cost"`   // 猜数字踩雷扣分
	RockPaperScissorsOpen bool `yaml:"rock_paper_scissors_open"` // 猜拳开关
	RockPaperScissors     int  `yaml:"rock_paper_scissors"`      // 猜拳每次积分
	BigSmallOpen          bool `yaml:"big_small_open"`           // 比大小开关
	BigSmallCost          int  `yaml:"big_small_cost"`           // 比大小参与积分
	BigSmallPlayers       int  `yaml:"big_small_players"`        // 比大小最大人数
	BigSmallMaxGames      int  `yaml:"big_small_max_games"`      // 比大小最大游戏数
	DuelOpen              bool `yaml:"duel_open"`                // 决斗开关
	DuelDefaultBet        int  `yaml:"duel_default_bet"`         // 决斗默认下注
	DuelMaxRooms          int  `yaml:"duel_max_rooms"`           // 决斗最大房间数
}

type FanLi struct {
	Appid    string
	Appkey   string
	Union_id string
}

type Yaml struct {
	Containers            []Container
	Tasks                 []Task
	Qrcode                string
	Account               string
	Master                string
	Mode                  string
	Static                string
	Database              string
	QywxKey               string `yaml:"qywx_key"`
	Resident              string
	UserAgent             string `yaml:"user_agent"` // 已弃用：自定义UA
	Theme                 string
	TelegramBotToken      string `yaml:"telegram_bot_token"`
	TelegramUserID        int    `yaml:"telegram_user_id"`
	QQID                  int    `yaml:"qquid"`
	QQGroupID             string `yaml:"qqgid"`               // QQ群号（机器人监听交互）
	WXGroupID             string `yaml:"wxgid"`               // 微信群号（机器人监听交互）
	ActivityPushQQGroupID string `yaml:"activity_push_qqgid"` // 活动推送QQ群号
	ActivityPushWXGroupID string `yaml:"activity_push_wxgid"` // 活动推送微信群号
	DefaultPriority       int    `yaml:"default_priority"`
	InviteGroupID         string
	Autocollection        string
	NoGhproxy             bool   `yaml:"no_ghproxy"` // 已弃用：不使用代理开关
	QbotPublicMode        bool   `yaml:"qbot_public_mode"`
	DailyAssetPushCron    string `yaml:"daily_asset_push_cron"` // 已弃用：每日资产推送Cron
	Version               string `yaml:"version"`
	CTime                 string `yaml:"AtTime"`
	IsHelp                bool   `yaml:"IsHelp"` // 已弃用：助力开关
	ApiToken              string `yaml:"ApiToken"`
	Invalid               string `yaml:"Invalid"`
	Query                 string `yaml:"Query"`
	Query1                string `yaml:"Query1"`
	TGURL                 string `yaml:"TGURL"`
	CXURL                 string `yaml:"CXURL"`
	SMSAddress            string `yaml:"SMSAddress"`
	IsAddFriend           bool   `yaml:"IsAddFriend"`
	Lim                   int    `yaml:"Lim"`
	Tyt                   int    `yaml:"Tyt"`
	IFC                   bool   `yaml:"IFC"`
	Later                 int    `yaml:"Later"`
	Jdcurl                string `yaml:"Jdcurl"` // 已弃用：NVJDC地址
	Madurl                string `yaml:"Madurl"`
	GAMEOPEN              bool   `yaml:"GameOpen"` // 已弃用：游戏功能开关
	IsOldV4               bool   `yaml:"IsOldV4"`
	WsToken               int    `yaml:"WsToken"`
	Wskey                 bool   `yaml:"Wskey"` // 已弃用：Wskey转换开关
	Pzz                   int    `yaml:"Pzz"`
	Jbzl                  int    `yaml:"Jbzl"`
	OpenQQ                bool   `yaml:"OpenQQ"`
	Note                  string `yaml:"Note"`
	Rotation              bool   `yaml:"Rotation"`
	Node                  string
	Npm                   string
	Python                string
	Pip                   string
	OpenFan               bool
	NoAdmin               bool   `yaml:"no_admin"`
	QbotConfigFile        string `yaml:"qbot_config_file"`
	Repos                 []Repo
	FanLis                FanLi
	Wx                    WxConfig
	WxProtocol            WxProtocolConfig `yaml:"wx_protocol"`
	Yyb                   YybConfig        `yaml:"yyb"`
	Game                  GameConfig       `yaml:"game"`
	HttpProxyServerPort   int              `yaml:"http_proxy_server_port"`
	Priority              int              `yaml:"Priority"`
	DailyCompletePush     string           `yaml:"daily_complete_push"`
	RefreshTime           int              `yaml:"refresh_time"`
	Title                 string
	PortalPublicURL       string      `yaml:"portal_public_url"` // 门户/上传资源公网地址，用于QQ/微信群推送图片
	Jpush                 JpushConfig `yaml:"jpush"`
}

var Balance = "balance"
var Parallel = "parallel"
var Special = "special"
var Vip = "vip"
var GhProxy = "https://ghproxy.com/"
var Cdle = false

var Config Yaml

// configMutex 配置热更新互斥锁
var configMutex sync.RWMutex

func initConfig() {
	confDir := ExecPath + "/conf"
	if _, err := os.Stat(confDir); err != nil {
		os.MkdirAll(confDir, os.ModePerm)
	}
	botDir := ExecPath + "/conf"
	if _, err := os.Stat(botDir); err != nil {
		os.MkdirAll(botDir, os.ModePerm)
	}

	for _, name := range []string{"app.conf", "config.yaml", "reply.php", "title.conf"} {
		f, err := os.OpenFile(ExecPath+"/conf/"+name, os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			Warn(err)
		}
		s, _ := ioutil.ReadAll(f)
		if len(s) == 0 {
			Info("下载配置%s", name)
			r, err := httplib.Get(GhProxy + "https://raw.githubusercontent.com/764763903a/xdd-plus/main/conf/demo_" + name).Response()
			if err == nil {
				io.Copy(f, r.Body)
			}
		}
		f.Close()
	}
	//title, _ := ioutil.ReadFile(ExecPath + "/conf/title.conf")

	content, err := ioutil.ReadFile(ExecPath + "/conf/config.yaml")
	if err != nil {
		Warn("解析config.yaml读取错误: %v", err)
	}
	err = yaml.Unmarshal(content, &Config)
	if err != nil {
		Warn("解析config.yaml出错: %v", err)
	}
	if ExecPath == "/Users/cdle/Desktop/xdd" || Config.NoAdmin {
		Cdle = true
	}
	if Config.Note == "" {
		Config.Note = "pin"
	}
	if Config.Master == "" {
		Config.Master = "xxxx"
	}
	if Config.Account == "" {
		Config.Account = "admin"
	}
	if Config.CTime == "" {
		Config.CTime = "10"
	}
	if Config.Mode != Parallel && Config.Mode != Vip {
		Config.Mode = Balance
	}
	if Config.Qrcode != "" {
		Config.Theme = Config.Qrcode
	}
	if Config.NoGhproxy {
		GhProxy = ""
	}
	if Config.Tyt == 0 {
		Config.Tyt = 8
	}

	if Config.Wx.Model == "" {
		Config.Wx.Model = "my"
	}
	if Config.WxProtocol.LoginBaseURL == "" {
		Config.WxProtocol.LoginBaseURL = "http://180.152.5.230:8011"
	}
	if Config.WxProtocol.ScanLoginCost == 0 {
		Config.WxProtocol.ScanLoginCost = 2000
	}
	if Config.WxProtocol.DeviceName == "" {
		Config.WxProtocol.DeviceName = "Xiaomi-M2012K11AC"
	}
	if Config.Yyb.MaxAccountsPerUser == 0 {
		Config.Yyb.MaxAccountsPerUser = 5
	}

	initConfigDefaults()

	if Config.Database == "" {
		Config.Database = ExecPath + "/.xdd.db"
	}
	if Config.Npm == "" {
		Config.Npm = "npm"
	}
	if Config.ApiToken == "" {
		Config.ApiToken = ""
	}
	if Config.Node == "" {
		Config.Node = "node"
	}
	if Config.RefreshTime == 0 {
		Config.RefreshTime = 10
	}
	if Config.Python == "" {
		Config.Python = "python3"
	}
	if Config.Pip == "" {
		Config.Pip = "pip3"
	}
	if Config.FanLis.Appid == "" || Config.FanLis.Appkey == "" || Config.FanLis.Union_id == "" {
		Config.OpenFan = false
	} else {
		Config.OpenFan = true
	}

	// 初始化游戏配置和微信协议默认值
	initConfigDefaults()
}

// initConfigDefaults 初始化配置默认值（供initConfig和ReloadConfig共用）
func initConfigDefaults() {
	if Config.WxProtocol.LoginBaseURL == "" {
		Config.WxProtocol.LoginBaseURL = "http://180.152.5.230:8011"
	}
	if Config.WxProtocol.ActiveProtocol == "" {
		Config.WxProtocol.ActiveProtocol = "old"
	}
	if Config.WxProtocol.ScanLoginCost == 0 {
		Config.WxProtocol.ScanLoginCost = 2000
	}
	if Config.WxProtocol.DeviceName == "" {
		Config.WxProtocol.DeviceName = "Xiaomi-M2012K11AC"
	}
	// 游戏配置默认值
	if Config.Game.GuessNumberCost == 0 {
		Config.Game.GuessNumberCost = 10
	}
	if Config.Game.GuessNumberTimes == 0 {
		Config.Game.GuessNumberTimes = 10
	}
	if Config.Game.GuessNumberReward == 0 {
		Config.Game.GuessNumberReward = 300
	}
	if Config.Game.GuessNumberMineCost == 0 {
		Config.Game.GuessNumberMineCost = 100
	}
	if Config.Game.RockPaperScissors == 0 {
		Config.Game.RockPaperScissors = 50
	}
	if Config.Game.BigSmallCost == 0 {
		Config.Game.BigSmallCost = 60
	}
	if Config.Game.BigSmallPlayers == 0 {
		Config.Game.BigSmallPlayers = 6
	}
	if Config.Game.BigSmallMaxGames == 0 {
		Config.Game.BigSmallMaxGames = 5
	}
	if Config.Game.DuelDefaultBet == 0 {
		Config.Game.DuelDefaultBet = 50
	}
	if Config.Game.DuelMaxRooms == 0 {
		Config.Game.DuelMaxRooms = 10
	}
}

// configReloadHooks 配置热更新后的回调（避免 models ↔ 业务模块循环依赖）
var configReloadHooks []func()
var configReloadHooksMu sync.Mutex

// RegisterConfigReloadHook 注册配置热更新回调
func RegisterConfigReloadHook(fn func()) {
	if fn == nil {
		return
	}
	configReloadHooksMu.Lock()
	configReloadHooks = append(configReloadHooks, fn)
	configReloadHooksMu.Unlock()
}

func runConfigReloadHooks() {
	configReloadHooksMu.Lock()
	hooks := append([]func(){}, configReloadHooks...)
	configReloadHooksMu.Unlock()
	for _, fn := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					Warn("配置热更新回调异常: %v", r)
				}
			}()
			fn()
		}()
	}
}

// ReloadConfig 热更新配置（无需重启）
func ReloadConfig() error {
	content, err := ioutil.ReadFile(ExecPath + "/conf/config.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var newConfig Yaml
	err = yaml.Unmarshal(content, &newConfig)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	configMutex.Lock()
	Config = newConfig
	initConfigDefaults()
	if envWxgid := GetEnv("WxGroupID"); envWxgid != "" {
		Config.WXGroupID = envWxgid
	}
	configMutex.Unlock()

	runConfigReloadHooks()

	Info("配置已热更新成功")
	return nil
}
