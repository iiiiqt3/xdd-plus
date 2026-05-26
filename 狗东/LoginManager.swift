import UIKit
import WebKit

// 京东登录管理
class LoginManager {
    static let shared = LoginManager()
    private init() {}

    @available(iOS 13.0, *)
    func handleLogin(from viewController: UIViewController, qqTextField: UITextField) {
        guard let qq = qqTextField.text, !qq.isEmpty else {
            let alert = UIAlertController(title: "提示", message: "请输入QQ号", preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "确定", style: .default))
            viewController.present(alert, animated: true)
            return
        }

        qqTextField.resignFirstResponder()
        UserDefaults.standard.set(qq, forKey: "qq")

        let webVC = WebViewController()
        viewController.navigationController?.pushViewController(webVC, animated: true)
    }
}

@available(iOS 13.0, *)
class WebViewController: UIViewController, WKNavigationDelegate {
    var webView: WKWebView!
    private let submitButton = UIButton(type: .system)
    private let loadingIndicator = UIActivityIndicatorView(style: .large)
    private let statusLabel = UILabel()
    private var jdCookieString: String?
    private var isLoggedIn = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .white
        title = "京东登录"
        setupUI()
        setupLoadingIndicator()
        setupStatusLabel()
        clearPreviousLoginState()
        loadJDLoginPage()
        NotificationCenter.default.addObserver(self, selector: #selector(handleQQAuthCallback), name: AppNotifications.jdAuthCallback, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(handleAppDidBecomeActive), name: UIApplication.didBecomeActiveNotification, object: nil)
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
        webView?.stopLoading()
        webView?.navigationDelegate = nil
    }

    private func setupUI() {
        setupWebView()
        setupSubmitButton()
    }

