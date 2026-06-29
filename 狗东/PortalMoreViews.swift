import UIKit


final class MoreViewController: BaseNativeViewController, UITableViewDataSource, UITableViewDelegate {
    private var unreadCount = 0
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)

    private enum Section: Int, CaseIterable {
        case features = 0
        case about
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemGroupedBackground
        navigationItem.largeTitleDisplayMode = .never
        tableView.dataSource = self
        tableView.delegate = self
        tableView.translatesAutoresizingMaskIntoConstraints = false

        let headerRow = UIView()
        let headerIcon = UILabel()
        headerIcon.text = "⚙️"
        headerIcon.font = .systemFont(ofSize: 24)
        let headerTitle = UILabel()
        headerTitle.text = "更多"
        headerTitle.font = .systemFont(ofSize: 20, weight: .bold)
        let headerSubtitle = UILabel()
        headerSubtitle.text = "通知 · 反馈 · 关于"
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

        view.addSubview(tableView)
        NSLayoutConstraint.activate([
            tableView.topAnchor.constraint(equalTo: headerRow.bottomAnchor, constant: 8),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
        NotificationCenter.default.addObserver(self, selector: #selector(reloadBadge), name: AppNotifications.sessionDidChange, object: nil)
    }

    deinit { NotificationCenter.default.removeObserver(self) }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        reloadBadge()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        navigationController?.setNavigationBarHidden(false, animated: animated)
    }

    func numberOfSections(in tableView: UITableView) -> Int { Section.allCases.count }

    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        switch Section(rawValue: section) {
        case .features: return 2
        case .about: return 2
        default: return 0
        }
    }

    func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        switch Section(rawValue: section) {
        case .features: return "常用功能"
        case .about: return "关于"
        default: return nil
        }
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.accessoryType = .disclosureIndicator
        switch Section(rawValue: indexPath.section) {
        case .features:
            if indexPath.row == 0 {
                cell.textLabel?.text = "消息通知"
                cell.detailTextLabel?.text = "查看管理员推送的所有公告与通知"
                cell.imageView?.image = UIImage(systemName: "bell.fill")
                cell.imageView?.tintColor = .systemBlue
                if unreadCount > 0 {
                    let badge = UILabel()
                    badge.text = unreadCount > 99 ? "99+" : "\(unreadCount)"
                    badge.font = .systemFont(ofSize: 12, weight: .bold)
                    badge.textColor = .white
                    badge.backgroundColor = .systemRed
                    badge.textAlignment = .center
                    badge.layer.cornerRadius = 10
                    badge.clipsToBounds = true
                    badge.sizeToFit()
                    let width = max(badge.bounds.width + 12, 20)
                    badge.frame.size = CGSize(width: width, height: 20)
                    cell.accessoryView = badge
                } else {
                    cell.accessoryView = nil
                }
            } else {
                cell.textLabel?.text = "投稿与反馈"
                cell.detailTextLabel?.text = "提交Bug、活动投稿或建议"
                cell.imageView?.image = UIImage(systemName: "exclamationmark.bubble.fill")
                cell.imageView?.tintColor = .systemOrange
            }
        case .about:
            if indexPath.row == 0 {
                cell.textLabel?.text = "关于版本"
                cell.detailTextLabel?.text = "当前版本与更新信息"
                cell.imageView?.image = UIImage(systemName: "info.circle.fill")
                cell.imageView?.tintColor = .systemGreen
            } else {
                cell.textLabel?.text = "关于作者"
                cell.detailTextLabel?.text = "大师 · QQ: 694738267"
                cell.imageView?.image = UIImage(systemName: "person.fill")
                cell.imageView?.tintColor = .systemPurple
            }
        default: break
        }
        cell.detailTextLabel?.textColor = .secondaryLabel
        cell.detailTextLabel?.numberOfLines = 0
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        switch Section(rawValue: indexPath.section) {
        case .features:
            if indexPath.row == 0 {
                navigationController?.pushViewController(NotificationListViewController(), animated: true)
            } else {
                navigationController?.pushViewController(FeedbackViewController(), animated: true)
            }
        case .about:
            if indexPath.row == 0 {
                navigationController?.pushViewController(AboutVersionViewController(), animated: true)
            } else {
                let url = URL(string: "mqqwpa://im/chat?chat_type=wpa&uin=694738267")
                if let url = url, UIApplication.shared.canOpenURL(url) {
                    UIApplication.shared.open(url)
                }
            }
        default: break
        }
    }

    @objc private func reloadBadge() {
        guard AppSessionStore.shared.isAuthenticated else {
            unreadCount = 0
            tableView.reloadData()
            return
        }
        PortalService.shared.fetchNotifications(includeContent: false) { [weak self] result in
            if case .success(let page) = result {
                self?.unreadCount = page.unread
                self?.tableView.reloadData()
            }
        }
    }
}


