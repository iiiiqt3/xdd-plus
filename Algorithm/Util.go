package Algorithm

import (
	"crypto/elliptic"
	"hash"
)

// 0x17000841 IOS 708
// 0x17000C2B IOS 712

//浏览器版本
//[]byte("Windows-QQBrowser")

var MmtlsShortHost = "extshort.weixin.qq.com" // "extshort.weixin.qq.com"	// "szshort.weixin.qq.com"
var MmtlsLongHost = "long.weixin.qq.com"
var MmtlsLongPort = 80

// ipad 参数
var IosBuildVersion = "19H386"
var KernelType = "Darwin"
var KernelVersion = "21.6.0"
var KernelRelease = "Darwin Kernel Version 21.6.0: Sun Oct 15 00:17:39 PDT 2023; root:xnu-8020.241.42~8/RELEASE_ARM64_T7000"

// ipad
//var IPadDeviceType = "iPad iPadOS18.0.1"
//var IPadDeviceName = "iPad Pro 13(M4)"
//var IPadModel = "iPad16,6"
//var IPadOsVersion = "18.0.1"

var IPadDeviceType = "iPad Air iPadOS18.8.1"
var IPadDeviceName = "iPad Air (第7代)"
var IPadModel = "iPad14,4"
var IPadOsVersion = "18.8.1"

// iphone
var IPhoneDeviceType = "iPhone iOS18.8.1"
var IPhoneDeviceName = "iPhone 16 Pro"
var IPhoneModel = "iPhone17,1"
var IPhoneOsVersion = "18.8.1"

// 安卓
var AndroidDeviceType = "android-34"
var AndroidManufacture = "HUAWEI Mate XT"
var AndroidDeviceName = "HUAWEI"
var AndroidModel = "GRL-AL10"
var AndroidOsVersion = "12"
var AndroidIncremental = "1"

// 安卓 pad
// var AndroidPadDeviceType = "pad-android-34"
// var AndroidPadModel = "HUAWEI MRO-W00" //HUAWEI MatePad Pro
// var AndroidPadDeviceName = "HUAWEI MatePad Pro"
// var AndroidPadOsVersion = "10"
var AndroidPadDeviceType = "pad-android-34"
var AndroidPadModel = "XMP7Pro-W00"
var AndroidPadDeviceName = "Xiaomi Pad 7 Pro"
var AndroidPadOsVersion = "15"

var WinUnifiedDeviceType = "UnifiedPCWindows 11 x86_64"
var WinUnifiedDeviceName = "DESKTOP-P0QLAW8"
var WinUnifiedModel = "ASUS"
var WinUnifiedOsVersion = "11"

// win
var WinDeviceType = "Windows 11 x64"
var WinDeviceName = "DESKTOP-P0QLAW8" //
var WinModel = "ASUS"
var WinOsVersion = "11"

var IPadDeviceTypeWin = "windows 10 x64"

// var IPadDeviceType = "iPhone iOS16.1.2"
var IPadModelWin = "windows 10 x64"

// 车载
var CarDeviceType = "car-31"
var CarDeviceName = "Xiaomi-M2012K11AC"
var CarModel = "Xiaomi-M2012K11AC"
var CarOsVersion = "10"

// mac
// var MacDeviceType = "iMac MacBookPro16,1 OSX OSX11.5.2 build(20G95)"
// var MacDeviceName = "MacBook Pro"
// var MacModel = "iMac MacBookPro16,1"
// var MacOsVersion = "11.5.2"
var MacDeviceType = "iMac MacBookPro16,1 OSX OSX11.5.2 build(20G95)"
var MacDeviceName = "MacBook Pro"
var MacModel = "MacBookPro16,1"
var MacOsVersion = "11.5.2"

// 版本号
// var IPadVersion = int32(0x18003926)
var IPadVersion = int32(0x18003d18) // 0x18003b20 0x18003d20 8.0.61

// var IPadVersion = int32(0x18003B26)  //ipad 0x18003727
var IPadVersionx = int32(0x18003926) //ipad绕过验证码int32(0x17000523)

var IPhoneVersion = int32(0x18003637) //62IPhone

// var AndroidVersion = int32(0x28003653) //A16Android
var AndroidVersion = int32(0x28003d34)  //A16Android 0x28003a35
var AndroidVersion1 = int32(0x28003a35) //A16Android

// var AndroidPadVersion = int32(0x28003653)  //安卓平板
var AndroidPadVersion = int32(0x28003d34)  // 安卓平板 861
var AndroidPadVersionx = int32(0x28003a35) //安卓平板绕过验证码

var WinVersion = int32(0x1308090B)        //win 0x63090C11
var WinUwpVersion = int32(0x620603C8)     //winuwp绕过验证码
var WinUnifiedVersion = int32(0x6400010D) //WinUnified

var CarVersion = int32(0x21000B1B) //车载 0x21000B1B 0x2100091B

var MacVersion = int32(0x13080a10) //mac 0x1308080B 0x1308090B 0x13080a10

var Md5OfMachOHeader = string("d55ce16228afb0ea5205380af376761e")

type HYBRID_STATUS int32

const (
	HYBRID_ENC HYBRID_STATUS = 0
	HYBRID_DEC HYBRID_STATUS = 1
)

type Client struct {
	PubKey     []byte
	Privkey    []byte
	InitPubKey []byte
	Externkey  []byte

	Version    int32
	DeviceType string

	clientHash hash.Hash
	serverHash hash.Hash

	curve elliptic.Curve

	IsAndroid bool

	Status HYBRID_STATUS
}

type PacketHeader struct {
	PacketCryptType byte
	Flag            uint16
	RetCode         uint32
	UICrypt         uint32
	Uin             uint32
	Cookies         []byte
	Data            []byte
}

type PackData struct {
	Reqdata          []byte
	Cgi              int
	Uin              uint32
	Cookie           []byte
	ClientVersion    int
	Sessionkey       []byte
	EncryptType      uint8
	Loginecdhkey     []byte
	Clientsessionkey []byte
	Serversessionkey []byte
	UseCompress      bool
	MMtlsClose       bool
}
