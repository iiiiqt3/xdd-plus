package com.goudong.jd.ui.home

import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.os.Bundle
import android.text.TextUtils
import android.util.TypedValue
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.ScrollView
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import kotlinx.coroutines.launch
import com.goudong.jd.AppServices
import com.goudong.jd.MainActivity
import com.goudong.jd.R
import com.goudong.jd.data.model.ApiError
import com.goudong.jd.data.model.PortalNotification
import com.goudong.jd.ui.auth.AuthActivity
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.heroCard
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.MainTabResettable
import com.goudong.jd.ui.common.findFirstScrollView
import com.goudong.jd.ui.common.wrapMainTabSwipe
import com.goudong.jd.push.NotificationHelper
import com.goudong.jd.push.PushCheckWorker
import com.goudong.jd.ui.more.NotificationListActivity
import android.os.Handler
import android.os.Looper
import com.goudong.jd.ui.common.themeColor

class HomeFragment : Fragment(), MainTabResettable {
    private lateinit var summaryText: TextView
    private lateinit var detailText: TextView
    private lateinit var coinText: TextView
    private lateinit var notificationSection: LinearLayout
    private lateinit var statsSection: LinearLayout
    private lateinit var swipeRefreshLayout: SwipeRefreshLayout
    private var contentScroll: ScrollView? = null
    
    private val handler = Handler(Looper.getMainLooper())
    private val pollRunnable = object : Runnable {
        override fun run() {
            checkNewMessages()
            handler.postDelayed(this, 30_000L)
        }
    }

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val (scroll, root) = requireContext().makeScrollContainer()
        contentScroll = scroll

        val swipeRefresh = SwipeRefreshLayout(requireContext()).apply {
            swipeRefreshLayout = this
            setColorSchemeColors(
                ContextCompat.getColor(context, R.color.brand_primary),
                ContextCompat.getColor(context, R.color.brand_secondary)
            )
            setOnRefreshListener {
                loadData(forceRefresh = true)
            }
            addView(scroll, android.view.ViewGroup.LayoutParams(
                android.view.ViewGroup.LayoutParams.MATCH_PARENT,
                android.view.ViewGroup.LayoutParams.MATCH_PARENT
            ))
        }

        val primaryColor = ContextCompat.getColor(requireContext(), R.color.brand_primary)

        val brandBlue = ContextCompat.getColor(requireContext(), R.color.brand_secondary)

