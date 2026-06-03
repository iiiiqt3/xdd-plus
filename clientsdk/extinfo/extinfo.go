package extinfo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math/rand"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/Cilent/wechat"
	"wechatdll/baseinfo"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/clientsdk/ccdata"
	"wechatdll/clientsdk/mmproto"
	v08 "wechatdll/clientsdk/v08"
	"wechatdll/comm"

	"github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type GetCcDataRep struct {
	Code int    `json:"code"`
	Data string `json:"data"`
	Msg  string `json:"msg"`
}

func MakeXorKey(key int64) uint8 {
	var un int64 = int64(0xffffffed)
	xorKey := (uint8)(key*un + 7)
	return xorKey
}

func exponent(a, n uint64) uint64 {
	result := uint64(1)
	for i := n; i > 0; i >>= 1 {
		if i&1 != 0 {
			result *= a
		}
		a *= a
	}
	return result
}

func Hex2int(hexB *[]byte) uint64 {
	var retInt uint64
	hexLen := len(*hexB)
	for k, v := range *hexB {
		retInt += b2m_map[v] * exponent(16, uint64(2*(hexLen-k-1)))
	}
	return retInt
}

func DeviceNumber(DeviceId string) int64 {
	ssss := []byte(baseutils.Md5Value(DeviceId))
	ccc := Hex2int(&ssss) >> 8
	ddd := ccc + 60000000000000000
	if ddd > 80000000000000000 {
		ddd = ddd - (80000000000000000 - ddd)
	}
	return int64(ddd)
}

var wifiPrefix = []string{"TP_", "360_", "ChinaNet-", "MERCURY_", "DL-", "VF_", "HUAW-"}

func BuildRandomWifiSsid() string {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	i := r.Intn(len(wifiPrefix))
	randChar := make([]byte, 6)
	for x := 0; x < 6; x++ {
		randChar[x] = byte(r.Intn(26) + 65)
	}
	return wifiPrefix[i] + string(randChar)
}

func CheckSoftType5() uint32 {
	sec := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(999) * 1000
	v79 := uint32(sec)&0xe | 1
	key := v79

	v77 := uint32(134217728)
	n := uint32(4)

	for true {
		dwTmp := n & 3
		if dwTmp == 0 {
			v79 = (3877*v79 + 5) & 0xf
		}

		dwTmp = uint32(((int(v79) >> int(dwTmp)) & 1)) << int(n)
		v77 ^= dwTmp
		n++
		if n == 24 {
			break
		}
	}
	return v77 | key
}

