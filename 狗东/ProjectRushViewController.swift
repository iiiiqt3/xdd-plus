import UIKit

enum KuwoTimeHelper {
    static let withdrawHours = [0, 9, 13, 17, 20]

    struct BeijingTime {
        let hour: Int
        let min: Int
        let sec: Int
        let totalMs: Int64
    }

    struct NextWithdrawInfo {
        let hour: Int
        let inWindow: Bool
        let diffMin: Int
    }

    static func getBeijingTime() -> BeijingTime {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(identifier: "Asia/Shanghai") ?? .current
        let now = Date()
        let h = cal.component(.hour, from: now)
        let m = cal.component(.minute, from: now)
        let s = cal.component(.second, from: now)
        let ms = Int64((h * 3600 + m * 60 + s)) * 1000 + Int64(cal.component(.nanosecond, from: now) / 1_000_000)
        return BeijingTime(hour: h, min: m, sec: s, totalMs: ms)
    }

    static func minutesUntilHour(_ targetHour: Int, hour: Int, min: Int) -> Int {
        var diffMin = targetHour * 60 - (hour * 60 + min)
        if diffMin < 0 { diffMin += 24 * 60 }
        return diffMin
    }

    /// 到目标整点剩余毫秒；整点后 graceAfterMin 分钟内返回 0（执行窗口）
    static func remainingMsUntilHour(_ targetHour: Int, graceAfterMin: Int = 30) -> Int64 {
        let bj = getBeijingTime()
        let targetMs = Int64(targetHour) * 3600 * 1000
        var remaining = targetMs - bj.totalMs
        if remaining <= 0 {
            let pastMs = -remaining
            if pastMs <= Int64(graceAfterMin) * 60 * 1000 {
                return 0
            }
            remaining += 24 * 3600 * 1000
        }
        return max(remaining, 0)
    }

    static func getNextWithdrawInfo() -> NextWithdrawInfo {
        let bj = getBeijingTime()
        var bestNext: NextWithdrawInfo?
        for h in withdrawHours {
            let diffMin = minutesUntilHour(h, hour: bj.hour, min: bj.min)
            if diffMin > 0 && diffMin <= 4 {
                return NextWithdrawInfo(hour: h, inWindow: true, diffMin: diffMin)
            }
            if diffMin == 0 {
                return NextWithdrawInfo(hour: h, inWindow: true, diffMin: 0)
            }
            if bestNext == nil || diffMin < bestNext!.diffMin {
                bestNext = NextWithdrawInfo(hour: h, inWindow: false, diffMin: diffMin)
            }
        }
        if let bestNext = bestNext {
            return bestNext
        }
        let fallbackHour = withdrawHours[0]
        return NextWithdrawInfo(
            hour: fallbackHour,
            inWindow: false,
            diffMin: minutesUntilHour(fallbackHour, hour: bj.hour, min: bj.min)
        )
    }

    static func formatClock(_ h: Int, _ m: Int, _ s: Int) -> String {
        String(format: "%02d:%02d:%02d", h, m, s)
    }

    static func formatHour(_ h: Int) -> String {
        String(format: "%02d:00", h)
    }
}

final class KuwoRushViewController: BaseNativeViewController {
    private let scrollView = UIScrollView()
    private let contentStack = UIStackView()

    private let authHint = UILabel()
    private let accountCheckboxStack = UIStackView()
    private let singleAccountFieldsWrap = UIStackView()
    private let multiSmsWrap = UIStackView()
    private let multiSmsStack = UIStackView()
    private let phoneField = UITextField()
    private let passwordField = UITextField()
    private let smsField = UITextField()
    private let smsStatus = UILabel()
    private let nowTimeLabel = UILabel()
    private let nextTimeLabel = UILabel()
    private let timeHintLabel = UILabel()
    private let withdrawButton = UIButton(type: .system)
    private let updateSmsButton = UIButton(type: .system)
    private let smsEditHint = UILabel()
    private let countdownLabel = UILabel()
    private let logTextView = UITextView()
    private let quotaStack = UIStackView()
    private let useProxySwitch = UISwitch()

    private var kuwoAuthorized = false
    private var kuwoAccounts: [(phone: String, password: String)] = []
    private var selectedAccountIndices = Set<Int>()
    private var accountToggleButtons: [UIButton] = []
    private var multiSmsFields: [String: UITextField] = [:]
    private var selectedQuotaId = "30002"
    private var activeTaskId: String?
    private var withdrawSubmitting = false
    private var taskLogIndex = 0
    private var monitorTimer: Timer?
    private var clockTimer: Timer?
    private var countdownTimer: Timer?

