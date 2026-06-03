#### linux编译
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64

# 生成 swagger
bee generate docs

#### windows 编译
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -tags windows -o wechat_win.exe main.go

go build -o main.exe main.go

# 生成 swagger
bee generate docs