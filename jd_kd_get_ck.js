// - export WECHAT_SERVER='http://172.17.0.7:8011'
// - wxjd：wxid，多号&分隔
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const axios = require('axios');

const SCRIPT_NAME = '京东快递-协议获取CK';
const APPID = 'wx73247c7819d61796';
const WECHAT_SERVER = (process.env.WECHAT_SERVER || 'http://172.17.0.7:8011').trim();
const WXJD = (process.env.wxjd || '').trim();
const CACHE_FILE = path.join(__dirname, 'jd_kd_ck.json');
const PUSH_API = (process.env.PUSH_API || 'http://127.0.0.1:5701/api/login/batch-smslogin').trim();
const PUSH_TOKEN = (process.env.PUSH_TOKEN || '123456').trim();
const CLIENT_VER = '2.0.2';
const JD_APPID = '599';
const SIGN_GSALT = 'sb2cwlYyaCSN1KUv5RHG3tmqxfEb8NKN';
const FINGER_BIZ_KEY = 'bce044c839bb9eb811aad5af18a629e199da4e13';
const REFERER = `https://servicewechat.com/${APPID}/864/page-frame.html`;
const UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13) UnifiedPCWindowsWechat(0xf254186b) XWEB/19481';
const FINGER_TK_DEFAULT = 'L64RTJ562VJEYNEQN67XMUWSR4UFLOIQHJYZ3MWERRIKJGP24SDSBDS4I4AMVU24Y3Y7A4UPDICN2';
const FINGER_ALPHABET = '23IL<N01c7KvwZO56RSTAfghiFyzWJqVabGH4PQdopUrsCuX*xeBjkltDEmn89.-';

