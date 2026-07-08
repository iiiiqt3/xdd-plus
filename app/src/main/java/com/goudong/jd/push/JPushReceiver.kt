package com.goudong.jd.push

import android.content.Context
import android.content.Intent
import android.util.Log
import cn.jpush.android.api.JPushMessage
import cn.jpush.android.api.NotificationMessage
import cn.jpush.android.service.JPushMessageReceiver
import com.goudong.jd.MainActivity
import com.goudong.jd.ui.more.NotificationDetailActivity
import org.json.JSONObject

class JPushReceiver : JPushMessageReceiver() {
    override fun onRegister(context: Context, registrationId: String) {
        Log.d(TAG, "onRegister: $registrationId")
        JPushHelper.registerDeviceIfReady(context)
    }

    override fun onAliasOperatorResult(context: Context, message: JPushMessage) {
        super.onAliasOperatorResult(context, message)
        if (message.errorCode == 0) {
            JPushHelper.registerDeviceIfReady(context)
        } else {
            Log.w(TAG, "setAlias failed: ${message.errorCode}")
        }
    }

    override fun onNotifyMessageOpened(context: Context, message: NotificationMessage) {
        val notificationId = parseNotificationId(message.notificationExtras)
        val intent = if (notificationId > 0) {
            Intent(context, NotificationDetailActivity::class.java).apply {
                putExtra(NotificationDetailActivity.EXTRA_NOTIFICATION_ID, notificationId)
            }
        } else {
            Intent(context, MainActivity::class.java).apply {
                putExtra(MainActivity.EXTRA_OPEN_NOTIFICATIONS, true)
            }
        }
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        context.startActivity(intent)
    }

    private fun parseNotificationId(extras: String?): Int {
        if (extras.isNullOrBlank()) return 0
        return runCatching { JSONObject(extras).optInt("notificationId", 0) }.getOrDefault(0)
    }

    companion object {
        private const val TAG = "JPushReceiver"
    }
}
