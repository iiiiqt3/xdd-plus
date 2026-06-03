package wxcore

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/TcpPoll"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/Login"
	"wechatdll/models/Msg"
	"wechatdll/srv/wxface"

	"github.com/astaxie/beego"

	"google.golang.org/protobuf/proto"
)

// WXModels 微信链接接口
type WXModels struct {
	wxconn *WXConnect
}

// NewWXReqInvoker 新建一个请求调用器
func NewWXModels(wxconn *WXConnect) wxface.IWXModels {
	return &WXModels{
		wxconn: wxconn,
	}
}

// 消息同步接口
func (m *WXModels) MsgSync(Data Msg.SyncParam) models.ResponseResult {
	return Msg.Sync(Data)
}

// 短链接心跳接口
func (m *WXModels) LoginHeartBeat(Wxid string) (models.ResponseResult, *mm.HeartBeatResponse) {
	return Login.HeartBeat(Wxid)
}

// 长连接心跳接口
func (m *WXModels) LoginHeartBeatLong(Wxid string) (models.ResponseResult, *mm.HeartBeatResponse) {
	tcpManager, err := TcpPoll.GetTcpManager()
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("出错了: %v", err.Error()),
			Data:    nil,
		}, nil
	}
	userInfo := m.wxconn.GetWXAccount().GetUserInfo()
	// 从缓存获取
	D, err := comm.GetLoginata(userInfo.Wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("LoginHeartBeatLong 出错了: %v [%v]", "未找到登录信息", userInfo.Wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("LoginHeartBeatLong 出错了: %v", err.Error())
		}
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: errorMsg,
			Data:    nil,
		}, nil
	}
	client, err := tcpManager.GetClient(userInfo, m.MsgListen)
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

// 二次登录接口
func (m *WXModels) LoginSecautoauth(Wxid string) (models.ResponseResult, *mm.UnifyAuthResponse) {
	return Login.Secautoauth(Wxid)
}

// 消息监听
func (m *WXModels) MsgListen(cmdId int) error {
	fmt.Println("接收到长链接消息，正在处理回调")
	WXDATA := Msg.Sync(Msg.SyncParam{Wxid: m.wxconn.GetWXAccount().GetUserInfo().Wxid, Synckey: "", Scene: 0})
	jsonValue, _ := json.Marshal(WXDATA)
	syncUrl := strings.Replace(beego.AppConfig.String("syncmessagebusinessuri"), "{0}", m.wxconn.GetWXAccount().GetUserInfo().Wxid, -1)
	reqBody := strings.NewReader(string(jsonValue))
	go comm.HttpPosthb(syncUrl, reqBody, nil, "", "", "", "")

	rabbitmqEnabled, err := beego.AppConfig.Bool("rabbitmq")
	if err != nil {
		return nil
	}
	if rabbitmqEnabled {
		comm.PublishRabbitMq(beego.AppConfig.String("rabbitmqexchange"), jsonValue)
	}

	return nil
}
