import UIKit
import UserNotifications


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
        LiquidGlassEffect.applyToTabBar(tabBar)
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
        tasks.tabBarItem = UITabBarItem(title: "积分任务", image: UIImage(systemName: "star"), selectedImage: UIImage(systemName: "star.fill"))

        let jd = AppNavigationController(rootViewController: JdTabRootViewController())
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
        let direction: Int = gesture.direction == .left ? 1 : (gesture.direction == .right ? -1 : 0)
        if direction == 0 { return }

        if let handler = currentInnerTabHandler() {
            if handler.consumeInnerSwipeBoundary?(direction: direction) == true { return }
            let idx = handler.innerTabIndex
            let total = handler.innerTabCount
            if direction > 0 && idx < total - 1 {
                handler.selectInnerTab(at: idx + 1)
                return
            }
            if direction < 0 && idx > 0 {
                handler.selectInnerTab(at: idx - 1)
                return
            }
        }

        if gesture.direction == .left {
            let next = selectedIndex + 1
            if next < count {
                animateTabSlide(to: next, fromRight: true)
            }
        } else if gesture.direction == .right {
            let prev = selectedIndex - 1
            if prev >= 0 {
                animateTabSlide(to: prev, fromRight: false)
            }
        }
    }

    private func currentInnerTabHandler() -> InnerTabSwipeHandling? {
        guard let nav = selectedViewController as? UINavigationController else { return nil }
        let top = nav.visibleViewController ?? nav.topViewController
        if let handler = top as? InnerTabSwipeHandling { return handler }
        if let jdRoot = top as? JdTabRootViewController {
            for child in jdRoot.children {
                if let handler = child as? InnerTabSwipeHandling { return handler }
            }
        }
        return nil
    }

    private func animateTabSlide(to index: Int, fromRight: Bool) {
        guard let fromView = selectedViewController?.view, let toVC = viewControllers?[index] else {
            selectedIndex = index
            return
        }
        let offset = fromRight ? view.bounds.width : -view.bounds.width
        toVC.view.frame = view.bounds.offsetBy(dx: offset, dy: 0)
        view.addSubview(toVC.view)
        UIView.animate(withDuration: 0.3, delay: 0, usingSpringWithDamping: 1.0, initialSpringVelocity: 0, options: .curveEaseInOut) {
            toVC.view.frame = self.view.bounds
            fromView.frame = self.view.bounds.offsetBy(dx: -offset, dy: 0)
        } completion: { _ in
            fromView.removeFromSuperview()
            self.selectedIndex = index
            self.resetScrollPosition(for: toVC)
            if let tabBarItems = self.tabBar.items, index < tabBarItems.count {
                let iconViews = self.tabBar.subviews.filter { String(describing: type(of: $0)).contains("UITabBarButton") }
                if index < iconViews.count {
                    LiquidGlassEffect.tabBarBounceAnimation(iconViews[index])
                }
            }
        }
    }

    private func resetScrollPosition(for vc: UIViewController) {
        if let nav = vc as? UINavigationController {
            nav.popToRootViewController(animated: false)
            if let topVC = nav.topViewController {
                if let resetable = topVC as? ResetableViewController {
                    resetable.resetToInitialState()
                }
                if let scrollView = findScrollView(in: topVC.view) {
                    scrollView.setContentOffset(.zero, animated: false)
                }
            }
        } else if let scrollView = findScrollView(in: vc.view) {
            scrollView.setContentOffset(.zero, animated: false)
        }
    }

    private func findScrollView(in view: UIView) -> UIScrollView? {
        if let scrollView = view as? UIScrollView {
            return scrollView
        }
        for subview in view.subviews {
            if let found = findScrollView(in: subview) {
                return found
            }
        }
        return nil
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
                if !AppSessionStore.shared.isAuthenticated {
                    self?.pendingProtectedIndex = index
                    AppSessionStore.shared.clearSession(requireLogin: true)
                }
            }
        }
        return true
    }

    func tabBarController(_ tabBarController: UITabBarController, didSelect viewController: UIViewController) {
        resetScrollPosition(for: viewController)
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


final class AppNavigationController: UINavigationController {
    override func viewDidLoad() {
        super.viewDidLoad()
        navigationBar.prefersLargeTitles = true
        LiquidGlassEffect.applyToNavBar(navigationBar)
        navigationBar.tintColor = .systemBlue
    }
}


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

final class CoinLogViewController: BaseNativeViewController, UITableViewDataSource, UITableViewDelegate {
    private let tableView = UITableView(frame: .zero, style: .grouped)
    private let tabScrollView = UIScrollView()
    private let tabStack = UIStackView()
    private let refreshControl = UIRefreshControl()
    private var logs: [CoinLog] = []
    private let tabs: [(source: String, title: String, icon: String)] = [
        ("", "全部", "tray.2.fill"),
        ("Web端", "Web端", "globe"),
        ("App端", "App端", "iphone"),
        ("微信", "微信", "message.fill"),
        ("后台及其他", "后台及其他", "gearshape.2.fill"),
    ]
    private var currentTabIndex = 0
    private var tabButtons: [UIButton] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "积分变动"
        view.backgroundColor = .systemGroupedBackground
        navigationItem.largeTitleDisplayMode = .automatic

        setupTabBar()
        setupTableView()
        loadLogs()
    }

    private func setupTabBar() {
        tabScrollView.showsHorizontalScrollIndicator = false
        tabScrollView.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(tabScrollView)

        tabStack.axis = .horizontal
        tabStack.spacing = 8
        tabStack.translatesAutoresizingMaskIntoConstraints = false
        tabScrollView.addSubview(tabStack)

        NSLayoutConstraint.activate([
            tabScrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 4),
            tabScrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tabScrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tabScrollView.heightAnchor.constraint(equalToConstant: 40),
            tabStack.topAnchor.constraint(equalTo: tabScrollView.topAnchor),
            tabStack.bottomAnchor.constraint(equalTo: tabScrollView.bottomAnchor),
            tabStack.leadingAnchor.constraint(equalTo: tabScrollView.leadingAnchor, constant: 16),
            tabStack.trailingAnchor.constraint(equalTo: tabScrollView.trailingAnchor, constant: -16),
            tabStack.heightAnchor.constraint(equalTo: tabScrollView.heightAnchor),
        ])

        for (index, tab) in tabs.enumerated() {
            let btn = UIButton(type: .system)
            let config = UIImage.SymbolConfiguration(pointSize: 11, weight: .semibold)
            btn.setImage(UIImage(systemName: tab.icon, withConfiguration: config), for: .normal)
            btn.setTitle(" \(tab.title)", for: .normal)
            btn.titleLabel?.font = .systemFont(ofSize: 13, weight: .semibold)
            btn.layer.cornerRadius = 18
            btn.contentEdgeInsets = UIEdgeInsets(top: 6, left: 14, bottom: 6, right: 14)
            btn.tag = index
            btn.addTarget(self, action: #selector(tabTapped(_:)), for: .touchUpInside)
            tabButtons.append(btn)
            tabStack.addArrangedSubview(btn)
        }
        updateTabAppearance(animated: false)
    }

    private func updateTabAppearance(animated: Bool) {
        let changes = {
            for (i, btn) in self.tabButtons.enumerated() {
                if i == self.currentTabIndex {
                    btn.backgroundColor = .systemBlue
                    btn.setTitleColor(.white, for: .normal)
                    btn.tintColor = .white
                    btn.transform = CGAffineTransform(scaleX: 1.0, y: 1.0)
                } else {
                    btn.backgroundColor = .secondarySystemGroupedBackground
                    btn.setTitleColor(.secondaryLabel, for: .normal)
                    btn.tintColor = .secondaryLabel
                    btn.transform = CGAffineTransform(scaleX: 0.95, y: 0.95)
                }
            }
        }
        if animated {
            UIView.animate(withDuration: 0.25, delay: 0, usingSpringWithDamping: 0.7, initialSpringVelocity: 0.5, options: .curveEaseInOut, animations: changes)
        } else {
            changes()
        }
    }

    @objc private func tabTapped(_ sender: UIButton) {
        guard sender.tag != currentTabIndex else { return }
        currentTabIndex = sender.tag
        updateTabAppearance(animated: true)
        loadLogs()
    }

    private func setupTableView() {
        tableView.dataSource = self
        tableView.delegate = self
        tableView.register(CoinLogCell.self, forCellReuseIdentifier: "CoinLogCell")
        tableView.separatorStyle = .none
        tableView.backgroundColor = .clear
        tableView.refreshControl = refreshControl
        refreshControl.addTarget(self, action: #selector(loadLogs), for: .valueChanged)
        tableView.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(tableView)
        NSLayoutConstraint.activate([
            tableView.topAnchor.constraint(equalTo: tabScrollView.bottomAnchor, constant: 4),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
        ])

        let swipeLeft = UISwipeGestureRecognizer(target: self, action: #selector(handleSwipe(_:)))
        swipeLeft.direction = .left
        tableView.addGestureRecognizer(swipeLeft)
        let swipeRight = UISwipeGestureRecognizer(target: self, action: #selector(handleSwipe(_:)))
        swipeRight.direction = .right
        tableView.addGestureRecognizer(swipeRight)
    }

    @objc private func handleSwipe(_ gesture: UISwipeGestureRecognizer) {
        if gesture.direction == .left && currentTabIndex < tabs.count - 1 {
            currentTabIndex += 1
        } else if gesture.direction == .right && currentTabIndex > 0 {
            currentTabIndex -= 1
        } else { return }
        updateTabAppearance(animated: true)
        tabButtons[currentTabIndex].isHidden = false
        tabScrollView.scrollRectToVisible(tabButtons[currentTabIndex].frame, animated: true)
        loadLogs()
    }

    @objc @discardableResult private func loadLogs() {
        let source = tabs[currentTabIndex].source
        PortalService.shared.fetchCoinLogs(source: source.isEmpty ? nil : source) { [weak self] result in
            DispatchQueue.main.async {
                guard let self else { return }
                self.refreshControl.endRefreshing()
                switch result {
                case .success(let logs):
                    self.logs = logs
                    self.tableView.reloadData()
                case .failure(let error):
                    self.showMessage(error.message)
                }
            }
        }
    }

    func numberOfSections(in tableView: UITableView) -> Int { 1 }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        return logs.isEmpty ? 1 : logs.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        if logs.isEmpty {
            let cell = UITableViewCell()
            var config = cell.defaultContentConfiguration()
            config.text = "暂无积分变动记录"
            config.textProperties.color = .tertiaryLabel
            config.textProperties.alignment = .center
            cell.contentConfiguration = config
            cell.selectionStyle = .none
            return cell
        }
        let cell = tableView.dequeueReusableCell(withIdentifier: "CoinLogCell", for: indexPath) as! CoinLogCell
        cell.configure(with: logs[indexPath.row])
        return cell
    }

    func tableView(_ tableView: UITableView, heightForRowAt indexPath: IndexPath) -> CGFloat {
        return logs.isEmpty ? 120 : UITableView.automaticDimension
    }

    func tableView(_ tableView: UITableView, viewForHeaderInSection section: Int) -> UIView? { nil }
    func tableView(_ tableView: UITableView, heightForHeaderInSection section: Int) -> CGFloat { 0 }
    func tableView(_ tableView: UITableView, viewForFooterInSection section: Int) -> UIView? { nil }
    func tableView(_ tableView: UITableView, heightForFooterInSection section: Int) -> CGFloat { 0 }
}

final class CoinLogCell: UITableViewCell {
    private let iconBg = UIView()
    private let iconView = UIImageView()
    private let typeLabel = UILabel()
    private let detailLabel = UILabel()
    private let amountLabel = UILabel()
    private let balanceLabel = UILabel()
    private let timeLabel = UILabel()
    private let sourcePill = UILabel()
    private let cardView = UIView()
    private let hStack = UIStackView()
    private let rightStack = UIStackView()

    override init(style: UITableViewCell.CellStyle, reuseIdentifier: String?) {
        super.init(style: style, reuseIdentifier: reuseIdentifier)
        selectionStyle = .none
        backgroundColor = .clear

        cardView.backgroundColor = .secondarySystemGroupedBackground
        cardView.layer.cornerRadius = 14
        contentView.addSubview(cardView)
        cardView.translatesAutoresizingMaskIntoConstraints = false

        iconBg.layer.cornerRadius = 20
        iconBg.translatesAutoresizingMaskIntoConstraints = false
        iconView.translatesAutoresizingMaskIntoConstraints = false
        iconView.contentMode = .scaleAspectFit
        iconView.preferredSymbolConfiguration = UIImage.SymbolConfiguration(pointSize: 16, weight: .medium)

        let leftStack = UIStackView(arrangedSubviews: [iconBg])
        leftStack.alignment = .center
        iconBg.addSubview(iconView)

        typeLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        typeLabel.textColor = .label

        detailLabel.font = .systemFont(ofSize: 12)
        detailLabel.textColor = .secondaryLabel
        detailLabel.numberOfLines = 1

        sourcePill.font = .systemFont(ofSize: 10, weight: .semibold)
        sourcePill.layer.cornerRadius = 8
        sourcePill.clipsToBounds = true
        sourcePill.textAlignment = .center

        let middleStack = UIStackView(arrangedSubviews: [typeLabel, detailLabel, sourcePill])
        middleStack.axis = .vertical
        middleStack.spacing = 3
        middleStack.alignment = .leading

        amountLabel.font = .systemFont(ofSize: 17, weight: .bold)
        amountLabel.textAlignment = .right

        balanceLabel.font = .systemFont(ofSize: 11)
        balanceLabel.textColor = .tertiaryLabel
        balanceLabel.textAlignment = .right

        rightStack.axis = .vertical
        rightStack.alignment = .trailing
        rightStack.spacing = 2
        rightStack.addArrangedSubview(amountLabel)
        rightStack.addArrangedSubview(balanceLabel)

        hStack.axis = .horizontal
        hStack.alignment = .center
        hStack.spacing = 12
        hStack.translatesAutoresizingMaskIntoConstraints = false
        hStack.addArrangedSubview(leftStack)
        hStack.addArrangedSubview(middleStack)
        hStack.addArrangedSubview(rightStack)
        cardView.addSubview(hStack)

        NSLayoutConstraint.activate([
            cardView.topAnchor.constraint(equalTo: contentView.topAnchor, constant: 5),
            cardView.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: 16),
            cardView.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -16),
            cardView.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -5),
            hStack.topAnchor.constraint(equalTo: cardView.topAnchor, constant: 12),
            hStack.leadingAnchor.constraint(equalTo: cardView.leadingAnchor, constant: 14),
            hStack.trailingAnchor.constraint(equalTo: cardView.trailingAnchor, constant: -14),
            hStack.bottomAnchor.constraint(equalTo: cardView.bottomAnchor, constant: -12),
            iconBg.widthAnchor.constraint(equalToConstant: 40),
            iconBg.heightAnchor.constraint(equalToConstant: 40),
            iconView.centerXAnchor.constraint(equalTo: iconBg.centerXAnchor),
            iconView.centerYAnchor.constraint(equalTo: iconBg.centerYAnchor),
            middleStack.widthAnchor.constraint(greaterThanOrEqualToConstant: 120),
            sourcePill.heightAnchor.constraint(equalToConstant: 18),
            sourcePill.widthAnchor.constraint(greaterThanOrEqualToConstant: 50),
        ])
    }

    required init?(coder: NSCoder) { fatalError() }

    func configure(with log: CoinLog) {
        let amount = log.amount
        let isPositive = amount > 0

        typeLabel.text = log.type ?? "其他"
        detailLabel.text = log.detail
        detailLabel.isHidden = log.detail?.isEmpty ?? true

        amountLabel.text = isPositive ? "+\(amount)" : "\(amount)"
        amountLabel.textColor = isPositive ? .systemGreen : .systemRed

        balanceLabel.text = "余额 \(log.balanceAfter)"

        let sourceColors: [String: (bg: UIColor, fg: UIColor)] = [
            "Web端": (.systemTeal.withAlphaComponent(0.15), .systemTeal),
            "App端": (.systemPurple.withAlphaComponent(0.15), .systemPurple),
            "微信": (.systemGreen.withAlphaComponent(0.15), .systemGreen),
            "后台及其他": (.systemOrange.withAlphaComponent(0.15), .systemOrange),
        ]
        let src = log.source ?? ""
        let colors = sourceColors[src] ?? (.systemGray.withAlphaComponent(0.12), .systemGray)
        sourcePill.text = "  \(src)  "
        sourcePill.backgroundColor = colors.bg
        sourcePill.textColor = colors.fg

        let iconBgColor: UIColor = isPositive ? .systemGreen.withAlphaComponent(0.12) : .systemRed.withAlphaComponent(0.12)
        iconBg.backgroundColor = iconBgColor
        iconView.tintColor = isPositive ? .systemGreen : .systemRed
        let iconName: String = {
            switch log.type {
            case "签到": return "checkmark.circle.fill"
            case "祈福": return "hands.sparkles.fill"
            case "充值": return "yensign.circle.fill"
            case "卡密兑换": return "ticket.fill"
            case "上车扣费": return "cart.fill"
            case "续费扣费": return "arrow.clockwise.circle.fill"
            case "退还": return "arrow.uturn.backward"
            case "转账": return "arrow.left.arrow.right"
            case "微信登录": return "message.fill"
            case "管理员操作": return "gearshape.fill"
            case "反馈奖励": return "gift.fill"
            default: return isPositive ? "arrow.down.circle.fill" : "arrow.up.circle.fill"
            }
        }()
        iconView.image = UIImage(systemName: iconName)
    }
}

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


