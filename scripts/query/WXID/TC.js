/**
 * 同程旅行 - 活动进度查询脚本（仅查询，不执行任何操作）
 *
 * 用法：
 *   node TC.js wxid1
 *   node TC.js "备注1#wxid1" "备注2#wxid2"
 *
 * 环境变量（由xdd后台自动传递）：
 *   WECHAT_SERVER     - 微信 code 服务地址
 *   WECHAT_SERVER_NEW - 新地址（可选）
 *   TC_APPID          - 小程序 appid，默认 wx336dcaf6a1ecf632
 */

"use strict";

const https = require("https");
const http  = require("http");
const zlib  = require("zlib");

// ============================================================
// 配置区
// ============================================================
// 从环境变量获取地址（由xdd后台自动传递）
const WECHAT_SERVER = (process.env.WECHAT_SERVER || "http://180.152.5.230:8011").trim();
const WECHAT_SERVER_NEW = (process.env.WECHAT_SERVER_NEW || "").trim();
const DEFAULT_APPID = process.env.TC_APPID || "wx336dcaf6a1ecf632";
const BASE_URL      = "https://wx.17u.cn";
const LOGIN_URL     = BASE_URL + "/wechatappapi/wxUser/login";

const SIGN_URL               = BASE_URL + "/wxmpsign/sign";
const HOME_URL               = BASE_URL + "/wxmpsign/home";
const TASK_URL               = BASE_URL + "/qiushiinnerapi/task";
const SHARE_URL              = BASE_URL + "/wxmpsign/share/mileage";
const GET_USER_PARTICIPATION = SHARE_URL + "/getUserParticipationInfo";
const FLOWER_BASE            = BASE_URL + "/platformflowpool/flowerGod";
const FLOWER_HOME            = FLOWER_BASE + "/home";

const CARD_TYPE_MAP = {
  0: "花神签(稀有)",
  1: "牡丹签",
  2: "梨花签",
  3: "荷花签",
  4: "桃花签",
  5: "梅花签",
};
const ALL_CARD_TYPES = [0, 1, 2, 3, 4, 5];

// ============================================================
// 工具
// ============================================================
function pad2(n) { return n < 10 ? "0" + n : "" + n; }
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/** 获取微信协议服务器地址（仅新地址） */
function getWxServerUrl() {
  return (WECHAT_SERVER_NEW || WECHAT_SERVER).replace(/\/+$/, "");
}

function generateApmat(openid) {
  openid = openid || "o498X0ScTl0oO6aj0taIYuXYvM9I";
  const now  = new Date();
  const ts   = "" + now.getFullYear() + pad2(now.getMonth() + 1) + pad2(now.getDate())
             + pad2(now.getHours()) + pad2(now.getMinutes());
  const rand = Math.floor(Math.random() * 900000) + 100000;
  return openid + "|" + ts + "|" + rand;
}

