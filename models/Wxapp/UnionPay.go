package Wxapp

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm2"
	"wechatdll/comm"
	"wechatdll/models"

	"github.com/forgoer/openssl"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

// UnionpayData 结构体包含GetOrder和云闪付接口所需的参数
type UnionpayData struct {
	AppId     string `json:"AppId"`
	NonceStr  string `json:"NonceStr"`
	TimeStamp string `json:"TimeStamp"`
	Package_  string `json:"Package"`
	PaySign   string `json:"PaySign"`
	Wxid      string `json:"Wxid"`
	SignType  string `json:"SignType"`
}

type PaymentResult struct {
	UnionApple   string `json:"UnionApple"`
	UnionAndroid string `json:"UnionAndroid"`
	UnionTn      string `json:"UnionTn"`
	WxPayKey     string `json:"WxPayKey"`
}

func Unionpay(Data UnionpayData) models.ResponseResult {
	D, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}
	req := &mm2.GetOrder{
		BaseRequest: &mm2.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(46),
		},
		Tmp2: proto.Uint64(0),
		Tmp3: proto.String("nonce"),
		Tmp4: &mm2.Tmp4{
			Tmp4_1: &mm2.Tmp4_1{
				BaseRequest: &mm2.BaseRequest{
					SessionKey:    D.Sessionkey,
					Uin:           proto.Uint32(D.Uin),
					DeviceId:      D.Deviceid_byte,
					ClientVersion: proto.Int32(int32(D.ClientVersion)),
					DeviceType:    []byte(D.DeviceType),
					Scene:         proto.Uint32(46),
				},
				Payparams: &mm2.Payparams{
					AppId:     proto.String(Data.AppId),
					NonceStr:  proto.String(Data.NonceStr),
					TimeStamp: proto.String(Data.TimeStamp),
					Package_:  proto.String(Data.Package_),
					PaySign:   proto.String(Data.PaySign),
					SignType:  proto.String(Data.SignType),
				},
			},
			Tmp4_2: proto.String(""),
			Tmp4_3: proto.Uint64(0),
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

	protobufdata, _, errtype, err := comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/mmpay-bin/tinyapppay",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              2576,
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
	// fmt.Println("TinyAppPay", hex.EncodeToString(protobufdata))
	//解包
	Response := mm2.GetOrderResponse{}
	err = proto.Unmarshal(protobufdata, &Response)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}
	}
	ErrMsg := Response.BaseResponse.ErrMsg.Value
	GetOrderResponse316 := Response.GetOrderResponse_3.GetOrderResponse_3_1.GetOrderResponse_3_1_6
	// ReqKey := Response.GetOrderResponse_3.GetOrderResponse_3_1.GetOrderResponse_3_1_6.ReqKey
	if GetOrderResponse316 == nil {
		return models.ResponseResult{
			Code:    -1,
			Success: false,
			Message: *ErrMsg,
			Data:    nil,
		}
	}
	ReqKey := *GetOrderResponse316.ReqKey
	uuid := uuid.New()

	req2 := &mm2.Toysf{
		BaseRequest: &mm2.BaseRequest{
			SessionKey:    D.Sessionkey,
			Uin:           proto.Uint32(D.Uin),
			DeviceId:      D.Deviceid_byte,
			ClientVersion: proto.Int32(int32(D.ClientVersion)),
			DeviceType:    []byte(D.DeviceType),
			Scene:         proto.Uint32(46),
		},
		Toysf_2: proto.Uint64(0),
		Toysf_3: proto.String("nonce"),
		Toysf_4: &mm2.Toysf_4{
			Toysf_4_1: &mm2.Toysf_4_1{
				BaseRequest: &mm2.BaseRequest{
					SessionKey:    D.Sessionkey,
					Uin:           proto.Uint32(D.Uin),
					DeviceId:      D.Deviceid_byte,
					ClientVersion: proto.Int32(int32(D.ClientVersion)),
					DeviceType:    []byte(D.DeviceType),
					Scene:         proto.Uint32(46),
				},
				Toysf_4_1_2: &mm2.Toysf_4_1_2{
					UUID1:         proto.String(uuid.String()),
					Toysf_4_1_2_2: proto.Uint64(0),
					UUID2:         proto.String(uuid.String()),
				},
				Key: proto.String(ReqKey),
			},
			Toysf_4_2: proto.String(""),
			Toysf_4_3: proto.Uint64(0),
		},
	}
	reqdata, err = proto.Marshal(req2)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
	}
	protobufdata, _, errtype, err = comm.SendRequest(comm.SendPostData{
		Ip:     D.Mmtlsip,
		Host:   D.ShortHost,
		Cgiurl: "/cgi-bin/mmpay-bin/yunshanfuordered",
		Proxy:  D.Proxy,
		PackData: Algorithm.PackData{
			Reqdata:          reqdata,
			Cgi:              2576,
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
	// fmt.Println("UnionPay", hex.EncodeToString(protobufdata))
	Response2 := mm2.ToysfResponse{}
	err = proto.Unmarshal(protobufdata, &Response2)
	fmt.Println(err)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("反序列化失败：%v", err.Error()),
			Data:    nil,
		}
	}
	tn := Response2.ToysfResponse_3.ToysfResponse_3_1.Tn
	errMsg := Response2.BaseResponse.ErrMsg.Value
	if tn == nil {
		return models.ResponseResult{
			Code:    -1,
			Success: false,
			Message: *errMsg,
			Data:    nil,
		}
	}
	android_pay := UnionPayAndroid(*tn)
	apple_pay := UnionPayApple(*tn)
	paymentData := PaymentResult{
		UnionApple:   apple_pay,
		UnionAndroid: android_pay,
		UnionTn:      *tn,
		WxPayKey:     ReqKey,
	}
	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    paymentData,
	}
}

func UnionPayApple(tn string) string {
	key := []byte("002023102420002800202310")
	data := map[string]interface{}{"scheme": nil, "tn": tn, "mode": "00", "merchantMode": "01"}
	jsonData, _ := json.Marshal(data)
	encodeData, _ := openssl.Des3ECBEncrypt([]byte(jsonData), key, openssl.PKCS7_PADDING)
	// block, _ := des.NewTripleDESCipher(key)
	// padding := des.BlockSize - (len(jsonData) % des.BlockSize)
	// padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	// paddedMessage := append(jsonData, padtext...)
	// ciphertext := make([]byte, len(paddedMessage))
	// for i := 0; i < len(paddedMessage); i += des.BlockSize {
	// 	end := i + des.BlockSize
	// 	block.Encrypt(ciphertext[i:end], paddedMessage[i:end])
	// }
	// hexCiphertext := hex.EncodeToString(ciphertext)
	hexCiphertext := hex.EncodeToString(encodeData)
	payLink := fmt.Sprintf("uppaywallet://uppay?paydata=%s&s=26732007520004206700", hexCiphertext)
	fmt.Println(payLink)
	return payLink
}

func UnionPayAndroid(unionTn string) string {
	unionPre := "upwrp://uppayservice/?style=token&paydata="
	unionMask := fmt.Sprintf("tn=%s,resultURL=exit,scheme=,packageName=,usetestmode=false", unionTn)
	unionBase64 := base64.StdEncoding.EncodeToString([]byte(unionMask))
	unionPay := unionPre + unionBase64
	fmt.Println(unionPay)
	return unionPay
}