final class HomeDashboardViewController: BaseNativeViewController {
    private let scrollView = UIScrollView()
    private let refreshControl = UIRefreshControl()
    private let stack = UIStackView()
    private let summaryLabel = UILabel()
    private let userInfoLabel = UILabel()
    private var cards: [InfoCardView] = []
    private var notificationSection: UIView?
    private var notificationStack: UIStackView?
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
        navigationController?.setNavigationBarHidden(true, animated: animated)
        reloadData()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        navigationController?.setNavigationBarHidden(false, animated: animated)
    }

    private func startWxAutoRefresh() {
        wxAutoRefreshTimer?.invalidate()
        wxAutoRefreshTimer = Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { [weak self] _ in
            guard AppSessionStore.shared.isAuthenticated else { return }
            self?.loadNotifications()
        }
    }

    private func setupUI() {
        let headerRow = UIView()
        let headerIcon = UILabel()
        headerIcon.text = "🏠"
        headerIcon.font = .systemFont(ofSize: 24)
        let headerTitle = UILabel()
        headerTitle.text = "首页"
        headerTitle.font = .systemFont(ofSize: 20, weight: .bold)
        let headerSubtitle = UILabel()
        headerSubtitle.text = "通知中心 · 项目概览"
        headerSubtitle.font = .systemFont(ofSize: 12)
        headerSubtitle.textColor = .secondaryLabel
        headerRow.addSubview(headerIcon)
        headerRow.addSubview(headerTitle)
        headerRow.addSubview(headerSubtitle)
        headerIcon.translatesAutoresizingMaskIntoConstraints = false
        headerTitle.translatesAutoresizingMaskIntoConstraints = false
        headerSubtitle.translatesAutoresizingMaskIntoConstraints = false
        headerRow.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(headerRow)
        NSLayoutConstraint.activate([
            headerRow.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 2),
            headerRow.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            headerRow.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            headerRow.heightAnchor.constraint(equalToConstant: 44),
            headerIcon.leadingAnchor.constraint(equalTo: headerRow.leadingAnchor),
            headerIcon.centerYAnchor.constraint(equalTo: headerRow.centerYAnchor),
            headerIcon.widthAnchor.constraint(equalToConstant: 32),
            headerTitle.leadingAnchor.constraint(equalTo: headerIcon.trailingAnchor, constant: 8),
            headerTitle.centerYAnchor.constraint(equalTo: headerRow.centerYAnchor),
            headerSubtitle.leadingAnchor.constraint(equalTo: headerTitle.trailingAnchor, constant: 8),
            headerSubtitle.centerYAnchor.constraint(equalTo: headerTitle.centerYAnchor)
        ])

        scrollView.translatesAutoresizingMaskIntoConstraints = false
        scrollView.refreshControl = refreshControl
        refreshControl.addTarget(self, action: #selector(reloadData), for: .valueChanged)
        stack.axis = .vertical
        stack.spacing = 16
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scrollView)
        scrollView.addSubview(stack)

        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: headerRow.bottomAnchor, constant: 8),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor),
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -12),
            stack.widthAnchor.constraint(equalTo: view.widthAnchor, constant: -32)
        ])

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
        if AppSessionStore.shared.isAuthenticated {
            let notifVC = NotificationListViewController()
            navigationController?.pushViewController(notifVC, animated: true)
        } else {
            tabBarController?.selectedIndex = 4
        }
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