        notificationSection = requireContext().cardView().apply {
            addView(requireContext().captionText("📢 通知中心"))
            addView(LinearLayout(requireContext()).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(0, requireContext().dp(6), 0, requireContext().dp(8))
                addView(TextView(context).apply {
                    text = "正在加载通知..."
                    setTextColor(requireContext().themeColor(R.color.text_hint))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                    layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                    id = View.generateViewId()
                })
                addView(TextView(context).apply {
                    text = "查看全部 ›"
                    setTextColor(brandBlue)
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                    setTypeface(typeface, Typeface.BOLD)
                    gravity = Gravity.END
                    foreground = context.obtainStyledAttributes(intArrayOf(android.R.attr.selectableItemBackground)).getDrawable(0)
                    setOnClickListener {
                        startActivity(Intent(requireContext(), NotificationListActivity::class.java))
                    }
                })
            })
        }
        root.addView(notificationSection)

        val profileCard = requireContext().cardView()
        profileCard.addView(requireContext().captionText("👤 账号信息").apply {
            textSize = 16f
            setTypeface(typeface, Typeface.BOLD)
        })
        summaryText = requireContext().bodyText("请稍候...").apply { textSize = 14f }
        detailText = requireContext().bodyText("正在加载首页数据")
        profileCard.addView(summaryText)
        profileCard.addView(detailText)
        root.addView(profileCard)

        statsSection = requireContext().cardView().apply {
            addView(requireContext().captionText("📊 项目统计"))

            coinText = TextView(context).apply {
                text = "当前积分：-"
                setTextColor(brandBlue)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
                setTypeface(typeface, Typeface.BOLD)
                setPadding(0, context.dp(8), 0, context.dp(4))
            }
            addView(coinText)

            val grid = LinearLayout(context).apply {
                orientation = LinearLayout.VERTICAL
            }

            val row1 = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
            row1.addView(statItem(context, "项目总数", "-", R.color.brand_primary))
            row1.addView(statItem(context, "已上车项目", "-", R.color.brand_green))
            grid.addView(row1)

            val row2 = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
            row2.addView(statItem(context, "有效CK", "-", R.color.brand_secondary))
            row2.addView(statItem(context, "即将过期", "-", R.color.brand_orange))
            grid.addView(row2)

            val row3 = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
            row3.addView(statItem(context, "已过期", "-", R.color.brand_red))
            grid.addView(row3)

            addView(grid)
        }
        root.addView(statsSection)

        return wrapMainTabSwipe(swipeRefresh)
    }

    override fun onResume() {
        super.onResume()
        handler.post(pollRunnable)
        
        if (summaryText.text.isNullOrBlank() || summaryText.text == "请稍候..." || summaryText.text == "登录已过期" || summaryText.text == "加载失败") {
            AppServices.sessionManager.loadHomeSummary()?.let {
                if (!it.detail.trimStart().startsWith("{")) {
                    summaryText.text = it.summary
                    detailText.text = it.detail
                    coinText.text = "当前积分：${it.coin}"
                }
            }
        }
        loadData(forceRefresh = false)
    }

    override fun onPause() {
        super.onPause()
        handler.removeCallbacks(pollRunnable)
    }

    private fun checkNewMessages() {
        if (!isAdded || !AppServices.sessionManager.isAuthenticated()) return
        
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchNotifications(includeContent = false) }
                .onSuccess { response ->
                    (activity as? MainActivity)?.updateMoreBadge(response.unread)
                }
        }
    }

    private fun statItem(ctx: android.content.Context, label: String, value: String, colorRes: Int): LinearLayout {
        return LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
            background = android.graphics.drawable.GradientDrawable().apply {
                setColor(ContextCompat.getColor(ctx, colorRes).let { c ->
                    Color.argb(10, Color.red(c), Color.green(c), Color.blue(c))
                })
                cornerRadius = ctx.dp(12).toFloat()
            }
            setPadding(ctx.dp(12), ctx.dp(12), ctx.dp(12), ctx.dp(12))
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                marginEnd = ctx.dp(6)
                bottomMargin = ctx.dp(6)
            }

            addView(TextView(ctx).apply {
                text = value
                setTextColor(ContextCompat.getColor(ctx, colorRes))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 20f)
                setTypeface(typeface, Typeface.BOLD)
                id = View.generateViewId()
            })

            addView(TextView(ctx).apply {
                text = label
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, ctx.dp(3), 0, 0)
            })
        }
    }

    fun loadData(forceRefresh: Boolean = false) {
        if (::swipeRefreshLayout.isInitialized) {
            swipeRefreshLayout.isRefreshing = true
        }
        lifecycleScope.launch {
            if (!forceRefresh) {
                AppServices.sessionManager.loadHomeSummary()?.let {
                    summaryText.text = it.summary
                    detailText.text = it.detail
                    coinText.text = "当前积分：${it.coin}"
                }
            }
            runCatching { AppServices.portalRepository.fetchHomeSnapshot() }
                .onSuccess { snapshot ->
                    val wxStatus = snapshot.wechatStatus?.status ?: if (!snapshot.profile.user?.Wxid.isNullOrBlank()) "已绑定" else "未绑定"
                    summaryText.text = buildString {
                        append("编号：${snapshot.dashboard.number}")
                        if (!snapshot.dashboard.nickname.isNullOrBlank()) {
                            append("  ·  ${snapshot.dashboard.nickname}")
                        }
                    }
                    detailText.text = buildString {
                        appendLine("积分：${snapshot.dashboard.coin}")
                        appendLine("登录：${snapshot.dashboard.lastLoginAt ?: "-"}")
                        appendLine("微信：$wxStatus")
                    }.trim()
                    coinText.text = "当前积分：${snapshot.dashboard.coin}"

                    updateStat(statsSection, 0, "${snapshot.dashboard.availableCount}")
                    updateStat(statsSection, 1, "${snapshot.dashboard.joinedCount}")
                    updateStat(statsSection, 2, "${snapshot.dashboard.validCkCount}")
                    updateStat(statsSection, 3, "${snapshot.dashboard.expiringCount}")
                    updateStat(statsSection, 4, "${snapshot.dashboard.expiredCount}")

                    if (snapshot.notifications != null) {
                        (activity as? MainActivity)?.updateMoreBadge(snapshot.notifications!!.unread)
                    } else {
                        lifecycleScope.launch {
                            runCatching { AppServices.portalRepository.fetchNotifications(includeContent = false) }
                                .onSuccess { response ->
                                    (activity as? MainActivity)?.updateMoreBadge(response.unread)
                                }
                        }
                    }

                    snapshot.topNotifications?.takeIf { it.isNotEmpty() }?.let { notifications ->
                        renderTopNotifications(notifications)
                    } ?: run {
                        clearNotificationPlaceholders()
                        addNoNotificationHint()
                    }
                    
                    lifecycleScope.launch {
                        runCatching { 
                            val allNotifications = AppServices.portalRepository.fetchNotifications(includeContent = true)
                            checkAndNotifyNew(allNotifications.list)
                        }.onFailure { error ->
                            android.util.Log.e("HomeFragment", "推送检测失败: ${error.message}", error)
                        }
                    }

                    AppServices.sessionManager.saveHomeSummary(
                        summaryText.text.toString(),
                        detailText.text.toString().trim(),
                        snapshot.dashboard.coin.toString(),
                        snapshot.dashboard.validCkCount.toString(),
                        snapshot.dashboard.expiringCount.toString(),
                        wxStatus
                    )
                }
                .onFailure { error ->
                    val apiError = error as? ApiError
                    if (apiError?.unauthorized == true) {
                        if (AppServices.isAuthInProgress) {
                            summaryText.text = "正在重新登录..."
                            detailText.text = "请稍候"
                        } else {
                            summaryText.text = "登录已过期"
                            detailText.text = "正在跳转登录页..."
                            (activity as? MainActivity)?.handleUnauthorized()
                        }
                    } else {
                        summaryText.text = "加载失败"
                        detailText.text = com.goudong.jd.ui.common.sanitizeErrorMessage(error.message)
                        coinText.text = "当前积分：-"
                    }
                }
            if (::swipeRefreshLayout.isInitialized) {
                swipeRefreshLayout.isRefreshing = false
            }
        }
    }

    private fun updateStat(parent: LinearLayout, index: Int, value: String) {
        val grid = parent.getChildAt(2) as? LinearLayout ?: return
        val rowIndex = index / 2
        val colIndex = index % 2
        val row = grid.getChildAt(rowIndex) as? LinearLayout ?: return
        val item = row.getChildAt(colIndex) as? LinearLayout ?: return
        (item.getChildAt(0) as? TextView)?.text = value
    }

    private fun renderTopNotifications(notifications: List<PortalNotification>) {
        clearNotificationPlaceholders()
        val ctx = requireContext()

        if (notifications.size > 1) {
            notificationSection.addView(TextView(ctx).apply {
                text = "共 ${notifications.size} 条通知"
                setTextColor(requireContext().themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, ctx.dp(4), 0, ctx.dp(6))
            })
        }

        val displayList = notifications.take(3)
        displayList.forEachIndexed { index, item ->
            val notifItem = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(0, ctx.dp(6), 0, ctx.dp(6))

                addView(TextView(context).apply {
                    text = "${index + 1}."
                    setTextColor(ContextCompat.getColor(context, R.color.brand_primary))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                    setTypeface(typeface, Typeface.BOLD)
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        marginEnd = ctx.dp(6)
                    }
                })

                addView(TextView(context).apply {
                    val prefix = when {
                        item.isTop == true -> "📌 "
                        !item.isRead -> "🔵 "
                        else -> "📄 "
                    }
                    text = "$prefix${item.title ?: "无标题"}"
                    setTextColor(if (item.isTop == true || !item.isRead) requireContext().themeColor(R.color.text_primary) else requireContext().themeColor(R.color.text_secondary))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                    setTypeface(typeface, if (item.isTop == true) Typeface.BOLD else Typeface.NORMAL)
                    maxLines = 1
                    ellipsize = TextUtils.TruncateAt.END
                    layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                })

                addView(TextView(context).apply {
                    text = "›"
                    setTextColor(ContextCompat.getColor(context, R.color.brand_secondary))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
                    setTypeface(typeface, Typeface.BOLD)
                    gravity = Gravity.CENTER
                })

                foreground = context.obtainStyledAttributes(intArrayOf(android.R.attr.selectableItemBackground)).getDrawable(0)
                setOnClickListener {
                    startActivity(Intent(requireContext(), NotificationListActivity::class.java))
                }
            }
            notificationSection.addView(notifItem)
        }

        if (notifications.size > 3) {
            notificationSection.addView(TextView(ctx).apply {
                text = "还有 ${notifications.size - 3} 条..."
                setTextColor(requireContext().themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                gravity = Gravity.END
                setPadding(0, ctx.dp(4), 0, 0)
                foreground = context.obtainStyledAttributes(intArrayOf(android.R.attr.selectableItemBackground)).getDrawable(0)
                setOnClickListener {
                    startActivity(Intent(requireContext(), NotificationListActivity::class.java))
                }
            })
        }
    }

    private fun clearNotificationPlaceholders() {
        val count = notificationSection.childCount
        if (count > 1) {
            for (i in count - 1 downTo 1) {
                notificationSection.removeViewAt(i)
            }
        }
    }

    private fun addNoNotificationHint() {
        val ctx = requireContext()
        notificationSection.addView(ctx.captionText("暂无新通知").apply {
            setPadding(0, ctx.dp(4), 0, 0)
            gravity = Gravity.CENTER
        })
    }

    private fun checkAndNotifyNew(notifications: List<PortalNotification>) {
        if (!PushCheckWorker.isAppInForeground) return
        if (notifications.isEmpty()) return

        val notifiedIds = PushCheckWorker.getNotifiedIds(requireContext())
        
        val newMessages = notifications.filter { it.id !in notifiedIds }
        if (newMessages.isEmpty()) return

        val newIds = newMessages.map { it.id }
        PushCheckWorker.addNotifiedIds(requireContext(), newIds)
        
        val allUnread = notifications.filter { !it.isRead }
        val displayCount = allUnread.size.coerceAtLeast(newMessages.size)
        val latestForDisplay = allUnread
            .sortedByDescending { it.id }
            .take(5)
            .ifEmpty { 
                newMessages.sortedByDescending { it.id }.take(5) 
            }
        
        NotificationHelper.showUnreadSummaryNotification(
            requireContext(),
            displayCount,
            latestForDisplay
        )
        
        android.util.Log.d("HomeFragment", "前台检测到 ${newMessages.size} 条新消息，已发送通知")
    }

    override fun resetToInitialState() {
        contentScroll?.scrollTo(0, 0)
        view?.findFirstScrollView()?.scrollTo(0, 0)
        if (::swipeRefreshLayout.isInitialized) {
            swipeRefreshLayout.isRefreshing = false
        }
        if (AppServices.sessionManager.isAuthenticated()) {
            loadData(forceRefresh = false)
        }
    }
}
