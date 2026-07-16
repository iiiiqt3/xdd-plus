# XDD 后端架构索引

> **用途**：供 AI / 新开发者快速定位模块与文件职责。  
> **仓库**：`github.com/cdle/xdd` · **入口**：`main.go` · **框架**：Beego v2 + GORM  
> **最后更新**：2026-07-16（应用宝青龙网关能力补全后）

---

## 1. 30 秒总览

```
客户端（门户 / 管理后台 / 青龙脚本 / 机器人）
        ↓
   main.go 路由 (Beego)
        ↓
   controllers/          ← HTTP 薄层，鉴权，参数解析
        ↓
   models/              ← 业务核心：DB、青龙、门户、京东、协议网关
        ↓
   yybportal/ ──→ yyb/  ← 应用宝协议栈（MMTLS、扫码、wxapp、公众号、刷步）
        ↓
   wechat08 / 青龙面板   ← 外部协议与任务面板
```

| 子系统 | 目录 | 一句话 |
|--------|------|--------|
| HTTP 入口 | `main.go` | 注册全部路由，异步启动 `yybportal.Init()` |
| 控制器 | `controllers/` | 门户、管理、登录、兼容网关、机器人 webhook |
| 领域层 | `models/` | 配置、DB、cron、门户、京东、青龙、协议分流 |
| 应用宝核心 | `yyb/` | 独立协议实现，不直接挂路由 |
| 应用宝胶水 | `yybportal/` | 连接 models ↔ yyb，门户/京东/cron/网关回调 |
| 前端静态 | `vweb/` | portal/admin HTML+JS，磁盘优先、embed 兜底 |
| 运行时配置 | `conf/` | `config.yaml`、activities、replies（gitignore，部署自备） |

---

## 2. 启动顺序

1. `models` 包 `init()`：`init.go` 串联 logger → config → DB → cron → bot → JD 调度器 → 青龙同步
2. `main()`：注册路由、静态目录、CORS
3. 后台 goroutine：`yybportal.Init()` 启动应用宝（失败不阻塞主服务）
4. 后台 goroutine：JD Cookie 批量落盘 `models.Save`

---

## 3. 路由地图（按业务）

### 3.1 门户用户 `/api/portal/*`

| 前缀 | 控制器 | 功能 |
|------|--------|------|
| `/api/portal/dashboard` 等 | `PortalController` | 仪表盘、项目、签到、积分、通知 |
| `/api/portal/wx/*` | `PortalController` | 微信协议设备：扫码、重登、唤醒、轮询 |
| `/api/portal/jd/*` | `PortalController` | 京东账号、短信登录、任务执行、日志 |
| `/api/portal/jd/yyb/*` | `PortalController` | 京东 CK 走应用宝刷新 |
| `/api/portal/protocol/*` | `PortalController` | 微信↔应用宝双绑、代理地区 |
| `/api/portal/kuwo/*` | `PortalController` | 酷我提现 |
| `/api/portal/yyb/*` | `PortalYybController` | 应用宝扫码、账号、wxapp 调试 |

页面：`/portal` → `vweb/html/portal.html`

### 3.2 管理后台 `/api/admin/*`

| 前缀 | 控制器 | 功能 |
|------|--------|------|
| `/api/admin/*` | `AdminApiController` | 用户、活动、青龙、京东容器、CK、日志、cron、通知 |
| `/api/admin/yyb/*` | `AdminYybController` | 应用宝账号、检测、预热、扫码调试 |
| `/api/admin/crontasks/*` | `AdminApiController` | 青龙面板定时任务 |
| `/api/admin/jdcrontasks/*` | `AdminApiController` | 京东容器定时任务 |

页面：`/admin` → `vweb/html/admin.html`（内嵌 API 文档，数据在 `vweb/js/api_docs_data.js`）

### 3.3 青龙兼容网关（核心）

**控制器**：`WxCompatProxyController`（`wx_compat_proxy.go` + `wx_compat_capability.go`）

