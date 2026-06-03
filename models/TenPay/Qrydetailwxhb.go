package TenPay

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	Jxml "wechatdll/Xml"
	"wechatdll/baseinfo"
	"wechatdll/bts"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/User"

	"github.com/golang/protobuf/proto"
)

type QrydetailwxhbParam struct {
	Xml              string
	Wxid             string
	Encrypt_key      string
	Encrypt_userinfo string
}

func Qrydetailwxhb(Data QrydetailwxhbParam) models.ResponseResult {
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

	Province := Info.GetUserInfo().GetProvince()

	Text := "channelId=1&encrypt_key=" + Data.Encrypt_key + "&encrypt_userinfo=" + Data.Encrypt_userinfo + "&msgType=1&nativeUrl=" + url.QueryEscape(HongBao.Appmsg.Wcpayinfo.Nativeurl) + "&province=" + Province + "&sendId=" + HongBao.Appmsg.Wcpayinfo.Paymsgid

	req := &mm.HongBaoReq{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		CgiCmd:    proto.Int(5),
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
		Cgiurl: "/cgi-bin/mmpay-bin/qrydetailwxhb",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              1585,
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

// 接受红包
func receivewxhb(O receivewxhbParam) (mm.HongBaoRes, error) {
	//解析xml组合
	var HongBao Jxml.HongBao
	_ = xml.Unmarshal([]byte(O.Xml), &HongBao)

	Text := "agreeDuty=0&channelId=1&city=" + O.City + "&encrypt_key=" + O.Encrypt_key + "&encrypt_userinfo=" + O.Encrypt_userinfo + "&inWay=" + O.InWay + "&msgType=1&nativeUrl=" + url.QueryEscape(HongBao.Appmsg.Wcpayinfo.Nativeurl) + "&province=" + O.Province + "&sendId=" + HongBao.Appmsg.Wcpayinfo.Paymsgid
	D := O.D
	//直接组包
	req := &mm.HongBaoReq{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		CgiCmd:    proto.Int(3),
		OutPutTyp: proto.Int(1),
		ReqText: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len([]byte(Text)))),
			Buffer: []byte(Text),
		},
	}
	//序列化
	reqdata, _ := proto.Marshal(req)

	protobufdata, _, _, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/mmpay-bin/receivewxhb",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              1581,
			Uin:              D.Uin,
			Cookie:           D.Cooike,
			Sessionkey:       D.Sessionkey,
			EncryptType:      5,
			Loginecdhkey:     D.Loginecdhkey,
			Clientsessionkey: D.Clientsessionkey,
			UseCompress:      false,
		},
	}, D.MmtlsKey)

	if err != nil {
		return mm.HongBaoRes{}, err
	}

	Response := mm.HongBaoRes{}
	err = proto.Unmarshal(protobufdata, &Response)

	return Response, nil
}

// 创建红包
func createwxhb(D comm.LoginData, hbItem RedPacket) (mm.HongBaoRes, error) {

	// ReqText
	strReqText := string("")
	strReqText = strReqText + "city=Guangzhou&"
	strReqText = strReqText + "hbType=" + strconv.Itoa(int(hbItem.RedType)) + "&"
	strReqText = strReqText + "headImg=" + "&"
	strReqText = strReqText + "inWay=" + strconv.Itoa(int(hbItem.From)) + "&"
	strReqText = strReqText + "needSendToMySelf=0" + "&"
	strReqText = strReqText + "nickName=" + url.QueryEscape(D.NickName) + "&"
	strReqText = strReqText + "perValue=" + strconv.Itoa(int(hbItem.Amount)) + "&"
	strReqText = strReqText + "province=Beijing" + "&"
	strReqText = strReqText + "receiveNickName=" + "&"
	strReqText = strReqText + "sendUserName=" + D.Wxid + "&"
	strReqText = strReqText + "username=" + hbItem.Username + "&"
	strReqText = strReqText + "wishing=" + url.QueryEscape(hbItem.Content)

	//直接组包
	req := &mm.HongBaoReq{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		CgiCmd:    proto.Int(0),
		OutPutTyp: proto.Int(0),
		ReqText: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len([]byte(strReqText)))),
			Buffer: []byte(strReqText),
		},
	}
	//序列化
	reqdata, _ := proto.Marshal(req)

	protobufdata, _, _, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/mmpay-bin/requestwxhb",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              1575,
			Uin:              D.Uin,
			Cookie:           D.Cooike,
			Sessionkey:       D.Sessionkey,
			EncryptType:      5,
			Loginecdhkey:     D.Loginecdhkey,
			Clientsessionkey: D.Clientsessionkey,
			UseCompress:      false,
		},
	}, D.MmtlsKey)

	if err != nil {
		return mm.HongBaoRes{}, err
	}

	Response := mm.HongBaoRes{}
	err = proto.Unmarshal(protobufdata, &Response)
	//fmt.Println(Response)

	return Response, nil
}

