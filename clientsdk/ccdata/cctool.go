package ccdata

// import (
// 	"encoding/hex"
// 	math_rand "math/rand"
// 	"strings"
// 	"wechatdll/Cilent/mm"
// 	"wechatdll/clientsdk/baseutils"

// 	"github.com/golang/protobuf/proto"
// )
// package clientsdk

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"wechatdll/Cilent/wechat"
	"wechatdll/clientsdk/baseutils"

	"github.com/golang/protobuf/proto"
)

// saeInfo ClientCheckData加密信息
var saeInfo *wechat.SaeInfo

func init() {
	initSaeInfo()
}

func initSaeInfo() {
	saeBytes, err := baseutils.ReadFile("08sae.dat") //二进制文件里面dump出来

	//saeBytes, err := baseutils.ReadFile("D:\\ipad-854\\assets\\08sae.dat")
	if err != nil {
		baseutils.PrintLog("initSaeInfo err: " + err.Error())
		return
	}
	saeInfo = &wechat.SaeInfo{}
	unmarshalErr := proto.Unmarshal(saeBytes, saeInfo)
	if unmarshalErr != nil {
		// log.Info(unmarshalErr)
	}
	pb10value := baseutils.SaePb10T(saeInfo.GetTableKey(), saeInfo.GetUnknowValue9())
	saeInfo.TableKey = pb10value
	pb12value := baseutils.SaePb12T(saeInfo.GetTableValue(), saeInfo.GetUnknowValue11())
	saeInfo.TableValue = pb12value
}

// GetSaeInfo GetSaeInfo
func GetSaeInfo() *wechat.SaeInfo {
	return saeInfo
}

// CircleShift CircleShift
func CircleShift(data []byte, offset uint32, pos uint32) []byte {
	retData := []byte{}
	retData = append(retData, data[0:]...)
	if pos == 1 {
		retData[offset+0] = data[offset+1]
		retData[offset+1] = data[offset+2]
		retData[offset+2] = data[offset+3]
		retData[offset+3] = data[offset+0]
	}

	if pos == 2 {
		retData[offset+0] = data[offset+2]
		retData[offset+2] = data[offset+0]
		retData[offset+1] = data[offset+3]
		retData[offset+3] = data[offset+1]
	}

	if pos == 3 {
		retData[offset+0] = data[offset+3]
		retData[offset+1] = data[offset+0]
		retData[offset+2] = data[offset+1]
		retData[offset+3] = data[offset+2]
	}

	return retData
}

// ShiftRows ShiftRows
func ShiftRows(data []byte) []byte {
	retData := CircleShift(data, 4, 1)
	retData = CircleShift(retData, 8, 2)
	retData = CircleShift(retData, 12, 3)
	return retData
}

// GetSecTable GetSecTable
func GetSecTable(secTable []byte, encryptRecordData []byte, secTableKey []byte, keyOffset int) []byte {
	// fmt.Println("keyOffset = ", keyOffset)
	tmpTableKeyOffset := 0
	for index := 0; index < 4; index++ {
		for secIndex := 0; secIndex < 4; secIndex++ {
			tmpOffset := index + secIndex*4
			tmpCount := 0
			recordIndex := 4*index + secIndex
			for threeIndex := 0; threeIndex < 64; threeIndex += 16 {
				tmpValue := 4 * int(encryptRecordData[recordIndex])
				tmpByte := secTableKey[keyOffset+tmpTableKeyOffset+tmpCount+tmpValue]
				secTable[tmpOffset+threeIndex] = tmpByte
				tmpCount++
			}
			tmpTableKeyOffset += 1024
		}
	}
	return secTable
}

// GetSecValue GetSecValue
func GetSecValue(encryptRecordData []byte, secTable []byte, secTableValue []byte, valueOffset int) []byte {
	// fmt.Println(secTable)
	secTableValueOffset := valueOffset
	for index := 0; index < 4; index++ {
		for secIndex := 0; secIndex < 4; secIndex++ {
			tmpValue := secTable[16*index+4*secIndex+3]
			tmpPtrOffset := 16*index + 4*secIndex + 2
			outBufferOffset := 4*index + secIndex
			for threeIndex := 0; threeIndex < 3; threeIndex++ {
				// 第一部分
				tmpHigh4Value := (secTable[tmpPtrOffset] & 0xF0) | (tmpValue&0xF0)>>4
				tmpValue12 := threeIndex * 0x100
				tmpValue14 := byte(secTableValue[secTableValueOffset+int(tmpHigh4Value&0x7F)+0x200-tmpValue12])
				if tmpHigh4Value&0x80 == 0 {
					tmpValue14 = tmpValue14 & 0x0F
				} else {
					tmpValue14 = tmpValue14 >> 4
				}

				// 第二部分
				tmpLow4Value := byte(tmpValue&0x0F | 16*secTable[tmpPtrOffset])
				tmpValue16 := byte(secTableValue[secTableValueOffset+int(tmpLow4Value&0x7F)+0x280-tmpValue12])
				if tmpLow4Value&0x80 == 0 {
					tmpValue16 = tmpValue16 & 0x0F
				} else {
					tmpValue16 = tmpValue16 >> 4
				}

				// 第三部分
				tmpValue = byte((tmpValue14 << 4) | tmpValue16)
				tmpPtrOffset = tmpPtrOffset - 1
				encryptRecordData[outBufferOffset] = tmpValue
			}
			secTableValueOffset = secTableValueOffset + 0x300
		}
	}

	return encryptRecordData
}

