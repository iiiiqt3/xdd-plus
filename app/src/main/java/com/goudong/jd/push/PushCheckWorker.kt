package com.goudong.jd.push

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.goudong.jd.AppServices
import com.goudong.jd.data.model.PortalNotification
import kotlinx.coroutines.delay
import java.util.concurrent.ConcurrentHashMap

class PushCheckWorker(
    context: Context,
    params: WorkerParameters
) : CoroutineWorker(context, params) {

    companion object {
        private const val TAG = "PushCheckWorker"
        private const val PREFS_NAME = "push_notifications"
        private const val KEY_NOTIFIED_IDS = "notified_ids"
        private const val KEY_LAST_CHECKED_ID = "last_checked_id"
        
        private val memoryCache = ConcurrentHashMap.newKeySet<Int>()
        private var lastCheckedId: Int = 0
            @Synchronized get
            @Synchronized set
        
        var isAppInForeground = false
            @Synchronized set
        
        fun getNotifiedIds(context: Context): Set<Int> {
            if (memoryCache.isEmpty()) {
                val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
                val idsStr = prefs.getString(KEY_NOTIFIED_IDS, "") ?: ""
                if (idsStr.isNotEmpty()) {
                    idsStr.split(",").filter { it.isNotEmpty() }.mapNotNull { it.toIntOrNull() }.forEach { memoryCache.add(it) }
                }
                lastCheckedId = prefs.getInt(KEY_LAST_CHECKED_ID, 0)
            }
            return memoryCache.toSet()
        }
        
        fun addNotifiedId(context: Context, id: Int) {
            memoryCache.add(id)
            if (id > lastCheckedId) {
                lastCheckedId = id
            }
            saveState(context)
        }
        
        fun addNotifiedIds(context: Context, ids: List<Int>) {
            ids.forEach { 
                memoryCache.add(it)
                if (it > lastCheckedId) {
                    lastCheckedId = it
                }
            }
            saveState(context)
        }
        
        private fun saveState(context: Context) {
            val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            prefs.edit()
                .putString(KEY_NOTIFIED_IDS, memoryCache.joinToString(","))
                .putInt(KEY_LAST_CHECKED_ID, lastCheckedId)
                .apply()
        }
        
        fun clearCache() {
            memoryCache.clear()
        }
        
        fun clearAll(context: Context) {
            memoryCache.clear()
            context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE).edit().clear().apply()
        }
    }

    override suspend fun doWork(): Result {
        return runCatching {
            if (!AppServices.sessionManager.isAuthenticated()) {
                Log.d(TAG, "用户未登录，跳过推送检查")
                return@runCatching Result.success()
            }

            if (isAppInForeground) {
                Log.d(TAG, "App在前台，跳过系统通知（由首页刷新处理）")
                return@runCatching Result.success()
            }

            delay(2000)

            val response = AppServices.portalRepository.fetchNotifications(includeContent = true)
            
            if (response.list.isEmpty()) {
                Log.d(TAG, "无任何消息")
                return@runCatching Result.success()
            }
            
            val notifiedIds = getNotifiedIds(applicationContext)
            
            val maxId = response.list.maxOfOrNull { it.id } ?: 0
            val newMessages = response.list.filter { it.id > lastCheckedId && it.id !in notifiedIds }
            
            if (newMessages.isNotEmpty()) {
                val newIds = newMessages.map { it.id }
                addNotifiedIds(applicationContext, newIds)
                
                val allUnread = response.list.filter { !it.isRead }
                val latestForDisplay = allUnread
                    .sortedByDescending { it.id }
                    .take(5)
                    .ifEmpty { 
                        newMessages.sortedByDescending { it.id }.take(5) 
                    }
                
                NotificationHelper.showUnreadSummaryNotification(
                    applicationContext,
                    allUnread.size.coerceAtLeast(newMessages.size),
                    latestForDisplay
                )
                
                Log.d(TAG, "✅ 发现 ${newMessages.size} 条新消息（ID范围：${newIds.min()} - ${newIds.max()}），已发送通知。当前最大ID: $maxId")
                Log.d(TAG, "   新消息标题: ${newMessages.mapNotNull { it.title }.take(3).joinToString(", ")}")
            } else {
                Log.d(TAG, "无新消息。当前最大ID: $maxId，上次检查ID: $lastCheckedId")
                
                val unreadCount = response.list.count { !it.isRead }
                if (unreadCount > 0 && notifiedIds.isNotEmpty()) {
                    val unreadNotNotified = response.list.filter { !it.isRead && it.id !in notifiedIds }
                    if (unreadNotNotified.isNotEmpty()) {
                        val ids = unreadNotNotified.map { it.id }
                        addNotifiedIds(applicationContext, ids)
                        
                        val latestUnread = response.list
                            .filter { !it.isRead }
                            .sortedByDescending { it.id }
                            .take(5)
                        
                        NotificationHelper.showUnreadSummaryNotification(
                            applicationContext,
                            unreadCount,
                            latestUnread
                        )
                        
                        Log.d(TAG, "发现 ${ids.size} 条未读但未推送过的消息，补充通知")
                    }
                }
            }

            Result.success()
        }.getOrElse { error ->
            Log.e(TAG, "推送检查失败: ${error.message}", error)
            Result.retry()
        }
    }
}