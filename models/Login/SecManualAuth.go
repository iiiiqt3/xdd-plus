package Login

import (
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/Mmtls"
	"wechatdll/baseinfo"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/clientsdk/extinfo"
	"wechatdll/comm"

	"github.com/golang/protobuf/proto"
)

func SecManualAuth(Data *comm.LoginData) (mm.UnifyAuthResponse, []byte, []byte, []byte, *mm.TrustResponse, error) {
	hec := InitHec(Data)
	if hec.IsAndroid {
		return SecManualAuthAndroid(Data)
	}
	prikey, pubkey := Algorithm.GetEcdh713Key()
	if Data.ShortHost == "" {
		Data.ShortHost = Algorithm.MmtlsShortHost
	}

	httpclient := Mmtls.GenNewHttpClient(Data.MmtlsKey, Data.ShortHost)

	Data.Aeskey = []byte(baseutils.RandSeq(16)) //获取随机密钥
	accountRequest := &mm.ManualAuthRsaReqData{
		RandomEncryKey: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(Data.Aeskey))),
			Buffer: Data.Aeskey,
		},
		CliPubEcdhkey: &mm.ECDHKey{
			Nid: proto.Int32(713),
			Key: &mm.SKBuiltinBufferT{
				ILen:   proto.Uint32(uint32(len(pubkey))),
				Buffer: pubkey,
			},
		},
		UserName: &Data.Wxid,
		Pwd:      &Data.Pwd,
		Pwd2:     &Data.Pwd,
	}

	Wcstf, _ := Algorithm.GetWcstf08(Data.Wxid)
	Wcste, _ := Algorithm.GetWcste08()
	EncryptData := extinfo.GetNewSpamDataV8(Data)
	ccData := &mm.CryptoData{
		Version:     []byte("00000008"),
		Type:        proto.Uint32(1),
		EncryptData: EncryptData,
		Timestamp:   proto.Uint32(uint32(time.Now().Unix())),
		Unknown5:    proto.Uint32(5),
		Unknown6:    proto.Uint32(0),
	}

	ccDataseq, _ := proto.Marshal(ccData)

	DeviceTokenCCD := &mm.DeviceToken{
		Version:   proto.String(""),
		Encrypted: proto.Uint32(1),
		Data: &mm.SKBuiltinStringT{
			String_: proto.String(Data.DeviceToken.GetTrustResponseData().GetDeviceToken()),
		},
		TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
		Optype:    proto.Uint32(2),
		Uin:       proto.Uint32(0),
	}
	DeviceTokenCCDPB, _ := proto.Marshal(DeviceTokenCCD)

	WCExtInfo := &mm.WCExtInfo{
		Wcstf: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(Wcstf))),
			Buffer: Wcstf,
		},
		Wcste: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(Wcste))),
			Buffer: Wcste,
		},
		CcData: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(ccDataseq))),
			Buffer: ccDataseq,
		},
		DeviceToken: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(DeviceTokenCCDPB))),
			Buffer: DeviceTokenCCDPB,
		},
	}

	WCExtInfoseq, _ := proto.Marshal(WCExtInfo)
	ClientSeqId := baseutils.GetClientSeqId(Data.Deviceid_str)
	Imei := baseinfo.IOSImei(Data.Deviceid_str)
	// TODO: 放到初始化上下文中生成
	SoftType := Data.SoftType
	if SoftType == "" {
		SoftType = baseinfo.SoftType_iPad(Data.Deviceid_str, Data.OsVersion, Data.RomModel)
	}
	uuid1, _ := baseinfo.IOSUuid(Data.Deviceid_str)

	deviceRequest := &mm.ManualAuthAesReqData{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    Data.Aeskey,
			Uin:           proto.Uint32(0),
			DeviceId:      Data.Deviceid_byte,
			ClientVersion: proto.Int32(int32(Data.ClientVersion)),
			DeviceType:    []byte(Data.DeviceType),
			Scene:         proto.Uint32(1),
		},
		BaseReqInfo:  &mm.BaseAuthReqInfo{},
		Imei:         &Imei,
		SoftType:     &SoftType,
		BuiltinIpseq: proto.Uint32(0),
		ClientSeqId:  &ClientSeqId,
		DeviceName:   proto.String(Data.DeviceName),
		DeviceType:   proto.String("iPad"),
		Language:     proto.String("zh_CN"),
		TimeZone:     proto.String("8.0"),
		Channel:      proto.Int(0),
		TimeStamp:    proto.Uint32(uint32(time.Now().Unix())),
		DeviceBrand:  proto.String("Apple"),
		Ostype:       proto.String(Data.DeviceType),
		RealCountry:  proto.String("CN"),
		BundleId:     proto.String("com.tencent.xin"),
		AdSource:     &uuid1,
		IphoneVer:    proto.String(Algorithm.IPadModel),
		InputType:    proto.Uint32(2),
		ExtSpamInfo: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(WCExtInfoseq))),
			Buffer: WCExtInfoseq,
		},
	}

	// accountReqData, err := proto.Marshal(accountRequest)
	// log.Println("account: " + hex.EncodeToString(accountReqData))
	// deviceReqData, err := proto.Marshal(deviceRequest)
	// log.Println("device: " + hex.EncodeToString(deviceReqData))

	requset := &mm.SecManualLoginRequest{
		RsaReqData: accountRequest,
		AesReqData: deviceRequest,
	}
	reqdata, err := proto.Marshal(requset)
	// log.Println(hex.EncodeToString(reqdata))
	// hec := InitHec(Data)
	hec = &Algorithm.Client{}
	hec.Init("IOS")
	hecData := hec.HybridEcdhPackIosEn(252, 0, nil, reqdata)

	recvData, _ := httpclient.MMtlsPost(Data.ShortHost, "/cgi-bin/micromsg-bin/secmanualauth", hecData, Data.Proxy)
	ph1 := hec.HybridEcdhPackIosUn(recvData)
	loginRes := mm.UnifyAuthResponse{}
	err = proto.Unmarshal(ph1.Data, &loginRes)

	if err != nil {
		return mm.UnifyAuthResponse{}, nil, nil, nil, &mm.TrustResponse{}, err
	}

	return loginRes, prikey, pubkey, ph1.Cookies, &mm.TrustResponse{}, nil
}

