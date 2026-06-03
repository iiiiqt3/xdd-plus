//go:build linux
// +build linux

package TcpPoll

import (
	"errors"
	"sync"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/comm"

	"github.com/astaxie/beego"
	log "github.com/sirupsen/logrus"
)

type TcpManager struct {
	running       bool                  // 控制是否消息loop
	connections   map[string]*TcpClient // 以关键字为key的连接池, 用于发送
	fdConnections map[int]*TcpClient    // 以fd为key的连接池, 用于接收
	poll          *epoll
}

// TcpManager单例, 使用sync.Once.Do解决并发时多次创建
var once sync.Once
var instance *TcpManager

// 获取单例Tcp
func GetTcpManager() (*TcpManager, error) {
	var err error
	longLinkEnabled, _ := beego.AppConfig.Bool("longlinkenabled")
	if !longLinkEnabled {
		return nil, errors.New("不支持长连接请求")
	}
	once.Do(func() {
		var epollInstance *epoll
		epollInstance, err = MkEpoll()
		if err != nil {
			return
		}
		instance = &TcpManager{
			running:       true,
			connections:   make(map[string]*TcpClient),
			fdConnections: make(map[int]*TcpClient),
			poll:          epollInstance,
		}
	})
	return instance, err
}

// 队列增加长连接. key: 关键字, conn: 长连接
func (manager *TcpManager) Add(key string, client *TcpClient) error {
	fd, err := manager.poll.Add(client.conn) // 将长连接增加到epoll
	if err != nil {
		return err
	}
	// 增加对照表
	manager.connections[key] = client
	manager.fdConnections[fd] = client
	return nil
}

// 队列移除长连接. client: TcpClient
func (manager *TcpManager) Remove(client *TcpClient) {
	fd := socketFD(client.conn)
	client.Terminate()
	manager.poll.Remove(client.conn)
	delete(manager.connections, client.model.Wxid)
	delete(manager.fdConnections, fd)
	client = nil
}

// 创建长连接并添加到epoll.
func (manager *TcpManager) GetClient(loginData *comm.LoginData, businessFunc BusinessFunc) (*TcpClient, error) {
	// 根据key查找是否存在已有连接, 如果已存在, 则返回
	client, ok := manager.connections[loginData.Wxid]
	if ok {
		client.model = loginData
		if businessFunc != nil {
			client.callbackMsg = businessFunc
		}
		return client, nil
	}
	// 检查LongHost
	if loginData.LongHost == "" {
		loginData.LongHost = Algorithm.MmtlsLongHost
	}
	// 创建新的连接
	client = NewTcpClient(loginData)
	if businessFunc != nil {
		client.callbackMsg = businessFunc
	}
	if err := client.Connect(); err != nil {
		return nil, err
	}
	// 将完成连接的client添加到epoll
	if err := manager.Add(loginData.Wxid, client); err != nil {
		return nil, err
	}
	timeoutSpan, _ := time.ParseDuration(beego.AppConfig.String("longlinkconnecttimeout"))
	timeoutTime := time.Now().Add(timeoutSpan)
	// 进入循环等待, 完成握手或者超时都将退出循环
	for time.Now().Before(timeoutTime) {
		time.Sleep(100 * time.Millisecond)
		// 通过client.handshaking判断是否已经完成握手
		if !client.handshaking {
			break
		}
	}
	if client.handshaking {
		// 超时没有完成握手, 报错
		client.SetStartTime(time.Now().UnixNano()) // 重置时间，关闭旧的 tcp 心跳
		manager.Remove(client)
		return nil, errors.New("mmtls 握手超时")
	}

	return client, nil
}

// 循环接收消息
func (manager *TcpManager) RunEventLoop() {
	// 无限循环直到running为false
	for manager.running == true {
		time.Sleep(100 * time.Millisecond)
		fds, waitErr := manager.poll.Wait()
		if waitErr != nil {
			log.Printf("failed to epoll wait %v", waitErr)
			continue
		}
		if len(fds) == 0 {
			continue
		}
		// fds 为有收到消息的连接文件描述
		for _, fd := range fds {
			client := manager.fdConnections[fd]
			if client == nil {
				continue
			}
			client.Once()
		}
	}
}
