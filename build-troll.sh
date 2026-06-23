#!/bin/bash
# 巨魔 (TrollStore) 安装包构建脚本 — 无需 Apple 签名
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

SCHEME="狗东"
CONFIG="Release"
DERIVED="$ROOT/build/DerivedData"
APP_PATH="$DERIVED/Build/Products/Release-iphoneos/狗东.app"
IPA_PATH="$ROOT/狗东-4.1.ipa"

echo "==> 编译 $SCHEME ($CONFIG, 无签名)..."
xcodebuild \
  -scheme "$SCHEME" \
  -configuration "$CONFIG" \
  -destination 'generic/platform=iOS' \
  -derivedDataPath "$DERIVED" \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGN_IDENTITY="-" \
  DEVELOPMENT_TEAM="" \
  build

if [[ ! -d "$APP_PATH" ]]; then
  echo "错误: 未找到 $APP_PATH"
  exit 1
fi

echo "==> 打包 IPA..."
rm -rf "$ROOT/build/Payload" "$IPA_PATH"
mkdir -p "$ROOT/build/Payload"
cp -R "$APP_PATH" "$ROOT/build/Payload/"
(
  cd "$ROOT/build"
  zip -qr "$IPA_PATH" Payload
  rm -rf Payload
)

echo ""
echo "✅ 完成: $IPA_PATH"
echo "   用 TrollStore 安装此 IPA 即可（无需签名）"
