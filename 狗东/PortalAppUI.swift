import UIKit
import UserNotifications

@available(iOS 13.0, *)
final class RootTabBarController: UITabBarController, UITabBarControllerDelegate {
    private let protectedIndexes: Set<Int> = [0, 1, 4]
    private var pendingProtectedIndex: Int?
    private var lastSafeIndex: Int = 3
    private var swipeLeft: UISwipeGestureRecognizer?
    private var swipeRight: UISwipeGestureRecognizer?

    override func viewDidLoad() {
        super.viewDidLoad()
        delegate = self
        setupAppearance()
        setupTabs()
        selectedIndex = 3
        lastSafeIndex = 3
        setupSwipeGestures()
        NotificationCenter.default.addObserver(self, selector: #selector(handleSessionChange), name: AppNotifications.sessionDidChange, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(handleLogout), name: AppNotifications.sessionDidLogout, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(handleRequiresLogin), name: AppNotifications.sessionRequiresLogin, object: nil)
        AppSessionStore.shared.bootstrap()
    }

    private func setupAppearance() {
        LiquidGlassEffect.applyGlassToTabBar(tabBar)
        tabBar.tintColor = .systemBlue
        tabBar.unselectedItemTintColor = .secondaryLabel
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
        swipeLeft.map { view.removeGestureRecognizer($0) }
        swipeRight.map { view.removeGestureRecognizer($0) }
    }

    private func setupTabs() {
        let home = AppNavigationController(rootViewController: HomeDashboardViewController())
        home.tabBarItem = UITabBarItem(title: "首页", image: UIImage(systemName: "house"), selectedImage: UIImage(systemName: "house.fill"))

        let projects = AppNavigationController(rootViewController: ProjectsRootViewController())
        projects.tabBarItem = UITabBarItem(title: "项目", image: UIImage(systemName: "shippingbox"), selectedImage: UIImage(systemName: "shippingbox.fill"))

        let tasks = AppNavigationController(rootViewController: CoinTasksViewController())
        tasks.tabBarItem = UITabBarItem(title: "任务", image: UIImage(systemName: "star"), selectedImage: UIImage(systemName: "star.fill"))

        let jd = AppNavigationController(rootViewController: MainUIViewController())
        jd.tabBarItem = UITabBarItem(title: "京东", image: UIImage(systemName: "bag"), selectedImage: UIImage(systemName: "bag.fill"))

        let more = AppNavigationController(rootViewController: MoreViewController())
        more.tabBarItem = UITabBarItem(title: "更多", image: UIImage(systemName: "line.3.horizontal"), selectedImage: UIImage(systemName: "line.3.horizontal"))

        viewControllers = [home, projects, tasks, jd, more]
    }

    private func setupSwipeGestures() {
        let left = UISwipeGestureRecognizer(target: self, action: #selector(handleSwipe(_:)))
        left.direction = .left
        let right = UISwipeGestureRecognizer(target: self, action: #selector(handleSwipe(_:)))
        right.direction = .right
        view.addGestureRecognizer(left)
        view.addGestureRecognizer(right)
        swipeLeft = left
        swipeRight = right
    }

    @objc private func handleSwipe(_ gesture: UISwipeGestureRecognizer) {
        guard let count = viewControllers?.count, count > 0 else { return }
        if gesture.direction == .left {
            let next = selectedIndex + 1
            if next < count {
                selectedIndex = next
            }
        } else if gesture.direction == .right {
            let prev = selectedIndex - 1
            if prev >= 0 {
                selectedIndex = prev
            }
        }
    }

    func tabBarController(_ tabBarController: UITabBarController, shouldSelect viewController: UIViewController) -> Bool {
        guard let index = viewControllers?.firstIndex(of: viewController) else { return true }
        guard protectedIndexes.contains(index) else {
            lastSafeIndex = index
            return true
        }
        if !AppSessionStore.shared.isAuthenticated {
            pendingProtectedIndex = index
            presentAuthFlow()
            return false
        }
        lastSafeIndex = index
        AppSessionStore.shared.refreshIfPossible(silent: true) { [weak self] success in
            if !success {
                // 不再盲目清 session：只有 refreshIfPossible 内部判断为真正未授权时才会清 session
                // 网络超时/服务器错误不会清除 session，只静默忽略
                if !AppSessionStore.shared.isAuthenticated {
                    self?.pendingProtectedIndex = index
                    AppSessionStore.shared.clearSession(requireLogin: true)
                }
            }
        }
        return true
    }

    private func presentAuthFlow() {
        if presentedViewController != nil { return }
        let login = PortalLoginViewController()
        login.onAuthSuccess = { [weak self] in
            guard let self = self else { return }
            let target = self.pendingProtectedIndex ?? 0
            self.pendingProtectedIndex = nil
            self.selectedIndex = target
            self.lastSafeIndex = target
        }
        let nav = AppNavigationController(rootViewController: login)
        nav.modalPresentationStyle = .formSheet
        present(nav, animated: true)
    }

    @objc private func handleSessionChange() {
        if let target = pendingProtectedIndex, AppSessionStore.shared.isAuthenticated {
            selectedIndex = target
            lastSafeIndex = target
            pendingProtectedIndex = nil
        }
    }

    @objc private func handleLogout() {
        if protectedIndexes.contains(selectedIndex) {
            selectedIndex = 3
            lastSafeIndex = 3
        }
    }

    @objc private func handleRequiresLogin() {
        if protectedIndexes.contains(selectedIndex) {
            pendingProtectedIndex = selectedIndex
            selectedIndex = lastSafeIndex
        }
        presentAuthFlow()
    }
}

@available(iOS 13.0, *)
final class AppNavigationController: UINavigationController {
    override func viewDidLoad() {
        super.viewDidLoad()
        navigationBar.prefersLargeTitles = true
        LiquidGlassEffect.applyGlassToNavBar(navigationBar)
        navigationBar.tintColor = .systemBlue
    }
}

@available(iOS 13.0, *)
final class PortalLoginViewController: BaseNativeViewController, UITextFieldDelegate {
    var onAuthSuccess: (() -> Void)?

    private let usernameField = UITextField()
    private let passwordField = UITextField()
    private let loginButton = UIButton(type: .system)
    private let registerButton = UIButton(type: .system)
    private let resetButton = UIButton(type: .system)
    private let helperLabel = UILabel()
    private let rememberSwitch = UISwitch()
    private let stack = UIStackView()

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemBackground
        title = "登录或注册"
        navigationItem.largeTitleDisplayMode = .never
        navigationItem.leftBarButtonItem = UIBarButtonItem(barButtonSystemItem: .close, target: self, action: #selector(closeTapped))
        setupUI()
    }

    private func setupUI() {
        usernameField.applyAppInputStyle(placeholder: "网页账号")
        usernameField.returnKeyType = .next
        usernameField.delegate = self
        usernameField.enablesReturnKeyAutomatically = true
        passwordField.applyAppInputStyle(placeholder: "登录密码")
        passwordField.isSecureTextEntry = true
        passwordField.returnKeyType = .go
        passwordField.delegate = self
        passwordField.enablesReturnKeyAutomatically = true
        passwordField.addTarget(self, action: #selector(loginTapped), for: .editingDidEndOnExit)

        rememberSwitch.isOn = CredentialStore.shared.load() != nil
        if let credentials = CredentialStore.shared.load() {
            usernameField.text = credentials.username
            passwordField.text = credentials.password
        }

        loginButton.setTitle("登录用户中心", for: .normal)
        loginButton.applyPrimaryStyle(color: .systemBlue)
        loginButton.addTarget(self, action: #selector(loginTapped), for: .touchUpInside)

        registerButton.setTitle("注册新账号", for: .normal)
        registerButton.applySecondaryStyle()
        registerButton.addTarget(self, action: #selector(registerTapped), for: .touchUpInside)

        resetButton.setTitle("忘记密码", for: .normal)
        resetButton.setTitleColor(.systemPurple, for: .normal)
        resetButton.titleLabel?.font = UIFont.systemFont(ofSize: 15, weight: .medium)
        resetButton.addTarget(self, action: #selector(resetTapped), for: .touchUpInside)

        helperLabel.text = "除京东页面外，其余功能都需要先登录或注册。"
        helperLabel.textColor = .secondaryLabel
        helperLabel.font = UIFont.systemFont(ofSize: 13)
        helperLabel.numberOfLines = 0

        let heroIcon = UIImageView()
        heroIcon.translatesAutoresizingMaskIntoConstraints = false
        heroIcon.image = UIImage(systemName: "person.circle.fill")
        heroIcon.tintColor = .systemBlue
        heroIcon.contentMode = .scaleAspectFit
        heroIcon.preferredSymbolConfiguration = UIImage.SymbolConfiguration(pointSize: 56)

        let heroTitle = UILabel()
        heroTitle.translatesAutoresizingMaskIntoConstraints = false
        heroTitle.text = "欢迎回来"
        heroTitle.font = UIFont.systemFont(ofSize: 28, weight: .bold)
        heroTitle.textColor = .label

        let heroSubtitle = UILabel()
        heroSubtitle.translatesAutoresizingMaskIntoConstraints = false
        heroSubtitle.text = "登录后即可管理项目、积分和微信协议"
        heroSubtitle.font = UIFont.systemFont(ofSize: 14)
        heroSubtitle.textColor = .secondaryLabel

        let heroStack = UIStackView(arrangedSubviews: [heroIcon, heroTitle, heroSubtitle])
        heroStack.axis = .vertical
        heroStack.alignment = .center
        heroStack.spacing = 10

        let rememberLabel = UILabel()
        rememberLabel.text = "记住账号密码"
        rememberLabel.font = UIFont.systemFont(ofSize: 15, weight: .medium)
        rememberLabel.textColor = .label
        let rememberRow = UIStackView(arrangedSubviews: [rememberLabel, rememberSwitch])
        rememberRow.axis = .horizontal
        rememberRow.alignment = .center
        rememberRow.distribution = .equalSpacing

        let stack = UIStackView(arrangedSubviews: [heroStack, usernameField, passwordField, rememberRow, loginButton, registerButton, resetButton, helperLabel])
        stack.axis = .vertical
        stack.spacing = 14
        stack.setCustomSpacing(28, after: heroStack)
        stack.setCustomSpacing(8, after: resetButton)
        stack.translatesAutoresizingMaskIntoConstraints = false

        [usernameField, passwordField, loginButton, registerButton].forEach {
            $0.heightAnchor.constraint(equalToConstant: 52).isActive = true
        }

        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissKeyboard))
        tap.cancelsTouchesInView = false
        view.addGestureRecognizer(tap)

        view.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 24),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -24),
            stack.centerYAnchor.constraint(equalTo: view.centerYAnchor, constant: -20)
        ])
    }

    @objc private func dismissKeyboard() {
        view.endEditing(true)
    }

    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        if textField === usernameField {
            passwordField.becomeFirstResponder()
        } else if textField === passwordField {
            loginTapped()
        }
        return true
    }

