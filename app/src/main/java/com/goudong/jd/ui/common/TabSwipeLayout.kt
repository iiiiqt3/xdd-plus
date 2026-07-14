package com.goudong.jd.ui.common

import android.content.Context
import android.util.AttributeSet
import android.view.MotionEvent
import android.view.View
import android.view.ViewConfiguration
import android.view.ViewGroup
import android.widget.FrameLayout
import android.widget.HorizontalScrollView
import androidx.fragment.app.Fragment
import com.goudong.jd.MainActivity
import kotlin.math.abs

class TabSwipeLayout @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
) : FrameLayout(context, attrs) {

    var onHorizontalSwipe: ((direction: Int) -> Boolean)? = null

    private var downX = 0f
    private var downY = 0f
    private var tracking = false
    private val touchSlop = ViewConfiguration.get(context).scaledTouchSlop
    private val minSwipeDistance = context.dp(56)

    override fun onInterceptTouchEvent(ev: MotionEvent): Boolean {
        when (ev.actionMasked) {
            MotionEvent.ACTION_DOWN -> {
                downX = ev.x
                downY = ev.y
                tracking = true
                if (isInsideHorizontalScroller(ev.x, ev.y)) return false
            }
            MotionEvent.ACTION_MOVE -> {
                if (!tracking) return false
                if (isInsideHorizontalScroller(downX, downY)) return false
                val dx = ev.x - downX
                val dy = ev.y - downY
                if (abs(dx) > touchSlop && abs(dx) > abs(dy) * 1.25f) {
                    parent?.requestDisallowInterceptTouchEvent(true)
                    return true
                }
            }
            MotionEvent.ACTION_CANCEL, MotionEvent.ACTION_UP -> tracking = false
        }
        return false
    }

    private fun isInsideHorizontalScroller(x: Float, y: Float): Boolean {
        var view: View? = findViewAt(this, x, y)
        while (view != null && view !== this) {
            if (view is HorizontalScrollView) return true
            if (view.canScrollHorizontally(1) || view.canScrollHorizontally(-1)) return true
            view = view.parent as? View
        }
        return false
    }

    private fun findViewAt(parent: View, x: Float, y: Float): View? {
        if (x < 0 || y < 0 || x > parent.width || y > parent.height) return null
        if (parent !is ViewGroup) return parent
        for (i in parent.childCount - 1 downTo 0) {
            val child = parent.getChildAt(i)
            val hit = findViewAt(child, x - child.left, y - child.top)
            if (hit != null) return hit
        }
        return parent
    }

    override fun onTouchEvent(ev: MotionEvent): Boolean {
        when (ev.actionMasked) {
            MotionEvent.ACTION_DOWN -> {
                downX = ev.x
                downY = ev.y
                tracking = true
                return true
            }
            MotionEvent.ACTION_MOVE -> {
                val dx = ev.x - downX
                val dy = ev.y - downY
                if (abs(dx) > touchSlop && abs(dx) > abs(dy) * 1.25f) {
                    parent?.requestDisallowInterceptTouchEvent(true)
                }
            }
            MotionEvent.ACTION_UP -> {
                if (!tracking) return false
                tracking = false
                val dx = ev.x - downX
                val dy = ev.y - downY
                if (abs(dx) >= minSwipeDistance && abs(dx) > abs(dy) * 1.25f) {
                    val direction = if (dx < 0) 1 else -1
                    return onHorizontalSwipe?.invoke(direction) == true
                }
            }
            MotionEvent.ACTION_CANCEL -> tracking = false
        }
        return false
    }
}

fun Context.tabSwipeContainer(onSwipe: (direction: Int) -> Boolean, content: View): TabSwipeLayout {
    return TabSwipeLayout(this).apply {
        layoutParams = ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.MATCH_PARENT,
        )
        onHorizontalSwipe = onSwipe
        addView(
            content,
            FrameLayout.LayoutParams(
                FrameLayout.LayoutParams.MATCH_PARENT,
                FrameLayout.LayoutParams.MATCH_PARENT,
            ),
        )
    }
}

fun Fragment.wrapMainTabSwipe(content: View): TabSwipeLayout {
    return requireContext().tabSwipeContainer({ direction ->
        (activity as? MainActivity)?.handleHorizontalSwipe(this, direction) ?: false
    }, content)
}