**路径**：`CompatGatewayPaths()` 约 80 条，含 `/api/v1/wx/*`、`/api/Wxapp/*`、`/api/OfficialAccounts/*`、`/TenPay/*` 等。

**分流逻辑**：

```
请求带 ref（wxid / openid）
  → models.ShouldRouteToYyb(ref)
      true  → 应用宝（yybportal → yyb）
      false → 转发 wechat08（models.ForwardWxProtoRequest）
  → 应用宝不支持 + 双绑账号 → 可回退 wechat08
```

**应用宝网关能力模块**（已与 xdd-G-1 对齐）：

| 模块 | 处理文件 | 底层 |
|------|----------|------|
| User | `wx_compat_proxy.go` | status / delete / getLatestUserKey |
| Wxapp | `wx_compat_proxy.go` | getCode、getPhone、operateWxData、callFunction |
| Session | `wx_compat_capability.go` | runtimeSession |
| 公众号 | `wx_compat_capability.go` | `yyb/internal/protocol/official.go` |
| Tools 刷步 | `wx_compat_capability.go` | `yyb/internal/protocol/step.go` |
| TenPay | `wx_compat_capability.go` | `yyb/internal/protocol/tenpay.go` |
| Refresh | `wx_compat_proxy.go` | login/again → 账号刷新 |

**两边均不支持**：设备登录 code/awake/twice/logout、小程序 addMobile/头像管理。

### 3.4 登录与遗留

| 路径 | 说明 |
|------|------|
| `/api/login/*` | 门户/管理员登录、注册、重置密码 |
| `/api/yyb/*` | 旧脚本 API（token），建议改用 `/api/v1/wx/*` |
| `/wx/receive` | 微信机器人 webhook |
| `/qq` | QQ 机器人 |
| `/api/account` |  legacy 管理员 CK 管理 |

---

## 4. 目录 → 文件索引

### 4.1 `controllers/`

| 文件 | 职责 |
|------|------|
| `base.go` | 基类：管理员/门户 session 鉴权、校验、JSON 响应 |
| `login.go` | 登录、注册、京东扫码、CK/SMS 登录 |
| `portal.go` | 门户主 API（最大文件之一） |
| `portal_yyb.go` | 门户应用宝页 API |
| `admin_api.go` | 管理后台 REST（最大文件之一） |
| `admin_yyb.go` | 管理后台应用宝 + 废弃 `YybScriptController` |
| `wx_compat_proxy.go` | 青龙网关入口、路径分类、Wxapp/User 处理 |
| `wx_compat_capability.go` | 网关能力：公众号、刷步、TenPay、session |
| `wx.go` / `wechat08_wx.go` | 机器人消息接入 |
| `account.go` / `config.go` | 遗留 CK、env 配置 API |
| `qq.go` | QQ echo |

### 4.2 `models/`（按领域）

**基础设施**：`init.go` `config.go` `db.go` `logger.go` `system.go` `cron.go` `cache.go` `signature.go`

**协议与网关**：`protocol_bind.go`（双绑） · `protocol_gateway.go`（分流、wechat08 转发、兼容响应包装） · `wechat08_util.go` `wechat08_reply.go`

**门户**：`web_portal.go` `web_user_account.go` `portal_wx_device.go` `wx_portal.go` `portal_jd_proxy.go` `web_notification.go` `push.go` `kuwo.go` `key.go`

**京东**：`jd_portal.go` `jd_task_scheduler.go` `jdtask.go` `jd_task_proxy.go` `jd_query.go` `handle.go` `container.go` `wskey.go` `wx_jd.go` `yyb_jd_bot.go`

**青龙**：`qinglong.go` `qinglong_sync.go` `jltask.go` `jltask_loader.go` `activity_project.go`

**应用宝配置**：`yyb_51proxy.go`（51 代理） · `config.go` 内 `YybConfig`