function getTodayRange() {
  const now = new Date();
  const d   = `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
  return { startDate: d + " 00:00:00", endDate: d + " 23:59:59" };
}

// ============================================================
// 颜色（仅 TTY 生效，非 TTY 全部返回原字符串）
// ============================================================
const isTTY = process.stdout.isTTY === true;

const CLR = {
  reset:   "\x1b[0m",
  bold:    "\x1b[1m",
  dim:     "\x1b[2m",
  red:     "\x1b[31m",
  green:   "\x1b[32m",
  yellow:  "\x1b[33m",
  blue:    "\x1b[34m",
  magenta: "\x1b[35m",
  cyan:    "\x1b[36m",
  white:   "\x1b[37m",
  gray:    "\x1b[90m",
};

/** 安全着色：非 TTY 时原样返回，绝不带任何 ANSI 码 */
function c(code, str) {
  if (!isTTY) return str;
  return code + str + CLR.reset;
}

// 分隔线（全用 ASCII，避免多字节乱码）
const LINE_BIG  = "=".repeat(54);
const LINE_THIN = "-".repeat(54);

// ============================================================
// 日志（过程阶段只输出登录/进度，查询细节在汇总中展示）
// ============================================================
const plog  = (...a) => console.log(c(CLR.cyan,   "[同程]"), ...a);
const pok   = (...a) => console.log(c(CLR.green,  "  [OK]"), ...a);
const pwarn = (...a) => console.log(c(CLR.yellow, "  [!!]"), ...a);
const perr  = (...a) => console.log(c(CLR.red,    "  [XX]"), ...a);
const pinfo = (...a) => console.log(c(CLR.gray,   "  [..]"), ...a);

// ============================================================
// HTTP
// ============================================================
function httpRequest(url, method, headers, body) {
  return new Promise((resolve, reject) => {
    const urlObj  = new URL(url);
    const isHttps = urlObj.protocol === "https:";
    const lib     = isHttps ? https : http;
    const port    = urlObj.port || (isHttps ? 443 : 80);
    const req     = lib.request(
      { hostname: urlObj.hostname, port, path: urlObj.pathname + urlObj.search, method, headers },
      (res) => {
        const enc = (res.headers["content-encoding"] || "").toLowerCase();
        let stream = res;
        if (enc === "gzip")    stream = res.pipe(zlib.createGunzip());
        else if (enc === "deflate") stream = res.pipe(zlib.createInflate());
        else if (enc === "br") stream = res.pipe(zlib.createBrotliDecompress());
        const chunks = [];
        stream.on("data", (ch) => chunks.push(Buffer.isBuffer(ch) ? ch : Buffer.from(ch)));
        stream.on("end",  () => {
          const data = Buffer.concat(chunks).toString("utf8");
          try { resolve(JSON.parse(data)); } catch { resolve({ raw: data }); }
        });
        stream.on("error", reject);
      }
    );
    req.on("error", reject);
    if (body && (method === "POST" || method === "PUT"))
      req.write(typeof body === "string" ? body : JSON.stringify(body));
    req.end();
  });
}

function requestWithLen(method, url, headers, body) {
  return new Promise((resolve, reject) => {
    const urlObj   = new URL(url);
    const bodyStr  = body !== undefined
      ? (typeof body === "string" ? body : JSON.stringify(body))
      : null;
    const fh = { ...headers };
    if (bodyStr && method === "POST")
      fh["Content-Length"] = Buffer.byteLength(bodyStr).toString();
    const req = https.request(
      { hostname: urlObj.hostname, port: 443, path: urlObj.pathname + urlObj.search, method, headers: fh },
      (res) => {
        const enc = (res.headers["content-encoding"] || "").toLowerCase();
        let stream = res;
        if (enc === "gzip")    stream = res.pipe(zlib.createGunzip());
        else if (enc === "deflate") stream = res.pipe(zlib.createInflate());
        else if (enc === "br") stream = res.pipe(zlib.createBrotliDecompress());
        const chunks = [];
        stream.on("data", (ch) => chunks.push(Buffer.isBuffer(ch) ? ch : Buffer.from(ch)));
        stream.on("end",  () => {
          const data = Buffer.concat(chunks).toString("utf8");
          try { resolve(JSON.parse(data)); } catch { resolve({ raw: data }); }
        });
        stream.on("error", reject);
      }
    );
    req.on("error", reject);
    if (bodyStr && method === "POST") req.write(bodyStr);
    req.end();
  });
}

function rawPost(path, payload, extraHeaders) {
  return new Promise((resolve, reject) => {
    const bodyStr = JSON.stringify(payload);
    const base = {
      Host: "wx.17u.cn",
      "Content-Type": "application/json;charset=UTF-8",
      "Content-Length": Buffer.byteLength(bodyStr).toString(),
      Accept: "application/json, text/plain, */*",
      "Accept-Language": "zh-CN,zh;q=0.9",
      "Accept-Encoding": "gzip, deflate, br",
      Connection: "keep-alive",
    };
    const fh = Object.assign(base, extraHeaders || {});
    const req = https.request(
      { hostname: "wx.17u.cn", path, method: "POST", headers: fh },
      (res) => {
        let data = "";
        res.on("data", (ch) => (data += ch));
        res.on("end",  () => {
          try { resolve({ status: res.statusCode, data: JSON.parse(data) }); }
          catch { resolve({ status: res.statusCode, data, raw: true }); }
        });
      }
    );
    req.on("error", reject);
    req.write(bodyStr);
    req.end();
  });
}

// ============================================================
// 账号解析
// ============================================================
function parseAccounts() {
  const args = process.argv.slice(2);
  const raw  = args.join("&");
  if (!raw) return [];
  const lines = raw.split("&").map((s) => s.trim()).join("\n")
                   .split("\n").map((s) => s.trim()).filter(Boolean);
  return lines.map((line, i) => {
    const p = line.split("#");
    let remark, wxid;
    if (p.length >= 2) { remark = p[0]?.trim() || `账号${i + 1}`; wxid = p[1]?.trim(); }
    else               { wxid = p[0]?.trim(); remark = `账号${i + 1}`; }
    if (!wxid) return null;
    return { wxid, appid: process.env.TC_APPID || DEFAULT_APPID, remark, index: i + 1 };
  }).filter(Boolean);
}

// ============================================================
// 登录
// ============================================================
/**
 * 查询设备在线状态
 */
async function queryDeviceStatus(wxid) {
  const url = getWxServerUrl();
  if (!url) return [];
  const results = [];
  try {
    const data = await httpRequest(url + "/api/v1/wx/user/status", "GET", {});
    const info = data?.data?.[wxid];
    if (info) results.push({ url, online: info.survival === 1, nickname: info.nickname || '' });
  } catch (_) {}
  return results;
}

async function getWxCode(wxid, appid) {
  const serverUrl = getWxServerUrl();
  if (!serverUrl) throw new Error("未配置微信协议服务器地址");
  try {
    const res = await httpRequest(
      serverUrl + "/api/v1/wx/app/get/code", "POST",
      { "Content-Type": "application/json" }, { appid, wxid }
    );
    if (res.Code === 0 || res.code === 0) return res.Data?.code || res.data?.code || res.Data;
    if (res.Success || res.status) return res.Data?.code || res.data?.code || res.Data;
    throw new Error(res.Message || res.msg || "获取 code 失败");
  } catch (e) {
    throw new Error(e.message || "获取 code 失败");
  }
}

async function tcLogin(wxCode) {
  const res     = await httpRequest(LOGIN_URL, "POST",
    { "Content-Type": "application/json" }, { code: wxCode, scene: "1001" }
  );
  const content = res.data?.content || res.content || {};
  if (!content.sectoken) throw new Error(res.msg || "登录成功但未返回 sectoken");
  return {
    sectoken:      content.sectoken,
    tcsectk:       content.sectoken,
    mallUserToken: content.sectoken,
    openid:        content.openId,
    unionid:       content.unionId,
    expts:         content.expts,
  };
}

// ============================================================
// Headers
// ============================================================
function buildHeaders(acct) {
  return {
    "Content-Type": "application/json",
    "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.60(0x18003c32) NetType/WIFI Language/zh_CN",
    tcsectk: acct.tcsectk,
    TCSecTk: acct.tcsectk,
    "tc-mall-platform-code": "WX_MP",
    "tc-mall-user-token":    acct.mallUserToken,
    secToken:     acct.mallUserToken,
    platform:     "WX_MP",
    osType:       "1",
    apmat:        generateApmat(acct.openid),
    TCxcxVersion: "7.8.9",
    TCReferer:    "page%2Fhome%2Fmall%2Fmall",
    TCPrivacy:    "1",
    Referer:      "https://servicewechat.com/wx336dcaf6a1ecf632/885/page-frame.html",
    Accept:             "*/*",
    "Accept-Encoding":  "gzip, compress, br, deflate",
    "Accept-Language":  "zh-CN,zh;q=0.9",
  };
}

function buildFlowerHeaders(acct) {
  return {
    "Content-Type": "application/json;charset=UTF-8",
    Accept: "application/json, text/plain, */*",
    "Accept-Language": "zh-CN,zh-Hans;q=0.9",
    "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.60(0x18003c32) NetType/WIFI Language/zh_CN miniProgram/wx336dcaf6a1ecf632",
    Referer:  "https://wx.17u.cn/wxweb/",
    Origin:   "https://wx.17u.cn",
    platform: "WX_MP",
    osType:   "1",
    accountSystem:      "1",
    "TC-PLATFORM-CODE": "WX_MP",
    "TC-OS-TYPE":       "1",
    "TC-USER-TOKEN":    acct.tcsectk,
    secToken:           acct.tcsectk,
  };
}

function buildH5Headers(tokenData, cookieObj) {
  const h = {
    "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.60(0x18003c32) NetType/4G Language/zh_CN miniProgram/wx336dcaf6a1ecf632",
    Referer:   "https://wx.17u.cn/memberft/student/studentcard/signIn?refid=2000845210",
    Origin:    "https://wx.17u.cn",
    platform:  "WX_MP",
    "TC-PLATFORM-CODE": "WX_MP",
    "TC-USER-TOKEN":    tokenData.sectoken,
    userToken:          tokenData.sectoken,
    userTokenMode:      "1",
    accountSystem:      "1",
    "TC-OS-TYPE": "1",
    Accept:             "application/json, text/plain, */*",
    "Accept-Language":  "zh-CN,zh-Hans;q=0.9",
    "Accept-Encoding":  "gzip, deflate, br",
    Connection:         "keep-alive",
  };
  if (tokenData.openid)  h["openId"]  = tokenData.openid;
  if (tokenData.unionid) h["userKey"] = tokenData.unionid;
  if (cookieObj && Object.keys(cookieObj).length > 0)
    h["Cookie"] = Object.entries(cookieObj).map(([k, v]) => k + "=" + v).join("; ");
  return h;
}

async function initCookie(sectoken) {
  const signInUrl = encodeURIComponent("https://wx.17u.cn/memberft/student/studentcard/signIn?refid=2000845210");
  const url = `https://wx.17u.cn/flight/getwxxcxopenid.html?sectoken=${encodeURIComponent(sectoken)}&url=${signInUrl}&wxAppScene=1089`;
  return new Promise((resolve) => {
    https.get(url, {
      headers: {
        "User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.60(0x18003c32) NetType/4G Language/zh_CN miniProgram/wx336dcaf6a1ecf632",
        Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        "Accept-Language": "zh-CN,zh-Hans;q=0.9",
        "Accept-Encoding": "identity",
        Connection: "keep-alive",
      },
    }, (res) => {
      const cookies = {};
      for (const sc of res.headers["set-cookie"] || []) {
        const part  = sc.split(";")[0].trim();
        const eqIdx = part.indexOf("=");
        if (eqIdx > 0) cookies[part.substring(0, eqIdx)] = part.substring(eqIdx + 1);
      }
      res.resume();
      res.on("end", () => resolve(cookies));
    }).on("error", () => resolve({}));
  });
}

