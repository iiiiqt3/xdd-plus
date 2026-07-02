# Go 依赖代理（国内网络 proxy.golang.org 常超时，改用 goproxy.cn）
export GOPROXY ?= https://goproxy.cn,direct
export GOSUMDB ?= sum.golang.google.cn

.PHONY: deps build run clean

deps:
	go mod download

build: deps
	go build -o xdd .

run: build
	./xdd

clean:
	rm -f xdd
