package TenPay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"wechatdll/Cilent/mm"
	"wechatdll/comm"
	"wechatdll/models"
	"wechatdll/models/Wxapp"
)

type GeMaSkdPayQCodeParam struct {
	Name   string
	Money  string
	Remark string
	Wxid   string
}

// LoginResponse 结构体定义，用于存储登录响应中的 code
type LoginResponse struct {
	Code string `json:"code"`
}

type Receipt struct {
	ReceiptID uint64 `json:"receipt_id"`
}

type Response struct {
	Data    interface{} `json:"data"`
	Receipt Receipt     `json:"receipt"`
}

type PostData struct {
	MiniprogramVersion string   `json:"miniprogram_version"`
	Fee                int      `json:"fee"`
	Remark             string   `json:"remark"`
	RemarkPicUrls      string   `json:"remark_pic_urls"`
	OptionList         []string `json:"option_list"`
	ReceiptItemList    []string `json:"receipt_item_list"`
	ShopId             uint64   `json:"shop_id"`
	Sid                string   `json:"sid"`
}

// GmSKDPayQCode 函数实现支付相关的逻辑
func GmSKDPayQCode(Data GeMaSkdPayQCodeParam) models.ResponseResult {
	// 调用 comm.GetLoginata 函数，传入 Wxid 参数，如果有错误则返回错误响应
	_, err := comm.GetLoginata(Data.Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -6,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}

	// 假设 Wxapp.JSLogin 返回的结果可以被 JSON 解析
	resultcode := Wxapp.JSLogin(Wxapp.DefaultParam{
		Wxid:  Data.Wxid,
		Appid: "wx264e9b6d4d484f51",
	})
	datajscode := GetJSJSLoginResponse(resultcode.Data)
	if datajscode.Code == nil {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: "请求失败",
			Data:    nil,
		}
	}

	// 获取登录响应中的 code
	code := *datajscode.Code

	url := "https://payapp.wechatpay.cn/receiptwxmgr/account/list?miniprogram_version=3.15.9&js_code=" + code

	// 发送 GET 请求获取账户列表，并复用 HTTP 客户端
	client := &http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		return models.ResponseResult{
			Code:    -9,
			Success: false,
			Message: fmt.Sprintf("请求错误：%v", err.Error()),
			Data:    nil,
		}
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return models.ResponseResult{
			Code:    -10,
			Success: false,
			Message: fmt.Sprintf("读取响应错误：%v", err.Error()),
			Data:    nil,
		}
	}

	// 解析响应 JSON 数据到一个 map
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return models.ResponseResult{
			Code:    -11,
			Success: false,
			Message: fmt.Sprintf("JSON 解析响应错误：%v", err.Error()),
			Data:    nil,
		}
	}

	// 尝试从结果中获取 sid
	sid, ok := result["sid"].(string)
	if !ok {
		return models.ResponseResult{
			Code:    -12,
			Success: false,
			Message: "未找到 sid",
			Data:    nil,
		}
	}

	// 尝试从结果中获取 account_list
	accountList, ok := result["data"].(map[string]interface{})["account_list"].([]interface{})
	if !ok {
		return models.ResponseResult{
			Code:    -13,
			Success: false,
			Message: "未找到 account_list",
			Data:    nil,
		}
	}

	var foundAccountId string
	found := false
	for _, account := range accountList {
		// 尝试将账户转换为 map
		accountMap, ok := account.(map[string]interface{})
		if !ok {
			continue
		}

		// 尝试从账户中获取 account_name
		accountName, ok := accountMap["account_name"].(string)
		if !ok {
			continue
		}

		// 如果账户名称与传入的参数匹配
		if accountName == Data.Name {
			// 尝试从账户中获取 account_id
			accountId, ok := accountMap["account_id"].(string)
			if !ok {
				continue
			}

			// 使用正则表达式判断 account_id 是否为纯数字
			matched, err := regexp.MatchString("^[0-9]+$", accountId)
			if err != nil {
				continue
			}
			if matched {
				found = true
				foundAccountId = accountId
				break
			}
		}
	}

	if !found {
		return models.ResponseResult{
			Code:    -17,
			Success: false,
			Message: "未找到匹配的账户",
			Data:    nil,
		}
	}

	// 新的请求获取 shop_id
	newUrl := "https://payapp.wechatpay.cn/receiptsjtmgr/account/get?miniprogram_version=3.15.9&account_id=" + foundAccountId + "&account_type=3&sid=" + sid
	resp, err = client.Get(newUrl)
	if err != nil {
		return models.ResponseResult{
			Code:    -18,
			Success: false,
			Message: fmt.Sprintf("新请求错误：%v", err.Error()),
			Data:    nil,
		}
	}
	defer resp.Body.Close()

	// 读取新请求的响应体
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return models.ResponseResult{
			Code:    -19,
			Success: false,
			Message: fmt.Sprintf("读取新响应错误：%v", err.Error()),
			Data:    nil,
		}
	}

	// 解析新响应的 JSON 数据到一个 map
	var newResult map[string]interface{}
	err = json.Unmarshal(body, &newResult)
	if err != nil {
		return models.ResponseResult{
			Code:    -20,
			Success: false,
			Message: fmt.Sprintf("解析新响应错误：%v", err.Error()),
			Data:    nil,
		}
	}
	// 检查是否有特定的错误消息
	if errcode, ok := newResult["msg"].(string); ok && errcode == "只有受邀用户才能使用" {
		return models.ResponseResult{
			Code:    -22,
			Success: false,
			Message: "只有受邀用户才能使用",
			Data:    nil,
		}
	}
	// 尝试从新结果中获取 auth_shop_list
	authShopListInterface, ok := newResult["data"].(map[string]interface{})["auth_shop_list"].([]interface{})
	if !ok {
		return models.ResponseResult{
			Code:    -21,
			Success: false,
			Message: "未找到 auth_shop_list",
			Data:    nil,
		}
	}

	var shopIdStr string
	if len(authShopListInterface) > 0 {
		authShopMap, ok := authShopListInterface[0].(map[string]interface{})
		if ok {
			shopIdInterface, ok := authShopMap["shop_id"]
			if ok {
				shopIdStr = fmt.Sprintf("%v", shopIdInterface)
				// 尝试将 shop_id 转换为纯数字形式
				matched, err := regexp.MatchString("^[0-9]+$", shopIdStr)
				if err != nil || !matched {
					// 如果不是纯数字形式，尝试进行转换
					f, err := strconv.ParseFloat(shopIdStr, 64)
					if err != nil {
						return models.ResponseResult{
							Code:    -23,
							Success: false,
							Message: "无法获取有效的 shop_id",
							Data:    nil,
						}
					}
					shopIdStr = fmt.Sprintf("%d", int(f))
				} else {
					shopIdStr = shopIdStr
				}
			}
		}
	} else {
		return models.ResponseResult{
			Code:    -24,
			Success: false,
			Message: "auth_shop_list 为空",
			Data:    nil,
		}
	}

	// 将 shop_id 转换为 uint64
	shopIdUint64, err := strconv.ParseUint(shopIdStr, 10, 64)
	if err != nil {
		return models.ResponseResult{
			Code:    -24,
			Success: false,
			Message: fmt.Sprintf("无法将 shop_id 转换为 uint64：%v", err.Error()),
			Data:    nil,
		}
	}

	// 将金额字符串转换为浮点数并乘以 100
	fee, err := strconv.ParseFloat(Data.Money, 64)
	if err != nil {
		return models.ResponseResult{
			Code:    -25,
			Success: false,
			Message: fmt.Sprintf("转换金额错误：%v", err.Error()),
			Data:    nil,
		}
	}
	fee *= 100

	postData := PostData{
		MiniprogramVersion: "3.15.9",
		Fee:                int(fee),
		Remark:             Data.Remark,
		RemarkPicUrls:      "",
		OptionList:         []string{},
		ReceiptItemList:    []string{},
		ShopId:             shopIdUint64,
		Sid:                sid,
	}

	postDataJSON, err := json.Marshal(postData)

	if err != nil {
		return models.ResponseResult{
			Code:    -26,
			Success: false,
			Message: fmt.Sprintf("JSON 编码错误：%v", err.Error()),
			Data:    nil,
		}
	}

	// 发送 POST 请求创建收据，并复用 HTTP 客户端
	resp, err = client.Post("https://payapp.wechatpay.cn/receiptsjtmgr/receipt/create?account_type=3&account_id="+foundAccountId+"&sid="+sid, "application/json", bytes.NewBuffer(postDataJSON))
	if err != nil {
		return models.ResponseResult{
			Code:    -26,
			Success: false,
			Message: fmt.Sprintf("POST 请求错误：%v", err.Error()),
			Data:    nil,
		}
	}
	defer resp.Body.Close()

	// 读取 POST 请求的响应体
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return models.ResponseResult{
			Code:    -27,
			Success: false,
			Message: fmt.Sprintf("读取 POST 响应错误：%v", err.Error()),
			Data:    nil,
		}
	}

	// 解析 POST 请求的响应 JSON 数据到一个 map
	var postResult map[string]interface{}
	err = json.Unmarshal(body, &postResult)
	if err != nil {
		return models.ResponseResult{
			Code:    -28,
			Success: false,
			Message: fmt.Sprintf("解析 POST 响应错误：%v", err.Error()),
			Data:    nil,
		}
	}

	// 尝试从 POST 响应结果中获取 receipt 数据
	receiptDataInterface, ok := postResult["data"].(map[string]interface{})["receipt"]
	if !ok {
		return models.ResponseResult{
			Code:    -29,
			Success: false,
			Message: "未找到 receipt 数据",
			Data:    nil,
		}
	}

	// 确保 receiptDataInterface 是一个 map
	receiptData, ok := receiptDataInterface.(map[string]interface{})
	if !ok {
		return models.ResponseResult{
			Code:    -29,
			Success: false,
			Message: "无法将 receipt 数据转换为 map",
			Data:    nil,
		}
	}

	// 尝试从 receipt 数据中获取 receipt_id
	receiptIdInterface, ok := receiptData["receipt_id"]
	if !ok {
		return models.ResponseResult{
			Code:    -30,
			Success: false,
			Message: "未找到有效的 receipt_id",
			Data:    nil,
		}

	}

	// 将 receipt_id 转换为字符串
	var receiptIdStr string
	switch v := receiptIdInterface.(type) {
	case string:
		receiptIdStr = v
	case float64:
		receiptIdStr = fmt.Sprintf("%d", int(v))
	default:
		return models.ResponseResult{
			Code:    -30,
			Success: false,
			Message: "无法将 receipt_id 转换为字符串",
			Data:    nil,
		}
	}

	// 将 receipt_id 转换为纯数字形式
	receiptIdFloat, err := strconv.ParseFloat(receiptIdStr, 64)
	if err != nil {
		return models.ResponseResult{
			Code:    -30,
			Success: false,
			Message: fmt.Sprintf("无法将 receipt_id 转换为数字：%v", err.Error()),
			Data:    nil,
		}
	}
	receiptId := fmt.Sprintf("%d", int(receiptIdFloat))

	// 获取二维码请求
	qrcodeUrl := "https://payapp.wechatpay.cn/receiptsjtmgr/receipt/getwxacode?miniprogram_version=3.15.9&wxacode_path_type=1&receipt_id=" + receiptId + "&account_id=" + foundAccountId + "&account_type=3&sid=" + sid
	resp, err = client.Get(qrcodeUrl)
	if err != nil {
		return models.ResponseResult{
			Code:    -31,
			Success: false,
			Message: fmt.Sprintf("获取二维码请求错误：%v", err.Error()),
			Data:    nil,
		}
	}
	defer resp.Body.Close()

	// 读取二维码响应体
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return models.ResponseResult{
			Code:    -32,
			Success: false,
			Message: fmt.Sprintf("读取二维码响应错误：%v", err.Error()),
			Data:    nil,
		}
	}

	// 解析二维码响应的 JSON 数据到一个 map
	var qrcodeResult map[string]interface{}
	err = json.Unmarshal(body, &qrcodeResult)
	if err != nil {
		return models.ResponseResult{
			Code:    -33,
			Success: false,
			Message: fmt.Sprintf("解析二维码响应错误：%v", err.Error()),
			Data:    nil,
		}
	}

	// 尝试从二维码结果中获取 data 数据
	qrcodeData, ok := qrcodeResult["data"].(map[string]interface{})
	if !ok {
		return models.ResponseResult{
			Code:    -34,
			Success: false,
			Message: "未找到二维码数据",
			Data:    nil,
		}
	}

	// 尝试从 data 数据中获取 qrcode
	qrcode, ok := qrcodeData["qrcode"].(string)
	if !ok {
		return models.ResponseResult{
			Code:    -35,
			Success: false,
			Message: "未找到有效的二维码",
			Data:    nil,
		}

	}

	// 返回成功响应，包含二维码数据
	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    qrcode,
	}
}

func GetJSJSLoginResponse(Data interface{}) mm.JSLoginResponse {
	var Buff mm.JSLoginResponse
	result, err := json.Marshal(&Data)
	if err != nil {
		return mm.JSLoginResponse{}
	}
	_ = json.Unmarshal(result, &Buff)
	return Buff
}
