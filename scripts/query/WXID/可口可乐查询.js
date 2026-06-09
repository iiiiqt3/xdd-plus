/**
 * 可口可乐吧 查询本
 * 用法：node 可口可乐查询.js wxid
 * 环境变量（由xdd后台自动传递）：
 *   WECHAT_SERVER: 微信协议服务器地址（旧地址）
 *   WECHAT_SERVER_NEW: 微信协议服务器新地址（可选）
 */

const axios = require('axios');
const fs = require('fs');
const path = require('path');

// ==================== 全局配置 ====================
const CONFIG = {
  APPID: 'wxa5811e0426a94686',
  BASE_URL: 'https://member-api.icoke.cn',
  UA: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.59(0x18003b2e) NetType/4G Language/zh_CN',
  REFERER: 'https://servicewechat.com/wxa5811e0426a94686/499/page-frame.html',
  // 从环境变量获取地址（由xdd后台自动传递）
  WECHAT_SERVER: (process.env.WECHAT_SERVER || 'http://180.152.5.230:8011').trim(),
  WECHAT_SERVER_NEW: (process.env.WECHAT_SERVER_NEW || '').trim(),
  CACHE_FILE: path.join(__dirname, '可口可乐吧_cache.json'),
  TIMEOUT: 15000
};

// ==================== 服务器地址获取 ====================
/**
 * 获取微信协议服务器地址（新旧地址）
 * 优先尝试新地址，失败后尝试旧地址
 */
function getWxServerUrls() {
  const oldUrl = CONFIG.WECHAT_SERVER.replace(/\/+$/, '');
  const newUrl = CONFIG.WECHAT_SERVER_NEW ? CONFIG.WECHAT_SERVER_NEW.replace(/\/+$/, '') : oldUrl;
  return { oldUrl, newUrl };
}

// ==================== 缓存管理 ====================
function loadCache() {
  try {
    if (fs.existsSync(CONFIG.CACHE_FILE)) {
      return JSON.parse(fs.readFileSync(CONFIG.CACHE_FILE, 'utf-8'));
    }
  } catch (e) {}
  return {};
}

function saveCache(data) {
  try {
    fs.writeFileSync(CONFIG.CACHE_FILE, JSON.stringify(data, null, 2), 'utf-8');
  } catch (e) {}
}

// ==================== wxid 换取 Token 流程 ====================

/**
 * Step1: wxid → code（自动尝试新旧地址）
 */
async function getWxCode(wxid) {
  const { oldUrl, newUrl } = getWxServerUrls();
  const urlsToTry = [...new Set([oldUrl, newUrl])]; // 去重

  for (let i = 0; i < urlsToTry.length; i++) {
    const serverUrl = urlsToTry[i];
    if (!serverUrl) continue;
    const label = i === 0 ? '旧地址' : '新地址';
    try {
      const url = `${serverUrl}/api/v1/wx/app/get/code`;
      const response = await axios.post(url, { wxid, appid: CONFIG.APPID }, {
        headers: { 'Content-Type': 'application/json' },
        timeout: CONFIG.TIMEOUT,
        validateStatus: s => s === 200
      });

      const data = response.data;
      const code = data?.Data?.code || data?.data?.code;

      if (code) {
        return code;
      }
      // 业务失败，继续尝试下一个地址
      if (data?.code !== undefined) {
        console.log(`⚠️  ${label} 业务错误: ${data?.Message || data?.msg || JSON.stringify(data)}`);
        continue;
      }
    } catch (e) {
      console.log(`⚠️  ${label} 请求异常: ${e.message}`);
      continue;
    }
  }

  throw new Error(`所有地址均无法获取code`);
}

/**
 * Step2: code → token
 */
async function codeToToken(code) {
  const url = `${CONFIG.BASE_URL}/api/sp-portal/store/icoke/wechat/loginNoCache/${code}`;
  const headers = {
    'Host': 'member-api.icoke.cn',
    'content-type': 'application/json',
    'Accept': 'application/json, text/plain, */*',
    'Accept-Encoding': 'gzip,compress,br,deflate',
    'User-Agent': CONFIG.UA,
    'Referer': CONFIG.REFERER,
  };

  const response = await axios.get(url, { headers, timeout: CONFIG.TIMEOUT, validateStatus: s => s === 200 });
  const body = response.data;

  if (body && body.jwtString) {
    return body.jwtString;
  }
  throw new Error(`换取token失败: ${JSON.stringify(body).substring(0, 200)}`);
}

