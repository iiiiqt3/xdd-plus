package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/core/logs"
	"github.com/cdle/xdd/models"
)

type AdminApiController struct {
	BaseController
}

// NextPrepare beego Prepare 钩子，触发登录验证
func (c *AdminApiController) NextPrepare() {
	c.Logined()
}

// ===================== 活动配置管理 =====================

// GetActivities 获取活动配置列表
func (c *AdminApiController) GetActivities() {
	configs := models.GetActivityConfigsForAdmin()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": configs,
	}
	c.ServeJSON()
}

// SaveActivities 保存活动配置（全量覆盖）
func (c *AdminApiController) SaveActivities() {
	var req struct {
		Activities string `json:"activities"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.Activities == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "内容不能为空"}
		c.ServeJSON()
		return
	}

	// 写入 activities.yaml
	err := ioutil.WriteFile(models.ExecPath+"/conf/activities.yaml", []byte(req.Activities), 0644)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "保存失败: " + err.Error()}
		c.ServeJSON()
		return
	}

	// 触发热加载
	msg := models.ReloadActivities()
	logs.Info("活动配置已更新: %s", msg)

	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// ===================== 青龙容器管理（活动配置关联的） =====================

// GetQingLongConfigs 获取青龙配置列表
func (c *AdminApiController) GetQingLongConfigs() {
	configs := models.GetQingLongConfigsForAdmin()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": configs,
	}
	c.ServeJSON()
}

// ===================== 京东容器管理（config.yaml中的） =====================

// GetJdContainers 获取京东容器列表
func (c *AdminApiController) GetJdContainers() {
	configs := models.GetJdContainersForAdmin()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": configs,
	}
	c.ServeJSON()
}

// GetJdConfig 获取京东相关配置
func (c *AdminApiController) GetJdConfig() {
	cfg := models.GetJdConfigForAdmin()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": cfg,
	}
	c.ServeJSON()
}

// SaveJdConfig 保存京东相关配置
func (c *AdminApiController) SaveJdConfig() {
	var req map[string]interface{}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	msg := models.SaveJdConfigForAdmin(req)
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// GetGameConfig 获取游戏配置
func (c *AdminApiController) GetGameConfig() {
	cfg := models.GetGameConfigForAdmin()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": cfg,
	}
	c.ServeJSON()
}

// SaveGameConfig 保存游戏配置（支持热更新）
func (c *AdminApiController) SaveGameConfig() {
	var req map[string]interface{}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	msg := models.SaveGameConfigForAdmin(req)
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// SendActivityToGroups 推送活动到QQ群/微信群
func (c *AdminApiController) SendActivityToGroups() {
	var req struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		PushToQQ bool   `json:"pushToQQ"`
		PushToWX bool   `json:"pushToWX"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	
	// 如果没有指定推送渠道，默认推送到所有渠道（兼容旧版本）
	if !req.PushToQQ && !req.PushToWX {
		req.PushToQQ = true
		req.PushToWX = true
	}
	
	msg := models.SendActivityToGroupsWithOptions(req.Title, req.Content, req.PushToQQ, req.PushToWX)
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// TestContainer 测试容器连通性
func (c *AdminApiController) TestContainer() {
	var req struct {
		Address string `json:"address"`
		Cid     string `json:"cid"`
		Secret  string `json:"secret"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.Address == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "地址不能为空"}
		c.ServeJSON()
		return
	}

	url := strings.TrimSuffix(req.Address, "/") + "/open/auth/token?client_id=" + req.Cid + "&client_secret=" + req.Secret
	resp, err := http.Get(url)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "连接失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if code, ok := result["code"].(float64); ok && code == 200 {
		c.Data["json"] = map[string]interface{}{"code": 0, "msg": "连接成功"}
	} else {
		msg := "连接失败"
		if m, ok := result["message"].(string); ok {
			msg = m
		}
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": msg}
	}
	c.ServeJSON()
}

// ===================== 环境变量管理（数据库 Env 表） =====================

func (c *AdminApiController) GetNotifications() {
	search := c.GetString("search")
	category := c.GetString("category")
	page := c.GetQueryInt("page")
	limit := c.GetQueryInt("limit")
	list, total := models.GetAdminNotifications(search, category, page, limit)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  list,
			"total": total,
		},
	}
	c.ServeJSON()
}

func (c *AdminApiController) SendNotification() {
	var req struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		Category    string `json:"category"`
		DisplayType string `json:"displayType"`
		IsTop       bool   `json:"isTop"`
		PushToQQ    bool   `json:"pushToQQ"`
		PushToWX    bool   `json:"pushToWX"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "主题不能为空"}
		c.ServeJSON()
		return
	}
	if req.Content == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "详细内容不能为空"}
		c.ServeJSON()
		return
	}
	title, content, category, displayType, isTop := req.Title, req.Content, req.Category, req.DisplayType, req.IsTop
	go func() {
		if _, err := models.CreateAdminWebNotification(title, content, category, displayType, isTop); err != nil {
			logs.Error("后台发送管理员通知失败: %v", err)
		}
		if req.PushToQQ || req.PushToWX {
			pushResult := models.SendActivityToGroupsWithOptions(title, content, req.PushToQQ, req.PushToWX)
			logs.Info("管理员通知群推送结果: %s", pushResult)
		}
	}()
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "发送请求已提交，通知正在后台写入"}
	c.ServeJSON()
}

