package Login

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/baseinfo"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/comm"

	"github.com/gogf/guuid"
)

/**
 * Ipad 登录初始化数据
 */

func GenIpadLoginData(request DataLogin) *comm.LoginData {
	deviceId := request.DeviceId
	if deviceId == "" || deviceId == "string" {
		deviceId = baseutils.CreateDeviceId("")
	}
	if deviceId[:2] != "49" {
		deviceId = "49" + deviceId[2:]
	}

	ClientVersion := request.ClientVersion
	if !(ClientVersion > 0) {
		ClientVersion = Algorithm.IPadVersion
	}
	deviceIdByte, _ := hex.DecodeString(deviceId)
	if request.DeviceName == "" || request.DeviceName == "string" {
		request.DeviceName = Algorithm.IPadDeviceName
	}
	D := &comm.LoginData{
		Wxid:          request.UserName,
		Pwd:           request.Password,
		Aeskey:        []byte(baseutils.RandSeq(16)), //随机密钥
		Deviceid_str:  deviceId,
		Deviceid_byte: deviceIdByte,
		DeviceType:    Algorithm.IPadDeviceType, // ipad ios17.0.0
		//DeviceType:    Algorithm.IPadDeviceType, // ipad ios17.0.0
		ClientVersion: int32(ClientVersion), //  0x1800000
		DeviceName:    request.DeviceName,   // iPad
		ShortHost:     Algorithm.MmtlsShortHost,
		LongHost:      Algorithm.MmtlsLongHost,
		Proxy:         request.Proxy,
		RomModel:      Algorithm.IPadModel,
		Imei:          baseinfo.IOSImei(deviceId),
		OsVersion:     Algorithm.IPadOsVersion,
	}
	D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	D.DeviceInfo = createDeviceInfo(D)
	return D
}

func UpdateIpadLoginData(D *comm.LoginData, Data DataLogin) *comm.LoginData {
	if D.DeviceType == "" {
		D.DeviceType = Algorithm.IPadDeviceType
	}
	if D.ClientVersion == 0 {
		D.ClientVersion = int32(Algorithm.IPadVersion)
	}
	if D.DeviceName == "" || D.DeviceName == "string" {
		if Data.DeviceName == "" || Data.DeviceName == "string" {
			D.DeviceName = Algorithm.IPadDeviceName
		} else {
			D.DeviceName = Data.DeviceName
		}
	}
	if D.ShortHost == "" {
		D.ShortHost = Algorithm.MmtlsShortHost
	}
	if D.LongHost == "" {
		D.LongHost = Algorithm.MmtlsLongHost
	}
	if D.RomModel == "" {
		D.RomModel = Algorithm.IPadModel
	}
	if D.Imei == "" {
		D.Imei = baseinfo.IOSImei(D.Deviceid_str)
	}
	if D.SoftType == "" {
		D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, Algorithm.IPadOsVersion, Algorithm.IPadModel)
	}
	if D.OsVersion == "" {
		D.OsVersion = Algorithm.IPadOsVersion
	}
	if D.DeviceInfo == nil {
		D.DeviceInfo = createDeviceInfo(D)
	}
	return D
}

/**
 * Iphone 登录初始化数据
 */

func GeniPhoneLoginData(request DataLogin) *comm.LoginData {
	deviceId := request.DeviceId
	if request.Data62 != "" && request.Data62 != "string" {
		deviceId = baseutils.Get62Key(request.Data62)
	}
	if deviceId == "" || deviceId == "string" {
		deviceId = baseutils.CreateDeviceId("")
	}
	ClientVersion := request.ClientVersion
	if !(ClientVersion > 0) {
		ClientVersion = Algorithm.IPhoneVersion
	}

	deviceIdByte, _ := hex.DecodeString(deviceId)
	if request.DeviceName == "" || request.DeviceName == "string" {
		request.DeviceName = Algorithm.IPhoneDeviceName
	}
	D := &comm.LoginData{
		Wxid:          request.UserName,
		Pwd:           request.Password,
		Aeskey:        []byte(baseutils.RandSeq(16)), //随机密钥
		Deviceid_str:  deviceId,
		Deviceid_byte: deviceIdByte,
		DeviceType:    Algorithm.IPhoneDeviceType,
		ClientVersion: int32(ClientVersion),
		DeviceName:    request.DeviceName,
		ShortHost:     Algorithm.MmtlsShortHost,
		LongHost:      Algorithm.MmtlsLongHost,
		Proxy:         request.Proxy,
		RomModel:      Algorithm.IPhoneModel,
		Imei:          baseinfo.IOSImei(deviceId),
		SoftType:      baseinfo.SoftType_iPhone(deviceId, Algorithm.IPhoneOsVersion, Algorithm.IPhoneModel),
		OsVersion:     Algorithm.IPhoneOsVersion,
	}
	D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	D.DeviceInfo = createDeviceInfo(D)
	return D
}

