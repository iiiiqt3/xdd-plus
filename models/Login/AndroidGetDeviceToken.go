package Login

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"fmt"
	"io"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/Mmtls"
	"wechatdll/comm"

	"github.com/golang/protobuf/proto"
)

// Android 设备刷新 deviceToken
func RrefreshTokenAndroid(tmpUserInfo *comm.LoginData, httpclient *Mmtls.HttpClientModel) {
	//这里有个刷新逻辑
	key := fmt.Sprintf("%s%s", "wechat:deviceId:", tmpUserInfo.Deviceid_str)
	exists := comm.Exists(key)
	if exists {
		return
	}
	ShortHost := tmpUserInfo.ShortHost
	if ShortHost == "" {
		tmpUserInfo.ShortHost = Algorithm.MmtlsShortHost
	}
	deviceTokenRsp, err := AndroidGetDeviceToken(tmpUserInfo, httpclient)
	if err != nil {
		fmt.Println("Android 请求 deviceTokenRequest error!")
	} else {
		fmt.Println("Android 请求 deviceTokenRequest ok!", deviceTokenRsp.GetTrustResponseData().GetDeviceToken())
		//保存5天
		comm.SETExpirationObj(key, &deviceTokenRsp, 60*60*24*5)
		tmpUserInfo.DeviceToken = deviceTokenRsp
	}
}

// Android 设备注册刷新deviceToken
func AndroidInitAndRrefresh(tmpUserInfo *comm.LoginData, httpclient *Mmtls.HttpClientModel) {
	//这里有个刷新逻辑
	key := fmt.Sprintf("%s%s", "wechat:deviceId:", tmpUserInfo.Deviceid_str)
	exists := comm.Exists(key)

	trustRes := &mm.TrustResponse{}
	if exists {
		//ios存redis
		_ = comm.GETObj(key, &trustRes)
	}
	// 判断出错
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("AndroidInitAndRrefresh from panic: %v\n", r)
			tmpUserInfo.DeviceToken = trustRes
		}
	}()
	deviceTokenRsp, err := AndroidGetDeviceToken(tmpUserInfo, httpclient)
	if err != nil {
		fmt.Println("Android 请求 deviceTokenRequest error!")
	} else {
		fmt.Println("Android 请求 deviceTokenRequest ok!", deviceTokenRsp.GetTrustResponseData().GetDeviceToken())
		//保存5天
		comm.SETExpirationObj(key, &deviceTokenRsp, 60*60*24*5)
		tmpUserInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.RefreshTokenDate = time.Now().Unix()
	}
}