// GetSecValueFinal GetSecValueFinal
func GetSecValueFinal(encryptRecordData []byte, saeTableFinal []byte) []byte {
	for index := 0; index < 4; index++ {
		for secIndex := 0; secIndex < 4; secIndex++ {
			recordIndex := index*4 + secIndex
			recordValue := int(encryptRecordData[recordIndex])
			tmpOffset := index*0x400 + secIndex*0x100
			tmpValue := saeTableFinal[tmpOffset+recordValue]
			encryptRecordData[recordIndex] = tmpValue
		}
	}

	return encryptRecordData
}

// EncodeZipData 加密ClientCheckData压缩后的数据
func EncodeZipData(data []byte, encodeType int) ([]byte, error) {
	retBytes := []byte{}
	tmpEncodeData := data
	saeInfo := GetSaeInfo()
	dataLen := len(data)
	if saeInfo == nil || dataLen <= 0 {
		return retBytes, errors.New("EncodeZipData err: saeInfo == nil || dataLen <= 0")
	}
	if encodeType != 0x3060 && encodeType != 0x4095 {
		return retBytes, errors.New("EncodeZipData err: encodeType != 0x3060 && encodeType != 0x4095")
	}

	// 先按16字节补齐
	lessLen := 16 - dataLen&0xF
	if lessLen < 16 {
		for index := 0; index < lessLen; index++ {
			tmpEncodeData = append(tmpEncodeData, byte(lessLen))
		}
	}

	// IV
	ivData := saeInfo.GetIv()
	lessEncodeLength := len(tmpEncodeData)
	secTable := make([]byte, 64)

	// 每次加密16字节
	count := lessEncodeLength / 16
	for index := 0; index < count; index++ {
		tmpOffset := index * 16
		outEncodeBuffer := make([]byte, 16)
		encryptRecordBuffer := make([]byte, 16)
		// 先跟IV异或
		for secIndex := 0; secIndex < 16; secIndex++ {
			outEncodeBuffer[secIndex] = tmpEncodeData[tmpOffset+secIndex] ^ ivData[secIndex]
		}

		// 第一次换算
		for secIndex := 0; secIndex < 4; secIndex++ {
			for threeIndex := 0; threeIndex < 4; threeIndex++ {
				encryptRecordBuffer[secIndex*4+threeIndex] = outEncodeBuffer[4*threeIndex+secIndex]
			}
		}

		// 行移位
		encryptRecordBuffer = ShiftRows(encryptRecordBuffer)

		// 下一步
		for secIndex := 0; secIndex < 9; secIndex++ {
			// 获取SecTable
			if (encodeType & 0x20) == 0x20 {
				secTable = GetSecTable(secTable, encryptRecordBuffer, saeInfo.GetTableKey(), secIndex*0x4000)
			}
			// 获取SecValue
			if (encodeType & 0x40) == 0x40 {
				encryptRecordBuffer = GetSecValue(encryptRecordBuffer, secTable, saeInfo.GetTableValue(), secIndex*0x3000)
			}
			encryptRecordBuffer = ShiftRows(encryptRecordBuffer)
		}

		// 获取最后的SecValue
		if (encodeType & 0x1000) == 0x1000 {
			encryptRecordBuffer = GetSecValueFinal(encryptRecordBuffer, saeInfo.GetUnknowValue18())
			ivData = outEncodeBuffer
			for secIndex := 0; secIndex < 4; secIndex++ {
				for threeIndex := 0; threeIndex < 4; threeIndex++ {
					outEncodeBuffer[secIndex+4*threeIndex] = encryptRecordBuffer[secIndex*4+threeIndex]
				}
			}
		}

		// 保存第Index次加密后的16字节数据
		retBytes = append(retBytes, outEncodeBuffer[0:]...)
	}
	return retBytes, nil
}

func CCDPbEncode(data []byte, timestamp uint32) []byte {

	crc32_v := crc32.ChecksumIEEE(data) //pb crc32
	fmt.Printf("crc32: %d\n", crc32_v)
	newClientCheckData := &wechat.NewClientCheckData{
		C32CData:  proto.Int64(int64(crc32_v)),
		TimeStamp: proto.Int64(int64(timestamp)), //时间戳
		DataBody:  data,
	}

	ccData, err := proto.Marshal(newClientCheckData) //这是带crc32的结构
	if err != nil {
		return nil
	}

	afterCompressionCCData := baseutils.CompressByteArray(ccData)

	fmt.Println("afterCompressionCCData:", hex.EncodeToString(afterCompressionCCData))

	afterEnData, err := EncodeZipData(afterCompressionCCData, 0x3060)
	if err != nil {
		return nil
	}
	return afterEnData
}
