import UIKit
import Foundation
import Security
import CryptoKit
import WebKit


enum SignatureHelper {
    private static let secret = "G0uD0ng@2024#S1gn@ture!K3y"
    private static let salt = "x9D$kL2mN#pQ7rT5"
    static let version = "v2"
    private static let appVersion = "2.0.0"

    static func getHeaders(for path: String) -> [String: String] {
        let timestamp = String(Int(Date().timeIntervalSince1970))
        let nonce = generateNonce(length: 16)
        let deviceId = getDeviceId()
        let signature = generateSignature(
            timestamp: timestamp,
            nonce: nonce,
            deviceId: deviceId,
            appVersion: appVersion,
            path: path
        )

        return [
            "X-Sign-Timestamp": timestamp,
            "X-Sign-Nonce": nonce,
            "X-Sign-DeviceID": deviceId,
            "X-Sign-Value": signature,
            "X-Sign-Version": version,
            "X-App-Version": appVersion
        ]
    }

    private static func generateSignature(
        timestamp: String,
        nonce: String,
        deviceId: String,
        appVersion: String,
        path: String
    ) -> String {
        let round1Input = "\(timestamp)|\(nonce)|\(deviceId)|\(appVersion)|\(path)|\(salt)"
        let round1 = sha256(round1Input)

        let round2 = hmacSha256(message: round1, secret: secret)

        let reversedTimestamp = String(timestamp.reversed())
        let reversedNonce = String(nonce.reversed())
        let round3Input = "\(round2)\(reversedTimestamp)\(reversedNonce)"
        return sha256(round3Input)
    }

    private static func sha256(_ input: String) -> String {
        let data = Data(input.utf8)
        let hash = SHA256.hash(data: data)
        return hash.map { String(format: "%02x", $0) }.joined()
    }

    private static func hmacSha256(message: String, secret: String) -> String {
        let key = SymmetricKey(data: Data(secret.utf8))
        let data = Data(message.utf8)
        let authenticationCode = HMAC<SHA256>.authenticationCode(for: data, using: key)
        return authenticationCode.map { String(format: "%02x", $0) }.joined()
    }

    private static func generateNonce(length: Int) -> String {
        let charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
        return String((0..<length).compactMap { _ in charset.randomElement() })
    }

    private static func getDeviceId() -> String {
        if let uuid = UserDefaults.standard.string(forKey: "app.device.uuid") {
            return uuid
        }
        let uuid = UUID().uuidString
        UserDefaults.standard.set(uuid, forKey: "app.device.uuid")
        return uuid
    }

    private static func needsSignature(_ path: String) -> Bool {
        let signaturePaths = ["/api/portal/checkin", "/api/portal/pray"]
        return signaturePaths.contains { path.contains($0) }
    }
}


enum AppEnvironment {
    static let baseURL = URL(string: "http://180.152.5.230:5701")!
    static let coinPurchaseURL = URL(string: "http://180.152.5.230:8005/#/")!
    static let groupURL = URL(string: "https://qm.qq.com/q/4gYwV6YzPW")!
    static let moreWoolURL = URL(string: "https://h5.lot-ml.com/ProductEn/Index/7ee6c54f2d550fab")!
}


enum AppNotifications {
    static let sessionDidChange = Notification.Name("PortalSessionDidChange")
    static let sessionDidLogout = Notification.Name("PortalSessionDidLogout")
    static let sessionRequiresLogin = Notification.Name("PortalSessionRequiresLogin")
    static let jdAuthCallback = Notification.Name("JDAuthCallbackNotification")
    static let yybStatusDidUpdate = Notification.Name("PortalYybStatusDidUpdate")
    static let protocolBindDidUpdate = Notification.Name("PortalProtocolBindDidUpdate")
}


struct APIError: Error {
    let message: String
    let isUnauthorized: Bool
}

func sanitizeErrorMessage(_ message: String?) -> String {
    guard let raw = message?.trimmingCharacters(in: .whitespacesAndNewlines), !raw.isEmpty else {
        return "操作失败"
    }
    let lower = raw.lowercased()
    if lower.contains("failed to connect")
        || lower.contains("could not connect")
        || lower.contains("timed out")
        || lower.contains("timeout")
        || lower.contains("network connection was lost")
        || lower.contains("not connected to internet")
        || lower.contains("unable to resolve")
        || lower.contains("econnrefused")
        || lower.contains("enotfound")
        || lower.contains("enetunreach")
        || lower.contains("nsurlerror")
        || lower.contains("failed to fetch")
    {
        return "连接服务器失败"
    }
    var msg = raw
    let ipPattern = #"https?://\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?[^\s]*"#
    let hostPattern = #"\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?"#
    let urlPattern = #"https?://[^\s]+"#
    msg = msg.replacingOccurrences(of: ipPattern, with: "服务器", options: .regularExpression)
    msg = msg.replacingOccurrences(of: hostPattern, with: "服务器", options: .regularExpression)
    msg = msg.replacingOccurrences(of: urlPattern, with: "服务器地址", options: .regularExpression)
    return msg
}


struct EmptyPayload: Decodable {}


struct APIEnvelope<T: Decodable>: Decodable {
    let code: Int
    let msg: String?
    let data: T?
}

struct PortalAccessInfo: Decodable {
    let allowed: Bool
    let coin: Int?
    let requiredCoin: Int?
    let gapCoin: Int?
    let message: String?
}

struct PortalAPIResponse<T: Decodable>: Decodable {
    let code: Int
    let msg: String?
    let data: T?
    let portalAccess: PortalAccessInfo?
}

final class PortalAccessStore {
    static let shared = PortalAccessStore()
    private(set) var current: PortalAccessInfo?
    func update(_ info: PortalAccessInfo?) {
        current = info
        NotificationCenter.default.post(name: .portalAccessDidChange, object: nil)
    }
}

extension Notification.Name {
    static let portalAccessDidChange = Notification.Name("portalAccessDidChange")
}


struct PortalDashboard: Decodable {
    let number: Int
    let qq: String?
    let wxid: String?
    let coin: Int
    let nickname: String?
    let accountId: Int?
    let username: String?
    let boundAt: String?
    let lastLoginAt: String?
    let availableCount: Int
    let projectCount: Int
    let joinedCount: Int
    let activeCount: Int
    let validCkCount: Int
    let expiringCount: Int
    let expiredCount: Int
    let notificationTotal: Int?
    let notificationUnread: Int?
    let checkedInToday: Bool?
    let continuousDays: Int?
    let nextCheckInBonus: Int?
    let daysUntilNextCheckInBonus: Int?
    let prayedToday: Bool?
    let canCheckIn: Bool?
    let canCheckInMessage: String?
    let todayCheckInCount: Int?
}


struct PortalNotification: Decodable {
    let id: Int
    let title: String?
    let content: String?
    let category: String?
    let source: String?
    let isRead: Bool
    let displayType: String?
    let isTop: Bool?
    let createdAt: String?
    let readAt: String?
}


struct PortalNotificationPage: Decodable {
    let list: [PortalNotification]
    let total: Int
    let unread: Int
}


struct WechatRechargeTier: Decodable {
    let yuan: Int
    let fen: Int
    let points: Int
}


struct WechatRechargeConfig: Decodable {
    let enabled: Bool
    let canRecharge: Bool
    let blockReason: String?
    let billAccountOnline: Bool
    let billAccountMessage: String?
    let pointsPerYuan: Int?
    let tiers: [WechatRechargeTier]?
    let timeoutMinutes: Int?
    let externalPurchaseUrl: String?
    let paymentNotice: [String]?

    enum CodingKeys: String, CodingKey {
        case enabled
        case canRecharge = "can_recharge"
        case blockReason = "block_reason"
        case billAccountOnline = "bill_account_online"
        case billAccountMessage = "bill_account_message"
        case pointsPerYuan = "points_per_yuan"
        case tiers
        case timeoutMinutes = "timeout_minutes"
        case externalPurchaseUrl = "external_purchase_url"
        case paymentNotice = "payment_notice"
    }
}


struct WechatRechargeOrder: Decodable {
    let orderNo: String
    let requestedFen: Int
    let paymentFen: Int
    let paidFen: Int?
    let points: Int?
    let status: String
    let createdAt: String?
    let expiresAt: String?
    let remainingSec: Int64?
    let coin: Int?
    let qrcodeUrl: String?
    let paidAt: String?
    let lastError: String?

    enum CodingKeys: String, CodingKey {
        case orderNo = "order_no"
        case requestedFen = "requested_fen"
        case paymentFen = "payment_fen"
        case paidFen = "paid_fen"
        case points
        case status
        case createdAt = "created_at"
        case expiresAt = "expires_at"
        case remainingSec = "remaining_sec"
        case coin
        case qrcodeUrl = "qrcode_url"
        case paidAt = "paid_at"
        case lastError = "last_error"
    }
}


struct WechatRechargeHistoryPage: Decodable {
    let list: [WechatRechargeOrder]
    let total: Int
}


struct WechatRechargeCreateResult {
    let order: WechatRechargeOrder
    let replacedPrevious: Bool
    let message: String?
}


struct SubmitFeedbackPayload: Encodable {
    let type: String
    let title: String
    let content: String
    let contact: String
}


struct PortalJdAccount: Decodable {
    let index: Int
    let pin: String?
    let nickname: String?
    let statusText: String?
    let valid: Bool
}


struct PortalJdSmsVerifyResult: Decodable {
    let message: String?
    let queryResult: String?
    let needIdVerify: Bool?
}


struct PortalJdWxDevice: Decodable {
    let index: Int
    let wxid: String?
    let nickname: String?
    let device: String?
    let serverType: String?
    let jdNickname: String?
}


struct PortalJdWxRefreshResult: Decodable {
    let success: Int?
    let fail: Int?
    let details: [String]?
    let needRiskVerify: Bool?
    let riskUrl: String?
    let riskMsg: String?

    private enum CodingKeys: String, CodingKey {
        case success, fail, details, needRiskVerify, riskUrl, riskMsg
        case riskUrlSnake = "risk_url"
        case jmpUrl = "jmp_url"
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        success = try container.decodeIfPresent(Int.self, forKey: .success)
        fail = try container.decodeIfPresent(Int.self, forKey: .fail)
        details = try container.decodeIfPresent([String].self, forKey: .details)
        needRiskVerify = try container.decodeIfPresent(Bool.self, forKey: .needRiskVerify)
        riskMsg = try container.decodeIfPresent(String.self, forKey: .riskMsg)
        riskUrl = try container.decodeIfPresent(String.self, forKey: .riskUrl)
            ?? container.decodeIfPresent(String.self, forKey: .riskUrlSnake)
            ?? container.decodeIfPresent(String.self, forKey: .jmpUrl)
    }
}

struct PortalJdYybAccount: Decodable {
    let index: Int
    let openid: String?
    let nickname: String?
    let status: String?
    let jdNickname: String?
    let proxyRegionCode: String?
    let proxyRegionName: String?
}

struct PortalYybAccount: Decodable {
    let bindingId: Int64?
    let yybAccountId: Int64?
    let openid: String?
    let uin: Int64?
    let nickname: String?
    let avatarUrl: String?
    let status: String?
    let lastCheckedAt: Int64?
    let createdAt: Int64?
    let loginAt: Int64?
    let expiresAt: Int64?
    let proxyRegionCode: String?
    let proxyRegionName: String?
}

struct PortalProtocolBinding: Decodable {
    let id: Int64?
    let userNumber: Int?
    let wxWxid: String?
    let yybOpenId: String?
    let nickname: String?
}

struct PortalProtocolBindQuota: Decodable {
    let onlineWxSlots: Int?
    let yybAccounts: Int?
    let boundPairs: Int?
    let freeSlots: Int?
    let scanLoginCost: Int?
    let scanCostHint: String?
}

struct PortalProxyConfig: Decodable {
    let proxyEnabled: Bool?
    let proxyAccountConfigured: Bool?
    let proxyDefaultPackid: String?
    let proxyBypassRegionName: String?
}

struct PortalYybCheckSummary: Decodable {
    let total: Int?
    let alive: Int?
    let dead: Int?
    let failed: Int?
    let cooldown: Int?
    let message: String?
}

