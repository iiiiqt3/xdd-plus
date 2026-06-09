const axios = require('axios');
const crypto = require('crypto');
const fs = require('fs');
const path = require('path');

// ==================== 全局配置 ====================
const CONFIG = {
    TOKEN: 'zeTLTYeG0bLetfRk',
    SYS_CODE: 'MCS-MIMP-CORE',
    HOST: 'mcs-mimp-web.sf-express.com',
    USER_AGENT: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 mediaCode=SFEXPRESSAPP-iOS-ML',
    PLATFORM: 'SFAPP',
    CHANNEL: 'apppart',
    TIMEOUT: 15000
};

// 微信协议配置（同步PY脚本）
const WX_CONFIG = {
    // 从环境变量获取地址（由xdd后台自动传递）
    WECHAT_SERVER: (process.env.WECHAT_SERVER || 'http://180.152.5.230:8011').trim(),
    WECHAT_SERVER_NEW: (process.env.WECHAT_SERVER_NEW || '').trim(),
    APPID: 'wxd4185d00bf7e08ac',
    PUBLIC_ID: 'gh_f9d9fca26a50',
    UCMP_BASE: 'https://ucmp.sf-express.com',
    UCMP_SIGN_APPID: 'wxapp-valid-0328',
    UCMP_SIGN_KEY: '2b08f7f6bf564a1dada1570535fd44ba',
    UA: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13) UnifiedPCWindowsWechat(0xf254186b) XWEB/19481',
    CACHE_FILE: path.join(__dirname, 'wxsf2.json')
};

// 环境变量名
const ENV_WXID = 'WXID_SF';
const ENV_CK = 'sfsyUrl';

// 禁用HTTPS警告（静默处理）
process.env.NODE_TLS_REJECT_UNAUTHORIZED = '0';
process.removeAllListeners('warning');

// ==================== 缓存管理 ====================
function loadCache() {
    const cachePath = WX_CONFIG.CACHE_FILE;
    if (fs.existsSync(cachePath)) {
        try {
            return JSON.parse(fs.readFileSync(cachePath, 'utf-8'));
        } catch (e) {
            return {};
        }
    }
    return {};
}

function saveCache(data) {
    const cachePath = WX_CONFIG.CACHE_FILE;
    fs.writeFileSync(cachePath, JSON.stringify(data, null, 2), 'utf-8');
}

// ==================== 服务器地址获取 ====================
// 缓存获取到的服务器地址

/**
 * 获取微信协议服务器地址（新旧地址）
 */
function getWxServerUrls() {
    const oldUrl = WX_CONFIG.WECHAT_SERVER.replace(/\/+$/, '');
    const newUrl = WX_CONFIG.WECHAT_SERVER_NEW ? WX_CONFIG.WECHAT_SERVER_NEW.replace(/\/+$/, '') : oldUrl;
    return { oldUrl, newUrl };
}

// ==================== UCMP 签名工具 ====================
function md5(str, upper = false) {
    const hash = crypto.createHash('md5').update(str).digest('hex');
    return upper ? hash.toUpperCase() : hash;
}

function randomString(n = 32) {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < n; i++) {
        result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
}

function randomMsgId() {
    return 'WA' + md5(Date.now().toString(), true);
}

function randomDeviceId() {
    return `${Date.now()}-${Math.floor(1000000 + Math.random() * 9000000)}`;
}

function getUcmpSign(body) {
    const nonce = randomString(32);
    const bodyStr = body ? JSON.stringify(body) : '';
    const raw = `appId=${WX_CONFIG.UCMP_SIGN_APPID}&nonceStr=${nonce}&requestBody=${bodyStr}&key=${WX_CONFIG.UCMP_SIGN_KEY}`;
    return {
        appId: WX_CONFIG.UCMP_SIGN_APPID,
        nonceStr: nonce,
        sign: md5(raw, true)
    };
}

