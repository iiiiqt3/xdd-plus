package com.goudong.jd

import android.app.Application
import com.goudong.jd.push.PushManager

class AndroidApp : Application() {
    override fun onCreate() {
        super.onCreate()
        AppServices.init(this)
        PushManager.init(this)
    }
}
