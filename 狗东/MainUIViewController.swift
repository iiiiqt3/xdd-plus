import UIKit
import WebKit

@available(iOS 13.0, *)
class MainUIViewController: UIViewController, UIScrollViewDelegate, UIGestureRecognizerDelegate {

    // --- 布局常量 ---
    private struct LayoutConstants {
        // 顶部区域
        static let appTitleTopMargin: CGFloat = 2.0
        static let appTitleHeight: CGFloat = 30.0
        
        static let qqAvatarTopMargin: CGFloat = 10.0
        static let qqAvatarTrailingMargin: CGFloat = -20.0
        static let qqAvatarSize: CGFloat = 48.0
        static let qqAvatarToTitleSpacing: CGFloat = 8.0
        
        // 公告区域
        static let noticeTopMargin: CGFloat = 62.0
        static let noticeHorizontalMargin: CGFloat = 20.0
        static let noticeHeightRatio: CGFloat = 0.30
        
        static let noticeHeaderTopMargin: CGFloat = 16.0
        static let noticeHeaderHorizontalMargin: CGFloat = 20.0
        static let noticeHeaderHeight: CGFloat = 40.0
        
        static let scrollViewTopToHeaderSpacing: CGFloat = 16.0
        static let scrollViewHorizontalMargin: CGFloat = 24.0
        static let scrollViewBottomMargin: CGFloat = -16.0
        
        static let noticeContentInsets = UIEdgeInsets(top: 0, left: 0, bottom: 0, right: 0)
        
        // 输入框和按钮区域
        static let qqTextFieldTopToNoticeSpacing: CGFloat = 40.0
        static let qqTextFieldHorizontalMargin: CGFloat = 32.0
        static let qqTextFieldHeight: CGFloat = 56.0
        
        static let buttonHeight: CGFloat = 52.0
        static let buttonHorizontalMargin: CGFloat = 32.0
        static let loginButtonTopToTextFieldSpacing: CGFloat = 28.0
        static let queryButtonTopToLoginButtonSpacing: CGFloat = 24.0
        static let moreWoolButtonTopToQueryButtonSpacing: CGFloat = 24.0
        static let joinGroupButtonTopToMoreWoolButtonSpacing: CGFloat = 24.0
        
        // 指示器
        static let moreIndicatorTrailingMargin: CGFloat = -20.0
        static let moreIndicatorBottomMargin: CGFloat = -12.0
        static let scrollHintToIndicatorSpacing: CGFloat = -6.0
        
        // 渐变层高度
        static let gradientLayerHeight: CGFloat = 30.0
    }
    
    // --- 数据持久化常量 ---
    private struct UserDefaultsKeys {
        static let savedQQNumber = "savedQQNumber"
        static let savedQQAvatarData = "savedQQAvatarData"
    }
    
    // QQ头像视图
    private let qqAvatarImageView: UIImageView = {
        let iv = UIImageView()
        iv.contentMode = .scaleAspectFill
        iv.clipsToBounds = true
        iv.isHidden = true
        return iv
    }()
    
    // App标题标签
    private let appTitleLabel: UILabel = {
        let label = UILabel()
        label.text = "大师京东提交查询系统"
        label.font = UIFont.systemFont(ofSize: 18, weight: .bold)
        label.textColor = .label
        label.textAlignment = .center
        return label
    }()
    
    // 公告视图
    private let noticeView: UIView = {
        let view = UIView()
        view.backgroundColor = .secondarySystemBackground
        view.layer.cornerRadius = 20
        view.layer.shadowColor = UIColor.black.cgColor
        view.layer.shadowOpacity = 0.08
        view.layer.shadowOffset = CGSize(width: 0, height: 4)
        view.layer.shadowRadius = 12
        view.layer.masksToBounds = false
        view.isUserInteractionEnabled = true
        view.clipsToBounds = false
        view.transform = CGAffineTransform(scaleX: 0.95, y: 0.95)
        view.alpha = 0
        return view
    }()
    
    // 公告标题栏
    private let noticeHeaderView: UIView = {
        let view = UIView()
        view.backgroundColor = UIColor.systemBlue.withAlphaComponent(0.1)
        view.layer.cornerRadius = 12
        view.layer.masksToBounds = true
        return view
    }()
    
    // 公告标题
    private let noticeTitleLabel: UILabel = {
        let label = UILabel()
        label.text = "公告"
        label.font = UIFont.systemFont(ofSize: 17, weight: .semibold)
        label.textColor = .systemBlue
        label.textAlignment = .center
        return label
    }()
    
