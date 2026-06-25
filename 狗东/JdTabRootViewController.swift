import UIKit

@available(iOS 13.0, *)
final class JdTabRootViewController: UIViewController {
    private var guestController: MainUIViewController?
    private var portalController: JdPortalViewController?
    private var showingPortal: Bool?

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemBackground
        NotificationCenter.default.addObserver(self, selector: #selector(sessionChanged), name: AppNotifications.sessionDidChange, object: nil)
        NotificationCenter.default.addObserver(self, selector: #selector(sessionChanged), name: AppNotifications.sessionDidLogout, object: nil)
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        navigationController?.setNavigationBarHidden(true, animated: animated)
        updateChildIfNeeded()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        navigationController?.setNavigationBarHidden(false, animated: animated)
        dismissChildEditing()
    }

    @objc private func sessionChanged() {
        updateChildIfNeeded()
    }

    private func updateChildIfNeeded() {
        let portal = AppSessionStore.shared.isAuthenticated
        if showingPortal == portal { return }

        dismissChildEditing()
        detachChild(guestController)
        guestController = nil
        detachChild(portalController)
        if !portal {
            portalController = nil
        }

        showingPortal = portal

        let child: UIViewController
        if portal {
            let vc = portalController ?? JdPortalViewController()
            portalController = vc
            child = vc
        } else {
            let vc = MainUIViewController()
            guestController = vc
            child = vc
        }

        attachChild(child)
    }

    private func dismissChildEditing() {
        view.endEditing(true)
        guestController?.view.endEditing(true)
        portalController?.view.endEditing(true)
    }

    private func detachChild(_ child: UIViewController?) {
        guard let child = child else { return }
        child.view.endEditing(true)
        child.willMove(toParent: nil)
        child.view.removeFromSuperview()
        child.removeFromParent()
    }

    private func attachChild(_ child: UIViewController) {
        addChild(child)
        child.view.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(child.view)
        NSLayoutConstraint.activate([
            child.view.topAnchor.constraint(equalTo: view.topAnchor),
            child.view.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            child.view.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            child.view.bottomAnchor.constraint(equalTo: view.bottomAnchor)
        ])
        child.didMove(toParent: self)
    }
}