function buildUcmpHeaders(body = {}, sessionId = '') {
    const s = getUcmpSign(body);
    const headers = {
        'User-Agent': WX_CONFIG.UA,
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'appId': s.appId,
        'nonceStr': s.nonceStr,
        'sign': s.sign,
        'wxapp-version': 'V17.58',
        'deviceId': randomDeviceId(),
        'msgid': randomMsgId()
    };
    if (sessionId) {
        headers['suuid'] = sessionId;
    }
    return headers;
}

// ==================== wxid 换取 Cookie 流程 ====================

/**
 * Step1: wxid → code（自动尝试新旧地址）
 */
async function getWxCode(wxid) {
    const { oldUrl, newUrl } = await getWxServerUrls();
    const urlsToTry = [...new Set([oldUrl, newUrl])]; // 去重

    for (const serverUrl of urlsToTry) {
        if (!serverUrl) continue;
        try {
            const url = `${serverUrl}/api/v1/wx/app/get/code`;
            const response = await axios.post(url, { wxid, appid: WX_CONFIG.APPID }, {
                headers: { 'Content-Type': 'application/json' },
                timeout: CONFIG.TIMEOUT,
                validateStatus: s => s === 200
            });

            const data = response.data;
            const code = data?.Data?.code || data?.data?.code;

            if (code) {
                return code;
            }
            // 如果是明确的业务失败，不继续尝试
            if (data?.code !== undefined) {
                throw new Error(`获取code失败: ${JSON.stringify(data)}`);
            }
        } catch (e) {
            if (e.message.includes('获取code失败')) throw e;
            console.log(`⚠️  地址 ${serverUrl} 请求异常: ${e.message}`);
            continue;
        }
    }

    throw new Error(`所有地址均无法获取code`);
}

/**
 * Step2: code → sessionId (UCMP登录)
 */
async function ucmpAppOnLogin(code) {
    const url = `${WX_CONFIG.UCMP_BASE}/wxaccess/weixin/appOnLogin?code=${code}&publicId=${WX_CONFIG.PUBLIC_ID}`;
    const headers = buildUcmpHeaders();

    const response = await axios.get(url, { headers, timeout: CONFIG.TIMEOUT, validateStatus: s => s === 200 });
    const data = response.data;

    if (data && data.sessionId) {
        return data;
    }

    throw new Error(`登录失败: ${JSON.stringify(data)}`);
}

/**
 * Step3: sessionId → memId + mobile
 */
async function ucmpWxMemIsBind(sessionId) {
    const url = `${WX_CONFIG.UCMP_BASE}/wxopen/weixin/wxMemIsBind`;
    const headers = buildUcmpHeaders({}, sessionId);

    const response = await axios.post(url, {}, { headers, timeout: CONFIG.TIMEOUT, validateStatus: s => s === 200 });
    const data = response.data;

    if (!data || typeof data !== 'object') {
        throw new Error(`接口异常: ${JSON.stringify(data)}`);
    }

    const obj = data.obj || {};
    const memId = obj.memId;
    const mobile = obj.mobile;

    if (!memId || !mobile) {
        throw new Error(`未绑定账号: ${JSON.stringify(data)}`);
    }

    return { memId, mobile };
}

/**
 * Step4: 初始化 MCS 登录态（手动跟踪重定向，收集所有 Set-Cookie）
 */