final class ProjectsRootViewController: BaseNativeViewController, UISearchBarDelegate, InnerTabSwipeHandling {
    private let segmented = UISegmentedControl(items: ["活动中心", "我的项目", "项目抢兑", "协议接入"])
    private let container = UIView()
    private let searchBar = UISearchBar()
    private let categoryFilterScroll = UIScrollView()
    private let categoryFilterStack = UIStackView()
    private let activitiesVC = ActivitiesListViewController()
    private let myProjectsVC = MyProjectsListViewController()
    private lazy var projectRushNav: UINavigationController = {
        let nav = AppNavigationController(rootViewController: ProjectRushListViewController())
        nav.navigationBar.prefersLargeTitles = false
        nav.setNavigationBarHidden(true, animated: false)
        return nav
    }()
    private let protocolVC = ProtocolAccessViewController()
    private var currentVC: UIViewController?
    private var containerTopToSearchBar: NSLayoutConstraint!
    private var containerTopToSegmented: NSLayoutConstraint!
    private var selectedCategoryFilter = ""
    private let activityCategories: [(value: String, title: String)] = [
        ("", "全部"), ("现金类", "现金类"), ("积分换实物", "积分换实物"), ("抽奖类", "抽奖类"), ("其他类", "其他类")
    ]

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemGroupedBackground
        setupUI()
        switchTo(index: 0)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        refreshVisibleList()
    }

    private func refreshVisibleList() {
        switch segmented.selectedSegmentIndex {
        case 0: activitiesVC.reloadFromServer()
        case 1: myProjectsVC.reloadFromServer()
        default: break
        }
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        navigationController?.setNavigationBarHidden(false, animated: animated)
    }

    override func resetToInitialState() {
        guard isViewLoaded else { return }
        segmented.selectedSegmentIndex = 0
        searchBar.text = nil
        searchBar.resignFirstResponder()
        switchTo(index: 0)
    }

    private func setupUI() {
        let headerRow = UIView()
        let headerIcon = UILabel()
        headerIcon.text = "📁"
        headerIcon.font = .systemFont(ofSize: 24)
        let headerTitle = UILabel()
        headerTitle.text = "项目"
        headerTitle.font = .systemFont(ofSize: 20, weight: .bold)
        let headerSubtitle = UILabel()
        headerSubtitle.text = "活动中心 · 项目抢兑 · 协议接入"
        headerSubtitle.font = .systemFont(ofSize: 12)
        headerSubtitle.textColor = .secondaryLabel
        headerRow.addSubview(headerIcon)
        headerRow.addSubview(headerTitle)
        headerRow.addSubview(headerSubtitle)
        headerIcon.translatesAutoresizingMaskIntoConstraints = false
        headerTitle.translatesAutoresizingMaskIntoConstraints = false
        headerSubtitle.translatesAutoresizingMaskIntoConstraints = false
        headerRow.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(headerRow)
        NSLayoutConstraint.activate([
            headerRow.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 2),
            headerRow.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            headerRow.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            headerRow.heightAnchor.constraint(equalToConstant: 44),
            headerIcon.leadingAnchor.constraint(equalTo: headerRow.leadingAnchor),
            headerIcon.centerYAnchor.constraint(equalTo: headerRow.centerYAnchor),
            headerIcon.widthAnchor.constraint(equalToConstant: 32),
            headerTitle.leadingAnchor.constraint(equalTo: headerIcon.trailingAnchor, constant: 8),
            headerTitle.centerYAnchor.constraint(equalTo: headerRow.centerYAnchor),
            headerSubtitle.leadingAnchor.constraint(equalTo: headerTitle.trailingAnchor, constant: 8),
            headerSubtitle.centerYAnchor.constraint(equalTo: headerTitle.centerYAnchor)
        ])

        segmented.selectedSegmentIndex = 0
        segmented.addTarget(self, action: #selector(segmentChanged), for: .valueChanged)
        if #available(iOS 13.0, *) {
            segmented.setTitleTextAttributes([.font: UIFont.systemFont(ofSize: 12, weight: .semibold)], for: .normal)
        }
        segmented.translatesAutoresizingMaskIntoConstraints = false
        container.translatesAutoresizingMaskIntoConstraints = false
        searchBar.delegate = self
        searchBar.placeholder = "搜索项目名、活动名称…"
        searchBar.searchBarStyle = .minimal
        searchBar.translatesAutoresizingMaskIntoConstraints = false
        categoryFilterScroll.showsHorizontalScrollIndicator = false
        categoryFilterScroll.translatesAutoresizingMaskIntoConstraints = false
        categoryFilterStack.axis = .horizontal
        categoryFilterStack.spacing = 8
        categoryFilterStack.alignment = .fill
        categoryFilterStack.distribution = .fill
        categoryFilterStack.translatesAutoresizingMaskIntoConstraints = false
        categoryFilterScroll.addSubview(categoryFilterStack)
        setupCategoryFilterButtons()
        view.addSubview(segmented)
        view.addSubview(categoryFilterScroll)
        view.addSubview(searchBar)
        view.addSubview(container)
        containerTopToSearchBar = container.topAnchor.constraint(equalTo: searchBar.bottomAnchor, constant: 4)
        containerTopToSegmented = container.topAnchor.constraint(equalTo: segmented.bottomAnchor, constant: 4)
        containerTopToSearchBar.isActive = true
        containerTopToSegmented.isActive = false
        NSLayoutConstraint.activate([
            segmented.topAnchor.constraint(equalTo: headerRow.bottomAnchor, constant: 6),
            segmented.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            segmented.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            categoryFilterScroll.topAnchor.constraint(equalTo: segmented.bottomAnchor, constant: 6),
            categoryFilterScroll.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            categoryFilterScroll.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            categoryFilterScroll.heightAnchor.constraint(equalToConstant: 36),
            categoryFilterStack.topAnchor.constraint(equalTo: categoryFilterScroll.topAnchor),
            categoryFilterStack.leadingAnchor.constraint(equalTo: categoryFilterScroll.leadingAnchor),
            categoryFilterStack.trailingAnchor.constraint(equalTo: categoryFilterScroll.trailingAnchor),
            categoryFilterStack.bottomAnchor.constraint(equalTo: categoryFilterScroll.bottomAnchor),
            categoryFilterStack.heightAnchor.constraint(equalTo: categoryFilterScroll.heightAnchor),
            searchBar.topAnchor.constraint(equalTo: categoryFilterScroll.bottomAnchor, constant: 4),
            searchBar.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            searchBar.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            container.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            container.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            container.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
        updateCategoryFilterVisibility(for: segmented.selectedSegmentIndex)
    }

    private func setupCategoryFilterButtons() {
        categoryFilterStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        for item in activityCategories {
            let button = UIButton(type: .system)
            button.setTitle(item.title, for: .normal)
            button.titleLabel?.font = .systemFont(ofSize: 13, weight: .semibold)
            button.layer.cornerRadius = 16
            button.contentEdgeInsets = UIEdgeInsets(top: 7, left: 14, bottom: 7, right: 14)
            button.tag = activityCategories.firstIndex(where: { $0.value == item.value }) ?? 0
            button.addTarget(self, action: #selector(categoryFilterTapped(_:)), for: .touchUpInside)
            categoryFilterStack.addArrangedSubview(button)
        }
        refreshCategoryFilterButtons()
    }

    private func refreshCategoryFilterButtons() {
        for case let button as UIButton in categoryFilterStack.arrangedSubviews {
            let value = activityCategories[button.tag].value
            let selected = value == selectedCategoryFilter
            button.backgroundColor = selected ? UIColor.systemIndigo : UIColor.secondarySystemGroupedBackground
            button.setTitleColor(selected ? .white : .secondaryLabel, for: .normal)
        }
    }

    private func updateCategoryFilterVisibility(for index: Int) {
        let show = index == 0
        categoryFilterScroll.isHidden = !show
    }

    @objc private func categoryFilterTapped(_ sender: UIButton) {
        selectedCategoryFilter = activityCategories[sender.tag].value
        refreshCategoryFilterButtons()
        activitiesVC.applyCategory(selectedCategoryFilter)
    }

    @objc private func segmentChanged() {
        switchTo(index: segmented.selectedSegmentIndex)
    }

    private func switchTo(index: Int) {
        if let nav = currentVC as? UINavigationController, nav.viewControllers.first is ProjectRushListViewController {
            nav.popToRootViewController(animated: false)
        }
        currentVC?.willMove(toParent: nil)
        currentVC?.view.removeFromSuperview()
        currentVC?.removeFromParent()
        let vc: UIViewController
        switch index {
        case 0: vc = activitiesVC
        case 1: vc = myProjectsVC
        case 2: vc = projectRushNav
        case 3: vc = protocolVC
        default: vc = activitiesVC
        }
        searchBar.text = nil
        searchBar.showsCancelButton = false
        searchBar.resignFirstResponder()
        activitiesVC.applySearch("")
        myProjectsVC.applySearch("")
        updateCategoryFilterVisibility(for: index)
        let hideSearch = index == 2 || index == 3
        if hideSearch {
            searchBar.isHidden = true
            containerTopToSearchBar.isActive = false
            containerTopToSegmented.isActive = true
        } else {
            searchBar.isHidden = false
            containerTopToSearchBar.isActive = true
            containerTopToSegmented.isActive = false
            searchBar.placeholder = index == 0 ? "搜索活动名称…" : "搜索项目名、备注…"
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
        if index == 0 { activitiesVC.reloadFromServer() }
        if index == 1 { myProjectsVC.reloadFromServer() }
    }

    func searchBar(_ searchBar: UISearchBar, textDidChange searchText: String) {
        let q = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        if segmented.selectedSegmentIndex == 0 {
            activitiesVC.applySearch(q)
        } else if segmented.selectedSegmentIndex == 1 {
            myProjectsVC.applySearch(q)
        }
    }

    func searchBarSearchButtonClicked(_ searchBar: UISearchBar) {
        searchBar.resignFirstResponder()
    }

    func searchBarCancelButtonClicked(_ searchBar: UISearchBar) {
        searchBar.text = nil
        searchBar.resignFirstResponder()
        activitiesVC.applySearch("")
        myProjectsVC.applySearch("")
    }

    var innerTabCount: Int { segmented.numberOfSegments }

    var innerTabIndex: Int { segmented.selectedSegmentIndex }

    func selectInnerTab(at index: Int) {
        guard index >= 0, index < innerTabCount else { return }
        segmented.selectedSegmentIndex = index
        switchTo(index: index)
    }

    func consumeInnerSwipeBoundary(direction: Int) -> Bool {
        guard segmented.selectedSegmentIndex == 2,
              let nav = currentVC as? UINavigationController,
              nav.viewControllers.count > 1,
              direction < 0 else { return false }
        nav.popViewController(animated: true)
        return true
    }
}


final class ActivitiesListViewController: UITableViewController {
    private var activities: [PortalActivity] = []
    private var filteredActivities: [PortalActivity] = []
    private var isFiltering = false
    private var categoryFilter = ""
    private var currentSearchText = ""

    private func normalizeActivityCategory(_ category: String?) -> String {
        switch category?.trimmingCharacters(in: .whitespacesAndNewlines) ?? "" {
        case "现金类", "积分换实物", "抽奖类", "其他类":
            return category!.trimmingCharacters(in: .whitespacesAndNewlines)
        default:
            return "其他类"
        }
    }

    private func currentSource() -> [PortalActivity] {
        if categoryFilter.isEmpty { return activities }
        return activities.filter { normalizeActivityCategory($0.category) == categoryFilter }
    }

    func applyCategory(_ category: String) {
        categoryFilter = category
        applySearch(currentSearchText)
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        tableView = UITableView(frame: .zero, style: .insetGrouped)
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "activity")
        tableView.rowHeight = UITableView.automaticDimension
        tableView.estimatedRowHeight = 132
        refreshControl = UIRefreshControl()
        refreshControl?.addTarget(self, action: #selector(handleRefresh), for: .valueChanged)
    }

    func reloadFromServer() {
        loadData()
    }

    @objc private func handleRefresh() {
        loadData()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reloadFromServer()
    }

    func applySearch(_ text: String) {
        currentSearchText = text
        let source = currentSource()
        if text.isEmpty {
            isFiltering = false
            filteredActivities = []
        } else {
            isFiltering = true
            let q = text.lowercased()
            filteredActivities = source.filter { ($0.name.lowercased().contains(q)) || ($0.qingLongConfig?.lowercased().contains(q) ?? false) }
        }
        tableView.reloadData()
    }

    private func loadData() {
        PortalService.shared.fetchActivities { result in
            DispatchQueue.main.async {
                self.refreshControl?.endRefreshing()
            }
            switch result {
            case .failure(let error):
                (self.parent as? BaseNativeViewController)?.handle(error)
            case .success(let activities):
                self.activities = activities
                self.tableView.reloadData()
                self.refreshControl?.endRefreshing()
            }
        }
    }

    private var isSearching: Bool {
        return isFiltering
    }

    private var displayedActivities: [PortalActivity] {
        return isFiltering ? filteredActivities : currentSource()
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
        let price: String
        let badgeText: String
        let badgeKind: StatusBadgeLabel.Kind
        if item.isDailyDeduct == true {
            price = "每天 \(item.dailyCoin ?? 0) 积分"
            badgeText = "按天授权"
            badgeKind = .active
        } else if item.isMonthlyDeduct == true {
            price = "每月 \(item.monthlyCoin ?? 0) 积分"
            badgeText = "按月授权"
            badgeKind = .expiring
        } else {
            price = "一次 \(item.needCoin ?? 0) 积分"
            badgeText = "一次上车"
            badgeKind = .active
        }
        card.priceLabel.text = price
        card.badgeLabel.configure(text: badgeText, kind: badgeKind)
        card.metaLabel.text = (card.metaLabel.text ?? "") + "\n类型：\(normalizeActivityCategory(item.category))"
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




final class ProjectFormViewController: BaseNativeViewController {
    private let activity: PortalActivity
    private let scrollView = UIScrollView()
    private let stack = UIStackView()
    private var fieldViews: [String: UITextField] = [:]
    private var isSubmitting = false

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
        let priceDesc: String
        if activity.isDailyDeduct == true {
            priceDesc = "每天扣 \(activity.dailyCoin ?? 0) 积分"
        } else if activity.isMonthlyDeduct == true {
            priceDesc = "每月扣 \(activity.monthlyCoin ?? 0) 积分"
        } else {
            priceDesc = "一次扣 \(activity.needCoin ?? 0) 积分"
        }
        meta.text = priceDesc + " · 青龙：\(activity.qingLongConfig ?? "默认容器")"
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

        if activity.isDailyDeduct == true {
            let minDays = activity.minDays ?? 1
            let dayLabel = UILabel()
            dayLabel.text = "授权天数（最少\(minDays)天）"
            dayLabel.font = UIFont.systemFont(ofSize: 14, weight: .medium)
            let dayInput = UITextField()
            dayInput.applyAppInputStyle(placeholder: "最少\(minDays)天，如 \(minDays)/7/30")
            dayInput.keyboardType = .numberPad
            dayInput.heightAnchor.constraint(equalToConstant: 50).isActive = true
            fieldViews["__months"] = dayInput
            stack.addArrangedSubview(dayLabel)
            stack.addArrangedSubview(dayInput)
        } else if activity.isMonthlyDeduct == true {
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
        // 验证输入
        if activity.isDailyDeduct == true {
            let minDays = activity.minDays ?? 1
            if months < minDays {
                showMessage("授权天数最少\(minDays)天")
                return
            }
            if months > 365 {
                showMessage("授权天数不能超过365天")
                return
            }
        } else if activity.isMonthlyDeduct == true && months <= 0 {
            showMessage("请输入正确的授权月数")
            return
        }
        let totalCoin: Int
        if activity.isDailyDeduct == true {
            totalCoin = (activity.dailyCoin ?? 0) * months
        } else if activity.isMonthlyDeduct == true {
            totalCoin = (activity.monthlyCoin ?? 0) * months
        } else {
            totalCoin = activity.needCoin ?? 0
        }
        let expireText: String? = {
            guard (activity.isDailyDeduct == true || activity.isMonthlyDeduct == true), months > 0 else { return nil }
            let calendar = Calendar.current
            let component: Calendar.Component = activity.isDailyDeduct == true ? .day : .month
            if let date = calendar.date(byAdding: component, value: months, to: Date()) {
                let formatter = DateFormatter()
                formatter.dateFormat = "yyyy-MM-dd"
                return formatter.string(from: date)
            }
            return nil
        }()
        var message = "项目：\(activity.name)\n备注名：\(remarks)\n将扣积分：\(totalCoin)"
        if activity.isDailyDeduct == true {
            message += "\n授权天数：\(months)"
            if let expireText, !expireText.isEmpty {
                message += "\n预计有效期至：\(expireText)"
            }
        } else if activity.isMonthlyDeduct == true {
            message += "\n授权月数：\(months)"
            if let expireText, !expireText.isEmpty {
                message += "\n预计有效期至：\(expireText)"
            }
        } else {
            message += "\n生效方式：一次性上车"
        }
        message += "\n确认后才会正式上车并扣除积分。"
        let alert = UIAlertController(title: "确认上车", message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "确认上车", style: .default) { _ in
            guard !self.isSubmitting else {
                self.showMessage("请勿重复提交，上一笔上车请求正在处理中")
                return
            }
            self.isSubmitting = true
            PortalService.shared.createProject(activityId: self.activity.id, inputs: inputs, remarks: remarks, months: months) { result in
                self.isSubmitting = false
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
    private var isFiltering = false
    private var projectActionBusyKeys: Set<String> = []

    override func viewDidLoad() {
        super.viewDidLoad()
        tableView = UITableView(frame: .zero, style: .insetGrouped)
        tableView.register(UITableViewCell.self, forCellReuseIdentifier: "project")
        tableView.rowHeight = UITableView.automaticDimension
        tableView.estimatedRowHeight = 132
        tableView.separatorStyle = .none
        refreshControl = UIRefreshControl()
        refreshControl?.addTarget(self, action: #selector(handleRefresh), for: .valueChanged)
    }

    func reloadFromServer() {
        loadData()
    }

    @objc private func handleRefresh() {
        loadData()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reloadFromServer()
    }

    private func loadData() {
        PortalService.shared.fetchProjects { result in
            DispatchQueue.main.async {
                self.refreshControl?.endRefreshing()
            }
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

    func applySearch(_ text: String) {
        if text.isEmpty {
            isFiltering = false
            groups = allGroups
        } else {
            isFiltering = true
            let q = text.lowercased()
            groups = allGroups.filter { group in
                if group.activityName.lowercased().contains(q) { return true }
                if (group.qingLongConfig ?? "").lowercased().contains(q) { return true }
                return group.items.contains { item in
                    (item.displayName ?? "").lowercased().contains(q) ||
                    (item.remark ?? "").lowercased().contains(q)
                }
            }
        }
        tableView.reloadData()
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
        }
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

        let card = UIView()
        card.applyCardStyle(cornerRadius: 16)
        card.translatesAutoresizingMaskIntoConstraints = false

        let titleLabel = UILabel()
        titleLabel.text = item.displayName?.isEmpty == false ? item.displayName : (item.remark?.isEmpty == false ? item.remark : item.activityName)
        titleLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        titleLabel.numberOfLines = 1

        let status = item.bizStatus ?? "gray"
        let statusBadge = UILabel()
        if status == "active" {
            statusBadge.text = " \(item.bizStatusText ?? "有效中") "
            statusBadge.textColor = .systemGreen
        } else if status == "expiring" {
            statusBadge.text = " \(item.bizStatusText ?? "快到期") "
            statusBadge.textColor = .systemOrange
        } else if status == "expired" {
            statusBadge.text = " \(item.bizStatusText ?? "已失效") "
            statusBadge.textColor = .systemRed
        } else {
            statusBadge.text = " \(item.statusText ?? "未知") "
            statusBadge.textColor = .secondaryLabel
        }
        statusBadge.font = .systemFont(ofSize: 11, weight: .medium)

        let titleRow = UIStackView(arrangedSubviews: [titleLabel, UIView(), statusBadge])
        titleRow.axis = .horizontal
        titleRow.spacing = 8

        let metaLabel = UILabel()
        metaLabel.text = "到期：\(item.expireDate ?? "长期") · 青龙：\(item.qingLongConfig ?? "默认") · \(item.priceText ?? "")"
        metaLabel.font = .systemFont(ofSize: 12)
        metaLabel.textColor = .secondaryLabel
        metaLabel.numberOfLines = 0

        let btnRow = UIStackView()
        btnRow.axis = .horizontal
        btnRow.spacing = 8
        btnRow.distribution = .fillEqually

        let actions: [(String, String, UIColor, () -> Void)] = [
            ("查询", "magnifyingglass", .systemBlue, { [weak self] in self?.queryIncome(for: item) }),
            ("修改", "pencil", .systemPurple, { [weak self] in self?.updateProject(item) }),
            ("续费", "arrow.clockwise", .systemGreen, { [weak self] in self?.renewProject(item) }),
            ("删除", "trash", .systemRed, { [weak self] in self?.deleteProject(item) })
        ]

        for (title, icon, color, action) in actions {
            let btn = UIButton(type: .system)
            btn.setTitle(" \(title)", for: .normal)
            btn.setImage(UIImage(systemName: icon), for: .normal)
            btn.titleLabel?.font = .systemFont(ofSize: 11, weight: .medium)
            btn.tintColor = color
            btn.backgroundColor = color.withAlphaComponent(0.1)
            btn.layer.cornerRadius = 8
            btn.heightAnchor.constraint(equalToConstant: 34).isActive = true
            btn.addAction(UIAction { _ in action() }, for: .touchUpInside)
            btnRow.addArrangedSubview(btn)
        }

        let innerStack = UIStackView(arrangedSubviews: [titleRow, metaLabel, btnRow])
        innerStack.axis = .vertical
        innerStack.spacing = 8
        innerStack.translatesAutoresizingMaskIntoConstraints = false
        card.addSubview(innerStack)
        NSLayoutConstraint.activate([
            innerStack.topAnchor.constraint(equalTo: card.topAnchor, constant: 12),
            innerStack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 12),
            innerStack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -12),
            innerStack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -12)
        ])

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
        let isDaily = item.isDailyDeduct == true
        let promptText = isDaily ? "请输入续费天数" : "请输入续费月数"
        let alert = UIAlertController(title: "续费", message: promptText, preferredStyle: .alert)
        alert.addTextField { field in
            field.keyboardType = .numberPad
            field.text = "1"
        }
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "确认", style: .default) { _ in
            let months = Int(alert.textFields?.first?.text ?? "1") ?? 1
            let busyKey = "renew:\(item.activityId):\(item.remark ?? "")"
            guard !self.projectActionBusyKeys.contains(busyKey) else {
                (self.parent as? BaseNativeViewController)?.showMessage("请勿重复提交，上一笔续费请求正在处理中")
                return
            }
            self.projectActionBusyKeys.insert(busyKey)
            PortalService.shared.renewProject(activityId: item.activityId, remarks: item.remark ?? "", months: months) { result in
                self.projectActionBusyKeys.remove(busyKey)
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
        let calcPaidDays: () -> Int = {
            guard let daysLeft = item.daysLeft, daysLeft > 0 else { return 0 }
            guard let grantDateStr = item.grantExpireDate, !grantDateStr.isEmpty, item.needCoin == 0 else { return daysLeft }
            let formatter = DateFormatter()
            formatter.dateFormat = "yyyy-MM-dd"
            guard let grantDate = formatter.date(from: grantDateStr) else { return daysLeft }
            let calendar = Calendar.current
            let grantEnd = calendar.date(byAdding: .day, value: 1, to: grantDate) ?? grantDate
            let grantRemain = grantEnd.timeIntervalSinceNow / 86400.0
            let granted = grantRemain > 1 ? Int(grantRemain - 1) : 0
            return max(daysLeft - granted, 0)
        }
        let refundTip: String = {
            // 按天计费退积分
            if item.isDailyDeduct == true, let dailyCoin = item.dailyCoin, dailyCoin > 0, let daysLeft = item.daysLeft, daysLeft > 0 {
                let paidDays = calcPaidDays()
                let estimated = dailyCoin * paidDays
                return estimated > 0 ? "预计返还积分：\(estimated)（最终以服务端结算为准）" : "赠送时长内，删除不退还积分"
            }
            if item.isDailyDeduct == true {
                return "预计返还积分：以服务端结算为准"
            }
            // 按月计费退积分
            if item.isMonthlyDeduct == true, let needCoin = item.needCoin, needCoin > 0 {
                return "该账号由一次性活动转换，删除不退还积分"
            }
            if item.isMonthlyDeduct == true, let monthlyCoin = item.monthlyCoin, monthlyCoin > 0, let daysLeft = item.daysLeft, daysLeft > 0 {
                let paidDays = calcPaidDays()
                let estimated = Int((Double(monthlyCoin) * Double(paidDays) / 30.0) + 0.5)
                return estimated > 0 ? "预计返还积分：\(estimated)（最终以服务端结算为准）" : "赠送时长内，删除不退还积分"
            }
            if item.isMonthlyDeduct == true {
                return "预计返还积分：以服务端结算为准"
            }
            // 一次性扣费
            return "此活动为一次性扣费，删除不退还积分"
        }()
        let alert = UIAlertController(
            title: "确认删除",
            message: "项目：\(item.displayName ?? item.activityName ?? "项目")\n到期：\(item.expireDate ?? "长期 / 未记录")\n\(refundTip)",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "删除", style: .destructive) { _ in
            let busyKey = "delete:\(item.activityId):\(item.remark ?? "")"
            guard !self.projectActionBusyKeys.contains(busyKey) else {
                (self.parent as? BaseNativeViewController)?.showMessage("请勿重复提交，上一笔删除请求正在处理中")
                return
            }
            self.projectActionBusyKeys.insert(busyKey)
            PortalService.shared.deleteProject(activityId: item.activityId, remarks: item.remark ?? "") { result in
                self.projectActionBusyKeys.remove(busyKey)
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
        navigationItem.largeTitleDisplayMode = .never
        applyCompactNavigationTitle(titleText)
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

    private func applyCompactNavigationTitle(_ text: String) {
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 16, weight: .semibold)
        label.textColor = .label
        label.textAlignment = .center
        label.numberOfLines = 1
        label.lineBreakMode = .byTruncatingTail
        label.adjustsFontSizeToFitWidth = true
        label.minimumScaleFactor = 0.85
        let maxWidth = view.bounds.width > 0 ? view.bounds.width - 96 : UIScreen.main.bounds.width - 96
        label.frame = CGRect(x: 0, y: 0, width: maxWidth, height: 22)
        navigationItem.titleView = label
    }
}


final class ProjectEditCKViewController: BaseNativeViewController {
    private let project: PortalProject
    private let onSaved: (String) -> Void
    private var fieldInputs: [String: UITextField] = [:]
    private var fieldNames: [String] = []
    private let previewLabel = UILabel()
    private var ckTemplate: String = ""
    private var isRawMode = false

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

        ckTemplate = project.ckTemplate ?? ""
        let envValue = project.envValue ?? ""
        let inputFields = project.inputFields ?? []

        let templateFieldKeys = getCkTemplateFields(ckTemplate)
        let parsed = splitCkValueByTemplate(ckTemplate, envValue)
        let visibleFields = inputFields.filter { templateFieldKeys.contains($0.key) }

        if parsed != nil && !visibleFields.isEmpty {
            isRawMode = false
            fieldNames = visibleFields.map { $0.key }
            for field in visibleFields {
                fieldInputs[field.key] = nil
            }
        } else {
            isRawMode = true
        }

        let remarkLabel = UILabel()
        remarkLabel.text = "当前备注：\(project.remark ?? project.displayName ?? project.activityName ?? "项目")"
        remarkLabel.font = .systemFont(ofSize: 14, weight: .medium)
        remarkLabel.textColor = .secondaryLabel
        remarkLabel.numberOfLines = 0

        let noticeLabel = UILabel()
        if isRawMode {
            noticeLabel.text = "当前活动无法按字段自动拆分，请直接编辑原始值。请保留字段名和连接符格式。"
        } else {
            noticeLabel.text = "已按活动配置拆分为字段编辑。请只修改字段值，字段名和连接符会在保存时自动组合。"
        }
        noticeLabel.font = .systemFont(ofSize: 12)
        noticeLabel.textColor = .tertiaryLabel
        noticeLabel.numberOfLines = 0

        let scrollView = UIScrollView()
        let stack = UIStackView()
        stack.axis = .vertical
        stack.spacing = 12
        stack.translatesAutoresizingMaskIntoConstraints = false
        scrollView.translatesAutoresizingMaskIntoConstraints = false

        view.addSubview(remarkLabel)
        view.addSubview(noticeLabel)
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
        remarkLabel.translatesAutoresizingMaskIntoConstraints = false
        noticeLabel.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            remarkLabel.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 8),
            remarkLabel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            remarkLabel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            noticeLabel.topAnchor.constraint(equalTo: remarkLabel.bottomAnchor, constant: 4),
            noticeLabel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            noticeLabel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            scrollView.topAnchor.constraint(equalTo: noticeLabel.bottomAnchor, constant: 8),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor),
            stack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -16),
            stack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -32)
        ])

        if isRawMode {
            let titleLabel = UILabel()
            titleLabel.text = "原始 CK 值"
            titleLabel.font = .systemFont(ofSize: 13, weight: .semibold)
            titleLabel.textColor = .secondaryLabel
            let input = UITextField()
            input.text = envValue
            input.font = .monospacedSystemFont(ofSize: 14, weight: .regular)
            input.borderStyle = .roundedRect
            input.autocorrectionType = .no
            input.autocapitalizationType = .none
            input.backgroundColor = .secondarySystemBackground
            input.clearButtonMode = .whileEditing
            input.accessibilityIdentifier = "__raw__"
            input.addTarget(self, action: #selector(fieldChanged), for: .editingChanged)
            fieldInputs["__raw__"] = input
            let card = UIView()
            card.applyCardStyle(cornerRadius: 12)
            let innerStack = UIStackView(arrangedSubviews: [titleLabel, input])
            innerStack.axis = .vertical
            innerStack.spacing = 6
            innerStack.translatesAutoresizingMaskIntoConstraints = false
            card.addSubview(innerStack)
            NSLayoutConstraint.activate([
                innerStack.topAnchor.constraint(equalTo: card.topAnchor, constant: 12),
                innerStack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 12),
                innerStack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -12),
                innerStack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -12),
                input.heightAnchor.constraint(greaterThanOrEqualToConstant: 80)
            ])
            stack.addArrangedSubview(card)
        } else {
            for field in visibleFields {
                let key = field.key
                let prompt = field.prompt.isEmpty ? key : field.prompt
                let value = parsed?[key] ?? ""

                let titleLabel = UILabel()
                titleLabel.text = key
                titleLabel.font = .systemFont(ofSize: 13, weight: .semibold)
                titleLabel.textColor = .systemBlue

                let input = UITextField()
                input.text = value
                input.font = .monospacedSystemFont(ofSize: 14, weight: .regular)
                input.borderStyle = .roundedRect
                input.autocorrectionType = .no
                input.autocapitalizationType = .none
                input.backgroundColor = .secondarySystemBackground
                input.clearButtonMode = .whileEditing
                input.accessibilityIdentifier = key
                input.addTarget(self, action: #selector(fieldChanged), for: .editingChanged)
                fieldInputs[key] = input

                let card = UIView()
                card.applyCardStyle(cornerRadius: 12)
                let subviews: [UIView] = prompt != key ? {
                    let promptLabel = UILabel()
                    promptLabel.text = prompt
                    promptLabel.font = .systemFont(ofSize: 11)
                    promptLabel.textColor = .tertiaryLabel
                    return [titleLabel, promptLabel, input]
                }() : [titleLabel, input]
                let innerStack = UIStackView(arrangedSubviews: subviews)
                innerStack.axis = .vertical
                innerStack.spacing = 6
                innerStack.translatesAutoresizingMaskIntoConstraints = false
                card.addSubview(innerStack)
                NSLayoutConstraint.activate([
                    innerStack.topAnchor.constraint(equalTo: card.topAnchor, constant: 12),
                    innerStack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 12),
                    innerStack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -12),
                    innerStack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -12),
                    input.heightAnchor.constraint(greaterThanOrEqualToConstant: 40)
                ])
                stack.addArrangedSubview(card)
            }
        }

        let previewCard = UIView()
        previewCard.applyCardStyle(cornerRadius: 12)
        let previewTitle = UILabel()
        previewTitle.text = "保存预览"
        previewTitle.font = .systemFont(ofSize: 13, weight: .semibold)
        previewTitle.textColor = .secondaryLabel
        previewLabel.font = .monospacedSystemFont(ofSize: 12, weight: .regular)
        previewLabel.textColor = .label
        previewLabel.numberOfLines = 0
        previewLabel.backgroundColor = .tertiarySystemBackground
        previewLabel.layer.cornerRadius = 8
        previewLabel.clipsToBounds = true
        let previewStack = UIStackView(arrangedSubviews: [previewTitle, previewLabel])
        previewStack.axis = .vertical
        previewStack.spacing = 8
        previewStack.translatesAutoresizingMaskIntoConstraints = false
        previewCard.addSubview(previewStack)
        NSLayoutConstraint.activate([
            previewStack.topAnchor.constraint(equalTo: previewCard.topAnchor, constant: 12),
            previewStack.leadingAnchor.constraint(equalTo: previewCard.leadingAnchor, constant: 12),
            previewStack.trailingAnchor.constraint(equalTo: previewCard.trailingAnchor, constant: -12),
            previewStack.bottomAnchor.constraint(equalTo: previewCard.bottomAnchor, constant: -12),
            previewLabel.heightAnchor.constraint(greaterThanOrEqualToConstant: 60)
        ])
        stack.addArrangedSubview(previewCard)

        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissKb))
        tap.cancelsTouchesInView = false
        view.addGestureRecognizer(tap)

        updatePreview()
    }

    @objc private func dismissKb() { view.endEditing(true) }

    @objc private func fieldChanged() { updatePreview() }

    private func buildPreview() -> String {
        if isRawMode {
            return fieldInputs["__raw__"]?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        }
        if ckTemplate.isEmpty || fieldNames.isEmpty { return "" }
        var result = ckTemplate
        for name in fieldNames {
            let value = fieldInputs[name]?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            result = result.replacingOccurrences(of: "{{.\(name)}}", with: value)
        }
        return result.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func updatePreview() {
        let text = buildPreview()
        previewLabel.text = text.isEmpty ? "（暂无数据）" : text
    }

    private func getCkTemplateFields(_ template: String) -> [String] {
        var fields: [String] = []
        var seen = Set<String>()
        let regex = try? NSRegularExpression(pattern: "\\{\\{\\.([^}]+)\\}\\}")
        let nsTemplate = template as NSString
        let results = regex?.matches(in: template, range: NSRange(location: 0, length: nsTemplate.length)) ?? []
        for match in results {
            if match.numberOfRanges >= 2 {
                let key = nsTemplate.substring(with: match.range(at: 1))
                if !seen.contains(key) {
                    seen.insert(key)
                    fields.append(key)
                }
            }
        }
        return fields
    }

    private func splitCkValueByTemplate(_ template: String, _ value: String) -> [String: String]? {
        let fieldKeys = getCkTemplateFields(template)
        if fieldKeys.isEmpty { return nil }

        let regex = try? NSRegularExpression(pattern: "\\{\\{\\.([^}]+)\\}\\}")
        let nsTemplate = template as NSString
        let matches = regex?.matches(in: template, range: NSRange(location: 0, length: nsTemplate.length)) ?? []
        if matches.isEmpty { return nil }

        struct Placeholder {
            let key: String
            let prefix: String
            let endIdx: Int
        }

        var placeholders: [Placeholder] = []
        var cursor = 0
        for match in matches {
            let key = nsTemplate.substring(with: match.range(at: 1))
            let startIdx = match.range.location
            let endIdx = match.range.location + match.range.length
            let prefix = nsTemplate.substring(with: NSRange(location: cursor, length: startIdx - cursor))
            placeholders.append(Placeholder(key: key, prefix: prefix, endIdx: endIdx))
            cursor = endIdx
        }
        let suffix = nsTemplate.substring(from: cursor)

        var pos = 0
        var result: [String: String] = [:]
        let nsValue = value as NSString

        for i in 0..<placeholders.count {
            let p = placeholders[i]
            if !p.prefix.isEmpty {
                let prefixRange = nsValue.range(of: p.prefix, options: [], range: NSRange(location: pos, length: nsValue.length - pos))
                if prefixRange.location == NSNotFound { return nil }
                pos = prefixRange.location + prefixRange.length
            }

            let nextPrefix: String
            if i + 1 < placeholders.count {
                nextPrefix = placeholders[i + 1].prefix
            } else {
                nextPrefix = suffix
            }

            var end = nsValue.length
            if !nextPrefix.isEmpty {
                let nextRange = nsValue.range(of: nextPrefix, options: [], range: NSRange(location: pos, length: nsValue.length - pos))
                if nextRange.location == NSNotFound { return nil }
                end = nextRange.location
            }

            result[p.key] = nsValue.substring(with: NSRange(location: pos, length: end - pos))
            pos = end
        }

        if !suffix.isEmpty {
            let suffixRange = nsValue.range(of: suffix, options: [], range: NSRange(location: pos, length: nsValue.length - pos))
            if suffixRange.location == NSNotFound { return nil }
            pos = suffixRange.location + suffixRange.length
        }

        if pos != nsValue.length { return nil }
        return result
    }

    @objc private func saveTapped() {
        let ckValue = buildPreview()
        if ckValue.isEmpty {
            showMessage("请至少填写一个字段")
            return
        }
        if !isRawMode {
            for name in fieldNames {
                let value = fieldInputs[name]?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                if value.isEmpty {
                    showMessage("\(name) 不能为空")
                    fieldInputs[name]?.becomeFirstResponder()
                    return
                }
            }
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


final class ProtocolAccessViewController: BaseNativeViewController {
    private let subTabScroll = UIScrollView()
    private let subTabRow = UIStackView()
    private let container = UIView()
    private let yybVC = YybProtocolViewController()
    private let wechatVC = WechatProtocolViewController()
    private let bindVC = ProtocolBindViewController(embedded: true)
    private var currentVC: UIViewController?
    private var selectedIndex = 0
    private var subTabButtons: [UIButton] = []

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemGroupedBackground

        subTabScroll.showsHorizontalScrollIndicator = false
        subTabScroll.translatesAutoresizingMaskIntoConstraints = false
        subTabRow.axis = .horizontal
        subTabRow.spacing = 8
        subTabRow.distribution = .fill
        subTabRow.translatesAutoresizingMaskIntoConstraints = false
        container.translatesAutoresizingMaskIntoConstraints = false

        ["应用宝协议", "微信协议", "协议双绑"].enumerated().forEach { index, title in
            let btn = makeSubTabButton(title: title, index: index)
            btn.widthAnchor.constraint(greaterThanOrEqualToConstant: 96).isActive = true
            subTabButtons.append(btn)
            subTabRow.addArrangedSubview(btn)
        }
        refreshSubTabs()
        subTabScroll.addSubview(subTabRow)

        view.addSubview(subTabScroll)
        view.addSubview(container)
        NSLayoutConstraint.activate([
            subTabScroll.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 6),
            subTabScroll.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            subTabScroll.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            subTabScroll.heightAnchor.constraint(equalToConstant: 38),
            subTabRow.topAnchor.constraint(equalTo: subTabScroll.topAnchor),
            subTabRow.leadingAnchor.constraint(equalTo: subTabScroll.leadingAnchor),
            subTabRow.trailingAnchor.constraint(equalTo: subTabScroll.trailingAnchor),
            subTabRow.bottomAnchor.constraint(equalTo: subTabScroll.bottomAnchor),
            subTabRow.heightAnchor.constraint(equalTo: subTabScroll.heightAnchor),
            container.topAnchor.constraint(equalTo: subTabScroll.bottomAnchor, constant: 8),
            container.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            container.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            container.bottomAnchor.constraint(equalTo: view.bottomAnchor),
        ])
        switchTo(index: 0)
    }

    private func makeSubTabButton(title: String, index: Int) -> UIButton {
        let btn = UIButton(type: .system)
        btn.setTitle(title, for: .normal)
        btn.titleLabel?.font = .systemFont(ofSize: 12, weight: .semibold)
        btn.titleLabel?.adjustsFontSizeToFitWidth = true
        btn.titleLabel?.minimumScaleFactor = 0.85
        btn.layer.cornerRadius = 8
        btn.tag = index
        btn.addTarget(self, action: #selector(subTabTapped(_:)), for: .touchUpInside)
        return btn
    }

    private func refreshSubTabs() {
        for (i, btn) in subTabButtons.enumerated() {
            let active = i == selectedIndex
            btn.setTitleColor(active ? .systemBlue : .tertiaryLabel, for: .normal)
            btn.titleLabel?.font = .systemFont(ofSize: 13, weight: active ? .bold : .medium)
            btn.backgroundColor = active ? UIColor.systemBlue.withAlphaComponent(0.10) : .clear
        }
    }

    @objc private func subTabTapped(_ sender: UIButton) {
        guard sender.tag != selectedIndex else { return }
        selectedIndex = sender.tag
        refreshSubTabs()
        switchTo(index: selectedIndex)
    }

    private func switchTo(index: Int) {
        currentVC?.willMove(toParent: nil)
        currentVC?.view.removeFromSuperview()
        currentVC?.removeFromParent()
        let vc: UIViewController
        switch index {
        case 1: vc = wechatVC
        case 2: vc = bindVC
        default: vc = yybVC
        }
        addChild(vc)
        vc.view.translatesAutoresizingMaskIntoConstraints = false
        container.addSubview(vc.view)
        NSLayoutConstraint.activate([
            vc.view.topAnchor.constraint(equalTo: container.topAnchor),
            vc.view.leadingAnchor.constraint(equalTo: container.leadingAnchor),
            vc.view.trailingAnchor.constraint(equalTo: container.trailingAnchor),
            vc.view.bottomAnchor.constraint(equalTo: container.bottomAnchor),
        ])
        vc.didMove(toParent: self)
        currentVC = vc
    }
}

final class ProtocolBindViewController: BaseNativeViewController, UITableViewDataSource, UITableViewDelegate {
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private var bindings: [PortalProtocolBinding] = []
    private var quota: PortalProtocolBindQuota?
    private var wxDevices: [PortalWxDevice] = []
    private var yybAccounts: [PortalYybAccount] = []
    private var selectedWx: PortalWxDevice?
    private var selectedYyb: PortalYybAccount?
    private var loading = false
    private let embedded: Bool

    init(embedded: Bool = false) {
        self.embedded = embedded
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) {
        self.embedded = false
        super.init(coder: coder)
    }

    private enum Section: Int, CaseIterable { case header = 0, form, bound }

    override func viewDidLoad() {
        super.viewDidLoad()
        if !embedded {
            title = "协议双绑"
            navigationItem.leftBarButtonItem = UIBarButtonItem(title: "关闭", style: .plain, target: self, action: #selector(closeTapped))
        }
        tableView.dataSource = self
        tableView.delegate = self
        if embedded {
            tableView.backgroundColor = .systemGroupedBackground
        }
        tableView.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(tableView)
        NSLayoutConstraint.activate([
            tableView.topAnchor.constraint(equalTo: view.topAnchor),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
        ])
        loadAll()
    }

    @objc private func closeTapped() {
        if let nav = navigationController, nav.viewControllers.first === self, nav.presentingViewController != nil {
            dismiss(animated: true)
        } else {
            navigationController?.popViewController(animated: true)
        }
    }

    private func loadAll() {
        loading = true
        tableView.reloadData()
        let group = DispatchGroup()

        group.enter()
        PortalService.shared.fetchProtocolBindings { [weak self] result in
            if case .success(let rows) = result { self?.bindings = rows }
            group.leave()
        }
        group.enter()
        PortalService.shared.fetchProtocolBindQuota { [weak self] result in
            if case .success(let q) = result { self?.quota = q }
            group.leave()
        }
        group.enter()
        PortalService.shared.fetchWxDevices { [weak self] result in
            if case .success(let rows) = result { self?.wxDevices = rows }
            group.leave()
        }
        group.enter()
        let cached = YybAccountStore.shared.accounts
        if !cached.isEmpty {
            yybAccounts = cached
            group.leave()
        } else {
            PortalService.shared.fetchYybStatus(autoCheck: false) { [weak self] result in
                if case .success(let st) = result { self?.yybAccounts = st.accounts ?? [] }
                group.leave()
            }
        }
        group.notify(queue: .main) { [weak self] in
            self?.loading = false
            self?.tableView.reloadData()
        }
    }

    func numberOfSections(in tableView: UITableView) -> Int { Section.allCases.count }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        switch Section(rawValue: section) {
        case .header: return 1
        case .form: return loading ? 0 : 3
        case .bound: return loading ? 0 : max(bindings.count, 1)
        default: return 0
        }
    }

    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        switch Section(rawValue: section) {
        case .form: return "新建双绑"
        case .bound: return "已绑定配对"
        default: return nil
        }
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        switch Section(rawValue: indexPath.section) {
        case .header:
            let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
            cell.selectionStyle = .none
            cell.textLabel?.text = "🔗 微信 wxid ↔ 应用宝 openid"
            cell.textLabel?.font = .systemFont(ofSize: 16, weight: .semibold)
            cell.textLabel?.numberOfLines = 0
            cell.detailTextLabel?.text = "绑定后青龙脚本原提交 CK 的微信 wxid，网关将自动路由微信协议 wxid 至应用宝请求协议 code；如果你之前没使用微信协议仅使用应用宝时可忽略本功能。绑定关系显示在下方账号卡片内。"
            cell.detailTextLabel?.numberOfLines = 0
            return cell
        case .form:
            let cell = UITableViewCell(style: .value1, reuseIdentifier: nil)
            switch indexPath.row {
            case 0:
                cell.textLabel?.text = "微信 wxid"
                cell.detailTextLabel?.text = selectedWx == nil ? "请选择" : (selectedWx?.nickname ?? shortenProtocolId(selectedWx?.wxid))
                cell.accessoryType = .disclosureIndicator
            case 1:
                cell.textLabel?.text = "应用宝 openid"
                cell.detailTextLabel?.text = selectedYyb == nil ? "请选择" : displayYybName(selectedYyb!)
                cell.accessoryType = .disclosureIndicator
            case 2:
                cell.textLabel?.text = "建立双绑"
                cell.textLabel?.textColor = .systemBlue
                cell.textLabel?.textAlignment = .center
                cell.accessoryType = .none
            default:
                break
            }
            return cell
        case .bound:
            if bindings.isEmpty {
                let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
                cell.selectionStyle = .none
                cell.textLabel?.text = "暂无绑定"
                cell.detailTextLabel?.text = "可在上方选择微信与应用宝账号建立双绑。"
                cell.detailTextLabel?.numberOfLines = 0
                return cell
            }
            let b = bindings[indexPath.row]
            let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
            let wx = wxDevices.first { $0.wxid == b.wxWxid }
            let yyb = yybAccounts.first { $0.openid == b.yybOpenId }
            cell.textLabel?.text = "\(wx?.nickname ?? "微信") ↔ \(yyb?.nickname ?? "应用宝")"
            cell.detailTextLabel?.text = "\(shortenProtocolId(b.wxWxid))  ·  \(shortenProtocolId(b.yybOpenId))"
            cell.detailTextLabel?.numberOfLines = 0
            cell.accessoryType = .none
            let unbind = UIButton(type: .system)
            unbind.setTitle("解绑", for: .normal)
            unbind.setTitleColor(.systemRed, for: .normal)
            unbind.titleLabel?.font = .systemFont(ofSize: 13, weight: .semibold)
            unbind.tag = indexPath.row
            unbind.addTarget(self, action: #selector(unbindTapped(_:)), for: .touchUpInside)
            unbind.sizeToFit()
            cell.accessoryView = unbind
            return cell
        default:
            return UITableViewCell()
        }
    }

    private func displayYybName(_ acc: PortalYybAccount) -> String {
        let nick = (acc.nickname ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        if !nick.isEmpty { return nick }
        return shortenProtocolId(acc.openid)
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        guard Section(rawValue: indexPath.section) == .form else { return }
        switch indexPath.row {
        case 0: showWxPicker()
        case 1: showYybPicker()
        case 2: submitBind()
        default: break
        }
    }

    private func boundWxSet() -> Set<String> { Set(bindings.compactMap { $0.wxWxid }.filter { !$0.isEmpty }) }
    private func boundOpenIdSet() -> Set<String> { Set(bindings.compactMap { $0.yybOpenId }.filter { !$0.isEmpty }) }

    private func showWxPicker() {
        let items = wxDevices.filter { let wx = $0.wxid ?? ""; return !wx.isEmpty && !boundWxSet().contains(wx) }
        guard !items.isEmpty else { showMessage("暂无可绑定的微信设备"); return }
        let sheet = UIAlertController(title: "选择微信", message: nil, preferredStyle: .actionSheet)
        for dev in items {
            let offline = dev.online == true ? "" : "（离线）"
            let title = "\(dev.nickname ?? "微信设备") · \(shortenProtocolId(dev.wxid))\(offline)"
            sheet.addAction(UIAlertAction(title: title, style: .default) { [weak self] _ in
                self?.selectedWx = dev
                self?.tableView.reloadSections(IndexSet(integer: Section.form.rawValue), with: .none)
            })
        }
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    private func showYybPicker() {
        let items = yybAccounts.filter { let oid = $0.openid ?? ""; return !oid.isEmpty && !boundOpenIdSet().contains(oid) }
        guard !items.isEmpty else { showMessage("暂无可绑定的应用宝账号"); return }
        let sheet = UIAlertController(title: "选择应用宝", message: nil, preferredStyle: .actionSheet)
        for acc in items {
            let alive = ((acc.status ?? "").lowercased() == "alive" || (acc.status ?? "").lowercased() == "online")
            let title = "\(displayYybName(acc)) · \(shortenProtocolId(acc.openid))\(alive ? "" : "（失效）")"
            sheet.addAction(UIAlertAction(title: title, style: .default) { [weak self] _ in
                self?.selectedYyb = acc
                self?.tableView.reloadSections(IndexSet(integer: Section.form.rawValue), with: .none)
            })
        }
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(sheet, animated: true)
    }

    private func submitBind() {
        guard let wx = selectedWx, let yyb = selectedYyb else {
            showMessage("请先选择微信和应用宝账号")
            return
        }
        PortalService.shared.protocolBind(
            wxWxid: wx.wxid ?? "",
            yybOpenId: yyb.openid ?? "",
            nickname: yyb.nickname ?? ""
        ) { [weak self] result in
            DispatchQueue.main.async {
                switch result {
                case .failure(let error):
                    self?.handle(error)
                case .success:
                    self?.showMessage("绑定成功")
                    self?.selectedWx = nil
                    self?.selectedYyb = nil
                    ProtocolBindStore.shared.reload()
                    self?.loadAll()
                }
            }
        }
    }

    @objc private func unbindTapped(_ sender: UIButton) {
        guard bindings.indices.contains(sender.tag) else { return }
        let b = bindings[sender.tag]
        let alert = UIAlertController(title: "解除双绑", message: "确定解除该配对？", preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "解绑", style: .destructive) { [weak self] _ in
            PortalService.shared.protocolUnbind(wxWxid: b.wxWxid ?? "", yybOpenId: b.yybOpenId ?? "") { result in
                DispatchQueue.main.async {
                    switch result {
                    case .failure(let error):
                        self?.handle(error)
                    case .success(let msg):
                        self?.showMessage(msg)
                        ProtocolBindStore.shared.reload()
                        self?.loadAll()
                    }
                }
            }
        })
        present(alert, animated: true)
    }
}


final class YybProtocolViewController: BaseNativeViewController {
    private let topStack = UIStackView()
    private let stickyHeaderRow = UIStackView()
    private let headerActionGroup = UIStackView()
    private let accountScrollView = UIScrollView()
    private let accountStack = UIStackView()
    private let sectionTitleLabel = UILabel()
    private let serviceDot = UIView()
    private let serviceLabel = UILabel()
    private let loadingIndicator = UIActivityIndicatorView(style: .medium)
    private let refreshAccountBtn = UIButton(type: .system)
    private let deleteAccountBtn = UIButton(type: .system)
    private var introBody = UILabel()
    private var introExpanded = false
    private var scanButton: UIButton!
    private var reloadButton: UIButton!
    private var accounts: [PortalYybAccount] = []
    private var selectedKey = ""
    private var pollTimer: Timer?
    private var scanning = false
    private var qrNav: UINavigationController?
    private var actionLockedUntil: Date = .distantPast
    private var didShowPendingAlert = false
    private var scanBusy = false
    private var checkBusy = false
    private var accountActionBusy = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemGroupedBackground
        setupUI()
        NotificationCenter.default.addObserver(self, selector: #selector(onStoreUpdated), name: AppNotifications.yybStatusDidUpdate, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(onProtocolBindUpdated), name: AppNotifications.protocolBindDidUpdate, object: nil)
        applyStore(showPendingAlert: false)
        if YybAccountStore.shared.status == nil, !YybAccountStore.shared.isLoading, !YybAccountStore.shared.sessionAutoChecked {
            YybAccountStore.shared.prefetchIfNeeded(autoCheck: false)
        }
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        applyStore(showPendingAlert: false)
        if !YybAccountStore.shared.sessionAutoChecked {
            YybAccountStore.shared.prefetchIfNeeded(autoCheck: false)
        }
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        pollTimer?.invalidate()
        pollTimer = nil
        scanning = false
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    private func setupUI() {
        topStack.axis = .vertical
        topStack.spacing = 14
        topStack.translatesAutoresizingMaskIntoConstraints = false

        stickyHeaderRow.axis = .horizontal
        stickyHeaderRow.alignment = .center
        stickyHeaderRow.spacing = 6
        stickyHeaderRow.translatesAutoresizingMaskIntoConstraints = false
        stickyHeaderRow.backgroundColor = .systemGroupedBackground

        accountScrollView.translatesAutoresizingMaskIntoConstraints = false
        accountScrollView.alwaysBounceVertical = true
        accountStack.axis = .vertical
        accountStack.spacing = 10
        accountStack.translatesAutoresizingMaskIntoConstraints = false

        view.addSubview(topStack)
        view.addSubview(stickyHeaderRow)
        view.addSubview(accountScrollView)
        accountScrollView.addSubview(accountStack)

        NSLayoutConstraint.activate([
            topStack.topAnchor.constraint(equalTo: view.topAnchor, constant: 10),
            topStack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            topStack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),

            stickyHeaderRow.topAnchor.constraint(equalTo: topStack.bottomAnchor, constant: 8),
            stickyHeaderRow.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stickyHeaderRow.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stickyHeaderRow.heightAnchor.constraint(equalToConstant: 36),

            accountScrollView.topAnchor.constraint(equalTo: stickyHeaderRow.bottomAnchor, constant: 4),
            accountScrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            accountScrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            accountScrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            accountStack.topAnchor.constraint(equalTo: accountScrollView.topAnchor, constant: 4),
            accountStack.leadingAnchor.constraint(equalTo: accountScrollView.leadingAnchor, constant: 16),
            accountStack.trailingAnchor.constraint(equalTo: accountScrollView.trailingAnchor, constant: -16),
            accountStack.bottomAnchor.constraint(equalTo: accountScrollView.bottomAnchor, constant: -24),
            accountStack.widthAnchor.constraint(equalTo: accountScrollView.widthAnchor, constant: -32),
        ])

        topStack.addArrangedSubview(buildIntroCard())

        let actionRow = UIStackView()
        actionRow.axis = .horizontal
        actionRow.spacing = 10
        actionRow.distribution = .fillEqually
        scanButton = makeActionButton(title: "扫码添加", style: .primary) { [weak self] in
            self?.startScan()
        }
        reloadButton = makeActionButton(title: "刷新检测", style: .secondary) { [weak self] in
            self?.reloadWithCheck()
        }
        actionRow.addArrangedSubview(scanButton)
        actionRow.addArrangedSubview(reloadButton)
        topStack.addArrangedSubview(actionRow)

        sectionTitleLabel.font = .systemFont(ofSize: 13, weight: .semibold)
        sectionTitleLabel.textColor = .secondaryLabel
        sectionTitleLabel.text = "账号列表 · 0"
        sectionTitleLabel.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        headerActionGroup.axis = .horizontal
        headerActionGroup.spacing = 6
        headerActionGroup.alignment = .center
        headerActionGroup.isHidden = true

        configureHeaderActionButton(refreshAccountBtn, title: "刷新", danger: false)
        configureHeaderActionButton(deleteAccountBtn, title: "删除", danger: true)
        refreshAccountBtn.addAction(UIAction { [weak self] _ in
            guard let self = self else { return }
            self.animateHeaderButton(self.refreshAccountBtn)
            self.refreshSelected()
        }, for: .touchUpInside)
        deleteAccountBtn.addAction(UIAction { [weak self] _ in
            guard let self = self else { return }
            self.animateHeaderButton(self.deleteAccountBtn)
            self.deleteSelected()
        }, for: .touchUpInside)
        headerActionGroup.addArrangedSubview(refreshAccountBtn)
        headerActionGroup.addArrangedSubview(deleteAccountBtn)

        serviceDot.translatesAutoresizingMaskIntoConstraints = false
        serviceDot.layer.cornerRadius = 3.5
        serviceDot.backgroundColor = .systemGray3

        serviceLabel.font = .systemFont(ofSize: 12, weight: .medium)
        serviceLabel.textColor = .tertiaryLabel
        serviceLabel.text = "待机"
        serviceLabel.setContentHuggingPriority(.required, for: .horizontal)

        loadingIndicator.hidesWhenStopped = true
        loadingIndicator.transform = CGAffineTransform(scaleX: 0.75, y: 0.75)

        let status = UIStackView(arrangedSubviews: [loadingIndicator, serviceDot, serviceLabel])
        status.axis = .horizontal
        status.spacing = 6
        status.alignment = .center
        status.setContentHuggingPriority(.required, for: .horizontal)

        let headerSpacer = UIView()
        headerSpacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
        headerSpacer.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        stickyHeaderRow.addArrangedSubview(sectionTitleLabel)
        stickyHeaderRow.addArrangedSubview(headerSpacer)
        stickyHeaderRow.addArrangedSubview(headerActionGroup)
        stickyHeaderRow.addArrangedSubview(status)

        NSLayoutConstraint.activate([
            serviceDot.widthAnchor.constraint(equalToConstant: 7),
            serviceDot.heightAnchor.constraint(equalToConstant: 7),
            refreshAccountBtn.widthAnchor.constraint(equalToConstant: 44),
            deleteAccountBtn.widthAnchor.constraint(equalToConstant: 44),
        ])
    }

    private func animateHeaderButton(_ button: UIButton) {
        UIView.animate(withDuration: 0.08, animations: {
            button.transform = CGAffineTransform(scaleX: 0.94, y: 0.94)
            button.alpha = 0.75
        }, completion: { _ in
            UIView.animate(withDuration: 0.12) {
                button.transform = .identity
                button.alpha = 1
            }
        })
    }

    private func configureHeaderActionButton(_ button: UIButton, title: String, danger: Bool) {
        button.setTitle(title, for: .normal)
        button.titleLabel?.font = .systemFont(ofSize: 12, weight: .semibold)
        button.layer.cornerRadius = 8
        button.contentEdgeInsets = UIEdgeInsets(top: 6, left: 4, bottom: 6, right: 4)
        button.setContentHuggingPriority(.required, for: .horizontal)
        button.setContentCompressionResistancePriority(.required, for: .horizontal)
        if danger {
            button.backgroundColor = UIColor.systemRed.withAlphaComponent(0.12)
            button.setTitleColor(.systemRed, for: .normal)
        } else {
            button.backgroundColor = UIColor.tertiarySystemFill
            button.setTitleColor(.label, for: .normal)
        }
    }

    private enum ActionStyle { case primary, secondary, danger, plain }

    private func makeActionButton(title: String, style: ActionStyle, action: @escaping () -> Void) -> UIButton {
        let btn = UIButton(type: .system)
        btn.setTitle(title, for: .normal)
        btn.titleLabel?.font = .systemFont(ofSize: 15, weight: .semibold)
        btn.layer.cornerRadius = 12
        btn.contentEdgeInsets = UIEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
        switch style {
        case .primary:
            btn.backgroundColor = .systemBlue
            btn.setTitleColor(.white, for: .normal)
        case .secondary:
            btn.backgroundColor = UIColor.secondarySystemGroupedBackground
            btn.setTitleColor(.label, for: .normal)
        case .danger:
            btn.backgroundColor = UIColor.systemRed.withAlphaComponent(0.12)
            btn.setTitleColor(.systemRed, for: .normal)
            btn.titleLabel?.font = .systemFont(ofSize: 13, weight: .semibold)
            btn.contentEdgeInsets = UIEdgeInsets(top: 8, left: 10, bottom: 8, right: 10)
        case .plain:
            btn.backgroundColor = UIColor.tertiarySystemFill
            btn.setTitleColor(.label, for: .normal)
            btn.titleLabel?.font = .systemFont(ofSize: 13, weight: .semibold)
            btn.contentEdgeInsets = UIEdgeInsets(top: 8, left: 10, bottom: 8, right: 10)
        }
        btn.addAction(UIAction { [weak self, weak btn] _ in
            guard let self = self, let btn = btn else { return }
            self.animatePress(btn)
            guard self.beginActionLock() else { return }
            action()
        }, for: .touchUpInside)
        return btn
    }

    private func buildIntroCard() -> UIView {
        let card = UIView()
        card.applyCardStyle()
        let titleRow = UIStackView()
        titleRow.axis = .horizontal
        titleRow.spacing = 8
        let icon = UILabel()
        icon.text = "📖"
        let title = UILabel()
        title.text = "什么是应用宝协议？"
        title.font = .systemFont(ofSize: 15, weight: .bold)
        let expand = UILabel()
        expand.text = introExpanded ? "▼" : "▶"
        expand.textColor = .secondaryLabel
        titleRow.addArrangedSubview(icon)
        titleRow.addArrangedSubview(title)
        titleRow.addArrangedSubview(UIView())
        titleRow.addArrangedSubview(expand)
        introBody.text = """
        应用宝协议通过提交应用宝 openid（owNAX 开头），向协议网关请求小程序登录凭证（CK）。无需保持微信长期在线，也没有封号风险。

        【主要作用】
        • 提交 openid 即可获取小程序 CK，适合青龙等自动化脚本
        • 扫码登录后有效期 30 天，到期前重新扫码可延期
        • 可单独使用应用宝协议，不必依赖微信协议

        【从微信协议迁移】
        若你此前使用微信协议，青龙脚本里提交的是微信 wxid 作为 CK：
        • 需先将 wxid 与对应的应用宝 openid 双绑
        • 绑定后脚本里仍填原 wxid，网关会自动路由到应用宝获取 code
        • 之后即使退出或删除微信协议设备，只要双绑关系保留，wxid 依然能路由到应用宝
        • 也可完全切换到应用宝，直接提交 openid 作为 CK
        """
        introBody.font = .systemFont(ofSize: 13)
        introBody.textColor = .secondaryLabel
        introBody.numberOfLines = 0
        introBody.isHidden = !introExpanded
        let stack = UIStackView(arrangedSubviews: [titleRow, introBody])
        stack.axis = .vertical
        stack.spacing = 6
        stack.translatesAutoresizingMaskIntoConstraints = false
        card.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: card.topAnchor, constant: 12),
            stack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 12),
            stack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -12),
            stack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -12),
        ])
        card.translatesAutoresizingMaskIntoConstraints = false
        let tap = UITapGestureRecognizer(target: self, action: #selector(toggleIntro))
        card.addGestureRecognizer(tap)
        card.isUserInteractionEnabled = true
        return card
    }

    @objc private func toggleIntro() {
        introExpanded.toggle()
        topStack.arrangedSubviews.first?.removeFromSuperview()
        topStack.insertArrangedSubview(buildIntroCard(), at: 0)
    }

    private func animatePress(_ button: UIButton) {
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        UIView.animate(withDuration: 0.08, animations: {
            button.transform = CGAffineTransform(scaleX: 0.96, y: 0.96)
            button.alpha = 0.85
        }, completion: { _ in
            UIView.animate(withDuration: 0.12) {
                button.transform = .identity
                button.alpha = 1
            }
        })
    }

    @discardableResult
    private func beginActionLock(seconds: TimeInterval = 1.2) -> Bool {
        let now = Date()
        if now < actionLockedUntil { return false }
        actionLockedUntil = now.addingTimeInterval(seconds)
        return true
    }

    @objc private func onProtocolBindUpdated() {
        renderAccounts()
    }

    @objc private func onStoreUpdated() {
        applyStore(showPendingAlert: isViewLoaded && view.window != nil)
    }

    private func applyStore(showPendingAlert: Bool) {
        let store = YybAccountStore.shared
        let ready = store.isServiceReady
        serviceDot.backgroundColor = ready ? .systemGreen : .systemOrange
        serviceLabel.text = store.serviceTitle
        serviceLabel.textColor = ready ? .tertiaryLabel : .systemOrange
        sectionTitleLabel.text = "账号列表 · \(store.accounts.count)"
        if checkBusy {
            loadingIndicator.startAnimating()
        } else {
            loadingIndicator.stopAnimating()
        }
        accounts = store.accounts
        if !accounts.contains(where: { accountKey($0) == selectedKey }) {
            selectedKey = accounts.first.map { accountKey($0) } ?? ""
        }
        renderAccounts()
        updateActionButtonsEnabled()

        if showPendingAlert, !didShowPendingAlert, let msg = store.consumePendingAlert() {
            didShowPendingAlert = true
            showMessage(msg, title: "检测完成")
        }
    }

    private func updateActionButtonsEnabled() {
        let ready = YybAccountStore.shared.isServiceReady || YybAccountStore.shared.status == nil
        let busy = scanBusy || checkBusy
        scanButton.isEnabled = ready && !busy
        scanButton.alpha = scanButton.isEnabled ? 1 : 0.5
        reloadButton.isEnabled = !busy
        reloadButton.alpha = reloadButton.isEnabled ? 1 : 0.5
        reloadButton.setTitle(checkBusy ? "检测中…" : "刷新检测", for: .normal)
    }

    private func reloadWithCheck() {
        checkBusy = true
        updateActionButtonsEnabled()
        loadingIndicator.startAnimating()
        YybAccountStore.shared.reload(autoCheck: true, showAlert: true) { [weak self] result in
            guard let self = self else { return }
            self.checkBusy = false
            if !YybAccountStore.shared.isLoading {
                self.loadingIndicator.stopAnimating()
            }
            self.updateActionButtonsEnabled()
            self.didShowPendingAlert = false
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success:
                if let msg = YybAccountStore.shared.consumePendingAlert() {
                    self.showMessage(msg, title: "检测完成")
                }
                self.applyStore(showPendingAlert: false)
            }
        }
    }

    private func accountKey(_ acc: PortalYybAccount) -> String {
        if let id = acc.bindingId, id > 0 { return String(id) }
        return acc.openid ?? ""
    }

    private func selectedAccount() -> PortalYybAccount? {
        accounts.first { accountKey($0) == selectedKey }
    }

    private func renderAccounts() {
        accountStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let hasSelection = selectedAccount() != nil
        headerActionGroup.isHidden = !hasSelection

        if accounts.isEmpty {
            let empty = UILabel()
            empty.text = (YybAccountStore.shared.isLoading && !YybAccountStore.shared.sessionAutoChecked)
                ? "正在同步账号…"
                : "暂无账号，点击上方「扫码添加」"
            empty.font = .systemFont(ofSize: 14)
            empty.textColor = .secondaryLabel
            empty.textAlignment = .center
            empty.numberOfLines = 0
            accountStack.addArrangedSubview(empty)
            return
        }

        for (index, acc) in accounts.enumerated() {
            accountStack.addArrangedSubview(buildAccountCard(acc, index: index))
        }
    }

    private func displayAccountName(_ acc: PortalYybAccount) -> String {
        let nick = (acc.nickname ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        if !nick.isEmpty { return nick }
        let oid = (acc.openid ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return oid.isEmpty ? "未命名" : oid
    }

    private func buildAccountCard(_ acc: PortalYybAccount, index: Int) -> UIView {
        let key = accountKey(acc)
        let selected = key == selectedKey
        let wrap = UIView()
        wrap.applyCardStyle(cornerRadius: 14)
        wrap.translatesAutoresizingMaskIntoConstraints = false
        wrap.layer.borderWidth = selected ? 1.5 : 0
        wrap.layer.borderColor = selected ? UIColor.systemBlue.cgColor : UIColor.clear.cgColor

        let name = UILabel()
        name.text = displayAccountName(acc)
        name.font = .systemFont(ofSize: 14, weight: .semibold)
        name.numberOfLines = 1
        name.lineBreakMode = .byTruncatingTail
        name.translatesAutoresizingMaskIntoConstraints = false

        let st = (acc.status ?? "").lowercased()
        let alive = st == "alive" || st == "online"
        let badge = UILabel()
        badge.text = alive ? "可用" : "失效"
        badge.font = .systemFont(ofSize: 10, weight: .bold)
        badge.textColor = alive ? .systemGreen : .systemRed
        badge.backgroundColor = (alive ? UIColor.systemGreen : UIColor.systemRed).withAlphaComponent(0.12)
        badge.layer.cornerRadius = 6
        badge.clipsToBounds = true
        badge.textAlignment = .center
        badge.translatesAutoresizingMaskIntoConstraints = false

        let meta = UILabel()
        let uinValue = (acc.uin ?? 0) > 0 ? "\(acc.uin!)" : "-"
        meta.text = "UIN \(uinValue)"
        meta.font = .systemFont(ofSize: 11)
        meta.textColor = .secondaryLabel
        meta.translatesAutoresizingMaskIntoConstraints = false

        let oidRow = UIStackView()
        oidRow.axis = .horizontal
        oidRow.spacing = 8
        oidRow.alignment = .center
        oidRow.translatesAutoresizingMaskIntoConstraints = false
        let oidLabel = UILabel()
        oidLabel.text = "OpenID"
        oidLabel.font = .systemFont(ofSize: 11, weight: .medium)
        oidLabel.textColor = .secondaryLabel
        let oid = UILabel()
        oid.text = acc.openid ?? "-"
        oid.font = .systemFont(ofSize: 11)
        oid.textColor = .tertiaryLabel
        oid.numberOfLines = 2
        oid.lineBreakMode = .byTruncatingMiddle
        oid.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        oidRow.addArrangedSubview(oidLabel)
        oidRow.addArrangedSubview(oid)

        let cardActionRow = UIStackView()
        cardActionRow.axis = .horizontal
        cardActionRow.spacing = 10
        cardActionRow.distribution = .fillEqually
        cardActionRow.translatesAutoresizingMaskIntoConstraints = false

        let copyBtn = makeCardActionButton(title: "复制 OpenID", tint: .systemBlue) {
            let text = (acc.openid ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
            guard !text.isEmpty else { return }
            UIPasteboard.general.string = text
            self.showMessage("已复制 OpenID")
        }
        cardActionRow.addArrangedSubview(copyBtn)

        wrap.addSubview(name)
        wrap.addSubview(badge)
        wrap.addSubview(meta)
        wrap.addSubview(oidRow)
        wrap.addSubview(cardActionRow)

        var actionTopAnchor: NSLayoutYAxisAnchor = oidRow.bottomAnchor

        if let binding = ProtocolBindStore.shared.binding(forOpenId: acc.openid) {
            let wxDev = ProtocolBindStore.shared.wxDevices.first { $0.wxid == binding.wxWxid }
            let peer = (wxDev?.nickname?.isEmpty == false ? wxDev?.nickname : nil) ?? binding.nickname ?? "微信设备"
            let boundLabel = UILabel()
            boundLabel.text = "已绑定微信 · \(peer) · \(shortenProtocolId(binding.wxWxid))"
            boundLabel.font = .systemFont(ofSize: 11)
            boundLabel.textColor = .secondaryLabel
            boundLabel.numberOfLines = 2
            boundLabel.translatesAutoresizingMaskIntoConstraints = false
            wrap.addSubview(boundLabel)
            NSLayoutConstraint.activate([
                boundLabel.topAnchor.constraint(equalTo: oidRow.bottomAnchor, constant: 6),
                boundLabel.leadingAnchor.constraint(equalTo: name.leadingAnchor),
                boundLabel.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            ])
            let unbindBtn = makeCardActionButton(title: "解除双绑", tint: .systemRed) { [weak self] in
                self?.confirmProtocolUnbind(wxWxid: binding.wxWxid ?? "", yybOpenId: binding.yybOpenId ?? "")
            }
            cardActionRow.addArrangedSubview(unbindBtn)
            actionTopAnchor = boundLabel.bottomAnchor
        }

        let expiry = formatYybExpiryText(acc)
        if !expiry.text.isEmpty {
            let expiryLabel = UILabel()
            expiryLabel.text = expiry.text
            expiryLabel.font = .systemFont(ofSize: 11)
            expiryLabel.numberOfLines = 0
            if expiry.expired {
                expiryLabel.textColor = .systemRed
            } else {
                expiryLabel.textColor = UIColor(red: 37 / 255, green: 99 / 255, blue: 235 / 255, alpha: 1)
                if expiry.warn {
                    expiryLabel.font = .boldSystemFont(ofSize: 11)
                }
            }
            expiryLabel.translatesAutoresizingMaskIntoConstraints = false
            wrap.addSubview(expiryLabel)
            NSLayoutConstraint.activate([
                expiryLabel.topAnchor.constraint(equalTo: actionTopAnchor, constant: 6),
                expiryLabel.leadingAnchor.constraint(equalTo: name.leadingAnchor),
                expiryLabel.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            ])
            actionTopAnchor = expiryLabel.bottomAnchor
        }

        NSLayoutConstraint.activate([
            wrap.heightAnchor.constraint(greaterThanOrEqualToConstant: 96),
            name.topAnchor.constraint(equalTo: wrap.topAnchor, constant: 12),
            name.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 12),
            name.trailingAnchor.constraint(lessThanOrEqualTo: badge.leadingAnchor, constant: -6),
            badge.centerYAnchor.constraint(equalTo: name.centerYAnchor),
            badge.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -10),
            badge.widthAnchor.constraint(greaterThanOrEqualToConstant: 36),
            badge.heightAnchor.constraint(equalToConstant: 20),
            meta.topAnchor.constraint(equalTo: name.bottomAnchor, constant: 6),
            meta.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            meta.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            oidRow.topAnchor.constraint(equalTo: meta.bottomAnchor, constant: 4),
            oidRow.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            oidRow.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            cardActionRow.topAnchor.constraint(equalTo: actionTopAnchor, constant: 10),
            cardActionRow.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            cardActionRow.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            cardActionRow.bottomAnchor.constraint(equalTo: wrap.bottomAnchor, constant: -12),
        ])

        let tap = UITapGestureRecognizer(target: self, action: #selector(accountTapped(_:)))
        tap.delegate = self
        wrap.isUserInteractionEnabled = true
        wrap.tag = index
        wrap.addGestureRecognizer(tap)
        return wrap
    }

    private func makeCardActionButton(title: String, tint: UIColor, action: @escaping () -> Void) -> UIButton {
        let btn = UIButton(type: .system)
        btn.setTitle(title, for: .normal)
        btn.titleLabel?.font = .systemFont(ofSize: 11, weight: .semibold)
        btn.setTitleColor(tint, for: .normal)
        btn.backgroundColor = tint.withAlphaComponent(0.10)
        btn.layer.cornerRadius = 8
        btn.contentEdgeInsets = UIEdgeInsets(top: 8, left: 10, bottom: 8, right: 10)
        btn.addAction(UIAction { _ in action() }, for: .touchUpInside)
        return btn
    }

    override func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldReceive touch: UITouch) -> Bool {
        if touch.view is UIButton || touch.view?.superview is UIButton {
            return false
        }
        return super.gestureRecognizer(gestureRecognizer, shouldReceive: touch)
    }

    @objc private func accountTapped(_ gesture: UITapGestureRecognizer) {
        guard let view = gesture.view, accounts.indices.contains(view.tag) else { return }
        selectedKey = accountKey(accounts[view.tag])
        renderAccounts()
    }

    private func confirmProtocolUnbind(wxWxid: String, yybOpenId: String) {
        let alert = UIAlertController(title: "解除双绑", message: "确定解除该微信与应用宝账号的双绑关系？", preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "解绑", style: .destructive) { [weak self] _ in
            PortalService.shared.protocolUnbind(wxWxid: wxWxid, yybOpenId: yybOpenId) { result in
                DispatchQueue.main.async {
                    switch result {
                    case .failure(let error):
                        self?.handle(error)
                    case .success(let msg):
                        self?.showMessage(msg)
                        ProtocolBindStore.shared.reload(yybAccounts: self?.accounts ?? [])
                    }
                }
            }
        })
        present(alert, animated: true)
    }

    private static func yybScanCostNote(cost: Int?, hint: String?) -> String? {
        formatYybScanCostNote(cost: cost, hint: hint)
    }

    private func startScan() {
        if scanBusy || checkBusy { return }
        scanBusy = true
        updateActionButtonsEnabled()
        PortalService.shared.fetchProtocolProxyConfig { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.scanBusy = false
                self.updateActionButtonsEnabled()
                switch result {
                case .failure:
                    self.beginYybQrScan(regionCode: "", regionName: "", useProxy: false, packId: "")
                case .success(let cfg):
                    if cfg.proxyEnabled == true {
                        self.presentRegionPicker(config: cfg)
                    } else {
                        self.beginYybQrScan(regionCode: "", regionName: "", useProxy: false, packId: cfg.proxyDefaultPackid ?? "")
                    }
                }
            }
        }
    }

    private func presentRegionPicker(config: PortalProxyConfig) {
        let packId = config.proxyDefaultPackid ?? ""
        PortalService.shared.fetchProtocolBindQuota { [weak self] quotaResult in
            PortalService.shared.fetchProtocolProxyAreas(packId: packId) { [weak self] result in
                DispatchQueue.main.async {
                    guard let self = self else { return }
                    switch result {
                    case .failure(let error):
                        self.showMessage(error.message)
                        return
                    case .success(let data):
                        let provinces = parseProxyAreaRows(data)
                        guard !provinces.isEmpty else {
                            self.showMessage("地区列表为空，请稍后重试")
                            return
                        }
                    let quota = try? quotaResult.get()
                    let costPreview = formatScanCostPreview(cost: quota?.scanLoginCost, hint: quota?.scanCostHint)
                    var message = {
                        if let bypass = config.proxyBypassRegionName, !bypass.isEmpty {
                            return "「\(bypass)」等地区免代理直连；其他地区请选择与你所在地一致的省/市。异地登录可能只有1天有效期。"
                        }
                        return "请选择与你当前所在地一致的省/市。异地登录可能只有1天有效期。"
                    }()
                    if !costPreview.text.isEmpty {
                        message += "\n\n" + costPreview.text
                    }
                    let sheet = UIAlertController(title: "选择登录地区", message: message, preferredStyle: .actionSheet)
                for province in provinces.prefix(20) {
                    sheet.addAction(UIAlertAction(title: province.name, style: .default) { _ in
                        self.pickCityAndScan(province: province, packId: packId, config: config)
                    })
                }
                sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
                self.present(sheet, animated: true)
                    }
                }
            }
        }
    }

    private func pickCityAndScan(province: (code: String, name: String), packId: String, config: PortalProxyConfig) {
        PortalService.shared.fetchProtocolProxyAreas(parentCode: province.code, packId: packId) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                switch result {
                case .failure(let error):
                    self.showMessage(error.message)
                case .success(let data):
                    let cities = parseProxyAreaRows(data)
                    guard !cities.isEmpty else {
                        self.showMessage("城市列表为空，请稍后重试")
                        return
                    }
                let sheet = UIAlertController(title: "选择城市", message: province.name, preferredStyle: .actionSheet)
                for city in cities.prefix(30) {
                    sheet.addAction(UIAlertAction(title: city.name, style: .default) { _ in
                        self.beginYybQrScan(regionCode: city.code, regionName: city.name, useProxy: true, packId: packId)
                    })
                }
                sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
                self.present(sheet, animated: true)
                }
            }
        }
    }

    private func beginYybQrScan(regionCode: String, regionName: String, useProxy: Bool, packId: String) {
        scanBusy = true
        updateActionButtonsEnabled()
        PortalService.shared.createYybQr(regionCode: regionCode, regionName: regionName, useProxy: useProxy, packId: packId) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.scanBusy = false
                self.updateActionButtonsEnabled()
                switch result {
                case .failure(let error):
                    self.handle(error)
                case .success(let data):
                    guard let sessionId = data.sessionId, let image = data.imageBase64, !image.isEmpty else {
                        self.showMessage(data.scanCostHint ?? "二维码生成失败")
                        return
                    }
                    let vc = WechatQRCodeViewController(
                        base64String: image,
                        message: "请使用微信扫码确认登录",
                        costNote: Self.yybScanCostNote(cost: data.scanLoginCost, hint: data.scanCostHint)
                    ) { [weak self] in
                        self?.scanning = false
                        self?.pollTimer?.invalidate()
                        self?.pollTimer = nil
                    }
                    let nav = UINavigationController(rootViewController: vc)
                    self.qrNav = nav
                    self.present(nav, animated: true)
                    self.scanning = true
                    self.pollScan(sessionId: sessionId, qrVC: vc)
                }
            }
        }
    }

    private func pollScan(sessionId: String, qrVC: WechatQRCodeViewController) {
        pollTimer?.invalidate()
        pollTimer = Timer.scheduledTimer(withTimeInterval: 0.8, repeats: true) { [weak self] _ in
            guard let self = self, self.scanning else { return }
            PortalService.shared.pollYybQr(sessionId: sessionId) { [weak self] result in
                DispatchQueue.main.async {
                    guard let self = self, self.scanning else { return }
                    switch result {
                    case .failure(let error):
                        let msg = error.message
                        if msg.lowercased().contains("deadline") || msg.lowercased().contains("timeout") {
                            qrVC.updateState("等待扫码中（网络较慢）…")
                            return
                        }
                        self.scanning = false
                        self.pollTimer?.invalidate()
                        qrVC.updateState(msg)
                    case .success(let data):
                        let status = (data.status ?? "").lowercased()
                        switch status {
                        case "scanned":
                            qrVC.updateState("已扫码，请在手机上点击「确认登录」")
                        case "authorized", "confirmed":
                            self.scanning = false
                            self.pollTimer?.invalidate()
                            qrVC.updateState("正在完成绑定…")
                            PortalService.shared.confirmYybQr(sessionId: sessionId) { [weak self] confirmResult in
                                DispatchQueue.main.async {
                                    guard let self = self else { return }
                                    self.qrNav?.dismiss(animated: true)
                                    switch confirmResult {
                                    case .failure(let error):
                                        self.handle(error)
                                    case .success(let conf):
                                        let msg = formatYybConfirmMessage(
                                            alreadyBound: conf.alreadyBound == true,
                                            cost: conf.cost ?? 0
                                        )
                                        self.showMessage(msg)
                                        self.didShowPendingAlert = false
                                        YybAccountStore.shared.reload(autoCheck: true, showAlert: true) { [weak self] _ in
                                            self?.didShowPendingAlert = false
                                            if let alert = YybAccountStore.shared.consumePendingAlert() {
                                                self?.showMessage(alert, title: "检测完成")
                                            }
                                            self?.applyStore(showPendingAlert: false)
                                        }
                                    }
                                }
                            }
                        case "expired", "cancelled", "unknown":
                            self.scanning = false
                            self.pollTimer?.invalidate()
                            qrVC.updateState(status == "expired" ? "二维码已过期，请重新生成" : "扫码已取消")
                        default:
                            break
                        }
                    }
                }
            }
        }
    }

    private func refreshSelected() {
        guard !accountActionBusy else { return }
        guard let acc = selectedAccount() else {
            showMessage("请先选择一个账号")
            return
        }
        accountActionBusy = true
        refreshAccountBtn.isEnabled = false
        let ref = accountKey(acc)
        PortalService.shared.refreshYybAccount(ref: ref) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.accountActionBusy = false
                self.refreshAccountBtn.isEnabled = true
                switch result {
                case .failure(let error):
                    self.handle(error)
                case .success:
                    self.showMessage("存活状态已刷新")
                    YybAccountStore.shared.reload(autoCheck: false, showAlert: false) { [weak self] _ in
                        self?.applyStore(showPendingAlert: false)
                    }
                }
            }
        }
    }

    private func deleteSelected() {
        guard let acc = selectedAccount() else {
            showMessage("请先选择一个账号")
            return
        }
        let alert = UIAlertController(title: "删除账号", message: "确定删除该应用宝账号？", preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "删除", style: .destructive) { [weak self] _ in
            guard let self = self else { return }
            PortalService.shared.deleteYybAccount(ref: self.accountKey(acc)) { result in
                DispatchQueue.main.async {
                    switch result {
                    case .failure(let error):
                        self.handle(error)
                    case .success(let msg):
                        self.showMessage(msg)
                        self.selectedKey = ""
                        YybAccountStore.shared.reload(autoCheck: false, showAlert: false) { [weak self] _ in
                            self?.applyStore(showPendingAlert: false)
                        }
                    }
                }
            }
        })
        present(alert, animated: true)
    }
}


