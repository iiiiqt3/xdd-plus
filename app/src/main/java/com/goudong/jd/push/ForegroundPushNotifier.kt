package com.goudong.jd.push

import android.app.Activity
import android.content.Context
import android.os.Handler
import android.os.Looper
import java.util.LinkedHashMap

/**
 * 前台推送分发：多条消息汇总为「您有 N 条新消息」横幅。
 */
object ForegroundPushNotifier {
    data class PushPayload(
        val title: String,
        val body: String,
        val notificationId: Int,
        val displayType: String,
        val action: String,
        val fullBody: String,
        val key: Int,
    )

    @Volatile
    var isAppInForeground: Boolean = false

    @Volatile
    var topActivity: Activity? = null

    private val mainHandler = Handler(Looper.getMainLooper())
    private val pending = LinkedHashMap<Int, PushPayload>()
    private var lastDuplicateId = 0
    private var lastDuplicateAt = 0L

    fun pendingCount(): Int = pending.size

    fun pendingList(): List<PushPayload> = pending.values.toList()

    fun onMessageArrived(
        context: Context,
        title: String,
        body: String,
        notificationId: Int,
        displayType: String = "normal",
        action: String = "",
        fullBody: String = "",
    ) {
        val safeTitle = title.trim().ifEmpty { "狗东通知" }
        val safeBody = body.trim().ifEmpty { "您有一条新消息" }
        val now = System.currentTimeMillis()
        if (notificationId > 0 && notificationId == lastDuplicateId && now - lastDuplicateAt < 3000) {
            return
        }
        lastDuplicateId = notificationId
        lastDuplicateAt = now

        val key = if (notificationId > 0) notificationId else -now.toInt()
        pending[key] = PushPayload(safeTitle, safeBody, notificationId, displayType, action, fullBody, key)

        mainHandler.post { refreshBanner(context) }
    }

    fun removePending(key: Int) {
        pending.remove(key)
    }

    fun clearPending() {
        pending.clear()
        lastDuplicateId = 0
        lastDuplicateAt = 0L
    }

    fun refreshBannerIfNeeded(context: Context) {
        mainHandler.post { refreshBanner(context) }
    }

    private fun refreshBanner(context: Context) {
        if (!isAppInForeground || pending.isEmpty()) return
        val activity = topActivity
        if (activity != null && !activity.isFinishing) {
            InAppPushBanner.showSummary(activity, pendingList())
            NotificationBadgeRefresher.refresh()
        } else {
            val latest = pendingList().last()
            NotificationHelper.showPushMessage(context, latest.title, latest.body, latest.notificationId)
        }
    }
}