    @objc private func closeTapped() {
        dismiss(animated: true)
    }

    @objc private func loginTapped() {
        dismissKeyboard()
        let username = usernameField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let password = passwordField.text ?? ""
        if username.isEmpty || password.isEmpty {
            showMessage("请填写账号和密码")
            return
        }
        loginButton.isEnabled = false
        PortalAuthService.shared.login(username: username, password: password, remember: rememberSwitch.isOn) { result in
            self.loginButton.isEnabled = true
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success:
                AppSessionStore.shared.refreshIfPossible { success in
                    if success {
                        self.dismiss(animated: true) {
                            self.onAuthSuccess?()
                        }
                    } else {
                        self.showMessage("登录成功，但刷新用户信息失败，请稍后重试")
                    }
                }
            }
        }
    }

    @objc private func registerTapped() {
        let vc = PortalRegisterViewController()
        vc.onAuthFinished = { [weak self] in
            self?.dismiss(animated: true) {
                self?.onAuthSuccess?()
            }
        }
        navigationController?.pushViewController(vc, animated: true)
    }

    @objc private func resetTapped() {
        navigationController?.pushViewController(PortalResetPasswordViewController(), animated: true)
    }
}

@available(iOS 13.0, *)
final class PortalRegisterViewController: BaseNativeViewController {
    var onAuthFinished: (() -> Void)?

    private let usernameField = UITextField()
    private let passwordField = UITextField()
    private let bindCodeField = UITextField()
    private let submitButton = UIButton(type: .system)
    private lazy var heroView = SectionHeroView(icon: "person.badge.plus.fill", title: "注册账号", subtitle: "绑定用户后即可进入项目中心", tint: .systemPurple)

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemBackground
        navigationItem.largeTitleDisplayMode = .never
        setupUI()
    }

    private func setupUI() {
        usernameField.applyAppInputStyle(placeholder: "新账号")
        passwordField.applyAppInputStyle(placeholder: "登录密码")
        passwordField.isSecureTextEntry = true
        bindCodeField.applyAppInputStyle(placeholder: "绑定码 / UserID")

        submitButton.setTitle("注册并进入用户中心", for: .normal)
        submitButton.applyPrimaryStyle(color: .systemPurple)
        submitButton.addTarget(self, action: #selector(submitTapped), for: .touchUpInside)

        let tip = UILabel()
        tip.text = "请先给机器人发送“账号注册”获取绑定码，再填写本页信息完成注册。"
        tip.font = UIFont.systemFont(ofSize: 13)
        tip.textColor = .secondaryLabel
        tip.numberOfLines = 0

        let stack = UIStackView(arrangedSubviews: [heroView, usernameField, passwordField, bindCodeField, submitButton, tip])
        stack.axis = .vertical
        stack.spacing = 14
        stack.setCustomSpacing(24, after: heroView)
        stack.translatesAutoresizingMaskIntoConstraints = false
        [usernameField, passwordField, bindCodeField, submitButton].forEach { $0.heightAnchor.constraint(equalToConstant: 52).isActive = true }
        view.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 24),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -24),
            stack.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 40)
        ])
    }

    @objc private func submitTapped() {
        let username = usernameField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let password = passwordField.text ?? ""
        let bindCode = bindCodeField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if username.isEmpty || password.isEmpty || bindCode.isEmpty {
            showMessage("请完整填写注册信息")
            return
        }
        submitButton.isEnabled = false
        PortalAuthService.shared.register(username: username, password: password, bindCode: bindCode) { result in
            self.submitButton.isEnabled = true
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success:
                AppSessionStore.shared.refreshIfPossible { success in
                    if success {
                        self.onAuthFinished?()
                    } else {
                        self.showMessage("注册成功，但未能同步用户信息")
                    }
                }
            }
        }
    }
}

@available(iOS 13.0, *)
final class PortalResetPasswordViewController: BaseNativeViewController {
    private let codeField = UITextField()
    private let usernameField = UITextField()
    private let passwordField = UITextField()
    private let verifyButton = UIButton(type: .system)
    private let submitButton = UIButton(type: .system)
    private lazy var heroView = SectionHeroView(icon: "key.fill", title: "重置密码", subtitle: "使用验证码快速重置网页登录密码", tint: .systemRed)

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemBackground
        navigationItem.largeTitleDisplayMode = .never
        setupUI()
    }

    private func setupUI() {
        codeField.applyAppInputStyle(placeholder: "6 位重置验证码")
        usernameField.applyAppInputStyle(placeholder: "网页账号")
        passwordField.applyAppInputStyle(placeholder: "新密码")
        passwordField.isSecureTextEntry = true

        verifyButton.setTitle("验证验证码", for: .normal)
        verifyButton.applySecondaryStyle()
        verifyButton.addTarget(self, action: #selector(verifyTapped), for: .touchUpInside)

        submitButton.setTitle("重置密码", for: .normal)
        submitButton.applyPrimaryStyle(color: .systemRed)
        submitButton.addTarget(self, action: #selector(submitTapped), for: .touchUpInside)

        let hint = UILabel()
        hint.text = "请先给机器人发送“忘记密码”获取 6 位重置验证码，再来这里重置网页登录密码。"
        hint.font = UIFont.systemFont(ofSize: 13)
        hint.textColor = .secondaryLabel
        hint.numberOfLines = 0

        let stack = UIStackView(arrangedSubviews: [heroView, hint, codeField, verifyButton, usernameField, passwordField, submitButton])
        stack.axis = .vertical
        stack.spacing = 14
        stack.setCustomSpacing(24, after: heroView)
        stack.translatesAutoresizingMaskIntoConstraints = false
        [codeField, verifyButton, usernameField, passwordField, submitButton].forEach { $0.heightAnchor.constraint(equalToConstant: 52).isActive = true }
        view.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 24),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -24),
            stack.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 40)
        ])
    }

    @objc private func verifyTapped() {
        let code = codeField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if code.isEmpty {
            showMessage("请输入验证码")
            return
        }
        PortalAuthService.shared.fetchResetInfo(code: code) { result in
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let username):
                self.usernameField.text = username
                self.showMessage("验证码可用，已自动回填账号")
            }
        }
    }

    @objc private func submitTapped() {
        let code = codeField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let username = usernameField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let password = passwordField.text ?? ""
        if code.isEmpty || username.isEmpty || password.isEmpty {
            showMessage("请完整填写信息")
            return
        }
        PortalAuthService.shared.resetPassword(code: code, username: username, password: password) { result in
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let message):
                self.showMessage(message) {
                    self.navigationController?.popViewController(animated: true)
                }
            }
        }
    }
}

@available(iOS 13.0, *)
final class HomeDashboardViewController: BaseNativeViewController {
    private let scrollView = UIScrollView()
    private let refreshControl = UIRefreshControl()
    private let stack = UIStackView()
    private let summaryLabel = UILabel()
    private let userInfoLabel = UILabel()
    private var cards: [InfoCardView] = []
    private var notificationSection: UIView?
    private var notificationStack: UIStackView?
    private lazy var heroView = SectionHeroView(icon: "house.fill", title: "首页", subtitle: "通知中心 · 项目概览 · 账号状态", tint: .systemBlue)
    private var wxAutoRefreshTimer: Timer?

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
        NotificationCenter.default.addObserver(self, selector: #selector(reloadData), name: AppNotifications.sessionDidChange, object: nil)
        startWxAutoRefresh()
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
        wxAutoRefreshTimer?.invalidate()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reloadData()
    }

    private func startWxAutoRefresh() {
        wxAutoRefreshTimer?.invalidate()
        wxAutoRefreshTimer = Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { [weak self] _ in
            guard AppSessionStore.shared.isAuthenticated else { return }
            self?.loadNotifications()
        }
    }

