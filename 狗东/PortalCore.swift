import UIKit
import Foundation
import Security


enum AppEnvironment {
    static let baseURL = URL(string: "http://180.152.5.230:5701")!
    static let coinPurchaseURL = URL(string: "http://180.152.5.230:8005/#/")!
    static let groupURL = URL(string: "https://qm.qq.com/q/4gYwV6YzPW")!
}


enum AppNotifications {
    static let sessionDidChange = Notification.Name("PortalSessionDidChange")
    static let sessionDidLogout = Notification.Name("PortalSessionDidLogout")
    static let sessionRequiresLogin = Notification.Name("PortalSessionRequiresLogin")
    static let jdAuthCallback = Notification.Name("JDAuthCallbackNotification")
}


struct APIError: Error {
    let message: String
    let isUnauthorized: Bool
}


struct EmptyPayload: Decodable {}


struct APIEnvelope<T: Decodable>: Decodable {
    let code: Int
    let msg: String?
    let data: T?
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
    let prayedToday: Bool?
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


struct SubmitFeedbackPayload: Encodable {
    let type: String
    let title: String
    let content: String
    let contact: String
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
    let qingLongConfig: String?
    let guide: String?
    let inputFields: [PortalActivityField]?
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
    let needCoin: Int?
    let bizStatus: String?
    let bizStatusText: String?
    let daysLeft: Int?
    let priceText: String?
    let inputFields: [PortalActivityField]?
    let ckTemplate: String?
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


struct PortalHomeSnapshot {
    let dashboard: PortalDashboard
    let profile: PortalProfile
    let wechatStatus: PortalWechatStatus?
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
        headers.forEach { request.setValue($1, forHTTPHeaderField: $0) }

        session.dataTask(with: request) { data, response, error in
            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: error.localizedDescription, isUnauthorized: false)))
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
                    // HTTP 层认证错误已在 requestEnvelope 中处理，此处只处理业务错误
                    // 不再根据消息文本判断 isUnauthorized，避免误判
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
        headers.forEach { request.setValue($1, forHTTPHeaderField: $0) }

        session.dataTask(with: request) { data, _, error in
            if let error = error {
                DispatchQueue.main.async {
                    completion(.failure(APIError(message: error.localizedDescription, isUnauthorized: false)))
                }
                return
            }
            let text = data.flatMap { String(data: $0, encoding: .utf8) }?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            DispatchQueue.main.async {
                completion(.success(text))
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
        var dashboard: PortalDashboard?
        var profile: PortalProfile?
        var wxStatus: PortalWechatStatus?
        var capturedUnauthorized: APIError?
        var capturedError: APIError?
        let group = DispatchGroup()

        group.enter()
        APIClient.shared.requestData(path: "/api/portal/dashboard") { (result: Result<PortalDashboard, APIError>) in
            if case .success(let value) = result { dashboard = value }
            if case .failure(let error) = result {
                if error.isUnauthorized { capturedUnauthorized = error }
                else { capturedError = error }
            }
            group.leave()
        }

        group.enter()
        APIClient.shared.requestData(path: "/api/portal/profile") { (result: Result<PortalProfile, APIError>) in
            if case .success(let value) = result { profile = value }
            if case .failure(let error) = result {
                if error.isUnauthorized { capturedUnauthorized = error }
                else { capturedError = error }
            }
            group.leave()
        }

        group.enter()
        APIClient.shared.requestData(path: "/api/portal/wx/status") { (result: Result<PortalWechatStatus, APIError>) in
            if case .success(let value) = result { wxStatus = value }
            // wx/status 失败不视为致命错误，不阻止首页加载
            group.leave()
        }

        group.notify(queue: .main) {
            // 认证错误优先
            if let authError = capturedUnauthorized {
                completion(.failure(authError))
                return
            }
            guard let dashboard = dashboard, let profile = profile else {
                completion(.failure(capturedError ?? APIError(message: "首页数据不完整", isUnauthorized: false)))
                return
            }
            completion(.success(PortalHomeSnapshot(dashboard: dashboard, profile: profile, wechatStatus: wxStatus)))
        }
    }

    func fetchActivities(completion: @escaping (Result<[PortalActivity], APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/activities", completion: completion)
    }

    func fetchProjects(completion: @escaping (Result<[PortalProject], APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/projects", completion: completion)
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
        APIClient.shared.requestMessage(path: "/api/portal/checkin", completion: completion)
    }

    func pray(completion: @escaping (Result<String, APIError>) -> Void) {
        APIClient.shared.requestMessage(path: "/api/portal/pray", completion: completion)
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
        APIClient.shared.requestData(path: path) { (result: Result<NotificationsData, APIError>) in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let data):
                completion(.success(PortalNotificationPage(list: data.list, total: data.total, unread: data.unread)))
            }
        }
    }

    func fetchNotificationDetail(id: Int, completion: @escaping (Result<PortalNotification, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/notification?id=\(id)", completion: completion)
    }

    func submitFeedback(type: String, title: String, content: String, contact: String, completion: @escaping (Result<String, APIError>) -> Void) {
        let payload: [String: Any] = ["type": type, "title": title, "content": content, "contact": contact]
        requestMessageJSON(path: "/api/portal/feedback", payload: payload, completion: completion)
    }

    func verifySession(completion: @escaping (Result<Void, APIError>) -> Void) {
        APIClient.shared.requestData(path: "/api/portal/dashboard") { (result: Result<PortalDashboard, APIError>) in
            switch result {
            case .success:
                completion(.success(()))
            case .failure(let error):
                completion(.failure(error))
            }
        }
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
}


@objc protocol ResetableViewController {
    func resetToInitialState()
}

class BaseNativeViewController: UIViewController, ResetableViewController {
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
        showMessage(error.message)
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


final class PromptTipView: UIView {
    private let textView = UITextView()

    init(text: String) {
        super.init(frame: .zero)
        backgroundColor = UIColor.systemGray6
        layer.cornerRadius = 16
        layer.borderWidth = 1
        layer.borderColor = UIColor.systemGray5.cgColor

        let badge = UILabel()
        badge.translatesAutoresizingMaskIntoConstraints = false
        badge.text = "玩法和说明"
        badge.font = UIFont.systemFont(ofSize: 12, weight: .semibold)
        badge.textColor = .systemBlue

        textView.translatesAutoresizingMaskIntoConstraints = false
        textView.isEditable = false
        textView.isScrollEnabled = false
        textView.showsVerticalScrollIndicator = false
        textView.backgroundColor = .clear
        textView.font = UIFont.systemFont(ofSize: 14, weight: .medium)
        textView.textColor = .label
        textView.text = text
        textView.textContainerInset = UIEdgeInsets(top: 0, left: -4, bottom: 0, right: -4)
        textView.setContentCompressionResistancePriority(.required, for: .vertical)

        addSubview(badge)
        addSubview(textView)
        NSLayoutConstraint.activate([
            badge.topAnchor.constraint(equalTo: topAnchor, constant: 14),
            badge.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            badge.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            textView.topAnchor.constraint(equalTo: badge.bottomAnchor, constant: 6),
            textView.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 12),
            textView.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -12),
            textView.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -12),
        ])
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
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

    @objc private func tapped() { tapAction?() }

    func updateDesc(_ text: String) { descLabel.text = text }
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
