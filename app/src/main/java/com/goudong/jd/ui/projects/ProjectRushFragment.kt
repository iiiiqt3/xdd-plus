package com.goudong.jd.ui.projects

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.fragment.app.FragmentContainerView
import androidx.fragment.app.commit

class ProjectRushFragment : Fragment() {
    private var containerId = View.NO_ID

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
        if (childFragmentManager.findFragmentById(containerId) == null) {
            showList()
        }
    }

    fun showList() {
        if (!isAdded) return
        childFragmentManager.commit {
            setReorderingAllowed(true)
            replace(containerId, ProjectRushListFragment(), TAG_LIST)
        }
    }

    fun openKuwo() {
        if (!isAdded) return
        childFragmentManager.commit {
            setReorderingAllowed(true)
            replace(containerId, KuwoRushFragment())
            addToBackStack(TAG_LIST)
        }
    }

    fun popToList() {
        if (!isAdded) return
        if (childFragmentManager.backStackEntryCount > 0) {
            childFragmentManager.popBackStackImmediate(TAG_LIST, 0)
        } else if (childFragmentManager.findFragmentById(containerId) !is ProjectRushListFragment) {
            showList()
        }
    }

    companion object {
        private const val TAG_LIST = "rush_list"
    }
}