func (c *AdminApiController) DeleteNotification() {
	idStr := c.Ctx.Input.Param(":id")
	id, _ := strconv.Atoi(idStr)
	if id <= 0 {
		id = c.GetQueryInt("id")
	}
	if err := models.DeleteAdminNotification(id); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

func (c *AdminApiController) BatchDeleteNotifications() {
	var req struct {
		IDs []int `json:"ids"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if err := models.DeleteAdminNotifications(req.IDs); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "批量删除成功"}
	c.ServeJSON()
}

func (c *AdminApiController) UpdateNotification() {
	var req struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Content     string `json:"content"`
		Category    string `json:"category"`
		DisplayType string `json:"displayType"`
		IsTop       bool   `json:"isTop"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.ID <= 0 {
		idStr := c.Ctx.Input.Param(":id")
		req.ID, _ = strconv.Atoi(idStr)
	}
	if err := models.UpdateAdminNotification(req.ID, req.Title, req.Content, req.Category, req.DisplayType, req.IsTop); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "修改成功"}
	c.ServeJSON()
}

func (c *AdminApiController) GetFeedbacks() {
	search := c.GetString("search")
	page := c.GetQueryInt("page")
	limit := c.GetQueryInt("limit")
	sortField := c.GetString("sortField")
	sortOrder := c.GetString("sortOrder")
	list, total := models.GetAdminAppFeedbacks(search, page, limit, sortField, sortOrder)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  list,
			"total": total,
		},
	}
	c.ServeJSON()
}

func (c *AdminApiController) GetFeedbackDetail() {
	idStr := c.Ctx.Input.Param(":id")
	id, _ := strconv.Atoi(idStr)
	if id <= 0 {
		id = c.GetQueryInt("id")
	}
	item, err := models.GetAdminAppFeedbackDetail(id)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": item}
	c.ServeJSON()
}

func (c *AdminApiController) UpdateFeedbackStatus() {
	var req struct {
		ID            int    `json:"id"`
		Status        string `json:"status"`
		Reply         string `json:"reply"`
		RewardCoin    int    `json:"rewardCoin"`
		Handler       string `json:"handler"`
		IsFirstReply  bool   `json:"isFirstReply"`
		NotifyWebApp  bool   `json:"notifyWebApp"`
		NotifyBot     bool   `json:"notifyBot"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.ID <= 0 {
		idStr := c.Ctx.Input.Param(":id")
		req.ID, _ = strconv.Atoi(idStr)
	}
	if req.ID <= 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "反馈ID不能为空"}
		c.ServeJSON()
		return
	}
	if req.RewardCoin < 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "奖励积分不能小于0"}
		c.ServeJSON()
		return
	}
	if req.Reply != "" || req.RewardCoin > 0 || req.Status == "processed" || req.Status == "done" {
		if err := models.ProcessAppFeedback(req.ID, req.Status, req.Reply, req.RewardCoin, req.Handler, req.IsFirstReply, req.NotifyWebApp, req.NotifyBot); err != nil {
			c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
			c.ServeJSON()
			return
		}
		c.Data["json"] = map[string]interface{}{"code": 0, "msg": "反馈已处理"}
		c.ServeJSON()
		return
	}
	if err := models.UpdateAppFeedbackStatus(req.ID, req.Status); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "状态已更新"}
	c.ServeJSON()
}

func (c *AdminApiController) BatchDeleteFeedbacks() {
	var req struct {
		IDs []int `json:"ids"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if err := models.BatchDeleteAppFeedbacks(req.IDs); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "批量删除成功"}
	c.ServeJSON()
}

func (c *AdminApiController) BatchReplyFeedbacks() {
	var req struct {
		IDs     []int  `json:"ids"`
		Reply   string `json:"reply"`
		Handler string `json:"handler"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	count, err := models.BatchReplyAppFeedbacks(req.IDs, req.Reply, req.Handler)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": fmt.Sprintf("成功回复 %d 条反馈", count)}
	c.ServeJSON()
}

func (c *AdminApiController) BatchRewardFeedbacks() {
	var req struct {
		IDs        []int `json:"ids"`
		RewardCoin int   `json:"rewardCoin"`
		Handler    string `json:"handler"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	count, err := models.BatchRewardAppFeedbacks(req.IDs, req.RewardCoin, req.Handler)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": fmt.Sprintf("成功奖励 %d 条反馈", count)}
	c.ServeJSON()
}

func (c *AdminApiController) GetNotificationCategories() {
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": models.AllNotifyCategories,
	}
	c.ServeJSON()
}

// GetEnvVars 获取环境变量列表（支持搜索和分页）
func (c *AdminApiController) GetEnvVars() {
	search := c.GetString("search")
	page := c.GetQueryInt("page")
	limit := c.GetQueryInt("limit")
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}

	envs, total := models.GetEnvVarsAdmin(search, page, limit)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  envs,
			"total": total,
		},
	}
	c.ServeJSON()
}

// CreateEnvVar 创建环境变量
func (c *AdminApiController) CreateEnvVar() {
	var env models.Env
	json.Unmarshal(c.Ctx.Input.RequestBody, &env)
	if env.Name == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "名称不能为空"}
		c.ServeJSON()
		return
	}
	err := models.CreateEnvVar(&env)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "创建失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "创建成功"}
	c.ServeJSON()
}

// UpdateEnvVar 更新环境变量
func (c *AdminApiController) UpdateEnvVar() {
	var env models.Env
	json.Unmarshal(c.Ctx.Input.RequestBody, &env)
	if env.ID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "ID不能为空"}
		c.ServeJSON()
		return
	}
	err := models.UpdateEnvVar(&env)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "更新失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "更新成功"}
	c.ServeJSON()
}

