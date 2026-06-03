package TenPay

import (
	"bytes"
	"crypto/des"
	"crypto/md5"
	"errors"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"

	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"

	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/User"
)

type GeMaPayQCodeParam struct {
	Name  string
	Money string
	Wxid  string
}

func GeneratePayQCode2(req GeMaPayQCodeParam) models.ResponseResult {

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

	var tenpayUrl = "delay_confirm_flag=0&desc=" + req.Name + "&fee=" + req.Money + "&fee_type=CNY&pay_scene=31&receiver_name=" + D.Wxid + "&scene=31&transfer_scene=2"

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
	reqItem := TenPayReqItem{
		CgiCMD:  94,
		ReqText: tenpayUrl,
	}

	return SendTenPayRequest2(D, reqItem)

}

type TenPayReqItem struct {
	CgiCMD  uint32
	ReqText string
}

func Md5ValueByte(data []byte, bUpper bool) string {
	has := md5.Sum(data)
	tmpString := "%x"
	if bUpper {
		tmpString = "%X"
	}
	md5str := fmt.Sprintf(tmpString, has)
	return md5str
}
func StringCut(srcStr string, index uint32, length uint32) string {
	srcBytes := []byte(srcStr)
	tmpBytes := make([]byte, 0)
	tmpBytes = append(tmpBytes, srcBytes[index:index+length]...)

	retString := string(tmpBytes)
	return retString
}
func HexStringToBytes(hexString string) []byte {
	retBytes := make([]byte, 0)
	count := len(hexString)
	for index := 0; index < count; index += 2 {
		tmpStr := StringCut(hexString, uint32(index), 2)
		value64, _ := strconv.ParseInt(tmpStr, 16, 16)
		retBytes = append(retBytes, byte(value64))
	}

	return retBytes
}
func EncryptDESECB(data []byte, keyByte []byte) ([]byte, error) {
	block, err := des.NewCipher(keyByte)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	//对明文数据进行补码
	data = PKCS5Padding(data, bs)
	if len(data)%bs != 0 {
		return nil, errors.New("Need a multiple of the blocksize")
	}
	out := make([]byte, len(data))
	dst := out
	for len(data) > 0 {
		//对明文按照blocksize进行分块加密
		//必要时可以使用go关键字进行并行加密
		block.Encrypt(dst, data[:bs])
		data = data[bs:]
		dst = dst[bs:]
	}
	return out, nil
}
func PKCS5Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}
func DecryptDESECB(data []byte, key []byte) ([]byte, error) {
	if len(key) > 8 {
		key = key[:8]
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	if len(data)%bs != 0 {
		return nil, errors.New("DecryptDES crypto/cipher: input not full blocks")
	}
	out := make([]byte, len(data))
	dst := out
	for len(data) > 0 {
		block.Decrypt(dst, data[:bs])
		data = data[bs:]
		dst = dst[bs:]
	}
	// out = PKCS5UnPadding(out)
	return out, nil
}
func Encrypt3DES(srcData []byte, key []byte) ([]byte, error) {
	if len(srcData) != 8 || len(key) != 16 {
		return nil, errors.New("Encrypt3DES err: srcLen != 8 || keyLen != 16")
	}
	tmpSrcData := make([]byte, 0)
	tmpSrcData = append(tmpSrcData, srcData...)
	encData, err := EncryptDESECB(tmpSrcData, key[0:8])
	if err != nil {
		return nil, err
	}
	decData, err := DecryptDESECB(encData, key[8:])
	if err != nil {
		return nil, err
	}
	encData, err = EncryptDESECB(decData[0:8], key[0:8])
	if err != nil {
		return nil, err
	}
	return encData[0:8], err
}
func BytesToHexString(data []byte, isBig bool) string {
	changeBytes := numberHexSmall
	if isBig {
		changeBytes = numberHexBig
	}
	length := len(data)
	retBytes := make([]byte, length*2)
	for index := 0; index < length; index++ {
		tmpByte := data[index]
		highIndex := ((tmpByte & 0xf0) >> 4)
		lowIndex := tmpByte & 0x0f
		retBytes[index*2] = changeBytes[highIndex]
		retBytes[index*2+1] = changeBytes[lowIndex]
	}

	return string(retBytes)
}

// 转换16进制
var numberHexSmall = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
var numberHexBig = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

func SendTenPayRequest2(D *comm.LoginData, reqItem TenPayReqItem) models.ResponseResult {
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

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    Response,
	}

}
