/**
 * 雀巢会员查询脚本 - 纯查询版
 * 小程序://雀巢会员/TUUuwGVLLskWyai
 *
 * 用法：
 *   node 雀巢_query.js wxid_xxx
 *   node 雀巢_query.js 备注#wxid_xxx
 *   node 雀巢_query.js wxid1&wxid2
 *
 * 环境变量：
 *   WECHAT_SERVER     微信代理服务地址（由xdd后台自动传递）
 *   WECHAT_SERVER_NEW 新地址（由xdd后台自动传递，可选）
 */

const axios = require('axios');
// 从环境变量获取地址（由xdd后台自动传递）
const WECHAT_SERVER = (process.env.WECHAT_SERVER || 'http://180.152.5.230:8011').trim();
const WECHAT_SERVER_NEW = (process.env.WECHAT_SERVER_NEW || '').trim();
const AppID = 'wxc5db704249c9bb31';
const baseUrl = 'https://crm.nestlechinese.com';

// ========== Token 缓存 ==========
const tokenCache = {};

// ========== 服务器地址获取 ==========
/**
 * 获取微信协议服务器地址（新旧地址）
 */
function getWxServerUrls() {
  const oldUrl = WECHAT_SERVER.replace(/\/+$/, '');
  const newUrl = WECHAT_SERVER_NEW ? WECHAT_SERVER_NEW.replace(/\/+$/, '') : oldUrl;
  return { oldUrl, newUrl };
}

/**
 * 查询设备在线状态（所有地址），返回设备所在的服务器地址列表（在线优先）
 * @param {string} wxid
 * @returns {Promise<{url: string, online: boolean, source: string}[]>}
 */
async function queryDeviceStatus(wxid) {
  const { oldUrl, newUrl } = getWxServerUrls();
  const urls = [...new Set([oldUrl, newUrl])];
  const results = [];

  for (const url of urls) {
    const label = url === oldUrl ? '旧地址' : '新地址';
    try {
      const { data } = await axios.get(`${url}/api/v1/wx/user/status`, { timeout: 10000 });
      const info = data?.data?.[wxid];
      if (info) {
        results.push({ url, online: info.survival === 1, source: label, nickname: info.nickname || '' });
      }
    } catch (e) {
      // 查询失败不影响结果
    }
  }
  return results;
}

/**
 * 智能获取微信code —— 先查设备在线状态，优先向设备在线的服务器请求
 */
async function getWxCodeSmart(wxid) {
  const { oldUrl, newUrl } = getWxServerUrls();

  // 第一步：查设备在哪台服务器上线
  const statusResults = await queryDeviceStatus(wxid);
  let priorityUrl = oldUrl; // 默认先旧

  if (statusResults.length > 0) {
    const onlineResult = statusResults.find(r => r.online);
    if (onlineResult) {
      priorityUrl = onlineResult.url;
    } else {
      priorityUrl = statusResults[0].url;
    }
  }

  // 第二步：按优先级尝试获取code
  const allUrls = [...new Set([priorityUrl, oldUrl, newUrl])];
  for (const serverUrl of allUrls) {
    if (!serverUrl) continue;
    const label = serverUrl === oldUrl ? '旧地址' : '新地址';
    try {
      const { data } = await axios.post(
        `${serverUrl}/api/v1/wx/app/get/code`,
        { wxid, appid: AppID },
        { timeout: 20000 }
      );
      const code = (data?.data || data?.Data || {}).code || data?.Data;
      if (code) return String(code);

      const errMsg = data?.Message || data?.msg || data?.message || '';
      if (errMsg) {
        console.log(`⚠️  ${label} 业务错误: ${errMsg}`);
      }
    } catch (e) {
      console.log(`⚠️  ${label} 请求异常: ${e.message}`);
    }
  }

  throw new Error(`所有地址均无法获取code`);
}

const UA = 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148';

// ========== 账号解析 ==========
function parseAccounts(raw) {
  return raw
    .split(/[@&\n]/)
    .map(s => s.trim())
    .filter(Boolean)
    .map(s => {
      const i = s.indexOf('#');
      if (i < 0) return { wxid: s, remark: s };
      const a = s.slice(0, i).trim();
      const b = s.slice(i + 1).trim();
      if (a.toLowerCase().startsWith('wxid')) return { wxid: a, remark: b || a };
      return { wxid: b, remark: a || b };
    })
    .filter(x => x.wxid);
}