func ToByteArray(value string, radix, width int) []byte {
	byteList := []byte{}
	for i := 0; i < len(value); i += width {
		endIndex := i + width
		if endIndex > len(value) {
			endIndex = len(value)
		}
		str := value[i:endIndex]
		intValue, err := strconv.ParseInt(str, radix, 0)
		if err != nil {
			return nil
		}
		byteValue := byte(intValue)
		byteList = append(byteList, byteValue)
	}
	return byteList
}

func RSAEncrypt_1(data []byte, key []byte, exponent string) ([]byte, error) {
	expBytes, err := hex.DecodeString(exponent)
	if err != nil {
		return nil, err
	}

	pubKey := &rsa.PublicKey{N: new(big.Int).SetBytes(key), E: int(new(big.Int).SetBytes(expBytes).Int64())}

	rsaLen := (pubKey.N.BitLen() + 7) / 8
	if len(data) > rsaLen-11 {
		blockCnt := (len(data) / (rsaLen - 11)) + func() int {
			if len(data)%(rsaLen-11) == 0 {
				return 0
			}
			return 1
		}()
		result := make([]byte, 0, rsaLen*blockCnt)
		for i := 0; i < blockCnt; i++ {
			blockSize := rsaLen - 11
			if i == blockCnt-1 {
				blockSize = len(data) - (i * blockSize)
			}
			temp := data[i*(rsaLen-11) : i*(rsaLen-11)+blockSize]
			cipherData, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, temp)
			if err != nil {
				return nil, err
			}
			result = append(result, cipherData...)
		}
		return result, nil
	} else {
		return rsa.EncryptPKCS1v15(rand.Reader, pubKey, data)
	}
}

func ToString(bytes []byte, radix, width int) string {
	if len(bytes) == 0 {
		return ""
	}
	builder := strings.Builder{}
	builder.Grow(len(bytes) * width)
	for _, b := range bytes {
		str := fmt.Sprintf("%X", b)
		str = strings.ToUpper(str)
		str = strings.Repeat("0", width-len(str)) + str
		builder.WriteString(str)
	}
	return builder.String()
}

// 已经验证
func BCDEncode(data []byte) string {
	bcdToAscii := make([]byte, len(data)*2)
	for i, b := range data {
		bcdToAscii[2*i] = b >> 4
		bcdToAscii[2*i+1] = b & 0xF
	}
	var strbcd = ToString(bcdToAscii, 16, 2)
	var hexBuf bytes.Buffer
	for i := 0; i < len(strbcd); i++ {
		if i%2 != 0 {
			hexBuf.WriteByte(strbcd[i])
		}
	}
	hexStr := hexBuf.String()

	return strings.ToUpper(hexStr)
}

func WXPayPasswordSign(password string) string {
	data := []byte(password)
	hash := md5.Sum(data)
	result := hex.EncodeToString(hash[:])
	upperStr := strings.ToUpper(result)
	message := fmt.Sprintf("%d%s", time.Now().Unix(), upperStr)
	//将得到的数据转化成字节数组
	byteResult := []byte(message)
	byteKey := ToByteArray("825de304a3c4da842dd4776a62a6f6218448c7295b111672fa9847e9403dcb36026f53f89f77d433e37b2ecd3e4e04f392b6096eebe739af69030f4713449649936bff52b9724b6934fad8af1e0dd35cb543ea96b1fe772cdf8569505c7bf645d1150c17e7ca3fd883b738ba3a93a0c930ab0029069a08482f37f2ef7c594339", 16, 2)
	jiami, _ := RSAEncrypt_1(byteResult, byteKey, "010001")
	strResult := BCDEncode(jiami)
	return strResult
}

func MD5(data []byte) []byte {
	hash := md5.Sum(data)
	return hash[:]
}