    // 滚动视图
    private let scrollView: UIScrollView = {
        let scrollView = UIScrollView()
        scrollView.showsVerticalScrollIndicator = false
        scrollView.showsHorizontalScrollIndicator = false
        scrollView.alwaysBounceVertical = true
        return scrollView
    }()
    
    // 内容容器视图
    private let contentView: UIView = {
        let view = UIView()
        return view
    }()
    
    // 公告内容
    private let noticeContentLabel: UILabel = {
        let label = UILabel()
        label.text = "加载中..."
        label.font = UIFont.systemFont(ofSize: 15)
        label.textColor = .secondaryLabel
        label.numberOfLines = 0
        label.lineBreakMode = .byWordWrapping
        label.textAlignment = .left
        return label
    }()
    
    // 加载指示器
    private let loadingIndicator: UIActivityIndicatorView = {
        let indicator = UIActivityIndicatorView(style: .medium)
        indicator.hidesWhenStopped = true
        indicator.color = .systemBlue
        return indicator
    }()
    
    // 下拉提示箭头
    private let moreIndicatorView: UIImageView = {
        let imageView = UIImageView(image: UIImage(systemName: "chevron-down.circle.fill"))
        imageView.tintColor = .systemBlue
        imageView.alpha = 0.0
        return imageView
    }()
    
    // 滚动提示文本
    private let scrollHintLabel: UILabel = {
        let label = UILabel()
        label.text = "下拉查看更多"
        label.font = UIFont.systemFont(ofSize: 12, weight: .medium)
        label.textColor = .systemBlue
        label.alpha = 0.0
        return label
    }()
    
    // 顶部渐变提示层
    private let topGradientLayer: CAGradientLayer = {
        let layer = CAGradientLayer()
        layer.colors = [UIColor.secondarySystemBackground.cgColor, UIColor.secondarySystemBackground.withAlphaComponent(0).cgColor]
        layer.startPoint = CGPoint(x: 0.5, y: 0)
        layer.endPoint = CGPoint(x: 0.5, y: 1)
        layer.opacity = 0.0
        return layer
    }()
    
    // 底部渐变提示层
    private let bottomGradientLayer: CAGradientLayer = {
        let layer = CAGradientLayer()
        layer.colors = [UIColor.secondarySystemBackground.withAlphaComponent(0).cgColor, UIColor.secondarySystemBackground.cgColor]
        layer.startPoint = CGPoint(x: 0.5, y: 0)
        layer.endPoint = CGPoint(x: 0.5, y: 1)
        layer.opacity = 0.0
        return layer
    }()
    
    // QQ输入框
    let qqTextField: UITextField = {
        let tf = UITextField()
        tf.placeholder = "请输入QQ号"
        tf.borderStyle = .none
        tf.keyboardType = .numberPad
        tf.font = UIFont.systemFont(ofSize: 16)
        tf.textColor = .label
        tf.backgroundColor = .secondarySystemBackground
        tf.layer.cornerRadius = 16
        tf.layer.shadowColor = UIColor.black.cgColor
        tf.layer.shadowOpacity = 0.05
        tf.layer.shadowOffset = CGSize(width: 0, height: 2)
        tf.layer.shadowRadius = 8
        tf.paddingLeft(16)
        tf.paddingRight(16)
        
        tf.attributedPlaceholder = NSAttributedString(
            string: "请输入QQ号",
            attributes: [NSAttributedString.Key.foregroundColor: UIColor.tertiaryLabel]
        )
        return tf
    }()
    
    // 登录按钮
    let loginButton: UIButton = {
        let btn = UIButton(type: .system)
        btn.setTitle("登录京东", for: .normal)
        btn.backgroundColor = .systemOrange
        btn.setTitleColor(.white, for: .normal)
        btn.layer.cornerRadius = 24
        btn.titleLabel?.font = UIFont.systemFont(ofSize: 17, weight: .medium)
        btn.layer.shadowColor = UIColor.systemOrange.cgColor
        btn.layer.shadowOpacity = 0.25
        btn.layer.shadowOffset = CGSize(width: 0, height: 4)
        btn.layer.shadowRadius = 12
        btn.layer.masksToBounds = false
        return btn
    }()
    
