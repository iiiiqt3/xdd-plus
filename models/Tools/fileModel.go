package Tools

// UploadAppAttachModel 上传文件请求模型
type UploadAppAttachModel struct {
	Wxid string
	// 文件base64编码内容
	FileData string `json:"fileData" binding:"required"`
}
