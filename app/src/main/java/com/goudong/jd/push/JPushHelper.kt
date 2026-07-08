package com.goudong.jd.push

import android.content.Context
import android.util.Log
import cn.jiguang.api.utils.JCollectionAuth
import cn.jpush.android.api.BasicPushNotificationBuilder
import cn.jpush.android.api.JPushInterface
import com.goudong.jd.AppServices
import com.goudong.jd.BuildConfig
import com.goudong.jd.R
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

object JPushHelper {
    private const val TAG = "JPushHelper"
    private const val ALIAS_SEQUENCE = 1

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    @Volatile
    private var initialized = false

    fun ensureInitialized(context: Context) {
        if (initialized) return
        val appContext = context.applicationContext
        JCollectionAuth.setAuth(appContext, true)
        JPushInterface.setDebugMode(BuildConfig.DEBUG)
        NotificationHelper.createNotificationChannel(appContext)
        configureDefaultNotificationBuilder(appContext)
        JPushInterface.init(appContext)
        initialized = true
    }

    private fun configureDefaultNotificationBuilder(context: Context) {
        runCatching {
            val builder = BasicPushNotificationBuilder(context).apply {
                developerArg0 = NotificationHelper.CHANNEL_ID
                statusBarDrawable = R.drawable.ic_notification
                notificationDefaults = android.app.Notification.DEFAULT_ALL
                notificationFlags = android.app.Notification.FLAG_AUTO_CANCEL
            }
            JPushInterface.setDefaultPushNotificationBuilder(builder)
        }.onFailure { error ->
            Log.w(TAG, "configureDefaultNotificationBuilder failed: ${error.message}")
        }
    }

    fun bindUser(context: Context) {
        if (!AppServices.sessionManager.isAuthenticated()) return
        ensureInitialized(context)
        scope.launch {
            runCatching {
                val dashboard = AppServices.portalRepository.fetchDashboard()
                val userNumber = dashboard.number.toInt()
                if (userNumber <= 0) return@runCatching
                val alias = portalAlias(userNumber)
                JPushInterface.resumePush(context.applicationContext)
                JPushInterface.setAlias(context.applicationContext, ALIAS_SEQUENCE, alias)
                registerDeviceIfReady(context, alias)
            }.onFailure { error ->
                Log.w(TAG, "bindUser failed: ${error.message}")
            }
        }
    }

    fun registerDeviceIfReady(context: Context, alias: String? = null) {
        if (!AppServices.sessionManager.isAuthenticated()) return
        scope.launch {
            runCatching {
                val resolvedAlias = alias ?: run {
                    val dashboard = AppServices.portalRepository.fetchDashboard()
                    portalAlias(dashboard.number.toInt())
                }
                val regId = JPushInterface.getRegistrationID(context.applicationContext)
                if (regId.isBlank()) return@runCatching
                AppServices.portalRepository.registerPushDevice(regId, resolvedAlias)
            }.onFailure { error ->
                Log.w(TAG, "registerDevice failed: ${error.message}")
            }
        }
    }

    fun unbindUser(context: Context) {
        val appContext = context.applicationContext
        scope.launch {
            runCatching {
                if (AppServices.sessionManager.isAuthenticated()) {
                    val regId = JPushInterface.getRegistrationID(appContext)
                    AppServices.portalRepository.unregisterPushDevice(regId)
                }
            }.onFailure { error ->
                Log.w(TAG, "unregisterPushDevice failed: ${error.message}")
            }
            JPushInterface.deleteAlias(appContext, ALIAS_SEQUENCE)
        }
    }

    private fun portalAlias(userNumber: Int): String = "portal_$userNumber"
}
