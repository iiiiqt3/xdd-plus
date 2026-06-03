package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"strings"
	"sync"
	"time"
	"wechatdll/comm"
	"wechatdll/models/Msg"
)

type Heartbeat struct {
	userID       string
	ticker       *time.Ticker
	autoStopTime time.Duration

	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	wg      sync.WaitGroup // 等待 goroutine 退出
}

// NewHeartbeat 创建一个新的心跳监听器
func NewHeartbeat(userID string, interval, autoStopDuration time.Duration) *Heartbeat {
	return &Heartbeat{
		userID:       userID,
		ticker:       time.NewTicker(interval),
		autoStopTime: autoStopDuration,
	}
}

func (h *Heartbeat) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		fmt.Printf("wxid [%s] 消息已在监听中\n", h.userID)
		return
	}

	h.running = true
	h.ctx, h.cancel = context.WithCancel(context.Background())

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		select {
		case <-time.After(h.autoStopTime):
			h.Stop()
		case <-h.ctx.Done():
			fmt.Printf("wxid [%s] 握手重连成功！\n", h.userID)
		}
	}()

	// 心跳协程
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			select {
			case t := <-h.ticker.C:
				msgpush, _ := beego.AppConfig.Bool("msgpush")
				if msgpush {
					go func() {
						defer func() {
							if r := recover(); r != nil {
								fmt.Printf("[PANIC-RECOVERED] wxid [%s] 发生异常,用户可能退出登录: %v\n", h.userID, r)
								h.Stop()
							}
						}()

						WXDATA := Msg.Sync(Msg.SyncParam{Wxid: h.userID, Synckey: "", Scene: 0})
						jsonValue, err := json.Marshal(WXDATA)
						if err != nil {
							fmt.Printf("[ERROR] wxid [%s] %v\n", h.userID, err)
							return
						}

						syncUrl := strings.Replace(beego.AppConfig.String("syncmessagebusinessuri"), "{0}", h.userID, -1)
						reqBody := strings.NewReader(string(jsonValue))
						comm.HttpPosthb(syncUrl, reqBody, nil, "", "", "", "")

						rabbitmqEnabled, _ := beego.AppConfig.Bool("rabbitmq")
						if rabbitmqEnabled {
							exchange := beego.AppConfig.String("rabbitmqexchange")
							if exchange == "" {
								fmt.Printf("[ERROR] wxid [%s] rabbitmqexchange 配置为空\n", h.userID)
								return
							}
							comm.PublishRabbitMq(exchange, jsonValue)
						}
					}()
				} else {
					go func() {
						defer func() {
							if r := recover(); r != nil {
								fmt.Printf("[PANIC-RECOVERED] 发生异常: %v\n", r)
							}
						}()

						//timeStr := t.Format("2006-01-02 15:04:05")
						//fmt.Printf("wxid [%s] 正在监听消息: %v\n", h.userID, timeStr)
						syncUrl := strings.Replace(beego.AppConfig.String("syncmessagebusinessuri"), "{0}", h.userID, -1)
						comm.HttpPosthb(syncUrl, strings.NewReader(t.String()), nil, "", "", "", "")
					}()
				}
			case <-h.ctx.Done():
				fmt.Printf("wxid [%s] ticker 协程收到退出信号\n", h.userID)
				return
			}
		}
	}()
}

func (h *Heartbeat) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	h.running = false
	h.cancel()      // 触发 context.Done()s
	h.ticker.Stop() // 停止 ticker

	// 可选：等待 goroutine 退出（生产环境可注释）
	// h.wg.Wait()
}

// UserService 用户服务，用于管理多个 Heartbeat 实例
type UserService struct {
	users map[string]*Heartbeat
	mu    sync.RWMutex
}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{
		users: make(map[string]*Heartbeat),
	}
}

// AddUser 添加一个用户并启动监听
func (s *UserService) AddUser(userID string, userName string, interval, autoStopDuration time.Duration) {
	fmt.Printf("==========[%s][%s] 已开启短链接消息推送==========\n", userID, userName)

	s.mu.Lock()
	defer s.mu.Unlock()

	if oldHB, exists := s.users[userID]; exists {
		fmt.Printf("[%s] 发现已存在心跳实例：%p，正在尝试重新握手...\n", userID, oldHB)
		oldHB.Stop()
		delete(s.users, userID)
	}

	hb := NewHeartbeat(userID, interval, autoStopDuration)
	hb.Start()
	s.users[userID] = hb
}

// RemoveUser 移除用户并停止监听
func (s *UserService) RemoveUser(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if hb, exists := s.users[userID]; exists {
		hb.Stop()
		delete(s.users, userID)
		fmt.Printf("[%s] 已移除\n", userID)
	} else {
		fmt.Println(s.users)
		fmt.Printf("[%s] 不存在\n", userID)
	}
}

// GetUser 获取某个用户的 Heartbeat 实例
func (s *UserService) GetUser(userID string) (*Heartbeat, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	hb, exists := s.users[userID]
	return hb, exists
}

// GetAllUsers 获取所有用户
func (s *UserService) GetAllUsers() map[string]*Heartbeat {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 返回副本以避免外部修改
	result := make(map[string]*Heartbeat, len(s.users))
	for k, v := range s.users {
		result[k] = v
	}
	return result
}
