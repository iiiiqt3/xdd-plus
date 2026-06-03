#!/bin/bash
rm ./wechat.pb.go
protoc --go_out=. wechat.proto
# 下载 
# 	protoc-gen-go v1.31.0
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
# 	protoc        v3.10.0
#   https://github.com/protocolbuffers/protobuf/releases