**机器人**：`bot.go` `command.go` `reply.go` `replies.go` `game.go` `tbot.go`

**管理**：`admin_service.go`（超大，管理端业务聚合）

### 4.3 `yyb/` 应用宝协议栈

```
yyb/
├── service.go          # Start/Close，全局 Service
├── export.go           # 对外 API：QR、账号、Wxapp、能力接口
├── capability_map.go   # map 入参 → protocol 结构（供网关）
├── config.go           # 模块默认配置
├── proxy_resolver.go   # 账号级 51 代理解析
├── qr_proxy_login.go   # 带代理的扫码登录
├── scan_store.go       # 扫码结果入库
└── internal/
    ├── httpapi/
    │   ├── app.go              # 核心：QR、账号 CRUD、wxapp、代理重试 5 次
    │   ├── capability_api.go   # official/tenpay/step/session/werun
    │   └── service_api.go      # Service 方法封装
    ├── protocol/
    │   ├── pool.go       # 连接池、session、run 业务
    │   ├── mmtls_*.go    # MMTLS 微信传输
    │   ├── ilink.go      # JSAPI 明文、transfer 包
    │   ├── official.go   # 公众号 CGI
    │   ├── tenpay.go     # 支付 CGI
    │   └── step.go       # 刷步 / 微信运动 / 硬件
    ├── store/            # 应用宝账号 GORM 模型
    └── qr/               # 二维码生成轮询
```

### 4.4 `yybportal/` 胶水层

| 文件 | 职责 |
|------|------|
| `init.go` | 迁移、启动 yyb、注册网关回调、JD cron |
| `gateway_register.go` | `models.SetProtocolYybHandlers` 注入 getCode/operate/refresh |
| `portal_api.go` | 门户：状态、账号列表、扫码三步、wxapp |
| `admin_api.go` | 管理：全站账号、一键检测、预热 |
| `script_api.go` | `/api/yyb/*` 旧脚本接口 |
| `capability_api.go` | 网关能力内部调用 |
| `qr_login.go` | 扫码会话、积分扣费 |
| `jd.go` / `jd_bridge.go` / `cron_jd.go` | 京东 CK 刷新、掉线检测 cron |
| `account_proxy.go` | 账号代理强制刷新回调 |
| `model.go` / `db.go` | `portal_yyb_bindings` 表 |

### 4.5 `vweb/` 前端

| 文件 | 职责 |
|------|------|
| `fs.go` | 读磁盘 vweb 或 embed 兜底 |
| `html/portal.html` | 用户门户 SPA |
| `html/admin.html` | 管理后台 SPA + API 文档 UI |
| `js/portal_yyb.js` | 应用宝门户 UI |
| `js/portal_protocol_bind.js` | 协议双绑 UI |
| `js/admin_yyb.js` | 应用宝管理 UI |
| `js/api_docs_data.js` | API 文档数据（`category`: xdd / 应用宝） |

---

## 5. 关键数据流

### 5.1 青龙脚本取小程序 code

```
脚本 POST /api/v1/wx/app/get/code  body.ref=openid
  → WxCompatProxyController
  → models.ShouldRouteToYyb → true
  → models.ProtocolGetWxAppCode
  → yybportal.InternalWxappGetCode
  → yyb.Service.WxappGetCode
  → protocol.Pool.GetCode（带代理重试）
```

### 5.2 门户应用宝扫码

```
POST /api/portal/yyb/qr → PortalYybController.CreateQR
  → yybportal.PortalCreateQR（积分预检、选地区）
  → GET poll → POST confirm
  → yybportal 写 portal_yyb_bindings + yyb 账号库
```

### 5.3 京东 CK 自动刷新（应用宝）

```
cron（yybportal/cron_jd.go，每 3h）
  → 遍历 portal_yyb_bindings
  → InternalWxappGetCode（京东 appId）
  → 写青龙环境变量 / 通知掉线
```

### 5.4 协议双绑

