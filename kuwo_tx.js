/**
 * 酷我Music提现插件 - JavaScript版本
 * 原Python版本转换
 * 手机号#密码方式登录
 */

const crypto = require('crypto');
const https = require('https');
const http = require('http');
const { URL } = require('url');
const zlib = require('zlib');

// 全局变量
let today_date = null;
let today_time = null;
let KuwoTXmoney = null;
let KuwoTXcoin = null;
let proxy_manager = null;
let withdraw_delay = 0.0;
let _time_offset = null;

// 安卓设备列表
const _ANDROID_DEVICES = [
  ['Pixel 8 Pro', 'AP4A.250405.002'],
  ['Pixel 7', 'AP2A.240805.005'],
  ['Pixel 9', 'AD4A.250605.001'],
  ['SM-S9280', 'UP1A.231005.007'],
  ['SM-S9110', 'UP1A.231005.007'],
  ['SM-A5560', 'TP1A.220624.014'],
  ['2211133C', 'TKQ1.220829.002'],
  ['23127PN0CC', 'UKQ1.231003.002'],
  ['2407FPN8EC', 'VKQ1.240610.001'],
  ['24122RKC7C', 'BP2A.250605.031'],
  ['V2329A', 'UP1A.231005.007'],
  ['V2336A', 'TP1A.220624.014'],
  ['PHZ110', 'TP1A.220905.001'],
  ['PJZ110', 'UKQ1.240118.001'],
  ['RMX3820', 'TP1A.220905.001'],
  ['LE2120', 'SKQ1.211006.001'],
  ['NE2210', 'TP1A.220905.001'],
  ['22081212C', 'V417IR.240305.001']
];

const _ANDROID_VERSIONS = [12, 13, 14, 15, 16];

const _CHROME_VERSIONS = [
  '120.0.6099.230', '122.0.6261.95', '124.0.6367.113', '126.0.6478.122',
  '128.0.6613.88', '130.0.6723.107', '133.0.6943.137', '136.0.7103.60',
  '140.0.7241.98', '144.0.7564.45', '146.0.7688.100', '148.0.7778.120'
];

const _phone_ua_cache = {};

/**
 * 代理管理器类
 */
class ProxyManager {
  constructor(proxy_api) {
    this.proxy_api = proxy_api;
    this.proxy_cache = {};
    this._proxy_pool = [];
    this.last_error = '';
    this._fatal_proxy_error = false;
  }

  _mask_proxy(proxy) {
    if (!proxy.includes('@')) return proxy;
    const parts = proxy.split('@');
    return `***@${parts[parts.length - 1]}`;
  }

  _valid_port(port) {
    const p = parseInt(port);
    return !isNaN(p) && p > 0 && p <= 65535;
  }

  _split_host_port(proxy) {
    const target = proxy.includes('@') ? proxy.split('@').pop().trim() : proxy;
    if (target.startsWith('[')) {
      const end = target.indexOf(']');
      if (end > 0 && target[end + 1] === ':') {
        const port = target.substring(end + 2);
        if (this._valid_port(port)) {
          return [target.substring(1, end), parseInt(port)];
        }
      }
      return [null, null];
    }
    if (!target.includes(':')) return [null, null];
    const lastColon = target.lastIndexOf(':');
    const host = target.substring(0, lastColon);
    const port = target.substring(lastColon + 1);
    if (host && this._valid_port(port)) {
      return [host, parseInt(port)];
    }
    return [null, null];
  }