// DeleteEnvVar 删除环境变量
func (c *AdminApiController) DeleteEnvVar() {
	idStr := c.Ctx.Input.Param(":id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "无效的ID"}
		c.ServeJSON()
		return
	}
	err = models.DeleteEnvVar(id)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "删除失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// ===================== 用户管理 =====================

// GetUsers 获取用户列表（支持搜索和分页）
func (c *AdminApiController) GetUsers() {
	search := c.GetString("search")
	page := c.GetQueryInt("page")
	limit := c.GetQueryInt("limit")
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}

	users, total := models.GetUsersAdmin(search, page, limit)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  users,
			"total": total,
		},
	}
	c.ServeJSON()
}

// UpdateUserCoin 修改用户积分
func (c *AdminApiController) UpdateUserCoin() {
	var req struct {
		Number int `json:"number"`
		Coin   int `json:"coin"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.Number == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "用户ID不能为空"}
		c.ServeJSON()
		return
	}

	err := models.SetUserCoin(req.Number, req.Coin)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "修改失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	models.RecordCoinLog(req.Number, req.Coin, "管理员操作", fmt.Sprintf("后台设置积分为%d", req.Coin))
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "修改成功"}
	c.ServeJSON()
}

// ===================== 积分变动 =====================

// GetCoinLogs 查询用户积分变动记录（支持时间筛选和分页）
func (c *AdminApiController) GetCoinLogs() {
	userNumber := c.GetQueryInt("number")
	days := c.GetQueryInt("days")
	page := c.GetQueryInt("page")
	limit := c.GetQueryInt("limit")
	if userNumber == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请输入用户编号"}
		c.ServeJSON()
		return
	}
	if days == 0 {
		days = 3
	}
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}
	logs, total := models.GetCoinLogsFiltered(userNumber, days, page, limit)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  logs,
			"total": total,
		},
	}
	c.ServeJSON()
}

// ===================== 系统配置 =====================

// GetSystemConfig 获取系统配置
func (c *AdminApiController) GetSystemConfig() {
	cfg := models.GetSystemConfigForAdmin()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": cfg,
	}
	c.ServeJSON()
}

// SaveSystemConfig 保存系统配置
func (c *AdminApiController) SaveSystemConfig() {
	var req map[string]interface{}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	msg := models.SaveSystemConfigForAdmin(req)
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// GetImageToken 自动获取图床Token（调用image.go的GetImageToken）
func (c *AdminApiController) GetImageToken() {
	models.GetImageToken()
	cfg := models.ListConfig()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"msg":  "获取成功",
		"data": cfg.ImageToken,
	}
	c.ServeJSON()
}

// ===================== 京东CK管理 =====================

// GetJdCookies 获取京东CK列表（已有API，包装一下分页）
func (c *AdminApiController) GetJdCookies() {
	var page = c.GetQueryInt("page")
	var limit = c.GetQueryInt("limit")
	search := c.GetString("search")

	cks, total := models.GetJdCookiesAdmin(search, page, limit)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  cks,
			"total": total,
		},
	}
	c.ServeJSON()
}

// DeleteJdCookie 删除京东CK
func (c *AdminApiController) DeleteJdCookie() {
	var req struct {
		ID int `json:"id"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.ID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "无效的ID"}
		c.ServeJSON()
		return
	}
	err := models.DeleteJdCookieById(req.ID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "删除失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// UpdateJdCookie 更新京东CK
func (c *AdminApiController) UpdateJdCookie() {
	var req struct {
		ID        int    `json:"id"`
		PtKey     string `json:"ptKey"`
		Priority  int    `json:"priority"`
		Available string `json:"available"`
		Note      string `json:"note"`
		Wxid      string `json:"wxid"`
		WxPid     string `json:"wxPid"`
		QQ        string `json:"qq"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.ID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "无效的ID"}
		c.ServeJSON()
		return
	}
	err := models.UpdateJdCookieById(req.ID, req.PtKey, req.Priority, req.Available, req.Note, req.Wxid, req.WxPid, req.QQ)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "更新失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "更新成功"}
	c.ServeJSON()
}

// ===================== 通用青龙环境变量管理 =====================

// GetQLEnvs 获取青龙面板环境变量
func (c *AdminApiController) GetQLEnvs() {
	configName := c.GetString("config")
	searchValue := c.GetString("search")

	envs, err := models.GetQLEnvsAdmin(configName, searchValue)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": envs,
	}
	c.ServeJSON()
}

// DeleteQLEnv 删除青龙环境变量
func (c *AdminApiController) DeleteQLEnv() {
	var req struct {
		ConfigName string `json:"configName"`
		EnvID      int    `json:"envId"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	err := models.DeleteQLEnvAdmin(req.ConfigName, req.EnvID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// EnableQLEnv 启用青龙环境变量
func (c *AdminApiController) EnableQLEnv() {
	var req struct {
		ConfigName string `json:"configName"`
		EnvID      int    `json:"envId"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	err := models.EnableQLEnvAdmin(req.ConfigName, req.EnvID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "操作成功"}
	c.ServeJSON()
}

// DisableQLEnv 禁用青龙环境变量
func (c *AdminApiController) DisableQLEnv() {
	var req struct {
		ConfigName string `json:"configName"`
		EnvID      int    `json:"envId"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	err := models.DisableQLEnvAdmin(req.ConfigName, req.EnvID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "操作成功"}
	c.ServeJSON()
}

// UpdateQLEnv 更新青龙环境变量
func (c *AdminApiController) UpdateQLEnv() {
	var req struct {
		ConfigName string `json:"configName"`
		EnvID      int    `json:"envId"`
		Name       string `json:"name"`
		Value      string `json:"value"`
		Remarks    string `json:"remarks"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.EnvID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "无效的ID"}
		c.ServeJSON()
		return
	}

	err := models.UpdateQLEnvAdmin(req.ConfigName, req.EnvID, req.Name, req.Value, req.Remarks)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "更新成功"}
	c.ServeJSON()
}

// DeleteUser 删除用户
func (c *AdminApiController) DeleteUser() {
	var req struct {
		ID int `json:"id"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.ID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "无效的ID"}
		c.ServeJSON()
		return
	}
	err := models.DeleteUserAdmin(req.ID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "删除失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// TestQLConnection 测试青龙容器连通性
func (c *AdminApiController) TestQLConnection() {
	var req struct {
		Host         string `json:"host"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.Host == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "地址不能为空"}
		c.ServeJSON()
		return
	}

	ok, msg := models.TestQingLongConnection(req.Host, req.ClientID, req.ClientSecret)
	if ok {
		c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	} else {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": msg}
	}
	c.ServeJSON()
}

func (c *AdminApiController) GetActivityAuthAccounts() {
	activityID := c.GetString("activityId")
	items, err := models.GetActivityAuthAccounts(activityID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": items}
	c.ServeJSON()
}

func (c *AdminApiController) DeleteActivityAuthAccount() {
	var req struct {
		ActivityID        string         `json:"activityId"`
		EnvID             int            `json:"envId"`
		EnvIDs            []int          `json:"envIds"`
		Reason            string         `json:"reason"`
		Channels          []string       `json:"channels"`
		ManualRefundCoins map[int]int    `json:"manualRefundCoins"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.ActivityID == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "活动ID不能为空"}
		c.ServeJSON()
		return
	}
	envIDs := req.EnvIDs
	if len(envIDs) == 0 && req.EnvID > 0 {
		envIDs = []int{req.EnvID}
	}
	if len(envIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "账号ID无效"}
		c.ServeJSON()
		return
	}
	channels := models.NormalizeNotifyChannels(req.Channels)
	deletedCount, refundCoin, err := models.DeleteActivityAuthAccounts(req.ActivityID, envIDs, req.Reason, channels, req.ManualRefundCoins)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	msg := fmt.Sprintf("已删除 %d 个账号并退还 %d 积分", deletedCount, refundCoin)
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg, "data": map[string]interface{}{"deletedCount": deletedCount, "refundCoin": refundCoin}}
	c.ServeJSON()
}

// GetCronTasks 获取青龙定时任务列表
func (c *AdminApiController) GetCronTasks() {
	configName := c.GetString("config")
	searchValue := c.GetString("search")

	tasks, err := models.GetCronTasksAdmin(configName, searchValue)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": tasks,
	}
	c.ServeJSON()
}

// GetCronTaskLogs 获取青龙任务日志列表
func (c *AdminApiController) GetCronTaskLogs() {
	configName := c.GetString("config")
	taskID, _ := strconv.Atoi(c.GetString("taskId"))
	page := c.GetQueryInt("page")
	limit := c.GetQueryInt("limit")
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}

	logs, total, err := models.GetCronTaskLogsAdmin(configName, taskID, page, limit)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  logs,
			"total": total,
		},
	}
	c.ServeJSON()
}

