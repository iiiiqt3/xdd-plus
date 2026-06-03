package comm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"wechatdll/Algorithm"
	"wechatdll/baseinfo"

	"github.com/astaxie/beego"
	log "github.com/sirupsen/logrus"

	//"github.com/astaxie/beego"
	"time"
	"wechatdll/Cilent/mm"
	"wechatdll/Mmtls"
	"wechatdll/models"

	"github.com/go-redis/redis"
)

// LoginDataInfo 62/16 数据登录
type LoginDataInfo struct {
	Type     byte
	UserName string
	PassWord string
	//伪密码
	NewPassWord string
	//登录数据 62/A16
	LoginData string
	Ticket    string
	NewType   int
	Language  string
}

var RedisClient *redis.Client

type LoginData struct {
	Uin                        uint32
	Wxid                       string
	Pwd                        string
	Uuid                       string
	Aeskey                     []byte
	NotifyKey                  []byte
	Deviceid_str               string
	Deviceid_byte              []byte
	DeviceType                 string
	ClientVersion              int32
	DeviceName                 string
	NickName                   string
	HeadUrl                    string
	Email                      string
	Alais                      string
	Mobile                     string
	Mmtlsip                    string
	ShortHost                  string
	LongHost                   string
	Sessionkey                 []byte
	Sessionkey_2               []byte
	Autoauthkey                []byte
	Autoauthkeylen             int32
	Clientsessionkey           []byte
	Serversessionkey           []byte
	HybridEcdhPrivkey          []byte
	HybridEcdhPubkey           []byte
	HybridEcdhInitServerPubKey []byte
	Loginecdhkey               []byte
	Cooike                     []byte
	LoginMode                  string
	Proxy                      models.ProxyInfo
	MmtlsKey                   *Mmtls.MmtlsClient
	DeviceToken                *mm.TrustResponse
	SyncKey                    []byte
	Data62                     string
	RomModel                   string
	Imei                       string
	SoftType                   string
	OsVersion                  string
	RsaPublicKey               []byte
	RsaPrivateKey              []byte
	Dns                        []Dns
	// 登录的Rsa 密钥版本
	LoginRsaVer uint32
	// 是否开启服务
	EnableService bool
	EcPublicKey   []byte `json:"ecpukey"`
	EcPrivateKey  []byte `json:"ecprkey"`
	Ticket        string
	LoginDataInfo LoginDataInfo
	// 设备信息62
	DeviceInfo *baseinfo.DeviceInfo
	//A16信息
	DeviceInfoA16 *baseinfo.AndroidDeviceInfo
	// 登录时间
	LoginDate int64
	// 刷新 tonken 时间
	RefreshTokenDate int64
}

func (u *LoginData) GetDeviceInfoA16() *baseinfo.AndroidDeviceInfo {
	if u.DeviceInfoA16 == nil {
		u.DeviceInfoA16 = &baseinfo.AndroidDeviceInfo{}
		u.DeviceInfoA16.BuildBoard = "bullhead"
	}
	return u.DeviceInfoA16

}

func (u *LoginData) GetNickName() string {
	return u.NickName
}

// GetUserName 取用户账号信息
func (u *LoginData) GetUserName() string {
	if u.Wxid == "" {
		return u.LoginDataInfo.UserName
	} else {
		return u.Wxid
	}
}

// LoginRsaVer 登录用到的RSA版本号
var LoginRsaVer = uint32(135)

var XJLoginRSAVer = uint32(133)

// DefaultLoginRsaVer 默认 登录RSA版本号
var DefaultLoginRsaVer = LoginRsaVer

// Md5OfMachOHeader wechat的MachOHeader md5值 4c541f4fca66dd93a351d4239ecaf7ae
var Md5OfMachOHeader = string("d05a80a94b6c2e3c31424403437b6e18") //

// FileHelperWXID 文件传输助手微信ID
var FileHelperWXID = string("filehelper")

// HomeDIR 当前程序的工作路径
var HomeDIR string

func (u LoginData) GetLoginRsaVer() uint32 {
	if u.LoginRsaVer == 0 {
		u.LoginRsaVer = DefaultLoginRsaVer
	}
	return u.LoginRsaVer
}

type Dns struct {
	Ip   string
	Host string
}

type DeviceTokenKey struct {
}

func GETObj(key string, i interface{}) error {
	// 清理首尾空格
	key = strings.TrimSpace(key)
	_var, _ := RedisClient.Get(key).Result()
	err := json.Unmarshal([]byte(_var), i)
	if err != nil {
		return err
	}
	return nil
}

