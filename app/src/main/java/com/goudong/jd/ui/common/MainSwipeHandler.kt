package com.goudong.jd.ui.common

import android.view.GestureDetector
import android.view.MotionEvent
import androidx.core.view.GestureDetectorCompat
import kotlin.math.abs

class MainSwipeHandler(
    context: android.content.Context,
    private val onSwipe: (direction: Int) -> Boolean,
) {
    private var startX = 0f
    private var startY = 0f
    private val minDistance = context.resources.displayMetrics.density * 72f

    private val detector = GestureDetectorCompat(
        context,
        object : GestureDetector.SimpleOnGestureListener() {
            override fun onDown(e: MotionEvent): Boolean = true

            override fun onFling(
                e1: MotionEvent?,
                e2: MotionEvent,
                velocityX: Float,
                velocityY: Float,
            ): Boolean {
                if (e1 == null) return false
                if (abs(velocityX) < abs(velocityY) * 1.1f) return false
                val direction = if (velocityX < 0) 1 else -1
                return onSwipe(direction)
            }
        },
    )

    fun onTouchEvent(event: MotionEvent): Boolean {
        detector.onTouchEvent(event)
        when (event.actionMasked) {
            MotionEvent.ACTION_DOWN -> {
                startX = event.x
                startY = event.y
            }
            MotionEvent.ACTION_UP -> {
                val dx = event.x - startX
                val dy = event.y - startY
                if (abs(dx) >= minDistance && abs(dx) > abs(dy) * 1.35f) {
                    val direction = if (dx < 0) 1 else -1
                    onSwipe(direction)
                }
            }
        }
        return false
    }
}