func formatYybCheckSummary(_ summary: PortalYybCheckSummary?) -> String? {
    guard let s = summary else { return nil }
    let total = s.total ?? 0
    if total <= 0 { return "暂无绑定账号" }
    let alive = s.alive ?? 0
    let dead = s.dead ?? 0
    let failed = s.failed ?? 0
    let cooldown = s.cooldown ?? 0
    let hint = (s.message ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    if !hint.isEmpty, cooldown >= total, failed >= total {
        return hint
    }
    var msg = "检测完成：共 \(total) 个，可用 \(alive) 个"
    if dead > 0 { msg += "，失效 \(dead) 个" }
    if cooldown > 0 { msg += "，冷却中 \(cooldown) 个" }
    let otherFailed = failed - cooldown
    if otherFailed > 0 { msg += "，失败 \(otherFailed) 个" }
    if !hint.isEmpty, otherFailed > 0 { msg += "（\(hint)）" }
    return msg
}

struct PortalYybStatus: Decodable {
    let enabled: Bool?
    let ready: Bool?
    let message: String?
    let coin: Int?
    let scanLoginCost: Int?
    let maxAccounts: Int?
    let accounts: [PortalYybAccount]?
    let checkSummary: PortalYybCheckSummary?
}

struct PortalYybQrCreateResult: Decodable {
    let sessionId: String?
    let status: String?
    let imageBase64: String?
    let scanLoginCost: Int?
    let scanCostHint: String?
}

struct PortalYybQrPollResult: Decodable {
    let status: String?
    let message: String?
}

struct PortalYybConfirmResult: Decodable {
    let account: PortalYybAccount?
    let cost: Int?
    let alreadyBound: Bool?
}

struct PortalJdTaskExecuteResult: Decodable {
    let taskId: String?
}

struct PortalJdTaskItem: Decodable {
    let id: String?
    let name: String?
    let coin: Int?
    let order: Int?
}

struct PortalJdProxyStatus: Decodable {
    let active: Bool?
    let expireAt: String?
    let monthlyCoin: Int?
    let userCoin: Int?
    let proxyReady: Bool?
}

struct KuwoAccountInfo: Decodable {
    let phone: String?
    let password: String?
}

struct KuwoCredentials: Decodable {
    let phone: String?
    let password: String?
    let accounts: [KuwoAccountInfo]?
}

struct KuwoScheduleResult: Decodable {
    let taskId: String?
    let targetHour: Int?
    let executeAt: String?
    let status: String?
    let reused: Bool?
}

struct KuwoTaskLog: Decodable {
    let time: String?
    let level: String?
    let message: String?
    let proxyHost: String?
}

struct KuwoWithdrawTask: Decodable {
    let id: String?
    let phone: String?
    let quotaID: String?
    let targetHour: Int?
    let executeAt: String?
    let status: String?
    let immediate: Bool?
    let smsFatal: Bool?
    let smsEditable: Bool?
    let logs: [KuwoTaskLog]?
}


struct PortalProfile: Decodable {
    let user: PortalUser?
    let account: PortalAccount?
}


struct PortalUser: Decodable {
    let Number: Int?
    let QQ: String?
    let Wxid: String?
    let Coin: Int?
    let Class: String?
    let Nickname: String?
}


struct PortalAccount: Decodable {
    let ID: Int?
    let Username: String?
    let UserNumber: Int?
    let BoundAt: String?
    let LastLoginAt: String?
}


struct PortalActivityField: Decodable {
    let key: String
    let prompt: String
    let required: Bool?
    let trimSpace: Bool?
    let timeoutSec: Int?
    let errorMsg: String?
}


struct PortalActivity: Decodable {
    let id: String
    let name: String
    let envKey: String?
    let needCoin: Int?
    let isMonthlyDeduct: Bool?
    let monthlyCoin: Int?
    let isDailyDeduct: Bool?
    let dailyCoin: Int?
    let minDays: Int?
    let qingLongConfig: String?
    let guide: String?
    let inputFields: [PortalActivityField]?
    let category: String?
    let ckTemplate: String?
    let isProtocolActivity: Bool?
}


struct ProtocolAccountOption: Decodable {
    let id: String
    let label: String
    let nickname: String?
    let mode: String?
    let wxid: String?
    let openid: String?
    let fillRef: String
    let usedInActivity: Bool?
    let selectable: Bool?
}


struct PortalProject: Decodable {
    let activityId: String
    let activityName: String?
    let envKey: String?
    let qingLongConfig: String?
    let remark: String?
    let displayName: String?
    let envValue: String?
    let expireDate: String?
    let status: Int?
    let statusText: String?
    let updatedAt: String?
    let createdAt: String?
    let isMonthlyDeduct: Bool?
    let monthlyCoin: Int?
    let isDailyDeduct: Bool?
    let dailyCoin: Int?
    let needCoin: Int?
    let grantExpireDate: String?
    let bizStatus: String?
    let bizStatusText: String?
    let daysLeft: Int?
    let priceText: String?
    let inputFields: [PortalActivityField]?
    let ckTemplate: String?
    let isProtocolActivity: Bool?
}


struct PortalWechatStatus: Decodable {
    let nickname: String?
    let wxid: String?
    let device: String?
    let status: String?
    let online: Bool?
    let loginTime: String?
    let refreshTime: String?
}

struct PortalWxDevice: Decodable {
    let id: Int
    let wxid: String?
    let nickname: String?
    let device: String?
    let status: String?
    let online: Bool?
    let isPrimary: Bool?
    let loginTime: String?
    let refreshTime: String?
}

struct CoinLog: Decodable {
    let id: Int
    let amount: Int
    let balanceAfter: Int
    let type: String?
    let detail: String?
    let source: String?
    let createdAt: String?
}


struct PortalWechatActionResult: Decodable {
    let message: String?
    let status: PortalWechatStatus?
    let qrBase64: String?
    let uuid: String?
    let cost: Int?
    let needPoll: Bool?

    enum CodingKeys: String, CodingKey {
        case message, status
        case qrBase64 = "qrBase64"
        case uuid
        case cost
        case needPoll = "needPoll"
    }
}


struct PortalHomePayload: Decodable {
    let dashboard: PortalDashboard
    let profile: PortalProfile
}


struct PortalHomeSnapshot {
    let dashboard: PortalDashboard
    let profile: PortalProfile
    let wechatStatus: PortalWechatStatus?
    let wxDevices: [PortalWxDevice]
    let yybStatus: PortalYybStatus?
    let protocolBindings: [PortalProtocolBinding]
}


final class CookieStorageManager {
    static let shared = CookieStorageManager()

    private let storageKey = "portal.saved.cookies"
    private init() {}

    func restoreCookies() {
        guard let saved = UserDefaults.standard.array(forKey: storageKey) as? [[String: Any]] else { return }
        for item in saved {
            var properties: [HTTPCookiePropertyKey: Any] = [:]
            for (key, value) in item {
                properties[HTTPCookiePropertyKey(key)] = value
            }
            if let cookie = HTTPCookie(properties: properties) {
                HTTPCookieStorage.shared.setCookie(cookie)
            }
        }
    }

    func persistCookies(for host: String) {
        guard let cookies = HTTPCookieStorage.shared.cookies else { return }
        let matched = cookies.filter { cookie in
            cookie.domain.contains(host) || host.contains(cookie.domain.replacingOccurrences(of: ".", with: ""))
        }
        let payload = matched.compactMap { cookie -> [String: Any]? in
            guard let properties = cookie.properties else { return nil }
            var mapped: [String: Any] = [:]
            for (key, value) in properties {
                mapped[key.rawValue] = value
            }
            return mapped
        }
        UserDefaults.standard.set(payload, forKey: storageKey)
    }

    func clearCookies() {
        HTTPCookieStorage.shared.cookies?.forEach { HTTPCookieStorage.shared.deleteCookie($0) }
        UserDefaults.standard.removeObject(forKey: storageKey)
    }
}


final class CredentialStore {
    static let shared = CredentialStore()

    private let service = "com.feiniao.portal.credentials"
    private let usernameAccount = "portal.username"
    private let passwordAccount = "portal.password"

    struct Credentials {
        let username: String
        let password: String
    }

    private init() {}

    func save(username: String, password: String) {
        saveValue(username, account: usernameAccount)
        saveValue(password, account: passwordAccount)
    }

    func load() -> Credentials? {
        guard let username = readValue(account: usernameAccount), let password = readValue(account: passwordAccount) else {
            return nil
        }
        return Credentials(username: username, password: password)
    }

    func clear() {
        deleteValue(account: usernameAccount)
        deleteValue(account: passwordAccount)
    }

    private func saveValue(_ value: String, account: String) {
        guard let data = value.data(using: .utf8) else { return }
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(query as CFDictionary)
        var payload = query
        payload[kSecValueData as String] = data
        payload[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlock
        SecItemAdd(payload as CFDictionary, nil)
    }

    private func readValue(account: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        guard status == errSecSuccess, let data = item as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private func deleteValue(account: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(query as CFDictionary)
    }
}


enum PortalRequestMeta {
    static let source = "app"
    static let platform = "ios"

    static var appVersion: String {
        Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "0"
    }

    static var defaultHeaders: [String: String] {
        [
            "X-Request-Source": source,
            "X-Client-Platform": platform,
            "X-App-Version": appVersion,
        ]
    }

    static func merged(with headers: [String: String]) -> [String: String] {
        var out = defaultHeaders
        headers.forEach { out[$0.key] = $0.value }
        return out
    }
}


final class APIClient {
    static let shared = APIClient()

    private let session: URLSession

    private init() {
        let configuration = URLSessionConfiguration.default
        configuration.httpCookieStorage = HTTPCookieStorage.shared
        configuration.httpShouldSetCookies = true
        configuration.requestCachePolicy = .reloadIgnoringLocalCacheData
        session = URLSession(configuration: configuration)
    }

    func requestEnvelope<T: Decodable>(
        path: String,
        method: String = "GET",
        headers: [String: String] = [:],
        body: Data? = nil,
        completion: @escaping (Result<APIEnvelope<T>, APIError>) -> Void
    ) {
        guard let url = URL(string: path, relativeTo: AppEnvironment.baseURL) else {
            completion(.failure(APIError(message: "请求地址无效", isUnauthorized: false)))
            return
        }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.httpBody = body
        request.timeoutInterval = 30
        PortalRequestMeta.merged(with: headers).forEach { request.setValue($1, forHTTPHeaderField: $0) }

        session.dataTask(with: request) { data, response, error in
            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: sanitizeErrorMessage(error.localizedDescription), isUnauthorized: false)))
                }
                return
            }
            guard let data = data else {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: "服务器无响应", isUnauthorized: false)))
                }
                return
            }

            let raw = String(data: data, encoding: .utf8) ?? "解析失败"
            let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
            let redirectedToLogin = trimmed.hasPrefix("<!DOCTYPE html") || trimmed.hasPrefix("<html") || trimmed.contains("<title>用户中心登录</title>") || trimmed.contains("/portal/login")
            if redirectedToLogin {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: "登录状态失效，请重新登录", isUnauthorized: true)))
                }
                return
            }

            if let http = response as? HTTPURLResponse, !(200...299).contains(http.statusCode) {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: "请求失败（\(http.statusCode)）", isUnauthorized: http.statusCode == 401 || http.statusCode == 403)))
                }
                return
            }

            do {
                let envelope = try JSONDecoder().decode(APIEnvelope<T>.self, from: data)
                DispatchQueue.main.async {
                    completion(.success(envelope))
                }
            } catch {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: raw, isUnauthorized: false)))
                }
            }
        }.resume()
    }

    func requestData<T: Decodable>(
        path: String,
        method: String = "GET",
        headers: [String: String] = [:],
        body: Data? = nil,
        completion: @escaping (Result<T, APIError>) -> Void
    ) {
        requestEnvelope(path: path, method: method, headers: headers, body: body) { (result: Result<APIEnvelope<T>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code == 0, let data = envelope.data {
                    completion(.success(data))
                } else {
                    let message = envelope.msg ?? "请求失败"
                    let unauthorized = (envelope.code == 401 || envelope.code == 403)
                    completion(.failure(APIError(message: message, isUnauthorized: unauthorized)))
                }
            }
        }
    }

    func requestList<T: Decodable>(
        path: String,
        method: String = "GET",
        headers: [String: String] = [:],
        body: Data? = nil,
        completion: @escaping (Result<[T], APIError>) -> Void
    ) {
        requestEnvelope(path: path, method: method, headers: headers, body: body) { (result: Result<APIEnvelope<[T]>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code == 0 {
                    completion(.success(envelope.data ?? []))
                } else {
                    let message = envelope.msg ?? "请求失败"
                    let unauthorized = (envelope.code == 401 || envelope.code == 403)
                    completion(.failure(APIError(message: message, isUnauthorized: unauthorized)))
                }
            }
        }
    }

    func requestMessage(
        path: String,
        method: String = "POST",
        headers: [String: String] = [:],
        body: Data? = nil,
        completion: @escaping (Result<String, APIError>) -> Void
    ) {
        requestEnvelope(path: path, method: method, headers: headers, body: body) { (result: Result<APIEnvelope<EmptyPayload>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code == 0 {
                    completion(.success(envelope.msg ?? "操作成功"))
                } else {
                    let message = envelope.msg ?? "请求失败"
                    let unauthorized = (envelope.code == 401 || envelope.code == 403)
                    completion(.failure(APIError(message: message, isUnauthorized: unauthorized)))
                }
            }
        }
    }

    func requestRaw(
        path: String,
        method: String = "POST",
        headers: [String: String] = [:],
        body: Data? = nil,
        completion: @escaping (Result<String, APIError>) -> Void
    ) {
        guard let url = URL(string: path, relativeTo: AppEnvironment.baseURL) else {
            completion(.failure(APIError(message: "请求地址无效", isUnauthorized: false)))
            return
        }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.timeoutInterval = 30
        request.httpBody = body
        PortalRequestMeta.merged(with: headers).forEach { request.setValue($1, forHTTPHeaderField: $0) }

        session.dataTask(with: request) { data, _, error in
            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: sanitizeErrorMessage(error.localizedDescription), isUnauthorized: false)))
                }
                return
            }
            let text = data.flatMap { String(data: $0, encoding: .utf8) }?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            DispatchQueue.main.async {
                completion(.success(text))
            }
        }.resume()
    }

    func downloadData(
        path: String,
        absoluteURL: URL? = nil,
        completion: @escaping (Result<Data, APIError>) -> Void
    ) {
        let url: URL?
        if let absoluteURL = absoluteURL {
            url = absoluteURL
        } else {
            url = URL(string: path, relativeTo: AppEnvironment.baseURL)
        }
        guard let requestURL = url else {
            completion(.failure(APIError(message: "请求地址无效", isUnauthorized: false)))
            return
        }
        var request = URLRequest(url: requestURL)
        request.httpMethod = "GET"
        request.timeoutInterval = 30
        PortalRequestMeta.merged(with: [:]).forEach { request.setValue($1, forHTTPHeaderField: $0) }

        session.dataTask(with: request) { data, response, error in
            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: sanitizeErrorMessage(error.localizedDescription), isUnauthorized: false)))
                }
                return
            }
            if let http = response as? HTTPURLResponse, !(200...299).contains(http.statusCode) {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: "图片加载失败（\(http.statusCode)）", isUnauthorized: false)))
                }
                return
            }
            guard let data = data else {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: "无响应体", isUnauthorized: false)))
                }
                return
            }
            DispatchQueue.main.async {
                completion(.success(data))
            }
        }.resume()
    }
}