async function initMcsSession(cookies) {
    const bizCode = {
        path: '/up-member/newPoints',
        linkCode: 'SFAC20230803190840424',
        supportShare: 'YES',
        subCategoryCode: '1',
        from: 'mypoint',
        categoryCode: '1'
    };

    const url = `${WX_CONFIG.UCMP_BASE}/wechat-act/weixin/activity/sfnewactivity`;
    const params = {
        bizCode: JSON.stringify(bizCode),
        regSource: 'mypoint',
        'wxapp-version': 'V17.58',
        suuid: cookies.sessionId || ''
    };

    const baseHeaders = {
        'User-Agent': WX_CONFIG.UA,
        'Accept': '*/*'
    };

    let currentUrl = url;
    let redirectCount = 0;
    const maxRedirects = 10;

    while (redirectCount < maxRedirects) {
        const cookieStr = Object.entries(cookies).map(([k, v]) => `${k}=${v}`).join('; ');
        const headers = { ...baseHeaders, 'Cookie': cookieStr };

        const response = await axios.get(currentUrl, {
            params: redirectCount === 0 ? params : undefined,
            headers,
            timeout: CONFIG.TIMEOUT,
            validateStatus: s => s < 400,
            maxRedirects: 0
        });

        const setCookies = response.headers['set-cookie'];
        if (setCookies) {
            const cookieArr = Array.isArray(setCookies) ? setCookies : [setCookies];
            cookieArr.forEach(cookieLine => {
                const main = cookieLine.split(';')[0].trim();
                if (main.includes('=')) {
                    const eqIdx = main.indexOf('=');
                    const k = main.substring(0, eqIdx).trim();
                    const v = main.substring(eqIdx + 1).trim();
                    if (k && v) {
                        cookies[k] = v;
                    }
                }
            });
        }

        const location = response.headers['location'];
        if (location && (response.status === 301 || response.status === 302 || response.status === 303 || response.status === 307 || response.status === 308)) {
            redirectCount++;
            if (location.startsWith('/')) {
                const parsedUrl = new URL(currentUrl);
                currentUrl = `${parsedUrl.protocol}//${parsedUrl.host}${location}`;
            } else if (location.startsWith('http')) {
                currentUrl = location;
            } else {
                const parsedUrl = new URL(currentUrl);
                currentUrl = `${parsedUrl.protocol}//${parsedUrl.host}/${location}`;
            }
            continue;
        }

        break;
    }

    return cookies;
}

/**
 * 通过 wxid 获取完整 Cookie（wxid → code → sessionId → mobile → MCS登录态）
 */
async function getCookieByWxid(wxid) {
    try {
        const code = await getWxCode(wxid);
        const loginData = await ucmpAppOnLogin(code);
        const sessionId = loginData.sessionId;
        const { memId, mobile } = await ucmpWxMemIsBind(sessionId);

        const cookies = {
            sessionId: sessionId,
            JSESSIONID: sessionId,
            _login_user_id_: memId,
            _login_mobile_: mobile
        };

        await initMcsSession(cookies);

        const cookieStr = Object.entries(cookies).map(([k, v]) => `${k}=${v}`).join(';');
        return cookieStr;
    } catch (e) {
        console.log(`❌ 获取Cookie失败: ${wxid} -> ${e.message}`);
        return null;
    }
}

/**
 * 从环境变量解析 wxid 列表
 * 格式：备注#wxid，多账号用 & 或换行分隔
 */
function getWxListFromEnv() {
    const raw = (process.env[ENV_WXID] || '').trim();
    if (!raw) {
        return [];
    }

    const accounts = [];
    const lines = raw.replace(/&/g, '\n').split('\n').map(l => l.trim()).filter(l => l);

    for (const line of lines) {
        let wxid, remark;
        if (line.includes('#')) {
            const idx = line.indexOf('#');
            remark = line.substring(0, idx).trim();
            wxid = line.substring(idx + 1).trim();
        } else {
            wxid = line.trim();
            remark = wxid;
        }

        if (wxid) {
            accounts.push({ wxid, remark });
        }
    }

    return accounts;
}

/**
 * 构建所有账号的 Cookie 列表并查询（优先缓存，失败重新换取）
 * @param {Array} wxList - 可选，外部传入的 wxid 列表；为空则从环境变量读取
 * @returns {number} 成功查询的账号数
 */