// iphone生成ccd
func GetiPhoneNewSpamData(userInfo *comm.LoginData) []byte {
	Deviceid := userInfo.Deviceid_str
	DeviceName := userInfo.DeviceName
	DeviceToken := userInfo.DeviceToken
	timeStamp := int(time.Now().Unix())
	xorKey := uint8((timeStamp * 0xffffffed) + 7)

	uuid1, uuid2 := baseinfo.IOSUuid(Deviceid)

	if len(Deviceid) < 32 {
		Dlen := 32 - len(Deviceid)
		Fill := "ff95DODUJ4EysYiogKZSmajWCUKUg9RX"
		Deviceid = Deviceid + Fill[:Dlen]
	}

	spamDataBody := &mm.SpamDataBody{
		UnKnown1:              proto.Int32(1),
		TimeStamp:             proto.Int32(int32(timeStamp)),
		KeyHash:               proto.Int32(int32(MakeKeyHash(xorKey))),
		Yes1:                  proto.String(XorEncodeStr("yes", xorKey)),
		Yes2:                  proto.String(XorEncodeStr("yes", xorKey)),
		IosVersion:            proto.String(XorEncodeStr(userInfo.OsVersion, xorKey)),
		DeviceType:            proto.String(XorEncodeStr("iPhone", xorKey)),
		UnKnown2:              proto.Int32(2),
		IdentifierForVendor:   proto.String(XorEncodeStr(uuid1, xorKey)),
		AdvertisingIdentifier: proto.String(XorEncodeStr(uuid2, xorKey)),
		Carrier:               proto.String(XorEncodeStr("中国移动", xorKey)),
		BatteryInfo:           proto.Int32(1),
		NetworkName:           proto.String(XorEncodeStr("en0", xorKey)),
		NetType:               proto.Int32(1),
		AppBundleId:           proto.String(XorEncodeStr("com.tencent.xin", xorKey)),
		DeviceName:            proto.String(XorEncodeStr(DeviceName, xorKey)),
		UserName:              proto.String(XorEncodeStr(userInfo.RomModel, xorKey)),
		Unknown3:              proto.Int64(baseinfo.IOSDeviceNumber(Deviceid[:29] + "FFF")),
		Unknown4:              proto.Int64(baseinfo.IOSDeviceNumber(Deviceid[:29] + "OOO")),
		Unknown5:              proto.Int32(1),
		Unknown6:              proto.Int32(4),
		Lang:                  proto.String(XorEncodeStr("zh", xorKey)),
		Country:               proto.String(XorEncodeStr("CN", xorKey)),
		Unknown7:              proto.Int32(4),
		DocumentDir:           proto.String(XorEncodeStr("/var/mobile/Containers/Data/Application/"+baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x10101201))+"/Documents", xorKey)),
		Unknown8:              proto.Int32(0),
		Unknown9:              proto.Int32(0),
		HeadMD5:               proto.String(XorEncodeStr(baseinfo.IOSGetCidMd5(Deviceid, baseinfo.IOSGetCid(0x0262626262626)), xorKey)),
		AppUUID:               proto.String(XorEncodeStr(uuid1, xorKey)),
		SyslogUUID:            proto.String(XorEncodeStr(uuid2, xorKey)),
		Unknown10:             proto.String(""),
		Unknown11:             proto.String(""),
		AppName:               proto.String(XorEncodeStr("微信", xorKey)),
		SshPath:               proto.String(""),
		TempTest:              proto.String(""),
		DevMD5:                proto.String(""),
		DevUser:               proto.String(""),
		Unknown12:             proto.String(""),
		IsModify:              proto.Int32(0),
		ModifyMD5:             proto.String(""),
		RqtHash:               proto.Int64(288529533794259264),
		Unknown43:             proto.Uint64(1586355322),
		Unknown44:             proto.Uint64(1586355519000),
		Unknown45:             proto.Uint64(0),
		Unknown46:             proto.Uint64(288529533794259264),
		Unknown47:             proto.Uint64(0),
		Unknown48:             proto.String(Deviceid),
		Unknown49:             proto.String(""),
		Unknown50:             proto.String(""),
		Unknown51:             proto.String(XorEncodeStr(DeviceToken.GetTrustResponseData().GetSoftData().GetSoftConfig(), xorKey)),
		Unknown52:             proto.Uint64(0),
		Unknown53:             proto.String(""),
		Unknown54:             proto.String(XorEncodeStr(DeviceToken.GetTrustResponseData().GetDeviceToken(), xorKey)),
	}
	wxFile := &mm.FileInfo{
		Fileuuid: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x098521236654))+"/WeChat.app/WeChat", xorKey)),
		Filepath: proto.String(XorEncodeStr(baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x30000001)), xorKey)),
	}
	spamDataBody.AppFileInfo = append(spamDataBody.AppFileInfo, wxFile)

	opensslFile := &mm.FileInfo{
		Fileuuid: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x098521236654))+"/WeChat.app/Frameworks/OpenSSL.framework/OpenSSL", xorKey)),
		Filepath: proto.String(XorEncodeStr(baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x30000002)), xorKey)),
	}
	spamDataBody.AppFileInfo = append(spamDataBody.AppFileInfo, opensslFile)

	protoFile := &mm.FileInfo{
		Fileuuid: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x098521236654))+"/WeChat.app/Frameworks/ProtobufLite.framework/ProtobufLite", xorKey)),
		Filepath: proto.String(XorEncodeStr(baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x30000003)), xorKey)),
	}
	spamDataBody.AppFileInfo = append(spamDataBody.AppFileInfo, protoFile)

	marsbridgenetworkFile := &mm.FileInfo{
		Fileuuid: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x098521236654))+"/WeChat.app/Frameworks/marsbridgenetwork.framework/marsbridgenetwork", xorKey)),
		Filepath: proto.String(XorEncodeStr(baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x30000004)), xorKey)),
	}
	spamDataBody.AppFileInfo = append(spamDataBody.AppFileInfo, marsbridgenetworkFile)

	matrixreportFile := &mm.FileInfo{
		Fileuuid: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x098521236654))+"/WeChat.app/Frameworks/matrixreport.framework/matrixreport", xorKey)),
		Filepath: proto.String(XorEncodeStr(baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x30000005)), xorKey)),
	}
	spamDataBody.AppFileInfo = append(spamDataBody.AppFileInfo, matrixreportFile)

	andromedaFile := &mm.FileInfo{
		Fileuuid: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x098521236654))+"/WeChat.app/Frameworks/andromeda.framework/andromeda", xorKey)),
		Filepath: proto.String(XorEncodeStr(baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x30000006)), xorKey)),
	}
	spamDataBody.AppFileInfo = append(spamDataBody.AppFileInfo, andromedaFile)

	marsFile := &mm.FileInfo{
		Fileuuid: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x098521236654))+"/WeChat.app/Frameworks/mars.framework/mars", xorKey)),
		Filepath: proto.String(XorEncodeStr(baseinfo.IOSGetCidUUid(Deviceid, baseinfo.IOSGetCid(0x30000007)), xorKey)),
	}
	spamDataBody.AppFileInfo = append(spamDataBody.AppFileInfo, marsFile)
	srcdata, _ := proto.Marshal(spamDataBody)

	newClientCheckData := &mm.NewClientCheckData{
		C32Cdata:  proto.Int64(int64(crc32.ChecksumIEEE([]byte(srcdata)))),
		TimeStamp: proto.Int64(int64(timeStamp)),
		Databody:  srcdata,
	}

	ccddata, _ := proto.Marshal(newClientCheckData)
	afterCompressionCCData := v08.Compress(ccddata)
	afterEnData, _ := ccdata.EncodeZipData(afterCompressionCCData, 0x3060)
	//压缩数据
	// compressdata := AE(ccddata)

	// // Zero: 03加密改06加密
	// // zt := new(ZT)
	// // zt.Init()
	// // encData := zt.Encrypt(compressdata)
	// encData := SaeEncrypt06(compressdata)

	return afterEnData
}

