package com.goudong.jd.push

import android.content.Context
import android.os.Handler
import android.os.Looper
import android.util.Log
import cn.jpush.android.api.JPushInterface
import cn.jpush.android.api.JPushMessage
import cn.jpush.android.api.NotificationMessage
import cn.jpush.android.service.JPushMessageReceiver

class JPushReceiver : JPushMessageReceiver() {
    private val mainHandler = Handler(Looper.getMainLooper())

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

    override fun onNotifyMessageArrived(context: Context, message: NotificationMessage) {
        val extras = message.notificationExtras
        val notificationId = PushExtrasParser.parseNotificationId(extras)
        val displayType = PushExtrasParser.parseDisplayType(extras)
        val action = PushExtrasParser.parseAction(extras)
        val fullBody = PushExtrasParser.parseFullBody(extras)
        val title = message.notificationTitle?.trim().orEmpty()
        val body = message.notificationContent?.trim().orEmpty()
        Log.d(TAG, "onNotifyMessageArrived id=$notificationId action=$action title=$title")

        if (ForegroundPushNotifier.isAppInForeground) {
            suppressJPushSystemNotification(context, message)
            ForegroundPushNotifier.onMessageArrived(
                context, title, body, notificationId, displayType, action, fullBody,
            )
            return
        }

        ForegroundPushNotifier.onMessageArrived(
            context, title, body, notificationId, displayType, action, fullBody,
        )
    }

    override fun onNotifyMessageOpened(context: Context, message: NotificationMessage) {
        PushNavigationHelper.openFromExtras(
            context,
            message.notificationExtras,
            fallbackTitle = message.notificationTitle?.trim().orEmpty(),
            fallbackBody = message.notificationContent?.trim().orEmpty(),
        )
    }

    /** 前台已由应用内横幅展示，取消极光自动弹出的系统通知，避免双横幅 */
    private fun suppressJPushSystemNotification(context: Context, message: NotificationMessage) {
        val appContext = context.applicationContext
        val jpushNotificationId = message.notificationId
        val clear = Runnable {
            if (jpushNotificationId > 0) {
                runCatching {
                    JPushInterface.clearNotificationById(appContext, jpushNotificationId)
                }.onFailure { error ->
                    Log.w(TAG, "clearNotificationById failed: ${error.message}")
                }
            }
        }
        clear.run()
        mainHandler.postDelayed(clear, 150)
        mainHandler.postDelayed(clear, 500)
    }

    companion object {
        private const val TAG = "JPushReceiver"
    }
}
