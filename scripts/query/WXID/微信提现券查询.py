#!/usr/bin/env python3
"""
微信支付提现券详细查询脚本 - 纯查询版（无任何写操作）
接口来源：抓包 78004fe03d3eebf2fef25332cb3cfaa3.har

查询内容：
  1. 用户信息（昵称、openid 等）
  2. 提现免手续费余额（total / free_amount）
  3. 可用提现券列表（listcreditedcashoutfree?only_can_use=true）
  4. 全部提现券列表（listcreditedcashoutfree?only_can_use=false，含已用）
  5. 折扣卡列表（listuserdiscountcards）
  6. 每日礼品券状态（querydailygiftcoupons）
  7. 开放任务列表（listopentasks）

环境变量：
  WECHAT_SERVER     微信代理服务地址（由xdd后台自动传递）
  WECHAT_SERVER_NEW 新地址（由xdd后台自动传递，可选）
  WXID_WXLQ         账号列表，纯 wxid 或 备注#wxid，& 或换行分割
  WX_APPID          小程序 AppID（默认内置）
  DEBUG             设为 1 开启详细日志
"""
from __future__ import annotations

import base64
import gzip
import json
import os
import re
import secrets
import struct
import sys
import time
import zlib
from typing import Any

import requests

# ================== 全局配置 ==================
# 从环境变量获取地址（由xdd后台自动传递）
WECHAT_SERVER: str = os.getenv("WECHAT_SERVER", "http://180.152.5.230:8011").strip()
WECHAT_SERVER_NEW: str = os.getenv("WECHAT_SERVER_NEW", "").strip()
WX_APPID: str = (os.getenv("WX_APPID") or "wxdb3c0e388702f785").strip()

DOMAIN = "https://discount.wxpapp.wechatpay.cn"
MODULE_NAME = "mmpaytxbbsmp"
PAGE_FRAME_VERSION = "185"          # 抓包确认版本号
PAGE_MINE = "pages/mine/index"      # 我的页面（登录 / 用户信息 / 余额 / 券列表）
PAGE_GIFT = "pages/gift/index"      # 礼品页（任务列表 / 弹窗）

CODE_COUNT = 2          # 每账号最多尝试几个 code
API_TIMEOUT = 15        # 业务 API 超时（秒）
BRIDGE_TIMEOUT = 35     # 获取 code 超时（秒）

# iOS UA（与抓包一致）
USER_AGENT = (
    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) "
    "AppleWebKit/605.1.15 (KHTML, like Gecko) "
    "Mobile/15E148 MicroMessenger/8.0.59(0x18003b2e) "
    "NetType/WIFI Language/zh_CN"
)

DEBUG: bool = os.getenv("DEBUG", "").lower() in ("1", "true", "yes")


class QueryError(RuntimeError):
    pass


# ================== 账号解析 ==================
def parse_wxid_str(raw: str) -> list[dict[str, str]]:
    """解析账号字符串，支持：备注#wxid、纯wxid，换行 / & 分隔。"""
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
            accounts.append({"wxid": wxid, "wxname": remark})
    return accounts


def get_wx_list_from_env() -> list[dict[str, str]]:
    """
    获取账号列表：
    1. 优先命令行参数（支持多账号 & 分隔 / 备注#wxid）
    2. 其次环境变量 WXID_WXLQ
    """
    if len(sys.argv) > 1:
        accounts = parse_wxid_str(" ".join(sys.argv[1:]).strip())
        if accounts:
            return accounts

    raw = (os.getenv("WXID_WXLQ") or "").strip()
    if not raw:
        print("❌ 未配置环境变量 WXID_WXLQ，且未提供命令行参数")
        return []

    accounts = parse_wxid_str(raw)
    print(f"📋 从环境变量加载到 {len(accounts)} 个微信账号")
    return accounts


# ================== 服务器地址获取 ==================
def get_wxserver_urls():
    """
    获取微信协议服务器地址（新旧地址）
    返回: (old_url, new_url)
    """
    old_url = WECHAT_SERVER
    new_url = WECHAT_SERVER_NEW if WECHAT_SERVER_NEW else old_url
    return old_url, new_url