// 08算法
func GetNewSpamDataV8(userInfo *comm.LoginData) []byte {
	start := time.Now()
	timeStamp := uint32(start.Unix())
	xorKey := MakeXorKey(int64(timeStamp))

	uuid1, uuid2 := baseutils.IOSUuid(userInfo.Deviceid_str)

	spamDataBody := wechat.SpamDataBody{
		UnKnown1:   proto.Int32(1),
		TimeStamp:  proto.Uint32(uint32(timeStamp)),
		KeyHash:    proto.Int32(int32(MakeKeyHash(xorKey))),
		Yes1:       proto.String(XorEncodeStr("yes", xorKey)),
		Yes2:       proto.String(XorEncodeStr("yes", xorKey)),
		IosVersion: proto.String(XorEncodeStr(userInfo.DeviceInfo.OsTypeNumber, xorKey)),
		DeviceType: proto.String(XorEncodeStr(userInfo.DeviceInfo.DeviceName, xorKey)),
		CoreCount:  proto.Int64(EncInt(int64(userInfo.DeviceInfo.CoreCount))),
		//IdentifierForVendor:   proto.String(XorEncodeStr(userInfo.DeviceInfo.UUIDOne, xorKey)),
		//AdvertisingIdentifier: proto.String(XorEncodeStr(userInfo.DeviceInfo.UUIDTwo, xorKey)),
		IdentifierForVendor:   proto.String(XorEncodeStr(uuid1, xorKey)),
		AdvertisingIdentifier: proto.String(XorEncodeStr(uuid2, xorKey)),
		Carrier:               proto.String(XorEncodeStr(userInfo.DeviceInfo.CarrierName, xorKey)),
		BatteryInfo:           proto.Int32(1),
		NetworkName: []string{
			XorEncodeStr("en0", xorKey),
			XorEncodeStr("utun3", xorKey),
		},
		NetType:             proto.Int32(1),
		AppBundleId:         proto.String(XorEncodeStr("com.tencent.xin", xorKey)),
		DeviceName:          proto.String(XorEncodeStr(userInfo.DeviceName, xorKey)),
		UserName:            proto.String(XorEncodeStr(userInfo.RomModel, xorKey)),
		GetVersion:          proto.Int64(EncInt(int64(userInfo.ClientVersion))),
		GetVersionFromPList: proto.Int64(EncInt(int64(userInfo.ClientVersion))),
		Unknown5:            proto.Int32(0), //IsJailbreak
		Unknown6:            proto.Int32(4),
		Lang:                proto.String(XorEncodeStr("zh", xorKey)),
		Country:             proto.String(XorEncodeStr("CN", xorKey)),
		Unknown7:            proto.Int32(4),
		DocumentDir:         proto.String(XorEncodeStr(fmt.Sprintf("/var/mobile/Containers/Data/Application/%s/Documents", userInfo.DeviceInfo.GUID1), xorKey)),
		Unknown8:            proto.Int32(0),
		Unknown9:            proto.Int32(0),
		HeadMD5:             proto.String(XorEncodeStr(Algorithm.Md5OfMachOHeader, xorKey)),
		//AppUUID:             proto.String(XorEncodeStr(Algorithm.AppUUID, xorKey)),
		//AppUUID: proto.String(XorEncodeStr(strings.ToUpper(uuid1), xorKey)),
		AppUUID: proto.String(XorEncodeStr(uuid1, xorKey)),

		SyslogUUID: proto.String(""),
		Unknown10:  proto.String(""),
		Unknown11:  proto.String(""),

		AppName:      proto.String(XorEncodeStr("微信", xorKey)),
		SshPath:      proto.String(XorEncodeStr("/usr/bin/ssh", xorKey)),
		TempTest:     proto.String(XorEncodeStr("/tmp/test.txt", xorKey)),
		DevMD5:       proto.String(""),
		DevUser:      proto.String(""),
		DevPrefix:    proto.String(""),
		AppFileInfo:  GetFileInfo(userInfo.DeviceInfo.GUID2, xorKey),
		Unknown12:    proto.String(""),
		IsModify:     proto.Int32(0),
		Sdi:          proto.String(XorEncodeStr(userInfo.DeviceInfo.Sdi, xorKey)),
		RqtHash:      proto.Int64(EncInt(int64(v08.Rqtx(userInfo.DeviceInfo.Sdi)))),
		InstallTime:  proto.Uint64(userInfo.DeviceInfo.InstallTime),
		KernBootTime: proto.Uint64(uint64(userInfo.DeviceInfo.KernBootTime)),
		Unknown55:    proto.Uint64(0),
		RqtHash56:    proto.Int64(EncInt(int64(v08.Rqtx(userInfo.DeviceInfo.Sdi)))),
		Unknown57:    proto.Uint64(0), //固定值0
		DeviceId:     proto.String(XorEncodeStr(userInfo.DeviceInfo.DeviceID, xorKey)),
		DeviceIdCrc:  proto.Int64(EncInt(int64(crc32.ChecksumIEEE([]byte(userInfo.DeviceInfo.DeviceID))))),
		Unknown61:    proto.String(XorEncodeStr("2FFC7F6DFEEFFF2B3FFCA029", xorKey)),
		Unknown62:    proto.Uint64(1175744137544159509),
	}

	appFileInfo := new(bytes.Buffer)
	appFile := make([]string, 0)
	// 定义文件路径和 UUID 的数组
	filePath := []string{
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/WeChat",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/ProtobufLite.framework/ProtobufLite",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/TPThirdParties.framework/TPThirdParties",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/TPFFmpeg.framework/TPFFmpeg",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/owl.framework/owl",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/ilink_network.framework/ilink_network",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/SoundTouch.framework/SoundTouch",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/JavaScriptCore2.framework/JavaScriptCore2",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/MMRouter.framework/MMRouter",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/Lottie.framework/Lottie",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/andromeda.framework/andromeda",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/openssl.framework/openssl",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/matrixreport.framework/matrixreport",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/NewMessageRingUtil.framework/NewMessageRingUtil",
		"/private/var/containers/Bundle/Application/" + userInfo.DeviceInfo.GUID2 + "/WeChat.app/Frameworks/App.framework/App",
	}

	fileUUid := []string{
		"12623847-C8BD-3445-834D-2A01ED0D89DB",
		"93F5C16E-5A38-3470-8693-5F2D61D44BC9",
		"AE00E121-7AE7-3D21-BD9D-EAE037070F51",
		"2A93E14D-74FB-399E-8CBC-6CCFB3CF20BA",
		"56D35845-AB27-3155-8346-135D5C13119C",
		"F7FEA368-B01B-3254-AB32-D2DC81731A3D",
		"B1E6FC94-E0EC-3510-B643-6C3FEB8EA1FC",
		"793CC474-E23C-3DA5-8E56-2E3A2AAF555E",
		"0B66FAB5-7E23-3A10-855E-0578061A6346",
		"38BA63F3-9650-38B1-8BEA-D1FE67986D21",
		"BEF1F572-4833-3C26-AEB7-2EFBEEC03EEB",
		"31B5B8CB-7931-3930-BA0C-E6FFA1C9A220",
		"1E7F06D2-DD36-31A8-AF3B-00D62054E1F9",
		"283E6705-73C8-3E56-AB1B-D218AC7B0A76",
		"3D25843A-86C6-3B6C-B5DA-770FDCBA679F",
	}

	// 将文件路径和 UUID 填充到 appFile 中
	for i := 0; i < len(filePath); i++ {
		appFile = append(appFile, filePath[i])
		appFile = append(appFile, fileUUid[i])
	}

	appFileInfo.WriteString(strings.Join(appFile, ""))
	encInt := EncInt(int64(crc32.ChecksumIEEE(appFileInfo.Bytes())))
	spamDataBody.AppFileInfoCrc = proto.String(fmt.Sprintf("%v", encInt))

	deviceToken := userInfo.DeviceToken
	if deviceToken != nil {
		//fp si deviceToken
		soft_config := deviceToken.GetTrustResponseData().GetSoftData().GetSoftConfig()
		soft_data := deviceToken.GetTrustResponseData().GetSoftData().GetSoftData()
		si := v08.Si(soft_config, string(soft_data))
		spamDataBody.Si = proto.String(XorEncodeStr(si, xorKey))
		spamDataBody.DeviceToken = proto.String(XorEncodeStr(deviceToken.GetTrustResponseData().GetDeviceToken(), xorKey))
	} else {
		log.Error("ccd deviceToken err")
	}

	//timestamp
	now := time.Now()
	spamDataBody.Now = proto.Uint32(EncodeGetTimeOfDay(uint64(now.Unix()), uint32(now.Nanosecond()/1e3)))
	usec := uint32(start.Nanosecond() / 1e3) //microsecond
	spamDataBody.NowUsec = proto.Uint32(usec)
	spamDataBody.NowUsecScale = proto.Uint32(uint32(uint32(usec)*0x68323 + 0x11))
	//file stat
	timespec := userInfo.DeviceInfo.Sysverplist.Stctime
	nsec := timespec.Tvsec * 1e9
	spamDataBody.SystemVersionCtime = proto.Uint64(v08.EncodeUInt64(nsec, 0xD91CA739, timeStamp))
	timespec = userInfo.DeviceInfo.Dyldcache.Stctime
	nsec = timespec.Tvsec * 1e9
	spamDataBody.DyldSharedCacheArm64Ctime = proto.Uint64(v08.EncodeUInt64(nsec, 0xA3071842, timeStamp))
	timespec = userInfo.DeviceInfo.Var.Stctime
	nsec = timespec.Tvsec * 1e9
	spamDataBody.PrivateVarDirCTime = proto.Uint64(v08.EncodeUInt64(nsec, 0x5A5F1852, timeStamp))

	spamDataBody.EmptyString = proto.String("")
	spamDataBody.IsCaptured = proto.Uint64(0)
	// spamDataBody.ShortBundleVersion = proto.String(v08.EncodeString(Algorithm.ShortBundleVersion, 0xD526377C, timeStamp))

	//filesystem
	spamDataBody.StatfsInfo = []*wechat.Statfs{
		{
			FType:        proto.Uint64(v08.EncodeUInt64(userInfo.DeviceInfo.Apfs.Type, 0x99687B8A, timeStamp)),
			FFstypename:  proto.String(v08.EncodeString(userInfo.DeviceInfo.Apfs.Fstypename, 0xE2868E99, timeStamp)),
			FFlags:       proto.Uint64(v08.EncodeUInt64(userInfo.DeviceInfo.Apfs.Flags, 0xC36FA9A2, timeStamp)),
			FMntonname:   proto.String(v08.EncodeString(userInfo.DeviceInfo.Apfs.Mntonname, 0xB62136B0, timeStamp)),
			FMntfromname: proto.String(v08.EncodeString(userInfo.DeviceInfo.Apfs.Mntfromname, 0x47EE23C4, timeStamp)),
			FFsid:        proto.Uint64(v08.EncodeUInt64(userInfo.DeviceInfo.Apfs.Fsid, 0xA19A7ED5, timeStamp)),
		},
	}

	crc := crc32Calc(uint64ToBytes(userInfo.DeviceInfo.Apfs.Flags), 0xFFFFFFFF)
	crc = crc32Calc(uint64ToBytes(userInfo.DeviceInfo.Apfs.Type), ^crc)
	crc = crc32Calc(uint64ToBytes(userInfo.DeviceInfo.Apfs.Fsid), ^crc)
	crc = crc32Calc([]byte(userInfo.DeviceInfo.Apfs.Fstypename), ^crc)
	crc = crc32Calc([]byte(userInfo.DeviceInfo.Apfs.Mntonname), ^crc)
	crc = crc32Calc([]byte(userInfo.DeviceInfo.Apfs.Mntfromname), ^crc)

	spamDataBody.StatfsCrc = proto.Uint32(EncodeStatfsCrc(crc, timeStamp))
	spamDataBody.FixedOne = proto.Uint64(v08.EncodeUInt64(1, 0x75A9F3FA, timeStamp))
	spamDataBody.AppexList = []string{
		v08.EncodeString("WeChatWatchNativeExtension.appex", 0x540CF80D, timeStamp),
		v08.EncodeString("WeChatSiriExtensionUI.appex", 0x540CF80D, timeStamp),
		v08.EncodeString("WeChatSiriExtension.appex", 0x540CF80D, timeStamp),
		v08.EncodeString("WeChatScreenCapture.appex", 0x540CF80D, timeStamp),
		v08.EncodeString("WeChatNotificationServiceExtension.appex", 0x540CF80D, timeStamp),
		v08.EncodeString("WeChatWidgetExtension.appex", 0x540CF80D, timeStamp),
	}
	spamDataBody.FrameworkList = []string{
		v08.EncodeString("ilink_network.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("TPFFmpeg.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("NewMessageRingUtil.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("SoundTouch.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("openssl.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("MMRouter.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("owl.framework.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("TPThirdParties.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("andromeda.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("matrixreport.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("Lottie.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("ProtobufLite.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("App.framework", 0x8C36EE2A, timeStamp),
		v08.EncodeString("JavaScriptCore2.framework", 0x8C36EE2A, timeStamp),
	}
	///private/var
	spamDataBody.PrivateVarDir = new(wechat.FileStat)
	timespec = userInfo.DeviceInfo.Var.Stbtime
	nsec = timespec.Tvsec*1e9 + timespec.Tvnsec
	spamDataBody.PrivateVarDir.StBirthtime = proto.Uint64(v08.EncodeUInt64(nsec, 0x2D6ACA52, timeStamp))
	timespec = userInfo.DeviceInfo.Var.Stctime
	nsec = timespec.Tvsec*1e9 + timespec.Tvnsec
	spamDataBody.PrivateVarDir.StCtime = proto.Uint64(v08.EncodeUInt64(nsec, 0x2D6ACA52, timeStamp))
	timespec = userInfo.DeviceInfo.Var.Stmtime
	nsec = timespec.Tvsec*1e9 + timespec.Tvnsec
	spamDataBody.PrivateVarDir.StMtime = proto.Uint64(v08.EncodeUInt64(nsec, 0x2D6ACA52, timeStamp))
	nsec = userInfo.DeviceInfo.InstallTime*1e9 + 0x12241352
	spamDataBody.WechatDocBirth = proto.Uint64(v08.EncodeUInt64(nsec, 0x30C0D228, timeStamp))
	spamDataBody.LAPolicyDeviceOwnerAuthentication = proto.Uint64(v08.EncodeUInt64(1, 0xA0DAC236, timeStamp))
	spamDataBody.LAPolicyDeviceOwnerAuthenticationWithBiometrics = proto.Uint64(v08.EncodeUInt64(1, 0x6C964322, timeStamp))
	spamDataBody.ICloudLogin = proto.Uint64(v08.EncodeUInt64(1, 0x92A52BCE, timeStamp))
	// spamDataBody.UbiquityIdentityToken = proto.String(v08.EncodeString(userInfo.DeviceInfo.UbiquityIdentityToken, 0x41209ADD, timeStamp))
	appStoreReceiptURL := fmt.Sprintf("/private/var/mobile/Containers/Data/Application/%s/StoreKit/Receipt", userInfo.DeviceInfo.GUID1)
	spamDataBody.AppStoreReceiptURL = proto.String(v08.EncodeString(appStoreReceiptURL, 0x985B22AA, timeStamp))
	spamDataBody.SandboxReceiptExist = proto.Uint64(v08.EncodeUInt64(0, 0x236DE3FF, timeStamp))
	spamDataBody.NSClassFromString = proto.Uint64(v08.EncodeUInt64(0, 0x3AA9A3CE, timeStamp))
	//system
	spamDataBody.IosBuildVersion = proto.String(v08.EncodeString(Algorithm.IosBuildVersion, 0x554738E7, timeStamp))
	spamDataBody.KernelType = proto.String(v08.EncodeString(Algorithm.KernelType, 0xEC1B9627, timeStamp))
	spamDataBody.DeviceModel = proto.String(v08.EncodeString(userInfo.RomModel, 0x562E8BDC, timeStamp))
	spamDataBody.KernelVersion = proto.String(v08.EncodeString(Algorithm.KernelVersion, 0x318BC072, timeStamp))
	spamDataBody.KernelRelease = proto.String(v08.EncodeString(Algorithm.KernelRelease, 0x3B696B8D, timeStamp))
	spamDataBody.DevciceType = proto.String(v08.EncodeString(userInfo.DeviceInfo.IphoneVer, 0x554738E7, timeStamp))

	data, _ := proto.Marshal(&spamDataBody)

	newClientCheckData := &wechat.NewClientCheckData{
		C32CData:  proto.Int64(int64(crc32.ChecksumIEEE(data))),
		TimeStamp: proto.Int64(time.Now().Unix()),
		DataBody:  data,
	}

	ccData, _ := proto.Marshal(newClientCheckData)

	afterCompressionCCData := v08.Compress(ccData)
	afterEnData, _ := ccdata.EncodeZipData(afterCompressionCCData, 0x3060)

	deviceRunningInfo := &wechat.DeviceRunningInfoNew{
		Version:     []byte("00000008"),
		Type:        proto.Uint32(1),
		EncryptData: afterEnData,
		Timestamp:   proto.Uint32(uint32(time.Now().Unix())),
		Unknown5:    proto.Uint32(5),
		Unknown6:    proto.Uint32(0),
	}
	result, _ := proto.Marshal(deviceRunningInfo)
	return result
}

// 获取FilePathCrc
func GetFileInfo(guid2 string, xorKey byte) []*wechat.FileInfo {
	fileInfos := []*wechat.FileInfo{
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/WeChat", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("12623847-C8BD-3445-834D-2A01ED0D89DB", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/ProtobufLite.framework/ProtobufLite", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("93F5C16E-5A38-3470-8693-5F2D61D44BC9", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/TPThirdParties.framework/TPThirdParties", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("AE00E121-7AE7-3D21-BD9D-EAE037070F51", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/TPFFmpeg.framework/TPFFmpeg", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("2A93E14D-74FB-399E-8CBC-6CCFB3CF20BA", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/owl.framework/owl", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("56D35845-AB27-3155-8346-135D5C13119C", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/ilink_network.framework/ilink_network", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("F7FEA368-B01B-3254-AB32-D2DC81731A3D", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/SoundTouch.framework/SoundTouch", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("B1E6FC94-E0EC-3510-B643-6C3FEB8EA1FC", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/JavaScriptCore2.framework/JavaScriptCore2", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("793CC474-E23C-3DA5-8E56-2E3A2AAF555E", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/MMRouter.framework/MMRouter", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("0B66FAB5-7E23-3A10-855E-0578061A6346", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/Lottie.framework/Lottie", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("38BA63F3-9650-38B1-8BEA-D1FE67986D21", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/andromeda.framework/andromeda", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("BEF1F572-4833-3C26-AEB7-2EFBEEC03EEB", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/openssl.framework/openssl", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("31B5B8CB-7931-3930-BA0C-E6FFA1C9A220", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/matrixreport.framework/matrixreport", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("1E7F06D2-DD36-31A8-AF3B-00D62054E1F9", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/NewMessageRingUtil.framework/NewMessageRingUtil", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("283E6705-73C8-3E56-AB1B-D218AC7B0A76", xorKey)),
		},
		{
			Filepath: proto.String(XorEncodeStr("/var/containers/Bundle/Application/"+guid2+"/WeChat.app/Frameworks/App.framework/App", xorKey)),
			Fileuuid: proto.String(XorEncodeStr("3D25843A-86C6-3B6C-B5DA-770FDCBA679F", xorKey)),
		},
	}
	return fileInfos
}

// 移位操作
func EncInt(d int64) int64 {
	a, b := int64(0), int64(0)
	for i := 0; i < 16; i++ {
		a |= ((1 << (2 * i)) & d) << (2 * i)
		b |= ((1 << (2*i + 1)) & d) << (2*i + 1)
	}
	return a | b
}

func EncodeGetTimeOfDay(seconds uint64, usec uint32) uint32 {
	w12 := usec & 0xffffffe0
	w12Diff := usec - w12
	x13 := seconds << w12Diff
	w14 := uint32(0x20)
	x12 := w14 - w12Diff
	x11 := seconds >> x12
	w11 := x11 | x13
	w22 := usec ^ uint32(w11)
	w9 := w22 & 0xfffffc3f
	w24 := w9 | 0x40
	return w24
}

// 自定义 CRC32 计算函数，支持指定初始 CRC 值
func crc32Calc(data []byte, initialCRC uint32) uint32 {
	// 使用 IEEE 标准的 CRC32 表
	table := crc32.IEEETable

	// 初始化 CRC 值
	crc := initialCRC

	// 计算 CRC32
	for _, b := range data {
		tableIndex := (crc ^ uint32(b)) & 0xFF
		crc = (crc >> 8) ^ table[tableIndex]
	}

	// 取反得到最终结果
	return ^crc
}

// 将 uint64 转换为字节切片
func uint64ToBytes(value uint64) []byte {
	bytes := make([]byte, 8) // uint64 是 8 字节
	binary.LittleEndian.PutUint64(bytes, value)
	return bytes
}

// 循环右移函数
func ror(value uint32, shift uint) uint32 {
	return (value >> shift) | (value << (32 - shift))
}

func EncodeStatfsCrc(w8 uint32, timestamp uint32) uint32 {
	w8 = timestamp ^ ror(w8, 31)
	timestamp = 0x6fe3 | (0xc5b2 << 16)
	w8 ^= timestamp
	w8 = ror(w8, 31)
	return w8
}

// 获取DeviceToken
func GetDeviceToken(deviceToken string) *mmproto.DeviceToken {
	curtime := uint32(time.Now().Unix())
	return &mmproto.DeviceToken{
		Version:   proto.String(""),
		Encrypted: proto.Uint32(1),
		Data: &mmproto.SKBuiltinStringt{
			String_: proto.String(deviceToken),
		},
		TimeStamp: &curtime,
		Optype:    proto.Uint32(2),
		Uin:       proto.Uint32(0),
	}
}
func GenGUId(DeviceId, Cid string) string {
	Md5Data := baseutils.Md5Value(DeviceId + Cid)
	return fmt.Sprintf("%x-%x-%x-%x-%x", Md5Data[0:8], Md5Data[2:6], Md5Data[3:7], Md5Data[1:5], Md5Data[20:32])
}
