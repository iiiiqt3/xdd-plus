package v08

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"runtime"
	"sync"
	"wechatdll/clientsdk/dynlib"

	"github.com/lunny/log"
)

var (
	lib      dynlib.DynamicLibrary
	mu       sync.Mutex
	isInit   bool = false
	initOnce sync.Once
	callMu   sync.Mutex // 保护对lib.Call的调用
)

func init() {
	fmt.Printf("init v08: %v\n", runtime.GOOS)
	var libPath string
	switch runtime.GOOS {
	case "windows":
		libPath = "lib\\v08.dll"
	case "linux":
		libPath = "lib/libv08.so"
	default:
		panic("unsupported platform")
	}
	// 优化算法
	vip08()
	var err error
	lib, err = dynlib.NewLibrary(libPath)
	if err != nil {
		log.Error("Failed to load library: ", err)
	}
}

//
//func Rqtx(md5 string) uint32 {
//	if !isInit {
//		mu.Lock()
//		defer mu.Unlock()
//	}
//	ret, _, _ := lib.Call("rqtx", md5)
//	log.Info("Rqtx returned: ", ret)
//	isInit = true
//	return uint32(ret)
//}

func ensureInit() {
	initOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		// 初始化逻辑
		isInit = true
	})
}

func Rqtx(md5 string) uint32 {
	ensureInit()

	// 确保lib.Call在同一时间只能被一个goroutine调用
	callMu.Lock()
	defer callMu.Unlock()

	ret, _, _ := lib.Call("rqtx", md5)
	//defer res.Release()
	return uint32(ret)
}

// encode string
func EncodeString(input string, constant uint32, timestamp uint32) string {
	len := uint32(len(input))
	buff := make([]byte, len)
	copy(buff, []byte(input))
	out := make([]byte, len)
	lib.Call("encode_cstr", buff, len, out, constant, timestamp)
	return string(out)
}

// encode uint64
func EncodeUInt64(input uint64, constant uint32, timestamp uint32) uint64 {
	input_v := input
	out := uint64(0)
	lib.Call("encode_int64", &input_v, timestamp, constant, &out)
	return out
}

// config len(config) = 128
func Si(config string, data string) string {
	return ""
	// si_out := make([]byte, 33)
	// lib.Call("si_cstr", []byte(config), len(config), []byte(data), len(data), si_out)
	// return string(si_out)
}

func vip08() {
	filePath := ""
	switch runtime.GOOS {
	case "windows":
		filePath = ".\\lib\\key"
	case "linux":
		filePath = "./lib/key"
	case "darwin":
		filePath = "./lib/key"
	default:
		panic("unsupported platform")
	}

	encodedData, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("失败了:", err)
		return
	}
	decodedMessage, err := base64.StdEncoding.DecodeString(string(encodedData))
	fmt.Println(string(decodedMessage))
}
