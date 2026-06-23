import UIKit

@available(iOS 13.0, *)
final class JdPortalViewController: BaseNativeViewController {
    private enum MainTab { case login, task }
    private enum LoginTab { case query, sms, wx }

    private struct TaskDef {
        let id: String
        let name: String
        let icon: String
        let desc: String
    }

    private let scrollView = UIScrollView()
    private let stack = UIStackView()
    private let mainTabRow = UIStackView()
    private let loginTabRow = UIStackView()
    private let toolbarRow = UIStackView()
    private let contentStack = UIStackView()

    private var mainTab: MainTab = .login
    private var loginTab: LoginTab = .query
    private var accounts: [PortalJdAccount] = []
    private var taskSelections: [String: Set<Int>] = [:]
    private var runningTasks: [String: String] = [:]
    private var logLines: [String] = []
    private let logLabel = UILabel()
    private var buttonActions: [ObjectIdentifier: () -> Void] = [:]

    private let smsPhoneField = UITextField()
    private let smsCodeField = UITextField()
    private let smsIdCardField = UITextField()
    private let smsIdCardStack = UIStackView()
    private let smsResultLabel = UILabel()
    private let wxDeviceStack = UIStackView()
    private let wxRiskStack = UIStackView()
    private let wxRiskLabel = UILabel()
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
        renderAll()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        if mainTab == .login && loginTab == .query { loadAccounts() }
    }

    private func setupLayout() {
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        stack.axis = .vertical
        stack.spacing = 12
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 8),
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -16),
            stack.widthAnchor.constraint(equalTo: view.widthAnchor, constant: -32)
        ])

        let header = UILabel()
        header.text = "🛒 京东工作台"
        header.font = .systemFont(ofSize: 20, weight: .bold)
        stack.addArrangedSubview(header)
        stack.addArrangedSubview(captionLabel("查询资产、登录与京东任务"))

        [mainTabRow, loginTabRow, toolbarRow, contentStack].forEach {
            $0.axis = .horizontal
            $0.spacing = 8
            $0.alignment = .center
            stack.addArrangedSubview($0)
        }
        contentStack.axis = .vertical
        contentStack.alignment = .fill
    }

    private func renderAll() {
        renderMainTabs()
        loginTabRow.isHidden = mainTab == .task
        toolbarRow.arrangedSubviews.forEach { $0.removeFromSuperview() }
        contentStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        switch mainTab {
        case .login: renderLogin()
        case .task: renderTasks()
        }
    }

    private func renderMainTabs() {
        mainTabRow.arrangedSubviews.forEach { $0.removeFromSuperview() }
        mainTabRow.addArrangedSubview(tabButton("🔍 查询与登录", active: mainTab == .login) { [weak self] in
            self?.mainTab = .login; self?.renderAll()
        })
        mainTabRow.addArrangedSubview(tabButton("⚡ 京东任务", active: mainTab == .task) { [weak self] in
            self?.mainTab = .task; self?.renderAll()
        })
    }

    private func renderLogin() {
        loginTabRow.isHidden = false
        loginTabRow.arrangedSubviews.forEach { $0.removeFromSuperview() }
        loginTabRow.addArrangedSubview(tabButton("查询", active: loginTab == .query) { [weak self] in
            self?.loginTab = .query; self?.renderAll(); self?.loadAccounts()
        })
        loginTabRow.addArrangedSubview(tabButton("短信登录", active: loginTab == .sms) { [weak self] in
            self?.loginTab = .sms; self?.renderAll()
        })
        loginTabRow.addArrangedSubview(tabButton("微信协议", active: loginTab == .wx) { [weak self] in
            self?.loginTab = .wx; self?.renderAll(); self?.loadWxDevices()
        })

        switch loginTab {
        case .query:
            toolbarRow.addArrangedSubview(actionButton("全部查询", color: .systemBlue) { [weak self] in self?.queryAccount(0) })
            toolbarRow.addArrangedSubview(actionButton("刷新", color: .systemGray) { [weak self] in self?.loadAccounts() })
            contentStack.addArrangedSubview(captionLabel("加载中..."))
            loadAccounts()
        case .sms: renderSms()
        case .wx: renderWx()
        }
    }

    private func renderSms() {
        let card = cardContainer()
        card.addArrangedSubview(titleLabel("短信验证码登录"))
        card.addArrangedSubview(captionLabel("输入京东绑定的手机号，验证码登录后自动绑定"))
        smsPhoneField.applyAppInputStyle(placeholder: "请输入11位手机号")
        smsPhoneField.keyboardType = .numberPad
        card.addArrangedSubview(smsPhoneField)
        card.addArrangedSubview(actionButton("发送验证码", color: .systemBlue) { [weak self] in self?.sendSms() })
        smsCodeField.applyAppInputStyle(placeholder: "请输入短信验证码")
        smsCodeField.keyboardType = .numberPad
        card.addArrangedSubview(smsCodeField)
        smsIdCardStack.axis = .vertical
        smsIdCardStack.spacing = 8
        smsIdCardStack.isHidden = true
        smsIdCardStack.addArrangedSubview(captionLabel("身份证验证"))
        smsIdCardField.applyAppInputStyle(placeholder: "身份证前两位+后四位")
        smsIdCardStack.addArrangedSubview(smsIdCardField)
        card.addArrangedSubview(smsIdCardStack)
        card.addArrangedSubview(actionButton("提交登录", color: .systemPurple) { [weak self] in self?.verifySms() })
        smsResultLabel.numberOfLines = 0
        smsResultLabel.font = .systemFont(ofSize: 13)
        smsResultLabel.textColor = .secondaryLabel
        smsResultLabel.isHidden = true
        card.addArrangedSubview(smsResultLabel)
        contentStack.addArrangedSubview(card)
    }

    private func renderWx() {
        contentStack.addArrangedSubview(captionLabel("💡 需先在「更多-微信协议」扫码绑定在线设备，再选择设备刷新京东 CK。"))
        toolbarRow.addArrangedSubview(actionButton("刷新设备", color: .systemGray) { [weak self] in self?.loadWxDevices() })
        wxDeviceStack.axis = .vertical
        wxDeviceStack.spacing = 10
        wxDeviceStack.addArrangedSubview(captionLabel("加载中..."))
        contentStack.addArrangedSubview(wxDeviceStack)
        wxRiskStack.axis = .vertical
        wxRiskStack.spacing = 8
        wxRiskStack.isHidden = true
        wxRiskLabel.numberOfLines = 0
        wxRiskStack.addArrangedSubview(wxRiskLabel)
        wxRiskStack.addArrangedSubview(actionButton("验证完成，继续刷新", color: .systemOrange) { [weak self] in self?.continueWxRisk() })
        contentStack.addArrangedSubview(wxRiskStack)
        wxResultLabel.numberOfLines = 0
        wxResultLabel.font = .systemFont(ofSize: 13)
        wxResultLabel.textColor = .secondaryLabel
        wxResultLabel.isHidden = true
        contentStack.addArrangedSubview(wxResultLabel)
        loadWxDevices()
    }

    private func renderTasks() {
        contentStack.addArrangedSubview(captionLabel("选择任务和账号，点击执行按钮开始任务"))
        PortalService.shared.fetchJdAccounts { [weak self] result in
            guard let self = self else { return }
            if case .success(let list) = result { self.accounts = list }
            self.taskDefs.forEach { task in
                self.contentStack.addArrangedSubview(self.buildTaskCard(task))
            }
        }
        let logCard = cardContainer()
        logCard.addArrangedSubview(titleLabel("执行日志"))
        logLabel.numberOfLines = 0
        logLabel.font = .monospacedSystemFont(ofSize: 11, weight: .regular)
        logLabel.text = "暂无日志，请执行任务"
        logLabel.backgroundColor = UIColor.secondarySystemBackground
        logLabel.layer.cornerRadius = 8
        logLabel.clipsToBounds = true
        logCard.addArrangedSubview(logLabel)
        logCard.addArrangedSubview(actionButton("清空", color: .systemGray) { [weak self] in
            self?.logLines = []
            self?.refreshLogs()
        })
        contentStack.addArrangedSubview(logCard)
        refreshLogs()
    }

    private func buildTaskCard(_ task: TaskDef) -> UIView {
        let card = cardContainer()
        card.addArrangedSubview(titleLabel("\(task.icon) \(task.name)"))
        card.addArrangedSubview(captionLabel(task.desc))
        let selection = taskSelections[task.id] ?? [0]
        let label = captionLabel(accountSelectionLabel(selection))
        card.addArrangedSubview(actionButton("选择账号", color: .systemGray) { [weak self] in
            self?.pickAccounts(taskId: task.id, label: label)
        })
        card.addArrangedSubview(label)
        let running = runningTasks[task.id] != nil
        let btn = actionButton(running ? "停止任务" : "执行任务", color: running ? .systemRed : .systemBlue) { [weak self] in
            guard let self = self else { return }
            if self.runningTasks[task.id] != nil {
                self.stopTask(task)
            } else {
                self.executeTask(task)
            }
        }
        card.addArrangedSubview(btn)
        return card
    }

    private func loadAccounts() {
        PortalService.shared.fetchJdAccounts { [weak self] result in
            guard let self = self else { return }
            self.contentStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
            switch result {
            case .failure(let error):
                self.handle(error)
                self.contentStack.addArrangedSubview(self.captionLabel(error.message))
            case .success(let list):
                self.accounts = list
                self.renderAccountList(list)
            }
        }
    }

    private func renderAccountList(_ list: [PortalJdAccount]) {
        contentStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
        if list.isEmpty {
            contentStack.addArrangedSubview(captionLabel("暂无绑定的京东账号，请先使用短信或微信协议登录"))
            return
        }
        let valid = list.filter { $0.valid }.count
        contentStack.addArrangedSubview(captionLabel("有效账号 \(valid) / \(list.count)"))
        list.sorted { $0.valid && !$1.valid }.forEach { acc in
            let card = cardContainer()
            card.addArrangedSubview(titleLabel(acc.nickname ?? acc.pin ?? "未知账号"))
            card.addArrangedSubview(captionLabel(acc.pin ?? ""))
            card.addArrangedSubview(captionLabel(acc.valid ? "✅ 有效" : "❌ 失效"))
            card.addArrangedSubview(actionButton("查询", color: .systemBlue) { [weak self] in
                self?.queryAccount(acc.index)
            })
            contentStack.addArrangedSubview(card)
        }
    }

    private func queryAccount(_ index: Int) {
        PortalService.shared.queryJdAccount(index: index) { [weak self] result in
            switch result {
            case .failure(let error): self?.handle(error)
            case .success(let text):
                let vc = QueryResultViewController(title: "查询结果", content: text)
                self?.navigationController?.pushViewController(vc, animated: true)
            }
        }
    }

    private func sendSms() {
        let phone = smsPhoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard phone.count == 11 else { showMessage("请输入11位手机号"); return }
        PortalService.shared.sendJdSms(phone: phone) { [weak self] result in
            switch result {
            case .failure(let error): self?.handle(error)
            case .success(let msg):
                self?.showMessage(msg)
                self?.smsIdCardStack.isHidden = true
            }
        }
    }

    private func verifySms() {
        let phone = smsPhoneField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let code = smsCodeField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let idCard = smsIdCardField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        PortalService.shared.verifyJdSms(phone: phone, code: code, idCard: idCard) { [weak self] result in
            switch result {
            case .failure(let error): self?.handle(error)
            case .success(let data):
                if data.needIdVerify == true {
                    self?.smsIdCardStack.isHidden = false
                    self?.showMessage(data.message ?? "需要身份证验证")
                    return
                }
                self?.smsIdCardStack.isHidden = true
                self?.smsResultLabel.text = data.queryResult ?? data.message ?? "登录成功"
                self?.smsResultLabel.isHidden = false
                self?.showMessage(data.message ?? "登录成功")
                self?.loadAccounts()
            }
        }
    }

    private func loadWxDevices() {
        PortalService.shared.fetchJdWxDevices { [weak self] result in
            guard let self = self else { return }
            self.wxDeviceStack.arrangedSubviews.forEach { $0.removeFromSuperview() }
            switch result {
            case .failure(let error):
                self.handle(error)
                self.wxDeviceStack.addArrangedSubview(self.captionLabel(error.message))
            case .success(let list):
                if list.isEmpty {
                    self.wxDeviceStack.addArrangedSubview(self.captionLabel("暂无在线微信协议设备，请先到「更多-微信协议」扫码登录"))
                } else {
                    list.forEach { device in
                        let row = self.cardContainer()
                        row.addArrangedSubview(self.titleLabel(device.nickname ?? device.wxid ?? "未知设备"))
                        row.addArrangedSubview(self.captionLabel(device.wxid ?? ""))
                        row.addArrangedSubview(self.actionButton("刷新 CK", color: .systemBlue) {
                            self.refreshWx(device.wxid ?? "")
                        })
                        self.wxDeviceStack.addArrangedSubview(row)
                    }
                }
            }
        }
    }

    private func refreshWx(_ wxid: String) {
        guard !wxid.isEmpty else { return }
        PortalService.shared.refreshJdWx(wxid: wxid) { [weak self] result in
            switch result {
            case .failure(let error): self?.handle(error)
            case .success(let data):
                self?.wxRiskStack.isHidden = data.needRiskVerify != true
                if data.needRiskVerify == true {
                    self?.wxRiskLabel.text = data.riskMsg ?? "账号需要短信验证"
                    if let urlString = data.riskUrl, let url = URL(string: urlString) {
                        self?.wxRiskStack.addArrangedSubview(self?.actionButton("打开验证链接", color: .systemTeal) {
                            UIApplication.shared.open(url)
                        } ?? UIView())
                    }
                }
                self?.wxResultLabel.text = "成功 \(data.success ?? 0) / 失败 \(data.fail ?? 0)\n\(data.details?.joined(separator: "\n") ?? "")"
                self?.wxResultLabel.isHidden = false
            }
        }
    }

    private func continueWxRisk() {
        PortalService.shared.continueJdWxRisk { [weak self] result in
            switch result {
            case .failure(let error): self?.handle(error)
            case .success(let data):
                self?.wxRiskStack.isHidden = true
                self?.wxResultLabel.text = data.details?.joined(separator: "\n") ?? "刷新完成"
                self?.wxResultLabel.isHidden = false
            }
        }
    }

    private func executeTask(_ task: TaskDef) {
        let selection = Array(taskSelections[task.id] ?? [0])
        guard !selection.isEmpty else { showMessage("请至少选择一个账号"); return }
        PortalService.shared.executeJdTask(taskId: task.id, taskName: task.name, accountIndexes: selection) { [weak self] result in
            switch result {
            case .failure(let error): self?.handle(error)
            case .success(let data):
                guard let logId = data.taskId else { self?.showMessage("任务启动失败"); return }
                self?.runningTasks[task.id] = logId
                self?.appendLog("[\(task.name)] 任务已启动...")
                PortalService.shared.streamJdTaskLogs(taskId: logId, onLine: { line in
                    self?.appendLog("[\(task.name)] \(line)")
                }, onDone: {
                    self?.appendLog("[\(task.name)] ✅ 任务执行完成")
                    self?.runningTasks.removeValue(forKey: task.id)
                }, onError: { error in
                    self?.appendLog("[\(task.name)] 错误: \(error.message)")
                    self?.runningTasks.removeValue(forKey: task.id)
                })
            }
        }
    }

    private func stopTask(_ task: TaskDef) {
        guard let logId = runningTasks[task.id] else { return }
        PortalService.shared.stopJdTask(taskId: logId) { [weak self] _ in
            self?.runningTasks.removeValue(forKey: task.id)
            self?.appendLog("[\(task.name)] ⚠️ 任务已手动停止")
        }
    }

    private func pickAccounts(taskId: String, label: UILabel) {
        let options = buildAccountOptions()
        let alert = UIAlertController(title: "选择账号", message: nil, preferredStyle: .actionSheet)
        var selection = taskSelections[taskId] ?? [0]
        options.forEach { idx, name in
            let selected = selection.contains(idx)
            alert.addAction(UIAlertAction(title: (selected ? "✓ " : "") + name, style: .default) { [weak self] _ in
                if idx == 0 { selection = [0] }
                else {
                    selection.remove(0)
                    if selection.contains(idx) { selection.remove(idx) } else { selection.insert(idx) }
                    if selection.isEmpty { selection = [0] }
                }
                self?.taskSelections[taskId] = selection
                label.text = self?.accountSelectionLabel(selection)
            })
        }
        alert.addAction(UIAlertAction(title: "完成", style: .cancel))
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

    private func tabButton(_ title: String, active: Bool, action: @escaping () -> Void) -> UIButton {
        let btn = UIButton(type: .system)
        btn.setTitle(title, for: .normal)
        btn.titleLabel?.font = .systemFont(ofSize: 13, weight: active ? .bold : .regular)
        btn.setTitleColor(active ? .systemBlue : .secondaryLabel, for: .normal)
        btn.backgroundColor = active ? UIColor.systemBlue.withAlphaComponent(0.1) : .clear
        btn.layer.cornerRadius = 8
        btn.contentEdgeInsets = UIEdgeInsets(top: 8, left: 12, bottom: 8, right: 12)
        bindAction(btn, action)
        return btn
    }

    private func actionButton(_ title: String, color: UIColor, action: @escaping () -> Void) -> UIButton {
        let btn = UIButton(type: .system)
        btn.setTitle(title, for: .normal)
        btn.applyPrimaryStyle(color: color)
        btn.heightAnchor.constraint(equalToConstant: 44).isActive = true
        bindAction(btn, action)
        return btn
    }

    private func bindAction(_ button: UIButton, _ action: @escaping () -> Void) {
        buttonActions[ObjectIdentifier(button)] = action
        button.addTarget(self, action: #selector(buttonTapped(_:)), for: .touchUpInside)
    }

    @objc private func buttonTapped(_ sender: UIButton) {
        buttonActions[ObjectIdentifier(sender)]?()
    }

    private func cardContainer() -> UIStackView {
        let card = UIStackView()
        card.axis = .vertical
        card.spacing = 10
        card.isLayoutMarginsRelativeArrangement = true
        card.layoutMargins = UIEdgeInsets(top: 16, left: 16, bottom: 16, right: 16)
        card.applyCardStyle()
        return card
    }

    private func titleLabel(_ text: String) -> UILabel {
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 16, weight: .bold)
        label.numberOfLines = 0
        return label
    }

    private func captionLabel(_ text: String) -> UILabel {
        let label = UILabel()
        label.text = text
        label.font = .systemFont(ofSize: 12)
        label.textColor = .secondaryLabel
        label.numberOfLines = 0
        return label
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
            label.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16)
        ])
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) has not been implemented") }
}
