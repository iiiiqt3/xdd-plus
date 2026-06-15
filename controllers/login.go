package controllers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/vweb"

	"github.com/beego/beego/v2/client/httplib"
	qrcode "github.com/skip2/go-qrcode"
)

// LoginController 登录控制器，处理管理员登录、门户登录、CK登录等
type LoginController struct {
	BaseController
}

type StepOne struct {
	SToken string `json:"s_token"`
}

type StepTwo struct {
	Token string `json:"token"`
}

type StepThree struct {
	CheckIP int    `json:"check_ip"`
	Errcode int    `json:"errcode"`
	Message string `json:"message"`
}

type StepThree1 struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

var JdCookieRunners sync.Map
var jdua = models.GetUserAgent

// GetUserInfo 根据pin获取京东用户Cookie信息（VIP功能）
func (c *LoginController) GetUserInfo() {
	if !models.Config.VIP {
		return
	}
	pin := c.GetString("pin")
	cookie, err := models.GetJdCookie(pin)
	ok := models.CookieOK(cookie)
	if err != nil {
		logs.Error(err)
		result := Result{
			Data:    "null",
			Code:    1,
			Message: "查无匹配的pin",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	} else if !ok {
		result := Result{
			Data:    "账号过期",
			Code:    0,
			Message: "账号过期",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	} else {
		result := Result{
			Data:    cookie.Query(),
			Code:    0,
			Message: "查询成功",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}
}

// GetUserPin 根据QQ号获取关联的京东pin列表（VIP功能）
func (c *LoginController) GetUserPin() {
	if !models.Config.VIP {
		return
	}
	qq := c.GetString("QQ")
	if strings.EqualFold(qq, strconv.Itoa(models.Config.QQID)) {
		return
	}
	pins := models.GetPinList(qq)
	if pins == nil {
		result := Result{
			Data:    "null",
			Code:    1,
			Message: "查无匹配的pin",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	} else {
		result := Result{
			Data:    pins,
			Code:    0,
			Message: "查询成功",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}
}

type Cookie struct {
	ck string
}

// GetQrcode1 通过第三方API获取京东登录二维码
func (c *LoginController) GetQrcode1() {
	rsp, err := httplib.Post("https://api.kukuqaq.com/jd/qrcode").Response()
	if err != nil {
		logs.Info(err)
	}
	body, err1 := ioutil.ReadAll(rsp.Body)
	if err1 == nil {
		fmt.Println(string(body))
	}
	s := &models.QQuery{}
	if len(body) > 0 {
		err := json.Unmarshal(body, &s)
		if err != nil {
			return
		}
	}
	//jsonByte, _ := json.Marshal(s)
	//ddd, _ := base64.StdEncoding.DecodeString(s.Data.QqLoginQrcode.Bytes)
	c.Ctx.WriteString(s.Data.QqLoginQrcode.Bytes)
	return
}

// GetQrcode 生成京东登录二维码并返回base64图片数据
func (c *LoginController) GetQrcode() {
	if v := c.GetSession("jd_token"); v != nil {
		token := v.(string)
		if v, ok := JdCookieRunners.Load(token); ok {
			if len(v.([]interface{})) >= 2 {
				var url = `https://plogin.m.jd.com/cgi-bin/m/tmauth?appid=300&client_type=m&token=` + token
				data, _ := qrcode.Encode(url, qrcode.Medium, 256)
				c.Ctx.WriteString(`{"url":"` + url + `","img":"` + base64.StdEncoding.EncodeToString(data) + `"}`)
				return
			}
		}
	}
	var state = time.Now().Unix()
	var url = fmt.Sprintf(`https://plogin.m.jd.com/cgi-bin/mm/new_login_entrance?lang=chs&appid=300&returnurl=https://wq.jd.com/passport/LoginRedirect?state=%d&returnurl=https://home.m.jd.com/myJd/newhome.action?sceneval=2&ufc=&/myJd/home.action&source=wq_passport`,
		state)
	req := httplib.Get(url)
	req.Header("Connection", "Keep-Alive")
	req.Header("Content-Type", "application/x-www-form-urlencoded")
	req.Header("Accept", "application/json, text/plain, */*")
	req.Header("Accept-Language", "zh-cn")
	req.Header("Referer", url)
	req.Header("User-Agent", jdua())
	req.Header("Host", "plogin.m.jd.com")
	rsp, err := req.Response()
	if err != nil {
		c.Ctx.WriteString(err.Error())
		return
	}
	data, err := ioutil.ReadAll(rsp.Body)
	so := StepOne{}
	err = json.Unmarshal(data, &so)
	if err != nil {
		c.Ctx.WriteString(err.Error())
		return
	}
	cookies := strings.Join(rsp.Header.Values("Set-Cookie"), " ")
	var cookie = strings.Join([]string{
		"guid=" + FetchJdCookieValue("guid", cookies),
		"lang=chs",
		"lsid=" + FetchJdCookieValue("lsid", cookies),
		"lstoken=" + FetchJdCookieValue("lstoken", cookies),
	}, ";")
	state = time.Now().Unix()
	req = httplib.Post(
		fmt.Sprintf(`https://plogin.m.jd.com/cgi-bin/m/tmauthreflogurl?s_token=%s&v=%d&remember=true`,
			so.SToken,
			state),
	)
	req.Header("Connection", "Keep-Alive")
	req.Header("Content-Type", "application/x-www-form-urlencoded; Charset=UTF-8")
	req.Header("Accept", "application/json, text/plain, */*")
	req.Header("Cookie", cookie)
	req.Header("Referer", fmt.Sprintf(`https://plogin.m.jd.com/login/login?appid=300&returnurl=https://wqlogin2.jd.com/passport/LoginRedirect?state=%d&returnurl=//home.m.jd.com/myJd/newhome.action?sceneval=2&ufc=&/myJd/home.action&source=wq_passport`,
		state),
	)
	req.Header("User-Agent", jdua())
	req.Header("Host", "plogin.m.jd.com")
	req.Body(fmt.Sprintf(`{
		'lang': 'chs',
		'appid': 300,
		'returnurl': 'https://wqlogin2.jd.com/passport/LoginRedirect?state=%dreturnurl=//home.m.jd.com/myJd/newhome.action?sceneval=2&ufc=&/myJd/home.action&source=wq_passport',
	 }`, state))
	rsp, err = req.Response()
	if err != nil {
		c.Ctx.WriteString(err.Error())
		return
	}
	data, err = ioutil.ReadAll(rsp.Body)
	st := StepTwo{}
	err = json.Unmarshal(data, &st)
	if err != nil {
		c.Ctx.WriteString(err.Error())
		return
	}
	url = `https://plogin.m.jd.com/cgi-bin/m/tmauth?client_type=m&appid=300&token=` + st.Token
	cookies = strings.Join(rsp.Header.Values("Set-Cookie"), " ")
	okl_token := FetchJdCookieValue("okl_token", cookies)
	data, _ = qrcode.Encode(url, qrcode.Medium, 256)
	bot := c.GetString("tp")
	uid := c.GetQueryInt("uid")
	gid := c.GetQueryInt("gid")
	mid := c.GetQueryInt("mid")
	unm := c.GetString("unm")
	JdCookieRunners.Store(st.Token, []interface{}{cookie, okl_token, bot, uid, gid, mid, unm})
	if bot != "" {
		c.Ctx.ResponseWriter.Write(data)
		return
	}
	c.SetSession("jd_token", st.Token)
	c.SetSession("jd_cookie", cookie)
	c.SetSession("jd_okl_token", okl_token)
	c.Ctx.WriteString(`{"url":"` + url + `","img":"` + base64.StdEncoding.EncodeToString(data) + `"}`) //"data:image/png;base64," +
}

func init() {
	go func() {
		for {
			time.Sleep(time.Second)
			JdCookieRunners.Range(func(k, v interface{}) bool {
				jd_token := k.(string)
				vv := v.([]interface{})
				if len(vv) >= 2 {
					cookie := vv[0].(string)
					okl_token := vv[1].(string)
					bot := vv[2].(string)
					uid := vv[3].(int)
					gid := vv[4].(int)
					result, ck := CheckLogin(jd_token, cookie, okl_token)
					switch result {
					case "成功":
						switch bot {
						case "qq", "qqg":
							ck.Update(models.QQ, uid)
							if gid != 0 {
								go models.SendQQGroup(gid, uid, "扫码成功")
							} else {
								go models.SendQQ(uid, "扫码成功")
							}
						case "tg", "tgg":
							ck.Update(models.Telegram, uid)
							if ck.Priority < 0 && models.GetEnv("AutoPriority") == models.True {
								ck.Update(models.Priority, -ck.Priority)
							}
							if gid != 0 {
								go models.SendTggMsg(int(gid), int(uid), "扫码成功", vv[5].(int), vv[6].(string))
							} else {
								go models.SendTgMsg(int(uid), "扫码成功")
							}
						}
					case "授权登录未确认":
					case "":
					default: //失效
						switch bot {
						case "qq", "qqg":
							if gid != 0 {
								go models.SendQQGroup(gid, uid, "扫码失败")
							} else {
								go models.SendQQ(uid, "扫码失败")
							}
						case "tg", "tgg":
							if gid != 0 {
								go models.SendTggMsg(int(gid), int(uid), "扫码失败", vv[5].(int), vv[6].(string))
							} else {
								go models.SendTgMsg(int(uid), "扫码失败")
							}
						}
					}
				}
				return true
			})
		}
	}()
}

// Query 轮询京东扫码登录状态，获取登录结果Cookie
func (c *LoginController) Query() {
	if v := c.GetSession("jd_token"); v == nil {
		c.Ctx.WriteString("重新获取二维码")
		return
	} else {
		token := v.(string)
		if v, ok := JdCookieRunners.Load(token); !ok {
			c.Ctx.WriteString("重新获取二维码")
			return
		} else {
			if len(v.([]interface{})) >= 2 {
				c.Ctx.WriteString("授权登录未确认")
				return
			} else {
				pin := v.([]interface{})[0].(string)
				c.SetSession("pin", pin)
				if note := c.GetString("note"); note != "" {
					if ck, err := models.GetJdCookie(pin); err == nil {
						ck.Update(models.Note, note)
					}
				}
				c.Ctx.WriteString("登录")
				// 	c.Ctx.WriteString("成功")
				return
			}
		}
	}
}

func CheckLogin(token, cookie, okl_token string) (string, *models.JdCookie) {
	state := time.Now().Unix()
	req := httplib.Post(
		fmt.Sprintf(`https://plogin.m.jd.com/cgi-bin/m/tmauthchecktoken?&token=%s&ou_state=0&okl_token=%s`,
			token,
			okl_token,
		),
	)
	req.Header("Referer", fmt.Sprintf(`https://plogin.m.jd.com/login/login?appid=300&returnurl=https://wqlogin2.jd.com/passport/LoginRedirect?state=%d&returnurl=//home.m.jd.com/myJd/newhome.action?sceneval=2&ufc=&/myJd/home.action&source=wq_passport`,
		state),
	)
	req.Header("Cookie", cookie)
	req.Header("Connection", "Keep-Alive")
	req.Header("Content-Type", "application/x-www-form-urlencoded; Charset=UTF-8")
	req.Header("Accept", "application/json, text/plain, */*")
	req.Header("User-Agent", jdua())
	req.Header("Host", "plogin.m.jd.com")

	req.Param("lang", "chs")
	req.Param("appid", "300")
	req.Param("returnurl", fmt.Sprintf("https://wqlogin2.jd.com/passport/LoginRedirect?state=%d&returnurl=//home.m.jd.com/myJd/newhome.action?sceneval=2&ufc=&/myJd/home.action", state))
	req.Param("source", "wq_passport")

	rsp, err := req.Response()
	if err != nil {
		return "", nil //err.Error()
	}
	data, err := ioutil.ReadAll(rsp.Body)
	sth := StepThree{}
	err = json.Unmarshal(data, &sth)
	if err != nil {
		return "", nil //err.Error()
	}
	switch sth.Errcode {
	case 0:
		cookies := strings.Join(rsp.Header.Values("Set-Cookie"), " ")
		pt_key := FetchJdCookieValue("pt_key", cookies)
		pt_pin := FetchJdCookieValue("pt_pin", cookies)
		if pt_pin == "" {
			JdCookieRunners.Delete(token)
			return sth.Message, nil
		}
		ck := models.JdCookie{
			PtKey: pt_key,
			PtPin: pt_pin,
		}
		if _, err := models.GetJdCookie(ck.PtPin); err == nil {
			models.UpdateCookie(&ck)
			msg := fmt.Sprintf("更新账号，%s", ck.PtPin)
			(&models.JdCookie{}).Push(msg)
			logs.Info(msg)
		} else {
			models.NewJdCookie(&ck)
			msg := fmt.Sprintf("添加账号，%s", ck.PtPin)
			(&models.JdCookie{}).Push(msg)
			logs.Info(msg)
		}
		go func() {
			models.Save <- &ck
		}()
		JdCookieRunners.Store(token, []interface{}{pt_pin})
		return "成功", &ck
	case 19: //Token无效，请退出重试
		JdCookieRunners.Delete(token)
		return sth.Message, nil
	case 21: //Token不存在，请退出重试
		JdCookieRunners.Delete(token)
		return sth.Message, nil
	case 176: //授权登录未确认
		return sth.Message, nil
	case 258: //务异常，请稍后重试
		return "", nil
	case 264: //出错了，请退出重试
	default:
		JdCookieRunners.Delete(token)
	}
	return "", nil
}

func FetchJdCookieValue(key string, cookies string) string {
	match := regexp.MustCompile(key + `=([^;]*);{0,1}`).FindStringSubmatch(cookies)
	if len(match) == 2 {
		return match[1]
	} else {
		return ""
	}
}

// ==================== 登录失败限制 ====================

var (
	loginFailMap sync.Map // key: IP, value: *loginFailRecord
)

const maxLoginFails = 5
const loginLockMinutes = 10

type loginFailRecord struct {
	count    int
	lockUntil time.Time
}

func checkLoginLocked(ip string) (locked bool, remainingSec int) {
	v, ok := loginFailMap.Load(ip)
	if !ok {
		return false, 0
	}
	r := v.(*loginFailRecord)
	if r.count >= maxLoginFails {
		if time.Now().Before(r.lockUntil) {
			return true, int(time.Until(r.lockUntil).Seconds())
		}
		// 锁定期过了，重置
		loginFailMap.Delete(ip)
		return false, 0
	}
	return false, 0
}

func recordLoginFail(ip string) int {
	v, _ := loginFailMap.LoadOrStore(ip, &loginFailRecord{})
	r := v.(*loginFailRecord)
	r.count++
	if r.count >= maxLoginFails {
		r.lockUntil = time.Now().Add(loginLockMinutes * time.Minute)
	}
	return r.count
}

func resetLoginFails(ip string) {
	loginFailMap.Delete(ip)
}

// ==================== 登录接口 ====================

// GetLoginStatus 检查当前IP是否被登录锁定（供前端轮询）
func (c *LoginController) GetLoginStatus() {
	ip := c.Ctx.Input.IP()
	locked, remaining := checkLoginLocked(ip)
	c.Data["json"] = map[string]interface{}{
		"locked":    locked,
		"remaining": remaining,
		"fails":     0,
	}
	if locked {
		c.Data["json"].(map[string]interface{})["fails"] = maxLoginFails
	}
	c.ServeJSON()
}

func (c *LoginController) RegisterUser() {
	username := c.GetString("username")
	password := c.GetString("password")
	bindCode := c.GetString("bindCode")

	userNumber, err := models.ConsumeRegisterBindCode(bindCode, username)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}

	account, user, err := models.CreateWebUserAccount(username, password, userNumber)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.SetSession("portal_account_id", account.ID)
	c.SetSession("portal_user_number", user.Number)
	models.UpdateUserActiveAt(user.Number)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"msg":  "注册并绑定成功",
		"data": map[string]interface{}{
			"username":   account.Username,
			"userNumber": user.Number,
		},
	}
	c.ServeJSON()
}

// SendResetPasswordSms 发送密码重置短信验证码
func (c *LoginController) SendResetPasswordSms() {
	// TODO: implement SMS sending logic
}

func (c *LoginController) GetResetPasswordInfo() {
	resetCode := c.GetString("code")
	account, err := models.GetPasswordResetAccountByCode(resetCode)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"msg":  "验证码可用",
		"data": map[string]interface{}{
			"username": account.Username,
		},
	}
	c.ServeJSON()
}

func (c *LoginController) ResetPassword() {
	resetCode := c.GetString("code")
	username := c.GetString("username")
	password := c.GetString("password")

	account, err := models.ConsumePasswordResetCode(resetCode, username)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}

	updated, err := models.ResetWebUserPassword(account.Username, password)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"msg":  "密码重置成功，请使用新密码登录",
		"data": map[string]interface{}{
			"username": updated.Username,
		},
	}
	c.ServeJSON()
}