    // 查询按钮
    let queryButton: UIButton = {
        let btn = UIButton(type: .system)
        btn.setTitle("查询数据", for: .normal)
        btn.backgroundColor = .systemBlue
        btn.setTitleColor(.white, for: .normal)
        btn.layer.cornerRadius = 24
        btn.titleLabel?.font = UIFont.systemFont(ofSize: 17, weight: .medium)
        btn.layer.shadowColor = UIColor.systemBlue.cgColor
        btn.layer.shadowOpacity = 0.25
        btn.layer.shadowOffset = CGSize(width: 0, height: 4)
        btn.layer.shadowRadius = 12
        btn.layer.masksToBounds = false
        return btn
    }()
    
    // 更多羊毛按钮
    let moreWoolButton: UIButton = {
        let btn = UIButton(type: .system)
        btn.setTitle("更多羊毛", for: .normal)
        btn.backgroundColor = .systemPurple
        btn.setTitleColor(.white, for: .normal)
        btn.layer.cornerRadius = 24
        btn.titleLabel?.font = UIFont.systemFont(ofSize: 17, weight: .medium)
        btn.layer.shadowColor = UIColor.systemPurple.cgColor
        btn.layer.shadowOpacity = 0.25
        btn.layer.shadowOffset = CGSize(width: 0, height: 4)
        btn.layer.shadowRadius = 12
        btn.layer.masksToBounds = false
        return btn
    }()
    
    // 加入交流群按钮
    let joinGroupButton: UIButton = {
        let btn = UIButton(type: .system)
        btn.setTitle("加入交流群", for: .normal)
        btn.backgroundColor = .systemGreen
        btn.setTitleColor(.white, for: .normal)
        btn.layer.cornerRadius = 24
        btn.titleLabel?.font = UIFont.systemFont(ofSize: 17, weight: .medium)
        btn.layer.shadowColor = UIColor.systemGreen.cgColor
        btn.layer.shadowOpacity = 0.25
        btn.layer.shadowOffset = CGSize(width: 0, height: 4)
        btn.layer.shadowRadius = 12
        btn.layer.masksToBounds = false
        return btn
    }()
    
    // 加载视图
    private var loadingView: UIView?
    
    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemBackground
        setupUI()
        addNoticeTapGesture()
        bindButtonActions()
        
        let tapGesture = UITapGestureRecognizer(target: self, action: #selector(dismissKeyboard))
        tapGesture.delegate = self
        view.addGestureRecognizer(tapGesture)
        
        // 恢复存储的QQ号和头像
        restoreSavedQQData()
        
        // 公告视图动画
        UIView.animate(withDuration: 0.5, delay: 0.1, usingSpringWithDamping: 0.7, initialSpringVelocity: 0.5, options: [], animations: {
            self.noticeView.transform = .identity
            self.noticeView.alpha = 1
        }, completion: { _ in
            self.fetchNotice()
        })
        
        // 按钮动画
        animateButtons()
        
        scrollView.delegate = self
        noticeView.layer.addSublayer(topGradientLayer)
        noticeView.layer.addSublayer(bottomGradientLayer)
    }
    
