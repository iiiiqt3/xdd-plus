package Msg

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/jm"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/comm"
	"wechatdll/models"
)

type SendGroupMassMsgTextParam struct {
	Wxid    string
	ToWxid  []string
	Content string
}

// 自定义 UnmarshalJSON 方法
func (p *SendGroupMassMsgTextParam) UnmarshalJSON(data []byte) error {
	// 临时结构体
	type Alias SendGroupMassMsgTextParam
	temp := &struct {
		ToWxid interface{} `json:"ToWxid"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	// 解析数据
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// 处理 ToWxid 字段，支持单个字符串或数组
	switch v := temp.ToWxid.(type) {
	case string:
		// 如果 ToWxid 是单个字符串，转为一个切片
		p.ToWxid = []string{v}
	case []interface{}:
		// 如果 ToWxid 是一个数组，转为 []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				p.ToWxid = append(p.ToWxid, str)
			}
		}
	default:
		return fmt.Errorf("invalid ToWxid format")
	}

	return nil
}

func SendGroupMassMsgText(Data SendGroupMassMsgTextParam) models.ResponseResult {
	D, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	// 将 ToWxid 数组转换为字符串
	toList := strings.Join(Data.ToWxid, ",")
	tolistmd5 := baseutils.MD5ToLower(toList)
	Databuff := []byte(Data.Content)
	ClientImgId := fmt.Sprintf("%v_%v", time.Now().Unix(), tolistmd5)

	// 消息组包
	MsgRequest := &jm.MassSendRequest{
		BaseRequest: &jm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		ToList:    proto.String(toList),
		ToListMd5: proto.String(tolistmd5),
		ClientId:  proto.String(ClientImgId),
		MsgType:   proto.Uint32(1),
		MediaTime: proto.Uint32(0),
		DataBuffer: &jm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len(Databuff))),
			Buffer: Databuff,
		},
		DataStartPos:  proto.Uint32(0),
		DataTotalLen:  proto.Uint32(uint32(len(Databuff))),
		ThumbTotalLen: proto.Uint32(0),
		ThumbStartPos: proto.Uint32(0),
		ThumbData: &jm.SKBuiltinBufferT{
			ILen:   proto.Uint32(0),
			Buffer: []byte{},
		},
		CameraType:   proto.Uint32(2),
		VideoSource:  proto.Uint32(0),
		ToListCount:  proto.Uint32(uint32(len(Data.ToWxid))),
		IsSendAgain:  proto.Uint32(1),
		CompressType: proto.Uint32(0),
		VoiceFormat:  proto.Uint32(0),
	}

	// 序列化
	reqdata, _ := proto.Marshal(MsgRequest)

	// 发包
	protobufdata, _, errtype, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/micromsg-bin/masssend",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              193,
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

	// 解包
	NewSendMsgRespone := jm.MassSendResponse{}
	err = proto.Unmarshal(protobufdata, &NewSendMsgRespone)
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
		Data:    NewSendMsgRespone,
	}
}