// PortalLogout 门户用户退出登录，清除Session
func (c *LoginController) PortalLogout() {
	c.DelSession("portal_account_id")
	c.DelSession("portal_user_number")
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "已退出登录"}
	c.ServeJSON()
}

func (c *LoginController) PortalLoginPage() {
	file, err := vweb.WebFs.ReadFile("html/portal_login.html")
	if err != nil {
		c.Ctx.WriteString("portal login page not found")
		return
	}
	c.Ctx.WriteString(string(file))
}

func (c *LoginController) IsAdmin() {
	ip := c.Ctx.Input.IP()
	loginType := c.GetString("type")
	if loginType == "" {
		loginType = "admin"
	}
	if locked, remaining := checkLoginLocked(ip); locked {
		c.Data["json"] = map[string]interface{}{
			"code":      2,
			"msg":       fmt.Sprintf("登录失败次数过多，请%d秒后再试", remaining),
			"remaining": remaining,
		}
		c.ServeJSON()
		return
	}

	account := c.GetString("account")
	pin := c.GetString("pin")
	if pin == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请输入密码"}
		c.ServeJSON()
		return
	}

	if loginType == "user" {
		webAccount, user, err := models.AuthenticateWebUserAccount(account, pin)
		if err == nil {
			resetLoginFails(ip)
			c.SetSession("portal_account_id", webAccount.ID)
			c.SetSession("portal_user_number", user.Number)
			models.UpdateUserActiveAt(user.Number)
			logs.Info("用户[%s]网页登录成功，绑定编号[%d]", webAccount.Username, user.Number)
			c.Ctx.WriteString("登录")
			return
		}
		fails := recordLoginFail(ip)
		c.Data["json"] = map[string]interface{}{
			"code":  1,
			"msg":   fmt.Sprintf("%s（%d/%d）", err.Error(), fails, maxLoginFails),
			"fails": fails,
		}
		c.ServeJSON()
		return
	}

	if models.Config.Account != "" && models.Config.Master != "" && models.Config.Master != "xxxx" {
		if account == models.Config.Account && pin == models.Config.Master {
			resetLoginFails(ip)
			c.SetSession("token", pin)
			logs.Info("管理员[%s]登录成功", account)
			c.Ctx.WriteString("登录")
			return
		}
	}
	value := models.GetCache("AdminToken")
	if value != "" && pin == value {
		resetLoginFails(ip)
		c.SetSession("token", value)
		logs.Info("随机Token登录成功")
		c.Ctx.WriteString("登录")
		return
	}
	fails := recordLoginFail(ip)
	c.Data["json"] = map[string]interface{}{
		"code":  1,
		"msg":   fmt.Sprintf("用户名或密码错误（%d/%d）", fails, maxLoginFails),
		"fails": fails,
	}
	c.ServeJSON()
}

