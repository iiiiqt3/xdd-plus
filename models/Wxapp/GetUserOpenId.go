package Wxapp

import (
	"fmt"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/wechat"
	"wechatdll/comm"
	"wechatdll/models"

	"github.com/golang/protobuf/proto"
)

type GetUserOpenIdParam struct {
	Appid  string
	ToWxId string
	Wxid   string
}

func GetUserOpenId(Data GetUserOpenIdParam) models.ResponseResult {
	D, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	req := &wechat.MMBizJsApiGetUserOpenIdRequest{
		BaseRequest: &wechat.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(1),
		},
		AppId:    proto.String(Data.Appid),
		UserName: proto.String(Data.ToWxId),
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
	Link := "/cgi-bin/mmbiz-bin/usrmsg/mmbizjsapi_getuseropenid"
	CGI := 1177

	//发包
	protobufdata, _, errtype, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: Link,
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              CGI,
			Uin:              D.Uin,
			Cookie:           D.Cooike,
			Sessionkey:       D.Sessionkey,
			EncryptType:      5,
			Loginecdhkey:     D.RsaPublicKey,
			Clientsessionkey: D.Clientsessionkey,
			UseCompress:      true,
		},
	}, D.MmtlsKey)
	//if errtype == 0 {
	//}
	/*
		if err != nil {

			return models.ResponseResult{
				Code:    errtype,
				Success: false,
				Message: err.Error(),
				Data:    nil,
			}

		}

		//解包
		Response := mm.GetUserOpenIdResp{}
		err = proto.Unmarshal(protobufdata, &Response)

		if err != nil {

			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
				Data:    nil,
			}

		}
	*/

	if err != nil {
		return models.ResponseResult{
			Code:    errtype,
			Success: false,
			Message: err.Error(),
			Data:    nil,
		}
	}

	//解包
	Response := wechat.MMBizJsApiGetUserOpenIdResponse{}
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