async function buildAccountCookiesAndQuery(wxList) {
    if (!wxList || wxList.length === 0) {
        wxList = getWxListFromEnv();
    }
    if (wxList.length === 0) {
        return 0;
    }

    const cache = loadCache();
    const newCache = {};
    let successCount = 0;

    for (const item of wxList) {
        const { wxid, remark } = item;
        console.log(`\n==================================`);
        console.log(`👤 [${remark}]`);

        const cacheItem = cache[wxid] || {};
        let cookie = cacheItem.cookie || '';
        let queryOk = false;

        // 1️⃣ 优先读取缓存Token查询
        if (cookie) {
            console.log('📦 读取缓存Token');
            queryOk = await tryQueryWithCookie(cookie);
            if (queryOk) {
                successCount++;
                // 缓存有效，写回缓存
                newCache[wxid] = {
                    cookie: cookie,
                    update_time: Math.floor(Date.now() / 1000)
                };
                continue;
            }
            console.log('🔄 缓存Token失效，重新换取...');
        }

        // 2️⃣ 缓存失败或无缓存，重新换取Token
        cookie = await getCookieByWxid(wxid);
        if (!cookie) {
            console.log(`❌ 跳过账号: ${wxid}`);
            continue;
        }

        // 3️⃣ 用新Token查询
        queryOk = await tryQueryWithCookie(cookie);
        if (queryOk) {
            successCount++;
            newCache[wxid] = {
                cookie: cookie,
                update_time: Math.floor(Date.now() / 1000)
            };
        } else {
            console.log('❌ 新Token查询失败');
            // 仍然缓存，下次可能恢复
            newCache[wxid] = {
                cookie: cookie,
                update_time: Math.floor(Date.now() / 1000)
            };
        }
    }

    saveCache(newCache);
    return successCount;
}

/**
 * 用 Cookie 尝试查询用户信息，成功返回 true 并输出结果
 */
async function tryQueryWithCookie(cookieStr) {
    try {
        const cookies = parseCookies(cookieStr);
        if (!cookies.sessionId || !cookies._login_mobile_) {
            return false;
        }

        const [userInfo, pointDetail, pointMonthly] = await Promise.all([
            queryUserInfo(cookies),
            queryPointDetail(cookies),
            queryPointMonthly(cookies)
        ]);

        if (!userInfo) {
            return false;
        }

        console.log(`[👤] 手机号：${userInfo.mobile}`);
        console.log(`[🏆] 会员等级：${userInfo.gradeName}（${userInfo.gradeVal}）`);
        console.log(`[📈] 成长值：${userInfo.expVal} | 注册时间：${userInfo.regTm}`);
        console.log(`[💰] 可用积分：${userInfo.usablePoint} 分`);
        console.log(`[⚠️] 即将过期积分：${userInfo.leavePoint} 分`);
        console.log(`[📅] 积分过期时间：${userInfo.pointClearCycle}`);
        console.log(`[✨] 今日新增积分：${pointDetail.todayPoints} 分`);
        console.log(`[📆] 本月新增积分：${pointMonthly.currentMonth.addSum} 分 | 本月扣除：${pointMonthly.currentMonth.subSum} 分`);
        console.log(`[💥] 累计过期/扣除积分：${pointMonthly.expirePoints} 分`);
        console.log(`[🕙] 最近积分变动：${pointDetail.latestPointTime}`);
        console.log(`[📊] 累计积分变动次数：${pointDetail.totalCount} 次`);
        console.log(`[🔑] 最后登录时间：${userInfo.appLastLoginTm}`);
        return true;
    } catch (e) {
        return false;
    }
}

// ==================== 原有查询逻辑 ====================

/**
 * 生成签名
 */
function generateSign(isActivityInterface = false) {
    const timestamp = Date.now().toString();
    const signStr = `token=${CONFIG.TOKEN}&timestamp=${timestamp}&sysCode=${CONFIG.SYS_CODE}`;
    const signature = crypto.createHash('md5').update(signStr).digest('hex');

    if (isActivityInterface) {
        return { syscode: CONFIG.SYS_CODE, timestamp, signature };
    }
    return { sysCode: CONFIG.SYS_CODE, timestamp, signature };
}

/**
 * 初始化请求头
 */