# ================== 获取 code ==================
def get_wx_code(wxid: str) -> str:
    old_url, new_url = get_wxserver_urls()
    urls_to_try = list(dict.fromkeys([old_url, new_url]))  # 去重保持顺序

    for idx, server_url in enumerate(urls_to_try):
        if not server_url:
            continue
        label = "旧地址" if idx == 0 else "新地址"
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
            # 业务失败（Code: -8 数据不存在 等），继续尝试下一个地址
            if data.get("Code") is not None or data.get("code") is not None:
                print(f"  ⚠️  {label} 业务错误: {data.get('Message') or data.get('msg', '')}")
                continue
        except Exception as e:
            print(f"  ⚠️  {label} 请求异常: {str(e)[:60]}")
            continue

    raise RuntimeError(f"所有地址均无法获取 code")


def get_login_codes(wxid: str, count: int = CODE_COUNT) -> list[str]:
    codes: list[str] = []
    for i in range(count):
        try:
            code = get_wx_code(wxid)
            codes.append(code)
            if DEBUG:
                print(f"  [debug] code[{i+1}] = {code[:8]}...")
        except Exception as err:
            print(f"  ⚠️ 获取第 {i+1} 个 code 失败：{err}")
    if not codes:
        raise QueryError(f"账号 {wxid} 未能获取到任何 code")
    return codes


# ================== 登录 ==================
def login_with_codes(session: requests.Session, codes: list[str], track_id: str) -> tuple[str, int]:
    errors: list[str] = []
    for index, code in enumerate(codes, start=1):
        try:
            data = api_get(
                session, "/txbbs-user/user/login",
                page=PAGE_MINE, track_id=track_id, jscode=code,
            )
            token = data.get("session_token")
            if not isinstance(token, str) or not token:
                raise QueryError(f"登录返回缺少 session_token：{data}")
            return token, index
        except Exception as err:
            errors.append(f"第{index}个code失败：{err}")
    raise QueryError("全部 code 登录失败：" + "；".join(errors))


# ================== 查询接口 ==================
def get_user_info(session: requests.Session, session_token: str, track_id: str) -> dict[str, Any]:
    """获取用户信息（昵称、头像等）。"""
    try:
        return api_get(
            session, "/txbbs-user/user/getuserinfo",
            page=PAGE_MINE, track_id=track_id, session_token=session_token,
        )
    except Exception as e:
        if DEBUG:
            print(f"  [debug] getuserinfo 失败：{e}")
        return {}


def get_balance(session: requests.Session, session_token: str, track_id: str) -> dict[str, Any]:
    """获取提现免手续费余额。"""
    try:
        return api_get(
            session, "/txbbs-mall/cashoutfree/getbalance",
            page=PAGE_MINE, track_id=track_id, session_token=session_token,
        )
    except Exception as e:
        if DEBUG:
            print(f"  [debug] getbalance 失败：{e}")
        return {}


def list_cashout_coupons(
    session: requests.Session,
    session_token: str,
    track_id: str,
    only_can_use: bool = True,
    page_size: int = 20,
) -> list[dict[str, Any]]:
    """
    查询提现免手续费券列表。
    only_can_use=True  → 仅可用；False → 全部（含已用/过期）
    """
    try:
        data = api_get(
            session,
            "/txbbs-mall/cashoutfree/listcreditedcashoutfree",
            page=PAGE_MINE,
            track_id=track_id,
            session_token=session_token,
            params={
                "page_token": "",
                "page_size": str(page_size),
                "only_can_use": "true" if only_can_use else "false",
                "coupon_type": "COUPON_TYPE_UNKNOW",
            },
        )
        items = data.get("coupon_items") or data.get("items") or []
        return [i for i in items if isinstance(i, dict)]
    except Exception as e:
        if DEBUG:
            print(f"  [debug] listcreditedcashoutfree 失败：{e}")
        return []


