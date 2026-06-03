package TenPay

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	Jxml "wechatdll/Xml"
	"wechatdll/bts"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/User"

	"github.com/golang/protobuf/proto"
)

type OpenwxhbParam struct {
	Xml              string
	Wxid             string
	SendUserName     string
	TimingIdentifier string
	Encrypt_key      string
	Encrypt_userinfo string
}

func Openwxhb(Data OpenwxhbParam) models.ResponseResult {
	D, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}
	//解析xml组合
	var HongBao Jxml.HongBao
	_ = xml.Unmarshal([]byte(Data.Xml), &HongBao)

	WxInfo := User.GetContractProfile(Data.Wxid)

	if WxInfo.Code != 0 {
		return models.ResponseResult{
			Code:    WxInfo.Code,
			Success: false,
			Message: fmt.Sprintf("个人信息获取异常：%v", WxInfo.Message),
			Data:    WxInfo.Data,
		}
	}

	Info := bts.GetProfile(WxInfo.Data)

	City := Info.GetUserInfo().GetCity()
	Province := Info.GetUserInfo().GetProvince()

	Text := "channelId=1&city=" + City + "&encrypt_key=" + Data.Encrypt_key + "&encrypt_userinfo=" + Data.Encrypt_userinfo + "&headImg=" + url.QueryEscape(Info.GetUserInfoExt().GetSmallHeadImgUrl()) + "&msgType=1&nativeUrl=" + url.QueryEscape(HongBao.Appmsg.Wcpayinfo.Nativeurl) + "&nickName=" + url.QueryEscape(Info.GetUserInfo().GetNickName().GetString_()) + "&province=" + Province + "&sendId=" + HongBao.Appmsg.Wcpayinfo.Paymsgid + "&sessionUserName=" + Data.SendUserName + "&timingIdentifier=" + Data.TimingIdentifier

	//拆开红包
	req := &mm.HongBaoReq{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		CgiCmd:    proto.Int(4),
		OutPutTyp: proto.Int(1),
		ReqText: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len([]byte(Text)))),
			Buffer: []byte(Text),
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
		Cgiurl: "/cgi-bin/mmpay-bin/openwxhb",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              1685,
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
	Response := mm.HongBaoRes{}
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
