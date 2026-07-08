package com.goudong.jd

import android.app.Activity
import android.app.Application
import android.os.Bundle
import com.goudong.jd.push.ForegroundPushNotifier
import com.goudong.jd.push.InAppPushBanner
import com.goudong.jd.push.PushManager

class AndroidApp : Application() {
    private var startedActivityCount = 0

    override fun onCreate() {
        super.onCreate()
        AppServices.init(this)
        PushManager.init(this)
        registerActivityLifecycleCallbacks(object : ActivityLifecycleCallbacks {
            override fun onActivityCreated(activity: Activity, savedInstanceState: Bundle?) = Unit

            override fun onActivityStarted(activity: Activity) {
                startedActivityCount++
                ForegroundPushNotifier.isAppInForeground = true
            }

            override fun onActivityResumed(activity: Activity) {
                ForegroundPushNotifier.topActivity = activity
                if (ForegroundPushNotifier.pendingCount() > 0 && !InAppPushBanner.isShowing()) {
                    ForegroundPushNotifier.refreshBannerIfNeeded(activity)
                }
            }

            override fun onActivityPaused(activity: Activity) {
                if (ForegroundPushNotifier.topActivity === activity) {
                    ForegroundPushNotifier.topActivity = null
                }
            }

            override fun onActivityStopped(activity: Activity) {
                startedActivityCount--
                if (startedActivityCount <= 0) {
                    startedActivityCount = 0
                    ForegroundPushNotifier.isAppInForeground = false
                    ForegroundPushNotifier.topActivity = null
                    ForegroundPushNotifier.clearPending()
                    InAppPushBanner.dismiss()
                }
            }

            override fun onActivitySaveInstanceState(activity: Activity, outState: Bundle) = Unit

            override fun onActivityDestroyed(activity: Activity) = Unit
        })
    }
}