final class NotificationListViewController: BaseNativeViewController {
    private var notifications: [PortalNotification] = []
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let refreshControl = UIRefreshControl()

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "消息通知"
        view.backgroundColor = .systemGroupedBackground
        tableView.dataSource = self
        tableView.delegate = self
        tableView.refreshControl = refreshControl
        refreshControl.addTarget(self, action: #selector(loadData), for: .valueChanged)
        tableView.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(tableView)
        NSLayoutConstraint.activate([
            tableView.topAnchor.constraint(equalTo: view.topAnchor),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
        loadData()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        loadData()
    }

    @objc private func loadData() {
        refreshControl.beginRefreshing()
        PortalService.shared.fetchNotifications(includeContent: true) { [weak self] result in
            self?.refreshControl.endRefreshing()
            switch result {
            case .failure(let error):
                self?.handle(error)
            case .success(let page):
                self?.notifications = page.list
                self?.tableView.reloadData()
                if page.unread > 0 {
                    self?.title = "消息通知(\(page.unread)条未读)"
                } else {
                    self?.title = "消息通知"
                }
            }
        }
    }
}


extension NotificationListViewController: UITableViewDataSource, UITableViewDelegate {
    func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int {
        if notifications.isEmpty {
            tableView.backgroundView = EmptyStateView(icon: "bell.slash", title: "暂无通知", desc: "暂时没有任何通知消息")
        } else {
            tableView.backgroundView = nil
        }
        return notifications.count
    }

    func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let item = notifications[indexPath.row]
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.textLabel?.text = item.title ?? "无标题"
        cell.textLabel?.numberOfLines = 2
        cell.textLabel?.font = .systemFont(ofSize: 15, weight: item.isTop == true ? .semibold : .regular)
        cell.textLabel?.textColor = item.isRead ? .secondaryLabel : .label

        var detailParts: [String] = []
        if let createdAt = item.createdAt, !createdAt.isEmpty { detailParts.append(createdAt) }
        let preview = item.content?.prefix(60) ?? ""
        if !preview.isEmpty { detailParts.append(String(preview)) }
        cell.detailTextLabel?.text = detailParts.joined(separator: " · ")
        cell.detailTextLabel?.numberOfLines = 2
        cell.detailTextLabel?.textColor = .tertiaryLabel

        if !item.isRead {
            let dot = UIView()
            dot.backgroundColor = .systemBlue
            dot.layer.cornerRadius = 4
            dot.translatesAutoresizingMaskIntoConstraints = false
            cell.contentView.addSubview(dot)
            NSLayoutConstraint.activate([
                dot.leadingAnchor.constraint(equalTo: cell.contentView.leadingAnchor, constant: 14),
                dot.centerYAnchor.constraint(equalTo: cell.textLabel!.centerYAnchor),
                dot.widthAnchor.constraint(equalToConstant: 8),
                dot.heightAnchor.constraint(equalToConstant: 8)
            ])
            cell.textLabel?.leadingAnchor.constraint(equalTo: dot.trailingAnchor, constant: 8).isActive = true
        }
        if item.isTop == true {
            let topBadge = UILabel()
            topBadge.text = " 置顶 "
            topBadge.font = .systemFont(ofSize: 10, weight: .bold)
            topBadge.textColor = .white
            topBadge.backgroundColor = .systemRed
            topBadge.layer.cornerRadius = 4
            topBadge.clipsToBounds = true
            topBadge.translatesAutoresizingMaskIntoConstraints = false
            cell.contentView.addSubview(topBadge)
            NSLayoutConstraint.activate([
                topBadge.trailingAnchor.constraint(equalTo: cell.contentView.trailingAnchor, constant: -14),
                topBadge.centerYAnchor.constraint(equalTo: cell.textLabel!.centerYAnchor),
                topBadge.heightAnchor.constraint(equalToConstant: 18)
            ])
        }
        cell.accessoryType = .disclosureIndicator
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let item = notifications[indexPath.row]
        if !item.isRead {
            PortalService.shared.markNotificationRead(id: item.id) { _ in }
            notifications[indexPath.row] = PortalNotification(
                id: item.id, title: item.title, content: item.content, category: item.category,
                source: item.source, isRead: true, displayType: item.displayType,
                isTop: item.isTop, createdAt: item.createdAt, readAt: item.readAt
            )
            tableView.reloadRows(at: [indexPath], with: .none)
            let unreadCount = notifications.filter { !$0.isRead }.count
            title = unreadCount > 0 ? "消息通知(\(unreadCount)条未读)" : "消息通知"
        }
        navigationController?.pushViewController(NotificationDetailViewController(notification: item), animated: true)
    }
}


final class NotificationDetailViewController: UIViewController {
    private let notification: PortalNotification
    private let textView = UITextView()

