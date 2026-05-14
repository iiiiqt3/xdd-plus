package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
	"net/http"
	"strings" 
)


//返利转口令接口
func GetGoodsLink(appid, appkey, unionid, gid string) string {
	requestURL := "http://japi.jingtuitui.com/api/get_goods_link"

	payload := map[string]interface{}{
		"appid":   appid,
		"appkey":  appkey,
		"unionid": unionid,
		"gid":     gid,
		"command": 1,
	}

	payloadBytes, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(requestURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	// 解析返回结果
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return ""
	}
	fmt.Printf("响应数据: %+v", response)
	// "return" 为 "0"获取成功
    if ret, ok := response["return"].(string); !ok || ret != "0" {
        return ""
    }

    if result, ok := response["result"].(map[string]interface{}); ok {
        if data, ok := result["data"].(map[string]interface{}); ok {
            if jShortCommand, ok := data["jShortCommand"].(string); ok {
                return jShortCommand
            }
        }
    }
    return ""
}

func GetUniversal(sender *Sender, gid string) {
	requestURL := "http://japi.jingtuitui.com/api/universal"
	appid := GetEnv("京推推ID") //京推推应用APP ID
	appkey := GetEnv("京推推KEY") //京推推应用APP KEY
	unionid := GetEnv("京东联盟ID") //京东联盟ID
	if appid == "" || appkey == "" || unionid == "" {
		sender.Reply("请先配置转链参数")
		return
	}
	fmt.Println("准备访问jingtuitui接口")

	payload := map[string]interface{}{
		"appid":   appid,
		"appkey":  appkey,
		"unionid": unionid,
		"content": gid,
		"v":       "v3",
	}
	payloadBytes, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 10 * time.Second}
	
	maxRetries := 3  // 重试次数，防止网络问题
	var resp *http.Response
	var err error
	for retry := 0; retry < maxRetries; retry++ {
		resp, err = client.Post(requestURL, "application/json", bytes.NewBuffer(payloadBytes))
		if err == nil {
			break
		}	
		fmt.Printf("请求失败 (尝试 %d/%d): %v", retry+1, maxRetries, err)
		if retry < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		sender.Reply("请求失败，请稍后再试")
		fmt.Printf("请求失败 (已尝试 %d 次): %v\n", maxRetries, err)
		return
	}
	defer resp.Body.Close()
	
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		sender.Reply("转链失败，请稍后再试")
		fmt.Sprintf("解析结果失败: %v", err)
		return
	}

	// "return" 0 返回成功
	if returnCode, ok := response["return"].(float64); ok && returnCode == 0 {
		// 获取返回数据中的 result 和 link_date 部分
		result, ok := response["result"].(map[string]interface{})
		if !ok {
			sender.Reply("返回数据格式不正确")
			return
		}

		// 如果链路信息存在，提取链路和商品信息
		if linkData, ok := result["link_date"].([]interface{}); ok && len(linkData) > 0 {
			// 提取第一个链路信息
			allCommissionPrice, _ := result["all_commission_price"].(float64) // 预估佣金
			mainImg, _ := result["main_img"].(string) // 商品图片
			fl := linkData[0].(map[string]interface{})
			goodsInfo, ok := fl["goods_info"].(map[string]interface{}) // 商品信息集合
			if !ok {
				sender.Reply("商品信息格式不正确")
				return
			}
			priceInfo, ok := goodsInfo["priceInfo"].(map[string]interface{}) // 价格信息
			if !ok {
				sender.Reply("价格信息格式不正确")
				return
			}
			shopInfo, ok := goodsInfo["shopInfo"].(map[string]interface{}) // 店铺信息
			if !ok {
				sender.Reply("店铺信息格式不正确")
				return
			}

			chainLink, _ := fl["chain_link"].(string) // 转链后商品链接
			skuName, _ := goodsInfo["skuName"].(string) // 商品名称
			shopName, _ := shopInfo["shopName"].(string) // 店铺名称
			shopLevel, _ := shopInfo["shopLevel"].(float64) // 店铺等级
			goodCommentsShare, _ := goodsInfo["goodCommentsShare"].(float64) // 商品好评率
			price, _ := priceInfo["price"].(float64) // 商品价格
			lowestPrice, _ := priceInfo["lowestPrice"].(float64) // 促销价
			lowestCouponPrice, _ := priceInfo["lowestCouponPrice"].(float64) // 券后价

			var kl string
			if chainLink != "" {
				kl = GetGoodsLink(appid, appkey, unionid, chainLink)
			}
			msgs := []string{}
			
			if sender.Type == "wx" || sender.Type == "wxg" {
				msgs = append(msgs, fmt.Sprintf("[Gift] 商品名称: %s", skuName))
				msgs = append(msgs, fmt.Sprintf("[Cake] 店铺名称: %s", shopName))
				msgs = append(msgs, fmt.Sprintf("[Sun] 店铺等级: %v", shopLevel))
				msgs = append(msgs, fmt.Sprintf("[ThumbsUp] 商品好评率: %v", goodCommentsShare))
				msgs = append(msgs, fmt.Sprintf("[Packet] 原价: ¥%.2f", toFloat(price)))
				msgs = append(msgs, fmt.Sprintf("[Fireworks] 促销价: ¥%.2f", toFloat(lowestPrice)))
				msgs = append(msgs, fmt.Sprintf("[Party] 券后价: ¥%.2f", toFloat(lowestCouponPrice)))
				if sender.IsAdmin {
					msgs = append(msgs, fmt.Sprintf("[Rich] 预估佣金: ¥%.2f", toFloat(allCommissionPrice)))
				}
				msgs = append(msgs, fmt.Sprintf("[Coffee] 京口令: %s", kl))
				msgs = append(msgs, fmt.Sprintf("[Rose] 领券下单链接: %s", chainLink))
			} else {
				msgs = append(msgs, fmt.Sprintf("🛒 商品名称: %s", skuName))
				msgs = append(msgs, fmt.Sprintf("🏪 店铺名称: %s", shopName))
				msgs = append(msgs, fmt.Sprintf("⭐ 店铺等级: %v", shopLevel))
				msgs = append(msgs, fmt.Sprintf("👍 商品好评率: %v", goodCommentsShare))
				msgs = append(msgs, fmt.Sprintf("💰 原价: ¥%.2f", toFloat(price)))
				msgs = append(msgs, fmt.Sprintf("🔥 促销价: ¥%.2f", toFloat(lowestPrice)))
				msgs = append(msgs, fmt.Sprintf("🎫 券后价: ¥%.2f", toFloat(lowestCouponPrice)))
				if sender.IsAdmin {
					msgs = append(msgs, fmt.Sprintf("💸 预估佣金: ¥%.2f", toFloat(allCommissionPrice)))
				}
				msgs = append(msgs, fmt.Sprintf("🔑 京口令: %s", kl))
				msgs = append(msgs, fmt.Sprintf("🔗 领券下单链接: %s", chainLink))
			}
			msgs = append(msgs, "-------------------------------------------------------------")
			msgs = append(msgs, "      极速跳转：直接复制上面所有信息打开京东APP即可自动跳转商品 ")
			msgs = append(msgs, "-------------------------------------------------------------")
			sender.SendImg2(mainImg)  //发送商品图片
			sender.Reply(strings.Join(msgs, "\n"))  //发送返利信息
	
		} else {
			sender.Reply("返回数据中没有转链信息")
			return
		}
	} else {
		// 如果返回的 "return" 字段不为 0,返利失败
		sender.Reply("此商品暂无价格信息")
		return
	}

	return
}


func toYuan(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val / 100
	case int:
		return float64(val) / 100
	case int64:
		return float64(val) / 100
	default:
		return 0
	}
}

func toFloat(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

