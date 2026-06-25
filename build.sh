#!/bin/bash
# 安卓狗东打包脚本
# 自动打包并重命名为 goudong-v{versionName}.apk

cd "$(dirname "$0")"

VERSION_NAME=$(grep 'versionName' app/build.gradle.kts | head -1 | sed 's/.*"\(.*\)".*/\1/')

echo "📦 开始打包 v${VERSION_NAME}..."
./gradlew assembleRelease --no-daemon

APK="app/build/outputs/apk/release/app-release.apk"
if [ -f "$APK" ]; then
    OUT="goudong-v${VERSION_NAME}.apk"
    cp "$APK" "$OUT"
    echo "✅ 打包完成！"
    echo "📍 文件位置: $(pwd)/$OUT"
else
    echo "❌ 打包失败，请检查错误信息"
    exit 1
fi
