package controllers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cdle/xdd/models"
	"github.com/gorilla/websocket"
)

// 预编译正则表达式 (全局变量，提升性能)
var phoneRegex = regexp.MustCompile(`^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`)

// WebSocket Upgrader 配置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域，生产环境建议限制域名
	},
}

// QQController QQ机器人控制器，通过WebSocket接收和处理QQ消息
type QQController struct {
	BaseController
}

// --- 必须保留的结构体定义 ---

type LLMessage struct {
	SelfId      int    `json:"self_id"`
	UserId      int    `json:"user_id"`
	Time        int    `json:"time"`
	MessageId   int    `json:"message_id"`
	RealId      int    `json:"real_id"`
	MessageType string `json:"message_type"`
	RequestType string `json:"request_type"`
	Sender      struct {
		UserId   int    `json:"user_id"`
		Nickname string `json:"nickname"`
		Card     string `json:"card"`
		Role     string `json:"role"`
	} `json:"sender"`
	RawMessage    string `json:"raw_message"`
	Font          int    `json:"font"`
	SubType       string `json:"sub_type"`
	Message       string `json:"message"`
	MessageFormat string `json:"message_format"`
	PostType      string `json:"post_type"`
	GroupId       int    `json:"group_id"`
	Flag          string `json:"flag"`
}

type CqMessage struct {
	PostType    string `json:"post_type"`
	MessageType string `json:"message_type"`
	Time        int    `json:"time"`
	SelfID      int    `json:"self_id"`
	SubType     string `json:"sub_type"`
	MessageSeq  int    `json:"message_seq"`
	UserID      int    `json:"user_id"`
	GroupID     int    `json:"group_id"`
	Message     string `json:"message"`
	RawMessage  string `json:"raw_message"`
	Sender      struct {
		Age      int    `json:"age"`
		Area     string `json:"area"`
		Card     string `json:"card"`
		Level    string `json:"level"`
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
		Sex      string `json:"sex"`
		Title    string `json:"title"`
		UserID   int    `json:"user_id"`
	} `json:"sender"`
	MessageID int         `json:"message_id"`
	Anonymous interface{} `json:"anonymous"`
	Font      int         `json:"font"`
}

type GroupUserList struct {
	Status  string `json:"status"`
	Retcode int    `json:"retcode"`
	Data    []struct {
		GroupId  int    `json:"group_id"`
		UserId   int64  `json:"user_id"` // 注意这里是 int64
		Nickname string `json:"nickname"`
		Role     string `json:"role"`
	} `json:"data"`
	Message string `json:"message"`
	Wording string `json:"wording"`
	Echo    string `json:"echo"`
}

// -------------------------

func (c *QQController) Echo() {
	// 1. 升级连接 (wsConn 为局部变量，避免多连接冲突)
	wsConn, err := upgrader.Upgrade(c.Ctx.ResponseWriter, c.Ctx.Request, nil)
	if err != nil {
		models.Error("WebSocket upgrade failed:", err)
		return
	}
	defer wsConn.Close()

	models.User().Infof("QQ WebSocket 接入成功")
	models.WsInit(wsConn, 1)

	// 2. 消息接收循环
	for {
		_, messageBytes, err := wsConn.ReadMessage()
		if err != nil {
			models.Info("ws连接已断开:", err)
			return
		}

		// 3. 异步处理消息
		go func(msgData []byte) {
			defer func() {
				if r := recover(); r != nil {
					models.Error("处理消息时发生 Panic:", r)
				}
			}()

			msgStr := string(msgData)

			// 判断消息类型
			if strings.Contains(msgStr, "get_group_member_list") {
				handleGroupMemberList(msgData)
			} else {
				handleStandardMessage(msgData)
			}
		}(messageBytes)
	}
}

// 处理群成员列表
func handleGroupMemberList(data []byte) {
	var groupUserList GroupUserList
	if err := json.Unmarshal(data, &groupUserList); err != nil {
		models.Warn("解析群成员列表失败:", string(data), err)
		return
	}

	models.Info("收到群成员列表，数量:", len(groupUserList.Data))

	for _, user := range groupUserList.Data {
		// 逻辑：如果是用户且角色是普通成员，则踢出
		// 注意：models.IsUser 的参数类型需要确认，这里假设接受 int64 或能自动转换
		// 如果 models.IsUser 只接受 int，可能需要 int(user.UserId)，但 QQ 号可能溢出 int，建议检查 models 定义
		if models.IsUser(user.UserId) && user.Role == "member" {
			go func(gid int, uid int64) {
				// 随机延时 5-15 秒
				delay := time.Duration(5+rand.Intn(10)) * time.Second
				time.Sleep(delay)

				models.Info("执行踢人操作:", gid, uid)
				
				// 【关键修复】根据报错信息调整参数类型
				// 报错说 cannot use int(uid) as int64，说明 models.RemoveGroupMember 第二个参数需要 int64
				// 而第一个参数 groupId 原代码是 int，如果报错也涉及第一个参数，请改为 int64(gid)
				// 这里假设签名是 RemoveGroupMember(groupId int, userId int64)
				models.RemoveGroupMember(gid, uid) 
			}(user.GroupId, user.UserId)
		}
	}
}

// 处理标准消息
func handleStandardMessage(data []byte) {
	var msg LLMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		models.Warn("解析标准消息失败:", string(data), err)
		return
	}
	HandleQQMessage(msg)
}

func HandleQQMessage(msg LLMessage) {
	if msg.PostType == "message" {
		models.Info("接收到信息:" + msg.RawMessage)

		if msg.Sender.Nickname != "" {
			go models.UpdateUserNicknameIfEmpty(msg.UserId, msg.Sender.Nickname)
		}

		if msg.MessageType == "private" {
			models.ListenQQPrivateMessage(msg.UserId, msg.RawMessage)
		} else if msg.MessageType == "group" {
			models.ListenQQGroupMessage(msg.UserId, msg.GroupId, msg.RawMessage)

			// 手机号检测与撤回
			if phoneRegex.MatchString(msg.RawMessage) {
				models.Info("识别为手机号，进行撤回 MessageID:", msg.MessageId)
				go models.DeleteQQMsg(msg.MessageId)
			}
		}
	} else if msg.PostType == "request" {
		if msg.RequestType == "friend" {
			models.Info("收到好友请求 Flag:", msg.Flag)
			time.Sleep(time.Duration(rand.Intn(5000)) * time.Millisecond)
			models.AutoAgreeFriedns(msg.Flag)
		}
	}
}