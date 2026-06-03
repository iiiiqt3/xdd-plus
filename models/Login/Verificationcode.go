package Login

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"wechatdll/comm"
	"wechatdll/models"
)

type ResponseData struct {
	UUID string `json:"uuid"`
}
type PrecheckRequest struct {
	UUID     string `json:"uuid"`
	Precheck struct {
		Ticket string `json:"ticket"`
	} `json:"precheck"`
}
type VerifyMethodRequest struct {
	UUID            string `json:"uuid"`
	GetVerifyMethod struct {
		Ticket string `json:"ticket"`
	} `json:"get_verify_method"`
}
type SubmitPinRequest struct {
	UUID      string `json:"uuid"`
	SubmitPin struct {
		Ticket string `json:"ticket"`
		Pin    string `json:"pin"`
	} `json:"submit_pin"`
}
type VerificationcodeParam struct {
	Uuid   string
	Data62 string
	Code   string
	Ticket string
}

func getEncodedUUID(data string) (string, error) {
	urlStr := "http://47.119.158.126:5200/WXDevGetuuid"
	payload := []byte(fmt.Sprintf(`{"data":"%s"}`, data))
	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}
	//req.Header.Add("Accept", "*/*")
	//req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36 NetType/WIFI MicroMessenger/7.0.20.1781(0x6700143B) WindowsWechat(0x63090a13) XWEB/8555")
	//req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var responseData ResponseData
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", err
	}
	if responseData.UUID != "" {
		encodedUUID := url.QueryEscape(responseData.UUID)
		return encodedUUID, nil
	}
	return "", nil
}
func precheck(uuid, Ticket string) (string, error) {
	url := "https://weixin110.qq.com/security/acct/extdevauthslavecgi?t=extdevsignin%2Fslaveverify&ticket=" + Ticket + "&step=precheck&wechat_real_lang=zh_CN"
	requestData := PrecheckRequest{
		UUID: uuid,
		Precheck: struct {
			Ticket string `json:"ticket"`
		}{
			Ticket: Ticket, //3_2d512e8104f3992c5fa8bfa1047ab3bb ios 18.0.1    pc 3_76cdadf7b44f1e05ad8c140ba26dc044
		},
	}
	fmt.Println("我是Ticket", Ticket)
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", err
	}
	payload := bytes.NewBuffer(jsonData)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36 NetType/WIFI MicroMessenger/7.0.20.1781(0x6700143B) WindowsWechat(0x63090a13) XWEB/85551")
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	ret, ok1 := data["ret"].(float64)

	errMsg, ok2 := data["err_msg"].(string)

	if ok1 && ok2 && ret == 0 && errMsg == "成功。" {
		return res.Header.Get("Set-Cookie"), nil
	} else {
		return "", nil
	}
}

func getVerifyID(uuid, cookie, Ticket string) (string, error) {
	url := "https://weixin110.qq.com/security/acct/extdevauthslavecgi?t=extdevsignin/slaveverify&ticket=" + Ticket + "&step=get_verify_method&wechat_real_lang=zh_CN"
	requestData := VerifyMethodRequest{
		UUID: uuid,
		GetVerifyMethod: struct {
			Ticket string `json:"ticket"`
		}{
			Ticket: Ticket,
		},
	}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", err
	}
	payload := bytes.NewBuffer(jsonData)
	fmt.Println(payload)
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", err
	}
	req.Header.Add("Cookie", cookie)
	//req.Header.Add("Accept", "*/*")
	//req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36 NetType/WIFI MicroMessenger/7.0.20.1781(0x6700143B) WindowsWechat(0x63090a13) XWEB/85551")
	//req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var responseData struct {
		Data struct {
			VerifyID string `json:"verify_id"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", err
	}
	if responseData.Data.VerifyID != "" {
		return responseData.Data.VerifyID, nil

	}

	return "", nil
}

func submitPinAndCheck(secverifyid, uuid, pin, Ticket string) (bool, error) {
	url := "https://weixin110.qq.com/security/acct/commverifypincgi?t=extdevsignin/slaveverify&ticket=" + Ticket + "&step=submit_pin&secverifyid=" + secverifyid + "&wechat_real_lang=zh_CN"
	requestData := SubmitPinRequest{
		UUID: uuid,
		SubmitPin: struct {
			Ticket string `json:"ticket"`
			Pin    string `json:"pin"`
		}{
			Ticket: Ticket,
			Pin:    pin,
		},
	}
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return false, err
	}
	payload := bytes.NewBuffer(jsonData)
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return false, err
	}
	//req.Header.Add("Accept", "*/*")
	//req.Header.Add("Accept-Encoding", "gzip, deflate, br")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36 NetType/WIFI MicroMessenger/7.0.20.1781(0x6700143B) WindowsWechat(0x63090a13) XWEB/8555")
	//req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return false, err
	}
	var responseData struct {
		Ret int `json:"ret"`
	}
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return false, err
	}
	return responseData.Ret == 0, nil
}

func runAllSteps(data, pin, Ticket string) (models.ResponseResult, error) {
	encodedUUID, err := getEncodedUUID(data)
	if err != nil {
		return models.ResponseResult{}, err
	}
	fmt.Println("我是encodedUUID", encodedUUID)
	if encodedUUID == "" {
		return models.ResponseResult{
			Code:    -1,
			Success: false,
			Message: "效验失败",
			Data:    nil,
		}, nil
	}
	setCookie, err := precheck(encodedUUID, Ticket)
	if err != nil {
		return models.ResponseResult{}, err
	}
	fmt.Println("我是setCookie", setCookie)
	if setCookie == "" {
		return models.ResponseResult{
			Code:    -2,
			Success: false,
			Message: "效验失败",
			Data:    nil,
		}, nil
	}
	verifyID, err := getVerifyID(encodedUUID, setCookie, Ticket)
	if err != nil {
		return models.ResponseResult{}, err
	}
	fmt.Println("我是verifyID", verifyID)
	if verifyID == "" {
		return models.ResponseResult{
			Code:    -3,
			Success: false,
			Message: "效验失败",
			Data:    nil,
		}, nil
	}
	success, err := submitPinAndCheck(verifyID, encodedUUID, pin, Ticket)
	if err != nil {
		return models.ResponseResult{}, err
	}
	var message string

	if success {
		message = "验证成功,请调用检测二维码继续登录"
	} else {
		message = "验证失败"
	}

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: message,
		Data:    nil,
	}, nil

}

func Verificationcode2(Data VerificationcodeParam) models.ResponseResult {

	D, err := comm.GetLoginata(Data.Uuid, nil)
	if err != nil || D == nil || D.Uuid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", Data.Uuid)
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

	result, err := runAllSteps(Data.Data62, Data.Code, Data.Ticket)
	if err != nil {
		fmt.Println(err)
	}
	return result
}
