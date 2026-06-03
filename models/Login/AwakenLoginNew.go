package Login

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"wechatdll/Cilent/mm"
	"wechatdll/Mmtls"
	"wechatdll/baseinfo"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/Tools"
)

func AwakenLoginNew(Data AwakenReq) models.ResponseResult {
	D, err := comm.GetLoginatas(Data.Wxid)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	req := &mm.PushLoginURLRequest{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		Autoauthticket: proto.String(""),
		Autoauthkey: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(D.Autoauthkeylen)),
			Buffer: D.Autoauthkey,
		},
		ClientId:   proto.String(fmt.Sprintf("iPad-Push-%s.110141", D.Deviceid_byte)),
		Devicename: proto.String(D.DeviceName),
		Opcode:     proto.Int32(3),
		RandomEncryKey: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(D.Sessionkey))),
			Buffer: D.Sessionkey,
		},
		Username: proto.String(D.Wxid),
	}

	reqdata, err := proto.Marshal(req)
	hecData := Tools.Pack(D, reqdata, baseinfo.MMRequestTypePushQrLogin, 1)

	//hec := &Algorithm.Client{}
	//hec.Init("IOS")
	//hecData := hec.HybridEcdhPackIosEn(654, D.Uin, D.Cooike, reqdata)

	httpclient := Mmtls.GenNewHttpClient2(D.MmtlsKey, D.ShortHost, Data.Proxy)

	recvData, err := httpclient.MMtlsPost(D.ShortHost, "/cgi-bin/micromsg-bin/pushloginurl", hecData, Data.Proxy)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}
	if len(recvData) < 32 {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：协议返回少于32字节"),
			Data:    nil,
		}
	}

	packHeader, errRep := Tools.DecodePackHeader(recvData, nil)

	if errRep != nil {

		if packHeader != nil && packHeader.GetRetCode() == baseinfo.MMRequestRetSessionTimeOut {
			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: fmt.Sprintf("链接失效：%v", errRep.Error()),
				Data:    nil,
			}
		}

		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", errRep.Error()),
			Data:    nil,
		}
	}

	//解包
	qrCodeResponse := &mm.PushLoginURLResponse{}

	err = Tools.ParseResponseData(D, packHeader, qrCodeResponse)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}
	}
	//保存redis
	err = comm.CreateLoginData(&comm.LoginData{
		Uuid:                       qrCodeResponse.GetUuid(),
		Aeskey:                     D.Sessionkey,
		NotifyKey:                  qrCodeResponse.GetNotifyKey().GetBuffer(),
		Deviceid_str:               D.Deviceid_str,
		Deviceid_byte:              D.Deviceid_byte,
		DeviceName:                 D.DeviceName,
		HybridEcdhPrivkey:          D.HybridEcdhPrivkey,
		HybridEcdhPubkey:           D.HybridEcdhPubkey,
		HybridEcdhInitServerPubKey: D.HybridEcdhInitServerPubKey,
		Cooike:                     packHeader.Session,
		MmtlsKey:                   D.MmtlsKey,
		Proxy:                      Data.Proxy,
	}, "", 300,nil)

	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("Redis ERROR：%v", err.Error()),
			Data:    nil,
		}
	}

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    qrCodeResponse,
	}
}