  _normalize_proxy(raw_proxy) {
    let proxy = (raw_proxy || '').trim().replace(/^["',;]+|["',;]+$/g, '');
    if (!proxy) return null;

    if (proxy.includes('://')) {
      try {
        const parsed = new URL(proxy);
        const host = parsed.hostname;
        const port = parsed.port;
        if (host && port && this._valid_port(port)) {
          let auth = '';
          if (parsed.username) {
            const username = encodeURIComponent(decodeURIComponent(parsed.username));
            let password = '';
            if (parsed.password) {
              password = ':' + encodeURIComponent(decodeURIComponent(parsed.password));
            }
            auth = `${username}${password}@`;
          }
          const host_text = host.includes(':') && !host.startsWith('[') ? `[${host}]` : host;
          return `${auth}${host_text}:${port}`;
        }
      } catch (e) {
        return null;
      }
    }

    proxy = proxy.replace(/\s+/g, '');

    if (!proxy.includes('@')) {
      const parts = proxy.split(':');
      if (parts.length >= 4 && this._valid_port(parts[1])) {
        const host = parts[0];
        const port = parts[1];
        const username = encodeURIComponent(decodeURIComponent(parts[2]));
        const password = encodeURIComponent(decodeURIComponent(parts.slice(3).join(':')));
        if (host && username) {
          return `${username}:${password}@${host}:${port}`;
        }
      }
    }

    const [host, port] = this._split_host_port(proxy);
    if (host && port) return proxy;
    return null;
  }

  _proxy_candidates_from_json(data) {
    const candidates = [];
    if (typeof data === 'object' && data !== null && !Array.isArray(data)) {
      const lower = {};
      for (const [k, v] of Object.entries(data)) {
        lower[k.toLowerCase()] = v;
      }
      const host = lower.ip || lower.host || lower.proxyhost || lower.server;
      const port = lower.port || lower.proxyport;
      if (host && port) {
        candidates.push(`${host}:${port}`);
      }
      const proxy_value = lower.proxy || lower.addr || lower.address;
      if (proxy_value) candidates.push(String(proxy_value));
      for (const value of Object.values(data)) {
        candidates.push(...this._proxy_candidates_from_json(value));
      }
    } else if (Array.isArray(data)) {
      for (const item of data) {
        candidates.push(...this._proxy_candidates_from_json(item));
      }
    } else if (typeof data === 'string') {
      candidates.push(data);
    }
    return candidates;
  }

  _proxy_error_from_json(data) {
    if (typeof data !== 'object' || data === null || Array.isArray(data)) return null;
    const lower = {};
    for (const [k, v] of Object.entries(data)) {
      lower[k.toLowerCase()] = v;
    }
    const message = lower.message || lower.msg || lower.error || lower.errmsg || lower.desc || lower.description;
    const code = lower.code;
    const data_value = lower.data;
    if (message && (data_value === null || !['0', '200', 'success', 'true', 'None'].includes(String(code)))) {
      return String(message);
    }
    return null;
  }

  _extract_proxies(response_text) {
    const proxies = [];
    const seen = new Set();
    const endpoint_index = {};

    const add = (candidate) => {
      const normalized = this._normalize_proxy(candidate);
      if (!normalized || seen.has(normalized)) return;
      const [host, port] = this._split_host_port(normalized);
      const endpoint = host && port ? `${host}:${port}` : null;
      if (endpoint && endpoint_index[endpoint] !== undefined) {
        const old_index = endpoint_index[endpoint];
        const old_proxy = proxies[old_index];
        if (normalized.includes('@') && !old_proxy.includes('@')) {
          seen.delete(old_proxy);
          proxies[old_index] = normalized;
        }
        seen.add(normalized);
        return;
      }
      if (normalized) {
        seen.add(normalized);
        if (endpoint) endpoint_index[endpoint] = proxies.length;
        proxies.push(normalized);
      }
    };

    const text = (response_text || '').trim();
    if (!text) return proxies;

    try {
      const data = JSON.parse(text);
      const json_error = this._proxy_error_from_json(data);
      for (const candidate of this._proxy_candidates_from_json(data)) {
        add(candidate);
      }
      if (json_error && proxies.length === 0) {
        this.last_error = json_error;
        this._fatal_proxy_error = true;
        console.log(`[代理] 代理API返回错误: ${json_error}`);
        return proxies;
      }
    } catch (e) {}

    const lines = text.replace(/\r/g, '\n').split('\n');
    for (const line of lines) {
      const parts = line.trim().split(/[\s,;]+/);
      for (const part of parts) {
        add(part);
      }
    }

    const proxy_pattern = /(?:(?:https?|socks5?)\/\/)?(?:[^\s\/@:]+(?::[^\s\/@]+)?@)?(?:\[[0-9A-Fa-f:]+\]|(?:\d{1,3}\.){3}\d{1,3}|localhost|(?:[A-Za-z0-9-]+\.)+[A-Za-z0-9-]+):\d{2,5}/g;
    let match;
    while ((match = proxy_pattern.exec(text)) !== null) {
      add(match[0]);
    }

    return proxies;
  }

  _short_response(text, limit = 120) {
    let brief = (text || '').trim().replace(/\s+/g, ' ');
    if (brief.length > limit) brief = brief.substring(0, limit) + '...';
    return brief || '<空响应>';
  }

  _debug_api_response(response, prefix = '代理API') {
    const body = response.body || '<无法读取响应体>';
    const display = body.length > 2000 ? body.substring(0, 2000) + '...<已截断>' : body;
    console.log(`[代理调试] ${prefix}状态码: ${response.statusCode || '<未知>'}`);
    console.log(`[代理调试] ${prefix}原始响应: ${display}`);
  }

  _debug_proxy_candidates(proxies) {
    if (proxies.length === 0) {
      console.log('[代理调试] 解析候选代理: []');
      return;
    }
    const masked = proxies.map(p => this._mask_proxy(p));
    console.log(`[代理调试] 解析候选代理: ${JSON.stringify(masked)}`);
  }

  get_last_error() {
    return this.last_error || '代理API未返回可用代理';
  }

  validate_proxy(proxy) {
    const normalized = this._normalize_proxy(proxy);
    if (!normalized) {
      console.log(`[代理] 验证失败: 无法识别代理格式: ${this._short_response(proxy)}`);
      return false;
    }
    const [host, port] = this._split_host_port(normalized);
    if (!host || !port) {
      console.log(`[代理] 验证失败: 无法解析代理地址: ${this._mask_proxy(normalized)}`);
      return false;
    }
    return new Promise((resolve) => {
      const net = require('net');
      const socket = new net.Socket();
      socket.setTimeout(3000);
      socket.on('connect', () => {
        socket.destroy();
        resolve(true);
      });
      socket.on('timeout', () => {
        socket.destroy();
        console.log(`[代理] 验证失败: TCP连接超时 proxy=${this._mask_proxy(normalized)} host=${host} port=${port}`);
        resolve(false);
      });
      socket.on('error', (err) => {
        socket.destroy();
        console.log(`[代理] 验证失败: proxy=${this._mask_proxy(normalized)} host=${host} port=${port} error=${err.message}`);
        resolve(false);
      });
      socket.connect(port, host);
    });
  }

  async get_proxy() {
    if (this._proxy_pool.length > 0) {
      return this._proxy_pool.shift();
    }
    if (this._fatal_proxy_error) return null;

    const max_retries = 3;
    this.last_error = '';

    for (let attempt = 0; attempt < max_retries; attempt++) {
      try {
        if (!this.proxy_api) {
          console.log('[错误] 未配置代理API');
          this.last_error = '未配置代理API';
          this._fatal_proxy_error = true;
          return null;
        }

        const response = await httpGet(this.proxy_api, { timeout: 10000 });
        if (response.statusCode !== 200) {
          console.log(`[错误] 代理API返回状态码: ${response.statusCode}`);
          this.last_error = `代理API返回状态码 ${response.statusCode}`;
          continue;
        }

        const proxies = this._extract_proxies(response.body);
        if (this._fatal_proxy_error) return null;

        this._debug_api_response(response);
        this._debug_proxy_candidates(proxies);

        if (proxies.length === 0) {
          if (!this.last_error) {
            this.last_error = `API响应未解析到代理: ${this._short_response(response.body)}`;
            console.log(`[代理] ${this.last_error}`);
          }
          continue;
        }

        for (const proxy of proxies) {
          const isValid = await this.validate_proxy(proxy);
          if (isValid) return proxy;
        }

        if (!this.last_error) this.last_error = '代理验证失败';
      } catch (e) {
        console.log(`[代理] 获取代理失败: ${e.message}`);
        this.last_error = `获取代理异常: ${e.message}`;
        continue;
      }
    }
    return null;
  }

  async prefetch_proxies(count) {
    console.log(`[代理] 开始预获取 ${count} 个代理...`);
    let fetched = 0;

    if (!this.proxy_api) {
      console.log('[错误] 未配置代理API');
      this.last_error = '未配置代理API';
      this._fatal_proxy_error = true;
      return;
    }
    if (this._fatal_proxy_error) return;

    for (let i = 0; i < count * 2; i++) {
      if (fetched >= count) break;
      try {
        const response = await httpGet(this.proxy_api, { timeout: 10000 });
        if (response.statusCode === 200) {
          const proxies = this._extract_proxies(response.body);
          if (this._fatal_proxy_error) break;

          this._debug_api_response(response, '预获取代理API');
          this._debug_proxy_candidates(proxies);

          if (proxies.length === 0) {
            if (!this.last_error) {
              this.last_error = `预获取响应未解析到代理: ${this._short_response(response.body)}`;
              console.log(`[代理] ${this.last_error}`);
            }
            continue;
          }

          for (const proxy of proxies) {
            if (fetched >= count) break;
            const isValid = await this.validate_proxy(proxy);
            if (isValid) {
              this._proxy_pool.push(proxy);
              fetched++;
              console.log(`[代理] 预获取成功 (${fetched}/${count}): ${this._mask_proxy(proxy)}`);
            }
          }
        } else {
          console.log(`[代理] 预获取API返回状态码: ${response.statusCode}`);
          this.last_error = `预获取代理API返回状态码 ${response.statusCode}`;
        }
      } catch (e) {
        console.log(`[代理] 预获取失败: ${e.message}`);
        this.last_error = `预获取代理异常: ${e.message}`;
        continue;
      }
    }
    console.log(`[代理] 预获取完成，共获取 ${fetched} 个代理`);
  }

  create_warmed_session(proxy, phone = '') {
    const session = {
      proxy: proxy,
      headers: {
        'User-Agent': generate_kuwo_ua(phone),
        'Accept': 'application/json, text/plain, */*',
        'Origin': 'https://h5app.kuwo.cn',
        'Sec-Fetch-Mode': 'cors',
        'Sec-Fetch-Site': 'same-site',
        'Referer': 'https://h5app.kuwo.cn/apps/earning-sign/cash_out.html',
        'Sec-Fetch-Dest': 'empty',
        'Accept-Language': 'zh-CN,zh-Hans;q=0.9'
      }
    };

    // 预热连接
    const warmup_url = 'https://integralapi.kuwo.cn/api/v1/online/sign/v1/getWithdraw';
    httpGet(warmup_url, { proxy, timeout: 5000, method: 'HEAD' })
      .then(() => console.log(`[Session] 连接预热成功: ${proxy}`))
      .catch(e => console.log(`[Session] 连接预热失败（不影响使用）: ${e.message}`));

    return session;
  }
}

/**
 * HTTP GET 请求封装
 */
function httpGet(url, options = {}) {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);
    const isHttps = urlObj.protocol === 'https:';
    const lib = isHttps ? https : http;

    const reqOptions = {
      hostname: urlObj.hostname,
      port: urlObj.port || (isHttps ? 443 : 80),
      path: urlObj.pathname + urlObj.search,
      method: options.method || 'GET',
      timeout: options.timeout || 10000,
      rejectUnauthorized: false,
      headers: {
        'Accept-Encoding': 'gzip, deflate',
        ...options.headers
      }
    };

    if (options.proxy) {
      // 简化的代理支持
      reqOptions.hostname = options.proxy.split('@').pop().split(':')[0];
      reqOptions.port = parseInt(options.proxy.split('@').pop().split(':')[1]);
      reqOptions.path = url;
    }

    const req = lib.request(reqOptions, (res) => {
      let chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => {
        const buffer = Buffer.concat(chunks);
        const encoding = res.headers['content-encoding'];

        let decompress;
        if (encoding === 'gzip') {
          decompress = zlib.gunzip;
        } else if (encoding === 'deflate') {
          decompress = zlib.inflate;
        } else {
          decompress = null;
        }

        if (decompress) {
          decompress(buffer, (err, decoded) => {
            if (err) {
              reject(err);
            } else {
              resolve({
                statusCode: res.statusCode,
                headers: res.headers,
                body: decoded.toString('utf8')
              });
            }
          });
        } else {
          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            body: buffer.toString('utf8')
          });
        }
      });
    });

    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('请求超时'));
    });

    req.end();
  });
}

/**
 * HTTP POST 请求封装
 */
function httpPost(url, data, options = {}) {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);
    const isHttps = urlObj.protocol === 'https:';
    const lib = isHttps ? https : http;

    const postData = typeof data === 'string' ? data : JSON.stringify(data);

    const reqOptions = {
      hostname: urlObj.hostname,
      port: urlObj.port || (isHttps ? 443 : 80),
      path: urlObj.pathname + urlObj.search,
      method: 'POST',
      timeout: options.timeout || 10000,
      rejectUnauthorized: false,
      headers: {
        'Content-Type': options.contentType || 'application/json',
        'Content-Length': Buffer.byteLength(postData),
        'Accept-Encoding': 'gzip, deflate',
        ...options.headers
      }
    };

    const req = lib.request(reqOptions, (res) => {
      let chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('end', () => {
        const buffer = Buffer.concat(chunks);
        const encoding = res.headers['content-encoding'];

        let decompress;
        if (encoding === 'gzip') {
          decompress = zlib.gunzip;
        } else if (encoding === 'deflate') {
          decompress = zlib.inflate;
        } else {
          decompress = null;
        }

        if (decompress) {
          decompress(buffer, (err, decoded) => {
            if (err) {
              reject(err);
            } else {
              resolve({
                statusCode: res.statusCode,
                headers: res.headers,
                body: decoded.toString('utf8')
              });
            }
          });
        } else {
          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            body: buffer.toString('utf8')
          });
        }
      });
    });

    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('请求超时'));
    });

    req.write(postData);
    req.end();
  });
}

/**
 * MD5 哈希
 */
function md5(str) {
  return crypto.createHash('md5').update(str).digest('hex');
}

/**
 * AES CBC 加密
 */
function aesEncrypt(text, key, iv) {
  const cipher = crypto.createCipheriv('aes-128-cbc', Buffer.from(key, 'base64'), Buffer.from(iv, 'base64'));
  cipher.setAutoPadding(true);
  let encrypted = cipher.update(text, 'utf8', 'base64');
  encrypted += cipher.final('base64');
  return encrypted;
}

/**
 * 加密手机号生成phone值
 */
function encrypt_phone(phone) {
  try {
    const key = 'eXNpVmtMSkhIbnZNV0NIcQ==';
    const iv = 'aWNoWW9vWCtNYjFnUmV0UA==';
    return aesEncrypt(phone, key, iv);
  } catch (e) {
    console.log(`[错误] 手机号加密失败: ${e.message}`);
    return null;
  }
}

