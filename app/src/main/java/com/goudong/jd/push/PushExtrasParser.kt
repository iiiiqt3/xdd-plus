package com.goudong.jd.push

import org.json.JSONObject

object PushExtrasParser {
    fun parseNotificationId(extras: String?): Int {
        if (extras.isNullOrBlank()) return 0
        return parseFromJson(extras.trim())
            ?: parseFromMapString(extras)
            ?: 0
    }

    fun parseDisplayType(extras: String?): String {
        if (extras.isNullOrBlank()) return "normal"
        return parseStringFromJson(extras.trim(), "displayType") ?: "normal"
    }

    fun parseAction(extras: String?): String {
        if (extras.isNullOrBlank()) return ""
        return parseStringFromJson(extras.trim(), "action").orEmpty()
    }

    fun parseFullBody(extras: String?): String {
        if (extras.isNullOrBlank()) return ""
        return parseStringFromJson(extras.trim(), "fullBody").orEmpty()
    }

    fun isEphemeral(extras: String?): Boolean {
        if (extras.isNullOrBlank()) return false
        return runCatching {
            val json = JSONObject(extras.trim())
            json.optBoolean("ephemeral", false) ||
                findBooleanInJson(json, "ephemeral") == true
        }.getOrDefault(false)
    }

    private fun parseFromJson(raw: String): Int? {
        return runCatching {
            val json = JSONObject(raw)
            json.optInt("notificationId", 0).takeIf { it > 0 }
                ?: findIntInJson(json, "notificationId")
                ?: json.optString("cn.jpush.android.EXTRA").takeIf { it.isNotBlank() }?.let { parseFromJson(it) }
        }.getOrNull()
    }

    private fun parseStringFromJson(raw: String, key: String): String? {
        return runCatching {
            val json = JSONObject(raw)
            json.optString(key).takeIf { it.isNotBlank() }
                ?: findStringInJson(json, key)
                ?: json.optString("cn.jpush.android.EXTRA").takeIf { it.isNotBlank() }?.let { parseStringFromJson(it, key) }
        }.getOrNull()
    }

    private fun findIntInJson(json: JSONObject, key: String): Int? {
        val direct = json.optInt(key, 0)
        if (direct > 0) return direct
        val keys = json.keys()
        while (keys.hasNext()) {
            val childKey = keys.next()
            when (val value = json.opt(childKey)) {
                is JSONObject -> findIntInJson(value, key)?.let { return it }
                is String -> {
                    if (childKey.contains("EXTRA", ignoreCase = true)) {
                        parseFromJson(value)?.let { return it }
                    }
                }
            }
        }
        return null
    }

    private fun findStringInJson(json: JSONObject, key: String): String? {
        json.optString(key).takeIf { it.isNotBlank() }?.let { return it }
        val keys = json.keys()
        while (keys.hasNext()) {
            val childKey = keys.next()
            when (val value = json.opt(childKey)) {
                is JSONObject -> findStringInJson(value, key)?.let { return it }
                is String -> {
                    if (childKey.contains("EXTRA", ignoreCase = true)) {
                        parseStringFromJson(value, key)?.let { return it }
                    }
                }
            }
        }
        return null
    }

    private fun findBooleanInJson(json: JSONObject, key: String): Boolean? {
        if (json.has(key)) return json.optBoolean(key, false)
        val keys = json.keys()
        while (keys.hasNext()) {
            val childKey = keys.next()
            when (val value = json.opt(childKey)) {
                is JSONObject -> findBooleanInJson(value, key)?.let { return it }
                is String -> {
                    if (childKey.contains("EXTRA", ignoreCase = true)) {
                        return runCatching { JSONObject(value).optBoolean(key, false) }.getOrNull()
                    }
                }
            }
        }
        return null
    }

    private fun parseFromMapString(raw: String): Int? {
        val regex = Regex("""notificationId\s*[=:]\s*(\d+)""", RegexOption.IGNORE_CASE)
        return regex.find(raw)?.groupValues?.getOrNull(1)?.toIntOrNull()?.takeIf { it > 0 }
    }
}