final class AppSessionStore {
    static let shared = AppSessionStore()

    private(set) var isAuthenticated = false
    private(set) var snapshot: PortalHomeSnapshot?
    private var isRefreshing = false

    private init() {}

    func bootstrap() {
        CookieStorageManager.shared.restoreCookies()
        refreshIfPossible(silent: true)
    }

    func refreshIfPossible(silent: Bool = false, allowAutoRelogin: Bool = true, completion: ((Bool) -> Void)? = nil) {
        guard !isRefreshing else { completion?(false); return }
        isRefreshing = true
        PortalService.shared.fetchHomeSnapshot { result in
            self.isRefreshing = false
            switch result {
            case .success(let snapshot):
                self.isAuthenticated = true
                self.snapshot = snapshot
                CookieStorageManager.shared.persistCookies(for: AppEnvironment.baseURL.host ?? "")
                NotificationCenter.default.post(name: AppNotifications.sessionDidChange, object: nil)
                YybAccountStore.shared.prefetch(autoCheck: false)
                completion?(true)
            case .failure(let error):
                if error.isUnauthorized, allowAutoRelogin, let credentials = CredentialStore.shared.load() {
                    PortalAuthService.shared.login(username: credentials.username, password: credentials.password, remember: true) { loginResult in
                        switch loginResult {
                        case .success:
                            self.refreshIfPossible(silent: silent, allowAutoRelogin: false, completion: completion)
                        case .failure(let loginError):
                            // 只有真正的认证失败才清 session，网络错误不清
                            if loginError.isUnauthorized {
                                self.clearSession(notify: !silent)
                            }
                            completion?(false)
                        }
                    }
                    return
                }
                if error.isUnauthorized {
                    self.clearSession(notify: !silent)
                }
                completion?(false)
            }
        }
    }

    func markAuthenticated(with snapshot: PortalHomeSnapshot? = nil) {
        isAuthenticated = true
        self.snapshot = snapshot
        CookieStorageManager.shared.persistCookies(for: AppEnvironment.baseURL.host ?? "")
        NotificationCenter.default.post(name: AppNotifications.sessionDidChange, object: nil)
        YybAccountStore.shared.prefetch(autoCheck: false)
    }

    func update(snapshot: PortalHomeSnapshot, notify: Bool = true) {
        isAuthenticated = true
        self.snapshot = snapshot
        if notify {
            NotificationCenter.default.post(name: AppNotifications.sessionDidChange, object: nil)
        }
    }

    func clearSession(notify: Bool = true, requireLogin: Bool = false) {
        guard isAuthenticated else { return }
        isAuthenticated = false
        snapshot = nil
        YybAccountStore.shared.clear()
        CookieStorageManager.shared.clearCookies()
        if notify {
            NotificationCenter.default.post(name: AppNotifications.sessionDidLogout, object: nil)
            NotificationCenter.default.post(name: AppNotifications.sessionDidChange, object: nil)
        }
        if requireLogin {
            NotificationCenter.default.post(name: AppNotifications.sessionRequiresLogin, object: nil)
        }
    }
}


/// 应用宝账号缓存：App 登录后后台预检，协议页直接复用，减少打开延迟
final class YybAccountStore {
    static let shared = YybAccountStore()

    private(set) var status: PortalYybStatus?
    private(set) var lastUpdatedAt: Date?
    private(set) var isLoading = false
    private(set) var sessionAutoChecked = false
    private var pendingAlertSummary: String?

    private init() {}

    var accounts: [PortalYybAccount] {
        status?.accounts ?? []
    }

    var isServiceReady: Bool {
        status?.enabled == true && status?.ready == true
    }

    var serviceTitle: String {
        guard let st = status else { return "待机" }
        if st.enabled == true && st.ready == true {
            return "已启动"
        }
        return "未启动"
    }

    func clear() {
        status = nil
        lastUpdatedAt = nil
        pendingAlertSummary = nil
        isLoading = false
        sessionAutoChecked = false
        NotificationCenter.default.post(name: AppNotifications.yybStatusDidUpdate, object: nil)
    }

    func consumePendingAlert() -> String? {
        let text = pendingAlertSummary
        pendingAlertSummary = nil
        return text
    }

    func prefetchIfNeeded(autoCheck: Bool = false) {
        guard AppSessionStore.shared.isAuthenticated else { return }
        if sessionAutoChecked, status != nil { return }
        prefetch(autoCheck: autoCheck)
    }

    func prefetch(autoCheck: Bool = false) {
        guard AppSessionStore.shared.isAuthenticated else { return }
        guard !isLoading else { return }
        isLoading = true
        PortalService.shared.fetchYybStatus(autoCheck: autoCheck) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.isLoading = false
                switch result {
                case .success(let st):
                    self.status = st
                    self.lastUpdatedAt = Date()
                    self.sessionAutoChecked = true
                    if autoCheck {
                        self.pendingAlertSummary = formatYybCheckSummary(st.checkSummary)
                    }
                case .failure:
                    break
                }
                NotificationCenter.default.post(name: AppNotifications.yybStatusDidUpdate, object: nil)
            }
        }
    }

    func reload(autoCheck: Bool, showAlert: Bool, completion: ((Result<PortalYybStatus, APIError>) -> Void)? = nil) {
        guard !isLoading else {
            completion?(.failure(APIError(message: "正在刷新，请稍候", isUnauthorized: false)))
            return
        }
        isLoading = true
        NotificationCenter.default.post(name: AppNotifications.yybStatusDidUpdate, object: nil)
        PortalService.shared.fetchYybStatus(autoCheck: autoCheck) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.isLoading = false
                switch result {
                case .success(let st):
                    self.status = st
                    self.lastUpdatedAt = Date()
                    self.sessionAutoChecked = true
                    if showAlert, autoCheck {
                        self.pendingAlertSummary = formatYybCheckSummary(st.checkSummary)
                    }
                    NotificationCenter.default.post(name: AppNotifications.yybStatusDidUpdate, object: nil)
                    completion?(.success(st))
                case .failure(let error):
                    NotificationCenter.default.post(name: AppNotifications.yybStatusDidUpdate, object: nil)
                    completion?(.failure(error))
                }
            }
        }
    }
}


func shortenProtocolId(_ id: String?, head: Int = 8, tail: Int = 6) -> String {
    let s = (id ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
    if s.count <= head + tail + 3 { return s }
    return String(s.prefix(head)) + "…" + String(s.suffix(tail))
}

func jsonValueAsString(_ value: Any?) -> String {
    if let text = value as? String {
        return text.trimmingCharacters(in: .whitespacesAndNewlines)
    }
    if let number = value as? NSNumber {
        return number.stringValue
    }
    if let number = value as? Int {
        return String(number)
    }
    if let number = value as? Double, number.rounded() == number {
        return String(Int(number))
    }
    return ""
}

func parseProxyAreaRows(_ data: [String: Any]?) -> [(code: String, name: String)] {
    guard let data else { return [] }
    let keys = ["list", "provinceList", "city", "cityList"]
    var rows: [Any] = []
    for key in keys {
        if let arr = data[key] as? [Any], !arr.isEmpty {
            rows = arr
            break
        }
    }
    return rows.compactMap { item in
        let row: [String: Any]
        if let dict = item as? [String: Any] {
            row = dict
        } else if let dict = item as? NSDictionary {
            row = dict as? [String: Any] ?? [:]
        } else {
            return nil
        }
        let code = jsonValueAsString(row["regionCode"]).isEmpty
            ? jsonValueAsString(row["region_code"])
            : jsonValueAsString(row["regionCode"])
        let name = jsonValueAsString(row["regionName"]).isEmpty
            ? jsonValueAsString(row["region_name"])
            : jsonValueAsString(row["regionName"])
        let displayName = name.isEmpty ? code : name
        return code.isEmpty ? nil : (code, displayName)
    }
}

func formatScanCostPreview(cost: Int?, hint: String?) -> (text: String, isFree: Bool) {
    if let cost, cost > 0 {
        return ("本次扫码将扣除 \(cost) 积分", false)
    }
    if let cost, cost == 0 {
        return ("本次扫码免费，不扣除积分", true)
    }
    return ("", true)
}

func formatYybScanCostNote(cost: Int?, hint: String?) -> String? {
    let preview = formatScanCostPreview(cost: cost, hint: hint)
    return preview.text.isEmpty ? nil : preview.text
}

func formatYybConfirmMessage(alreadyBound: Bool, cost: Int) -> String {
    if alreadyBound { return "续登录成功，未扣除积分" }
    if cost > 0 { return "扫码成功，已扣除 \(cost) 积分" }
    return "扫码成功，账号已绑定"
}

func formatYybExpiryText(_ acc: PortalYybAccount) -> (text: String, warn: Bool, expired: Bool) {
    let loginSec = acc.loginAt ?? acc.createdAt ?? 0
    if loginSec <= 0 { return ("", false, false) }
    let expireSec = acc.expiresAt ?? (loginSec + 30 * 24 * 3600)
    let remain = expireSec - Int64(Date().timeIntervalSince1970)
    if remain <= 0 {
        return ("登录已过期，请重新扫码登录延期", false, true)
    }
    let days = remain / 86400
    let hours = (remain % 86400) / 3600
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "zh_CN")
    formatter.dateFormat = "MM-dd HH:mm"
    let expireText = formatter.string(from: Date(timeIntervalSince1970: TimeInterval(expireSec)))
    let warn = remain < 3 * 86400
    return ("剩余有效期 \(days) 天 \(hours) 小时（至 \(expireText)）", warn, false)
}


/// 协议双绑缓存：微信 wxid ↔ 应用宝 openid
final class ProtocolBindStore {
    static let shared = ProtocolBindStore()