func UpdateiPhoneLoginData(D *comm.LoginData, Data DataLogin) *comm.LoginData {
	D.Pwd = Data.Password
	D.Data62 = Data.Data62
	if D.Data62 != "" && D.Data62 != "string" {
		D.Deviceid_str = baseutils.Get62Key(D.Data62)
		D.Deviceid_byte, _ = hex.DecodeString(D.Deviceid_str)
	}
	if Data.UserName != "" && Data.UserName != "string" {
		D.LoginDataInfo.UserName = Data.UserName
	}
	if Data.Password != "" && Data.Password != "string" {
		D.LoginDataInfo.PassWord = Data.Password
	}
	if D.DeviceType == "" {
		D.DeviceType = Algorithm.IPhoneDeviceType
	}
	if D.ClientVersion == 0 {
		D.ClientVersion = int32(Algorithm.IPhoneVersion)
	}
	if D.DeviceName == "" || D.DeviceName == "string" {
		if Data.DeviceName == "" || Data.DeviceName == "string" {
			D.DeviceName = Algorithm.IPhoneDeviceName
		} else {
			D.DeviceName = Data.DeviceName
		}
	}
	if D.ShortHost == "" {
		D.ShortHost = Algorithm.MmtlsShortHost
	}
	if D.LongHost == "" {
		D.LongHost = Algorithm.MmtlsLongHost
	}
	if D.RomModel == "" {
		D.RomModel = Algorithm.IPhoneModel
	}
	if D.Imei == "" {
		D.Imei = baseinfo.IOSImei(D.Deviceid_str)
	}
	if D.SoftType == "" {
		D.SoftType = baseinfo.SoftType_iPhone(D.Deviceid_str, Algorithm.IPhoneOsVersion, Algorithm.IPhoneModel)
	}
	if D.OsVersion == "" {
		D.OsVersion = Algorithm.IPhoneOsVersion
	}
	if D.DeviceInfo == nil {
		D.DeviceInfo = createDeviceInfo(D)
	}
	return D
}

/**
 * AndroidPad 登录初始化数据
 */
func GenAndroidPadLoginData(request DataLogin) *comm.LoginData {
	deviceId := request.DeviceId
	if deviceId == "" || deviceId == "string" {
		deviceId = baseutils.CreateDeviceId("")
	}
	ClientVersion := request.ClientVersion
	if !(ClientVersion > 0) {
		ClientVersion = Algorithm.AndroidPadVersion
	}

	deviceIdByte, _ := hex.DecodeString(deviceId)
	if request.DeviceName == "" || request.DeviceName == "string" {
		request.DeviceName = Algorithm.AndroidPadDeviceName
	}
	D := &comm.LoginData{
		Wxid:          request.UserName,
		Pwd:           request.Password,
		Aeskey:        []byte(baseutils.RandSeq(16)), //随机密钥
		Deviceid_str:  deviceId,
		Deviceid_byte: deviceIdByte,
		DeviceType:    Algorithm.AndroidPadDeviceType,
		ClientVersion: int32(ClientVersion),
		DeviceName:    request.DeviceName,
		ShortHost:     Algorithm.MmtlsShortHost,
		LongHost:      Algorithm.MmtlsLongHost,
		Proxy:         request.Proxy,
		RomModel:      Algorithm.AndroidPadModel,
		Imei:          baseinfo.IOSImei(deviceId),
		OsVersion:     Algorithm.AndroidPadOsVersion,
	}
	D.DeviceInfoA16 = createDeviceInfoA16()
	return D
}