// TenPaySignDes3 支付相关的加密算法
func WXPaySign(srcData string) (string, error) {
	var md51 = MD5([]byte(srcData))
	var key = ToByteArray("3E952ABBACA5A7B067D23", 16, 2)
	bcd_to_ascii := make([]byte, len(md51)*2)
	for i := 0; i < len(md51); i++ {
		bcd_to_ascii[2*i] = byte(md51[i] >> 4)
	}
	encData, _ := baseutils.Encrypt3DESPassword(bcd_to_ascii, key)
	var result1 = ToString(encData, 16, 2)
	return result1, nil
}

// 确认支付
func paywxhb(D comm.LoginData, item ConfirmPreTransfer) (mm.HongBaoRes, error) {

	var req_text = "auto_deduct_flag=0&bank_type=" + item.BankType + "&nickname=" + D.NickName + "&bind_serial=" + item.BankSerial + "&busi_sms_flag=99&flag=30&passwd=" + WXPayPasswordSign(item.PayPassword) + "&pay_scene=379&req_key=" + item.ReqKey + "&use_touch=10"
	wcPaySign, err := WXPaySign(req_text)
	if err != nil {
		return mm.HongBaoRes{}, err
	}
	req_text += "&WCPaySign=" + wcPaySign

	//直接组包
	req := &mm.HongBaoReqPlus{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		CgiCmd:    proto.Int(0),
		OutPutTyp: proto.Int(1),
		ReqText: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len([]byte(req_text)))),
			Buffer: []byte(req_text),
		},
		ReqTextWx: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len([]byte("")))),
			Buffer: []byte(""),
		},
	}
	//序列化
	reqdata, _ := proto.Marshal(req)

	protobufdata, _, _, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/micromsg-bin/tenpay",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              385,
			Uin:              D.Uin,
			Cookie:           D.Cooike,
			Sessionkey:       D.Sessionkey,
			EncryptType:      5,
			Loginecdhkey:     D.Loginecdhkey,
			Clientsessionkey: D.Clientsessionkey,
			UseCompress:      false,
		},
	}, D.MmtlsKey)

	if err != nil {
		return mm.HongBaoRes{}, err
	}

	Response := mm.HongBaoRes{}
	err = proto.Unmarshal(protobufdata, &Response)
	return Response, nil
}

// 红包列表
func listwxhb(O receivewxhbParam, item HongBaoDetail) (mm.HongBaoRes, error) {

	if item.Size == 0 {
		item.Size = 10
	}
	//解析xml组合
	var HongBao Jxml.HongBao
	_ = xml.Unmarshal([]byte(O.Xml), &HongBao)

	// ReqText
	strReqText := string("")
	strReqText = strReqText + "channelId=1" + "&"
	strReqText = strReqText + "msgType=1" + "&"
	strReqText = strReqText + "nativeUrl=" + url.QueryEscape(HongBao.Appmsg.Wcpayinfo.Nativeurl) + "&province=&"
	strReqText = strReqText + "sendId=" + HongBao.Appmsg.Wcpayinfo.Paymsgid + "&"
	strReqText = strReqText + "limit=" + strconv.FormatInt(item.Size, 10) + "&"
	strReqText = strReqText + "offset=" + strconv.FormatInt(item.Offset, 10)

	D := O.D
	//直接组包
	req := &mm.HongBaoReq{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		CgiCmd:    proto.Int(3),
		OutPutTyp: proto.Int(1),
		ReqText: &mm.SKBuiltinBufferT{
			ILen:   proto.Uint32(uint32(len([]byte(strReqText)))),
			Buffer: []byte(strReqText),
		},
	}
	//序列化
	reqdata, _ := proto.Marshal(req)

	protobufdata, _, _, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/mmpay-bin/qrydetailwxhb",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              1585,
			Uin:              D.Uin,
			Cookie:           D.Cooike,
			Sessionkey:       D.Sessionkey,
			EncryptType:      5,
			Loginecdhkey:     D.Loginecdhkey,
			Clientsessionkey: D.Clientsessionkey,
			UseCompress:      false,
		},
	}, D.MmtlsKey)

	if err != nil {
		return mm.HongBaoRes{}, err
	}

	Response := mm.HongBaoRes{}
	err = proto.Unmarshal(protobufdata, &Response)

	return Response, nil
}