// 安卓设备刷新 token
func AndroidGetDeviceToken(tmpUserInfo *comm.LoginData, httpclient *Mmtls.HttpClientModel) (*mm.TrustResponse, error) {
	info := tmpUserInfo.DeviceInfoA16
	td := &mm.TrustReq{
		Td: &mm.TrustData{
			Tdi: []*mm.TrustDeviceInfo{
				{Key: proto.String("IMEI"), Val: proto.String(info.AndriodImei(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("AndroidID"), Val: proto.String(info.AndriodID(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("PhoneSerial"), Val: proto.String(info.AndriodPhoneSerial(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("cid"), Val: proto.String("")},
				{Key: proto.String("WidevineDeviceID"), Val: proto.String(info.AndriodWidevineDeviceID(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("WidevineProvisionID"), Val: proto.String(info.AndriodWidevineProvisionID(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("GSFID"), Val: proto.String("")},
				{Key: proto.String("SoterID"), Val: proto.String("")},
				{Key: proto.String("SoterUid"), Val: proto.String("")},
				{Key: proto.String("FSID"), Val: proto.String(info.AndriodFSID(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("BootID"), Val: proto.String("")},
				{Key: proto.String("IMSI"), Val: proto.String("")},
				{Key: proto.String("PhoneNum"), Val: proto.String("")},
				{Key: proto.String("WeChatInstallTime"), Val: proto.String("1730105747")}, //1730105747
				{Key: proto.String("PhoneModel"), Val: proto.String(info.AndroidPhoneModel(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("BuildBoard"), Val: proto.String("bullhead")},
				{Key: proto.String("BuildBootloader"), Val: proto.String(info.AndroidBuildBoard(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("SystemBuildDate"), Val: proto.String("Fri Sep 28 23:37:27 UTC 2024")},
				{Key: proto.String("SystemBuildDateUTC"), Val: proto.String("1730103286")},
				{Key: proto.String("BuildFP"), Val: proto.String(info.AndroidBuildFP(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("BuildID"), Val: proto.String(info.AndroidBuildID(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("BuildBrand"), Val: proto.String("HUAWEI")},
				{Key: proto.String("BuildDevice"), Val: proto.String("bullhead")},
				{Key: proto.String("BuildProduct"), Val: proto.String("bullhead")},
				{Key: proto.String("Manufacturer"), Val: proto.String(info.AndroidManufacturer(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("RadioVersion"), Val: proto.String(info.AndroidRadioVersion(tmpUserInfo.Deviceid_str))},
				{Key: proto.String("AndroidVersion"), Val: proto.String(info.AndroidVersion())},
				{Key: proto.String("SdkIntVersion"), Val: proto.String("34")},
				{Key: proto.String("ScreenWidth"), Val: proto.String("1080")},
				{Key: proto.String("ScreenHeight"), Val: proto.String("1794")},
				{Key: proto.String("SensorList"), Val: proto.String("BMI160 accelerometer#Bosch#0.004788#1,BMI160 gyroscope#Bosch#0.000533#1,BMM150 magnetometer#Bosch#0.000000#1,BMP280 pressure#Bosch#0.005000#1,BMP280 temperature#Bosch#0.010000#1,RPR0521 Proximity Sensor#Rohm#1.000000#1,RPR0521 Light Sensor#Rohm#10.000000#1,Orientation#Google#1.000000#1,BMI160 Step detector#Bosch#1.000000#1,Significant motion#Google#1.000000#1,Gravity#Google#1.000000#1,Linear Acceleration#Google#1.000000#1,Rotation Vector#Google#1.000000#1,Geomagnetic Rotation Vector#Google#1.000000#1,Game Rotation Vector#Google#1.000000#1,Pickup Gesture#Google#1.000000#1,Tilt Detector#Google#1.000000#1,BMI160 Step counter#Bosch#1.000000#1,BMM150 magnetometer (uncalibrated)#Bosch#0.000000#1,BMI160 gyroscope (uncalibrated)#Bosch#0.000533#1,Sensors Sync#Google#1.000000#1,Double Twist#Google#1.000000#1,Double Tap#Google#1.000000#1,Device Orientation#Google#1.000000#1,BMI160 accelerometer (uncalibrated)#Bosch#0.004788#1")},
				{Key: proto.String("DefaultInputMethod"), Val: proto.String("com.google.android.inputmethod.latin")},
				{Key: proto.String("InputMethodList"), Val: proto.String("Google \345\215\260\345\272\246\350\257\255\351\224\256\347\233\230#com.google.android.apps.inputmethod.hindi,Google \350\257\255\351\237\263\350\276\223\345\205\245#com.google.android.googlequicksearchbox,Google \346\227\245\350\257\255\350\276\223\345\205\245\346\263\225#com.google.android.inputmethod.japanese,Google \351\237\251\350\257\255\350\276\223\345\205\245\346\263\225#com.google.android.inputmethod.korean,Gboard#com.google.android.inputmethod.latin,\350\260\267\346\255\214\346\213\274\351\237\263\350\276\223\345\205\245\346\263\225#com.google.android.inputmethod.pinyin")},
				{Key: proto.String("DeviceID"), Val: proto.String(tmpUserInfo.Deviceid_str)},
				{Key: proto.String("OAID"), Val: proto.String("")},
			},
		},
	}

	pb, _ := proto.Marshal(td)

	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(pb)
	w.Close()

	zt := new(Algorithm.ZT)
	zt.Init()
	encData := zt.Encrypt(b.Bytes())

	randKey := make([]byte, 16)
	io.ReadFull(rand.Reader, randKey)

	fp := &mm.FPFresh{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    []byte{},
			Uin:           proto.Uint32(0),
			DeviceId:      append([]byte(tmpUserInfo.Deviceid_str), 0),
			ClientVersion: proto.Int32(int32(Algorithm.AndroidVersion)),
			DeviceType:    []byte(Algorithm.AndroidDeviceType),
			Scene:         proto.Uint32(0),
		},
		SessKey: randKey,
		Ztdata: &mm.ZTData{
			Version:   proto.String("00000008\x00"),
			Encrypted: proto.Uint32(1),
			Data:      encData,
			TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
			Optype:    proto.Uint32(5),
			Uin:       proto.Uint32(0),
		},
	}

	reqdata, _ := proto.Marshal(fp)

	hec := &Algorithm.Client{}
	hec.Init("Android")
	hecData := hec.HybridEcdhPackAndroidEn(3789, 10002, 0, nil, reqdata)
	recvData, err := httpclient.MMtlsPost(tmpUserInfo.ShortHost, "/cgi-bin/micromsg-bin/fpinitnl", hecData, tmpUserInfo.Proxy)
	if err != nil {
		return &mm.TrustResponse{}, err
	}
	ph := hec.HybridEcdhPackAndroidUn(recvData)
	DTResp := &mm.TrustResponse{}
	_ = proto.Unmarshal(ph.Data, DTResp)
	return DTResp, nil
}
