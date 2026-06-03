package TenPay

import "wechatdll/comm"

type receiveHongBao struct {
	Retcode                 int
	Retmsg                  string
	SendId                  string
	Wishing                 string
	IsSender                int
	ReceiveStatus           int
	HbStatus                int
	StatusMess              string
	HbType                  int
	Watermark               string
	ScenePicSwitch          int
	PreStrainFlag           int
	SendUserName            string
	TimingIdentifier        string
	ShowYearExpression      int
	Expression_md5          string
	ShowRecNormalExpression int
}

type receiveListHongBao struct {
	Retcode         int           `json:"retcode"`
	Retmsg          string        `json:"retmsg"`
	RecNum          int           `json:"recNum"`
	TotalNum        int           `json:"totalNum"`
	TotalAmount     int           `json:"totalAmount"`
	SendId          string        `json:"sendId"`
	Amount          int           `json:"amount"`
	Wishing         string        `json:"wishing"`
	IsSender        int           `json:"isSender"`
	ReceiveId       string        `json:"receiveId"`
	HasWriteAnswer  int           `json:"hasWriteAnswer"`
	OperationHeader []interface{} `json:"operationHeader"`
	HbType          int           `json:"hbType"`
	IsContinue      int           `json:"isContinue"`
	HbStatus        int           `json:"hbStatus"`
	ReceiveStatus   int           `json:"receiveStatus"`
	StatusMess      string        `json:"statusMess"`
	HeadTitle       string        `json:"headTitle"`
	CanShare        int           `json:"canShare"`
	HbKind          int           `json:"hbKind"`
	RecAmount       int           `json:"recAmount"`
	Record          []struct {
		ReceiveAmount int    `json:"receiveAmount"`
		ReceiveTime   string `json:"receiveTime"`
		Answer        string `json:"answer"`
		ReceiveId     string `json:"receiveId"`
		State         int    `json:"state"`
		ReceiveOpenId string `json:"receiveOpenId"`
		UserName      string `json:"userName"`
	} `json:"record"`
	OperationTail struct {
		Enable int `json:"enable"`
	} `json:"operationTail"`
	AtomicFunc struct {
		Enable int `json:"enable"`
	} `json:"atomicFunc"`
	JumpChange                 int    `json:"jumpChange"`
	ChangeWording              string `json:"changeWording"`
	SendUserName               string `json:"sendUserName"`
	ChangeUrl                  string `json:"changeUrl"`
	JumpChangeType             int    `json:"jumpChangeType"`
	ShowDetailNormalExpression int    `json:"showDetailNormalExpression"`
	EnableAnswerByExpression   int    `json:"enableAnswerByExpression"`
	EnableAnswerBySelfie       int    `json:"enableAnswerBySelfie"`
}

type payHongBao struct {
	Retcode    string `json:"retcode"`
	Retmsg     string `json:"retmsg"`
	Token      string `json:"token"`
	IsFreeSms  string `json:"is_free_sms"`
	PayFlag    string `json:"pay_flag"`
	BindSerial string `json:"bind_serial"`
	ReturnUrl  string `json:"return_url"`
	EndFlag    string `json:"end_flag"`
	Payresult  []struct {
		TransactionId    string        `json:"transaction_id"`
		PayStatus        string        `json:"pay_status"`
		PayStatusName    string        `json:"pay_status_name"`
		BuyBankName      string        `json:"buy_bank_name"`
		PayTime          string        `json:"pay_time"`
		PayTimestamp     string        `json:"pay_timestamp "`
		CardTail         string        `json:"card_tail"`
		FeeType          string        `json:"fee_type"`
		ActivityInfo     []interface{} `json:"activity_info"`
		TotalFee         int           `json:"total_fee"`
		OriginalTotalFee int           `json:"original_total_fee"`
		DiscountArray    []interface{} `json:"discount_array"`
	} `json:"payresult"`
	PayResultTips        string `json:"pay_result_tips"`
	BalanceMobile        string `json:"balance_mobile"`
	BalanceHelpUrl       string `json:"balance_help_url"`
	IsUseNewPaidSuccPage int    `json:"is_use_new_paid_succ_page"`
	PaySuccBtnWording    string `json:"pay_succ_btn_wording"`
	VerifyCreTailInfo    struct {
		IsCanVerifyTail int `json:"is_can_verify_tail"`
	} `json:"verify_cre_tail_info"`
	ShoWInfo            []interface{} `json:"sho w_info"`
	FetchChargeShowInfo []interface{} `json:"fetch_charge_show_info"`
}
type createHongBao struct {
	Retcode    int    `json:"retcode"`
	Retmsg     string `json:"retmsg"`
	SendId     string `json:"sendId"`
	Reqkey     string `json:"reqkey"`
	Scene      int    `json:"scene"`
	HbKind     int    `json:"hbKind"`
	SendMsgXml string `json:"sendMsgXml"`
	IdSign     string `json:"id_sign"`
}

type HongBaoParam struct {
	Wxid         string
	Xml          string
	SendUserName string
}

type CollectmoneyModel struct {
	InvalidTime   string
	TransFerId    string
	TransactionId string
	ToUserName    string
	Wxid          string
}

type GeneratePayQCodeModel struct {
	Name  string
	Money string
	Wxid  string
}

type HongBaoQid struct {
	Wxid   string
	Xml    string
	SendID string
}

type HongBaoDetail struct {
	Wxid   string
	Xml    string
	Offset int64
	Size   int64
}

type RedPacket struct {
	RedType  uint32
	Username string
	From     uint32
	Count    uint32
	Amount   uint32
	Content  string
	Wxid     string
}

// 确认支付
type ConfirmPreTransfer struct {
	BankType    string
	BankSerial  string
	ReqKey      string
	PayPassword string
	Wxid        string
}

type receivewxhbParam struct {
	Xml              string
	D                comm.LoginData
	City             string
	Province         string
	Encrypt_key      string
	Encrypt_userinfo string
	InWay            string
}

type qianwxhbParam struct {
	Xml              string
	D                comm.LoginData
	City             string
	Province         string
	Encrypt_key      string
	Encrypt_userinfo string
	SendID           string
	InWay            uint32
	channelId        string
	MsgType          string
}

type TransferOperationParam struct {
	Wxid   string
	Xml    string
	ToWxid string
}