func UpdateAndroidPadLoginData(D *comm.LoginData, Data DataLogin) *comm.LoginData {
	if D.DeviceType == "" {
		D.DeviceType = Algorithm.AndroidPadDeviceType
	}
	if D.ClientVersion == 0 {
		D.ClientVersion = int32(Algorithm.AndroidPadVersion)
	}
	if D.DeviceName == "" || D.DeviceName == "string" {
		if Data.DeviceName == "" || Data.DeviceName == "string" {
			D.DeviceName = Algorithm.AndroidPadDeviceName
		} else {
			D.DeviceName = Data.DeviceName
		}
	}
	if D.ShortHost == "" {
		D.ShortHost = Algorithm.MmtlsShortHost
	}
	if D.LongHost == "" {
		D.LongHost = Algorithm.MmtlsLongHost
	}
	if D.RomModel == "" {
		D.RomModel = Algorithm.AndroidPadModel
	}
	if D.Imei == "" {
		D.Imei = baseinfo.IOSImei(D.Deviceid_str)
	}
	if D.OsVersion == "" {
		D.OsVersion = Algorithm.AndroidPadOsVersion
	}
	D.DeviceInfo = nil
	if D.DeviceInfoA16 == nil {
		D.DeviceInfoA16 = createDeviceInfoA16()
	}
	return D
}

/**
 * Android 登录初始化数据
 */

func GenAndroidLoginData(request DataLogin) *comm.LoginData {
	//deviceId := request.A16
	deviceId := request.A16[:15]
	ClientVersion := request.ClientVersion
	if !(ClientVersion > 0) {
		ClientVersion = Algorithm.AndroidVersion
	}

	deviceIdByte := []byte(deviceId)
	if request.DeviceName == "" || request.DeviceName == "string" {
		request.DeviceName = Algorithm.AndroidDeviceName
	}
	D := &comm.LoginData{
		Wxid:          request.UserName,
		Pwd:           request.Password,
		Aeskey:        []byte(baseutils.RandSeq(16)), //随机密钥
		Deviceid_str:  deviceId,
		Deviceid_byte: deviceIdByte,
		DeviceType:    Algorithm.AndroidDeviceType,
		ClientVersion: int32(ClientVersion),
		DeviceName:    request.DeviceName,
		ShortHost:     Algorithm.MmtlsShortHost,
		LongHost:      Algorithm.MmtlsLongHost,
		Proxy:         request.Proxy,
		RomModel:      Algorithm.AndroidModel,
		Imei:          baseinfo.IOSImei(deviceId),
		OsVersion:     Algorithm.AndroidOsVersion,
	}
	D.DeviceInfoA16 = createDeviceInfoA16()
	return D
}

func UpdateAndroidLoginData(D *comm.LoginData, Data DataLogin) *comm.LoginData {
	D.Pwd = Data.Password
	if D.DeviceType == "" {
		D.DeviceType = Algorithm.AndroidDeviceType
	}
	if Data.A16 != "" && Data.A16 != "string" {
		D.Deviceid_str = Data.A16
		D.Deviceid_byte = []byte(Data.A16)
	}
	if Data.UserName != "" && Data.UserName != "string" {
		D.LoginDataInfo.UserName = Data.UserName
	}
	if Data.Password != "" && Data.Password != "string" {
		D.LoginDataInfo.PassWord = Data.Password
	}
	if D.ClientVersion == 0 {
		D.ClientVersion = int32(Algorithm.AndroidVersion)
	}
	if D.DeviceName == "" || D.DeviceName == "string" {
		if Data.DeviceName == "" || Data.DeviceName == "string" {
			D.DeviceName = Algorithm.AndroidDeviceName
		} else {
			D.DeviceName = Data.DeviceName
		}
	}
	if D.ShortHost == "" {
		D.ShortHost = Algorithm.MmtlsShortHost
	}
	if D.LongHost == "" {
		D.LongHost = Algorithm.MmtlsLongHost
	}
	if D.RomModel == "" {
		D.RomModel = Algorithm.AndroidModel
	}
	if D.Imei == "" {
		D.Imei = baseinfo.IOSImei(D.Deviceid_str)
	}
	if D.OsVersion == "" {
		D.OsVersion = Algorithm.AndroidOsVersion
	}
	if D.DeviceInfoA16 == nil {
		D.DeviceInfoA16 = createDeviceInfoA16()
	}
	D.DeviceInfo = nil
	return D
}

