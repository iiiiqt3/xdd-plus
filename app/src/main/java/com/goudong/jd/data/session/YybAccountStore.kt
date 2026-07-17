package com.goudong.jd.data.session

import com.goudong.jd.AppServices
import com.goudong.jd.data.model.PortalYybAccount
import com.goudong.jd.data.model.PortalYybCheckSummary
import com.goudong.jd.data.model.PortalYybStatus
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch
import java.util.concurrent.CopyOnWriteArrayList

/** 应用宝账号缓存：本次 App 会话内只自动检测一次 */
object YybAccountStore {
    var status: PortalYybStatus? = null
        private set

    @Volatile
    var isLoading: Boolean = false
        private set

    /** 本次打开 App 后是否已做过自动刷新检测 */
    var sessionAutoChecked: Boolean = false
        private set

    private var pendingAlertSummary: String? = null
    private val listeners = CopyOnWriteArrayList<() -> Unit>()

    val accounts: List<PortalYybAccount>
        get() = status?.accounts.orEmpty()

    val isServiceReady: Boolean
        get() = status?.enabled == true && status?.ready == true

    fun addListener(listener: () -> Unit) {
        listeners.add(listener)
    }

    fun removeListener(listener: () -> Unit) {
        listeners.remove(listener)
    }

    private fun notifyChanged() {
        listeners.forEach { it.invoke() }
    }

    fun clear() {
        status = null
        sessionAutoChecked = false
        isLoading = false
        pendingAlertSummary = null
        notifyChanged()
    }

    fun consumePendingAlert(): String? {
        val msg = pendingAlertSummary
        pendingAlertSummary = null
        return msg
    }

    fun prefetchIfNeeded(scope: CoroutineScope, autoCheck: Boolean = false) {
        if (!AppServices.sessionManager.isAuthenticated()) return
        if (sessionAutoChecked && status != null) {
            notifyChanged()
            return
        }
        if (isLoading) return
        reload(scope, autoCheck = autoCheck, showAlert = false, force = false)
    }

    fun reload(
        scope: CoroutineScope,
        autoCheck: Boolean,
        showAlert: Boolean,
        force: Boolean = true,
        onComplete: ((Result<PortalYybStatus>) -> Unit)? = null,
    ) {
        if (!AppServices.sessionManager.isAuthenticated()) return
        if (isLoading) return
        if (!force && sessionAutoChecked && status != null) {
            onComplete?.invoke(Result.success(status!!))
            notifyChanged()
            return
        }
        isLoading = true
        notifyChanged()
        scope.launch {
            val result = runCatching { AppServices.portalRepository.fetchYybStatus(autoCheck) }
            isLoading = false
            result.onSuccess { st ->
                status = st
                sessionAutoChecked = true
                if (showAlert && autoCheck) {
                    pendingAlertSummary = formatCheckSummary(st.checkSummary)
                }
            }
            notifyChanged()
            onComplete?.invoke(result)
        }
    }

    private fun formatCheckSummary(summary: PortalYybCheckSummary?): String? {
        if (summary == null || summary.total <= 0) return null
        var msg = "检测完成：共 ${summary.total} 个，可用 ${summary.alive} 个"
        if (summary.dead > 0) msg += "，失效 ${summary.dead} 个"
        if (summary.failed > 0) msg += "，失败 ${summary.failed} 个"
        return msg
    }
}
