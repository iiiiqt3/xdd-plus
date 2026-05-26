import UIKit

@available(iOS 13.0, *)
final class MoreViewController: BaseNativeViewController {
    private let scrollView = UIScrollView()
    private let stack = UIStackView()
    private let notificationBadgeLabel = UILabel()
    private lazy var heroView = SectionHeroView(icon: "line.3.horizontal", title: "更多", subtitle: "消息通知、反馈与系统信息", tint: .systemBlue)

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
        NotificationCenter.default.addObserver(self, selector: #selector(reloadBadge), name: AppNotifications.sessionDidChange, object: nil)
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        reloadBadge()
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
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -16),
            stack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -32)
        ])

        stack.addArrangedSubview(heroView)

        let notificationAction = ActionButton(icon: "bell.fill", title: "消息通知", desc: "查看管理员推送的所有公告与通知", tint: .systemBlue)
        notificationAction.tapAction = { [weak self] in
            self?.navigationController?.pushViewController(NotificationListViewController(), animated: true)
        }
        notificationBadgeLabel.translatesAutoresizingMaskIntoConstraints = false
        notificationBadgeLabel.textColor = .white
        notificationBadgeLabel.font = UIFont.systemFont(ofSize: 11, weight: .bold)
        notificationBadgeLabel.textAlignment = .center
        notificationBadgeLabel.backgroundColor = .systemRed
        notificationBadgeLabel.layer.cornerRadius = 10
        notificationBadgeLabel.clipsToBounds = true
        notificationBadgeLabel.isHidden = true
        notificationAction.addSubview(notificationBadgeLabel)
        NSLayoutConstraint.activate([
            notificationBadgeLabel.topAnchor.constraint(equalTo: notificationAction.topAnchor, constant: 6),
            notificationBadgeLabel.trailingAnchor.constraint(equalTo: notificationAction.trailingAnchor, constant: -6),
            notificationBadgeLabel.heightAnchor.constraint(equalToConstant: 20),
            notificationBadgeLabel.widthAnchor.constraint(greaterThanOrEqualToConstant: 20)
        ])

        let feedbackAction = ActionButton(icon: "exclamationmark.bubble.fill", title: "投稿与反馈", desc: "提交Bug、活动投稿或建议", tint: .systemOrange)
        feedbackAction.tapAction = { [weak self] in
            self?.navigationController?.pushViewController(FeedbackViewController(), animated: true)
        }

        let notifCard = makeSectionCard(title: "常用功能", items: [notificationAction, feedbackAction])
        stack.addArrangedSubview(notifCard)

        let versionAction = ActionButton(icon: "info.circle.fill", title: "关于版本", desc: "当前版本号与更新信息", tint: .systemGreen)
        versionAction.tapAction = { [weak self] in
            self?.navigationController?.pushViewController(AboutVersionViewController(), animated: true)
        }

        let authorAction = ActionButton(icon: "person.fill", title: "关于作者", desc: "大师 · QQ: 694738267", tint: .systemPurple)
        authorAction.tapAction = {
            let url = URL(string: "mqqwpa://im/chat?chat_type=wpa&uin=694738267")
            if let url = url, UIApplication.shared.canOpenURL(url) {
                UIApplication.shared.open(url)
            }
        }

        let aboutCard = makeSectionCard(title: "关于", items: [versionAction, authorAction])
        stack.addArrangedSubview(aboutCard)
    }

    private func makeSectionCard(title: String, items: [UIView]) -> UIView {
        let card = UIView()
        card.applyCardStyle()
        let titleLabel = UILabel()
        titleLabel.text = title
        titleLabel.font = UIFont.systemFont(ofSize: 13, weight: .semibold)
        titleLabel.textColor = .secondaryLabel
        let innerStack = UIStackView(arrangedSubviews: [titleLabel] + items)
        innerStack.axis = .vertical
        innerStack.spacing = 10
        innerStack.translatesAutoresizingMaskIntoConstraints = false
        card.addSubview(innerStack)
        NSLayoutConstraint.activate([
            innerStack.topAnchor.constraint(equalTo: card.topAnchor, constant: 18),
            innerStack.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 18),
            innerStack.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -18),
            innerStack.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -18)
        ])
        return card
    }

    @objc private func reloadBadge() {
        guard AppSessionStore.shared.isAuthenticated else {
            notificationBadgeLabel.isHidden = true
            return
        }
        PortalService.shared.fetchNotifications(includeContent: false) { [weak self] result in
            if case .success(let page) = result {
                let count = page.unread
                if count > 0 {
                    self?.notificationBadgeLabel.text = count > 99 ? "99+" : "\(count)"
                    self?.notificationBadgeLabel.isHidden = false
                    let width = count > 99 ? 32 : 24
                    self?.notificationBadgeLabel.widthAnchor.constraint(equalToConstant: CGFloat(width)).isActive = true
                } else {
                    self?.notificationBadgeLabel.isHidden = true
                }
            }
        }
    }
}

