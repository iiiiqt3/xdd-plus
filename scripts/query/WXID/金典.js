/**
 * 金典鲜活查询.js
 * 缓存：共用金典活动.js 的 jindian_cache.json
 * 说明：查询鲜活挑战账号信息，不执行任务/抽奖/组队，仅输出当前状态
 *
 * 用法：
 *   node 金典鲜活查询.js wxid_xxx
 *   node 金典鲜活查询.js 备注#wxid_xxx
 *   node 金典鲜活查询.js wxid1&wxid2
 *
 * 环境变量：
 *   WECHAT_SERVER       微信代理服务地址（由xdd后台自动传递）
 *   WECHAT_SERVER_NEW   新地址（由xdd后台自动传递，可选）
 *   JINDIAN_XH_APP_KEY   活动 app_key（默认沿用金典活动）
 *   JINDIAN_PROXY        直连代理
 *   JINDIAN_PROXY_API    代理 API
 */

const fs = require('fs');
const path = require('path');
const https = require('https');
const { URL } = require('url');
const axios = require('axios');
let HttpsProxyAgent = null;
try { HttpsProxyAgent = require('https-proxy-agent').HttpsProxyAgent || require('https-proxy-agent'); } catch (_) {}

const SCRIPT_NAME = '微信协议-金典鲜活查询';
const APPID = 'wxf32616183fb4511e';
const TENANT_ID = '1718857849685876737';
const APP_KEY = String(process.env.JINDIAN_XH_APP_KEY || process.env.JINDIAN_APP_KEY || 'zd123a10187c995e97').trim();
// 从环境变量获取地址（由xdd后台自动传递）
const WECHAT_SERVER = String(process.env.WECHAT_SERVER || 'http://180.152.5.230:8011').trim();
const WECHAT_SERVER_NEW = String(process.env.WECHAT_SERVER_NEW || '').trim();
const CACHE_FILE = path.join(__dirname, 'jindian_cache.json');
const MS_BASE = 'https://msmarket.msx.digitalyili.com';
const API_BASE = 'https://wx-camp-hc-api-01.mscampapi.digitalyili.com/wx-camp-jddyr/stage';
const PROXY_DIRECT = String(process.env.JINDIAN_PROXY || '').trim();
const PROXY_API = String(process.env.JINDIAN_PROXY_API || '').trim();
const PROXY_RETRY = Math.max(1, Number(process.env.JINDIAN_PROXY_RETRY || 5));
const UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13) UnifiedPCWindowsWechat(0xf254186b) XWEB/19481';

const sleep = (ms) => new Promise(r => setTimeout(r, ms));
const rand = (a, b) => Math.floor(Math.random() * (b - a + 1)) + a;
const log = (s = '') => console.log(s);
const proxyState = { url: '', agent: null };

// ========== 账号解析 ==========
function parseAccounts(raw) {
  return String(raw || '')
    .split(/[@&\n]+/)
    .map(s => s.trim()).filter(Boolean)
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
  const args = process.argv.slice(2).join(' ').trim();
  if (args) {
    const list = parseAccounts(args);
    if (list.length) return list;
  }
  return [];
}

/** 获取微信协议服务器地址（仅新地址） */
function getWxServerUrl() {
  return (WECHAT_SERVER_NEW || WECHAT_SERVER).replace(/\/+$/, '');
}

