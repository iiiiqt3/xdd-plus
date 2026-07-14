import UIKit

@available(iOS 13.0, *)
final class JdPortalViewController: BaseNativeViewController, UITextFieldDelegate, UISearchBarDelegate, InnerTabSwipeHandling {
    override var shouldEnableKeyboardDismissOnTap: Bool { false }

    private enum MainTab { case query, login, task }
    private enum LoginTab { case sms, yyb, wx }

    private struct TaskDef {
        let id: String
        let name: String
        let icon: String
        let desc: String
    }

    private let scrollView = UIScrollView()
    private let stack = UIStackView()
    private let loginPanel = UIView()
    private let headerRow = UIView()
    private let mainSegmented = UISegmentedControl(items: ["查询", "登录", "京东任务"])
    private let loginTabRow = UIStackView()
    private let queryContentStack = UIStackView()
    private let loginContentStack = UIStackView()
    private let taskContentStack = UIStackView()
    private let taskStickyBar = UIStackView()
    private var scrollTopToSegment: NSLayoutConstraint!
    private var scrollTopToSticky: NSLayoutConstraint!

    private var mainTab: MainTab = .query
    private var loginTab: LoginTab = .sms
    private var accounts: [PortalJdAccount] = []
    private var globalTaskAccountSelection: Set<Int> = [0]
    private var jdProxyStatus: PortalJdProxyStatus?
    private var proxyMetaLabel: UILabel?
    private var proxyBadgeLabel: UILabel?
    private var proxyBuyButton: UIButton?
    private var proxyMonthsButton: UIButton?
    private var proxyPurchaseMonths = 1
    private var proxyStatsStack: UIStackView?
    private var accountChipsStack: UIStackView?
    private var runningTasks: [String: String] = [:]
    private var logStreamers: [String: JdTaskLogStreamer] = [:]
    private var taskButtonRefs: [String: UIButton] = [:]
    private var logLines: [String] = []
    private let logTextView = UITextView()
    private var buttonActions: [ObjectIdentifier: () -> Void] = [:]
    private var buttonOriginalTitles: [ObjectIdentifier: String] = [:]
    private var isQueryingAll = false
    private let buttonHaptic = UIImpactFeedbackGenerator(style: .light)

    private let smsPhoneField = UITextField()
    private let smsCodeField = UITextField()
    private let smsIdCardField = UITextField()
    private let smsIdCardStack = UIStackView()
    private let smsResultLabel = UILabel()
    private lazy var smsLoginCard = buildSmsLoginCard()
    private var smsSendButton: UIButton!
    private var smsVerifyButton: UIButton!
    private var accountsRequestToken = 0
    private var lastRenderedMainTab: MainTab?
    private var lastRenderedLoginTab: LoginTab?
    private let wxDeviceGridStack = UIStackView()
    private let wxRiskStack = UIStackView()
    private let wxRiskLabel = UILabel()
    private let wxRiskLink = UILabel()
    private let wxResultLabel = UILabel()
    private let yybAccountGridStack = UIStackView()
    private let yybRiskStack = UIStackView()
    private let yybRiskLabel = UILabel()
    private let yybRiskLink = UILabel()
    private let yybResultLabel = UILabel()
    private var yybRiskUrl: String?