def list_discount_cards(
    session: requests.Session,
    session_token: str,
    track_id: str,
    page_size: int = 20,
) -> list[dict[str, Any]]:
    """查询折扣卡列表（可用状态）。"""
    try:
        data = api_get(
            session,
            "/txbbs-mall/coupon/listuserdiscountcards",
            page=PAGE_MINE,
            track_id=track_id,
            session_token=session_token,
            params={
                "consume_state_list": "USER_DISCOUNT_CARD_CONSUME_STATE_CAN_USE",
                "scene": "LIST_USER_DISCOUNT_CARDS_SCENE_HOME_PAGE",
                "page_size": str(page_size),
            },
        )
        return [i for i in (data.get("items") or []) if isinstance(i, dict)]
    except Exception as e:
        if DEBUG:
            print(f"  [debug] listuserdiscountcards 失败：{e}")
        return []


def query_daily_gift_coupons(
    session: requests.Session,
    session_token: str,
    track_id: str,
) -> list[dict[str, Any]]:
    """查询每日礼品券列表（领取状态）。"""
    try:
        data = api_get(
            session, "/txbbs-mall/coupon/querydailygiftcoupons",
            page=PAGE_GIFT, track_id=track_id, session_token=session_token,
        )
        return [i for i in (data.get("coupon_items") or []) if isinstance(i, dict)]
    except Exception as e:
        if DEBUG:
            print(f"  [debug] querydailygiftcoupons 失败：{e}")
        return []


def list_open_tasks(
    session: requests.Session,
    session_token: str,
    track_id: str,
) -> list[dict[str, Any]]:
    """查询开放任务列表（pages/gift/index 页面）。"""
    try:
        data = api_get(
            session, "/txbbs-task/finance/listopentasks",
            page=PAGE_GIFT, track_id=track_id, session_token=session_token,
        )
        tasks = data.get("task_items") or data.get("tasks") or data.get("items") or []
        return [t for t in tasks if isinstance(t, dict)]
    except Exception as e:
        if DEBUG:
            print(f"  [debug] listopentasks 失败：{e}")
        return []


# ================== 单账号主流程 ==================
def run_account(account: dict[str, str]) -> dict[str, Any]:
    wxid = account["wxid"]
    track_id = make_track_id()
    session = requests.Session()

    codes = get_login_codes(wxid, count=CODE_COUNT)
    session_token, _ = login_with_codes(session, codes, track_id)

    user_info      = get_user_info(session, session_token, track_id)
    balance        = get_balance(session, session_token, track_id)
    coupons_avail  = list_cashout_coupons(session, session_token, track_id, only_can_use=True)
    coupons_all    = list_cashout_coupons(session, session_token, track_id, only_can_use=False)
    discount_cards = list_discount_cards(session, session_token, track_id)
    daily_gifts    = query_daily_gift_coupons(session, session_token, track_id)
    open_tasks     = list_open_tasks(session, session_token, track_id)

    return {
        "user_info":      user_info,
        "balance":        balance,
        "coupons_avail":  coupons_avail,
        "coupons_all":    coupons_all,
        "discount_cards": discount_cards,
        "daily_gifts":    daily_gifts,
        "open_tasks":     open_tasks,
    }


# ================== 入口 ==================
def main() -> int:
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except AttributeError:
        pass

    accounts = get_wx_list_from_env()
    if not accounts:
        print("\n❌ 未检测到账号")
        print("   用法一：python 微信提现券查询.py wxid_xxx")
        print("   用法二：python 微信提现券查询.py 备注#wxid_xxx")
        print("   用法三：python 微信提现券查询.py wxid1&wxid2")
        print("   用法四：配置环境变量 WXID_WXLQ")
        return 1

    # 显示地址获取方式
    old_url, new_url = get_wxserver_urls()
    print(f"\n📡 协议服务器地址：旧={old_url} 新={new_url}")

    ok_count = 0
    total = len(accounts)
    for index, account in enumerate(accounts, start=1):
        wxid = account["wxid"]
        wxname = account["wxname"]
        prefix = f"🌸 账号[{index}/{total}]({wxname})"
        try:
            result = run_account(account)
            print_result(prefix, wxid, result)
            ok_count += 1
        except Exception as err:
            print(f"{prefix} ❌ 查询失败（{mask_id(wxid)}）：{err}")

        if index < total:
            print(f"\n⏳ 等待 10 秒后查询下一个账号...\n")
            time.sleep(10)

    return 0 if ok_count == total else 1