/**
 * Windows 登录初始化数据
 */

func GenWinLoginData(request DataLogin) *comm.LoginData {
	deviceId := request.DeviceId
	if deviceId == "" || deviceId == "string" {
		deviceId = baseutils.CreateDeviceId("")
	}
	ClientVersion := request.ClientVersion
	if !(ClientVersion > 0) {
		ClientVersion = Algorithm.WinVersion
	}

	deviceIdByte, _ := hex.DecodeString(deviceId)
	if request.DeviceName == "" || request.DeviceName == "string" {
		request.DeviceName = Algorithm.WinDeviceName
	}
	D := &comm.LoginData{
		Wxid:          request.UserName,
		Pwd:           request.Password,
		Aeskey:        []byte(baseutils.RandSeq(16)), //随机密钥
		Deviceid_str:  deviceId,
		Deviceid_byte: deviceIdByte,
		DeviceType:    Algorithm.WinDeviceType,
		ClientVersion: int32(ClientVersion),
		DeviceName:    request.DeviceName,
		ShortHost:     Algorithm.MmtlsShortHost,
		LongHost:      Algorithm.MmtlsLongHost,
		Proxy:         request.Proxy,
		RomModel:      Algorithm.WinModel,
		Imei:          baseinfo.IOSImei(deviceId),
		OsVersion:     Algorithm.WinOsVersion,
	}
	D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	D.DeviceInfo = createDeviceInfo(D)
	return D
}

func UpdateWinLoginData(D *comm.LoginData, Data DataLogin) *comm.LoginData {
	if D.DeviceType == "" {
		D.DeviceType = Algorithm.WinDeviceType
	}
	if D.ClientVersion == 0 {
		D.ClientVersion = int32(Algorithm.WinVersion)
	}
	if D.DeviceName == "" || D.DeviceName == "string" {
		if Data.DeviceName == "" || Data.DeviceName == "string" {
			D.DeviceName = Algorithm.WinDeviceName
		} else {
			D.DeviceName = Data.DeviceName
		}
	}
	if D.ShortHost == "" {
		D.ShortHost = Algorithm.MmtlsShortHost
	}
	if D.LongHost == "" {
		D.LongHost = Algorithm.MmtlsLongHost
	}
	if D.RomModel == "" {
		D.RomModel = Algorithm.WinModel
	}
	if D.Imei == "" {
		D.Imei = baseinfo.IOSImei(D.Deviceid_str)
	}
	if D.SoftType == "" {
		D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	}
	if D.OsVersion == "" {
		D.OsVersion = Algorithm.WinOsVersion
	}
	if D.DeviceInfo == nil {
		D.DeviceInfo = createDeviceInfo(D)
	}
	D.DeviceInfoA16 = nil
	return D
}

/**
 *  Winunified 登录初始化数据
 */
func GenWinUnifiedLoginData(request DataLogin) *comm.LoginData {
	deviceId := request.DeviceId
	if deviceId == "" || deviceId == "string" {
		deviceId = baseutils.CreateDeviceId("")
	}
	ClientVersion := request.ClientVersion
	if !(ClientVersion > 0) {
		ClientVersion = Algorithm.WinUnifiedVersion
	}

	deviceIdByte, _ := hex.DecodeString(deviceId)
	if request.DeviceName == "" || request.DeviceName == "string" {
		request.DeviceName = Algorithm.WinUnifiedDeviceName
	}
	D := &comm.LoginData{
		Wxid:          request.UserName,
		Pwd:           request.Password,
		Aeskey:        []byte(baseutils.RandSeq(16)), //随机密钥
		Deviceid_str:  deviceId,
		Deviceid_byte: deviceIdByte,
		DeviceType:    Algorithm.WinUnifiedDeviceType,
		ClientVersion: int32(ClientVersion),
		DeviceName:    request.DeviceName,
		ShortHost:     Algorithm.MmtlsShortHost,
		LongHost:      Algorithm.MmtlsLongHost,
		Proxy:         request.Proxy,
		RomModel:      Algorithm.WinUnifiedModel,
		Imei:          baseinfo.IOSImei(deviceId),
		OsVersion:     Algorithm.WinUnifiedOsVersion,
	}
	D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	D.DeviceInfo = createDeviceInfo(D)
	return D
}