    private func setupUI() {
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        scrollView.refreshControl = refreshControl
        refreshControl.addTarget(self, action: #selector(reloadData), for: .valueChanged)
        stack.axis = .vertical
        stack.spacing = 16
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scrollView)
        scrollView.addSubview(stack)

        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 20),
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -20),
            stack.widthAnchor.constraint(equalTo: view.widthAnchor, constant: -32)
        ])

        stack.addArrangedSubview(heroView)

        let notifCard = UIView()
        notifCard.applyCardStyle()
        let notifTitle = UILabel()
        notifTitle.text = "📢 通知中心"
        notifTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let notifRow = UIView()
        let notifLabel = UILabel()
        notifLabel.text = "正在加载通知..."
        notifLabel.textColor = .tertiaryLabel
        notifLabel.font = UIFont.systemFont(ofSize: 12)
        notifLabel.tag = 1001
        let viewAllLabel = UILabel()
        viewAllLabel.text = "查看全部 ›"
        viewAllLabel.textColor = .systemBlue
        viewAllLabel.font = UIFont.systemFont(ofSize: 12, weight: .bold)
        viewAllLabel.isUserInteractionEnabled = true
        viewAllLabel.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(openNotifications)))
        notifRow.addSubview(notifLabel)
        notifRow.addSubview(viewAllLabel)
        notifLabel.translatesAutoresizingMaskIntoConstraints = false
        viewAllLabel.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            notifLabel.leadingAnchor.constraint(equalTo: notifRow.leadingAnchor),
            notifLabel.centerYAnchor.constraint(equalTo: notifRow.centerYAnchor),
            viewAllLabel.trailingAnchor.constraint(equalTo: notifRow.trailingAnchor),
            viewAllLabel.centerYAnchor.constraint(equalTo: notifRow.centerYAnchor)
        ])
        notifRow.translatesAutoresizingMaskIntoConstraints = false
        notifRow.heightAnchor.constraint(equalToConstant: 24).isActive = true
        let notifInnerStack = UIStackView(arrangedSubviews: [notifTitle, notifRow])
        notifInnerStack.axis = .vertical
        notifInnerStack.spacing = 8
        notifInnerStack.translatesAutoresizingMaskIntoConstraints = false
        notificationStack = notifInnerStack
        notifCard.addSubview(notifInnerStack)
        NSLayoutConstraint.activate([
            notifInnerStack.topAnchor.constraint(equalTo: notifCard.topAnchor, constant: 18),
            notifInnerStack.leadingAnchor.constraint(equalTo: notifCard.leadingAnchor, constant: 18),
            notifInnerStack.trailingAnchor.constraint(equalTo: notifCard.trailingAnchor, constant: -18),
            notifInnerStack.bottomAnchor.constraint(equalTo: notifCard.bottomAnchor, constant: -18)
        ])
        notificationSection = notifCard
        stack.addArrangedSubview(notifCard)

        let headerCard = UIView()
        headerCard.applyCardStyle(cornerRadius: 22)
        let headerStack = UIStackView(arrangedSubviews: [summaryLabel, userInfoLabel])
        headerStack.axis = .vertical
        headerStack.spacing = 8
        headerStack.translatesAutoresizingMaskIntoConstraints = false
        headerCard.addSubview(headerStack)
        NSLayoutConstraint.activate([
            headerStack.topAnchor.constraint(equalTo: headerCard.topAnchor, constant: 18),
            headerStack.leadingAnchor.constraint(equalTo: headerCard.leadingAnchor, constant: 18),
            headerStack.trailingAnchor.constraint(equalTo: headerCard.trailingAnchor, constant: -18),
            headerStack.bottomAnchor.constraint(equalTo: headerCard.bottomAnchor, constant: -18)
        ])
        summaryLabel.font = UIFont.systemFont(ofSize: 18, weight: .bold)
        summaryLabel.numberOfLines = 0
        userInfoLabel.font = UIFont.systemFont(ofSize: 13)
        userInfoLabel.textColor = .secondaryLabel
        userInfoLabel.numberOfLines = 0
        stack.addArrangedSubview(headerCard)

        let grid = UIStackView()
        grid.axis = .vertical
        grid.spacing = 12
        grid.translatesAutoresizingMaskIntoConstraints = false
        let row1 = UIStackView()
        row1.axis = .horizontal
        row1.spacing = 12
        row1.distribution = .fillEqually
        let row2 = UIStackView()
        row2.axis = .horizontal
        row2.spacing = 12
        row2.distribution = .fillEqually
        cards = [
            InfoCardView(title: "快到期", value: "-", tint: .systemOrange),
            InfoCardView(title: "有效项目", value: "-", tint: .systemGreen),
            InfoCardView(title: "当前积分", value: "-", tint: .systemPurple),
            InfoCardView(title: "已上车项目", value: "-", tint: .systemBlue)
        ]
        row1.addArrangedSubview(cards[0])
        row1.addArrangedSubview(cards[1])
        row2.addArrangedSubview(cards[2])
        row2.addArrangedSubview(cards[3])
        grid.addArrangedSubview(row1)
        grid.addArrangedSubview(row2)
        stack.addArrangedSubview(grid)

        let actionCard = UIView()
        actionCard.applyCardStyle()
        let actionTitle = UILabel()
        actionTitle.text = "快捷入口"
        actionTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let projectsAction = ActionButton(icon: "shippingbox.fill", title: "项目中心", desc: "活动中心与项目管理", tint: .systemIndigo)
        projectsAction.tapAction = { [weak self] in self?.tabBarController?.selectedIndex = 1 }
        let tasksAction = ActionButton(icon: "star.fill", title: "积分任务", desc: "打卡、祈福与积分购买", tint: .systemOrange)
        tasksAction.tapAction = { [weak self] in self?.tabBarController?.selectedIndex = 2 }
        let jdAction = ActionButton(icon: "bag.fill", title: "京东页面", desc: "登录京东、查询与手机卡", tint: .systemBlue)
        jdAction.tapAction = { [weak self] in self?.tabBarController?.selectedIndex = 3 }
        let moreAction = ActionButton(icon: "line.3.horizontal", title: "更多", desc: "通知、反馈与系统信息", tint: .systemPurple)
        moreAction.tapAction = { [weak self] in self?.tabBarController?.selectedIndex = 4 }
        let actionGrid = UIStackView(arrangedSubviews: [
            UIStackView(arrangedSubviews: [projectsAction, tasksAction]),
            UIStackView(arrangedSubviews: [jdAction, moreAction])
        ])
        actionGrid.axis = .vertical
        actionGrid.spacing = 10
        actionGrid.translatesAutoresizingMaskIntoConstraints = false
        actionGrid.arrangedSubviews.forEach { row in
            (row as? UIStackView)?.axis = .horizontal
            (row as? UIStackView)?.spacing = 10
            (row as? UIStackView)?.distribution = .fillEqually
        }
        let actionStack = UIStackView(arrangedSubviews: [actionTitle, actionGrid])
        actionStack.axis = .vertical
        actionStack.spacing = 12
        actionStack.translatesAutoresizingMaskIntoConstraints = false
        actionCard.addSubview(actionStack)
        NSLayoutConstraint.activate([
            actionStack.topAnchor.constraint(equalTo: actionCard.topAnchor, constant: 18),
            actionStack.leadingAnchor.constraint(equalTo: actionCard.leadingAnchor, constant: 18),
            actionStack.trailingAnchor.constraint(equalTo: actionCard.trailingAnchor, constant: -18),
            actionStack.bottomAnchor.constraint(equalTo: actionCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(actionCard)
    }

    @objc private func openNotifications() {
        tabBarController?.selectedIndex = 4
    }

    @objc private func reloadData() {
        guard AppSessionStore.shared.isAuthenticated else {
            summaryLabel.text = "请先登录用户中心"
            userInfoLabel.text = "登录后可查看积分、项目、微信协议等信息。"
            cards.forEach { $0.updateValue("-") }
            return
        }
        refreshControl.beginRefreshing()
        PortalService.shared.fetchHomeSnapshot { result in
            self.refreshControl.endRefreshing()
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let snapshot):
                AppSessionStore.shared.update(snapshot: snapshot, notify: false)
                self.render(snapshot)
            }
        }
        loadNotifications()
    }

    private func loadNotifications() {
        guard AppSessionStore.shared.isAuthenticated else { return }
        PortalService.shared.fetchNotifications(includeContent: false) { [weak self] result in
            if case .success(let page) = result {
                self?.renderNotifications(page)
            }
        }
    }

    private func renderNotifications(_ page: PortalNotificationPage) {
        guard let notifStack = notificationStack else { return }
        let existingViews = notifStack.arrangedSubviews
        for view in existingViews where view.tag >= 2000 {
            notifStack.removeArrangedSubview(view)
            view.removeFromSuperview()
        }
        if let notifRow = existingViews.last, let label = notifRow.viewWithTag(1001) as? UILabel {
            if page.unread > 0 {
                label.text = "未读通知：\(page.unread)条"
                label.textColor = .systemBlue
                label.font = UIFont.systemFont(ofSize: 13, weight: .semibold)
            } else if page.list.isEmpty {
                label.text = "暂无新通知"
                label.textColor = .tertiaryLabel
            } else {
                label.text = "所有通知已读"
                label.textColor = .secondaryLabel
            }
        }
        let displayList = page.list.prefix(3)
        for (index, item) in displayList.enumerated() {
            let row = UIView()
            row.tag = 2000 + index
            let dotLabel = UILabel()
            dotLabel.text = item.isRead ? "📄" : "🔵"
            dotLabel.font = UIFont.systemFont(ofSize: 12)
            let titleLabel = UILabel()
            titleLabel.text = item.title ?? "无标题"
            titleLabel.font = UIFont.systemFont(ofSize: 13, weight: item.isTop == true ? .bold : .regular)
            titleLabel.textColor = item.isTop == true || !item.isRead ? .label : .secondaryLabel
            titleLabel.numberOfLines = 1
            let chevron = UILabel()
            chevron.text = "›"
            chevron.textColor = .systemBlue
            chevron.font = UIFont.systemFont(ofSize: 16, weight: .bold)
            row.addSubview(dotLabel)
            row.addSubview(titleLabel)
            row.addSubview(chevron)
            dotLabel.translatesAutoresizingMaskIntoConstraints = false
            titleLabel.translatesAutoresizingMaskIntoConstraints = false
            chevron.translatesAutoresizingMaskIntoConstraints = false
            NSLayoutConstraint.activate([
                dotLabel.leadingAnchor.constraint(equalTo: row.leadingAnchor),
                dotLabel.centerYAnchor.constraint(equalTo: row.centerYAnchor),
                titleLabel.leadingAnchor.constraint(equalTo: dotLabel.trailingAnchor, constant: 6),
                titleLabel.centerYAnchor.constraint(equalTo: row.centerYAnchor),
                titleLabel.trailingAnchor.constraint(equalTo: chevron.leadingAnchor, constant: -4),
                chevron.trailingAnchor.constraint(equalTo: row.trailingAnchor),
                chevron.centerYAnchor.constraint(equalTo: row.centerYAnchor),
                row.heightAnchor.constraint(equalToConstant: 24)
            ])
            row.isUserInteractionEnabled = true
            let tap = UITapGestureRecognizer(target: self, action: #selector(openNotifications))
            row.addGestureRecognizer(tap)
            notifStack.addArrangedSubview(row)
        }
    }

    private func render(_ snapshot: PortalHomeSnapshot) {
        let dashboard = snapshot.dashboard
        let profile = snapshot.profile
        let displayName = dashboard.nickname ?? dashboard.username ?? "用户"
        summaryLabel.text = "编号：\(dashboard.number)  ·  \(displayName)"
        let wxStatusText = snapshot.wechatStatus?.status ?? (profile.user?.Wxid?.isEmpty == false ? "已绑定" : "未绑定")
        userInfoLabel.text = "积分：\(dashboard.coin)  ·  登录：\(dashboard.lastLoginAt ?? "-")\n微信：\(wxStatusText)"
        cards[0].updateValue("\(dashboard.expiringCount)")
        cards[1].updateValue("\(dashboard.activeCount)")
        cards[2].updateValue("\(dashboard.coin)")
        cards[3].updateValue("\(dashboard.joinedCount)")
    }
}

