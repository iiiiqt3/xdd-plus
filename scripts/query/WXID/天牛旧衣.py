# -*- coding: utf-8 -*-
"""
天牛旧衣服回收 微信小程序 账号查询脚本（仅查询，不签到）

用法:
  python tnjy_query.py wxid
  python tnjy_query.py 备注#wxid
  python tnjy_query.py wxid1&wxid2
  python tnjy_query.py 大师#wxid1&小号#wxid2

环境变量兜底: tnjy=备注#wxid&备注#wxid
依赖: WECHAT_SERVER（获取微信 code 的桥接服务）
缓存: 脚本同目录 tnjy_token_cache.json（与 tnjy.py 共用）
"""

import os
import sys
import re
import json
import requests

# ================== 配置 ==================
WX_APPID      = "wx887c2f947bffa76e"
WECHAT_SERVER = os.getenv("WECHAT_SERVER", "http://180.152.5.230:8011",).strip()
BASE_URL      = "https://tianniunew.fzjingzhou.com"
SCRIPT_DIR    = os.path.dirname(os.path.abspath(__file__))
TOKEN_FILE    = os.path.join(SCRIPT_DIR, "tnjy_token_cache.json")

LOGIN_TOKEN   = "wek2020123456788wek"

BRIDGE_TIMEOUT = 20
REQ_TIMEOUT    = 15

COMMON_HEADERS = {
    "platform":     "MP-WEIXIN",
    "content-type": "application/x-www-form-urlencoded",
    "User-Agent": (
        "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) "
        "AppleWebKit/605.1.15 (KHTML, like Gecko) "
        "Mobile/15E148 MicroMessenger/8.0.60(0x18003c32) "
        "NetType/WIFI Language/zh_CN"
    ),
    "Referer": f"https://servicewechat.com/{WX_APPID}/6/page-frame.html",
    "Accept-Encoding": "gzip,compress,br,deflate",
}

# ================== 工具函数 ==================
def mask_phone(phone):
    if not phone:
        return "-"
    s = str(phone)
    if len(s) < 7:
        return s
    return f"{s[:3]}****{s[-4:]}"

# ================== 账号解析 ==================
def parse_accounts(raw):
    accounts = []
    if not raw:
        return accounts
    for x in re.split(r"[\n&]+", raw):
        x = x.strip()
        if not x:
            continue
        if "#" in x:
            remark, wxid = x.split("#", 1)
            accounts.append({"remark": remark.strip(), "wxid": wxid.strip()})
        else:
            accounts.append({"remark": x, "wxid": x})
    return accounts

def get_accounts():
    if len(sys.argv) > 1:
        raw = "&".join(sys.argv[1:])
        accounts = parse_accounts(raw)
        if accounts:
            return accounts
    return parse_accounts(os.getenv("tnjy", ""))

# ================== 缓存管理 ==================
def load_cache():
    try:
        with open(TOKEN_FILE, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return {}

def save_cache(data):
    try:
        with open(TOKEN_FILE, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
    except Exception as e:
        print(f"缓存写入失败: {e}")

TOKEN_CACHE = load_cache()

# ================== 登录链路 ==================
def get_wx_code(wxid):
    if not WECHAT_SERVER:
        raise Exception("未配置 WECHAT_SERVER")
    url = f"{WECHAT_SERVER.rstrip('/')}/api/v1/wx/app/get/code"
    resp = requests.post(url, json={"wxid": wxid, "appid": WX_APPID}, timeout=BRIDGE_TIMEOUT)
    resp.raise_for_status()
    data = resp.json()
    code = (
        (data.get("data") or {}).get("code")
        or (data.get("Data") or {}).get("code")
        or data.get("Data")
    )
    if not code:
        raise Exception(f"获取 code 失败: {data}")
    return str(code)

def wx_login(code):
    url = f"{BASE_URL}/api/login/getWxMiniProgramSessionKey"
    payload = {"code": code, "gdtVid": "", "token": LOGIN_TOKEN}
    resp = requests.post(url, headers=COMMON_HEADERS, data=payload, timeout=REQ_TIMEOUT)
    resp.raise_for_status()
    data = resp.json()
    if str(data.get("code")) != "1000":
        raise Exception(f"登录失败: {data}")
    return data["data"]["token"], data["data"].get("personInfo", {})

def query_user(token):
    resp = requests.post(
        f"{BASE_URL}/api/Person/index",
        headers=COMMON_HEADERS,
        data={"token": token},
        timeout=REQ_TIMEOUT,
    )
    resp.raise_for_status()
    data = resp.json()
    if str(data.get("code")) != "1000":
        raise Exception("token 无效")
    return data.get("data", {})

def get_token(wxid):
    cache = TOKEN_CACHE.get(wxid, {})
    token = cache.get("token")

    if token:
        try:
            person = query_user(token)
            TOKEN_CACHE[wxid]["personInfo"] = person
            save_cache(TOKEN_CACHE)
            return token, person
        except Exception:
            TOKEN_CACHE.pop(wxid, None)
            save_cache(TOKEN_CACHE)

    code = get_wx_code(wxid)
    token, person_info = wx_login(code)
    TOKEN_CACHE[wxid] = {"token": token, "personInfo": person_info}
    save_cache(TOKEN_CACHE)
    return token, person_info

# ================== 主入口 ==================
if __name__ == "__main__":
    accounts = get_accounts()
    if not accounts:
        print("未检测到账号")
        print("用法: python tnjy_query.py 备注#wxid&备注#wxid")
        sys.exit(1)

    for acc in accounts:
        try:
            token, p = get_token(acc["wxid"])
            nick    = p.get("nickname", "-")
            phone   = mask_phone(p.get("mobile", ""))
            score   = p.get("score", 0)
            days    = p.get("days", "-")
            cash    = p.get("exchange", 0)
            prop    = p.get("proportion", "-")
        except Exception as e:
            nick  = "-"
            phone = "-"
            score = "-"
            days  = "-"
            cash  = "-"
            prop  = str(e)

        print(f"\n{'─' * 42}")
        print(f"  【{acc['remark']}】")
        print(f"{'─' * 42}")
        print(f"  👤 昵称：{nick}")
        print(f"  📱 手机：{phone}")
        print(f"  💰 积分：{score}")
        print(f"  💵 现金：{cash}元")
        print(f"  📅 连签：{days}天")
        print(f"  🏷️  加成：{prop}")
        print(f"{'─' * 42}")