function log(msg) { process.stderr.write(msg + '\n'); }
function sleep(ms) { return new Promise(resolve => setTimeout(resolve, ms)); }
function md5(s) { return crypto.createHash('md5').update(String(s), 'utf8').digest('hex'); }
function uuid() { return crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`; }
function randHex(n) { return crypto.randomBytes(n).toString('hex'); }

function fingerEncode(obj) {
  const text = encodeURIComponent(JSON.stringify(obj));
  let out = '';
  let i = 0;
  do {
    const e = text.charCodeAt(i++);
    const r = text.charCodeAt(i++);
    const u = text.charCodeAt(i++);
    const a = e >> 2;
    const c = (3 & e) << 4 | r >> 4;
    let s = (15 & r) << 2 | u >> 6;
    let f = 63 & u;
    if (Number.isNaN(r)) s = f = 64;
    else if (Number.isNaN(u)) f = 64;
    out += FINGER_ALPHABET.charAt(a) + FINGER_ALPHABET.charAt(c) + FINGER_ALPHABET.charAt(s) + FINGER_ALPHABET.charAt(f);
  } while (i < text.length);
  return out + '/';
}

function parseAccounts(raw) {
  return String(raw || '')
    .split('&')
    .map(s => s.trim())
    .filter(Boolean)
    .map(s => ({ wxid: s }));
}

function loadCache() {
  try {
    return fs.readFileSync(CACHE_FILE, 'utf8')
      .split('\n')
      .map(s => s.trim())
      .filter(Boolean)
      .reduce((acc, line) => {
        const m = line.match(/pt_pin=([^;]+)/);
        if (m) acc[m[1]] = line;
        return acc;
      }, {});
  } catch { return {}; }
}

function saveCache(cache) {
  const lines = Object.values(cache).join('\n');
  fs.writeFileSync(CACHE_FILE, lines + '\n', 'utf8');
}

async function wxPost(paths, body, timeout = 20000) {
  const base = WECHAT_SERVER.replace(/\/$/, '');
  let lastErr;
  for (const p of paths) {
    try {
      const { data } = await axios.post(`${base}${p}`, body, { timeout, headers: { 'Content-Type': 'application/json' } });
      return data;
    } catch (e) {
      lastErr = e;
    }
  }
  throw lastErr || new Error('微信协议请求失败');
}

async function getWxCode(wxid) {
  const data = await wxPost(['/api/v1/wx/app/get/code', '/wx/app/get/code'], { wxid, appid: APPID }, 15000);
  const code = data?.Data?.code || data?.data?.code || '';
  if (!code) throw new Error(`获取 wx code 失败: ${JSON.stringify(data).slice(0, 300)}`);
  return code;
}

async function getEidToken(wxid) {
  const payload = { api_name: 'webapi_getuserinfo', data: { lang: 'zh_CN' }, with_credentials: true };
  try {
    const data = await wxPost(['/api/v1/wx/app/call/function', '/wx/app/call/function'], { wxid, appid: APPID, data: JSON.stringify(payload) }, 15000);
    const inner = data?.Data || data?.data || {};
    const decoded = JSON.parse(Buffer.from(inner.data || '', 'base64').toString('utf8'));
    return decoded.eid_token || decoded.eidToken || decoded.eid || '';
  } catch {
    return '';
  }
}

async function getFingerTk() {
  const now = Date.now();
  const env = {
    sv: '1.0.3.4',
    clist: now,
    vlv: '3.16.0',
    ve: '4.1.8.107',
    fs: -1,
    la: 'zh_CN',
    br: 'microsoft',
    mo: 'microsoft',
    pr: 1,
    pl: 'windows',
    sh: 780,
    sw: 414,
    sbh: '',
    sy: 'Windows 10',
    wh: 780,
    ww: 414,
    bl: '',
    nt: 'wifi',
    vid: APPID,
    bk: FINGER_BIZ_KEY,
    cliet: now,
    fp: randHex(16)
  };
  const resp = await axios.post(`https://we.jd.com/stone/1/${FINGER_TK_DEFAULT}`, fingerEncode(env), {
    timeout: 15000,
    validateStatus: () => true,
    headers: {
      'User-Agent': UA,
      Referer: REFERER,
      'Content-Type': 'application/json',
      Accept: '*/*'
    }
  });
  const tk = resp.data?.data?.tk || resp.data?.tk || '';
  if (!tk) throw new Error(`finger_tk 获取失败: status=${resp.status}, body=${JSON.stringify(resp.data).slice(0, 300)}`);
  return tk;
}

function signSilentAuth(data) {
  const extra = { cmd: 52, sub_cmd: 1, gsalt: SIGN_GSALT };
  const order = ['appid', 'wxappid', 'client_ver', 'ts', 'cmd', 'sub_cmd', 'gsalt'];
  const raw = order.map(k => {
    if (data[k] != null && data[k] !== '') return data[k];
    if (extra[k] != null) return extra[k];
    return '';
  }).join('');
  return md5(raw);
}

async function silentAuthLogin({ code, eidToken = '' }) {
  const ts = Math.floor(Date.now() / 1000);
  const data = {
    globalTokenSource: '',
    code,
    token: '',
    salt: '',
    user_data: '',
    user_iv: '',
    eid_token: eidToken || '',
    goToLogin: true,
    returnurl: '/pages/login/web-view/web-view',
    wxappid: APPID,
    appid: JD_APPID,
    client_ver: CLIENT_VER,
    ts
  };
  data.sign = signSilentAuth(data);
  const body = new URLSearchParams(data).toString();
  const resp = await axios.post('https://wxapplogin.m.jd.com/cgi-bin/jxpp/silentauthlogin', body, {
    timeout: 20000,
    validateStatus: () => true,
    headers: {
      'User-Agent': UA,
      Referer: REFERER,
      'Content-Type': 'application/x-www-form-urlencoded',
      cookie: 'guid=; pt_pin=; pt_key=; pt_token=',
      Accept: '*/*'
    }
  });
  const out = resp.data || {};
  if (resp.status !== 200 || out.err_code !== 0 || !out.pt_key || !out.pt_pin) {
    throw new Error(`silentauthlogin 失败: status=${resp.status} body=${JSON.stringify(out).slice(0, 400)}`);
  }
  return { data: out };
}