function initHeaders(cookies, isActivityInterface = false) {
    const signData = generateSign(isActivityInterface);
    const cookieStr = Object.entries(cookies).map(([k, v]) => `${k}=${v}`).join('; ');

    const headers = {
        'Host': CONFIG.HOST,
        'Accept': 'application/json, text/plain, */*',
        'Content-Type': 'application/json;charset=UTF-8',
        'Cookie': cookieStr,
        'Origin': `https://${CONFIG.HOST}`,
        ...signData
    };

    if (isActivityInterface) {
        headers['channel'] = 'xcxpart';
        headers['platform'] = 'MINI_PROGRAM';
        headers['User-Agent'] = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36 MicroMessenger/7.0.20.1781(0x6700143B) NetType/WIFI MiniProgramEnv/Windows WindowsWechat/WMPF WindowsWechat(0x63090a13) UnifiedPCWindowsWechat(0xf254173b) XWEB/19027';
        headers['Referer'] = `https://${CONFIG.HOST}/origin/a/mimp-activity/anniversary2026?mobile=${cookies._login_mobile_ || ''}`;
    } else {
        headers['channel'] = CONFIG.CHANNEL;
        headers['sysCode'] = CONFIG.SYS_CODE;
        headers['platform'] = CONFIG.PLATFORM;
        headers['deviceId'] = 'D2K_E-UR85-_pfEZacHGMGRJfE9f2YdSploYqpdjCCUncX7c';
        headers['User-Agent'] = CONFIG.USER_AGENT;
        headers['Referer'] = `https://${CONFIG.HOST}/origin/a/mimp-activity/anniversary2026?mobile=${cookies._login_mobile_ || ''}`;
    }

    return headers;
}

/**
 * 解析CK字符串（兼容GO传递的多格式CK）
 */
function parseCookies(input) {
    const cookies = {};
    try {
        const decoded = decodeURIComponent(input).trim();
        decoded.split(';').forEach(item => {
            item = item.trim();
            if (item.includes('=')) {
                const eqIdx = item.indexOf('=');
                const k = item.substring(0, eqIdx).trim();
                const v = item.substring(eqIdx + 1).trim();
                if (k) cookies[k] = v;
            }
        });
        if (!cookies.JSESSIONID) cookies.JSESSIONID = cookies.sessionId || '';
        if (!cookies._login_user_id_) cookies._login_user_id_ = 'default_' + Date.now();
    } catch (e) {
        console.log(`[❌] Cookie解析失败：${e.message}`);
    }
    return cookies;
}

/**
 * 手机号打码
 */
function maskMobile(mobile) {
    if (!mobile || mobile.length !== 11) return '未知';
    return mobile.substring(0, 3) + '****' + mobile.substring(7);
}

/**
 * 查询用户核心信息
 */
async function queryUserInfo(cookies) {
    try {
        const url = `https://${CONFIG.HOST}/mcs-mimp/commonPost/~memberIntegral~userInfoService~queryUserInfo`;
        const headers = initHeaders(cookies, false);
        const requestData = {
            sysCode: "ESG-CEMP-CORE",
            optionalColumns: ["usablePoint", "cycleSub", "leavePoint", "pointClearCycle"],
            token: CONFIG.TOKEN
        };
        const response = await axios.post(url, requestData, { headers, timeout: CONFIG.TIMEOUT, validateStatus: s => s === 200 });
        const res = response.data;
        if (res.success && res.obj) {
            const mobile = cookies._login_mobile_ || '未知';
            return {
                mobile: maskMobile(mobile),
                usablePoint: res.obj.usablePoint || 0,
                leavePoint: res.obj.leavePoint || 0,
                pointClearCycle: res.obj.pointClearCycle || '无',
                gradeName: res.obj.gradeName || '普通会员',
                gradeVal: res.obj.gradeVal || 'V0',
                expVal: res.obj.expVal || 0,
                regTm: res.obj.regTm || '未知',
                appLastLoginTm: res.obj.appLastLoginTm || '未知'
            };
        }
        throw new Error(res.errorMessage || '用户信息查询失败');
    } catch (e) {
        return null;
    }
}

