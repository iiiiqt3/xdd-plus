package controllers

// CompatGatewayPaths 青龙脚本兼容网关路径（与 xdd-G 对齐，避免 404）
func CompatGatewayPaths() []string {
	return []string{
		"/api/v1/wx/user/status",
		"/api/wx/user/status",
		"/api/WxApi/UserStatus",
		"/wechat/api/getLatestUserKey",
		"/wx/oauth",
		"/api/wx/oauth",
		"/TenPay/Collectmoney",
		"/TenPay/ConfirmPreTransferApi",
		"/TenPay/mchtransferTransferAutoConfirm",
		"/api/TenPay/mchtransferTransferAutoConfirm",
		"/api/v1/wx/user/delete",
		"/api/WxApi/UserDelete",
		"/api/v1/wx/login/code",
		"/api/WxApi/LoginCode",
		"/api/v1/wx/login/status",
		"/api/WxApi/LoginStatus",
		"/api/v1/wx/login/again",
		"/api/WxApi/LoginAgain",
		"/api/v1/wx/login/awake",
		"/api/WxApi/LoginAwake",
		"/api/v1/wx/login/twice",
		"/api/WxApi/LoginTwice",
		"/api/v1/wx/login/logout",
		"/api/WxApi/LoginLogout",
		"/api/v1/wx/app/get/code",
		"/api/wx/app/get/code",
		"/wx/app/get/code",
		"/wx/app/code",
		"/api/v1/wx/app/get/jscode",
		"/api/Wxapp/JSLogin",
		"/api/Wxapp/v1/GetCode",
		"/api/v1/wx/app/get/sessionid",
		"/api/v1/wx/app/get/sessionid/qrcode",
		"/api/v1/wx/app/qrcode/auth/login",
		"/api/Wxapp/JSGetSessionid",
		"/api/Wxapp/JSGetSessionidQRcode",
		"/api/Wxapp/v1/GetSessionid",
		"/api/Wxapp/v1/GetSessionidQRcode",
		"/api/Wxapp/QrcodeAuthLogin",
		"/api/Wxapp/v1/QrcodeAuthLogin",
		"/api/v1/wx/app/get/all/mobile",
		"/api/v1/wx/app/get/all",
		"/wx/app/get/all/mobile",
		"/api/Wxapp/GetAllMobile",
		"/api/Wxapp/v1/GetAllMobile",
		"/api/v1/wx/app/get/user/openid",
		"/api/Wxapp/GetUserOpenId",
		"/api/Wxapp/v1/GetUserOpenId",
		"/api/v1/wx/app/get/userinfo",
		"/api/v1/wx/app/get/user/info",
		"/api/v1/wx/app/call/function",
		"/api/v1/wx/app/call/funtion",
		"/api/wx/app/call/function",
		"/wx/app/call/function",
		"/api/Wxapp/CloudCallFunction",
		"/api/Wxapp/v1/CallFunction",
		"/api/v1/wx/app/operate/wxdata",
		"/api/wx/app/operate/wxdata",
		"/api/Wxapp/JSOperateWxData",
		"/api/Wxapp/v1/OperateWxData",
		"/api/v1/wx/app/add/record",
		"/api/Wxapp/AddWxAppRecord",
		"/api/Wxapp/v1/AddRecord",
		"/api/v1/wx/app/add/mobile",
		"/api/Wxapp/AddMobile",
		"/api/Wxapp/v1/AddMobile",
		"/api/v1/wx/app/del/mobile",
		"/api/Wxapp/DelMobile",
		"/api/Wxapp/v1/DelMobile",
		"/api/v1/wx/app/upload/avatar",
		"/api/Wxapp/UploadAvatarImg",
		"/api/Wxapp/v1/UploadAvatar",
		"/api/v1/wx/app/add/avatar",
		"/api/Wxapp/AddAvatar",
		"/api/Wxapp/v1/AddAvatar",
		"/api/v1/wx/app/get/random/avatar",
		"/api/Wxapp/GetRandomAvatar",
		"/api/Wxapp/v1/GetRandomAvatar",
		"/api/v1/wx/tools/update/step",
		"/api/v1/wx/tools/report/motion",
		"/api/v1/wx/tools/get/werun",
		"/api/Tools/UpdateStepNumberApi",
		"/api/User/ReportMotion",
		"/api/Tools/GetWeRunData",
		"/api/Tools/GetBoundHardDevices",
		"/api/Tools/GetA8Key",
		"/api/v1/wx/offical/get/a8key",
		"/api/v1/wx/c",
		"/api/OfficialAccounts/Follow",
		"/api/OfficialAccounts/GetAppMsgExt",
		"/api/OfficialAccounts/GetAppMsgExtLike",
		"/api/OfficialAccounts/JSAPIPreVerify",
		"/api/OfficialAccounts/MpGetA8Key",
		"/api/OfficialAccounts/OauthAuthorize",
		"/api/OfficialAccounts/Quit",
	}
}

type compatPathKind int

const (
	compatKindStatus compatPathKind = iota + 1
	compatKindLatestUserKey
	compatKindWxOAuth
	compatKindDelete
	compatKindGetCode
	compatKindSessionID
	compatKindGetPhone
	compatKindGetOpenID
	compatKindGetUserInfo
	compatKindCallFunction
	compatKindOperateWxData
	compatKindRefresh
	compatKindTools
	compatKindOfficial
	compatKindTenPay
	compatKindLoginMisc
	compatKindWxappMisc
	compatKindUnknown
)