function getAccounts() {
  // 1. 优先命令行参数
  const args = process.argv.slice(2).join(' ').trim();
  if (args) {
    const list = parseAccounts(args);
    if (list.length) return list;
  }
  // 2. 其次环境变量
  const raw = process.env.WXID_QC || '';
  if (raw.trim()) return parseAccounts(raw);
  return [];
}

// ========== 获取 Token ==========
async function getNestleToken(auth_code, wxid) {
  const url = `${baseUrl}/openapi/identityservice/connect/token`;
  const headers = {
    'Content-Type': 'application/x-www-form-urlencoded',
    'User-Agent': UA,
  };
  const form = new URLSearchParams({
    client_id: 'wechatMini',
    client_secret: 'secret',
    grant_type: 'wechat_auth_code',
    auth_code,
  });

  const resp = await axios.post(url, form.toString(), { headers, timeout: 15000 });
  const token = resp.data?.access_token;
  if (!token) throw new Error(`Token获取失败: ${JSON.stringify(resp.data)}`);
  tokenCache[wxid] = token;
  return token;
}

async function ensureToken(wxid) {
  if (tokenCache[wxid]) return tokenCache[wxid];
  const code = await getWxCodeSmart(wxid);
  return await getNestleToken(code, wxid);
}

async function refreshToken(wxid) {
  delete tokenCache[wxid];
  const code = await getWxCodeSmart(wxid);
  return await getNestleToken(code, wxid);
}

// ========== 通用请求（401自动刷新） ==========
async function sendRequest(ctx, url, method, data = null, _retry = true) {
  const headers = {
    'User-Agent': UA,
    'content-type': 'application/json',
    'referer': `https://servicewechat.com/${AppID}/353/page-frame.html`,
    'authorization': `Bearer ${tokenCache[ctx.wxid] || ''}`,
  };

  try {
    const options = { url, method, headers, timeout: 15000, validateStatus: () => true };
    if (data) options.data = data;

    const resp = await axios(options);
    const status = resp.status;

    if ((status === 401 || status === 403) && _retry) {
      const newToken = await refreshToken(ctx.wxid);
      if (newToken) {
        headers.authorization = `Bearer ${newToken}`;
        return sendRequest(ctx, url, method, data, false);
      }
      return { errcode: 401, errmsg: 'Token刷新失败' };
    }

    return resp.data;
  } catch (e) {
    return { errcode: -1, errmsg: e.message };
  }
}

// ========== 查询 ==========
async function getUserInfo(ctx) {
  const data = await sendRequest(ctx, `${baseUrl}/openapi/member/api/User/GetUserInfo`, 'get');
  if (data.errcode === 200) return data.data || {};
  return {};
}

async function getUserBalance(ctx) {
  const data = await sendRequest(ctx, `${baseUrl}/openapi/pointsservice/api/Points/getuserbalance`, 'post');
  if (data.errcode === 200) return data.data;
  return -1;
}

// ========== 入口 ==========
(async () => {
  const accounts = getAccounts();
  if (!accounts.length) {
    console.log('未提供账号');
    console.log('用法: node 雀巢_query.js wxid_xxx');
    console.log('用法: node 雀巢_query.js 备注#wxid_xxx');
    process.exit(1);
  }

  for (let i = 0; i < accounts.length; i++) {
    const acc = accounts[i];
    const ctx = { wxid: acc.wxid };

    try {
      await ensureToken(acc.wxid);

      const info = await getUserInfo(ctx);
      const balance = await getUserBalance(ctx);

      const nickname = info.nickname || '-';
      const mobile = info.mobile || '-';

      if (accounts.length === 1) {
        // 单账号：简洁输出
        console.log(`用户：${nickname}`);
        console.log(`手机：${mobile}`);
        console.log(`巢币：${balance >= 0 ? balance : '查询失败'}`);
      } else {
        // 多账号：带分隔
        console.log(`${acc.remark}(${acc.wxid})`);
        console.log(`  ${nickname} | ${mobile} | 巢币：${balance >= 0 ? balance : '-'}`);
      }
    } catch (e) {
      console.log(`${acc.remark} 查询失败: ${e.message}`);
      delete tokenCache[acc.wxid];
    }
  }
})().catch(e => {
  console.error(e);
  process.exit(1);
});