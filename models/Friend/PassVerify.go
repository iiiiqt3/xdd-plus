package Friend

import (
	"fmt"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/clientsdk/extinfo"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/Login"

	"github.com/golang/protobuf/proto"
)

type PassVerifyParam struct {
	Wxid  string
	V1    string
	V2    string
	Scene int
}

func PassVerify(Data PassVerifyParam) models.ResponseResult {
	D, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", Data.Wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("异常：%v", err.Error())
		}
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: errorMsg,
			Data:    nil,
		}
	}

	if Data.V1 == "" || Data.V2 == "" {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: "v1和v2是必须参数",
			Data:    nil,
		}
	}

	if Data.Scene <= 0 {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: "来源[Scene]必须填写",
			Data:    nil,
		}
	}

	VerifyUserList := make([]*mm.VerifyUser, 0)

	VerifyUserList = append(VerifyUserList, &mm.VerifyUser{
		Value:               proto.String(Data.V1),
		VerifyUserTicket:    proto.String(Data.V2),
		AntispamTicket:      proto.String(""),
		FriendFlag:          proto.Uint32(0),
		ChatRoomUserName:    proto.String(""),
		SourceUserName:      proto.String(""),
		SourceNickName:      proto.String(""),
		ScanQrcodeFromScene: proto.Uint32(0),
		ReportInfo:          proto.String(""),
		OuterUrl:            proto.String(""),
		SubScene:            proto.Int32(0),
	})

	Wcstf, _ := Algorithm.GetWcstf08(Data.Wxid)
	Wcste, _ := Algorithm.GetWcste08()
	var EncryptData []byte
	hec := Login.InitHec(D)
	if hec.IsAndroid {
		EncryptData = extinfo.GetiPhoneNewSpamData(D)
	} else {
		EncryptData = extinfo.GetNewSpamDataV8(D)
	}

	ccData := &mm.CryptoData{
		Version:     []byte("00000008"),
		Type:        proto.Uint32(1),
		EncryptData: EncryptData,
		Timestamp:   proto.Uint32(uint32(time.Now().Unix())),
		Unknown5:    proto.Uint32(5),
		Unknown6:    proto.Uint32(0),
	}
	ccDataseq, _ := proto.Marshal(ccData)

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
	}

	WCExtInfoseq, _ := proto.Marshal(WCExtInfo)

	req := &mm.VerifyUserRequest{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		Opcode:             proto.Int32(3),
		VerifyUserListSize: proto.Uint32(1),
		VerifyUserList:     VerifyUserList,
		SceneListCount:     proto.Uint32(1),
		SceneList:          []byte{byte(Data.Scene)},
		ExtSpamInfo: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(WCExtInfoseq))),
			Buffer: WCExtInfoseq,
		},
		NeedConfirm: proto.Uint32(1),
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
		Cgiurl: "/cgi-bin/micromsg-bin/verifyuser",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              137,
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
	Response := mm.VerifyUserResponse{}
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