// CkLogin 通过京东Cookie（pt_key+pt_pin）登录，支持新增和更新Cookie
func (c *LoginController) CkLogin() {
	pin := c.GetString("pin")
	key := c.GetString("key")
	qq, _ := c.GetInt("qq")
	bz := c.GetString("bz")
	push := c.GetString("push")

	if key != "" && pin != "" {
		ck := &models.JdCookie{
			PtKey:    key,
			PtPin:    pin,
			QQ:       qq,
			Note:     bz,
			PushPlus: push,
		}
		if key != "" && pin != "" {
			if models.CookieOK(ck) {
				query := ck.Query()
				result := Result{
					Data: query,
					Code: 0,
				}

				if !models.HasPin(pin) {
					models.NewJdCookie(ck)
					result.Message = fmt.Sprintf("添加成功")
					jsons, errs := json.Marshal(result)
					if errs != nil {
						fmt.Println(errs.Error())
					}
					c.Ctx.WriteString(string(jsons))
				} else if !models.HasKey(key) {
					ck, _ := models.GetJdCookie(pin)
					models.UpdateCookie(ck)
					result.Message = fmt.Sprintf("更新成功")
					jsons, errs := json.Marshal(result)
					if errs != nil {
						fmt.Println(errs.Error())
					}
					c.Ctx.WriteString(string(jsons))
				}
				result.Message = "登录成功"
				jsons, errs := json.Marshal(result)
				if errs != nil {
					fmt.Println(errs.Error())
				}
				c.Ctx.WriteString(string(jsons))
			} else {
				result := Result{
					Data:    "null",
					Code:    1,
					Message: "CK过期",
				}
				jsons, errs := json.Marshal(result)
				if errs != nil {
					fmt.Println(errs.Error())
				}
				c.Ctx.WriteString(string(jsons))
			}
		}
	} else {
		result := Result{
			Data:    "null",
			Code:    2,
			Message: "ck格式错误",
		}
		jsons, errs := json.Marshal(result)
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}

}