/**
 * 生成10位随机数字的appuid
 */
function generate_appuid() {
  let uid = '';
  for (let i = 0; i < 10; i++) {
    uid += Math.floor(Math.random() * 10);
  }
  return uid;
}

/**
 * 为每个手机号生成固定的随机 User-Agent
 */
function generate_kuwo_ua(phone) {
  if (_phone_ua_cache[phone]) return _phone_ua_cache[phone];

  const seed = parseInt(md5(phone).substring(0, 8), 16);
  const [model, build] = _ANDROID_DEVICES[seed % _ANDROID_DEVICES.length];
  const av = _ANDROID_VERSIONS[seed % _ANDROID_VERSIONS.length];
  const cv = _CHROME_VERSIONS[(seed >> 8) % _CHROME_VERSIONS.length];

  const ua = `Mozilla/5.0 (Linux; Android ${av}; ${model} Build/${build}; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/${cv} Mobile Safari/537.36/ kuwopage`;
  _phone_ua_cache[phone] = ua;
  return ua;
}

/**
 * 获取NTP时间偏移量
 */
async function sync_time_offset() {
  const time_apis = [
    { url: 'http://api.m.taobao.com/rest/api3.do?api=mtop.common.getTimestamp', type: 'taobao' },
    { url: 'http://worldtimeapi.org/api/timezone/Asia/Shanghai', type: 'worldtime' }
  ];

  for (const api of time_apis) {
    try {
      const local_before = Date.now() / 1000;
      const response = await httpGet(api.url, { timeout: 3000 });
      const local_after = Date.now() / 1000;

      if (response.statusCode !== 200) continue;

      const data = JSON.parse(response.body);
      const local_mid = (local_before + local_after) / 2;
      let server_time;

      if (api.type === 'taobao' && data.data) {
        server_time = parseInt(data.data.t) / 1000;
      } else if (api.type === 'worldtime' && data.unixtime) {
        server_time = parseFloat(data.unixtime);
      } else {
        continue;
      }

      _time_offset = server_time - local_mid;
      console.log(`[时间] HTTP API同步成功 (${api.type})，偏移量: ${_time_offset.toFixed(3)}秒`);
      return;
    } catch (e) {
      console.log(`[警告] 从API ${api.url} 获取时间失败: ${e.message}`);
      continue;
    }
  }

  _time_offset = 0;
  console.log('[警告] 所有时间源都失败，使用本地时间（偏移量=0）');
}

/**
 * 基于偏移量快速获取精确的当前时间
 */
function get_precise_time() {
  if (_time_offset === null) {
    sync_time_offset();
  }
  return new Date(Date.now() + (_time_offset || 0) * 1000);
}

/**
 * 获取北京时间
 */
function get_beijing_time() {
  return get_precise_time();
}

/**
 * 高精度等待到目标时间
 */
async function precision_wait(target_time) {
  const now = get_precise_time();
  let wait_seconds = (target_time.getTime() - now.getTime()) / 1000;

  if (wait_seconds <= 0) return;

  // 阶段1：粗等待
  if (wait_seconds > 2) {
    const coarse_wait = wait_seconds - 1.5;
    console.log(`[等待] 粗等待 ${coarse_wait.toFixed(1)} 秒...`);
    await sleep(coarse_wait * 1000);
  }

  // 阶段2：精等待
  console.log('[等待] 进入精确等待模式...');
  const target_ts = target_time.getTime();
  const offset = (_time_offset || 0) * 1000;

  while (true) {
    const current_ts = Date.now() + offset;
    if (current_ts >= target_ts) break;
    const remaining = (target_ts - current_ts) / 1000;
    if (remaining > 0.05) {
      await sleep(1);
    }
  }

  const actual_time = get_precise_time();
  const diff_ms = (actual_time.getTime() - target_time.getTime());
  console.log(`[等待] 等待完成，实际偏差: ${diff_ms.toFixed(1)}ms`);
}

/**
 * 延时函数
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * 验证码识别
 */
async function recognize_captcha(image_base64) {
  try {
    const ocr_url = 'https://ddddocr.linzixuan.work/classification';
    if (image_base64.includes(',')) {
      image_base64 = image_base64.split(',')[1];
    }
    image_base64 = image_base64.replace(/data:image\/(jpeg|png);base64,/, '');

    const response = await httpPost(ocr_url, { image: image_base64 }, { timeout: 10000 });
    const result = JSON.parse(response.body);

    if (!result || !result.result) {
      throw new Error('验证码识别失败: 返回结果无效');
    }
    return result.result.trim();
  } catch (e) {
    console.log(`[错误] 验证码识别出错: ${e.message}`);
    throw e;
  }
}

/**
 * 登录获取最新的loginUid和loginSid
 */
async function login_for_withdraw(phone, password) {
  try {
    // 获取验证码
    const captcha_url = 'http://www.kuwo.cn/api/common/captcha/getcode';
    const captcha_params = '?reqId=bb7dd120-d1b7-11ef-b9c9-9dd176f54932&httpsStatus=1';

    const captcha_headers = {
      'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.6261.95 Safari/537.36',
      'Accept': 'application/json, text/plain, */*',
      'Accept-Encoding': 'gzip, deflate',
      'Content-Type': 'application/json',
      'Referer': 'http://www.kuwo.cn/',
      'Accept-Language': 'zh-CN,zh;q=0.9'
    };

    const captchaResp = await httpGet(captcha_url + captcha_params, { headers: captcha_headers });
    const captchaResult = JSON.parse(captchaResp.body);

    if (!captchaResult.data) {
      throw new Error('获取验证码失败');
    }

    const captcha_data = captchaResult.data;
    const image_data = captcha_data.img;
    const token = captcha_data.token;

    // 识别验证码
    const verify_code = await recognize_captcha(image_data.replace('data:image/jpeg;base64,', ''));

    // 执行登录
    const login_url = 'https://wapi.kuwo.cn/api/www/login/loginByKw';
    const login_data = JSON.stringify({
      userIp: 'www.kuwo.cn',
      uname: phone,
      password: password,
      verifyCode: verify_code,
      img: image_data,
      verifyCodeToken: token
    });

    const login_headers = {
      'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.6261.95 Safari/537.36',
      'Accept': 'application/json, text/plain, */*',
      'Content-Type': 'application/json',
      'Origin': 'http://www.kuwo.cn',
      'Referer': 'http://www.kuwo.cn/',
      'Accept-Language': 'zh-CN,zh;q=0.9'
    };

    const loginResp = await httpPost(login_url + '?httpsStatus=1', login_data, {
      headers: login_headers,
      timeout: 10000
    });

    const result = JSON.parse(loginResp.body);

    if (result.code !== 200) {
      const error_msg = result.msg || '未知错误';
      if (error_msg.includes('picture captcha error')) {
        return { loginUid: null, loginSid: null, error: '登录接口抽风，请再试一次即可' };
      }
      throw new Error(`登录失败: ${error_msg}`);
    }

    const cookies = result.data?.cookies;
    if (!cookies || typeof cookies !== 'object') {
      throw new Error('登录响应中没有找到有效的cookies');
    }

    const loginSid = cookies.websid;
    const loginUid = cookies.userid;

    if (!loginSid || !loginUid) {
      throw new Error('登录响应中缺少必要的cookie信息');
    }

    return { loginUid, loginSid, error: null };
  } catch (e) {
    console.log(`[错误] 登录过程异常: ${e.message}`);
    if (e.message.includes('picture captcha error')) {
      return { loginUid: null, loginSid: null, error: '登录接口抽风，请再试一次即可' };
    }
    if (e.message.includes('登录响应中')) {
      return { loginUid: null, loginSid: null, error: '登录失败: 服务器返回数据格式异常' };
    }
    return { loginUid: null, loginSid: null, error: `登录异常: ${e.message}` };
  }
}

/**
 * 登录账号
 */
async function login(value) {
  try {
    const values = value.split('#');
    if (values.length !== 2) {
      return { account: '登录参数格式错误', token: null, success: false };
    }

    const [phone, password] = values;
    const appUid = generate_appuid();
    const phone_value = encrypt_phone(phone);

    if (!phone_value) {
      return { account: '手机号加密失败', token: null, success: false };
    }

    // 重新登录获取最新的loginUid和loginSid
    const { loginUid, loginSid, error } = await login_for_withdraw(phone, password);
    if (error) {
      return { account: error, token: null, success: false };
    }

    // 发送短信验证码
    const url = 'https://integralapi.kuwo.cn/api/v1/online/sign/v1/userBindPhone';
    const params = new URLSearchParams({
      loginUid,
      loginSid,
      mobile: phone_value
    }).toString();

    const headers = {
      'User-Agent': generate_kuwo_ua(phone),
      'Accept': 'application/json, text/plain, */*',
      'Origin': 'https://h5app.kuwo.cn',
      'Sec-Fetch-Mode': 'cors',
      'Sec-Fetch-Site': 'same-site',
      'Referer': 'https://h5app.kuwo.cn/apps/earning-sign/cash_out.html',
      'Sec-Fetch-Dest': 'empty',
      'Accept-Language': 'zh-CN,zh-Hans;q=0.9'
    };

    const response = await httpGet(`${url}?${params}`, { headers });
    if (response.statusCode !== 200) {
      return { account: '发送验证码失败', token: null, success: false };
    }

    const result = JSON.parse(response.body);
    if (result.code !== 200) {
      return { account: `发送验证码失败: ${result.msg || '未知错误'}`, token: null, success: false };
    }

    // 返回需要验证码的状态
    return {
      account: phone,
      token: null,
      success: false,
      need_sms: true,
      loginUid,
      loginSid,
      phone_value,
      phone
    };
  } catch (e) {
    console.log(`[错误] 登录过程异常: ${e.message}`);
    return { account: `登录异常: ${e.message}`, token: null, success: false };
  }
}