    private func setupStatusLabel() {
        statusLabel.text = "状态: 加载中..."
        statusLabel.textAlignment = .center
        statusLabel.font = .systemFont(ofSize: 14)
        statusLabel.textColor = .gray
        statusLabel.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(statusLabel)

        NSLayoutConstraint.activate([
            statusLabel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 20),
            statusLabel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -20),
            statusLabel.bottomAnchor.constraint(equalTo: submitButton.topAnchor, constant: -10)
        ])
    }

    private func setupLoadingIndicator() {
        loadingIndicator.translatesAutoresizingMaskIntoConstraints = false
        loadingIndicator.color = .systemBlue
        loadingIndicator.hidesWhenStopped = true
        view.addSubview(loadingIndicator)

        NSLayoutConstraint.activate([
            loadingIndicator.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            loadingIndicator.centerYAnchor.constraint(equalTo: view.centerYAnchor)
        ])
    }

    func setupWebView() {
        let config = WKWebViewConfiguration()
        let preferences = WKPreferences()
        preferences.javaScriptEnabled = true
        config.preferences = preferences
        config.websiteDataStore = WKWebsiteDataStore.default()
        webView = WKWebView(frame: .zero, configuration: config)
        webView.translatesAutoresizingMaskIntoConstraints = false
        webView.navigationDelegate = self
        webView.isHidden = true
        view.addSubview(webView)

        NSLayoutConstraint.activate([
            webView.topAnchor.constraint(equalTo: view.topAnchor),
            webView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            webView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor, constant: -100)
        ])
    }

    func setupSubmitButton() {
        submitButton.setTitle("登录后点击任意页面后提交", for: .normal)
        submitButton.titleLabel?.font = .systemFont(ofSize: 16, weight: .bold)
        submitButton.backgroundColor = .systemGray
        submitButton.setTitleColor(.white, for: .normal)
        submitButton.layer.cornerRadius = 8
        submitButton.translatesAutoresizingMaskIntoConstraints = false
        submitButton.addTarget(self, action: #selector(submitButtonTapped), for: .touchUpInside)
        submitButton.isEnabled = false

        view.addSubview(submitButton)

        NSLayoutConstraint.activate([
            submitButton.heightAnchor.constraint(equalToConstant: 50),
            submitButton.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 20),
            submitButton.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -20),
            submitButton.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor, constant: -10)
        ])

        view.bringSubviewToFront(submitButton)
    }

    private func clearPreviousLoginState() {
        jdCookieString = nil
        isLoggedIn = false
        updateStatus("准备加载登录页面...")
        clearJDCache { [weak self] in
            self?.submitButton.isEnabled = false
            self?.submitButton.setTitle("登录后点击京东任意页面后提交", for: .normal)
            self?.submitButton.backgroundColor = .systemGray
        }
    }

    private func updateStatus(_ text: String) {
        DispatchQueue.main.async {
            self.statusLabel.text = "状态: \(text)"
        }
    }

    func loadJDLoginPage() {
        loadingIndicator.startAnimating()
        updateStatus("正在加载登录页面...")

        let loginUrl = "https://plogin.m.jd.com/login/login?appid=300&returnurl=https%3A%2F%2Fwq.jd.com%2Fpassport%2FLoginRedirect%3Fstate%3D1101806886554%26returnurl%3Dhttps%253A%252F%252Fhome.m.jd.com%252FmyJd%252Fnewhome.action%253Fsceneval%253D2%2526ufc%253D%2526&source=wq_passport"
        guard let url = URL(string: loginUrl) else {
            loadingIndicator.stopAnimating()
            showAlert(message: "登录地址错误")
            return
        }
        let request = URLRequest(url: url, timeoutInterval: 30)
        webView.load(request)

        DispatchQueue.main.asyncAfter(deadline: .now() + 10) { [weak self] in
            guard let self = self else { return }
            if self.loadingIndicator.isAnimating {
                self.loadingIndicator.stopAnimating()
                self.showNetworkErrorAlert()
            }
        }
    }

    func reloadJDLoginPageAfterQQAuth() {
        updateStatus("QQ 授权返回，正在刷新京东登录状态...")
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
            self.webView.reload()
            self.updateCookies()
        }
    }

    @objc private func handleQQAuthCallback() {
        reloadJDLoginPageAfterQQAuth()
    }

    @objc private func handleAppDidBecomeActive() {
        if !isLoggedIn {
            updateCookies()
        }
    }

    private func showNetworkErrorAlert() {
        let alert = UIAlertController(
            title: "网络连接失败",
            message: "无法连接到网络，请检查：\n1. 手机网络是否正常开启\n2. WIFI/蜂窝数据是否可用\n3. 网络是否能正常访问京东",
            preferredStyle: .alert
        )

        alert.addAction(UIAlertAction(title: "重试", style: .default) { [weak self] _ in
            self?.loadJDLoginPage()
        })

        alert.addAction(UIAlertAction(title: "返回", style: .cancel) { [weak self] _ in
            self?.navigationController?.popViewController(animated: true)
        })

        present(alert, animated: true)
    }

    func webView(_ webView: WKWebView, didStartProvisionalNavigation navigation: WKNavigation!) {
        updateStatus("页面加载中...")
    }

    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        loadingIndicator.stopAnimating()
        updateStatus("页面加载失败")
        let nsError = error as NSError
        if nsError.domain == NSURLErrorDomain {
            showNetworkErrorAlert()
            return
        }
        showAlert(message: "页面加载失败: \(error.localizedDescription)")
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        loadingIndicator.stopAnimating()
        UIView.animate(withDuration: 0.3) {
            self.webView.isHidden = false
            self.webView.alpha = 1.0
        }
        updateCookies()
    }

    private func updateCookies() {
        webView.configuration.websiteDataStore.httpCookieStore.getAllCookies { [weak self] cookies in
            guard let self = self else { return }
            DispatchQueue.global().async {
                let jdCookies = cookies.filter { cookie in
                    let domain = cookie.domain.lowercased()
                    return domain.contains("jd.com") || domain.contains(".jd.") || domain.contains("jingdong")
                }

                let sortedCookies = jdCookies.sorted {
                    if $0.domain != $1.domain { return $0.domain < $1.domain }
                    return $0.name < $1.name
                }

                let jdCookieString = sortedCookies.map { "\($0.name)=\($0.value)" }.joined(separator: "; ")
                let hasPTKey = sortedCookies.contains { $0.name == "pt_key" }

                DispatchQueue.main.async {
                    self.jdCookieString = jdCookieString
                    if hasPTKey {
                        self.isLoggedIn = true
                        self.updateStatus("登录成功，可提交Cookie")
                        self.submitButton.setTitle("点击提交Cookie", for: .normal)
                        self.submitButton.backgroundColor = .systemGreen
                        self.submitButton.isEnabled = true
                    } else {
                        self.isLoggedIn = false
                        self.updateStatus("请完成京东账号登录")
                        self.submitButton.isEnabled = false
                        self.submitButton.backgroundColor = .systemGray
                    }
                }
            }
        }
    }

    func webView(_ webView: WKWebView, decidePolicyFor navigationAction: WKNavigationAction, decisionHandler: @escaping (WKNavigationActionPolicy) -> Void) {
        guard let currentUrl = navigationAction.request.url?.absoluteString else {
            decisionHandler(.allow)
            return
        }
        if currentUrl.contains("home.m.jd.com") || currentUrl.contains("m.jd.com") {
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) { [weak self] in
                self?.updateCookies()
            }
        }
        decisionHandler(.allow)
    }

    @objc func submitButtonTapped() {
        guard isLoggedIn else {
            showAlert(message: "请先完成京东账号登录（获取pt_key）")
            return
        }
        guard let cookieString = jdCookieString, !cookieString.isEmpty else {
            showAlert(message: "未能获取到登录信息，请重新登录")
            updateCookies()
            return
        }
        if !cookieString.contains("pt_key") || !cookieString.contains("pt_pin") {
            showAlert(message: "Cookie不完整，缺少pt_key或pt_pin，请重新登录")
            return
        }
        submitButton.setTitle("提交中...", for: .normal)
        submitButton.isEnabled = false
        updateStatus("正在提交Cookie...")
        postDataToServer(cookie: cookieString)
    }

    func postDataToServer(cookie: String) {
        guard let qq = UserDefaults.standard.string(forKey: "qq") else {
            DispatchQueue.main.async {
                self.showAlert(message: "数据异常，请返回上一页重试。")
                self.resetSubmitButton()
            }
            return
        }

        guard let url = URL(string: "http://180.152.5.230:5701/api/login/smslogin") else {
            DispatchQueue.main.async {
                self.showAlert(message: "服务器地址错误，请稍后重试。")
                self.resetSubmitButton()
            }
            return
        }

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.timeoutInterval = 60
        let boundary = "Boundary-\(UUID().uuidString)"
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        let httpBody = createFormDataBody(boundary: boundary, parameters: [
            "qq": qq,
            "ck": cookie,
            "token": "123456"
        ])
        request.httpBody = httpBody

        URLSession.shared.dataTask(with: request) { [weak self] data, response, error in
            DispatchQueue.main.async {
                defer { self?.resetSubmitButton() }
                if let error = error {
                    self?.showAlert(message: "提交失败，请检查网络连接后重试。\n\(error.localizedDescription)")
                    return
                }
                if let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 {
                    self?.showAlert(message: "已成功提交") {
                        self?.clearAllLoginState {
                            self?.navigationController?.popViewController(animated: true)
                        }
                    }
                    return
                }
                let message = data.flatMap { String(data: $0, encoding: .utf8) } ?? "无法解析服务器响应，请稍后重试。"
                self?.showAlert(message: message)
            }
        }.resume()
    }

    private func resetSubmitButton() {
        submitButton.setTitle("点击提交Cookie", for: .normal)
        submitButton.backgroundColor = .systemGreen
        submitButton.isEnabled = true
        updateStatus("提交完成，可重新提交")
    }

    private func clearAllLoginState(completion: @escaping () -> Void) {
        jdCookieString = nil
        isLoggedIn = false
        clearJDCache { completion() }
    }

    func createFormDataBody(boundary: String, parameters: [String: String]) -> Data {
        let body = NSMutableData()
        for (key, value) in parameters {
            body.appendString("--\(boundary)\r\n")
            body.appendString("Content-Disposition: form-data; name=\"\(key)\"\r\n\r\n")
            body.appendString("\(value)\r\n")
        }
        body.appendString("--\(boundary)--\r\n")
        return body as Data
    }

    func showAlert(message: String, completion: (() -> Void)? = nil) {
        let alert = UIAlertController(title: "提示", message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "确定", style: .default, handler: { _ in completion?() }))
        present(alert, animated: true)
    }

    private func clearJDCache(completion: @escaping () -> Void) {
        let dataStore = WKWebsiteDataStore.default()
        let websiteDataTypes = WKWebsiteDataStore.allWebsiteDataTypes()
        let date = Date(timeIntervalSince1970: 0)
        dataStore.removeData(ofTypes: websiteDataTypes, modifiedSince: date) {
            HTTPCookieStorage.shared.cookies?.forEach {
                if $0.domain.contains("jd.com") {
                    HTTPCookieStorage.shared.deleteCookie($0)
                }
            }
            DispatchQueue.main.async { completion() }
        }
    }
}

extension NSMutableData {
    func appendString(_ string: String) {
        if let data = string.data(using: .utf8) {
            append(data)
        }
    }
}
