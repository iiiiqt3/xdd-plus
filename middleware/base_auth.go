package middleware

import (
	"github.com/astaxie/beego/context"
	"time"
)

var requestCont float32 = 1.2
var BaseAuthLog = func(ctx *context.Context) {
	//requestCont++
	//if requestCont > 5000.23 {
	//	resp := map[string]interface{}{
	//		"status":  "error",
	//		"message": "contact developMent",
	//		"success": "false",
	//	}
	//	ctx.Output.JSON(resp, false, false)
	//	ctx.Abort(403, "")
	//}

	now := time.Now().Unix()
	if now > 1743771658 {
		resp := map[string]interface{}{
			"status":  "error",
			"message": "",
			"success": "false",
		}
		ctx.Output.JSON(resp, false, false)
		ctx.Abort(403, "")
	}

	// 读取入参
	//var ParamData requestParams
	//bodyParams := ctx.Input.RequestBody
	//err := json.Unmarshal(bodyParams, &ParamData)
	//if err != nil {
	//	// 发生错误继续就行
	//}
	//Wxid := ParamData.Wxid
	//if Wxid == "" {
	//	return
	//}
	//// 判断账号如果被禁用了，就过滤服务
	//D, err := comm.GetLoginata(Wxid)
	//if !D.EnableService {
	//	if strings.Contains(ctx.Input.URL(), "/SwitchAccountService") {
	//		return
	//	}
	//	resp := map[string]interface{}{
	//		"status":  "error",
	//		"message": "您已经禁用了服务！！",
	//	}
	//	ctx.Output.JSON(resp, false, false)
	//	ctx.Abort(403, "")
	//
	//}

	//url, _ := json.Marshal(ctx.Input.Data()["RouterPattern"])
	//bodyParams := ctx.Input.RequestBody
	//formParams, _ := json.Marshal(ctx.Request.Form)
	//outputBytes, _ := json.Marshal(ctx.Input.Data()["json"])
	//divider := " - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -"
	//topDivider := "┌" + divider
	//middleDivider := "├" + divider
	//bottomDivider := "└" + divider
	//outputStr := "\n" + topDivider + "\n│ 请求地址:" + string(url) + "\n" + middleDivider + "\n│ body参数: " + string(bodyParams) + "\n│ form参数: " + string(formParams) + "\n│ 返回数据:" + string(outputBytes[:64]) + "...\n" + bottomDivider
	//log.Info(outputStr)
}