func UpdateWinUnifiedLoginData(D *comm.LoginData, Data DataLogin) *comm.LoginData {
	if D.DeviceType == "" {
		D.DeviceType = Algorithm.WinUnifiedDeviceType
	}
	if D.ClientVersion == 0 {
		D.ClientVersion = int32(Algorithm.WinUnifiedVersion)
	}
	if D.DeviceName == "" || D.DeviceName == "string" {
		if Data.DeviceName == "" || Data.DeviceName == "string" {
			D.DeviceName = Algorithm.WinUnifiedDeviceName
		} else {
			D.DeviceName = Data.DeviceName
		}
	}
	if D.ShortHost == "" {
		D.ShortHost = Algorithm.MmtlsShortHost
	}
	if D.LongHost == "" {
		D.LongHost = Algorithm.MmtlsLongHost
	}
	if D.RomModel == "" {
		D.RomModel = Algorithm.WinUnifiedModel
	}
	if D.Imei == "" {
		D.Imei = baseinfo.IOSImei(D.Deviceid_str)
	}
	if D.SoftType == "" {
		D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	}
	if D.OsVersion == "" {
		D.OsVersion = Algorithm.WinUnifiedOsVersion
	}
	if D.DeviceInfo == nil {
		D.DeviceInfo = createDeviceInfo(D)
	}
	D.DeviceInfoA16 = nil
	return D
}

/**
 *  CarDevice 登录初始化数据
 */

func GenCarLoginData(request DataLogin) *comm.LoginData {
	deviceId := request.DeviceId
	if deviceId == "" || deviceId == "string" {
		deviceId = baseutils.CreateDeviceId("")
	}
	ClientVersion := request.ClientVersion
	if !(ClientVersion > 0) {
		ClientVersion = Algorithm.CarVersion
	}

	deviceIdByte, _ := hex.DecodeString(deviceId)
	if request.DeviceName == "" || request.DeviceName == "string" {
		request.DeviceName = Algorithm.CarDeviceName
	}
	D := &comm.LoginData{
		Wxid:          request.UserName,
		Pwd:           request.Password,
		Aeskey:        []byte(baseutils.RandSeq(16)), //随机密钥
		Deviceid_str:  deviceId,
		Deviceid_byte: deviceIdByte,
		DeviceType:    Algorithm.CarDeviceType,
		ClientVersion: int32(ClientVersion),
		DeviceName:    request.DeviceName,
		ShortHost:     Algorithm.MmtlsShortHost,
		LongHost:      Algorithm.MmtlsLongHost,
		Proxy:         request.Proxy,
		RomModel:      Algorithm.CarModel,
		Imei:          baseinfo.IOSImei(deviceId),
		OsVersion:     Algorithm.CarOsVersion,
	}
	D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	D.DeviceInfo = createDeviceInfo(D)
	return D
}

func UpdateCarLoginData(D *comm.LoginData, Data DataLogin) *comm.LoginData {
	if D.DeviceType == "" {
		D.DeviceType = Algorithm.CarDeviceType
	}
	if D.ClientVersion == 0 {
		D.ClientVersion = int32(Algorithm.CarVersion)
	}
	if D.DeviceName == "" || D.DeviceName == "string" {
		if Data.DeviceName == "" || Data.DeviceName == "string" {
			D.DeviceName = Algorithm.CarDeviceName
		} else {
			D.DeviceName = Data.DeviceName
		}
	}
	if D.ShortHost == "" {
		D.ShortHost = Algorithm.MmtlsShortHost
	}
	if D.LongHost == "" {
		D.LongHost = Algorithm.MmtlsLongHost
	}
	if D.RomModel == "" {
		D.RomModel = Algorithm.CarModel
	}
	if D.Imei == "" {
		D.Imei = baseinfo.IOSImei(D.Deviceid_str)
	}
	if D.SoftType == "" {
		D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	}
	if D.OsVersion == "" {
		D.OsVersion = Algorithm.CarOsVersion
	}
	if D.DeviceInfo == nil {
		D.DeviceInfo = createDeviceInfo(D)
	}
	D.DeviceInfoA16 = nil
	return D
}

