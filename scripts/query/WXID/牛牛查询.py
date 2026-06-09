#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
牛牛短剧 - 查询本
支持多账号混搭模式（WX协议 + 手动Token）
功能：查询用户金币、余额、邀请信息
作者：foglamb

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 账号传入方式（优先级从高到低）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 方式1 - 命令行 wxid（WX协议模式）：
   python 牛牛查询.py wxid_xxx
   python 牛牛查询.py 备注#wxid_xxx
   python 牛牛查询.py 张三#wxid_xxx&李四#wxid_yyy
   python 牛牛查询.py wxid_xxx&wxid_yyy

 方式2 - 命令行 Token（手动Token模式）：
   python 牛牛查询.py eyJ0eX...（JWT token字符串，以eyJ开头）
   python 牛牛查询.py token1&token2

 方式3 - 命令行从文件读取：
   python 牛牛查询.py @wxid.txt   （文件每行一个 wxid 或 备注#wxid）
   python 牛牛查询.py @token.txt  （文件每行一个 token）

 方式4 - 环境变量 wxid（WX协议模式）：
   WXID_niuniu="备注#wxid_xxx&wxid_yyy"

 方式5 - 环境变量 Token（手动Token模式）：
   NIUNIU_TOKENS="token1&token2"

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 其他环境变量
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  WECHAT_SERVER  - WX协议服务地址（可选，优先从后台获取）
  XDD_API_URL    - 后台API地址，用于获取微信协议服务器地址
  NIUNIU_APPID   - 小程序AppID（默认 wxcb95401f250e9a53）

 缓存文件：niuniu_cache.json（与牛牛短剧.py共用，脚本同目录）
