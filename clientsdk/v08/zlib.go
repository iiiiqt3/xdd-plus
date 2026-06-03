package v08

import (
	"runtime"
	"wechatdll/clientsdk/dynlib"

	"github.com/lunny/log"
)

var (
	zlib dynlib.DynamicLibrary
)

func init() {
	var libPath string
	switch runtime.GOOS {
	case "windows":
		libPath = "lib\\zlib.dll"
	case "linux":
		libPath = "lib/libz.so"
	default:
		panic("unsupported platform")
	}
	var err error
	//libPath = "D:\\zlib.dll" //test
	zlib, err = dynlib.NewLibrary(libPath)
	if err != nil {
		log.Error("Failed to load library: ", err)
	}
}

func Compress(input []byte) []byte {
	log.Info("Compress raw: ", len(input))
	sourceLen := len(input)
	bound, _, err := zlib.Call("compressBound", uint32(sourceLen))
	if err != nil {
		return nil
	}
	maxCompressedSize := int(bound)
	out := make([]byte, maxCompressedSize)
	outlen := uint32(maxCompressedSize)
	zlib.Call("compress", &out[0], &outlen, &input[0], sourceLen)
	log.Info("Compress after: ", outlen)
	return out[:outlen]
}