# ================== 结果打印 ==================
SEP = "─" * 52


def print_result(prefix: str, wxid: str, result: dict[str, Any]) -> None:
    print(f"\n{SEP}")
#    print(f"{prefix} ✅ 登录成功（{mask_id(wxid)}）")

    # ---- 用户信息 ----
    user_info = result.get("user_info") or {}
    if user_info:
        nickname = user_info.get("nickname") or user_info.get("nick_name") or ""
        openid   = user_info.get("openid") or ""
        if nickname:
            print(f"  👤 昵称：{nickname}")
        if openid:
            print(f"  🆔 openid：{mask_id(openid)}")

    # ---- 余额（含总额度统计）----
    coupons_avail = result.get("coupons_avail") or []
    _print_balance(result.get("balance") or {}, coupons_avail)
    print(f"\n  💳 可用提现免手续费券（{len(coupons_avail)} 张）：")
    if coupons_avail:
        for c in coupons_avail:
            _print_cashout_coupon(c)
    else:
        print("     暂无可用提现券")

    # ---- 历史提现券（仅已用/过期） ----
    coupons_all = result.get("coupons_all") or []
    used_coupons = [c for c in coupons_all if not _cashout_can_use(c)]
    if used_coupons:
        print(f"\n  🗂  历史提现券（已使用/已过期，{len(used_coupons)} 张）：")
        for c in used_coupons:
            _print_cashout_coupon(c, show_state=True)

    # ---- 折扣卡 ----
    discount_cards = result.get("discount_cards") or []
    if discount_cards:
        print(f"\n  🎫 折扣卡（{len(discount_cards)} 张）：")
        for card in discount_cards:
            _print_discount_card(card)
    else:
        print(f"\n  🎫 折扣卡：无")

    # ---- 每日礼品券 ----
    daily_gifts = result.get("daily_gifts") or []
    if daily_gifts:
        print(f"\n  🎁 今日礼品券（{len(daily_gifts)} 项）：")
        for gift in daily_gifts:
            _print_daily_gift(gift)

    # ---- 开放任务 ----
    open_tasks = result.get("open_tasks") or []
    if open_tasks:
        print(f"\n  📋 开放任务（{len(open_tasks)} 项）：")
        for task in open_tasks:
            _print_task(task)
    else:
        print(f"\n  📋 开放任务：无")

    print(f"{SEP}")


# ---- 余额打印 ----
def _print_balance(balance: dict[str, Any], coupons: list[dict[str, Any]] | None = None) -> None:
    if not balance:
        return
    # 兼容两种字段名
    balance_str = balance.get("balance")       # 新接口（deflate JSON）: "56000"
    free_amount = balance.get("free_amount")   # 老接口字段
    total_str   = balance.get("total_amount")  # 老接口字段
    pending     = balance.get("pending_balance")
    used_str    = balance.get("total_deduct_face_value")
    frozen      = balance.get("frozen_amount")
    expiry      = balance.get("expire_time") or balance.get("expire_at")

    parts: list[str] = []

    # 总额度（新接口：balance 字段，"56000" / 分）
    if balance_str and isinstance(balance_str, str):
        fen = int(balance_str)
        parts.append(f"总额 {_fen2yuan(fen)}")

    # 可用（老接口）
    if isinstance(free_amount, int):
        parts.append(f"可用 {_fen2yuan(free_amount)}")

    # 待生效
    if pending and isinstance(pending, str) and pending != "0":
        parts.append(f"待生效 {_fen2yuan(int(pending))}")

    # 已用面额
    if used_str and isinstance(used_str, str) and used_str != "0":
        parts.append(f"已用 {used_str}元")

    # 冻结
    if isinstance(frozen, int) and frozen:
        parts.append(f"冻结 {_fen2yuan(frozen)}")

    # 到期
    if expiry:
        parts.append(f"到期 {expiry}")

    # 券总额（从可用券列表累加）
    if coupons:
        total_quota = _calc_total_quota(coupons)
        if total_quota:
            parts.append(f"券总额 {_fen2yuan(total_quota)}")

    if parts:
        print(f"\n  💰 提现余额：{'  |  '.join(parts)}")


