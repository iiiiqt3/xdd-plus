#!/bin/bash
# 安卓狗东 Debug 打包（使用本机 Android Studio 自带 JDK）
set -euo pipefail

cd "$(dirname "$0")"

export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
export PATH="$JAVA_HOME/bin:$PATH"

if [[ ! -d "$JAVA_HOME" ]]; then
    echo "❌ 未找到 Android Studio JBR: $JAVA_HOME"
    exit 1
fi

VERSION_NAME=$(grep 'versionName' app/build.gradle.kts | head -1 | sed 's/.*"\(.*\)".*/\1/')

echo "📦 使用 Android Studio JDK 打包 Debug v${VERSION_NAME}..."
./gradlew assembleDebug -x lint --no-daemon

APK="app/build/outputs/apk/debug/app-debug.apk"
OUT="goudong-v${VERSION_NAME}-debug.apk"
if [[ -f "$APK" ]]; then
    cp "$APK" "$OUT"
    echo "✅ 打包完成！"
    echo "📍 文件位置: $(pwd)/$OUT"
else
    echo "❌ 打包失败，请检查错误信息"
    exit 1
fi
