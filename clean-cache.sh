#!/bin/bash
# 清理安卓狗东无效构建缓存（使用 Android Studio JDK）
set -euo pipefail

cd "$(dirname "$0")"

export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
export PATH="$JAVA_HOME/bin:$PATH"

echo "🧹 清理 Gradle 与本地构建缓存..."
./gradlew clean --no-daemon || true

rm -rf app/build .gradle build

echo "✅ 缓存已清理"