# ---- 提现券打印 ----
def _print_cashout_coupon(coupon: dict[str, Any], show_state: bool = False) -> None:
    info        = coupon.get("coupon_info") or {}
    name        = info.get("name") or coupon.get("name") or "提现免手续费券"
    face_val    = info.get("face_value") or coupon.get("face_value")
    min_amt     = info.get("minimum_amount") or coupon.get("minimum_amount")
    coupon_id   = info.get("coupon_id") or coupon.get("coupon_id") or "—"
    end_time    = info.get("end_time") or coupon.get("end_time") or coupon.get("expire_time") or ""
    state_raw   = coupon.get("consume_state") or coupon.get("state") or ""

    amount_str = _fen2yuan(face_val) if isinstance(face_val, int) else "—"
    min_str    = f"  满{_fen2yuan(min_amt)}可用" if isinstance(min_amt, int) and min_amt else ""
    expire_str = f"  有效期至 {end_time}" if end_time else ""
    id_str     = f"  ID:{coupon_id}"

    state_map = {
        "COUPON_CONSUME_STATE_USABLE":  "✅ 可用",
        "COUPON_CONSUME_STATE_USED":    "⬛ 已使用",
        "COUPON_CONSUME_STATE_EXPIRED": "🔴 已过期",
        "COUPON_CONSUME_STATE_FREEZE":  "🔒 冻结",
    }
    state_str = f"  [{state_map.get(state_raw, state_raw)}]" if show_state and state_raw else ""

    print(f"     • {name}  {amount_str}{min_str}{expire_str}{id_str}{state_str}")


def _calc_total_quota(coupons: list[dict[str, Any]]) -> int | None:
    """从可用提现券列表累加面值，返回总额（分）。"""
    total = 0
    for c in coupons:
        info = c.get("coupon_info") or {}
        fv = info.get("face_value") or c.get("face_value")
        if isinstance(fv, int):
            total += fv
    return total if total else None


def _cashout_can_use(coupon: dict[str, Any]) -> bool:
    state = coupon.get("consume_state") or coupon.get("state") or ""
    return state == "COUPON_CONSUME_STATE_USABLE" or not state


# ---- 折扣卡打印 ----
def _print_discount_card(card: dict[str, Any]) -> None:
    info         = card.get("card_info") or {}
    name         = info.get("name") or card.get("name") or "折扣卡"
    discount     = info.get("discount_rate") or card.get("discount_rate")
    end_time     = info.get("end_time") or card.get("end_time") or card.get("expire_time") or ""
    merchant     = info.get("merchant_name") or card.get("merchant_name") or ""

    discount_str = f"  {discount}折" if isinstance(discount, (int, float)) else ""
    expire_str   = f"  有效期至 {end_time}" if end_time else ""
    merchant_str = f"  [{merchant}]" if merchant else ""

    print(f"     • {name}{discount_str}{expire_str}{merchant_str}")


# ---- 每日礼品券打印 ----
GIFT_TYPE_MAP = {
    "DGCT_PLATFORM": "提现免手续费券",
    "DGCT_MCH":      "商家券",
}