@available(iOS 13.0, *)
final class NotificationListViewController: BaseNativeViewController {
    private let tableView = UITableView(frame: .zero, style: .insetGrouped)
    private let refreshControl = UIRefreshControl()
    private var notifications: [PortalNotification] = []
    private lazy var heroView = SectionHeroView(icon: "bell.fill", title: "消息通知", subtitle: "管理员推送的所有公告与通知", tint: .systemBlue)

    override func viewDidLoad() {
        super.viewDidLoad()
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemGroupedBackground
        setupUI()
        loadData()
    }

    private func setupUI() {
        heroView.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(heroView)
        tableView.translatesAutoresizingMaskIntoConstraints = false
        tableView.dataSource = self
        tableView.delegate = self
        tableView.refreshControl = refreshControl
        refreshControl.addTarget(self, action: #selector(loadData), for: .valueChanged)
        view.addSubview(tableView)
        NSLayoutConstraint.activate([
            heroView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 12),
            heroView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            heroView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            tableView.topAnchor.constraint(equalTo: heroView.bottomAnchor, constant: 12),
            tableView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            tableView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            tableView.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
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
                }
            }
        }
    }
}

@available(iOS 13.0, *)
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
        cell.textLabel?.font = UIFont.systemFont(ofSize: 15, weight: item.isTop == true ? .bold : .regular)
        cell.textLabel?.textColor = item.isRead ? .secondaryLabel : .label
        let preview = item.content?.prefix(80) ?? ""
        cell.detailTextLabel?.text = String(preview)
        cell.detailTextLabel?.numberOfLines = 2
        cell.detailTextLabel?.textColor = .tertiaryLabel
        cell.detailTextLabel?.font = UIFont.systemFont(ofSize: 13)
        if !item.isRead {
            let dot = UIView()
            dot.translatesAutoresizingMaskIntoConstraints = false
            dot.backgroundColor = .systemBlue
            dot.layer.cornerRadius = 4
            cell.contentView.addSubview(dot)
            NSLayoutConstraint.activate([
                dot.leadingAnchor.constraint(equalTo: cell.contentView.leadingAnchor, constant: 12),
                dot.centerYAnchor.constraint(equalTo: cell.textLabel!.centerYAnchor),
                dot.widthAnchor.constraint(equalToConstant: 8),
                dot.heightAnchor.constraint(equalToConstant: 8)
            ])
            cell.textLabel?.leadingAnchor.constraint(equalTo: dot.trailingAnchor, constant: 8).isActive = true
        }
        if item.isTop == true {
            let topBadge = UILabel()
            topBadge.text = " 置顶 "
            topBadge.font = UIFont.systemFont(ofSize: 10, weight: .bold)
            topBadge.textColor = .white
            topBadge.backgroundColor = .systemRed
            topBadge.layer.cornerRadius = 4
            topBadge.clipsToBounds = true
            topBadge.translatesAutoresizingMaskIntoConstraints = false
            cell.contentView.addSubview(topBadge)
            NSLayoutConstraint.activate([
                topBadge.trailingAnchor.constraint(equalTo: cell.contentView.trailingAnchor, constant: -12),
                topBadge.topAnchor.constraint(equalTo: cell.contentView.topAnchor, constant: 8),
                topBadge.heightAnchor.constraint(equalToConstant: 18)
            ])
        }
        cell.accessoryType = .disclosureIndicator
        return cell
    }

    func tableView(_ tableView: UITableView, didSelectRowAt indexPath: IndexPath) {
        tableView.deselectRow(at: indexPath, animated: true)
        let item = notifications[indexPath.row]
        let detail = NotificationDetailViewController(notification: item)
        navigationController?.pushViewController(detail, animated: true)
    }
}

@available(iOS 13.0, *)
final class NotificationDetailViewController: BaseNativeViewController {
    private let notification: PortalNotification
    private let textView = UITextView()

