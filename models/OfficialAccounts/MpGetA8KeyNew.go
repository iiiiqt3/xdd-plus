package OfficialAccounts

import (
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/comm"
	"wechatdll/models"
)

func MpGetA8KeyNew(Data ReadParam) models.ResponseResult {
	D, err := comm.GetLoginatas(Data.Wxid)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}
	cookieBytes, err := base64.StdEncoding.DecodeString("")
	if err != nil {
		cookieBytes = []byte{}
	}
	req := &mm.GetA8KeyReq{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		CodeType:    proto.Uint32(19),
		CodeVersion: proto.Uint32(5),
		Flag:        proto.Uint32(0),
		FontScale:   proto.Uint32(100),
		NetType:     proto.String(""),
		OpCode:      proto.Uint32(2),
		UserName:    proto.String(D.Wxid),
		ReqUrl: &mm.SKBuiltinStringT{
			String_: proto.String(Data.Url),
		},
		FriendQq: proto.Uint32(0),
		Scene:    proto.Uint32(4),
		Cookie: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(cookieBytes))),
			Buffer: cookieBytes,
		},
	}

	reqdata, err := proto.Marshal(req)

	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}

	//发包
	protobufdata, _, errtype, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/micromsg-bin/mp-geta8key",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              238,
			Uin:              D.Uin,
			Cookie:           D.Cooike,
			Sessionkey:       D.Sessionkey,
			EncryptType:      5,
			Loginecdhkey:     D.RsaPublicKey,
			Clientsessionkey: D.Clientsessionkey,
			UseCompress:      false,
		},
	}, D.MmtlsKey)

	if err != nil {
		return models.ResponseResult{
			Code:    errtype,
			Success: false,
			Message: err.Error(),
			Data:    nil,
		}
	}

	//解包
	Response := mm.GetA8KeyResp{}
	err = proto.Unmarshal(protobufdata, &Response)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}
	}

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    Response,
	}

}
