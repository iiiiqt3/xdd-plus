package controllers

import (
	"encoding/json"
	"fmt"
	"wechatdll/models"
	"wechatdll/models/TenPay"

	"github.com/astaxie/beego"
)

type TenPayController struct {
	beego.Controller
}

// @Summary 自定义个人收款二维码
// @Param	body		body 	TenPay.GeMaPayQCodeParam    true	"注意参数"
// @Success 200
// @router /GeMaPayQCode [post]
func (c *TenPayController) GeMaPayQCode() {
	var ParamData TenPay.GeMaPayQCodeParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &ParamData)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.GeneratePayQCode2(ParamData)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 自定义经营个人收款单
// @Param	body		body 	TenPay.GeMaSkdPayQCodeParam    true	"注意参数"
// @Success 200
// @router /GeMaSkdPayQCode [post]
func (c *TenPayController) GeMaSkdPayQCode() {
	var ParamData TenPay.GeMaSkdPayQCodeParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &ParamData)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.GmSKDPayQCode(ParamData)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 自定义商家收款单
// @Param	body		body 	TenPay.SjSkdPayQCodeParam    true	"注意参数"
// @Success 200
// @router /SjSkdPayQCode [post]
func (c *TenPayController) SjSkdPayQCode() {
	var ParamData TenPay.SjSkdPayQCodeParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &ParamData)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.SJSKDPayQCode(ParamData)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 获取加密信息
// @Param	wxid		query 	string	true		"请输入登录后的wxid"
// @Success 200
// @router /GetEncryptInfo [post]
func (c *TenPayController) GetEncryptInfo() {
	wxid := c.GetString("wxid")
	WXDATA := TenPay.GetEncryptInfo(wxid)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 查看红包
// @Param	body	body	TenPay.QrydetailwxhbParam	true	"请求参数"
// @Success 200
// @router /Qrydetailwxhb [post]
func (c *TenPayController) Qrydetailwxhb() {
	var Data TenPay.QrydetailwxhbParam
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &Data)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.Qrydetailwxhb(Data)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 查看红包领取列表入口
func (c *TenPayController) GetRedPacketListApi() {
	var ParamData TenPay.HongBaoDetail
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &ParamData)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.GetRedPacketListApi(ParamData)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 打开红包
// @Param	body	body	TenPay.ReceivewxhbParam	true	"请求参数"
// @Success 200
// @router /Receivewxhb [post]
func (c *TenPayController) Receivewxhb() {
	var Data TenPay.ReceivewxhbParam
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &Data)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.Receivewxhb(Data)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 拆开红包
// @Param	body	body	TenPay.OpenwxhbParam	true	"请求参数"
// @Success 200
// @router /Openwxhb [post]
func (c *TenPayController) Openwxhb() {
	var Data TenPay.OpenwxhbParam
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &Data)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.Openwxhb(Data)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 抢红包
// @Param	body		body 	TenPay.HongBaoParam    true	"注意参数"
// @Success 200
// @router /OpenHongBao [post]
func (c *TenPayController) AutoHongBao() {
	var ParamData TenPay.HongBaoParam
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &ParamData)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.AutoHongBao(ParamData)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 确认收款
// @Param	body		body 	TenPay.CollectmoneyModel    true	""
// @Success 200
// @router /Collectmoney [post]
func (c *TenPayController) Collectmoney() {
	var ParamData TenPay.CollectmoneyModel
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &ParamData)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.Collectmoney(ParamData)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 创建红包
func (c *TenPayController) WXCreateRedPacketApi() {
	var ParamData TenPay.RedPacket
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &ParamData)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.WXCreateRedPacketApi(ParamData)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}

// @Summary 确认支付
func (c *TenPayController) ConfirmPreTransferApi() {
	var ParamData TenPay.ConfirmPreTransfer
	data := c.Ctx.Input.RequestBody
	err := json.Unmarshal(data, &ParamData)
	if err != nil {
		Result := models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("系统异常：%v", err.Error()),
			Data:    nil,
		}
		c.Data["json"] = &Result
		c.ServeJSON()
		return
	}
	WXDATA := TenPay.ConfirmPreTransferApi(ParamData)
	c.Data["json"] = &WXDATA
	c.ServeJSON()
}