def _print_daily_gift(gift: dict[str, Any]) -> None:
    info        = gift.get("coupon_info") or {}
    name        = info.get("name") or gift.get("name") or "礼品券"
    gift_type   = gift.get("daily_gift_type") or ""
    is_claimed  = gift.get("is_claimed", False)
    face_val    = info.get("face_value") or gift.get("face_value")
    end_time    = info.get("end_time") or gift.get("expire_time") or ""

    type_str   = f"（{GIFT_TYPE_MAP.get(gift_type, gift_type)}）" if gift_type else ""
    amount_str = f"  {_fen2yuan(face_val)}" if isinstance(face_val, int) else ""
    expire_str = f"  有效期至 {end_time}" if end_time else ""
    status     = "✅ 已领" if is_claimed else "⬜ 未领"

    print(f"     {status}  {name}{type_str}{amount_str}{expire_str}")


# ---- 任务打印 ----
TASK_STATE_MAP = {
    "TASK_STATE_IN_PROGRESS": "进行中",
    "TASK_STATE_COMPLETED":   "已完成",
    "TASK_STATE_NOT_START":   "未开始",
    "TASK_STATE_REWARDED":    "已领奖",
}


def _print_task(task: dict[str, Any]) -> None:
    name       = task.get("task_name") or task.get("name") or "未命名任务"
    reward     = task.get("reward_amount") or task.get("reward")
    progress   = task.get("current_progress") or task.get("progress")
    target     = task.get("target_progress") or task.get("target")
    state      = task.get("task_state") or task.get("state") or ""
    desc       = task.get("task_desc") or task.get("desc") or ""

    state_str  = TASK_STATE_MAP.get(state, state)
    reward_str = f"  奖励 {_fen2yuan(reward)}" if isinstance(reward, int) else ""
    prog_str   = f"  进度 {progress}/{target}" if progress is not None and target else ""
    desc_str   = f"  {desc}" if desc else ""

    print(f"     • [{state_str}] {name}{reward_str}{prog_str}{desc_str}")


# ================== HTTP 工具 ==================
def api_get(
    session: requests.Session,
    path: str,
    *,
    page: str,
    track_id: str,
    jscode: str | None = None,
    session_token: str | None = None,
    params: dict[str, str] | None = None,
) -> dict[str, Any]:
    headers = make_headers(page, track_id, jscode=jscode, session_token=session_token)
    response = session.get(
        f"{DOMAIN}{path}", headers=headers, params=params, timeout=API_TIMEOUT
    )
    return unwrap_response(response, path)


def unwrap_response(response: requests.Response, action: str) -> dict[str, Any]:
    if DEBUG:
        print(f"  [debug] {action} → {response.status_code} {response.text[:300]}")
    try:
        response.raise_for_status()
        payload = response.json()
        if not isinstance(payload, dict):
            raise QueryError(f"{action} 返回格式异常：{payload!r}")
        if payload.get("errcode") != 0:
            raise QueryError(
                f"{action} 返回失败：errcode={payload.get('errcode')}，{payload}"
            )
        data = payload.get("data")
        return data if isinstance(data, dict) else {}
    except Exception:
        # JSON 解析失败 → 尝试 deflate / raw deflate 解压后解析 JSON
        raw = response.content
        for dec in _try_deflate(raw):
            try:
                text = dec.decode("utf-8")
                payload = json.loads(text)
                if isinstance(payload, dict) and not payload.get("errcode", 0):
                    data = payload.get("data")
                    return data if isinstance(data, dict) else {}
            except Exception:
                pass
        # 最后尝试 protobuf
        return decode_protobuf(action, raw)


def _try_deflate(raw: bytes):
    """尝试多种解压方式，yeild 解压后的 bytes。"""
    import gzip, zlib
    # 1. zlib raw deflate（Content-Encoding: deflate）
    try:
        yield zlib.decompress(raw, -zlib.MAX_WBITS)
    except Exception:
        pass
    # 2. zlib with zlib header
    try:
        yield zlib.decompress(raw, zlib.MAX_WBITS)
    except Exception:
        pass
    # 3. raw bytes（无压缩）
    yield raw