// ============================================================
// 查询各模块（只读）
// ============================================================
async function queryBasicInfo(h) {
  const r = await requestWithLen("POST", HOME_URL + "/top", h, {});
  if (r?.code !== 200) return null;
  const d = r.data || {};
  return { showCoin: d.showCoin || 0, monthExpireCoin: d.monthExpireCoin || 0, nickname: d.nickname || "" };
}

async function querySignStatus(h) {
  const r = await requestWithLen("POST", SIGN_URL + "/getSignInfo", h, {});
  if (r?.code !== 200) return null;
  const d = r.data || {};
  return {
    todaySigned:             d.todaySigned             || false,
    periodContinuedSignDays: d.periodContinuedSignDays || 0,
    totalSignDays:           d.totalSignDays            || 0,
    canSign:                 d.canSign                  || false,
  };
}

async function queryTaskList(h) {
  const r = await requestWithLen("POST", TASK_URL + "/detailList", h, {
    detailGuid: "", pageNum: 1, pageSize: 999, schemeGuid: "task-2025-nflygijg",
  });
  if (r?.code !== 0) return null;
  return (r.data?.taskDetails || []).map((t) => ({
    name:   t.title     || "未知任务",
    status: t.status,
    prize:  t.prizeTitle || "0",
  }));
}

