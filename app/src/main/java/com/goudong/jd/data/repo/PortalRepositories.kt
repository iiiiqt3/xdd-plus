package com.goudong.jd.data.repo

import com.goudong.jd.data.model.ApiEnvelope
import com.goudong.jd.data.model.ApiError
import com.goudong.jd.data.model.KuwoCredentials
import com.goudong.jd.data.model.KuwoScheduleResult
import com.goudong.jd.data.model.KuwoWithdrawTask
import com.goudong.jd.data.model.PortalDashboard
import com.goudong.jd.data.model.PortalHomeSnapshot
import com.goudong.jd.data.model.PortalNotificationPage
import com.goudong.jd.data.model.PortalWechatActionResult
import com.goudong.jd.data.model.ResetInfoPayload
import com.goudong.jd.data.model.SubmitFeedbackPayload
import com.goudong.jd.data.network.ApiClient
import com.goudong.jd.data.session.SessionManager
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

class AuthRepository(
    private val apiClient: ApiClient,
    private val sessionManager: SessionManager,
) {
    suspend fun login(username: String, password: String) {
        val body = apiClient.formBody(
            mapOf(
                "type" to "user",
                "account" to username,
                "pin" to password,
            )
        )
        val text = apiClient.requestText(
            path = "/api/login/admin",
            method = "POST",
            headers = mapOf("Content-Type" to "application/x-www-form-urlencoded"),
            body = body,
            skipAuthCheck = true,
        ).trim()
        if (text == "登录") {
            sessionManager.saveCredentials(username, password)
            sessionManager.setAuthenticated(true)
            return
        }
        throw parseLoginFailure(text)
    }

    private fun parseLoginFailure(text: String): ApiError {
        return runCatching {
            val json = apiClient.gson.fromJson(text, com.google.gson.JsonObject::class.java)
            val msg = json?.get("msg")?.asString ?: "登录失败，请重新输入账号密码"
            ApiError(msg, unauthorized = true)
        }.getOrElse {
            ApiError(
                if (text.contains("用户中心登录") || text.contains("/portal/login")) {
                    "登录状态失效，请重新登录"
                } else {
                    "登录失败，请重新输入账号密码"
                },
                unauthorized = true,
            )
        }
    }

    suspend fun register(username: String, password: String, bindCode: String): String {
        val message = apiClient.requestMessage(
            path = "/api/login/register",
            method = "POST",
            headers = mapOf("Content-Type" to "application/x-www-form-urlencoded"),
            body = apiClient.formBody(
                mapOf(
                    "username" to username,
                    "password" to password,
                    "bindCode" to bindCode,
                )
            ),
        )
        sessionManager.saveCredentials(username, password)
        sessionManager.setAuthenticated(true)
        return message
    }

    suspend fun fetchResetInfo(code: String): String {
        val envelope = apiClient.requestEnvelope<ResetInfoPayload>(
            path = "/api/login/reset/info",
            method = "POST",
            headers = mapOf("Content-Type" to "application/x-www-form-urlencoded"),
            body = apiClient.formBody(mapOf("code" to code)),
        )
        if (envelope.code == 0) {
            return envelope.data?.username ?: ""
        }
        throw ApiError(envelope.msg ?: "验证码无效")
    }

    suspend fun resetPassword(code: String, username: String, password: String): String {
        return apiClient.requestMessage(
            path = "/api/login/reset/password",
            method = "POST",
            headers = mapOf("Content-Type" to "application/x-www-form-urlencoded"),
            body = apiClient.formBody(
                mapOf(
                    "code" to code,
                    "username" to username,
                    "password" to password,
                )
            ),
        )
    }

    suspend fun logout(): String {
        val message = apiClient.requestMessage(path = "/api/login/logout")
        sessionManager.clearCredentials()
        sessionManager.setAuthenticated(false)
        apiClient.clearCookies()
        return message
    }
}