/**
* Mac 设备
 */
func GenMacLoginData(request DataLogin) *comm.LoginData {
	deviceId := request.DeviceId
	if deviceId == "" || deviceId == "string" {
		deviceId = baseutils.CreateDeviceId("")
	}
	ClientVersion := request.ClientVersion
	if !(ClientVersion > 0) {
		ClientVersion = Algorithm.MacVersion
	}

	deviceIdByte, _ := hex.DecodeString(deviceId)
	if request.DeviceName == "" || request.DeviceName == "string" {
		request.DeviceName = Algorithm.MacDeviceName
	}
	D := &comm.LoginData{
		Wxid:          request.UserName,
		Pwd:           request.Password,
		Aeskey:        []byte(baseutils.RandSeq(16)), //随机密钥
		Deviceid_str:  deviceId,
		Deviceid_byte: deviceIdByte,
		DeviceType:    Algorithm.MacDeviceType,
		ClientVersion: int32(ClientVersion),
		DeviceName:    request.DeviceName,
		ShortHost:     Algorithm.MmtlsShortHost,
		LongHost:      Algorithm.MmtlsLongHost,
		Proxy:         request.Proxy,
		RomModel:      Algorithm.MacModel,
		Imei:          baseinfo.IOSImei(deviceId),
		OsVersion:     Algorithm.MacOsVersion,
	}
	D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	D.DeviceInfo = createDeviceInfo(D)
	return D
}

func UpdateMacLoginData(D *comm.LoginData, Data DataLogin) *comm.LoginData {
	if D.DeviceType == "" {
		D.DeviceType = Algorithm.MacDeviceType
	}
	if D.ClientVersion == 0 {
		D.ClientVersion = int32(Algorithm.MacVersion)
	}
	if D.DeviceName == "" || D.DeviceName == "string" {
		if Data.DeviceName == "" || Data.DeviceName == "string" {
			D.DeviceName = Algorithm.MacDeviceName
		} else {
			D.DeviceName = Data.DeviceName
		}
	}
	if D.ShortHost == "" {
		D.ShortHost = Algorithm.MmtlsShortHost
	}
	if D.LongHost == "" {
		D.LongHost = Algorithm.MmtlsLongHost
	}
	if D.RomModel == "" {
		D.RomModel = Algorithm.MacModel
	}
	if D.Imei == "" {
		D.Imei = baseinfo.IOSImei(D.Deviceid_str)
	}
	if D.SoftType == "" {
		D.SoftType = baseinfo.SoftType_iPad(D.Deviceid_str, D.OsVersion, D.RomModel)
	}
	if D.OsVersion == "" {
		D.OsVersion = Algorithm.MacOsVersion
	}
	if D.DeviceInfo == nil {
		D.DeviceInfo = createDeviceInfo(D)
	}
	D.DeviceInfoA16 = nil
	return D
}

