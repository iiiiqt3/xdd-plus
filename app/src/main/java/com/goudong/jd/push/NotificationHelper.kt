package com.goudong.jd.push

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.core.app.NotificationCompat
import com.goudong.jd.MainActivity
import com.goudong.jd.R
import com.goudong.jd.data.model.PortalNotification

object NotificationHelper {
    const val CHANNEL_ID = "admin_push_channel"
    private const val CHANNEL_NAME = "狗东通知"
    private const val CHANNEL_DESC = "接收系统通知、活动提醒与掉线提醒"
    private const val NOTIFICATION_ID = 1001

    fun createNotificationChannel(context: Context) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                CHANNEL_NAME,
                NotificationManager.IMPORTANCE_HIGH,
            ).apply {
                description = CHANNEL_DESC
                enableLights(true)
                enableVibration(true)
                setShowBadge(true)
                lockscreenVisibility = Notification.VISIBILITY_PUBLIC
            }
            val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            manager.createNotificationChannel(channel)
        }
    }

    fun showPushMessage(context: Context, title: String, body: String, notificationId: Int) {
        createNotificationChannel(context)
        val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        val detailIntent = buildTapIntent(context, notificationId, openList = false)
        val pendingIntent = PendingIntent.getActivity(
            context,
            notificationId.coerceAtLeast(1),
            detailIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(body)
            .setStyle(NotificationCompat.BigTextStyle().bigText(body))
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setDefaults(NotificationCompat.DEFAULT_ALL)
            .setCategory(NotificationCompat.CATEGORY_MESSAGE)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()
        manager.notify(notificationId.coerceAtLeast(1) + 20000, notification)
    }

    fun showUnreadSummaryNotification(context: Context, unreadCount: Int, latestNotifications: List<PortalNotification>) {
        if (unreadCount <= 0) return

        createNotificationChannel(context)
        val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

        val intent = buildTapIntent(context, notificationId = 0, openList = true)

        val pendingIntent = PendingIntent.getActivity(
            context,
            System.currentTimeMillis().toInt(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )

        val title = "您有 $unreadCount 条新消息"

        val contentText = when {
            unreadCount == 1 && latestNotifications.isNotEmpty() -> {
                val notif = latestNotifications[0]
                notif.title ?: "点击查看详情"
            }
            latestNotifications.size >= 2 -> {
                val titles = latestNotifications.take(2).mapNotNull { it.title }
                "${titles[0]} 等${unreadCount}条消息"
            }
            else -> "点击查看所有消息"
        }

        val inboxStyle = NotificationCompat.InboxStyle()
            .setBigContentTitle(title)
            .setSummaryText("狗东助手")

        latestNotifications.take(5).forEach { notif ->
            val line = "• ${notif.title ?: "新消息"}"
            if (notif.isRead) {
                inboxStyle.addLine(android.text.SpannableString(line).apply {
                    setSpan(android.text.style.StrikethroughSpan(), 0, length, 0)
                })
            } else {
                inboxStyle.addLine(line)
            }
        }

        if (unreadCount > 5) {
            inboxStyle.addLine("... 还有 ${unreadCount - 5} 条消息")
        }

        val builder = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(contentText)
            .setStyle(inboxStyle)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .setDefaults(NotificationCompat.DEFAULT_ALL)
            .setCategory(NotificationCompat.CATEGORY_MESSAGE)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            builder.setGroup("admin_push_group")
            builder.setNumber(unreadCount)
        }

        manager.notify(NOTIFICATION_ID, builder.build())
    }

    private fun buildTapIntent(context: Context, notificationId: Int, openList: Boolean): Intent {
        return Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            if (notificationId > 0) {
                putExtra(MainActivity.EXTRA_NOTIFICATION_ID, notificationId)
            } else if (openList) {
                putExtra(MainActivity.EXTRA_OPEN_NOTIFICATIONS, true)
            }
        }
    }

    fun cancelAll(context: Context) {
        val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.cancelAll()
    }
}