/**
 * 验证提现接口
 */
async function verify_withdraw(loginUid, loginSid, phone_value, sms_code, phone) {
  try {
    const withdraw_url = 'https://integralapi.kuwo.cn/api/v1/online/sign/v1/getWithdraw';
    const params = new URLSearchParams({
      encry: '',
      type: '',
      quotaId: '30002',
      loginUid,
      loginSid,
      appuid: generate_appuid(),
      source: 'kwplayer_ar_12.1.4.0_40.apk',
      version: '1',
      phone: phone_value,
      code: sms_code
    }).toString();

    const headers = {
      'User-Agent': generate_kuwo_ua(phone),
      'Accept': 'application/json, text/plain, */*',
      'Origin': 'https://h5app.kuwo.cn',
      'Sec-Fetch-Mode': 'cors',
      'Sec-Fetch-Site': 'same-site',
      'Referer': 'https://h5app.kuwo.cn/apps/earning-sign/cash_out.html',
      'Sec-Fetch-Dest': 'empty',
      'Accept-Language': 'zh-CN,zh-Hans;q=0.9'
    };

    const response = await httpGet(`${withdraw_url}?${params}`, { headers });
    if (response.statusCode !== 200) {
      return { success: false, error: '账号验证失败' };
    }

    const result = JSON.parse(response.body);
    const text = result.data?.text || '';

    const valid_messages = [
      '每日仅能提现一次', '今日提现次数已用完', '账号存在风险',
      '提现额度已用完', '提现次数已用完', '提现时间未到',
      '当前时段额度已提完', '当前账户金币余额不足', '提现成功'
    ];

    if (valid_messages.some(msg => text.includes(msg))) {
      console.log(`[验证] 账号有效: ${text}`);
      const devId = crypto.randomBytes(8).toString('hex');
      const token = `${loginUid}#${devId}#${loginSid}#${phone_value}`;
      return { success: true, token, phone };
    }

    return { success: false, error: `账号验证失败: ${text}` };
  } catch (e) {
    return { success: false, error: `验证异常: ${e.message}` };
  }
}

/**
 * 执行提现操作
 */
async function execute_withdraw(loginUid, loginSid, phone_value, sms_code, phone, proxy) {
  try {
    const withdraw_url = 'https://integralapi.kuwo.cn/api/v1/online/sign/v1/getWithdraw';
    const params = new URLSearchParams({
      encry: '',
      type: '',
      quotaId: '30002',
      loginUid,
      loginSid,
      appuid: generate_appuid(),
      source: 'kwplayer_ar_12.1.4.0_40.apk',
      version: '1',
      phone: phone_value,
      code: sms_code
    }).toString();

    const headers = {
      'User-Agent': generate_kuwo_ua(phone),
      'Accept': 'application/json, text/plain, */*',
      'Origin': 'https://h5app.kuwo.cn',
      'Sec-Fetch-Mode': 'cors',
      'Sec-Fetch-Site': 'same-site',
      'Referer': 'https://h5app.kuwo.cn/apps/earning-sign/cash_out.html',
      'Sec-Fetch-Dest': 'empty',
      'Accept-Language': 'zh-CN,zh-Hans;q=0.9'
    };

    const response = await httpGet(`${withdraw_url}?${params}`, {
      headers,
      proxy,
      timeout: 8000
    });

    if (response.statusCode !== 200) {
      return { success: false, error: '提现请求失败' };
    }

    const result = JSON.parse(response.body);
    const text = result.data?.text || '';

    if (text.includes('提现成功') || text.includes('提现申请发起成功')) {
      return { success: true, message: text };
    }

    return { success: false, error: text };
  } catch (e) {
    return { success: false, error: `提现异常: ${e.message}` };
  }
}

/**
 * Middleware 接口封装
 * 注意：这里需要根据实际的middleware实现进行调整
 */
class Middleware {
  constructor() {
    // 这里需要根据实际的XDD/middleware实现来初始化
    this.data = {};
  }

  getSenderID() {
    // 返回发送者ID
    return 'default_sender';
  }

  getUserID() {
    // 返回用户ID
    return 'default_user';
  }

  getMessage() {
    // 返回消息内容
    return '';
  }

  isAdmin() {
    // 检查是否是管理员
    return false;
  }

  bucketGet(bucket, key) {
    // 从存储桶获取数据
    return this.data[`${bucket}:${key}`] || null;
  }

  bucketSet(bucket, key, value) {
    // 设置存储桶数据
    this.data[`${bucket}:${key}`] = value;
  }

  bucketDel(bucket, key) {
    // 删除存储桶数据
    delete this.data[`${bucket}:${key}`];
  }

  bucketAll(bucket) {
    // 获取存储桶所有数据
    const result = {};
    for (const [k, v] of Object.entries(this.data)) {
      if (k.startsWith(`${bucket}:`)) {
        const key = k.replace(`${bucket}:`, '');
        result[key] = v;
      }
    }
    return result;
  }

  reply(message) {
    // 回复消息
    console.log(`[回复] ${message}`);
  }

  replyImage(url) {
    // 回复图片
    console.log(`[回复图片] ${url}`);
  }

  input(timeout, max, multi) {
    // 获取用户输入
    return '';
  }

  listen(timeout) {
    // 监听消息
    return null;
  }

  waitPay(q, timeout) {
    // 等待支付
    return null;
  }

  atWaitPay() {
    // 检查是否有人正在支付
    return false;
  }
}

/**
 * 获取插件配置数据
 */
function PluginsData(middleware) {
  const KuwoTXmoney = middleware.bucketGet('dd_KuwoTX_PluginsData', 'KuwoTXmoney');
  const KuwoTXcoin = middleware.bucketGet('dd_KuwoTX_PluginsData', 'KuwoTXcoin');
  const proxy_api = middleware.bucketGet('dd_KuwoTX_PluginsData', 'proxy_api');
  const withdraw_delay_str = middleware.bucketGet('dd_KuwoTX_PluginsData', 'withdraw_delay');

  if (!proxy_api) {
    middleware.reply('未配置代理API，请检查配置');
    process.exit(0);
  }

  let money = 0;
  if (!KuwoTXmoney || KuwoTXmoney === '0') {
    money = 0;
  } else {
    money = parseFloat(KuwoTXmoney);
    if (money < 0.5) {
      middleware.reply('提现单价不能低于0.5元，请修改配置');
      process.exit(0);
    }
  }

  let coin = 9999;
  if (KuwoTXcoin) {
    coin = parseInt(KuwoTXcoin);
  }

  let delay = 0.0;
  if (withdraw_delay_str) {
    delay = parseFloat(withdraw_delay_str);
    delay = Math.max(0.0, Math.min(5.0, delay));
  }

  return { KuwoTXmoney: money, KuwoTXcoin: coin, proxy_api, withdraw_delay: delay };
}

/**
 * 增加用户提现次数
 */
function empower(middleware, user_id, count) {
  const current_count = parseInt(middleware.bucketGet('dd_KuwoTX_UserCount', user_id) || '0');
  const new_count = current_count + count;
  middleware.bucketSet('dd_KuwoTX_UserCount', user_id, String(new_count));
  return new_count;
}

/**
 * 减少用户提现次数
 */
function decrease_user_withdraw_count(middleware, user_id) {
  const current_count = parseInt(middleware.bucketGet('dd_KuwoTX_UserCount', user_id) || '0');
  const new_count = Math.max(0, current_count - 1);
  middleware.bucketSet('dd_KuwoTX_UserCount', user_id, String(new_count));
  return new_count;
}

/**
 * 获取用户提现次数
 */
function get_user_withdraw_count(middleware, user_id) {
  return middleware.bucketGet('dd_KuwoTX_UserCount', user_id) || '0';
}

/**
 * 迁移账号级别次数到用户级别
 */
function migrate_account_counts_to_user(middleware) {
  const all_binds = middleware.bucketAll('dd_KuwoTX_bind');
  const migration_results = [];

  for (const [user_id, uservalue] of Object.entries(all_binds)) {
    try {
      const accounts = JSON.parse(uservalue);
      let total_account_count = 0;
      const migrated_accounts = [];

      for (const account of accounts) {
        const account_count = parseInt(middleware.bucketGet('dd_KuwoTX_UserCount', account) || '0');
        if (account_count > 0) {
          total_account_count += account_count;
          migrated_accounts.push([account, account_count]);
        }
        middleware.bucketDel('dd_KuwoTX_UserCount', account);
      }

      if (total_account_count > 0) {
        const user_count = parseInt(middleware.bucketGet('dd_KuwoTX_UserCount', user_id) || '0');
        const new_user_count = user_count + total_account_count;
        middleware.bucketSet('dd_KuwoTX_UserCount', user_id, String(new_user_count));
        migration_results.push(`用户 ${user_id}: 账号次数 ${total_account_count} + 用户次数 ${user_count} = 新次数 ${new_user_count}`);
        console.log(`[迁移] 用户 ${user_id}: 账号次数 ${total_account_count} + 用户次数 ${user_count} = 新次数 ${new_user_count}`);
      }
    } catch (e) {
      console.log(`[错误] 迁移用户 ${user_id} 次数时出错: ${e.message}`);
      continue;
    }
  }

  return migration_results;
}

