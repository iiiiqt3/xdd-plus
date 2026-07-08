package com.goudong.jd.push

import android.app.Activity
import android.content.Context
import android.content.Intent
import com.goudong.jd.AppServices
import com.goudong.jd.MainActivity
import com.goudong.jd.ui.auth.AuthActivity
import com.goudong.jd.ui.common.ResultTextActivity
import com.goudong.jd.ui.more.FeedbackActivity
import com.goudong.jd.ui.more.NotificationDetailActivity
import com.goudong.jd.ui.more.NotificationListActivity

object PushNavigationHelper {
    private const val PREFS_NAME = "push_navigation"
    private const val KEY_PENDING_NOTIFICATION_ID = "pending_notification_id"
    private const val KEY_PENDING_OPEN_LIST = "pending_open_list"
    private const val KEY_PENDING_ACTION = "pending_action"
    private const val KEY_PENDING_TITLE = "pending_title"
    private const val KEY_PENDING_BODY = "pending_body"

    const val ACTION_OPEN_WX = "open_wx"
    const val ACTION_OPEN_JD_YYB = "open_jd_yyb"
    const val ACTION_OPEN_PROJECTS = "open_projects"
    const val ACTION_OPEN_FEEDBACK = "open_feedback"
    const val ACTION_SHOW_BODY = "show_body"

    fun openFromExtras(
        context: Context,
        extras: String?,
        fallbackTitle: String = "",
        fallbackBody: String = "",
    ) {
        val notificationId = PushExtrasParser.parseNotificationId(extras)
        val ephemeral = PushExtrasParser.isEphemeral(extras) || notificationId <= 0
        if (ephemeral) {
            openEphemeral(
                context = context,
                action = PushExtrasParser.parseAction(extras),
                title = fallbackTitle.ifBlank { "狗东通知" },
                fullBody = PushExtrasParser.parseFullBody(extras).ifBlank { fallbackBody },
            )
            return
        }
        openNotification(context, notificationId, openList = false)
    }

    fun openNotification(context: Context, notificationId: Int = 0, openList: Boolean = false) {
        if (!AppServices.sessionManager.isAuthenticated()) {
            savePending(notificationId = notificationId, openList = openList)
            launchAuth(context)
            return
        }
        navigatePersistent(context, notificationId, openList)
    }

    fun openEphemeral(
        context: Context,
        action: String,
        title: String,
        fullBody: String,
    ) {
        val resolvedAction = action.trim().ifBlank { ACTION_SHOW_BODY }
        if (!AppServices.sessionManager.isAuthenticated()) {
            savePending(action = resolvedAction, title = title, body = fullBody)
            launchAuth(context)
            return
        }
        navigateEphemeral(context, resolvedAction, title, fullBody)
    }

    fun consumePendingAfterLogin(context: Context) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        val action = prefs.getString(KEY_PENDING_ACTION, "").orEmpty().trim()
        val title = prefs.getString(KEY_PENDING_TITLE, "").orEmpty()
        val body = prefs.getString(KEY_PENDING_BODY, "").orEmpty()
        val notificationId = prefs.getInt(KEY_PENDING_NOTIFICATION_ID, 0)
        val openList = prefs.getBoolean(KEY_PENDING_OPEN_LIST, false)
        if (action.isBlank() && notificationId <= 0 && !openList) return
        prefs.edit().clear().apply()
        if (action.isNotBlank()) {
            navigateEphemeral(context, action, title, body)
            return
        }
        navigatePersistent(context, notificationId, openList)
    }

    private fun savePending(
        notificationId: Int = 0,
        openList: Boolean = false,
        action: String = "",
        title: String = "",
        body: String = "",
    ) {
        AppServices.appContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .putInt(KEY_PENDING_NOTIFICATION_ID, notificationId.coerceAtLeast(0))
            .putBoolean(KEY_PENDING_OPEN_LIST, openList)
            .putString(KEY_PENDING_ACTION, action)
            .putString(KEY_PENDING_TITLE, title)
            .putString(KEY_PENDING_BODY, body)
            .apply()
    }

    private fun launchAuth(context: Context) {
        val intent = Intent(context, AuthActivity::class.java)
        if (context !is Activity) {
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        }
        context.startActivity(intent)
    }

    private fun navigatePersistent(context: Context, notificationId: Int, openList: Boolean) {
        val intent = when {
            notificationId > 0 -> Intent(context, NotificationDetailActivity::class.java).apply {
                putExtra(NotificationDetailActivity.EXTRA_NOTIFICATION_ID, notificationId)
            }
            openList -> Intent(context, NotificationListActivity::class.java)
            else -> Intent(context, MainActivity::class.java).apply {
                putExtra(MainActivity.EXTRA_OPEN_NOTIFICATIONS, true)
            }
        }
        start(context, intent)
    }

    private fun navigateEphemeral(context: Context, action: String, title: String, fullBody: String) {
        val intent = when (action) {
            ACTION_OPEN_WX -> mainIntent(context).apply {
                putExtra(MainActivity.EXTRA_PUSH_ACTION, action)
                putExtra(MainActivity.EXTRA_PROJECTS_INNER_TAB, MainActivity.PROJECTS_TAB_WX)
            }
            ACTION_OPEN_JD_YYB -> mainIntent(context).apply {
                putExtra(MainActivity.EXTRA_PUSH_ACTION, ACTION_OPEN_JD_YYB)
            }
            ACTION_OPEN_PROJECTS -> mainIntent(context).apply {
                putExtra(MainActivity.EXTRA_PUSH_ACTION, ACTION_OPEN_PROJECTS)
            }
            ACTION_OPEN_FEEDBACK -> Intent(context, FeedbackActivity::class.java)
            else -> ResultTextActivity.intent(
                context,
                title.ifBlank { "系统通知" },
                fullBody.ifBlank { title },
            )
        }
        start(context, intent)
    }

    private fun mainIntent(context: Context): Intent {
        return Intent(context, MainActivity::class.java)
    }

    private fun start(context: Context, intent: Intent) {
        if (context !is Activity) {
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        }
        context.startActivity(intent)
    }
}