// CreateDeviceInfo 生成新的设备信息 ipad
func createDeviceInfo(dbUserInfo *comm.LoginData) *baseinfo.DeviceInfo {
	deviceInfo := &baseinfo.DeviceInfo{}
	deviceInfo.UUIDOne = baseutils.RandomUUID() //idfv
	deviceInfo.UUIDTwo = ""                     //idfa  //高版本取不到
	deviceInfo.DeviceID = baseutils.Md5Value(strings.Replace(deviceInfo.UUIDOne, "-", "", -1))
	deviceInfo.Imei = deviceInfo.DeviceID
	deviceInfo.DeviceName = dbUserInfo.DeviceName
	deviceInfo.TimeZone = "8.00"
	deviceInfo.Language = "zh_CN"
	deviceInfo.DeviceBrand = "Apple"
	deviceInfo.RealCountry = "CN"
	deviceInfo.IphoneVer = dbUserInfo.RomModel
	deviceInfo.BundleID = "com.tencent.xin"
	deviceInfo.OsTypeNumber = dbUserInfo.OsVersion
	deviceInfo.OsType = dbUserInfo.DeviceType
	deviceInfo.CoreCount = 3 // 3核
	deviceInfo.AdSource = "" //idfa ""
	// 运营商名
	deviceInfo.CarrierName = "中国电信"
	// ClientCheckDataXML
	deviceInfo.ClientCheckDataXML = ""

	deviceInfo.GUID1 = guuid.New().String()
	deviceInfo.GUID2 = guuid.New().String()

	deviceInfo.Sdi = baseutils.Md5Value(guuid.New().String())

	//微信安装时间
	deviceInfo.InstallTime = uint64(baseutils.GetRandomTimeInPast5m().Unix()) // 5分钟内安装
	//系统上次开机时间
	deviceInfo.KernBootTime = uint64(baseutils.GetRandomTimeInPastWeek().Unix()) //一周前开机
	//系统安装时间
	SystemInstallTime := uint64(baseutils.GetRandomTimeInPastHalfYear().Unix()) //6个月前安装时间

	deviceInfo.Sysverplist = GenSysverplist(SystemInstallTime)
	deviceInfo.Dyldcache = GenDyldcache(SystemInstallTime)
	deviceInfo.Var = GenVar(SystemInstallTime)
	deviceInfo.Etcgroup = GenEtcgroup(SystemInstallTime)
	deviceInfo.Etchosts = GenEtchosts(SystemInstallTime)
	deviceInfo.Apfs = GenApfsStat()
	return deviceInfo
}

// 生成A16设备信息
func createDeviceInfoA16() *baseinfo.AndroidDeviceInfo {
	deviceInfo := &baseinfo.AndroidDeviceInfo{}
	deviceInfo.BuildBoard = "bullhead"
	return deviceInfo
}

// 生成系统文件时间
func GenSysverplist(SystemInstallTime uint64) *baseinfo.Stat {
	rand.Seed(time.Now().UnixNano())
	stat := &baseinfo.Stat{}
	stat.Inode = uint64(rand.Int63())
	stat.Statime.Tvsec = SystemInstallTime
	stat.Statime.Tvnsec = 0
	stat.Stmtime.Tvsec = SystemInstallTime
	stat.Stmtime.Tvnsec = 0
	stat.Stctime.Tvsec = SystemInstallTime
	stat.Stctime.Tvnsec = 0
	stat.Stbtime.Tvsec = SystemInstallTime
	stat.Stbtime.Tvnsec = 0
	return stat
}

// 生成dyldcache文件时间
func GenDyldcache(SystemInstallTime uint64) *baseinfo.Stat {
	rand.Seed(time.Now().UnixNano())
	stat := &baseinfo.Stat{}
	stat.Inode = uint64(rand.Int63())
	stat.Statime.Tvsec = SystemInstallTime
	stat.Statime.Tvnsec = 0
	stat.Stmtime.Tvsec = SystemInstallTime
	stat.Stmtime.Tvnsec = 0
	stat.Stctime.Tvsec = SystemInstallTime
	stat.Stctime.Tvnsec = 0
	stat.Stbtime.Tvsec = SystemInstallTime
	stat.Stbtime.Tvnsec = 0
	return stat
}

// 生成var目录时间
func GenVar(SystemInstallTime uint64) *baseinfo.Stat {
	rand.Seed(time.Now().UnixNano())
	stat := &baseinfo.Stat{}
	nsec1 := uint64(rand.Int63())
	nsec2 := uint64(rand.Int63())
	stat.Inode = 2
	last := uint64(baseutils.GetRandomTimeInPastYear().Unix())
	stat.Statime.Tvsec = SystemInstallTime - 2592000
	stat.Statime.Tvnsec = nsec1
	stat.Stmtime.Tvsec = last
	stat.Stmtime.Tvnsec = nsec2
	stat.Stctime.Tvsec = last
	stat.Stctime.Tvnsec = nsec2
	stat.Stbtime.Tvsec = SystemInstallTime - 2592000
	stat.Stbtime.Tvnsec = nsec1
	return stat
}

