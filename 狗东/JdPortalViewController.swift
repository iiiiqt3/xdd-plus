import UIKit

@available(iOS 13.0, *)
final class JdPortalViewController: BaseNativeViewController {
    private enum MainTab { case query, login, task }
    private enum LoginTab { case sms, wx }

    private struct TaskDef {
        let id: String
        let name: String
        let icon: String
        let desc: String
    }

    private let scrollView = UIScrollView()
    private let stack = UIStackView()
    private let headerRow = UIView()
    private let mainSegmented = UISegmentedControl(items: ["查询", "登录", "京东任务"])
    private let loginTabRow = UIStackView()
    private let contentStack = UIStackView()

    private var mainTab: MainTab = .query
    private var loginTab: LoginTab = .sms
    private var accounts: [PortalJdAccount] = []
    private var taskSelections: [String: Set<Int>] = [:]
    private var runningTasks: [String: String] = [:]
    private var logStreamers: [String: JdTaskLogStreamer] = [:]
    private var taskButtonRefs: [String: UIButton] = [:]
    private var logLines: [String] = []
    private let logLabel = UILabel()
    private var buttonActions: [ObjectIdentifier: () -> Void] = [:]
    private var buttonOriginalTitles: [ObjectIdentifier: String] = [:]
    private var isQueryingAll = false
    private let buttonHaptic = UIImpactFeedbackGenerator(style: .light)

    private let smsPhoneField = UITextField()
    private let smsCodeField = UITextField()
    private let smsIdCardField = UITextField()
    private let smsIdCardStack = UIStackView()
    private let smsResultLabel = UILabel()
    private let wxDeviceGridStack = UIStackView()
    private let wxRiskStack = UIStackView()
    private let wxRiskLabel = UILabel()
    private let wxRiskLink = UILabel()
    private let wxResultLabel = UILabel()