func compatPathKind(path string) compatPathKind {
	switch path {
	case "/api/v1/wx/user/status", "/api/wx/user/status", "/api/WxApi/UserStatus":
		return compatKindStatus
	case "/wechat/api/getLatestUserKey":
		return compatKindLatestUserKey
	case "/wx/oauth", "/api/wx/oauth":
		return compatKindWxOAuth
	case "/api/v1/wx/user/delete", "/api/WxApi/UserDelete":
		return compatKindDelete
	case "/api/v1/wx/app/get/code", "/api/wx/app/get/code", "/wx/app/get/code", "/wx/app/code",
		"/api/v1/wx/app/get/jscode", "/api/Wxapp/JSLogin", "/api/Wxapp/v1/GetCode":
		return compatKindGetCode
	case "/api/v1/wx/app/get/sessionid", "/api/v1/wx/app/get/sessionid/qrcode",
		"/api/v1/wx/app/qrcode/auth/login",
		"/api/Wxapp/JSGetSessionid", "/api/Wxapp/JSGetSessionidQRcode",
		"/api/Wxapp/v1/GetSessionid", "/api/Wxapp/v1/GetSessionidQRcode",
		"/api/Wxapp/QrcodeAuthLogin", "/api/Wxapp/v1/QrcodeAuthLogin":
		return compatKindSessionID
	case "/api/v1/wx/app/get/all/mobile", "/api/v1/wx/app/get/all", "/wx/app/get/all/mobile",
		"/api/Wxapp/GetAllMobile", "/api/Wxapp/v1/GetAllMobile":
		return compatKindGetPhone
	case "/api/v1/wx/app/get/user/openid", "/api/Wxapp/GetUserOpenId", "/api/Wxapp/v1/GetUserOpenId":
		return compatKindGetOpenID
	case "/api/v1/wx/app/get/userinfo", "/api/v1/wx/app/get/user/info":
		return compatKindGetUserInfo
	case "/api/v1/wx/app/call/function", "/api/v1/wx/app/call/funtion",
		"/api/wx/app/call/function", "/wx/app/call/function",
		"/api/Wxapp/CloudCallFunction", "/api/Wxapp/v1/CallFunction":
		return compatKindCallFunction
	case "/api/v1/wx/app/operate/wxdata", "/api/wx/app/operate/wxdata",
		"/api/Wxapp/JSOperateWxData", "/api/Wxapp/v1/OperateWxData":
		return compatKindOperateWxData
	case "/api/v1/wx/login/again", "/api/WxApi/LoginAgain":
		return compatKindRefresh
	case "/api/v1/wx/tools/update/step", "/api/v1/wx/tools/report/motion", "/api/v1/wx/tools/get/werun",
		"/api/Tools/UpdateStepNumberApi", "/api/User/ReportMotion",
		"/api/Tools/GetWeRunData", "/api/Tools/GetBoundHardDevices":
		return compatKindTools
	case "/api/Tools/GetA8Key", "/api/v1/wx/offical/get/a8key", "/api/v1/wx/c",
		"/api/OfficialAccounts/MpGetA8Key", "/api/OfficialAccounts/OauthAuthorize",
		"/api/OfficialAccounts/JSAPIPreVerify", "/api/OfficialAccounts/GetAppMsgExt",
		"/api/OfficialAccounts/GetAppMsgExtLike", "/api/OfficialAccounts/Follow",
		"/api/OfficialAccounts/Quit":
		return compatKindOfficial
	case "/TenPay/Collectmoney", "/TenPay/ConfirmPreTransferApi",
		"/TenPay/mchtransferTransferAutoConfirm", "/api/TenPay/mchtransferTransferAutoConfirm":
		return compatKindTenPay
	case "/api/v1/wx/login/code", "/api/WxApi/LoginCode",
		"/api/v1/wx/login/status", "/api/WxApi/LoginStatus",
		"/api/v1/wx/login/awake", "/api/WxApi/LoginAwake",
		"/api/v1/wx/login/twice", "/api/WxApi/LoginTwice",
		"/api/v1/wx/login/logout", "/api/WxApi/LoginLogout":
		return compatKindLoginMisc
	case "/api/v1/wx/app/add/record", "/api/Wxapp/AddWxAppRecord", "/api/Wxapp/v1/AddRecord",
		"/api/v1/wx/app/add/mobile", "/api/Wxapp/AddMobile", "/api/Wxapp/v1/AddMobile",
		"/api/v1/wx/app/del/mobile", "/api/Wxapp/DelMobile", "/api/Wxapp/v1/DelMobile",
		"/api/v1/wx/app/upload/avatar", "/api/Wxapp/UploadAvatarImg", "/api/Wxapp/v1/UploadAvatar",
		"/api/v1/wx/app/add/avatar", "/api/Wxapp/AddAvatar", "/api/Wxapp/v1/AddAvatar",
		"/api/v1/wx/app/get/random/avatar", "/api/Wxapp/GetRandomAvatar", "/api/Wxapp/v1/GetRandomAvatar":
		return compatKindWxappMisc
	default:
		return compatKindUnknown
	}
}

func compatGatewayAction(path string) string {
	switch compatPathKind(path) {
	case compatKindGetCode:
		return "getCode"
	case compatKindGetPhone:
		return "getPhone"
	case compatKindCallFunction:
		return "callFunction"
	case compatKindOperateWxData:
		return "operateWxData"
	case compatKindStatus:
		return "userStatus"
	case compatKindLatestUserKey:
		return "getLatestUserKey"
	case compatKindGetOpenID:
		return "getOpenID"
	case compatKindGetUserInfo:
		return "getUserInfo"
	case compatKindSessionID:
		return "sessionID"
	case compatKindRefresh:
		return "refresh"
	case compatKindDelete:
		return "delete"
	case compatKindTools:
		return "tools"
	case compatKindOfficial:
		return "official"
	case compatKindTenPay:
		return "tenPay"
	case compatKindWxOAuth:
		return "wxOAuth"
	default:
		return "other"
	}
}