    init(notification: PortalNotification) {
        self.notification = notification
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        title = notification.title ?? "通知详情"
        navigationItem.largeTitleDisplayMode = .never
        view.backgroundColor = .systemBackground
        setupUI()
        loadDetail()
    }

    private func setupUI() {
        let headerCard = UIView()
        headerCard.applyCardStyle(cornerRadius: 22)
        headerCard.translatesAutoresizingMaskIntoConstraints = false

        let titleLabel = UILabel()
        titleLabel.text = notification.title ?? "无标题"
        titleLabel.font = UIFont.systemFont(ofSize: 20, weight: .bold)
        titleLabel.textColor = .label
        titleLabel.numberOfLines = 0

        let metaLabel = UILabel()
        var metaParts: [String] = []
        if let category = notification.category { metaParts.append("分类：\(category)") }
        if let source = notification.source { metaParts.append("来源：\(source)") }
        if let createdAt = notification.createdAt { metaParts.append(createdAt) }
        metaLabel.text = metaParts.joined(separator: " · ")
        metaLabel.font = UIFont.systemFont(ofSize: 12)
        metaLabel.textColor = .secondaryLabel
        metaLabel.numberOfLines = 0

        let headerStack = UIStackView(arrangedSubviews: [titleLabel, metaLabel])
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

        textView.translatesAutoresizingMaskIntoConstraints = false
        textView.isEditable = false
        textView.font = UIFont.systemFont(ofSize: 15)
        textView.textColor = .label
        textView.backgroundColor = .secondarySystemBackground
        textView.layer.cornerRadius = 18
        textView.textContainerInset = UIEdgeInsets(top: 16, left: 14, bottom: 16, right: 14)
        textView.text = "加载中..."

        view.addSubview(headerCard)
        view.addSubview(textView)
        NSLayoutConstraint.activate([
            headerCard.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 16),
            headerCard.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            headerCard.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            textView.topAnchor.constraint(equalTo: headerCard.bottomAnchor, constant: 12),
            textView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
            textView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
            textView.bottomAnchor.constraint(equalTo: view.bottomAnchor, constant: -16)
        ])
    }

    private func loadDetail() {
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

@available(iOS 13.0, *)
final class FeedbackViewController: BaseNativeViewController {
    private let typeLabel = UILabel()
    private let titleField = UITextField()
    private let contentTextView = UITextView()
    private let contactField = UITextField()
    private var selectedType = ""
    private var isSubmitting = false
    private lazy var heroView = SectionHeroView(icon: "exclamationmark.bubble.fill", title: "投稿与反馈", subtitle: "提交Bug、活动投稿或建议", tint: .systemOrange)

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
        stack.spacing = 14
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -16),
            stack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -32)
        ])

        stack.addArrangedSubview(heroView)

        let typeCard = UIView()
        typeCard.applyCardStyle()
        let typeTitle = UILabel()
        typeTitle.text = "反馈类型（必选）"
        typeTitle.font = UIFont.systemFont(ofSize: 15, weight: .semibold)
        typeTitle.textColor = .systemOrange
        typeLabel.text = "请点击选择反馈类型 ›"
        typeLabel.font = UIFont.systemFont(ofSize: 15)
        typeLabel.textColor = .tertiaryLabel
        typeLabel.isUserInteractionEnabled = true
        typeLabel.addGestureRecognizer(UITapGestureRecognizer(target: self, action: #selector(selectType)))
        let tipLabel = UILabel()
        tipLabel.text = "反馈被采纳将获得积分奖励！"
        tipLabel.font = UIFont.systemFont(ofSize: 12, weight: .bold)
        tipLabel.textColor = .systemOrange
        let typeStack = UIStackView(arrangedSubviews: [typeTitle, typeLabel, tipLabel])
        typeStack.axis = .vertical
        typeStack.spacing = 8
        typeStack.translatesAutoresizingMaskIntoConstraints = false
        typeCard.addSubview(typeStack)
        NSLayoutConstraint.activate([
            typeStack.topAnchor.constraint(equalTo: typeCard.topAnchor, constant: 18),
            typeStack.leadingAnchor.constraint(equalTo: typeCard.leadingAnchor, constant: 18),
            typeStack.trailingAnchor.constraint(equalTo: typeCard.trailingAnchor, constant: -18),
            typeStack.bottomAnchor.constraint(equalTo: typeCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(typeCard)

        titleField.applyAppInputStyle(placeholder: "请输入主题（必填）")
        titleField.heightAnchor.constraint(equalToConstant: 50).isActive = true
        stack.addArrangedSubview(titleField)

        let contentCard = UIView()
        contentCard.applyCardStyle()
        let contentTitle = UILabel()
        contentTitle.text = "详细内容（必填）"
        contentTitle.font = UIFont.systemFont(ofSize: 15, weight: .semibold)
        contentTextView.font = UIFont.systemFont(ofSize: 15)
        contentTextView.textColor = .label
        contentTextView.backgroundColor = .tertiarySystemBackground
        contentTextView.layer.cornerRadius = 12
        contentTextView.textContainerInset = UIEdgeInsets(top: 12, left: 8, bottom: 12, right: 8)
        contentTextView.translatesAutoresizingMaskIntoConstraints = false
        contentTextView.heightAnchor.constraint(equalToConstant: 150).isActive = true
        let contentStack = UIStackView(arrangedSubviews: [contentTitle, contentTextView])
        contentStack.axis = .vertical
        contentStack.spacing = 10
        contentStack.translatesAutoresizingMaskIntoConstraints = false
        contentCard.addSubview(contentStack)
        NSLayoutConstraint.activate([
            contentStack.topAnchor.constraint(equalTo: contentCard.topAnchor, constant: 18),
            contentStack.leadingAnchor.constraint(equalTo: contentCard.leadingAnchor, constant: 18),
            contentStack.trailingAnchor.constraint(equalTo: contentCard.trailingAnchor, constant: -18),
            contentStack.bottomAnchor.constraint(equalTo: contentCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(contentCard)

        contactField.applyAppInputStyle(placeholder: "联系方式（选填，QQ或微信）")
        contactField.heightAnchor.constraint(equalToConstant: 50).isActive = true
        stack.addArrangedSubview(contactField)

        let submitButton = UIButton(type: .system)
        submitButton.setTitle("提交反馈", for: .normal)
        submitButton.applyPrimaryStyle(color: .systemBlue)
        submitButton.heightAnchor.constraint(equalToConstant: 52).isActive = true
        submitButton.addTarget(self, action: #selector(submitTapped), for: .touchUpInside)
        stack.addArrangedSubview(submitButton)

        let hintCard = UIView()
        hintCard.backgroundColor = UIColor.systemBlue.withAlphaComponent(0.06)
        hintCard.layer.cornerRadius = 14
        let hintText = UILabel()
        hintText.text = "提示\n• Bug反馈：发现App问题请选择此项\n• 活动投稿：参与活动或分享经验\n• 建议：功能改进建议或其他想法\n\n提交后管理员会在后台处理，处理结果会在消息通知中推送给你。"
        hintText.font = UIFont.systemFont(ofSize: 12)
        hintText.textColor = .secondaryLabel
        hintText.numberOfLines = 0
        let hintStack = UIStackView(arrangedSubviews: [hintText])
        hintStack.translatesAutoresizingMaskIntoConstraints = false
        hintCard.addSubview(hintStack)
        NSLayoutConstraint.activate([
            hintStack.topAnchor.constraint(equalTo: hintCard.topAnchor, constant: 14),
            hintStack.leadingAnchor.constraint(equalTo: hintCard.leadingAnchor, constant: 14),
            hintStack.trailingAnchor.constraint(equalTo: hintCard.trailingAnchor, constant: -14),
            hintStack.bottomAnchor.constraint(equalTo: hintCard.bottomAnchor, constant: -14)
        ])
        stack.addArrangedSubview(hintCard)

        let tap = UITapGestureRecognizer(target: self, action: #selector(dismissKeyboard))
        tap.cancelsTouchesInView = false
        view.addGestureRecognizer(tap)
    }

    @objc private func dismissKeyboard() {
        view.endEditing(true)
    }

    @objc private func selectType() {
        let alert = UIAlertController(title: "选择反馈类型", message: nil, preferredStyle: .actionSheet)
        alert.addAction(UIAlertAction(title: "bug反馈", style: .default) { [weak self] _ in self?.setType("bug反馈") })
        alert.addAction(UIAlertAction(title: "活动投稿", style: .default) { [weak self] _ in self?.setType("活动投稿") })
        alert.addAction(UIAlertAction(title: "建议", style: .default) { [weak self] _ in self?.setType("建议") })
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(alert, animated: true)
    }

    private func setType(_ type: String) {
        selectedType = type
        typeLabel.text = "已选：\(type)"
        typeLabel.textColor = .label
        typeLabel.font = UIFont.systemFont(ofSize: 15, weight: .semibold)
    }

    @objc private func submitTapped() {
        guard !isSubmitting else { return }
        if selectedType.isEmpty {
            showMessage("请先选择反馈类型")
            return
        }
        let title = titleField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let content = contentTextView.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let contact = contactField.text?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if title.isEmpty {
            showMessage("请输入主题")
            return
        }
        if content.isEmpty {
            showMessage("请输入详细内容")
            return
        }
        isSubmitting = true
        PortalService.shared.submitFeedback(type: selectedType, title: title, content: content, contact: contact) { [weak self] result in
            self?.isSubmitting = false
            switch result {
            case .failure(let error):
                self?.handle(error)
            case .success(let msg):
                self?.showMessage(msg) {
                    self?.navigationController?.popViewController(animated: true)
                }
            }
        }
    }
}

@available(iOS 13.0, *)
final class AboutVersionViewController: BaseNativeViewController {
    private lazy var heroView = SectionHeroView(icon: "info.circle.fill", title: "关于版本", subtitle: "当前版本信息与系统详情", tint: .systemGreen)

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
        stack.spacing = 14
        stack.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(scrollView)
        scrollView.addSubview(stack)
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            stack.topAnchor.constraint(equalTo: scrollView.topAnchor, constant: 16),
            stack.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor, constant: 16),
            stack.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor, constant: -16),
            stack.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor, constant: -16),
            stack.widthAnchor.constraint(equalTo: scrollView.widthAnchor, constant: -32)
        ])

        stack.addArrangedSubview(heroView)

        let infoCard = UIView()
        infoCard.applyCardStyle()
        let version = Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "未知"
        let build = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "未知"
        let device = UIDevice.current.model
        let system = UIDevice.current.systemVersion
        let infoText = UILabel()
        infoText.numberOfLines = 0
        infoText.font = UIFont.systemFont(ofSize: 14)
        infoText.textColor = .label
        infoText.text = "版本号：v\(version) (\(build))\n设备：\(device)\n系统版本：iOS \(system)\n服务端地址：\(AppEnvironment.baseURL.absoluteString)"
        let infoStack = UIStackView(arrangedSubviews: [infoText])
        infoStack.translatesAutoresizingMaskIntoConstraints = false
        infoCard.addSubview(infoStack)
        NSLayoutConstraint.activate([
            infoStack.topAnchor.constraint(equalTo: infoCard.topAnchor, constant: 18),
            infoStack.leadingAnchor.constraint(equalTo: infoCard.leadingAnchor, constant: 18),
            infoStack.trailingAnchor.constraint(equalTo: infoCard.trailingAnchor, constant: -18),
            infoStack.bottomAnchor.constraint(equalTo: infoCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(infoCard)

        let changelogCard = UIView()
        changelogCard.applyCardStyle()
        let changelogTitle = UILabel()
        changelogTitle.text = "更新内容"
        changelogTitle.font = UIFont.systemFont(ofSize: 17, weight: .bold)
        let changelogText = UILabel()
        changelogText.numberOfLines = 0
        changelogText.font = UIFont.systemFont(ofSize: 14)
        changelogText.textColor = .secondaryLabel
        changelogText.text = "• 新增消息通知中心，实时查看管理员推送\n• 新增投稿与反馈功能\n• 项目中心新增微信协议子页面\n• 首页增加通知模块和项目统计\n• 支持页面滑动切换标签\n• 液态玻璃视觉效果\n• 优化自动登录体验"
        let changelogStack = UIStackView(arrangedSubviews: [changelogTitle, changelogText])
        changelogStack.axis = .vertical
        changelogStack.spacing = 10
        changelogStack.translatesAutoresizingMaskIntoConstraints = false
        changelogCard.addSubview(changelogStack)
        NSLayoutConstraint.activate([
            changelogStack.topAnchor.constraint(equalTo: changelogCard.topAnchor, constant: 18),
            changelogStack.leadingAnchor.constraint(equalTo: changelogCard.leadingAnchor, constant: 18),
            changelogStack.trailingAnchor.constraint(equalTo: changelogCard.trailingAnchor, constant: -18),
            changelogStack.bottomAnchor.constraint(equalTo: changelogCard.bottomAnchor, constant: -18)
        ])
        stack.addArrangedSubview(changelogCard)
    }
}

@available(iOS 13.0, *)
final class LiquidGlassTabBar: UITabBarController {
    private var swipeLeft: UISwipeGestureRecognizer?
    private var swipeRight: UISwipeGestureRecognizer?

    override func viewDidLoad() {
        super.viewDidLoad()
        setupSwipeGestures()
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
}

@available(iOS 13.0, *)
enum LiquidGlassEffect {
    static func applyFrostedGlass(to view: UIView, intensity: CGFloat = 0.6) {
        let blurEffect = UIBlurEffect(style: .systemUltraThinMaterial)
        let blurView = UIVisualEffectView(effect: blurEffect)
        blurView.translatesAutoresizingMaskIntoConstraints = false
        blurView.alpha = intensity
        view.insertSubview(blurView, at: 0)
        NSLayoutConstraint.activate([
            blurView.topAnchor.constraint(equalTo: view.topAnchor),
            blurView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            blurView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            blurView.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
        view.backgroundColor = .clear
        let overlay = UIView()
        overlay.translatesAutoresizingMaskIntoConstraints = false
        overlay.backgroundColor = UIColor.white.withAlphaComponent(0.15)
        overlay.isUserInteractionEnabled = false
        blurView.contentView.addSubview(overlay)
        NSLayoutConstraint.activate([
            overlay.topAnchor.constraint(equalTo: blurView.contentView.topAnchor),
            overlay.leadingAnchor.constraint(equalTo: blurView.contentView.leadingAnchor),
            overlay.trailingAnchor.constraint(equalTo: blurView.contentView.trailingAnchor),
            overlay.bottomAnchor.constraint(equalTo: blurView.contentView.bottomAnchor)
        ])
    }

    static func applyGlassStyle(to view: UIView, cornerRadius: CGFloat = 20, borderOpacity: CGFloat = 0.2) {
        view.layer.cornerRadius = cornerRadius
        view.layer.masksToBounds = false
        view.layer.borderWidth = 0.5
        view.layer.borderColor = UIColor.white.withAlphaComponent(borderOpacity).cgColor
        view.layer.shadowColor = UIColor.black.cgColor
        view.layer.shadowOpacity = 0.06
        view.layer.shadowOffset = CGSize(width: 0, height: 4)
        view.layer.shadowRadius = 16
    }

    static func createGlassCard() -> UIView {
        let card = UIView()
        let blur = UIBlurEffect(style: .systemMaterial)
        let blurView = UIVisualEffectView(effect: blur)
        blurView.translatesAutoresizingMaskIntoConstraints = false
        card.addSubview(blurView)
        NSLayoutConstraint.activate([
            blurView.topAnchor.constraint(equalTo: card.topAnchor),
            blurView.leadingAnchor.constraint(equalTo: card.leadingAnchor),
            blurView.trailingAnchor.constraint(equalTo: card.trailingAnchor),
            blurView.bottomAnchor.constraint(equalTo: card.bottomAnchor)
        ])
        applyGlassStyle(to: card)
        return card
    }

    static func applyGlassToTabBar(_ tabBar: UITabBar) {
        let appearance = UITabBarAppearance()
        appearance.configureWithTransparentBackground()
        appearance.backgroundEffect = UIBlurEffect(style: .systemUltraThinMaterial)
        appearance.backgroundColor = UIColor.systemBackground.withAlphaComponent(0.7)
        appearance.shadowColor = UIColor.clear
        tabBar.standardAppearance = appearance
        if #available(iOS 15.0, *) {
            tabBar.scrollEdgeAppearance = appearance
        }
    }

    static func applyGlassToNavBar(_ navBar: UINavigationBar) {
        let appearance = UINavigationBarAppearance()
        appearance.configureWithTransparentBackground()
        appearance.backgroundEffect = UIBlurEffect(style: .systemUltraThinMaterial)
        appearance.backgroundColor = UIColor.systemBackground.withAlphaComponent(0.7)
        appearance.shadowColor = UIColor.clear
        appearance.titleTextAttributes = [.foregroundColor: UIColor.label]
        appearance.largeTitleTextAttributes = [.foregroundColor: UIColor.label]
        navBar.standardAppearance = appearance
        navBar.scrollEdgeAppearance = appearance
        navBar.compactAppearance = appearance
    }
}
