package Login

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/golang/protobuf/proto"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/TcpPoll"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/srv/sync"
)

var userService = sync.NewUserService()

func HeartBeatLong(wxid string) (models.ResponseResult, *mm.HeartBeatResponse) {
	D, err := comm.GetLoginata(wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("异常：%v", err.Error())
		}
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: errorMsg,
			Data:    nil,
		}, nil
	}
	// http同步
	// syncUrl := strings.Replace(beego.AppConfig.String("syncmessagebusinessuri"), "{0}", D.Wxid, -1)
	// go comm.HttpPost(syncUrl, *new(url.Values), nil, "", "", "", "")

	tcpManager, err := TcpPoll.GetTcpManager()
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("出错了: %v", err.Error()),
			Data:    nil,
		}, nil
	}
	client, err := tcpManager.GetClient(D, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("出错了: %v", err.Error()),
			Data:    nil,
		}, nil
	}

	req := &mm.HeartBeatRequest{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(2),
		},
		TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
	}

	reqdata, err := proto.Marshal(req)
	// AES组包: Cgiurl: "/cgi-bin/micromsg-bin/heartbeat",Cgi: 518,EncryptType: 5,UseCompress: true
	sendData := Algorithm.Pack(reqdata, 518, D.Uin, D.Sessionkey, D.Cooike, D.Clientsessionkey, D.RsaPublicKey, 5, false)
	// mmtls发包
	cmdId := 238
	protobufdata, err := client.MmtlsSend(sendData, cmdId, "238心跳")
	if err != nil {
		tcpManager.Remove(client)
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}, nil
	}
	//解包
	HeartBeatResponse := mm.HeartBeatResponse{}
	err = proto.Unmarshal(*protobufdata, &HeartBeatResponse)
	if err != nil {
		tcpManager.Remove(client)
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}, nil
	}
	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    HeartBeatResponse,
	}, &HeartBeatResponse
}

func HeartBeat(Wxid string) (models.ResponseResult, *mm.HeartBeatResponse) {
	//syncmessage, _ := beego.AppConfig.Bool("syncmessage")
	//if syncmessage {
	//	userService.AddUser(Wxid, "fsdf", 1*time.Second, 10*time.Minute)
	//}

	D, err := comm.GetLoginata(Wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", Wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("异常：%v", err.Error())
		}
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: errorMsg,
			Data:    nil,
		}, nil
	}

	//syncUrl := strings.Replace(beego.AppConfig.String("syncmessagebusinessuri"), "{0}", D.Wxid, -1)
	//go comm.HttpPost(syncUrl, *new(url.Values), nil, "", "", "", "")

	req := &mm.HeartBeatRequest{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
		Scene:     proto.Uint32(0),
	}

	reqdata, err := proto.Marshal(req)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}, nil
	}

	//发包
	protobufdata, _, errtype, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/micromsg-bin/heartbeat",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              518,
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
		}, nil
	}

	//解包
	HeartBeatResponse := mm.HeartBeatResponse{}
	err = proto.Unmarshal(protobufdata, &HeartBeatResponse)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}, nil
	}

	syncmessage, _ := beego.AppConfig.Bool("syncmessage")
	if syncmessage {
		userService.AddUser(Wxid, D.NickName, 1*time.Second, 10*time.Minute)
	}

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    HeartBeatResponse,
	}, &HeartBeatResponse
}
