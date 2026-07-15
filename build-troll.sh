#!/bin/bash
# 巨魔 (TrollStore) 安装包构建脚本 — 无需 Apple 签名
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

SCHEME="狗东"
CONFIG="Release"
DERIVED="$ROOT/build/DerivedData"
APP_PATH="$DERIVED/Build/Products/Release-iphoneos/狗东.app"
ENTITLEMENTS="$ROOT/狗东.entitlements"
VERSION=$(grep 'MARKETING_VERSION' 狗东.xcodeproj/project.pbxproj | head -1 | sed 's/.*= //;s/;//')
IPA_PATH="$ROOT/狗东-${VERSION}.ipa"

if ! command -v ldid >/dev/null 2>&1; then
  echo "❌ 未找到 ldid，请先安装: brew install ldid"
  exit 1
fi

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
  echo "❌ 未找到 $APP_PATH"
  exit 1
fi

EXEC_NAME=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "$APP_PATH/Info.plist")
if [[ -z "$EXEC_NAME" || ! -f "$APP_PATH/$EXEC_NAME" ]]; then
  echo "❌ Info.plist 或主程序缺失"
  exit 1
fi

echo "==> ldid 签名 $EXEC_NAME ..."
ldid -S"$ENTITLEMENTS" "$APP_PATH/$EXEC_NAME"

echo "==> 打包 IPA..."
rm -rf "$ROOT/build/Payload" "$IPA_PATH"
mkdir -p "$ROOT/build/Payload"
ditto "$APP_PATH" "$ROOT/build/Payload/狗东.app"

if [[ ! -f "$ROOT/build/Payload/狗东.app/Info.plist" ]]; then
  echo "❌ Payload 缺少 Info.plist，打包中止"
  exit 1
fi

(
  cd "$ROOT/build"
  zip -qr "$IPA_PATH" Payload
  rm -rf Payload
)

VERIFY_DIR="$ROOT/build/ipa-verify"
rm -rf "$VERIFY_DIR"
mkdir -p "$VERIFY_DIR"
unzip -q "$IPA_PATH" -d "$VERIFY_DIR"
if [[ ! -f "$VERIFY_DIR/Payload/狗东.app/Info.plist" ]]; then
  echo "❌ IPA 校验失败：缺少 Info.plist"
  exit 1
fi
rm -rf "$VERIFY_DIR"

echo ""
echo "✅ 完成: $IPA_PATH"
echo "   用 TrollStore 安装此 IPA 即可"
