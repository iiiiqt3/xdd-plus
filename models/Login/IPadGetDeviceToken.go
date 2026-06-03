package Login

import (
	"bytes"
	"compress/zlib"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"wechatdll/Algorithm"
	"wechatdll/Cilent/mm"
	"wechatdll/Mmtls"
	"wechatdll/clientsdk/baseutils"
	v08 "wechatdll/clientsdk/v08"
	"wechatdll/comm"

	"github.com/golang/protobuf/proto"
)

// ios 设备刷新 deviceToken
func FpRrefreshTokenIos(tmpUserInfo *comm.LoginData, httpclient *Mmtls.HttpClientModel) {
	//这里有个刷新逻辑
	key := fmt.Sprintf("%s%s", "wechat:deviceId:", tmpUserInfo.Deviceid_str)
	exists := comm.Exists(key)
	si := ""
	if exists {
		return
	}
	ShortHost := tmpUserInfo.ShortHost
	if ShortHost == "" {
		ShortHost = Algorithm.MmtlsShortHost
	}
	deviceTokenRsp, err := SendIosDeviceTokenRequest(httpclient, tmpUserInfo, si)
	if err != nil {
		fmt.Println("ios 请求 deviceTokenRequest error!")
	} else {
		fmt.Println("ios 请求 deviceTokenRequest ok!", deviceTokenRsp.GetTrustResponseData().GetDeviceToken())
		//保存5天
		comm.SETExpirationObj(key, &deviceTokenRsp, 60*60*24*5)
		tmpUserInfo.DeviceInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.DeviceToken = deviceTokenRsp
	}
}

// ios设备注册刷新deviceToken
func FpInitAndRrefresh(tmpUserInfo *comm.LoginData, httpclient *Mmtls.HttpClientModel) {
	//这里有个刷新逻辑
	key := fmt.Sprintf("%s%s", "wechat:deviceId:", tmpUserInfo.Deviceid_str)
	exists := comm.Exists(key)

	si := ""
	trustRes := &mm.TrustResponse{}
	if exists {
		//ios存redis
		error := comm.GETObj(key, &trustRes)
		if error != nil {
			fmt.Println("ios redis deviceTokenIos is error=" + error.Error())
		} else {
			soft_config := trustRes.GetTrustResponseData().GetSoftData().GetSoftConfig()
			soft_data := trustRes.GetTrustResponseData().GetSoftData().GetSoftData()
			si = v08.Si(soft_config, string(soft_data))
		}
	}
	// 判断出错
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("FpInitAndRrefresh from panic: %v\n", r)
			// 这里可以记录日志或者执行其他的恢复操作
			tmpUserInfo.DeviceInfo.DeviceToken = trustRes
			tmpUserInfo.DeviceToken = trustRes
		}
	}()
	deviceTokenRsp, err := SendIosDeviceTokenRequest(httpclient, tmpUserInfo, si)
	if err != nil {
		fmt.Println("ios 请求 deviceTokenRequest error!")
		tmpUserInfo.DeviceInfo.DeviceToken = trustRes
		tmpUserInfo.DeviceToken = trustRes
	} else {
		fmt.Println("ios 请求 deviceTokenRequest ok!", deviceTokenRsp.GetTrustResponseData().GetDeviceToken())
		//保存5天
		comm.SETExpirationObj(key, &deviceTokenRsp, 60*60*24*5)
		tmpUserInfo.DeviceInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.RefreshTokenDate = time.Now().Unix()
	}
}

// ios设备注册刷新deviceToken
func FpInitAndRrefreshWin(tmpUserInfo *comm.LoginData, httpclient *Mmtls.HttpClientModel) {
	//这里有个刷新逻辑
	key := fmt.Sprintf("%s%s", "wechat:deviceId:", tmpUserInfo.Deviceid_str)
	exists := comm.Exists(key)

	si := ""
	trustRes := &mm.TrustResponse{}
	if exists {
		//ios存redis
		error := comm.GETObj(key, &trustRes)
		if error != nil {
			fmt.Println("ios redis deviceTokenIos is error=" + error.Error())
		} else {
			soft_config := trustRes.GetTrustResponseData().GetSoftData().GetSoftConfig()
			soft_data := trustRes.GetTrustResponseData().GetSoftData().GetSoftData()
			si = v08.Si(soft_config, string(soft_data))
		}
	}
	// 判断出错
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("FpInitAndRrefresh from panic: %v\n", r)
			// 这里可以记录日志或者执行其他的恢复操作
			tmpUserInfo.DeviceInfo.DeviceToken = trustRes
			tmpUserInfo.DeviceToken = trustRes
		}
	}()
	deviceTokenRsp, err := SendIosDeviceTokenRequestWin(httpclient, tmpUserInfo, si)
	if err != nil {
		fmt.Println("ios 请求 deviceTokenRequest error!")
		tmpUserInfo.DeviceInfo.DeviceToken = trustRes
		tmpUserInfo.DeviceToken = trustRes
	} else {
		fmt.Println("ios 请求 deviceTokenRequest ok!", deviceTokenRsp.GetTrustResponseData().GetDeviceToken())
		//保存5天
		comm.SETExpirationObj(key, &deviceTokenRsp, 60*60*24*5)
		tmpUserInfo.DeviceInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.RefreshTokenDate = time.Now().Unix()
	}
}

