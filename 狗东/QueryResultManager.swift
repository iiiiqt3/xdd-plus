import UIKit

// 查询管理单例类（优化并发请求+超时处理）
class QueryManager {
    static let shared = QueryManager()
    private init() {}
    
    // 处理查询逻辑
    @available(iOS 13.0, *)
    func handleQuery(from viewController: MainUIViewController) {
        DispatchQueue.main.async { [weak viewController] in
            guard let viewController = viewController else { return }
            
            // 检查登录状态
            guard let qq = UserDefaults.standard.string(forKey: "qq"), !qq.isEmpty else {
                // 直接在这里实现弹窗逻辑，避免使用扩展方法
                self.showAlert(in: viewController, message: "请先登录")
                return
            }
            
            viewController.showLoadingView()
            
            // 异步获取PIN列表
            DispatchQueue.global().async {
                self.getUserPins(qq: qq) { [weak viewController] pins in
                    guard let viewController = viewController else { return }
                    
                    if let pins = pins, !pins.isEmpty {
                        // 并发查询所有PIN的用户信息
                        self.queryAllUserInfoConcurrently(pins: pins) { resultText in
                            DispatchQueue.main.async {
                                viewController.hideLoadingView()
                                self.showResultViewController(from: viewController, resultText: resultText)
                            }
                        }
                    } else {
                        DispatchQueue.main.async {
                            viewController.hideLoadingView()
                            // 直接在这里实现弹窗逻辑，避免使用扩展方法
                            self.showAlert(in: viewController, message: "未获取到有效PIN列表")
                        }
                    }
                }
            }
        }
    }
    
    // 添加一个内部方法来显示弹窗，避免使用UIViewController扩展
    private func showAlert(in viewController: UIViewController, message: String) {
        DispatchQueue.main.async {
            let alert = UIAlertController(title: "提示", message: message, preferredStyle: .alert)
            alert.addAction(UIAlertAction(title: "确定", style: .default))
            viewController.present(alert, animated: true)
        }
    }
    
    
    // 第一步：获取用户PIN列表
    private func getUserPins(qq: String, completion: @escaping ([String]?) -> Void) {
        // 方法实现不变...
        guard let encodedQQ = qq.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) else {
            print("QQ号编码失败: \(qq)")
            completion(nil)
            return
        }
        
        let urlString = "http://180.152.5.230:5701/api/getUserPin?QQ=\(encodedQQ)"
        print("请求PIN列表: \(urlString)")
        
        guard let url = URL(string: urlString) else {
            print("无效URL: \(urlString)")
            completion(nil)
            return
        }
        
        let task = URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error {
                print("获取PIN列表失败: \(error.localizedDescription)")
                completion(nil)
                return
            }
            
            guard let data = data else {
                print("PIN列表接口无返回数据")
                completion(nil)
                return
            }
            