    private let quotaOptions: [(id: String, title: String)] = [
        ("30002", "2元"),
        ("60004", "1元"),
        ("60001", "10元"),
    ]

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "酷我提现"
        navigationItem.hidesBackButton = true
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        navigationController?.interactivePopGestureRecognizer?.isEnabled = true
        navigationController?.interactivePopGestureRecognizer?.delegate = nil
        loadKuwoPage()
        startClock()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        stopTimers()
    }

    private func setupUI() {
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        contentStack.axis = .vertical
        contentStack.spacing = 12
        contentStack.isLayoutMarginsRelativeArrangement = true
        contentStack.layoutMargins = UIEdgeInsets(top: 8, left: 16, bottom: 24, right: 16)
        contentStack.translatesAutoresizingMaskIntoConstraints = false
        scrollView.addSubview(contentStack)
        view.addSubview(scrollView)
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            contentStack.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor),
            contentStack.leadingAnchor.constraint(equalTo: scrollView.contentLayoutGuide.leadingAnchor),
            contentStack.trailingAnchor.constraint(equalTo: scrollView.contentLayoutGuide.trailingAnchor),
            contentStack.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor),
            contentStack.widthAnchor.constraint(equalTo: scrollView.frameLayoutGuide.widthAnchor),
        ])

        let subtitle = UILabel()
        subtitle.text = "定时抢兑，与网页端功能一致"
        subtitle.font = .systemFont(ofSize: 12)
        subtitle.textColor = .secondaryLabel
        contentStack.addArrangedSubview(subtitle)

        authHint.font = .systemFont(ofSize: 12.5, weight: .semibold)
        authHint.textColor = .systemRed
        authHint.numberOfLines = 0
        authHint.isHidden = true
        authHint.backgroundColor = UIColor.systemRed.withAlphaComponent(0.08)
        authHint.layer.cornerRadius = 10
        authHint.clipsToBounds = true
        authHint.layoutMargins = UIEdgeInsets(top: 10, left: 12, bottom: 10, right: 12)
        contentStack.addArrangedSubview(paddedView(authHint, insets: UIEdgeInsets(top: 0, left: 0, bottom: 0, right: 0)))

        contentStack.addArrangedSubview(buildAccountCard())
        contentStack.addArrangedSubview(buildSettingsCard())
        contentStack.addArrangedSubview(buildLogCard())

        restoreLogs()
        updateTimeDisplay()
    }

    private func buildAccountCard() -> UIView {
        let card = UIStackView()
        card.axis = .vertical
        card.spacing = 10
        card.isLayoutMarginsRelativeArrangement = true
        card.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        card.applyCardStyle(cornerRadius: 14)

        let title = UILabel()
        title.text = "账号配置"
        title.font = .systemFont(ofSize: 14, weight: .bold)
        card.addArrangedSubview(title)

        let tip = UILabel()
        tip.text = "账号密码自动读取自「酷我音乐」上车信息，修改请前往「我的项目」。"
        tip.font = .systemFont(ofSize: 11.5)
        tip.textColor = .secondaryLabel
        tip.numberOfLines = 0
        card.addArrangedSubview(tip)

        accountCheckboxStack.axis = .vertical
        accountCheckboxStack.spacing = 6
        accountCheckboxStack.isHidden = true
        let accountLabel = UILabel()
        accountLabel.text = "选择抢兑账号（可多选）"
        accountLabel.font = .systemFont(ofSize: 11.5)
        accountLabel.textColor = .secondaryLabel
        card.addArrangedSubview(accountLabel)
        card.addArrangedSubview(accountCheckboxStack)

        singleAccountFieldsWrap.axis = .vertical
        singleAccountFieldsWrap.spacing = 10
        singleAccountFieldsWrap.addArrangedSubview(fieldBlock(label: "手机号", field: phoneField, secure: false, readonly: true))
        singleAccountFieldsWrap.addArrangedSubview(fieldBlock(label: "密码", field: passwordField, secure: true, readonly: true))
        singleAccountFieldsWrap.addArrangedSubview(fieldBlock(label: "短信验证码", field: smsField, secure: false, readonly: false))

        let smsRow = UIStackView()
        smsRow.axis = .horizontal
        smsRow.spacing = 10
        smsRow.alignment = .center
        let sendBtn = UIButton(type: .system)
        sendBtn.setTitle("发送验证码", for: .normal)
        sendBtn.titleLabel?.font = .systemFont(ofSize: 12, weight: .bold)
        sendBtn.backgroundColor = .systemBlue
        sendBtn.setTitleColor(.white, for: .normal)
        sendBtn.layer.cornerRadius = 8
        sendBtn.contentEdgeInsets = UIEdgeInsets(top: 8, left: 14, bottom: 8, right: 14)
        bindAction(sendBtn) { [weak self] in self?.sendSms() }
        smsStatus.font = .systemFont(ofSize: 11)
        smsStatus.textColor = .secondaryLabel
        smsStatus.numberOfLines = 2
        smsRow.addArrangedSubview(sendBtn)
        smsRow.addArrangedSubview(smsStatus)
        singleAccountFieldsWrap.addArrangedSubview(smsRow)
        card.addArrangedSubview(singleAccountFieldsWrap)

        multiSmsWrap.axis = .vertical
        multiSmsWrap.spacing = 8
        multiSmsWrap.isHidden = true
        let multiLabel = UILabel()
        multiLabel.text = "各账号验证码"
        multiLabel.font = .systemFont(ofSize: 11.5)
        multiLabel.textColor = .secondaryLabel
        multiSmsWrap.addArrangedSubview(multiLabel)
        multiSmsStack.axis = .vertical
        multiSmsStack.spacing = 8
        multiSmsWrap.addArrangedSubview(multiSmsStack)
        let multiSendBtn = UIButton(type: .system)
        multiSendBtn.setTitle("批量发送验证码", for: .normal)
        multiSendBtn.titleLabel?.font = .systemFont(ofSize: 12, weight: .bold)
        multiSendBtn.backgroundColor = .systemBlue
        multiSendBtn.setTitleColor(.white, for: .normal)
        multiSendBtn.layer.cornerRadius = 8
        multiSendBtn.contentEdgeInsets = UIEdgeInsets(top: 8, left: 14, bottom: 8, right: 14)
        bindAction(multiSendBtn) { [weak self] in self?.sendSms() }
        multiSmsWrap.addArrangedSubview(multiSendBtn)
        card.addArrangedSubview(multiSmsWrap)
        return card
    }

    private func buildSettingsCard() -> UIView {
        let card = UIStackView()
        card.axis = .vertical
        card.spacing = 12
        card.isLayoutMarginsRelativeArrangement = true
        card.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        card.applyCardStyle(cornerRadius: 14)

        let title = UILabel()
        title.text = "提现设置"
        title.font = .systemFont(ofSize: 14, weight: .bold)
        card.addArrangedSubview(title)

        let quotaTitle = UILabel()
        quotaTitle.text = "抢兑档位"
        quotaTitle.font = .systemFont(ofSize: 11.5)
        quotaTitle.textColor = .secondaryLabel
        card.addArrangedSubview(quotaTitle)

        quotaStack.axis = .horizontal
        quotaStack.spacing = 8
        quotaStack.distribution = .fillEqually
        for (idx, opt) in quotaOptions.enumerated() {
            let btn = UIButton(type: .system)
            btn.setTitle(opt.title, for: .normal)
            btn.tag = idx
            btn.layer.cornerRadius = 10
            btn.layer.borderWidth = 1.5
            btn.titleLabel?.font = .systemFont(ofSize: 13, weight: .semibold)
            let index = idx
            bindAction(btn) { [weak self] in self?.selectQuota(index: index) }
            quotaStack.addArrangedSubview(btn)
        }
        refreshQuotaButtons()
        card.addArrangedSubview(quotaStack)

        let proxyRow = UIStackView()
        proxyRow.axis = .horizontal
        proxyRow.alignment = .center
        proxyRow.spacing = 8
        let proxyLabel = UILabel()
        proxyLabel.text = "启用代理抢兑"
        proxyLabel.font = .systemFont(ofSize: 13)
        useProxySwitch.isOn = prefs().object(forKey: "kuwo_use_proxy") as? Bool ?? true
        useProxySwitch.onTintColor = .systemIndigo
        useProxySwitch.addTarget(self, action: #selector(useProxySwitchChanged), for: .valueChanged)
        proxyRow.addArrangedSubview(proxyLabel)
        proxyRow.addArrangedSubview(UIView())
        proxyRow.addArrangedSubview(useProxySwitch)
        card.addArrangedSubview(proxyRow)

        let timeBox = UIStackView()
        timeBox.axis = .vertical
        timeBox.spacing = 4
        timeBox.isLayoutMarginsRelativeArrangement = true
        timeBox.layoutMargins = UIEdgeInsets(top: 12, left: 12, bottom: 12, right: 12)
        timeBox.backgroundColor = UIColor.secondarySystemGroupedBackground
        timeBox.layer.cornerRadius = 10
        [nowTimeLabel, nextTimeLabel, timeHintLabel].forEach {
            $0.font = .systemFont(ofSize: 12.5)
            $0.numberOfLines = 0
            timeBox.addArrangedSubview($0)
        }
        timeHintLabel.font = .systemFont(ofSize: 12, weight: .semibold)
        card.addArrangedSubview(timeBox)

        let guide = UILabel()
        guide.text = "💡 抢兑说明\n• 抢兑时段：00:00、09:00、13:00、17:00、20:00\n• 抢兑时段前4分钟内开始，将倒计时到点抢兑\n• 非抢兑时段点击开始，将立即提交抢兑"
        guide.font = .systemFont(ofSize: 11.5)
        guide.textColor = .secondaryLabel
        guide.numberOfLines = 0
        guide.backgroundColor = UIColor.systemIndigo.withAlphaComponent(0.08)
        guide.layer.cornerRadius = 10
        guide.clipsToBounds = true
        card.addArrangedSubview(paddedView(guide, insets: UIEdgeInsets(top: 10, left: 12, bottom: 10, right: 12)))

        withdrawButton.setTitle("开始抢兑", for: .normal)
        withdrawButton.titleLabel?.font = .systemFont(ofSize: 14, weight: .bold)
        withdrawButton.backgroundColor = .systemIndigo
        withdrawButton.setTitleColor(.white, for: .normal)
        withdrawButton.layer.cornerRadius = 10
        withdrawButton.contentEdgeInsets = UIEdgeInsets(top: 12, left: 20, bottom: 12, right: 20)
        bindAction(withdrawButton) { [weak self] in self?.startWithdraw() }

        updateSmsButton.setTitle("更新验证码", for: .normal)
        updateSmsButton.titleLabel?.font = .systemFont(ofSize: 13, weight: .semibold)
        updateSmsButton.backgroundColor = UIColor.secondarySystemGroupedBackground
        updateSmsButton.setTitleColor(.label, for: .normal)
        updateSmsButton.layer.cornerRadius = 10
        updateSmsButton.layer.borderWidth = 1
        updateSmsButton.layer.borderColor = UIColor.systemGray4.cgColor
        updateSmsButton.contentEdgeInsets = UIEdgeInsets(top: 10, left: 16, bottom: 10, right: 16)
        updateSmsButton.isHidden = true
        bindAction(updateSmsButton) { [weak self] in self?.updateSmsCode() }

        smsEditHint.text = "倒计时中可修改上方验证码，改完后点击「更新验证码」同步到后端"
        smsEditHint.font = .systemFont(ofSize: 11.5, weight: .semibold)
        smsEditHint.textColor = .systemIndigo
        smsEditHint.numberOfLines = 0
        smsEditHint.isHidden = true

        countdownLabel.font = .systemFont(ofSize: 15, weight: .bold)
        countdownLabel.textColor = .white
        countdownLabel.textAlignment = .center
        countdownLabel.backgroundColor = .systemOrange
        countdownLabel.layer.cornerRadius = 10
        countdownLabel.clipsToBounds = true
        countdownLabel.isHidden = true
        countdownLabel.numberOfLines = 0

        card.addArrangedSubview(withdrawButton)
        card.addArrangedSubview(updateSmsButton)
        card.addArrangedSubview(smsEditHint)
        card.addArrangedSubview(countdownLabel)
        return card
    }

    private func buildLogCard() -> UIView {
        let card = UIStackView()
        card.axis = .vertical
        card.spacing = 8
        card.isLayoutMarginsRelativeArrangement = true
        card.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        card.applyCardStyle(cornerRadius: 14)

        let header = UIStackView()
        header.axis = .horizontal
        let logTitle = UILabel()
        logTitle.text = "执行日志"
        logTitle.font = .systemFont(ofSize: 14, weight: .bold)
        let clearBtn = UIButton(type: .system)
        clearBtn.setTitle("清空", for: .normal)
        clearBtn.titleLabel?.font = .systemFont(ofSize: 12)
        bindAction(clearBtn) { [weak self] in self?.clearLogs() }
        header.addArrangedSubview(logTitle)
        header.addArrangedSubview(UIView())
        header.addArrangedSubview(clearBtn)
        card.addArrangedSubview(header)

        let hint = UILabel()
        hint.text = "倒计时抢兑会实时同步后端每一步"
        hint.font = .systemFont(ofSize: 11)
        hint.textColor = .secondaryLabel
        card.addArrangedSubview(hint)

        logTextView.isEditable = false
        logTextView.font = .monospacedSystemFont(ofSize: 11, weight: .regular)
        logTextView.backgroundColor = UIColor.secondarySystemGroupedBackground
        logTextView.layer.cornerRadius = 8
        logTextView.textContainerInset = UIEdgeInsets(top: 8, left: 8, bottom: 8, right: 8)
        logTextView.heightAnchor.constraint(greaterThanOrEqualToConstant: 160).isActive = true
        card.addArrangedSubview(logTextView)
        return card
    }

    private func fieldBlock(label: String, field: UITextField, secure: Bool, readonly: Bool) -> UIView {
        let wrap = UIStackView()
        wrap.axis = .vertical
        wrap.spacing = 4
        let lbl = UILabel()
        lbl.text = label
        lbl.font = .systemFont(ofSize: 11.5)
        lbl.textColor = .secondaryLabel
        field.borderStyle = .roundedRect
        field.isSecureTextEntry = secure
        field.isEnabled = !readonly
        field.font = .systemFont(ofSize: 14)
        if readonly {
            field.backgroundColor = UIColor.secondarySystemGroupedBackground
            field.textColor = .secondaryLabel
        }
        wrap.addArrangedSubview(lbl)
        wrap.addArrangedSubview(field)
        return wrap
    }

    private func paddedView(_ child: UIView, insets: UIEdgeInsets) -> UIView {
        let box = UIView()
        child.translatesAutoresizingMaskIntoConstraints = false
        box.addSubview(child)
        NSLayoutConstraint.activate([
            child.topAnchor.constraint(equalTo: box.topAnchor, constant: insets.top),
            child.leadingAnchor.constraint(equalTo: box.leadingAnchor, constant: insets.left),
            child.trailingAnchor.constraint(equalTo: box.trailingAnchor, constant: -insets.right),
            child.bottomAnchor.constraint(equalTo: box.bottomAnchor, constant: -insets.bottom),
        ])
        return box
    }

    @objc private func useProxySwitchChanged() {
        prefs().set(useProxySwitch.isOn, forKey: "kuwo_use_proxy")
    }

    private func useKuwoProxy() -> Bool {
        useProxySwitch.isOn
    }

    private func selectQuota(index: Int) {
        guard index >= 0, index < quotaOptions.count else { return }
        selectedQuotaId = quotaOptions[index].id
        refreshQuotaButtons()
    }

    private func refreshQuotaButtons() {
        for (idx, btn) in quotaStack.arrangedSubviews.enumerated() {
            guard let button = btn as? UIButton else { continue }
            let selected = quotaOptions[idx].id == selectedQuotaId
            button.layer.borderColor = (selected ? UIColor.systemIndigo : UIColor.systemGray4).cgColor
            button.backgroundColor = selected ? UIColor.systemIndigo.withAlphaComponent(0.1) : UIColor.secondarySystemGroupedBackground
            button.setTitleColor(selected ? .systemIndigo : .label, for: .normal)
        }
    }

    private func prefs() -> UserDefaults { UserDefaults.standard }

    private func appendLog(_ msg: String) {
        let current = logTextView.text ?? ""
        logTextView.text = current.isEmpty ? msg : current + "\n" + msg
        prefs().set(logTextView.text, forKey: "kuwo_log")
        let bottom = NSRange(location: (logTextView.text as NSString).length - 1, length: 1)
        logTextView.scrollRangeToVisible(bottom)
    }

    private func restoreLogs() {
        logTextView.text = prefs().string(forKey: "kuwo_log") ?? ""
    }

    private func clearLogs() {
        logTextView.text = ""
        taskLogIndex = 0
        prefs().removeObject(forKey: "kuwo_log")
    }

    private func renderTaskLog(_ entry: KuwoTaskLog) {
        let icons = ["info": "ℹ️", "success": "✅", "warn": "⚠️", "error": "❌"]
        let icon = icons[entry.level ?? ""] ?? "•"
        var line = "[\(entry.time ?? "--")] \(icon) \(entry.message ?? "")"
        if let proxy = entry.proxyHost, !proxy.isEmpty {
            line += " [代理:\(proxy)]"
        }
        appendLog(line)
    }

    private func flushTaskLogs(_ logs: [KuwoTaskLog]?) {
        guard let logs = logs, logs.count > taskLogIndex else { return }
        for i in taskLogIndex..<logs.count { renderTaskLog(logs[i]) }
        taskLogIndex = logs.count
    }

    private func loadKuwoPage() {
        PortalService.shared.checkKuwoAuth { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let (authorized, msg)):
                self.kuwoAuthorized = authorized
                self.authHint.isHidden = authorized
                self.authHint.text = msg.isEmpty ? "酷我活动授权已到期，请前往我的项目续费后再使用" : msg
            }
        }
        PortalService.shared.fetchKuwoCredentials { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .failure:
                self.kuwoAccounts = []
                self.selectedAccountIndices.removeAll()
                self.accountCheckboxStack.isHidden = true
                self.phoneField.text = ""
                self.passwordField.text = ""
                self.updateSmsUiMode()
            case .success(let cred):
                self.applyKuwoCredentials(cred)
            }
        }
    }

    private func maskPhone(_ phone: String) -> String {
        guard phone.count >= 11 else { return phone }
        return String(phone.prefix(3)) + "****" + String(phone.suffix(4))
    }

    private func resolveKuwoAccounts(_ cred: KuwoCredentials) -> [(phone: String, password: String)] {
        let fromList = cred.accounts?.compactMap { acc -> (String, String)? in
            let phone = acc.phone?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            guard !phone.isEmpty else { return nil }
            return (phone, acc.password ?? "")
        } ?? []
        if !fromList.isEmpty { return fromList }
        let phone = cred.phone?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !phone.isEmpty else { return [] }
        return [(phone, cred.password ?? "")]
    }

    private func getSelectedAccounts() -> [(phone: String, password: String)] {
        selectedAccountIndices.sorted().compactMap { idx in
            guard idx >= 0, idx < kuwoAccounts.count else { return nil }
            return kuwoAccounts[idx]
        }
    }

    private func renderAccountCheckboxes() {
        accountCheckboxStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        accountToggleButtons.removeAll()
        guard !kuwoAccounts.isEmpty else {
            accountCheckboxStack.isHidden = true
            return
        }
        accountCheckboxStack.isHidden = false
        if selectedAccountIndices.isEmpty {
            selectedAccountIndices.insert(0)
        }
        for (idx, acc) in kuwoAccounts.enumerated() {
            let row = UIStackView()
            row.axis = .horizontal
            row.spacing = 8
            row.alignment = .center
            let btn = UIButton(type: .system)
            btn.contentHorizontalAlignment = .left
            btn.titleLabel?.font = .systemFont(ofSize: 13, weight: .medium)
            btn.tag = idx
            updateAccountToggleButton(btn, phone: acc.phone, selected: selectedAccountIndices.contains(idx))
            bindAction(btn) { [weak self] in
                guard let self = self else { return }
                let index = btn.tag
                if self.selectedAccountIndices.contains(index) {
                    self.selectedAccountIndices.remove(index)
                } else {
                    self.selectedAccountIndices.insert(index)
                }
                self.updateAccountToggleButton(btn, phone: self.kuwoAccounts[index].phone, selected: self.selectedAccountIndices.contains(index))
                self.updateSmsUiMode()
            }
            accountToggleButtons.append(btn)
            row.addArrangedSubview(btn)
            accountCheckboxStack.addArrangedSubview(row)
        }
    }

    private func updateAccountToggleButton(_ btn: UIButton, phone: String, selected: Bool) {
        let icon = selected ? "☑" : "☐"
        btn.setTitle("\(icon) \(maskPhone(phone))", for: .normal)
        btn.setTitleColor(selected ? .systemIndigo : .label, for: .normal)
    }

    private func updateSmsUiMode() {
        let selected = getSelectedAccounts()
        let multi = selected.count > 1
        singleAccountFieldsWrap.isHidden = multi || selected.isEmpty
        multiSmsWrap.isHidden = !multi
        if selected.count == 1 {
            phoneField.text = selected[0].phone
            passwordField.text = selected[0].password
        }
        if multi {
            multiSmsStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
            multiSmsFields.removeAll()
            selected.forEach { acc in
                let wrap = UIStackView()
                wrap.axis = .vertical
                wrap.spacing = 4
                let lbl = UILabel()
                lbl.text = maskPhone(acc.phone)
                lbl.font = .systemFont(ofSize: 11.5)
                lbl.textColor = .secondaryLabel
                let field = UITextField()
                field.borderStyle = .roundedRect
                field.placeholder = "验证码"
                field.font = .systemFont(ofSize: 14)
                multiSmsFields[acc.phone] = field
                wrap.addArrangedSubview(lbl)
                wrap.addArrangedSubview(field)
                multiSmsStack.addArrangedSubview(wrap)
            }
        }
    }

    private func collectSessionPayload() -> [(phone: String, password: String, smsCode: String)] {
        let selected = getSelectedAccounts()
        guard !selected.isEmpty else { return [] }
        let multi = selected.count > 1
        let globalSms = smsField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return selected.map { acc in
            let smsCode: String
            if multi {
                smsCode = multiSmsFields[acc.phone]?.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            } else {
                smsCode = globalSms
            }
            return (phone: acc.phone, password: acc.password, smsCode: smsCode)
        }
    }

    private func applyKuwoCredentials(_ cred: KuwoCredentials) {
        let previousPhone = phoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        kuwoAccounts = resolveKuwoAccounts(cred)
        if !previousPhone.isEmpty, let matched = kuwoAccounts.firstIndex(where: { $0.phone == previousPhone }) {
            selectedAccountIndices = [matched]
        }
        if selectedAccountIndices.isEmpty, !kuwoAccounts.isEmpty {
            selectedAccountIndices.insert(0)
        }
        renderAccountCheckboxes()
        updateSmsUiMode()
        let primaryPhone = getSelectedAccounts().first?.phone ?? cred.phone ?? ""
        guard !kuwoAccounts.isEmpty else {
            phoneField.text = cred.phone
            passwordField.text = cred.password
            restoreKuwoState(phone: cred.phone ?? "")
            return
        }
        restoreKuwoState(phone: primaryPhone)
    }

    private func startClock() {
        clockTimer?.invalidate()
        clockTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            self?.updateTimeDisplay()
        }
    }

    private func stopTimers() {
        clockTimer?.invalidate()
        countdownTimer?.invalidate()
    }

    private func updateTimeDisplay() {
        let bj = KuwoTimeHelper.getBeijingTime()
        let info = KuwoTimeHelper.getNextWithdrawInfo()
        let nextLabel = KuwoTimeHelper.formatHour(info.hour)
        nowTimeLabel.text = "🕐 当前北京时间：\(KuwoTimeHelper.formatClock(bj.hour, bj.min, bj.sec))"
        if info.inWindow && info.diffMin > 0 {
            nextTimeLabel.text = "⏰ 下次抢兑：\(nextLabel)（\(info.diffMin)分钟后）"
            timeHintLabel.text = "✅ 可点击【开始抢兑】自动倒计时到点提交"
            timeHintLabel.textColor = .systemGreen
        } else if info.inWindow && info.diffMin == 0 {
            nextTimeLabel.text = "⏰ 下次抢兑：\(nextLabel)（当前时段）"
            timeHintLabel.text = "🚀 当前为抢兑时段，点击可直接提交"
            timeHintLabel.textColor = .systemRed
        } else {
            nextTimeLabel.text = "⏰ 下次抢兑：\(nextLabel)（还有\(info.diffMin)分钟）"
            timeHintLabel.text = "💡 点击【开始抢兑】将立即提交（非倒计时时段）"
            timeHintLabel.textColor = .systemIndigo
        }
    }

    private func clearActiveState() {
        monitorTimer?.invalidate()
        countdownTimer?.invalidate()
        activeTaskId = nil
        prefs().removeObject(forKey: "kuwo_state")
    }

    private func sendSms() {
        guard kuwoAuthorized else { showMessage("酷我活动授权已到期，请前往我的项目续费"); return }
        let sessions = getSelectedAccounts()
        guard !sessions.isEmpty else { showMessage("请至少选择一个账号"); return }
        clearActiveState()
        restoreWithdrawUi()
        smsField.text = ""
        multiSmsFields.values.forEach { $0.text = "" }
        smsStatus.text = "正在登录并发送验证码..."
        appendLog("正在登录酷我账号...")
        if sessions.count == 1 {
            let acc = sessions[0]
            PortalService.shared.sendKuwoSms(phone: acc.phone, password: acc.password) { [weak self] result in
                guard let self = self else { return }
                switch result {
                case .failure(let error):
                    self.smsStatus.text = "❌ \(error.message)"
                    self.smsStatus.textColor = .systemRed
                    self.appendLog("发送失败: \(error.message)")
                    self.handle(error)
                case .success:
                    self.smsStatus.text = "✅ 已发送至 \(self.maskPhone(acc.phone))"
                    self.smsStatus.textColor = .systemGreen
                    self.appendLog("验证码已发送，请输入验证码")
                    self.showMessage("验证码已发送")
                }
            }
        } else {
            PortalService.shared.sendKuwoSmsBatch(sessions: sessions.map { ($0.phone, $0.password) }) { [weak self] result in
                guard let self = self else { return }
                switch result {
                case .failure(let error):
                    self.smsStatus.text = "❌ \(error.message)"
                    self.smsStatus.textColor = .systemRed
                    self.appendLog("批量发送失败: \(error.message)")
                    self.handle(error)
                case .success(let batch):
                    let lines = batch.accounts ?? []
                    let ok = lines.filter { $0.success == true }.count
                    let fail = lines.count - ok
                    self.smsStatus.text = "✅ 成功 \(ok) / 失败 \(fail)"
                    self.smsStatus.textColor = .systemGreen
                    lines.forEach { item in
                        let phone = item.phone ?? "?"
                        if item.success == true {
                            self.appendLog("[\(self.maskPhone(phone))] 验证码已发送")
                        } else {
                            self.appendLog("[\(self.maskPhone(phone))] 发送失败: \(item.message ?? "未知错误")")
                        }
                    }
                    self.showMessage("批量发送完成：成功 \(ok)，失败 \(fail)")
                }
            }
        }
    }

    private func startWithdraw() {
        guard kuwoAuthorized else { showMessage("酷我活动授权已到期，请前往我的项目续费"); return }
        guard !withdrawSubmitting else { showMessage("任务提交中，请稍候"); return }
        let sessions = collectSessionPayload()
        guard !sessions.isEmpty else { showMessage("请至少选择一个账号"); return }
        guard sessions.allSatisfy({ !$0.smsCode.isEmpty }) else { showMessage("请为每个账号输入验证码"); return }
        let primary = sessions[0]
        let fallbackSms = primary.smsCode
        let info = KuwoTimeHelper.getNextWithdrawInfo()
        if info.inWindow && info.diffMin > 0 && info.diffMin <= 4 {
            appendLog("🎯 提交定时抢兑任务，后端将在 \(KuwoTimeHelper.formatHour(info.hour)) 自动执行（3轮错峰）")
            setWithdrawUiLocked(true, allowSmsEdit: true)
            saveKuwoState(phone: primary.phone, password: primary.password, smsCode: fallbackSms, quotaId: selectedQuotaId, targetHour: info.hour, taskId: nil, immediate: false, useProxy: useKuwoProxy())
            scheduleOnBackend(sessions: sessions, fallbackSms: fallbackSms, quotaId: selectedQuotaId, targetHour: info.hour, info: info, immediate: false)
            return
        }
        appendLog("⚡ 立即提交抢兑...")
        setWithdrawUiLocked(true)
        scheduleOnBackend(sessions: sessions, fallbackSms: fallbackSms, quotaId: selectedQuotaId, targetHour: nil, info: nil, immediate: true)
    }

    private func scheduleOnBackend(sessions: [(phone: String, password: String, smsCode: String)], fallbackSms: String, quotaId: String, targetHour: Int?, info: KuwoTimeHelper.NextWithdrawInfo?, immediate: Bool) {
        guard !withdrawSubmitting else { return }
        withdrawSubmitting = true
        PortalService.shared.scheduleKuwoWithdrawSessions(sessions: sessions, quotaId: quotaId, fallbackSmsCode: fallbackSms, targetHour: targetHour, immediate: immediate, useProxy: useKuwoProxy()) { [weak self] result in
            guard let self = self else { return }
            self.withdrawSubmitting = false
            switch result {
            case .failure(let error):
                self.appendLog("❌ 提交失败: \(error.message)")
                self.handle(error)
                self.restoreWithdrawUi()
            case .success(let data):
                guard let taskId = data.taskId, !taskId.isEmpty else {
                    self.appendLog("❌ 提交失败：未返回任务ID")
                    self.restoreWithdrawUi()
                    return
                }
                self.appendLog((data.reused == true ? "♻️ 复用已有任务" : "✅ 任务已提交") + "，ID: \(taskId)")
                self.taskLogIndex = 0
                self.activeTaskId = taskId
                self.updateSavedTaskId(taskId: taskId, immediate: immediate, targetHour: targetHour)
                if !immediate { self.setWithdrawUiLocked(true, allowSmsEdit: true) }
                self.monitorTask(taskId: taskId, immediate: immediate)
                if !immediate, let info = info {
                    self.runCountdown(info: info, taskId: taskId)
                }
            }
        }
    }

    private func setWithdrawUiLocked(_ locked: Bool, allowSmsEdit: Bool = false) {
        withdrawButton.isHidden = locked
        countdownLabel.isHidden = !locked
        updateSmsButton.isHidden = !(locked && allowSmsEdit)
        smsEditHint.isHidden = !(locked && allowSmsEdit)
        phoneField.isEnabled = !locked
        passwordField.isEnabled = !locked
        useProxySwitch.isEnabled = !locked
        accountToggleButtons.forEach { $0.isEnabled = !locked }
        smsField.isEnabled = !locked || allowSmsEdit
        multiSmsFields.values.forEach { $0.isEnabled = !locked || allowSmsEdit }
    }

    private func restoreWithdrawUi() {
        withdrawSubmitting = false
        countdownTimer?.invalidate()
        activeTaskId = nil
        setWithdrawUiLocked(false)
        countdownLabel.text = ""
    }

    private func updateSmsCode() {
        guard let taskId = activeTaskId, !taskId.isEmpty else { showMessage("暂无进行中的任务"); return }
        let smsCode = smsField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !smsCode.isEmpty else { showMessage("请输入验证码"); return }
        PortalService.shared.updateKuwoSmsCode(taskId: taskId, smsCode: smsCode) { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .failure(let error):
                self.showMessage(error.message)
                self.handle(error)
            case .success:
                self.appendLog("✅ 验证码已同步到后端")
                self.showMessage("验证码已更新")
                if var dict = self.prefs().dictionary(forKey: "kuwo_state") {
                    dict["smsCode"] = smsCode
                    self.prefs().set(dict, forKey: "kuwo_state")
                }
            }
        }
    }

    private func monitorTask(taskId: String, immediate: Bool) {
        monitorTimer?.invalidate()
        var pollCount = 0
        var errorCount = 0
        let maxPoll = immediate ? 120 : 600
        let interval = immediate ? 0.25 : 0.8
        monitorTimer = Timer.scheduledTimer(withTimeInterval: interval, repeats: true) { [weak self] timer in
            guard let self = self else { timer.invalidate(); return }
            pollCount += 1
            PortalService.shared.fetchKuwoWithdrawStatus(taskId: taskId) { result in
                switch result {
                case .failure:
                    errorCount += 1
                    if errorCount > 20 {
                        timer.invalidate()
                        self.countdownLabel.text = "⏰ 网络异常，请返回页面查看结果"
                    }
                case .success(let task):
                    guard let task = task else { return }
                    errorCount = 0
                    if task.status == "pending" || task.status == "running" {
                        _ = self.applyTaskResult(task: task, immediate: immediate)
                        return
                    }
                    if self.applyTaskResult(task: task, immediate: immediate) {
                        timer.invalidate()
                    }
                }
            }
            if pollCount > maxPoll {
                self.countdownLabel.text = "⏰ 仍在等待后端，继续同步…"
            }
        }
    }

    @discardableResult
    private func applyTaskResult(task: KuwoWithdrawTask, immediate: Bool) -> Bool {
        flushTaskLogs(task.logs)
        switch task.status {
        case "pending":
            if !immediate {
                setWithdrawUiLocked(true, allowSmsEdit: true)
                if task.smsFatal == true {
                    appendLog("⚠️ 后端反馈疑似验证码错误，请修改后点击「更新验证码」")
                }
            }
            return false
        case "running":
            if immediate { countdownLabel.text = "🚀 后端正在执行..." }
            return false
        case "completed":
            monitorTimer?.invalidate()
            countdownTimer?.invalidate()
            countdownLabel.text = "✅ 抢兑已完成"
            appendLog("--- 任务结束 ---")
            restoreWithdrawUi()
            clearSavedState()
            return true
        case "failed":
            monitorTimer?.invalidate()
            countdownTimer?.invalidate()
            countdownLabel.text = "❌ 抢兑失败"
            appendLog("--- 任务失败 ---")
            restoreWithdrawUi()
            clearSavedState()
            return true
        default:
            return false
        }
    }

    private func runCountdown(info: KuwoTimeHelper.NextWithdrawInfo, taskId: String) {
        countdownTimer?.invalidate()
        let targetHour = info.hour
        let targetLabel = KuwoTimeHelper.formatHour(targetHour)
        var fired = false
        countdownTimer = Timer.scheduledTimer(withTimeInterval: 0.1, repeats: true) { [weak self] timer in
            guard let self = self else { timer.invalidate(); return }
            let remaining = KuwoTimeHelper.remainingMsUntilHour(targetHour)
            if remaining > 20 * 3600 * 1000 {
                if !fired {
                    fired = true
                    self.countdownLabel.text = "🚀 后端正在执行 \(targetLabel) 抢兑..."
                    self.monitorTask(taskId: taskId, immediate: false)
                }
                timer.invalidate()
                return
            }
            if remaining > 0 {
                let rMin = remaining / 60000
                let rSec = (remaining % 60000) / 1000
                self.countdownLabel.text = "⏳ \(targetLabel) 自动抢兑 — 剩余 \(rMin)分\(rSec)秒"
            } else if !fired {
                fired = true
                self.countdownLabel.text = "🚀 后端正在执行 \(targetLabel) 抢兑..."
                self.appendLog("⏰ 到达抢兑时间 \(targetLabel)，等待后端日志同步...")
                self.monitorTask(taskId: taskId, immediate: false)
                timer.invalidate()
            }
        }
    }

    private func saveKuwoState(phone: String, password: String, smsCode: String, quotaId: String, targetHour: Int?, taskId: String?, immediate: Bool, useProxy: Bool = true) {
        var dict: [String: Any] = [
            "phone": phone, "password": password, "smsCode": smsCode, "quotaId": quotaId,
            "immediate": immediate, "useProxy": useProxy, "savedAt": Date().timeIntervalSince1970,
        ]
        if let targetHour = targetHour { dict["targetHour"] = targetHour }
        if let taskId = taskId { dict["taskId"] = taskId }
        prefs().set(dict, forKey: "kuwo_state")
    }

    private func updateSavedTaskId(taskId: String, immediate: Bool, targetHour: Int?) {
        guard var dict = prefs().dictionary(forKey: "kuwo_state") else { return }
        dict["taskId"] = taskId
        dict["immediate"] = immediate
        if let targetHour = targetHour { dict["targetHour"] = targetHour }
        prefs().set(dict, forKey: "kuwo_state")
    }

    private func clearSavedState() {
        prefs().removeObject(forKey: "kuwo_state")
    }

    private func restoreKuwoState(phone: String) {
        guard let dict = prefs().dictionary(forKey: "kuwo_state") else { return }
        if let savedPhone = dict["phone"] as? String, !savedPhone.isEmpty, !phone.isEmpty, savedPhone != phone {
            clearActiveState()
            return
        }
        guard let taskIdFromState = dict["taskId"] as? String, !taskIdFromState.isEmpty else {
            clearActiveState()
            return
        }
        PortalService.shared.fetchKuwoWithdrawStatus(taskId: taskIdFromState) { [weak self] result in
            guard let self = self else { return }
            let task: KuwoWithdrawTask?
            switch result {
            case .success(let t): task = t
            case .failure: task = nil
            }
            if let task = task, task.status == "completed" || task.status == "failed" {
                self.taskLogIndex = task.logs?.count ?? 0
                _ = self.applyTaskResult(task: task, immediate: (dict["immediate"] as? Bool) ?? false)
                return
            }
            guard let task = task, task.status == "pending" || task.status == "running" else {
                self.clearActiveState()
                self.restoreWithdrawUi()
                return
            }
            self.activeTaskId = task.id ?? taskIdFromState
            self.phoneField.text = (dict["phone"] as? String).flatMap { $0.isEmpty ? nil : $0 } ?? phone
            self.passwordField.text = dict["password"] as? String
            if task.status == "pending" {
                self.smsField.text = dict["smsCode"] as? String
            }
            let quotaId = dict["quotaId"] as? String ?? "30002"
            self.selectedQuotaId = quotaId
            if let useProxy = dict["useProxy"] as? Bool {
                self.useProxySwitch.isOn = useProxy
                self.prefs().set(useProxy, forKey: "kuwo_use_proxy")
            }
            self.refreshQuotaButtons()
            self.appendLog("🔄 检测到进行中的抢兑任务，已恢复监控")
            self.taskLogIndex = task.logs?.count ?? 0
            let immediate = (dict["immediate"] as? Bool) ?? task.immediate ?? false
            let hour = dict["targetHour"] as? Int ?? task.targetHour ?? 0
            self.setWithdrawUiLocked(true, allowSmsEdit: !immediate && task.status == "pending")
            self.monitorTask(taskId: self.activeTaskId!, immediate: immediate)
            if !immediate {
                let remaining = KuwoTimeHelper.remainingMsUntilHour(hour)
                if remaining > 0 {
                    self.runCountdown(info: KuwoTimeHelper.NextWithdrawInfo(hour: hour, inWindow: true, diffMin: 1), taskId: self.activeTaskId!)
                } else {
                    self.countdownLabel.text = "🚀 后端正在执行 \(KuwoTimeHelper.formatHour(hour)) 抢兑..."
                    self.appendLog("🔄 倒计时已结束，等待后端执行结果…")
                }
            }
        }
    }

    private func bindAction(_ button: UIButton, _ action: @escaping () -> Void) {
        let wrapper = ActionWrapper(action)
        objc_setAssociatedObject(button, &ActionWrapper.key, wrapper, .OBJC_ASSOCIATION_RETAIN_NONATOMIC)
        button.addTarget(wrapper, action: #selector(ActionWrapper.invoke), for: .touchUpInside)
    }
}