// 自动心跳列表，每个 id 存 100 条最新的
var AutoHeartBeatList = make(map[string][]string)
var AutoHeartBeatListLock = make(chan int, 1)

func AutoHeartBeatListAdd(wxid string, content string) {
	AutoHeartBeatListLock <- 1
	defer func() {
		<-AutoHeartBeatListLock
	}()
	if _, ok := AutoHeartBeatList[wxid]; !ok {
		AutoHeartBeatList[wxid] = make([]string, 0)
	}
	AutoHeartBeatList[wxid] = append([]string{content}, AutoHeartBeatList[wxid]...)
	if len(AutoHeartBeatList[wxid]) > 100 {
		AutoHeartBeatList[wxid] = AutoHeartBeatList[wxid][:100]
	}
	SETExpirationObj("AutoHeartBeatList:"+wxid, AutoHeartBeatList[wxid], 0)
}

// 清理心跳包
func AutoHeartBeatListClear(wxid string) {
	AutoHeartBeatListLock <- 1
	defer func() {
		<-AutoHeartBeatListLock
	}()
	delete(AutoHeartBeatList, wxid)
	RedisClient.Del("AutoHeartBeatList:" + wxid)
}

// 从 redis 中初始化全部用户的心跳包， return 出去
func GetAutoHeartBeatList() map[string][]string {
	AutoHeartBeatListLock <- 1
	defer func() {
		<-AutoHeartBeatListLock
	}()
	temAutoHeartBeatList := make(map[string][]string)
	AutoHeartBeatListKeys, _ := RedisClient.Keys("AutoHeartBeatList:*").Result()
	for _, key := range AutoHeartBeatListKeys {
		AutoHeartBeatList[key] = make([]string, 0)
		item := make([]string, 0)
		GETObj(key, &item)
		temAutoHeartBeatList[key] = item
	}
	return temAutoHeartBeatList
}

func Exists(k string) bool {
	//检查是否存在key值
	exists := RedisClient.Exists(k)
	if exists.Val() > 0 {
		return true
	}
	return false
}

func SETExpirationObj(k string, i interface{}, expiration int64) error {
	// 清理首尾空格
	k = strings.TrimSpace(k)
	iData, err := json.Marshal(&i)
	if expiration > 0 {
		err = RedisClient.Set(k, string(iData), time.Duration(expiration)*time.Second).Err()
	} else {
		err = RedisClient.Set(k, string(iData), 0).Err()
	}

	if err != nil {
		//logger.Errorln(err)
		return err
	}
	return nil
}

func RedisInitialize() *redis.Client {
	dbNum, err := beego.AppConfig.Int("redisdbnum")
	if err != nil {
		log.Errorf("读取redisdbnum配置失败.")
	}
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     beego.AppConfig.String("redislink"), // redis地址
		Password: beego.AppConfig.String("redispass"), // redis密码，没有则留空
		DB:       dbNum,                               // 默认数据库，默认是0
	})

	return RedisClient
}

// 为每个 key 加锁

var groupMu sync.Mutex
var groupMap = make(map[string]*sync.Mutex)
var groupMapLock = make(chan int, 1)

func GetLoginLock(key string) *sync.Mutex {
	groupMapLock <- 1
	defer func() {
		<-groupMapLock
	}()
	// 去掉头尾空格
	key = strings.TrimSpace(key)
	if key == "" {
		key = "default"
	}
	groupMu.Lock()
	defer groupMu.Unlock()
	if _, ok := groupMap[key]; !ok {
		groupMap[key] = &sync.Mutex{}
	}
	return groupMap[key]
}