// 生成etc/group时间
func GenEtcgroup(SystemInstallTime uint64) *baseinfo.Stat {
	rand.Seed(time.Now().UnixNano())
	stat := &baseinfo.Stat{}
	stat.Inode = uint64(rand.Int63())
	stat.Statime.Tvsec = SystemInstallTime
	stat.Statime.Tvnsec = 0
	stat.Stmtime.Tvsec = SystemInstallTime
	stat.Stmtime.Tvnsec = 0
	stat.Stctime.Tvsec = SystemInstallTime
	stat.Stctime.Tvnsec = 0
	stat.Stbtime.Tvsec = SystemInstallTime
	stat.Stbtime.Tvnsec = 0
	return stat
}

// 生成etc/hosts时间
func GenEtchosts(SystemInstallTime uint64) *baseinfo.Stat {
	rand.Seed(time.Now().UnixNano())
	stat := &baseinfo.Stat{}
	stat.Inode = uint64(rand.Int63())
	stat.Statime.Tvsec = SystemInstallTime
	stat.Statime.Tvnsec = 0
	stat.Stmtime.Tvsec = SystemInstallTime
	stat.Stmtime.Tvnsec = 0
	stat.Stctime.Tvsec = SystemInstallTime
	stat.Stctime.Tvnsec = 0
	stat.Stbtime.Tvsec = SystemInstallTime
	stat.Stbtime.Tvnsec = 0
	return stat
}

// 生成apfs文件系统信息
func GenApfsStat() *baseinfo.Statfs {
	rand.Seed(time.Now().UnixNano())
	fs := &baseinfo.Statfs{}
	fs.Type = 26
	fs.Fstypename = "apfs"
	fs.Flags = 1417728009
	fs.Mntonname = "/"
	fs.Mntfromname = fmt.Sprintf("com.apple.os.update-{%s}@/dev/disk0s1s1", baseutils.RandomSmallHexString(40))
	fs.Fsid = 112508010497
	return fs
}

func InitHec(D *comm.LoginData) *Algorithm.Client {
	hec := &Algorithm.Client{}
	if D.DeviceType == Algorithm.AndroidPadDeviceType || D.ClientVersion == Algorithm.AndroidPadVersion {
		hec.Init("AndroidPad")
		hec.IsAndroid = true
	}
	if D.DeviceType == Algorithm.AndroidPadDeviceType || D.ClientVersion == Algorithm.AndroidPadVersionx {
		hec.Init("AndroidPad")
		hec.IsAndroid = true
		D.ClientVersion = Algorithm.AndroidPadVersion
	}
	if D.DeviceType == Algorithm.IPadDeviceType || D.ClientVersion == Algorithm.IPadVersion {
		hec.Init("IOS")
	}
	if D.DeviceType == Algorithm.IPadDeviceType || D.ClientVersion == Algorithm.IPadVersionx {
		hec.Init("IOS")
		D.ClientVersion = Algorithm.IPadVersion
	}
	if D.DeviceType == Algorithm.WinDeviceType || D.ClientVersion == Algorithm.WinVersion {
		hec.Init("Windows")
	}
	if D.ClientVersion == Algorithm.WinUwpVersion {
		hec.Init("WindowsUwp")
	}
	if D.DeviceType == Algorithm.WinUnifiedDeviceType || D.ClientVersion == Algorithm.WinUnifiedVersion {
		hec.Init("WinUnified")
	}
	if D.DeviceType == Algorithm.CarDeviceType || D.ClientVersion == Algorithm.CarVersion {
		hec.Init("Car")
	}
	if D.DeviceType == Algorithm.MacDeviceType || D.ClientVersion == Algorithm.MacVersion {
		hec.Init("MAC")
	}
	if D.DeviceType == Algorithm.AndroidDeviceType || D.ClientVersion == Algorithm.AndroidVersion || D.ClientVersion == Algorithm.AndroidVersion1 {
		hec.Init("Android")
		hec.IsAndroid = true
	}
	return hec
}