async function queryShareActivity(h) {
  const { startDate, endDate } = getTodayRange();
  const r = await requestWithLen("POST", GET_USER_PARTICIPATION, h, { startDate, limitAmount: 1, endDate });
  if (r?.code !== 200) return { active: false };
  const today = (r.data || [])[0] || null;
  return {
    active:    true,
    joined:    !!today,
    checkedIn: today ? (today.finishStatus || -1) === 1 : false,
  };
}

async function queryFlowerStatus(acct) {
  const fh = buildFlowerHeaders(acct);
  const r  = await requestWithLen("GET", FLOWER_HOME, fh, undefined);
  if (r?.code !== 0) return { active: false, msg: r?.msg || "活动未开启" };
  const d = r.data || {};
  return {
    active:          true,
    remainDrawCount: d.remainDrawCount ?? 0,
    totalDrawCount:  d.totalDrawCount  ?? 0,
    cards: (d.cardList || []).map((c) => ({
      type: c.cardType, name: CARD_TYPE_MAP[c.cardType] || `卡片${c.cardType}`, count: c.cardCount || 0,
    })),
  };
}

async function queryCashSign(tokenData, cookieObj) {
  const h5 = {
    "TC-MALL-PLATFORM-CODE": "WX_MP",
    accountSystem: "1",
    platform: "WX_H5",
    "TC-MALL-USER-TOKEN": tokenData.sectoken,
    userToken: "",
    ...buildH5Headers(tokenData, cookieObj),
  };

  let actInfo = null;
  try {
    const r = await rawPost("/platformflowpool/signTask/getActInfo", {}, h5);
    if (!r.raw && r.data?.code === 0) actInfo = r.data.data;
  } catch { /* ignore */ }
  if (!actInfo) return { active: false };

  const cashActId = actInfo.cashAwardAct?.actId      || "sign:2026:0758";
  const chalActId = actInfo.challengeAwardAct?.actId || "sign:2026:8366";

  let cashInfo = null, chalInfo = null, rewardRecords = null;
  try {
    const r = await rawPost("/platformflowpool/signTask/getTaskInfo", { actId: cashActId }, h5);
    if (!r.raw && r.data?.code === 0) cashInfo = r.data.data;
  } catch { /* ignore */ }
  try {
    const r = await rawPost("/platformflowpool/signTask/getTaskInfo", { actId: chalActId }, h5);
    if (!r.raw && r.data?.code === 0) chalInfo = r.data.data;
  } catch { /* ignore */ }
  try {
    const r = await rawPost("/platformflowpool/signTask/getRewardRecord", { pageNum: 1, pageSize: 5 }, h5);
    if (!r.raw && r.data?.code === 0) rewardRecords = r.data.data;
  } catch { /* ignore */ }

  return { active: true, cashActId, chalActId, cashInfo, chalInfo, rewardRecords };
}

