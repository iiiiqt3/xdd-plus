package com.goudong.jd.data.session

import android.content.Context
import android.content.SharedPreferences

class SessionManager(context: Context) {
    private val prefs: SharedPreferences = context.getSharedPreferences("android_jd_session", Context.MODE_PRIVATE)

    fun saveCredentials(username: String, password: String) {
        prefs.edit()
            .putString(KEY_USERNAME, username)
            .putString(KEY_PASSWORD, password)
            .apply()
    }

    fun loadCredentials(): Pair<String, String>? {
        val username = prefs.getString(KEY_USERNAME, null)
        val password = prefs.getString(KEY_PASSWORD, null)
        return if (username.isNullOrBlank() || password.isNullOrBlank()) null else username to password
    }

    fun clearCredentials() {
        prefs.edit().remove(KEY_USERNAME).remove(KEY_PASSWORD).apply()
    }

    fun setAuthenticated(value: Boolean) {
        prefs.edit().putBoolean(KEY_AUTHENTICATED, value).apply()
    }

    fun isAuthenticated(): Boolean = prefs.getBoolean(KEY_AUTHENTICATED, false)

    fun saveQq(value: String) {
        prefs.edit().putString(KEY_QQ, value).apply()
    }

    fun loadQq(): String = prefs.getString(KEY_QQ, "") ?: ""

    fun saveHomeSummary(summary: String, detail: String, coin: String, active: String, expiring: String, wechat: String) {
        prefs.edit()
            .putString(KEY_HOME_SUMMARY, summary)
            .putString(KEY_HOME_DETAIL, detail)
            .putString(KEY_HOME_COIN, coin)
            .putString(KEY_HOME_ACTIVE, active)
            .putString(KEY_HOME_EXPIRING, expiring)
            .putString(KEY_HOME_WECHAT, wechat)
            .apply()
    }

    fun loadHomeSummary(): HomeCache? {
        val summary = prefs.getString(KEY_HOME_SUMMARY, null) ?: return null
        return HomeCache(
            summary = summary,
            detail = prefs.getString(KEY_HOME_DETAIL, "") ?: "",
            coin = prefs.getString(KEY_HOME_COIN, "-") ?: "-",
            active = prefs.getString(KEY_HOME_ACTIVE, "-") ?: "-",
            expiring = prefs.getString(KEY_HOME_EXPIRING, "-") ?: "-",
            wechat = prefs.getString(KEY_HOME_WECHAT, "-") ?: "-",
        )
    }

    data class HomeCache(
        val summary: String,
        val detail: String,
        val coin: String,
        val active: String,
        val expiring: String,
        val wechat: String,
    )

    companion object {
        private const val KEY_USERNAME = "username"
        private const val KEY_PASSWORD = "password"
        private const val KEY_AUTHENTICATED = "authenticated"
        private const val KEY_QQ = "qq"
        private const val KEY_HOME_SUMMARY = "home_summary"
        private const val KEY_HOME_DETAIL = "home_detail"
        private const val KEY_HOME_COIN = "home_coin"
        private const val KEY_HOME_ACTIVE = "home_active"
        private const val KEY_HOME_EXPIRING = "home_expiring"
        private const val KEY_HOME_WECHAT = "home_wechat"
    }
}