            do {
                guard let json = try JSONSerialization.jsonObject(with: data, options: []) as? [String: Any] else {
                    let rawString = String(data: data, encoding: .utf8) ?? "无法解析"
                    print("PIN列表JSON格式错误: \(rawString)")
                    completion(nil)
                    return
                }
                
                // 检查接口状态码
                guard let code = json["code"] as? Int, code == 0 else {
                    let errorMsg = json["message"] as? String ?? "未知错误"
                    print("PIN列表接口错误: code=\(json["code"] ?? "N/A"), msg=\(errorMsg)")
                    completion(nil)
                    return
                }
                
                // 提取PIN数组
                guard let pinArray = json["data"] as? [String] else {
                    print("PIN列表格式错误: 预期[String]类型")
                    completion(nil)
                    return
                }
                
                completion(pinArray)
                
            } catch {
                let rawString = String(data: data, encoding: .utf8) ?? "解析失败"
                print("PIN列表解析异常: \(error.localizedDescription), 原始数据: \(rawString)")
                completion(nil)
            }
        }
        task.resume()
    }
    
    // 第二步：并发查询所有PIN的用户信息（核心优化）
    private func queryAllUserInfoConcurrently(pins: [String], completion: @escaping (String) -> Void) {
        var results = [String: String]() // 存储结果（PIN为key）
        let group = DispatchGroup()      // 控制并发任务完成
        
        for pin in pins {
            group.enter()    // 进入任务组
            
            // 异步查询单个PIN信息
            DispatchQueue.global().async {
                let task = self.getUserInfo(pin: pin) { info in
                    results[pin] = info
                    group.leave()    // 离开任务组
                }
                
                // 超时处理（60秒）
                DispatchQueue.main.asyncAfter(deadline: .now() + 60) {
                    if task.state == .running {
                        task.cancel()
                        results[pin] = "查询超时（60秒）"
                        group.leave()
                    }
                }
            }
        }
        
        // 所有任务完成后拼接结果
        group.notify(queue: .global()) {
            var resultText = ""
            // 按原始PIN顺序排列结果
            for (index, pin) in pins.enumerated() {
                let info = results[pin] ?? "查询失败"
                resultText += "【\(index + 1)/\(pins.count)】PIN: \(pin)\n\(info)\n\n"
            }
            completion(resultText)
        }
    }
    
    // 第三步：查询单个PIN的用户信息
    private func getUserInfo(pin: String, completion: @escaping (String) -> Void) -> URLSessionDataTask {
        // 方法实现不变...
        guard let encodedPin = pin.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) else {
            completion("无效PIN参数")
            let task = URLSession.shared.dataTask(with: URL(string: "about:blank")!)
            task.cancel()
            return task
        }
        
        let urlString = "http://180.152.5.230:5701/api/getUserInfo?pin=\(encodedPin)"
        guard let url = URL(string: urlString) else {
            completion("无效请求URL")
            let task = URLSession.shared.dataTask(with: URL(string: "about:blank")!)
            task.cancel()
            return task
        }
        
        let task = URLSession.shared.dataTask(with: url) { data, response, error in
            if let error = error as NSError? {
                completion(error.code == NSURLErrorCancelled ? "查询被取消" : "查询失败: \(error.localizedDescription)")
                return
            }
            
            guard let data = data else {
                completion("无返回数据")
                return
            }
            
            do {
                if let json = try JSONSerialization.jsonObject(with: data, options: []) as? [String: Any],
                   let code = json["code"] as? Int, code == 0 {
                    
                    // 处理两种返回格式：字典或字符串
                    if let dataDict = json["data"] as? [String: Any] {
                        completion(self.formatDictionary(dataDict))
                    } else if let dataStr = json["data"] as? String {
                        completion(dataStr)
                    } else {
                        let rawData = String(describing: json["data"])
                        completion("数据格式不支持: \(rawData)")
                    }
                } else {
                    // 接口返回错误信息
                    let rawResponse = String(data: data, encoding: .utf8) ?? "无法解析"
                    completion("接口返回错误: \(rawResponse)")
                }
            } catch {
                let rawData = String(data: data, encoding: .utf8) ?? "解析异常"
                completion("JSON解析失败: \(rawData)")
            }
        }
        task.resume()
        return task
    }
    
    // 格式化字典为易读字符串
    private func formatDictionary(_ dict: [String: Any]) -> String {
        // 方法实现不变...
        var result = ""
        // 按key排序（保证输出顺序一致）
        let sortedKeys = dict.keys.sorted()
        for key in sortedKeys {
            let value = dict[key] ?? "nil"
            result += "\(key): \(value)\n"
        }
        return result.isEmpty ? "无数据" : result.dropLast().description
    }
    
    // 展示结果页面
    @available(iOS 13.0, *)
    private func showResultViewController(from viewController: UIViewController, resultText: String) {
        let resultVC = ResultViewController()
        resultVC.resultText = resultText
        let navController = UINavigationController(rootViewController: resultVC)
        // 直接设置，删除多余的版本判断
        navController.modalPresentationStyle = .fullScreen
        viewController.present(navController, animated: true)
    }
    
    // 结果展示视图控制器（优化UI流畅性）
    @available(iOS 13.0, *)
    class ResultViewController: UIViewController {
        // 类实现不变...
        private let resultTextView: UITextView = {
            let tv = UITextView()
            tv.backgroundColor = .secondarySystemBackground
            tv.layer.cornerRadius = 16
            tv.layer.shadowColor = UIColor.black.cgColor
            tv.layer.shadowOpacity = 0.08
            tv.layer.shadowOffset = CGSize(width: 0, height: 4)
            tv.layer.shadowRadius = 12
            tv.isEditable = false
            tv.font = UIFont.systemFont(ofSize: 14)
            tv.textColor = .label
            tv.textContainerInset = UIEdgeInsets(top: 16, left: 16, bottom: 16, right: 16)
            
            let paragraphStyle = NSMutableParagraphStyle()
            paragraphStyle.lineSpacing = 4
            paragraphStyle.paragraphSpacing = 8
            tv.typingAttributes = [
                .font: UIFont.systemFont(ofSize: 14),
                .foregroundColor: UIColor.label,
                .paragraphStyle: paragraphStyle
            ]
            
            return tv
        }()
        
        private let loadingIndicator: UIActivityIndicatorView = {
            let indicator = UIActivityIndicatorView(style: .medium)
            indicator.hidesWhenStopped = true
            indicator.color = .systemBlue
            return indicator
        }()
        
        var resultText: String?
        private let primaryColor = UIColor.systemBlue
        
        override func viewDidLoad() {
            super.viewDidLoad()
            view.backgroundColor = .systemBackground
            
            title = "查询结果"
            navigationController?.navigationBar.prefersLargeTitles = false
            navigationItem.leftBarButtonItem = UIBarButtonItem(
                image: UIImage(systemName: "chevron.left"),
                style: .plain,
                target: self,
                action: #selector(dismissSelf)
            )
            navigationItem.leftBarButtonItem?.tintColor = primaryColor
            
            navigationItem.rightBarButtonItem = UIBarButtonItem(
                barButtonSystemItem: .done,
                target: self,
                action: #selector(dismissSelf)
            )
            navigationItem.rightBarButtonItem?.tintColor = primaryColor
            
            setupUI()
            loadResultText()
        }
        
        private func loadResultText() {
            loadingIndicator.startAnimating()
            
            DispatchQueue.global().async { [weak self] in
                guard let self = self else { return }
                let text = self.resultText ?? "暂无有效查询结果"
                
                DispatchQueue.main.async {
                    self.resultTextView.setContentOffset(.zero, animated: false)
                    self.resultTextView.text = text
                    self.loadingIndicator.stopAnimating()
                    self.animateShowTextView()
                }
            }
        }
        
        private func animateShowTextView() {
            let animator = UIViewPropertyAnimator(duration: 0.5, dampingRatio: 0.7) {
                self.resultTextView.alpha = 1
                self.resultTextView.transform = .identity
            }
            animator.startAnimation()
        }
        
        private func setupUI() {
            view.addSubview(resultTextView)
            view.addSubview(loadingIndicator)
            
            [resultTextView, loadingIndicator].forEach {
                $0.translatesAutoresizingMaskIntoConstraints = false
            }
            
            resultTextView.alpha = 0
            resultTextView.transform = CGAffineTransform(scaleX: 0.95, y: 0.95)
            
            NSLayoutConstraint.activate([
                resultTextView.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor, constant: 16),
                resultTextView.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
                resultTextView.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
                resultTextView.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor, constant: -16),
                
                loadingIndicator.centerXAnchor.constraint(equalTo: view.centerXAnchor),
                loadingIndicator.centerYAnchor.constraint(equalTo: view.centerYAnchor)
            ])
        }
        
        @objc private func dismissSelf() {
            dismiss(animated: true)
        }
        
        override func viewDidAppear(_ animated: Bool) {
            super.viewDidAppear(animated)
            resultTextView.flashScrollIndicators()
        }
    }
}

