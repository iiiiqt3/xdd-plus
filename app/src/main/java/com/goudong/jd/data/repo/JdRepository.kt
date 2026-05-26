package com.goudong.jd.data.repo

import com.goudong.jd.data.network.ApiClient
import org.json.JSONObject

class JdRepository(
    private val apiClient: ApiClient,
    private val gson: com.google.gson.Gson,
) {
    suspend fun submitCookie(qq: String, cookie: String): String {
        return apiClient.requestText(
            path = "/api/login/smslogin",
            method = "POST",
            body = apiClient.formBody(
                mapOf(
                    "qq" to qq,
                    "ck" to cookie,
                    "token" to "123456",
                )
            ),
        ).trim().ifBlank { "提交成功" }
    }

    suspend fun fetchPins(qq: String): List<String> {
        val text = apiClient.requestText(path = "/api/getUserPin?QQ=${apiClient.urlEncode(qq)}").trim()
        if (text.isBlank()) {
            throw IllegalStateException("接口返回为空，请稍后重试")
        }
        if (!text.startsWith("{")) {
            throw IllegalStateException("接口未返回有效 JSON：${text.take(80)}")
        }
        val json = JSONObject(text)
        if (json.optInt("code", 1) != 0) {
            val message = json.optString("message").ifBlank { json.optString("msg") }.ifBlank { "未查询到有效 PIN 列表" }
            throw IllegalStateException(message)
        }
        val array = json.optJSONArray("data") ?: return emptyList()
        return buildList {
            for (i in 0 until array.length()) add(array.optString(i))
        }.filter { it.isNotBlank() }
    }

    suspend fun fetchUserInfo(pin: String): String {
        val text = apiClient.requestText(path = "/api/getUserInfo?pin=${apiClient.urlEncode(pin)}").trim()
        if (text.isBlank()) {
            return "接口返回为空"
        }
        if (!text.startsWith("{")) {
            return text
        }
        val json = JSONObject(text)
        if (json.optInt("code", 1) != 0) {
            return json.optString("message").ifBlank { json.optString("msg") }.ifBlank { text }
        }
        val data = json.opt("data")
        return when (data) {
            is JSONObject -> data.keys().asSequence().sorted().joinToString("\n") { key -> "$key: ${data.optString(key)}" }
            is String -> data
            null -> "暂无数据"
            else -> gson.toJson(data)
        }
    }
}