    private(set) var bindings: [PortalProtocolBinding] = []
    private(set) var quota: PortalProtocolBindQuota?
    private(set) var wxDevices: [PortalWxDevice] = []
    private(set) var isLoading = false

    private init() {}

    func binding(forWxid wxid: String?) -> PortalProtocolBinding? {
        let key = (wxid ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return bindings.first { ($0.wxWxid ?? "") == key }
    }

    func binding(forOpenId openid: String?) -> PortalProtocolBinding? {
        let key = (openid ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return bindings.first { ($0.yybOpenId ?? "") == key }
    }

    func boundWxSet() -> Set<String> {
        Set(bindings.compactMap { $0.wxWxid }.filter { !$0.isEmpty })
    }

    func boundOpenIdSet() -> Set<String> {
        Set(bindings.compactMap { $0.yybOpenId }.filter { !$0.isEmpty })
    }

    func unboundWxDevices() -> [PortalWxDevice] {
        let bound = boundWxSet()
        return wxDevices.filter { let wx = $0.wxid ?? ""; return !wx.isEmpty && !bound.contains(wx) }
    }

    func unboundYybAccounts(_ accounts: [PortalYybAccount]) -> [PortalYybAccount] {
        let bound = boundOpenIdSet()
        return accounts.filter { let oid = $0.openid ?? ""; return !oid.isEmpty && !bound.contains(oid) }
    }

    func shouldShowBindUI(yybAccounts: [PortalYybAccount]) -> Bool {
        !wxDevices.isEmpty && !yybAccounts.isEmpty &&
            (!unboundWxDevices().isEmpty || !unboundYybAccounts(yybAccounts).isEmpty)
    }

    func clear() {
        bindings = []
        quota = nil
        wxDevices = []
        isLoading = false
        NotificationCenter.default.post(name: AppNotifications.protocolBindDidUpdate, object: nil)
    }

    func reload(yybAccounts: [PortalYybAccount] = [], completion: (() -> Void)? = nil) {
        guard AppSessionStore.shared.isAuthenticated else {
            completion?()
            return
        }
        guard !isLoading else {
            completion?()
            return
        }
        isLoading = true
        let group = DispatchGroup()
        var capturedError: APIError?

        group.enter()
        PortalService.shared.fetchProtocolBindings { [weak self] result in
            if case .success(let rows) = result { self?.bindings = rows }
            else if case .failure(let error) = result { capturedError = error }
            group.leave()
        }
        group.enter()
        PortalService.shared.fetchProtocolBindQuota { [weak self] result in
            if case .success(let q) = result { self?.quota = q }
            group.leave()
        }
        if wxDevices.isEmpty {
            group.enter()
            PortalService.shared.fetchWxDevices { [weak self] result in
                if case .success(let devices) = result { self?.wxDevices = devices }
                group.leave()
            }
        }
        _ = yybAccounts

        group.notify(queue: .main) { [weak self] in
            self?.isLoading = false
            NotificationCenter.default.post(name: AppNotifications.protocolBindDidUpdate, object: nil)
            completion?()
        }
    }
}


final class PortalAuthService {
    static let shared = PortalAuthService()
    private init() {}

    func login(username: String, password: String, remember: Bool = true, completion: @escaping (Result<Void, APIError>) -> Void) {
        let body = "type=user&account=\(username.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")&pin=\(password.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")"
        APIClient.shared.requestRaw(path: "/api/login/admin", method: "POST", headers: ["Content-Type": "application/x-www-form-urlencoded"], body: body.data(using: .utf8)) { result in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let text):
                if text == "登录" {
                    CookieStorageManager.shared.persistCookies(for: AppEnvironment.baseURL.host ?? "")
                    if remember {
                        CredentialStore.shared.save(username: username, password: password)
                    } else {
                        CredentialStore.shared.clear()
                    }
                    completion(.success(()))
                    return
                }
                if let data = text.data(using: .utf8),
                   let messageEnvelope = try? JSONDecoder().decode(APIEnvelope<EmptyPayload>.self, from: data) {
                    completion(.failure(APIError(message: messageEnvelope.msg ?? "登录失败", isUnauthorized: false)))
                } else {
                    completion(.failure(APIError(message: text.isEmpty ? "登录失败" : text, isUnauthorized: false)))
                }
            }
        }
    }

    func register(username: String, password: String, bindCode: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let body = "username=\(username.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")&password=\(password.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")&bindCode=\(bindCode.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")"
        APIClient.shared.requestEnvelope(path: "/api/login/register", method: "POST", headers: ["Content-Type": "application/x-www-form-urlencoded"], body: body.data(using: .utf8)) { (result: Result<APIEnvelope<RegisterPayload>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code == 0 {
                    CookieStorageManager.shared.persistCookies(for: AppEnvironment.baseURL.host ?? "")
                    CredentialStore.shared.save(username: username, password: password)
                    completion(.success(envelope.msg ?? "注册成功"))
                } else {
                    completion(.failure(APIError(message: envelope.msg ?? "注册失败", isUnauthorized: false)))
                }
            }
        }
    }

    func fetchResetInfo(code: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let body = "code=\(code.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")"
        APIClient.shared.requestEnvelope(path: "/api/login/reset/info", method: "POST", headers: ["Content-Type": "application/x-www-form-urlencoded"], body: body.data(using: .utf8)) { (result: Result<APIEnvelope<ResetInfoPayload>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code == 0, let username = envelope.data?.username {
                    completion(.success(username))
                } else {
                    completion(.failure(APIError(message: envelope.msg ?? "验证码无效", isUnauthorized: false)))
                }
            }
        }
    }

    func resetPassword(code: String, username: String, password: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let body = "code=\(code.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")&username=\(username.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")&password=\(password.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? "")"
        APIClient.shared.requestMessage(path: "/api/login/reset/password", method: "POST", headers: ["Content-Type": "application/x-www-form-urlencoded"], body: body.data(using: .utf8), completion: completion)
    }

    func logout(completion: @escaping (Result<String, APIError>) -> Void) {
        APIClient.shared.requestMessage(path: "/api/login/logout") { result in
            if case .success = result {
                CredentialStore.shared.clear()
                AppSessionStore.shared.clearSession()
            }
            completion(result)
        }
    }
}


struct ResetInfoPayload: Decodable {
    let username: String?
}


struct RegisterPayload: Decodable {
    let username: String?
    let userNumber: Int?
}


final class PortalService {
    static let shared = PortalService()
    private init() {}

    func fetchHomeSnapshot(completion: @escaping (Result<PortalHomeSnapshot, APIError>) -> Void) {
        var homePayload: PortalHomePayload?
        var wxStatus: PortalWechatStatus?
        var wxDevices: [PortalWxDevice] = []
        var yybStatus: PortalYybStatus?
        var protocolBindings: [PortalProtocolBinding] = []
        var capturedUnauthorized: APIError?
        var capturedError: APIError?
        let group = DispatchGroup()

        group.enter()
        APIClient.shared.requestData(path: "/api/portal/home") { (result: Result<PortalHomePayload, APIError>) in
            if case .success(let value) = result { homePayload = value }
            if case .failure(let error) = result {
                if error.isUnauthorized { capturedUnauthorized = error }
                else { capturedError = error }
            }
            group.leave()
        }

        group.enter()
        APIClient.shared.requestData(path: "/api/portal/wx/status") { (result: Result<PortalWechatStatus, APIError>) in
            if case .success(let value) = result { wxStatus = value }
            group.leave()
        }

        group.enter()
        APIClient.shared.requestList(path: "/api/portal/wx/devices") { (result: Result<[PortalWxDevice], APIError>) in
            if case .success(let value) = result { wxDevices = value }
            group.leave()
        }

        group.enter()
        APIClient.shared.requestData(path: "/api/portal/yyb/status") { (result: Result<PortalYybStatus, APIError>) in
            if case .success(let value) = result { yybStatus = value }
            group.leave()
        }

        group.enter()
        APIClient.shared.requestList(path: "/api/portal/protocol/bindings") { (result: Result<[PortalProtocolBinding], APIError>) in
            if case .success(let value) = result { protocolBindings = value }
            group.leave()
        }

        group.notify(queue: .main) {
            if let authError = capturedUnauthorized {
                completion(.failure(authError))
                return
            }
            guard let payload = homePayload else {
                completion(.failure(capturedError ?? APIError(message: "首页数据不完整", isUnauthorized: false)))
                return
            }
            completion(.success(PortalHomeSnapshot(
                dashboard: payload.dashboard,
                profile: payload.profile,
                wechatStatus: wxStatus,
                wxDevices: wxDevices,
                yybStatus: yybStatus,
                protocolBindings: protocolBindings
            )))
        }
    }

    func fetchActivities(completion: @escaping (Result<[PortalActivity], APIError>) -> Void) {
        requestPortalData(path: "/api/portal/activities", completion: completion)
    }