final class WechatProtocolViewController: BaseNativeViewController {
    private let statusLabel = UILabel()
    private let detailLabel = UILabel()
    private let deviceContainer = UIStackView()
    private let refreshDeviceBtn = UIButton(type: .system)
    private var pollingTimer: Timer?
    private var autoRefreshTimer: Timer?
    private var qrModalNavigationController: UINavigationController?
    private var currentUUID: String?
    private var currentDeductCoin = false
    private var pollFailureCount = 0
    private var lastKnownStatus: String?
    private var introExpanded = false
    private let introBody = UILabel()

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemGroupedBackground
        setupUI()
        startAutoRefresh()
    }

    deinit {
        pollingTimer?.invalidate()
        autoRefreshTimer?.invalidate()
    }

    override func resetToInitialState() {
        guard isViewLoaded else { return }
        introExpanded = false
        introBody.isHidden = true
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        loadStatus()
        loadDevices()
        ProtocolBindStore.shared.reload()
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
        let introCard = UIView()
        introCard.applyCardStyle()
        let introTitleRow = UIView()
        let introIcon = UILabel()
        introIcon.text = "📖"
        introIcon.font = .systemFont(ofSize: 16)
        let introTitle = UILabel()
        introTitle.text = "什么是微信协议？"
        introTitle.font = .systemFont(ofSize: 15, weight: .bold)
        introTitle.textColor = .label
        let expandIcon = UILabel()
        expandIcon.text = "▸"
        expandIcon.font = .systemFont(ofSize: 14)
        expandIcon.textColor = .secondaryLabel
        introTitleRow.addSubview(introIcon)
        introTitleRow.addSubview(introTitle)
        introTitleRow.addSubview(expandIcon)
        introIcon.translatesAutoresizingMaskIntoConstraints = false
        introTitle.translatesAutoresizingMaskIntoConstraints = false
        expandIcon.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            introIcon.leadingAnchor.constraint(equalTo: introTitleRow.leadingAnchor),
            introIcon.centerYAnchor.constraint(equalTo: introTitleRow.centerYAnchor),
            introIcon.widthAnchor.constraint(equalToConstant: 24),
            introTitle.leadingAnchor.constraint(equalTo: introIcon.trailingAnchor, constant: 4),
            introTitle.centerYAnchor.constraint(equalTo: introTitleRow.centerYAnchor),
            expandIcon.trailingAnchor.constraint(equalTo: introTitleRow.trailingAnchor),
            expandIcon.centerYAnchor.constraint(equalTo: introTitleRow.centerYAnchor),
            introTitleRow.heightAnchor.constraint(equalToConstant: 24)
        ])
        introTitleRow.isUserInteractionEnabled = true
        introTitleRow.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(toggleIntro)))

        introBody.text = "微信协议是一种自动化工具，主要用于获取微信小程序的登录凭证（CK）。\n\n【主要作用】\n\u{2022} 自动获取小程序CK，无需手动抓包\n\u{2022} CK通常有有效期，配合微信协议可保持永不过期\n\u{2022} 支持微信协议的项目，只需提交微信ID即可自动上车\n\n【使用场景】\n如果某个项目标注「支持微信协议」，你只需要：\n1. 在此页面扫码或提交微信ID绑定设备\n2. 在项目中心选择支持微信协议的项目上车\n3. 系统会自动使用你的微信身份完成任务"
        introBody.font = .systemFont(ofSize: 13)
        introBody.textColor = .secondaryLabel
        introBody.numberOfLines = 0
        introBody.isHidden = true

        let introStack = UIStackView(arrangedSubviews: [introTitleRow, introBody])
        introStack.axis = .vertical
        introStack.spacing = 4
        introStack.translatesAutoresizingMaskIntoConstraints = false
        introCard.addSubview(introStack)
        NSLayoutConstraint.activate([
            introStack.topAnchor.constraint(equalTo: introCard.topAnchor, constant: 8),
            introStack.leadingAnchor.constraint(equalTo: introCard.leadingAnchor, constant: 10),
            introStack.trailingAnchor.constraint(equalTo: introCard.trailingAnchor, constant: -10),
            introStack.bottomAnchor.constraint(equalTo: introCard.bottomAnchor, constant: -8)
        ])
        introCard.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(introCard)

        let scrollView = UIScrollView()
        let mainStack = UIStackView()
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        mainStack.axis = .vertical
        mainStack.spacing = 4
        mainStack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scrollView)
        scrollView.addSubview(mainStack)
        NSLayoutConstraint.activate([
            introCard.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 2),
            introCard.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            introCard.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            scrollView.topAnchor.constraint(equalTo: introCard.bottomAnchor, constant: 4),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            mainStack.topAnchor.constraint(equalTo: scrollView.topAnchor),
            mainStack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: 16),
            mainStack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -16),
            mainStack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -12),
            mainStack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -32)
        ])

        let deviceLabel = UILabel()
        deviceLabel.text = "📱 监控设备"
        deviceLabel.font = .systemFont(ofSize: 17, weight: .bold)
        refreshDeviceBtn.setTitle("刷新设备列表", for: .normal)
        refreshDeviceBtn.titleLabel?.font = .systemFont(ofSize: 13, weight: .medium)
        refreshDeviceBtn.addTarget(self, action: #selector(loadDevices), for: .touchUpInside)
        let deviceHeader = UIStackView(arrangedSubviews: [deviceLabel, UIView(), refreshDeviceBtn])
        deviceHeader.axis = .horizontal
        deviceContainer.axis = .vertical
        deviceContainer.spacing = 6
        let deviceStack = UIStackView(arrangedSubviews: [deviceHeader, deviceContainer])
        deviceStack.axis = .vertical
        deviceStack.spacing = 6
        mainStack.addArrangedSubview(deviceStack)

        let gridLabel = UILabel()
        gridLabel.text = "快捷操作"
        gridLabel.font = .systemFont(ofSize: 17, weight: .bold)
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
        gridStack.spacing = 6
        mainStack.addArrangedSubview(gridStack)
        deleteActionBtn.heightAnchor.constraint(equalToConstant: 68).isActive = true
    }

    @objc private func toggleIntro() {
        introExpanded.toggle()
        introBody.isHidden = !introExpanded
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

    @objc private func loadDevices() {
        refreshDeviceBtn.isEnabled = false
        refreshDeviceBtn.setTitle("正在刷新...", for: .normal)
        PortalService.shared.fetchWxDevices { [weak self] result in
            self?.refreshDeviceBtn.isEnabled = true
            self?.refreshDeviceBtn.setTitle("✅ 已刷新", for: .normal)
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) {
                self?.refreshDeviceBtn.setTitle("刷新设备列表", for: .normal)
            }
            switch result {
            case .failure: break
            case .success(let devices):
                ProtocolBindStore.shared.reload { [weak self] in
                    self?.renderDevices(devices)
                }
            }
        }
    }

    private func renderDevices(_ devices: [PortalWxDevice]) {
        deviceContainer.arrangedSubviews.forEach { $0.removeFromSuperview() }
        if devices.isEmpty {
            let emptyLabel = UILabel()
            emptyLabel.text = "暂无微信协议设备\n首次使用请点击上方「扫码登录」添加主设备"
            emptyLabel.numberOfLines = 0
            emptyLabel.textAlignment = .center
            emptyLabel.font = .systemFont(ofSize: 13)
            emptyLabel.textColor = .secondaryLabel
            deviceContainer.addArrangedSubview(emptyLabel)
            return
        }
        for device in devices {
            deviceContainer.addArrangedSubview(makeDeviceCard(device))
        }
    }

    private func makeDeviceCard(_ device: PortalWxDevice) -> UIView {
        let card = UIView()
        card.applyCardStyle(cornerRadius: 16)
        if device.isPrimary == true {
            card.layer.borderWidth = 1
            card.layer.borderColor = UIColor.systemBlue.withAlphaComponent(0.3).cgColor
        }

        let isOnline = device.online == true
        let isPrimary = device.isPrimary == true

        var badges: [String] = []
        if isPrimary { badges.append("⭐ 主设备") }
        else { badges.append("📋 监控") }
        badges.append(isOnline ? "🟢 在线" : "🔴 离线")

        let badgesText = badges.joined(separator: "  ")
        let badgesLabel = UILabel()
        badgesLabel.text = badgesText
        badgesLabel.font = .systemFont(ofSize: 12, weight: .medium)
        badgesLabel.textColor = isOnline ? .systemGreen : .systemRed

        let nameLabel = UILabel()
        nameLabel.text = device.nickname ?? device.wxid ?? "未知设备"
        nameLabel.font = .systemFont(ofSize: 16, weight: .semibold)

        let infoLabel = UILabel()
        infoLabel.text = "设备：\(device.device ?? "-")\n微信ID：\(device.wxid ?? "-")\n登录时间：\(device.loginTime ?? "-")"
        infoLabel.font = .systemFont(ofSize: 12)
        infoLabel.textColor = .secondaryLabel
        infoLabel.numberOfLines = 0

        let btnRow1 = UIStackView()
        btnRow1.axis = .horizontal
        btnRow1.spacing = 8
        btnRow1.distribution = .fillEqually
        let btnRow2 = UIStackView()
        btnRow2.axis = .horizontal
        btnRow2.spacing = 8
        btnRow2.distribution = .fillEqually

        let deviceName = device.nickname ?? device.wxid ?? ""
        [("唤醒", "bell.fill", "/api/portal/wx/wake-login"), ("重登", "arrow.clockwise", "/api/portal/wx/relogin")].forEach { (title, icon, path) in
            let btn = UIButton(type: .system)
            btn.setTitle(" \(title)", for: .normal)
            btn.setImage(UIImage(systemName: icon), for: .normal)
            btn.titleLabel?.font = .systemFont(ofSize: 12, weight: .medium)
            btn.backgroundColor = .secondarySystemBackground
            btn.layer.cornerRadius = 8
            btn.heightAnchor.constraint(equalToConstant: 36).isActive = true
            btn.addAction(UIAction { [weak self] _ in
                self?.confirmWxActionForDevice(title: title, message: "确认对设备「\(deviceName)」执行\(title)？", path: path, wxid: device.wxid)
            }, for: .touchUpInside)
            btnRow1.addArrangedSubview(btn)
        }
        let logoutBtn = UIButton(type: .system)
        logoutBtn.setTitle(" 登出", for: .normal)
        logoutBtn.setImage(UIImage(systemName: "power"), for: .normal)
        logoutBtn.titleLabel?.font = .systemFont(ofSize: 12, weight: .medium)
        logoutBtn.backgroundColor = .secondarySystemBackground
        logoutBtn.layer.cornerRadius = 8
        logoutBtn.heightAnchor.constraint(equalToConstant: 36).isActive = true
        logoutBtn.addAction(UIAction { [weak self] _ in
            self?.confirmWxActionForDevice(title: "登出", message: "确认对设备「\(deviceName)」执行登出？\n\n登出不扣积分，可随时重新登录。", path: "/api/portal/wx/logout", wxid: device.wxid)
        }, for: .touchUpInside)
        btnRow2.addArrangedSubview(logoutBtn)

        let removeBtn = UIButton(type: .system)
        removeBtn.setTitle(" 移除", for: .normal)
        removeBtn.setImage(UIImage(systemName: "trash"), for: .normal)
        removeBtn.titleLabel?.font = .systemFont(ofSize: 12, weight: .medium)
        removeBtn.setTitleColor(.systemRed, for: .normal)
        removeBtn.backgroundColor = .secondarySystemBackground
        removeBtn.layer.cornerRadius = 8
        removeBtn.heightAnchor.constraint(equalToConstant: 36).isActive = true
        removeBtn.addAction(UIAction { [weak self] _ in
            let alert = UIAlertController(title: "确认移除", message: "确认移除设备「\(deviceName)」？\n\n⚠️ 移除后需要重新扫码登录，将再次扣除积分。", preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "取消", style: .cancel))
            alert.addAction(UIAlertAction(title: "确认移除", style: .destructive) { _ in
                PortalService.shared.removeWxDevice(id: device.id) { result in
                    if case .failure(let error) = result { self?.handle(error) }
                    else { self?.loadDevices() }
                }
            })
            self?.present(alert, animated: true)
        }, for: .touchUpInside)
        btnRow2.addArrangedSubview(removeBtn)

        let btnStack = UIStackView(arrangedSubviews: [btnRow1, btnRow2])
        btnStack.axis = .vertical
        btnStack.spacing = 8

        var stackViews: [UIView] = [badgesLabel, nameLabel, infoLabel]
        if let binding = ProtocolBindStore.shared.binding(forWxid: device.wxid) {
            let yybAcc = YybAccountStore.shared.accounts.first { $0.openid == binding.yybOpenId }
            let peer = (yybAcc?.nickname?.isEmpty == false ? yybAcc?.nickname : nil) ?? "应用宝账号"
            let boundRow = UIStackView()
            boundRow.axis = .horizontal
            boundRow.spacing = 8
            let boundLabel = UILabel()
            boundLabel.text = "已绑定应用宝 · \(peer) · \(shortenProtocolId(binding.yybOpenId))"
            boundLabel.font = .systemFont(ofSize: 11)
            boundLabel.textColor = .secondaryLabel
            boundLabel.numberOfLines = 2
            let unbindBtn = UIButton(type: .system)
            unbindBtn.setTitle("解绑", for: .normal)
            unbindBtn.titleLabel?.font = .systemFont(ofSize: 11, weight: .semibold)
            unbindBtn.setTitleColor(.systemRed, for: .normal)
            unbindBtn.addAction(UIAction { [weak self] _ in
                self?.confirmProtocolUnbind(wxWxid: binding.wxWxid ?? "", yybOpenId: binding.yybOpenId ?? "")
            }, for: .touchUpInside)
            boundRow.addArrangedSubview(boundLabel)
            boundRow.addArrangedSubview(unbindBtn)
            stackViews.append(boundRow)
        }
        stackViews.append(btnStack)

        let innerStack = UIStackView(arrangedSubviews: stackViews)
        innerStack.axis = .vertical
        innerStack.spacing = 8
        innerStack.translatesAutoresizingMaskIntoConstraints = false
        card.addSubview(innerStack)
        NSLayoutConstraint.activate([
            innerStack.topAnchor.constraint(equalTo: card.topAnchor, constant: 14),
            innerStack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 14),
            innerStack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -14),
            innerStack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -14)
        ])
        return card
    }

    private func confirmProtocolUnbind(wxWxid: String, yybOpenId: String) {
        let alert = UIAlertController(title: "解除双绑", message: "确定解除该微信与应用宝账号的双绑关系？", preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "解绑", style: .destructive) { [weak self] _ in
            PortalService.shared.protocolUnbind(wxWxid: wxWxid, yybOpenId: yybOpenId) { result in
                DispatchQueue.main.async {
                    switch result {
                    case .failure(let error):
                        self?.handle(error)
                    case .success(let msg):
                        self?.showMessage(msg)
                        ProtocolBindStore.shared.reload { self?.loadDevices() }
                    }
                }
            }
        })
        present(alert, animated: true)
    }

    private func confirmWxActionForDevice(title: String, message: String, path: String, wxid: String?) {
        let alert = UIAlertController(title: title, message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "确认", style: .default) { _ in
            if let wxid = wxid {
                self.performWxActionForDevice(path: path, wxid: wxid)
            } else {
                self.performAction(path: path, deductCoin: false)
            }
        })
        present(alert, animated: true)
    }

    private func performWxActionForDevice(path: String, wxid: String) {
        PortalService.shared.performWechatActionForDevice(path: path, wxid: wxid) { [weak self] result in
            switch result {
            case .failure(let error): self?.handle(error)
            case .success(let data):
                if let qr = data.qrBase64, let uuid = data.uuid {
                    self?.currentUUID = uuid
                    self?.currentDeductCoin = false
                    let vc = WechatQRCodeViewController(base64String: qr, message: data.message ?? "请扫码", onDismiss: { [weak self] in
                        self?.stopPolling()
                        self?.qrModalNavigationController = nil
                    })
                    let nav = AppNavigationController(rootViewController: vc)
                    nav.modalPresentationStyle = .formSheet
                    self?.qrModalNavigationController = nav
                    self?.present(nav, animated: true) {
                        self?.startPolling()
                    }
                } else {
                    self?.showMessage(data.message ?? "操作成功")
                    self?.loadStatus()
                    self?.loadDevices()
                }
            }
        }
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
                    self.loadDevices()
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
                self.loadDevices()
                AppSessionStore.shared.refreshIfPossible(silent: true)
            }
        }
    }
}


