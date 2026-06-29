package com.goudong.jd.data.network

import com.goudong.jd.data.model.ApiEnvelope
import com.goudong.jd.data.model.ApiError
import com.goudong.jd.data.model.AppEnvironment
import com.goudong.jd.data.session.PersistentCookieJar
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import okhttp3.FormBody
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import java.net.URLEncoder
import java.util.Locale
import java.util.concurrent.TimeUnit

class ApiClient(
    private val cookieJar: PersistentCookieJar,
    val gson: Gson,
    private val appContext: android.content.Context? = null,
) {
    private val client: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .cookieJar(cookieJar)
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .connectionPool(okhttp3.ConnectionPool(5, 1, java.util.concurrent.TimeUnit.MINUTES))
            .build()
    }

    suspend fun requestText(
        path: String = "",
        absoluteUrl: String? = null,
        method: String = "GET",
        headers: Map<String, String> = emptyMap(),
        body: RequestBody? = null,
        skipAuthCheck: Boolean = false,
        skipSignature: Boolean = false,
    ): String = withContext(Dispatchers.IO) {
        val resolvedMethod = method.uppercase(Locale.ROOT)
        val safeBody = when {
            body != null -> body
            resolvedMethod == "POST" || resolvedMethod == "PUT" || resolvedMethod == "PATCH" -> "".toRequestBody(null)
            else -> null
        }

        val allHeaders = mutableMapOf<String, String>()
        allHeaders.putAll(headers)

        if (!skipSignature && appContext != null && needsSignature(path)) {
            val signHeaders = SignatureHelper.getHeaders(appContext, path)
            allHeaders.putAll(signHeaders)
        }

        val request = Request.Builder()
            .url(absoluteUrl ?: AppEnvironment.BASE_URL + path.removePrefix("/"))
            .method(resolvedMethod, safeBody)
            .apply { allHeaders.forEach { (key, value) -> addHeader(key, value) } }
            .build()

        client.newCall(request).execute().use { response ->
            val text = response.body?.string().orEmpty()
            val trimmed = text.trim()
            if (!skipAuthCheck && looksUnauthorized(trimmed)) {
                throw ApiError("登录状态失效，请重新登录", true)
            }
            if (!response.isSuccessful) {
                val unauthorized = response.code == 401 || response.code == 403
                throw ApiError("请求失败（${response.code}）", unauthorized)
            }
            return@withContext text
        }
    }

    private fun needsSignature(path: String): Boolean {
        val signaturePaths = listOf(
            "/api/portal/checkin",
            "/api/portal/pray"
        )
        return signaturePaths.any { path.contains(it) }
    }

    suspend inline fun <reified T> requestEnvelope(
        path: String,
        method: String = "GET",
        headers: Map<String, String> = emptyMap(),
        body: RequestBody? = null,
    ): ApiEnvelope<T> {
        val text = requestText(path = path, method = method, headers = headers, body = body)
        return parseEnvelope(text)
    }

    suspend inline fun <reified T> requestData(
        path: String,
        method: String = "GET",
        headers: Map<String, String> = emptyMap(),
        body: RequestBody? = null,
    ): T {
        val envelope = requestEnvelope<T>(path = path, method = method, headers = headers, body = body)
        if (envelope.code == 0 && envelope.data != null) {
            return envelope.data
        }
        val message = envelope.msg ?: "请求失败"
        val unauthorized = envelope.code == 401 || envelope.code == 403
        throw ApiError(message, unauthorized)
    }

    suspend fun requestMessage(
        path: String,
        method: String = "POST",
        headers: Map<String, String> = emptyMap(),
        body: RequestBody? = null,
    ): String {
        val envelope = requestEnvelope<Any>(path = path, method = method, headers = headers, body = body)
        if (envelope.code == 0) {
            return envelope.msg ?: "操作成功"
        }
        val message = envelope.msg ?: "请求失败"
        val unauthorized = envelope.code == 401 || envelope.code == 403
        throw ApiError(message, unauthorized)
    }

    inline fun <reified T> parseEnvelope(text: String): ApiEnvelope<T> {
        return try {
            val type = object : TypeToken<ApiEnvelope<T>>() {}.type
            gson.fromJson<ApiEnvelope<T>>(text, type)
        } catch (error: Exception) {
            throw ApiError("数据解析失败，请稍后重试")
        }
    }

    inline fun <reified T> parseListEnvelope(text: String, clazz: Class<T>): List<T> {
        return try {
            val jsonElement = gson.fromJson(text, com.google.gson.JsonObject::class.java)
            val code = jsonElement?.get("code")?.asInt ?: -1
            if (code != 0) {
                val msg = jsonElement?.get("msg")?.asString ?: "请求失败"
                throw ApiError(msg)
            }
            val dataElement = jsonElement?.get("data")
            if (dataElement == null || dataElement.isJsonNull) return emptyList()
            val dataArray = dataElement.asJsonArray
            dataArray.map { gson.fromJson(it, clazz) }
        } catch (e: ApiError) {
            throw e
        } catch (error: Exception) {
            throw ApiError("数据解析失败，请稍后重试")
        }
    }

    fun formBody(values: Map<String, String>): RequestBody {
        val builder = FormBody.Builder()
        values.forEach { (key, value) -> builder.add(key, value) }
        return builder.build()
    }

    fun jsonBody(values: Map<String, Any?>): RequestBody {
        return gson.toJson(values).toRequestBody("application/json; charset=utf-8".toMediaType())
    }

    fun multipartBody(values: Map<String, String>): RequestBody {
        val builder = MultipartBody.Builder().setType(MultipartBody.FORM)
        values.forEach { (key, value) -> builder.addFormDataPart(key, value) }
        return builder.build()
    }

    fun clearCookies() {
        cookieJar.clear()
    }

    fun urlEncode(value: String): String = URLEncoder.encode(value, Charsets.UTF_8.name())

    suspend fun streamSse(
        path: String,
        onLine: (String) -> Unit,
        onDone: () -> Unit,
        onError: (Throwable) -> Unit,
    ) {
        val mainHandler = android.os.Handler(android.os.Looper.getMainLooper())
        return kotlinx.coroutines.suspendCancellableCoroutine { cont ->
            val thread = Thread {
                val request = Request.Builder()
                    .url(AppEnvironment.BASE_URL + path.removePrefix("/"))
                    .get()
                    .build()
                try {
                    client.newCall(request).execute().use { response ->
                        if (!response.isSuccessful) {
                            mainHandler.post { onError(ApiError("日志连接失败（${response.code}）")) }
                            if (cont.isActive) cont.resumeWith(Result.success(Unit))
                            return@Thread
                        }
                        val reader = response.body?.charStream() ?: throw ApiError("无响应体")
                        reader.buffered().useLines { lines ->
                            for (line in lines) {
                                when {
                                    line.startsWith("data: ") -> {
                                        val data = line.removePrefix("data: ").trim()
                                        if (data.contains("任务执行完成") || data.contains("DONE")) {
                                            mainHandler.post { onDone() }
                                            if (cont.isActive) cont.resumeWith(Result.success(Unit))
                                            return@Thread
                                        }
                                        mainHandler.post { onLine(data) }
                                    }
                                    line.startsWith("event: done") -> {
                                        mainHandler.post { onDone() }
                                        if (cont.isActive) cont.resumeWith(Result.success(Unit))
                                        return@Thread
                                    }
                                    line.startsWith("event: error") -> {
                                    }
                                }
                            }
                        }
                        mainHandler.post { onDone() }
                        if (cont.isActive) cont.resumeWith(Result.success(Unit))
                    }
                } catch (error: Exception) {
                    mainHandler.post { onError(error) }
                    if (cont.isActive) cont.resumeWith(Result.success(Unit))
                }
            }
            cont.invokeOnCancellation { thread.interrupt() }
            thread.start()
        }
    }

    private fun looksUnauthorized(text: String): Boolean {
        return text.startsWith("<!DOCTYPE html", ignoreCase = true) ||
            text.startsWith("<html", ignoreCase = true) ||
            text.contains("/portal/login") ||
            text.contains("用户中心登录")
    }
}
