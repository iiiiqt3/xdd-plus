package com.goudong.jd.ui.common

import android.content.Context
import android.util.AttributeSet
import android.view.MotionEvent
import android.view.View
import android.view.ViewConfiguration
import android.view.ViewGroup
import android.widget.FrameLayout
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
            }
            MotionEvent.ACTION_MOVE -> {
                if (!tracking) return false
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