func FpInitAndRrefreshCar(tmpUserInfo *comm.LoginData, httpclient *Mmtls.HttpClientModel) {
	//这里有个刷新逻辑
	key := fmt.Sprintf("%s%s", "wechat:deviceId:", tmpUserInfo.Deviceid_str)
	exists := comm.Exists(key)

	si := ""
	trustRes := &mm.TrustResponse{}
	if exists {
		//ios存redis
		error := comm.GETObj(key, &trustRes)
		if error != nil {
			fmt.Println("ios redis deviceTokenIos is error=" + error.Error())
		} else {
			soft_config := trustRes.GetTrustResponseData().GetSoftData().GetSoftConfig()
			soft_data := trustRes.GetTrustResponseData().GetSoftData().GetSoftData()
			si = v08.Si(soft_config, string(soft_data))
		}
	}
	// 判断出错
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("FpInitAndRrefresh from panic: %v\n", r)
			// 这里可以记录日志或者执行其他的恢复操作
			tmpUserInfo.DeviceInfo.DeviceToken = trustRes
			tmpUserInfo.DeviceToken = trustRes
		}
	}()
	deviceTokenRsp, err := SendIosDeviceTokenRequestCar(httpclient, tmpUserInfo, si)
	if err != nil {
		fmt.Println("ios 请求 deviceTokenRequest error!")
		tmpUserInfo.DeviceInfo.DeviceToken = trustRes
		tmpUserInfo.DeviceToken = trustRes
	} else {
		fmt.Println("ios 请求 deviceTokenRequest ok!", deviceTokenRsp.GetTrustResponseData().GetDeviceToken())
		//保存5天
		comm.SETExpirationObj(key, &deviceTokenRsp, 60*60*24*5)
		tmpUserInfo.DeviceInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.RefreshTokenDate = time.Now().Unix()
	}
}

func FpInitAndRrefreshMac(tmpUserInfo *comm.LoginData, httpclient *Mmtls.HttpClientModel) {
	//这里有个刷新逻辑
	key := fmt.Sprintf("%s%s", "wechat:deviceId:", tmpUserInfo.Deviceid_str)
	exists := comm.Exists(key)

	si := ""
	trustRes := &mm.TrustResponse{}
	if exists {
		//ios存redis
		error := comm.GETObj(key, &trustRes)
		if error != nil {
			fmt.Println("ios redis deviceTokenIos is error=" + error.Error())
		} else {
			soft_config := trustRes.GetTrustResponseData().GetSoftData().GetSoftConfig()
			soft_data := trustRes.GetTrustResponseData().GetSoftData().GetSoftData()
			si = v08.Si(soft_config, string(soft_data))
		}
	}
	// 判断出错
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("FpInitAndRrefresh from panic: %v\n", r)
			// 这里可以记录日志或者执行其他的恢复操作
			tmpUserInfo.DeviceInfo.DeviceToken = trustRes
			tmpUserInfo.DeviceToken = trustRes
		}
	}()
	deviceTokenRsp, err := SendIosDeviceTokenRequestMac(httpclient, tmpUserInfo, si)
	if err != nil {
		fmt.Println("ios 请求 deviceTokenRequest error!")
		tmpUserInfo.DeviceInfo.DeviceToken = trustRes
		tmpUserInfo.DeviceToken = trustRes
	} else {
		fmt.Println("ios 请求 deviceTokenRequest ok!", deviceTokenRsp.GetTrustResponseData().GetDeviceToken())
		//保存5天
		comm.SETExpirationObj(key, &deviceTokenRsp, 60*60*24*5)
		tmpUserInfo.DeviceInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.DeviceToken = deviceTokenRsp
		tmpUserInfo.RefreshTokenDate = time.Now().Unix()
	}
}

