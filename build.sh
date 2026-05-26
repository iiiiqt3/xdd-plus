#!/bin/bash
# 安卓狗东打包脚本
# 自动打包并重命名为 goudong-v1.9.0.apk

cd "$(dirname "$0")"

echo "📦 开始打包..."
./gradlew assembleDebug --no-daemon

if [ -f "app/build/outputs/apk/debug/app-debug.apk" ]; then
    cp "app/build/outputs/apk/debug/app-debug.apk" "goudong-v2.3.0.apk"
    echo "✅ 打包完成！"
    echo "📍 文件位置: $(pwd)/goudong-v2.3.0.apk"
else
    echo "❌ 打包失败，请检查错误信息"
fi

echo ""
echo "按任意键退出..."
read -n 1 -s