class PortalRepository(
    private val apiClient: ApiClient,
    private val sessionManager: SessionManager,
) {
    suspend fun verifySession() {
        val envelope = apiClient.requestEnvelope<Any>("/api/portal/dashboard")
        if (envelope.code != 0) {
            val message = envelope.msg ?: "登录状态失效，请重新登录"
            throw ApiError(message, true)
        }
        sessionManager.setAuthenticated(true)
    }

    suspend fun fetchDashboard(): PortalDashboard {
        return apiClient.requestData<PortalDashboard>("/api/portal/dashboard")
    }

    suspend fun fetchHomeSnapshot(): PortalHomeSnapshot = coroutineScope {
        val dashboardDeferred = async { apiClient.requestData<com.goudong.jd.data.model.PortalDashboard>("/api/portal/dashboard") }
        val profileDeferred = async { apiClient.requestData<com.goudong.jd.data.model.PortalProfile>("/api/portal/profile") }
        val wechatDeferred = async { runCatching { apiClient.requestData<com.goudong.jd.data.model.PortalWechatStatus>("/api/portal/wx/status") }.getOrNull() }
        val notificationsDeferred = async { runCatching { fetchNotifications(includeContent = true).list.filter { it.isTop == true }.take(3) }.getOrDefault(emptyList()) }
        val snapshot = PortalHomeSnapshot(
            dashboard = dashboardDeferred.await(),
            profile = profileDeferred.await(),
            wechatStatus = wechatDeferred.await(),
            topNotifications = notificationsDeferred.await(),
        )
        sessionManager.setAuthenticated(true)
        snapshot
    }

    suspend fun fetchActivities(): List<com.goudong.jd.data.model.PortalActivity> {
        val text = apiClient.requestText(path = "/api/portal/activities")
        return apiClient.parseListEnvelope(text, com.goudong.jd.data.model.PortalActivity::class.java)
    }

    suspend fun fetchProjects(): List<com.goudong.jd.data.model.PortalProject> {
        val text = apiClient.requestText(path = "/api/portal/projects")
        return apiClient.parseListEnvelope(text, com.goudong.jd.data.model.PortalProject::class.java)
    }

    suspend fun createProject(activityId: String, inputs: Map<String, String>, remarks: String, months: Int): String {
        return apiClient.requestMessage(
            path = "/api/portal/project",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("activityId" to activityId, "inputs" to inputs, "remarks" to remarks, "months" to months)),
        )
    }

    suspend fun renewProject(activityId: String, remarks: String, months: Int): String {
        return apiClient.requestMessage(
            path = "/api/portal/project/renew",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("activityId" to activityId, "remarks" to remarks, "months" to months)),
        )
    }

    suspend fun deleteProject(activityId: String, remarks: String): String {
        return apiClient.requestMessage(
            path = "/api/portal/project/delete",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("activityId" to activityId, "remarks" to remarks)),
        )
    }

    suspend fun updateProject(activityId: String, remarks: String, ckValue: String): String {
        return apiClient.requestMessage(
            path = "/api/portal/project/update",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("activityId" to activityId, "remarks" to remarks, "ckValue" to ckValue)),
        )
    }

    suspend fun queryIncome(activityId: String, remarks: String): String {
        val envelope = apiClient.requestEnvelope<String>(
            path = "/api/portal/project/income",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("activityId" to activityId, "remarks" to remarks)),
        )
        if (envelope.code == 0) {
            return envelope.data ?: envelope.msg ?: "暂无查询结果"
        }
        throw ApiError(envelope.msg ?: "查询失败")
    }

    suspend fun redeemKey(token: String): String {
        return apiClient.requestMessage(
            path = "/api/portal/redeem-key",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("token" to token)),
        )
    }

    suspend fun fetchWechatStatus() = apiClient.requestData<com.goudong.jd.data.model.PortalWechatStatus>("/api/portal/wx/status")

    suspend fun performWechatAction(path: String): PortalWechatActionResult {
        return apiClient.requestData(path = path, method = "POST")
    }

    suspend fun performWechatAction(path: String, wxid: String): PortalWechatActionResult {
        return apiClient.requestData(
            path = path,
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("wxid" to wxid)),
        )
    }

    suspend fun fetchWxDevices() = apiClient.requestData<List<com.goudong.jd.data.model.PortalWxDevice>>("/api/portal/wx/devices")

    suspend fun addWxDevice(wxid: String): String {
        return apiClient.requestMessage(
            path = "/api/portal/wx/add-device",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("wxid" to wxid)),
        )
    }

    suspend fun removeWxDevice(id: Int): String {
        return apiClient.requestMessage(
            path = "/api/portal/wx/remove-device",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("id" to id)),
        )
    }

    suspend fun pollWechatLogin(uuid: String, deductCoin: Boolean): PortalWechatActionResult {
        return apiClient.requestData(
            path = "/api/portal/wx/poll-login",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("uuid" to uuid, "deductCoin" to deductCoin)),
        )
    }

    suspend fun checkin(): String = apiClient.requestMessage(path = "/api/portal/checkin")

    suspend fun pray(): String = apiClient.requestMessage(path = "/api/portal/pray")

    suspend fun fetchCoinLogs(source: String? = null): List<com.goudong.jd.data.model.CoinLog> {
        val param = if (!source.isNullOrEmpty()) "?source=$source" else ""
        val text = apiClient.requestText(path = "/api/portal/coin-logs$param")
        return apiClient.parseListEnvelope(text, com.goudong.jd.data.model.CoinLog::class.java)
    }

    suspend fun fetchNotifications(includeContent: Boolean = false): PortalNotificationPage {
        val includeParam = if (includeContent) "&includeContent=1" else ""
        return apiClient.requestData("/api/portal/notifications?${includeParam}")
    }

    suspend fun markNotificationRead(notificationId: Int): com.goudong.jd.data.model.PortalNotification {
        return apiClient.requestData("/api/portal/notification?id=$notificationId")
    }

    suspend fun registerPushDevice(registrationId: String, alias: String) {
        apiClient.requestMessage(
            path = "/api/portal/push/register",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(
                mapOf(
                    "registrationId" to registrationId,
                    "alias" to alias,
                    "platform" to ApiClient.CLIENT_PLATFORM,
                )
            ),
        )
    }

    suspend fun submitFeedback(payload: SubmitFeedbackPayload): String {
        return apiClient.requestMessage(
            path = "/api/portal/feedback",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf(
                "type" to payload.type,
                "title" to payload.title,
                "content" to payload.content,
                "contact" to payload.contact,
            )),
        )
    }

    suspend fun fetchJdAccounts(): List<com.goudong.jd.data.model.PortalJdAccount> {
        val text = apiClient.requestText(path = "/api/portal/jd/accounts")
        return apiClient.parseListEnvelope(text, com.goudong.jd.data.model.PortalJdAccount::class.java)
    }

    suspend fun queryJdAccount(index: Int): String {
        return apiClient.requestData(
            path = "/api/portal/jd/query",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("index" to index)),
        )
    }

    suspend fun sendJdSms(phone: String): String {
        return apiClient.requestMessage(
            path = "/api/portal/jd/sms/send",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("phone" to phone)),
        )
    }

    suspend fun verifyJdSms(phone: String, code: String, idCard: String): com.goudong.jd.data.model.PortalJdSmsVerifyResult {
        return apiClient.requestData(
            path = "/api/portal/jd/sms/verify",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("phone" to phone, "code" to code, "idCard" to idCard)),
        )
    }

    suspend fun fetchJdWxDevices(): List<com.goudong.jd.data.model.PortalJdWxDevice> {
        val text = apiClient.requestText(path = "/api/portal/jd/wx/devices")
        return apiClient.parseListEnvelope(text, com.goudong.jd.data.model.PortalJdWxDevice::class.java)
    }

    suspend fun refreshJdWx(wxid: String, riskConfirmed: Boolean = false): com.goudong.jd.data.model.PortalJdWxRefreshResult {
        return apiClient.requestData(
            path = "/api/portal/jd/wx/refresh",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("wxid" to wxid, "riskConfirmed" to riskConfirmed)),
        )
    }

    suspend fun continueJdWxRisk(): com.goudong.jd.data.model.PortalJdWxRefreshResult {
        return apiClient.requestData(path = "/api/portal/jd/wx/continue-risk", method = "POST")
    }

    suspend fun executeJdTask(taskId: String, taskName: String, accountIndexes: List<Int>): com.goudong.jd.data.model.PortalJdTaskExecuteResult {
        return apiClient.requestData(
            path = "/api/portal/jd/task/execute",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(
                mapOf(
                    "taskId" to taskId,
                    "taskName" to taskName,
                    "accountIndexes" to accountIndexes,
                )
            ),
        )
    }

    suspend fun stopJdTask(taskId: String) {
        apiClient.requestMessage(
            path = "/api/portal/jd/task/stop",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("taskId" to taskId)),
        )
    }

    suspend fun streamJdTaskLogs(
        taskId: String,
        onLine: (String) -> Unit,
        onDone: () -> Unit,
        onError: (Throwable) -> Unit,
    ) {
        apiClient.streamSse("/api/portal/jd/task/logs?taskId=${apiClient.urlEncode(taskId)}", onLine, onDone, onError)
    }

    suspend fun checkKuwoAuth(): Pair<Boolean, String> {
        val text = apiClient.requestText(path = "/api/portal/kuwo/check-auth")
        val json = apiClient.gson.fromJson(text, com.google.gson.JsonObject::class.java)
        val code = json?.get("code")?.asInt ?: 1
        if (code != 0) throw ApiError(json?.get("msg")?.asString ?: "检查授权失败")
        return (json.get("authorized")?.asBoolean ?: false) to (json.get("msg")?.asString ?: "")
    }

    suspend fun fetchKuwoCredentials(): KuwoCredentials {
        return apiClient.requestData(path = "/api/portal/kuwo/credentials")
    }

    suspend fun sendKuwoSms(phone: String, password: String) {
        apiClient.requestMessage(
            path = "/api/portal/kuwo/send-sms",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("phone" to phone, "password" to password)),
        )
    }

    suspend fun scheduleKuwoWithdraw(
        phone: String,
        password: String,
        quotaId: String,
        smsCode: String,
        targetHour: Int?,
        immediate: Boolean,
    ): KuwoScheduleResult {
        val payload = mutableMapOf<String, Any>(
            "sessions" to listOf(mapOf("phone" to phone, "password" to password)),
            "quotaId" to quotaId,
            "smsCode" to smsCode,
            "immediate" to immediate,
        )
        if (!immediate && targetHour != null) {
            payload["targetHour"] = targetHour
        }
        return apiClient.requestData(
            path = "/api/portal/kuwo/schedule-withdraw",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(payload),
        )
    }

    suspend fun updateKuwoSmsCode(taskId: String, smsCode: String) {
        apiClient.requestMessage(
            path = "/api/portal/kuwo/update-sms-code",
            method = "POST",
            headers = mapOf("Content-Type" to "application/json"),
            body = apiClient.jsonBody(mapOf("taskId" to taskId, "smsCode" to smsCode)),
        )
    }

    suspend fun fetchKuwoWithdrawStatus(taskId: String? = null, phone: String? = null): KuwoWithdrawTask? {
        val path = when {
            !taskId.isNullOrBlank() -> "/api/portal/kuwo/withdraw-status?taskId=${apiClient.urlEncode(taskId)}"
            !phone.isNullOrBlank() -> "/api/portal/kuwo/withdraw-status?phone=${apiClient.urlEncode(phone)}"
            else -> "/api/portal/kuwo/withdraw-status"
        }
        val envelope = apiClient.requestEnvelope<com.google.gson.JsonElement>(path = path)
        if (envelope.code != 0) throw ApiError(envelope.msg ?: "查询任务失败")
        val data = envelope.data ?: return null
        if (data.isJsonNull) return null
        return apiClient.gson.fromJson(data, KuwoWithdrawTask::class.java)
    }
}