// GetCronTaskLogContent 获取任务日志内容
func (c *AdminApiController) GetCronTaskLogContent() {
	configName := c.GetString("config")
	taskID, _ := strconv.Atoi(c.GetString("taskId"))
	logFile := c.GetString("logFile") // 可选：指定历史日志文件名

	var content string
	var err error
	if logFile != "" {
		content, err = models.GetCronTaskLogContentAdmin(configName, taskID, logFile)
	} else {
		content, err = models.GetCronTaskLogContentAdmin(configName, taskID)
	}
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": content,
	}
	c.ServeJSON()
}

// GetCronTaskScript 获取任务脚本内容
func (c *AdminApiController) GetCronTaskScript() {
	configName := c.GetString("config")
	taskID, _ := strconv.Atoi(c.GetString("taskId"))

	command, err := models.GetCronTaskScriptAdmin(configName, taskID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": command,
	}
	c.ServeJSON()
}

// UpdateCronTaskScript 更新任务脚本
func (c *AdminApiController) UpdateCronTaskScript() {
	var req struct {
		ConfigName string `json:"configName"`
		TaskID     int    `json:"taskId"`
		Command    string `json:"command"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.TaskID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "任务ID不能为空"}
		c.ServeJSON()
		return
	}
	if req.Command == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "脚本内容不能为空"}
		c.ServeJSON()
		return
	}

	err := models.UpdateCronTaskScriptAdmin(req.ConfigName, req.TaskID, req.Command)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "脚本更新成功"}
	c.ServeJSON()
}

// GetCronTaskScriptFile 获取任务脚本文件内容（读取实际脚本文件而非 command 字段）
func (c *AdminApiController) GetCronTaskScriptFile() {
	configName := c.GetString("config")
	taskID, _ := strconv.Atoi(c.GetString("taskId"))

	content, command, err := models.GetCronTaskScriptFileAdmin(configName, taskID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"content": content,
			"command": command,
		},
	}
	c.ServeJSON()
}

// UpdateCronTaskScriptFile 更新任务脚本文件内容
func (c *AdminApiController) UpdateCronTaskScriptFile() {
	var req struct {
		ConfigName string `json:"configName"`
		TaskID     int    `json:"taskId"`
		Content    string `json:"content"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.TaskID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "任务ID不能为空"}
		c.ServeJSON()
		return
	}

	err := models.UpdateCronTaskScriptFileAdmin(req.ConfigName, req.TaskID, req.Content)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "脚本文件更新成功"}
	c.ServeJSON()
}