"""

import requests
import json
import time
import os
import sys
from datetime import datetime
from typing import Optional, Dict, List
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

# ============================================================================
# 基础配置
# ============================================================================

# 保留环境变量兼容，但优先从后台获取
WX_API_BASE    = os.getenv("WECHAT_SERVER", "").strip().rstrip("/")
XDD_API_URL    = os.getenv("XDD_API_URL", "").strip().rstrip("/")
APPID          = os.getenv("NIUNIU_APPID", "wxcb95401f250e9a53")
BUSINESS_API_BASE = "https://api.tianjinzhitongdaohe.com/sqx_fast"
CACHE_FILE     = os.path.join(os.path.dirname(os.path.abspath(__file__)), "niuniu_cache.json")

REQUEST_TIMEOUT = 30
MAX_RETRIES     = 3
RETRY_DELAY     = 5

# ============================================================================
# 工具函数（提前定义，供参数解析使用）
# ============================================================================

# 缓存获取到的服务器地址
_cached_server_urls = None

def get_wxserver_urls():
    """
    从后台API获取微信协议服务器地址（新旧地址）
    优先级：环境变量 WECHAT_SERVER > 后台API > 默认值
    返回: (old_url, new_url)
    """
    global _cached_server_urls
    if _cached_server_urls:
        return _cached_server_urls

    if WX_API_BASE:
        _cached_server_urls = (WX_API_BASE, WX_API_BASE)
        return _cached_server_urls

    if XDD_API_URL:
        try:
            resp = requests.get(f"{XDD_API_URL}/api/wxserver", timeout=5, verify=False)
            if resp.status_code == 200:
                data = resp.json()
                if data.get("code") == 0:
                    old_url = data.get("data", {}).get("old_url", "").strip().rstrip("/")
                    new_url = data.get("data", {}).get("new_url", "").strip().rstrip("/")
                    if old_url and new_url:
                        _cached_server_urls = (old_url, new_url)
                        return _cached_server_urls
        except Exception as e:
            print(f"⚠️  从后台获取地址失败: {str(e)[:60]}")

    default_url = "http://180.152.5.230:8011"
    _cached_server_urls = (default_url, default_url)
    return _cached_server_urls

def parse_wxid_list(raw: str) -> list:
    """
    解析 wxid 列表，支持两种格式混用，用 & 或换行分隔：
      备注#wxid_xxx  →  remark="备注", wxid="wxid_xxx"
      wxid_xxx       →  remark="wxid_xxx", wxid="wxid_xxx"
    """
    result = []
    if not raw:
        return result
    for item in raw.replace("\n", "&").split("&"):
        item = item.strip()
        if not item:
            continue
        if "#" in item:
            remark, wxid = item.split("#", 1)
            remark = remark.strip()
            wxid   = wxid.strip()
        else:
            wxid   = item
            remark = item
        if wxid:
            result.append({"wxid": wxid, "remark": remark or wxid})
    return result

def is_token(s: str) -> bool:
    """判断字符串是否为 JWT Token（以 eyJ 开头）"""
    return s.strip().startswith("eyJ")

def read_file_lines(filepath: str) -> str:
    """读取文件内容，返回用 & 连接的字符串"""
    try:
        with open(filepath, "r", encoding="utf-8") as f:
            lines = [l.strip() for l in f.readlines() if l.strip()]
        return "&".join(lines)
    except Exception as e:
        print(f"[❌] 读取文件失败: {filepath} -> {e}")
        sys.exit(1)

# ============================================================================
# 命令行参数 + 环境变量解析
# 优先级：命令行参数 > 环境变量
# ============================================================================

WXID_LIST:     List[dict] = []
MANUAL_TOKENS: List[str]  = []

def _parse_input_source():
    """解析账号来源，填充 WXID_LIST 和 MANUAL_TOKENS"""
    global WXID_LIST, MANUAL_TOKENS

    # ── 1. 命令行参数 ──────────────────────────────────────────
    argv = sys.argv[1:]
    if argv:
        raw = " ".join(argv).strip()

        # 从文件读取：@文件路径
        if raw.startswith("@"):
            filepath = raw[1:].strip()
            raw = read_file_lines(filepath)

        # 判断是 wxid 模式还是 token 模式
        # 规则：如果第一段以 eyJ 开头 → token 模式；否则 → wxid 模式
        first_item = raw.replace("&", "\n").split("\n")[0].strip()
        if is_token(first_item):
            # Token 模式：& 分隔
            for t in raw.replace("\n", "&").split("&"):
                t = t.strip()
                if t:
                    MANUAL_TOKENS.append(t)
        else:
            # wxid 模式：支持 备注#wxid 和纯 wxid
            WXID_LIST = parse_wxid_list(raw)
        return

    # ── 2. 环境变量 ────────────────────────────────────────────
    env_wxid   = os.getenv("WXID_niuniu", "").strip()
    env_tokens = os.getenv("NIUNIU_TOKENS", "").strip()

    if env_wxid:
        WXID_LIST = parse_wxid_list(env_wxid)

    if env_tokens:
        for t in env_tokens.replace("\n", "&").split("&"):
            t = t.strip()
            if t:
                MANUAL_TOKENS.append(t)

_parse_input_source()

# ── 无账号时打印帮助并退出 ──────────────────────────────────────
if not WXID_LIST and not MANUAL_TOKENS:
    print("=" * 60)
    print("❌ 错误: 未传入任何账号信息")
    print("=" * 60)
    print(__doc__)
    sys.exit(1)

# ============================================================================
# HTTP 工具
# ============================================================================

def create_session() -> requests.Session:
    session = requests.Session()
    retry = Retry(
        total=MAX_RETRIES,
        backoff_factor=1,
        status_forcelist=[429, 500, 502, 503, 504],
        allowed_methods=["GET", "POST"]
    )
    adapter = HTTPAdapter(max_retries=retry)
    session.mount("http://", adapter)
    session.mount("https://", adapter)
    return session

def safe_request(method: str, url: str, **kwargs) -> Optional[requests.Response]:
    session = create_session()
    for attempt in range(MAX_RETRIES + 1):
        try:
            resp = session.request(method.upper(), url, timeout=REQUEST_TIMEOUT, **kwargs)
            return resp
        except (requests.exceptions.ConnectionError, requests.exceptions.Timeout):
            if attempt < MAX_RETRIES:
                time.sleep(RETRY_DELAY)
        except Exception:
            break
    return None

def mask_str(s: str, keep_start: int = 3, keep_end: int = 4) -> str:
    if not s or len(s) <= keep_start + keep_end:
        return s
    return s[:keep_start] + "****" + s[-keep_end:]

# ============================================================================
# 缓存管理（与牛牛短剧.py 共用同一缓存文件）
# ============================================================================

def load_cache() -> Dict:
    if os.path.exists(CACHE_FILE):
        try:
            with open(CACHE_FILE, "r", encoding="utf-8") as f:
                return json.load(f)
        except Exception:
            pass
    return {"tokens": {}}

def save_cache(cache: Dict):
    try:
        with open(CACHE_FILE, "w", encoding="utf-8") as f:
            json.dump(cache, f, ensure_ascii=False, indent=2)
    except Exception as e:
        print(f"  ⚠️  保存缓存失败: {e}")

def get_cached_token(cache: Dict, wxid: str) -> Optional[str]:
    return cache.get("tokens", {}).get(wxid, {}).get("token")

def save_token_to_cache(cache: Dict, wxid: str, token: str):
    if "tokens" not in cache:
        cache["tokens"] = {}
    cache["tokens"][wxid] = {
        "token": token,
        "update_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    }
    save_cache(cache)

# ============================================================================
# WX 协议：wxid → code → openId → token
# ============================================================================

def get_wechat_code(wxid: str) -> Optional[str]:
    old_url, new_url = get_wxserver_urls()
    urls_to_try = list(dict.fromkeys([old_url, new_url]))  # 去重保持顺序

    for server_url in urls_to_try:
        if not server_url:
            continue
        url = f"{server_url}/api/v1/wx/app/get/code"
        try:
            resp = safe_request("POST", url, json={"appid": APPID, "wxid": wxid})
            if not resp:
                continue
            data = resp.json()
            if data.get("Code") != 0:
                # 如果是明确的业务失败，不继续尝试
                if data.get("Code") is not None:
                    print(f"  ❌ 获取code失败: {data.get('Message', '未知错误')}")
                    return None
                continue
            code = data.get("Data", {}).get("code")
            if code:
                print(f"  ✅ 获取code成功")
                return code
        except Exception as e:
            print(f"  ⚠️  地址 {server_url} 请求异常: {str(e)[:60]}")
            continue

    print(f"  ❌ 所有地址均无法获取code")
    return None

def wxlogin(code: str) -> Optional[dict]:
    url = f"{BUSINESS_API_BASE}/app/Login/wxLogin"
    resp = safe_request("GET", url, params={"code": code})
    if not resp:
        return None
    data = resp.json()
    if data.get("code") != 0:
        print(f"  ❌ wxLogin失败: {data.get('msg', '')}")
        return None
    return data.get("data")

def insert_wx_user(open_id: str) -> Optional[str]:
    url = f"{BUSINESS_API_BASE}/app/Login/insertWxUser"
    payload = {
        "openId": open_id,
        "userName": "游客",
        "avatar": "https://nnduanju.oss-cn-beijing.aliyuncs.com/01image/re-512.png",
        "sex": 1,
        "phone": "",
        "inviterCode": "",
        "qdCode": ""
    }
    headers = {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        "Content-Type": "application/json",
        "xweb_xhr": "1",
        "referer": f"https://servicewechat.com/{APPID}/19/page-frame.html"
    }
    resp = safe_request("POST", url, json=payload, headers=headers)
    if not resp:
        return None
    data = resp.json()
    if data.get("code") == 0:
        token = data.get("data", {}).get("token") or data.get("token")
        if token:
            print(f"  ✅ 获取token成功")
            return token
    print(f"  ❌ 获取token失败: {data.get('msg', '')}")
    return None

def get_token_by_wxid(wxid: str) -> Optional[str]:
    """wxid → code → openId → token 完整流程"""
    print(f"  🔄 通过WX协议获取token...")
    code = get_wechat_code(wxid)
    if not code:
        return None
    time.sleep(1)
    login_data = wxlogin(code)
    if not login_data:
        return None
    open_id = login_data.get("open_id")
    if not open_id:
        print(f"  ❌ 未获取到open_id")
        return None
    return insert_wx_user(open_id)

# ============================================================================
# 业务接口
# ============================================================================

def make_headers(token: str) -> dict:
    return {
        "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.59(0x18003b2e) NetType/4G Language/zh_CN",
        "content-type": "application/x-www-form-urlencoded",
        "xweb_xhr": "1",
        "token": token,
        "referer": f"https://servicewechat.com/{APPID}/19/page-frame.html",
        "accept-language": "zh-CN,zh;q=0.9"
    }

def check_token_valid(data: dict) -> bool:
    if not data:
        return False
    code = data.get("code")
    msg  = str(data.get("msg", "")).lower()
    if code == 401:
        return False
    if "token" in msg and ("无效" in msg or "过期" in msg):
        return False
    if "登录" in msg and ("过期" in msg or "失效" in msg):
        return False
    return True

def query_integral(token: str) -> Optional[int]:
    url  = f"{BUSINESS_API_BASE}/app/integral/selectByUserId"
    resp = safe_request("GET", url, headers=make_headers(token))
    if not resp:
        return None
    data = resp.json()
    if not check_token_valid(data):
        return None
    if data.get("code") == 0:
        return data.get("data", {}).get("integralNum", 0)
    return None

def query_user_info(token: str) -> Optional[Dict]:
    url  = f"{BUSINESS_API_BASE}/app/user/getUserInfo"
    resp = safe_request("GET", url, headers=make_headers(token))
    if not resp:
        return None
    data = resp.json()
    if not check_token_valid(data):
        return None
    if data.get("code") == 0:
        return data.get("data", {})
    return None

def query_invite_money(token: str) -> Optional[Dict]:
    """查询邀请金额和余额信息（抓包接口 /app/invite/selectInviteMoney）"""
    url  = f"{BUSINESS_API_BASE}/app/invite/selectInviteMoney"
    resp = safe_request("GET", url, headers=make_headers(token))
    if not resp:
        return None
    data = resp.json()
    if not check_token_valid(data):
        return None
    if data.get("code") == 0:
        return data.get("data", {})
    return None

# ============================================================================
# 查询并展示账号信息
# ============================================================================

def query_and_display(token: str, remark: str) -> bool:
    """用 token 查询全部信息并输出，成功返回 True，token 失效返回 False"""
    integral = query_integral(token)
    if integral is None:
        return False  # token 失效

    user_info  = query_user_info(token)   or {}
    invite_data = query_invite_money(token) or {}

    money_info   = invite_data.get("inviteMoney", {})
    invite_count = invite_data.get("inviteCount", 0)

    nick_name   = user_info.get("nickName", "未知")
    phone       = user_info.get("phone", "")
    user_id     = user_info.get("userId", "")
    vip_status  = user_info.get("vipStatus", 0)
    vip_expire  = user_info.get("vipExpireTime", "")

    gold_sum    = money_info.get("goldSum", integral)
    money_sum   = money_info.get("moneySum", 0)
    money_avail = money_info.get("money", 0)
    cash_out    = money_info.get("cashOut", 0)
    sync_time   = money_info.get("syncTime", "")

    print(f"  👤 昵称：{nick_name}  {'📱 手机：' + mask_str(phone) if phone else ''}")
    if user_id:
        print(f"  🆔 用户ID：{user_id}")
    if vip_status:
        print(f"  👑 VIP状态：已开通  到期：{vip_expire or '未知'}")
    print(f"  💰 当前金币：{integral:,} 枚")
    print(f"  🪙 总金币（累计）：{gold_sum:,} 枚")
    print(f"  💵 总收益：¥{money_sum:.2f}  可提现：¥{money_avail:.2f}  已提现：¥{cash_out:.2f}")
    print(f"  👥 已邀请人数：{invite_count} 人")
    if sync_time:
        print(f"  🕐 数据同步时间：{sync_time}")
    return True

# ============================================================================
# 处理单个账号
# ============================================================================

def process_wxid_account(wxid: str, remark: str, cache: Dict) -> bool:
    """WX协议账号：优先缓存，失效时重新换取"""
    print(f"\n{'='*55}")
    print(f"👤 [{remark}]  (WX协议)")
    print(f"{'='*55}")

    # 1. 优先读缓存 token
    token = get_cached_token(cache, wxid)
    if token:
        print(f"  📦 使用缓存Token")
        if query_and_display(token, remark):
            return True
        print(f"  🔄 缓存Token已失效，重新换取...")

    # 2. 重新换取
    token = get_token_by_wxid(wxid)
    if not token:
        print(f"  ❌ 无法获取Token，跳过")
        return False

    # 3. 保存并查询
    save_token_to_cache(cache, wxid, token)
    if not query_and_display(token, remark):
        print(f"  ❌ 新Token查询失败")
        return False
    return True

def process_manual_account(token: str, index: int) -> bool:
    """手动Token账号"""
    print(f"\n{'='*55}")
    print(f"👤 [手动账号 {index}]  (手动Token)")
    print(f"{'='*55}")
    if not query_and_display(token, f"手动账号{index}"):
        print(f"  ❌ Token已失效或无效")
        return False
    return True

# ============================================================================
# 主函数
# ============================================================================

def main():
    start_time = datetime.now()
#    print(f"\n{'='*55}")
    print(f"🎬 牛牛短剧 - 查询本")
    print(f"⏰ {start_time.strftime('%Y-%m-%d %H:%M:%S')}")
  #  print(f"{'='*55}")
 #   print(f"📁 缓存文件: {CACHE_FILE}")
  #  print(f"📱 WX协议账号: {len(WXID_LIST)} 个  |  手动Token: {len(MANUAL_TOKENS)} 个  |  合计: {len(WXID_LIST) + len(MANUAL_TOKENS)} 个")

    cache   = load_cache()
    success = 0
    fail    = 0

    # WX 协议账号
    for item in WXID_LIST:
        try:
            ok = process_wxid_account(item["wxid"], item["remark"], cache)
            success += ok
            fail    += not ok
        except Exception as e:
            fail += 1
            print(f"  ❌ 处理异常: {e}")
        time.sleep(1)

    # 手动 Token 账号
    for i, token in enumerate(MANUAL_TOKENS, 1):
        try:
            ok = process_manual_account(token, i)
            success += ok
            fail    += not ok
        except Exception as e:
            fail += 1
            print(f"  ❌ 处理异常: {e}")
        if i < len(MANUAL_TOKENS):
            time.sleep(1)

    elapsed = (datetime.now() - start_time).total_seconds()
    print(f"\n{'='*55}")
 #   print(f"📊 查询完成：✅ 成功 {success} 个  ❌ 失败 {fail} 个")
  #  print(f"⏱️  耗时 {elapsed:.2f} 秒")
 #   print(f"{'='*55}\n")

if __name__ == "__main__":
    main()