    func fetchProjects(completion: @escaping (Result<[PortalProject], APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/projects", completion: completion)
    }

    func fetchProtocolAccountOptions(activityId: String, remarks: String? = nil, completion: @escaping (Result<[ProtocolAccountOption], APIError>) -> Void) {
        var path = "/api/portal/protocol/account-options?activityId=\(activityId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? activityId)"
        if let remarks, !remarks.isEmpty {
            path += "&remarks=\(remarks.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? remarks)"
        }
        APIClient.shared.requestData(path: path, completion: completion)
    }

    func createProject(activityId: String, inputs: [String: String], remarks: String, months: Int, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = [
            "activityId": activityId,
            "inputs": inputs,
            "remarks": remarks,
            "months": months
        ]
        requestMessageJSON(path: "/api/portal/project", payload: payload, completion: completion)
    }

    func renewProject(activityId: String, remarks: String, months: Int, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["activityId": activityId, "remarks": remarks, "months": months]
        requestMessageJSON(path: "/api/portal/project/renew", payload: payload, completion: completion)
    }

    func deleteProject(activityId: String, remarks: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["activityId": activityId, "remarks": remarks]
        requestMessageJSON(path: "/api/portal/project/delete", payload: payload, completion: completion)
    }

    func updateProject(activityId: String, remarks: String, ckValue: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["activityId": activityId, "remarks": remarks, "ckValue": ckValue]
        requestMessageJSON(path: "/api/portal/project/update", payload: payload, completion: completion)
    }

    func queryIncome(activityId: String, remarks: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["activityId": activityId, "remarks": remarks]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求数据错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestEnvelope(path: "/api/portal/project/income", method: "POST", headers: ["Content-Type": "application/json"], body: body) { (result: Result<APIEnvelope<String>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code == 0 {
                    completion(.success(envelope.data ?? envelope.msg ?? "暂无查询结果"))
                } else {
                    let message = envelope.msg ?? "查询失败"
                    let unauthorized = (envelope.code == 401 || envelope.code == 403)
                    completion(.failure(APIError(message: message, isUnauthorized: unauthorized)))
                }
            }
        }
    }

    func redeemKey(token: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["token": token]
        requestMessageJSON(path: "/api/portal/redeem-key", payload: payload, completion: completion)
    }

    func fetchWechatStatus(completion: @escaping (Result<PortalWechatStatus, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/wx/status", completion: completion)
    }

    func performWechatAction(path: String, completion: @escaping (Result<PortalWechatActionResult, APIError>) -> Void) {
        APIClient.shared.requestData(path: path, method: "POST", completion: completion)
    }

    func pollWechatLogin(uuid: String, deductCoin: Bool, completion: @escaping (Result<PortalWechatActionResult, APIError>) -> Void) {
        let payload: [String: Any] = ["uuid": uuid, "deductCoin": deductCoin]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/wx/poll-login", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func checkin(completion: @escaping (Result<String, APIError>) -> Void) {
        let path = "/api/portal/checkin"
        let signHeaders = SignatureHelper.getHeaders(for: path)
        APIClient.shared.requestMessage(path: path, headers: signHeaders, completion: completion)
    }

    func pray(completion: @escaping (Result<String, APIError>) -> Void) {
        let path = "/api/portal/pray"
        let signHeaders = SignatureHelper.getHeaders(for: path)
        APIClient.shared.requestMessage(path: path, headers: signHeaders, completion: completion)
    }

    func fetchCoinLogs(source: String? = nil, completion: @escaping (Result<[CoinLog], APIError>) -> Void) {
        var path = "/api/portal/coin-logs"
        if let source = source, !source.isEmpty {
            path += "?source=\(source.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? source)"
        }
        let signHeaders = SignatureHelper.getHeaders(for: path)
        APIClient.shared.requestList(path: path, headers: signHeaders, completion: completion)
    }

    func markNotificationRead(id: Int, completion: @escaping (Result<Void, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/notification?id=\(id)") { (result: Result<PortalNotification, APIError>) in
            switch result {
            case .success: completion(.success(()))
            case .failure(let error): completion(.failure(error))
            }
        }
    }

    func fetchNotifications(includeContent: Bool = true, completion: @escaping (Result<PortalNotificationPage, APIError>) -> Void) {
        let path = "/api/portal/notifications?includeContent=\(includeContent ? "1" : "0")&limit=100"
        struct NotificationsData: Decodable {
            let list: [PortalNotification]
            let total: Int
            let unread: Int
        }
        requestPortalEnvelope(path: path) { (result: Result<PortalAPIResponse<NotificationsData>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code == 0, let data = envelope.data {
                    PortalAccessStore.shared.update(envelope.portalAccess)
                    completion(.success(PortalNotificationPage(list: data.list, total: data.total, unread: data.unread)))
                } else {
                    completion(.failure(APIError(message: envelope.msg ?? "请求失败", isUnauthorized: envelope.code == 401 || envelope.code == 403)))
                }
            }
        }
    }

    func fetchNotificationDetail(id: Int, completion: @escaping (Result<PortalNotification, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/notification?id=\(id)", completion: completion)
    }

    func submitFeedback(type: String, title: String, content: String, contact: String, attachments: [String] = [], completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["type": type, "title": title, "content": content, "contact": contact, "attachments": attachments]
        requestMessageJSON(path: "/api/portal/feedback", payload: payload, completion: completion)
    }

    func uploadFeedbackFile(data: Data, filename: String, mimeType: String, completion: @escaping (Result<String, APIError>) -> Void) {
        guard let url = URL(string: "/api/portal/feedback/upload", relativeTo: AppEnvironment.baseURL) else {
            completion(.failure(APIError(message: "请求地址无效", isUnauthorized: false)))
            return
        }
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.timeoutInterval = 120
        let boundary = "Boundary-\(UUID().uuidString)"
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        PortalRequestMeta.merged(with: [:]).forEach { request.setValue($1, forHTTPHeaderField: $0) }
        var body = Data()
        body.append("--\(boundary)\r\n".data(using: .utf8)!)
        body.append("Content-Disposition: form-data; name=\"file\"; filename=\"\(filename)\"\r\n".data(using: .utf8)!)
        body.append("Content-Type: \(mimeType)\r\n\r\n".data(using: .utf8)!)
        body.append(data)
        body.append("\r\n--\(boundary)--\r\n".data(using: .utf8)!)
        request.httpBody = body
        URLSession.shared.dataTask(with: request) { data, _, error in
            DispatchQueue.main.async {
                if let error = error {
                    completion(.failure(APIError(message: sanitizeErrorMessage(error.localizedDescription), isUnauthorized: false)))
                    return
                }
                guard let data = data,
                      let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                      let code = json["code"] as? Int else {
                    completion(.failure(APIError(message: "上传失败", isUnauthorized: false)))
                    return
                }
                if code == 0, let payload = json["data"] as? [String: Any], let fileURL = payload["url"] as? String {
                    completion(.success(fileURL))
                } else {
                    completion(.failure(APIError(message: (json["msg"] as? String) ?? "上传失败", isUnauthorized: false)))
                }
            }
        }.resume()
    }

    func verifySession(completion: @escaping (Result<Void, APIError>) -> Void) {
        fetchDashboard { result in
            switch result {
            case .success:
                completion(.success(()))
            case .failure(let error):
                completion(.failure(error))
            }
        }
    }

    func fetchDashboard(completion: @escaping (Result<PortalDashboard, APIError>) -> Void) {
        requestPortalData(path: "/api/portal/dashboard", completion: completion)
    }

    private func requestPortalData<T: Decodable>(path: String, completion: @escaping (Result<T, APIError>) -> Void) {
        requestPortalEnvelope(path: path) { (result: Result<PortalAPIResponse<T>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code == 0, let data = envelope.data {
                    PortalAccessStore.shared.update(envelope.portalAccess)
                    completion(.success(data))
                } else {
                    let message = envelope.msg ?? "请求失败"
                    let unauthorized = (envelope.code == 401 || envelope.code == 403)
                    completion(.failure(APIError(message: message, isUnauthorized: unauthorized)))
                }
            }
        }
    }

    private func requestPortalEnvelope<T: Decodable>(path: String, method: String = "GET", headers: [String: String] = [:], body: Data? = nil, completion: @escaping (Result<PortalAPIResponse<T>, APIError>) -> Void) {
        guard let url = URL(string: path, relativeTo: AppEnvironment.baseURL) else {
            completion(.failure(APIError(message: "请求地址无效", isUnauthorized: false)))
            return
        }
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.httpBody = body
        request.timeoutInterval = 30
        PortalRequestMeta.merged(with: headers).forEach { request.setValue($1, forHTTPHeaderField: $0) }
        URLSession.shared.dataTask(with: request) { data, response, error in
            DispatchQueue.main.async {
                if let error = error {
                    completion(.failure(APIError(message: sanitizeErrorMessage(error.localizedDescription), isUnauthorized: false)))
                    return
                }
                guard let data = data else {
                    completion(.failure(APIError(message: "服务器无响应", isUnauthorized: false)))
                    return
                }
                do {
                    let envelope = try JSONDecoder().decode(PortalAPIResponse<T>.self, from: data)
                    completion(.success(envelope))
                } catch {
                    completion(.failure(APIError(message: String(data: data, encoding: .utf8) ?? "解析失败", isUnauthorized: false)))
                }
            }
        }.resume()
    }

    func performWechatActionForDevice(path: String, wxid: String, completion: @escaping (Result<PortalWechatActionResult, APIError>) -> Void) {
        let payload: [String: Any] = ["wxid": wxid]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: path, method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func fetchWxDevices(completion: @escaping (Result<[PortalWxDevice], APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/wx/devices", completion: completion)
    }

    func addWxDevice(wxid: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["wxid": wxid]
        requestMessageJSON(path: "/api/portal/wx/add-device", payload: payload, completion: completion)
    }

    func removeWxDevice(id: Int, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["id": id]
        requestMessageJSON(path: "/api/portal/wx/remove-device", payload: payload, completion: completion)
    }

    private func requestMessageJSON(path: String, payload: [String: Any], completion: @escaping (Result<String, APIError>) -> Void) {
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestMessage(path: path, method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func fetchJdAccounts(completion: @escaping (Result<[PortalJdAccount], APIError>) -> Void) {
        APIClient.shared.requestList(path: "/api/portal/jd/accounts", completion: completion)
    }

    func queryJdAccount(index: Int, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["index": index]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/jd/query", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func sendJdSms(phone: String, completion: @escaping (Result<String, APIError>) -> Void) {
        requestMessageJSON(path: "/api/portal/jd/sms/send", payload: ["phone": phone], completion: completion)
    }

    func verifyJdSms(phone: String, code: String, idCard: String, completion: @escaping (Result<PortalJdSmsVerifyResult, APIError>) -> Void) {
        let payload: [String: Any] = ["phone": phone, "code": code, "idCard": idCard]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/jd/sms/verify", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func fetchJdWxDevices(completion: @escaping (Result<[PortalJdWxDevice], APIError>) -> Void) {
        APIClient.shared.requestList(path: "/api/portal/jd/wx/devices", completion: completion)
    }

    func refreshJdWx(wxid: String, riskConfirmed: Bool = false, completion: @escaping (Result<PortalJdWxRefreshResult, APIError>) -> Void) {
        let payload: [String: Any] = ["wxid": wxid, "riskConfirmed": riskConfirmed]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/jd/wx/refresh", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func continueJdWxRisk(completion: @escaping (Result<PortalJdWxRefreshResult, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/jd/wx/continue-risk", method: "POST", completion: completion)
    }

    func fetchYybStatus(autoCheck: Bool = false, completion: @escaping (Result<PortalYybStatus, APIError>) -> Void) {
        let q = autoCheck ? "?check=1" : ""
        APIClient.shared.requestData(path: "/api/portal/yyb/status\(q)", completion: completion)
    }

    func createYybQr(
        regionCode: String = "",
        regionName: String = "",
        useProxy: Bool = false,
        packId: String = "",
        completion: @escaping (Result<PortalYybQrCreateResult, APIError>) -> Void
    ) {
        let payload: [String: Any] = [
            "regionCode": regionCode,
            "regionName": regionName,
            "useProxy": useProxy,
            "packId": packId,
        ]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(
            path: "/api/portal/yyb/qr",
            method: "POST",
            headers: ["Content-Type": "application/json"],
            body: body,
            completion: completion
        )
    }

    func fetchProtocolBindings(completion: @escaping (Result<[PortalProtocolBinding], APIError>) -> Void) {
        APIClient.shared.requestList(path: "/api/portal/protocol/bindings", completion: completion)
    }

    func fetchProtocolBindQuota(completion: @escaping (Result<PortalProtocolBindQuota, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/protocol/bind/quota", completion: completion)
    }

    func protocolBind(wxWxid: String, yybOpenId: String, nickname: String = "", completion: @escaping (Result<PortalProtocolBinding, APIError>) -> Void) {
        let payload: [String: Any] = ["wxWxid": wxWxid, "yybOpenId": yybOpenId, "nickname": nickname]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/protocol/bind", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func protocolUnbind(wxWxid: String, yybOpenId: String, completion: @escaping (Result<String, APIError>) -> Void) {
        requestMessageJSON(path: "/api/portal/protocol/unbind", payload: ["wxWxid": wxWxid, "yybOpenId": yybOpenId], completion: completion)
    }

    func fetchProtocolProxyConfig(completion: @escaping (Result<PortalProxyConfig, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/protocol/proxy/config", completion: completion)
    }

    func fetchProtocolProxyAreas(parentCode: String = "", packId: String = "", completion: @escaping (Result<[String: Any], APIError>) -> Void) {
        let payload: [String: Any] = ["parent_code": parentCode, "packid": packId]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestRaw(path: "/api/portal/protocol/proxy/areas", method: "POST", headers: ["Content-Type": "application/json"], body: body) { result in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let raw):
                guard let data = raw.data(using: .utf8),
                      let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                      let code = json["code"] as? Int else {
                    completion(.failure(APIError(message: "数据解析失败", isUnauthorized: false)))
                    return
                }
                if code == 0, let payload = json["data"] as? [String: Any] {
                    completion(.success(payload))
                } else {
                    completion(.failure(APIError(message: (json["msg"] as? String) ?? "请求失败", isUnauthorized: code == 401 || code == 403)))
                }
            }
        }
    }

    func pollYybQr(sessionId: String, completion: @escaping (Result<PortalYybQrPollResult, APIError>) -> Void) {
        let encoded = sessionId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? sessionId
        APIClient.shared.requestData(path: "/api/portal/yyb/qr/\(encoded)/poll", completion: completion)
    }

    func confirmYybQr(sessionId: String, completion: @escaping (Result<PortalYybConfirmResult, APIError>) -> Void) {
        let encoded = sessionId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? sessionId
        APIClient.shared.requestData(path: "/api/portal/yyb/qr/\(encoded)/confirm", method: "POST", completion: completion)
    }

    func refreshYybAccount(ref: String, completion: @escaping (Result<[String: Any], APIError>) -> Void) {
        requestAnyJSON(path: "/api/portal/yyb/accounts/refresh", payload: ["ref": ref], completion: completion)
    }

    func resyncYybAccount(ref: String, completion: @escaping (Result<PortalYybAccount, APIError>) -> Void) {
        let payload: [String: Any] = ["ref": ref]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/yyb/accounts/resync", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func deleteYybAccount(ref: String, completion: @escaping (Result<String, APIError>) -> Void) {
        requestMessageJSON(path: "/api/portal/yyb/accounts/delete", payload: ["ref": ref], completion: completion)
    }

    func yybWxappGetCode(ref: String, appId: String, completion: @escaping (Result<[String: Any], APIError>) -> Void) {
        requestAnyJSON(path: "/api/portal/yyb/wxapp/getCode", payload: ["ref": ref, "appId": appId], completion: completion)
    }

    func yybWxappGetPhone(ref: String, appId: String, completion: @escaping (Result<[String: Any], APIError>) -> Void) {
        requestAnyJSON(path: "/api/portal/yyb/wxapp/getPhoneNumber", payload: ["ref": ref, "appId": appId], completion: completion)
    }

    func yybWxappOperate(ref: String, appId: String, payload: [String: Any], completion: @escaping (Result<[String: Any], APIError>) -> Void) {
        requestAnyJSON(path: "/api/portal/yyb/wxapp/operateWxData", payload: ["ref": ref, "appId": appId, "payload": payload], completion: completion)
    }

    func fetchJdYybAccounts(completion: @escaping (Result<[PortalJdYybAccount], APIError>) -> Void) {
        APIClient.shared.requestList(path: "/api/portal/jd/yyb/accounts", completion: completion)
    }

    func refreshJdYyb(openid: String, riskConfirmed: Bool = false, completion: @escaping (Result<PortalJdWxRefreshResult, APIError>) -> Void) {
        let payload: [String: Any] = ["openid": openid, "riskConfirmed": riskConfirmed]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/jd/yyb/refresh", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func continueJdYybRisk(completion: @escaping (Result<PortalJdWxRefreshResult, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/jd/yyb/continue-risk", method: "POST", completion: completion)
    }

    private func requestAnyJSON(path: String, payload: [String: Any], completion: @escaping (Result<[String: Any], APIError>) -> Void) {
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestRaw(path: path, method: "POST", headers: ["Content-Type": "application/json"], body: body) { result in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let text):
                guard let raw = text.data(using: .utf8),
                      let obj = try? JSONSerialization.jsonObject(with: raw) as? [String: Any] else {
                    completion(.failure(APIError(message: "数据解析失败", isUnauthorized: false)))
                    return
                }
                let code = obj["code"] as? Int ?? -1
                if code != 0 {
                    completion(.failure(APIError(message: (obj["msg"] as? String) ?? "请求失败", isUnauthorized: code == 401 || code == 403)))
                    return
                }
                if let dict = obj["data"] as? [String: Any] {
                    completion(.success(dict))
                } else {
                    completion(.success([:]))
                }
            }
        }
    }

    func executeJdTask(taskId: String, taskName: String, accountIndexes: [Int], completion: @escaping (Result<PortalJdTaskExecuteResult, APIError>) -> Void) {
        let payload: [String: Any] = ["taskId": taskId, "taskName": taskName, "accountIndexes": accountIndexes]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/jd/task/execute", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func fetchJdTasks(completion: @escaping (Result<[PortalJdTaskItem], APIError>) -> Void) {
        APIClient.shared.requestList(path: "/api/portal/jd/tasks", method: "GET", completion: completion)
    }

    func fetchJdProxyStatus(completion: @escaping (Result<PortalJdProxyStatus, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/jd/proxy/status", completion: completion)
    }

    func buyJdProxy(months: Int, completion: @escaping (Result<PortalJdProxyStatus, APIError>) -> Void) {
        let payload: [String: Any] = ["months": months]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/jd/proxy/buy", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func stopJdTask(taskId: String, completion: @escaping (Result<String, APIError>) -> Void) {
        requestMessageJSON(path: "/api/portal/jd/task/stop", payload: ["taskId": taskId], completion: completion)
    }

    @discardableResult
    func streamJdTaskLogs(taskId: String, onLine: @escaping (String) -> Void, onDone: @escaping () -> Void, onError: @escaping (APIError) -> Void) -> JdTaskLogStreamer {
        let streamer = JdTaskLogStreamer(taskId: taskId, onLine: onLine, onDone: onDone, onError: onError)
        streamer.start()
        return streamer
    }

    func checkKuwoAuth(completion: @escaping (Result<(Bool, String), APIError>) -> Void) {
        struct KuwoAuthResponse: Decodable {
            let code: Int
            let authorized: Bool?
            let msg: String?
        }
        guard let url = URL(string: "/api/portal/kuwo/check-auth", relativeTo: AppEnvironment.baseURL) else {
            completion(.failure(APIError(message: "请求地址无效", isUnauthorized: false)))
            return
        }
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.timeoutInterval = 30
        PortalRequestMeta.defaultHeaders.forEach { request.setValue($1, forHTTPHeaderField: $0) }
        URLSession.shared.dataTask(with: request) { data, response, error in
            if let error = error {
                DispatchQueue.main.async { completion(.failure(APIError(message: sanitizeErrorMessage(error.localizedDescription), isUnauthorized: false))) }
                return
            }
            guard let data = data else {
                DispatchQueue.main.async { completion(.failure(APIError(message: "服务器无响应", isUnauthorized: false))) }
                return
            }
            do {
                let resp = try JSONDecoder().decode(KuwoAuthResponse.self, from: data)
                DispatchQueue.main.async {
                    if resp.code != 0 {
                        completion(.failure(APIError(message: resp.msg ?? "检查授权失败", isUnauthorized: false)))
                    } else {
                        completion(.success((resp.authorized ?? false, resp.msg ?? "")))
                    }
                }
            } catch {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: String(data: data, encoding: .utf8) ?? "解析失败", isUnauthorized: false)))
                }
            }
        }.resume()
    }