/**
 * 查询积分明细
 */
async function queryPointDetail(cookies) {
    try {
        const url = `https://${CONFIG.HOST}/mcs-mimp/commonPost/~memberIntegral~memberPoint~queryMemberPointDetailInfo`;
        const headers = initHeaders(cookies, false);
        const data = { type: 'ADD', pageNo: 1, pageSize: 10 };
        const response = await axios.post(url, data, { headers, timeout: CONFIG.TIMEOUT, validateStatus: s => s === 200 });
        const res = response.data;

        const today = new Date().toISOString().split('T')[0].replace(/-/g, '');
        let todayPoints = 0, latestPointTime = '无';
        if (res.success && res.obj?.data) {
            res.obj.data.forEach(item => {
                const createDate = item.createTm.split(' ')[0].replace(/-/g, '');
                if (createDate === today) todayPoints += Number(item.pointVal) || 0;
                if (!latestPointTime) latestPointTime = item.createTm;
            });
        }
        return { todayPoints, latestPointTime, totalCount: res.obj?.totalCount || 0 };
    } catch (e) {
        return { todayPoints: 0, latestPointTime: '无', totalCount: 0 };
    }
}

/**
 * 查询月度积分统计
 */
async function queryPointMonthly(cookies) {
    try {
        const url = `https://${CONFIG.HOST}/mcs-mimp/commonPost/~memberIntegral~memberPoint~queryPointCountMonthly`;
        const headers = initHeaders(cookies, false);
        const response = await axios.post(url, {}, { headers, timeout: CONFIG.TIMEOUT, validateStatus: s => s === 200 });
        const res = response.data;

        let monthlyData = [], expirePoints = 0;
        if (res.success && res.obj) {
            monthlyData = res.obj;
            expirePoints = res.obj.reduce((sum, item) => sum + (item.subSum || 0), 0);
        }
        return {
            expirePoints,
            currentMonth: monthlyData[0] || { time: '无', addSum: 0, subSum: 0 }
        };
    } catch (e) {
        return { expirePoints: 0, currentMonth: { time: '无', addSum: 0, subSum: 0 } };
    }
}

