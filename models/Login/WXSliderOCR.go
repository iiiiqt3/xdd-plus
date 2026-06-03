package Login

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"wechatdll/comm"
	"wechatdll/models"

	"github.com/astaxie/beego"
)

type OCRXMLResponse struct {
	Url string `xml:"Url"`
}

type SmsData struct {
	Sessionid    string `json:"sessionid"`
	Qrcodeticket string `json:"qrcodeticket"`
	Mobile       string `json:"mobile"`
	Cc           string `json:"cc"`
}

type ErrMsg struct {
	XMLName      xml.Name `xml:"e"`
	ShowType     int      `xml:"ShowType"`
	Content      string   `xml:",Content"`
	Url          string   `xml:",Url"`
	DispSec      int      `xml:",DispSec"`
	Title        string   `xml:",Title"`
	Action       int      `xml:",Action"`
	DelayConnSec int      `xml:",DelayConnSec"`
	Countdown    int      `xml:",Countdown"`
	Ok           string   `xml:",Ok"`
	Cancel       string   `xml:",Cancel"`
}

/*
	type SilderOCR struct {
		Flag    int    `json:"flag"`
		Data    string `json:"data"`
		Code    int    `json:"code"`
		Message string `json:"message"`
		URL     string `json:"url"`
		Remark  string `json:"remark"`
	}
*/
type SilderOCR struct {
	Flag    int    `json:"flag"`
	Data    string `json:"data"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	URL     string `json:"url"`
	Remark  string `json:"remark"`
	Success bool   `json:"Success"`
}

type OCRXMLErrMsg struct {
	Content string `xml:"Content"`
	Url     string `xml:"Url"`
}

// 0 滑块成功 -1滑块失败
//
//	func LoginOCR(Data string) error {
//		println(Data)
//		var XmlData ErrMsg
//		err := xml.Unmarshal([]byte(Data), &XmlData)
//		if err != nil {
//			return err
//		}
//		if XmlData.Url == "" {
//			return err
//		}
//		Parameter := Getparameter(XmlData.Url)
//		if Parameter["secticket"] == "" {
//			return err
//		}
//		flag := WxSliderOCRRequest("2000000038", Parameter["secticket"]).Flag
//		if flag == 0 {
//			return nil
//		} else {
//			return errors.New("自动过滑块失败")
//		}
//	}
func LoginOCR(xmlData string) error {
	var parsed OCRXMLResponse
	// 1. 解析XML
	err := xml.Unmarshal([]byte(xmlData), &parsed)
	if err != nil {
		return fmt.Errorf("解析XML失败: %w", err)
	}
	if parsed.Url == "" {
		return errors.New("未找到Url字段")
	}
	fullUrl := parsed.Url
	encodedUrl := url.QueryEscape(fullUrl)
	result := WxSliderOCRRequest(encodedUrl)
	//fmt.Printf("OCR处理结果: %+v\n", result)

	if result.Success {
		fmt.Println("✅ 自动过滑块成功")
		return nil
	} else {
		return errors.New("❌ 自动过滑块失败")
	}
}

// 死号强开
func LoginOCRS(data string) error {
	var parsedXML OCRXMLErrMsg
	err := xml.Unmarshal([]byte(data), &parsedXML)
	if err != nil {
		fmt.Println("❌ XML解析失败:", err)
		return fmt.Errorf("XML解析失败: %w", err)
	}
	if parsedXML.Url == "" {
		fmt.Println("❌ XML中未找到Url字段")
		return errors.New("XML中未找到Url字段")
	}

	params := ParseUrlParameters(parsedXML.Url)
	ticket, ok := params["ticket"]
	if !ok || ticket == "" {
		fmt.Println("❌ URL中未提取到ticket参数")
		return errors.New("URL中未提取到ticket参数")
	}

	// ⏱ 最多尝试3次
	for i := 1; i <= 3; i++ {
		fmt.Printf("🔁 第 %d 次尝试强开...\n", i)
		result := WxSliderOCRRequestS(ticket)
		fmt.Printf("OCR结果: %+v\n", result)

		if result.Message == "强开成功" {
			fmt.Println("✅ 强开成功")
			return nil
		}
		time.Sleep(500 * time.Millisecond) // 可选：稍微等待再试
	}

	return errors.New("强开死号失败")
}

func ParseUrlParameters(url string) map[string]string {
	result := make(map[string]string)

	index := strings.Index(url, "?")
	if index == -1 {
		return result
	}

	params := url[index+1:]
	pairs := strings.Split(params, "&")
	for _, pair := range pairs {
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}

	return result
}

// 死号 go滑
func WxSliderOCRRequestS(ticket string) SilderOCR {
	ocrUrl := beego.AppConfig.String("ocrurlgo") // 应配置为 http://47.119.158.126:5550/unban?tick=
	fullUrl := ocrUrl + ticket

	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		fmt.Println("请求创建失败:", err)
		return SilderOCR{Message: "请求失败"}
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "http://47.119.158.126:5550/")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Proxy-Connection", "keep-alive")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("请求发送失败:", err)
		return SilderOCR{Message: "请求失败"}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return SilderOCR{Message: "读取失败"}
	}
	//fmt.Println("返回原始内容：", string(body))
	// 默认返回结构
	var result SilderOCR
	_ = json.Unmarshal(body, &result) // 忽略结构不符问题，下面单独解析 message 字段

	// 尝试手动解析 message 字段
	var msgMap map[string]interface{}
	if err := json.Unmarshal(body, &msgMap); err == nil {
		if msg, ok := msgMap["message"].(string); ok {
			result.Message = msg
			if msg == "强开成功" {
				result.Success = true
			}
		}
	}

	return result
}

// 62登录短信辅助
func WeChatSMS(Data string, ua string, proxyAddr string, proxyUser string, proxyPass string) (checkUrl, againUrl, setCookie string) {
	var XmlData ErrMsg
	xml.Unmarshal([]byte(Data), &XmlData)
	Parameter := Getparameter(XmlData.Url)
	ticket := Parameter["ticket"]
	idc := Parameter["idc"]
	headers := &map[string]string{
		"Cookie": setCookie,
	}
	setCookie += comm.HttpGetAndSetCookie("https://shminorshort.weixin.qq.com/security/readtemplate?t=login_verify_entrances/intro&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc, headers, ua, proxyAddr, proxyUser, proxyPass)
	//获取sessionId
	headers = &map[string]string{
		"Cookie": setCookie,
	}
	sessionId := comm.HttpGetAndSetCookie("https://shminorshort.weixin.qq.com/security/secondauth?t=login_verify_entrances/intro&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&step=41", headers, ua, proxyAddr, proxyUser, proxyPass)
	setCookie += sessionId
	sessionId = strings.Trim(strings.Split(sessionId, "=")[1], ";")
	headers = &map[string]string{
		"Cookie": setCookie,
	}
	comm.HttpGet("https://shminorshort.weixin.qq.com/security/readtemplate?&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&t=login_verify_entrances/entrances", headers, ua, proxyAddr, proxyUser, proxyPass)
	step3Ret := comm.HttpGet("https://shminorshort.weixin.qq.com/security/secondauth?&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&t=login_verify_entrances/entrances&step=1&sessionid="+sessionId, headers, ua, proxyAddr, proxyUser, proxyPass)

	var NewSessionIdJson SmsData
	json.Unmarshal([]byte(step3Ret), &NewSessionIdJson)
	setCookie = strings.Replace(setCookie, sessionId, NewSessionIdJson.Sessionid, -1)
	sessionId = NewSessionIdJson.Sessionid
	cc := NewSessionIdJson.Cc
	Mobile := NewSessionIdJson.Mobile
	headers = &map[string]string{
		"Cookie": setCookie,
	}
	comm.HttpGet("https://shminorshort.weixin.qq.com/security/secondauth?&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&t=login_verify_entrances/entrances&step=31&sessionid="+sessionId, headers, ua, proxyAddr, proxyUser, proxyPass)
	comm.HttpGet("https://shminorshort.weixin.qq.com/security/readtemplate?t=login_verify_entrances/sms&&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&type=down&cc="+cc+"&mobile="+Mobile, headers, ua, proxyAddr, proxyUser, proxyPass)

	checkUrl = "https://shminorshort.weixin.qq.com/security/secondauth?t=login_verify_entrances/sms&&ticket=" + ticket + "&wechat_real_lang=zh_CN&idc=" + idc + "&type=down&cc=" + cc + "&mobile=" + Mobile + "&sessionid=" + sessionId + "&step=32&verifycode=[[[verifycode]]]"
	againUrl = "https://shminorshort.weixin.qq.com/security/secondauth?t=login_verify_entrances/sms&&ticket=" + ticket + "&wechat_real_lang=zh_CN&idc=" + idc + "&type=down&cc=" + cc + "&mobile=" + Mobile + "&sessionid=" + sessionId + "&step=31"

	return checkUrl, againUrl, setCookie

}

func WechatSMS1(Data string, ua string, proxy models.ProxyInfo) (checkUrl, againUrl, setCookie string) {
	return WeChatSMS(Data, ua, proxy.ProxyIp, proxy.ProxyUser, proxy.ProxyPassword)

}

// 62登录二维码辅助
func WeChatQrCode(Data string, ua string, proxyAddr string, proxyUser string, proxyPass string) (QrUrl, checkUrl string) {
	var XmlData ErrMsg
	xml.Unmarshal([]byte(Data), &XmlData)
	Parameter := Getparameter(XmlData.Url)
	ticket := Parameter["ticket"]
	idc := Parameter["idc"]
	setCookie := ""
	setCookie += comm.HttpGetAndSetCookie("https://shminorshort.weixin.qq.com/security/readtemplate?t=login_verify_entrances/intro&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc, nil, ua, proxyAddr, proxyUser, proxyPass)
	//获取sessionId
	headers := &map[string]string{
		"Cookie": setCookie,
	}
	sessionId := comm.HttpGetAndSetCookie("https://shminorshort.weixin.qq.com/security/secondauth?t=login_verify_entrances/intro&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&step=41", headers, ua, proxyAddr, proxyUser, proxyPass)
	setCookie += sessionId
	sessionId = strings.Trim(strings.Split(sessionId, "=")[1], ";")
	headers = &map[string]string{
		"Cookie": setCookie,
	}
	comm.HttpGet("https://shminorshort.weixin.qq.com/security/secondauth?t=login_verify_entrances/intro&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&step=42&sessionid="+sessionId, headers, ua, proxyAddr, proxyUser, proxyPass)
	comm.HttpGet("https://shminorshort.weixin.qq.com/security/readtemplate?&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&t=login_verify_entrances/entrances", headers, ua, proxyAddr, proxyUser, proxyPass)
	step3Ret := comm.HttpGet("https://shminorshort.weixin.qq.com/security/secondauth?&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&t=login_verify_entrances/entrances&step=1", headers, ua, proxyAddr, proxyUser, proxyPass)
	var NewSessionIdJson SmsData
	json.Unmarshal([]byte(step3Ret), &NewSessionIdJson)
	setCookie = strings.Replace(setCookie, sessionId, NewSessionIdJson.Sessionid, -1)
	sessionId = NewSessionIdJson.Sessionid
	headers = &map[string]string{
		"Cookie": setCookie,
	}
	comm.HttpGet("https://shminorshort.weixin.qq.com/security/secondauth?&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&t=login_verify_entrances/entrances&step=9&sessionid="+sessionId+"&secondauthtype=11", headers, ua, proxyAddr, proxyUser, proxyPass)
	qrcodeTicketJsonStr := comm.HttpGet("https://shminorshort.weixin.qq.com/security/secondauth?&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&t=login_verify_entrances/entrances&step=21&sessionid="+sessionId, headers, ua, proxyAddr, proxyUser, proxyPass)

	var QrCodeRicketJson SmsData
	json.Unmarshal([]byte(qrcodeTicketJsonStr), &QrCodeRicketJson)

	qrcodeticket := QrCodeRicketJson.Qrcodeticket

	comm.HttpGet("https://shminorshort.weixin.qq.com/security/readtemplate?t=simple_auth/w_qrcode_show&&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&qrcliticket="+qrcodeticket, headers, ua, proxyAddr, proxyUser, proxyPass)
	qrCodeUUIDStr := comm.HttpGet("https://login.weixin.qq.com/jslogin?appid=wx_newdev_verify&t=simple_auth/w_qrcode_show&&ticket="+ticket+"&wechat_real_lang=zh_CN&idc="+idc+"&qrcliticket="+qrcodeticket, headers, ua, proxyAddr, proxyUser, proxyPass)

	QrUUID := GetBetweenStr(qrCodeUUIDStr, "QRLogin.uuid = \"", "\";")

	QrUrl = "https://login.weixin.qq.com/qrcode/" + QrUUID + "?appid=wx_newdev_verify&t=simple_auth/w_qrcode_show&&ticket=" + ticket + "&wechat_real_lang=zh_CN&idc=" + idc + "&qrcliticket=" + qrcodeticket
	checkUrl = "https://login.weixin.qq.com/cgi-bin/mmwebwx-bin/login?uuid=" + QrUUID + "&r=[[[currentMilliseStamp]]]&t=simple_auth/w_qrcode_show&&ticket=" + ticket + "&wechat_real_lang=zh_CN&idc=" + idc + "&qrcliticket=" + qrcodeticket

	return QrUrl, checkUrl

}

func WeChatQrCode1(Data string, ua string, proxy models.ProxyInfo) (QrUrl, checkUrl string) {
	return WeChatQrCode(Data, ua, proxy.ProxyIp, proxy.ProxyUser, proxy.ProxyPassword)

}

func WxSliderOCRRequest(Ticket string) SilderOCR {
	ocrUrl := beego.AppConfig.String("ocrurlhgo") //
	fullUrl := ocrUrl + Ticket

	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return SilderOCR{Flag: -1, Code: -1}
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	req.Header.Set("Proxy-Connection", "keep-alive")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return SilderOCR{Flag: -1, Code: -1}
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return SilderOCR{Flag: -1, Code: -1}
	}

	//fmt.Println("🧾 返回原始内容：", string(body))

	// 默认返回结构
	var result SilderOCR
	_ = json.Unmarshal(body, &result) // 忽略结构不符问题，下面单独解析 message 字段

	// 尝试手动解析 message 字段
	var msgMap map[string]interface{}
	if err := json.Unmarshal(body, &msgMap); err == nil {
		if msg, ok := msgMap["message"].(string); ok {
			result.Message = msg
			if msg == "过滑块成功" {
				result.Success = true
			}
		}
	}

	return result
}

/*
func WxSliderOCRRequest(AID, Ticket string) SilderOCR {
	ocrUrl := beego.AppConfig.String("ocrurl")
	payload := strings.NewReader(`{ "AID": "` + AID + `", "Ticket": "` + Ticket + `"}`)
	client := &http.Client{}
	req, err := http.NewRequest("POST", ocrUrl, payload)
	if err != nil {
		return SilderOCR{
			Flag: -1,
			Code: -1,
		}
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return SilderOCR{
			Flag: -1,
			Code: -1,
		}
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return SilderOCR{
			Flag: -1,
			Code: -1,
		}
	}
	var result SilderOCR
	json.Unmarshal(body, &result)
	return result
}*/

func Getparameter(Url string) map[string]string {
	//查找字符串的位置
	questionIndex := strings.Index(Url, "?")
	//打散成数组
	rs := []rune(Url)
	//用于存储请求的参数字典
	parameterDict := make(map[string]string)
	//参数地址
	parameterStr := ""
	//判断是否存在 ?
	if questionIndex != -1 {
		//判断url的长度
		parameterStr = string(rs[questionIndex+1 : len(Url)])
		//参数数组
		parameterArray := strings.Split(parameterStr, "&")
		//生成参数字典
		for i := 0; i < len(parameterArray); i++ {
			str := parameterArray[i]
			if len(str) > 0 {
				tem := strings.Split(str, "=")
				if len(tem) > 0 && len(tem) == 1 {
					parameterDict[tem[0]] = ""
				} else if len(tem) > 1 {
					parameterDict[tem[0]] = tem[1]
				}
			}
		}
	}

	return parameterDict
}

func GetBetweenStr(str, start, end string) string {
	n := strings.Index(str, start)
	if n == -1 {
		n = 0
	} else {
		n = n + len(start) // 增加了else，不加的会把start带上
	}
	str = string([]byte(str)[n:])
	m := strings.Index(str, end)
	if m == -1 {
		m = len(str)
	}
	str = string([]byte(str)[:m])
	return str
}