/**
 * 通过 wxid 获取完整 Token
 */
async function getTokenByWxid(wxid) {
  try {
    const code = await getWxCode(wxid);
    const token = await codeToToken(code);
    return token;
  } catch (e) {
    console.log(`❌ 获取Token失败: ${wxid} -> ${e.message}`);
    return null;
  }
}

// ==================== API 请求封装 ====================
function getHeaders(token) {
  return {
    'Host': 'member-api.icoke.cn',
    'Authorization': token,
    'content-type': 'application/json',
    'Accept': 'application/json, text/plain, */*',
    'Accept-Encoding': 'gzip,compress,br,deflate',
    'User-Agent': CONFIG.UA,
    'Referer': CONFIG.REFERER,
  };
}

async function apiGet(path, token, params) {
  try {
    const res = await axios.get(`${CONFIG.BASE_URL}${path}`, {
      headers: getHeaders(token),
      params,
      timeout: CONFIG.TIMEOUT,
    });
    return res.data;
  } catch (e) {
    if (e.response?.status === 401) {
      return { code: 401, message: "token失效" };
    }
    return null;
  }
}

function isTokenInvalid(data) {
  if (!data) return false;
  const msg = (data.message || data.msg || '').toLowerCase();
  return msg.includes('未登录') || msg.includes('token') || msg.includes('登录') ||
    data.code === '401' || data.code === 401 || data.status === 401;
}

// ==================== 查询接口 ====================

/**
 * 查询用户基本信息
 */
async function queryUserInfo(token) {
  const res = await apiGet('/api/icoke-customer/icoke/mini/customer/main/base/info', token);
  if (!res || isTokenInvalid(res)) return null;
  return {
    name: res.name || '未知',
    mobile: res.mobile || '',
    grade: res.grade || '',
    phone: res.phone || ''
  };
}

/**
 * 查询积分
 */
async function queryPoints(token) {
  const res = await apiGet('/api/icoke-customer/icoke/mini/customer/main/points', token);
  if (!res || isTokenInvalid(res)) return null;
  return {
    point: res.point || 0,
    experiencePoints: res.experiencePoints || 0,
    frozenPoint: res.frozenPoint || 0
  };
}

/**
 * 查询签到记录
 */
async function querySignRecords(token) {
  const res = await apiGet('/api/icoke-sign/icoke/mini/sign/main/getSignOutline', token);
  if (!res || isTokenInvalid(res)) return null;

  const today = new Date();
  const y = today.getFullYear(), m = today.getMonth() + 1, d = today.getDate();
  const list = res.data || [];

  // 今日是否签到
  const todayRecord = list.find(r => r.year === y && r.month === m && r.day === d);
  // 本月签到天数
  const monthDays = list.filter(r => r.year === y && r.month === m && r.exist).length;
  // 连续签到天数
  let continuousDays = 0;
  const sortedList = [...list].sort((a, b) => {
    return new Date(b.year, b.month - 1, b.day) - new Date(a.year, a.month - 1, a.day);
  });
  for (const r of sortedList) {
    if (r.exist) continuousDays++;
    else break;
  }

  return {
    todaySigned: !!(todayRecord && todayRecord.exist),
    todayPoint: todayRecord?.point || 0,
    monthDays,
    continuousDays,
    totalRecords: list.filter(r => r.exist).length
  };
}

/**
 * 查询积分商城信息
 */
async function queryMallInfo(token) {
  const res = await apiGet('/api/icoke-pointmall/icoke/mini/point/mall/main/home', token);
  if (!res || isTokenInvalid(res)) return null;
  return {
    banners: res.banners || [],
    categories: res.categories || []
  };
}

// ==================== 查询执行 ====================

/**
 * 用 Token 尝试查询，成功返回 true
 */