async function queryCashClockin(tokenData, cookieObj) {
  const h = {
    "TC-MALL-PLATFORM-CODE": "WX_MP",
    accountSystem: "1",
    platform: "WX_H5",
    "TC-MALL-USER-TOKEN": tokenData.sectoken,
    userToken: "",
    ...buildH5Headers(tokenData, cookieObj),
  };
  try {
    const r = await rawPost("/wxmpsign/clockin/getClockinInfo", { channel: "923f1661d1e24aa399f0e4e6ba0b0fba" }, h);
    if (r.data?.code === 0) return r.data.data;
    return null;
  } catch { return null; }
}

// ============================================================
// 汇总面板生成（纯文本行，颜色仅通过 c() 在 TTY 时附加）
// ============================================================
function buildSummary(remark, r) {
  const lines = [];

  // 标题
  lines.push(LINE_BIG);
  lines.push("  " + c(CLR.bold + CLR.cyan, "【同程旅行】") + c(CLR.bold + CLR.white, remark));
  lines.push(LINE_THIN);

  // 1. 里程
  if (r.basic) {
    const expire = r.basic.monthExpireCoin > 0
      ? "  " + c(CLR.yellow, "(本月到期 " + r.basic.monthExpireCoin + " 里程，请及时使用！)")
      : "";
    lines.push("  " + c(CLR.bold, "💰 里程余额：") + c(CLR.bold + CLR.green, String(r.basic.showCoin)) + " 里程" + expire);
  } else {
    lines.push("  💰 里程余额：" + c(CLR.yellow, "获取失败"));
  }

  lines.push("");

  // 2. 签到
  if (r.sign) {
    const s    = r.sign;
    const icon = s.todaySigned ? "✅" : "⚠️";
    const stat = s.todaySigned
      ? c(CLR.green, "今日已签到") + "  连续 " + c(CLR.bold, String(s.periodContinuedSignDays)) + " 天  累计 " + s.totalSignDays + " 天"
      : (s.canSign ? c(CLR.yellow, "今日未签到（可签）") : c(CLR.yellow, "今日未签到"));
    lines.push("  " + icon + " " + c(CLR.bold, "每日签到：") + stat);
  } else {
    lines.push("  📝 每日签到：" + c(CLR.yellow, "获取失败"));
  }

  // 3. 每日任务
  if (r.tasks) {
    const tasks    = r.tasks;
    const total    = tasks.length;
    const claimed  = tasks.filter((t) => t.status === 3).length;
    const canClaim = tasks.filter((t) => t.status === 2).length;
    const ongoing  = tasks.filter((t) => t.status === 1).length;
    const allDone  = claimed === total;

    const taskStat = allDone
      ? c(CLR.green, "已领 " + claimed + "/" + total + "  全部完成！")
      : "已领 " + c(CLR.bold, claimed + "/" + total)
        + (canClaim > 0 ? "  " + c(CLR.cyan,   "待领 " + canClaim) : "")
        + (ongoing  > 0 ? "  " + c(CLR.yellow, "进行中 " + ongoing) : "");

    lines.push("  " + (allDone ? "✅" : "📋") + " " + c(CLR.bold, "每日任务：") + taskStat);

    // 列出未完成任务（不超过5条）
    if (!allDone) {
      const pendingTasks = tasks.filter((t) => t.status !== 3);
      if (pendingTasks.length > 0 && pendingTasks.length <= 5) {
        for (const t of pendingTasks) {
          const stIcon = { 0: "⏳", 1: "🔄", 2: "🎁" }[t.status] || "❓";
          lines.push("     " + stIcon + " " + t.name + c(CLR.gray, " (+" + t.prize + "里程)"));
        }
      }
    }
  } else {
    lines.push("  📋 每日任务：" + c(CLR.yellow, "获取失败"));
  }

  lines.push("");

  // 4. 里程瓜分
  if (r.share) {
    const s = r.share;
    let shareStr;
    if (!s.active) {
      shareStr = c(CLR.gray, "活动未开启");
    } else if (!s.joined) {
      shareStr = c(CLR.yellow, "未报名");
    } else if (!s.checkedIn) {
      shareStr = c(CLR.green, "已报名") + "  " + c(CLR.yellow, "今日未打卡 ⚠️");
    } else {
      shareStr = c(CLR.green, "已报名  今日已打卡 ✅");
    }
    lines.push("  " + (s.checkedIn ? "✅" : s.joined ? "⚠️" : "➖") + " " + c(CLR.bold, "里程瓜分：") + shareStr);
  }

  // 5. 花神祈福
  if (r.flower) {
    const f = r.flower;
    if (!f.active) {
      lines.push("  ➖ " + c(CLR.bold, "花神祈福：") + c(CLR.gray, f.msg || "活动未开启"));
    } else {
      const allTypes   = ALL_CARD_TYPES.map((t) => ({ type: t, name: CARD_TYPE_MAP[t] }));
      const owned      = allTypes.filter((t) => (f.cards.find((cc) => cc.type === t.type)?.count || 0) > 0).length;
      const missList   = allTypes.filter((t) => (f.cards.find((cc) => cc.type === t.type)?.count || 0) === 0);
      const drawRemain = f.remainDrawCount;
      const allGot     = owned === allTypes.length;

      lines.push("  " + (allGot ? "🏆" : "🌸") + " " + c(CLR.bold, "花神祈福：")
        + c(allGot ? CLR.green : CLR.bold, owned + "/" + allTypes.length + " 种")
        + "  剩余摇签 " + c(CLR.bold, String(drawRemain)) + " 次");

      // 每张卡状态，每行3个
      const cardRow = allTypes.map((t) => {
        const card  = f.cards.find((cc) => cc.type === t.type);
        const count = card?.count || 0;
        return (count > 0 ? "✅" : "⬜")
             + " " + (count > 0 ? t.name : c(CLR.gray, t.name))
             + c(CLR.gray, " x" + count);
      });
      for (let i = 0; i < cardRow.length; i += 3) {
        lines.push("     " + cardRow.slice(i, i + 3).join("   "));
      }

      if (allGot) {
        lines.push("     " + c(CLR.bold + CLR.green, "★ 已集齐全套，可兑换大奖！"));
      } else {
        lines.push("     " + c(CLR.yellow, "还缺：") + missList.map((t) => t.name).join("、"));
      }
    }
  }

  lines.push("");

  // 6. 挑战现金 + 现金奖励打卡
  if (r.cashSign) {
    const cs = r.cashSign;
    if (!cs.active) {
      lines.push("  ➖ " + c(CLR.bold, "挑战现金：") + c(CLR.gray, "活动未开启"));
    } else {
      // 挑战现金
      const chal     = cs.chalInfo;
      const chalCal  = chal?.calendarInfo;
      const chalDays = chalCal?.continueSignCount || 0;
      const chalDone = chalCal?.isTodayVisit === 1;
      const chalRecv = chal?.recPrizeAmount || 0;
      lines.push("  " + (chalDone ? "✅" : "⚠️") + " " + c(CLR.bold, "挑战现金：")
        + "连续 " + c(CLR.bold, String(chalDays)) + " 天  "
        + (chalDone ? c(CLR.green, "今日已打卡") : c(CLR.yellow, "今日未打卡"))
        + "  已领奖励 " + c(CLR.green, "¥" + chalRecv));

      // 奖励阶梯（紧凑版）
      const prizeList = chal?.prizeList || [];
      if (prizeList.length > 0) {
        const prizeRow = prizeList.map((p) => {
          const claimed = p.recState === 1;
          const reach   = chalDays >= p.minDay;
          const icon    = claimed ? c(CLR.green, "[已领]") : (reach ? c(CLR.cyan, "[可领]") : c(CLR.gray, "[" + p.minDay + "天]"));
          return icon + c(CLR.gray, " " + p.prizeName);
        });
        lines.push("     阶梯：" + prizeRow.join("  "));
      }

      // 现金奖励打卡
      const cash     = cs.cashInfo;
      const cashCal  = cash?.calendarInfo;
      const cashDays = cashCal?.continueSignCount || 0;
      const cashDone = cashCal?.isTodayVisit === 1;
      const cashAmt  = cash?.allNewAmount || cash?.partNewAmount || 0;
      lines.push("  " + (cashDone ? "✅" : "⚠️") + " " + c(CLR.bold, "现金奖励打卡：")
        + "连续 " + c(CLR.bold, String(cashDays)) + " 天  "
        + (cashDone ? c(CLR.green, "今日已打卡") : c(CLR.yellow, "今日未打卡"))
        + (cashAmt ? "  单次 ¥" + cashAmt : ""));

      // 近期奖励记录（最多3条）
      const records = cs.rewardRecords?.recordList || [];
      if (records.length > 0) {
        const recStrs = records.slice(0, 3).map((rec) =>
          c(CLR.gray, (rec.calendar || "")) + " 连续" + rec.continueDay + "天 " + c(CLR.green, "+¥" + (rec.totalAmount || 0))
        );
        lines.push("     近期：" + recStrs.join("  |  "));
      }
    }
  }

  // 7. 攒现金
  if (r.cashClockin) {
    const cc     = r.cashClockin;
    const ccDone = cc.isTodayClocked === true;
    lines.push("  " + (ccDone ? "✅" : "⚠️") + " " + c(CLR.bold, "打卡攒现金：")
      + (ccDone ? c(CLR.green, "今日已打卡") : c(CLR.yellow, "今日未打卡"))
      + "  金币 " + c(CLR.bold, String(cc.clockinAmount))
      + "  可提现 " + c(CLR.green, "¥" + (cc.canWithdrawalAmount || 0)));
  }

  lines.push(LINE_BIG);
  return lines.join("\n");
}

