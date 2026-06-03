package Mmtls

import (
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"time"
	"wechatdll/models"
)

func GenNewHttpClient2(Data *MmtlsClient, domain string, P models.ProxyInfo) (httpclient *HttpClientModel) {

	mmtlsClient := &MmtlsClient{
		//不需要发送队列。
		ServerSeq: 1,
		ClientSeq: 1,
	}
	// 如果传入了 MmtlsClient，使用传入的 MmtlsClient
	if Data != nil {
		mmtlsClient = Data
	}
	// 默认的 http.Client 配置
	var httpclientModel *HttpClientModel
	var Client *http.Client
	var err error
	//设定代理
	if P.ProxyIp != "" && P.ProxyIp != "string" {
		var ProxyUser *proxy.Auth
		//设定账号和用户名
		if P.ProxyUser != "" && P.ProxyUser != "string" && P.ProxyPassword != "" && P.ProxyPassword != "string" {
			ProxyUser = &proxy.Auth{
				User:     P.ProxyUser,
				Password: P.ProxyPassword,
			}
		} else {
			ProxyUser = nil
		}
		Client, err = Socks5Client(P.ProxyIp, ProxyUser)
		if err != nil {
		}
	} else {
		Client = &http.Client{
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					conn, err := net.DialTimeout(netw, addr, time.Second*15) //设置建立连接超时
					if err != nil {
						return nil, err
					}
					conn.SetDeadline(time.Now().Add(time.Second * 15)) //设置发送接受数据超时
					return conn, nil
				},
				ResponseHeaderTimeout: time.Second * 15,
				MaxIdleConnsPerHost:   -1,   //禁用连接池缓存
				DisableKeepAlives:     true, //禁用客户端连接缓存到连接池
			},
		}
	}
	// 使用带有代理的 http.Transport 创建 http.Client
	httpclientModel = &HttpClientModel{
		mmtlsClient: mmtlsClient,
		httpClient:  Client,
		curShortip:  domain,
	}

	// 返回配置好的 HttpClientModel
	return httpclientModel
}
func GenNewHttpClient(Data *MmtlsClient, domain string) (httpclient *HttpClientModel) {

	mmtlsClient := &MmtlsClient{
		//不需要发送队列。
		ServerSeq: 1,
		ClientSeq: 1,
	}

	if Data != nil {
		mmtlsClient = Data
	}

	httpclientModel := &HttpClientModel{
		mmtlsClient: mmtlsClient,
		httpClient:  &http.Client{},
		curShortip:  domain,
	}

	return httpclientModel
}

/*func GenNewTcpClient(Data *MmtlsClient) (tcpClient *TcpClientModel) {

}*/
