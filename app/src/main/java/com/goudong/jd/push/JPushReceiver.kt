package com.goudong.jd.push

import android.content.Context
import android.util.Log
import cn.jpush.android.api.JPushMessage
import cn.jpush.android.api.NotificationMessage
import cn.jpush.android.service.JPushMessageReceiver

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

    override fun onNotifyMessageArrived(context: Context, message: NotificationMessage) {
        val extras = message.notificationExtras
        val notificationId = PushExtrasParser.parseNotificationId(extras)
        val displayType = PushExtrasParser.parseDisplayType(extras)
        val action = PushExtrasParser.parseAction(extras)
        val fullBody = PushExtrasParser.parseFullBody(extras)
        val title = message.notificationTitle?.trim().orEmpty()
        val body = message.notificationContent?.trim().orEmpty()
        Log.d(TAG, "onNotifyMessageArrived id=$notificationId action=$action title=$title")
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

    companion object {
        private const val TAG = "JPushReceiver"
    }
}