// SMSLogin 短信验证码登录，通过手机号验证码进行门户登录
func (c *LoginController) SMSLogin() {
	cookie := c.GetString("ck")
	qq := c.GetString("qq")
	token := c.GetString("token")
	logs.Info(cookie)

	if token == models.Config.ApiToken || models.Config.ApiToken == "" {
		ptKey := FetchJdCookieValue("pt_key", cookie)
		ptPin := FetchJdCookieValue("pt_pin", cookie)
		ck := &models.JdCookie{
			PtKey: ptKey,
			PtPin: ptPin,
			QQ:    0,
		}
		if qq != "" {
			ck.QQ, _ = strconv.Atoi(qq)
		}

		if ptKey != "" && ptPin != "" {
			if models.CookieOK(ck) {
				if nck, err := models.GetJdCookie(ck.PtPin); err == nil {
					if qq != "" && len(qq) > 6 {
						atoi, _ := strconv.Atoi(qq)
						ck.Updates(models.JdCookie{
							PtKey:    ptKey,
							PtPin:    ptPin,
							QQ:       atoi,
							Available:   "true",
							IsApp:     "true",
							UpdateAt: time.Now().Local().Format("2006-01-02"),
						})
					} else {
						ck.Updates(models.JdCookie{
							PtKey:    ptKey,
							PtPin:    ptPin,
							Available:   "true",
							IsApp:     "true",
							UpdateAt: time.Now().Local().Format("2006-01-02"),
						})
					}
					msg := fmt.Sprintf("来自APP的更新,账号：%s,QQ: %v", nck.PtPin, qq)
					ck.Push(ck.Query())
					(&models.JdCookie{}).Push(msg)
					go func() {
						models.Save <- &models.JdCookie{}
					}()

				} else {
					models.NewJdCookie(ck)
					if qq != "" {
						msg := fmt.Sprintf("来自APP的添加,账号：%s,QQ: %v", ck.PtPin, qq)
						(&models.JdCookie{}).Push(msg)
					} else {
						msg := fmt.Sprintf("来自APP的添加,账号：%s", ck.PtPin)
						(&models.JdCookie{}).Push(msg)
					}
					ck.Push(ck.Query())
					go func() {
						models.Save <- &models.JdCookie{}
					}()


				}

				result := Result{
					Data:    "null",
					Code:    200,
					Message: "添加成功",
				}
				jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
				if errs != nil {
					fmt.Println(errs.Error())
				}
				c.Ctx.WriteString(string(jsons))

			} else {
				result := Result{
					Data:    "null",
					Code:    300,
					Message: "CK过期",
				}
				jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
				if errs != nil {
					fmt.Println(errs.Error())
				}
				msg := fmt.Sprintf("传入过期CK，请小心攻击，账号：%s", ck.PtPin)
				(&models.JdCookie{}).Push(msg)
				c.Ctx.WriteString(string(jsons))
			}
		} else {
			result := Result{
				Data:    "null",
				Code:    300,
				Message: "CK错误",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			msg := fmt.Sprintf("传入错误CK，请小心攻击，账号：%s", ck.PtPin)
			(&models.JdCookie{}).Push(msg)
			c.Ctx.WriteString(string(jsons))
		}
	} else {
		result := Result{
			Data:    "null",
			Code:    300,
			Message: "Token错误",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		msg := fmt.Sprintf("传入错误Token，请小心攻击")
		(&models.JdCookie{}).Push(msg)
		c.Ctx.WriteString(string(jsons))
	}

}

// BatchSMSLogin 批量提交CK，处理完成后只推送一条汇总消息
func (c *LoginController) BatchSMSLogin() {
	cksStr := c.GetString("cks")
	token := c.GetString("token")

	if token != models.Config.ApiToken && models.Config.ApiToken != "" {
		result := Result{Data: "null", Code: 300, Message: "Token错误"}
		jsons, _ := json.Marshal(result)
		(&models.JdCookie{}).Push("批量推送：传入错误Token，请小心攻击")
		c.Ctx.WriteString(string(jsons))
		return
	}

	var ckList []string
	if err := json.Unmarshal([]byte(cksStr), &ckList); err != nil {
		result := Result{Data: "null", Code: 300, Message: "cks参数格式错误，需要JSON数组"}
		jsons, _ := json.Marshal(result)
		c.Ctx.WriteString(string(jsons))
		return
	}

	total := len(ckList)
	if total == 0 {
		result := Result{Data: "null", Code: 300, Message: "cks为空"}
		jsons, _ := json.Marshal(result)
		c.Ctx.WriteString(string(jsons))
		return
	}

	successCount := 0
	failCount := 0
	var failDetails []string
	var added []string
	var updated []string

	for _, cookie := range ckList {
		ptKey := FetchJdCookieValue("pt_key", cookie)
		ptPin := FetchJdCookieValue("pt_pin", cookie)

		if ptKey == "" || ptPin == "" {
			failCount++
			failDetails = append(failDetails, fmt.Sprintf("无效CK: %s", truncateStr(cookie, 30)))
			continue
		}

		ck := &models.JdCookie{
			PtKey: ptKey,
			PtPin: ptPin,
		}

		if !models.CookieOK(ck) {
			failCount++
			failDetails = append(failDetails, fmt.Sprintf("CK过期: %s", ptPin))
			continue
		}

		if nck, err := models.GetJdCookie(ck.PtPin); err == nil {
			nck.Updates(models.JdCookie{
				PtKey:     ptKey,
				PtPin:     ptPin,
				Available: "true",
				IsApp:     "true",
				UpdateAt:  time.Now().Local().Format("2006-01-02"),
			})
			updated = append(updated, ptPin)
			successCount++
		} else {
			models.NewJdCookie(ck)
			added = append(added, ptPin)
			successCount++
		}
	}

	go func() {
		models.Save <- &models.JdCookie{}
	}()

	summaryMsg := fmt.Sprintf("快递CK批量推送完成，共%d条，成功%d条，失败%d条", total, successCount, failCount)
	if len(added) > 0 {
		summaryMsg += fmt.Sprintf("\n新增%d条", len(added))
	}
	if len(updated) > 0 {
		summaryMsg += fmt.Sprintf("\n更新%d条", len(updated))
	}
	(&models.JdCookie{}).Push(summaryMsg)

	result := Result{
		Code:    200,
		Message: summaryMsg,
		Data: map[string]interface{}{
			"total":   total,
			"success": successCount,
			"failed":  failCount,
			"added":   len(added),
			"updated": len(updated),
			"details": failDetails,
		},
	}
	jsons, _ := json.Marshal(result)
	c.Ctx.WriteString(string(jsons))
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// WskeyLogin 通过wskey方式登录京东，获取Cookie
func (c *LoginController) WskeyLogin() {
	cookie := string(c.Ctx.Input.RequestBody)
	cookie, _ = url.QueryUnescape(cookie)
	logs.Info(cookie)
	Wskey := FetchJdCookieValue("wskey", cookie)
	ptPin := FetchJdCookieValue("pin", cookie)
	ptPin = url.QueryEscape(ptPin)
	ck := &models.JdCookie{
		WsKey: Wskey,
		PtPin: ptPin,
		QQ:    0,
	}
	if Wskey != "" && ptPin != "" {
		ok, s := models.CheckWskeyOK(ck)
		if ok {
			if nck, err := models.GetJdCookie(ck.PtPin); err == nil {
				msg := fmt.Sprintf("Wskey账号更新,账号：%s", nck.PtPin)
				ck.Updates(models.JdCookie{
					WsKey:    Wskey,
					PtPin:    ptPin,
					UpdateAt: time.Now().Local().Format("2006-01-02"),
				})
				(&models.JdCookie{}).Push(msg)
			} else {
				models.NewJdCookie(ck)
				msg := fmt.Sprintf("添加新的Wskey账号,账号：%s", ck.PtPin)
				(&models.JdCookie{}).Push(msg)
			}
			result := Result{
				Data:    "null",
				Code:    200,
				Message: "添加成功",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			c.Ctx.WriteString(string(jsons))
		} else {
			result := Result{
				Data:    "null",
				Code:    0,
				Message: s,
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			c.Ctx.WriteString(string(jsons))
		}
	} else {
		result := Result{
			Data:    "null",
			Code:    0,
			Message: "CK错误",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		c.Ctx.WriteString(string(jsons))
	}
}
