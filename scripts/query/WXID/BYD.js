/**
 * 比亚迪王朝（协议版）- 纯查询版【无签到】
 * 流程：wxid -> 拉 code -> decryptCode 拿 session_id -> 查询积分/签到日历
 *
 * 用法：node BYD.js wxid
 *
 * 环境变量（由xdd后台自动传递）：
 *   WECHAT_SERVER     取 code 服务地址
 *   WECHAT_SERVER_NEW 新地址（可选）
 */

const crypto = require('crypto');
const axios = require('axios');

// ====== 配置中心 ======
const CFG = {
  // 从环境变量获取地址（由xdd后台自动传递）
  wechatServer: (process.env.WECHAT_SERVER || 'http://180.152.5.230:8011').trim(),
  wechatServerNew: (process.env.WECHAT_SERVER_NEW || '').trim(),
  codeApiPath: process.env.CODE_API_PATH || '/api/v1/wx/app/get/code',
  appid: process.env.APPID || 'wxa28c31d4ff7ae869',
  bydDomain: process.env.BYD_DOMAIN || 'https://weixin90.bydauto.com.cn',

  appKey: 'wcMinaApi',
  appSecret: 'wAWUW(jC6bZ%)Hhs',
  aesKey: '6298134637481073',
  aesIv: 'PacWxpOwHnVgsQrh',
  appVersion: '240',
  appClient: 'mina',
  belongBrand: 'wc',

  ua: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13) UnifiedPCWindowsWechat(0xf254181d) XWEB/19201',
  referer: 'https://servicewechat.com/wxa28c31d4ff7ae869/668/page-frame.html'
};

// ====== 接口定义 ======
const API = {
  decryptCode: '/?service=mina.decryptCode',
  calendar: '/?s=ForCommonUcSrv.forward&serviceDir=activity/sign/getSignCalendar',
  userIntegral: '/?service=App.ForInterfaceMina.forward&serverFlag=integralMallApi&serviceDir=Order.getUserIntegralNew',
  bindStatus: '/?s=App.ForInterfaceMina.forward&serverFlag=uc_sso&serviceDir=oauth.GetBindStatus'
};

// ====== 工具函数 ======
const NONCE_CHARS = 'ABCDEFGHJKMNPQRSTWXYZabcdefhijkmnprstwxyz2345678';

/**
 * 获取微信协议服务器地址（新旧地址）
 */
function getWxServerUrls() {
  const oldUrl = CFG.wechatServer.replace(/\/+$/, '');
  const newUrl = CFG.wechatServerNew ? CFG.wechatServerNew.replace(/\/+$/, '') : oldUrl;
  return { oldUrl, newUrl };
}

// 从命令行参数获取 wxid
function getAccounts() {
  const cliWxid = process.argv[2]?.trim();
  if (cliWxid) {
    return [{ wxid: cliWxid, appid: CFG.appid }];
  }
  return [];
}

function randomNonce(len = 16) {
  let out = '';
  for (let i = 0; i < len; i++) {
    out += NONCE_CHARS.charAt(Math.floor(Math.random() * NONCE_CHARS.length));
  }
  return out;
}

function traceId() {
  const b = crypto.randomBytes(16);
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  const h = b.toString('hex');
  return `mina-${h.slice(0, 8)}-${h.slice(8, 12)}-${h.slice(12, 16)}-${h.slice(16, 20)}-${h.slice(20, 32)}`;
}

function signHeaders() {
  const nonce = randomNonce(16);
  const curtime = String(Math.floor(Date.now() / 1000));
  const checksum = crypto
    .createHash('sha256')
    .update(`${CFG.appSecret}${nonce}${curtime}`)
    .digest('hex')
    .toLowerCase();

  return {
    Host: new URL(CFG.bydDomain).host,
    Appkey: CFG.appKey,
    Nonce: nonce,
    Curtime: curtime,
    Checksum: checksum,
    Algo: '1',
    'x-clienttraceid': traceId(),
    'Content-Type': 'application/json',
    Accept: '*/*',
    'User-Agent': CFG.ua,
    Referer: CFG.referer
  };
}

