import UIKit
import WebKit

/// QQ Scheme唤起测试页面 - 独立文件，直接运行即可测试
class QQSchemeTestViewController: UIViewController {
    
    // 测试按钮（代码创建，无需故事板/IB配置）
    private let testQQBtn = UIButton(type: .system)
    
    override func viewDidLoad() {
        super.viewDidLoad()
        setupUI() // 初始化页面UI
        autoTestQQScheme() // 页面加载后自动执行一次测试（可选）
    }
    
    // MARK: - 页面UI初始化（代码创建，无依赖）
    private func setupUI() {
        view.backgroundColor = .white
        title = "QQ Scheme唤起测试"
        
        // 配置测试按钮
        testQQBtn.setTitle("点击测试唤起QQ", for: .normal)
        testQQBtn.titleLabel?.font = UIFont.systemFont(ofSize: 18, weight: .medium)
        testQQBtn.setTitleColor(.blue, for: .normal)
        testQQBtn.addTarget(self, action: #selector(testOpenQQAction), for: .touchUpInside)
        testQQBtn.frame = CGRect(x: 50, y: 200, width: view.bounds.width - 100, height: 44)
        view.addSubview(testQQBtn)
    }
    
    // MARK: - 自动测试（页面加载后1秒触发，无需手动点击）
    private func autoTestQQScheme() {
        DispatchQueue.main.asyncAfter(deadline: .now() + 1) { [weak self] in
            self?.testOpenQQ()
        }
    }
    
    // MARK: - 手动点击按钮触发测试
    @objc private func testOpenQQAction() {
        testOpenQQ()
    }
    
    // MARK: - 核心测试方法（检测Scheme配置+唤起QQ）
    private func testOpenQQ() {
        guard let qqUrl = URL(string: "wtloginmqq://") else {
            print("【QQ Scheme测试】URL格式错误，无法创建wtloginmqq://链接")
            return
        }
        
        // 关键检测：系统是否允许App打开该Scheme（直接反映Info.plist配置是否生效）
        let canOpen = UIApplication.shared.canOpenURL(qqUrl)
        if canOpen {
            // 尝试唤起QQ
            UIApplication.shared.open(qqUrl) { [weak self] success in
                self?.printTestResult(canOpen: true, openSuccess: success)
            }
        } else {
            printTestResult(canOpen: false, openSuccess: false)
        }
    }
    
    // MARK: - 打印测试结果（控制台直接看解读，无需分析）
    private func printTestResult(canOpen: Bool, openSuccess: Bool) {
        print("=====================================")
        print("【QQ Scheme唤起测试-结果解读】")
        print("=====================================")
        if !canOpen {
            print("❌ 检测失败：系统禁止打开wtloginmqq://")
            print("👉 原因：1.Info.plist未配置LSApplicationQueriesSchemes 2.配置未生效 3.漏配wtloginmqq")
            print("👉 解决：检查Info.plist配置 + 清理Xcode缓存 + 重启真机")
        } else {
            print("✅ 检测成功：Info.plist的Scheme配置已生效（系统允许唤起）")
            if openSuccess {
                print("✅ 唤起成功：QQ App已被正常唤起，Scheme无问题")
                print("👉 提示：若京东页面仍无法唤起，是WKWebView拦截了Scheme，需手动处理")
            } else {
                print("⚠️ 唤起失败：系统允许但QQ拒绝被唤起（未在QQ互联平台注册）")
                print("👉 提示：浏览器能唤起是因为QQ默认信任系统级应用，第三方App需授权")
            }
        }
        print("=====================================\n")
    }
}