// UpdateCronTaskSchedule 更新任务定时规则
func (c *AdminApiController) UpdateCronTaskSchedule() {
	var req struct {
		ConfigName string `json:"configName"`
		TaskID     int    `json:"taskId"`
		Cron       string `json:"cron"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.TaskID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "任务ID不能为空"}
		c.ServeJSON()
		return
	}
	if req.Cron == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "定时规则不能为空"}
		c.ServeJSON()
		return
	}

	err := models.UpdateCronTaskScheduleAdmin(req.ConfigName, req.TaskID, req.Cron)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "定时规则更新成功"}
	c.ServeJSON()
}

// EnableCronTask 启用任务
func (c *AdminApiController) EnableCronTask() {
	var req struct {
		ConfigName string `json:"configName"`
		TaskIDs    []int  `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.EnableCronTaskAdmin(req.ConfigName, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "启用成功"}
	c.ServeJSON()
}

// DisableCronTask 禁用任务
func (c *AdminApiController) DisableCronTask() {
	var req struct {
		ConfigName string `json:"configName"`
		TaskIDs    []int  `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.DisableCronTaskAdmin(req.ConfigName, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "禁用成功"}
	c.ServeJSON()
}

// DeleteCronTask 删除任务
func (c *AdminApiController) DeleteCronTask() {
	var req struct {
		ConfigName string `json:"configName"`
		TaskIDs    []int  `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.DeleteCronTaskAdmin(req.ConfigName, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// RunCronTask 手动运行任务
func (c *AdminApiController) RunCronTask() {
	var req struct {
		ConfigName string `json:"configName"`
		TaskIDs    []int  `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.RunCronTaskAdmin(req.ConfigName, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "任务已触发运行"}
	c.ServeJSON()
}

// StopCronTask 停止任务
func (c *AdminApiController) StopCronTask() {
	var req struct {
		ConfigName string `json:"configName"`
		TaskIDs    []int  `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.StopCronTaskAdmin(req.ConfigName, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "任务已停止"}
	c.ServeJSON()
}

// GetJdContainerNames 获取京东容器名称列表（用于前端容器选择下拉）
func (c *AdminApiController) GetJdContainerNames() {
	containers := models.GetJdContainerNamesForAdmin()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": containers,
	}
	c.ServeJSON()
}

// ===================== 京东容器 Cron 任务管理 =====================

// GetJdCronTasks 获取京东容器定时任务列表
func (c *AdminApiController) GetJdCronTasks() {
	containerIdx, _ := strconv.Atoi(c.GetString("containerIdx"))
	searchValue := c.GetString("search")

	tasks, err := models.GetJdCronTasksAdmin(containerIdx, searchValue)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": tasks,
	}
	c.ServeJSON()
}

// GetJdCronTaskLogs 获取京东容器任务日志列表
func (c *AdminApiController) GetJdCronTaskLogs() {
	containerIdx, _ := strconv.Atoi(c.GetString("containerIdx"))
	taskID, _ := strconv.Atoi(c.GetString("taskId"))
	page := c.GetQueryInt("page")
	limit := c.GetQueryInt("limit")
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}

	logs, total, err := models.GetJdCronTaskLogsAdmin(containerIdx, taskID, page, limit)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":  logs,
			"total": total,
		},
	}
	c.ServeJSON()
}

// GetJdCronTaskLogContent 获取京东容器任务日志内容
func (c *AdminApiController) GetJdCronTaskLogContent() {
	containerIdx, _ := strconv.Atoi(c.GetString("containerIdx"))
	taskID, _ := strconv.Atoi(c.GetString("taskId"))
	logFile := c.GetString("logFile") // 可选：指定历史日志文件名

	var content string
	var err error
	if logFile != "" {
		content, err = models.GetJdCronTaskLogContentAdmin(containerIdx, taskID, logFile)
	} else {
		content, err = models.GetJdCronTaskLogContentAdmin(containerIdx, taskID)
	}
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": content,
	}
	c.ServeJSON()
}

// GetJdCronTaskScript 获取京东容器任务脚本
func (c *AdminApiController) GetJdCronTaskScript() {
	containerIdx, _ := strconv.Atoi(c.GetString("containerIdx"))
	taskID, _ := strconv.Atoi(c.GetString("taskId"))

	command, err := models.GetJdCronTaskScriptAdmin(containerIdx, taskID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": command,
	}
	c.ServeJSON()
}

// UpdateJdCronTaskScript 更新京东容器任务脚本
func (c *AdminApiController) UpdateJdCronTaskScript() {
	var req struct {
		ContainerIdx int    `json:"containerIdx"`
		TaskID       int    `json:"taskId"`
		Command      string `json:"command"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.TaskID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "任务ID不能为空"}
		c.ServeJSON()
		return
	}
	if req.Command == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "脚本内容不能为空"}
		c.ServeJSON()
		return
	}

	err := models.UpdateJdCronTaskScriptAdmin(req.ContainerIdx, req.TaskID, req.Command)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "脚本更新成功"}
	c.ServeJSON()
}

// GetJdCronTaskScriptFile 获取京东容器任务脚本文件内容
func (c *AdminApiController) GetJdCronTaskScriptFile() {
	containerIdx, _ := strconv.Atoi(c.GetString("containerIdx"))
	taskID, _ := strconv.Atoi(c.GetString("taskId"))

	content, command, err := models.GetJdCronTaskScriptFileAdmin(containerIdx, taskID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"content": content,
			"command": command,
		},
	}
	c.ServeJSON()
}

// UpdateJdCronTaskScriptFile 更新京东容器任务脚本文件内容
func (c *AdminApiController) UpdateJdCronTaskScriptFile() {
	var req struct {
		ContainerIdx int    `json:"containerIdx"`
		TaskID       int    `json:"taskId"`
		Content      string `json:"content"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.TaskID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "任务ID不能为空"}
		c.ServeJSON()
		return
	}

	err := models.UpdateJdCronTaskScriptFileAdmin(req.ContainerIdx, req.TaskID, req.Content)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "脚本文件更新成功"}
	c.ServeJSON()
}