// 抢微信红包
func AutoHongBao(Data HongBaoParam) models.ResponseResult {
	D, err := comm.GetLoginatas(Data.Wxid)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	// 获取加密信息
	encrypt_userinfo := url.QueryEscape(D.Deviceid_str)
	encrypt_key := url.QueryEscape(string(D.Aeskey))

	//读取个人信息
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

	// 1 表示 个人 0 表示群红包
	InWay := "1"
	if Data.SendUserName != "" && Data.SendUserName != "string" {
		InWay = "0"
	}

	//先打开红包
	receivewxhb, err := receivewxhb(receivewxhbParam{
		Xml:              Data.Xml,
		D:                *D,
		City:             City,
		Province:         Province,
		Encrypt_key:      encrypt_key,
		Encrypt_userinfo: encrypt_userinfo,
		InWay:            InWay,
	})

	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("红包打开异常：%v", err.Error()),
			Data:    receivewxhb,
		}
	}

	if receivewxhb.GetErrorType() != 0 {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: fmt.Sprintf("红包打开异常：%v", receivewxhb.GetErrorMsg()),
			Data:    receivewxhb,
		}
	}

	//解析xml组合
	var HongBao Jxml.HongBao
	_ = xml.Unmarshal([]byte(Data.Xml), &HongBao)

	//解析打开红包后的json
	var receive receiveHongBao
	json.Unmarshal(receivewxhb.RetText.Buffer, &receive)

	sessionUserName := receive.SendUserName
	if Data.SendUserName != "" && Data.SendUserName != "string" {
		sessionUserName = Data.SendUserName
	}

	Text := "channelId=1&city=" + City + "&encrypt_key=" + encrypt_key + "&encrypt_userinfo=" + encrypt_userinfo + "&headImg=" + url.QueryEscape(Info.GetUserInfoExt().GetSmallHeadImgUrl()) + "&msgType=1&nativeUrl=" + url.QueryEscape(HongBao.Appmsg.Wcpayinfo.Nativeurl) + "&nickName=" + url.QueryEscape(Info.GetUserInfo().GetNickName().GetString_()) + "&province=" + Province + "&sendId=" + receive.SendId + "&sessionUserName=" + sessionUserName + "&timingIdentifier=" + receive.TimingIdentifier

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

	//序列化
	reqdata, _ := proto.Marshal(req)

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
			Loginecdhkey:     D.Loginecdhkey,
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
	ResponseOpen := mm.HongBaoRes{}
	err = proto.Unmarshal(protobufdata, &ResponseOpen)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}
	}

	if ResponseOpen.GetErrorType() != 0 {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: fmt.Sprintf("红包打开异常：%v", receivewxhb.GetErrorMsg()),
			Data:    receivewxhb,
		}
	}

	//解析打开红包后的json
	var resOpenWxhb receiveListHongBao
	json.Unmarshal(ResponseOpen.RetText.Buffer, &resOpenWxhb)
	// 保存抢红包信息
	comm.SetTodayMoney(D.Wxid, time.Now().Format("2006-01-02"), float64(resOpenWxhb.Amount), 1)

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    resOpenWxhb,
	}
}

// TenPaySignDes3 支付相关的加密算法
func TenPaySignDes3(srcData string, encKey string) (string, error) {
	srcMD5Bytes := []byte(baseutils.Md5ValueByte([]byte(srcData), true))
	keyMD5Bytes := baseutils.Md5ValueByte([]byte(encKey), true)
	desKey := baseutils.HexStringToBytes(keyMD5Bytes)

	encBytes := make([]byte, 0)
	for index := 0; index < 4; index++ {
		currentOffset := index * 8
		tmpSrcData := srcMD5Bytes[currentOffset : currentOffset+8]
		encData, err := baseutils.Encrypt3DES(tmpSrcData, desKey)
		if err != nil {
			return "", err
		}
		encBytes = append(encBytes, encData...)
	}
	return baseutils.BytesToHexString(encBytes, true), nil
}

// 确定收款
func SendTenPayRequest(D *comm.LoginData, reqItem *baseinfo.TenPayReqItem) models.ResponseResult {
	//直接组包
	req := &mm.TenPayRequest{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(0),
		},
		CgiCmd:     &reqItem.CgiCMD,
		OutPutType: proto.Uint32(1),
		ReqText: &mm.SKBuiltinString_S{
			ILen:   proto.Uint32(uint32(len(reqItem.ReqText))),
			Buffer: proto.String(reqItem.ReqText),
		},
		ReqTextWx: &mm.SKBuiltinString_S{
			ILen:   proto.Uint32(uint32(len(""))),
			Buffer: proto.String(""),
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
		Cgiurl: "/cgi-bin/micromsg-bin/tenpay",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              385,
			Uin:              D.Uin,
			Cookie:           D.Cooike,
			Sessionkey:       D.Sessionkey,
			EncryptType:      5,
			Loginecdhkey:     D.RsaPublicKey,
			Clientsessionkey: D.Clientsessionkey,
			UseCompress:      true,
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
	Response := mm.TenPayResponse{}
	err = proto.Unmarshal(protobufdata, &Response)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}
	}

	// 保存抢红包信息
	comm.SetTodayMoney(D.Wxid, time.Now().Format("2006-01-02"), getZZFee(Response), 2)

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    Response,
	}

}

