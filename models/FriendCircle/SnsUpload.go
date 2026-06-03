package FriendCircle

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/baseinfo"
	"wechatdll/clientsdk"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/comm"
	"wechatdll/models"
)

type SnsUploadParam struct {
	Wxid   string
	Base64 string
}

type DownloadMediaModel struct {
	Key  string
	Url  string
	Wxid string
}
type SnsVideoDownloadItem struct {
	Seq           uint32             // 代表第几个请求
	URL           string             // 视频加密地址
	RangeStart    uint32             // 起始地址
	RangeEnd      uint32             // 结束地址
	XSnsVideoFlag string             // 视频标志
	CDNDns        mm.CDNDnsInfo // DNS信息
}
type cdnInfo struct {
	snsDns  *mm.CDNDnsInfo
	appDns  *mm.CDNDnsInfo
	cdnDns  *mm.CDNDnsInfo
	fakeDns *mm.CDNDnsInfo
}
func SnsUpload(Data SnsUploadParam) models.ResponseResult {
	var err error
	var protobufdata []byte
	var errtype int64
	var Bs64Data []byte

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

	Base64Data := strings.Split(Data.Base64, ",")

	if len(Base64Data) > 1 {
		Bs64Data, _ = base64.StdEncoding.DecodeString(Base64Data[1])
	} else {
		Bs64Data, _ = base64.StdEncoding.DecodeString(Data.Base64)
	}

	Stream := bytes.NewBuffer(Bs64Data)

	Bs64MD5 := baseutils.GetFileMD5Hash(Bs64Data)

	Startpos := 0
	datalen := 50000
	datatotalength := Stream.Len()

	ClientImgId := fmt.Sprintf("%v_%v", Data.Wxid, time.Now().Unix())

	I := 0

	for {
		Startpos = I * datalen
		count := 0
		if datatotalength-Startpos > datalen {
			count = datalen
		} else {
			count = datatotalength - Startpos
		}
		if count < 0 {
			break
		}

		Databuff := make([]byte, count)
		_, _ = Stream.Read(Databuff)

		req := &mm.SnsUploadRequest{
			BaseRequest: &mm.BaseRequest{
				SessionKey:    D.Sessionkey,
				Uin:           proto.Uint32(D.Uin),
				DeviceId:      D.Deviceid_byte,
				ClientVersion: proto.Int32(int32(D.ClientVersion)),
				DeviceType:    []byte(D.DeviceType),
				Scene:         proto.Uint32(0),
			},
			Type:     proto.Uint32(2),
			StartPos: proto.Uint32(uint32(Startpos)),
			TotalLen: proto.Uint32(uint32(datatotalength)),
			Buffer: &mm.SKBuiltinBufferT{
				ILen:   proto.Uint32(uint32(len(Databuff))),
				Buffer: Databuff,
			},
			ClientId: proto.String(ClientImgId),
			MD5:      proto.String(Bs64MD5),
		}

		//序列化
		reqdata, _ := proto.Marshal(req)

		//发包
		protobufdata, _, errtype, err = comm.SendRequest(comm.SendPostData{
			Ip:     D.Mmtlsip,
			Host:   D.ShortHost,
			Cgiurl: "/cgi-bin/micromsg-bin/mmsnsupload", ///cgi-bin/micromsg-bin/uploadvideo
			Proxy:  D.Proxy,
			PackData: Algorithm.PackData{
				Reqdata:          reqdata,
				Cgi:              207,
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
			break
		}

		I++
	}

	if err != nil {
		return models.ResponseResult{
			Code:    errtype,
			Success: false,
			Message: err.Error(),
			Data:    nil,
		}
	}

	//解包
	Response := mm.SnsUploadResponse{}
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
func SendCdnSnsVideoDownloadReuqest(req DownloadMediaModel) models.ResponseResult{
	retFileData := []byte{}
	lessLength := uint32(2000000)
	encLen := uint32(0)
	videoFlag := string("V2")
	tmpEncKey, _ := strconv.Atoi(req.Key)
	retryCount := uint32(0)
	D, err := comm.GetLoginatas(req.Wxid)
	var protobufdata []byte
	var errtype int64
	var cdnInfos *cdnInfo
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", req.Wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("异常：%v", err.Error())
		}
		return models.ResponseResult{
			Code:    -7,
			Success: false,
			Message: errorMsg,
			Data:    nil,
		}
	}
	video, _ := base64.StdEncoding.DecodeString(req.Url)
	videoUrl := string(video)
	for {
		// 生产SnsImgItem
		var snsVideoItem baseinfo.SnsVideoDownloadItem
		snsVideoItem.Seq = uint32(rand.Intn(10))
		snsVideoItem.URL = videoUrl
		snsVideoItem.RangeStart = uint32(len(retFileData))
		snsVideoItem.RangeEnd = snsVideoItem.RangeStart + lessLength
		snsVideoItem.XSnsVideoFlag = videoFlag

		req := &mm.GetCDNDnsRequest{
			BaseRequest: &mm.BaseRequest{
				SessionKey:    []byte{},
				Uin:           proto.Uint32(D.Uin),
				DeviceId:      D.Deviceid_byte,
				ClientVersion: proto.Int32(int32(D.ClientVersion)),
				DeviceType:    []byte(D.DeviceType),
				Scene:         proto.Uint32(0),
			},
			ClientIp: proto.String(""),
		}
		resp := &mm.GetCDNDnsResponse{}
		//序列化

		reqdata, _ := proto.Marshal(req)
		protobufdata, _, errtype, err = comm.SendRequest(comm.SendPostData{
			Ip:     D.Mmtlsip,
			Host:   D.ShortHost,
			Cgiurl: "/cgi-bin/micromsg-bin/getcdndns",
			Proxy:  D.Proxy,
			PackData: Algorithm.PackData{
				Reqdata:          reqdata,
				Cgi:              379,
				Uin:              D.Uin,
				Cookie:           D.Cooike,
				Sessionkey:       D.Sessionkey,
				EncryptType:      5,
				Loginecdhkey:     D.RsaPublicKey,
				Clientsessionkey: D.Clientsessionkey,
				UseCompress:      true,
			},
		}, D.MmtlsKey)
		err = proto.Unmarshal(protobufdata, resp)
		if err != nil {
			fmt.Println(errtype)
		}
		cdnInfos = &cdnInfo{
			appDns:  resp.AppDnsInfo,
			snsDns:  resp.SnsDnsInfo,
			cdnDns:  resp.DnsInfo,
			fakeDns: resp.FakeDnsInfo,
		}

		// 发送分片下载请求
		response, err := SendCdnSnsVideoDownloadReuqestPiece(D,cdnInfos, &snsVideoItem)
		if err != nil {
			fmt.Printf("cdn返回错误%v", err.Error())
			if retryCount < 3 {
				retryCount++
				continue
			}
			return models.ResponseResult{
				Code:    -8,
				Success: false,
				Message: "errcode",
				Data:    nil,
			}
		}
		retryCount = 0
		// 判断错误码
		if response.RetCode != 0 {
			return models.ResponseResult{
				Code:    -9,
				Success: false,
				Message: "errcode",
				Data:    nil,
			}
		}
		fmt.Println("88888")
		// 设置加密的字节数
		if encLen == 0 {
			encLen = response.XEncLen
		}

		// 合并数据
		retFileData = append(retFileData, response.FileData[0:]...)
		currentLen := uint32(len(retFileData))
		if currentLen >= response.TotalSize {
			break
		}

		// 如果没有读取完
		lessLength = response.TotalSize - currentLen
		videoFlag = response.XSnsVideoFlag
	}
	if tmpEncKey != 0 {
		retFileData = baseutils.DecryptSnsVideoData(retFileData, encLen, uint64(tmpEncKey))
	}
	// ioutil.WriteFile("log/1.mp4", retFileData, 0777)
	// 解密数据
	//return retFileData, nil
	fmt.Println(base64.StdEncoding.EncodeToString(retFileData))
	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: base64.StdEncoding.EncodeToString(retFileData),
		Data:    nil,
	}
}

