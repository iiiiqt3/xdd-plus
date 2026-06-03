#!/bin/bash
set -e

CONF_DIR="/root/wechat08/conf"
CONF_FILE="${CONF_DIR}/app.conf"

# 自动生成配置文件（如果不存在）
if [ ! -f "$CONF_FILE" ]; then
    echo "[entrypoint] 配置文件不存在，自动生成..."
    mkdir -p "$CONF_DIR"
    cat > "$CONF_FILE" << 'EOF'
appname = wxapi
httpaddr = "0.0.0.0"
httpport = 9059
runmode = dev
viewspath = "template"
autorender = false
copyrequestbody = true
EnableDocs = true
redislink = 0.0.0.0:6379
redispass = 00000
redisdbnum = 8

# 管理后台登录账号密码
adminuser = admin
adminpass = admin123

syncmessage = true
msgpush = false
syncmessagebusinessuri = "http://127.0.0.1:8088/msg/SyncMessage/{0}"
logoutbusinessuri = ""

sessionon = true

SessionName = "wxapi"
ServerName = "wxapi"

longlinkenabled = true
longlinkconnecttimeout = "30m"

rabbitmq = false
rabbitmqurl = "amqp://guest:guest@10.10.11.7:5672/"
rabbitmqexchange = "wxapi"

ocrurl = ""
ocrurlgo= "http://47.119.158.126:5550/unban?tick="
ocrurlhgo="http://47.119.158.126:5550/unbanlife?huaKuaiUrl="
EOF
    echo "[entrypoint] 配置文件已生成: ${CONF_FILE}"
else
    echo "[entrypoint] 使用已有配置文件: ${CONF_FILE}"
fi

echo "[entrypoint] 启动 wechat08..."
cd /root/wechat08
exec ./wechat08