async function tryQueryWithToken(token) {
  try {
    const [userInfo, points, signRecords] = await Promise.all([
      queryUserInfo(token),
      queryPoints(token),
      querySignRecords(token)
    ]);

    if (!userInfo) return false;

    // 手机号打码
    let mobile = userInfo.mobile || userInfo.phone || '未知';
    if (mobile.length === 11) {
      mobile = mobile.substring(0, 3) + '****' + mobile.substring(7);
    }

    console.log(`[👤] 昵称：${userInfo.name}`);
    console.log(`[📱] 手机：${mobile}`);
    console.log(`[🏆] 会员等级：${userInfo.grade || '普通会员'}`);
    console.log(`[💰] 可用积分：${points?.point || 0} 分`);
    console.log(`[🧊] 冻结积分：${points?.frozenPoint || 0} 分`);
    console.log(`[📈] 经验值：${points?.experiencePoints || 0}`);
    console.log(`[📅] 本月签到：${signRecords?.monthDays || 0} 天`);
    console.log(`[🔥] 连续签到：${signRecords?.continuousDays || 0} 天`);
    console.log(`[✅] 今日签到：${signRecords?.todaySigned ? '已签到 +' + (signRecords?.todayPoint || 0) + '分' : '未签到'}`);
    console.log(`[📊] 累计签到：${signRecords?.totalRecords || 0} 天`);

    return true;
  } catch (e) {
    return false;
  }
}

/**
 * 构建所有账号的 Token 并查询（优先缓存，失败重新换取）
 */
async function buildAccountTokensAndQuery(wxList) {
  if (!wxList || wxList.length === 0) {
    wxList = getWxListFromEnv();
  }
  if (wxList.length === 0) {
    console.log(`❌ 未配置账号，请设置环境变量 ${ENV_WXID}=wxid_xxx`);
    return 0;
  }

  const cache = loadCache();
  const newCache = {};
  let successCount = 0;

  for (let i = 0; i < wxList.length; i++) {
    const wxid = wxList[i];
    console.log(`\n==================================`);
    console.log(`👤 [账号${i + 1}]`);

    const cacheItem = cache[wxid] || {};
    let token = cacheItem.token || '';
    let queryOk = false;

    // 1️⃣ 优先读取缓存Token查询
    if (token) {
      console.log('📦 读取缓存Token');
      queryOk = await tryQueryWithToken(token);
      if (queryOk) {
        successCount++;
        newCache[wxid] = {
          token: token,
          update_time: Math.floor(Date.now() / 1000)
        };
        continue;
      }
      console.log('🔄 缓存Token失效，重新换取...');
    }

    // 2️⃣ 缓存失败或无缓存，重新换取Token
    token = await getTokenByWxid(wxid);
    if (!token) {
      console.log(`❌ 跳过账号: ${wxid}`);
      continue;
    }

    // 3️⃣ 用新Token查询
    queryOk = await tryQueryWithToken(token);
    if (queryOk) {
      successCount++;
      newCache[wxid] = {
        token: token,
        update_time: Math.floor(Date.now() / 1000)
      };
    } else {
      console.log('❌ 新Token查询失败');
      newCache[wxid] = {
        token: token,
        update_time: Math.floor(Date.now() / 1000)
      };
    }
  }

  saveCache(newCache);
  return successCount;
}

// ==================== 主函数 ====================
async function main() {
  // 从命令行参数获取 wxid
  const argvRaw = process.argv.slice(2).join(' ');
  const wxList = argvRaw.replace(/&/g, '\n').split('\n').map(l => l.trim()).filter(l => l);

  if (wxList.length === 0) {
    console.log('❌ 请通过命令行传入 wxid');
    process.exit(1);
  }

  const count = await buildAccountTokensAndQuery(wxList);

  console.log(`\n==================================`);
  if (count === 0) {
    console.log(`[❌] 所有账号查询失败`);
    process.exit(1);
  } else {
    console.log(`[✅] 成功查询 ${count} 个账号 - ${new Date().toLocaleString()}`);
  }
}

main().catch(err => {
  console.log(`[❌] 脚本执行异常：${err.message}`);
  process.exit(1);
});