// ==================== 主函数 ====================
async function main() {
    // 🔥 优先使用环境变量 WXID_SF 或 命令行传入的 wxid（wxid模式）
    let wxList = getWxListFromEnv();

    // 命令行参数补充 wxid
    if (wxList.length === 0 && process.argv.length > 2) {
        const argvRaw = process.argv.slice(2).join(' ');
        // 不含 = 号视为 wxid 模式
        const isWxidMode = !argvRaw.includes('=');
        if (isWxidMode && argvRaw.trim()) {
            const raw = argvRaw.replace(/&/g, '\n').replace(/\s+/g, '\n');
            const lines = raw.split('\n').map(l => l.trim()).filter(l => l);
            for (const line of lines) {
                let wxid, remark;
                if (line.includes('#')) {
                    const idx = line.indexOf('#');
                    remark = line.substring(0, idx).trim();
                    wxid = line.substring(idx + 1).trim();
                } else {
                    wxid = line.trim();
                    remark = wxid;
                }
                if (wxid) {
                    wxList.push({ wxid, remark });
                }
            }
        }
    }

    if (wxList.length > 0) {
        // wxid模式：优先缓存 → 失败重新换取 → 查询输出
        const count = await buildAccountCookiesAndQuery(wxList);
        if (count === 0) {
            console.log(`\n[❌] 所有账号查询失败`);
            process.exit(1);
        }
    } else {
        // 兼容旧模式：命令行传入 CK 字符串 或 环境变量 sfsyUrl
        let accounts = [];
        let ckParam = '';

        if (process.env[ENV_CK]) {
            ckParam = process.env[ENV_CK];
        } else if (process.argv.length > 2) {
            const argvParts = process.argv.slice(2);

            if (argvParts.length > 1) {
                ckParam = argvParts.join(';');
            } else if (argvParts[0].startsWith('@')) {
                const filePath = argvParts[0].substring(1);
                try {
                    ckParam = fs.readFileSync(filePath, 'utf-8').trim();
                } catch (e) {
                    console.log(`[❌] 读取CK文件失败: ${filePath}`);
                    process.exit(1);
                }
            } else {
                ckParam = argvParts[0];
            }
        }

        if (ckParam) {
            let decodedParam = ckParam;
            try {
                const testDecode = decodeURIComponent(ckParam);
                if (testDecode.includes('sessionId') || testDecode.includes('_login_mobile_')) {
                    decodedParam = testDecode;
                }
            } catch (e) { /* 解码失败就用原始值 */ }

            if (decodedParam.includes('||')) {
                accounts = decodedParam.split('||').filter(item => item.trim());
            } else if (decodedParam.includes('&') && !decodedParam.includes('sessionId=')) {
                accounts = decodedParam.split('&').filter(item => item.trim());
            } else {
                accounts = [decodedParam.trim()];
            }

            for (let i = 0; i < accounts.length; i++) {
                console.log(`\n==================================`);
                const cookies = parseCookies(accounts[i].trim());
                if (!cookies.sessionId || !cookies._login_mobile_) {
                    console.log(`[❌] 第${i + 1}个账号 - 登录失效`);
                    continue;
                }

                const [userInfo, pointDetail, pointMonthly] = await Promise.all([
                    queryUserInfo(cookies),
                    queryPointDetail(cookies),
                    queryPointMonthly(cookies)
                ]);

                if (!userInfo) {
                    console.log(`[❌] 第${i + 1}个账号 - 查询失败`);
                    continue;
                }

                console.log(`[👤] 手机号：${userInfo.mobile}`);
                console.log(`[🏆] 会员等级：${userInfo.gradeName}（${userInfo.gradeVal}）`);
                console.log(`[📈] 成长值：${userInfo.expVal} | 注册时间：${userInfo.regTm}`);
                console.log(`[💰] 可用积分：${userInfo.usablePoint} 分`);
                console.log(`[⚠️] 即将过期积分：${userInfo.leavePoint} 分`);
                console.log(`[📅] 积分过期时间：${userInfo.pointClearCycle}`);
                console.log(`[✨] 今日新增积分：${pointDetail.todayPoints} 分`);
                console.log(`[📆] 本月新增积分：${pointMonthly.currentMonth.addSum} 分 | 本月扣除：${pointMonthly.currentMonth.subSum} 分`);
                console.log(`[💥] 累计过期/扣除积分：${pointMonthly.expirePoints} 分`);
                console.log(`[🕙] 最近积分变动：${pointDetail.latestPointTime}`);
                console.log(`[📊] 累计积分变动次数：${pointDetail.totalCount} 次`);
                console.log(`[🔑] 最后登录时间：${userInfo.appLastLoginTm}`);

                if (i < accounts.length - 1) await new Promise(res => setTimeout(res, 1000));
            }
        } else {
            console.log(`[❌] 未传入任何账号信息，请使用以下任一方式：`);
            console.log(`   方式1 - 环境变量wxid: set ${ENV_WXID}=备注#wxid_xxx&备注2#wxid_yyy && node sfsy.js`);
            console.log(`   方式2 - 环境变量CK: set ${ENV_CK}=sessionId=xxx;_login_mobile_=xxx && node sfsy.js`);
            console.log(`   方式3 - 命令行wxid: node sfsy.js 备注#wxid_xxx`);
            console.log(`   方式4 - 从文件读取CK: node sfsy.js @ck.txt`);
            process.exit(1);
        }
    }

    console.log(`\n==================================`);
    console.log(`[✅] 所有账号查询完成 - ${new Date().toLocaleString()}`);
}

main().catch(err => {
    console.log(`[❌] 脚本执行异常：${err.message}`);
    process.exit(1);
});
