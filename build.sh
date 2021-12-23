#! /bin/bash
## author:arjun
## 919M build docker image script

mode=$1
ver=$2

if [ ! -n "$mode" ]; then
  mode="release"
fi

if [ ! -n "$ver" ]; then
  ver="1.0.0"
fi
#
#if [[ ! -d "./ci/app" ]]; then
#    mkdir ./ci/app
# else
#     echo "文件夹已存在"
#fi

case $mode in
  "release")
  CGO_ENABLED=0  GOOS=linux  GOARCH=amd64  go build -ldflags "-s -w" -o server main.go
#  cp ./ci/Dockerfile-Release ./ci/Dockerfile
#  cp config.yaml ./ci/app/
  ;;

  "debug")
  CGO_ENABLED=0  GOOS=linux  GOARCH=amd64  go build -o server main.go
#  cp ./ci/Dockerfile-Debug ./ci/Dockerfile
#  cp config-test.yaml ./ci/app/config.yaml
  ver=${ver}".SNAPSHOT"
  ;;
esac

#mv server ./ci/app/
#
#cd ./ci
#
#docker build --tag=registry.cn-shenzhen.aliyuncs.com/topcode/dal-sk919m:${ver} .
#
#if [ "$?" == 0 ]
#then
#  echo $?
#  exit 1
#fi

## docker tag registry.cn-shenzhen.aliyuncs.com/topcode/dal-sk919m:1.0.0.SNAPSHOT topcodes/dal-sk919m

## docker login --username=caiyagang@163.com registry.cn-shenzhen.aliyuncs.com

#docker push registry.cn-shenzhen.aliyuncs.com/topcode/dal-sk919m:${ver}

#rm Dockerfile
#rm -rf ./app