// SendCdnSnsVideoDownloadReuqestPiece 分片下载
func SendCdnSnsVideoDownloadReuqestPiece(userInfo *comm.LoginData,info *cdnInfo, snsVideoItem *baseinfo.SnsVideoDownloadItem) (*baseinfo.CdnSnsVideoDownloadResponse, error) {
	// 创建朋友圈视频下载请求
	fmt.Println(1)
	request, err := CreateSnsVideoDownloadRequest(userInfo, info,snsVideoItem)
	if err != nil {
		fmt.Sprintf("创建朋友圈视频错误：%v", err.Error())
		return nil, err
	}
	jsonData, err := json.Marshal(request)
	fmt.Println(string(jsonData))

	// 打包请求
	sendData := clientsdk.PackCdnSnsVideoDownloadRequest(request)
	// 连接Cdn服务器
	serverIP := *info.snsDns.ZoneIPList[0].String_
	serverPort := info.snsDns.ZoneIPPortList[0].PortList[0]
	conn, err := ConnectCdnServer(serverIP, serverPort)
	if err != nil {
	   fmt.Sprintf("cdn错误：%v", err.Error())
		return nil, err
	}
	// 发送数据
	conn.Write(sendData)
	defer conn.Close()

	// 接收响应信息
	// 接收响应信息，解析
	retData := CDNRecvData(conn)
	response, err := clientsdk.DecodeSnsVideoDownloadResponse(retData)
	if err != nil {
		return nil, err
	}
	// 判断错误码
	if response.RetCode != 0 {
		return nil, errors.New("下载朋友圈视频失败: ErrCode = " + clientsdk.GetErrStringByRetCode(response.RetCode))
	}
	return response, nil
}
// CreateSnsVideoDownloadRequest 创建Cdn下载朋友圈视频请求
func CreateSnsVideoDownloadRequest(userInfo *comm.LoginData, info *cdnInfo,snsVideoItem *baseinfo.SnsVideoDownloadItem) (*baseinfo.CdnSnsVideoDownloadRequest, error) {
	request := &baseinfo.CdnSnsVideoDownloadRequest{}
	request.Ver = 1
	request.WeiXinNum = uint32(info.snsDns.GetUin())
	request.Seq = snsVideoItem.Seq
	request.ClientVersion = uint32(userInfo.ClientVersion)

	if !(int(request.ClientVersion) > 0) {
		request.ClientVersion = baseinfo.ClientVersion
	}
	if userInfo.DeviceInfo == nil {
		request.ClientOsType = Algorithm.AndroidDeviceType
	} else {
		request.ClientOsType = userInfo.DeviceInfo.OsType
	}
	request.AuthKey = info.snsDns.AuthKey.GetBuffer()
	request.NetType = 1
	request.AcceptDupack = 1
	request.Signal = ""
	request.Scene = ""
	request.URL = snsVideoItem.URL
	request.RangeStart = snsVideoItem.RangeStart
	request.RangeEnd = snsVideoItem.RangeEnd
	request.LastRetCode = 0
	request.IPSeq = 0
	request.RedirectType = 0
	request.LastVideoFormat = 0
	request.VideoFormat = 2
	request.XSnsVideoFlag = snsVideoItem.XSnsVideoFlag
	return request, nil
}
// ConnectCdnServer 链接Cdn服务器
func ConnectCdnServer(ipAddress string, port uint32) (*net.TCPConn, error) {
	fmt.Println(ipAddress)
	strPort := strconv.Itoa(int(port))
	serverAddr := ipAddress + ":" + strPort

	fmt.Printf("cdn服务器地址%v:",serverAddr)
	tcpAddr, err := net.ResolveTCPAddr("tcp", serverAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
// CDNRecvData 发送Cdn数据
func CDNRecvData(conn *net.TCPConn) []byte {
	// 写数据
	// 接收数据
	retData := make([]byte, 0)
	buffer := make([]byte, 25)
	count, err := conn.Read(buffer)
	if err != nil {
		return []byte{}
	}

	// 读取返回数据
	retData = append(retData, buffer[0:count]...)
	// 数据总长度
	totalLength := ParseCdnResponseDataLength(retData)
	currentLength := uint32(len(retData))
	for currentLength < totalLength {
		lessCount := totalLength - currentLength
		buffer := make([]byte, lessCount)
		count, err := conn.Read(buffer)
		if err != nil {
			return []byte{}
		}
		if count > 0 {
			retData = append(retData, buffer[0:count]...)
			currentLength = uint32(len(retData))
		} else {
			break
		}
	}

	return retData
}
func ParseCdnResponseDataLength(data []byte) uint32 {
	totalLength := baseutils.BytesToInt32(data[1:5])
	return totalLength
}