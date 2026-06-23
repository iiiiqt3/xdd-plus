package com.goudong.jd.ui.jd

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.FrameLayout
import androidx.fragment.app.Fragment
import androidx.fragment.app.commit
import com.goudong.jd.AppServices

class JdFragment : Fragment() {
    private val containerId = View.generateViewId()
    private var showingPortal: Boolean? = null

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        return FrameLayout(requireContext()).apply { id = containerId }
    }

    override fun onResume() {
        super.onResume()
        val portal = AppServices.sessionManager.isAuthenticated()
        if (showingPortal == portal) return
        showingPortal = portal
        val tag = if (portal) TAG_PORTAL else TAG_GUEST
        val fragment = if (portal) JdPortalFragment() else JdGuestFragment()
        childFragmentManager.commit {
            replace(containerId, fragment, tag)
        }
    }

    companion object {
        private const val TAG_GUEST = "jd_guest"
        private const val TAG_PORTAL = "jd_portal"
    }
}
