package com.goudong.jd.push

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.content.ContextCompat
import com.goudong.jd.AppServices

object PushManager {
    private var isInitialized = false

    fun init(context: Context) {
        if (isInitialized) return
        NotificationHelper.createNotificationChannel(context)
        JPushHelper.ensureInitialized(context)
        isInitialized = true
    }

    fun startMonitoring(context: Context) {
        if (!AppServices.sessionManager.isAuthenticated()) return
        JPushHelper.bindUser(context)
    }

    fun stopMonitoring(context: Context) {
        JPushHelper.unbindUser(context)
    }

    fun onAppForeground() {
        ForegroundPushNotifier.isAppInForeground = true
    }

    fun onAppBackground() {
        ForegroundPushNotifier.isAppInForeground = false
        ForegroundPushNotifier.topActivity = null
        InAppPushBanner.dismiss()
    }

    fun onUserLogout() {
        JPushHelper.unbindUser(AppServices.appContext)
        NotificationHelper.cancelAll(AppServices.appContext)
        ForegroundPushNotifier.clearPending()
        InAppPushBanner.dismiss()
    }

    fun onUserLogin(context: Context) {
        requestNotificationPermissionIfNeeded(context)
        JPushHelper.bindUser(context)
    }

    fun hasNotificationPermission(context: Context): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return true
        return ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.POST_NOTIFICATIONS,
        ) == PackageManager.PERMISSION_GRANTED
    }

    fun requestNotificationPermissionIfNeeded(context: Context) {
        // 由 Activity 在合适时机调用权限请求；此处仅保留检查入口
        if (!hasNotificationPermission(context)) {
            android.util.Log.d("PushManager", "通知权限未授予，推送可能无法展示")
        }
    }
}