@available(iOS 13.0, *)
final class ProjectsRootViewController: BaseNativeViewController {
    private let segmented = UISegmentedControl(items: ["活动中心", "我的项目", "微信协议"])
    private let container = UIView()
    private let activitiesVC = ActivitiesListViewController()
    private let myProjectsVC = MyProjectsListViewController()
    private let wechatVC = WechatProtocolViewController()
    private lazy var heroView = SectionHeroView(icon: "shippingbox.fill", title: "项目中心", subtitle: "活动中心、我的项目与微信协议", tint: .systemIndigo)
    private var currentVC: UIViewController?

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
        switchTo(index: 0)
    }

    private func setupUI() {
        segmented.selectedSegmentIndex = 0
        segmented.addTarget(self, action: #selector(segmentChanged), for: .valueChanged)
        segmented.translatesAutoresizingMaskIntoConstraints = false
        container.translatesAutoresizingMaskIntoConstraints = false
        heroView.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(heroView)
        view.addSubview(segmented)
        view.addSubview(container)
        NSLayoutConstraint.activate([
            heroView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 12),
            heroView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            heroView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            segmented.topAnchor.constraint(equalTo: heroView.bottomAnchor, constant: 12),
            segmented.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            segmented.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            container.topAnchor.constraint(equalTo: segmented.bottomAnchor, constant: 12),
            container.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            container.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            container.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
    }

    @objc private func segmentChanged() {
        switchTo(index: segmented.selectedSegmentIndex)
    }

    private func switchTo(index: Int) {
        currentVC?.willMove(toParent: nil)
        currentVC?.view.removeFromSuperview()
        currentVC?.removeFromParent()
        let vc: UIViewController
        switch index {
        case 0: vc = activitiesVC
        case 1: vc = myProjectsVC
        case 2: vc = wechatVC
        default: vc = activitiesVC
        }
        addChild(vc)
        vc.view.translatesAutoresizingMaskIntoConstraints = false
        container.addSubview(vc.view)
        NSLayoutConstraint.activate([
            vc.view.topAnchor.constraint(equalTo: container.topAnchor),
            vc.view.leadingAnchor.constraint(equalTo: container.leadingAnchor),
            vc.view.trailingAnchor.constraint(equalTo: container.trailingAnchor),
            vc.view.bottomAnchor.constraint(equalTo: container.bottomAnchor)
        ])
        vc.didMove(toParent: self)
        currentVC = vc
    }
}

@available(iOS 13.0, *)
final class ActivitiesListViewController: UITableViewController {
    private var activities: [PortalActivity] = []
    private var filteredActivities: [PortalActivity] = []
    private let searchController = UISearchController(searchResultsController: nil)

    override func viewDidLoad() {
        super.viewDidLoad()
        tableView = UITableView(frame: .zero, style: .insetGrouped)
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "activity")
        tableView.rowHeight = UITableView.automaticDimension
        tableView.estimatedRowHeight = 132
        searchController.searchResultsUpdater = self
        searchController.obscuresBackgroundDuringPresentation = false
        searchController.searchBar.placeholder = "搜索活动名称、青龙配置…"
        navigationItem.searchController = searchController
        definesPresentationContext = true
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        loadData()
    }

    private func loadData() {
        PortalService.shared.fetchActivities { result in
            switch result {
            case .failure(let error):
                (self.parent as? BaseNativeViewController)?.handle(error)
            case .success(let activities):
                self.activities = activities
                self.tableView.reloadData()
            }
        }
    }

    private var isSearching: Bool {
        return searchController.isActive && !(searchController.searchBar.text ?? "").isEmpty
    }

    private var displayedActivities: [PortalActivity] {
        return isSearching ? filteredActivities : activities
    }

    override func numberOfSections(in tableView: UITableView) -> Int { 1 }
    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { displayedActivities.count }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let item = displayedActivities[indexPath.row]
        let cell = UITableViewCell(style: .default, reuseIdentifier: nil)
        cell.selectionStyle = .none
        let card = ProjectSummaryCardView()
        card.translatesAutoresizingMaskIntoConstraints = false
        card.titleLabel.text = item.name
        let guideText = item.guide?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if guideText.isEmpty {
            card.metaLabel.text = "青龙：\(item.qingLongConfig ?? "默认容器")"
        } else {
            let preview = guideText.count > 52 ? String(guideText.prefix(52)) + "..." : guideText
            card.metaLabel.text = "青龙：\(item.qingLongConfig ?? "默认容器")\n玩法摘要：\(preview)"
        }
        let price = item.isMonthlyDeduct == true ? "每月 \(item.monthlyCoin ?? 0) 积分" : "一次 \(item.needCoin ?? 0) 积分"
        card.priceLabel.text = price
        card.badgeLabel.configure(text: item.isMonthlyDeduct == true ? "按月授权" : "一次上车", kind: item.isMonthlyDeduct == true ? .expiring : .active)
        cell.contentView.addSubview(card)
        NSLayoutConstraint.activate([
            card.topAnchor.constraint(equalTo: cell.contentView.topAnchor, constant: 8),
            card.leadingAnchor.constraint(equalTo: cell.contentView.leadingAnchor, constant: 16),
            card.trailingAnchor.constraint(equalTo: cell.contentView.trailingAnchor, constant: -16),
            card.bottomAnchor.constraint(equalTo: cell.contentView.bottomAnchor, constant: -8)
        ])
        return cell
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let vc = ProjectFormViewController(activity: displayedActivities[indexPath.row])
        navigationController?.pushViewController(vc, animated: true)
    }
}

@available(iOS 13.0, *)
extension ActivitiesListViewController: UISearchResultsUpdating {
    func updateSearchResults(for searchController: UISearchController) {
        let query = (searchController.searchBar.text ?? "").lowercased().trimmingCharacters(in: .whitespaces)
        if query.isEmpty {
            filteredActivities = []
        } else {
            filteredActivities = activities.filter {
                ($0.name ?? "").lowercased().contains(query) ||
                ($0.envKey ?? "").lowercased().contains(query) ||
                ($0.qingLongConfig ?? "").lowercased().contains(query)
            }
        }
        tableView.reloadData()
    }
}

@available(iOS 13.0, *)
final class ProjectFormViewController: BaseNativeViewController {
    private let activity: PortalActivity
    private let scrollView = UIScrollView()
    private let stack = UIStackView()
    private var fieldViews: [String: UITextField] = [:]

    init(activity: PortalActivity) {
        self.activity = activity
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemGroupedBackground
        title = activity.name
        navigationItem.largeTitleDisplayMode = .never
        setupUI()
    }

    private func setupUI() {
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        stack.axis = .vertical
        stack.spacing = 14
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 20),
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -20),
            stack.widthAnchor.constraint(equalTo: view.widthAnchor, constant: -32)
        ])

        let meta = UILabel()
        meta.text = (activity.isMonthlyDeduct == true ? "每月扣 \(activity.monthlyCoin ?? 0) 积分" : "一次扣 \(activity.needCoin ?? 0) 积分") + " · 青龙：\(activity.qingLongConfig ?? "默认容器")"
        meta.font = UIFont.systemFont(ofSize: 13)
        meta.textColor = .secondaryLabel
        meta.numberOfLines = 0
        stack.addArrangedSubview(meta)

        if let guide = activity.guide?.trimmingCharacters(in: .whitespacesAndNewlines), !guide.isEmpty {
            let guideView = PromptTipView(text: guide)
            stack.addArrangedSubview(guideView)
        }

        (activity.inputFields ?? []).forEach { field in
            let label = UILabel()
            label.text = field.prompt
            label.font = UIFont.systemFont(ofSize: 14, weight: .medium)
            label.numberOfLines = 0
            let input = UITextField()
            input.applyAppInputStyle(placeholder: field.prompt)
            input.heightAnchor.constraint(equalToConstant: 50).isActive = true
            fieldViews[field.key] = input
            stack.addArrangedSubview(label)
            stack.addArrangedSubview(input)
        }

        let remarkLabel = UILabel()
        remarkLabel.text = "用户备注名"
        remarkLabel.font = UIFont.systemFont(ofSize: 14, weight: .medium)
        let remarkInput = UITextField()
        remarkInput.applyAppInputStyle(placeholder: "请输入唯一备注名")
        remarkInput.heightAnchor.constraint(equalToConstant: 50).isActive = true
        fieldViews["__remarks"] = remarkInput
        stack.addArrangedSubview(remarkLabel)
        stack.addArrangedSubview(remarkInput)

        if activity.isMonthlyDeduct == true {
            let monthLabel = UILabel()
            monthLabel.text = "授权月数"
            monthLabel.font = UIFont.systemFont(ofSize: 14, weight: .medium)
            let monthInput = UITextField()
            monthInput.applyAppInputStyle(placeholder: "请输入 1-12")
            monthInput.keyboardType = .numberPad
            monthInput.heightAnchor.constraint(equalToConstant: 50).isActive = true
            fieldViews["__months"] = monthInput
            stack.addArrangedSubview(monthLabel)
            stack.addArrangedSubview(monthInput)
        }

        let submit = UIButton(type: .system)
        submit.setTitle("确认上车", for: .normal)
        submit.applyPrimaryStyle(color: .systemBlue)
        submit.heightAnchor.constraint(equalToConstant: 52).isActive = true
        submit.addTarget(self, action: #selector(submitTapped), for: .touchUpInside)
        stack.addArrangedSubview(submit)
    }

    @objc private func submitTapped() {
        var inputs: [String: String] = [:]
        for (key, field) in fieldViews where !key.hasPrefix("__") {
            inputs[key] = field.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        }
        let remarks = fieldViews["__remarks"]?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let months = Int(fieldViews["__months"]?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "") ?? 0
        if remarks.isEmpty {
            showMessage("请输入备注名")
            return
        }
        if activity.isMonthlyDeduct == true && months <= 0 {
            showMessage("请输入正确的授权月数")
            return
        }
        let totalCoin = activity.isMonthlyDeduct == true ? (activity.monthlyCoin ?? 0) * months : (activity.needCoin ?? 0)
        let expireText: String? = {
            guard activity.isMonthlyDeduct == true, months > 0 else { return nil }
            let calendar = Calendar.current
            if let date = calendar.date(byAdding: .month, value: months, to: Date()) {
                let formatter = DateFormatter()
                formatter.dateFormat = "yyyy-MM-dd"
                return formatter.string(from: date)
            }
            return nil
        }()
        var message = "项目：\(activity.name)\n备注名：\(remarks)\n将扣积分：\(totalCoin)"
        if activity.isMonthlyDeduct == true {
            message += "\n授权月数：\(months)"
            if let expireText, !expireText.isEmpty {
                message += "\n预计有效期至：\(expireText)"
            }
        }
        message += "\n确认后才会正式上车并扣除积分。"
        let alert = UIAlertController(title: "确认上车", message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "确认上车", style: .default) { _ in
            PortalService.shared.createProject(activityId: self.activity.id, inputs: inputs, remarks: remarks, months: months) { result in
                switch result {
                case .failure(let error):
                    self.handle(error)
                case .success(let message):
                    self.showMessage(message) {
                        self.navigationController?.popViewController(animated: true)
                    }
                }
            }
        })
        present(alert, animated: true)
    }
}