// UpdateJdCronTaskSchedule 更新京东容器任务定时规则
func (c *AdminApiController) UpdateJdCronTaskSchedule() {
	var req struct {
		ContainerIdx int    `json:"containerIdx"`
		TaskID       int    `json:"taskId"`
		Cron         string `json:"cron"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.TaskID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "任务ID不能为空"}
		c.ServeJSON()
		return
	}
	if req.Cron == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "定时规则不能为空"}
		c.ServeJSON()
		return
	}

	err := models.UpdateJdCronTaskScheduleAdmin(req.ContainerIdx, req.TaskID, req.Cron)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "定时规则更新成功"}
	c.ServeJSON()
}

// EnableJdCronTask 启用京东容器任务
func (c *AdminApiController) EnableJdCronTask() {
	var req struct {
		ContainerIdx int   `json:"containerIdx"`
		TaskIDs      []int `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.EnableJdCronTaskAdmin(req.ContainerIdx, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "启用成功"}
	c.ServeJSON()
}

// DisableJdCronTask 禁用京东容器任务
func (c *AdminApiController) DisableJdCronTask() {
	var req struct {
		ContainerIdx int   `json:"containerIdx"`
		TaskIDs      []int `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.DisableJdCronTaskAdmin(req.ContainerIdx, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "禁用成功"}
	c.ServeJSON()
}

// DeleteJdCronTask 删除京东容器任务
func (c *AdminApiController) DeleteJdCronTask() {
	var req struct {
		ContainerIdx int   `json:"containerIdx"`
		TaskIDs      []int `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.DeleteJdCronTaskAdmin(req.ContainerIdx, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// RunJdCronTask 手动运行京东容器任务
func (c *AdminApiController) RunJdCronTask() {
	var req struct {
		ContainerIdx int   `json:"containerIdx"`
		TaskIDs      []int `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.RunJdCronTaskAdmin(req.ContainerIdx, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "任务已触发运行"}
	c.ServeJSON()
}

// StopJdCronTask 停止京东容器任务
func (c *AdminApiController) StopJdCronTask() {
	var req struct {
		ContainerIdx int   `json:"containerIdx"`
		TaskIDs      []int `json:"taskIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if len(req.TaskIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择任务"}
		c.ServeJSON()
		return
	}

	err := models.StopJdCronTaskAdmin(req.ContainerIdx, req.TaskIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "任务已停止"}
	c.ServeJSON()
}

// ===================== 京东容器 CRUD =====================

// AddJdContainer 添加京东容器
func (c *AdminApiController) AddJdContainer() {
	var req struct {
		Address  string `json:"address"`
		Cid      string `json:"cid"`
		Secret   string `json:"secret"`
		Weight   int    `json:"weight"`
		Limit    int    `json:"limit"`
		Mode     string `json:"mode"`
		Resident string `json:"resident"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.Address == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "地址不能为空"}
		c.ServeJSON()
		return
	}
	if req.Mode == "" {
		req.Mode = "balance"
	}
	if req.Weight == 0 {
		req.Weight = 100
	}
	if req.Limit == 0 {
		req.Limit = 200
	}
	err := models.AddJdContainer(req.Address, req.Cid, req.Secret, req.Weight, req.Limit, req.Mode, req.Resident)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "添加成功，重启后生效"}
	c.ServeJSON()
}

// UpdateJdContainer 更新京东容器
func (c *AdminApiController) UpdateJdContainer() {
	var req struct {
		Index    int    `json:"index"`
		Address  string `json:"address"`
		Cid      string `json:"cid"`
		Secret   string `json:"secret"`
		Weight   int    `json:"weight"`
		Limit    int    `json:"limit"`
		Mode     string `json:"mode"`
		Resident string `json:"resident"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	err := models.UpdateJdContainer(req.Index, req.Address, req.Cid, req.Secret, req.Weight, req.Limit, req.Mode, req.Resident)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "更新成功，重启后生效"}
	c.ServeJSON()
}

// DeleteJdContainer 删除京东容器
func (c *AdminApiController) DeleteJdContainer() {
	var req struct {
		Index int `json:"index"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	err := models.DeleteJdContainer(req.Index)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功，重启后生效"}
	c.ServeJSON()
}

// ===================== 青龙活动容器 CRUD =====================

// AddQLConfig 添加青龙配置
func (c *AdminApiController) AddQLConfig() {
	var req struct {
		Name         string `json:"name"`
		Host         string `json:"host"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		Timeout      int    `json:"timeout"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.Name == "" || req.Host == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "名称和地址不能为空"}
		c.ServeJSON()
		return
	}
	if req.Timeout == 0 {
		req.Timeout = 80
	}
	err := models.AddQLConfig(req.Name, req.Host, req.ClientID, req.ClientSecret, req.Timeout)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "添加成功"}
	c.ServeJSON()
}

// UpdateQLConfig 更新青龙配置
func (c *AdminApiController) UpdateQLConfig() {
	var req struct {
		OldName      string `json:"oldName"`
		Host         string `json:"host"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		Timeout      int    `json:"timeout"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	err := models.UpdateQLConfig(req.OldName, req.Host, req.ClientID, req.ClientSecret, req.Timeout)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "更新成功"}
	c.ServeJSON()
}

// DeleteQLConfig 删除青龙配置
func (c *AdminApiController) DeleteQLConfig() {
	var req struct {
		Name string `json:"name"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	err := models.DeleteQLConfig(req.Name)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// ===================== 用户管理扩展 =====================

// CreateUser 创建用户
func (c *AdminApiController) CreateUser() {
	var req struct {
		Wxid     string `json:"wxid"`
		QQ       string `json:"qq"`
		Coin     int    `json:"coin"`
		IsAdmin  bool   `json:"isAdmin"`
		Nickname string `json:"nickname"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	err := models.CreateUser(req.Wxid, req.QQ, req.Coin, req.IsAdmin, req.Nickname)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "创建失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "创建成功"}
	c.ServeJSON()
}

// UpdateUser 更新用户
func (c *AdminApiController) UpdateUser() {
	var req struct {
		ID       int    `json:"id"`
		Wxid     string `json:"wxid"`
		QQ       string `json:"qq"`
		Coin     int    `json:"coin"`
		Class    string `json:"class"`
		IsAdmin  bool   `json:"isAdmin"`
		Nickname string `json:"nickname"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.ID == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "无效的ID"}
		c.ServeJSON()
		return
	}
	err := models.UpdateUser(req.ID, req.Wxid, req.QQ, req.Coin, req.Class, req.IsAdmin, req.Nickname)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "修改失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "修改成功"}
	c.ServeJSON()
}

// BatchDeleteUsers 批量删除用户
func (c *AdminApiController) BatchDeleteUsers() {
	var req struct {
		IDs []int `json:"ids"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if len(req.IDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择要删除的用户"}
		c.ServeJSON()
		return
	}
	err := models.BatchDeleteUsers(req.IDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "删除失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// BatchUpdateUserCoins 批量修改用户积分
func (c *AdminApiController) BatchUpdateUserCoins() {
	var req struct {
		Numbers []int `json:"numbers"`
		Coin    int   `json:"coin"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if len(req.Numbers) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择要修改的用户"}
		c.ServeJSON()
		return
	}
	err := models.BatchUpdateUserCoins(req.Numbers, req.Coin)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "修改失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	for _, num := range req.Numbers {
		models.RecordCoinLog(num, req.Coin, "管理员操作", fmt.Sprintf("后台批量设置积分为%d", req.Coin))
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "修改成功"}
	c.ServeJSON()
}

// ===================== 批量操作 =====================

// BatchDeleteJdCookies 批量删除京东CK
func (c *AdminApiController) BatchDeleteJdCookies() {
	var req struct {
		IDs []int `json:"ids"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if len(req.IDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择要删除的记录"}
		c.ServeJSON()
		return
	}
	err := models.BatchDeleteJdCookies(req.IDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "删除失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// BatchUpdateJdCookies 批量修改京东CK
func (c *AdminApiController) BatchUpdateJdCookies() {
	var req struct {
		IDs       []int  `json:"ids"`
		Priority  int    `json:"priority"`
		Available string `json:"available"`
		Note      string `json:"note"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if len(req.IDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择要修改的记录"}
		c.ServeJSON()
		return
	}
	err := models.BatchUpdateJdCookies(req.IDs, req.Priority, req.Available, req.Note)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "修改失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "修改成功"}
	c.ServeJSON()
}

// BatchDeleteEnvVars 批量删除xdd环境变量
func (c *AdminApiController) BatchDeleteEnvVars() {
	var req struct {
		IDs []int `json:"ids"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if len(req.IDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择要删除的记录"}
		c.ServeJSON()
		return
	}
	err := models.BatchDeleteEnvVars(req.IDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "删除失败: " + err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// CreateQLEnv 创建青龙环境变量
func (c *AdminApiController) CreateQLEnv() {
	var req struct {
		ConfigName string `json:"configName"`
		Name       string `json:"name"`
		Value      string `json:"value"`
		Remarks    string `json:"remarks"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if req.ConfigName == "" || req.Name == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "容器名和变量名不能为空"}
		c.ServeJSON()
		return
	}
	err := models.CreateQLEnv(req.ConfigName, req.Name, req.Value, req.Remarks)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "创建成功"}
	c.ServeJSON()
}

// BatchDeleteQLEnvs 批量删除青龙环境变量
func (c *AdminApiController) BatchDeleteQLEnvs() {
	var req struct {
		ConfigName string `json:"configName"`
		EnvIDs     []int  `json:"envIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if len(req.EnvIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择要删除的变量"}
		c.ServeJSON()
		return
	}
	err := models.BatchDeleteQLEnvs(req.ConfigName, req.EnvIDs)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// BatchUpdateQLEnvs 批量修改青龙环境变量
func (c *AdminApiController) BatchUpdateQLEnvs() {
	var req struct {
		ConfigName string `json:"configName"`
		EnvIDs     []int  `json:"envIds"`
		Remarks    string `json:"remarks"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if len(req.EnvIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择要修改的变量"}
		c.ServeJSON()
		return
	}
	err := models.BatchUpdateQLEnvs(req.ConfigName, req.EnvIDs, req.Remarks)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "修改成功"}
	c.ServeJSON()
}

// ===================== 活动人数统计 =====================

// GetActivityStats 获取活动人数统计
func (c *AdminApiController) GetActivityStats() {
	force := c.GetString("refresh") == "1" || c.GetString("force") == "1"
	snapshot := models.GetActivityStatsSnapshot(force)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":        snapshot.Stats,
			"totalAll":    snapshot.TotalAll,
			"validAll":    snapshot.ValidAll,
			"expiringAll": snapshot.ExpiringAll,
			"expiredAll":  snapshot.ExpiredAll,
			"cached":      snapshot.Cached,
			"refreshing":  snapshot.Refreshing,
			"updatedAt":   snapshot.UpdatedAt,
		},
	}
	c.ServeJSON()
}