/**
 * 主执行函数
 */
async function main() {
  // 初始化
  const middleware = new Middleware();
  const senderID = middleware.getSenderID();
  const userid = middleware.getUserID();
  const uservalue = middleware.bucketGet('dd_KuwoTX_bind', userid);

  // 获取配置
  const config = PluginsData(middleware);
  proxy_manager = new ProxyManager(config.proxy_api);

  // 同步时间
  await sync_time_offset();

  const message = middleware.getMessage();

  // 处理不同指令
  if (message === '酷我提现授权检测') {
    check_authorization(middleware);
    return;
  }

  if (message === '酷我提现次数迁移') {
    if (middleware.isAdmin()) {
      const results = migrate_account_counts_to_user(middleware);
      if (results.length > 0) {
        middleware.reply('=====次数迁移结果=====\n' + results.join('\n') + '\n===================');
      } else {
        middleware.reply('没有需要迁移的次数数据');
      }
    } else {
      middleware.reply('只有管理员可以执行此操作');
    }
    return;
  }

  // 进入管理界面
  await Administration(middleware, config);
}

/**
 * 管理界面
 */
async function Administration(middleware, config) {
  const userid = middleware.getUserID();
  const uservalue = middleware.bucketGet('dd_KuwoTX_bind', userid);

  let base_message = '=====酷我提现=====\n';
  base_message += '1️⃣ 提交账号\n';
  base_message += '2️⃣ 授权账号\n';
  base_message += '3️⃣ 删除账号\n';
  base_message += '4️⃣ 账号提现\n';

  if (middleware.isAdmin()) {
    base_message += '5️⃣ 用户授权\n';
  }
  base_message += '⚠️ 输入q退出操作\n===================';

  middleware.reply(base_message);

  const choice = await middleware.input(60000, 1, false);
  if (choice.toLowerCase() === 'q') {
    middleware.reply('退出操作');
    return;
  }

  try {
    const choiceNum = parseInt(choice);

    if (choiceNum === 1) {
      // 提交账号
      await bind(middleware);
    } else if (choiceNum === 2) {
      // 授权账号
      await authorizeAccount(middleware, config, userid, uservalue);
    } else if (choiceNum === 3) {
      // 删除账号
      await deleteAccount(middleware, userid, uservalue);
    } else if (choiceNum === 4) {
      // 提现功能
      await withdrawAccount(middleware, config, userid, uservalue);
    } else if (choiceNum === 5 && middleware.isAdmin()) {
      // 用户授权
      await userAuthorize(middleware);
    } else {
      middleware.reply('输入无效');
    }
  } catch (e) {
    middleware.reply('输入无效');
  }
}

/**
 * 绑定账号
 */
async function bind(middleware) {
  middleware.reply(
    '=====酷我提现=====\n' +
    '🎵 请输入登录参数:\n' +
    '📝 格式: 手机号#密码\n' +
    '⚠️ 建议私聊登录,密码泄露风险自负\n' +
    '⭐ 输入q退出操作\n' +
    '====================='
  );

  const login_value = await middleware.input(120000, 1, false);
  if (!login_value) {
    middleware.reply('输入超时！');
    return;
  }
  if (login_value.toLowerCase() === 'q') {
    middleware.reply('退出操作！');
    return;
  }

  const result = await login(login_value);

  if (result.need_sms) {
    // 需要验证码
    middleware.reply(`验证码已发送至 ${result.phone.substring(0, 3)}****${result.phone.substring(7)}\n请输入收到的验证码:`);
    const sms_code = await middleware.input(60000, 1, false);

    if (!sms_code) {
      middleware.reply('验证码输入超时');
      return;
    }

    const verifyResult = await verify_withdraw(
      result.loginUid,
      result.loginSid,
      result.phone_value,
      sms_code,
      result.phone
    );

    if (verifyResult.success) {
      const account = result.phone;
      const token = verifyResult.token;

      middleware.bucketSet('dd_KuwoTX_account', account, token);
      middleware.bucketSet('dd_KuwoTX_login', account, login_value);

      let accounts = [];
      const uservalue = middleware.bucketGet('dd_KuwoTX_bind', middleware.getUserID());
      if (!uservalue) {
        accounts = [account];
        middleware.bucketSet('dd_KuwoTX_bind', middleware.getUserID(), JSON.stringify(accounts));
        middleware.reply(
          '=====登录成功=====\n' +
          '✅ 账号添加成功\n' +
          '🎮 发送[酷我提现]管理账号\n' +
          '==================='
        );
      } else {
        accounts = JSON.parse(uservalue);
        if (accounts.includes(account)) {
          middleware.reply('更新账号成功，可对我说"酷我提现"对账号进行管理！');
        } else {
          accounts.push(account);
          middleware.bucketSet('dd_KuwoTX_bind', middleware.getUserID(), JSON.stringify(accounts));
          middleware.reply(
            '=====登录成功=====\n' +
            '✅ 账号添加成功\n' +
            '🎮 发送[酷我提现]管理账号\n' +
            '==================='
          );
        }
      }
    } else {
      middleware.reply(verifyResult.error);
    }
  } else {
    middleware.reply(result.account);
  }
}

/**
 * 授权账号
 */
async function authorizeAccount(middleware, config, userid, uservalue) {
  if (!uservalue) {
    middleware.reply('未绑定任何账号,请先提交账号');
    return;
  }

  const accounts = JSON.parse(uservalue);
  const user_withdraw_count = get_user_withdraw_count(middleware, userid);

  let message = '=====账号授权=====\n';
  message += `🔢 当前可用次数: ${user_withdraw_count}次\n`;
  message += `📱 绑定账号数: ${accounts.length}个\n`;
  message += '-------------------\n';

  let count = 1;
  for (const account of accounts) {
    const login_info = middleware.bucketGet('dd_KuwoTX_login', account);
    let phone;
    if (login_info) {
      phone = login_info.split('#')[0];
    } else {
      const token = middleware.bucketGet('dd_KuwoTX_account', account);
      if (!token) continue;
      phone = token.split('#')[0];
    }
    const phone_masked = phone.substring(0, 3) + '****' + phone.substring(7);
    message += `[${count}] 账号: ${phone_masked}\n`;
    count++;
  }

  message += '-------------------\n';
  message += '请输入充值次数(输入q退出):';

  middleware.reply(message);

  const count_input = await middleware.input(60000, 1, false);
  if (count_input.toLowerCase() === 'q') {
    middleware.reply('退出操作');
    return;
  }

  try {
    const count = parseInt(count_input);
    if (count <= 0) {
      middleware.reply('充值次数必须大于0');
      return;
    }

    const project = '酷我提现次数充值';
    await zf(middleware, config, project, count, userid);
  } catch (e) {
    middleware.reply('输入的次数无效');
  }
}

/**
 * 删除账号
 */
async function deleteAccount(middleware, userid, uservalue) {
  if (!uservalue) {
    middleware.reply('未绑定任何账号');
    return;
  }

  const accounts = JSON.parse(uservalue);
  let message = '=====选择账号=====\n';
  let count = 1;

  for (const account of accounts) {
    const login_info = middleware.bucketGet('dd_KuwoTX_login', account);
    let phone;
    if (login_info) {
      phone = login_info.split('#')[0];
    } else {
      const token = middleware.bucketGet('dd_KuwoTX_account', account);
      if (!token) continue;
      phone = token.split('#')[0];
    }
    const phone_masked = phone.substring(0, 3) + '****' + phone.substring(7);
    message += `[${count}] 账号: ${phone_masked}\n-------------------\n`;
    count++;
  }

  message += '⚠️ 输入q退出操作\n==================';
  middleware.reply(message);

  const acc_choice = await middleware.input(60000, 1, false);
  if (acc_choice.toLowerCase() === 'q') {
    middleware.reply('退出操作');
    return;
  }

  try {
    const acc_num = parseInt(acc_choice);
    if (acc_num < 1 || acc_num >= count) {
      middleware.reply('输入的账号序号无效');
      return;
    }

    const selected_account = accounts[acc_num - 1];
    const login_info = middleware.bucketGet('dd_KuwoTX_login', selected_account);
    let phone;
    if (login_info) {
      phone = login_info.split('#')[0];
    } else {
      const token = middleware.bucketGet('dd_KuwoTX_account', selected_account);
      if (!token) return;
      phone = token.split('#')[0];
    }
    const phone_masked = phone.substring(0, 3) + '****' + phone.substring(7);

    middleware.reply(
      '=====删除确认=====\n' +
      `📱 账号: ${phone_masked}\n` +
      '是否确认删除?\n' +
      '[y]确认 | [n]取消\n' +
      '==================='
    );

    const confirm = await middleware.input(60000, 1, false);
    if (confirm.toLowerCase() === 'y') {
      const idx = accounts.indexOf(selected_account);
      if (idx > -1) {
        accounts.splice(idx, 1);
      }
      if (accounts.length > 0) {
        middleware.bucketSet('dd_KuwoTX_bind', userid, JSON.stringify(accounts));
      } else {
        middleware.bucketDel('dd_KuwoTX_bind', userid);
      }
      middleware.bucketDel('dd_KuwoTX_account', selected_account);
      middleware.bucketDel('dd_KuwoTX_login', selected_account);
      middleware.reply('删除成功');
    } else if (confirm.toLowerCase() === 'n') {
      middleware.reply('已取消删除');
    } else {
      middleware.reply('输入无效');
    }
  } catch (e) {
    middleware.reply('输入无效');
  }
}