// 保存redis缓存, 如果Expiration大于0, 则有限临时缓存, 等于0持久缓存, 小于0无限临时缓存
func CreateLoginData(data *LoginData, key string, Expiration int64, temMu *sync.Mutex) error {
	// 存和取同时上锁，避免并发覆盖
	mu := GetLoginLock(key)
	if temMu != mu {
		mu.Lock()
		defer mu.Unlock()
	}
	var ExpTime time.Duration
	// Zero: 增加redis分组, 持久保存的为PERM:, 临时的保存为TEMP:, 这样做避免临时键覆盖了持久键
	prefixStr := "PERM1:"
	if key == "" {
		key = data.Uuid
	}

	if Expiration > 0 {
		ExpTime = time.Second * time.Duration(Expiration)
		prefixStr = "TEMP1:"
	} else {
		ExpTime = 0
		if Expiration < 0 {
			prefixStr = "TEMP1:"
		}
	}
	fmt.Println(prefixStr)
	JsonData, _ := json.Marshal(&data)
	err := RedisClient.Set(key, string(JsonData), ExpTime).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetKeyJsonData(Key string) (ret string, err error) {
	// 优先读取持久键值
	val, _ := RedisClient.Get("PERM1:" + Key).Result()
	if val != "" {
		return val, nil
	}
	// 兼容原版无前缀键值,次优读取
	val, _ = RedisClient.Get(Key).Result()
	if val != "" {
		return val, nil
	}
	// 读取临时键值
	val, _ = RedisClient.Get("TEMP1:" + Key).Result()
	if val == "" {
		return ret, errors.New(fmt.Sprintf("[Key:%v]数据不存在", Key))
	}
	return val, nil
}

func GetLoginata(key string, temMu *sync.Mutex) (*LoginData, error) {
	// 存和取同时上锁，避免并发覆盖
	mu := GetLoginLock(key)
	if temMu != mu {
		mu.Lock()
		defer mu.Unlock()
	}
	loginData := &LoginData{}
	P, err := GetKeyJsonData(key)
	if err == nil {
		_ = json.Unmarshal([]byte(P), loginData)
	}
	// 确保 ShortHost 和 LongHost 不为空
	if loginData.ShortHost == "" {
		loginData.ShortHost = Algorithm.MmtlsShortHost
	}
	if loginData.LongHost == "" {
		loginData.LongHost = Algorithm.MmtlsLongHost
	}
	return loginData, nil
}

func GetLoginatas(key string) (*LoginData, error) {
	P, err := GetKeyJsonData(key)
	if err != nil {
		return &LoginData{}, err
	}
	D := &LoginData{}
	err = json.Unmarshal([]byte(P), D)
	if err != nil {
		return &LoginData{}, err
	}

	return D, nil
}
func GetLoginataByDevId(key string) (*LoginData, error) {
	// 清理首尾空格
	key = strings.TrimSpace(key)
	if key == "" || key == "string" {
		return &LoginData{}, nil
	}
	// 根据 设备ID获取 userName
	userName, err := RedisClient.Get("devId:" + key).Result()
	if err != nil {
		return &LoginData{}, err
	}
	return GetLoginata(userName, nil)
}

func DelLoginata(key string) error {
	return RedisClient.Del(key).Err()
}

/*
*
设置今天抢红包的数额

	1 表示红包
	2 表示转账
*/
func SetTodayMoney(key string, fieldKey string, data float64, dataType int) error {
	prefixStr := ""
	switch dataType {
	case 1:
		{
			prefixStr = "wxhb:"
			break
		}
	case 2:
		{
			prefixStr = "wxzz:"
			break
		}

	}
	moneyKey := prefixStr + key
	// 首先获取今天的金额
	todayMoney, _ := RedisClient.HGet(moneyKey, fieldKey).Float64()
	totalMoney := todayMoney + data
	err := RedisClient.HSet(moneyKey, fieldKey, totalMoney).Err()
	if err != nil {
		return err
	}
	return nil
}

// hash写入数据
func GetTodayMoney(key string, dataType int) map[string]string {
	prefixStr := ""
	switch dataType {
	case 1:
		{
			prefixStr = "wxhb:"
			break
		}
	case 2:
		{
			prefixStr = "wxzz:"
			break
		}

	}
	moneyKey := prefixStr + key

	// 优先读取持久键值
	//cmd, _ := RedisClient.HGetAll(moneyKey)
	cmd := RedisClient.HGetAll(moneyKey)
	result, err := cmd.Result()
	if err != nil {
		return nil
	}
	return result

}

// ==================== 微信设备索引管理 ====================
// 用于维护已登录微信设备的wxid索引，供WxApiController查询设备列表

const wxLoggedInSetKey = "wx:logged_in_set"

// AddWxidToIndex 将wxid添加到已登录设备索引
func AddWxidToIndex(wxid string) {
	if wxid == "" || RedisClient == nil {
		return
	}
	RedisClient.SAdd(wxLoggedInSetKey, wxid)
}

// RemoveWxidFromIndex 从已登录设备索引中移除wxid
func RemoveWxidFromIndex(wxid string) {
	if wxid == "" || RedisClient == nil {
		return
	}
	RedisClient.SRem(wxLoggedInSetKey, wxid)
}

// GetAllLoggedInWxids 获取所有已登录的wxid列表
func GetAllLoggedInWxids() ([]string, error) {
	if RedisClient == nil {
		return nil, fmt.Errorf("Redis未初始化")
	}
	return RedisClient.SMembers(wxLoggedInSetKey).Result()
}