@available(iOS 13.0, *)
final class MyProjectsListViewController: UITableViewController {
    struct Group {
        let key: String
        let activityId: String
        let activityName: String
        let qingLongConfig: String?
        let priceText: String?
        var items: [PortalProject]
        var activeCount: Int
        var expiringCount: Int
        var expiredCount: Int
    }

    private var allGroups: [Group] = []
    private var groups: [Group] = []
    private var expandedGroupKeys: Set<String> = []
    private let searchController = UISearchController(searchResultsController: nil)

    override func viewDidLoad() {
        super.viewDidLoad()
        tableView = UITableView(frame: .zero, style: .insetGrouped)
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "project")
        tableView.rowHeight = UITableView.automaticDimension
        tableView.estimatedRowHeight = 132
        tableView.separatorStyle = .none
        searchController.searchResultsUpdater = self
        searchController.obscuresBackgroundDuringPresentation = false
        searchController.searchBar.placeholder = "搜索项目名、备注名、青龙配置…"
        navigationItem.searchController = searchController
        definesPresentationContext = true
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        loadData()
    }

    private func loadData() {
        PortalService.shared.fetchProjects { result in
            switch result {
            case .failure(let error):
                (self.parent as? BaseNativeViewController)?.handle(error)
            case .success(let projects):
                let grouped = Dictionary(grouping: projects) { item in
                    [
                        item.activityId,
                        item.qingLongConfig ?? "默认容器"
                    ].joined(separator: "__")
                }
                self.groups = grouped.values.map { items in
                    let first = items.first!
                    return Group(
                        key: [
                            first.activityId,
                            first.activityName ?? "未命名项目",
                            first.qingLongConfig ?? "默认容器",
                            first.priceText ?? "-"
                        ].joined(separator: "__"),
                        activityId: first.activityId,
                        activityName: first.activityName ?? "未命名项目",
                        qingLongConfig: first.qingLongConfig,
                        priceText: first.priceText,
                        items: items.sorted { ($0.displayName ?? $0.remark ?? "") < ($1.displayName ?? $1.remark ?? "") },
                        activeCount: items.filter { $0.bizStatus == "active" }.count,
                        expiringCount: items.filter { $0.bizStatus == "expiring" }.count,
                        expiredCount: items.filter { $0.bizStatus == "expired" }.count
                    )
                }.sorted { $0.activityName < $1.activityName }
                self.allGroups = self.groups
                self.tableView.backgroundView = self.groups.isEmpty ? EmptyStateView(icon: "tray", title: "暂无项目", desc: "还没有任何可管理的项目，可以先去上车项目页面添加。") : nil
                self.tableView.reloadData()
            }
        }
    }

    override func numberOfSections(in tableView: UITableView) -> Int { groups.count }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        let group = groups[section]
        return expandedGroupKeys.contains(group.key) ? group.items.count + 1 : 1
    }

    override func tableView(_ tableView: UITableView, heightForHeaderInSection section: Int) -> CGFloat { 8 }
    override func tableView(_ tableView: UITableView, viewForHeaderInSection section: Int) -> UIView? { UIView() }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        if indexPath.row == 0 {
            return makeGroupCell(section: indexPath.section)
        }
        return makeItemCell(section: indexPath.section, itemIndex: indexPath.row - 1)
    }

    override func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        if indexPath.row == 0 {
            toggleGroup(section: indexPath.section)
            return
        }
        presentAction(for: groups[indexPath.section].items[indexPath.row - 1])
    }

    private func makeGroupCell(section: Int) -> UITableViewCell {
        let group = groups[section]
        let cell = UITableViewCell(style: .default, reuseIdentifier: nil)
        cell.selectionStyle = .none
        cell.backgroundColor = .clear
        let summary = ProjectGroupSummaryView()
        summary.translatesAutoresizingMaskIntoConstraints = false
        summary.configure(group: group, isExpanded: expandedGroupKeys.contains(group.key))
        summary.queryButton.tag = section
        summary.expandButton.tag = section
        summary.queryButton.addTarget(self, action: #selector(queryGroupButtonTapped(_:)), for: .touchUpInside)
        summary.expandButton.addTarget(self, action: #selector(toggleGroupButtonTapped(_:)), for: .touchUpInside)
        cell.contentView.addSubview(summary)
        NSLayoutConstraint.activate([
            summary.topAnchor.constraint(equalTo: cell.contentView.topAnchor, constant: 8),
            summary.leadingAnchor.constraint(equalTo: cell.contentView.leadingAnchor, constant: 16),
            summary.trailingAnchor.constraint(equalTo: cell.contentView.trailingAnchor, constant: -16),
            summary.bottomAnchor.constraint(equalTo: cell.contentView.bottomAnchor, constant: -8)
        ])
        return cell
    }

    private func makeItemCell(section: Int, itemIndex: Int) -> UITableViewCell {
        let item = groups[section].items[itemIndex]
        let cell = UITableViewCell(style: .default, reuseIdentifier: nil)
        cell.selectionStyle = .none
        cell.backgroundColor = .clear
        let card = ProjectSummaryCardView()
        card.translatesAutoresizingMaskIntoConstraints = false
        card.titleLabel.text = item.displayName?.isEmpty == false ? item.displayName : (item.remark?.isEmpty == false ? item.remark : item.activityName)
        let line1 = item.expireDate ?? "长期 / 未记录"
        let line2 = item.qingLongConfig ?? "默认容器"
        card.metaLabel.text = "到期：\(line1)\n青龙：\(line2)\n点击卡片可进行续费、查询、删除"
        card.priceLabel.text = item.priceText ?? "-"
        let status = item.bizStatus ?? "gray"
        if status == "active" {
            card.badgeLabel.configure(text: item.bizStatusText ?? "有效中", kind: .active)
        } else if status == "expiring" {
            card.badgeLabel.configure(text: item.bizStatusText ?? "快到期", kind: .expiring)
        } else if status == "expired" {
            card.badgeLabel.configure(text: item.bizStatusText ?? "已失效", kind: .expired)
        } else {
            card.badgeLabel.configure(text: item.statusText ?? "未知", kind: .gray)
        }
        cell.contentView.addSubview(card)
        NSLayoutConstraint.activate([
            card.topAnchor.constraint(equalTo: cell.contentView.topAnchor, constant: 6),
            card.leadingAnchor.constraint(equalTo: cell.contentView.leadingAnchor, constant: 28),
            card.trailingAnchor.constraint(equalTo: cell.contentView.trailingAnchor, constant: -16),
            card.bottomAnchor.constraint(equalTo: cell.contentView.bottomAnchor, constant: -6)
        ])
        return cell
    }

    @objc private func queryGroupButtonTapped(_ sender: UIButton) {
        queryGroupIncome(for: groups[sender.tag])
    }

    @objc private func toggleGroupButtonTapped(_ sender: UIButton) {
        toggleGroup(section: sender.tag)
    }

    private func toggleGroup(section: Int) {
        let key = groups[section].key
        if expandedGroupKeys.contains(key) {
            expandedGroupKeys.remove(key)
        } else {
            expandedGroupKeys.insert(key)
        }
        tableView.reloadSections(IndexSet(integer: section), with: .automatic)
    }

    private func queryGroupIncome(for group: Group) {
        guard !group.items.isEmpty else {
            (self.parent as? BaseNativeViewController)?.showMessage("当前项目组暂无账号")
            return
        }
        let loading = UIAlertController(title: "批量查询中", message: "正在依次查询 \(group.items.count) 个账号...", preferredStyle: .alert)
        present(loading, animated: true)
        var outputs: [String] = []

        func run(index: Int) {
            if index >= group.items.count {
                loading.dismiss(animated: true) {
                    let title = "批量查询：\(group.activityName)"
                    let content = outputs.isEmpty ? "暂无查询结果" : outputs.joined(separator: "\n\n")
                    let vc = ProjectIncomeViewController(titleText: title, content: content)
                    self.navigationController?.pushViewController(vc, animated: true)
                }
                return
            }
            let item = group.items[index]
            let name = item.displayName?.isEmpty == false ? item.displayName! : (item.remark?.isEmpty == false ? item.remark! : (item.activityName ?? "未命名账号"))
            PortalService.shared.queryIncome(activityId: item.activityId, remarks: item.remark ?? "") { result in
                switch result {
                case .failure(let error):
                    outputs.append("【\(name)】\n查询失败：\(error.message)")
                case .success(let text):
                    outputs.append("【\(name)】\n\(text)")
                }
                run(index: index + 1)
            }
        }

        run(index: 0)
    }

    private func presentAction(for item: PortalProject) {
        let sheet = UIAlertController(title: item.displayName ?? item.activityName ?? "项目", message: item.priceText, preferredStyle: .actionSheet)
        sheet.addAction(UIAlertAction(title: "查询收入", style: .default) { _ in
            self.queryIncome(for: item)
        })
        sheet.addAction(UIAlertAction(title: "修改", style: .default) { _ in
            self.updateProject(item)
        })
        sheet.addAction(UIAlertAction(title: "续费", style: .default) { _ in
            self.renewProject(item)
        })
        sheet.addAction(UIAlertAction(title: "删除", style: .destructive) { _ in
            self.deleteProject(item)
        })
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    private func queryIncome(for item: PortalProject) {
        PortalService.shared.queryIncome(activityId: item.activityId, remarks: item.remark ?? "") { result in
            switch result {
            case .failure(let error):
                (self.parent as? BaseNativeViewController)?.handle(error)
            case .success(let text):
                let vc = ProjectIncomeViewController(titleText: item.displayName ?? item.activityName ?? "查询结果", content: text)
                self.navigationController?.pushViewController(vc, animated: true)
            }
        }
    }

    private func updateProject(_ item: PortalProject) {
        let vc = ProjectEditCKViewController(project: item) { [weak self] message in
            (self?.parent as? BaseNativeViewController)?.showMessage(message)
            self?.loadData()
        }
        navigationController?.pushViewController(vc, animated: true)
    }

    private func renewProject(_ item: PortalProject) {
        let alert = UIAlertController(title: "续费", message: "请输入续费月数", preferredStyle: .alert)
        alert.addTextField { field in
            field.keyboardType = .numberPad
            field.text = "1"
        }
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "确认", style: .default) { _ in
            let months = Int(alert.textFields?.first?.text ?? "1") ?? 1
            PortalService.shared.renewProject(activityId: item.activityId, remarks: item.remark ?? "", months: months) { result in
                switch result {
                case .failure(let error):
                    (self.parent as? BaseNativeViewController)?.handle(error)
                case .success(let message):
                    (self.parent as? BaseNativeViewController)?.showMessage(message)
                    self.loadData()
                }
            }
        })
        present(alert, animated: true)
    }

    private func deleteProject(_ item: PortalProject) {
        let refundTip: String = {
            if item.isMonthlyDeduct == true, let needCoin = item.needCoin, needCoin > 0 {
                return "该账号由一次性活动转换，删除不退还积分"
            }
            if item.isMonthlyDeduct == true, let monthlyCoin = item.monthlyCoin, monthlyCoin > 0, let daysLeft = item.daysLeft, daysLeft > 0 {
                let estimated = Int((Double(monthlyCoin) * Double(daysLeft) / 30.0) + 0.5)
                return "预计返还积分：\(estimated)（最终以服务端结算为准）"
            }
            if item.isMonthlyDeduct == true {
                return "预计返还积分：以服务端结算为准"
            }
            return "此活动为一次性扣费，删除不退还积分"
        }()
        let alert = UIAlertController(
            title: "确认删除",
            message: "项目：\(item.displayName ?? item.activityName ?? "项目")\n到期：\(item.expireDate ?? "长期 / 未记录")\n\(refundTip)",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "删除", style: .destructive) { _ in
            PortalService.shared.deleteProject(activityId: item.activityId, remarks: item.remark ?? "") { result in
                switch result {
                case .failure(let error):
                    (self.parent as? BaseNativeViewController)?.handle(error)
                case .success(let message):
                    (self.parent as? BaseNativeViewController)?.showMessage(message)
                    self.loadData()
                }
            }
        })
        present(alert, animated: true)
    }
}

