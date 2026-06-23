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
    }

    @objc private func sessionChanged() {
        updateChildIfNeeded(force: true)
    }

    private func updateChildIfNeeded(force: Bool = false) {
        let portal = AppSessionStore.shared.isAuthenticated
        if !force, showingPortal == portal { return }
        showingPortal = portal

        guestController?.willMove(toParent: nil)
        guestController?.view.removeFromSuperview()
        guestController?.removeFromParent()
        portalController?.willMove(toParent: nil)
        portalController?.view.removeFromSuperview()
        portalController?.removeFromParent()

        let child: UIViewController
        if portal {
            let vc = JdPortalViewController()
            portalController = vc
            child = vc
        } else {
            let vc = MainUIViewController()
            guestController = vc
            child = vc
        }

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