    func fetchKuwoCredentials(completion: @escaping (Result<KuwoCredentials, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/kuwo/credentials", completion: completion)
    }

    func sendKuwoSms(phone: String, password: String, completion: @escaping (Result<String, APIError>) -> Void) {
        requestMessageJSON(path: "/api/portal/kuwo/send-sms", payload: ["phone": phone, "password": password], completion: completion)
    }

    func scheduleKuwoWithdraw(phone: String, password: String, quotaId: String, smsCode: String, targetHour: Int?, immediate: Bool, completion: @escaping (Result<KuwoScheduleResult, APIError>) -> Void) {
        var payload: [String: Any] = [
            "sessions": [["phone": phone, "password": password]],
            "quotaId": quotaId,
            "smsCode": smsCode,
            "immediate": immediate,
        ]
        if !immediate, let targetHour = targetHour {
            payload["targetHour"] = targetHour
        }
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestData(path: "/api/portal/kuwo/schedule-withdraw", method: "POST", headers: ["Content-Type": "application/json"], body: body, completion: completion)
    }

    func updateKuwoSmsCode(taskId: String, smsCode: String, completion: @escaping (Result<String, APIError>) -> Void) {
        requestMessageJSON(path: "/api/portal/kuwo/update-sms-code", payload: ["taskId": taskId, "smsCode": smsCode], completion: completion)
    }

    func fetchKuwoWithdrawStatus(taskId: String? = nil, phone: String? = nil, completion: @escaping (Result<KuwoWithdrawTask?, APIError>) -> Void) {
        var path = "/api/portal/kuwo/withdraw-status"
        if let taskId = taskId, !taskId.isEmpty {
            path += "?taskId=\(taskId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? taskId)"
        } else if let phone = phone, !phone.isEmpty {
            path += "?phone=\(phone.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? phone)"
        }
        APIClient.shared.requestEnvelope(path: path) { (result: Result<APIEnvelope<KuwoWithdrawTask>, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let envelope):
                if envelope.code != 0 {
                    completion(.failure(APIError(message: envelope.msg ?? "查询任务失败", isUnauthorized: false)))
                    return
                }
                completion(.success(envelope.data))
            }
        }
    }

    func fetchWechatRechargeConfig(completion: @escaping (Result<WechatRechargeConfig, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/wechat-recharge/config", completion: completion)
    }

    func createWechatRechargeOrder(fen: Int, completion: @escaping (Result<WechatRechargeCreateResult, APIError>) -> Void) {
        let payload: [String: Any] = ["fen": fen]
        guard let body = try? JSONSerialization.data(withJSONObject: payload) else {
            completion(.failure(APIError(message: "请求参数错误", isUnauthorized: false)))
            return
        }
        APIClient.shared.requestRaw(path: "/api/portal/wechat-recharge/orders", method: "POST", headers: ["Content-Type": "application/json"], body: body) { result in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let text):
                guard let data = text.data(using: .utf8),
                      let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
                    completion(.failure(APIError(message: "数据解析失败", isUnauthorized: false)))
                    return
                }
                let code = json["code"] as? Int ?? -1
                if code != 0 {
                    let msg = json["msg"] as? String ?? "创建订单失败"
                    completion(.failure(APIError(message: msg, isUnauthorized: code == 401 || code == 403)))
                    return
                }
                guard let orderJSON = json["data"] as? [String: Any],
                      let orderData = try? JSONSerialization.data(withJSONObject: orderJSON),
                      let order = try? JSONDecoder().decode(WechatRechargeOrder.self, from: orderData) else {
                    completion(.failure(APIError(message: "创建订单失败", isUnauthorized: false)))
                    return
                }
                let replaced = (json["replaced_previous"] as? Bool) == true
                let message = json["msg"] as? String
                completion(.success(WechatRechargeCreateResult(order: order, replacedPrevious: replaced, message: message)))
            }
        }
    }

    func fetchWechatRechargeOrder(orderNo: String, completion: @escaping (Result<WechatRechargeOrder, APIError>) -> Void) {
        let encoded = orderNo.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? orderNo
        APIClient.shared.requestData(path: "/api/portal/wechat-recharge/orders/\(encoded)", completion: completion)
    }

    func fetchWechatRechargeHistory(page: Int = 1, limit: Int = 20, completion: @escaping (Result<WechatRechargeHistoryPage, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/wechat-recharge/orders?page=\(page)&limit=\(limit)", completion: completion)
    }

    func downloadWechatRechargeQR(pathOrUrl: String, completion: @escaping (Result<Data, APIError>) -> Void) {
        if pathOrUrl.hasPrefix("http"), let url = URL(string: pathOrUrl) {
            APIClient.shared.downloadData(path: "", absoluteURL: url, completion: completion)
        } else {
            APIClient.shared.downloadData(path: pathOrUrl, completion: completion)
        }
    }
}

/// 京东任务 SSE 日志流（与 portal.html EventSource 行为一致）
final class JdTaskLogStreamer: NSObject, URLSessionDataDelegate {
    private let taskId: String
    private let onLine: (String) -> Void
    private let onDone: () -> Void
    private let onError: (APIError) -> Void
    private var buffer = ""
    private var session: URLSession!
    private var dataTask: URLSessionDataTask?
    private var finished = false

    init(taskId: String, onLine: @escaping (String) -> Void, onDone: @escaping () -> Void, onError: @escaping (APIError) -> Void) {
        self.taskId = taskId
        self.onLine = onLine
        self.onDone = onDone
        self.onError = onError
        super.init()
        let config = URLSessionConfiguration.default
        config.httpCookieStorage = HTTPCookieStorage.shared
        config.httpShouldSetCookies = true
        config.timeoutIntervalForRequest = 300
        config.timeoutIntervalForResource = 600
        session = URLSession(configuration: config, delegate: self, delegateQueue: .main)
    }

    func start() {
        guard let url = URL(
            string: "/api/portal/jd/task/logs?taskId=\(taskId.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? taskId)",
            relativeTo: AppEnvironment.baseURL
        ) else {
            onError(APIError(message: "日志地址无效", isUnauthorized: false))
            return
        }
        var request = URLRequest(url: url)
        request.timeoutInterval = 300
        PortalRequestMeta.defaultHeaders.forEach { request.setValue($1, forHTTPHeaderField: $0) }
        dataTask = session.dataTask(with: request)
        dataTask?.resume()
    }

    func cancel() {
        finished = true
        dataTask?.cancel()
        dataTask = nil
        session.invalidateAndCancel()
    }

    func urlSession(_ session: URLSession, dataTask: URLSessionDataTask, didReceive data: Data) {
        guard let chunk = String(data: data, encoding: .utf8) else { return }
        buffer += chunk
        drainBuffer()
    }

    func urlSession(_ session: URLSession, task: URLSessionTask, didCompleteWithError error: Error?) {
        guard !finished else { return }
        if let error = error as NSError?, error.code != NSURLErrorCancelled {
            onError(APIError(message: sanitizeErrorMessage(error.localizedDescription), isUnauthorized: false))
        } else if !finished {
            finish()
        }
    }

    private func drainBuffer() {
        while let range = buffer.range(of: "\n\n") {
            let block = String(buffer[..<range.lowerBound])
            buffer = String(buffer[range.upperBound...])
            parseSSEBlock(block)
        }
    }

