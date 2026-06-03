package Algorithm

import (
	"bytes"
	"compress/zlib"

	"crypto/rand"
	"hash/crc32"
	"io"

	math_rand "math/rand"
	"time"
	"wechatdll/Cilent/mm"
	"wechatdll/Cilent/wechat"
	"wechatdll/baseinfo"

	"wechatdll/clientsdk/ccdata"
	v08 "wechatdll/clientsdk/v08"

	"github.com/golang/protobuf/proto"
)

func GetWcstf08(userName string) ([]byte, error) {
	curtime := time.Now().UnixNano() / 1000000
	contentLen := len(userName)
	ct := make([]uint64, 0)
	ut := curtime
	for i := 0; i < contentLen; i++ {
		ut += math_rand.Int63n(10000)
		ct = append(ct, uint64(ut))
	}
	wcstf := &wechat.WCSTF{
		StartTime: proto.Uint64(uint64(curtime)),
		CheckTime: proto.Uint64(uint64(curtime)),
		Count:     proto.Uint32(uint32(contentLen)),
		EndTime:   ct,
	}
	ccData, err := proto.Marshal(wcstf)
	if err != nil {
		return nil, err
	}

	afterCompressionCCData := v08.Compress(ccData)
	afterEnData, err := ccdata.EncodeZipData(afterCompressionCCData, 0x3060)
	if err != nil {
		return nil, err
	}

	deviceRunningInfo := &wechat.DeviceRunningInfoNew{
		Version:     []byte("00000008"),
		Type:        proto.Uint32(1),
		EncryptData: afterEnData,
		Timestamp:   proto.Uint32(uint32(curtime)),
		Unknown5:    proto.Uint32(5),
		Unknown6:    proto.Uint32(0),
	}
	return proto.Marshal(deviceRunningInfo)
}

func GetWcste08() ([]byte, error) {
	curtime := time.Now().Unix()
	curNanoTime := time.Now().UnixNano() / 1000000
	wcste := &wechat.WCSTE{
		CheckId:   proto.String("<LoginByID>"),
		StartTime: proto.Uint32(uint32(curtime)),
		CheckTime: proto.Uint32(uint32(curtime)),
		Count1:    proto.Uint32(0),
		Count2:    proto.Uint32(1),
		Count3:    proto.Uint32(0),
		Const1:    proto.Uint64(384214787666497617),
		Const2:    proto.Uint64(uint64(curNanoTime)),
		Const3:    proto.Uint64(uint64(curNanoTime)),
		Const4:    proto.Uint64(uint64(curNanoTime)),
		Const5:    proto.Uint64(uint64(curNanoTime)),
		Const6:    proto.Uint64(384002236977512448),
	}
	ccData, err := proto.Marshal(wcste)
	if err != nil {
		return nil, err
	}

	afterCompressionCCData := v08.Compress(ccData)
	afterEnData, err := ccdata.EncodeZipData(afterCompressionCCData, 0x3060)
	if err != nil {
		return nil, err
	}

	deviceRunningInfo := &wechat.DeviceRunningInfoNew{
		Version:     []byte("00000008"),
		Type:        proto.Uint32(1),
		EncryptData: afterEnData,
		Timestamp:   proto.Uint32(uint32(curtime)),
		Unknown5:    proto.Uint32(5),
		Unknown6:    proto.Uint32(0),
	}
	return proto.Marshal(deviceRunningInfo)
}

// android生成wcstf, 使用01加密
func AndroidWcstf(Username string) []byte {
	curtime := uint64(time.Now().UnixNano() / 1e6)
	contentlen := len(Username)

	var ct []uint64
	ut := curtime
	for i := 0; i < contentlen; i++ {
		ut += uint64(math_rand.Intn(10000))
		ct = append(ct, ut)
	}
	ccd := &mm.Wcstf{
		StartTime: &curtime,
		CheckTime: &curtime,
		Count:     proto.Uint32(uint32(contentlen)),
		EndTime:   ct,
	}

	pb, _ := proto.Marshal(ccd)

	// Zero: 03加密改08加密
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(pb)
	w.Close()

	zt := new(ZT)
	zt.Init()
	encData := zt.Encrypt(b.Bytes())
	// compressData := DoZlibCompress(pb)
	// encData := SaeEncrypt01(compressData)

	Ztdata := &mm.ZTData{
		Version:   proto.String("00000008\x00"),
		Encrypted: proto.Uint32(1),
		Data:      encData,
		TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
		Optype:    proto.Uint32(5),
		Uin:       proto.Uint32(0),
	}
	MS, _ := proto.Marshal(Ztdata)
	return MS
}