@available(iOS 13.0, *)
extension MyProjectsListViewController: UISearchResultsUpdating {
    func updateSearchResults(for searchController: UISearchController) {
        let query = (searchController.searchBar.text ?? "").lowercased().trimmingCharacters(in: .whitespaces)
        if query.isEmpty {
            groups = allGroups
        } else {
            groups = allGroups.compactMap { group in
                let groupMatch = group.activityName.lowercased().contains(query) ||
                    (group.qingLongConfig ?? "").lowercased().contains(query)
                let filteredItems = group.items.filter {
                    ($0.activityName ?? "").lowercased().contains(query) ||
                    ($0.remark ?? "").lowercased().contains(query) ||
                    ($0.displayName ?? "").lowercased().contains(query) ||
                    ($0.envKey ?? "").lowercased().contains(query)
                }
                if groupMatch {
                    return group
                } else if !filteredItems.isEmpty {
                    var g = group
                    g = Group(key: group.key, activityId: group.activityId, activityName: group.activityName,
                              qingLongConfig: group.qingLongConfig, priceText: group.priceText,
                              items: filteredItems, activeCount: filteredItems.filter { $0.bizStatus == "active" }.count,
                              expiringCount: filteredItems.filter { $0.bizStatus == "expiring" }.count,
                              expiredCount: filteredItems.filter { $0.bizStatus == "expired" }.count)
                    return g
                }
                return nil
            }
        }
        tableView.backgroundView = groups.isEmpty ? EmptyStateView(icon: "magnifyingglass", title: "无搜索结果", desc: "没有找到匹配的项目") : nil
        tableView.reloadData()
    }
}

@available(iOS 13.0, *)
final class ProjectGroupSummaryView: UIView {
    let queryButton = UIButton(type: .system)
    let expandButton = UIButton(type: .system)

    private let titleLabel = UILabel()
    private let metaLabel = UILabel()
    private let statsLabel = UILabel()

    override init(frame: CGRect) {
        super.init(frame: frame)
        applyCardStyle(cornerRadius: 18)

        titleLabel.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        titleLabel.textColor = .label
        titleLabel.numberOfLines = 2

        metaLabel.font = UIFont.systemFont(ofSize: 13)
        metaLabel.textColor = .secondaryLabel
        metaLabel.numberOfLines = 0

        statsLabel.font = UIFont.systemFont(ofSize: 12, weight: .semibold)
        statsLabel.textColor = .systemBlue
        statsLabel.numberOfLines = 0

        queryButton.setTitle("查询", for: .normal)
        queryButton.applySecondaryStyle()
        queryButton.titleLabel?.font = UIFont.systemFont(ofSize: 14, weight: .semibold)

        expandButton.applyPrimaryStyle(color: .systemBlue)
        expandButton.titleLabel?.font = UIFont.systemFont(ofSize: 14, weight: .semibold)

        let buttons = UIStackView(arrangedSubviews: [queryButton, expandButton])
        buttons.axis = .horizontal
        buttons.spacing = 10
        buttons.distribution = .fillEqually

        let stack = UIStackView(arrangedSubviews: [titleLabel, metaLabel, statsLabel, buttons])
        stack.axis = .vertical
        stack.spacing = 10
        stack.translatesAutoresizingMaskIntoConstraints = false

        addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: bottomAnchor, constant: -16),
            queryButton.heightAnchor.constraint(equalToConstant: 40),
            expandButton.heightAnchor.constraint(equalToConstant: 40)
        ])
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    func configure(group: MyProjectsListViewController.Group, isExpanded: Bool) {
        titleLabel.text = group.activityName
        metaLabel.text = "账号数：\(group.items.count) · 费用：\(group.priceText ?? "-")\n青龙：\(group.qingLongConfig ?? "默认容器")"
        statsLabel.text = "有效 \(group.activeCount) · 快到期 \(group.expiringCount) · 过期 \(group.expiredCount)"
        expandButton.setTitle(isExpanded ? "收起账号" : "展开账号", for: .normal)
    }
}

@available(iOS 13.0, *)
final class ProjectIncomeViewController: BaseNativeViewController {
    private let content: String
    private let titleText: String

    init(titleText: String, content: String) {
        self.titleText = titleText
        self.content = content
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = titleText
        view.backgroundColor = .systemBackground
        let textView = UITextView()
        textView.translatesAutoresizingMaskIntoConstraints = false
        textView.isEditable = false
        textView.font = UIFont.systemFont(ofSize: 14)
        textView.text = content
        textView.backgroundColor = .secondarySystemBackground
        textView.layer.cornerRadius = 18
        view.addSubview(textView)
        NSLayoutConstraint.activate([
            textView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 16),
            textView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            textView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            textView.bottomAnchor.constraint(equalTo: view.bottomAnchor, constant: -16)
        ])
    }
}

@available(iOS 13.0, *)
final class ProjectEditCKViewController: BaseNativeViewController {
    private let project: PortalProject
    private let onSaved: (String) -> Void
    private let textView = UITextView()

    init(project: PortalProject, onSaved: @escaping (String) -> Void) {
        self.project = project
        self.onSaved = onSaved
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "修改 CK"
        view.backgroundColor = .systemGroupedBackground
        navigationItem.largeTitleDisplayMode = .never
        navigationItem.rightBarButtonItem = UIBarButtonItem(title: "保存", style: .done, target: self, action: #selector(saveTapped))

        let remarkLabel = UILabel()
        remarkLabel.translatesAutoresizingMaskIntoConstraints = false
        remarkLabel.text = "当前备注：\(project.remark ?? project.displayName ?? project.activityName ?? "项目")"
        remarkLabel.font = UIFont.systemFont(ofSize: 14, weight: .medium)
        remarkLabel.textColor = .secondaryLabel
        remarkLabel.numberOfLines = 0

        textView.translatesAutoresizingMaskIntoConstraints = false
        textView.font = UIFont.monospacedSystemFont(ofSize: 15, weight: .regular)
        textView.textColor = .label
        textView.backgroundColor = .secondarySystemBackground
        textView.layer.cornerRadius = 18
        textView.layer.borderWidth = 1
        textView.layer.borderColor = UIColor.systemGray5.cgColor
        textView.textContainerInset = UIEdgeInsets(top: 16, left: 14, bottom: 16, right: 14)
        textView.text = project.envValue ?? ""
        textView.autocorrectionType = .no
        textView.autocapitalizationType = .none

        view.addSubview(remarkLabel)
        view.addSubview(textView)
        NSLayoutConstraint.activate([
            remarkLabel.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 16),
            remarkLabel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            remarkLabel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            textView.topAnchor.constraint(equalTo: remarkLabel.bottomAnchor, constant: 12),
            textView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            textView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            textView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor, constant: -16)
        ])
    }

    @objc private func saveTapped() {
        let ckValue = textView.text.trimmingCharacters(in: .whitespacesAndNewlines)
        if ckValue.isEmpty {
            showMessage("请输入新的 CK 值")
            return
        }
        navigationItem.rightBarButtonItem?.isEnabled = false
        PortalService.shared.updateProject(activityId: project.activityId, remarks: project.remark ?? "", ckValue: ckValue) { result in
            self.navigationItem.rightBarButtonItem?.isEnabled = true
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let message):
                self.onSaved(message)
                self.navigationController?.popViewController(animated: true)
            }
        }
    }
}