// ============================================================
// 处理单个账号
// ============================================================
async function queryAccount(cfg) {
  plog("账号 " + cfg.index + "：" + cfg.remark + "  登录中...");

  const result = { basic: null, sign: null, tasks: null, share: null, flower: null, cashSign: null, cashClockin: null };

  let tokenData;
  try {
    const code = await getWxCode(cfg.wxid, cfg.appid);
    tokenData  = await tcLogin(code);
    pok("登录成功  " + c(CLR.gray, "openid: " + tokenData.openid?.substring(0, 20) + "..."));
  } catch (e) {
    perr("登录失败：" + e.message);
    return { ...result, loginError: e.message };
  }

  const h = buildHeaders(tokenData);

 // pinfo("查询基础信息...");
  try { result.basic  = await queryBasicInfo(h); } catch { /* ignore */ }

 // pinfo("查询签到状态...");
  try { result.sign   = await querySignStatus(h); } catch { /* ignore */ }

 // pinfo("查询每日任务...");
  try { result.tasks  = await queryTaskList(h); } catch { /* ignore */ }

 // await sleep(300);
 // pinfo("查询里程瓜分...");
  try { result.share  = await queryShareActivity(h); } catch { /* ignore */ }

 // pinfo("查询花神祈福...");
  try { result.flower = await queryFlowerStatus(tokenData); } catch { /* ignore */ }

 // await sleep(300);
  const cookieObj = await initCookie(tokenData.sectoken).catch(() => ({}));

 // pinfo("查询现金打卡...");
  try { result.cashSign     = await queryCashSign(tokenData, cookieObj); } catch { /* ignore */ }

  //pinfo("查询攒现金...");
  try { result.cashClockin  = await queryCashClockin(tokenData, cookieObj); } catch { /* ignore */ }

  return result;
}

