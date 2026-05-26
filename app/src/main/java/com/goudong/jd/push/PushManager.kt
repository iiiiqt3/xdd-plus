package com.goudong.jd.push

import android.content.Context
import androidx.work.Constraints
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import com.goudong.jd.AppServices
import java.util.concurrent.TimeUnit

object PushManager {
    private const val WORK_NAME = "push_check_work"
    private var isInitialized = false
    
    fun init(context: Context) {
        if (isInitialized) return
        
        NotificationHelper.createNotificationChannel(context)
        schedulePeriodicCheck(context)
        isInitialized = true
    }
    
    fun startMonitoring(context: Context) {
        if (AppServices.sessionManager.isAuthenticated()) {
            schedulePeriodicCheck(context, replace = true)
        }
    }

    fun stopMonitoring(context: Context) {
        WorkManager.getInstance(context).cancelUniqueWork(WORK_NAME)
    }

    fun onAppForeground() {
        PushCheckWorker.isAppInForeground = true
    }

    fun onAppBackground() {
        PushCheckWorker.isAppInForeground = false
    }

    fun onUserLogout() {
        stopMonitoring(AppServices.appContext)
        NotificationHelper.cancelAll(AppServices.appContext)
        PushCheckWorker.clearCache()
    }

    fun onUserLogin(context: Context) {
        PushCheckWorker.clearCache()
        startMonitoring(context)
    }

    private fun schedulePeriodicCheck(
        context: Context,
        replace: Boolean = false
    ) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .setRequiresBatteryNotLow(true)
            .build()

        val workRequest = PeriodicWorkRequestBuilder<PushCheckWorker>(
            15, TimeUnit.MINUTES,
            5, TimeUnit.MINUTES
        )
            .setConstraints(constraints)
            .setInitialDelay(10, TimeUnit.SECONDS)
            .build()

        val policy = if (replace) {
            ExistingPeriodicWorkPolicy.UPDATE
        } else {
            ExistingPeriodicWorkPolicy.KEEP
        }

        WorkManager.getInstance(context).enqueueUniquePeriodicWork(WORK_NAME, policy, workRequest)
    }
}