```
portal_protocol_bindings: wx_wxid ↔ yyb_openid（全局 1:1）
models.ResolveProtocolRoute(ref) → Backend: yyb | wechat
双绑账号：应用宝失败时可回退 wechat08 同路径
```

---

## 6. 配置与运行时文件

| 路径 | 说明 |
|------|------|
| `conf/config.yaml` | 主配置：容器、DB、YYB、微信协议、游戏、JD（**不提交 git**） |
| `conf/activities.yaml` | 青龙活动定义，热重载 |
| `conf/replies.yaml` | 机器人自动回复，热重载 |
| `conf/jd_manual_tasks.yaml` | 门户手动京东任务（运行时） |
| `.xdd.db` | 默认 SQLite |
| `yyb/resource/db/` | 应用宝本地 SQLite |
| `logs/` | 分类日志 |

配置结构体见 `models/config.go`（`Yaml`、`YybConfig`、`WxProtocolConfig` 等）。

---

## 7. 与 xdd-G-1 的差异（备忘）

| 项 | xdd | xdd-G-1 |
|----|-----|---------|
| 青龙网关应用宝能力 | ✅ 已对齐 | 基准 |
| 门户应用宝 API | 三步 qr/poll/confirm | scan_login + poll 单流程 |
| 协议转换 API | 双绑 `protocol/bind` | `protocol/convert/to_yyb` |
| 君品荟脚本 | 无（不需要） | `/scripts/junpinhui/*` |
| 双绑回退 wechat08 | ✅ 有 | 弱/无 |
| `proxy_business_enabled` | 按账号凭证自动 | 独立开关 |

---

## 8. 构建与开发

```bash
make deps    # go mod tidy
make build   # → ./xdd
make run     # 本地运行
```

- 改 `vweb/` 下 HTML/JS：**无需重新编译**，刷新浏览器即可（磁盘优先加载）。
- 改 Go 代码：需重新 `make build` 并重启进程。
- 更新 API 文档数据：`python3 scripts/update_api_docs.py`（若脚本存在）。

---

## 9. AI 检索建议

按问题类型直接跳转章节：

| 你想… | 看 |
|--------|-----|
| 加青龙网关接口 | §3.3 + `wx_compat_proxy.go` + `wx_compat_capability.go` |
| 改应用宝协议底层 | `yyb/internal/protocol/` + `httpapi/app.go` |
| 改门户应用宝扫码/扣费 | `yybportal/qr_login.go` `coin.go` `portal_yyb.go` |
| 改京东任务 | `models/jd_task_scheduler.go` `jd_portal.go` |
| 改双绑/路由 | `models/protocol_bind.go` `protocol_gateway.go` |
| 改管理后台 API | `controllers/admin_api.go` `models/admin_service.go` |
| 改前端门户 | `vweb/html/portal.html` + `vweb/js/*` |

---

## 10. 文档维护约定

**建议更新本文件的时机**（不必每次小改都动）：

- 新增/删除 **路由** 或 **顶层目录**
- **应用宝/协议网关** 能力块变更
- **models 大包职责** 调整（拆分/合并）
- **配置项** 结构性变化
- 与 xdd-G-1 **对齐状态** 变化

**可不更新**：单函数 bugfix、文案、样式、版本号小改。

维护时同步：
1. 改 `ARCHITECTURE.md` 对应章节
2. 若涉及 HTTP 接口：运行 `scripts/update_api_docs.py` 更新 `api_docs_data.js`
3. 更新文首「最后更新」日期

---

## 11. 相关文档

| 文件 | 内容 |
|------|------|
| 本文 `ARCHITECTURE.md` | 项目总索引（**应纳入 git**） |
| `vweb/js/api_docs_data.js` | 管理后台可检索的 API 列表 |
| `conf/jd_manual_tasks.yaml.example` | 手动京东任务配置示例 |
| `发送文本消息.md` | 外部微信机器人插件 API（非 xdd REST） |
