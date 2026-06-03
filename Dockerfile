FROM golang:1.15.4
MAINTAINER "Jacky123"

WORKDIR /usr/wic-go
RUN go env -w GO111MODULE=on
RUN ADD . /usr/wic-go
RUN go mod download
RUN go build main.go

RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

EXPOSE 8005
# 启动时执行go
ENTRYPOINT ["./main"]