final class WechatQRCodeViewController: BaseNativeViewController {
    private let base64String: String
    private let messageText: String
    private let costNote: String?
    private let onDismiss: (() -> Void)?
    private let imageView = UIImageView()
    private let stateLabel = UILabel()
    private let costLabel = UILabel()

    init(base64String: String, message: String, costNote: String? = nil, onDismiss: (() -> Void)? = nil) {
        self.base64String = base64String
        self.messageText = message
        self.costNote = costNote
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

        let titleLabel = UILabel()
        titleLabel.text = "扫码登录"
        titleLabel.font = .systemFont(ofSize: 20, weight: .bold)

        let subtitleLabel = UILabel()
        subtitleLabel.text = "请用微信扫码，完成后页面会自动刷新状态"
        subtitleLabel.font = .systemFont(ofSize: 14)
        subtitleLabel.textColor = .secondaryLabel

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

        costLabel.translatesAutoresizingMaskIntoConstraints = false
        if let note = costNote, !note.isEmpty {
            costLabel.text = note
            costLabel.isHidden = false
        } else {
            costLabel.text = nil
            costLabel.isHidden = true
        }
        costLabel.numberOfLines = 0
        costLabel.textAlignment = .center
        costLabel.font = .systemFont(ofSize: 14, weight: .semibold)
        costLabel.textColor = .systemOrange

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

        // iPad适配：使用ScrollView包裹内容，防止遮挡
        let scrollView = UIScrollView()
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        scrollView.alwaysBounceVertical = true
        view.addSubview(scrollView)

        let contentStack = UIStackView(arrangedSubviews: [titleLabel, subtitleLabel, imageView, costLabel, stateLabel, hintLabel])
        contentStack.axis = .vertical
        contentStack.alignment = .center
        contentStack.spacing = 0
        contentStack.translatesAutoresizingMaskIntoConstraints = false
        scrollView.addSubview(contentStack)

        // 二维码尺寸自适应：取屏幕宽度的50%，最大260
        let qrSize: CGFloat = min(UIScreen.main.bounds.width * 0.5, 260)

        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),