    private let taskDefs: [TaskDef] = [
        TaskDef(id: "plantBean", name: "种豆得豆", icon: "🫘", desc: "种豆得豆任务"),
        TaskDef(id: "dwapp", name: "话费积分", icon: "📱", desc: "话费积分签到"),
        TaskDef(id: "price", name: "一键保价", icon: "💰", desc: "自动保价退款"),
        TaskDef(id: "autoEval", name: "一键评价", icon: "⭐", desc: "自动评价订单"),
        TaskDef(id: "insight", name: "问卷调查", icon: "📝", desc: "问卷调查得豆"),
    ]

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupLayout()
        buttonHaptic.prepare()
        renderAll()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        if mainTab == .query { loadAccounts() }
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
        scrollView.keyboardDismissMode = .onDrag
        stack.axis = .vertical
        stack.spacing = 14
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
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -20),
            stack.widthAnchor.constraint(equalTo: view.widthAnchor, constant: -32),
        ])

        mainSegmented.selectedSegmentIndex = 0
        mainSegmented.addTarget(self, action: #selector(mainSegmentChanged), for: .valueChanged)
        stack.addArrangedSubview(mainSegmented)

        loginTabRow.axis = .horizontal
        loginTabRow.spacing = 8
        loginTabRow.distribution = .fillEqually
        contentStack.axis = .vertical
        contentStack.spacing = 12
        contentStack.alignment = .fill

        stack.addArrangedSubview(loginTabRow)
        stack.addArrangedSubview(contentStack)
    }

    @objc private func mainSegmentChanged() {
        switch mainSegmented.selectedSegmentIndex {
        case 0: mainTab = .query
        case 1: mainTab = .login
        default: mainTab = .task
        }
        renderAll()
        if mainTab == .query { loadAccounts() }
    }

    private func renderAll() {
        syncMainSegmentIndex()
        loginTabRow.isHidden = mainTab != .login
        stack.setCustomSpacing(mainTab == .login ? 8 : 14, after: mainSegmented)
        contentStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        taskButtonRefs.removeAll()
        switch mainTab {
        case .query: renderQuery()
        case .login: renderLogin()
        case .task: renderTasks()
        }
    }

    private func syncMainSegmentIndex() {
        switch mainTab {
        case .query: mainSegmented.selectedSegmentIndex = 0
        case .login: mainSegmented.selectedSegmentIndex = 1
        case .task: mainSegmented.selectedSegmentIndex = 2
        }
    }

    // MARK: - 查询

    private func renderQuery() {
        contentStack.addArrangedSubview(placeholderLabel("加载账号中..."))
        loadAccounts()
    }

    private func loadAccounts(refreshButton: UIButton? = nil) {
        setButtonLoading(refreshButton, loading: true, title: "刷新中...")
        PortalService.shared.fetchJdAccounts { [weak self] result in
            guard let self = self else { return }
            DispatchQueue.main.async {
                self.setButtonLoading(refreshButton, loading: false, title: "刷新")
                self.contentStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
                switch result {
                case .failure(let error):
                    self.handle(error)
                    self.contentStack.addArrangedSubview(self.placeholderLabel(error.message))
                case .success(let list):
                    self.accounts = list
                    self.renderAccountGrid(list)
                }
            }
        }
    }

    private func renderAccountGrid(_ list: [PortalJdAccount]) {
        if list.isEmpty {
            contentStack.addArrangedSubview(placeholderLabel("暂无绑定的京东账号\n请前往「登录」使用短信或微信协议刷新"))
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
        contentStack.addArrangedSubview(toolbar)

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
            contentStack.addArrangedSubview(row)
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
        loginTabRow.isHidden = false
        loginTabRow.arrangedSubviews.forEach { $0.removeFromSuperview() }
        loginTabRow.addArrangedSubview(segmentButton("短信登录", active: loginTab == .sms) { [weak self] in
            self?.loginTab = .sms
            self?.renderAll()
        })
        loginTabRow.addArrangedSubview(segmentButton("协议刷新", active: loginTab == .wx) { [weak self] in
            self?.loginTab = .wx
            self?.renderAll()
        })

        switch loginTab {
        case .sms: renderSms()
        case .wx: renderWx()
        }
    }

    private func renderSms() {
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
        smsPhoneField.heightAnchor.constraint(equalToConstant: 48).isActive = true
        card.addArrangedSubview(smsPhoneField)

        var sendBtn: UIButton!
        sendBtn = compactButton("发送验证码", color: .systemBlue) { [weak self] in
            self?.sendSms(button: sendBtn)
        }
        card.addArrangedSubview(sendBtn)

        smsCodeField.applyAppInputStyle(placeholder: "请输入短信验证码")
        smsCodeField.keyboardType = .numberPad
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
        smsIdCardField.heightAnchor.constraint(equalToConstant: 48).isActive = true
        smsIdCardStack.addArrangedSubview(smsIdCardField)
        card.addArrangedSubview(smsIdCardStack)

        var verifyBtn: UIButton!
        verifyBtn = compactButton("提交登录", color: .systemPurple) { [weak self] in
            self?.verifySms(button: verifyBtn)
        }
        card.addArrangedSubview(verifyBtn)

        smsResultLabel.numberOfLines = 0
        smsResultLabel.font = .systemFont(ofSize: 13)
        smsResultLabel.textColor = .secondaryLabel
        smsResultLabel.isHidden = true
        card.addArrangedSubview(smsResultLabel)

        contentStack.addArrangedSubview(card)
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
        tip.text = "请先在「项目 → 微信协议」扫码绑定在线设备，再选择设备刷新京东 CK。"
        tip.font = .systemFont(ofSize: 13)
        tip.textColor = .secondaryLabel
        tip.numberOfLines = 0
        tipCard.addArrangedSubview(tipTitle)
        tipCard.addArrangedSubview(tip)
        contentStack.addArrangedSubview(tipCard)

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
        contentStack.addArrangedSubview(refreshBar)

        wxDeviceGridStack.axis = .vertical
        wxDeviceGridStack.spacing = 10
        wxDeviceGridStack.addArrangedSubview(placeholderLabel("加载中..."))
        contentStack.addArrangedSubview(wxDeviceGridStack)

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
        contentStack.addArrangedSubview(wxRiskStack)

        wxResultLabel.numberOfLines = 0
        wxResultLabel.font = .systemFont(ofSize: 13)
        wxResultLabel.textColor = .secondaryLabel
        wxResultLabel.isHidden = true
        contentStack.addArrangedSubview(wxResultLabel)

        loadWxDevices()
    }

    private func loadWxDevices(refreshButton: UIButton? = nil) {
        setButtonLoading(refreshButton, loading: true, title: "刷新中...")
        PortalService.shared.fetchJdWxDevices { [weak self] result in
            guard let self = self else { return }
            DispatchQueue.main.async {
                self.setButtonLoading(refreshButton, loading: false, title: "刷新设备列表")
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
        name.text = device.nickname ?? device.wxid ?? "未知设备"
        name.font = .systemFont(ofSize: 14, weight: .semibold)
        name.numberOfLines = 1
        name.translatesAutoresizingMaskIntoConstraints = false

        let wxid = UILabel()
        wxid.text = device.wxid ?? ""
        wxid.font = .systemFont(ofSize: 11)
        wxid.textColor = .secondaryLabel
        wxid.numberOfLines = 2
        wxid.translatesAutoresizingMaskIntoConstraints = false

        let online = UILabel()
        online.text = device.device ?? device.serverType ?? "微信设备"
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
        wrap.addSubview(online)
        wrap.addSubview(btn)
        NSLayoutConstraint.activate([
            wrap.heightAnchor.constraint(greaterThanOrEqualToConstant: 108),
            name.topAnchor.constraint(equalTo: wrap.topAnchor, constant: 12),
            name.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 12),
            name.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -12),
            wxid.topAnchor.constraint(equalTo: name.bottomAnchor, constant: 4),
            wxid.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            wxid.trailingAnchor.constraint(equalTo: name.trailingAnchor),
            online.topAnchor.constraint(equalTo: wxid.bottomAnchor, constant: 6),
            online.leadingAnchor.constraint(equalTo: name.leadingAnchor),
            btn.leadingAnchor.constraint(equalTo: wrap.leadingAnchor, constant: 10),
            btn.trailingAnchor.constraint(equalTo: wrap.trailingAnchor, constant: -10),
            btn.bottomAnchor.constraint(equalTo: wrap.bottomAnchor, constant: -10),
            btn.heightAnchor.constraint(equalToConstant: 32),
            btn.topAnchor.constraint(greaterThanOrEqualTo: online.bottomAnchor, constant: 10),
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
        let hint = placeholderLabel("选择任务和账号，点击执行开始（每行 2 个任务）")
        contentStack.addArrangedSubview(hint)

        let gridPlaceholder = UIStackView()
        gridPlaceholder.axis = .vertical
        gridPlaceholder.spacing = 10
        gridPlaceholder.tag = 9001
        contentStack.addArrangedSubview(gridPlaceholder)

        PortalService.shared.fetchJdAccounts { [weak self] result in
            guard let self = self else { return }
            if case .success(let list) = result { self.accounts = list }
            gridPlaceholder.arrangedSubviews.forEach { $0.removeFromSuperview() }
            var idx = 0
            while idx < self.taskDefs.count {
                let row = UIStackView()
                row.axis = .horizontal
                row.spacing = 10
                row.distribution = .fillEqually
                row.addArrangedSubview(self.buildTaskCard(self.taskDefs[idx]))
                if idx + 1 < self.taskDefs.count {
                    row.addArrangedSubview(self.buildTaskCard(self.taskDefs[idx + 1]))
                } else {
                    row.addArrangedSubview(UIView())
                }
                gridPlaceholder.addArrangedSubview(row)
                idx += 2
            }
        }

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

        logLabel.numberOfLines = 0
        logLabel.font = .monospacedSystemFont(ofSize: 11, weight: .regular)
        logLabel.text = "暂无日志，请执行任务"
        logLabel.backgroundColor = UIColor.tertiarySystemBackground
        logLabel.layer.cornerRadius = 8
        logLabel.clipsToBounds = true
        logCard.addArrangedSubview(logLabel)
        contentStack.addArrangedSubview(logCard)
        refreshLogs()
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

        let selection = taskSelections[task.id] ?? []
        let label = UILabel()
        label.text = accountSelectionLabel(selection)
        label.font = .systemFont(ofSize: 11)
        label.textColor = .secondaryLabel
        label.numberOfLines = 2
        wrap.addArrangedSubview(label)

        let pickBtn = compactButton("选账号", color: .secondaryLabel, filled: false) { [weak self] in
            self?.pickAccounts(taskId: task.id, label: label)
        }
        wrap.addArrangedSubview(pickBtn)

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
        let selection = Array(taskSelections[task.id] ?? [])
        guard !selection.isEmpty else { showMessage("请至少选择一个账号"); return }
        guard button.isEnabled else { return }
        setButtonLoading(button, loading: true, title: "启动中...")
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

    private func pickAccounts(taskId: String, label: UILabel) {
        let options = buildAccountOptions()
        let alert = UIAlertController(title: "选择账号", message: nil, preferredStyle: .actionSheet)
        var selection = taskSelections[taskId] ?? []
        options.forEach { idx, name in
            let selected = selection.contains(idx)
            alert.addAction(UIAlertAction(title: (selected ? "✓ " : "") + name, style: .default) { [weak self] _ in
                if idx == 0 { selection = [0] }
                else {
                    selection.remove(0)
                    if selection.contains(idx) { selection.remove(idx) } else { selection.insert(idx) }
                }
                self?.taskSelections[taskId] = selection
                label.text = self?.accountSelectionLabel(selection)
            })
        }
        alert.addAction(UIAlertAction(title: "完成", style: .cancel))
        if let pop = alert.popoverPresentationController {
            pop.sourceView = view
            pop.sourceRect = CGRect(x: view.bounds.midX, y: view.bounds.midY, width: 1, height: 1)
        }
        present(alert, animated: true)
    }

    private func buildAccountOptions() -> [(Int, String)] {
        var options: [(Int, String)] = [(0, "所有有效账号")]
        var validIdx = 1
        accounts.filter { $0.valid }.forEach { acc in
            options.append((validIdx, acc.nickname ?? acc.pin ?? "账号\(validIdx)"))
            validIdx += 1
        }
        return options
    }

    private func accountSelectionLabel(_ selection: Set<Int>) -> String {
        if selection.contains(0) { return "已选：所有有效账号" }
        let names = buildAccountOptions().filter { selection.contains($0.0) }.map { $0.1 }
        return names.isEmpty ? "未选择账号" : "已选：" + names.joined(separator: "、")
    }

    private func appendLog(_ line: String) {
        logLines.append(line)
        if logLines.count > 200 { logLines.removeFirst() }
        refreshLogs()
    }

    private func refreshLogs() {
        logLabel.text = logLines.isEmpty ? "暂无日志，请执行任务" : logLines.joined(separator: "\n")
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