async function checkLopCookie(ptKey, ptPin) {
  const resp = await axios.post('https://lop-proxy.jd.com/vip/queryAccountInfo', [{ pin: 'uid' }], {
    timeout: 15000,
    validateStatus: () => true,
    headers: {
      Host: 'lop-proxy.jd.com',
      clientVersion: '1779421844000',
      client: 'WX-XCX',
      requestid: uuid(),
      'LOP-DN': 'logistics-mrd.jd.com',
      'Content-Type': 'application/json;charset=UTF-8',
      Cookie: `pt_key=${ptKey}; pin=${encodeURIComponent(ptPin)};`,
      sessiontraceid: uuid(),
      ClientInfo: JSON.stringify({ appName: 'c2c', client: 'm' }),
      'User-Agent': UA,
      'bff-client': 'MP',
      Referer: REFERER
    }
  });
  return { status: resp.status, data: resp.data };
}

async function refresh(account) {
  const code = await getWxCode(account.wxid);
  let eidToken = await getEidToken(account.wxid);
  if (!eidToken) eidToken = await getFingerTk();
  const { data } = await silentAuthLogin({ code, eidToken });
  const cred = {
    remark: account.remark,
    wxid: account.wxid,
    pt_key: data.pt_key,
    pt_pin: data.pt_pin,
    pin: data.pt_pin,
    guid: data.guid || '',
    expire_time: data.expire_time || 0,
    refresh_time: data.refresh_time || 0,
    rawCk: `pt_key=${data.pt_key};pt_pin=${data.pt_pin};`,
    ck: `wxid:${account.wxid}\ncookie:pt_key=${data.pt_key};pt_pin=${data.pt_pin};`,
    lopCookie: `pt_key=${data.pt_key}; pin=${encodeURIComponent(data.pt_pin)};`,
    updatedAt: Date.now()
  };
  return cred;
}

async function pushToServer() {
  if (!PUSH_API) return;
  let lines;
  try {
    lines = fs.readFileSync(CACHE_FILE, 'utf8').split('\n').map(s => s.trim()).filter(Boolean);
  } catch {
    log('缓存文件不存在，跳过推送');
    return;
  }
  if (lines.length === 0) {
    log('缓存文件为空，跳过推送');
    return;
  }
  log(`开始推送 ${lines.length} 条CK到服务器...`);
  try {
    const form = new URLSearchParams();
    form.append('cks', JSON.stringify(lines));
    form.append('token', PUSH_TOKEN);
    const { data } = await axios.post(PUSH_API, form.toString(), {
      timeout: 60000,
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
    });
    if (data.code === 200) {
      log(`推送完成: ${data.message}`);
    } else {
      log(`推送失败: ${data.message}`);
    }
  } catch (e) {
    log(`推送异常: ${e.message}`);
  }
}

async function main() {
  if (!WXJD) throw new Error('未配置 wxjd（格式：wxid#备注，多号换行/@/&）');
  const accounts = parseAccounts(WXJD);
  const cache = loadCache();
  const results = [];
  for (let i = 0; i < accounts.length; i++) {
    const a = accounts[i];
    try {
      const cred = await refresh(a);
      cache[cred.pt_pin] = cred.rawCk;
      saveCache(cache);
      results.push({ wxid: a.wxid, remark: a.remark, ok: true, ck: cred.ck });
    } catch (e) {
      results.push({ remark: a.remark, ok: false, msg: e.message });
    }
    if (i < accounts.length - 1) await sleep(1500);
  }
  for (const r of results) {
    if (r.ok) console.log(r.ck);
  }
  await pushToServer();
}

main().catch(e => {
  process.stderr.write(e.stack || e.message + '\n');
  process.exit(1);
});