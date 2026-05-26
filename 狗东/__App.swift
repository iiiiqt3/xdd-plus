import SwiftUI
import UIKit

// 指定最低支持的iOS版本
@available(iOS 14.0, *)
@main
struct MyApp: App {
    var body: some Scene {
        WindowGroup {
            UIKitViewControllerWrapper()
        }
    }
}

@available(iOS 13.0, *)
struct UIKitViewControllerWrapper: UIViewControllerRepresentable {
    func makeUIViewController(context: Context) -> UIViewController {
        RootTabBarController()
    }

    func updateUIViewController(_ uiViewController: UIViewController, context: Context) {}
}