/**
 * 提现功能
 */
async function withdrawAccount(middleware, config, userid, uservalue) {
  if (!uservalue) {
    middleware.reply('未绑定任何账号,请先提交账号');
    return;
  }

  const user_withdraw_count = get_user_withdraw_count(middleware, userid);

  // 检查账号级别次数并迁移
  const accounts = JSON.parse(uservalue);
  let account_total_count = 0;
  for (const account of accounts) {
    const account_count = parseInt(middleware.bucketGet('dd_KuwoTX_UserCount', account) || '0');
    account_total_count += account_count;
  }

  if (account_total_count > 0) {
    const migration_results = migrate_account_counts_to_user(middleware);
    const new_count = get_user_withdraw_count(middleware, userid);
    middleware.reply(`检测到账号级别次数，已自动迁移到用户级别\n当前可用次数: ${new_count}次`);
  }

  const current_count = parseInt(get_user_withdraw_count(middleware, userid));
  if (current_count <= 0) {
    middleware.reply('您当前没有可用的提现次数，请先充值');
    return;
  }

  let message = '=====账号提现=====\n';
  message += `🔢 当前可用次数: ${current_count}次\n`;
  message += '-------------------\n';

  let count = 1;
  const valid_accounts = [];

  for (const account of accounts) {
    const login_info = middleware.bucketGet('dd_KuwoTX_login', account);
    const token = middleware.bucketGet('dd_KuwoTX_account', account);

    let phone;
    if (login_info) {
      phone = login_info.split('#')[0];
    } else if (token) {
      phone = token.split('#')[0];
    } else {
      continue;
    }

    const phone_masked = phone.substring(0, 3) + '****' + phone.substring(7);
    message += `[${count}] 账号: ${phone_masked}\n-------------------\n`;

    valid_accounts.push({
      index: count - 1,
      account,
      phone_masked,
      login_info,
      token
    });
    count++;
  }

  if (valid_accounts.length === 0) {
    middleware.reply('没有可用的已授权账号，请先授权后再使用提现功能');
    return;
  }

  message += '0️⃣ 批量提现\n⚠️ 输入q退出操作\n==================';
  middleware.reply(message);

  const acc_choice = await middleware.input(60000, 1, false);
  if (acc_choice.toLowerCase() === 'q') {
    middleware.reply('退出操作');
    return;
  }

  let selected_indices = [];

  if (acc_choice === '0') {
    middleware.reply(
      '=====批量提现=====\n' +
      '请输入账号序号\n' +
      '格式1: 起始序号-结束序号 (例如: 1-3)\n' +
      '格式2: 单独序号,序号,序号 (例如: 1,3,5)\n' +
      '==================='
    );

    const range_choice = await middleware.input(60000, 1, false);
    if (!range_choice) {
      middleware.reply('输入超时');
      return;
    }

    try {
      if (range_choice.includes('-')) {
        const [start, end] = range_choice.split('-').map(Number);
        selected_indices = Array.from({ length: end - start + 1 }, (_, i) => start - 1 + i);
      } else if (range_choice.includes(',')) {
        selected_indices = range_choice.split(',').map(idx => parseInt(idx.trim()) - 1);
      } else {
        selected_indices = [parseInt(range_choice.trim()) - 1];
      }
    } catch (e) {
      middleware.reply('输入格式错误');
      return;
    }
  } else {
    try {
      selected_indices = [parseInt(acc_choice) - 1];
    } catch (e) {
      middleware.reply('输入格式错误，请输入有效的账号序号');
      return;
    }
  }

  const valid_indices = valid_accounts.map(acc => acc.index);
  selected_indices = selected_indices.filter(i => valid_indices.includes(i));

  if (selected_indices.length === 0) {
    middleware.reply('未选择任何有效账号');
    return;
  }

  const selected_accounts = valid_accounts.filter(acc => selected_indices.includes(acc.index));

  let select_message = '=====已选择账号=====\n';
  for (const acc of selected_accounts) {
    select_message += `📱 账号: ${acc.phone_masked}\n`;
  }
  select_message += `共选择了 ${selected_accounts.length} 个账号\n===================`;
  middleware.reply(select_message);

  // 询问是否等待整点提现
  middleware.reply(
    '=====提现时间=====\n' +
    '是否等待整点提现?\n' +
    '[y]是 | [n]否\n' +
    '==================='
  );

  const wait_choice = await middleware.input(60000, 1, false);
  if (!wait_choice) {
    middleware.reply('输入超时');
    return;
  }

  const now = get_beijing_time();
  const current_hour = now.getHours();
  const current_minute = now.getMinutes();

  const withdraw_hours = [0, 9, 13, 17, 20];
  const wait_hours = [23, 8, 12, 16, 19];
  const today_wait_hours = [8, 12, 16, 19, 23];

  const accounts_info = [];

  if (wait_choice.toLowerCase() === 'y') {
    // 等待整点提现
    let is_wait_time = false;
    let target_hour = null;

    if (current_hour === 23 && current_minute >= 55) {
      is_wait_time = true;
      target_hour = 0;
    } else {
      for (const hour of wait_hours) {
        if (current_hour === hour && current_minute >= 55) {
          is_wait_time = true;
          target_hour = withdraw_hours[(wait_hours.indexOf(hour) + 1) % withdraw_hours.length];
          break;
        }
      }
    }

    if (!is_wait_time) {
      let next_wait_hour = null;
      for (const hour of today_wait_hours) {
        if (hour > current_hour || (hour === current_hour && current_minute < 55)) {
          next_wait_hour = hour;
          break;
        }
      }

      let target_time;
      if (next_wait_hour === null) {
        target_time = new Date(now);
        target_time.setDate(target_time.getDate() + 1);
        target_time.setHours(today_wait_hours[0], 55, 0, 0);
      } else {
        target_time = new Date(now);
        target_time.setHours(next_wait_hour, 55, 0, 0);
      }

      middleware.reply(
        `当前不在提现等待时间段\n` +
        `当前北京时间: ${now.getHours()}:${String(now.getMinutes()).padStart(2, '0')}\n` +
        `下次等待时间: ${target_time.getHours()}:${String(target_time.getMinutes()).padStart(2, '0')}\n` +
        '请在该时间后再试'
      );
      return;
    }

    // 为每个选中的账号获取验证码
    for (const acc of selected_accounts) {
      try {
        const proxy = await proxy_manager.get_proxy();
        if (!proxy) {
          middleware.reply(`获取代理失败，跳过账号 ${acc.phone_masked}\n原因: ${proxy_manager.get_last_error()}`);
          continue;
        }

        const login_info = acc.login_info;
        if (!login_info) {
          middleware.reply(`账号 ${acc.phone_masked} 缺少登录信息，跳过`);
          continue;
        }

        const login_values = login_info.split('#');
        const phone = login_values[0];
        const password = login_values[1] || '';

        middleware.reply(`正在为账号 ${acc.phone_masked} 重新登录获取凭证...`);

        const { loginUid, loginSid, error } = await login_for_withdraw(phone, password);
        if (error) {
          middleware.reply(`账号 ${acc.phone_masked} 重新登录失败: ${error}`);
          continue;
        }

        const phone_value = encrypt_phone(phone);
        if (!phone_value) {
          middleware.reply(`账号 ${acc.phone_masked} 手机号加密失败`);
          continue;
        }

        const new_token = `${loginUid}#${phone}#${loginSid}#${phone_value}`;
        middleware.bucketSet('dd_KuwoTX_account', acc.account, new_token);
        middleware.reply(`账号 ${acc.phone_masked} 重新登录成功`);

        // 发送验证码
        const url = 'https://integralapi.kuwo.cn/api/v1/online/sign/v1/userBindPhone';
        const params = new URLSearchParams({
          loginUid,
          loginSid,
          mobile: phone_value
        }).toString();

        const headers = {
          'User-Agent': generate_kuwo_ua(phone),
          'Accept': 'application/json, text/plain, */*',
          'Origin': 'https://h5app.kuwo.cn',
          'Sec-Fetch-Mode': 'cors',
          'Sec-Fetch-Site': 'same-site',
          'Referer': 'https://h5app.kuwo.cn/apps/earning-sign/cash_out.html',
          'Sec-Fetch-Dest': 'empty',
          'Accept-Language': 'zh-CN,zh-Hans;q=0.9'
        };

        const response = await httpGet(`${url}?${params}`, { headers, proxy });
        if (response.statusCode !== 200) {
          middleware.reply(`账号 ${acc.phone_masked} 发送验证码失败`);
          continue;
        }

        const result = JSON.parse(response.body);

        // 检查是否返回"用户未登录"错误
        const data_status = result.data?.status;
        const data_desc = result.data?.description || '';
        if (data_status === 0 && data_desc === '用户未登录') {
          middleware.reply(`账号 ${acc.phone_masked} 登录凭证已失效！\n请重新执行「1️⃣ 提交账号」绑定账号`);
          continue;
        }

        if (result.code !== 200) {
          middleware.reply(`账号 ${acc.phone_masked} 发送验证码失败: ${result.msg || '未知错误'}`);
          continue;
        }

        middleware.reply(`请输入账号 ${acc.phone_masked} 的验证码:`);
        const sms_code = await middleware.input(60000, 1, false);
        if (!sms_code) {
          middleware.reply(`账号 ${acc.phone_masked} 验证码输入超时`);
          continue;
        }

        accounts_info.push({
          phone_masked: acc.phone_masked,
          phone_raw: phone,
          loginUid,
          loginSid,
          phone_value,
          sms_code,
          proxy
        });
      } catch (e) {
        middleware.reply(`处理账号 ${acc.phone_masked} 时出错: ${e.message}`);
        continue;
      }
    }

    if (accounts_info.length === 0) {
      middleware.reply('没有成功准备好的账号，退出操作');
      return;
    }

    // 计算目标提现时间
    let target_time;
    if (current_hour === 23) {
      target_time = new Date(now);
      target_time.setDate(target_time.getDate() + 1);
      target_time.setHours(0, 0, 0, 0);
    } else {
      target_time = new Date(now);
      target_time.setHours(current_hour + 1, 0, 0, 0);
    }

    const actual_delay = config.withdraw_delay > 0 ? config.withdraw_delay : 1.0;
    target_time = new Date(target_time.getTime() + actual_delay * 1000);

    const wait_seconds = Math.max(0, (target_time.getTime() - now.getTime()) / 1000);

    middleware.reply(
      '=====提现准备就绪=====\n' +
      `📱 账号数: ${accounts_info.length}个\n` +
      `⏰ 目标时间: ${target_time.getHours()}:${String(target_time.getMinutes()).padStart(2, '0')}:${String(target_time.getSeconds()).padStart(2, '0')}\n` +
      `🚀 延后发包: ${actual_delay}秒\n` +
      `⏳ 等待: ${Math.floor(wait_seconds)}秒\n` +
      '======================'
    );

    // 等待期间执行预热操作
    console.log('[优化] 重新同步NTP时间...');
    await sync_time_offset();

    console.log('[优化] 预获取代理到代理池...');
    await proxy_manager.prefetch_proxies(accounts_info.length);

    middleware.reply('预热完成，等待整点...');

    // 高精度等待
    await precision_wait(target_time);

    // 整点到达，立即并发提现
    const fire_time = get_precise_time();
    middleware.reply(`开始批量提现... 发包时间: ${fire_time.getHours()}:${String(fire_time.getMinutes()).padStart(2, '0')}:${String(fire_time.getSeconds()).padStart(2, '0')}.${String(fire_time.getMilliseconds()).padStart(3, '0')}`);

    let success_count = 0;
    let fail_count = 0;

    // 并发处理提现
    const promises = accounts_info.map(async (acc_info) => {
      const result = await execute_withdraw(
        acc_info.loginUid,
        acc_info.loginSid,
        acc_info.phone_value,
        acc_info.sms_code,
        acc_info.phone_raw,
        acc_info.proxy
      );

      if (result.success) {
        const new_count = decrease_user_withdraw_count(middleware, userid);
        middleware.reply(`✅ 账号 ${acc_info.phone_masked} 提现成功: ${result.message}\n剩余提现次数: ${new_count}次`);
        return { success: true };
      } else {
        middleware.reply(`❌ 账号 ${acc_info.phone_masked} 提现失败: ${result.error}`);
        return { success: false };
      }
    });

    const results = await Promise.all(promises);
    success_count = results.filter(r => r.success).length;
    fail_count = results.filter(r => !r.success).length;

    middleware.reply(`批量提现完成\n✅ 成功: ${success_count}个\n❌ 失败: ${fail_count}个`);

  } else if (wait_choice.toLowerCase() === 'n') {
    // 直接执行提现
    for (const acc of selected_accounts) {
      try {
        const proxy = await proxy_manager.get_proxy();
        if (!proxy) {
          middleware.reply(`获取代理失败，跳过账号 ${acc.phone_masked}\n原因: ${proxy_manager.get_last_error()}`);
          continue;
        }

        const login_info = acc.login_info;
        if (!login_info) {
          middleware.reply(`账号 ${acc.phone_masked} 缺少登录信息，跳过`);
          continue;
        }

        const login_values = login_info.split('#');
        const phone = login_values[0];
        const password = login_values[1] || '';

        middleware.reply(`正在为账号 ${acc.phone_masked} 重新登录获取凭证...`);

        const { loginUid, loginSid, error } = await login_for_withdraw(phone, password);
        if (error) {
          middleware.reply(`账号 ${acc.phone_masked} 重新登录失败: ${error}`);
          continue;
        }

        const phone_value = encrypt_phone(phone);
        if (!phone_value) {
          middleware.reply(`账号 ${acc.phone_masked} 手机号加密失败`);
          continue;
        }

        const new_token = `${loginUid}#${phone}#${loginSid}#${phone_value}`;
        middleware.bucketSet('dd_KuwoTX_account', acc.account, new_token);
        middleware.reply(`账号 ${acc.phone_masked} 重新登录成功`);

        // 发送验证码
        const url = 'https://integralapi.kuwo.cn/api/v1/online/sign/v1/userBindPhone';
        const params = new URLSearchParams({
          loginUid,
          loginSid,
          mobile: phone_value
        }).toString();

        const headers = {
          'User-Agent': generate_kuwo_ua(phone),
          'Accept': 'application/json, text/plain, */*',
          'Origin': 'https://h5app.kuwo.cn',
          'Sec-Fetch-Mode': 'cors',
          'Sec-Fetch-Site': 'same-site',
          'Referer': 'https://h5app.kuwo.cn/apps/earning-sign/cash_out.html',
          'Sec-Fetch-Dest': 'empty',
          'Accept-Language': 'zh-CN,zh-Hans;q=0.9'
        };

        const response = await httpGet(`${url}?${params}`, { headers, proxy });
        if (response.statusCode !== 200) {
          middleware.reply(`账号 ${acc.phone_masked} 发送验证码失败`);
          continue;
        }

        const result = JSON.parse(response.body);

        const data_status = result.data?.status;
        const data_desc = result.data?.description || '';
        if (data_status === 0 && data_desc === '用户未登录') {
          middleware.reply(`账号 ${acc.phone_masked} 登录凭证已失效！\n请重新执行「1️⃣ 提交账号」绑定账号`);
          continue;
        }

        if (result.code !== 200) {
          middleware.reply(`账号 ${acc.phone_masked} 发送验证码失败: ${result.msg || '未知错误'}`);
          continue;
        }

        middleware.reply(`请输入账号 ${acc.phone_masked} 的验证码:`);
        const sms_code = await middleware.input(60000, 1, false);
        if (!sms_code) {
          middleware.reply(`账号 ${acc.phone_masked} 验证码输入超时`);
          continue;
        }

        // 执行提现
        const withdrawResult = await execute_withdraw(
          loginUid,
          loginSid,
          phone_value,
          sms_code,
          phone,
          proxy
        );

        if (withdrawResult.success) {
          const new_count = decrease_user_withdraw_count(middleware, userid);
          middleware.reply(`✅ 账号 ${acc.phone_masked} 提现成功: ${withdrawResult.message}\n剩余提现次数: ${new_count}次`);
        } else {
          middleware.reply(`❌ 账号 ${acc.phone_masked} 提现失败: ${withdrawResult.error}`);
        }
      } catch (e) {
        middleware.reply(`处理账号 ${acc.phone_masked} 时出错: ${e.message}`);
        continue;
      }
    }
  } else {
    middleware.reply('输入无效,已取消操作');
  }
}