@available(iOS 13.0, *)
final class WechatProtocolViewController: BaseNativeViewController {
    private let statusLabel = UILabel()
    private let detailLabel = UILabel()
    private lazy var heroView = SectionHeroView(icon: "message.fill", title: "微信协议", subtitle: "扫码登录、在线状态与设备管理", tint: .systemGreen)
    private var pollingTimer: Timer?
    private var autoRefreshTimer: Timer?
    private var qrModalNavigationController: UINavigationController?
    private var currentUUID: String?
    private var currentDeductCoin = false
    private var pollFailureCount = 0
    private var lastKnownStatus: String?

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
        startAutoRefresh()
    }

    deinit {
        pollingTimer?.invalidate()
        autoRefreshTimer?.invalidate()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        loadStatus()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        autoRefreshTimer?.invalidate()
    }

    private func startAutoRefresh() {
        autoRefreshTimer?.invalidate()
        autoRefreshTimer = Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { [weak self] _ in
            guard AppSessionStore.shared.isAuthenticated else { return }
            self?.loadStatus(silent: true)
        }
    }

    private func setupUI() {
        let card = UIView()
        card.applyCardStyle(cornerRadius: 22)
        card.translatesAutoresizingMaskIntoConstraints = false
        statusLabel.font = UIFont.systemFont(ofSize: 22, weight: .bold)
        detailLabel.font = UIFont.systemFont(ofSize: 14)
        detailLabel.textColor = .secondaryLabel
        detailLabel.numberOfLines = 0
        let statusStack = UIStackView(arrangedSubviews: [statusLabel, detailLabel])
        statusStack.axis = .vertical
        statusStack.spacing = 10
        statusStack.translatesAutoresizingMaskIntoConstraints = false
        card.addSubview(statusStack)

        let gridLabel = UILabel()
        gridLabel.text = "快捷操作"
        gridLabel.font = UIFont.systemFont(ofSize: 17, weight: .bold)

        let scanAction = ActionButton(icon: "qrcode.viewfinder", title: "扫码登录", desc: "微信扫码授权登录", tint: .systemBlue)
        scanAction.tapAction = { [weak self] in self?.scanLogin() }
        let reloginAction = ActionButton(icon: "arrow.clockwise", title: "重新登录", desc: "重新获取登录凭证", tint: .systemPurple)
        reloginAction.tapAction = { [weak self] in self?.relogin() }
        let wakeAction = ActionButton(icon: "bell.fill", title: "唤醒登录", desc: "唤醒已登录设备", tint: .systemOrange)
        wakeAction.tapAction = { [weak self] in self?.wakeLogin() }
        let logoutActionBtn = ActionButton(icon: "power", title: "登出设备", desc: "安全退出当前设备", tint: .systemGray)
        logoutActionBtn.tapAction = { [weak self] in self?.logoutAction() }
        let deleteActionBtn = ActionButton(icon: "trash.fill", title: "删除设备", desc: "清除设备数据后重新扫码", tint: .systemRed)
        deleteActionBtn.tapAction = { [weak self] in self?.deleteAction() }

        let row1 = UIStackView(arrangedSubviews: [scanAction, reloginAction])
        row1.axis = .horizontal
        row1.spacing = 10
        row1.distribution = .fillEqually
        let row2 = UIStackView(arrangedSubviews: [wakeAction, logoutActionBtn])
        row2.axis = .horizontal
        row2.spacing = 10
        row2.distribution = .fillEqually
        let gridStack = UIStackView(arrangedSubviews: [gridLabel, row1, row2, deleteActionBtn])
        gridStack.axis = .vertical
        gridStack.spacing = 10
        gridStack.translatesAutoresizingMaskIntoConstraints = false

        heroView.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(heroView)
        view.addSubview(card)
        view.addSubview(gridStack)
        NSLayoutConstraint.activate([
            heroView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 12),
            heroView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            heroView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            card.topAnchor.constraint(equalTo: heroView.bottomAnchor, constant: 16),
            card.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            card.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            statusStack.topAnchor.constraint(equalTo: card.topAnchor, constant: 18),
            statusStack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 18),
            statusStack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -18),
            statusStack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -18),
            gridStack.topAnchor.constraint(equalTo: card.bottomAnchor, constant: 20),
            gridStack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            gridStack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            deleteActionBtn.heightAnchor.constraint(equalToConstant: 68)
        ])
    }

    private func loadStatus(silent: Bool = false) {
        PortalService.shared.fetchWechatStatus { [weak self] result in
            switch result {
            case .failure(let error):
                if !silent {
                    self?.statusLabel.text = "未绑定或未在线"
                    self?.detailLabel.text = error.message
                }
            case .success(let status):
                let newStatus = status.status ?? "未知"
                if let last = self?.lastKnownStatus, last != newStatus {
                    let content = UNMutableNotificationContent()
                    content.title = "微信协议状态变更"
                    content.body = "状态从「\(last)」变为「\(newStatus)」"
                    content.sound = .default
                    let request = UNNotificationRequest(identifier: "wx_status_\(Date().timeIntervalSince1970)", content: content, trigger: nil)
                    UNUserNotificationCenter.current().add(request, withCompletionHandler: nil)
                }
                self?.lastKnownStatus = newStatus
                self?.render(status)
            }
        }
    }

    private func render(_ status: PortalWechatStatus) {
        statusLabel.text = status.status ?? "未知状态"
        detailLabel.text = "微信ID：\(status.wxid ?? "-")\n昵称：\(status.nickname ?? "-")\n设备：\(status.device ?? "-")\n登录时间：\(status.loginTime ?? "-")\n刷新时间：\(status.refreshTime ?? "-")"
    }

    @objc private func scanLogin() { confirmWechatAction(title: "扫码登录", message: "确认开始微信扫码登录吗？扫码成功后将进入自动轮询状态。", path: "/api/portal/wx/scan-login", deductCoin: true) }
    @objc private func relogin() { confirmWechatAction(title: "重新登录", message: "确认重新获取微信登录二维码吗？", path: "/api/portal/wx/relogin", deductCoin: false) }
    @objc private func wakeLogin() { confirmWechatAction(title: "唤醒登录", message: "确认执行微信唤醒登录吗？", path: "/api/portal/wx/wake-login", deductCoin: false) }
    @objc private func logoutAction() { confirmWechatAction(title: "登出设备", message: "确认登出当前微信设备吗？", path: "/api/portal/wx/logout", deductCoin: false) }
    @objc private func deleteAction() { confirmWechatAction(title: "删除设备", message: "确认删除当前微信设备数据吗？删除后需要重新扫码登录。", path: "/api/portal/wx/delete", deductCoin: false) }


    private func confirmWechatAction(title: String, message: String, path: String, deductCoin: Bool) {
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "确认", style: .default) { _ in
            self.performAction(path: path, deductCoin: deductCoin)
        })
        present(alert, animated: true)
    }

    private func performAction(path: String, deductCoin: Bool) {
        PortalService.shared.performWechatAction(path: path) { result in
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let data):
                if let qr = data.qrBase64, let uuid = data.uuid {
                    self.currentUUID = uuid
                    self.currentDeductCoin = deductCoin
                    let vc = WechatQRCodeViewController(base64String: qr, message: data.message ?? "请扫码", onDismiss: { [weak self] in
                        self?.stopPolling()
                        self?.qrModalNavigationController = nil
                    })
                    let nav = AppNavigationController(rootViewController: vc)
                    nav.modalPresentationStyle = .formSheet
                    self.qrModalNavigationController = nav
                    self.present(nav, animated: true) {
                        self.startPolling()
                    }
                } else {
                    self.showMessage(data.message ?? "操作成功")
                    self.loadStatus()
                }
            }
        }
    }

    private func startPolling() {
        pollingTimer?.invalidate()
        pollingTimer = nil
        pollFailureCount = 0
        pollStatus()
        pollingTimer = Timer.scheduledTimer(withTimeInterval: 3.0, repeats: true) { _ in
            self.pollStatus()
        }
    }

    private func stopPolling() {
        pollingTimer?.invalidate()
        pollingTimer = nil
        currentUUID = nil
        currentDeductCoin = false
        pollFailureCount = 0
    }

    private func pollStatus() {
        guard let uuid = currentUUID else { return }
        PortalService.shared.pollWechatLogin(uuid: uuid, deductCoin: currentDeductCoin) { result in
            switch result {
            case .failure(let error):
                self.pollFailureCount += 1
                if self.pollFailureCount < 4 {
                    if let qrVC = self.qrModalNavigationController?.topViewController as? WechatQRCodeViewController {
                        qrVC.updatePollingStatus("服务连接中断，正在重试（\(self.pollFailureCount)/3）")
                    }
                    return
                }
                let nav = self.qrModalNavigationController
                self.stopPolling()
                self.qrModalNavigationController = nil
                nav?.dismiss(animated: true) {
                    self.handle(error)
                }
            case .success(let data):
                self.pollFailureCount = 0
                if data.needPoll == true {
                    if let qrVC = self.qrModalNavigationController?.topViewController as? WechatQRCodeViewController {
                        qrVC.updatePollingStatus(data.message ?? "等待扫码中")
                    }
                    return
                }
                let nav = self.qrModalNavigationController
                self.stopPolling()
                self.qrModalNavigationController = nil
                nav?.dismiss(animated: true) {
                    self.showMessage(data.message ?? "登录成功")
                }
                self.loadStatus()
                AppSessionStore.shared.refreshIfPossible(silent: true)
            }
        }
    }
}

@available(iOS 13.0, *)
final class WechatQRCodeViewController: BaseNativeViewController {
    private let base64String: String
    private let messageText: String
    private let onDismiss: (() -> Void)?
    private let imageView = UIImageView()
    private let stateLabel = UILabel()