    private var taskDefs: [TaskDef] = []
    private var taskSearchQuery = ""
    private let taskSearchBar = UISearchBar()
    private let taskGridStack = UIStackView()

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupLayout()
        buttonHaptic.prepare()
        renderAll()
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(keyboardWillChangeFrame(_:)),
            name: UIResponder.keyboardWillChangeFrameNotification,
            object: nil
        )
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        view.endEditing(true)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        if mainTab == .query { loadAccounts() }
        if mainTab == .task && !taskContentStack.arrangedSubviews.isEmpty {
            loadTaskTabData()
        }
    }

    @objc private func keyboardWillChangeFrame(_ notification: Notification) {
        guard mainTab != .login else { return }
        guard
            let frame = notification.userInfo?[UIResponder.keyboardFrameEndUserInfoKey] as? CGRect,
            let duration = notification.userInfo?[UIResponder.keyboardAnimationDurationUserInfoKey] as? TimeInterval
        else { return }
        let keyboardInView = view.convert(frame, from: nil)
        let overlap = max(0, view.bounds.maxY - keyboardInView.minY - view.safeAreaInsets.bottom)
        UIView.animate(withDuration: duration) {
            self.scrollView.contentInset.bottom = overlap
            self.scrollView.verticalScrollIndicatorInsets.bottom = overlap
        }
    }

    // MARK: - Layout

    private func setupLayout() {
        let headerIcon = UILabel()
        headerIcon.text = "🛒"
        headerIcon.font = .systemFont(ofSize: 24)
        let headerTitle = UILabel()
        headerTitle.text = "京东工作台"
        headerTitle.font = .systemFont(ofSize: 20, weight: .bold)
        let headerSubtitle = UILabel()
        headerSubtitle.text = "查询 · 登录 · 京东任务"
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

        scrollView.translatesAutoresizingMaskIntoConstraints = false
        scrollView.keyboardDismissMode = .interactive
        scrollView.delaysContentTouches = true
        scrollView.canCancelContentTouches = true
        stack.axis = .vertical
        stack.spacing = 12
        stack.translatesAutoresizingMaskIntoConstraints = false

        mainSegmented.translatesAutoresizingMaskIntoConstraints = false
        loginTabRow.translatesAutoresizingMaskIntoConstraints = false
        loginPanel.translatesAutoresizingMaskIntoConstraints = false

        view.addSubview(headerRow)
        view.addSubview(mainSegmented)
        view.addSubview(loginTabRow)
        view.addSubview(loginPanel)
        view.addSubview(taskStickyBar)
        view.addSubview(scrollView)
        scrollView.addSubview(stack)

        mainSegmented.selectedSegmentIndex = 0
        if #available(iOS 13.0, *) {
            mainSegmented.setTitleTextAttributes([.font: UIFont.systemFont(ofSize: 12, weight: .semibold)], for: .normal)
        }
        mainSegmented.addTarget(self, action: #selector(mainSegmentChanged), for: .valueChanged)

        loginTabRow.axis = .horizontal
        loginTabRow.spacing = 8
        loginTabRow.distribution = .fillEqually
        loginTabRow.isHidden = true

        [queryContentStack, loginContentStack, taskContentStack].forEach {
            $0.axis = .vertical
            $0.spacing = 12
            $0.alignment = .fill
            $0.translatesAutoresizingMaskIntoConstraints = false
        }

        loginPanel.addSubview(loginContentStack)
        stack.addArrangedSubview(queryContentStack)
        stack.addArrangedSubview(taskContentStack)

        setupTaskStickyBar()

        scrollTopToSegment = scrollView.topAnchor.constraint(equalTo: mainSegmented.bottomAnchor, constant: 10)
        scrollTopToSticky = scrollView.topAnchor.constraint(equalTo: taskStickyBar.bottomAnchor, constant: 8)
        scrollTopToSegment.isActive = true

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

            mainSegmented.topAnchor.constraint(equalTo: headerRow.bottomAnchor, constant: 10),
            mainSegmented.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            mainSegmented.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),

            loginTabRow.topAnchor.constraint(equalTo: mainSegmented.bottomAnchor, constant: 8),
            loginTabRow.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            loginTabRow.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            loginTabRow.heightAnchor.constraint(equalToConstant: 40),

            loginPanel.topAnchor.constraint(equalTo: loginTabRow.bottomAnchor, constant: 8),
            loginPanel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            loginPanel.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            loginPanel.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor, constant: -8),

            loginContentStack.topAnchor.constraint(equalTo: loginPanel.topAnchor),
            loginContentStack.leadingAnchor.constraint(equalTo: loginPanel.leadingAnchor),
            loginContentStack.trailingAnchor.constraint(equalTo: loginPanel.trailingAnchor),
            loginContentStack.bottomAnchor.constraint(lessThanOrEqualTo: loginPanel.bottomAnchor),

            taskStickyBar.topAnchor.constraint(equalTo: mainSegmented.bottomAnchor, constant: 8),
            taskStickyBar.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            taskStickyBar.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),

            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor),

            // 使用 contentLayoutGuide 约束，让 scrollView 可以滚动
            stack.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor),
            stack.leadingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor, constant: -20),
            stack.widthAnchor.constraint(equalTo: scrollView.frameLayoutGuide.widthAnchor, constant: -32),
        ])

        setupSmsFields()
    }

    private var activeContentStack: UIStackView {
        switch mainTab {
        case .query: return queryContentStack
        case .login: return loginContentStack
        case .task: return taskContentStack
        }
    }

    private func clearContentStack(_ stack: UIStackView) {
        stack.arrangedSubviews.forEach { $0.removeFromSuperview() }
    }

    // MARK: - 查询

    private func renderQuery() {
        clearContentStack(queryContentStack)
        queryContentStack.addArrangedSubview(placeholderLabel("加载账号中..."))
        loadAccounts()
    }

    private func loadAccounts(refreshButton: UIButton? = nil) {
        accountsRequestToken += 1
        let requestToken = accountsRequestToken
        setButtonLoading(refreshButton, loading: true, title: "刷新中...")
        PortalService.shared.fetchJdAccounts { [weak self] result in
            guard let self = self else { return }
            DispatchQueue.main.async {
                self.setButtonLoading(refreshButton, loading: false, title: "刷新")
                guard requestToken == self.accountsRequestToken, self.mainTab == .query else { return }
                self.clearContentStack(self.queryContentStack)
                switch result {
                case .failure(let error):
                    self.handle(error)
                    self.queryContentStack.addArrangedSubview(self.placeholderLabel(error.message))
                case .success(let list):
                    self.accounts = list
                    self.renderAccountGrid(list)
                }
            }
        }
    }

    private func setupSmsFields() {
        [smsPhoneField, smsCodeField, smsIdCardField].forEach {
            $0.delegate = self
            $0.isUserInteractionEnabled = true
            $0.autocorrectionType = .no
            $0.inputAccessoryView = Self.numberPadToolbar(target: self, action: #selector(dismissNumberPad))
        }
        _ = smsLoginCard
    }

    func textFieldShouldBeginEditing(_ textField: UITextField) -> Bool {
        textField.isUserInteractionEnabled && textField.window != nil
    }

    func textField(_ textField: UITextField, shouldChangeCharactersIn range: NSRange, replacementString string: String) -> Bool {
        true
    }

    @objc private func dismissNumberPad() {
        view.endEditing(true)
    }

    private static func numberPadToolbar(target: Any?, action: Selector) -> UIToolbar {
        let toolbar = UIToolbar(frame: CGRect(x: 0, y: 0, width: UIScreen.main.bounds.width, height: 44))
        toolbar.sizeToFit()
        let flex = UIBarButtonItem(barButtonSystemItem: .flexibleSpace, target: nil, action: nil)
        let done = UIBarButtonItem(title: "完成", style: .done, target: target, action: action)
        toolbar.items = [flex, done]
        return toolbar
    }

    func textFieldShouldReturn(_ textField: UITextField) -> Bool {
        if textField === smsPhoneField {
            smsCodeField.becomeFirstResponder()
        } else {
            textField.resignFirstResponder()
        }
        return true
    }

    @objc private func mainSegmentChanged() {
        switch mainSegmented.selectedSegmentIndex {
        case 0: mainTab = .query
        case 1: mainTab = .login
        default: mainTab = .task
        }
        renderAll()
        if mainTab == .login {
            lastRenderedLoginTab = nil
        }
        if mainTab == .query { loadAccounts() }
        if mainTab == .task {
            clearContentStack(taskContentStack)
            renderTasks()
        }
    }

    private func renderAll() {
        syncMainSegmentIndex()
        if lastRenderedMainTab != mainTab {
            view.endEditing(true)
        }

        let isLogin = mainTab == .login
        loginTabRow.isHidden = !isLogin
        loginPanel.isHidden = !isLogin
        scrollView.isHidden = isLogin
        taskStickyBar.isHidden = mainTab != .task
        scrollTopToSegment.isActive = mainTab != .task
        scrollTopToSticky.isActive = mainTab == .task
        queryContentStack.isHidden = mainTab != .query
        taskContentStack.isHidden = mainTab != .task

        switch mainTab {
        case .query:
            if queryContentStack.arrangedSubviews.isEmpty { renderQuery() }
        case .login:
            renderLogin()
        case .task:
            if taskContentStack.arrangedSubviews.isEmpty { renderTasks() }
        }

        lastRenderedMainTab = mainTab
    }

    private func syncMainSegmentIndex() {
        switch mainTab {
        case .query: mainSegmented.selectedSegmentIndex = 0
        case .login: mainSegmented.selectedSegmentIndex = 1
        case .task: mainSegmented.selectedSegmentIndex = 2
        }
    }

    // MARK: - 查询

    private func renderAccountGrid(_ list: [PortalJdAccount]) {
        if list.isEmpty {
            queryContentStack.addArrangedSubview(placeholderLabel("暂无绑定的京东账号\n请前往「登录」使用短信或微信协议刷新"))
            return
        }

        let valid = list.filter { $0.valid }.count
        let toolbar = UIView()
        toolbar.translatesAutoresizingMaskIntoConstraints = false
        toolbar.applyCardStyle(cornerRadius: 14)

        let stats = UILabel()
        stats.text = "有效 \(valid) / 共 \(list.count) 个账号"
        stats.font = .systemFont(ofSize: 14, weight: .semibold)
        stats.textColor = .label
        stats.translatesAutoresizingMaskIntoConstraints = false

        let btnRow = UIStackView()
        btnRow.axis = .horizontal
        btnRow.spacing = 8
        btnRow.translatesAutoresizingMaskIntoConstraints = false
        var queryAllBtn: UIButton!
        queryAllBtn = compactButton("全部查询", color: .systemBlue) { [weak self] in
            self?.queryAccount(0, button: queryAllBtn)
        }
        var refreshBtn: UIButton!
        refreshBtn = compactButton("刷新", color: .secondaryLabel, filled: false) { [weak self] in
            self?.loadAccounts(refreshButton: refreshBtn)
        }
        btnRow.addArrangedSubview(queryAllBtn)
        btnRow.addArrangedSubview(refreshBtn)

        toolbar.addSubview(stats)
        toolbar.addSubview(btnRow)
        NSLayoutConstraint.activate([
            toolbar.heightAnchor.constraint(greaterThanOrEqualToConstant: 52),
            stats.leadingAnchor.constraint(equalTo: toolbar.leadingAnchor, constant: 14),
            stats.centerYAnchor.constraint(equalTo: toolbar.centerYAnchor),
            btnRow.trailingAnchor.constraint(equalTo: toolbar.trailingAnchor, constant: -12),
            btnRow.centerYAnchor.constraint(equalTo: toolbar.centerYAnchor),
            stats.trailingAnchor.constraint(lessThanOrEqualTo: btnRow.leadingAnchor, constant: -8),
        ])
        queryContentStack.addArrangedSubview(toolbar)

        let sorted = list.sorted { $0.valid && !$1.valid }
        var index = 0
        while index < sorted.count {
            let row = UIStackView()
            row.axis = .horizontal
            row.spacing = 10
            row.distribution = .fillEqually
            row.addArrangedSubview(buildAccountCard(sorted[index]))
            if index + 1 < sorted.count {
                row.addArrangedSubview(buildAccountCard(sorted[index + 1]))
            } else {
                row.addArrangedSubview(UIView())
            }
            queryContentStack.addArrangedSubview(row)
            index += 2
        }
    }

    private func buildAccountCard(_ acc: PortalJdAccount) -> UIView {
        let wrap = UIView()
        wrap.applyCardStyle(cornerRadius: 14)
        wrap.translatesAutoresizingMaskIntoConstraints = false

        let initial = String((acc.nickname ?? acc.pin ?? "?").prefix(1))
        let avatar = UILabel()
        avatar.text = initial
        avatar.font = .systemFont(ofSize: 18, weight: .bold)
        avatar.textAlignment = .center
        avatar.textColor = acc.valid ? .systemBlue : .systemGray
        avatar.backgroundColor = (acc.valid ? UIColor.systemBlue : UIColor.systemGray).withAlphaComponent(0.12)
        avatar.layer.cornerRadius = 20
        avatar.clipsToBounds = true
        avatar.translatesAutoresizingMaskIntoConstraints = false

        let name = UILabel()
        name.text = acc.nickname ?? acc.pin ?? "未知账号"
        name.font = .systemFont(ofSize: 14, weight: .semibold)
        name.numberOfLines = 1
        name.translatesAutoresizingMaskIntoConstraints = false

        let pin = UILabel()
        pin.text = acc.pin ?? ""
        pin.font = .systemFont(ofSize: 11)
        pin.textColor = .secondaryLabel
        pin.numberOfLines = 1
        pin.translatesAutoresizingMaskIntoConstraints = false

        let badge = UILabel()
        badge.text = acc.valid ? "有效" : "失效"
        badge.font = .systemFont(ofSize: 10, weight: .bold)
        badge.textColor = acc.valid ? .systemGreen : .systemRed
        badge.backgroundColor = (acc.valid ? UIColor.systemGreen : UIColor.systemRed).withAlphaComponent(0.12)
        badge.textAlignment = .center
        badge.layer.cornerRadius = 8
        badge.clipsToBounds = true
        badge.translatesAutoresizingMaskIntoConstraints = false

        var queryBtn: UIButton!
        queryBtn = compactButton("查询", color: .systemBlue) { [weak self] in
            self?.queryAccount(acc.index, button: queryBtn)
        }
        queryBtn.translatesAutoresizingMaskIntoConstraints = false

        wrap.addSubview(avatar)
        wrap.addSubview(name)
        wrap.addSubview(pin)
        wrap.addSubview(badge)
        wrap.addSubview(queryBtn)

        NSLayoutConstraint.activate([
            wrap.heightAnchor.constraint(greaterThanOrEqualToConstant: 118),
            avatar.topAnchor.constraint(equalTo: wrap.topAnchor, constant: 12),
            avatar.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 12),
            avatar.widthAnchor.constraint(equalToConstant: 40),
            avatar.heightAnchor.constraint(equalToConstant: 40),
            name.topAnchor.constraint(equalTo: wrap.topAnchor, constant: 12),
            name.leadingAnchor.constraint(equalTo: avatar.trailingAnchor, constant: 8),
            name.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -10),
            pin.topAnchor.constraint(equalTo: name.bottomAnchor, constant: 2),
            pin.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            pin.trailingAnchor.constraint(equalTo: name.trailingAnchor),
            badge.topAnchor.constraint(equalTo: pin.bottomAnchor, constant: 6),
            badge.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            badge.widthAnchor.constraint(greaterThanOrEqualToConstant: 36),
            badge.heightAnchor.constraint(equalToConstant: 18),
            queryBtn.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 10),
            queryBtn.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -10),
            queryBtn.bottomAnchor.constraint(equalTo: wrap.bottomAnchor, constant: -10),
            queryBtn.heightAnchor.constraint(equalToConstant: 32),
            queryBtn.topAnchor.constraint(greaterThanOrEqualTo: badge.bottomAnchor, constant: 10),
        ])
        return wrap
    }

    private func queryAccount(_ index: Int, button: UIButton? = nil) {
        if index == 0 {
            guard !isQueryingAll else { return }
            isQueryingAll = true
        }
        guard button?.isEnabled != false else { return }

        let defaultTitle = index == 0 ? "全部查询" : "查询"
        setButtonLoading(button, loading: true, title: "查询中...")
        PortalService.shared.queryJdAccount(index: index) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                if index == 0 { self.isQueryingAll = false }
                switch result {
                case .failure(let error):
                    self.setButtonLoading(button, loading: false, title: defaultTitle)
                    self.handle(error)
                case .success(let text):
                    if let btn = button {
                        btn.alpha = 1
                        self.buttonOriginalTitles.removeValue(forKey: ObjectIdentifier(btn))
                        self.flashButtonSuccess(btn, message: "✅ 查询完成", restore: defaultTitle)
                    }
                    let vc = QueryResultViewController(title: "查询结果", content: text)
                    self.navigationController?.pushViewController(vc, animated: true)
                }
            }
        }
    }

    // MARK: - 登录

    private func renderLogin() {
        loginTabRow.arrangedSubviews.forEach { $0.removeFromSuperview() }
        loginTabRow.addArrangedSubview(segmentButton("短信登录", active: loginTab == .sms) { [weak self] in
            guard let self = self, self.loginTab != .sms else { return }
            self.loginTab = .sms
            self.renderLogin()
        })
        loginTabRow.addArrangedSubview(segmentButton("应用宝刷新", active: loginTab == .yyb) { [weak self] in
            guard let self = self, self.loginTab != .yyb else { return }
            self.loginTab = .yyb
            self.renderLogin()
        })
        loginTabRow.addArrangedSubview(segmentButton("微信协议", active: loginTab == .wx) { [weak self] in
            guard let self = self, self.loginTab != .wx else { return }
            self.loginTab = .wx
            self.renderLogin()
        })

        guard lastRenderedLoginTab != loginTab else { return }

        if loginTab != .sms {
            view.endEditing(true)
        }

        clearContentStack(loginContentStack)
        switch loginTab {
        case .sms:
            loginContentStack.addArrangedSubview(smsLoginCard)
        case .yyb:
            renderYyb()
        case .wx:
            renderWx()
        }
        lastRenderedLoginTab = loginTab
    }

    private func buildSmsLoginCard() -> UIStackView {
        let card = UIStackView()
        card.axis = .vertical
        card.spacing = 16
        card.isLayoutMarginsRelativeArrangement = true
        card.layoutMargins = UIEdgeInsets(top: 20, left: 18, bottom: 20, right: 18)
        card.applyCardStyle(cornerRadius: 16)

        let title = UILabel()
        title.text = "短信验证码登录"
        title.font = .systemFont(ofSize: 18, weight: .bold)

        let hint = UILabel()
        hint.text = "输入京东绑定的手机号，验证码登录后自动绑定到账号"
        hint.font = .systemFont(ofSize: 13)
        hint.textColor = .secondaryLabel
        hint.numberOfLines = 0

        card.addArrangedSubview(title)
        card.setCustomSpacing(10, after: title)
        card.addArrangedSubview(hint)
        card.setCustomSpacing(20, after: hint)

        smsPhoneField.applyAppInputStyle(placeholder: "请输入11位手机号")
        smsPhoneField.keyboardType = .numberPad
        smsPhoneField.translatesAutoresizingMaskIntoConstraints = false
        smsPhoneField.heightAnchor.constraint(equalToConstant: 48).isActive = true
        card.addArrangedSubview(smsPhoneField)

        smsSendButton = compactButton("发送验证码", color: .systemBlue) { [weak self] in
            guard let self = self else { return }
            self.sendSms(button: self.smsSendButton)
        }
        card.addArrangedSubview(smsSendButton)

        smsCodeField.applyAppInputStyle(placeholder: "请输入短信验证码")
        smsCodeField.keyboardType = .numberPad
        smsCodeField.translatesAutoresizingMaskIntoConstraints = false
        smsCodeField.heightAnchor.constraint(equalToConstant: 48).isActive = true
        card.addArrangedSubview(smsCodeField)

        smsIdCardStack.axis = .vertical
        smsIdCardStack.spacing = 10
        smsIdCardStack.isHidden = true
        let idHint = UILabel()
        idHint.text = "需要补充身份证验证"
        idHint.font = .systemFont(ofSize: 13, weight: .medium)
        idHint.textColor = .systemOrange
        smsIdCardStack.addArrangedSubview(idHint)
        smsIdCardField.applyAppInputStyle(placeholder: "身份证前两位 + 后四位")
        smsIdCardField.translatesAutoresizingMaskIntoConstraints = false
        smsIdCardField.heightAnchor.constraint(equalToConstant: 48).isActive = true
        smsIdCardStack.addArrangedSubview(smsIdCardField)
        card.addArrangedSubview(smsIdCardStack)

        smsVerifyButton = compactButton("提交登录", color: .systemPurple) { [weak self] in
            guard let self = self else { return }
            self.verifySms(button: self.smsVerifyButton)
        }
        card.addArrangedSubview(smsVerifyButton)

        smsResultLabel.numberOfLines = 0
        smsResultLabel.font = .systemFont(ofSize: 13)
        smsResultLabel.textColor = .secondaryLabel
        smsResultLabel.isHidden = true
        card.addArrangedSubview(smsResultLabel)

        return card
    }

    private func sendSms(button: UIButton) {
        let phone = smsPhoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard phone.count == 11 else { showMessage("请输入11位手机号"); return }
        setButtonLoading(button, loading: true, title: "发送中...")
        PortalService.shared.sendJdSms(phone: phone) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.setButtonLoading(button, loading: false, title: "发送验证码")
                switch result {
                case .failure(let error): self.handle(error)
                case .success(let msg):
                    self.flashButtonSuccess(button, message: "✅ 已发送", restore: "发送验证码")
                    self.showMessage(msg)
                    self.smsIdCardStack.isHidden = true
                }
            }
        }
    }

    private func verifySms(button: UIButton) {
        let phone = smsPhoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let code = smsCodeField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let idCard = smsIdCardField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        setButtonLoading(button, loading: true, title: "提交中...")
        PortalService.shared.verifyJdSms(phone: phone, code: code, idCard: idCard) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.setButtonLoading(button, loading: false, title: "提交登录")
                switch result {
                case .failure(let error): self.handle(error)
                case .success(let data):
                    if data.needIdVerify == true {
                        self.smsIdCardStack.isHidden = false
                        self.showMessage(data.message ?? "需要身份证验证")
                        return
                    }
                    self.flashButtonSuccess(button, message: "✅ 登录成功", restore: "提交登录")
                    self.smsIdCardStack.isHidden = true
                    self.smsResultLabel.text = data.queryResult ?? data.message ?? "登录成功"
                    self.smsResultLabel.isHidden = false
                    self.showMessage(data.message ?? "登录成功")
                }
            }
        }
    }

    private func renderYyb() {
        let tipCard = UIStackView()
        tipCard.axis = .vertical
        tipCard.spacing = 8
        tipCard.isLayoutMarginsRelativeArrangement = true
        tipCard.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        tipCard.applyCardStyle(cornerRadius: 14)
        let tipTitle = UILabel()
        tipTitle.text = "京东应用宝协议刷新"
        tipTitle.font = .systemFont(ofSize: 16, weight: .bold)
        let tip = UILabel()
        tip.text = "请先在「项目 → 协议接入 → 应用宝协议」扫码绑定账号，再选择账号刷新京东 CK。"
        tip.font = .systemFont(ofSize: 13)
        tip.textColor = .secondaryLabel
        tip.numberOfLines = 0
        tipCard.addArrangedSubview(tipTitle)
        tipCard.addArrangedSubview(tip)
        loginContentStack.addArrangedSubview(tipCard)

        let refreshBar = UIView()
        refreshBar.translatesAutoresizingMaskIntoConstraints = false
        var refreshAllBtn: UIButton!
        refreshAllBtn = compactButton("刷新全部CK", color: .systemBlue) { [weak self] in
            self?.refreshYybAll(button: refreshAllBtn)
        }
        var refreshBtn: UIButton!
        refreshBtn = compactButton("刷新账号", color: .secondaryLabel, filled: false) { [weak self] in
            self?.loadYybAccounts(refreshButton: refreshBtn)
        }
        let barStack = UIStackView(arrangedSubviews: [refreshAllBtn, refreshBtn])
        barStack.axis = .horizontal
        barStack.spacing = 8
        barStack.translatesAutoresizingMaskIntoConstraints = false
        refreshBar.addSubview(barStack)
        NSLayoutConstraint.activate([
            refreshBar.heightAnchor.constraint(equalToConstant: 36),
            barStack.trailingAnchor.constraint(equalTo: refreshBar.trailingAnchor),
            barStack.centerYAnchor.constraint(equalTo: refreshBar.centerYAnchor),
        ])
        loginContentStack.addArrangedSubview(refreshBar)

        yybAccountGridStack.axis = .vertical
        yybAccountGridStack.spacing = 10
        yybAccountGridStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        yybAccountGridStack.addArrangedSubview(placeholderLabel("加载中..."))
        loginContentStack.addArrangedSubview(yybAccountGridStack)

        yybRiskStack.axis = .vertical
        yybRiskStack.spacing = 8
        yybRiskStack.isHidden = true
        yybRiskStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        yybRiskLabel.numberOfLines = 0
        yybRiskStack.addArrangedSubview(yybRiskLabel)
        yybRiskLink.numberOfLines = 0
        yybRiskLink.font = .systemFont(ofSize: 13)
        yybRiskLink.textColor = .systemBlue
        yybRiskLink.isUserInteractionEnabled = true
        yybRiskLink.gestureRecognizers?.forEach { yybRiskLink.removeGestureRecognizer($0) }
        yybRiskLink.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(openYybRiskUrl)))
        yybRiskStack.addArrangedSubview(yybRiskLink)
        var riskBtn: UIButton!
        riskBtn = compactButton("验证完成，继续刷新", color: .systemOrange) { [weak self] in
            self?.continueYybRisk(button: riskBtn)
        }
        yybRiskStack.addArrangedSubview(riskBtn)
        loginContentStack.addArrangedSubview(yybRiskStack)

        yybResultLabel.numberOfLines = 0
        yybResultLabel.font = .systemFont(ofSize: 13)
        yybResultLabel.textColor = .secondaryLabel
        yybResultLabel.isHidden = true
        loginContentStack.addArrangedSubview(yybResultLabel)

        loadYybAccounts()
    }

    private func loadYybAccounts(refreshButton: UIButton? = nil) {
        setButtonLoading(refreshButton, loading: true, title: "刷新中...")
        PortalService.shared.fetchJdYybAccounts { [weak self] result in
            guard let self = self else { return }
            DispatchQueue.main.async {
                self.setButtonLoading(refreshButton, loading: false, title: "刷新账号")
                guard self.mainTab == .login, self.loginTab == .yyb else { return }
                self.yybAccountGridStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
                switch result {
                case .failure(let error):
                    self.handle(error)
                    self.yybAccountGridStack.addArrangedSubview(self.placeholderLabel(error.message))
                case .success(let list):
                    if let btn = refreshButton {
                        self.flashButtonSuccess(btn, message: "✅ 已刷新", restore: "刷新账号")
                    }
                    if list.isEmpty {
                        self.yybAccountGridStack.addArrangedSubview(self.placeholderLabel("暂无可用应用宝账号"))
                    } else {
                        var idx = 0
                        while idx < list.count {
                            let row = UIStackView()
                            row.axis = .horizontal
                            row.spacing = 10
                            row.distribution = .fillEqually
                            row.addArrangedSubview(self.buildYybAccountCard(list[idx]))
                            if idx + 1 < list.count {
                                row.addArrangedSubview(self.buildYybAccountCard(list[idx + 1]))
                            } else {
                                row.addArrangedSubview(UIView())
                            }
                            self.yybAccountGridStack.addArrangedSubview(row)
                            idx += 2
                        }
                    }
                }
            }
        }
    }

    private func nonEmpty(_ value: String?) -> String? {
        let text = (value ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        return text.isEmpty ? nil : text
    }

    private func buildYybAccountCard(_ account: PortalJdYybAccount) -> UIView {
        let wrap = UIView()
        wrap.applyCardStyle(cornerRadius: 14)
        wrap.translatesAutoresizingMaskIntoConstraints = false

        let st = (account.status ?? "").lowercased()
        let alive = st == "alive" || st == "online" || st.isEmpty

        let name = UILabel()
        name.text = nonEmpty(account.nickname) ?? nonEmpty(account.openid) ?? "账号"
        name.font = .systemFont(ofSize: 14, weight: .semibold)
        name.numberOfLines = 1
        name.lineBreakMode = .byTruncatingTail
        name.translatesAutoresizingMaskIntoConstraints = false

        let badge = UILabel()
        badge.text = alive ? "可用" : "失效"
        badge.font = .systemFont(ofSize: 10, weight: .bold)
        badge.textColor = alive ? .systemGreen : .systemRed
        badge.backgroundColor = (alive ? UIColor.systemGreen : UIColor.systemRed).withAlphaComponent(0.12)
        badge.layer.cornerRadius = 6
        badge.clipsToBounds = true
        badge.textAlignment = .center
        badge.translatesAutoresizingMaskIntoConstraints = false

        let openid = UILabel()
        openid.text = nonEmpty(account.openid) ?? ""
        openid.font = .systemFont(ofSize: 11)
        openid.textColor = .secondaryLabel
        openid.numberOfLines = 2
        openid.lineBreakMode = .byTruncatingMiddle
        openid.translatesAutoresizingMaskIntoConstraints = false

        let jdNick = UILabel()
        if let jd = nonEmpty(account.jdNickname) {
            jdNick.text = "京东 \(jd)"
            jdNick.isHidden = false
        } else {
            jdNick.text = ""
            jdNick.isHidden = true
        }
        jdNick.font = .systemFont(ofSize: 11, weight: .medium)
        jdNick.textColor = .systemOrange
        jdNick.numberOfLines = 1
        jdNick.lineBreakMode = .byTruncatingTail
        jdNick.translatesAutoresizingMaskIntoConstraints = false

        var btn: UIButton!
        btn = compactButton("刷新 CK", color: .systemTeal) { [weak self] in
            self?.refreshYyb(account: account, button: btn)
        }
        btn.translatesAutoresizingMaskIntoConstraints = false

        wrap.addSubview(name)
        wrap.addSubview(badge)
        wrap.addSubview(openid)
        wrap.addSubview(jdNick)
        wrap.addSubview(btn)
        NSLayoutConstraint.activate([
            wrap.heightAnchor.constraint(greaterThanOrEqualToConstant: 118),
            name.topAnchor.constraint(equalTo: wrap.topAnchor, constant: 12),
            name.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 12),
            name.trailingAnchor.constraint(lessThanOrEqualTo: badge.leadingAnchor, constant: -6),
            badge.centerYAnchor.constraint(equalTo: name.centerYAnchor),
            badge.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -10),
            badge.widthAnchor.constraint(greaterThanOrEqualToConstant: 36),
            badge.heightAnchor.constraint(equalToConstant: 20),
            openid.topAnchor.constraint(equalTo: name.bottomAnchor, constant: 4),
            openid.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            openid.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            jdNick.topAnchor.constraint(equalTo: openid.bottomAnchor, constant: 4),
            jdNick.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            jdNick.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            btn.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 10),
            btn.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -10),
            btn.bottomAnchor.constraint(equalTo: wrap.bottomAnchor, constant: -10),
            btn.heightAnchor.constraint(equalToConstant: 32),
            btn.topAnchor.constraint(greaterThanOrEqualTo: jdNick.bottomAnchor, constant: 8),
        ])
        return wrap
    }

    private func showYybRefreshResult(_ data: PortalJdWxRefreshResult) {
        if data.needRiskVerify == true {
            yybRiskStack.isHidden = false
            yybRiskLabel.text = data.riskMsg ?? "账号需要短信验证，请打开链接完成验证"
            yybRiskUrl = data.riskUrl
            yybRiskLink.text = data.riskUrl ?? "验证链接"
            yybRiskLink.isHidden = (data.riskUrl ?? "").isEmpty
            yybResultLabel.isHidden = true
        } else {
            yybRiskStack.isHidden = true
            let details = data.details?.joined(separator: "\n") ?? ""
            let summary = "成功 \(data.success ?? 0)，失败 \(data.fail ?? 0)"
            yybResultLabel.text = details.isEmpty ? summary : "\(summary)\n\(details)"
            yybResultLabel.isHidden = false
        }
    }

    private func refreshYyb(account: PortalJdYybAccount, button: UIButton) {
        guard let openid = account.openid, !openid.isEmpty else {
            showMessage("账号 OpenID 无效")
            return
        }
        setButtonLoading(button, loading: true, title: "刷新中...")
        PortalService.shared.refreshJdYyb(openid: openid) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.setButtonLoading(button, loading: false, title: "刷新 CK")
                switch result {
                case .failure(let error):
                    self.handle(error)
                case .success(let data):
                    self.showYybRefreshResult(data)
                    if (data.success ?? 0) > 0 {
                        self.flashButtonSuccess(button, message: "✅ 已刷新", restore: "刷新 CK")
                        self.loadAccounts()
                    }
                }
            }
        }
    }

    private func refreshYybAll(button: UIButton) {
        setButtonLoading(button, loading: true, title: "刷新中...")
        PortalService.shared.refreshJdYyb(openid: "all") { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.setButtonLoading(button, loading: false, title: "刷新全部CK")
                switch result {
                case .failure(let error):
                    self.handle(error)
                case .success(let data):
                    self.showYybRefreshResult(data)
                    if (data.success ?? 0) > 0 {
                        self.flashButtonSuccess(button, message: "✅ 完成", restore: "刷新全部CK")
                        self.loadAccounts()
                    }
                }
            }
        }
    }

    private func continueYybRisk(button: UIButton) {
        setButtonLoading(button, loading: true, title: "刷新中...")
        PortalService.shared.continueJdYybRisk { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.setButtonLoading(button, loading: false, title: "验证完成，继续刷新")
                switch result {
                case .failure(let error):
                    self.handle(error)
                case .success(let data):
                    self.showYybRefreshResult(data)
                    if data.needRiskVerify != true {
                        self.flashButtonSuccess(button, message: "✅ 刷新完成", restore: "验证完成，继续刷新")
                        self.loadAccounts()
                    }
                }
            }
        }
    }

    @objc private func openYybRiskUrl() {
        guard let raw = yybRiskUrl, let url = URL(string: raw) else { return }
        UIApplication.shared.open(url)
    }

    private func renderWx() {
        let tipCard = UIStackView()
        tipCard.axis = .vertical
        tipCard.spacing = 8
        tipCard.isLayoutMarginsRelativeArrangement = true
        tipCard.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        tipCard.applyCardStyle(cornerRadius: 14)
        let tipTitle = UILabel()
        tipTitle.text = "京东微信协议刷新"
        tipTitle.font = .systemFont(ofSize: 16, weight: .bold)
        let tip = UILabel()
        tip.text = "请先在「项目 → 协议接入 → 微信协议」扫码绑定在线设备，再选择设备刷新京东 CK。"
        tip.font = .systemFont(ofSize: 13)
        tip.textColor = .secondaryLabel
        tip.numberOfLines = 0
        tipCard.addArrangedSubview(tipTitle)
        tipCard.addArrangedSubview(tip)
        loginContentStack.addArrangedSubview(tipCard)

        let refreshBar = UIView()
        refreshBar.translatesAutoresizingMaskIntoConstraints = false
        var refreshBtn: UIButton!
        refreshBtn = compactButton("刷新设备列表", color: .secondaryLabel, filled: false) { [weak self] in
            self?.loadWxDevices(refreshButton: refreshBtn)
        }
        refreshBtn.translatesAutoresizingMaskIntoConstraints = false
        refreshBar.addSubview(refreshBtn)
        NSLayoutConstraint.activate([
            refreshBar.heightAnchor.constraint(equalToConstant: 36),
            refreshBtn.trailingAnchor.constraint(equalTo: refreshBar.trailingAnchor),
            refreshBtn.centerYAnchor.constraint(equalTo: refreshBar.centerYAnchor),
            refreshBtn.widthAnchor.constraint(greaterThanOrEqualToConstant: 110),
        ])
        loginContentStack.addArrangedSubview(refreshBar)

        wxDeviceGridStack.axis = .vertical
        wxDeviceGridStack.spacing = 10
        wxDeviceGridStack.addArrangedSubview(placeholderLabel("加载中..."))
        loginContentStack.addArrangedSubview(wxDeviceGridStack)

        wxRiskStack.axis = .vertical
        wxRiskStack.spacing = 8
        wxRiskStack.isHidden = true
        wxRiskLabel.numberOfLines = 0
        wxRiskStack.addArrangedSubview(wxRiskLabel)
        wxRiskLink.numberOfLines = 0
        wxRiskLink.font = .systemFont(ofSize: 13)
        wxRiskLink.textColor = .systemBlue
        wxRiskLink.isUserInteractionEnabled = true
        let tapGesture = UITapGestureRecognizer(target: self, action: #selector(openRiskUrl))
        wxRiskLink.addGestureRecognizer(tapGesture)
        wxRiskStack.addArrangedSubview(wxRiskLink)
        var riskBtn: UIButton!
        riskBtn = compactButton("验证完成，继续刷新", color: .systemOrange) { [weak self] in
            self?.continueWxRisk(button: riskBtn)
        }
        wxRiskStack.addArrangedSubview(riskBtn)
        loginContentStack.addArrangedSubview(wxRiskStack)

        wxResultLabel.numberOfLines = 0
        wxResultLabel.font = .systemFont(ofSize: 13)
        wxResultLabel.textColor = .secondaryLabel
        wxResultLabel.isHidden = true
        loginContentStack.addArrangedSubview(wxResultLabel)

        loadWxDevices()
    }

    private func loadWxDevices(refreshButton: UIButton? = nil) {
        setButtonLoading(refreshButton, loading: true, title: "刷新中...")
        PortalService.shared.fetchJdWxDevices { [weak self] result in
            guard let self = self else { return }
            DispatchQueue.main.async {
                self.setButtonLoading(refreshButton, loading: false, title: "刷新设备列表")
                guard self.mainTab == .login, self.loginTab == .wx else { return }
                self.wxDeviceGridStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
                switch result {
                case .failure(let error):
                    self.handle(error)
                    self.wxDeviceGridStack.addArrangedSubview(self.placeholderLabel(error.message))
                case .success(let list):
                    if let btn = refreshButton {
                        self.flashButtonSuccess(btn, message: "✅ 已刷新", restore: "刷新设备列表")
                    }
                    if list.isEmpty {
                        self.wxDeviceGridStack.addArrangedSubview(self.placeholderLabel("暂无在线微信协议设备"))
                        return
                    }
                    var idx = 0
                    while idx < list.count {
                        let row = UIStackView()
                        row.axis = .horizontal
                        row.spacing = 10
                        row.distribution = .fillEqually
                        row.addArrangedSubview(self.buildWxDeviceCard(list[idx]))
                        if idx + 1 < list.count {
                            row.addArrangedSubview(self.buildWxDeviceCard(list[idx + 1]))
                        } else {
                            row.addArrangedSubview(UIView())
                        }
                        self.wxDeviceGridStack.addArrangedSubview(row)
                        idx += 2
                    }
                }
            }
        }
    }

    private func buildWxDeviceCard(_ device: PortalJdWxDevice) -> UIView {
        let wrap = UIView()
        wrap.applyCardStyle(cornerRadius: 14)
        wrap.translatesAutoresizingMaskIntoConstraints = false

        let name = UILabel()
        name.text = nonEmpty(device.nickname) ?? nonEmpty(device.wxid) ?? "未知设备"
        name.font = .systemFont(ofSize: 14, weight: .semibold)
        name.numberOfLines = 1
        name.translatesAutoresizingMaskIntoConstraints = false

        let wxid = UILabel()
        wxid.text = nonEmpty(device.wxid) ?? ""
        wxid.font = .systemFont(ofSize: 11)
        wxid.textColor = .secondaryLabel
        wxid.numberOfLines = 2
        wxid.translatesAutoresizingMaskIntoConstraints = false

        let jdNick = UILabel()
        if let jd = nonEmpty(device.jdNickname) {
            jdNick.text = "京东 \(jd)"
            jdNick.isHidden = false
        } else {
            jdNick.text = ""
            jdNick.isHidden = true
        }
        jdNick.font = .systemFont(ofSize: 11, weight: .medium)
        jdNick.textColor = .systemOrange
        jdNick.numberOfLines = 1
        jdNick.lineBreakMode = .byTruncatingTail
        jdNick.translatesAutoresizingMaskIntoConstraints = false

        let online = UILabel()
        online.text = nonEmpty(device.device) ?? nonEmpty(device.serverType) ?? "微信设备"
        online.font = .systemFont(ofSize: 11)
        online.textColor = .secondaryLabel
        online.translatesAutoresizingMaskIntoConstraints = false

        var btn: UIButton!
        btn = compactButton("刷新 CK", color: .systemTeal) { [weak self] in
            self?.refreshWx(device.wxid ?? "", button: btn)
        }
        btn.translatesAutoresizingMaskIntoConstraints = false

        wrap.addSubview(name)
        wrap.addSubview(wxid)
        wrap.addSubview(jdNick)
        wrap.addSubview(online)
        wrap.addSubview(btn)
        NSLayoutConstraint.activate([
            wrap.heightAnchor.constraint(greaterThanOrEqualToConstant: 118),
            name.topAnchor.constraint(equalTo: wrap.topAnchor, constant: 12),
            name.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 12),
            name.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            wxid.topAnchor.constraint(equalTo: name.bottomAnchor, constant: 4),
            wxid.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            wxid.trailingAnchor.constraint(equalTo: name.trailingAnchor),
            jdNick.topAnchor.constraint(equalTo: wxid.bottomAnchor, constant: 4),
            jdNick.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            jdNick.trailingAnchor.constraint(equalTo: name.trailingAnchor),
            online.topAnchor.constraint(equalTo: jdNick.bottomAnchor, constant: 4),
            online.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            online.trailingAnchor.constraint(equalTo: name.trailingAnchor),
            btn.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 10),
            btn.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -10),
            btn.bottomAnchor.constraint(equalTo: wrap.bottomAnchor, constant: -10),
            btn.heightAnchor.constraint(equalToConstant: 32),
            btn.topAnchor.constraint(greaterThanOrEqualTo: online.bottomAnchor, constant: 8),
        ])
        return wrap
    }

    private func refreshWx(_ wxid: String, button: UIButton) {
        guard !wxid.isEmpty else { return }
        guard button.isEnabled else { return }
        setButtonLoading(button, loading: true, title: "刷新中...")
        PortalService.shared.refreshJdWx(wxid: wxid) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.setButtonLoading(button, loading: false, title: "刷新 CK")
                switch result {
                case .failure(let error): self.handle(error)
                case .success(let data):
                    self.flashButtonSuccess(button, message: "✅ 已刷新", restore: "刷新 CK")
                    self.wxRiskStack.isHidden = data.needRiskVerify != true
                    if data.needRiskVerify == true {
                        self.wxRiskLabel.text = data.riskMsg ?? "账号需要短信验证"
                        self.currentRiskUrl = data.riskUrl
                        if let url = data.riskUrl, !url.isEmpty {
                            self.wxRiskLink.text = url
                            self.wxRiskLink.isHidden = false
                        } else {
                            self.wxRiskLink.isHidden = true
                        }
                    }
                    self.wxResultLabel.text = "成功 \(data.success ?? 0) / 失败 \(data.fail ?? 0)\n\(data.details?.joined(separator: "\n") ?? "")"
                    self.wxResultLabel.isHidden = false
                }
            }
        }
    }

    private var currentRiskUrl: String?

    @objc private func openRiskUrl() {
        guard let urlStr = currentRiskUrl, let url = URL(string: urlStr) else { return }
        UIApplication.shared.open(url)
    }

    private func continueWxRisk(button: UIButton) {
        guard button.isEnabled else { return }
        setButtonLoading(button, loading: true, title: "处理中...")
        PortalService.shared.continueJdWxRisk { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.setButtonLoading(button, loading: false, title: "验证完成，继续刷新")
                switch result {
                case .failure(let error): self.handle(error)
                case .success(let data):
                    self.flashButtonSuccess(button, message: "✅ 刷新完成", restore: "验证完成，继续刷新")
                    if data.needRiskVerify == true {
                        self.wxRiskStack.isHidden = false
                        self.wxRiskLabel.text = data.riskMsg ?? "账号需要短信验证"
                        self.currentRiskUrl = data.riskUrl
                        if let url = data.riskUrl, !url.isEmpty {
                            self.wxRiskLink.text = url
                            self.wxRiskLink.isHidden = false
                        }
                    } else {
                        self.wxRiskStack.isHidden = true
                    }
                    self.wxResultLabel.text = data.details?.joined(separator: "\n") ?? "刷新完成"
                    self.wxResultLabel.isHidden = false
                }
            }
        }
    }

    // MARK: - 京东任务

    private func renderTasks() {
        taskSearchBar.delegate = self
        taskSearchBar.placeholder = "搜索任务名称、ID…"
        taskSearchBar.searchBarStyle = .minimal
        taskSearchBar.text = taskSearchQuery.isEmpty ? nil : taskSearchQuery
        taskContentStack.addArrangedSubview(taskSearchBar)

        taskGridStack.axis = .vertical
        taskGridStack.spacing = 10
        taskContentStack.addArrangedSubview(taskGridStack)

        loadTaskTabData()

        let logCard = UIStackView()
        logCard.axis = .vertical
        logCard.spacing = 10
        logCard.isLayoutMarginsRelativeArrangement = true
        logCard.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        logCard.applyCardStyle(cornerRadius: 14)

        let logHeader = UIStackView()
        logHeader.axis = .horizontal
        logHeader.distribution = .equalSpacing
        let logTitle = UILabel()
        logTitle.text = "执行日志"
        logTitle.font = .systemFont(ofSize: 15, weight: .bold)
        logHeader.addArrangedSubview(logTitle)
        logHeader.addArrangedSubview(compactButton("清空", color: .secondaryLabel, filled: false) { [weak self] in
            self?.logLines = []
            self?.refreshLogs()
        })
        logCard.addArrangedSubview(logHeader)

        logTextView.isEditable = false
        logTextView.isScrollEnabled = false   // 禁用内部滚动，让外层 scrollView 统一滚动
        logTextView.font = .monospacedSystemFont(ofSize: 11, weight: .regular)
        logTextView.text = "暂无日志，请执行任务"
        logTextView.backgroundColor = UIColor.tertiarySystemBackground
        logTextView.layer.cornerRadius = 8
        logTextView.textContainerInset = UIEdgeInsets(top: 8, left: 8, bottom: 8, right: 8)
        logCard.addArrangedSubview(logTextView)
        taskContentStack.addArrangedSubview(logCard)
        refreshLogs()
    }

    private func loadTaskTabData() {
        PortalService.shared.fetchJdAccounts { [weak self] result in
            guard let self = self else { return }
            PortalService.shared.fetchJdTasks { [weak self] taskResult in
                guard let self = self else { return }
                PortalService.shared.fetchJdProxyStatus { [weak self] proxyResult in
                    DispatchQueue.main.async {
                        guard let self = self, self.mainTab == .task else { return }
                        if case .success(let list) = result {
                            self.accounts = list
                            if self.globalTaskAccountSelection.isEmpty {
                                self.globalTaskAccountSelection = self.defaultAccountSelection()
                            }
                            self.renderAccountChips()
                        }
                        if case .success(let items) = taskResult {
                            self.taskDefs = items.compactMap { item in
                                guard let id = item.id?.trimmingCharacters(in: .whitespacesAndNewlines), !id.isEmpty else { return nil }
                                let name = (item.name?.trimmingCharacters(in: .whitespacesAndNewlines)).flatMap { $0.isEmpty ? nil : $0 } ?? id
                                let coin = item.coin ?? 0
                                return TaskDef(id: id, name: name, icon: "⚡", desc: coin > 0 ? "每账号扣 \(coin) 积分" : "免费")
                            }
                        } else {
                            self.taskDefs = []
                        }
                        if case .success(let status) = proxyResult {
                            self.jdProxyStatus = status
                        } else {
                            self.jdProxyStatus = nil
                        }
                        self.updateProxyCard()
                        self.renderTaskGrid()
                    }
                }
            }
        }
    }

    private func defaultAccountSelection() -> Set<Int> {
        let validCount = accounts.filter { $0.valid }.count
        return validCount > 0 ? [0] : []
    }

    private func setupTaskStickyBar() {
        taskStickyBar.axis = .vertical
        taskStickyBar.spacing = 6
        taskStickyBar.isLayoutMarginsRelativeArrangement = true
        taskStickyBar.layoutMargins = UIEdgeInsets(top: 8, left: 10, bottom: 8, right: 10)
        taskStickyBar.applyCardStyle(cornerRadius: 12)
        taskStickyBar.translatesAutoresizingMaskIntoConstraints = false
        taskStickyBar.isHidden = true

        let accountRow = UIStackView()
        accountRow.axis = .horizontal
        accountRow.spacing = 8
        accountRow.alignment = .center

        let accountTitle = UILabel()
        accountTitle.text = "执行账号"
        accountTitle.font = .systemFont(ofSize: 11, weight: .bold)
        accountTitle.setContentHuggingPriority(.required, for: .horizontal)

        let scroll = UIScrollView()
        scroll.showsHorizontalScrollIndicator = false
        scroll.translatesAutoresizingMaskIntoConstraints = false
        accountChipsStack = UIStackView()
        accountChipsStack?.axis = .horizontal
        accountChipsStack?.spacing = 6
        accountChipsStack?.translatesAutoresizingMaskIntoConstraints = false
        scroll.addSubview(accountChipsStack!)
        NSLayoutConstraint.activate([
            accountChipsStack!.topAnchor.constraint(equalTo: scroll.contentLayoutGuide.topAnchor),
            accountChipsStack!.leadingAnchor.constraint(equalTo: scroll.contentLayoutGuide.leadingAnchor),
            accountChipsStack!.trailingAnchor.constraint(equalTo: scroll.contentLayoutGuide.trailingAnchor),
            accountChipsStack!.bottomAnchor.constraint(equalTo: scroll.contentLayoutGuide.bottomAnchor),
            accountChipsStack!.heightAnchor.constraint(equalTo: scroll.frameLayoutGuide.heightAnchor),
            scroll.heightAnchor.constraint(equalToConstant: 30),
        ])

        let accountHint = UILabel()
        accountHint.text = "可多选"
        accountHint.font = .systemFont(ofSize: 10)
        accountHint.textColor = .secondaryLabel
        accountHint.setContentHuggingPriority(.required, for: .horizontal)

        accountRow.addArrangedSubview(accountTitle)
        accountRow.addArrangedSubview(scroll)
        accountRow.addArrangedSubview(accountHint)
        taskStickyBar.addArrangedSubview(accountRow)
        taskStickyBar.addArrangedSubview(buildCompactProxyCard())
    }

    private func buildCompactProxyCard() -> UIView {
        let row = UIStackView()
        row.axis = .horizontal
        row.spacing = 6
        row.alignment = .center

        let title = UILabel()
        title.text = "任务代理"
        title.font = .systemFont(ofSize: 11, weight: .bold)
        title.setContentHuggingPriority(.required, for: .horizontal)

        proxyBadgeLabel = UILabel()
        proxyBadgeLabel?.text = "未开通"
        proxyBadgeLabel?.font = .systemFont(ofSize: 9, weight: .semibold)
        proxyBadgeLabel?.textColor = .secondaryLabel
        proxyBadgeLabel?.backgroundColor = UIColor.secondarySystemGroupedBackground
        proxyBadgeLabel?.layer.cornerRadius = 8
        proxyBadgeLabel?.clipsToBounds = true
        proxyBadgeLabel?.textAlignment = .center
        proxyBadgeLabel?.setContentHuggingPriority(.required, for: .horizontal)

        proxyStatsStack = UIStackView()
        proxyStatsStack?.axis = .horizontal
        proxyStatsStack?.spacing = 8
        proxyStatsStack?.isHidden = true
        proxyStatsStack?.addArrangedSubview(proxyStatView(label: "到期", value: "-", tag: 100))
        proxyStatsStack?.addArrangedSubview(proxyStatView(label: "积分", value: "-", tag: 101))

        proxyMetaLabel = UILabel()
        proxyMetaLabel?.font = .systemFont(ofSize: 10)
        proxyMetaLabel?.textColor = .secondaryLabel
        proxyMetaLabel?.numberOfLines = 1
        proxyMetaLabel?.lineBreakMode = .byTruncatingTail
        proxyMetaLabel?.text = "加载中..."
        proxyMetaLabel?.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        let months = makeProxyMonthsButton()

        proxyBuyButton = compactButton("购买", color: .systemBlue) { [weak self] in
            self?.buyJdProxy()
        }
        proxyBuyButton?.titleLabel?.font = .systemFont(ofSize: 11, weight: .semibold)
        proxyBuyButton?.setContentHuggingPriority(.required, for: .horizontal)

        row.addArrangedSubview(title)
        row.addArrangedSubview(proxyBadgeLabel!)
        row.addArrangedSubview(proxyStatsStack!)
        row.addArrangedSubview(proxyMetaLabel!)
        row.addArrangedSubview(months)
        row.addArrangedSubview(proxyBuyButton!)
        return row
    }

    private func makeProxyMonthsButton() -> UIButton {
        let btn = UIButton(type: .system)
        btn.tag = 200
        btn.titleLabel?.font = .systemFont(ofSize: 11, weight: .medium)
        btn.setTitleColor(.label, for: .normal)
        btn.backgroundColor = UIColor.secondarySystemGroupedBackground
        btn.layer.cornerRadius = 8
        btn.layer.borderWidth = 1
        btn.layer.borderColor = UIColor.systemGray4.cgColor
        btn.contentEdgeInsets = UIEdgeInsets(top: 5, left: 10, bottom: 5, right: 10)
        btn.setContentHuggingPriority(.required, for: .horizontal)
        proxyMonthsButton = btn
        updateProxyMonthsButtonTitle()
        if #available(iOS 14.0, *) {
            btn.showsMenuAsPrimaryAction = true
            btn.menu = makeProxyMonthsMenu()
        } else {
            btn.addAction(UIAction { [weak self] _ in self?.presentProxyMonthsPicker(from: btn) }, for: .touchUpInside)
        }
        return btn
    }

    @available(iOS 14.0, *)
    private func makeProxyMonthsMenu() -> UIMenu {
        UIMenu(children: [1, 3, 6, 12].map { month in
            UIAction(
                title: "\(month)个月",
                state: proxyPurchaseMonths == month ? .on : .off
            ) { [weak self] _ in
                self?.proxyPurchaseMonths = month
                self?.updateProxyMonthsButtonTitle()
                self?.proxyMonthsButton?.menu = self?.makeProxyMonthsMenu()
            }
        })
    }

    private func updateProxyMonthsButtonTitle() {
        proxyMonthsButton?.setTitle("\(proxyPurchaseMonths)个月 ▾", for: .normal)
    }

    private func presentProxyMonthsPicker(from source: UIButton) {
        let sheet = UIAlertController(title: "选择购买月数", message: nil, preferredStyle: .actionSheet)
        [1, 3, 6, 12].forEach { month in
            let mark = month == proxyPurchaseMonths ? " ✓" : ""
            sheet.addAction(UIAlertAction(title: "\(month)个月\(mark)", style: .default) { [weak self] _ in
                self?.proxyPurchaseMonths = month
                self?.updateProxyMonthsButtonTitle()
            })
        }
        sheet.addAction(UIAlertAction(title: "取消", style: .cancel))
        if let pop = sheet.popoverPresentationController {
            pop.sourceView = source
            pop.sourceRect = source.bounds
        }
        present(sheet, animated: true)
    }

    private func proxyStatView(label: String, value: String, tag: Int) -> UIView {
        let wrap = UIStackView()
        wrap.axis = .horizontal
        wrap.spacing = 2
        wrap.tag = tag
        let title = UILabel()
        title.text = label
        title.font = .systemFont(ofSize: 9)
        title.textColor = .secondaryLabel
        let val = UILabel()
        val.text = value
        val.font = .systemFont(ofSize: 10, weight: .bold)
        val.tag = 1
        wrap.addArrangedSubview(title)
        wrap.addArrangedSubview(val)
        return wrap
    }

    private func renderAccountChips() {
        guard let stack = accountChipsStack else { return }
        stack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        if globalTaskAccountSelection.isEmpty {
            globalTaskAccountSelection = defaultAccountSelection()
        }

        func addChip(title: String, value: Int) {
            let selected = globalTaskAccountSelection.contains(value)
            let btn = UIButton(type: .system)
            btn.setTitle(title, for: .normal)
            btn.titleLabel?.font = .systemFont(ofSize: 11, weight: .medium)
            btn.setTitleColor(selected ? .white : .secondaryLabel, for: .normal)
            btn.backgroundColor = selected ? .systemBlue : UIColor.secondarySystemGroupedBackground
            btn.layer.cornerRadius = 12
            btn.contentEdgeInsets = UIEdgeInsets(top: 4, left: 10, bottom: 4, right: 10)
            btn.tag = value
            btn.addAction(UIAction { [weak self] _ in
                guard let self = self else { return }
                if value == 0 {
                    self.globalTaskAccountSelection = selected ? [] : [0]
                } else {
                    self.globalTaskAccountSelection.remove(0)
                    if selected { self.globalTaskAccountSelection.remove(value) } else { self.globalTaskAccountSelection.insert(value) }
                    if self.globalTaskAccountSelection.isEmpty { self.globalTaskAccountSelection = [0] }
                }
                self.renderAccountChips()
            }, for: .touchUpInside)
            stack.addArrangedSubview(btn)
        }

        addChip(title: "所有有效账号", value: 0)
        var validIdx = 1
        accounts.filter { $0.valid }.forEach { acc in
            addChip(title: acc.nickname ?? acc.pin ?? "账号\(validIdx)", value: validIdx)
            validIdx += 1
        }
    }

    private func updateProxyCard() {
        let st = jdProxyStatus
        let monthly = st?.monthlyCoin ?? 0
        let active = st?.active == true
        let ready = st?.proxyReady == true
        proxyBadgeLabel?.text = active ? "已开通" : "未开通"
        proxyBadgeLabel?.textColor = active ? UIColor.systemGreen : .secondaryLabel
        proxyBadgeLabel?.backgroundColor = active ? UIColor.systemGreen.withAlphaComponent(0.15) : UIColor.secondarySystemGroupedBackground

        guard ready, monthly > 0 else {
            proxyStatsStack?.isHidden = true
            proxyMetaLabel?.text = monthly <= 0 ? "管理员尚未开放代理购买" : "管理员尚未配置任务代理，暂不可购买"
            proxyBuyButton?.isHidden = true
            return
        }

        proxyStatsStack?.isHidden = false
        proxyBuyButton?.isHidden = false
        (proxyStatsStack?.viewWithTag(100)?.viewWithTag(1) as? UILabel)?.text = active ? (st?.expireAt ?? "-") : "未开通"
        (proxyStatsStack?.viewWithTag(101)?.viewWithTag(1) as? UILabel)?.text = "\(st?.userCoin ?? 0)"
        proxyMetaLabel?.text = active ? "走代理" : "直连 · \(monthly)积分/月"
        proxyBuyButton?.setTitle(active ? "续费" : "购买", for: .normal)
    }

    private func buyJdProxy() {
        let months = proxyPurchaseMonths
        let monthly = jdProxyStatus?.monthlyCoin ?? 0
        let need = monthly * months
        guard monthly > 0 else { showMessage("代理订阅暂未开放"); return }
        guard (jdProxyStatus?.userCoin ?? 0) >= need else {
            showMessage("积分不足，需要 \(need) 积分")
            return
        }
        let isRenew = jdProxyStatus?.active == true
        let alert = UIAlertController(
            title: isRenew ? "续费任务代理" : "购买任务代理",
            message: "确认购买 \(months) 个月任务代理？将扣除 \(need) 积分。",
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        alert.addAction(UIAlertAction(title: "确认", style: .default) { [weak self] _ in
            guard let self = self, let btn = self.proxyBuyButton else { return }
            self.setButtonLoading(btn, loading: true, title: "购买中...")
            PortalService.shared.buyJdProxy(months: months) { result in
                DispatchQueue.main.async {
                    self.setButtonLoading(btn, loading: false, title: self.jdProxyStatus?.active == true ? "续费" : "购买")
                    switch result {
                    case .failure(let error): self.handle(error)
                    case .success(let status):
                        self.jdProxyStatus = status
                        self.updateProxyCard()
                        let expire = (status.expireAt ?? "-").prefix(10)
                        let action = isRenew ? "续费" : "购买"
                        self.showMessage(
                            "已扣除 \(need) 积分，\(action) \(months) 个月任务代理。\n到期时间：\(expire)\n执行任务将自动使用代理线路。",
                            title: "购买成功"
                        )
                    }
                }
            }
        })
        present(alert, animated: true)
    }

    private func globalAccountIndexes() -> [Int] {
        if globalTaskAccountSelection.isEmpty || globalTaskAccountSelection.contains(0) {
            return [0]
        }
        return Array(globalTaskAccountSelection).sorted()
    }

    private func filteredTaskDefs() -> [TaskDef] {
        let q = taskSearchQuery.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !q.isEmpty else { return taskDefs }
        return taskDefs.filter { task in
            task.name.lowercased().contains(q)
                || task.id.lowercased().contains(q)
                || task.desc.lowercased().contains(q)
        }
    }

    private func renderTaskGrid() {
        taskGridStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        let tasks = filteredTaskDefs()
        if tasks.isEmpty {
            if !taskSearchQuery.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                taskGridStack.addArrangedSubview(placeholderLabel("未找到匹配「\(taskSearchQuery)」的任务"))
            } else {
                taskGridStack.addArrangedSubview(placeholderLabel("暂无可用任务，请联系管理员在后台启用"))
            }
            return
        }
        var idx = 0
        while idx < tasks.count {
            let row = UIStackView()
            row.axis = .horizontal
            row.spacing = 10
            row.distribution = .fillEqually
            row.addArrangedSubview(buildTaskCard(tasks[idx]))
            if idx + 1 < tasks.count {
                row.addArrangedSubview(buildTaskCard(tasks[idx + 1]))
            } else {
                row.addArrangedSubview(UIView())
            }
            taskGridStack.addArrangedSubview(row)
            idx += 2
        }
        runningTasks.keys.forEach { updateTaskButton($0) }
    }

    func searchBar(_ searchBar: UISearchBar, textDidChange searchText: String) {
        guard searchBar === taskSearchBar else { return }
        taskSearchQuery = searchText
        renderTaskGrid()
    }

    func searchBarSearchButtonClicked(_ searchBar: UISearchBar) {
        searchBar.resignFirstResponder()
    }

    private func buildTaskCard(_ task: TaskDef) -> UIView {
        let wrap = UIStackView()
        wrap.axis = .vertical
        wrap.spacing = 8
        wrap.isLayoutMarginsRelativeArrangement = true
        wrap.layoutMargins = UIEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
        wrap.applyCardStyle(cornerRadius: 14)

        let header = UILabel()
        header.text = "\(task.icon) \(task.name)"
        header.font = .systemFont(ofSize: 14, weight: .bold)
        wrap.addArrangedSubview(header)

        let desc = UILabel()
        desc.text = task.desc
        desc.font = .systemFont(ofSize: 11)
        desc.textColor = .secondaryLabel
        desc.numberOfLines = 2
        wrap.addArrangedSubview(desc)

        let idLabel = UILabel()
        idLabel.text = "ID: \(task.id)"
        idLabel.font = .systemFont(ofSize: 11)
        idLabel.textColor = .secondaryLabel
        wrap.addArrangedSubview(idLabel)

        let running = runningTasks[task.id] != nil
        var execBtn: UIButton!
        execBtn = compactButton(running ? "停止" : "执行", color: running ? .systemRed : .systemBlue) { [weak self] in
            guard let self = self else { return }
            if self.runningTasks[task.id] != nil {
                self.stopTask(task, button: execBtn)
            } else {
                self.executeTask(task, button: execBtn)
            }
        }
        taskButtonRefs[task.id] = execBtn
        wrap.addArrangedSubview(execBtn)

        return wrap
    }

    private func updateTaskButton(_ taskId: String) {
        guard let btn = taskButtonRefs[taskId] else { return }
        let running = runningTasks[taskId] != nil
        btn.isEnabled = true
        btn.alpha = 1
        btn.setTitle(running ? "停止" : "执行", for: .normal)
        btn.backgroundColor = running ? UIColor.systemRed : UIColor.systemBlue
    }

    private func executeTask(_ task: TaskDef, button: UIButton) {
        let selection = globalAccountIndexes()
        guard !selection.isEmpty else { showMessage("请至少选择一个账号"); return }
        guard button.isEnabled else { return }
        setButtonLoading(button, loading: true, title: "启动中...")
        if jdProxyStatus?.active == true {
            appendLog("[\(task.name)] 已开通任务代理，本次执行将使用代理")
        }
        appendLog("[\(task.name)] 开始执行任务...")
        PortalService.shared.executeJdTask(taskId: task.id, taskName: task.name, accountIndexes: selection) { [weak self] result in
            DispatchQueue.main.async {
                guard let self = self else { return }
                switch result {
                case .failure(let error):
                    self.setButtonLoading(button, loading: false, title: "执行")
                    self.handle(error)
                    self.appendLog("[\(task.name)] 启动失败: \(error.message)")
                case .success(let data):
                    guard let logId = data.taskId else {
                        self.setButtonLoading(button, loading: false, title: "执行")
                        self.appendLog("[\(task.name)] 任务启动失败")
                        return
                    }
                    self.runningTasks[task.id] = logId
                    self.setButtonLoading(button, loading: false, title: "停止")
                    self.updateTaskButton(task.id)
                    self.appendLog("[\(task.name)] 任务已启动，连接日志流...")
                    self.logStreamers[task.id]?.cancel()
                    let streamer = PortalService.shared.streamJdTaskLogs(taskId: logId, onLine: { line in
                        self.appendLog("[\(task.name)] \(line)")
                    }, onDone: {
                        self.appendLog("[\(task.name)] ✅ 任务执行完成")
                        self.runningTasks.removeValue(forKey: task.id)
                        self.logStreamers.removeValue(forKey: task.id)
                        self.updateTaskButton(task.id)
                    }, onError: { error in
                        self.appendLog("[\(task.name)] 错误: \(error.message)")
                        self.runningTasks.removeValue(forKey: task.id)
                        self.logStreamers.removeValue(forKey: task.id)
                        self.updateTaskButton(task.id)
                    })
                    self.logStreamers[task.id] = streamer
                }
            }
        }
    }

    private func stopTask(_ task: TaskDef, button: UIButton) {
        guard let logId = runningTasks[task.id] else { return }
        guard button.isEnabled else { return }
        setButtonLoading(button, loading: true, title: "停止中...")
        logStreamers[task.id]?.cancel()
        logStreamers.removeValue(forKey: task.id)
        PortalService.shared.stopJdTask(taskId: logId) { [weak self] _ in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.runningTasks.removeValue(forKey: task.id)
                self.setButtonLoading(button, loading: false, title: "执行")
                self.updateTaskButton(task.id)
                self.flashButtonSuccess(button, message: "✅ 已停止", restore: "执行")
                self.appendLog("[\(task.name)] ⚠️ 任务已手动停止")
            }
        }
    }

    private func appendLog(_ line: String) {
        logLines.append(line)
        if logLines.count > 200 { logLines.removeFirst() }
        refreshLogs()
    }

    private func refreshLogs() {
        logTextView.text = logLines.isEmpty ? "暂无日志，请执行任务" : logLines.joined(separator: "\n")
        // 自动滚动到底部
        if !logLines.isEmpty {
            let bottom = NSMakeRange(logTextView.text.count - 1, 1)
            logTextView.scrollRangeToVisible(bottom)
        }
    }

    // MARK: - UI Helpers

    /// 二级菜单：色块按钮（与项目页二级子页操作区风格一致）
    private func segmentButton(_ title: String, active: Bool, action: @escaping () -> Void) -> UIButton {
        let btn = UIButton(type: .system)
        btn.setTitle(title, for: .normal)
        btn.titleLabel?.font = .systemFont(ofSize: 13, weight: active ? .bold : .medium)
        btn.titleLabel?.numberOfLines = 2
        btn.titleLabel?.textAlignment = .center
        btn.setTitleColor(active ? .white : .secondaryLabel, for: .normal)
        btn.backgroundColor = active ? .systemBlue : UIColor.secondarySystemBackground
        btn.layer.cornerRadius = 10
        btn.contentEdgeInsets = UIEdgeInsets(top: 10, left: 6, bottom: 10, right: 6)
        bindAction(btn, action)
        return btn
    }

    private func compactButton(_ title: String, color: UIColor, filled: Bool = true, action: @escaping () -> Void) -> UIButton {
        let btn = UIButton(type: .system)
        btn.setTitle(title, for: .normal)
        if filled {
            btn.backgroundColor = color == .secondaryLabel ? UIColor.secondarySystemBackground : color
            btn.setTitleColor(color == .secondaryLabel ? .label : .white, for: .normal)
            if color == .secondaryLabel {
                btn.layer.borderWidth = 1
                btn.layer.borderColor = UIColor.systemGray4.cgColor
            }
        } else {
            btn.backgroundColor = UIColor.secondarySystemBackground
            btn.setTitleColor(.label, for: .normal)
            btn.layer.borderWidth = 1
            btn.layer.borderColor = UIColor.systemGray4.cgColor
        }
        btn.titleLabel?.font = .systemFont(ofSize: 13, weight: .semibold)
        btn.layer.cornerRadius = 8
        btn.contentEdgeInsets = UIEdgeInsets(top: 6, left: 12, bottom: 6, right: 12)
        btn.heightAnchor.constraint(equalToConstant: 34).isActive = true
        bindAction(btn, action)
        return btn
    }

    private func placeholderLabel(_ text: String) -> UILabel {
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 13)
        label.textColor = .secondaryLabel
        label.numberOfLines = 0
        label.textAlignment = .center
        return label
    }

    private func bindAction(_ button: UIButton, _ action: @escaping () -> Void) {
        buttonActions[ObjectIdentifier(button)] = action
        button.addTarget(self, action: #selector(buttonTapped(_:)), for: .touchUpInside)
        attachPressAnimation(to: button)
    }

    @objc private func buttonTapped(_ sender: UIButton) {
        guard sender.isEnabled else { return }
        buttonHaptic.impactOccurred()
        buttonActions[ObjectIdentifier(sender)]?()
    }

    @objc private func buttonTouchDown(_ sender: UIButton) {
        guard sender.isEnabled else { return }
        UIView.animate(withDuration: 0.12, delay: 0, options: [.allowUserInteraction, .beginFromCurrentState]) {
            sender.transform = CGAffineTransform(scaleX: 0.94, y: 0.94)
            sender.alpha = sender.isEnabled ? 0.82 : sender.alpha
        }
    }

    @objc private func buttonTouchUp(_ sender: UIButton) {
        UIView.animate(withDuration: 0.16, delay: 0, options: [.allowUserInteraction, .beginFromCurrentState]) {
            sender.transform = .identity
            if sender.isEnabled {
                sender.alpha = 1
            }
        }
    }

    private func attachPressAnimation(to button: UIButton) {
        button.addTarget(self, action: #selector(buttonTouchDown(_:)), for: .touchDown)
        button.addTarget(self, action: #selector(buttonTouchUp(_:)), for: [.touchUpInside, .touchUpOutside, .touchCancel, .touchDragExit])
    }

    private func setButtonLoading(_ button: UIButton?, loading: Bool, title: String? = nil) {
        guard let button = button else { return }
        let key = ObjectIdentifier(button)
        if loading {
            if buttonOriginalTitles[key] == nil {
                buttonOriginalTitles[key] = button.title(for: .normal)
            }
            button.isEnabled = false
            UIView.animate(withDuration: 0.15) {
                button.alpha = 0.65
                button.transform = .identity
            }
            if let title = title {
                button.setTitle(title, for: .normal)
            }
        } else {
            button.isEnabled = true
            let restore = title ?? buttonOriginalTitles[key] ?? button.title(for: .normal) ?? ""
            button.setTitle(restore, for: .normal)
            buttonOriginalTitles.removeValue(forKey: key)
            UIView.animate(withDuration: 0.15) {
                button.alpha = 1
            }
        }
    }

    private func flashButtonSuccess(_ button: UIButton, message: String, restore: String, delay: TimeInterval = 1.2) {
        button.isEnabled = false
        button.setTitle(message, for: .normal)
        UIView.animate(withDuration: 0.18, animations: {
            button.transform = CGAffineTransform(scaleX: 1.05, y: 1.05)
        }) { _ in
            UIView.animate(withDuration: 0.18) {
                button.transform = .identity
            }
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + delay) { [weak button] in
            guard let button = button else { return }
            button.isEnabled = true
            button.setTitle(restore, for: .normal)
        }
    }

    var innerTabCount: Int { mainSegmented.numberOfSegments }

    var innerTabIndex: Int { mainSegmented.selectedSegmentIndex }

    func selectInnerTab(at index: Int) {
        guard index >= 0, index < innerTabCount else { return }
        mainSegmented.selectedSegmentIndex = index
        mainSegmentChanged()
    }
}

@available(iOS 13.0, *)
private final class QueryResultViewController: UIViewController {
    init(title: String, content: String) {
        super.init(nibName: nil, bundle: nil)
        self.title = title
        let label = UILabel()
        label.text = content
        label.numberOfLines = 0
        label.font = .systemFont(ofSize: 13)
        label.translatesAutoresizingMaskIntoConstraints = false
        view.backgroundColor = .systemBackground
        view.addSubview(label)
        NSLayoutConstraint.activate([
            label.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 16),
            label.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            label.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
        ])
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }
}