/**
 * 用户授权功能
 */
async function userAuthorize(middleware) {
  middleware.reply(
    '=====用户授权=====\n' +
    '1️⃣ 单用户授权\n' +
    '2️⃣ 全部用户授权\n' +
    '⚠️ 输入q退出操作\n' +
    '==================='
  );

  const auth_choice = await middleware.input(60000, 1, false);
  if (auth_choice.toLowerCase() === 'q') {
    middleware.reply('退出操作');
    return;
  }

  try {
    const auth_num = parseInt(auth_choice);

    if (auth_num === 1) {
      // 单用户授权
      middleware.reply('请输入用户ID:');
      const target_userid = await middleware.input(60000, 1, false);
      if (!target_userid) {
        middleware.reply('输入超时');
        return;
      }

      const target_uservalue = middleware.bucketGet('dd_KuwoTX_bind', target_userid);
      if (!target_uservalue) {
        middleware.reply('该用户未绑定任何账号');
        return;
      }

      const current_count = middleware.bucketGet('dd_KuwoTX_UserCount', target_userid) || '0';
      const accounts = JSON.parse(target_uservalue);

      let message = '=====用户信息=====\n';
      message += `👤 用户ID: ${target_userid}\n`;
      message += `🔢 当前次数: ${current_count}次\n`;
      message += `📱 绑定账号数: ${accounts.length}个\n`;
      message += '-------------------\n';

      for (let i = 0; i < accounts.length; i++) {
        const login_info = middleware.bucketGet('dd_KuwoTX_login', accounts[i]);
        let phone;
        if (login_info) {
          phone = login_info.split('#')[0];
        } else {
          const token = middleware.bucketGet('dd_KuwoTX_account', accounts[i]);
          if (!token) continue;
          phone = token.split('#')[0];
        }
        const phone_masked = phone.substring(0, 3) + '****' + phone.substring(7);
        message += `[${i + 1}] 账号: ${phone_masked}\n`;
      }

      message += '-------------------\n';
      message += '请输入充值次数:';
      middleware.reply(message);

      const count_input = await middleware.input(60000, 1, false);
      if (!count_input) {
        middleware.reply('输入超时');
        return;
      }

      try {
        const count = parseInt(count_input);
        if (count <= 0) {
          middleware.reply('充值次数必须大于0');
          return;
        }

        const new_count = empower(middleware, target_userid, count);
        middleware.reply(
          '=====充值成功=====\n' +
          `👤 用户ID: ${target_userid}\n` +
          `🔢 充值次数: ${count}次\n` +
          `📊 当前可用次数: ${new_count}次\n` +
          '==================='
        );
      } catch (e) {
        middleware.reply('充值次数必须为数字');
      }
    } else if (auth_num === 2) {
      // 全部用户授权
      middleware.reply('请输入充值次数:');
      const count_input = await middleware.input(60000, 1, false);
      if (!count_input) {
        middleware.reply('输入超时');
        return;
      }

      try {
        const count = parseInt(count_input);
        if (count <= 0) {
          middleware.reply('充值次数必须大于0');
          return;
        }

        const all_binds = middleware.bucketAll('dd_KuwoTX_bind');
        if (!all_binds || Object.keys(all_binds).length === 0) {
          middleware.reply('没有找到任何用户绑定信息');
          return;
        }

        let success_count = 0;
        let failed_count = 0;
        let result_message = '=====全部授权结果=====\n';

        for (const [user_id, uservalue] of Object.entries(all_binds)) {
          try {
            const new_count = empower(middleware, user_id, count);
            success_count++;
            result_message += `✅ 用户 ${user_id}: 充值成功，当前次数 ${new_count}\n`;
          } catch (e) {
            failed_count++;
            result_message += `❌ 用户 ${user_id} 处理失败: ${e.message}\n`;
          }
        }

        result_message += '-------------------\n';
        result_message += `📊 统计信息:\n`;
        result_message += `👤 用户总数: ${Object.keys(all_binds).length}\n`;
        result_message += `✅ 成功充值: ${success_count}\n`;
        result_message += `❌ 充值失败: ${failed_count}\n`;
        result_message += `🔢 充值次数: ${count}次\n`;
        result_message += '===================';
        middleware.reply(result_message);
      } catch (e) {
        middleware.reply('充值次数必须为数字');
      }
    } else {
      middleware.reply('输入无效');
    }
  } catch (e) {
    middleware.reply('输入无效');
  }
}

