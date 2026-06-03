package Login

import (
	"container/list"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/comm"
	"wechatdll/models"

	"github.com/golang/protobuf/proto"
)

type A16LoginParam struct {
	UserName   string
	Password   string
	A16        string
	DeviceName string
	Proxy      models.ProxyInfo
}

func AndroidA16Login(Data A16LoginParam, domain string) models.ResponseResult {
	// 获取username为key的缓存
	D, _ := comm.GetLoginata(Data.UserName, nil)
	reqDataLogin := DataLogin{
		UserName:      Data.UserName,
		Password:      Data.Password,
		A16:           Data.A16,
		DeviceName:    Data.DeviceName,
		DeviceId:      "",
		Proxy:         Data.Proxy,
		ClientVersion: Algorithm.AndroidVersion,
	}
	if D == nil || D.Wxid == "" || D.ClientVersion != Algorithm.AndroidVersion {
		// 没有缓存, 初始化新的账号环境
		D = GenAndroidLoginData(reqDataLogin)
	} else {
		D = UpdateAndroidLoginData(D, reqDataLogin)
	}
	if domain == "" {
		domain = D.ShortHost
	}

	//初始化Mmtls
	httpclient, MmtlsClient, err := comm.MmtlsInitialize(Data.Proxy, domain)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("MMTLS初始化失败：%v", err.Error()),
			Data:    nil,
		}
	}

	//获取DeviceToken
	AndroidInitAndRrefresh(D, httpclient)
	DeviceInfoA16 := D.GetDeviceInfoA16()

	passwordhash := md5.Sum([]byte(Data.Password))
	prikey, pubkey := Algorithm.GetEcdh713Key()

	Wcstf := Algorithm.AndroidWcstf(Data.UserName)
	Wcste := Algorithm.AndroidWcste(384214787666497617, 384002236977512448)
	AndroidCcData := Algorithm.AndroidCcData(Data.A16, *D.GetDeviceInfoA16(), D.DeviceToken)
	CcData3PB, _ := proto.Marshal(AndroidCcData)

	curtime := uint32(time.Now().Unix())
	DeviceTokenCCD := &mm.DeviceToken{
		Version:   proto.String(""),
		Encrypted: proto.Uint32(1),
		Data: &mm.SKBuiltinStringT{
			String_: proto.String(D.DeviceToken.GetTrustResponseData().GetDeviceToken()),
		},
		TimeStamp: &curtime,
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
			ILen:   proto.Uint32(uint32(len(CcData3PB))),
			Buffer: CcData3PB,
		},
		DeviceToken: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(DeviceTokenCCDPB))),
			Buffer: DeviceTokenCCDPB,
		},
	}

	WCExtInfoPB, _ := proto.Marshal(WCExtInfo)

	aeskey := []byte(baseutils.RandSeq(16))

	secmanualauth := &mm.SecManualLoginRequest{
		RsaReqData: &mm.ManualAuthRsaReqData{
			RandomEncryKey: &mm.SKBuiltinBufferT{
				ILen:   proto.Uint32(uint32(len(aeskey))),
				Buffer: aeskey,
			},
			CliPubEcdhkey: &mm.ECDHKey{
				Nid: proto.Int32(713),
				Key: &mm.SKBuiltinBufferT{
					ILen:   proto.Uint32(uint32(len(pubkey))),
					Buffer: pubkey,
				},
			},
			UserName: proto.String(Data.UserName),
			Pwd:      proto.String(hex.EncodeToString(passwordhash[:])),
			Pwd2:     proto.String(hex.EncodeToString(passwordhash[:])),
		},
		AesReqData: &mm.ManualAuthAesReqData{
			BaseRequest: &mm.BaseRequest{
				SessionKey:    []byte{},
				Uin:           proto.Uint32(0),
				DeviceId:      D.Deviceid_byte,
				ClientVersion: proto.Int32(D.ClientVersion),
				DeviceType:    []byte(D.DeviceType),
				Scene:         proto.Uint32(1),
			},
			Imei:         proto.String(DeviceInfoA16.AndriodImei(Data.A16)),
			SoftType:     proto.String(DeviceInfoA16.AndriodGetSoftType(Data.A16)),
			BuiltinIpseq: proto.Uint32(0),
			ClientSeqId:  proto.String(fmt.Sprintf("%s_%d", Data.A16, (time.Now().UnixNano() / 1e6))),
			Signature:    proto.String(DeviceInfoA16.AndriodPackageSign(Data.A16)),
			DeviceName:   proto.String(DeviceInfoA16.AndroidManufacturer(Data.A16) + "-" + DeviceInfoA16.AndroidPhoneModel(Data.A16)),
			DeviceType:   proto.String(DeviceInfoA16.AndriodDeviceType(Data.A16)),
			Language:     proto.String("zh_CN"),
			TimeZone:     proto.String("8.0"),
			Channel:      proto.Int32(0),
			TimeStamp:    proto.Uint32(0),
			DeviceBrand:  proto.String("HUAWEI"),
			DeviceModel:  proto.String(DeviceInfoA16.AndroidPhoneModel(Data.A16) + DeviceInfoA16.AndroidArch(Data.A16)),
			Ostype:       proto.String(D.DeviceType),
			RealCountry:  proto.String(""),
			InputType:    proto.Uint32(2),
			ExtSpamInfo: &mm.SKBuiltinBufferT{
				ILen:   proto.Uint32(uint32(len(WCExtInfoPB))),
				Buffer: WCExtInfoPB,
			},
		},
	}

	reqdata, _ := proto.Marshal(secmanualauth)
	hec := InitHec(D)
	hecData := hec.HybridEcdhPackAndroidEn(252, 10002, 0, nil, reqdata)
	recvData, err := httpclient.MMtlsPost(domain, "/cgi-bin/micromsg-bin/secmanualauth", hecData, Data.Proxy)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}
	ph1 := hec.HybridEcdhPackAndroidUn(recvData)
	loginRes := mm.UnifyAuthResponse{}
	err = proto.Unmarshal(ph1.Data, &loginRes)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}

	if loginRes.GetBaseResponse().GetRet() == 0 {
		var LoginData = D
		LoginData.Cooike = ph1.Cookies
		LoginData.ShortHost = domain
		LoginData.MmtlsKey = MmtlsClient
		ecdhkey := Algorithm.DoECDH713(loginRes.GetAuthSectResp().GetSvrPubEcdhkey().GetKey().GetBuffer(), prikey)
		LoginData.Loginecdhkey = ecdhkey
		LoginData.Uin = loginRes.GetAuthSectResp().GetUin()
		LoginData.Wxid = loginRes.GetAcctSectResp().GetUserName()
		LoginData.Alais = loginRes.GetAcctSectResp().GetAlias()
		LoginData.Mobile = loginRes.GetAcctSectResp().GetBindMobile()
		LoginData.NickName = loginRes.GetAcctSectResp().GetNickName()
		LoginData.Sessionkey = Algorithm.AESDecrypt(loginRes.GetAuthSectResp().GetSessionKey().GetBuffer(), ecdhkey)
		LoginData.Sessionkey_2 = loginRes.GetAuthSectResp().GetSessionKey().GetBuffer()
		LoginData.Autoauthkey = loginRes.GetAuthSectResp().GetAutoAuthKey().GetBuffer()
		LoginData.Autoauthkeylen = int32(loginRes.GetAuthSectResp().GetAutoAuthKey().GetILen())
		LoginData.Serversessionkey = loginRes.GetAuthSectResp().GetServerSessionKey().GetBuffer()
		LoginData.Clientsessionkey = loginRes.GetAuthSectResp().GetClientSessionKey().GetBuffer()
		LoginData.RsaPublicKey = pubkey
		LoginData.RsaPrivateKey = prikey

		err := comm.CreateLoginData(LoginData, LoginData.Wxid, 0, nil)

		if err != nil {
			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: fmt.Sprintf("系统异常：%v", err.Error()),
				Data:    nil,
			}
		}

		return models.ResponseResult{
			Code:    0,
			Success: true,
			Message: "成功",
			Data:    &loginRes,
		}
	}

	//30系列转向
	if loginRes.GetBaseResponse().GetRet() == -301 {
		var Wx_newLongIPlist, Wx_newshortIplist, Wx_newshortextipList list.List
		var Wx_newLong_Host, Wx_newshort_Host, Wx_newshortext_Host list.List

		dns_info := loginRes.GetNetworkSectResp().GetNewHostList().GetList()
		for _, v := range dns_info {
			if v.GetHost() == "long.weixin.qq.com" {
				ip_info := loginRes.GetNetworkSectResp().GetBuiltinIplist().GetLongConnectIplist()
				for _, ip := range ip_info {
					host := ip.GetHost()
					host = strings.Replace(host, string(byte(0x00)), "", -1)
					if host == v.GetRedirect() {
						ipaddr := ip.GetIp()
						ipaddr = strings.Replace(ipaddr, string(byte(0x00)), "", -1)
						Wx_newLongIPlist.PushBack(ipaddr)
						Wx_newLong_Host.PushBack(host)
					}
				}
			} else if v.GetHost() == "short.weixin.qq.com" {
				ip_info := loginRes.GetNetworkSectResp().GetBuiltinIplist().GetShortConnectIplist()
				for _, ip := range ip_info {
					host := ip.GetHost()
					host = strings.Replace(host, string(byte(0x00)), "", -1)
					if host == v.GetRedirect() {
						ipaddr := ip.GetIp()
						ipaddr = strings.Replace(ipaddr, string(byte(0x00)), "", -1)
						Wx_newshortIplist.PushBack(ipaddr)
						Wx_newshort_Host.PushBack(host)
					}
				}
			} else if v.GetHost() == "extshort.weixin.qq.com" {
				ip_info := loginRes.GetNetworkSectResp().GetBuiltinIplist().GetShortConnectIplist()
				for _, ip := range ip_info {
					host := ip.GetHost()
					host = strings.Replace(host, string(byte(0x00)), "", -1)
					if host == v.GetRedirect() {
						ipaddr := ip.GetIp()
						ipaddr = strings.Replace(ipaddr, string(byte(0x00)), "", -1)
						Wx_newshortextipList.PushBack(ipaddr)
						Wx_newshortext_Host.PushBack(host)
					}
				}
			}
		}
		return AndroidA16Login(Data, Wx_newshort_Host.Front().Value.(string))
	}
	/*
		// 自动过滑块
		if strings.Index(loginRes.GetBaseResponse().ErrMsg.String(), "环境存在异常") >= 0 {
			// 过滑块并根据结果判断
			if err := LoginOCR(loginRes.GetBaseResponse().ErrMsg.String()); err == nil {
				// 滑块成功, 再次登录
				return AndroidA16Login(Data, domain)
			} else {
				// 返回异常
				return models.ResponseResult{
					Code:    -8,
					Success: false,
					Message: fmt.Sprintf("系统异常：%v", err.Error()),
					Data:    nil,
				}
			}
		}
	*/
	//活号
	fmt.Println(loginRes.GetBaseResponse().ErrMsg.String())
	if strings.Index(loginRes.GetBaseResponse().ErrMsg.String(), "环境存在异常") >= 0 {
		// 过滑块并根据结果判断
		if err := LoginOCR(loginRes.GetBaseResponse().ErrMsg.String()); err == nil {
			// 滑块成功, 再次登录
			return AndroidA16Login(Data, domain)
		} else {
			// 返回异常
			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: fmt.Sprintf("系统异常：%v", err.Error()),
				Data:    nil,
			}
		}
	}
	//死号
	if strings.Index(loginRes.GetBaseResponse().ErrMsg.String(), "ticket") >= 0 {
		// 过滑块并根据结果判断
		if err := LoginOCRS(loginRes.GetBaseResponse().ErrMsg.String()); err == nil {
			// 滑块成功, 再次登录
			return AndroidA16Login(Data, domain)
		} else {
			// 返回异常
			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: fmt.Sprintf("系统异常：%v", err.Error()),
				Data:    nil,
			}
		}
	}

	return models.ResponseResult{
		Code:    -8,
		Success: false,
		Message: "失败",
		Data:    &loginRes,
	}
}

