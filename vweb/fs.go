package vweb

import "embed"

// WebFs 嵌入前端HTML静态文件
//
//go:embed html/*
var WebFs embed.FS

// JsFs 嵌入前端JS静态文件
//
//go:embed js/*
var JsFs embed.FS