    private func parseSSEBlock(_ block: String) {
        var eventName = "message"
        var dataLines: [String] = []
        block.components(separatedBy: "\n").forEach { line in
            if line.hasPrefix("event: ") {
                eventName = String(line.dropFirst(7)).trimmingCharacters(in: .whitespaces)
            } else if line.hasPrefix("data: ") {
                dataLines.append(String(line.dropFirst(6)))
            }
        }
        let payload = dataLines.joined(separator: "\n")
        if eventName == "done" || payload.contains("任务执行完成") || payload.contains("=====DONE=====") {
            if !payload.isEmpty && !payload.contains("任务执行完成") {
                onLine(payload)
            }
            finish()
            return
        }
        if eventName == "error" || eventName == "timeout" {
            onError(APIError(message: payload.isEmpty ? "日志连接异常" : payload, isUnauthorized: false))
            finish()
            return
        }
        if !payload.isEmpty {
            onLine(payload)
        }
    }

    private func finish() {
        guard !finished else { return }
        finished = true
        dataTask?.cancel()
        onDone()
    }
}


@objc protocol ResetableViewController {
    func resetToInitialState()
}

@objc protocol InnerTabSwipeHandling: AnyObject {
    var innerTabCount: Int { get }
    var innerTabIndex: Int { get }
    func selectInnerTab(at index: Int)
    @objc optional func consumeInnerSwipeBoundary(direction: Int) -> Bool
}

class BaseNativeViewController: UIViewController, ResetableViewController, UIGestureRecognizerDelegate {
    /// 子类可关闭全屏点击收键盘（如京东短信登录页）
    var shouldEnableKeyboardDismissOnTap: Bool { true }

    override func viewDidLoad() {
        super.viewDidLoad()
        guard shouldEnableKeyboardDismissOnTap else { return }
        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissKeyboardAnywhere))
        tap.cancelsTouchesInView = false
        tap.delegate = self
        view.addGestureRecognizer(tap)
    }

    @objc private func dismissKeyboardAnywhere() {
        view.endEditing(true)
    }

    func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldReceive touch: UITouch) -> Bool {
        var current: UIView? = touch.view
        while let view = current {
            if view is UIControl || view is UITextField || view is UITextView {
                return false
            }
            if view.accessibilityIdentifier == "protocol_account_picker_row" {
                return false
            }
            current = view.superview
        }
        return true
    }

    func showMessage(_ message: String, title: String = "提示", completion: (() -> Void)? = nil) {
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "确定", style: .default) { _ in completion?() })
        present(alert, animated: true)
    }

    func handle(_ error: APIError) {
        if error.isUnauthorized {
            AppSessionStore.shared.clearSession(requireLogin: true)
            return
        }
        showMessage(sanitizeErrorMessage(error.message))
    }

    func resetToInitialState() {}
}


extension UIView {
    func applyCardStyle(cornerRadius: CGFloat = 18) {
        backgroundColor = .secondarySystemBackground
        layer.cornerRadius = cornerRadius
        layer.shadowColor = UIColor.black.cgColor
        layer.shadowOpacity = 0.08
        layer.shadowOffset = CGSize(width: 0, height: 6)
        layer.shadowRadius = 12
        layer.masksToBounds = false
    }
}


extension UIButton {
    func applyPrimaryStyle(color: UIColor) {
        backgroundColor = color
        setTitleColor(.white, for: .normal)
        titleLabel?.font = UIFont.systemFont(ofSize: 17, weight: .semibold)
        layer.cornerRadius = 16
        layer.shadowColor = color.cgColor
        layer.shadowOpacity = 0.22
        layer.shadowOffset = CGSize(width: 0, height: 4)
        layer.shadowRadius = 12
    }

    func applySecondaryStyle() {
        backgroundColor = .secondarySystemBackground
        setTitleColor(.label, for: .normal)
        titleLabel?.font = UIFont.systemFont(ofSize: 16, weight: .medium)
        layer.cornerRadius = 16
        layer.borderWidth = 1
        layer.borderColor = UIColor.systemGray4.cgColor
    }
}


extension UITextField {
    func applyAppInputStyle(placeholder: String) {
        self.placeholder = placeholder
        borderStyle = .none
        backgroundColor = .secondarySystemBackground
        layer.cornerRadius = 14
        layer.borderWidth = 1
        layer.borderColor = UIColor.systemGray5.cgColor
        font = UIFont.systemFont(ofSize: 16)
        textColor = .label
        autocorrectionType = .no
        autocapitalizationType = .none
        leftView = UIView(frame: CGRect(x: 0, y: 0, width: 14, height: 1))
        leftViewMode = .always
    }
}


final class InfoCardView: UIView {
    private let titleLabel = UILabel()
    private let valueLabel = UILabel()

    init(title: String, value: String, tint: UIColor) {
        super.init(frame: .zero)
        applyCardStyle()
        titleLabel.text = title
        titleLabel.font = UIFont.systemFont(ofSize: 13, weight: .medium)
        titleLabel.textColor = tint
        valueLabel.text = value
        valueLabel.font = UIFont.systemFont(ofSize: 24, weight: .bold)
        valueLabel.textColor = .label
        valueLabel.numberOfLines = 0
        let stack = UIStackView(arrangedSubviews: [titleLabel, valueLabel])
        stack.axis = .vertical
        stack.spacing = 8
        stack.translatesAutoresizingMaskIntoConstraints = false
        addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -16)
        ])
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func updateValue(_ value: String) {
        valueLabel.text = value
    }
}


final class SectionHeroView: UIView {
    private let iconWrap = UIView()
    private let iconView = UIImageView()
    private let titleLabel = UILabel()
    private let subtitleLabel = UILabel()

    init(icon: String, title: String, subtitle: String, tint: UIColor) {
        super.init(frame: .zero)
        layer.cornerRadius = 24
        backgroundColor = tint.withAlphaComponent(0.08)
        layer.borderWidth = 1
        layer.borderColor = tint.withAlphaComponent(0.12).cgColor

        iconWrap.translatesAutoresizingMaskIntoConstraints = false
        iconWrap.backgroundColor = tint
        iconWrap.layer.cornerRadius = 20

        iconView.translatesAutoresizingMaskIntoConstraints = false
        iconView.image = UIImage(systemName: icon)
        iconView.tintColor = .white
        iconView.contentMode = .scaleAspectFit

        titleLabel.translatesAutoresizingMaskIntoConstraints = false
        titleLabel.text = title
        titleLabel.font = UIFont.systemFont(ofSize: 24, weight: .bold)
        titleLabel.textColor = .label

        subtitleLabel.translatesAutoresizingMaskIntoConstraints = false
        subtitleLabel.text = subtitle
        subtitleLabel.font = UIFont.systemFont(ofSize: 13)
        subtitleLabel.textColor = .secondaryLabel
        subtitleLabel.numberOfLines = 0

        let textStack = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel])
        textStack.axis = .vertical
        textStack.spacing = 6
        textStack.translatesAutoresizingMaskIntoConstraints = false

        addSubview(iconWrap)
        iconWrap.addSubview(iconView)
        addSubview(textStack)

        NSLayoutConstraint.activate([
            iconWrap.topAnchor.constraint(equalTo: topAnchor, constant: 18),
            iconWrap.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 18),
            iconWrap.widthAnchor.constraint(equalToConstant: 40),
            iconWrap.heightAnchor.constraint(equalToConstant: 40),
            iconView.centerXAnchor.constraint(equalTo: iconWrap.centerXAnchor),
            iconView.centerYAnchor.constraint(equalTo: iconWrap.centerYAnchor),
            iconView.widthAnchor.constraint(equalToConstant: 20),
            iconView.heightAnchor.constraint(equalToConstant: 20),
            textStack.centerYAnchor.constraint(equalTo: iconWrap.centerYAnchor),
            textStack.leadingAnchor.constraint(equalTo: iconWrap.trailingAnchor, constant: 12),
            textStack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -18),
            bottomAnchor.constraint(equalTo: subtitleLabel.bottomAnchor, constant: 18)
        ])
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
}


final class PromptTipView: UIView, WKNavigationDelegate {
    private let webView: WKWebView
    private var heightConstraint: NSLayoutConstraint?

    init(text: String) {
        let config = WKWebViewConfiguration()
        config.dataDetectorTypes = [.link]
        if #available(iOS 10.0, *) {
            config.mediaTypesRequiringUserActionForPlayback = []
        }
        webView = WKWebView(frame: .zero, configuration: config)
        webView.translatesAutoresizingMaskIntoConstraints = false
        webView.isOpaque = false
        webView.backgroundColor = .clear
        webView.scrollView.backgroundColor = .clear
        webView.scrollView.isScrollEnabled = false
        webView.scrollView.bounces = false

        super.init(frame: .zero)
        webView.navigationDelegate = self

        backgroundColor = UIColor.systemGray6
        layer.cornerRadius = 16
        layer.borderWidth = 1
        layer.borderColor = UIColor.systemGray5.cgColor

        let badge = UILabel()
        badge.translatesAutoresizingMaskIntoConstraints = false
        badge.text = "玩法和说明"
        badge.font = UIFont.systemFont(ofSize: 12, weight: .semibold)
        badge.textColor = .systemBlue

        let heightC = webView.heightAnchor.constraint(greaterThanOrEqualToConstant: 60)
        self.heightConstraint = heightC

