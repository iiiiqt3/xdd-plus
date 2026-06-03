package Mmtls

import (
	"encoding/hex"
	"wechatdll/clientsdk/baseutils"
)

type Separatea struct {
	title  string
	length uint64
	val    []byte
}

// 分包
func Separate(Data []byte) []Separatea {
	var NewData []Separatea
	for {
		if len(Data) > 0 {
			Len := Data[3:5]
			title := hex.EncodeToString(Data[:1])
			NewData = append(NewData, Separatea{
				title:  title,
				length: baseutils.Hex2int(&Len),
				val:    Data[5 : int64(baseutils.Hex2int(&Len))+5],
			})
			Data = Data[5+int64(baseutils.Hex2int(&Len)):]
		} else {
			break
		}
	}
	return NewData
}
