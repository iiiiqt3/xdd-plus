import UIKit

class SceneDelegate: UIResponder, UIWindowSceneDelegate {
    var window: UIWindow?

    @available(iOS 13.0, *)
    func scene(_ scene: UIScene, openURLContexts URLContexts: Set<UIOpenURLContext>) {
        guard let url = URLContexts.first?.url else { return }
        let scheme = url.scheme?.lowercased() ?? ""
        let qqSchemes = ["mqq", "mqqapi", "mqqopensdkapi", "qqauth", "wtloginmqq"]
        if qqSchemes.contains(scheme) {
            NotificationCenter.default.post(name: AppNotifications.jdAuthCallback, object: nil)
        }
    }
}