        addSubview(badge)
        addSubview(webView)
        NSLayoutConstraint.activate([
            badge.topAnchor.constraint(equalTo: topAnchor, constant: 14),
            badge.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            badge.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            webView.topAnchor.constraint(equalTo: badge.bottomAnchor, constant: 4),
            webView.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 8),
            webView.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -8),
            webView.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -8),
            heightC,
        ])

        let html = Self.markdownToHtml(text)
        let baseURL = URL(string: "http://180.152.5.230:5701/")
        webView.loadHTMLString(html, baseURL: baseURL)
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        webView.evaluateJavaScript("document.body.scrollHeight") { [weak self] result, _ in
            if let height = result as? CGFloat, height > 0 {
                DispatchQueue.main.async {
                    self?.heightConstraint?.constant = height + 20
                }
            }
        }
    }

    /// Markdown转HTML
    static func markdownToHtml(_ md: String) -> String {
        var h = md
            .replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;")

        // 代码块
        h = h.replacingOccurrences(of: #"```(\w*)\n([\s\S]*?)```"#, with: "<pre><code>$2</code></pre>", options: .regularExpression)
        h = h.replacingOccurrences(of: #"`([^`]+)`"#, with: "<code>$1</code>", options: .regularExpression)
        // 图片/视频/音频
        h = replaceMediaTags(in: h)
        // 链接
        h = h.replacingOccurrences(of: #"\[([^\]]+)\]\(([^)]+)\)"#, with: #"<a href="$2" target="_blank">$1</a>"#, options: .regularExpression)
        // 标题（逐行处理，不用anchorsMatchLines）
        h = processLineByLine(h) { line in
            if line.hasPrefix("### ") { return "<h3>" + String(line.dropFirst(4)) + "</h3>" }
            if line.hasPrefix("## ") { return "<h2>" + String(line.dropFirst(3)) + "</h2>" }
            if line.hasPrefix("# ") { return "<h1>" + String(line.dropFirst(2)) + "</h1>" }
            if line.hasPrefix("&gt; ") { return #"<blockquote style="border-left:3px solid #6366f1;padding-left:12px;color:#64748b;margin:8px 0;">"# + String(line.dropFirst(5)) + "</blockquote>" }
            if line == "---" { return #"<hr style="border:none;border-top:1px solid #e2e8f0;margin:12px 0;">"# }
            if line.hasPrefix("- ") { return "<li>" + String(line.dropFirst(2)) + "</li>" }
            if let r = line.range(of: #"^\d+\. "#, options: .regularExpression) {
                return "<li>" + String(line[r.upperBound...]) + "</li>"
            }
            return nil
        }
        // 粗体、斜体、删除线
        h = h.replacingOccurrences(of: #"\*\*(.+?)\*\*"#, with: "<strong>$1</strong>", options: .regularExpression)
        h = h.replacingOccurrences(of: #"\*(.+?)\*"#, with: "<em>$1</em>", options: .regularExpression)
        h = h.replacingOccurrences(of: #"~~(.+?)~~"#, with: "<del>$1</del>", options: .regularExpression)
        // 换行
        h = h.replacingOccurrences(of: "\n\n", with: "</p><p>")
        h = h.replacingOccurrences(of: "\n", with: "<br>")
        h = "<p>" + h + "</p>"
        h = h.replacingOccurrences(of: #"<p>\s*</p>"#, with: "", options: .regularExpression)

        return """
        <!DOCTYPE html><html><head><meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1">
        <meta name="color-scheme" content="light dark">
        <style>
        body{font-family:-apple-system,sans-serif;font-size:14px;font-weight:500;color:#1c1c1e;line-height:1.7;margin:0;padding:0 4px;-webkit-text-size-adjust:100%;}
        img,video{max-width:100%;max-height:300px;object-fit:contain;border-radius:8px;margin:6px 0;}
        audio{width:100%;margin:6px 0;}
        pre{background:#1e293b;color:#e2e8f0;padding:12px;border-radius:6px;overflow-x:auto;}
        code{background:#f1f5f9;padding:2px 5px;border-radius:3px;font-size:12px;}pre code{background:none;color:inherit;padding:0;}
        a{color:#6366f1;}h1{font-size:17px;font-weight:700;margin:12px 0 6px;}h2{font-size:15px;font-weight:700;margin:10px 0 6px;}
        h3{font-size:14px;font-weight:700;margin:8px 0 4px;}ul{padding-left:20px;}
        blockquote{border-left:3px solid #6366f1;padding-left:12px;color:#64748b;margin:8px 0;}
        hr{border:none;border-top:1px solid #e2e8f0;margin:12px 0;}
        @media(prefers-color-scheme:dark){
            body{color:#e5e5e7;}
            code{background:#2d2d3a;color:#e5e5e7;}
            pre{background:#1a1a2e;color:#e2e8f0;}
            blockquote{color:#a1a1aa;}
            hr{border-top-color:#3a3a4a;}
            a{color:#818cf8;}
        }
        </style></head>
        <body>\(h)</body></html>
        """
    }

    /// 根据文件扩展名替换媒体标签
    private static func replaceMediaTags(in text: String) -> String {
        guard let regex = try? NSRegularExpression(pattern: #"!\[([^\]]*)\]\(([^)]+)\)"#) else { return text }
        let nsText = text as NSString
        let matches = regex.matches(in: text, range: NSRange(location: 0, length: nsText.length))
        var result = text
        for match in matches.reversed() {
            let fullRange = match.range
            let url = nsText.substring(with: match.range(at: 2))
            let alt = nsText.substring(with: match.range(at: 1))
            let ext = (url.split(separator: ".").last?.split(separator: "?").first.map(String.init) ?? "").lowercased()
            let tag: String
            if ["mp4", "webm", "mov", "avi"].contains(ext) {
                tag = #"<video src="\#(url)" controls preload="metadata" playsinline webkit-playsinline style="max-width:100%;max-height:300px;object-fit:contain;border-radius:8px;margin:6px 0;"></video>"#
            } else if ["mp3", "wav", "ogg", "m4a", "aac", "flac"].contains(ext) {
                tag = #"<audio src="\#(url)" controls preload="metadata" style="width:100%;margin:6px 0;"></audio>"#
            } else {
                tag = #"<img src="\#(url)" alt="\#(alt)" style="max-width:100%;max-height:300px;object-fit:contain;border-radius:8px;margin:6px 0;">"#
            }
            if let swiftRange = Range(fullRange, in: result) {
                result.replaceSubrange(swiftRange, with: tag)
            }
        }
        return result
    }

    /// 逐行处理Markdown，支持标题/引用/列表/分割线
    private static func processLineByLine(_ text: String, transform: (String) -> String?) -> String {
        let lines = text.components(separatedBy: "\n")
        var result: [String] = []
        for line in lines {
            if let transformed = transform(line) {
                result.append(transformed)
            } else {
                result.append(line)
            }
        }
        return result.joined(separator: "\n")
    }
}


final class StatusBadgeLabel: UILabel {
    enum Kind { case active, expiring, expired, gray }

    func configure(text: String, kind: Kind) {
        self.text = "  \(text)  "
        font = UIFont.systemFont(ofSize: 11, weight: .bold)
        layer.cornerRadius = 8
        clipsToBounds = true
        textAlignment = .center
        switch kind {
        case .active:
            backgroundColor = UIColor.systemGreen.withAlphaComponent(0.12)
            textColor = .systemGreen
        case .expiring:
            backgroundColor = UIColor.systemOrange.withAlphaComponent(0.12)
            textColor = .systemOrange
        case .expired:
            backgroundColor = UIColor.systemRed.withAlphaComponent(0.12)
            textColor = .systemRed
        case .gray:
            backgroundColor = UIColor.systemGray5
            textColor = .secondaryLabel
        }
        sizeToFit()
    }
}


final class ActionButton: UIView {
    private let iconView = UIImageView()
    private let titleLabel = UILabel()
    private let descLabel = UILabel()
    var tapAction: (() -> Void)?

    init(icon: String, title: String, desc: String, tint: UIColor) {
        super.init(frame: .zero)
        backgroundColor = .secondarySystemBackground
        layer.cornerRadius = 16
        layer.borderWidth = 1
        layer.borderColor = UIColor.systemGray5.cgColor

        iconView.translatesAutoresizingMaskIntoConstraints = false
        iconView.image = UIImage(systemName: icon)
        iconView.tintColor = tint
        iconView.contentMode = .scaleAspectFit

        titleLabel.translatesAutoresizingMaskIntoConstraints = false
        titleLabel.text = title
        titleLabel.font = UIFont.systemFont(ofSize: 16, weight: .semibold)
        titleLabel.textColor = .label

        descLabel.translatesAutoresizingMaskIntoConstraints = false
        descLabel.text = desc
        descLabel.font = UIFont.systemFont(ofSize: 12)
        descLabel.textColor = .secondaryLabel
        descLabel.numberOfLines = 0

        let textStack = UIStackView(arrangedSubviews: [titleLabel, descLabel])
        textStack.axis = .vertical
        textStack.spacing = 3
        textStack.translatesAutoresizingMaskIntoConstraints = false

        let bg = UIView()
        bg.translatesAutoresizingMaskIntoConstraints = false
        bg.backgroundColor = tint.withAlphaComponent(0.1)
        bg.layer.cornerRadius = 10
        bg.addSubview(iconView)

        addSubview(bg)
        addSubview(textStack)

        NSLayoutConstraint.activate([
            bg.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 14),
            bg.centerYAnchor.constraint(equalTo: centerYAnchor),
            bg.widthAnchor.constraint(equalToConstant: 36),
            bg.heightAnchor.constraint(equalToConstant: 36),
            iconView.centerXAnchor.constraint(equalTo: bg.centerXAnchor),
            iconView.centerYAnchor.constraint(equalTo: bg.centerYAnchor),
            iconView.widthAnchor.constraint(equalToConstant: 18),
            iconView.heightAnchor.constraint(equalToConstant: 18),
            textStack.leadingAnchor.constraint(equalTo: bg.trailingAnchor, constant: 12),
            textStack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -14),
            textStack.centerYAnchor.constraint(equalTo: centerYAnchor),
            heightAnchor.constraint(equalToConstant: 68)
        ])

        let tap = UITapGestureRecognizer(target: self, action: #selector(tapped))
        addGestureRecognizer(tap)
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    @objc private func tapped() {
        guard isEnabled else { return }
        tapAction?()
    }

    func updateDesc(_ text: String) { descLabel.text = text }

    func updateTitle(_ text: String) { titleLabel.text = text }

    var isEnabled = true {
        didSet {
            alpha = isEnabled ? 1 : 0.5
            isUserInteractionEnabled = isEnabled
        }
    }
}


final class EmptyStateView: UIView {
    private let iconView = UIImageView()
    private let titleLabel = UILabel()
    private let descLabel = UILabel()

    init(icon: String, title: String, desc: String) {
        super.init(frame: .zero)
        iconView.translatesAutoresizingMaskIntoConstraints = false
        iconView.image = UIImage(systemName: icon)
        iconView.tintColor = .systemGray3
        iconView.contentMode = .scaleAspectFit

        titleLabel.translatesAutoresizingMaskIntoConstraints = false
        titleLabel.text = title
        titleLabel.font = UIFont.systemFont(ofSize: 18, weight: .bold)
        titleLabel.textColor = .label

        descLabel.translatesAutoresizingMaskIntoConstraints = false
        descLabel.text = desc
        descLabel.font = UIFont.systemFont(ofSize: 14)
        descLabel.textColor = .secondaryLabel
        descLabel.numberOfLines = 0
        descLabel.textAlignment = .center

        let stack = UIStackView(arrangedSubviews: [iconView, titleLabel, descLabel])
        stack.axis = .vertical
        stack.spacing = 12
        stack.alignment = .center
        stack.translatesAutoresizingMaskIntoConstraints = false
        addSubview(stack)
        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: centerXAnchor),
            stack.centerYAnchor.constraint(equalTo: centerYAnchor),
            stack.leadingAnchor.constraint(greaterThanOrEqualTo: leadingAnchor, constant: 40),
            stack.trailingAnchor.constraint(lessThanOrEqualTo: trailingAnchor, constant: -40),
            iconView.widthAnchor.constraint(equalToConstant: 48),
            iconView.heightAnchor.constraint(equalToConstant: 48)
        ])
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
}


final class ProjectSummaryCardView: UIView {
    let titleLabel = UILabel()
    let metaLabel = UILabel()
    let priceLabel = UILabel()
    let badgeLabel = StatusBadgeLabel()
    let chevronView = UIImageView()

    override init(frame: CGRect) {
        super.init(frame: frame)
        backgroundColor = .secondarySystemBackground
        layer.cornerRadius = 18
        layer.borderWidth = 1
        layer.borderColor = UIColor.systemGray5.cgColor

        titleLabel.font = UIFont.systemFont(ofSize: 17, weight: .semibold)
        titleLabel.textColor = .label
        titleLabel.numberOfLines = 2
        titleLabel.translatesAutoresizingMaskIntoConstraints = false

        metaLabel.font = UIFont.systemFont(ofSize: 12)
        metaLabel.textColor = .secondaryLabel
        metaLabel.numberOfLines = 2
        metaLabel.translatesAutoresizingMaskIntoConstraints = false

        priceLabel.font = UIFont.systemFont(ofSize: 13, weight: .bold)
        priceLabel.textColor = .systemBlue
        priceLabel.translatesAutoresizingMaskIntoConstraints = false

        badgeLabel.translatesAutoresizingMaskIntoConstraints = false

        chevronView.translatesAutoresizingMaskIntoConstraints = false
        chevronView.image = UIImage(systemName: "chevron.right")
        chevronView.tintColor = .systemGray3
        chevronView.contentMode = .scaleAspectFit

        addSubview(titleLabel)
        addSubview(metaLabel)
        addSubview(priceLabel)
        addSubview(badgeLabel)
        addSubview(chevronView)

        NSLayoutConstraint.activate([
            titleLabel.topAnchor.constraint(equalTo: topAnchor, constant: 16),
            titleLabel.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            titleLabel.trailingAnchor.constraint(lessThanOrEqualTo: chevronView.leadingAnchor, constant: -12),
            badgeLabel.topAnchor.constraint(equalTo: titleLabel.bottomAnchor, constant: 10),
            badgeLabel.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            metaLabel.topAnchor.constraint(equalTo: badgeLabel.bottomAnchor, constant: 10),
            metaLabel.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            metaLabel.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            priceLabel.topAnchor.constraint(equalTo: metaLabel.bottomAnchor, constant: 10),
            priceLabel.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            priceLabel.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -16),
            chevronView.centerYAnchor.constraint(equalTo: centerYAnchor),
            chevronView.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            chevronView.widthAnchor.constraint(equalToConstant: 12),
            chevronView.heightAnchor.constraint(equalToConstant: 18)
        ])
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
}


enum QRImageDecoder {
    static func decodeImage(from raw: String) -> UIImage? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty { return nil }
        if let url = URL(string: trimmed), trimmed.hasPrefix("http"), let data = try? Data(contentsOf: url) {
            return UIImage(data: data)
        }
        let payload: String
        if let range = trimmed.range(of: "base64,") {
            payload = String(trimmed[range.upperBound...])
        } else if let comma = trimmed.firstIndex(of: ","), trimmed.contains("data:image") {
            payload = String(trimmed[trimmed.index(after: comma)...])
        } else {
            payload = trimmed
        }
        let normalized = payload.replacingOccurrences(of: "\n", with: "")
            .replacingOccurrences(of: "\r", with: "")
            .replacingOccurrences(of: " ", with: "")
        let padding = normalized.count % 4
        let padded = padding == 0 ? normalized : normalized + String(repeating: "=", count: 4 - padding)
        guard let data = Data(base64Encoded: padded, options: .ignoreUnknownCharacters) else { return nil }
        return UIImage(data: data)
    }
}
