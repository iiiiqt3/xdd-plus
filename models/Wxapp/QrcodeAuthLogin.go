package Wxapp

import (
	"fmt"
	"net/http"
	"strings"
	"wechatdll/bts"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/Tools"
)

func QrcodeAuthLogin(Data QrcodeAuthLoginParam) models.ResponseResult {
	_, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}
	if strings.Contains(Data.Url, "https://") {
		if strings.Contains(Data.Url, "https://open.weixin.qq.com/connect/confirm?uuid=") {

		} else {
			return models.ResponseResult{
				Code:    -1,
				Success: false,
				Message: "异常:链接格式不正确",
				Data:    nil,
			}
		}
	} else {
		Data.Url = "https://open.weixin.qq.com/connect/confirm?uuid=" + Data.Url
	}

	a8key := Tools.GetA8Key(Tools.GetA8KeyParam{
		Wxid:        Data.Wxid,
		OpCode:      2,
		Scene:       4,
		CodeType:    19,
		CodeVersion: 5,
		ReqUrl:      Data.Url,
	})

	getA8key := bts.GetA8KeyResponse(a8key.Data)

	if getA8key.FullURL == nil {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: "请求失败",
			Data:    nil,
		}
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("POST", *getA8key.FullURL, strings.NewReader("s=1"))
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	req.Header.Set("Origin", "https://open.weixin.qq.com")
	resp, err := client.Do(req)

	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}
	defer resp.Body.Close()

	str := *getA8key.FullURL
	delimiter := "https://open.weixin.qq.com/connect/confirm?"
	rightText := GetRightText(str, delimiter)
	url := "https://open.weixin.qq.com/connect/confirm_reply?" + rightText + "&snsapi_login=on&allow=allow"
	//fmt.Println(url)
	// 判断第一个请求成功后执行第二个请求
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		req, err = http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("创建第二个 GET 请求失败:", err)
			return models.ResponseResult{
				Code:    -9,
				Success: false,
				Message: "第二个请求创建失败",
				Data:    nil,
			}
		}

		// 发送第二个请求
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("发送第二个 GET 请求失败:", err)
			return models.ResponseResult{
				Code:    -9,
				Success: false,
				Message: "第二个请求发送失败",
				Data:    nil,
			}
		}
		defer resp.Body.Close()
	} else {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: "第一个请求未成功，不执行第二个请求",
			Data:    nil,
		}
	}

	return models.ResponseResult{
		Code:    0,
		Success: false,
		Message: "登录成功",
		Data:    nil,
	}
}

// GetRightText 从给定的字符串中获取右边的文本
func GetRightText(s string, delimiter string) string {
	index := strings.LastIndex(s, delimiter)
	if index == -1 {
		return s
	}
	return s[index+len(delimiter):]
}
