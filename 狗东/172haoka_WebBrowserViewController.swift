//
//  WebBrowserViewController.swift
//  YourAppName
//
//  Created by Your Name on 2023/10/27.
//

import UIKit
import WebKit

@available(iOS 13.0, *)
class WebBrowserViewController: UIViewController {

    private let webView: WKWebView
    private let url: URL
    private let progressView = UIProgressView(progressViewStyle: .default)
    
    // MARK: - Initialization
    
    init(url: URL) {
        self.url = url
        let configuration = WKWebViewConfiguration()
        self.webView = WKWebView(frame: .zero, configuration: configuration)
        super.init(nibName: nil, bundle: nil)
    }
    
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }
    
    // MARK: - Lifecycle Methods
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI()
        setupNavigationBar()
        loadWebPage()
        setupProgressObservation()
    }
    
    // MARK: - Setup
    
    private func setupUI() {
        view.backgroundColor = .systemBackground
        
        progressView.translatesAutoresizingMaskIntoConstraints = false
        progressView.tintColor = .systemBlue
        view.addSubview(progressView)
        
        webView.translatesAutoresizingMaskIntoConstraints = false
        webView.navigationDelegate = self // 这里会产生强引用
        webView.allowsBackForwardNavigationGestures = true
        view.addSubview(webView)
        
        NSLayoutConstraint.activate([
            progressView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            progressView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            progressView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            
            webView.topAnchor.constraint(equalTo: progressView.bottomAnchor),
            webView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            webView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor)
        ])
    }
    
    private func setupNavigationBar() {
        let backButton = UIBarButtonItem(
            image: UIImage(systemName: "arrow.left"),
            style: .plain,
            target: self,
            action: #selector(backButtonTapped)
        )
        
        let forwardButton = UIBarButtonItem(
            image: UIImage(systemName: "arrow.right"),
            style: .plain,
            target: self,
            action: #selector(forwardButtonTapped)
        )
        
        let refreshButton = UIBarButtonItem(
            barButtonSystemItem: .refresh,
            target: self,
            action: #selector(refreshButtonTapped)
        )
        
        let closeButton = UIBarButtonItem(
            barButtonSystemItem: .close,
            target: self,
            action: #selector(closeButtonTapped)
        )
        
        navigationItem.leftBarButtonItems = [backButton, forwardButton]
        navigationItem.rightBarButtonItems = [refreshButton, closeButton]
        
        backButton.isEnabled = false
        forwardButton.isEnabled = false
    }
    
    private func setupProgressObservation() {
        webView.addObserver(self, forKeyPath: #keyPath(WKWebView.estimatedProgress), options: .new, context: nil)
        webView.addObserver(self, forKeyPath: #keyPath(WKWebView.canGoBack), options: .new, context: nil)
        webView.addObserver(self, forKeyPath: #keyPath(WKWebView.canGoForward), options: .new, context: nil)
    }
    
    // MARK: - Deinitialization (修复的核心)
    
    deinit {
        print("WebBrowserViewController deinit called.") // 可以在控制台看到是否被销毁
        
        // 1. 移除 navigationDelegate，打破循环引用
        webView.navigationDelegate = nil
        
        // 2. 移除所有 KVO 监听
        webView.removeObserver(self, forKeyPath: #keyPath(WKWebView.estimatedProgress))
        webView.removeObserver(self, forKeyPath: #keyPath(WKWebView.canGoBack))
        webView.removeObserver(self, forKeyPath: #keyPath(WKWebView.canGoForward))
    }
    
    // MARK: - KVO Observer
    
    override func observeValue(forKeyPath keyPath: String?, of object: Any?, change: [NSKeyValueChangeKey : Any]?, context: UnsafeMutableRawPointer?) {
        if keyPath == #keyPath(WKWebView.estimatedProgress) {
            progressView.progress = Float(webView.estimatedProgress)
            if webView.estimatedProgress >= 1.0 {
                UIView.animate(withDuration: 0.3, animations: {
                    self.progressView.alpha = 0
                }, completion: { _ in
                    self.progressView.progress = 0
                    self.progressView.alpha = 1
                })
            }
        } else if keyPath == #keyPath(WKWebView.canGoBack) {
            navigationItem.leftBarButtonItems?[0].isEnabled = webView.canGoBack
        } else if keyPath == #keyPath(WKWebView.canGoForward) {
            navigationItem.leftBarButtonItems?[1].isEnabled = webView.canGoForward
        } else {
            super.observeValue(forKeyPath: keyPath, of: object, change: change, context: context)
        }
    }
    
    // MARK: - Loading Content
    
    private func loadWebPage() {
        let request = URLRequest(url: url)
        webView.load(request)
    }
    
    // MARK: - Actions
    
    @objc private func backButtonTapped() {
        if webView.canGoBack {
            webView.goBack()
        }
    }
    
    @objc private func forwardButtonTapped() {
        if webView.canGoForward {
            webView.goForward()
        }
    }
    
    @objc private func refreshButtonTapped() {
        if webView.isLoading {
            webView.stopLoading()
        } else {
            webView.reload()
        }
    }
    
    @objc private func closeButtonTapped() {
        // 关闭页面，不会影响其他页面
        navigationController?.popViewController(animated: true)
    }
}

// MARK: - WKNavigationDelegate

@available(iOS 13.0, *)
extension WebBrowserViewController: WKNavigationDelegate {
    
    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        title = webView.title
    }
    
    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        let alert = UIAlertController(
            title: "加载失败",
            message: error.localizedDescription,
            preferredStyle: .alert
        )
        alert.addAction(UIAlertAction(title: "重试", style: .default, handler: { _ in
            self.webView.reload()
        }))
        alert.addAction(UIAlertAction(title: "取消", style: .cancel))
        present(alert, animated: true)
    }
}
