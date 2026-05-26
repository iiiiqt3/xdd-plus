package com.goudong.jd

import android.content.Context
import com.goudong.jd.data.network.ApiClient
import com.goudong.jd.data.repo.AuthRepository
import com.goudong.jd.data.repo.JdRepository
import com.goudong.jd.data.repo.PortalRepository
import com.goudong.jd.data.session.PersistentCookieJar
import com.goudong.jd.data.session.SessionManager
import com.google.gson.Gson

object AppServices {
    lateinit var appContext: Context
        private set

    @Volatile
    var isAuthInProgress = false

    lateinit var sessionManager: SessionManager
        private set
    lateinit var cookieJar: PersistentCookieJar
        private set
    lateinit var gson: Gson
        private set
    lateinit var apiClient: ApiClient
        private set
    lateinit var authRepository: AuthRepository
        private set
    lateinit var portalRepository: PortalRepository
        private set
    lateinit var jdRepository: JdRepository
        private set

    fun init(context: Context) {
        if (::appContext.isInitialized) return
        appContext = context.applicationContext
        sessionManager = SessionManager(appContext)
        cookieJar = PersistentCookieJar(appContext)
        gson = Gson()
        apiClient = ApiClient(cookieJar, gson)
        authRepository = AuthRepository(apiClient, sessionManager)
        portalRepository = PortalRepository(apiClient, sessionManager)
        jdRepository = JdRepository(apiClient, gson)
    }
}
