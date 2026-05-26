package com.goudong.jd.data.session

import android.content.Context
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import okhttp3.Cookie
import okhttp3.CookieJar
import okhttp3.HttpUrl

class PersistentCookieJar(context: Context) : CookieJar {
    private val prefs = context.getSharedPreferences("android_jd_cookies", Context.MODE_PRIVATE)
    private val gson = Gson()
    private val lock = Any()
    private val cache = LinkedHashMap<String, MutableList<Cookie>>()

    init {
        restore()
    }

    override fun saveFromResponse(url: HttpUrl, cookies: List<Cookie>) {
        if (cookies.isEmpty()) return
        synchronized(lock) {
            val host = url.host
            val bucket = cache.getOrPut(host) { mutableListOf() }
            cookies.forEach { incoming ->
                bucket.removeAll { it.name == incoming.name && it.matches(url) }
                if (incoming.expiresAt > System.currentTimeMillis()) {
                    bucket.add(incoming)
                }
            }
            persist()
        }
    }

    override fun loadForRequest(url: HttpUrl): List<Cookie> {
        val now = System.currentTimeMillis()
        synchronized(lock) {
            val result = mutableListOf<Cookie>()
            var expired = false
            cache.values.forEach { list ->
                val iterator = list.iterator()
                while (iterator.hasNext()) {
                    val cookie = iterator.next()
                    if (cookie == null) {
                        iterator.remove()
                        continue
                    }
                    if (cookie.expiresAt <= now) {
                        iterator.remove()
                        expired = true
                    } else if (cookie.matches(url)) {
                        result.add(cookie)
                    }
                }
            }
            if (expired) persist()
            return result
        }
    }

    fun clear() {
        synchronized(lock) {
            cache.clear()
            prefs.edit().remove(KEY_COOKIES).apply()
        }
    }

    private fun restore() {
        val raw = prefs.getString(KEY_COOKIES, null) ?: return
        val persisted: List<PersistedCookie> = runCatching {
            gson.fromJson<List<PersistedCookie>>(raw, object : TypeToken<List<PersistedCookie>>() {}.type)
        }.getOrNull() ?: emptyList()
        synchronized(lock) {
            cache.clear()
            persisted.forEach { item ->
                val cookie = item.toCookie() ?: return@forEach
                cache.getOrPut(item.hostBucket) { mutableListOf() }.add(cookie)
            }
        }
    }

    private fun persist() {
        val persisted = cache.flatMap { (host, list) ->
            list.distinctBy { it.name + "@" + it.domain + ":" + it.path }.mapNotNull { PersistedCookie.from(host, it) }
        }
        prefs.edit().putString(KEY_COOKIES, gson.toJson(persisted)).apply()
    }

    private data class PersistedCookie(
        val hostBucket: String,
        val name: String,
        val value: String,
        val expiresAt: Long,
        val domain: String,
        val path: String,
        val secure: Boolean,
        val httpOnly: Boolean,
        val hostOnly: Boolean,
        val persistent: Boolean,
    ) {
        fun toCookie(): Cookie? {
            return runCatching {
                val builder = Cookie.Builder()
                    .name(name)
                    .value(value)
                    .path(path)
                if (hostOnly) builder.hostOnlyDomain(domain) else builder.domain(domain)
                if (persistent) builder.expiresAt(expiresAt)
                if (secure) builder.secure()
                if (httpOnly) builder.httpOnly()
                builder.build()
            }.getOrNull()
        }

        companion object {
            fun from(hostBucket: String, cookie: Cookie): PersistedCookie? {
                return runCatching {
                    PersistedCookie(
                        hostBucket = hostBucket,
                        name = cookie.name,
                        value = cookie.value,
                        expiresAt = cookie.expiresAt,
                        domain = cookie.domain,
                        path = cookie.path,
                        secure = cookie.secure,
                        httpOnly = cookie.httpOnly,
                        hostOnly = cookie.hostOnly,
                        persistent = cookie.persistent,
                    )
                }.getOrNull()
            }
        }
    }

    companion object {
        private const val KEY_COOKIES = "cookies"
    }
}
