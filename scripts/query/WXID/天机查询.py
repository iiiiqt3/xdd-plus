#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
天机馆用户信息查询脚本 - 纯查询版（无任务执行）
仅供学习研究使用，严禁用于商业或违规用途。

【环境变量模式（青龙）】
  WECHAT_SERVER   取 code 服务地址
  WXID_TJ         wxid，支持"备注#wxid"或直接"wxid"，多账号用换行或&分隔

【命令行模式（Go调用 / 直接运行）】
  python3 天机查询.py wxid
  python3 天机查询.py 备注#wxid
  python3 天机查询.py wxid1&wxid2

缓存文件：脚本同目录下 天机_token_cache.json（与天机.py 共用）
"""

import warnings
from urllib3.exceptions import InsecureRequestWarning
warnings.simplefilter("ignore", InsecureRequestWarning)

import requests
import json
import os
import sys
import re
import time

# ========== 常量配置 ==========
BASE_URL = "https://xcx.tianjiguan.cn"
APPID = "wx7829675630d0305e"
AUTO_LOGIN_URL = f"{BASE_URL}/api/user/autoLogin"
USER_INFO_URL = f"{BASE_URL}/api/user/userinfo"
WECHAT_CODE_URL_PATH = "/api/v1/wx/app/get/code"

WECHAT_SERVER = os.getenv("WECHAT_SERVER", "http://180.152.5.230:8011",).strip().rstrip("/")
WXID_TJ_ENV = os.getenv("WXID_TJ", "").strip()

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
CACHE_FILE = os.path.join(SCRIPT_DIR, "天机_token_cache.json")

REQUEST_TIMEOUT = 15
TOKEN_EXPIRE_BUFFER = 60

IPHONE_UA = (
    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) "
    "AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 "
    "MicroMessenger/8.0.59(0x18003b2e) NetType/4G Language/zh_CN"
)
WIN_UA = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 "
    "MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows "
    "WindowsWechat/WMPF WindowsWechat(0x63090a13) UnifiedPCWindowsWechat(0xf254181c) XWEB/19201"
)
REFERER = f"https://servicewechat.com/{APPID}/8/page-frame.html"


# ========== 工具函数 ==========
def now_ts():
    return int(time.time())


def mask_phone(phone):
    """手机号打码：保留前3位和后4位，中间用****代替"""
    phone = str(phone or "").strip()
    if len(phone) == 11 and phone.isdigit():
        return f"{phone[:3]}****{phone[7:]}"
    if len(phone) > 6:
        keep = max(3, len(phone) // 4)
        return f"{phone[:keep]}{'*' * (len(phone) - keep * 2)}{phone[-keep:]}"
    return "***"


def safe_json(data, limit=200):
    try:
        return json.dumps(data, ensure_ascii=False)[:limit]
    except Exception:
        return str(data)[:limit]


# ========== 账号解析 ==========
def parse_wxid_str(raw):
    """
    解析账号字符串，支持：
    - 直接 wxid
    - 备注#wxid
    - 多账号：换行 / & 分隔
    返回 list of {"remark": str, "wxid": str}
    """
    accounts = []
    if not raw:
        return accounts
    parts = [p.strip() for p in re.split(r"[\n&]", raw) if p.strip()]
    for idx, item in enumerate(parts, start=1):
        if "#" in item:
            remark, wxid = item.split("#", 1)
            remark = remark.strip() or f"账号{idx}"
            wxid = wxid.strip()
        else:
            wxid = item.strip()
            remark = wxid
        if wxid:
            accounts.append({"remark": remark, "wxid": wxid})
    return accounts


def get_accounts():
    """
    获取账号列表：
    1. 优先命令行参数（支持多账号 & 分隔 / 备注#wxid）
    2. 其次环境变量 WXID_TJ
    """
    if len(sys.argv) > 1:
        cli_input = " ".join(sys.argv[1:]).strip()
        accounts = parse_wxid_str(cli_input)
        if accounts:
            return accounts

    return parse_wxid_str(WXID_TJ_ENV)


# ========== 缓存管理 ==========
def load_cache():
    if not os.path.exists(CACHE_FILE):
        return {}
    try:
        with open(CACHE_FILE, "r", encoding="utf-8") as f:
            data = json.load(f)
        return data if isinstance(data, dict) else {}
    except Exception:
        return {}


def save_cache(cache_map):
    try:
        with open(CACHE_FILE, "w", encoding="utf-8") as f:
            json.dump(cache_map, f, ensure_ascii=False, indent=2)
    except Exception as e:
        print(f"⚠️ 缓存保存失败：{str(e)[:80]}")


def clear_cache_entry(wxid):
    cache = load_cache()
    if wxid in cache:
        del cache[wxid]
        save_cache(cache)
        print("🗑️  已清除失效缓存")


def get_cached_token(wxid):
    """
    从缓存获取有效 token。
    若 expiretime 存在且已过期，立即删除并返回 None。
    若 expiretime 为 0（无过期信息），直接返回 token（依靠业务接口判断失效）。
    """
    cache = load_cache()
    item = cache.get(wxid, {})
    token = str(item.get("token") or "").strip()
    if not token:
        return None

    expiretime = int(item.get("expiretime") or 0)
    if expiretime and expiretime <= now_ts() + TOKEN_EXPIRE_BUFFER:
        print("⚠️  缓存 token 已过期，准备重新换取")
        clear_cache_entry(wxid)
        return None

    return token


def write_token_cache(account, token, expiretime):
    cache = load_cache()
    cache[account["wxid"]] = {
        "remark": account["remark"],
        "wxid": account["wxid"],
        "token": token,
        "expiretime": int(expiretime or 0),
        "updated_at": now_ts(),
    }
    save_cache(cache)


# ========== 登录链路 ==========
def get_code(wxid):
    """wxid -> code"""
    if not WECHAT_SERVER:
        print("❌ 未配置 WECHAT_SERVER 环境变量")
        return None
    url = f"{WECHAT_SERVER}{WECHAT_CODE_URL_PATH}"
    try:
        resp = requests.post(
            url,
            json={"wxid": wxid, "appid": APPID},
            timeout=10,
            verify=False,
        )
        body = resp.json()
        code = (
            body.get("Data", {}).get("code")
            or body.get("data", {}).get("code")
            or body.get("code")
        )
        if not code:
            print(f"❌ 换取 code 失败，响应：{safe_json(body)}")
            return None
        return str(code)
    except Exception as e:
        print(f"❌ 换取 code 异常：{str(e)[:100]}")
        return None


def code_to_token(code):
    """code -> token（天机馆 autoLogin）"""
    try:
        resp = requests.post(
            AUTO_LOGIN_URL,
            headers={
                "host": "xcx.tianjiguan.cn",
                "content-type": "application/json",
                "accept": "application/json, text/plain, */*",
                "accept-encoding": "gzip,compress,br,deflate",
                "user-agent": IPHONE_UA,
                "referer": REFERER,
            },
            json={"code": code},
            timeout=REQUEST_TIMEOUT,
            verify=False,
        )
        body = resp.json()
        data = body.get("data", {}) if isinstance(body, dict) else {}
        if body.get("code") == 1 and data.get("token"):
            return {
                "token": str(data["token"]).strip(),
                "expiretime": int(data.get("expiretime") or 0),
            }
        print(f"❌ code 换 token 失败：{safe_json(body)}")
        return None
    except Exception as e:
        print(f"❌ code 换 token 异常：{str(e)[:100]}")
        return None


def fetch_new_token(account):
    """完整换取新 token 并写入缓存"""
    print(f"📌 {account['remark']} 正在换取微信 code...")
    code = get_code(account["wxid"])
    if not code:
        return None
    print(f"✅ code 获取成功：{code[:10]}...")
    token_info = code_to_token(code)
    if not token_info:
        return None
    write_token_cache(account, token_info["token"], token_info["expiretime"])
    print(f"✅ {account['remark']} 新 token 已缓存")
    return token_info["token"]


def get_valid_token(account):
    """获取可用 token：缓存优先，失效则立即重取"""
    token = get_cached_token(account["wxid"])
    if token:
        print(f"📦 {account['remark']} 命中缓存 token")
        return token
    return fetch_new_token(account)


# ========== 查询接口 ==========
def query_user_info(token):
    """查询用户信息"""
    try:
        resp = requests.get(
            USER_INFO_URL,
            headers={
                "host": "xcx.tianjiguan.cn",
                "x-access-token": token,
                "x-requested-with": "XMLHttpRequest",
                "user-agent": WIN_UA,
                "xweb_xhr": "1",
                "content-type": "application/x-www-form-urlencoded; charset=UTF-8",
                "accept": "*/*",
                "referer": REFERER,
                "accept-encoding": "gzip, deflate, br",
                "accept-language": "zh-CN,zh;q=0.9",
            },
            params={"token": token},
            timeout=REQUEST_TIMEOUT,
            verify=False,
        )
        if resp.status_code == 401:
            return {"token_invalid": True}
        body = resp.json()
        if body.get("code") == 1:
            return {"success": True, "data": body.get("data", {})}
        # 判断 token 失效
        msg = str(body.get("msg") or "").lower()
        if any(k in msg for k in ["未登录", "token", "登录失效", "无效", "expired"]):
            return {"token_invalid": True}
        return {"success": False, "msg": body.get("msg", "未知")}
    except Exception as e:
        return {"success": False, "msg": str(e)[:120]}


# ========== 单账号执行 ==========
def run_one(account, index):
    wxid_show = account["wxid"]
    if len(wxid_show) > 14:
        wxid_show = wxid_show[:12] + "..."

    print(f"\n{'─' * 44}")
    print(f"  账号 {index} │ {account['remark']}  ({wxid_show})")
    print(f"{'─' * 44}")

    # 获取有效 token
    token = get_valid_token(account)
    if not token:
        print(f"│ ❌ 获取 token 失败，跳过")
        return

    # 查询用户信息（token 失效则立即重取再查一次）
    result = query_user_info(token)
    if result.get("token_invalid"):
        print(f"│ 🔄 token 已失效，立即重新换取...")
        clear_cache_entry(account["wxid"])
        token = fetch_new_token(account)
        if not token:
            print(f"│ ❌ 重新获取 token 失败，跳过")
            return
        result = query_user_info(token)

    if not result.get("success"):
        print(f"│ ❌ 查询失败：{result.get('msg', '未知')}")
        return

    data = result.get("data", {})
    phone_raw = str(data.get("mobile") or "")
    phone_display = mask_phone(phone_raw) if phone_raw else "未绑定"

    print(f"│")
    print(f"│  👤 昵称：{data.get('nickname', '未知')}")
    print(f"│  📱 手机：{phone_display}")
    print(f"│  💰 积分：{data.get('score', 0)}")
    print(f"│  🏆 等级：{data.get('level', 0)}")
    print(f"│  🔄 兑换次数：{data.get('exchange_num', 0)}")
    print(f"│")
    print(f"  ✨ {account['remark']} 查询完成")


# ========== 主入口 ==========
if __name__ == "__main__":
    print("🚀 天机馆用户信息查询")
    print(f"⏱  {time.strftime('%Y-%m-%d %H:%M:%S')}")

    accounts = get_accounts()
    if not accounts:
        print("\n❌ 未检测到账号")
        print("   用法一（命令行）：python3 天机查询.py wxid_xxx")
        print("   用法二（命令行）：python3 天机查询.py 备注#wxid_xxx")
        print("   用法三（环境变量）：WXID_TJ=备注#wxid_xxx python3 天机查询.py")
        sys.exit(1)

    if not WECHAT_SERVER:
        print("\n❌ 未配置 WECHAT_SERVER 环境变量（取 code 服务地址）")
        sys.exit(1)

    print(f"\n共检测到 {len(accounts)} 个账号")

    for idx, acc in enumerate(accounts, start=1):
        run_one(acc, idx)

    print(f"\n{'═' * 44}")
    print("🎉  全部账号查询完成！")
    print(f"⏱  {time.strftime('%Y-%m-%d %H:%M:%S')}")
