# -*- coding: utf-8 -*-
"""
旧衣小二 微信小程序 账号查询脚本（仅查询，不签到）

用法:
  python jyxe_query.py wxid
  python jyxe_query.py 备注#wxid
  python jyxe_query.py wxid1&wxid2
  python jyxe_query.py 大师#wxid1&小号#wxid2

环境变量兜底: jyxe=备注#wxid&备注#wxid
依赖: WECHAT_SERVER（获取微信 code 的桥接服务，由xdd后台自动传递）
      WECHAT_SERVER_NEW（新地址，由xdd后台自动传递，可选）
缓存: 脚本同目录 jyxe_token_cache.json（与 jyxe.py 共用）
"""

import os
import sys
import re
import json
import requests

# ================== 配置 ==================
WX_APPID      = "wx426d52c8130b8559"
# 从环境变量获取地址（由xdd后台自动传递）
WECHAT_SERVER = os.getenv("WECHAT_SERVER", "http://180.152.5.230:8011").strip()
WECHAT_SERVER_NEW = os.getenv("WECHAT_SERVER_NEW", "").strip()
BASE_URL      = "https://jiuyixiaoer.fzjingzhou.com"
SCRIPT_DIR    = os.path.dirname(os.path.abspath(__file__))
TOKEN_FILE    = os.path.join(SCRIPT_DIR, "jyxe_token_cache.json")

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
    "Referer": f"https://servicewechat.com/{WX_APPID}/5/page-frame.html",
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
    # 1. 命令行优先
    if len(sys.argv) > 1:
        raw = "&".join(sys.argv[1:])
        accounts = parse_accounts(raw)
        if accounts:
            return accounts
    # 2. 环境变量兜底
    return parse_accounts(os.getenv("jyxe", ""))

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

# ================== 服务器地址获取 ==================
def get_wxserver_urls():
    """
    获取微信协议服务器地址（新旧地址）
    返回: (old_url, new_url)
    """
    old_url = WECHAT_SERVER
    new_url = WECHAT_SERVER_NEW if WECHAT_SERVER_NEW else old_url
    return old_url, new_url


# ================== 登录链路 ==================
def get_wx_code(wxid):
    old_url, new_url = get_wxserver_urls()
    urls_to_try = list(dict.fromkeys([old_url, new_url]))  # 去重保持顺序

    for server_url in urls_to_try:
        if not server_url:
            continue
        url = f"{server_url.rstrip('/')}/api/v1/wx/app/get/code"
        try:
            resp = requests.post(url, json={"wxid": wxid, "appid": WX_APPID}, timeout=BRIDGE_TIMEOUT)
            resp.raise_for_status()
            data = resp.json()
            code = (
                (data.get("data") or {}).get("code")
                or (data.get("Data") or {}).get("code")
                or data.get("Data")
            )
            if code:
                return str(code)
            # 如果是明确的业务失败，不继续尝试
            if data.get("Code") is not None or data.get("code") is not None:
                raise Exception(f"获取 code 失败: {data}")
        except Exception as e:
            if "获取 code 失败" in str(e):
                raise
            print(f"⚠️  地址 {server_url} 请求异常: {str(e)[:60]}")
            continue

    raise Exception(f"所有地址均无法获取 code")

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
    """POST /api/Person/index  返回用户信息"""
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

# ================== 单账号 ==================
def run_one(acc):
    try:
        token, person = get_token(acc["wxid"])
        return {
            "remark": acc["remark"],
            "nick":   person.get("nickname", "-"),
            "phone":  mask_phone(person.get("mobile", "")),
            "score":  str(person.get("score", "-")),
            "days":   str(person.get("days", "-")),
            "vip":    person.get("vipName", "-"),
        }
    except Exception as e:
        return {
            "remark": acc["remark"],
            "nick":   "-",
            "phone":  "-",
            "score":  "-",
            "days":   "-",
            "vip":    str(e),
        }

# ================== 主入口 ==================
if __name__ == "__main__":
    accounts = get_accounts()
    if not accounts:
        print("未检测到账号")
        print("用法: python jyxe_query.py 备注#wxid&备注#wxid")
        sys.exit(1)

    for i, acc in enumerate(accounts):
        r = run_one(acc)
        wxid_show = acc["wxid"]
        if len(wxid_show) > 14:
            wxid_show = wxid_show[:12] + "..."

        print(f"  【{r['remark']}】")
        print(f"{'─' * 42}")
        print(f"  👤 昵称：{r['nick']}")
        print(f"  📱 手机：{r['phone']}")
        print(f"  💰 积分：{r['score']}")
        print(f"  📅 连签：{r['days']}天")
        print(f"  🏆 等级：{r['vip']}")
   