final class ElmRushViewController: BaseNativeViewController {
    private let scrollView = UIScrollView()
    private let contentStack = UIStackView()
    private let authHint = UILabel()
    private let windowLabel = UILabel()
    private let accountStack = UIStackView()
    private let productStack = UIStackView()
    private let slotSegment = UISegmentedControl(items: ["10:00 场", "15:00 场"])
    private let fetchButton = UIButton(type: .system)
    private let startButton = UIButton(type: .system)
    private let stopButton = UIButton(type: .system)
    private let countdownLabel = UILabel()
    private let logView = UITextView()

    private var elmAuthorized = false
    private var accounts: [ElmAccount] = []
    private var selectedRefs = Set<String>()
    private var ckReadyRefs = Set<String>()
    private var selectedSlot = 0
    private var windowInfo: ElmWindowInfo?
    private var ckData: ElmCkFetchResult?
    private var activeTaskId: String?
    private var taskLogIndex = 0
    private var monitorTimer: Timer?
    private var countdownTimer: Timer?
    private var clockTimer: Timer?

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        restoreLogs()
        loadPage()
        startClock()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        clockTimer?.invalidate()
        countdownTimer?.invalidate()
    }

    private func setupUI() {
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        contentStack.axis = .vertical
        contentStack.spacing = 12
        contentStack.isLayoutMarginsRelativeArrangement = true
        contentStack.layoutMargins = UIEdgeInsets(top: 8, left: 16, bottom: 24, right: 16)
        contentStack.translatesAutoresizingMaskIntoConstraints = false
        scrollView.addSubview(contentStack)
        view.addSubview(scrollView)
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            contentStack.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor),
            contentStack.leadingAnchor.constraint(equalTo: scrollView.contentLayoutGuide.leadingAnchor),
            contentStack.trailingAnchor.constraint(equalTo: scrollView.contentLayoutGuide.trailingAnchor),
            contentStack.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor),
            contentStack.widthAnchor.constraint(equalTo: scrollView.frameLayoutGuide.widthAnchor),
        ])

        let subtitle = UILabel()
        subtitle.text = "每周五 10:00 / 15:00 场，与网页端逻辑一致"
        subtitle.font = .systemFont(ofSize: 12)
        subtitle.textColor = .secondaryLabel
        contentStack.addArrangedSubview(subtitle)

        authHint.font = .systemFont(ofSize: 12.5, weight: .semibold)
        authHint.textColor = .systemRed
        authHint.numberOfLines = 0
        authHint.isHidden = true
        contentStack.addArrangedSubview(authHint)

        let card = UIStackView()
        card.axis = .vertical
        card.spacing = 10
        card.isLayoutMarginsRelativeArrangement = true
        card.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        card.applyCardStyle(cornerRadius: 14)

        windowLabel.font = .systemFont(ofSize: 12.5)
        windowLabel.numberOfLines = 0
        card.addArrangedSubview(windowLabel)

        let accountTitle = UILabel()
        accountTitle.text = "选择账号（可多选）"
        accountTitle.font = .systemFont(ofSize: 11.5)
        accountTitle.textColor = .secondaryLabel
        card.addArrangedSubview(accountTitle)

        accountStack.axis = .vertical
        accountStack.spacing = 6
        card.addArrangedSubview(accountStack)

        let slotTitle = UILabel()
        slotTitle.text = "选择场次"
        slotTitle.font = .systemFont(ofSize: 11.5)
        slotTitle.textColor = .secondaryLabel
        card.addArrangedSubview(slotTitle)

        slotSegment.selectedSegmentIndex = UISegmentedControl.noSegment
        slotSegment.addTarget(self, action: #selector(slotChanged), for: .valueChanged)
        card.addArrangedSubview(slotSegment)

        productStack.axis = .vertical
        productStack.spacing = 6
        card.addArrangedSubview(productStack)
        renderCkProducts(nil)

        stylePrimary(fetchButton, title: "获取 CK", color: .systemGreen)
        stylePrimary(startButton, title: "参与抢兑", color: .systemIndigo)
        stylePrimary(stopButton, title: "停止", color: .systemRed)
        stopButton.isHidden = true
        bindAction(fetchButton) { [weak self] in self?.fetchCk() }
        bindAction(startButton) { [weak self] in self?.startExchange() }
        bindAction(stopButton) { [weak self] in self?.stopExchange() }

        let btnRow = UIStackView(arrangedSubviews: [fetchButton, startButton, stopButton])
        btnRow.axis = .horizontal
        btnRow.spacing = 8
        btnRow.distribution = .fillEqually
        card.addArrangedSubview(btnRow)

        countdownLabel.font = .systemFont(ofSize: 14, weight: .bold)
        countdownLabel.textColor = .white
        countdownLabel.textAlignment = .center
        countdownLabel.isHidden = true
        countdownLabel.backgroundColor = .systemOrange
        countdownLabel.layer.cornerRadius = 10
        countdownLabel.clipsToBounds = true
        card.addArrangedSubview(countdownLabel)

        contentStack.addArrangedSubview(card)

        let logCard = UIStackView()
        logCard.axis = .vertical
        logCard.spacing = 8
        logCard.isLayoutMarginsRelativeArrangement = true
        logCard.layoutMargins = UIEdgeInsets(top: 14, left: 14, bottom: 14, right: 14)
        logCard.applyCardStyle(cornerRadius: 14)

        let logHeader = UIStackView()
        logHeader.axis = .horizontal
        let logTitle = UILabel()
        logTitle.text = "执行日志"
        logTitle.font = .systemFont(ofSize: 14, weight: .bold)
        let clearBtn = UIButton(type: .system)
        clearBtn.setTitle("清空", for: .normal)
        clearBtn.titleLabel?.font = .systemFont(ofSize: 12)
        bindAction(clearBtn) { [weak self] in self?.clearLogs() }
        logHeader.addArrangedSubview(logTitle)
        logHeader.addArrangedSubview(UIView())
        logHeader.addArrangedSubview(clearBtn)
        logCard.addArrangedSubview(logHeader)

        logView.isEditable = false
        logView.font = .monospacedSystemFont(ofSize: 11, weight: .regular)
        logView.backgroundColor = .secondarySystemBackground
        logView.layer.cornerRadius = 10
        logView.heightAnchor.constraint(equalToConstant: 220).isActive = true
        logCard.addArrangedSubview(logView)
        contentStack.addArrangedSubview(logCard)
    }

    private func stylePrimary(_ button: UIButton, title: String, color: UIColor) {
        button.setTitle(title, for: .normal)
        button.setTitleColor(.white, for: .normal)
        button.titleLabel?.font = .systemFont(ofSize: 14, weight: .bold)
        button.backgroundColor = color
        button.layer.cornerRadius = 10
        button.heightAnchor.constraint(equalToConstant: 42).isActive = true
    }

    @objc private func slotChanged() {
        selectedSlot = slotSegment.selectedSegmentIndex == 0 ? 10 : (slotSegment.selectedSegmentIndex == 1 ? 15 : 0)
        if let data = ckData { renderCkProducts(data) }
        updateButtons()
    }

    private func isSlotSelectable(_ hour: Int) -> Bool {
        windowInfo?.selectableSlots?.contains(hour) == true
    }

    private func isSlotBeforeExecute(_ slot: ElmSlotProduct) -> Bool? {
        let executeAt = slot.executeAt ?? ""
        if executeAt.contains("每周") { return nil }
        let fmt = DateFormatter()
        fmt.locale = Locale(identifier: "zh_CN")
        fmt.timeZone = TimeZone(identifier: "Asia/Shanghai")
        fmt.dateFormat = "yyyy-MM-dd HH:mm:ss"
        if let parsed = fmt.date(from: executeAt) {
            return parsed.timeIntervalSinceNow > 0
        }
        var cal = Calendar.current
        cal.timeZone = TimeZone(identifier: "Asia/Shanghai") ?? .current
        let hour = slot.targetHour ?? 0
        var exec = cal.dateComponents([.year, .month, .day], from: Date())
        exec.hour = hour
        exec.minute = 0
        exec.second = 0
        guard let execDate = cal.date(from: exec) else { return true }
        return Date() < execDate
    }

    private func slotStatusPrefix(_ slot: ElmSlotProduct) -> String {
        let hour = slot.targetHour ?? 0
        if isSlotSelectable(hour) {
            return selectedSlot == hour ? "🎯 已选 · " : ""
        }
        switch isSlotBeforeExecute(slot) {
        case nil: return "📅 非抢兑日 · "
        case true?: return "⏳ 场次未到 · "
        default: return "⏱ 场次已结束 · "
        }
    }

    private func resetCkState() {
        ckData = nil
        ckReadyRefs.removeAll()
        selectedSlot = 0
        slotSegment.selectedSegmentIndex = UISegmentedControl.noSegment
        renderCkProducts(nil)
        updateButtons()
    }

    private func syncCkReadyFromData() {
        let refs = selectedRefs
        ckReadyRefs = Set(
            ckData?.accounts
                .filter { ($0.success ?? false) && refs.contains($0.ref ?? "") }
                .compactMap { $0.ref } ?? []
        )
    }

    private func renderCkProducts(_ data: ElmCkFetchResult?) {
        productStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        guard let data = data else {
            productStack.addArrangedSubview(captionLabel("请先选择账号并点击「获取 CK」"))
            return
        }

        let readyAccounts = data.accounts.filter { $0.success == true }
        let isSingle = data.accounts.count == 1 && readyAccounts.count == 1

        if isSingle, let acc = readyAccounts.first {
            productStack.addArrangedSubview(captionLabel("✅ CK 已获取 · 通道：\(acc.route ?? "-")"))
            let star = (acc.starBalance ?? -1) >= 0 ? "\(acc.starBalance ?? 0)" : "--"
            let starLabel = captionLabel("💎 幸运星余额：\(star) · 账号：\(acc.remark ?? acc.ref ?? "")")
            starLabel.textColor = .systemOrange
            productStack.addArrangedSubview(starLabel)
            if let msg = data.message, !msg.isEmpty {
                productStack.addArrangedSubview(captionLabel(msg))
            }
        } else {
            let total = data.totalCount > 0 ? data.totalCount : data.accounts.count
            let summary = captionLabel("CK 就绪：\(data.readyCount)/\(total)")
            summary.font = .systemFont(ofSize: 12.5, weight: .semibold)
            productStack.addArrangedSubview(summary)
            if let msg = data.message, !msg.isEmpty {
                productStack.addArrangedSubview(captionLabel(msg))
            }
            for item in data.accounts {
                let icon = (item.success ?? false) ? "✅" : "❌"
                var extra = ""
                if let route = item.route { extra += " · \(route)" }
                if (item.success ?? false), (item.starBalance ?? -1) >= 0 { extra += " · \(item.starBalance ?? 0)星" }
                productStack.addArrangedSubview(captionLabel("\(icon) \(item.remark ?? item.ref ?? "") — \(item.message ?? "")\(extra)"))
            }
        }

        if data.slots.isEmpty {
            productStack.addArrangedSubview(captionLabel("暂无场次商品信息"))
            return
        }
        for slot in data.slots {
            productStack.addArrangedSubview(buildSlotCard(slot))
        }
    }

    private func buildSlotCard(_ slot: ElmSlotProduct) -> UIView {
        let hour = slot.targetHour ?? 0
        let selectable = isSlotSelectable(hour)
        let active = selectedSlot == hour
        let prefix = slotStatusPrefix(slot)
        let card = UIStackView()
        card.axis = .vertical
        card.spacing = 4
        card.isLayoutMarginsRelativeArrangement = true
        card.layoutMargins = UIEdgeInsets(top: 10, left: 12, bottom: 10, right: 12)
        card.applyCardStyle(cornerRadius: 10)
        card.alpha = selectable ? 1 : 0.45
        if active {
            card.layer.borderWidth = 2
            card.layer.borderColor = UIColor.systemIndigo.cgColor
        }

        let title = captionLabel("\(prefix)\(slot.slotLabel ?? "\(hour):00 场")")
        title.font = .systemFont(ofSize: 13, weight: .semibold)
        title.textColor = .label
        card.addArrangedSubview(title)
        card.addArrangedSubview(captionLabel("执行时间：\(slot.executeAt ?? "-")"))

        if let p = slot.product {
            let name = captionLabel(p.title ?? "")
            name.font = .systemFont(ofSize: 12.5, weight: .semibold)
            name.textColor = .label
            card.addArrangedSubview(name)
            card.addArrangedSubview(captionLabel("所需幸运星：\(p.cost ?? 0) · 状态：\(p.status ?? "")"))
        } else {
            card.addArrangedSubview(captionLabel("暂未识别到该场次商品"))
        }
        return card
    }

    private func prefs() -> UserDefaults { UserDefaults.standard }

    private func appendLog(_ msg: String) {
        let ts = DateFormatter.localizedString(from: Date(), dateStyle: .none, timeStyle: .medium)
        let line = "[\(ts)] \(msg)\n"
        logView.text = (logView.text ?? "") + line
        prefs().set(logView.text, forKey: "elm_rush_log")
    }

    private func restoreLogs() {
        logView.text = prefs().string(forKey: "elm_rush_log") ?? ""
    }

    private func clearLogs() {
        logView.text = ""
        taskLogIndex = 0
        prefs().removeObject(forKey: "elm_rush_log")
    }

    private func flushTaskLogs(_ logs: [ElmTaskLog]?) {
        guard let logs = logs, logs.count > taskLogIndex else { return }
        for i in taskLogIndex..<logs.count {
            let entry = logs[i]
            let icon = entry.level == "success" ? "✅" : (entry.level == "error" ? "❌" : (entry.level == "warn" ? "⚠️" : "•"))
            let prefix = entry.time.map { "\($0) " } ?? ""
            appendLog("\(prefix)\(icon) \(entry.message ?? "")")
        }
        taskLogIndex = logs.count
    }

    private func loadPage() {
        PortalService.shared.checkElmAuth { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success(let pair):
                self.elmAuthorized = pair.0
                self.authHint.isHidden = pair.0
                if !pair.0 { self.authHint.text = pair.1.isEmpty ? "请先上车饿了么活动" : pair.1 }
            }
        }
        PortalService.shared.fetchElmAccounts { [weak self] result in
            guard let self = self else { return }
            if case .success(let list) = result {
                self.accounts = list
                self.renderAccounts()
            }
        }
        refreshWindow()
        resumeActiveTask()
    }

    private func startClock() {
        clockTimer?.invalidate()
        clockTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            self?.refreshWindow()
        }
    }

    private func refreshWindow() {
        PortalService.shared.fetchElmWindow { [weak self] result in
            guard let self = self, case .success(let info) = result else { return }
            self.windowInfo = info
            let slots = info.selectableSlots?.map { "\($0):00" }.joined(separator: " / ") ?? "--"
            self.windowLabel.text = "当前场次：\(slots)\n\(info.message ?? "")"
            let slotSet = Set(info.selectableSlots ?? [])
            self.slotSegment.setEnabled(slotSet.contains(10), forSegmentAt: 0)
            self.slotSegment.setEnabled(slotSet.contains(15), forSegmentAt: 1)
            if self.selectedSlot > 0 && !slotSet.contains(self.selectedSlot) {
                self.selectedSlot = 0
                self.slotSegment.selectedSegmentIndex = UISegmentedControl.noSegment
            }
            if let auto = info.autoSlot, self.selectedSlot == 0, slotSet.contains(auto) {
                self.selectedSlot = auto
                self.slotSegment.selectedSegmentIndex = auto == 10 ? 0 : 1
            }
            if let data = self.ckData { self.renderCkProducts(data) }
            self.updateButtons()
        }
    }

    private func renderAccounts() {
        accountStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        if accounts.isEmpty {
            accountStack.addArrangedSubview(captionLabel("暂无饿了么账号，请先在项目中上车"))
            return
        }
        if selectedRefs.isEmpty {
            accounts.first(where: { !($0.ref ?? "").isEmpty })?.ref.map { selectedRefs.insert($0) }
        }
        for acc in accounts {
            guard let ref = acc.ref, !ref.isEmpty else { continue }
            let cb = UISwitch()
            cb.isOn = selectedRefs.contains(ref)
            cb.addAction(UIAction { [weak self] _ in
                guard let self = self else { return }
                if cb.isOn { self.selectedRefs.insert(ref) } else {
                    self.selectedRefs.remove(ref)
                    self.ckReadyRefs.remove(ref)
                }
                if self.selectedRefs.isEmpty {
                    self.resetCkState()
                } else {
                    self.syncCkReadyFromData()
                    self.updateButtons()
                }
            }, for: .valueChanged)
            let row = UIStackView(arrangedSubviews: [captionLabel("\(acc.remark ?? ref) (\(acc.route ?? "协议"))"), cb])
            row.axis = .horizontal
            row.distribution = .equalSpacing
            accountStack.addArrangedSubview(row)
        }
        updateButtons()
    }

    private func captionLabel(_ text: String) -> UILabel {
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 12.5)
        return label
    }

    private func updateButtons() {
        let running = !(activeTaskId?.isEmpty ?? true)
        fetchButton.isEnabled = elmAuthorized && !selectedRefs.isEmpty && !running
        fetchButton.alpha = fetchButton.isEnabled ? 1 : 0.5
        let canStart = elmAuthorized && !selectedRefs.isEmpty && selectedRefs.isSubset(of: ckReadyRefs) && selectedSlot > 0 && (windowInfo?.canStart ?? false) && !running
        startButton.isHidden = running
        startButton.isEnabled = canStart
        startButton.alpha = canStart ? 1 : 0.5
        stopButton.isHidden = !running
    }

    private func fetchCk() {
        let refs = Array(selectedRefs)
        guard !refs.isEmpty else { showMessage("请至少选择一个账号"); return }
        appendLog("开始获取 CK…")
        PortalService.shared.fetchElmCk(refs: refs) { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .failure(let error):
                self.appendLog("CK 失败: \(error.message)")
                self.handle(error)
            case .success(let data):
                self.ckData = data
                self.syncCkReadyFromData()
                self.renderCkProducts(data)
                let total = data.totalCount > 0 ? data.totalCount : refs.count
                self.appendLog("CK 完成：\(data.readyCount)/\(total)")
                for acc in data.accounts {
                    let icon = (acc.success ?? false) ? "✅" : "❌"
                    self.appendLog("\(icon) CK \(acc.remark ?? acc.ref ?? "")：\(acc.message ?? "")")
                }
                self.refreshWindow()
                self.updateButtons()
            }
        }
    }

    private func startExchange() {
        guard elmAuthorized else { showMessage("请先上车饿了么活动"); return }
        guard selectedSlot > 0 else { showMessage("请选择场次"); return }
        let refs = Array(selectedRefs)
        appendLog("创建抢兑任务…")
        PortalService.shared.scheduleElmExchange(refs: refs, targetHour: selectedSlot) { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .failure(let error):
                self.appendLog("创建失败: \(error.message)")
                self.handle(error)
            case .success(let task):
                self.activeTaskId = task.id
                self.taskLogIndex = 0
                self.appendLog("任务已创建：\(task.id ?? "")")
                self.applyTask(task)
                self.monitorTask(taskId: task.id ?? "")
            }
        }
    }

    private func stopExchange() {
        guard let taskId = activeTaskId else { return }
        PortalService.shared.cancelElmExchange(taskId: taskId) { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .failure(let error):
                self.handle(error)
            case .success:
                self.appendLog("任务已停止")
                self.finishTask(nil)
            }
        }
    }

    private func monitorTask(taskId: String) {
        monitorTimer?.invalidate()
        monitorTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] timer in
            guard let self = self else { timer.invalidate(); return }
            PortalService.shared.fetchElmExchangeStatus(taskId: taskId) { result in
                guard case .success(let task) = result, let task = task else { return }
                self.applyTask(task)
                if task.status == "completed" || task.status == "failed" || task.status == "cancelled" {
                    timer.invalidate()
                    self.finishTask(task)
                }
            }
        }
    }

    private func applyTask(_ task: ElmExchangeTask) {
        flushTaskLogs(task.logs)
        updateButtons()
        if task.status == "pending" { startCountdown(executeAt: task.executeAt) }
        if task.status == "running" {
            countdownLabel.isHidden = false
            countdownLabel.text = "🚀 正在执行抢兑…"
        }
    }

    private func finishTask(_ task: ElmExchangeTask?) {
        monitorTimer?.invalidate()
        countdownTimer?.invalidate()
        activeTaskId = nil
        countdownLabel.isHidden = true
        if let task = task { applyTask(task) }
        updateButtons()
    }

    private func resumeActiveTask() {
        PortalService.shared.fetchElmExchangeStatus(taskId: nil) { [weak self] result in
            guard let self = self, case .success(let task) = result, let task = task else { return }
            if task.status == "pending" || task.status == "running" {
                self.activeTaskId = task.id
                self.appendLog("🔄 恢复进行中的任务")
                self.flushTaskLogs(task.logs)
                self.applyTask(task)
                self.monitorTask(taskId: task.id ?? "")
            }
        }
    }

    private func startCountdown(executeAt: String?) {
        countdownTimer?.invalidate()
        guard let executeAt = executeAt else { return }
        let fmt = DateFormatter()
        fmt.dateFormat = "yyyy-MM-dd HH:mm:ss"
        guard let target = fmt.date(from: executeAt) else { return }
        countdownLabel.isHidden = false
        countdownTimer = Timer.scheduledTimer(withTimeInterval: 0.2, repeats: true) { [weak self] timer in
            guard let self = self else { timer.invalidate(); return }
            let remain = target.timeIntervalSinceNow
            if remain <= 0 {
                self.countdownLabel.text = "🚀 正在执行抢兑…"
                timer.invalidate()
                return
            }
            let h = Int(remain) / 3600
            let m = (Int(remain) % 3600) / 60
            let s = Int(remain) % 60
            self.countdownLabel.text = String(format: "⏱ 倒计时 %02d:%02d:%02d", h, m, s)
        }
    }

    private func bindAction(_ button: UIButton, _ action: @escaping () -> Void) {
        let wrapper = ActionWrapper(action)
        objc_setAssociatedObject(button, &ActionWrapper.key, wrapper, .OBJC_ASSOCIATION_RETAIN_NONATOMIC)
        button.addTarget(wrapper, action: #selector(ActionWrapper.invoke), for: .touchUpInside)
    }
}

private final class ActionWrapper: NSObject {
    static var key: UInt8 = 0
    private let action: () -> Void
    init(_ action: @escaping () -> Void) { self.action = action }
    @objc func invoke() { action() }
}