function aesEncrypt(obj) {
  const text = typeof obj === 'string' ? obj : JSON.stringify(obj);
  const cipher = crypto.createCipheriv('aes-128-cbc', Buffer.from(CFG.aesKey, 'utf8'), Buffer.from(CFG.aesIv, 'utf8'));
  let encrypted = cipher.update(text, 'utf8', 'base64');
  encrypted += cipher.final('base64');
  return encrypted;
}

function aesDecrypt(base64Text) {
  const decipher = crypto.createDecipheriv('aes-128-cbc', Buffer.from(CFG.aesKey, 'utf8'), Buffer.from(CFG.aesIv, 'utf8'));
  let plain = decipher.update(base64Text, 'base64', 'utf8');
  plain += decipher.final('utf8');
  return JSON.parse(plain);
}

async function postJson(url, body, timeoutMs = 15000, headers = {}) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...headers },
      body: JSON.stringify(body),
      signal: controller.signal
    });
    const txt = await res.text();
    let data = txt;
    try { data = JSON.parse(txt); } catch (_) {}
    return { ok: res.ok, status: res.status, data, text: txt };
  } finally {
    clearTimeout(timer);
  }
}

async function postEncrypted(path, payload) {
  const url = `${CFG.bydDomain}${path}`;
  const bodyCipher = aesEncrypt(payload);
  const resp = await postJson(url, bodyCipher, 20000, signHeaders());

  let raw = resp.data;
  if (typeof raw === 'object' && raw && raw.ret !== undefined) return raw;
  if (typeof raw === 'object' && raw && typeof raw.data === 'string') raw = raw.data;
  if (typeof raw !== 'string') throw new Error(`返回格式异常 status=${resp.status}`);

  return aesDecrypt(raw);
}

// ====== 核心业务 ======
/**
 * 查询设备在线状态（所有地址），返回设备所在的服务器地址列表
 */
async function queryDeviceStatus(wxid) {
  const { oldUrl, newUrl } = getWxServerUrls();
  const urls = [...new Set([oldUrl, newUrl])];
  const results = [];
  for (const url of urls) {
    const label = url === oldUrl ? '旧地址' : '新地址';
    try {
      const controller = new AbortController();
      const timer = setTimeout(() => controller.abort(), 10000);
      const resp = await fetch(`${url}/api/v1/wx/user/status`, { signal: controller.signal });
      clearTimeout(timer);
      const data = await resp.json();
      const info = data?.data?.[wxid];
      if (info) results.push({ url, online: info.survival === 1, source: label, nickname: info.nickname || '' });
    } catch (_) {}
  }
  return results;
}

async function getCodeByWxid(wxid, appid) {
  const { oldUrl, newUrl } = getWxServerUrls();

  // 智能选择地址：先查设备在线状态
  const statusResults = await queryDeviceStatus(wxid);
  let priorityUrl = oldUrl;
  if (statusResults.length > 0) {
    const onlineResult = statusResults.find(r => r.online);
    if (onlineResult) {
      priorityUrl = onlineResult.url;
    } else {
      priorityUrl = statusResults[0].url;
    }
  }

  const allUrls = [...new Set([priorityUrl, oldUrl, newUrl])];

  let extra = {};
  if (process.env.CODE_EXTRA_JSON) {
    try { extra = JSON.parse(process.env.CODE_EXTRA_JSON); } catch (e) {}
  }

  for (let i = 0; i < allUrls.length; i++) {
    const serverUrl = allUrls[i];
    if (!serverUrl) continue;
    const label = serverUrl === oldUrl ? '旧地址' : '新地址';
    try {
      const url = `${serverUrl.replace(/\/$/, '')}${CFG.codeApiPath}`;
      const body = { wxid, appid, ...extra };
      const res = await postJson(url, body, 15000);
      const code = res.data?.Data?.code || res.data?.data?.code || res.data?.code;
      if (code) return code;
      // 业务失败，继续尝试下一个地址
      if (res.data?.code !== undefined) {
        console.log(`⚠️  ${label} 业务错误: ${res.data?.Message || res.data?.msg || ''}`);
        continue;
      }
    } catch (e) {
      console.log(`⚠️  ${label} 请求异常: ${e.message}`);
      continue;
    }
  }

  throw new Error(`所有地址均无法获取code`);
}

