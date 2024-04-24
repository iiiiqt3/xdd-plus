package models

import (
	"crypto/md5"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func LJtoKL(url string) string {
	rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
	rsp.Param("url", url)
	rsp.Param("type", "kl")
	//rsp.Body(fmt.Sprintf(`url=%s&type=hy`, msg))
	data, err := rsp.Response()
	if err != nil {
		return "口令转换失败"
	}
	body, _ := ioutil.ReadAll(data.Body)
	logs.Info(string(body))
	if strings.Contains(string(body), "口令转换失败") {
		return "口令转换失败"
	} else {
		return string(body)
	}
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
	logs.Info(string(body))
	if strings.Contains(string(body), "口令转换失败") {
		return []byte("口令转换失败")
	} else {
		return body
	}
}

func KLtoLJ(kl string) string {
	rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
	rsp.Param("url", kl)
	rsp.Param("type", "hy")
	//rsp.Body(fmt.Sprintf(`url=%s&type=hy`, msg))
	data, err := rsp.Response()

	if err != nil {
		return "口令转换失败"
	}
	body, _ := ioutil.ReadAll(data.Body)
	if strings.Contains(string(body), "口令转换失败") {
		return "口令转换失败"
	} else {
		return string(body)
	}
}

func NolanKlToLj(kl string) string {

	resp, err := http.Post("https://api.nolanstore.top/JComExchange", "application/json", strings.NewReader(fmt.Sprintf("{\n  \"code\": \"%s\"\n}", kl)))
	if err != nil {
		logs.Info("post请求失败 error: %+v", err)

	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Info("读取Body失败 error: %+v", err)

	}
	logs.Info(string(body))
	val, _ := jsonparser.GetString(body, "data", "jumpUrl")
	return val

}

func NolanLJToKL(lj string, title string) string {

	resp, err := http.Post("http://nolan.smxy.xyz/JCommand", "application/json", strings.NewReader(fmt.Sprintf("{\n  \"url\": \"%s\",\n  \"title\": \"%s\",\n  \"img\": \"\"\n}", lj, title)))
	if err != nil {
		logs.Info("post请求失败 error: %+v", err)
		JdCookie{}.Push("口令转换失败，请查看是否存在CF墙")
		return "口令转换失败，请重新获取"
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logs.Info("获取失败")
		}
	}(resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Info("读取Body失败 error: %+v", err)

	}
	logs.Info(string(body))
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
