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
        for h in withdrawHours {
            let diffMin = minutesUntilHour(h, hour: bj.hour, min: bj.min)
            if diffMin > 0 && diffMin <= 4 {
                return NextWithdrawInfo(hour: h, inWindow: true, diffMin: diffMin)
            }
            if diffMin == 0 {
                return NextWithdrawInfo(hour: h, inWindow: true, diffMin: 0)
            }
        }
        for h in withdrawHours {
            let diffMin = minutesUntilHour(h, hour: bj.hour, min: bj.min)
            if diffMin > 4 {
                return NextWithdrawInfo(hour: h, inWindow: false, diffMin: diffMin)
            }
        }
        let nextDayMin = minutesUntilHour(withdrawHours[0], hour: bj.hour, min: bj.min)
        return NextWithdrawInfo(hour: withdrawHours[0], inWindow: false, diffMin: nextDayMin)
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
    private let phoneField = UITextField()
    private let passwordField = UITextField()
    private let smsField = UITextField()
    private let smsStatus = UILabel()
    private let nowTimeLabel = UILabel()
    private let nextTimeLabel = UILabel()
    private let timeHintLabel = UILabel()
    private let withdrawButton = UIButton(type: .system)
    private let countdownLabel = UILabel()
    private let logTextView = UITextView()
    private let quotaStack = UIStackView()

    private var kuwoAuthorized = false
    private var selectedQuotaId = "60004"
    private var withdrawSubmitting = false
    private var taskLogIndex = 0
    private var monitorTimer: Timer?
    private var clockTimer: Timer?
    private var countdownTimer: Timer?

    private let quotaOptions: [(id: String, title: String)] = [
        ("60004", "1元"),
        ("30002", "2元"),
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

        card.addArrangedSubview(fieldBlock(label: "手机号", field: phoneField, secure: false, readonly: true))
        card.addArrangedSubview(fieldBlock(label: "密码", field: passwordField, secure: true, readonly: true))
        card.addArrangedSubview(fieldBlock(label: "短信验证码", field: smsField, secure: false, readonly: false))

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
        card.addArrangedSubview(smsRow)
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

        countdownLabel.font = .systemFont(ofSize: 15, weight: .bold)
        countdownLabel.textColor = .white
        countdownLabel.textAlignment = .center
        countdownLabel.backgroundColor = .systemOrange
        countdownLabel.layer.cornerRadius = 10
        countdownLabel.clipsToBounds = true
        countdownLabel.isHidden = true
        countdownLabel.numberOfLines = 0

        card.addArrangedSubview(withdrawButton)
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
                self.phoneField.text = ""
                self.passwordField.text = ""
            case .success(let cred):
                self.phoneField.text = cred.phone
                self.passwordField.text = cred.password
                self.restoreKuwoState(phone: cred.phone ?? "")
            }
        }
    }

    private func startClock() {
        clockTimer?.invalidate()
        clockTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            self?.updateTimeDisplay()
        }
    }

    private func stopTimers() {
        clockTimer?.invalidate()
        monitorTimer?.invalidate()
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

    private func sendSms() {
        guard kuwoAuthorized else { showMessage("酷我活动授权已到期，请前往我的项目续费"); return }
        let phone = phoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let password = passwordField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !phone.isEmpty, !password.isEmpty else { showMessage("未读取到酷我账号，请先在项目中上车"); return }
        smsStatus.text = "正在登录并发送验证码..."
        appendLog("正在登录酷我账号...")
        PortalService.shared.sendKuwoSms(phone: phone, password: password) { [weak self] result in
            guard let self = self else { return }
            switch result {
            case .failure(let error):
                self.smsStatus.text = "❌ \(error.message)"
                self.smsStatus.textColor = .systemRed
                self.appendLog("发送失败: \(error.message)")
                self.handle(error)
            case .success:
                let masked = phone.count >= 11 ? String(phone.prefix(3)) + "****" + String(phone.suffix(4)) : phone
                self.smsStatus.text = "✅ 已发送至 \(masked)"
                self.smsStatus.textColor = .systemGreen
                self.appendLog("验证码已发送，请输入验证码")
                self.showMessage("验证码已发送")
            }
        }
    }

    private func startWithdraw() {
        guard kuwoAuthorized else { showMessage("酷我活动授权已到期，请前往我的项目续费"); return }
        guard !withdrawSubmitting else { showMessage("任务提交中，请稍候"); return }
        let phone = phoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let password = passwordField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let smsCode = smsField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !phone.isEmpty, !password.isEmpty else { showMessage("未读取到酷我账号"); return }
        guard !smsCode.isEmpty else { showMessage("请输入验证码"); return }

        let info = KuwoTimeHelper.getNextWithdrawInfo()
        if info.inWindow && info.diffMin > 0 && info.diffMin <= 4 {
            appendLog("🎯 提交定时抢兑任务，后端将在 \(KuwoTimeHelper.formatHour(info.hour)) 自动执行")
            setWithdrawUiLocked(true)
            saveKuwoState(phone: phone, password: password, smsCode: smsCode, quotaId: selectedQuotaId, targetHour: info.hour, taskId: nil, immediate: false)
            scheduleOnBackend(phone: phone, password: password, smsCode: smsCode, quotaId: selectedQuotaId, targetHour: info.hour, info: info, immediate: false)
            return
        }
        appendLog("⚡ 立即提交抢兑...")
        setWithdrawUiLocked(true)
        scheduleOnBackend(phone: phone, password: password, smsCode: smsCode, quotaId: selectedQuotaId, targetHour: nil, info: nil, immediate: true)
    }

    private func scheduleOnBackend(phone: String, password: String, smsCode: String, quotaId: String, targetHour: Int?, info: KuwoTimeHelper.NextWithdrawInfo?, immediate: Bool) {
        guard !withdrawSubmitting else { return }
        withdrawSubmitting = true
        PortalService.shared.scheduleKuwoWithdraw(phone: phone, password: password, quotaId: quotaId, smsCode: smsCode, targetHour: targetHour, immediate: immediate) { [weak self] result in
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
                self.updateSavedTaskId(taskId: taskId, immediate: immediate, targetHour: targetHour)
                self.monitorTask(taskId: taskId, immediate: immediate)
                if !immediate, let info = info {
                    self.runCountdown(info: info, taskId: taskId)
                }
            }
        }
    }

    private func setWithdrawUiLocked(_ locked: Bool) {
        withdrawButton.isHidden = locked
        countdownLabel.isHidden = !locked
        phoneField.isEnabled = !locked
        passwordField.isEnabled = !locked
        smsField.isEnabled = !locked
    }

    private func restoreWithdrawUi() {
        withdrawSubmitting = false
        countdownTimer?.invalidate()
        setWithdrawUiLocked(false)
        countdownLabel.text = ""
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

    private func saveKuwoState(phone: String, password: String, smsCode: String, quotaId: String, targetHour: Int?, taskId: String?, immediate: Bool) {
        var dict: [String: Any] = [
            "phone": phone, "password": password, "smsCode": smsCode, "quotaId": quotaId,
            "immediate": immediate, "savedAt": Date().timeIntervalSince1970,
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
        if let savedPhone = dict["phone"] as? String, !savedPhone.isEmpty, !phone.isEmpty, savedPhone != phone { return }
        smsField.text = dict["smsCode"] as? String
        if let quotaId = dict["quotaId"] as? String {
            selectedQuotaId = quotaId
            refreshQuotaButtons()
        }
        let savedPhone = dict["phone"] as? String ?? phone
        let taskIdFromState = dict["taskId"] as? String

        func finishRestore(taskId: String, active: KuwoWithdrawTask?, taskById: KuwoWithdrawTask? = nil) {
            if let active = active, active.status == "completed" || active.status == "failed" {
                self.taskLogIndex = active.logs?.count ?? 0
                _ = self.applyTaskResult(task: active, immediate: (dict["immediate"] as? Bool) ?? false)
                return
            }
            guard !taskId.isEmpty else { return }
            self.appendLog("🔄 检测到进行中的抢兑任务，已恢复监控")
            let logs = active?.logs ?? taskById?.logs
            self.taskLogIndex = logs?.count ?? 0
            self.setWithdrawUiLocked(true)
            let immediate = (dict["immediate"] as? Bool) ?? active?.immediate ?? false
            let hour = dict["targetHour"] as? Int ?? active?.targetHour ?? taskById?.targetHour ?? 0
            self.monitorTask(taskId: taskId, immediate: immediate)
            if !immediate {
                let remaining = KuwoTimeHelper.remainingMsUntilHour(hour)
                if remaining > 0 {
                    self.runCountdown(info: KuwoTimeHelper.NextWithdrawInfo(hour: hour, inWindow: true, diffMin: 1), taskId: taskId)
                } else {
                    self.countdownLabel.text = "🚀 后端正在执行 \(KuwoTimeHelper.formatHour(hour)) 抢兑..."
                    self.appendLog("🔄 倒计时已结束，等待后端执行结果…")
                }
            }
        }

        if let taskId = taskIdFromState, !taskId.isEmpty {
            PortalService.shared.fetchKuwoWithdrawStatus(taskId: taskId) { [weak self] result in
                guard let self = self else { return }
                let taskById: KuwoWithdrawTask?
                switch result {
                case .success(let task): taskById = task
                case .failure: taskById = nil
                }
                if let taskById = taskById, taskById.status == "completed" || taskById.status == "failed" {
                    finishRestore(taskId: taskId, active: taskById)
                    return
                }
                PortalService.shared.fetchKuwoWithdrawStatus(phone: savedPhone) { result2 in
                    var resolvedTaskId = taskId
                    var active: KuwoWithdrawTask?
                    switch result2 {
                    case .success(let task): active = task
                    case .failure: active = nil
                    }
                    if let active = active, active.status == "pending" || active.status == "running" {
                        resolvedTaskId = active.id ?? taskId
                    } else if active == nil {
                        active = taskById
                    }
                    finishRestore(taskId: resolvedTaskId, active: active, taskById: taskById)
                }
            }
        } else {
            PortalService.shared.fetchKuwoWithdrawStatus(phone: savedPhone) { [weak self] result in
                guard let self = self else { return }
                let active: KuwoWithdrawTask?
                switch result {
                case .success(let task): active = task
                case .failure: active = nil
                }
                let taskId = active?.id ?? ""
                finishRestore(taskId: taskId, active: active)
            }
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