// android生成wcste, 使用01加密
func AndroidWcste(A, B uint64) []byte {
	curtime := uint32(time.Now().Unix())
	curNanoTime := uint64(time.Now().UnixNano())
	ccd := &mm.Wcste{
		Checkid:   proto.String("<LoginByID>"),
		StartTime: &curtime,
		CheckTime: &curtime,
		Count1:    proto.Uint32(0),
		Count2:    proto.Uint32(1),
		Count3:    proto.Uint32(0),
		Const1:    proto.Uint64(A),
		Const2:    &curNanoTime,
		Const3:    &curNanoTime,
		Const4:    &curNanoTime,
		Const5:    &curNanoTime,
		Const6:    proto.Uint64(B),
	}

	pb, _ := proto.Marshal(ccd)
	// compressData := DoZlibCompress(pb)
	// encData := SaeEncrypt01(compressData)
	// Zero: 03加密改08加密
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(pb)
	w.Close()

	zt := new(ZT)
	zt.Init()
	encData := zt.Encrypt(b.Bytes())

	Ztdata := &mm.ZTData{
		Version:   proto.String("00000008\x00"),
		Encrypted: proto.Uint32(1),
		Data:      encData,
		TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
		Optype:    proto.Uint32(5),
		Uin:       proto.Uint32(0),
	}

	MS, _ := proto.Marshal(Ztdata)
	return MS
}

func AndroidCcData(DeviceId string, info baseinfo.AndroidDeviceInfo, DeviceToken *mm.TrustResponse) *mm.ZTData {
	curtime := uint32(time.Now().Unix())
	ccd3body := &mm.AndroidSpamDataBody{
		Loc:                  proto.Uint32(0),
		Root:                 proto.Uint32(0),
		Debug:                proto.Uint32(0),
		PackageSign:          proto.String(info.AndriodPackageSign(DeviceId)),
		RadioVersion:         proto.String(info.AndroidRadioVersion(DeviceId)),
		BuildVersion:         proto.String(info.AndroidVersion()),
		DeviceId:             proto.String(info.AndriodImei(DeviceId)),
		AndroidId:            proto.String(info.AndroidBuildID(DeviceId)),
		SerialId:             proto.String(info.AndriodPhoneSerial(DeviceId)),
		Model:                proto.String(info.AndroidPhoneModel(DeviceId)),
		CpuCount:             proto.Uint32(8),
		CpuBrand:             proto.String(info.AndroidHardware(DeviceId)),
		CpuExt:               proto.String(info.AndroidFeatures()),
		WlanAddress:          proto.String(info.AndriodWLanAddress(DeviceId)),
		Ssid:                 proto.String(info.AndriodSsid(DeviceId)),
		Bssid:                proto.String(info.AndriodBssid(DeviceId)),
		SimOperator:          proto.String(""),
		WifiName:             proto.String(info.AndroidWifiName(DeviceId)),
		BuildFP:              proto.String(info.AndroidBuildFP(DeviceId)),
		BuildBoard:           proto.String("bullhead"),
		BuildBootLoader:      proto.String(info.AndroidBuildBoard(DeviceId)),
		BuildBrand:           proto.String("google"),
		BuildDevice:          proto.String("bullhead"),
		GsmSimOperatorNumber: proto.String(""),
		SoterId:              proto.String(""),
		KernelReleaseNumber:  proto.String(info.AndroidKernelReleaseNumber(DeviceId)),
		UsbState:             proto.Uint32(0),
		Sign:                 proto.String(info.AndriodPackageSign(DeviceId)),
		PackageFlag:          proto.Uint32(14),
		AccessFlag:           proto.Uint32(uint32(info.AndriodAccessFlag(DeviceId))),
		Unkonwn:              proto.Uint32(3),
		TbVersionCrc:         proto.Uint32(uint32(info.AndriodTbVersionCrc(DeviceId))),
		SfMD5:                proto.String(info.AndriodSfMD5(DeviceId)),
		SfArmMD5:             proto.String(info.AndriodSfArmMD5(DeviceId)),
		SfArm64MD5:           proto.String(info.AndriodSfArm64MD5(DeviceId)),
		SbMD5:                proto.String(info.AndriodSbMD5(DeviceId)),
		SoterId2:             proto.String(""),
		WidevineDeviceID:     proto.String(info.AndriodWidevineDeviceID(DeviceId)),
		FSID:                 proto.String(info.AndriodFSID(DeviceId)),
		Oaid:                 proto.String(""),
		TimeCheck:            proto.Uint32(0),
		NanoTime:             proto.Uint32(uint32(info.AndriodNanoTime(DeviceId))),
		Refreshtime:          proto.Uint32(DeviceToken.GetTrustResponseData().GetTimeStamp()),
		SoftConfig:           proto.String(DeviceToken.GetTrustResponseData().GetSoftData().GetSoftConfig()),
		SoftData:             DeviceToken.GetTrustResponseData().GetSoftData().GetSoftData(),
	}

	pb, _ := proto.Marshal(ccd3body)

	crc := crc32.ChecksumIEEE(pb)

	ccd3 := &mm.AndroidCcdDataBody{
		Crc:       &crc,
		TimeStamp: &curtime,
		Body:      ccd3body,
	}

	pb, _ = proto.Marshal(ccd3)

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(pb)
	w.Close()

	zt := new(ZT)
	zt.Init()
	encData := zt.Encrypt(b.Bytes())

	randKey := make([]byte, 16)
	io.ReadFull(rand.Reader, randKey)
	// compressData := DoZlibCompress(pb)
	// encData := SaeEncrypt01(compressData)

	Ztdata := &mm.ZTData{
		Version:   proto.String("00000008\x00"),
		Encrypted: proto.Uint32(1),
		Data:      encData,
		TimeStamp: &curtime,
		Optype:    proto.Uint32(5),
		Uin:       proto.Uint32(0),
	}
	return Ztdata
}
