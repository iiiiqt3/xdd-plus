package models

import (
	"crypto/md5"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/buger/jsonparser"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
	"encoding/json"
	"strconv"
	"bytes"
)

func LJtoKL1(url string) string {
	rsp := httplib.Post("https://jd.zack.xin/api/jd/ulink.php")
	rsp.Param("url", url)
	rsp.Param("type", "kl")
	rsp.Param("u", "jApp")
	rsp.Param("model", "json")
	data, err := rsp.Response()
	if err != nil {
		Error("请求失败:", err)
		return "口令转换失败"
	}
	body, _ := ioutil.ReadAll(data.Body)
	Info("响应内容:", string(body))

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		Error("解析 JSON 失败:", err)
		return "解析失败"
	}
	if code, exists := response["code"].(string); exists {
		decodedString := decodeUnicode(code)
		decodedString = strings.Replace(decodedString, "[jApp]【ZACK口令】", "【复制打开JDapp】", -1)
		return decodedString
	} else {
		return "未找到 code 字段"
	}
}

func decodeUnicode(s string) string {
	var result strings.Builder
	for len(s) > 0 {
		if strings.HasPrefix(s, `\u`) && len(s) >= 6 {
			r, err := strconv.ParseInt(s[2:6], 16, 32)
			if err == nil {
				result.WriteRune(rune(r))
				s = s[6:]
				continue
			}
		}
		result.WriteByte(s[0])
		s = s[1:]
	}
	return result.String()
}

func LJtoLJ(url string) []byte {
	rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
	rsp.Param("url", url)
	rsp.Param("type", "su")
	rsp.Param("u", "sq.jd.com")
	rsp.Param("model", "json")
	data, err := rsp.Response()
	if err != nil {
		return []byte("口令转换失败")
	}
	body, _ := ioutil.ReadAll(data.Body)
	Info(string(body))
	if strings.Contains(string(body), "口令转换失败") {
		return []byte("口令转换失败")
	} else {
		return body
	}
}



func ZKLtoLJ(kl string) string {
	rsp := httplib.Post("https://jd.zack.xin/api/jd/ulink.php")
	rsp.Param("url", kl)
	rsp.Param("type", "kl")
	rsp.Param("u", "jApp")
	rsp.Param("model", "json")
	data, err := rsp.Response()
	if err != nil {
		Error("请求失败:", err)
		return "口令转换失败"
	}
	body, _ := ioutil.ReadAll(data.Body)
	Info("响应内容:", string(body))

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		Error("解析 JSON 失败:", err)
		return "解析失败"
	}

	if code, exists := response["code"].(string); exists {
		decodedString := decodeUnicode(code)
		return decodedString
	} else {
		return "未找到 code 字段"
	}
}

func KLtoLJ(kl string) string {
	type Data struct {
		Img      string `json:"img"`
		HeadImg  string `json:"headImg"`
		Title    string `json:"title"`
		UserName string `json:"userName"`
		JumpUrl  string `json:"jumpUrl"`
	}
	type Response struct {
		Code string `json:"code"`
		Data Data   `json:"data"`
	}
	url := "http://m.jing521.cn:8899/JDSign/jCommand?token=123dashi"
	data := map[string]string{
		"code": kl,
	}
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("网络请求错误:", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("读取响应体失败:", err)
			return ""
		}
		var result Response
		if err := json.Unmarshal(body, &result); err != nil {
			fmt.Println("JSON 解码错误:", err)
			fmt.Println("响应内容:", string(body))
			return ""
		}
		if result.Code == "0" {
			return fmt.Sprintf("标题: %s\npin：%s\n链接: %s", result.Data.Title, result.Data.UserName, result.Data.JumpUrl)
		} else {
			return "请求失败，返回错误码: " + result.Code
		}
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println("请求失败，状态码:", resp.StatusCode)
		fmt.Println("响应内容:", string(body))
		return ""
	}
}
func NolanKlToLj(kl string) string {

	resp, err := http.Post("https://api.nolanstore.cc/JComExchange", "application/json", strings.NewReader(fmt.Sprintf("{\n  \"code\": \"%s\"\n}", kl)))
	if err != nil {
		Info("post请求失败 error: %+v", err)

	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Info("读取Body失败 error: %+v", err)

	}
	Info(string(body))
	val, _ := jsonparser.GetString(body, "data", "jumpUrl")
	return val

}

func LJtoKL(klURL string) string {
	type Response struct {
		Code string `json:"code"`
		Data string `json:"data"`
	}

	url := "http://m.jing521.cn:8899/JDSign/jCommand?token=123dashi"
	payload := map[string]string{
		"url": klURL,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("JSON 编码错误:", err)
		return ""
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("网络请求错误:", err)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应体失败:", err)
		return ""
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("JSON 解码错误:", err)
		fmt.Println("响应内容:", string(body))
		return ""
	}

	if result.Code == "0" {
		return result.Data
	} else {
		return fmt.Sprintf("请求失败，返回错误码: %s", result.Code)
	}
}



func NolanLJToKL(lj string, title string) string {

	resp, err := http.Post("http://nolan.smxy.xyz/JCommand", "application/json", strings.NewReader(fmt.Sprintf("{\n  \"url\": \"%s\",\n  \"title\": \"%s\",\n  \"img\": \"\"\n}", lj, title)))
	if err != nil {
		Info("post请求失败 error: %+v", err)
		JdCookie{}.Push("口令转换失败，请查看是否存在CF墙")
		return "口令转换失败，请重新获取"
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			Info("获取失败")
		}
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Info("读取Body失败 error: %+v", err)

	}
	Info(string(body))
	val, _ := jsonparser.GetString(body, "data")
	if val != "" {
		return val
	}
	return "口令转换失败，请重新获取"

}

//随机slice数组
func randShuffle(slice []JdCookie) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

func getMd5String1(str string) string {
	m := md5.New()
	io.WriteString(m, str)
	arr := m.Sum(nil)
	return fmt.Sprintf("%x", arr)
}

func FetchJdCookieValue(key string, cookies string) string {
	match := regexp.MustCompile(key + `=([^;]*);{0,1}`).FindStringSubmatch(cookies)
	if len(match) == 2 {
		return match[1]
	} else {
		return ""
	}
}