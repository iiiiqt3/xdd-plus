package Login

import (
	"encoding/base64"
	"fmt"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/clientsdk/baseutils"

	"github.com/golang/protobuf/proto"

	"wechatdll/Cilent/mm"
	"wechatdll/comm"
	"wechatdll/models"
)

// @Summary 获取二维码(iPad-绕过验证码)
func GetQRCODEx(Data GetQRReq) models.ResponseResult2 {
	D, _ := comm.GetLoginataByDevId(Data.DeviceID)
	reqDataLogin := DataLogin{
		UserName:      "",
		Data62:        "",
		DeviceName:    Data.DeviceName,
		DeviceId:      Data.DeviceID,
		Proxy:         Data.Proxy,
		ClientVersion: int32(Algorithm.IPadVersionx),
	}
	if D == nil || D.Wxid == "" || D.ClientVersion != Algorithm.IPadVersionx {
		// 没有缓存, 初始化新的账号环境
		D = GenIpadLoginData(reqDataLogin)
	} else {
		D = UpdateIpadLoginData(D, reqDataLogin)
	}
	//初始化Mmtls
	httpclient, MmtlsClient, err := comm.MmtlsInitialize(Data.Proxy, D.ShortHost)
	if err != nil {
		return models.ResponseResult2{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("MMTLS初始化失败：%v", err.Error()),
			Data:    nil,
		}
	}

	D.Aeskey = []byte(baseutils.RandSeq(16)) //获取随机密钥

	FpInitAndRrefresh(D, httpclient)
	if D.DeviceToken == nil {
		D.DeviceToken = &mm.TrustResponse{}
	}

	req := &mm.GetLoginQRCodeRequest{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    []byte{},
			Uin:           proto.Uint32(0),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		RandomEncryKey: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(D.Aeskey))),
			Buffer: D.Aeskey,
		},
		Opcode:           proto.Uint32(0),
		MsgContextPubKey: nil,
	}

	reqdata, err := proto.Marshal(req)

	if err != nil {
		return models.ResponseResult2{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}

	hec := InitHec(D)
	hypack := hec.HybridEcdhPackIosEn(502, 0, nil, reqdata)
	recvData, err := httpclient.MMtlsPost(D.ShortHost, "/cgi-bin/micromsg-bin/getloginqrcode", hypack, Data.Proxy)
	if err != nil {
		return models.ResponseResult2{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}
	ph1 := hec.HybridEcdhPackIosUn(recvData)
	getloginQRRes := mm.GetLoginQRCodeResponse{}

	err = proto.Unmarshal(ph1.Data, &getloginQRRes)

	if err != nil {
		return models.ResponseResult2{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}
	}

	if getloginQRRes.GetBaseResponse().GetRet() == 0 {
		if getloginQRRes.Uuid == nil || *getloginQRRes.Uuid == "" {
			return models.ResponseResult2{
				Code:    -9,
				Success: false,
				Message: "取码过于频繁",
				Data:    getloginQRRes.GetBaseResponse(),
			}
		}

		//保存redis
		D.Uuid = getloginQRRes.GetUuid()
		D.NotifyKey = getloginQRRes.GetNotifyKey().GetBuffer()
		D.Cooike = ph1.Cookies
		D.MmtlsKey = MmtlsClient
		err := comm.CreateLoginData(D, "", 300, nil)

		if err == nil {
			return models.ResponseResult2{
				Code:    1,
				Success: true,
				Message: "成功",
				Data: GetQRRes{
					baseResponse: GetQRResErr{
						Ret:   getloginQRRes.GetBaseResponse().GetRet(),
						Error: getloginQRRes.GetBaseResponse().GetErrMsg().GetString_(),
					},
					QrBase64:    fmt.Sprintf("data:image/jpg;base64,%v", base64.StdEncoding.EncodeToString(getloginQRRes.GetQrcode().GetBuffer())),
					Uuid:        getloginQRRes.GetUuid(),
					QrUrl:       "https://api.qrserver.com/v1/create-qr-code/?data=http://weixin.qq.com/x/" + getloginQRRes.GetUuid(),
					ExpiredTime: time.Now().Format("2006-01-02 15:04:05"),
				},
				Data62:   baseutils.Get62Data(D.Deviceid_str),
				DeviceId: D.Deviceid_str,
			}
		}
	}

	return models.ResponseResult2{
		Code:    -0,
		Success: false,
		Message: "未知的错误",
		Data:    &getloginQRRes,
	}
}