function maskPhone(phone) {
  const s = String(phone || '-');
  if (s.length !== 11) return s;
  return s.slice(0, 3) + '****' + s.slice(7);
}
function readJson(file, def = {}) { try { return JSON.parse(fs.readFileSync(file, 'utf8')); } catch { return def; } }
function writeJson(file, obj) { fs.writeFileSync(file, JSON.stringify(obj, null, 2), 'utf8'); }
function normalizeProxy(raw) { const s = String(raw || '').trim(); return s ? (/^https?:\/\//i.test(s) ? s : `http://${s}`) : ''; }
function extractProxy(text) {
  const line = String(text || '').trim().split(/\r?\n/).map(x => x.trim()).find(Boolean) || '';
  const m = line.match(/((?:https?:\/\/)?[^\s]+:\d+)/i);
  return normalizeProxy(m ? m[1] : line);
}
async function fetchProxy() {
  if (PROXY_DIRECT) return normalizeProxy(PROXY_DIRECT);
  if (!PROXY_API) return '';
  try {
    const { data } = await axios.get(PROXY_API, { timeout: 12000, proxy: false, validateStatus: () => true });
    return extractProxy(typeof data === 'string' ? data : JSON.stringify(data));
  } catch (e) { log(`⚠️ 获取代理失败: ${e.message}`); return ''; }
}
async function switchProxy(force = false) {
  if (!force && proxyState.url) return true;
  const p = await fetchProxy();
  if (!p) return false;
  if (!HttpsProxyAgent) { log('⚠️ 未安装 https-proxy-agent，无法使用代理'); return false; }
  proxyState.url = p;
  proxyState.agent = new HttpsProxyAgent(p);
  axios.defaults.proxy = false;
  axios.defaults.httpAgent = proxyState.agent;
  axios.defaults.httpsAgent = proxyState.agent;
  log(`🌐 使用代理: ${p}`);
  return true;
}
function isWafBlocked(x) { return /WAF|拦截|block-pages|status:403|status=403|ErrorCode:639|请求已中断/i.test(String(x || '')); }
function getMsg(d) {
  if (!d) return '未知错误';
  if (typeof d === 'string') return d.replace(/\s+/g, ' ').slice(0, 300);
  return d.msg || d.message || d.error?.message || d.error?.msg || JSON.stringify(d).slice(0, 300);
}
function jwtExp(token) {
  try {
    const p = String(token || '').split('.')[1];
    if (!p) return 0;
    const obj = JSON.parse(Buffer.from(p.replace(/-/g, '+').replace(/_/g, '/'), 'base64').toString('utf8'));
    return Number(obj.exp || 0) * 1000;
  } catch { return 0; }
}
function tokenValid(token) { const exp = jwtExp(token); return !!token && (!exp || exp - Date.now() > 5 * 60 * 1000); }
function isTokenBad(data) { return data?.code === -1 || /TOKEN已失效|token.*失效|登录过期/i.test(getMsg(data)); }

// ========== 网络请求 ==========
function rawHttpsRequest({ url, method = 'GET', headers = {}, body = null, timeout = 25000 }) {
  return new Promise(resolve => {
    let u; try { u = new URL(url); } catch (e) { return resolve({ status: 0, data: 'invalid url:' + e.message }); }
    const buf = body == null ? null : Buffer.from(typeof body === 'string' ? body : JSON.stringify(body), 'utf8');
    const finalHeaders = { ...headers };
    if (buf) finalHeaders['Content-Length'] = buf.length;
    const opts = { method, hostname: u.hostname, port: u.port || 443, path: u.pathname + u.search, headers: finalHeaders, timeout };
    if (proxyState.agent) opts.agent = proxyState.agent;
    const req = https.request(opts, res => {
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => {
        const text = Buffer.concat(chunks).toString('utf8');
        let data = text;
        if (/json/i.test(res.headers['content-type'] || '')) { try { data = JSON.parse(text); } catch {} }
        resolve({ status: res.statusCode, data, headers: res.headers });
      });
    });
    req.on('timeout', () => req.destroy(new Error('timeout')));
    req.on('error', e => resolve({ status: 0, data: 'ERR:' + e.message }));
    if (buf) req.write(buf);
    req.end();
  });
}
function msHeaders(accessToken = '') {
  return {
    Host: 'msmarket.msx.digitalyili.com', Connection: 'keep-alive', 'register-source': '', shareid: '', xweb_xhr: '1', scene: '1000',
    'access-token': accessToken || '', 'User-Agent': UA, channel: 'copyUrl', 'Content-Type': 'application/json',
    'tenant-id': '\t' + TENANT_ID, Accept: '*/*', Referer: `https://servicewechat.com/${APPID}/815/page-frame.html`,
  };
}

// ========== 登录链路 ==========
/**
 * 查询设备在线状态
 */
async function queryDeviceStatus(wxid) {
  const url = getWxServerUrl();
  if (!url) return [];
  const results = [];
  try {
    const { data } = await axios.get(`${url}/api/v1/wx/user/status`, { timeout: 10000 });
    const info = data?.data?.[wxid];
    if (info) results.push({ url, online: info.survival === 1, nickname: info.nickname || '' });
  } catch (_) {}
  return results;
}

async function wxGetCode(wxid) {
  const serverUrl = getWxServerUrl();
  if (!serverUrl) throw new Error('未配置微信协议服务器地址');
  const paths = ['/api/v1/wx/app/get/code', '/api/v1/wx/app/get/jscode', '/api/wx/app/get/code'];
  const base = serverUrl.replace(/\/$/, '');
  const errs = [];

  for (const p of paths) {
    try {
      const { data } = await axios.post(base + p, { wxid, appid: APPID }, { timeout: 20000, validateStatus: () => true });
      const code = data?.Data?.code || data?.data?.code || data?.code || data?.Data?.jsCode || data?.data?.jsCode || data?.jsCode;
      if (code) return String(code).trim();
      const errMsg = data?.Message || data?.msg || '';
      if (errMsg) log(`⚠️  业务错误: ${errMsg}`);
      errs.push(`${base}${p}: no-code`);
    } catch (e) { errs.push(`${base}${p}: ${e.message}`); }
  }

  throw new Error('协议接口未返回code: ' + errs.join(' | '));
}
async function msLogin(jsCode) {
  let last = '';
  for (let i = 1; i <= PROXY_RETRY; i++) {
    const { status, data } = await rawHttpsRequest({ url: `${MS_BASE}/gateway/api/auth/account/login`, method: 'POST', body: { jsCode }, headers: msHeaders('') });
    const d = data?.data || data || {};
    const token = d.accessToken || d.access_token || d.token || '';
    if (token) return String(token).trim();
    last = `ms登录失败(status:${status}): ${getMsg(data)}`;
    if (i < PROXY_RETRY && isWafBlocked(last)) { await switchProxy(true); log(`⚠️ ms登录WAF，重试 ${i}/${PROXY_RETRY}`); await sleep(5000); continue; }
    break;
  }
  throw new Error(last);
}
function extractAuthCode(data) {
  const d = data?.data;
  const arr = [data?.authorization_code, data?.authorizationCode, data?.code, d?.authorization_code, d?.authorizationCode, d?.code, typeof d === 'string' ? d : ''];
  return String(arr.find(x => /^[0-9a-f]{32}$/i.test(String(x || '').trim())) || '').trim();
}
async function msAuthorize(msToken) {
  let last = '';
  for (let i = 1; i <= PROXY_RETRY; i++) {
    const { status, data } = await rawHttpsRequest({ url: `${MS_BASE}/developer/oauth2/buyer/authorize?app_key=${encodeURIComponent(APP_KEY)}`, method: 'GET', headers: msHeaders(msToken) });
    const code = extractAuthCode(data);
    if (code) return code;
    last = `授权码失败(status:${status}): ${getMsg(data)}`;
    if (i < PROXY_RETRY && isWafBlocked(last)) { await switchProxy(true); log(`⚠️ authorize WAF，重试 ${i}/${PROXY_RETRY}`); await sleep(5000); continue; }
    break;
  }
  throw new Error(last);
}

// ========== 客户端 ==========
class Client {
  constructor(acc, cache, total) {
    this.wxid = acc.wxid;
    this.remark = acc.remark;
    this.cache = cache;
    this.total = total;
    const c = cache[this.wxid] || cache[this.remark] || {};
    this.token = c.jddyrToken || '';
    this.msToken = c.msToken || c.msAccessToken || '';
    this.openId = c.jddyrOpenId || '';
    this.teamId = c.jddyrTeamId || '';
    this.user = c.jddyrUser || {};
  }
  save() {
    const old = this.cache[this.wxid] || {};
    this.cache[this.wxid] = {
      ...old,
      wxid: this.wxid,
      remark: this.remark,
      msToken: this.msToken || old.msToken || '',
      jddyrToken: this.token || '',
      jddyrOpenId: this.openId || '',
      jddyrTeamId: this.teamId || '',
      jddyrUser: this.user || {},
      jddyrUpdateAt: new Date().toLocaleString('zh-CN', { hour12: false }),
    };
    writeJson(CACHE_FILE, this.cache);
  }
  headers() {
    return {
      Host: 'wx-camp-hc-api-01.mscampapi.digitalyili.com', Connection: 'keep-alive', Authorization: this.token || '',
      'User-Agent': UA, Accept: 'application/json, text/plain, */*', xweb_xhr: '1', 'Content-Type': 'application/json',
      Referer: `https://servicewechat.com/${APPID}/815/page-frame.html`, 'Accept-Language': 'zh-CN,zh;q=0.9',
    };
  }
  async api(pathname, body = {}, retry = true) {
    const { status, data } = await rawHttpsRequest({ url: `${API_BASE}${pathname}`, method: 'POST', body, headers: this.headers() });
    if (retry && (status === 401 || status === 403 || isTokenBad(data))) {
      log(`♻️ ${this.remark} 鲜活CK失效，重新登录`);
      this.token = '';
      await this.login(true);
      return this.api(pathname, body, false);
    }
    return data;
  }
  async login(force = false) {
    if (!force && tokenValid(this.token)) {
      log(`✅ ${this.remark} 使用缓存CK`);
      return;
    }
    log(`🔄 ${this.remark} 协议登录鲜活挑战...`);
    const jsCode = await wxGetCode(this.wxid);
    this.msToken = await msLogin(jsCode);
    const authCode = await msAuthorize(this.msToken);
    const oldToken = this.token;
    this.token = oldToken || '';
    const { status, data } = await rawHttpsRequest({
      url: `${API_BASE}/userLogin`, method: 'POST', body: { code: authCode, byOpenId: '' },
      headers: this.headers(),
    });
    const d = data?.data || {};
    if (!d.token) throw new Error(`鲜活登录失败(status:${status}): ${getMsg(data)}`);
    this.token = String(d.token).trim();
    this.openId = d.openId || this.openId || '';
    this.save();
    log(`💾 ${this.remark} 新CK已缓存`);
  }
  async query() {
    await this.login(false);

    // 活动信息
    const act = await this.api('/qryActivity', {});
    const actName = act?.data?.activityName || act?.data?.name || '';

    // 奖池
    const prize = await this.api('/all/prize', {});
    const prizeList = Array.isArray(prize?.data) ? prize.data.map(x => `  ${x.name}`).join('\n') : '-';

    // 用户信息
    const ud = await this.api('/qryUserInfo', {});
    const u = ud?.data || {};
    this.user = u;
    this.openId = u.openId || this.openId;
    this.teamId = u.teamId || this.teamId || '';
    this.save();

    // 队伍信息
    let teamInfo = '-';
    if (this.teamId) {
      const td = await this.api('/qryTeam', { teamId: this.teamId });
      if (td?.code === 1 && td.data) {
        const members = (td.data.teamMembers || []).map(m => m.nickName || m.openId || '?').join('、');
        teamInfo = `${this.teamId} (${td.data.teamMembers?.length || 0}/3) [${members}]`;
      } else {
        teamInfo = this.teamId + ' (查询失败)';
      }
    }

    // 积分/挑战次数
    const lottery = Number(u.lotteryNum || 0);
    const totalLottery = Number(u.totalLotteryNum || 0);
    const integral = u.integral ?? u.score ?? u.point ?? u.points ?? '-';
    const totalIntegral = u.totalIntegral ?? u.totalScore ?? '-';

    // 近期奖励记录
    const ap = await this.api('/qryAwardPage', { page: 1, pageSize: 10 });
    const awardList = (ap?.data?.list || []);
    const awards = awardList.length
      ? awardList.map(x => `  ${x.awardName}${x.createTime ? ' (' + x.createTime.slice(5, 16) + ')' : ''}`).join('\n')
      : '  暂无';

    if (this.total === 1) {
      // 单账号：简洁输出
      log(`用户：${u.nickName || '-'}`);
      log(`手机：${maskPhone(u.mobilePhone)}`);
      log(`挑战次数：${lottery} / 累计 ${totalLottery}`);
      if (integral !== '-') log(`积分：${integral}${totalIntegral !== '-' ? ' / 累计 ' + totalIntegral : ''}`);
      log(`队伍：${teamInfo}`);
      log(`\n🎁 奖池：`);
      log(prizeList);
      log(`\n📦 近期奖励：`);
      log(awards);
    } else {
      // 多账号：带分隔
      log(`----------------------------------------`);
      log(`📌 账号   : ${this.remark} (${this.wxid})`);
      if (actName) log(`🏷️  活动   : ${actName}`);
      log(`👤 昵称   : ${u.nickName || '-'}`);
      log(`📱 手机   : ${maskPhone(u.mobilePhone)}`);
      log(`🎯 挑战次数: ${lottery} / 累计 ${totalLottery}`);
      if (integral !== '-') log(`💎 积分   : ${integral}${totalIntegral !== '-' ? ' / 累计 ' + totalIntegral : ''}`);
      log(`👥 队伍   : ${teamInfo}`);
      log(`\n🎁 奖池：`);
      log(prizeList);
      log(`\n📦 近期奖励：`);
      log(awards);
      log(`----------------------------------------`);
    }
  }
}

// ========== 入口 ==========
(async () => {
  const accounts = getAccounts();
  if (!accounts.length) {
    console.log('未提供账号');
    console.log('用法: node 金典鲜活查询.js wxid_xxx');
    console.log('用法: node 金典鲜活查询.js 备注#wxid_xxx');
    console.log('用法: node 金典鲜活查询.js wxid1&wxid2');
    process.exit(1);
  }

  console.log(`\n======== ${SCRIPT_NAME} ========`);

  const cache = readJson(CACHE_FILE, {});
  for (let i = 0; i < accounts.length; i++) {
    const acc = accounts[i];
    if (accounts.length > 1) console.log(`\n======== 查询 ${i + 1}/${accounts.length} ${acc.remark} ========`);
    try {
      const c = new Client(acc, cache, accounts.length);
      await c.query();
    } catch (e) {
      console.log(`❌ ${acc.remark} 查询失败: ${e.message}`);
    }
    if (i < accounts.length - 1) await sleep(rand(800, 1500));
  }
  console.log('\n======== 查询结束 ========');
})().catch(e => {
  console.error(e);
  process.exit(1);
});