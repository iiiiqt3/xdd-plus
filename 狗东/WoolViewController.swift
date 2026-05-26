//
//  WoolViewController.swift
//  YourAppName
//
//  Created by Your Name on 2023/10/27.
//

import UIKit

@available(iOS 13.0, *)
class WoolViewController: UIViewController {
    
    // --- 视觉风格统一化 ---
    private let primaryColor = UIColor.systemBlue
    private let secondaryColor = UIColor.systemOrange
    private let backgroundColor = UIColor.systemBackground
    private let cardBackgroundColor = UIColor.secondarySystemBackground
    private let textPrimaryColor = UIColor.label
    private let textSecondaryColor = UIColor.secondaryLabel
    
    // --- 布局常量 ---
    private struct LayoutConstants {
        static let buttonHeight: CGFloat = 52.0
        static let buttonHorizontalMargin: CGFloat = 32.0
        static let topButtonTopMargin: CGFloat = 40.0
    }
    
    // MARK: - UI Components
    
    private let phoneCardButton: UIButton = {
        let btn = UIButton(type: .system)
        btn.setTitle("大流量手机卡", for: .normal)
        btn.backgroundColor = .systemTeal
        btn.setTitleColor(.white, for: .normal)
        btn.layer.cornerRadius = 24
        btn.titleLabel?.font = UIFont.systemFont(ofSize: 17, weight: .medium)
        btn.layer.shadowColor = UIColor.systemTeal.cgColor
        btn.layer.shadowOpacity = 0.25
        btn.layer.shadowOffset = CGSize(width: 0, height: 4)
        btn.layer.shadowRadius = 12
        btn.layer.masksToBounds = false
        return btn
    }()
    
    // MARK: - Lifecycle Methods
    
    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = backgroundColor
        title = "更多羊毛" // 导航栏标题
        
        setupUI()
        bindButtonActions()
    }
    
    // MARK: - Setup
    
    private func setupUI() {
        view.addSubview(phoneCardButton)
        
        phoneCardButton.translatesAutoresizingMaskIntoConstraints = false
        
        NSLayoutConstraint.activate([
            phoneCardButton.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: LayoutConstants.topButtonTopMargin),
            phoneCardButton.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: LayoutConstants.buttonHorizontalMargin),
            phoneCardButton.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -LayoutConstants.buttonHorizontalMargin),
            phoneCardButton.heightAnchor.constraint(equalToConstant: LayoutConstants.buttonHeight)
        ])
    }
    
    // MARK: - Button Actions
    
    private func bindButtonActions() {
        phoneCardButton.addTarget(self, action: #selector(phoneCardButtonTapped), for: .touchUpInside)
        phoneCardButton.addTarget(self, action: #selector(buttonTouchDown(_:)), for: .touchDown)
        phoneCardButton.addTarget(self, action: #selector(buttonTouchUp(_:)), for: [.touchUpInside, .touchUpOutside, .touchCancel])
    }
    
    // 核心改动：使用 push 跳转
    @objc private func phoneCardButtonTapped() {
        guard let url = URL(string: "https://h5.lot-ml.com/ProductEn/Index/7ee6c54f2d550fab") else {
            showAlert(message: "链接无效")
            return
        }
        let webBrowserVC = WebBrowserViewController(url: url)
        navigationController?.pushViewController(webBrowserVC, animated: true)
    }
    
    // 按钮点击动画
    @objc private func buttonTouchDown(_ sender: UIButton) {
        UIView.animate(withDuration: 0.15, animations: {
            sender.transform = CGAffineTransform(scaleX: 0.96, y: 0.96)
            sender.layer.shadowOpacity = 0.15
        })
    }
    
    @objc private func buttonTouchUp(_ sender: UIButton) {
        UIView.animate(withDuration: 0.15, animations: {
            sender.transform = .identity
            sender.layer.shadowOpacity = 0.25
        })
    }
    
    // MARK: - Helper
    
    private func showAlert(message: String) {
        let alert = UIAlertController(title: "提示", message: message, preferredStyle: .alert)
        alert.addAction(UIAlertAction(title: "确定", style: .default))
        present(alert, animated: true)
    }
}