// ============================================================
// 主函数
// ============================================================
async function main() {
  const now = new Date().toLocaleString("zh-CN", { timeZone: "Asia/Shanghai" });

  console.log(LINE_BIG);
 // console.log("  " + c(CLR.bold + CLR.cyan, ">> 同程旅行 活动进度查询（只读）"));
 // console.log("     " + c(CLR.gray, now));
  //console.log(LINE_BIG);

  const accts = parseAccounts();
  if (!accts.length) {
    perr("未找到有效账号");
    pinfo("用法一（命令行）：node TC.js <wxid1> [wxid2] ...");
    pinfo("用法二（带备注）：node TC.js \"备注#wxid\" ...");
    console.log();
    process.exit(1);
  }
//  pinfo("共 " + accts.length + " 个账号");
  console.log();

  const summaries = [];
  for (const a of accts) {
    try {
      const r = await queryAccount(a);
      if (r.loginError) {
        summaries.push(LINE_BIG + "\n  " + c(CLR.red, "【同程旅行】" + a.remark + " 登录失败：" + r.loginError) + "\n" + LINE_BIG);
      } else {
        summaries.push(buildSummary(a.remark, r));
      }
    } catch (e) {
      summaries.push(LINE_BIG + "\n  " + c(CLR.red, "【同程旅行】" + a.remark + " 异常：" + e.message) + "\n" + LINE_BIG);
    }
  //  console.log(); // 账号间空行
  }

  // 输出所有汇总
//  console.log("\n" + "=".repeat(54));
 //console.log("  " + c(CLR.bold + CLR.white, ">> 查询汇总"));
//  console.log("=".repeat(54) + "\n");

  for (const s of summaries) {
    console.log(s);
    console.log();
  }

 // console.log(c(CLR.bold + CLR.green, "  >> 查询完成！"));
  console.log();
}

main().catch((e) => {
  perr("Fatal：" + e.message);
  process.exit(1);
});