    init(notification: PortalNotification) {
        self.notification = notification
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = notification.title ?? "通知详情"
        view.backgroundColor = .systemBackground

        let titleLabel = UILabel()
        titleLabel.text = notification.title ?? "无标题"
        titleLabel.font = .systemFont(ofSize: 20, weight: .bold)
        titleLabel.numberOfLines = 0

        var metaParts: [String] = []
        if let category = notification.category { metaParts.append("分类：\(category)") }
        if let createdAt = notification.createdAt { metaParts.append(createdAt) }
        let metaLabel = UILabel()
        metaLabel.text = metaParts.joined(separator: " · ")
        metaLabel.font = .systemFont(ofSize: 12)
        metaLabel.textColor = .secondaryLabel

        textView.isEditable = false
        textView.font = .systemFont(ofSize: 15)
        textView.text = notification.content ?? "加载中..."
        textView.textContainerInset = UIEdgeInsets(top: 12, left: 8, bottom: 12, right: 8)

        let separator = UIView()
        separator.backgroundColor = .separator
        separator.heightAnchor.constraint(equalToConstant: 1.0 / UIScreen.main.scale).isActive = true

        let stack = UIStackView(arrangedSubviews: [titleLabel, metaLabel, separator, textView])
        stack.axis = .vertical
        stack.spacing = 12
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: view.bottomAnchor, constant: -16)
        ])

        if let content = notification.content, !content.isEmpty {
            textView.text = content
        }
        PortalService.shared.fetchNotificationDetail(id: notification.id) { [weak self] result in
            if case .success(let detail) = result, let content = detail.content, !content.isEmpty {
                self?.textView.text = content
            }
        }
    }
}


final class FeedbackViewController: BaseNativeViewController, UITextFieldDelegate {
    private let typeLabel = UILabel()
    private let titleField = UITextField()
    private let contentTextView = UITextView()
    private let contactField = UITextField()
    private var selectedType = ""
    private var isSubmitting = false

    override func viewDidLoad() {
        super.viewDidLoad()
        title = "投稿与反馈"
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
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 20),
            stack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -20),
            stack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -32)
        ])

        typeLabel.text = "请点击选择反馈类型 ›"
        typeLabel.font = .systemFont(ofSize: 15)
        typeLabel.textColor = .systemBlue
        typeLabel.isUserInteractionEnabled = true
        typeLabel.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(selectType)))

        titleField.applyAppInputStyle(placeholder: "请输入主题（必填）")
        titleField.heightAnchor.constraint(equalToConstant: 44).isActive = true

        contentTextView.font = .systemFont(ofSize: 15)
        contentTextView.textColor = .label
        contentTextView.backgroundColor = .secondarySystemBackground
        contentTextView.layer.cornerRadius = 10
        contentTextView.textContainerInset = UIEdgeInsets(top: 10, left: 8, bottom: 10, right: 8)
        contentTextView.heightAnchor.constraint(equalToConstant: 120).isActive = true

        contactField.applyAppInputStyle(placeholder: "联系方式（选填，QQ或微信）")
        contactField.heightAnchor.constraint(equalToConstant: 44).isActive = true

        let submitBtn = UIButton(type: .system)
        submitBtn.setTitle("提交反馈", for: .normal)
        submitBtn.titleLabel?.font = .systemFont(ofSize: 17, weight: .semibold)
        submitBtn.backgroundColor = .systemBlue
        submitBtn.setTitleColor(.white, for: .normal)
        submitBtn.layer.cornerRadius = 12
        submitBtn.heightAnchor.constraint(equalToConstant: 50).isActive = true
        submitBtn.addTarget(self, action: #selector(submitTapped), for: .touchUpInside)

        let tipLabel = UILabel()
        tipLabel.text = "提示：反馈被采纳将获得积分奖励。Bug反馈、活动投稿、建议等均可提交。"
        tipLabel.font = .systemFont(ofSize: 12)
        tipLabel.textColor = .secondaryLabel
        tipLabel.numberOfLines = 0

        [typeLabel, titleField, contentTextView, contactField, submitBtn, tipLabel].forEach {
            stack.addArrangedSubview($0)
        }

        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissKeyboard))
        tap.cancelsTouchesInView = false
        view.addGestureRecognizer(tap)
    }

    @objc private func dismissKeyboard() { view.endEditing(true) }

    @objc private func selectType() {
        let alert = UIAlertController(title: "选择反馈类型", message: nil, preferredStyle: .actionSheet)
        ["bug反馈", "活动投稿", "建议"].forEach { type in
            alert.addAction(UIAlertAction(title: type, style: .default) { [weak self] _ in
                self?.selectedType = type
                self?.typeLabel.text = "已选：\(type)"
                self?.typeLabel.textColor = .label
            })
        }
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(alert, animated: true)
    }

    @objc private func submitTapped() {
        guard !isSubmitting else { return }
        if selectedType.isEmpty { showMessage("请先选择反馈类型"); return }
        let title = titleField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let content = contentTextView.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let contact = contactField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if title.isEmpty { showMessage("请输入主题"); return }
        if content.isEmpty { showMessage("请输入详细内容"); return }
        isSubmitting = true
        PortalService.shared.submitFeedback(type: selectedType, title: title, content: content, contact: contact) { [weak self] result in
            self?.isSubmitting = false
            switch result {
            case .failure(let error): self?.handle(error)
            case .success(let msg):
                self?.showMessage(msg) { self?.navigationController?.popViewController(animated: true) }
            }
        }
    }
}