def decode_protobuf(action: str, data: bytes) -> dict[str, Any]:
    """手动解析 protobuf，提取关键字段。"""
    result: dict[str, Any] = {}
    i = 0
    while i < len(data):
        tag = data[i]
        i += 1
        field_num = tag >> 3
        wire_type = tag & 0x07

        if wire_type == 0:  # Varint
            val = 0
            shift = 0
            while i < len(data):
                b = data[i]
                i += 1
                val |= (b & 0x7F) << shift
                if not (b & 0x80):
                    break
                shift += 7
            _put_field(result, field_num, val)

        elif wire_type == 2:  # Length-delimited (string/bytes/embedded)
            length = 0
            shift = 0
            j = i
            while j < len(data):
                b = data[j]
                j += 1
                length |= (b & 0x7F) << shift
                if not (b & 0x80):
                    break
                shift += 7
            start = j
            i = start + length
            val = bytes(data[start:i])
            _put_field(result, field_num, val)

        elif wire_type == 5:  # Fixed32
            if i + 4 <= len(data):
                val = struct.unpack("<I", data[i:i+4])[0]
                i += 4
                _put_field(result, field_num, val)

        else:
            break

    # 转换为可读结构
    return _flatten_protobuf(result, action)


def _put_field(result: dict, field_num: int, val: Any) -> None:
    """将字段写入 result（支持 repeated）。"""
    key = str(field_num)
    if key in result:
        existing = result[key]
        if isinstance(existing, list):
            existing.append(val)
        else:
            result[key] = [existing, val]
    else:
        result[key] = val


def _flatten_protobuf(result: dict[str, Any], action: str) -> dict[str, Any]:
    """将数字字段名的 protobuf 转为语义化结构。"""
    if not result:
        return {}

    # ---------- getbalance ----------
    if "getbalance" in action:
        # 新接口字段（从 deflate JSON）：balance="56000"（分）
        b_str = result.get("1", b"") or result.get("balance", b"")
        p_str = result.get("2", b"") or result.get("pending_balance", b"")
        u_str = result.get("3", b"") or result.get("total_deduct_face_value", b"")
        if isinstance(b_str, bytes): b_str = _b2s(b_str)
        if isinstance(p_str, bytes): p_str = _b2s(p_str)
        if isinstance(u_str, bytes): u_str = _b2s(u_str)
        try:
            balance_fen = int(b_str)
            return {"balance": str(balance_fen), "pending_balance": str(p_str or "0"), "total_deduct_face_value": str(u_str or "0")}
        except Exception:
            return {"balance": str(result.get("1", 0))}

    # ---------- listcreditedcashoutfree ----------
    if "listcreditedcashoutfree" in action:
        items_raw = result.get("1", [])
        if not isinstance(items_raw, list):
            items_raw = [items_raw]
        coupons = []
        for item in items_raw:
            if not isinstance(item, dict):
                continue
            coupons.append(_flatten_coupon_item(item))
        return {"coupon_items": [c for c in coupons if c]}

    # ---------- getuserinfo ----------
    if "getuserinfo" in action:
        # 新接口（deflate JSON）：user_info_po 嵌套
        ui = result.get("1", {})
        if not isinstance(ui, dict):
            ui = {"nick_name": _b2s(result.get("3", b"")),
                  "openid":    _b2s(result.get("4", b"")),
                  "avatar_url": _b2s(result.get("5", b""))}
        return {
            "nickname":   _b2s(ui.get("nick_name", ui.get("2", b""))),
            "openid":     _b2s(ui.get("openid", b"")),
            "avatar_url": _b2s(ui.get("head_url", ui.get("1", b""))),
        }

    # ---------- listopentasks ----------
    if "listopentasks" in action:
        tasks_raw = result.get("1", [])
        if not isinstance(tasks_raw, list):
            tasks_raw = [tasks_raw]
        tasks = []
        for t in tasks_raw:
            if not isinstance(t, dict):
                continue
            tasks.append({
                "task_name":   _b2s(t.get("2", b"")),
                "task_state":  _state_name(t.get("4", 0)),
                "reward_amount": t.get("5", 0) if isinstance(t.get("5"), int) else 0,
                "task_desc":   _b2s(t.get("8", b"")),
            })
        return {"task_items": [tt for tt in tasks if tt.get("task_name")]}

    # ---------- querydailygiftcoupons ----------
    if "querydailygiftcoupons" in action:
        gifts_raw = result.get("1", [])
        if not isinstance(gifts_raw, list):
            gifts_raw = [gifts_raw]
        gifts = []
        for g in gifts_raw:
            if not isinstance(g, dict):
                continue
            info = g.get("2", {})
            gifts.append({
                "coupon_info": {
                    "name":        _b2s(info.get("2", b"")) if isinstance(info, dict) else "",
                    "face_value":  info.get("3", 0) if isinstance(info, dict) and isinstance(info.get("3"), int) else 0,
                    "end_time":    _b2s(info.get("4", b"")) if isinstance(info, dict) else "",
                },
                "is_claimed":   g.get("4", 0) == 1,
                "daily_gift_type": _gift_type_name(g.get("5", 0)),
            })
        return {"coupon_items": gifts}

    return result