    // 控制导航栏显示/隐藏
    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
    }
    
    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        navigationController?.setNavigationBarHidden(false, animated: animated)
        hideLoadingView()
    }
    
    override func viewDidLayoutSubviews() {
        super.viewDidLayoutSubviews()
        updateGradientLayers()
        qqAvatarImageView.layer.cornerRadius = qqAvatarImageView.bounds.width / 2
        checkIfContentNeedsScrolling()
    }
    
    // MARK: - 数据持久化相关
    /// 恢复存储的QQ号和头像
    private func restoreSavedQQData() {
        let defaults = UserDefaults.standard
        
        // 恢复QQ号
        if let savedQQ = defaults.string(forKey: UserDefaultsKeys.savedQQNumber) {
            qqTextField.text = savedQQ
        }
        
        // 恢复头像
        if let savedAvatarData = defaults.data(forKey: UserDefaultsKeys.savedQQAvatarData),
           let savedAvatar = UIImage(data: savedAvatarData) {
            qqAvatarImageView.image = savedAvatar
            qqAvatarImageView.isHidden = false
            qqAvatarImageView.alpha = 1.0
        }
    }
    
    /// 保存QQ号到本地
    private func saveQQNumber(_ qqNumber: String) {
        UserDefaults.standard.set(qqNumber, forKey: UserDefaultsKeys.savedQQNumber)
        UserDefaults.standard.synchronize() // 确保立即写入
    }
    
    /// 保存头像到本地
    private func saveQQAvatar(_ image: UIImage) {
        if let imageData = image.pngData() {
            UserDefaults.standard.set(imageData, forKey: UserDefaultsKeys.savedQQAvatarData)
            UserDefaults.standard.synchronize() // 确保立即写入
        }
    }
    
    // MARK: - UI Setup
    private func setupUI() {
        view.addSubview(appTitleLabel)
        view.addSubview(qqAvatarImageView)
        view.addSubview(noticeView)
        noticeView.addSubview(noticeHeaderView)
        noticeHeaderView.addSubview(noticeTitleLabel)
        noticeView.addSubview(scrollView)
        scrollView.addSubview(contentView)
        contentView.addSubview(noticeContentLabel)
        noticeView.addSubview(loadingIndicator)
        noticeView.addSubview(moreIndicatorView)
        noticeView.addSubview(scrollHintLabel)
        
        view.addSubview(qqTextField)
        view.addSubview(loginButton)
        view.addSubview(queryButton)
        view.addSubview(moreWoolButton)
        view.addSubview(joinGroupButton)
        
        setupConstraints()
    }
    
    private func setupConstraints() {
        [appTitleLabel, qqAvatarImageView, noticeView, noticeHeaderView, noticeTitleLabel, scrollView, contentView,
         noticeContentLabel, loadingIndicator, moreIndicatorView, scrollHintLabel,
         qqTextField, loginButton, queryButton, moreWoolButton, joinGroupButton].forEach {
            $0.translatesAutoresizingMaskIntoConstraints = false
        }
        
        NSLayoutConstraint.activate([
            // App标题
            appTitleLabel.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: LayoutConstants.appTitleTopMargin),
            appTitleLabel.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: LayoutConstants.noticeHorizontalMargin),
            appTitleLabel.trailingAnchor.constraint(equalTo: qqAvatarImageView.leadingAnchor, constant: LayoutConstants.qqAvatarToTitleSpacing),
            appTitleLabel.heightAnchor.constraint(equalToConstant: LayoutConstants.appTitleHeight),
            
            // QQ头像
            qqAvatarImageView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: LayoutConstants.qqAvatarTopMargin),
            qqAvatarImageView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: LayoutConstants.qqAvatarTrailingMargin),
            qqAvatarImageView.widthAnchor.constraint(equalToConstant: LayoutConstants.qqAvatarSize),
            qqAvatarImageView.heightAnchor.constraint(equalToConstant: LayoutConstants.qqAvatarSize),
            
            // 公告视图
            noticeView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: LayoutConstants.noticeTopMargin),
            noticeView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: LayoutConstants.noticeHorizontalMargin),
            noticeView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -LayoutConstants.noticeHorizontalMargin),
            noticeView.heightAnchor.constraint(equalTo: view.heightAnchor, multiplier: LayoutConstants.noticeHeightRatio),
            
            // 公告标题栏
            noticeHeaderView.topAnchor.constraint(equalTo: noticeView.topAnchor, constant: LayoutConstants.noticeHeaderTopMargin),
            noticeHeaderView.leadingAnchor.constraint(equalTo: noticeView.leadingAnchor, constant: LayoutConstants.noticeHeaderHorizontalMargin),
            noticeHeaderView.trailingAnchor.constraint(equalTo: noticeView.trailingAnchor, constant: -LayoutConstants.noticeHeaderHorizontalMargin),
            noticeHeaderView.heightAnchor.constraint(equalToConstant: LayoutConstants.noticeHeaderHeight),
            
            noticeTitleLabel.centerXAnchor.constraint(equalTo: noticeHeaderView.centerXAnchor),
            noticeTitleLabel.centerYAnchor.constraint(equalTo: noticeHeaderView.centerYAnchor),
            
            // 滚动区域
            scrollView.topAnchor.constraint(equalTo: noticeHeaderView.bottomAnchor, constant: LayoutConstants.scrollViewTopToHeaderSpacing),
            scrollView.leadingAnchor.constraint(equalTo: noticeView.leadingAnchor, constant: LayoutConstants.scrollViewHorizontalMargin),
            scrollView.trailingAnchor.constraint(equalTo: noticeView.trailingAnchor, constant: -LayoutConstants.scrollViewHorizontalMargin),
            scrollView.bottomAnchor.constraint(equalTo: noticeView.bottomAnchor, constant: LayoutConstants.scrollViewBottomMargin),
            
            contentView.topAnchor.constraint(equalTo: scrollView.topAnchor),
            contentView.leadingAnchor.constraint(equalTo: scrollView.leadingAnchor),
            contentView.trailingAnchor.constraint(equalTo: scrollView.trailingAnchor),
            contentView.bottomAnchor.constraint(equalTo: scrollView.bottomAnchor),
            contentView.widthAnchor.constraint(equalTo: scrollView.widthAnchor),
            
            noticeContentLabel.topAnchor.constraint(equalTo: contentView.topAnchor, constant: LayoutConstants.noticeContentInsets.top),
            noticeContentLabel.leadingAnchor.constraint(equalTo: contentView.leadingAnchor, constant: LayoutConstants.noticeContentInsets.left),
            noticeContentLabel.trailingAnchor.constraint(equalTo: contentView.trailingAnchor, constant: -LayoutConstants.noticeContentInsets.right),
            noticeContentLabel.bottomAnchor.constraint(equalTo: contentView.bottomAnchor, constant: -LayoutConstants.noticeContentInsets.bottom),
            
            // 指示器
            loadingIndicator.centerXAnchor.constraint(equalTo: noticeView.centerXAnchor),
            loadingIndicator.centerYAnchor.constraint(equalTo: noticeView.centerYAnchor),
            
            moreIndicatorView.trailingAnchor.constraint(equalTo: noticeView.trailingAnchor, constant: LayoutConstants.moreIndicatorTrailingMargin),
            moreIndicatorView.bottomAnchor.constraint(equalTo: noticeView.bottomAnchor, constant: LayoutConstants.moreIndicatorBottomMargin),
            
            scrollHintLabel.trailingAnchor.constraint(equalTo: moreIndicatorView.leadingAnchor, constant: LayoutConstants.scrollHintToIndicatorSpacing),
            scrollHintLabel.centerYAnchor.constraint(equalTo: moreIndicatorView.centerYAnchor),
            
            // 输入框和按钮
            qqTextField.topAnchor.constraint(equalTo: noticeView.bottomAnchor, constant: LayoutConstants.qqTextFieldTopToNoticeSpacing),
            qqTextField.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: LayoutConstants.qqTextFieldHorizontalMargin),
            qqTextField.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -LayoutConstants.qqTextFieldHorizontalMargin),
            qqTextField.heightAnchor.constraint(equalToConstant: LayoutConstants.qqTextFieldHeight),
            
            loginButton.topAnchor.constraint(equalTo: qqTextField.bottomAnchor, constant: LayoutConstants.loginButtonTopToTextFieldSpacing),
            loginButton.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: LayoutConstants.buttonHorizontalMargin),
            loginButton.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -LayoutConstants.buttonHorizontalMargin),
            loginButton.heightAnchor.constraint(equalToConstant: LayoutConstants.buttonHeight),
            
            queryButton.topAnchor.constraint(equalTo: loginButton.bottomAnchor, constant: LayoutConstants.queryButtonTopToLoginButtonSpacing),
            queryButton.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: LayoutConstants.buttonHorizontalMargin),
            queryButton.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -LayoutConstants.buttonHorizontalMargin),
            queryButton.heightAnchor.constraint(equalToConstant: LayoutConstants.buttonHeight),
            
            moreWoolButton.topAnchor.constraint(equalTo: queryButton.bottomAnchor, constant: LayoutConstants.moreWoolButtonTopToQueryButtonSpacing),
            moreWoolButton.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: LayoutConstants.buttonHorizontalMargin),
            moreWoolButton.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -LayoutConstants.buttonHorizontalMargin),
            moreWoolButton.heightAnchor.constraint(equalToConstant: LayoutConstants.buttonHeight),
            
            joinGroupButton.topAnchor.constraint(equalTo: moreWoolButton.bottomAnchor, constant: LayoutConstants.joinGroupButtonTopToMoreWoolButtonSpacing),
            joinGroupButton.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: LayoutConstants.buttonHorizontalMargin),
            joinGroupButton.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -LayoutConstants.buttonHorizontalMargin),
            joinGroupButton.heightAnchor.constraint(equalToConstant: LayoutConstants.buttonHeight)
        ])
    }
    
    // MARK: - Button Actions
    private func bindButtonActions() {
        loginButton.addTarget(self, action: #selector(loginAction), for: .touchUpInside)
        queryButton.addTarget(self, action: #selector(queryAction), for: .touchUpInside)
        moreWoolButton.addTarget(self, action: #selector(moreWoolAction), for: .touchUpInside)
        joinGroupButton.addTarget(self, action: #selector(joinGroupAction), for: .touchUpInside)
        
        // 添加按钮点击效果
        [loginButton, queryButton, moreWoolButton, joinGroupButton].forEach { button in
            button.addTarget(self, action: #selector(buttonTouchDown(_:)), for: .touchDown)
            button.addTarget(self, action: #selector(buttonTouchUp(_:)), for: [.touchUpInside, .touchUpOutside, .touchCancel])
        }
    }
    
    @objc private func buttonTouchDown(_ sender: UIButton) {
        UIView.animate(withDuration: 0.15) {
            sender.transform = CGAffineTransform(scaleX: 0.96, y: 0.96)
            sender.layer.shadowOpacity = 0.15
        }
    }
    
    @objc private func buttonTouchUp(_ sender: UIButton) {
        UIView.animate(withDuration: 0.15) {
            sender.transform = .identity
            sender.layer.shadowOpacity = 0.25
        }
    }
    
    @objc func loginAction() {
        guard let qqNumber = qqTextField.text, !qqNumber.isEmpty else {
            showAlert(message: "请输入QQ号")
            return
        }
        
        // 保存QQ号到本地
        saveQQNumber(qqNumber)
        
        // 加载并保存头像
        loadQQAvatar(qqNumber: qqNumber)
        
        // 原有登录逻辑
        LoginManager.shared.handleLogin(from: self, qqTextField: qqTextField)
    }
    
    @objc func queryAction() {
        QueryManager.shared.handleQuery(from: self)
    }
    
    @objc private func moreWoolAction() {
        let woolVC = WoolViewController()
        navigationController?.pushViewController(woolVC, animated: true)
    }
    
    @objc private func joinGroupAction() {
        let groupURLString = "https://qm.qq.com/q/4gYwV6YzPW"
        guard let url = URL(string: groupURLString) else {
            showAlert(message: "交流群链接无效")
            return
        }
        
        if UIApplication.shared.canOpenURL(url) {
            UIApplication.shared.open(url, options: [:])
        } else {
            showAlert(message: "无法打开交流群链接")
        }
    }
    
    // MARK: - Notice Management
    private func addNoticeTapGesture() {
        let tapGesture = UITapGestureRecognizer(target: self, action: #selector(noticeTapped))
        noticeView.addGestureRecognizer(tapGesture)
    }
    
    @objc private func noticeTapped() {
        loadingIndicator.startAnimating()
        noticeContentLabel.isHidden = true
        
        UIView.animate(withDuration: 0.3) {
            self.noticeView.transform = CGAffineTransform(scaleX: 0.98, y: 0.98)
        } completion: { _ in
            UIView.animate(withDuration: 0.3) {
                self.noticeView.transform = .identity
            }
        }
        
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            self.fetchNotice()
        }
    }
    
    private func fetchNotice() {
        guard let url = URL(string: "https://gitee.com/feiniao520/notice/raw/master/notice") else {
            updateNoticeContent(text: "公告链接无效")
            return
        }
        
        let task = URLSession.shared.dataTask(with: url) { [weak self] data, response, error in
            DispatchQueue.main.async {
                guard let self = self else { return }
                
                self.loadingIndicator.stopAnimating()
                self.noticeContentLabel.isHidden = false
                
                if let error = error {
                    self.updateNoticeContent(text: "公告加载失败：\(error.localizedDescription)")
                    return
                }
                
                guard let data = data, let content = String(data: data, encoding: .utf8) else {
                    self.updateNoticeContent(text: "公告内容为空")
                    return
                }
                
                self.updateNoticeContentWithAnimation(text: content)
            }
        }
        task.resume()
    }
    
    private func updateNoticeContentWithAnimation(text: String) {
        noticeContentLabel.alpha = 0
        updateNoticeContent(text: text)
        
        UIView.animate(withDuration: 0.3) {
            self.noticeContentLabel.alpha = 1
        } completion: { _ in
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) {
                self.checkIfContentNeedsScrolling()
            }
        }
    }
    
    private func updateNoticeContent(text: String) {
        let paragraphStyle = NSMutableParagraphStyle()
        paragraphStyle.lineSpacing = 8
        paragraphStyle.firstLineHeadIndent = 0
        let attributedText = NSAttributedString(string: text, attributes: [
            .font: UIFont.systemFont(ofSize: 13),
            .foregroundColor: UIColor.secondaryLabel,
            .paragraphStyle: paragraphStyle
        ])
        noticeContentLabel.attributedText = attributedText
    }
    
    // MARK: - Scroll View Management
    private func checkIfContentNeedsScrolling() {
        let contentSize = scrollView.contentSize.height
        let scrollViewHeight = scrollView.bounds.height
        let contentOffsetY = scrollView.contentOffset.y
        
        let contentExceedsScrollView = contentSize > scrollViewHeight + 1
        let isScrolledToBottom = contentOffsetY >= (contentSize - scrollViewHeight) - 10
        
        if contentExceedsScrollView && !isScrolledToBottom {
            showScrollIndicators()
        } else {
            hideScrollIndicators()
        }
    }
    
    private func showScrollIndicators() {
        UIView.animate(withDuration: 0.5) {
            self.topGradientLayer.opacity = 0.7
            self.bottomGradientLayer.opacity = 0.7
            self.scrollHintLabel.alpha = 0.7
            self.moreIndicatorView.alpha = 0.7
        }
        startArrowBreathingAnimation()
    }
    
    private func hideScrollIndicators() {
        UIView.animate(withDuration: 0.3) {
            self.topGradientLayer.opacity = 0.0
            self.bottomGradientLayer.opacity = 0.0
            self.moreIndicatorView.alpha = 0.0
            self.scrollHintLabel.alpha = 0.0
        }
        moreIndicatorView.layer.removeAllAnimations()
    }
    
    private func startArrowBreathingAnimation() {
        moreIndicatorView.alpha = 0.7
        
        let animation = CABasicAnimation(keyPath: "opacity")
        animation.fromValue = 0.3
        animation.toValue = 0.8
        animation.duration = 1.5
        animation.repeatCount = .infinity
        animation.autoreverses = true
        moreIndicatorView.layer.add(animation, forKey: "breathingAnimation")
    }
    
    private func updateGradientLayers() {
        topGradientLayer.frame = CGRect(x: 0, y: noticeHeaderView.frame.maxY, width: noticeView.bounds.width, height: LayoutConstants.gradientLayerHeight)
        bottomGradientLayer.frame = CGRect(x: 0, y: noticeView.bounds.height - LayoutConstants.gradientLayerHeight, width: noticeView.bounds.width, height: LayoutConstants.gradientLayerHeight)
    }
    
    // MARK: - QQ Avatar (优化：加载并保存头像)
    private func loadQQAvatar(qqNumber: String) {
        // 先检查本地是否有缓存头像，有则直接使用
        let defaults = UserDefaults.standard
        if let savedAvatarData = defaults.data(forKey: UserDefaultsKeys.savedQQAvatarData),
           let savedAvatar = UIImage(data: savedAvatarData) {
            qqAvatarImageView.image = savedAvatar
            qqAvatarImageView.isHidden = false
            qqAvatarImageView.alpha = 1.0
            return
        }
        
        // 本地无缓存，从网络加载
        let avatarURLString = "http://q1.qlogo.cn/g?b=qq&nk=\(qqNumber)&s=100"
        guard let url = URL(string: avatarURLString) else { return }
        
        URLSession.shared.dataTask(with: url) { [weak self] data, response, error in
            DispatchQueue.main.async {
                guard let self = self, let data = data, let image = UIImage(data: data) else {
                    // 加载失败时显示默认头像（可选）
                    self?.qqAvatarImageView.image = UIImage(systemName: "person.circle.fill")
                    self?.qqAvatarImageView.tintColor = .secondaryLabel
                    self?.qqAvatarImageView.isHidden = false
                    self?.qqAvatarImageView.alpha = 1.0
                    return
                }
                
                // 保存头像到本地
                self.saveQQAvatar(image)
                
                // 显示头像
                self.qqAvatarImageView.alpha = 0
                self.qqAvatarImageView.image = image
                self.qqAvatarImageView.isHidden = false
                
                UIView.animate(withDuration: 0.3) {
                    self.qqAvatarImageView.alpha = 1
                }
            }
        }.resume()
    }
    
    // MARK: - Animations
    private func animateButtons() {
        moreWoolButton.transform = CGAffineTransform(scaleX: 0.95, y: 0.95)
        moreWoolButton.alpha = 0
        joinGroupButton.transform = CGAffineTransform(scaleX: 0.95, y: 0.95)
        joinGroupButton.alpha = 0
        
        UIView.animate(withDuration: 0.5, delay: 0.3, usingSpringWithDamping: 0.7, initialSpringVelocity: 0.5, options: []) {
            self.moreWoolButton.transform = .identity
            self.moreWoolButton.alpha = 1
        }
        
        UIView.animate(withDuration: 0.5, delay: 0.4, usingSpringWithDamping: 0.7, initialSpringVelocity: 0.5, options: []) {
            self.joinGroupButton.transform = .identity
            self.joinGroupButton.alpha = 1
        }
    }
    
    // MARK: - Helper Methods
    @objc private func dismissKeyboard() {
        view.endEditing(true)
    }
    
    func showAlert(message: String) {
        DispatchQueue.main.async {
            let alert = UIAlertController(title: "提示", message: message, preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "确定", style: .default))
            self.present(alert, animated: true)
        }
    }
    
    func showLoadingView() {
        guard loadingView == nil else { return }
        
        let loadingView = UIView(frame: view.bounds)
        loadingView.backgroundColor = UIColor.black.withAlphaComponent(0.6)
        view.addSubview(loadingView)
        
        let activityIndicator = UIActivityIndicatorView(style: .whiteLarge)
        activityIndicator.translatesAutoresizingMaskIntoConstraints = false
        loadingView.addSubview(activityIndicator)
        
        let label = UILabel()
        label.text = "正在查询中..."
        label.textColor = .white
        label.font = UIFont.systemFont(ofSize: 18, weight: .medium)
        label.translatesAutoresizingMaskIntoConstraints = false
        loadingView.addSubview(label)
        
        NSLayoutConstraint.activate([
            activityIndicator.centerXAnchor.constraint(equalTo: loadingView.centerXAnchor),
            activityIndicator.centerYAnchor.constraint(equalTo: loadingView.centerYAnchor),
            label.topAnchor.constraint(equalTo: activityIndicator.bottomAnchor, constant: 20),
            label.centerXAnchor.constraint(equalTo: loadingView.centerXAnchor)
        ])
        
        self.loadingView = loadingView
        view.isUserInteractionEnabled = false
        
        loadingView.alpha = 0
        UIView.animate(withDuration: 0.3) {
            loadingView.alpha = 1
        } completion: { _ in
            activityIndicator.startAnimating()
        }
    }
    
    func hideLoadingView() {
        DispatchQueue.main.async {
            guard let loadingView = self.loadingView else { return }
            
            UIView.animate(withDuration: 0.3, animations: {
                loadingView.alpha = 0
            }) { _ in
                loadingView.removeFromSuperview()
                self.loadingView = nil
                self.view.isUserInteractionEnabled = true
            }
        }
    }
    
    // MARK: - UIScrollViewDelegate
    func scrollViewWillBeginDragging(_ scrollView: UIScrollView) {
        moreIndicatorView.layer.removeAllAnimations()
        moreIndicatorView.alpha = 0.0
        scrollHintLabel.alpha = 0.0
    }
    
    func scrollViewDidScroll(_ scrollView: UIScrollView) {
        let contentSize = scrollView.contentSize.height
        let scrollViewHeight = scrollView.bounds.height
        let contentOffsetY = scrollView.contentOffset.y
        
        if contentSize > scrollViewHeight {
            let topAlpha = min(1.0, max(0.0, 1.0 - (contentOffsetY / 20)))
            topGradientLayer.opacity = Float(topAlpha)
            
            let bottomAlpha = min(1.0, max(0.0, 1.0 - ((contentSize - contentOffsetY - scrollViewHeight) / 20)))
            bottomGradientLayer.opacity = Float(bottomAlpha)
        }
        
        checkIfContentNeedsScrolling()
    }
    
    func scrollViewDidEndDecelerating(_ scrollView: UIScrollView) {
        checkIfContentNeedsScrolling()
    }
    
    func scrollViewDidEndDragging(_ scrollView: UIScrollView, willDecelerate decelerate: Bool) {
        if !decelerate {
            checkIfContentNeedsScrolling()
        }
    }

    // MARK: - UIGestureRecognizerDelegate
    func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldReceive touch: UITouch) -> Bool {
        return !(touch.view?.isDescendant(of: scrollView) ?? false)
    }
}

// 输入框内边距扩展
extension UITextField {
    func paddingLeft(_ padding: CGFloat) {
        let paddingView = UIView(frame: CGRect(x: 0, y: 0, width: padding, height: frame.height))
        leftView = paddingView
        leftViewMode = .always
    }
    
    func paddingRight(_ padding: CGFloat) {
        let paddingView = UIView(frame: CGRect(x: 0, y: 0, width: padding, height: frame.height))
        rightView = paddingView
        rightViewMode = .always
    }
}