async function getSessionByCode(code) {
  const ret = await postEncrypted(API.decryptCode, { code });
  if (ret?.ret !== 200 || !ret?.data?.session_id) {
    throw new Error(`会话获取失败`);
  }
  return ret.data;
}

function findTodayNode(calData) {
  const today = calData?.today;
  const cal = calData?.calendar;
  if (!today || !Array.isArray(cal)) return null;
  for (const monthArr of cal) {
    if (!Array.isArray(monthArr)) continue;
    const node = monthArr.find(x => x && x.sign_date === today);
    if (node) return node;
  }
  return null;
}

async function runQueryTasks(sessionId) {
  const [calRes, userRes] = await Promise.all([
    postEncrypted(API.calendar, {
      belong_brand: CFG.belongBrand,
      session_id: sessionId,
      app_version: CFG.appVersion,
      app_client: CFG.appClient
    }),
    postEncrypted(API.userIntegral, {
      session_id: sessionId,
      app_version: CFG.appVersion,
      app_client: CFG.appClient
    })
  ]);

  let bindRes = null;
  if (userRes?.ret !== 200) {
    bindRes = await postEncrypted(API.bindStatus, {
      session_id: sessionId,
      app_version: CFG.appVersion,
      app_client: CFG.appClient
    });
  }

  return { calRes, userRes, bindRes };
}

function parseAccountInfo(userRes, bindRes) {
  const phone = userRes?.data?.data?.phone_no || bindRes?.data?.mobile || '未知';
  const points =
    userRes?.data?.data?.integral ??
    userRes?.data?.data?.available_integral_sum ??
    userRes?.data?.integral ??
    '0';
  return { phone, points };
}

function fmtTs(sec) {
  if (!sec) return '-';
  return new Date(sec * 1000).toLocaleString('zh-CN', { hour12: false, timeZone: 'Asia/Shanghai' });
}

// ====== 美化日志输出 ======
async function runOne(acc, idx) {
  const wxidShow = acc.wxid.length > 12 ? `${acc.wxid.slice(0, 12)}...` : acc.wxid;
  console.log(`\n 账号 ${idx} | wxid: ${wxidShow} `);

  try {
    const code = await getCodeByWxid(acc.wxid, acc.appid);
    console.log(`│ ✅ Code获取成功`);

    const sess = await getSessionByCode(code);
  //  console.log(`│ ✅ 登录成功`);
 //   console.log(`│ ⏰ 有效期：${fmtTs(sess.expired_timestamp)}`);

    const { calRes, userRes, bindRes } = await runQueryTasks(sess.session_id);
    const { phone, points } = parseAccountInfo(userRes, bindRes);

    console.log(`│`);
    console.log(`│ 👤 手机号：${phone}`);
    console.log(`│ 💰 积分：${points}`);

    if (calRes?.ret === 200) {
      const d = calRes.data || {};
      const todayNode = findTodayNode(d);
      console.log(`│`);
      console.log(`│ 📅 日期：${d.today || '-'}`);
      console.log(`│ 📆 连签：${d.durationDays ?? '-'}天`);
      console.log(`│ ✅ 今日已签：${todayNode?.is_sign ? '是' : '否'}`);
      d.tips && console.log(`│ 💡 ${d.tips}`);
    } else {
      console.log(`│ ❌ 签到记录查询失败`);
    }

  } catch (e) {
    console.log(`│ ❌ 失败：${e.message}`);
  }

//  console.log(`└────────────────────────────────────┘`);
}

// ====== 入口执行 ======
(async () => {
  console.log('🚀 比亚迪查询');

  try {
    const accounts = getAccounts();
    if (!accounts.length) {
      console.log('❌ 请通过命令行传入 wxid');
      process.exit(1);
    }

    for (let i = 0; i < accounts.length; i++) {
      await runOne(accounts[i], i + 1);
    }

    console.log('\n🎉 全部查询完成！');

  } catch (e) {
    console.log('\n💥 脚本异常：' + e.message);
    process.exit(1);
  }
})();