def _flatten_coupon_item(item: dict[str, Any]) -> dict[str, Any]:
    """解析 coupon item 结构（protobuf nested）。"""
    info = {}
    info_raw = item.get("2", {})
    if isinstance(info_raw, dict):
        info = {
            "coupon_id":     _b2s(info_raw.get("1", b"")),
            "name":          _b2s(info_raw.get("2", b"")),
            "face_value":    info_raw.get("3", 0) if isinstance(info_raw.get("3"), int) else 0,
            "minimum_amount": info_raw.get("4", 0) if isinstance(info_raw.get("4"), int) else 0,
            "end_time":      _b2s(info_raw.get("5", b"")),
        }
    return {
        "coupon_info":   info,
        "consume_state": _state_name(item.get("3", 0)),
    }


def _b2s(val: Any) -> str:
    if isinstance(val, bytes):
        try:
            return val.decode("utf-8")
        except Exception:
            return val.decode("latin-1", errors="replace")
    if isinstance(val, int):
        return str(val)
    return str(val) if val else ""


def _state_name(v: Any) -> str:
    mapping = {
        1: "COUPON_CONSUME_STATE_USABLE",
        2: "COUPON_CONSUME_STATE_USED",
        3: "COUPON_CONSUME_STATE_EXPIRED",
        4: "COUPON_CONSUME_STATE_FREEZE",
    }
    return mapping.get(int(v) if isinstance(v, int) else 0, "") or str(v)


def _gift_type_name(v: Any) -> str:
    mapping = {
        1: "DGCT_PLATFORM",
        2: "DGCT_MCH",
    }
    return mapping.get(int(v) if isinstance(v, int) else 0, "") or str(v)


# ================== Headers ==================
def make_headers(
    page: str,
    track_id: str,
    *,
    jscode: str | None = None,
    session_token: str | None = None,
) -> dict[str, str]:
    h: dict[str, str] = {
        "User-Agent":      USER_AGENT,
        "content-type":    "application/json",
        "X-Page":          page,
        "X-Track-Id":      track_id,
        "X-Module-Name":   MODULE_NAME,
        "X-Appid":         WX_APPID,
        "Accept-Encoding": "gzip,compress,br,deflate",
        "Referer": (
            f"https://servicewechat.com/{WX_APPID}/{PAGE_FRAME_VERSION}/page-frame.html"
        ),
        "Accept-Language": "zh-CN,zh;q=0.9",
    }
    if jscode:
        h["jscode"] = jscode
    if session_token:
        h["session-token"] = session_token
    return h


def make_track_id() -> str:
    return "T" + "".join(secrets.choice("0123456789ABCDEF") for _ in range(31))


# ================== 工具函数 ==================
def _fen2yuan(fen: Any) -> str:
    """分转元。"""
    if not isinstance(fen, int):
        return "—"
    return f"{fen // 100}元" if fen % 100 == 0 else f"{fen / 100:.2f}元"


def mask_id(s: str) -> str:
    return s if len(s) <= 12 else f"{s[:6]}...{s[-4:]}"


if __name__ == "__main__":
    raise SystemExit(main())