func SecManualAuthAndroid(Data *comm.LoginData) (mm.UnifyAuthResponse, []byte, []byte, []byte, *mm.TrustResponse, error) {
	prikey, pubkey := Algorithm.GetEcdh713Key()
	if Data.ShortHost == "" {
		Data.ShortHost = Algorithm.MmtlsShortHost
	}
	httpclient := Mmtls.GenNewHttpClient(Data.MmtlsKey, Data.ShortHost)

	Data.Aeskey = []byte(baseutils.RandSeq(16)) //获取随机密钥
	accountRequest := &mm.ManualAuthRsaReqData{
		RandomEncryKey: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(Data.Aeskey))),
			Buffer: Data.Aeskey,
		},
		CliPubEcdhkey: &mm.ECDHKey{
			Nid: proto.Int32(713),
			Key: &mm.SKBuiltinBufferT{
				ILen:   proto.Uint32(uint32(len(pubkey))),
				Buffer: pubkey,
			},
		},
		UserName: &Data.Wxid,
		Pwd:      &Data.Pwd,
		Pwd2:     &Data.Pwd,
	}
	Wcstf, _ := Algorithm.GetWcstf08(Data.Wxid)
	Wcste, _ := Algorithm.GetWcste08()
	EncryptData := extinfo.GetiPhoneNewSpamData(Data)
	ccData := &mm.CryptoData{
		Version:     []byte("00000008"),
		Type:        proto.Uint32(1),
		EncryptData: EncryptData,
		Timestamp:   proto.Uint32(uint32(time.Now().Unix())),
		Unknown5:    proto.Uint32(5),
		Unknown6:    proto.Uint32(0),
	}

	ccDataseq, _ := proto.Marshal(ccData)

	DeviceTokenCCD := &mm.DeviceToken{
		Version:   proto.String(""),
		Encrypted: proto.Uint32(1),
		Data: &mm.SKBuiltinStringT{
			String_: proto.String(Data.DeviceToken.GetTrustResponseData().GetDeviceToken()),
		},
		TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
		Optype:    proto.Uint32(2),
		Uin:       proto.Uint32(0),
	}
	DeviceTokenCCDPB, _ := proto.Marshal(DeviceTokenCCD)

	WCExtInfo := &mm.WCExtInfo{
		Wcstf: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(Wcstf))),
			Buffer: Wcstf,
		},
		Wcste: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(Wcste))),
			Buffer: Wcste,
		},
		CcData: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(ccDataseq))),
			Buffer: ccDataseq,
		},
		DeviceToken: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(DeviceTokenCCDPB))),
			Buffer: DeviceTokenCCDPB,
		},
	}
	WCExtInfoseq, _ := proto.Marshal(WCExtInfo)
	ClientSeqId := baseutils.GetClientSeqId(Data.Deviceid_str)
	Imei := baseinfo.IOSImei(Data.Deviceid_str)
	// TODO: 放到初始化上下文中生成
	SoftType := baseinfo.SoftType_iPad(Data.Deviceid_str, Data.OsVersion, Data.RomModel)
	uuid1, _ := baseutils.IOSUuid(Data.Deviceid_str)

	deviceRequest := &mm.ManualAuthAesReqData{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    Data.Aeskey,
			Uin:           proto.Uint32(0),
			DeviceId:      Data.Deviceid_byte,
			ClientVersion: proto.Int32(int32(Data.ClientVersion)),
			DeviceType:    []byte(Data.DeviceType),
			Scene:         proto.Uint32(1),
		},
		BaseReqInfo:  &mm.BaseAuthReqInfo{},
		Imei:         &Imei,
		SoftType:     &SoftType,
		BuiltinIpseq: proto.Uint32(0),
		ClientSeqId:  &ClientSeqId,
		DeviceName:   proto.String(Data.DeviceName),
		DeviceType:   proto.String("pad-android-31"),
		Language:     proto.String("zh_CN"),
		TimeZone:     proto.String("8.0"),
		Channel:      proto.Int(0),
		TimeStamp:    proto.Uint32(uint32(time.Now().Unix())),
		DeviceBrand:  proto.String("pad-android-31"),
		Ostype:       proto.String(Data.DeviceType),
		RealCountry:  proto.String("CN"),
		BundleId:     proto.String("com.tencent.xin"),
		AdSource:     &uuid1,
		IphoneVer:    proto.String(Data.RomModel),
		InputType:    proto.Uint32(2),
		ExtSpamInfo: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(WCExtInfoseq))),
			Buffer: WCExtInfoseq,
		},
	}

	requset := &mm.SecManualLoginRequest{
		RsaReqData: accountRequest,
		AesReqData: deviceRequest,
	}
	reqdata, _ := proto.Marshal(requset)
	hec := &Algorithm.Client{}
	hec.Init("IOS")
	hecData := hec.HybridEcdhPackIosEn(252, 0, nil, reqdata)

	recvData, err := httpclient.MMtlsPost(Data.ShortHost, "/cgi-bin/micromsg-bin/secmanualauth", hecData, Data.Proxy)
	if err != nil {
		return mm.UnifyAuthResponse{}, nil, nil, nil, &mm.TrustResponse{}, err
	}
	ph1 := hec.HybridEcdhPackIosUn(recvData)

	loginRes := mm.UnifyAuthResponse{}
	err = proto.Unmarshal(ph1.Data, &loginRes)

	if err != nil {
		return mm.UnifyAuthResponse{}, nil, nil, nil, &mm.TrustResponse{}, err
	}

	return loginRes, prikey, pubkey, ph1.Cookies, &mm.TrustResponse{}, nil
}
