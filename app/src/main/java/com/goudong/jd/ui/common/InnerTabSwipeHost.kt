package com.goudong.jd.ui.common

import android.widget.ScrollView
import android.view.View
import android.view.ViewGroup

interface InnerTabSwipeHost {
    val innerTabCount: Int
    val innerTabIndex: Int
    fun selectInnerTab(index: Int)
    /** @return true if swipe was consumed (e.g. popped nested page) */
    fun onInnerSwipeBoundary(direction: Int): Boolean = false
}

interface MainTabResettable {
    fun resetToInitialState()
}

fun View.findFirstScrollView(): ScrollView? {
    if (this is ScrollView) return this
    if (this is ViewGroup) {
        for (i in 0 until childCount) {
            getChildAt(i).findFirstScrollView()?.let { return it }
        }
    }
    return null
}
