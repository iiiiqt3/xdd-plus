package com.goudong.jd.ui.common

import android.view.MotionEvent
import kotlin.math.abs

class MainSwipeHandler(
    context: android.content.Context,
    private val onSwipe: (direction: Int) -> Boolean,
) {
    private var startX = 0f
    private var startY = 0f
    private var tracking = false
    private var handledThisGesture = false
    private val minDistance = context.resources.displayMetrics.density * 88f

    fun onTouchEvent(event: MotionEvent): Boolean {
        when (event.actionMasked) {
            MotionEvent.ACTION_DOWN -> {
                startX = event.x
                startY = event.y
                tracking = true
                handledThisGesture = false
            }
            MotionEvent.ACTION_UP, MotionEvent.ACTION_CANCEL -> {
                if (!tracking || handledThisGesture) {
                    tracking = false
                    return false
                }
                tracking = false
                val dx = event.x - startX
                val dy = event.y - startY
                if (abs(dx) >= minDistance && abs(dx) > abs(dy) * 1.5f) {
                    val direction = if (dx < 0) 1 else -1
                    handledThisGesture = onSwipe(direction)
                }
            }
            MotionEvent.ACTION_POINTER_UP -> Unit
            else -> Unit
        }
        return false
    }
}
