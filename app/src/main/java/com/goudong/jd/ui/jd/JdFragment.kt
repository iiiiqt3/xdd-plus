package com.goudong.jd.ui.jd

import android.graphics.Color
import android.os.Bundle
import android.util.Log
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.TextView
import androidx.fragment.app.Fragment
import androidx.fragment.app.FragmentContainerView
import androidx.fragment.app.commit
import com.goudong.jd.AppServices

class JdFragment : Fragment() {
    private var containerId = View.NO_ID
    private var showingPortal: Boolean? = null
    private var pendingSync = false

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        if (containerId == View.NO_ID) {
            containerId = View.generateViewId()
        }
        return FragmentContainerView(requireContext()).apply {
            id = containerId
            layoutParams = ViewGroup.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.MATCH_PARENT,
            )
        }
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        view.post { syncChildFragment(force = true) }
    }

    override fun onResume() {
        super.onResume()
        view?.post { syncChildFragment(force = false) }
    }

    private fun syncChildFragment(force: Boolean) {
        if (!isAdded || view == null) return
        if (pendingSync) return

        try {
            val portal = AppServices.sessionManager.isAuthenticated()
            val wantTag = if (portal) TAG_PORTAL else TAG_GUEST
            val existing = childFragmentManager.findFragmentById(containerId)

            if (!force && showingPortal == portal && existing != null && existing.tag == wantTag) {
                return
            }

            if (childFragmentManager.isStateSaved) {
                view?.post { syncChildFragment(force = force) }
                return
            }

            showingPortal = portal
            pendingSync = true

            val fragment = when {
                existing != null && existing.tag == wantTag -> existing
                portal -> JdPortalFragment()
                else -> JdGuestFragment()
            }

            childFragmentManager.commit {
                setReorderingAllowed(true)
                replace(containerId, fragment, wantTag)
            }
            childFragmentManager.executePendingTransactions()
        } catch (e: Exception) {
            Log.e(TAG, "syncChildFragment failed", e)
            showLoadError(e)
        } finally {
            pendingSync = false
        }
    }

    private fun showLoadError(error: Exception) {
        val host = view as? ViewGroup ?: return
        host.removeAllViews()
        host.addView(
            TextView(requireContext()).apply {
                text = "京东页面加载失败: ${error.message ?: error.javaClass.simpleName}"
                setTextColor(Color.RED)
                setPadding(48, 48, 48, 48)
            },
        )
    }

    companion object {
        private const val TAG = "JdFragment"
        private const val TAG_GUEST = "jd_guest"
        private const val TAG_PORTAL = "jd_portal"
    }
}