// 解析转账金额
func getZZFee(response mm.TenPayResponse) float64 {
	jsonData := *response.ReqText.Buffer
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &obj)
	if err != nil {
		fmt.Println("--------------->发生异常了！！", err)
	}
	receFee, ok := obj["fee"].(float64)
	if !ok {
		fmt.Println("--------------->发生异常了！！", err)
	}
	return receFee

}

// 确认收款
func Collectmoney(req CollectmoneyModel) models.ResponseResult {

	D, err := comm.GetLoginata(req.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	//读取个人信息
	WxInfo := User.GetContractProfile(req.Wxid)

	if WxInfo.Code != 0 {
		return models.ResponseResult{
			Code:    WxInfo.Code,
			Success: false,
			Message: fmt.Sprintf("个人信息获取异常：%v", WxInfo.Message),
			Data:    WxInfo.Data,
		}
	}

	tenpayUrl := "invalid_time=" + req.InvalidTime + "&op=confirm&total_fee=0&trans_id=" + req.TransFerId + "&transaction_id=" + req.TransactionId + "&username=" + req.ToUserName
	wcPaySign, err := TenPaySignDes3(tenpayUrl, "%^&*Tenpay!@#$")
	if err != nil {
		return models.ResponseResult{
			Code:    WxInfo.Code,
			Success: false,
			Message: fmt.Sprintf("支付sign错误：%v", WxInfo.Message),
			Data:    WxInfo.Data,
		}
	}
	tenpayUrl += "&WCPaySign=" + wcPaySign
	//构建请求
	reqItem := &baseinfo.TenPayReqItem{
		CgiCMD:  85,
		ReqText: tenpayUrl,
	}

	return SendTenPayRequest(D, reqItem)
}

// 创建红包
func WXCreateRedPacketApi(Data RedPacket) models.ResponseResult {
	D, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	//创建红包
	receivewxhb, err := createwxhb(*D, Data)

	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("查看红包异常：%v", err.Error()),
			Data:    receivewxhb,
		}
	}

	if receivewxhb.GetErrorType() != 0 {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: fmt.Sprintf("查看打开异常~：%v", receivewxhb.GetErrorMsg()),
			Data:    receivewxhb,
		}
	}

	//解析打开红包后的json
	var receive createHongBao
	json.Unmarshal(receivewxhb.RetText.Buffer, &receive)

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    receive,
	}
}

// 确认支付
func ConfirmPreTransferApi(Data ConfirmPreTransfer) models.ResponseResult {
	D, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	//确认支付
	receivewxhb, err := paywxhb(*D, Data)

	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("查看红包异常：%v", err.Error()),
			Data:    receivewxhb,
		}
	}

	if receivewxhb.GetErrorType() != 0 {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: fmt.Sprintf("查看打开异常~：%v", receivewxhb.GetErrorMsg()),
			Data:    receivewxhb,
		}
	}

	//解析打开红包后的json
	var receive payHongBao
	json.Unmarshal(receivewxhb.RetText.Buffer, &receive)

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    receive,
	}
}

// 查看红包领取列表
func GetRedPacketListApi(Data HongBaoDetail) models.ResponseResult {
	D, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	encrypt_userinfo := url.QueryEscape(D.Deviceid_str)
	encrypt_key := url.QueryEscape(string(D.Aeskey))

	//读取个人信息
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

	//得到红包列表
	receivewxhb, err := listwxhb(receivewxhbParam{
		Xml:              Data.Xml,
		D:                *D,
		City:             City,
		Province:         Province,
		Encrypt_key:      encrypt_key,
		Encrypt_userinfo: encrypt_userinfo,
	}, Data)

	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("查看红包异常：%v", err.Error()),
			Data:    receivewxhb,
		}
	}

	if receivewxhb.GetErrorType() != 0 {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: fmt.Sprintf("查看打开异常~：%v", receivewxhb.GetErrorMsg()),
			Data:    receivewxhb,
		}
	}

	//解析打开红包后的json
	var receive receiveListHongBao
	json.Unmarshal(receivewxhb.RetText.Buffer, &receive)

	fmt.Println(receive)

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    receive,
	}
}