    init(base64String: String, message: String, onDismiss: (() -> Void)? = nil) {
        self.base64String = base64String
        self.messageText = message
        self.onDismiss = onDismiss
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemBackground
        navigationItem.rightBarButtonItem = UIBarButtonItem(barButtonSystemItem: .close, target: self, action: #selector(closeTapped))

        let heroView = SectionHeroView(icon: "qrcode.viewfinder", title: "扫码登录", subtitle: "请用微信扫码，完成后页面会自动刷新状态", tint: .systemBlue)
        heroView.translatesAutoresizingMaskIntoConstraints = false

        imageView.translatesAutoresizingMaskIntoConstraints = false
        imageView.contentMode = .scaleAspectFit
        imageView.layer.cornerRadius = 24
        imageView.layer.masksToBounds = true
        imageView.backgroundColor = .secondarySystemBackground
        imageView.layer.borderWidth = 1
        imageView.layer.borderColor = UIColor.systemGray5.cgColor

        if let image = QRImageDecoder.decodeImage(from: base64String) {
            imageView.image = image
        } else {
            let fallback = UIImage.SymbolConfiguration(pointSize: 54, weight: .light)
            imageView.image = UIImage(systemName: "qrcode", withConfiguration: fallback)
            imageView.tintColor = .systemGray3
        }

        stateLabel.translatesAutoresizingMaskIntoConstraints = false
        stateLabel.text = messageText
        stateLabel.numberOfLines = 0
        stateLabel.textAlignment = .center
        stateLabel.font = UIFont.systemFont(ofSize: 14)
        stateLabel.textColor = .secondaryLabel

        let hintLabel = UILabel()
        hintLabel.translatesAutoresizingMaskIntoConstraints = false
        hintLabel.text = "若二维码未显示，请关闭后重新点击扫码登录。"
        hintLabel.numberOfLines = 0
        hintLabel.textAlignment = .center
        hintLabel.font = UIFont.systemFont(ofSize: 12)
        hintLabel.textColor = .systemGray

        view.addSubview(heroView)
        view.addSubview(imageView)
        view.addSubview(stateLabel)
        view.addSubview(hintLabel)
        NSLayoutConstraint.activate([
            heroView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 12),
            heroView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 20),
            heroView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -20),
            imageView.centerXAnchor.constraint(equalTo: view.centerXAnchor),
            imageView.topAnchor.constraint(equalTo: heroView.bottomAnchor, constant: 24),
            imageView.widthAnchor.constraint(equalToConstant: 260),
            imageView.heightAnchor.constraint(equalToConstant: 260),
            stateLabel.topAnchor.constraint(equalTo: imageView.bottomAnchor, constant: 16),
            stateLabel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 20),
            stateLabel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -20),
            hintLabel.topAnchor.constraint(equalTo: stateLabel.bottomAnchor, constant: 10),
            hintLabel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 28),
            hintLabel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -28)
        ])
    }

    func updatePollingStatus(_ text: String) {
        stateLabel.text = text
    }

    @objc private func closeTapped() {
        onDismiss?()
        dismiss(animated: true)
    }
}

@available(iOS 13.0, *)
final class CoinTasksViewController: BaseNativeViewController {
    private lazy var heroView = SectionHeroView(icon: "star.fill", title: "积分任务", subtitle: "每日打卡、祈福与积分补充入口", tint: .systemOrange)
    private let redeemField = UITextField()
    private let redeemButton = UIButton(type: .system)
    private var isRedeeming = false

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
    }

    private func setupUI() {
        let scrollView = UIScrollView()
        let stack = UIStackView()
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        stack.axis = .vertical
        stack.spacing = 16
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -16),
            stack.widthAnchor.constraint(equalTo: view.widthAnchor, constant: -32)
        ])

        stack.addArrangedSubview(heroView)

        let checkinAction = ActionButton(icon: "checkmark.circle.fill", title: "每日打卡", desc: "签到领取积分奖励", tint: .systemBlue)
        checkinAction.tapAction = { [weak self] in self?.checkinTapped() }
        let prayAction = ActionButton(icon: "hands.sparkles.fill", title: "每日祈福", desc: "祈福获得额外积分", tint: .systemPurple)
        prayAction.tapAction = { [weak self] in self?.prayTapped() }

        let actionsCard = UIView()
        actionsCard.applyCardStyle()
        let actionsTitle = UILabel()
        actionsTitle.text = "每日任务"
        actionsTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let actionsStack = UIStackView(arrangedSubviews: [actionsTitle, checkinAction, prayAction])
        actionsStack.axis = .vertical
        actionsStack.spacing = 10
        actionsStack.translatesAutoresizingMaskIntoConstraints = false
        [checkinAction, prayAction].forEach { $0.heightAnchor.constraint(equalToConstant: 68).isActive = true }
        actionsCard.addSubview(actionsStack)
        NSLayoutConstraint.activate([
            actionsStack.topAnchor.constraint(equalTo: actionsCard.topAnchor, constant: 18),
            actionsStack.leadingAnchor.constraint(equalTo: actionsCard.leadingAnchor, constant: 18),
            actionsStack.trailingAnchor.constraint(equalTo: actionsCard.trailingAnchor, constant: -18),
            actionsStack.bottomAnchor.constraint(equalTo: actionsCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(actionsCard)

        let buyCard = UIView()
        buyCard.applyCardStyle()
        let buyIcon = UIImageView()
        buyIcon.translatesAutoresizingMaskIntoConstraints = false
        buyIcon.image = UIImage(systemName: "creditcard.fill")
        buyIcon.tintColor = .systemOrange
        buyIcon.contentMode = .scaleAspectFit
        let buyTitle = UILabel()
        buyTitle.text = "积分购买"
        buyTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let buyDesc = UILabel()
        buyDesc.text = "积分不足时可通过购买页面补充积分"
        buyDesc.font = UIFont.systemFont(ofSize: 13)
        buyDesc.textColor = .secondaryLabel
        buyDesc.numberOfLines = 0
        let buyBtn = UIButton(type: .system)
        buyBtn.setTitle("前往购买", for: .normal)
        buyBtn.applyPrimaryStyle(color: .systemOrange)
        buyBtn.heightAnchor.constraint(equalToConstant: 48).isActive = true
        buyBtn.addTarget(self, action: #selector(buyTapped), for: .touchUpInside)
        let buyTip = UILabel()
        buyTip.text = "购买后可在下方卡密兑换处直接完成充值"
        buyTip.font = UIFont.systemFont(ofSize: 12)
        buyTip.textColor = .secondaryLabel
        buyTip.numberOfLines = 0
        let buyStack = UIStackView(arrangedSubviews: [buyIcon, buyTitle, buyDesc, buyTip, buyBtn])
        buyStack.axis = .vertical
        buyStack.alignment = .fill
        buyStack.spacing = 10
        buyStack.translatesAutoresizingMaskIntoConstraints = false
        buyIcon.widthAnchor.constraint(equalToConstant: 36).isActive = true
        buyIcon.heightAnchor.constraint(equalToConstant: 36).isActive = true
        buyCard.addSubview(buyStack)
        NSLayoutConstraint.activate([
            buyStack.topAnchor.constraint(equalTo: buyCard.topAnchor, constant: 18),
            buyStack.leadingAnchor.constraint(equalTo: buyCard.leadingAnchor, constant: 18),
            buyStack.trailingAnchor.constraint(equalTo: buyCard.trailingAnchor, constant: -18),
            buyStack.bottomAnchor.constraint(equalTo: buyCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(buyCard)

        let redeemCard = UIView()
        redeemCard.applyCardStyle()
        let redeemIcon = UIImageView()
        redeemIcon.translatesAutoresizingMaskIntoConstraints = false
        redeemIcon.image = UIImage(systemName: "ticket.fill")
        redeemIcon.tintColor = .systemGreen
        redeemIcon.contentMode = .scaleAspectFit
        let redeemTitle = UILabel()
        redeemTitle.text = "卡密兑换"
        redeemTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let redeemDesc = UILabel()
        redeemDesc.text = "购买卡密后可直接兑换积分，支持普通卡密和赠送卡密。"
        redeemDesc.font = UIFont.systemFont(ofSize: 13)
        redeemDesc.textColor = .secondaryLabel
        redeemDesc.numberOfLines = 0
        redeemField.applyAppInputStyle(placeholder: "请输入卡密（XDD... 或 ZSKM...）")
        redeemField.autocapitalizationType = .allCharacters
        redeemField.autocorrectionType = .no
        redeemField.heightAnchor.constraint(equalToConstant: 50).isActive = true
        let redeemTip = UILabel()
        redeemTip.text = "卡密格式以 XDD 或 ZSKM 开头"
        redeemTip.font = UIFont.systemFont(ofSize: 12)
        redeemTip.textColor = .secondaryLabel
        redeemTip.numberOfLines = 0
        redeemButton.setTitle("兑换积分", for: .normal)
        redeemButton.applyPrimaryStyle(color: .systemGreen)
        redeemButton.heightAnchor.constraint(equalToConstant: 48).isActive = true
        redeemButton.addTarget(self, action: #selector(redeemTapped), for: .touchUpInside)
        let redeemStack = UIStackView(arrangedSubviews: [redeemIcon, redeemTitle, redeemDesc, redeemField, redeemTip, redeemButton])
        redeemStack.axis = .vertical
        redeemStack.alignment = .fill
        redeemStack.spacing = 10
        redeemStack.translatesAutoresizingMaskIntoConstraints = false
        redeemIcon.widthAnchor.constraint(equalToConstant: 36).isActive = true
        redeemIcon.heightAnchor.constraint(equalToConstant: 36).isActive = true
        redeemCard.addSubview(redeemStack)
        NSLayoutConstraint.activate([
            redeemStack.topAnchor.constraint(equalTo: redeemCard.topAnchor, constant: 18),
            redeemStack.leadingAnchor.constraint(equalTo: redeemCard.leadingAnchor, constant: 18),
            redeemStack.trailingAnchor.constraint(equalTo: redeemCard.trailingAnchor, constant: -18),
            redeemStack.bottomAnchor.constraint(equalTo: redeemCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(redeemCard)
    }

    @objc private func checkinTapped() {
        PortalService.shared.checkin { result in
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let message):
                self.showMessage(message)
                AppSessionStore.shared.refreshIfPossible(silent: true)
            }
        }
    }

    @objc private func prayTapped() {
        PortalService.shared.pray { result in
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let message):
                self.showMessage(message)
                AppSessionStore.shared.refreshIfPossible(silent: true)
            }
        }
    }

    @objc private func buyTapped() {
        let vc = WebBrowserViewController(url: AppEnvironment.coinPurchaseURL)
        navigationController?.pushViewController(vc, animated: true)
    }

    @objc private func redeemTapped() {
        if isRedeeming { return }
        let token = redeemField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if token.isEmpty {
            showMessage("请输入卡密")
            redeemField.becomeFirstResponder()
            return
        }
        isRedeeming = true
        redeemButton.isEnabled = false
        redeemButton.setTitle("兑换中...", for: .normal)
        PortalService.shared.redeemKey(token: token) { result in
            self.isRedeeming = false
            self.redeemButton.isEnabled = true
            self.redeemButton.setTitle("兑换积分", for: .normal)
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let message):
                self.redeemField.text = ""
                self.showMessage(message)
                AppSessionStore.shared.refreshIfPossible(silent: true)
            }
        }
    }
}