final class AboutVersionViewController: UITableViewController {
    override func viewDidLoad() {
        super.viewDidLoad()
        title = "关于版本"
        tableView = UITableView(frame: .zero, style: .insetGrouped)
    }

    override func numberOfSections(in tableView: UITableView) -> Int { 2 }

    override func tableView(_ tableView: UITableView, numberOfRowsInSection section: Int) -> Int { 1 }

    override func tableView(_ tableView: UITableView, titleForHeaderInSection section: Int) -> String? {
        section == 0 ? "版本信息" : "更新内容"
    }

    override func tableView(_ tableView: UITableView, cellForRowAt indexPath: IndexPath) -> UITableViewCell {
        let cell = UITableViewCell(style: .subtitle, reuseIdentifier: nil)
        cell.selectionStyle = .none
        if indexPath.section == 0 {
            let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "4.5"
            cell.textLabel?.text = "狗东 v\(version)"
            cell.textLabel?.font = .systemFont(ofSize: 17, weight: .semibold)
            cell.detailTextLabel?.text = "设备：\(UIDevice.current.model) · iOS \(UIDevice.current.systemVersion)"
            cell.detailTextLabel?.textColor = .secondaryLabel
            cell.detailTextLabel?.numberOfLines = 0
        } else {
            cell.textLabel?.text = "• 新增消息通知中心\n• 新增投稿与反馈功能\n• 项目中心整合微信协议\n• 首页增加通知模块\n• 支持页面滑动切换\n• 液态玻璃视觉效果\n• 优化自动登录体验"
            cell.textLabel?.numberOfLines = 0
            cell.textLabel?.font = .systemFont(ofSize: 14)
            cell.textLabel?.textColor = .secondaryLabel
        }
        return cell
    }
}


enum LiquidGlassEffect {
    static func applyToTabBar(_ tabBar: UITabBar) {
        let appearance = UITabBarAppearance()
        appearance.configureWithDefaultBackground()
        appearance.backgroundEffect = UIBlurEffect(style: .systemMaterial)
        appearance.backgroundColor = UIColor.systemBackground
        appearance.shadowColor = UIColor.separator.withAlphaComponent(0.2)
        tabBar.standardAppearance = appearance
        if #available(iOS 15.0, *) {
            tabBar.scrollEdgeAppearance = appearance
        }
    }

    static func applyToNavBar(_ navBar: UINavigationBar) {
        let appearance = UINavigationBarAppearance()
        appearance.configureWithTransparentBackground()
        appearance.backgroundEffect = UIBlurEffect(style: .systemMaterial)
        appearance.backgroundColor = .clear
        appearance.shadowColor = .clear
        appearance.titleTextAttributes = [.foregroundColor: UIColor.label]
        appearance.largeTitleTextAttributes = [.foregroundColor: UIColor.label]
        navBar.standardAppearance = appearance
        navBar.scrollEdgeAppearance = appearance
        navBar.compactAppearance = appearance
    }

    static func applyGlassStyle(to view: UIView, cornerRadius: CGFloat = 16) {
        let blur = UIBlurEffect(style: .systemMaterial)
        let blurView = UIVisualEffectView(effect: blur)
        blurView.frame = view.bounds
        blurView.autoresizingMask = [.flexibleWidth, .flexibleHeight]
        blurView.layer.cornerRadius = cornerRadius
        blurView.clipsToBounds = true
        view.insertSubview(blurView, at: 0)
        view.backgroundColor = .clear
    }

    static func tabBarBounceAnimation(_ view: UIView) {
        UIView.animate(withDuration: 0.15, delay: 0, usingSpringWithDamping: 0.7, initialSpringVelocity: 0.5, options: []) {
            view.transform = CGAffineTransform(scaleX: 0.85, y: 0.85)
        } completion: { _ in
            UIView.animate(withDuration: 0.25, delay: 0, usingSpringWithDamping: 0.5, initialSpringVelocity: 0.3, options: []) {
                view.transform = .identity
            }
        }
    }
}