            contentStack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 20),
            contentStack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: 20),
            contentStack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -20),
            contentStack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -20),
            contentStack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -40),

            titleLabel.leadingAnchor.constraint(equalTo: contentStack.leadingAnchor),

            subtitleLabel.leadingAnchor.constraint(equalTo: contentStack.leadingAnchor),
            subtitleLabel.topAnchor.constraint(equalTo: titleLabel.bottomAnchor, constant: 4),

            imageView.topAnchor.constraint(equalTo: subtitleLabel.bottomAnchor, constant: 24),
            imageView.widthAnchor.constraint(equalToConstant: qrSize),
            imageView.heightAnchor.constraint(equalToConstant: qrSize),

            costLabel.leadingAnchor.constraint(equalTo: contentStack.leadingAnchor),
            costLabel.trailingAnchor.constraint(equalTo: contentStack.trailingAnchor),
            costLabel.topAnchor.constraint(equalTo: imageView.bottomAnchor, constant: costLabel.isHidden ? 0 : 14),

            stateLabel.leadingAnchor.constraint(equalTo: contentStack.leadingAnchor),
            stateLabel.trailingAnchor.constraint(equalTo: contentStack.trailingAnchor),
            stateLabel.topAnchor.constraint(equalTo: costLabel.bottomAnchor, constant: costLabel.isHidden ? 16 : 8),

            hintLabel.leadingAnchor.constraint(equalTo: contentStack.leadingAnchor, constant: 8),
            hintLabel.trailingAnchor.constraint(equalTo: contentStack.trailingAnchor, constant: -8),
            hintLabel.topAnchor.constraint(equalTo: stateLabel.bottomAnchor, constant: 10),
        ])
    }

    func updatePollingStatus(_ text: String) {
        stateLabel.text = text
    }

    @objc private func closeTapped() {
        onDismiss?()
        dismiss(animated: true)
    }

    func updateState(_ text: String) {
        stateLabel.text = text
    }
}