// 获取DeviceToken IOS
func SendIosDeviceTokenRequest(httpclient *Mmtls.HttpClientModel, userInfo *comm.LoginData, si string) (*mm.TrustResponse, error) {
	deviceIos := userInfo.DeviceInfo
	Version := userInfo.ClientVersion
	uuid1, uuid2 := baseutils.IOSUuid(userInfo.Deviceid_str)
	hec := InitHec(userInfo)
	td := &mm.TrustReq{
		Td: &mm.TrustData{
			Si: proto.String(si),
			Tdi: []*mm.TrustDeviceInfo{

				{Key: proto.String("deviceid"), Val: proto.String(userInfo.DeviceInfo.DeviceID)},
				{Key: proto.String("sdi"), Val: proto.String(deviceIos.Sdi)},
				{Key: proto.String("idfv"), Val: proto.String(uuid1)},
				{Key: proto.String("idfa"), Val: proto.String(uuid2)},
				{Key: proto.String("device_model"), Val: proto.String(deviceIos.IphoneVer)},
				{Key: proto.String("os_version"), Val: proto.String(deviceIos.OsTypeNumber)},
				{Key: proto.String("core_count"), Val: proto.String(strconv.FormatUint(uint64(deviceIos.CoreCount), 10))},
				{Key: proto.String("carrier_name"), Val: proto.String(deviceIos.CarrierName)},
				{Key: proto.String("is_jailbreak"), Val: proto.String("0")},
				{Key: proto.String("device_name"), Val: proto.String(deviceIos.DeviceName)},
				{Key: proto.String("client_version"), Val: proto.String(fmt.Sprintf("%v", Version))},
				{Key: proto.String("plist_version"), Val: proto.String(fmt.Sprintf("%v", Version))},
				{Key: proto.String("language"), Val: proto.String("zh")},
				{Key: proto.String("locale_country"), Val: proto.String("CN")},
				{Key: proto.String("screen_width"), Val: proto.String("768")},
				{Key: proto.String("screen_height"), Val: proto.String("1024")},
				{Key: proto.String("install_time"), Val: proto.String(strconv.FormatUint(deviceIos.InstallTime, 10))},
				{Key: proto.String("kern_boottime"), Val: proto.String(strconv.FormatUint(deviceIos.KernBootTime, 10))},

				{Key: proto.String("ft_sysverplist_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Sysverplist.Inode))},
				{Key: proto.String("ft_sysverplist_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Statime.Tvsec, deviceIos.Sysverplist.Statime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stmtime.Tvsec, deviceIos.Sysverplist.Stmtime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stctime.Tvsec, deviceIos.Sysverplist.Stctime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stbtime.Tvsec, deviceIos.Sysverplist.Stbtime.Tvnsec))},

				{Key: proto.String("ft_var_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Var.Inode))},
				{Key: proto.String("ft_var_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Statime.Tvsec, deviceIos.Var.Statime.Tvnsec))},
				{Key: proto.String("ft_var_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stmtime.Tvsec, deviceIos.Var.Stmtime.Tvnsec))},
				{Key: proto.String("ft_var_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stctime.Tvsec, deviceIos.Var.Stctime.Tvnsec))},
				{Key: proto.String("ft_var_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stbtime.Tvsec, deviceIos.Var.Stbtime.Tvnsec))},

				{Key: proto.String("ft_etcgroup_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Etcgroup.Inode))},
				{Key: proto.String("ft_etcgroup_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Statime.Tvsec, deviceIos.Etcgroup.Statime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stmtime.Tvsec, deviceIos.Etcgroup.Stmtime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stctime.Tvsec, deviceIos.Etcgroup.Stctime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stbtime.Tvsec, deviceIos.Etcgroup.Stbtime.Tvnsec))},

				{Key: proto.String("ft_etchosts_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Etchosts.Inode))},
				{Key: proto.String("ft_etchosts_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Statime.Tvsec, deviceIos.Etchosts.Statime.Tvnsec))},
				{Key: proto.String("ft_etchosts_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stmtime.Tvsec, deviceIos.Etchosts.Stmtime.Tvnsec))},
				{Key: proto.String("ft_etchosts_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stctime.Tvsec, deviceIos.Etchosts.Stctime.Tvnsec))},
				{Key: proto.String("ft_etchosts_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stbtime.Tvsec, deviceIos.Etchosts.Stbtime.Tvnsec))},

				{Key: proto.String("ft_dyldcache_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Dyldcache.Inode))},
				{Key: proto.String("ft_dyldcache_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Statime.Tvsec, deviceIos.Dyldcache.Statime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stmtime.Tvsec, deviceIos.Dyldcache.Stmtime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stctime.Tvsec, deviceIos.Dyldcache.Stctime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stbtime.Tvsec, deviceIos.Dyldcache.Stbtime.Tvnsec))},
			},
		},
	}
	pb, _ := proto.Marshal(td)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(pb)
	w.Close()

	zt := new(Algorithm.ZT)
	zt.Init()
	encData := zt.Encrypt(b.Bytes())
	randKey := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, randKey)
	fp := &mm.FPFresh{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    []byte{},
			Uin:           proto.Uint32(0),
			DeviceId:      append([]byte(userInfo.DeviceInfo.DeviceID), 0),
			ClientVersion: proto.Int32(int32(userInfo.ClientVersion)),
			DeviceType:    []byte(userInfo.DeviceType),
			Scene:         proto.Uint32(0),
		},
		SessKey: randKey,
		Ztdata: &mm.ZTData{
			Version:   proto.String("00000008\x00"),
			Encrypted: proto.Uint32(1),
			Data:      encData,
			TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
			Optype:    proto.Uint32(5),
			Uin:       proto.Uint32(0),
		},
	}
	reqData, _ := proto.Marshal(fp)

	hecData := hec.HybridEcdhPackIosEn(3789, 0, nil, reqData)
	recvData, err := httpclient.MMtlsPost(userInfo.ShortHost, "/cgi-bin/micromsg-bin/fpinitnl", hecData, userInfo.Proxy)
	if err != nil {
		return &mm.TrustResponse{}, err
	}
	if len(recvData) <= 31 {
		return &mm.TrustResponse{}, errors.New(hex.EncodeToString(recvData))
	}
	ph := hec.HybridEcdhPackIosUn(recvData)
	DTResp := &mm.TrustResponse{}
	_ = proto.Unmarshal(ph.Data, DTResp)
	return DTResp, nil
}

// 20250320 改成绕ipad模式 采用win模式
func SendIosDeviceTokenRequestWin(httpclient *Mmtls.HttpClientModel, userInfo *comm.LoginData, si string) (*mm.TrustResponse, error) {
	deviceIos := userInfo.DeviceInfo
	Version := userInfo.ClientVersion
	uuid1, uuid2 := baseutils.IOSUuid(userInfo.Deviceid_str)
	hec := InitHec(userInfo)
	td := &mm.TrustReq{
		Td: &mm.TrustData{
			Si: proto.String(si),
			Tdi: []*mm.TrustDeviceInfo{

				{Key: proto.String("deviceid"), Val: proto.String(userInfo.DeviceInfo.DeviceID)},
				{Key: proto.String("sdi"), Val: proto.String(deviceIos.Sdi)},
				{Key: proto.String("idfv"), Val: proto.String(uuid1)},
				{Key: proto.String("idfa"), Val: proto.String(uuid2)},
				{Key: proto.String("device_model"), Val: proto.String(Algorithm.IPadDeviceTypeWin)},
				{Key: proto.String("os_version"), Val: proto.String(Algorithm.IPadModelWin)},
				{Key: proto.String("core_count"), Val: proto.String(strconv.FormatUint(uint64(deviceIos.CoreCount), 10))},
				{Key: proto.String("carrier_name"), Val: proto.String(deviceIos.CarrierName)},
				{Key: proto.String("is_jailbreak"), Val: proto.String("0")},
				{Key: proto.String("device_name"), Val: proto.String(deviceIos.DeviceName)},
				{Key: proto.String("client_version"), Val: proto.String(fmt.Sprintf("%v", Version))},
				{Key: proto.String("plist_version"), Val: proto.String(fmt.Sprintf("%v", Version))},
				{Key: proto.String("language"), Val: proto.String("zh")},
				{Key: proto.String("locale_country"), Val: proto.String("CN")},
				{Key: proto.String("screen_width"), Val: proto.String("768")},
				{Key: proto.String("screen_height"), Val: proto.String("1024")},
				{Key: proto.String("install_time"), Val: proto.String(strconv.FormatUint(deviceIos.InstallTime, 10))},
				{Key: proto.String("kern_boottime"), Val: proto.String(strconv.FormatUint(deviceIos.KernBootTime, 10))},

				{Key: proto.String("ft_sysverplist_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Sysverplist.Inode))},
				{Key: proto.String("ft_sysverplist_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Statime.Tvsec, deviceIos.Sysverplist.Statime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stmtime.Tvsec, deviceIos.Sysverplist.Stmtime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stctime.Tvsec, deviceIos.Sysverplist.Stctime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stbtime.Tvsec, deviceIos.Sysverplist.Stbtime.Tvnsec))},

				{Key: proto.String("ft_var_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Var.Inode))},
				{Key: proto.String("ft_var_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Statime.Tvsec, deviceIos.Var.Statime.Tvnsec))},
				{Key: proto.String("ft_var_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stmtime.Tvsec, deviceIos.Var.Stmtime.Tvnsec))},
				{Key: proto.String("ft_var_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stctime.Tvsec, deviceIos.Var.Stctime.Tvnsec))},
				{Key: proto.String("ft_var_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stbtime.Tvsec, deviceIos.Var.Stbtime.Tvnsec))},

				{Key: proto.String("ft_etcgroup_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Etcgroup.Inode))},
				{Key: proto.String("ft_etcgroup_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Statime.Tvsec, deviceIos.Etcgroup.Statime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stmtime.Tvsec, deviceIos.Etcgroup.Stmtime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stctime.Tvsec, deviceIos.Etcgroup.Stctime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stbtime.Tvsec, deviceIos.Etcgroup.Stbtime.Tvnsec))},

				{Key: proto.String("ft_etchosts_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Etchosts.Inode))},
				{Key: proto.String("ft_etchosts_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Statime.Tvsec, deviceIos.Etchosts.Statime.Tvnsec))},
				{Key: proto.String("ft_etchosts_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stmtime.Tvsec, deviceIos.Etchosts.Stmtime.Tvnsec))},
				{Key: proto.String("ft_etchosts_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stctime.Tvsec, deviceIos.Etchosts.Stctime.Tvnsec))},
				{Key: proto.String("ft_etchosts_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stbtime.Tvsec, deviceIos.Etchosts.Stbtime.Tvnsec))},

				{Key: proto.String("ft_dyldcache_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Dyldcache.Inode))},
				{Key: proto.String("ft_dyldcache_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Statime.Tvsec, deviceIos.Dyldcache.Statime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stmtime.Tvsec, deviceIos.Dyldcache.Stmtime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stctime.Tvsec, deviceIos.Dyldcache.Stctime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stbtime.Tvsec, deviceIos.Dyldcache.Stbtime.Tvnsec))},
			},
		},
	}
	pb, _ := proto.Marshal(td)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(pb)
	w.Close()

	zt := new(Algorithm.ZT)
	zt.Init()
	encData := zt.Encrypt(b.Bytes())
	randKey := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, randKey)
	fp := &mm.FPFresh{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    []byte{},
			Uin:           proto.Uint32(0),
			DeviceId:      append([]byte(userInfo.DeviceInfo.DeviceID), 0),
			ClientVersion: proto.Int32(int32(userInfo.ClientVersion)),
			DeviceType:    []byte(userInfo.DeviceType),
			Scene:         proto.Uint32(0),
		},
		SessKey: randKey,
		Ztdata: &mm.ZTData{
			Version:   proto.String("00000008\x00"),
			Encrypted: proto.Uint32(1),
			Data:      encData,
			TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
			Optype:    proto.Uint32(5),
			Uin:       proto.Uint32(0),
		},
	}
	reqData, _ := proto.Marshal(fp)

	hecData := hec.HybridEcdhPackIosEn(3789, 0, nil, reqData)
	recvData, err := httpclient.MMtlsPost(userInfo.ShortHost, "/cgi-bin/micromsg-bin/fpinitnl", hecData, userInfo.Proxy)
	if err != nil {
		return &mm.TrustResponse{}, err
	}
	if len(recvData) <= 31 {
		return &mm.TrustResponse{}, errors.New(hex.EncodeToString(recvData))
	}
	ph := hec.HybridEcdhPackIosUn(recvData)
	DTResp := &mm.TrustResponse{}
	_ = proto.Unmarshal(ph.Data, DTResp)
	return DTResp, nil
}

func SendIosDeviceTokenRequestCar(httpclient *Mmtls.HttpClientModel, userInfo *comm.LoginData, si string) (*mm.TrustResponse, error) {
	deviceIos := userInfo.DeviceInfo
	Version := userInfo.ClientVersion
	uuid1, uuid2 := baseutils.IOSUuid(userInfo.Deviceid_str)
	hec := InitHec(userInfo)
	td := &mm.TrustReq{
		Td: &mm.TrustData{
			Si: proto.String(si),
			Tdi: []*mm.TrustDeviceInfo{

				{Key: proto.String("deviceid"), Val: proto.String(userInfo.DeviceInfo.DeviceID)},
				{Key: proto.String("sdi"), Val: proto.String(deviceIos.Sdi)},
				{Key: proto.String("idfv"), Val: proto.String(uuid1)},
				{Key: proto.String("idfa"), Val: proto.String(uuid2)},
				{Key: proto.String("device_model"), Val: proto.String(Algorithm.CarModel)},
				{Key: proto.String("os_version"), Val: proto.String(Algorithm.CarOsVersion)},
				{Key: proto.String("core_count"), Val: proto.String(strconv.FormatUint(uint64(deviceIos.CoreCount), 10))},
				{Key: proto.String("carrier_name"), Val: proto.String(deviceIos.CarrierName)},
				{Key: proto.String("is_jailbreak"), Val: proto.String("0")},
				{Key: proto.String("device_name"), Val: proto.String(deviceIos.DeviceName)},
				{Key: proto.String("client_version"), Val: proto.String(fmt.Sprintf("%v", Version))},
				{Key: proto.String("plist_version"), Val: proto.String(fmt.Sprintf("%v", Version))},
				{Key: proto.String("language"), Val: proto.String("zh")},
				{Key: proto.String("locale_country"), Val: proto.String("CN")},
				{Key: proto.String("screen_width"), Val: proto.String("768")},
				{Key: proto.String("screen_height"), Val: proto.String("1024")},
				{Key: proto.String("install_time"), Val: proto.String(strconv.FormatUint(deviceIos.InstallTime, 10))},
				{Key: proto.String("kern_boottime"), Val: proto.String(strconv.FormatUint(deviceIos.KernBootTime, 10))},

				{Key: proto.String("ft_sysverplist_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Sysverplist.Inode))},
				{Key: proto.String("ft_sysverplist_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Statime.Tvsec, deviceIos.Sysverplist.Statime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stmtime.Tvsec, deviceIos.Sysverplist.Stmtime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stctime.Tvsec, deviceIos.Sysverplist.Stctime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stbtime.Tvsec, deviceIos.Sysverplist.Stbtime.Tvnsec))},

				{Key: proto.String("ft_var_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Var.Inode))},
				{Key: proto.String("ft_var_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Statime.Tvsec, deviceIos.Var.Statime.Tvnsec))},
				{Key: proto.String("ft_var_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stmtime.Tvsec, deviceIos.Var.Stmtime.Tvnsec))},
				{Key: proto.String("ft_var_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stctime.Tvsec, deviceIos.Var.Stctime.Tvnsec))},
				{Key: proto.String("ft_var_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stbtime.Tvsec, deviceIos.Var.Stbtime.Tvnsec))},

				{Key: proto.String("ft_etcgroup_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Etcgroup.Inode))},
				{Key: proto.String("ft_etcgroup_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Statime.Tvsec, deviceIos.Etcgroup.Statime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stmtime.Tvsec, deviceIos.Etcgroup.Stmtime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stctime.Tvsec, deviceIos.Etcgroup.Stctime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stbtime.Tvsec, deviceIos.Etcgroup.Stbtime.Tvnsec))},

				{Key: proto.String("ft_etchosts_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Etchosts.Inode))},
				{Key: proto.String("ft_etchosts_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Statime.Tvsec, deviceIos.Etchosts.Statime.Tvnsec))},
				{Key: proto.String("ft_etchosts_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stmtime.Tvsec, deviceIos.Etchosts.Stmtime.Tvnsec))},
				{Key: proto.String("ft_etchosts_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stctime.Tvsec, deviceIos.Etchosts.Stctime.Tvnsec))},
				{Key: proto.String("ft_etchosts_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stbtime.Tvsec, deviceIos.Etchosts.Stbtime.Tvnsec))},

				{Key: proto.String("ft_dyldcache_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Dyldcache.Inode))},
				{Key: proto.String("ft_dyldcache_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Statime.Tvsec, deviceIos.Dyldcache.Statime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stmtime.Tvsec, deviceIos.Dyldcache.Stmtime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stctime.Tvsec, deviceIos.Dyldcache.Stctime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stbtime.Tvsec, deviceIos.Dyldcache.Stbtime.Tvnsec))},
			},
		},
	}
	pb, _ := proto.Marshal(td)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(pb)
	w.Close()

	zt := new(Algorithm.ZT)
	zt.Init()
	encData := zt.Encrypt(b.Bytes())
	randKey := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, randKey)
	fp := &mm.FPFresh{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    []byte{},
			Uin:           proto.Uint32(0),
			DeviceId:      append([]byte(userInfo.DeviceInfo.DeviceID), 0),
			ClientVersion: proto.Int32(int32(userInfo.ClientVersion)),
			DeviceType:    []byte(userInfo.DeviceType),
			Scene:         proto.Uint32(0),
		},
		SessKey: randKey,
		Ztdata: &mm.ZTData{
			Version:   proto.String("00000008\x00"),
			Encrypted: proto.Uint32(1),
			Data:      encData,
			TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
			Optype:    proto.Uint32(5),
			Uin:       proto.Uint32(0),
		},
	}
	reqData, _ := proto.Marshal(fp)

	hecData := hec.HybridEcdhPackIosEn(3789, 0, nil, reqData)
	recvData, err := httpclient.MMtlsPost(userInfo.ShortHost, "/cgi-bin/micromsg-bin/fpinitnl", hecData, userInfo.Proxy)
	if err != nil {
		return &mm.TrustResponse{}, err
	}
	if len(recvData) <= 31 {
		return &mm.TrustResponse{}, errors.New(hex.EncodeToString(recvData))
	}
	ph := hec.HybridEcdhPackIosUn(recvData)
	DTResp := &mm.TrustResponse{}
	_ = proto.Unmarshal(ph.Data, DTResp)
	return DTResp, nil
}

func SendIosDeviceTokenRequestMac(httpclient *Mmtls.HttpClientModel, userInfo *comm.LoginData, si string) (*mm.TrustResponse, error) {
	deviceIos := userInfo.DeviceInfo
	Version := userInfo.ClientVersion
	uuid1, uuid2 := baseutils.IOSUuid(userInfo.Deviceid_str)
	hec := InitHec(userInfo)
	td := &mm.TrustReq{
		Td: &mm.TrustData{
			Si: proto.String(si),
			Tdi: []*mm.TrustDeviceInfo{

				{Key: proto.String("deviceid"), Val: proto.String(userInfo.DeviceInfo.DeviceID)},
				{Key: proto.String("sdi"), Val: proto.String(deviceIos.Sdi)},
				{Key: proto.String("idfv"), Val: proto.String(uuid1)},
				{Key: proto.String("idfa"), Val: proto.String(uuid2)},
				{Key: proto.String("device_model"), Val: proto.String(Algorithm.MacModel)},
				{Key: proto.String("os_version"), Val: proto.String(Algorithm.MacOsVersion)},
				{Key: proto.String("core_count"), Val: proto.String(strconv.FormatUint(uint64(deviceIos.CoreCount), 10))},
				{Key: proto.String("carrier_name"), Val: proto.String(deviceIos.CarrierName)},
				{Key: proto.String("is_jailbreak"), Val: proto.String("0")},
				{Key: proto.String("device_name"), Val: proto.String(deviceIos.DeviceName)},
				{Key: proto.String("client_version"), Val: proto.String(fmt.Sprintf("%v", Version))},
				{Key: proto.String("plist_version"), Val: proto.String(fmt.Sprintf("%v", Version))},
				{Key: proto.String("language"), Val: proto.String("zh")},
				{Key: proto.String("locale_country"), Val: proto.String("CN")},
				{Key: proto.String("screen_width"), Val: proto.String("768")},
				{Key: proto.String("screen_height"), Val: proto.String("1024")},
				{Key: proto.String("install_time"), Val: proto.String(strconv.FormatUint(deviceIos.InstallTime, 10))},
				{Key: proto.String("kern_boottime"), Val: proto.String(strconv.FormatUint(deviceIos.KernBootTime, 10))},

				{Key: proto.String("ft_sysverplist_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Sysverplist.Inode))},
				{Key: proto.String("ft_sysverplist_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Statime.Tvsec, deviceIos.Sysverplist.Statime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stmtime.Tvsec, deviceIos.Sysverplist.Stmtime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stctime.Tvsec, deviceIos.Sysverplist.Stctime.Tvnsec))},
				{Key: proto.String("ft_sysverplist_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Sysverplist.Stbtime.Tvsec, deviceIos.Sysverplist.Stbtime.Tvnsec))},

				{Key: proto.String("ft_var_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Var.Inode))},
				{Key: proto.String("ft_var_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Statime.Tvsec, deviceIos.Var.Statime.Tvnsec))},
				{Key: proto.String("ft_var_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stmtime.Tvsec, deviceIos.Var.Stmtime.Tvnsec))},
				{Key: proto.String("ft_var_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stctime.Tvsec, deviceIos.Var.Stctime.Tvnsec))},
				{Key: proto.String("ft_var_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Var.Stbtime.Tvsec, deviceIos.Var.Stbtime.Tvnsec))},

				{Key: proto.String("ft_etcgroup_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Etcgroup.Inode))},
				{Key: proto.String("ft_etcgroup_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Statime.Tvsec, deviceIos.Etcgroup.Statime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stmtime.Tvsec, deviceIos.Etcgroup.Stmtime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stctime.Tvsec, deviceIos.Etcgroup.Stctime.Tvnsec))},
				{Key: proto.String("ft_etcgroup_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etcgroup.Stbtime.Tvsec, deviceIos.Etcgroup.Stbtime.Tvnsec))},

				{Key: proto.String("ft_etchosts_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Etchosts.Inode))},
				{Key: proto.String("ft_etchosts_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Statime.Tvsec, deviceIos.Etchosts.Statime.Tvnsec))},
				{Key: proto.String("ft_etchosts_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stmtime.Tvsec, deviceIos.Etchosts.Stmtime.Tvnsec))},
				{Key: proto.String("ft_etchosts_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stctime.Tvsec, deviceIos.Etchosts.Stctime.Tvnsec))},
				{Key: proto.String("ft_etchosts_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Etchosts.Stbtime.Tvsec, deviceIos.Etchosts.Stbtime.Tvnsec))},

				{Key: proto.String("ft_dyldcache_inode"), Val: proto.String(fmt.Sprintf("%d", deviceIos.Dyldcache.Inode))},
				{Key: proto.String("ft_dyldcache_at"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Statime.Tvsec, deviceIos.Dyldcache.Statime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_mt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stmtime.Tvsec, deviceIos.Dyldcache.Stmtime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_ct"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stctime.Tvsec, deviceIos.Dyldcache.Stctime.Tvnsec))},
				{Key: proto.String("ft_dyldcache_bt"), Val: proto.String(fmt.Sprintf("%d_%d", deviceIos.Dyldcache.Stbtime.Tvsec, deviceIos.Dyldcache.Stbtime.Tvnsec))},
			},
		},
	}
	pb, _ := proto.Marshal(td)
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(pb)
	w.Close()

	zt := new(Algorithm.ZT)
	zt.Init()
	encData := zt.Encrypt(b.Bytes())
	randKey := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, randKey)
	fp := &mm.FPFresh{
		BaseRequest: &mm.BaseRequest{
			SessionKey:    []byte{},
			Uin:           proto.Uint32(0),
			DeviceId:      append([]byte(userInfo.DeviceInfo.DeviceID), 0),
			ClientVersion: proto.Int32(int32(userInfo.ClientVersion)),
			DeviceType:    []byte(userInfo.DeviceType),
			Scene:         proto.Uint32(0),
		},
		SessKey: randKey,
		Ztdata: &mm.ZTData{
			Version:   proto.String("00000008\x00"),
			Encrypted: proto.Uint32(1),
			Data:      encData,
			TimeStamp: proto.Uint32(uint32(time.Now().Unix())),
			Optype:    proto.Uint32(5),
			Uin:       proto.Uint32(0),
		},
	}
	reqData, _ := proto.Marshal(fp)

	hecData := hec.HybridEcdhPackIosEn(3789, 0, nil, reqData)
	recvData, err := httpclient.MMtlsPost(userInfo.ShortHost, "/cgi-bin/micromsg-bin/fpinitnl", hecData, userInfo.Proxy)
	if err != nil {
		return &mm.TrustResponse{}, err
	}
	if len(recvData) <= 31 {
		return &mm.TrustResponse{}, errors.New(hex.EncodeToString(recvData))
	}
	ph := hec.HybridEcdhPackIosUn(recvData)
	DTResp := &mm.TrustResponse{}
	_ = proto.Unmarshal(ph.Data, DTResp)
	return DTResp, nil
}