func (c *AdminApiController) RefreshActivityStats() {
	activityID := c.GetString("activityId")
	if activityID == "" {
		activityID = c.Ctx.Input.Param(":id")
	}
	if activityID == "" {
		go models.RefreshActivityAdminStatsCache()
		c.Data["json"] = map[string]interface{}{"code": 0, "msg": "已触发后台刷新"}
		c.ServeJSON()
		return
	}
	if err := models.RefreshSingleActivityAdminStats(activityID); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "刷新成功"}
	c.ServeJSON()
}

// TriggerNotifyDeleteExpiredCKs 手动触发过期CK通知/删除检查
func (c *AdminApiController) TriggerNotifyDeleteExpiredCKs() {
	var req struct {
		Channels    []string `json:"channels"`
		ActivityIDs []string `json:"activityIds"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	channels := models.NormalizeNotifyChannels(req.Channels)
	go models.NotifyDeleteExpiredCKsWithChannels(nil, channels, req.ActivityIDs)
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "已触发过期CK通知/删除检查，后台执行中"}
	c.ServeJSON()
}

// ===================== 青龙活动授权管理 =====================

// GetActivityAuthList 获取活动授权管理列表（含用户统计）
func (c *AdminApiController) GetActivityAuthList() {
	stats := models.GetActivityAuthList()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list": stats,
		},
	}
	c.ServeJSON()
}

// BatchUpdateActivityAuth 批量增减活动授权天数
func (c *AdminApiController) BatchUpdateActivityAuth() {
	var req struct {
		ActivityID string   `json:"activityId"`
		Direction  string   `json:"direction"`
		Days       int      `json:"days"`
		EnvIDs     []int    `json:"envIds"`
		Channels   []string `json:"channels"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.ActivityID == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "活动ID不能为空"}
		c.ServeJSON()
		return
	}
	if req.Direction != "add" && req.Direction != "sub" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "操作方向必须为 add 或 sub"}
		c.ServeJSON()
		return
	}
	if req.Days < 1 || req.Days > 365 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "天数需在1-365之间"}
		c.ServeJSON()
		return
	}
	if len(req.EnvIDs) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请选择要调整的账号"}
		c.ServeJSON()
		return
	}

	channels := models.NormalizeNotifyChannels(req.Channels)
	updated, failed, err := models.BatchUpdateActivityAuth(req.ActivityID, req.Direction, req.Days, req.EnvIDs, channels)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}

	dirText := "增加"
	if req.Direction == "sub" {
		dirText = "减少"
	}
	msg := fmt.Sprintf("已完成：%s%d天授权，成功%d个", dirText, req.Days, updated)
	if failed > 0 {
		msg += fmt.Sprintf("，失败%d个", failed)
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// ConvertActivityToMonthly 一次性活动转为月扣费活动
func (c *AdminApiController) ConvertActivityToMonthly() {
	var req struct {
		ActivityID  string   `json:"activityId"`
		MonthlyCoin int      `json:"monthlyCoin"`
		SyncUsers   bool     `json:"syncUsers"`
		GrantDays   int      `json:"grantDays"`
		Channels    []string `json:"channels"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)

	if req.ActivityID == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "活动ID不能为空"}
		c.ServeJSON()
		return
	}
	if req.MonthlyCoin <= 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "每月积分必须大于0"}
		c.ServeJSON()
		return
	}
	if req.SyncUsers && (req.GrantDays < 1 || req.GrantDays > 3650) {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "授权天数需在1-3650之间"}
		c.ServeJSON()
		return
	}

	channels := models.NormalizeNotifyChannels(req.Channels)
	migrated, err := models.ConvertActivityToMonthly(req.ActivityID, req.MonthlyCoin, req.SyncUsers, req.GrantDays, channels)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}

	msg := fmt.Sprintf("活动已转为月扣费（每月%d积分）", req.MonthlyCoin)
	if req.SyncUsers {
		msg += fmt.Sprintf("，已同步迁移 %d 个用户", migrated)
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg, "data": map[string]interface{}{"migrated": migrated}}
	c.ServeJSON()
}

// ===================== 微信协议设备管理 =====================

// GetWxDevices 获取微信设备列表
func (c *AdminApiController) GetWxDevices() {
	search := c.GetString("search", "")
	list, total, online, offline := models.GetWxDeviceList(search)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":    list,
			"total":   total,
			"online":  online,
			"offline": offline,
		},
	}
	c.ServeJSON()
}

// GetWxDeviceStats 获取微信设备统计数据（仪表盘用）
func (c *AdminApiController) GetWxDeviceStats() {
	total, online, offline := models.GetWxDeviceStats()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"total":   total,
			"online":  online,
			"offline": offline,
		},
	}
	c.ServeJSON()
}

// GetUserOnlineStats 获取用户在线统计数据（仪表盘用）
func (c *AdminApiController) GetUserOnlineStats() {
	stats := models.GetUserOnlineStats()
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": stats,
	}
	c.ServeJSON()
}

// NotifyWxOffline 向所有掉线用户发送通知消息
func (c *AdminApiController) NotifyWxOffline() {
	var req struct {
		Channels []string `json:"channels"`
		WxIDs    []string `json:"wxids"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	channels := models.NormalizeNotifyChannels(req.Channels)
	go models.CheckWxOfflineAndNotifyWithChannels(channels, req.WxIDs, true)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"msg":  "通知已触发，正在后台发送",
	}
	c.ServeJSON()
}