func AndroidA16Login1(Data A16LoginParam, domain string) models.ResponseResult {

	// 获取username为key的缓存
	D, _ := comm.GetLoginata(Data.UserName, nil)
	reqDataLogin := DataLogin{
		UserName:      Data.UserName,
		Password:      Data.Password,
		A16:           Data.A16,
		DeviceName:    Data.DeviceName,
		DeviceId:      "",
		Proxy:         Data.Proxy,
		ClientVersion: Algorithm.AndroidVersion1,
	}
	if D == nil || D.Wxid == "" || D.ClientVersion != Algorithm.AndroidVersion1 {
		// 没有缓存, 初始化新的账号环境
		D = GenAndroidLoginData(reqDataLogin)
	} else {
		D = UpdateAndroidLoginData(D, reqDataLogin)
	}
	if domain == "" {
		domain = D.ShortHost
	}
	//初始化Mmtls
	httpclient, MmtlsClient, err := comm.MmtlsInitialize(Data.Proxy, domain)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("MMTLS初始化失败：%v", err.Error()),
			Data:    nil,
		}
	}

	//获取DeviceToken
	AndroidInitAndRrefresh(D, httpclient)
	DeviceInfoA16 := D.GetDeviceInfoA16()

	Deviceid := []byte(Data.A16[:15])
	passwordhash := md5.Sum([]byte(Data.Password))
	prikey, pubkey := Algorithm.GetEcdh713Key()

	Wcstf := Algorithm.AndroidWcstf(Data.UserName)
	Wcste := Algorithm.AndroidWcste(384214787666497617, 384002236977512448)
	AndroidCcData := Algorithm.AndroidCcData(Data.A16, *DeviceInfoA16, D.DeviceToken)
	CcData3PB, _ := proto.Marshal(AndroidCcData)

	curtime := uint32(time.Now().Unix())
	DeviceTokenCCD := &mm.DeviceToken{
		Version:   proto.String(""),
		Encrypted: proto.Uint32(1),
		Data: &mm.SKBuiltinStringT{
			String_: proto.String(D.DeviceToken.GetTrustResponseData().GetDeviceToken()),
		},
		TimeStamp: &curtime,
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
			ILen:   proto.Uint32(uint32(len(CcData3PB))),
			Buffer: CcData3PB,
		},
		DeviceToken: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(DeviceTokenCCDPB))),
			Buffer: DeviceTokenCCDPB,
		},
	}

	WCExtInfoPB, _ := proto.Marshal(WCExtInfo)

	aeskey := []byte(baseutils.RandSeq(16))

	secmanualauth := &mm.SecManualLoginRequest{
		RsaReqData: &mm.ManualAuthRsaReqData{
			RandomEncryKey: &mm.SKBuiltinBufferT{
				ILen:   proto.Uint32(uint32(len(aeskey))),
				Buffer: aeskey,
			},
			CliPubEcdhkey: &mm.ECDHKey{
				Nid: proto.Int32(713),
				Key: &mm.SKBuiltinBufferT{
					ILen:   proto.Uint32(uint32(len(pubkey))),
					Buffer: pubkey,
				},
			},
			UserName: proto.String(Data.UserName),
			Pwd:      proto.String(hex.EncodeToString(passwordhash[:])),
			Pwd2:     proto.String(hex.EncodeToString(passwordhash[:])),
		},
		AesReqData: &mm.ManualAuthAesReqData{
			BaseRequest: &mm.BaseRequest{
				SessionKey:    []byte{},
				Uin:           proto.Uint32(0),
				DeviceId:      D.Deviceid_byte,
				ClientVersion: proto.Int32(int32(Algorithm.AndroidVersion1)),
				DeviceType:    []byte(Algorithm.AndroidDeviceType),
				Scene:         proto.Uint32(1),
			},
			Imei:         proto.String(DeviceInfoA16.AndriodImei(Data.A16)),
			SoftType:     proto.String(DeviceInfoA16.AndriodGetSoftType(Data.A16)),
			BuiltinIpseq: proto.Uint32(0),
			ClientSeqId:  proto.String(fmt.Sprintf("%s_%d", Data.A16, (time.Now().UnixNano() / 1e6))),
			Signature:    proto.String(DeviceInfoA16.AndriodPackageSign(Data.A16)),
			DeviceName:   proto.String(DeviceInfoA16.AndroidManufacturer(Data.A16) + "-" + DeviceInfoA16.AndroidPhoneModel(Data.A16)),
			DeviceType:   proto.String(DeviceInfoA16.AndriodDeviceType(Data.A16)),
			Language:     proto.String("zh_CN"),
			TimeZone:     proto.String("8.00"),
			Channel:      proto.Int32(0),
			TimeStamp:    proto.Uint32(0),
			DeviceBrand:  proto.String("google"),
			DeviceModel:  proto.String(DeviceInfoA16.AndroidPhoneModel(Data.A16) + DeviceInfoA16.AndroidArch(Data.A16)),
			Ostype:       proto.String(Algorithm.AndroidDeviceType),
			RealCountry:  proto.String(""),
			InputType:    proto.Uint32(2),
			ExtSpamInfo: &mm.SKBuiltinBufferT{
				ILen:   proto.Uint32(uint32(len(WCExtInfoPB))),
				Buffer: WCExtInfoPB,
			},
		},
	}

	reqdata, _ := proto.Marshal(secmanualauth)
	hec := InitHec(D)
	hecData := hec.HybridEcdhPackAndroidEn(252, 10002, 0, nil, reqdata)
	recvData, err := httpclient.MMtlsPost(domain, "/cgi-bin/micromsg-bin/secmanualauth", hecData, Data.Proxy)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}
	ph1 := hec.HybridEcdhPackAndroidUn(recvData)
	loginRes := mm.UnifyAuthResponse{}
	err = proto.Unmarshal(ph1.Data, &loginRes)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}

	if loginRes.GetBaseResponse().GetRet() == 0 {
		var LoginData comm.LoginData
		LoginData.Cooike = ph1.Cookies
		LoginData.ShortHost = domain
		LoginData.Deviceid_str = Data.A16
		LoginData.Deviceid_byte = Deviceid
		LoginData.MmtlsKey = MmtlsClient
		LoginData.ClientVersion = Algorithm.AndroidVersion1
		LoginData.DeviceType = Algorithm.AndroidDeviceType

		ecdhkey := Algorithm.DoECDH713(loginRes.GetAuthSectResp().GetSvrPubEcdhkey().GetKey().GetBuffer(), prikey)
		LoginData.Loginecdhkey = ecdhkey
		LoginData.Uin = loginRes.GetAuthSectResp().GetUin()
		LoginData.Wxid = loginRes.GetAcctSectResp().GetUserName()
		LoginData.Alais = loginRes.GetAcctSectResp().GetAlias()
		LoginData.Mobile = loginRes.GetAcctSectResp().GetBindMobile()
		LoginData.NickName = loginRes.GetAcctSectResp().GetNickName()
		LoginData.Sessionkey = Algorithm.AESDecrypt(loginRes.GetAuthSectResp().GetSessionKey().GetBuffer(), ecdhkey)
		LoginData.Sessionkey_2 = loginRes.GetAuthSectResp().GetSessionKey().GetBuffer()
		LoginData.Autoauthkey = loginRes.GetAuthSectResp().GetAutoAuthKey().GetBuffer()
		LoginData.Autoauthkeylen = int32(loginRes.GetAuthSectResp().GetAutoAuthKey().GetILen())
		LoginData.Serversessionkey = loginRes.GetAuthSectResp().GetServerSessionKey().GetBuffer()
		LoginData.Clientsessionkey = loginRes.GetAuthSectResp().GetClientSessionKey().GetBuffer()
		LoginData.RsaPublicKey = pubkey
		LoginData.RsaPrivateKey = prikey

		err := comm.CreateLoginData(&LoginData, LoginData.Wxid, 0, nil)

		if err != nil {
			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: fmt.Sprintf("系统异常：%v", err.Error()),
				Data:    nil,
			}
		}

		return models.ResponseResult{
			Code:    0,
			Success: true,
			Message: "成功",
			Data:    &loginRes,
		}
	}

	//30系列转向
	if loginRes.GetBaseResponse().GetRet() == -301 {
		var Wx_newLongIPlist, Wx_newshortIplist, Wx_newshortextipList list.List
		var Wx_newLong_Host, Wx_newshort_Host, Wx_newshortext_Host list.List

		dns_info := loginRes.GetNetworkSectResp().GetNewHostList().GetList()
		for _, v := range dns_info {
			if v.GetHost() == "long.weixin.qq.com" {
				ip_info := loginRes.GetNetworkSectResp().GetBuiltinIplist().GetLongConnectIplist()
				for _, ip := range ip_info {
					host := ip.GetHost()
					host = strings.Replace(host, string(byte(0x00)), "", -1)
					if host == v.GetRedirect() {
						ipaddr := ip.GetIp()
						ipaddr = strings.Replace(ipaddr, string(byte(0x00)), "", -1)
						Wx_newLongIPlist.PushBack(ipaddr)
						Wx_newLong_Host.PushBack(host)
					}
				}
			} else if v.GetHost() == "short.weixin.qq.com" {
				ip_info := loginRes.GetNetworkSectResp().GetBuiltinIplist().GetShortConnectIplist()
				for _, ip := range ip_info {
					host := ip.GetHost()
					host = strings.Replace(host, string(byte(0x00)), "", -1)
					if host == v.GetRedirect() {
						ipaddr := ip.GetIp()
						ipaddr = strings.Replace(ipaddr, string(byte(0x00)), "", -1)
						Wx_newshortIplist.PushBack(ipaddr)
						Wx_newshort_Host.PushBack(host)
					}
				}
			} else if v.GetHost() == "extshort.weixin.qq.com" {
				ip_info := loginRes.GetNetworkSectResp().GetBuiltinIplist().GetShortConnectIplist()
				for _, ip := range ip_info {
					host := ip.GetHost()
					host = strings.Replace(host, string(byte(0x00)), "", -1)
					if host == v.GetRedirect() {
						ipaddr := ip.GetIp()
						ipaddr = strings.Replace(ipaddr, string(byte(0x00)), "", -1)
						Wx_newshortextipList.PushBack(ipaddr)
						Wx_newshortext_Host.PushBack(host)
					}
				}
			}
		}
		return AndroidA16Login1(Data, Wx_newshort_Host.Front().Value.(string))
	}

	//活号
	if strings.Index(loginRes.GetBaseResponse().ErrMsg.String(), "环境存在异常") >= 0 {
		// 过滑块并根据结果判断
		if err := LoginOCR(loginRes.GetBaseResponse().ErrMsg.String()); err == nil {
			// 滑块成功, 再次登录
			return AndroidA16Login1(Data, domain)
		} else {
			// 返回异常
			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: fmt.Sprintf("系统异常：%v", err.Error()),
				Data:    nil,
			}
		}
	}
	//死号
	if strings.Index(loginRes.GetBaseResponse().ErrMsg.String(), "若账号内有资金，可轻触“确定”按相关指引进行操作") >= 0 {
		// 过滑块并根据结果判断
		if err := LoginOCRS(loginRes.GetBaseResponse().ErrMsg.String()); err == nil {
			// 滑块成功, 再次登录
			return AndroidA16Login1(Data, domain)
		} else {
			// 返回异常
			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: fmt.Sprintf("系统异常：%v", err.Error()),
				Data:    nil,
			}
		}
	}

	return models.ResponseResult{
		Code:    -8,
		Success: false,
		Message: "失败",
		Data:    &loginRes,
	}
}
