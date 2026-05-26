package com.goudong.jd.update

import android.content.Context
import com.goudong.jd.BuildConfig
import com.goudong.jd.data.model.AppEnvironment
import com.goudong.jd.data.network.ApiClient
import com.google.gson.Gson

class UpdateChecker(
    private val apiClient: ApiClient,
    private val context: Context,
) {
    private val gson = Gson()

    suspend fun check(): UpdateInfo? {
        val text = try {
            apiClient.requestText(absoluteUrl = AppEnvironment.UPDATE_CHECK_URL)
        } catch (_: Exception) {
            return null
        }
        val info = try {
            gson.fromJson(text.trim(), UpdateInfo::class.java)
        } catch (_: Exception) {
            return null
        } ?: return null
        return if (info.versionCode > BuildConfig.VERSION_CODE) info else null
    }
}