final class CoinTasksViewController: BaseNativeViewController {
    private let redeemField = UITextField()
    private let redeemButton = UIButton(type: .system)
    private var isRedeeming = false
    private var dashboard: PortalDashboard?
    private let authHintLabel = UILabel()
    private let statsHost = UIStackView()
    private var checkinAction: ActionButton?
    private var prayAction: ActionButton?

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        loadDashboard()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        navigationController?.setNavigationBarHidden(false, animated: animated)
    }

    private func setupUI() {
        let headerRow = UIView()
        let headerIcon = UILabel()
        headerIcon.text = "⭐"
        headerIcon.font = .systemFont(ofSize: 24)
        let headerTitle = UILabel()
        headerTitle.text = "积分任务"
        headerTitle.font = .systemFont(ofSize: 20, weight: .bold)
        let headerSubtitle = UILabel()
        headerSubtitle.text = "每日打卡 · 祈福 · 积分补充"
        headerSubtitle.font = .systemFont(ofSize: 12)
        headerSubtitle.textColor = .secondaryLabel
        headerSubtitle.numberOfLines = 1
        headerRow.addSubview(headerIcon)
        headerRow.addSubview(headerTitle)
        headerRow.addSubview(headerSubtitle)
        headerIcon.translatesAutoresizingMaskIntoConstraints = false
        headerTitle.translatesAutoresizingMaskIntoConstraints = false
        headerSubtitle.translatesAutoresizingMaskIntoConstraints = false
        headerRow.translatesAutoresizingMaskIntoConstraints = false
        headerSubtitle.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        let scrollView = UIScrollView()
        let stack = UIStackView()
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        stack.axis = .vertical
        stack.spacing = 16
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(headerRow)
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
        NSLayoutConstraint.activate([
            headerRow.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 2),
            headerRow.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            headerRow.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            headerRow.heightAnchor.constraint(equalToConstant: 44),
            headerIcon.leadingAnchor.constraint(equalTo: headerRow.leadingAnchor),
            headerIcon.centerYAnchor.constraint(equalTo: headerRow.centerYAnchor),
            headerIcon.widthAnchor.constraint(equalToConstant: 32),
            headerTitle.leadingAnchor.constraint(equalTo: headerIcon.trailingAnchor, constant: 8),
            headerTitle.centerYAnchor.constraint(equalTo: headerRow.centerYAnchor),
            headerSubtitle.leadingAnchor.constraint(equalTo: headerTitle.trailingAnchor, constant: 8),
            headerSubtitle.trailingAnchor.constraint(lessThanOrEqualTo: headerRow.trailingAnchor),
            headerSubtitle.centerYAnchor.constraint(equalTo: headerTitle.centerYAnchor),
            scrollView.topAnchor.constraint(equalTo: headerRow.bottomAnchor, constant: 8),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor),
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -16),
            stack.widthAnchor.constraint(equalTo: view.widthAnchor, constant: -32)
        ])

        let checkinAction = ActionButton(icon: "checkmark.circle.fill", title: "每日打卡", desc: "签到领取积分奖励", tint: .systemBlue)
        checkinAction.tapAction = { [weak self] in self?.checkinTapped() }
        let prayAction = ActionButton(icon: "hands.sparkles.fill", title: "每日祈福", desc: "祈福获得额外积分", tint: .systemPurple)
        prayAction.tapAction = { [weak self] in self?.prayTapped() }
        self.checkinAction = checkinAction
        self.prayAction = prayAction

        authHintLabel.font = UIFont.systemFont(ofSize: 12)
        authHintLabel.textColor = .systemRed
        authHintLabel.numberOfLines = 0
        authHintLabel.isHidden = true

        statsHost.axis = .vertical
        statsHost.spacing = 8

        let actionsCaption = UILabel()
        actionsCaption.text = "打卡与祈福需有效的按月/按天项目"
        actionsCaption.font = UIFont.systemFont(ofSize: 12)
        actionsCaption.textColor = .secondaryLabel
        actionsCaption.numberOfLines = 0

        let actionsCard = UIView()
        actionsCard.applyCardStyle()
        let actionsTitle = UILabel()
        actionsTitle.text = "每日任务"
        actionsTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let actionsStack = UIStackView(arrangedSubviews: [actionsTitle, actionsCaption, statsHost, authHintLabel, checkinAction, prayAction])
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

        let phoneCard = UIView()
        phoneCard.applyCardStyle()
        let phoneIcon = UIImageView()
        phoneIcon.translatesAutoresizingMaskIntoConstraints = false
        phoneIcon.image = UIImage(systemName: "simcard.fill")
        phoneIcon.tintColor = .systemTeal
        phoneIcon.contentMode = .scaleAspectFit
        let phoneTitle = UILabel()
        phoneTitle.text = "手机卡业务"
        phoneTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let phoneDesc = UILabel()
        phoneDesc.text = "办理超值大流量手机卡"
        phoneDesc.font = UIFont.systemFont(ofSize: 13)
        phoneDesc.textColor = .secondaryLabel
        phoneDesc.numberOfLines = 0
        let phoneBtn = UIButton(type: .system)
        phoneBtn.setTitle("前往办理", for: .normal)
        phoneBtn.applyPrimaryStyle(color: .systemTeal)
        phoneBtn.heightAnchor.constraint(equalToConstant: 48).isActive = true
        phoneBtn.addTarget(self, action: #selector(phoneCardTapped), for: .touchUpInside)
        let phoneStack = UIStackView(arrangedSubviews: [phoneIcon, phoneTitle, phoneDesc, phoneBtn])
        phoneStack.axis = .vertical
        phoneStack.alignment = .fill
        phoneStack.spacing = 10
        phoneStack.translatesAutoresizingMaskIntoConstraints = false
        phoneIcon.widthAnchor.constraint(equalToConstant: 36).isActive = true
        phoneIcon.heightAnchor.constraint(equalToConstant: 36).isActive = true
        phoneCard.addSubview(phoneStack)
        NSLayoutConstraint.activate([
            phoneStack.topAnchor.constraint(equalTo: phoneCard.topAnchor, constant: 18),
            phoneStack.leadingAnchor.constraint(equalTo: phoneCard.leadingAnchor, constant: 18),
            phoneStack.trailingAnchor.constraint(equalTo: phoneCard.trailingAnchor, constant: -18),
            phoneStack.bottomAnchor.constraint(equalTo: phoneCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(phoneCard)

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

        // 积分变动记录
        let logCard = UIView()
        logCard.applyCardStyle()
        let logIcon = UIImageView()
        logIcon.translatesAutoresizingMaskIntoConstraints = false
        logIcon.image = UIImage(systemName: "list.bullet.rectangle.fill")
        logIcon.tintColor = .systemPurple
        logIcon.contentMode = .scaleAspectFit
        let logTitle = UILabel()
        logTitle.text = "积分变动记录"
        logTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let logDesc = UILabel()
        logDesc.text = "查看积分收支明细，了解积分来源与去向"
        logDesc.font = UIFont.systemFont(ofSize: 13)
        logDesc.textColor = .secondaryLabel
        logDesc.numberOfLines = 0
        let logBtn = UIButton(type: .system)
        logBtn.setTitle("查看记录", for: .normal)
        logBtn.applyPrimaryStyle(color: .systemPurple)
        logBtn.heightAnchor.constraint(equalToConstant: 48).isActive = true
        logBtn.addTarget(self, action: #selector(showCoinLogs), for: .touchUpInside)
        let logStack = UIStackView(arrangedSubviews: [logIcon, logTitle, logDesc, logBtn])
        logStack.axis = .vertical
        logStack.alignment = .fill
        logStack.spacing = 10
        logStack.translatesAutoresizingMaskIntoConstraints = false
        logIcon.widthAnchor.constraint(equalToConstant: 36).isActive = true
        logIcon.heightAnchor.constraint(equalToConstant: 36).isActive = true
        logCard.addSubview(logStack)
        NSLayoutConstraint.activate([
            logStack.topAnchor.constraint(equalTo: logCard.topAnchor, constant: 18),
            logStack.leadingAnchor.constraint(equalTo: logCard.leadingAnchor, constant: 18),
            logStack.trailingAnchor.constraint(equalTo: logCard.trailingAnchor, constant: -18),
            logStack.bottomAnchor.constraint(equalTo: logCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(logCard)
    }

    private func loadDashboard() {
        PortalService.shared.fetchDashboard { [weak self] result in
            guard let self else { return }
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let dashboard):
                self.dashboard = dashboard
                self.applyDashboard(dashboard)
            }
        }
    }

    private func applyDashboard(_ d: PortalDashboard) {
        statsHost.arrangedSubviews.forEach { $0.removeFromSuperview() }

        let continuous = d.continuousDays ?? 0
        let checkedIn = d.checkedInToday ?? false
        let nextBonus = d.nextCheckInBonus ?? 0
        let daysUntil = d.daysUntilNextCheckInBonus ?? 0
        let todayCount = d.todayCheckInCount ?? 0
        let bonusText = nextBonus > 0 ? "+\(nextBonus)（还差\(daysUntil)天）" : "已达最高档"
        let statusText = checkedIn ? "已打卡" : "未打卡"
        let statusColor: UIColor = checkedIn ? .systemGreen : .secondaryLabel

        let topRow = UIStackView(arrangedSubviews: [
            makeStatTile(label: "连续打卡", value: "\(continuous) 天", color: .systemBlue),
            makeStatTile(label: "今日状态", value: statusText, color: statusColor),
        ])
        topRow.axis = .horizontal
        topRow.spacing = 8
        topRow.distribution = .fillEqually

        let bottomRow = UIStackView(arrangedSubviews: [
            makeStatTile(label: "下次奖励", value: bonusText, color: .systemOrange),
            makeStatTile(label: "今日打卡", value: "\(todayCount) 人", color: .systemPurple),
        ])
        bottomRow.axis = .horizontal
        bottomRow.spacing = 8
        bottomRow.distribution = .fillEqually

        statsHost.addArrangedSubview(topRow)
        statsHost.addArrangedSubview(bottomRow)

        let canCheckIn = d.canCheckIn ?? false
        if canCheckIn {
            authHintLabel.isHidden = true
        } else {
            authHintLabel.text = d.canCheckInMessage?.isEmpty == false
                ? d.canCheckInMessage
                : "请先前往「项目中心」上车有效的按月/按天活动"
            authHintLabel.isHidden = false
        }

        let checkinEnabled = canCheckIn && !checkedIn
        checkinAction?.isEnabled = checkinEnabled
        checkinAction?.updateTitle(checkedIn ? "今日已打卡" : "每日打卡")

        let prayedToday = d.prayedToday ?? false
        let prayEnabled = canCheckIn && !prayedToday
        prayAction?.isEnabled = prayEnabled
        prayAction?.updateTitle(prayedToday ? "今日已祈福" : "每日祈福")
    }

    private func makeStatTile(label: String, value: String, color: UIColor) -> UIView {
        let container = UIView()
        container.backgroundColor = UIColor.secondarySystemBackground
        container.layer.cornerRadius = 10

        let labelView = UILabel()
        labelView.text = label
        labelView.font = UIFont.systemFont(ofSize: 11, weight: .semibold)
        labelView.textColor = .secondaryLabel

        let valueView = UILabel()
        valueView.text = value
        valueView.font = UIFont.systemFont(ofSize: 14, weight: .bold)
        valueView.textColor = color
        valueView.numberOfLines = 2

        let stack = UIStackView(arrangedSubviews: [labelView, valueView])
        stack.axis = .vertical
        stack.spacing = 4
        stack.translatesAutoresizingMaskIntoConstraints = false
        container.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: container.topAnchor, constant: 10),
            stack.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: 10),
            stack.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -10),
            stack.bottomAnchor.constraint(equalTo: container.bottomAnchor, constant: -10),
        ])
        return container
    }

    @objc private func showCoinLogs() {
        let vc = CoinLogViewController()
        navigationController?.pushViewController(vc, animated: true)
    }

    @objc private func checkinTapped() {
        if let d = dashboard, d.canCheckIn != true {
            showMessage(d.canCheckInMessage ?? "暂无法打卡")
            return
        }
        PortalService.shared.checkin { result in
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let message):
                self.showMessage(message)
                AppSessionStore.shared.refreshIfPossible(silent: true)
                self.loadDashboard()
            }
        }
    }

    @objc private func prayTapped() {
        if let d = dashboard, d.canCheckIn != true {
            showMessage(d.canCheckInMessage ?? "暂无法祈福")
            return
        }
        PortalService.shared.pray { result in
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let message):
                self.showMessage(message)
                AppSessionStore.shared.refreshIfPossible(silent: true)
                self.loadDashboard()
            }
        }
    }

    @objc private func phoneCardTapped() {
        guard let url = URL(string: "https://h5.lot-ml.com/ProductEn/Index/7ee6c54f2d550fab") else { return }
        let vc = WebBrowserViewController(url: url)
        navigationController?.pushViewController(vc, animated: true)
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