/**
 * 支付处理
 */
async function zf(middleware, config, project, count, user_id) {
  if (config.KuwoTXmoney === 0) {
    return true;
  }

  const money = count * config.KuwoTXmoney;
  const user_points = middleware.bucketGet('dd_sign_points', middleware.getUserID()) || '0';
  const jfsl = middleware.bucketGet('dd_KuwoTX_PluginsData', 'KuwoTXcoin') || '200';
  const total_points = parseInt(jfsl) * count;

  let pay_menu = '=====选择支付方式=====';
  let option_num = 1;
  const options_map = {};

  // 这里简化处理，实际需要根据配置显示支付方式
  pay_menu += `\n${option_num}️⃣ 微信支付\n   💰 ${money}元/${count}次`;
  options_map[String(option_num)] = 'wechat';
  option_num++;

  if (total_points > 0) {
    pay_menu += `\n${option_num}️⃣ 积分支付\n   🎯 ${total_points}积分/${count}次\n   💫 当前积分: ${user_points}`;
    options_map[String(option_num)] = 'points';
  }

  pay_menu += '\n-------------------\n回复数字选择方式\n回复\'q\'退出操作\n===================';
  middleware.reply(pay_menu);

  const choice = await middleware.input(60000, 1, false);
  if (!choice) {
    middleware.reply('输入超时');
    return false;
  }
  if (choice.toLowerCase() === 'q') {
    middleware.reply('退出支付');
    return false;
  }

  const selected_pay = options_map[choice];

  if (selected_pay === 'wechat') {
    // 微信支付流程
    middleware.reply(
      '=====订单信息=====\n' +
      `🎈名称:${project}\n` +
      `🎉数量:${count}次\n` +
      `💰应付:${money}元\n` +
      '⚠️ 输入q退出支付\n' +
      '==================='
    );

    const ddzf = await middleware.waitPay('q', 100 * 1000);
    if (ddzf === 'q') {
      middleware.reply('退出支付');
      return false;
    }

    try {
      let payData;
      if (typeof ddzf === 'string') {
        try {
          payData = JSON.parse(ddzf);
        } catch (e) {
          if (ddzf.includes('二维码赞赏到账')) {
            const amount = ddzf.split('收款金额￥')[1].split('\n')[0];
            const pay_t = ddzf.split('到账时间')[1].split('\n')[0];
            payData = { Money: parseFloat(amount), Time: pay_t.trim() };
          } else {
            middleware.reply('解析收款信息失败');
            return false;
          }
        }
      } else {
        payData = ddzf;
      }

      const Money = parseFloat(payData.Money || payData.money || 0);
      const pay_time = payData.Time || payData.time || new Date().toISOString().replace('T', ' ').split('.')[0];

      if (Money >= money) {
        const new_count = empower(middleware, user_id, count);
        middleware.reply(
          '=====支付成功=====\n' +
          `🎈 商品: ${project}\n` +
          `🎉 次数: ${count}次\n` +
          `💰 支付: ${Money}元\n` +
          `⏰ 时间: ${pay_time}\n` +
          `🔢 当前可提现次数: ${new_count}次\n` +
          '==================='
        );
        return true;
      } else {
        middleware.reply(`支付金额错误\n应付:${money}元\n实付:${Money}元\n请联系管理员处理退款！`);
        return false;
      }
    } catch (e) {
      middleware.reply(`处理支付结果时出错: ${e.message}`);
      return false;
    }
  } else if (selected_pay === 'points') {
    // 积分支付
    const current_points = parseInt(user_points);
    if (current_points < total_points) {
      middleware.reply(
        '=====积分不足=====\n' +
        `💰 当前积分: ${current_points}\n` +
        `💵 所需积分: ${total_points}\n` +
        '==================='
      );
      return false;
    }

    middleware.reply(
      '=====积分支付=====\n' +
      `💰 当前积分: ${user_points}\n` +
      `💵 所需积分: ${total_points}\n` +
      `💡 购买次数: ${count}次\n` +
      '是否确认支付?\n' +
      '[y]确认 | [n]取消'
    );

    const confirm = await middleware.input(60000, 1, false);
    if (confirm.toLowerCase() === 'y' || confirm.toLowerCase() === '是') {
      const new_balance = current_points - total_points;
      middleware.bucketSet('dd_sign_points', middleware.getUserID(), String(new_balance));
      const new_count = empower(middleware, user_id, count);
      middleware.reply(
        '=====支付成功=====\n' +
        `🎈 商品: ${project}\n` +
        `🎉 次数: ${count}次\n` +
        `💰 支付: ${total_points}积分\n` +
        `💎 剩余: ${new_balance}积分\n` +
        `🔢 当前可提现次数: ${new_count}次\n` +
        '==================='
      );
      return true;
    } else {
      return false;
    }
  } else {
    middleware.reply('输入无效');
    return false;
  }
}

/**
 * 检查授权状态
 */
function check_authorization(middleware) {
  const all_binds = middleware.bucketAll('dd_KuwoTX_bind');
  if (!all_binds || Object.keys(all_binds).length === 0) return;

  let total_zero_count = 0;

  for (const [userid, uservalue] of Object.entries(all_binds)) {
    try {
      const user_withdraw_count = get_user_withdraw_count(middleware, userid);
      if (parseInt(user_withdraw_count) <= 0) {
        total_zero_count++;
        const accounts = JSON.parse(uservalue);
        const account_info = [];

        for (const account of accounts) {
          const login_info = middleware.bucketGet('dd_KuwoTX_login', account);
          if (login_info) {
            const phone = login_info.split('#')[0];
            account_info.push(phone.substring(0, 3) + '****' + phone.substring(7));
          } else {
            const token = middleware.bucketGet('dd_KuwoTX_account', account);
            if (!token) continue;
            const phone = token.split('#')[0];
            account_info.push(phone.substring(0, 3) + '****' + phone.substring(7));
          }
        }

        let message = '=====酷我提现次数不足通知=====\n';
        message += `🔢 当前可用次数: ${user_withdraw_count}次\n`;
        message += `📱 绑定账号数: ${account_info.length}个\n`;
        if (account_info.length > 0) {
          message += '-------------------\n';
          message += '绑定的账号:\n';
          for (let i = 0; i < account_info.length; i++) {
            message += `[${i + 1}] ${account_info[i]}\n`;
          }
        }
        message += '-------------------\n';
        message += '请及时充值以继续使用提现功能\n';
        message += '===================';

        try {
          // 这里需要根据实际的middleware实现发送通知
          console.log(`[通知] 向用户 ${userid} 发送通知: ${message}`);
        } catch (e) {
          console.log(`[通知] 向用户 ${userid} 发送通知失败`);
        }
      }
    } catch (e) {
      console.log(`[错误] 处理用户 ${userid} 时出错: ${e.message}`);
      continue;
    }
  }

  console.log(`[通知] 提现次数检测完毕，共发现 ${total_zero_count} 个提现次数不足用户`);
}

// 导出模块
module.exports = {
  main,
  login,
  login_for_withdraw,
  verify_withdraw,
  execute_withdraw,
  ProxyManager,
  encrypt_phone,
  generate_kuwo_ua,
  get_beijing_time,
  precision_wait
};

// 如果直接运行此文件
if (require.main === module) {
  main().catch(console.error);
}
