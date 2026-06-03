package Tools

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/golang/protobuf/proto"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/wechat"
	"wechatdll/comm"
	"wechatdll/models"
)

func UploadAppAttachService(queryKey string, m UploadAppAttachModel) models.ResponseResult {
	D, err := comm.GetLoginata(queryKey, nil)
	println(queryKey)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", queryKey)
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

	// 处理文件数据
	fileData := m.FileData
	sFileBase := strings.Split(fileData, ",")
	if len(sFileBase) > 1 {
		fileData = sFileBase[1]
	}
	fileBytes, err := base64.StdEncoding.DecodeString(fileData)
	if err != nil {
		//return vo.NewFail("文件解码失败：" + err.Error())
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	// 上传文件
	resp, err := UploadAppAttach(fileBytes, D.Wxid)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("上传文件失败：%v", err.Error()),
			Data:    nil,
		}
	}

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "上传文件成功",
		Data:    resp,
	}
}

// UploadAppAttach 上传文件
func UploadAppAttach(fileData []byte, key string) (*wechat.UploadAppAttachResponse, error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("UploadAppAttach error: %v\n", r)
		}
	}()
	packHeader, err := SendUploadAppAttach(key, fileData)
	D, err := comm.GetLoginata(key, nil)
	if err != nil {
		return nil, err
	}

	response := &wechat.UploadAppAttachResponse{}
	err = ParseResponseData(D, (*PackHeader)(packHeader), response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// SendUploadAppAttach 上传文件
func SendUploadAppAttach(key string, fileData []byte) (*PackHeader, error) {
	D, err := comm.GetLoginata(key, nil)
	if err != nil {
		return nil, err
	}
	Stream := bytes.NewBuffer(fileData)

	datalen := 50000
	datatotalength := Stream.Len()
	FileMD5 := GetFileMD5Hash(fileData)

	ClientAppDataId := fmt.Sprintf("%v_%v_UploadFile", key, time.Now().Unix())

	Startpos := 0
	I := 0
	var protobufdata []byte

	for {
		Startpos = I * datalen
		count := 0
		if datatotalength-Startpos > datalen {
			count = datalen
		} else {
			count = datatotalength - Startpos
		}
		if count <= 0 {
			break
		}

		Databuff := make([]byte, count)
		_, _ = Stream.Read(Databuff)

		req := &wechat.UploadAppAttachRequest{
			BaseRequest:     GetBaseRequest(D),
			AppId:           proto.String(""),
			SdkVersion:      proto.Uint32(0),
			ClientAppDataId: proto.String(ClientAppDataId),
			UserName:        proto.String(key),
			TotalLen:        proto.Uint32(uint32(datatotalength)),
			StartPos:        proto.Uint32(uint32(Startpos)),
			DataLen:         proto.Uint32(uint32(len(Databuff))),
			Data: &wechat.SKBuiltinBufferT{
				ILen:   proto.Uint32(uint32(len(Databuff))),
				Buffer: Databuff,
			},
			Type: proto.Uint32(6),
			Md5:  proto.String(FileMD5),
		}

		// 序列化请求数据
		reqdata, err := proto.Marshal(req)
		if err != nil {
			return nil, err
		}

		// 发送请求
		//sendEncodeData := Pack(D, reqdata, 220, 5)

		//发包
		resp, _, _, err := comm.SendRequest(comm.SendPostData{
			Ip:     D.Mmtlsip,
			Host:   D.ShortHost,
			Cgiurl: "/cgi-bin/micromsg-bin/uploadappattach",
			Proxy:  D.Proxy,
			PackData: Algorithm.PackData{
				Reqdata:          reqdata,
				Cgi:              220,
				Uin:              D.Uin,
				Cookie:           D.Cooike,
				Sessionkey:       D.Sessionkey,
				EncryptType:      5,
				Loginecdhkey:     D.RsaPublicKey,
				Clientsessionkey: D.Clientsessionkey,
				UseCompress:      false,
			},
		}, D.MmtlsKey)

		//resp, err := mmtls.MMHTTPPostData(userInfo.MMInfo, "/cgi-bin/micromsg-bin/uploadappattach", sendEncodeData)
		if err != nil {
			return nil, err
		}
		//解包
		//Response := mm.FileInfo{}
		//err = proto.Unmarshal(protobufdata, &Response)

		protobufdata = resp
		I++
	}

	return DecodePackHeader(protobufdata, nil)
}

func GetFileMD5Hash(Data []byte) string {
	hash := md5.New()
	hash.Write(Data)
	retVal := hash.Sum(nil)
	return hex.EncodeToString(retVal)
}

func GetBaseRequest(D *comm.LoginData) *wechat.BaseRequest {
	ret := &wechat.BaseRequest{}
	ret.SessionKey = []byte(D.Sessionkey)
	ret.Uin = &D.Uin
	if !strings.HasPrefix(D.Deviceid_str, "A") {
		ret.DeviceId = D.Deviceid_byte
		ret.ClientVersion = proto.Int32(int32(D.ClientVersion))
		ret.Scene = proto.Uint32(0)
	} else {
		ret.ClientVersion = proto.Int32(int32(D.ClientVersion))
		ret.DeviceId = D.Deviceid_byte
		ret.Scene = proto.Uint32(1)
	}
	return ret
}
