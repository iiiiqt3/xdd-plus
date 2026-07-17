package com.goudong.jd.ui.projects

import android.app.AlertDialog
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.text.InputType
import android.util.TypedValue
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.FrameLayout
import android.widget.HorizontalScrollView
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.session.YybAccountStore
import com.goudong.jd.data.model.PortalActivity
import com.goudong.jd.data.model.PortalProject
import com.goudong.jd.ui.common.ResultTextActivity
import com.goudong.jd.ui.common.actionGridTile
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.badge
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.formatScanCostPreview
import com.goudong.jd.ui.common.formatYybConfirmMessage
import com.goudong.jd.ui.common.formatYybScanCostNote
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.heroCard
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.sanitizeErrorMessage
import com.goudong.jd.ui.common.toast
import com.goudong.jd.ui.common.applyCompactTabs
import com.goudong.jd.ui.common.InnerTabSwipeHost
import com.goudong.jd.ui.common.MainTabResettable
import com.goudong.jd.ui.common.findFirstScrollView
import com.goudong.jd.ui.common.wrapMainTabSwipe
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.primaryButton
import com.google.android.material.tabs.TabLayout
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.launch
import com.goudong.jd.ui.common.themeColor

class ProjectsFragment : Fragment(), InnerTabSwipeHost, MainTabResettable {
    private lateinit var contentRoot: LinearLayout
    private lateinit var tabs: TabLayout
    private lateinit var swipeRefreshLayout: SwipeRefreshLayout
    private var currentTab = 0
    private var loadedOnce = false
    private val expandedKeys = linkedSetOf<String>()
    private var refreshDeviceButton: Button? = null
    private var deviceLoadingIndicator: ProgressBar? = null
    private var searchBox: EditText? = null
    private var searchQuery: String = ""
    private var selectedCategory: String = ""
    private var categoryContainer: HorizontalScrollView? = null
    private var categoryChipRow: LinearLayout? = null
    private var rushHost: FrameLayout? = null
    private var contentScroll: android.widget.ScrollView? = null
    private val rushFragment = ProjectRushFragment()
    private val categories = listOf("全部", "现金类", "积分换实物", "抽奖类", "其他类")
    /** 协议接入子页：0=应用宝协议，1=微信协议，2=协议双绑 */
    private var protocolSubIndex = 0
    private var yybAccounts: List<com.goudong.jd.data.model.PortalYybAccount> = emptyList()
    private var yybSelectedKey: String = ""
    private var yybPolling = false
    private var yybSectionTitle: TextView? = null
    private var yybServiceDot: View? = null
    private var yybServiceLabel: TextView? = null
    private var yybLoadingBar: ProgressBar? = null
    private var yybManageRow: LinearLayout? = null
    private var yybRefreshAccountBtn: TextView? = null
    private var yybDeleteAccountBtn: TextView? = null
    private var yybReloadBtn: Button? = null
    private var protocolBindWxPickBtn: TextView? = null
    private var protocolBindYybPickBtn: TextView? = null
    private var protocolBindQuotaLabel: TextView? = null
    private var protocolBindListHost: LinearLayout? = null
    private var protocolBindSelectedWx: com.goudong.jd.data.model.PortalWxDevice? = null
    private var protocolBindSelectedYyb: com.goudong.jd.data.model.PortalYybAccount? = null
    private var yybServiceReady = false
    private var yybScanBusy = false
    private var yybManualRefreshing = false
    private var yybAccountActionBusy = false
    private var tabLoadGeneration = 0
    private val yybStoreListener = {
        if (yybManualRefreshing && !YybAccountStore.isLoading) {
            yybManualRefreshing = false
        }
        applyYybFromStore()
    }
    private var protocolBindings: List<com.goudong.jd.data.model.PortalProtocolBinding> = emptyList()
    private var protocolBindQuota: com.goudong.jd.data.model.PortalProtocolBindQuota? = null
    private var protocolProxyConfig: com.goudong.jd.data.model.PortalProxyConfig? = null

    // 缓存
    private var cachedActivities: List<PortalActivity>? = null
    private var cachedProjects: List<PortalProject>? = null

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        loadedOnce = false // 重新创建视图时重置，确保 onResume 能触发加载
        val wrapper = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        val (scroll, root) = requireContext().makeScrollContainer()
        contentScroll = scroll
        contentRoot = root
        swipeRefreshLayout = SwipeRefreshLayout(requireContext()).apply {
            setColorSchemeColors(ContextCompat.getColor(requireContext(), R.color.brand_primary))
            setOnRefreshListener {
                when (currentTab) {
                    0 -> cachedActivities = null
                    1 -> cachedProjects = null
                    3 -> when (protocolSubIndex) {
                        0 -> loadYybPanel(autoCheck = true, showAlert = false, manual = true)
                        1 -> loadWxDevices()
                        2 -> loadProtocolBindings()
                    }
                }
                if (currentTab == 3) {
                    swipeRefreshLayout.isRefreshing = false
                } else {
                    renderCurrentTab(forceRefresh = true)
                }
            }
            addView(scroll)
        }

        tabs = TabLayout(requireContext()).apply {
            setPadding(requireContext().dp(4), 0, requireContext().dp(4), requireContext().dp(8))
            addTab(newTab().setText("活动中心"))
            addTab(newTab().setText("我的项目"))
            addTab(newTab().setText("项目抢兑"))
            addTab(newTab().setText("协议接入"))
            applyCompactTabs()
            addOnTabSelectedListener(object : TabLayout.OnTabSelectedListener {
                override fun onTabSelected(tab: TabLayout.Tab) {
                    currentTab = tab.position
                    if (tab.position == 0) cachedActivities = null
                    if (tab.position == 1) cachedProjects = null
                    renderCurrentTab(forceRefresh = true)
                }
                override fun onTabUnselected(tab: TabLayout.Tab) = Unit
                override fun onTabReselected(tab: TabLayout.Tab) {
                    if (tab.position == currentTab) renderCurrentTab(forceRefresh = true)
                }
            })
        }
        wrapper.addView(tabs)

        searchBox = EditText(requireContext()).apply {
            hint = "搜索活动、项目、备注名…"
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
            setPadding(requireContext().dp(14), requireContext().dp(10), requireContext().dp(14), requireContext().dp(10))
            setBackgroundDrawable(GradientDrawable().apply {
                setColor(requireContext().themeColor(R.color.chip_bg))
                cornerRadius = requireContext().dp(10).toFloat()
            })
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginStart = requireContext().dp(14)
                marginEnd = requireContext().dp(14)
                bottomMargin = requireContext().dp(8)
            }
            maxLines = 1
            setCompoundDrawablesWithIntrinsicBounds(android.R.drawable.ic_menu_search, 0, 0, 0)
            addTextChangedListener(object : android.text.TextWatcher {
                override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
                override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
                override fun afterTextChanged(s: android.text.Editable?) {
                    searchQuery = s?.toString()?.trim() ?: ""
                    if (currentTab == 0 || currentTab == 1) {
                        renderCurrentTab(forceRefresh = false)
                    }
                }
            })
        }
        wrapper.addView(searchBox)

        // 分类筛选栏：等分屏幕宽度，避免末尾「其他类」被遮挡
        categoryChipRow = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(requireContext().dp(14), 0, requireContext().dp(14), requireContext().dp(8))
            visibility = View.GONE
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            )
        }
        categoryContainer = null
        renderCategoryChips()
        wrapper.addView(categoryChipRow)

        val contentLp = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, 0, 1f)
        swipeRefreshLayout.layoutParams = contentLp
        wrapper.addView(swipeRefreshLayout)

        rushHost = FrameLayout(requireContext()).apply {
            id = View.generateViewId()
            visibility = View.GONE
            layoutParams = contentLp
        }
        wrapper.addView(rushHost)
        return wrapMainTabSwipe(wrapper)
    }

    override fun onStart() {
        super.onStart()
        YybAccountStore.addListener(yybStoreListener)
    }

    override fun onStop() {
        YybAccountStore.removeListener(yybStoreListener)
        super.onStop()
    }

    override fun onResume() {
        super.onResume()
        when (currentTab) {
            0 -> cachedActivities = null
            1 -> cachedProjects = null
        }
        if (currentTab == 0 || currentTab == 1) {
            renderCurrentTab(forceRefresh = true)
            loadedOnce = true
        } else if (!loadedOnce) {
            renderCurrentTab(forceRefresh = false)
            loadedOnce = true
        }
    }

    fun refreshCurrentTab() {
        when (currentTab) {
            0 -> cachedActivities = null
            1 -> cachedProjects = null
            else -> Unit
        }
        renderCurrentTab(forceRefresh = true)
    }

    private fun renderCurrentTab(forceRefresh: Boolean) {
        tabLoadGeneration++
        val generation = tabLoadGeneration
        val isRushTab = currentTab == 2
        val hideSearchTab = currentTab == 2 || currentTab == 3
        searchBox?.visibility = if (hideSearchTab) View.GONE else View.VISIBLE
        swipeRefreshLayout.visibility = if (isRushTab) View.GONE else View.VISIBLE
        rushHost?.visibility = if (isRushTab) View.VISIBLE else View.GONE

        val isProtocolTab = currentTab == 3
        if (!isRushTab) {
            if (rushFragment.isAdded) rushFragment.popToList()
            contentRoot.removeAllViews()
            if (!swipeRefreshLayout.isRefreshing && forceRefresh && !isProtocolTab) {
                swipeRefreshLayout.isRefreshing = true
            }
        } else {
            swipeRefreshLayout.isRefreshing = false
        }
        if (isProtocolTab) {
            swipeRefreshLayout.isRefreshing = false
        }
        categoryChipRow?.visibility = if (currentTab == 0) View.VISIBLE else View.GONE
        when (currentTab) {
            0 -> loadActivities(forceRefresh, generation)
            1 -> loadProjects(forceRefresh, generation)
            2 -> renderProjectRush()
            3 -> renderProtocolAccess(forceRefresh)
        }
    }

    private fun renderProjectRush() {
        val host = rushHost ?: return
        if (!rushFragment.isAdded) {
            childFragmentManager.beginTransaction()
                .replace(host.id, rushFragment, "project_rush")
                .commitNowAllowingStateLoss()
        }
    }

    private fun showLoading() {
        contentRoot.addView(LinearLayout(requireContext()).apply {
            gravity = Gravity.CENTER
            orientation = LinearLayout.VERTICAL
            setPadding(0, requireContext().dp(48), 0, 0)
            addView(ProgressBar(requireContext()).apply {
                layoutParams = LinearLayout.LayoutParams(requireContext().dp(32), requireContext().dp(32))
            })
            addView(TextView(requireContext()).apply {
                text = "加载中..."
                setTextColor(requireContext().themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                gravity = Gravity.CENTER
                setPadding(0, requireContext().dp(10), 0, 0)
            })
        })
    }

    private fun renderCategoryChips() {
        val container = categoryChipRow ?: return
        container.removeAllViews()
        val brandBlue = ContextCompat.getColor(requireContext(), R.color.brand_primary)
        val gap = requireContext().dp(6)
        categories.forEachIndexed { index, cat ->
            val isSelected = (cat == "全部" && selectedCategory == "") || cat == selectedCategory
            val chip = TextView(requireContext()).apply {
                text = cat
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                maxLines = 1
                ellipsize = android.text.TextUtils.TruncateAt.END
                setTextColor(if (isSelected) requireContext().themeColor(R.color.chip_active_text) else requireContext().themeColor(R.color.text_secondary))
                background = GradientDrawable().apply {
                    setColor(if (isSelected) brandBlue else requireContext().themeColor(R.color.chip_bg))
                    cornerRadius = requireContext().dp(16).toFloat()
                }
                setPadding(requireContext().dp(4), requireContext().dp(7), requireContext().dp(4), requireContext().dp(7))
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                    if (index < categories.lastIndex) marginEnd = gap
                }
                setOnClickListener {
                    selectedCategory = if (cat == "全部") "" else cat
                    renderCategoryChips()
                    renderCurrentTab(forceRefresh = false)
                }
            }
            container.addView(chip)
        }
    }

    private fun loadActivities(forceRefresh: Boolean, generation: Int = tabLoadGeneration) {
        // 有缓存且不强制刷新，直接用缓存
        if (!forceRefresh && cachedActivities != null) {
            renderActivities(cachedActivities!!)
            swipeRefreshLayout.isRefreshing = false
            return
        }
        showLoading()
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchActivities() }
                .onSuccess { list ->
                    if (generation != tabLoadGeneration || currentTab != 0) return@launch
                    cachedActivities = list
                    contentRoot.removeAllViews()
                    renderActivities(list)
                    swipeRefreshLayout.isRefreshing = false
                }
                .onFailure {
                    if (generation != tabLoadGeneration || currentTab != 0) return@launch
                    contentRoot.removeAllViews()
                    handlePortalError(it)
                    contentRoot.addView(emptyCard(sanitizeErrorMessage(it.message)))
                    swipeRefreshLayout.isRefreshing = false
                }
        }
    }

    private fun renderActivities(list: List<PortalActivity>) {
        val filtered = list.filter { item ->
            // 分类筛选
            val matchCategory = selectedCategory.isEmpty() || item.category == selectedCategory
            // 搜索筛选
            val matchSearch = searchQuery.isEmpty() || run {
                val q = searchQuery.lowercase()
                (item.name ?: "").lowercase().contains(q) ||
                (item.envKey ?: "").lowercase().contains(q) ||
                (item.qingLongConfig ?: "").lowercase().contains(q)
            }
            matchCategory && matchSearch
        }
        if (filtered.isEmpty()) contentRoot.addView(emptyCard("暂无可用活动"))
        else filtered.forEach { contentRoot.addView(activityCard(it)) }
    }

    private fun loadProjects(forceRefresh: Boolean, generation: Int = tabLoadGeneration) {
        if (!forceRefresh && cachedProjects != null) {
            renderProjects(cachedProjects!!)
            swipeRefreshLayout.isRefreshing = false
            return
        }
        showLoading()
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchProjects() }
                .onSuccess { list ->
                    if (generation != tabLoadGeneration || currentTab != 1) return@launch
                    cachedProjects = list
                    contentRoot.removeAllViews()
                    renderProjects(list)
                    swipeRefreshLayout.isRefreshing = false
                }
                .onFailure {
                    if (generation != tabLoadGeneration || currentTab != 1) return@launch
                    contentRoot.removeAllViews()
                    handlePortalError(it)
                    contentRoot.addView(emptyCard(sanitizeErrorMessage(it.message)))
                    swipeRefreshLayout.isRefreshing = false
                }
        }
    }

    private fun renderProjects(list: List<PortalProject>) {
        val filtered = if (searchQuery.isEmpty()) list else list.filter {
            val q = searchQuery.lowercase()
            (it.activityName ?: "").lowercase().contains(q) ||
            (it.remark ?: "").lowercase().contains(q) ||
            (it.displayName ?: "").lowercase().contains(q) ||
            (it.envKey ?: "").lowercase().contains(q)
        }
        if (filtered.isEmpty()) {
            contentRoot.addView(emptyCard("暂无项目"))
        } else {
            filtered.groupBy { listOf(it.activityId ?: "", it.qingLongConfig ?: "默认容器").joinToString("__") }
                .toSortedMap()
                .forEach { (key, items) -> contentRoot.addView(groupCard(key, items)) }
        }
    }

    private fun activityCard(item: PortalActivity): View {
        val brandBlue = ContextCompat.getColor(requireContext(), R.color.brand_secondary)
        val daily = item.isDailyDeduct == true
        val monthly = item.isMonthlyDeduct == true
        return requireContext().cardView().apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(context.dp(16), context.dp(14), context.dp(14), context.dp(14))
            // 左侧信息
            val infoWrap = LinearLayout(context).apply {
                orientation = LinearLayout.VERTICAL
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            }
            infoWrap.addView(TextView(context).apply {
                text = item.name ?: "未命名项目"
                setTextColor(requireContext().themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
            })
            infoWrap.addView(TextView(context).apply {
                text = "青龙：${item.qingLongConfig ?: "默认容器"}"
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11.5f)
                setPadding(0, context.dp(4), 0, context.dp(6))
            })
            // 积分标签行
            val tagRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
            }
            // 扣费方式标签
            tagRow.addView(TextView(context).apply {
                text = when {
                    daily -> "按天授权"
                    monthly -> "按月授权"
                    else -> "一次性上车"
                }
                setTextColor(when {
                    daily -> requireContext().themeColor(R.color.positive)
                    monthly -> Color.parseColor("#7C3AED")
                    else -> Color.parseColor("#0369A1")
                })
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 10.5f)
                setTypeface(typeface, Typeface.BOLD)
                background = GradientDrawable().apply {
                    setColor(when {
                        daily -> Color.parseColor("#D1FAE5")
                        monthly -> Color.parseColor("#F3E8FF")
                        else -> Color.parseColor("#E0F2FE")
                    })
                    cornerRadius = context.dp(6).toFloat()
                }
                setPadding(context.dp(7), context.dp(2), context.dp(7), context.dp(2))
            })
            // 积分
            val coinText = when {
                daily -> "${item.dailyCoin ?: 0} 积分/天"
                monthly -> "${item.monthlyCoin ?: 0} 积分/月"
                else -> "${item.needCoin ?: 0} 积分"
            }
            tagRow.addView(TextView(context).apply {
                text = coinText
                setTextColor(brandBlue)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setTypeface(typeface, Typeface.BOLD)
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    marginStart = context.dp(8)
                }
            })
            infoWrap.addView(tagRow)
            item.guide?.trim()?.takeIf { it.isNotEmpty() }?.let { guideText ->
                val preview = if (guideText.length > 56) guideText.take(56) + "..." else guideText
                infoWrap.addView(TextView(context).apply {
                    text = "玩法摘要：$preview"
                    setTextColor(requireContext().themeColor(R.color.text_muted))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    setLineSpacing(0f, 1.25f)
                    setPadding(0, context.dp(8), 0, 0)
                })
            }
            addView(infoWrap)
            // 右箭头
            addView(TextView(context).apply {
                text = "›"
                setTextColor(brandBlue)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 24f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                layoutParams = LinearLayout.LayoutParams(context.dp(28), context.dp(28)).apply { marginStart = context.dp(6) }
            })
            setOnClickListener {
                context.startActivity(ProjectFormActivity.intent(context, item))
            }
            foreground = context.obtainStyledAttributes(intArrayOf(android.R.attr.selectableItemBackground)).getDrawable(0)
        }
    }

    private fun groupCard(groupKey: String, items: List<PortalProject>): View {
        val first = items.first()
        val expanded = expandedKeys.contains(groupKey)
        return requireContext().cardView().apply {
            setPadding(context.dp(14), context.dp(12), context.dp(14), context.dp(10))
            val topRow = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL ; gravity = Gravity.CENTER_VERTICAL }
            topRow.addView(requireContext().bodyText(first.activityName ?: "未命名项目").apply {
                textSize = 14.5f
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            })
            topRow.addView(TextView(context).apply {
                text = "共 ${items.size} 个账号"
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            })
            addView(topRow)
            addView(requireContext().captionText("费用：${first.priceText ?: "-"}  ·  ${first.qingLongConfig ?: "默认容器"}").apply {
                setPadding(0, context.dp(3), 0, context.dp(7))
            })
            val badgeRow = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
            badgeRow.addView(requireContext().badge("有效 ${items.count { it.bizStatus == "active" }}", ContextCompat.getColor(requireContext(), R.color.brand_green)))
            badgeRow.addView(requireContext().badge("快到期 ${items.count { it.bizStatus == "expiring" }}", ContextCompat.getColor(requireContext(), R.color.brand_orange)).apply {
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply { marginStart = context.dp(6) }
            })
            badgeRow.addView(requireContext().badge("过期 ${items.count { it.bizStatus == "expired" }}", ContextCompat.getColor(requireContext(), R.color.brand_red)).apply {
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply { marginStart = context.dp(6) }
            })
            addView(badgeRow)

            addView(TextView(context).apply {
                text = "💡 点击「展开账号」可查看每个账号的查询、续费、修改、删除等操作"
                setTextColor(Color.parseColor("#0891B2"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setLineSpacing(0f, 1.3f)
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#F0F9FF"))
                    cornerRadius = context.dp(8).toFloat()
                    setStroke(context.dp(1), Color.parseColor("#BAE6FD"))
                }
                setPadding(context.dp(10), context.dp(8), context.dp(10), context.dp(8))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    topMargin = context.dp(8)
                }
            })

            val actionRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(0, context.dp(9), 0, 0)
            }
            val queryBtn = Button(context).apply {
                text = "批量查询"
                setAllCaps(false)
                textSize = 12f
                setPadding(context.dp(10), 0, context.dp(10), 0)
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, context.dp(36))
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#E0F2FE"))
                    cornerRadius = context.dp(10).toFloat()
                }
                setTextColor(Color.parseColor("#0369A1"))
                setOnClickListener {
                    val loading = loadingDialog("正在批量查询，请稍候...")
                    lifecycleScope.launch {
                        val text = items.map { item ->
                            async {
                                val name = item.displayName ?: item.remark ?: item.activityName ?: "未命名账号"
                                val result = runCatching { queryIncomeSmart(item) }.getOrElse { sanitizeErrorMessage(it.message) }
                                "【$name】\n$result"
                            }
                        }.awaitAll().joinToString("\n\n")
                        loading.dismiss()
                        startActivity(ResultTextActivity.intent(requireContext(), first.activityName ?: "批量查询", text))
                    }
                }
            }
            val expandBtn = Button(context).apply {
                text = if (expanded) "收起账号 ▲" else "展开账号 ▼"
                setAllCaps(false)
                textSize = 12f
                setPadding(context.dp(10), 0, context.dp(10), 0)
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, context.dp(36))
                background = GradientDrawable().apply {
                    setColor(if (expanded) Color.parseColor("#FEF3C7") else Color.parseColor("#DCFCE7"))
                    cornerRadius = context.dp(10).toFloat()
                }
                setTextColor(if (expanded) Color.parseColor("#B45309") else requireContext().themeColor(R.color.positive))
            }
            actionRow.addView(queryBtn)
            actionRow.addView(View(context).apply {
                layoutParams = LinearLayout.LayoutParams(0, 1, 1f)
            })
            actionRow.addView(expandBtn)
            addView(actionRow)

            val detailsContainer = LinearLayout(context).apply {
                orientation = LinearLayout.VERTICAL
                alpha = if (expanded) 1f else 0f
                visibility = if (expanded) View.VISIBLE else View.GONE
            }
            if (expanded) items.forEach { item -> detailsContainer.addView(projectItemCard(item)) }
            addView(detailsContainer)

            expandBtn.setOnClickListener {
                val nowExpanded = expandedKeys.contains(groupKey)
                if (nowExpanded) {
                    expandedKeys.remove(groupKey)
                    detailsContainer.animate().alpha(0f).setDuration(160).withEndAction {
                        detailsContainer.visibility = View.GONE
                        detailsContainer.removeAllViews()
                        expandBtn.text = "展开账号 ▼"
                    }.start()
                } else {
                    expandedKeys.add(groupKey)
                    detailsContainer.removeAllViews()
                    items.forEach { item -> detailsContainer.addView(projectItemCard(item)) }
                    detailsContainer.visibility = View.VISIBLE
                    detailsContainer.alpha = 0f
                    detailsContainer.animate().alpha(1f).setDuration(180).start()
                    expandBtn.text = "收起账号 ▲"
                }
            }
        }
    }

    private fun projectItemCard(item: PortalProject): View {
        return LinearLayout(context).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(context.dp(12), context.dp(10), context.dp(12), context.dp(10))
            background = GradientDrawable().apply {
                setColor(requireContext().themeColor(R.color.input_bg))
                cornerRadius = context.dp(12).toFloat()
                setStroke(context.dp(1), requireContext().themeColor(R.color.border_default))
            }
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                bottomMargin = context.dp(8)
            }
            val nameRow = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL ; gravity = Gravity.CENTER_VERTICAL }
            nameRow.addView(TextView(context).apply {
                text = item.displayName ?: item.remark ?: item.activityName ?: "项目"
                setTextColor(requireContext().themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTypeface(typeface, Typeface.BOLD)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            })
            nameRow.addView(TextView(context).apply {
                text = item.bizStatusText ?: item.statusText ?: "未知"
                setTextColor(when (item.bizStatus) {
                    "active" -> ContextCompat.getColor(context, R.color.brand_green)
                    "expiring" -> ContextCompat.getColor(context, R.color.brand_orange)
                    "expired" -> ContextCompat.getColor(context, R.color.brand_red)
                    else -> requireContext().themeColor(R.color.text_muted)
                })
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            })
            addView(nameRow)
            addView(TextView(context).apply {
                text = "到期：${item.expireDate ?: "长期"}  ·  ${item.qingLongConfig ?: "默认容器"}"
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, context.dp(3), 0, context.dp(8))
            })
            val row = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                weightSum = 4f
            }
            val mkBtn = { label: String, bgColor: String, textColor: Int, onClick: () -> Unit ->
                Button(context).apply {
                    text = label
                    setAllCaps(false)
                    textSize = 11f
                    minWidth = context.dp(62)
                    setPadding(context.dp(8), 0, context.dp(8), 0)
                    layoutParams = LinearLayout.LayoutParams(0, context.dp(32), 1f).apply {
                        if (label != "查询") marginStart = context.dp(6)
                    }
                    background = GradientDrawable().apply {
                        cornerRadius = context.dp(8).toFloat()
                        setColor(Color.parseColor(bgColor))
                    }
                    setTextColor(textColor)
                    setOnClickListener { onClick() }
                }
            }
            row.addView(mkBtn("查询", "#E0F2FE", Color.parseColor("#0369A1")) {
                lifecycleScope.launch {
                    val loading = loadingDialog("正在查询，请稍候...")
                    runCatching { queryIncomeSmart(item) }
                        .onSuccess { startActivity(ResultTextActivity.intent(requireContext(), item.displayName ?: "查询结果", it)) }
                        .onFailure { handlePortalError(it) }
                    loading.dismiss()
                }
            })
            row.addView(mkBtn("续费", "#DCFCE7", ContextCompat.getColor(context, R.color.brand_green)) {
                showRenewDialog(item)
            })
            row.addView(mkBtn("修改", "#F3E8FF", Color.parseColor("#7C3AED")) {
                showUpdateDialog(item)
            })
            row.addView(mkBtn("删除", "#FEF2F2", ContextCompat.getColor(context, R.color.brand_red)) {
                showDeleteConfirm(item)
            })
            addView(row)
        }
    }

    private fun loadingDialog(message: String): AlertDialog {
        return AlertDialog.Builder(requireContext())
            .setTitle("请稍候")
            .setMessage(message)
            .setCancelable(false)
            .create()
            .also { it.show() }
    }

    private suspend fun queryIncomeSmart(item: PortalProject): String {
        val first = runCatching { AppServices.portalRepository.queryIncome(item.activityId ?: "", item.remark.orEmpty()) }
        val value = first.getOrNull()
        if (value != null) return value
        val error = first.exceptionOrNull()
        val message = error?.message.orEmpty()
        if (message.contains("未找到") || message.contains("对应项目") || message.contains("记录")) {
            cachedProjects = runCatching { AppServices.portalRepository.fetchProjects() }.getOrNull()
            return AppServices.portalRepository.queryIncome(item.activityId ?: "", item.remark.orEmpty())
        }
        throw error ?: IllegalStateException("查询失败")
    }

    private fun showUpdateDialog(item: PortalProject) {
        startActivity(CkEditActivity.intent(
            requireContext(),
            item.remark.orEmpty(),
            item.activityId ?: "",
            item.envValue.orEmpty(),
            item.ckTemplate.orEmpty(),
            item.inputFields ?: emptyList(),
        ))
    }

    private fun showDeleteConfirm(item: PortalProject) {
        fun calcPaidDays(): Int {
            val total = item.daysLeft ?: 0
            if (total <= 0) return 0
            val grantDateStr = item.grantExpireDate
            if (grantDateStr.isNullOrEmpty() || (item.needCoin ?: 0) > 0) return total
            return try {
                val sdf = java.text.SimpleDateFormat("yyyy-MM-dd", java.util.Locale.getDefault())
                val grantDate = sdf.parse(grantDateStr) ?: return total
                val grantEnd = java.util.Calendar.getInstance().apply {
                    time = grantDate
                    add(java.util.Calendar.DAY_OF_YEAR, 1)
                }.time
                val grantRemain = (grantEnd.time - System.currentTimeMillis()) / 86400000.0
                val granted = if (grantRemain > 1) (grantRemain - 1).toInt() else 0
                maxOf(total - granted, 0)
            } catch (_: Exception) { total }
        }
        val refundTip = when {
            // 按天计费退积分
            item.isDailyDeduct == true && (item.daysLeft ?: 0) > 0 && (item.dailyCoin ?: 0) > 0 -> {
                val paidDays = calcPaidDays()
                val estimated = (item.dailyCoin ?: 0) * paidDays
                if (estimated > 0) "预计返还积分：$estimated（最终以服务端结算为准）" else "赠送时长内，删除不退还积分"
            }
            item.isDailyDeduct == true -> {
                "预计返还积分：以服务端结算为准"
            }
            // 按月计费退积分
            item.isMonthlyDeduct == true && (item.needCoin ?: 0) > 0 -> {
                "该账号由一次性活动转换，删除不退还积分"
            }
            item.isMonthlyDeduct == true && (item.daysLeft ?: 0) > 0 && (item.monthlyCoin ?: 0) > 0 -> {
                val paidDays = calcPaidDays()
                val estimated = (((item.monthlyCoin ?: 0).toDouble() * paidDays.toDouble() / 30.0) + 0.5).toInt()
                if (estimated > 0) "预计返还积分：$estimated（最终以服务端结算为准）" else "赠送时长内，删除不退还积分"
            }
            item.isMonthlyDeduct == true -> {
                "预计返还积分：以服务端结算为准"
            }
            // 一次性扣费
            else -> {
                "此活动为一次性扣费，删除不退还积分"
            }
        }
        androidx.appcompat.app.AlertDialog.Builder(requireContext())
            .setTitle("确认删除")
            .setMessage("项目：${item.displayName ?: item.remark ?: item.activityName ?: "项目"}\n到期：${item.expireDate ?: "长期 / 未记录"}\n$refundTip")
            .setNegativeButton("取消", null)
            .setPositiveButton("确认删除") { _, _ ->
                val busyKey = "delete:${item.activityId}:${item.remark.orEmpty()}"
                if (!projectActionBusyKeys.add(busyKey)) {
                    toast("请勿重复提交，上一笔删除请求正在处理中")
                    return@setPositiveButton
                }
                lifecycleScope.launch {
                    try {
                        runCatching { AppServices.portalRepository.deleteProject(item.activityId ?: "", item.remark.orEmpty()) }
                            .onSuccess {
                                cachedProjects = null
                                alert(it) { renderCurrentTab(forceRefresh = true) }
                            }
                            .onFailure { handlePortalError(it) }
                    } finally {
                        projectActionBusyKeys.remove(busyKey)
                    }
                }
            }
            .show()
    }

    private fun showRenewDialog(item: PortalProject) {
        val isDaily = item.isDailyDeduct == true
        val unit = if (isDaily) "天" else "月"
        val unitPrice = if (isDaily) item.dailyCoin else item.monthlyCoin
        val priceLine = unitPrice?.let { "每${unit}约扣 $it 积分" } ?: item.priceText ?: "将按当前项目规则扣除积分"

        val input = EditText(requireContext()).apply {
            hint = "请输入续费${unit}数，例如 1"
            inputType = InputType.TYPE_CLASS_NUMBER
            setText("1")
            setSelectAllOnFocus(true)
            setPadding(requireContext().dp(14), requireContext().dp(10), requireContext().dp(14), requireContext().dp(10))
        }
        val msgView = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(requireContext().dp(14), requireContext().dp(8), requireContext().dp(14), 0)
        }
        msgView.addView(TextView(requireContext()).apply {
            text = "账号：${item.displayName ?: item.remark ?: "项目"}\n$priceLine"
            textSize = 15f
        })
        val totalView = TextView(requireContext()).apply {
            textSize = 14f
            setPadding(0, requireContext().dp(6), 0, 0)
        }
        fun updateTotal() {
            val n = input.text?.toString()?.toIntOrNull() ?: 0
            if (unitPrice != null && n > 0) {
                val total = unitPrice * n
                totalView.text = "💰 预计扣除：$total 积分（${n}${unit} × ${unitPrice}积分/${unit}）"
                totalView.setTextColor(requireContext().themeColor(R.color.brand_red))
            } else {
                totalView.text = ""
            }
        }
        updateTotal()
        input.addTextChangedListener(object : android.text.TextWatcher {
            override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
            override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
            override fun afterTextChanged(s: android.text.Editable?) { updateTotal() }
        })
        msgView.addView(totalView)
        msgView.addView(input)

        AlertDialog.Builder(requireContext())
            .setTitle("确认续费")
            .setView(msgView)
            .setNegativeButton("取消", null)
            .setPositiveButton("确认续费") { _, _ ->
                val months = input.text?.toString()?.toIntOrNull() ?: 0
                if (months <= 0) {
                    alert("请输入正确的续费${unit}数")
                    return@setPositiveButton
                }
                val busyKey = "renew:${item.activityId}:${item.remark.orEmpty()}"
                if (!projectActionBusyKeys.add(busyKey)) {
                    toast("请勿重复提交，上一笔续费请求正在处理中")
                    return@setPositiveButton
                }
                lifecycleScope.launch {
                    try {
                        runCatching { AppServices.portalRepository.renewProject(item.activityId ?: "", item.remark.orEmpty(), months) }
                            .onSuccess {
                                cachedProjects = null
                                alert(it) { renderCurrentTab(forceRefresh = true) }
                            }
                            .onFailure { handlePortalError(it) }
                    } finally {
                        projectActionBusyKeys.remove(busyKey)
                    }
                }
            }
            .show()
    }

    private fun emptyCard(message: String): View {
        return requireContext().cardView().apply {
            addView(requireContext().bodyText(message).apply { textSize = 15f })
        }
    }

    private var wxPolling = false
    private var wxDevices: List<com.goudong.jd.data.model.PortalWxDevice> = emptyList()
    private val projectActionBusyKeys = mutableSetOf<String>()

    private fun isYybProtocolTabActive(): Boolean = currentTab == 3 && protocolSubIndex == 0

    private fun renderProtocolAccess(forceRefresh: Boolean) {
        swipeRefreshLayout.isRefreshing = false
        contentRoot.removeAllViews()
        val ctx = requireContext()
        val pillRow = LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = ctx.dp(10) }
        }
        listOf("应用宝协议", "微信协议", "协议双绑").forEachIndexed { index, label ->
            val active = protocolSubIndex == index
            pillRow.addView(TextView(ctx).apply {
                text = label
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setTypeface(typeface, if (active) Typeface.BOLD else Typeface.NORMAL)
                setTextColor(
                    if (active) ContextCompat.getColor(ctx, R.color.brand_primary)
                    else ctx.themeColor(R.color.text_muted)
                )
                gravity = Gravity.CENTER
                background = GradientDrawable().apply {
                    setColor(if (active) Color.parseColor("#EFF6FF") else Color.TRANSPARENT)
                    cornerRadius = ctx.dp(8).toFloat()
                }
                setPadding(ctx.dp(8), ctx.dp(8), ctx.dp(8), ctx.dp(8))
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                    if (index < 2) marginEnd = ctx.dp(4)
                }
                setOnClickListener {
                    if (protocolSubIndex == index) return@setOnClickListener
                    protocolSubIndex = index
                    renderProtocolAccess(forceRefresh = false)
                }
            })
        }
        contentRoot.addView(pillRow)
        when (protocolSubIndex) {
            0 -> renderYybProtocol(forceRefresh)
            1 -> renderWxProtocol()
            2 -> renderProtocolBindTab()
        }
        loadProtocolBindings()
    }

    private fun loadProtocolBindings() {
        lifecycleScope.launch {
            runCatching {
                protocolBindings = AppServices.portalRepository.fetchProtocolBindings()
                protocolBindQuota = AppServices.portalRepository.fetchProtocolBindQuota()
                if (wxDevices.isEmpty()) {
                    wxDevices = AppServices.portalRepository.fetchWxDevices()
                }
                if (yybAccounts.isEmpty()) {
                    yybAccounts = YybAccountStore.accounts.ifEmpty {
                        AppServices.portalRepository.fetchYybStatus(false).accounts.orEmpty()
                    }
                }
            }
            when (protocolSubIndex) {
                1 -> if (currentTab == 3) renderWxDeviceList(wxDevices)
                0 -> if (isYybProtocolTabActive()) renderYybAccounts()
                2 -> if (currentTab == 3) {
                    renderProtocolBindQuota()
                    renderProtocolBindList()
                }
            }
        }
    }

    private fun bindingByWx(wxid: String?) =
        protocolBindings.firstOrNull { it.wxWxid == wxid?.trim() }

    private fun bindingByOpenId(openid: String?) =
        protocolBindings.firstOrNull { it.yybOpenId == openid?.trim() }

    private fun shortenProtocolId(id: String?, head: Int = 8, tail: Int = 6): String {
        val s = id?.trim().orEmpty()
        if (s.length <= head + tail + 3) return s
        return s.take(head) + "…" + s.takeLast(tail)
    }

    private fun formatYybExpiry(acc: com.goudong.jd.data.model.PortalYybAccount): Pair<String, Boolean> {
        val loginSec = acc.loginAt ?: acc.createdAt
        if (loginSec <= 0L) return "" to false
        val expireMs = (acc.expiresAt ?: (loginSec + 30L * 24 * 3600)) * 1000
        val remain = expireMs - System.currentTimeMillis()
        if (remain <= 0L) return "登录已过期，请重新扫码登录延期" to true
        val days = remain / 86400000
        val hours = (remain % 86400000) / 3600000
        val warn = remain < 3 * 86400000
        val expireText = java.text.SimpleDateFormat("MM-dd HH:mm", java.util.Locale.CHINA)
            .format(java.util.Date(expireMs))
        return "剩余有效期 $days 天 $hours 小时（至 $expireText）" to warn
    }

    private fun parseProxyAreaList(data: com.google.gson.JsonObject?): List<Pair<String, String>> {
        if (data == null) return emptyList()
        val arr = data.get("list")?.asJsonArray
            ?: data.get("provinceList")?.asJsonArray
            ?: data.get("city")?.asJsonArray
            ?: return emptyList()
        return arr.mapNotNull { el ->
            val obj = el.asJsonObject
            val code = obj.get("regionCode")?.asString ?: obj.get("region_code")?.asString ?: return@mapNotNull null
            val name = obj.get("regionName")?.asString ?: obj.get("region_name")?.asString ?: code
            code to name
        }
    }

    private fun boundWxSet() = protocolBindings.mapNotNull { it.wxWxid }.toSet()
    private fun boundOpenIdSet() = protocolBindings.mapNotNull { it.yybOpenId }.toSet()

    private fun yybAccountKey(acc: com.goudong.jd.data.model.PortalYybAccount): String {
        return if (acc.bindingId > 0) acc.bindingId.toString() else (acc.openid ?: "")
    }

    private fun yybAccountRef(acc: com.goudong.jd.data.model.PortalYybAccount): String = yybAccountKey(acc)

    private fun renderYybProtocol(forceRefresh: Boolean) {
        val ctx = requireContext()

        contentRoot.addView(buildCollapsibleIntroCard(
            title = "📖 什么是应用宝协议？",
            content = """
                应用宝协议通过提交应用宝 openid（owNAX 开头），向协议网关请求小程序登录凭证（CK）。无需保持微信长期在线，也没有封号风险。

                【主要作用】
                • 提交 openid 即可获取小程序 CK，适合青龙等自动化脚本
                • 扫码登录后有效期 30 天，到期前重新扫码可延期
                • 可单独使用应用宝协议，不必依赖微信协议

                【从微信协议迁移】
                若你此前使用微信协议，青龙脚本里提交的是微信 wxid 作为 CK：
                • 需先将 wxid 与对应的应用宝 openid 双绑
                • 绑定后脚本里仍填原 wxid，网关会自动路由到应用宝获取 code
                • 之后即使退出或删除微信协议设备，只要双绑关系保留，wxid 依然能路由到应用宝
                • 也可完全切换到应用宝，直接提交 openid 作为 CK
            """.trimIndent(),
        ))

        val actionRow = LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, 0, 0, ctx.dp(10))
        }
        actionRow.addView(ctx.primaryButton("扫码添加").apply {
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                marginEnd = ctx.dp(6)
            }
            setOnClickListener { startYybScan() }
        })
        yybReloadBtn = Button(ctx).apply {
            text = "刷新检测"
            setAllCaps(false)
            setTextColor(ctx.themeColor(R.color.text_primary))
            background = GradientDrawable().apply {
                setColor(ctx.themeColor(R.color.chip_bg))
                cornerRadius = ctx.dp(10).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            setOnClickListener { loadYybPanel(autoCheck = true, showAlert = true, manual = true) }
        }.also { actionRow.addView(it) }
        contentRoot.addView(actionRow)

        fun manageBtn(label: String, danger: Boolean = false, onClick: () -> Unit): TextView {
            return TextView(ctx).apply {
                text = label
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setTypeface(typeface, Typeface.BOLD)
                setTextColor(if (danger) Color.parseColor("#DC2626") else ctx.themeColor(R.color.text_primary))
                gravity = Gravity.CENTER
                background = GradientDrawable().apply {
                    setColor(
                        if (danger) Color.parseColor("#FEE2E2")
                        else ctx.themeColor(R.color.chip_bg)
                    )
                    cornerRadius = ctx.dp(8).toFloat()
                }
                setPadding(ctx.dp(8), ctx.dp(6), ctx.dp(8), ctx.dp(6))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                ).apply { marginStart = ctx.dp(4) }
                minWidth = ctx.dp(40)
                setOnClickListener { onClick() }
            }
        }

        val sectionRow = LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(ctx.dp(2), ctx.dp(6), ctx.dp(2), ctx.dp(8))
            setBackgroundColor(ctx.themeColor(R.color.surface_soft))
        }
        yybSectionTitle = TextView(ctx).apply {
            text = "账号列表 · 0"
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTypeface(typeface, Typeface.BOLD)
            setTextColor(ctx.themeColor(R.color.text_secondary))
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
        }.also { sectionRow.addView(it) }
        yybRefreshAccountBtn = manageBtn("刷新") {
            if (yybAccountActionBusy) return@manageBtn
            yybRefreshAccountBtn?.let { animateAccountActionBtn(it) }
            refreshSelectedYyb()
        }.also {
            it.visibility = View.GONE
            sectionRow.addView(it)
        }
        yybDeleteAccountBtn = manageBtn("删除", danger = true) { deleteSelectedYyb() }.also {
            it.visibility = View.GONE
            sectionRow.addView(it)
        }
        yybLoadingBar = ProgressBar(ctx).apply {
            isIndeterminate = true
            visibility = View.GONE
            layoutParams = LinearLayout.LayoutParams(ctx.dp(14), ctx.dp(14)).apply {
                marginStart = ctx.dp(4)
            }
        }.also { sectionRow.addView(it) }
        yybServiceDot = View(ctx).apply {
            background = GradientDrawable().apply {
                shape = GradientDrawable.OVAL
                setColor(Color.parseColor("#CBD5E1"))
            }
            layoutParams = LinearLayout.LayoutParams(ctx.dp(7), ctx.dp(7)).apply {
                marginStart = ctx.dp(6)
                marginEnd = ctx.dp(6)
            }
        }.also { sectionRow.addView(it) }
        yybServiceLabel = TextView(ctx).apply {
            text = "待机"
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            setTextColor(ctx.themeColor(R.color.text_muted))
        }.also { sectionRow.addView(it) }
        contentRoot.addView(sectionRow)

        contentRoot.addView(LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            tag = "yyb_account_list"
        })

        yybManageRow = null

        updateYybActionEnabled()
        applyYybFromStore()
        if (!YybAccountStore.sessionAutoChecked) {
            YybAccountStore.prefetchIfNeeded(lifecycleScope, autoCheck = false)
        }
    }

    private fun buildCollapsibleIntroCard(title: String, content: String): View {
        return requireContext().cardView().apply {
            val titleRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
            }
            val titleText = TextView(context).apply {
                text = title
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            }
            val toggleIcon = TextView(context).apply {
                text = "▶"
                setTextColor(requireContext().themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            }
            titleRow.addView(titleText)
            titleRow.addView(toggleIcon)
            addView(titleRow)
            val detailContent = requireContext().bodyText(content).apply {
                setLineSpacing(0f, 1.5f)
                setPadding(0, requireContext().dp(6), 0, 0)
                visibility = View.GONE
            }
            addView(detailContent)
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT,
            ).apply { bottomMargin = requireContext().dp(10) }
            titleRow.setOnClickListener {
                if (detailContent.visibility == View.GONE) {
                    detailContent.visibility = View.VISIBLE
                    toggleIcon.text = "▼"
                } else {
                    detailContent.visibility = View.GONE
                    toggleIcon.text = "▶"
                }
            }
        }
    }

    private fun applyYybFromStore() {
        if (!isYybProtocolTabActive()) return
        val ready = YybAccountStore.isServiceReady
        val title = if (ready) "已启动" else "未启动"
        yybAccounts = YybAccountStore.accounts
        if (yybAccounts.none { yybAccountKey(it) == yybSelectedKey }) {
            yybSelectedKey = yybAccounts.firstOrNull()?.let { yybAccountKey(it) }.orEmpty()
        }
        applyYybHeader(ready, title, yybAccounts.size)
        renderYybAccounts()
        updateYybActionEnabled()
    }

    private fun selectedYybAccount(): com.goudong.jd.data.model.PortalYybAccount? {
        return yybAccounts.firstOrNull { yybAccountKey(it) == yybSelectedKey }
    }

    private fun updateYybActionEnabled() {
        val busy = (yybManualRefreshing || YybAccountStore.isLoading) || yybScanBusy
        yybReloadBtn?.isEnabled = !busy
        yybReloadBtn?.alpha = if (busy) 0.5f else 1f
        yybReloadBtn?.text = when {
            yybManualRefreshing -> "检测中…"
            YybAccountStore.isLoading -> "同步中…"
            else -> "刷新检测"
        }
        yybLoadingBar?.visibility = if (busy && !yybScanBusy) View.VISIBLE else View.GONE
    }

    private fun applyYybHeader(ready: Boolean, title: String, count: Int) {
        yybServiceReady = ready
        yybSectionTitle?.text = "账号列表 · $count"
        (yybServiceDot?.background as? GradientDrawable)?.setColor(
            if (ready) Color.parseColor("#22C55E") else Color.parseColor("#F59E0B")
        )
        yybServiceLabel?.apply {
            text = title
            setTextColor(
                if (ready) requireContext().themeColor(R.color.text_muted)
                else Color.parseColor("#F59E0B")
            )
        }
    }

    private fun loadYybPanel(autoCheck: Boolean, showAlert: Boolean = autoCheck, manual: Boolean = false) {
        if (manual && YybAccountStore.isLoading) {
            toast("正在刷新，请稍候")
            swipeRefreshLayout.isRefreshing = false
            return
        }
        if (manual) {
            yybManualRefreshing = true
            updateYybActionEnabled()
        }
        YybAccountStore.reload(
            scope = lifecycleScope,
            autoCheck = autoCheck,
            showAlert = showAlert,
            force = true,
        ) { result ->
            if (manual) {
                yybManualRefreshing = false
                updateYybActionEnabled()
            }
            swipeRefreshLayout.isRefreshing = false
            result.onFailure {
                if (showAlert) handlePortalError(it)
                if (isYybProtocolTabActive()) renderYybAccounts(it.message)
            }
            result.onSuccess { st ->
                if (!st.enabled || !st.ready) {
                    if (showAlert) toast(st.message ?: "应用宝服务暂不可用")
                } else if (showAlert && autoCheck) {
                    YybAccountStore.consumePendingAlert()?.let { msg ->
                        AlertDialog.Builder(requireContext())
                            .setTitle("检测完成")
                            .setMessage(msg)
                            .setPositiveButton("好的", null)
                            .show()
                    }
                }
                applyYybFromStore()
            }
        }
    }

    private fun renderYybAccounts(error: String? = null) {
        if (!isYybProtocolTabActive()) return
        val host = contentRoot.findViewWithTag<LinearLayout>("yyb_account_list") ?: return
        host.removeAllViews()
        val showActions = selectedYybAccount() != null && error == null && yybAccounts.isNotEmpty()
        yybRefreshAccountBtn?.visibility = if (showActions) View.VISIBLE else View.GONE
        yybDeleteAccountBtn?.visibility = if (showActions) View.VISIBLE else View.GONE
        when {
            error != null -> host.addView(emptyCard(error))
            yybAccounts.isEmpty() -> host.addView(
                emptyCard(
                    if (YybAccountStore.isLoading && !YybAccountStore.sessionAutoChecked) {
                        "正在同步账号…"
                    } else {
                        "暂无账号，点击上方「扫码添加」"
                    }
                )
            )
            else -> {
                yybAccounts.forEach { acc ->
                    host.addView(buildYybProtocolCard(acc))
                }
            }
        }
    }

    private fun buildYybProtocolCard(acc: com.goudong.jd.data.model.PortalYybAccount): View {
        val ctx = requireContext()
        val key = yybAccountKey(acc)
        val selected = key == yybSelectedKey
        val st = (acc.status ?: "").lowercase()
        val alive = st == "alive" || st == "online"
        val name = acc.nickname?.trim()?.takeIf { it.isNotEmpty() }
            ?: acc.openid?.trim()?.takeIf { it.isNotEmpty() }
            ?: "未命名"
        return ctx.cardView().apply {
            setPadding(ctx.dp(12), ctx.dp(12), ctx.dp(12), ctx.dp(12))
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = ctx.dp(10) }
            if (selected) {
                background = GradientDrawable().apply {
                    setColor(ctx.themeColor(R.color.surface_card))
                    cornerRadius = ctx.dp(14).toFloat()
                    setStroke(ctx.dp(2), Color.parseColor("#3B82F6"))
                }
            }
            addView(LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                addView(TextView(ctx).apply {
                    text = name
                    setTextColor(ctx.themeColor(R.color.text_primary))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                    setTypeface(typeface, Typeface.BOLD)
                    maxLines = 1
                    ellipsize = android.text.TextUtils.TruncateAt.END
                    layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                        marginEnd = ctx.dp(6)
                    }
                })
                addView(TextView(ctx).apply {
                    text = if (alive) "可用" else "失效"
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f)
                    setTypeface(typeface, Typeface.BOLD)
                    setTextColor(if (alive) Color.parseColor("#16A34A") else Color.parseColor("#DC2626"))
                    gravity = Gravity.CENTER
                    background = GradientDrawable().apply {
                        setColor(if (alive) Color.parseColor("#DCFCE7") else Color.parseColor("#FEE2E2"))
                        cornerRadius = ctx.dp(6).toFloat()
                    }
                    setPadding(ctx.dp(8), ctx.dp(4), ctx.dp(8), ctx.dp(4))
                    minWidth = ctx.dp(36)
                    minHeight = ctx.dp(20)
                })
            })
            val uinText = if ((acc.uin ?: 0L) > 0) acc.uin.toString() else "-"
            addView(ctx.captionText("UIN $uinText").apply {
                setPadding(0, ctx.dp(6), 0, 0)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            })
            addView(LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(0, ctx.dp(2), 0, 0)
                addView(ctx.captionText("OpenID").apply {
                    layoutParams = LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.WRAP_CONTENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT,
                    ).apply { marginEnd = ctx.dp(8) }
                })
                addView(TextView(ctx).apply {
                    text = acc.openid ?: "-"
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    setTextColor(ctx.themeColor(R.color.text_muted))
                    maxLines = 2
                    ellipsize = android.text.TextUtils.TruncateAt.MIDDLE
                    layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                })
            })
            bindingByOpenId(acc.openid)?.let { binding ->
                val wxDev = wxDevices.firstOrNull { it.wxid == binding.wxWxid }
                val peerName = wxDev?.nickname?.takeIf { it.isNotBlank() } ?: binding.nickname ?: "微信设备"
                addView(ctx.captionText("已绑定微信 · $peerName · ${shortenProtocolId(binding.wxWxid)}").apply {
                    setPadding(0, ctx.dp(6), 0, 0)
                })
            }
            val (expiryText, warn) = formatYybExpiry(acc)
            if (expiryText.isNotBlank()) {
                addView(TextView(ctx).apply {
                    text = expiryText
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    setTypeface(typeface, if (warn) Typeface.BOLD else Typeface.NORMAL)
                    setTextColor(
                        if (expiryText.startsWith("登录已过期")) Color.parseColor("#DC2626")
                        else ctx.themeColor(R.color.brand_primary)
                    )
                    setPadding(0, ctx.dp(6), 0, 0)
                })
            }
            fun cardActionBtn(label: String, color: Int, onClick: () -> Unit): TextView {
                return TextView(ctx).apply {
                    text = label
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    setTypeface(typeface, Typeface.BOLD)
                    setTextColor(color)
                    gravity = Gravity.CENTER
                    background = GradientDrawable().apply {
                        setColor(Color.argb(26, Color.red(color), Color.green(color), Color.blue(color)))
                        cornerRadius = ctx.dp(8).toFloat()
                    }
                    setPadding(ctx.dp(10), ctx.dp(8), ctx.dp(10), ctx.dp(8))
                    layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                        marginEnd = ctx.dp(5)
                    }
                    setOnClickListener { onClick() }
                }
            }
            addView(LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                setPadding(0, ctx.dp(10), 0, 0)
                addView(cardActionBtn("复制 OpenID", Color.parseColor("#2563EB")) {
                    val oid = acc.openid?.trim().orEmpty()
                    if (oid.isEmpty()) {
                        toast("无可复制的 OpenID")
                    } else {
                        val clip = ctx.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                        clip.setPrimaryClip(ClipData.newPlainText("openid", oid))
                        toast("已复制 OpenID")
                    }
                }.apply {
                    (layoutParams as LinearLayout.LayoutParams).marginEnd = ctx.dp(5)
                })
                bindingByOpenId(acc.openid)?.let { binding ->
                    addView(cardActionBtn("解除双绑", Color.parseColor("#DC2626")) {
                        confirmProtocolUnbind(binding.wxWxid.orEmpty(), binding.yybOpenId.orEmpty())
                    }.apply {
                        (layoutParams as LinearLayout.LayoutParams).marginEnd = 0
                    })
                }
            })
            setOnClickListener {
                yybSelectedKey = key
                renderYybAccounts()
            }
        }
    }

    private fun startYybScan() {
        if (yybScanBusy || yybManualRefreshing) return
        yybScanBusy = true
        updateYybActionEnabled()
        lifecycleScope.launch {
            val cfg = runCatching { AppServices.portalRepository.fetchProtocolProxyConfig() }.getOrNull()
            protocolProxyConfig = cfg
            yybScanBusy = false
            updateYybActionEnabled()
            if (cfg?.proxyEnabled == true) {
                showYybRegionDialog(cfg)
            } else {
                beginYybQrScan("", "", useProxy = false, packId = cfg?.proxyDefaultPackid.orEmpty())
            }
        }
    }

    private fun showYybRegionDialog(cfg: com.goudong.jd.data.model.PortalProxyConfig) {
        val ctx = requireContext()
        val dialogView = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(ctx.dp(8), ctx.dp(4), ctx.dp(8), 0)
        }
        val hint = cfg.proxyBypassRegionName?.takeIf { it.isNotBlank() }?.let {
            "「$it」等地区免代理直连；其他地区请选择与你所在地一致的省/市。异地登录可能只有1天有效期。"
        } ?: "请选择与你当前所在地一致的省/市。异地登录可能只有1天有效期。"
        dialogView.addView(ctx.bodyText(hint).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            setPadding(0, 0, 0, ctx.dp(10))
        })
        val provinceSpinner = android.widget.Spinner(ctx)
        val citySpinner = android.widget.Spinner(ctx)
        dialogView.addView(provinceSpinner)
        dialogView.addView(citySpinner.apply { setPadding(0, ctx.dp(8), 0, 0) })
        val scanCostLabel = TextView(ctx).apply {
            visibility = View.GONE
        }
        dialogView.addView(scanCostLabel)
        val dialog = androidx.appcompat.app.AlertDialog.Builder(ctx)
            .setTitle("选择登录地区")
            .setView(dialogView)
            .setNegativeButton("取消", null)
            .setPositiveButton("生成二维码", null)
            .create()
        dialog.show()
        var provinces = emptyList<Pair<String, String>>()
        var cities = emptyList<Pair<String, String>>()
        lifecycleScope.launch {
            val quota = runCatching { AppServices.portalRepository.fetchProtocolBindQuota() }.getOrNull()
            applyScanCostLabel(scanCostLabel, quota?.scanLoginCost, quota?.scanCostHint)
            val packId = cfg.proxyDefaultPackid.orEmpty()
            provinces = parseProxyAreaList(
                runCatching { AppServices.portalRepository.fetchProtocolProxyAreas(packId = packId) }.getOrNull()
            )
            provinceSpinner.adapter = android.widget.ArrayAdapter(
                ctx,
                android.R.layout.simple_spinner_dropdown_item,
                listOf("请选择省份") + provinces.map { it.second },
            )
        }
        provinceSpinner.onItemSelectedListener = object : android.widget.AdapterView.OnItemSelectedListener {
            override fun onNothingSelected(parent: android.widget.AdapterView<*>?) {}
            override fun onItemSelected(parent: android.widget.AdapterView<*>?, view: View?, position: Int, id: Long) {
                if (position <= 0) {
                    cities = emptyList()
                    citySpinner.adapter = android.widget.ArrayAdapter(
                        ctx, android.R.layout.simple_spinner_dropdown_item, listOf("请选择城市")
                    )
                    return
                }
                val code = provinces[position - 1].first
                lifecycleScope.launch {
                    cities = parseProxyAreaList(
                        runCatching {
                            AppServices.portalRepository.fetchProtocolProxyAreas(
                                parentCode = code,
                                packId = cfg.proxyDefaultPackid.orEmpty(),
                            )
                        }.getOrNull()
                    )
                    citySpinner.adapter = android.widget.ArrayAdapter(
                        ctx,
                        android.R.layout.simple_spinner_dropdown_item,
                        listOf("请选择城市") + cities.map { it.second },
                    )
                }
            }
        }
        dialog.getButton(androidx.appcompat.app.AlertDialog.BUTTON_POSITIVE).setOnClickListener {
            val cityPos = citySpinner.selectedItemPosition
            if (cityPos <= 0) {
                toast("请选择城市")
                return@setOnClickListener
            }
            val region = cities[cityPos - 1]
            dialog.dismiss()
            beginYybQrScan(region.first, region.second, useProxy = true, packId = cfg.proxyDefaultPackid.orEmpty())
        }
    }

    private fun beginYybQrScan(regionCode: String, regionName: String, useProxy: Boolean, packId: String) {
        if (yybScanBusy || yybManualRefreshing) return
        yybScanBusy = true
        updateYybActionEnabled()
        lifecycleScope.launch {
            runCatching {
                AppServices.portalRepository.createYybQr(
                    regionCode = regionCode,
                    regionName = regionName,
                    useProxy = useProxy,
                    packId = packId,
                )
            }
                .onSuccess { data ->
                    val sessionId = data.sessionId
                    if (sessionId.isNullOrBlank() || data.imageBase64.isNullOrBlank()) {
                        toast(data.scanCostHint ?: "二维码生成失败")
                        return@onSuccess
                    }
                    showYybQrDialog(
                        sessionId,
                        data.imageBase64,
                        cost = data.scanLoginCost,
                        hint = data.scanCostHint
                    )
                }
                .onFailure {
                    toast(it.message ?: "扫码失败")
                    handlePortalError(it)
                }
            yybScanBusy = false
            updateYybActionEnabled()
        }
    }

    private fun applyScanCostLabel(label: TextView, cost: Int?, hint: String?) {
        val preview = formatScanCostPreview(cost, hint)
        if (preview.text.isBlank()) {
            label.visibility = View.GONE
            return
        }
        label.visibility = View.VISIBLE
        label.text = preview.text
        val isFree = preview.isFree
        label.setTextColor(if (isFree) Color.parseColor("#16A34A") else Color.parseColor("#D97706"))
        label.setTypeface(label.typeface, Typeface.BOLD)
        label.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
        label.setPadding(requireContext().dp(10), requireContext().dp(8), requireContext().dp(10), requireContext().dp(4))
    }

    private fun animateAccountActionBtn(view: View) {
        view.animate().scaleX(0.94f).scaleY(0.94f).setDuration(80).withEndAction {
            view.animate().scaleX(1f).scaleY(1f).setDuration(120).start()
        }.start()
    }

    private fun showYybQrDialog(sessionId: String, base64: String, cost: Int?, hint: String?) {
        yybPolling = true
        val ctx = requireContext()
        val dialogView = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
            setPadding(ctx.dp(24), ctx.dp(24), ctx.dp(24), ctx.dp(24))
        }
        val qrImage = android.widget.ImageView(ctx).apply {
            layoutParams = LinearLayout.LayoutParams(ctx.dp(220), ctx.dp(220))
            scaleType = android.widget.ImageView.ScaleType.FIT_CENTER
        }
        decodeQrImage(base64)?.let { qrImage.setImageBitmap(it) }
        dialogView.addView(qrImage)

        formatYybScanCostNote(cost, hint)?.let { note ->
            val preview = formatScanCostPreview(cost, hint)
            dialogView.addView(TextView(ctx).apply {
                text = note
                setTextColor(Color.parseColor(if (preview.isFree) "#16A34A" else "#D97706"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                setPadding(0, ctx.dp(12), 0, 0)
            })
        }

        val stateText = ctx.captionText("请使用微信扫码确认登录").apply {
            gravity = Gravity.CENTER
            setPadding(0, ctx.dp(10), 0, 0)
        }
        dialogView.addView(stateText)
        val dialog = androidx.appcompat.app.AlertDialog.Builder(ctx)
            .setTitle("应用宝扫码")
            .setView(dialogView)
            .setNegativeButton("关闭") { _, _ -> yybPolling = false }
            .create()
        dialog.setOnDismissListener { yybPolling = false }
        dialog.show()
        lifecycleScope.launch {
            while (yybPolling) {
                kotlinx.coroutines.delay(800)
                if (!yybPolling) break
                val poll = runCatching { AppServices.portalRepository.pollYybQr(sessionId) }
                if (poll.isFailure) {
                    val msg = poll.exceptionOrNull()?.message.orEmpty()
                    if (msg.contains("deadline", true) || msg.contains("timeout", true)) {
                        stateText.text = "等待扫码中（网络较慢）…"
                        continue
                    }
                    yybPolling = false
                    stateText.text = msg.ifBlank { "登录失败" }
                    toast(msg.ifBlank { "扫码确认失败" })
                    break
                }
                val status = (poll.getOrNull()?.status ?: "").lowercase()
                when (status) {
                    "scanned" -> stateText.text = "已扫码，请在手机上点击「确认登录」"
                    "authorized", "confirmed" -> {
                        yybPolling = false
                        stateText.text = "正在完成绑定…"
                        runCatching { AppServices.portalRepository.confirmYybQr(sessionId) }
                            .onSuccess { result ->
                                dialog.dismiss()
                                val msg = formatYybConfirmMessage(result.alreadyBound, result.cost)
                                toast(msg)
                                loadYybPanel(autoCheck = false, showAlert = false)
                            }
                            .onFailure {
                                dialog.dismiss()
                                toast(it.message ?: "确认失败")
                                handlePortalError(it)
                            }
                    }
                    "expired", "cancelled", "unknown" -> {
                        yybPolling = false
                        stateText.text = if (status == "expired") "二维码已过期，请重新生成" else "扫码已取消"
                    }
                }
            }
        }
    }

    private fun refreshSelectedYyb() {
        val acc = selectedYybAccount() ?: return toast("请先选择一个账号")
        if (yybAccountActionBusy) return
        yybAccountActionBusy = true
        yybRefreshAccountBtn?.isEnabled = false
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.refreshYybAccount(yybAccountRef(acc)) }
                .onSuccess {
                    toast("存活状态已刷新")
                    loadYybPanel(autoCheck = false, showAlert = false)
                }
                .onFailure {
                    toast(it.message ?: "刷新失败")
                    handlePortalError(it)
                }
            yybAccountActionBusy = false
            yybRefreshAccountBtn?.isEnabled = true
        }
    }

    private fun confirmProtocolUnbind(wxWxid: String, yybOpenId: String) {
        androidx.appcompat.app.AlertDialog.Builder(requireContext())
            .setTitle("解除双绑")
            .setMessage("确定解除该微信与应用宝账号的双绑关系？")
            .setNegativeButton("取消", null)
            .setPositiveButton("解绑") { _, _ ->
                lifecycleScope.launch {
                    runCatching { AppServices.portalRepository.protocolUnbind(wxWxid, yybOpenId) }
                        .onSuccess {
                            toast(it)
                            loadProtocolBindings()
                        }
                        .onFailure {
                            toast(it.message ?: "解绑失败")
                            handlePortalError(it)
                        }
                }
            }
            .show()
    }

    private fun deleteSelectedYyb() {
        val acc = selectedYybAccount() ?: return toast("请先选择一个账号")
        androidx.appcompat.app.AlertDialog.Builder(requireContext())
            .setTitle("删除账号")
            .setMessage("确定删除该应用宝账号？")
            .setNegativeButton("取消", null)
            .setPositiveButton("删除") { _, _ ->
                lifecycleScope.launch {
                    runCatching { AppServices.portalRepository.deleteYybAccount(yybAccountRef(acc)) }
                        .onSuccess {
                            toast(it)
                            yybSelectedKey = ""
                            loadYybPanel(autoCheck = false, showAlert = false)
                        }
                        .onFailure {
                            toast(it.message ?: "删除失败")
                            handlePortalError(it)
                        }
                }
            }
            .show()
    }

    private fun renderProtocolBindTab() {
        val ctx = requireContext()
        val scroll = android.widget.ScrollView(ctx).apply {
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT,
            )
        }
        val host = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
        }
        scroll.addView(host)
        contentRoot.addView(scroll)

        host.addView(ctx.cardView().apply {
            setPadding(ctx.dp(16), ctx.dp(14), ctx.dp(16), ctx.dp(14))
            addView(TextView(context).apply {
                text = "🔗 微信 wxid ↔ 应用宝 openid"
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
                setTypeface(typeface, Typeface.BOLD)
                setTextColor(ctx.themeColor(R.color.text_primary))
            })
            addView(ctx.bodyText(
                "绑定后青龙脚本原提交 CK 的微信 wxid，网关将自动路由微信协议 wxid 至应用宝请求协议 code；如果你之前没使用微信协议仅使用应用宝时可忽略本功能。绑定关系显示在下方账号卡片内。"
            ).apply {
                setPadding(0, ctx.dp(8), 0, 0)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setLineSpacing(0f, 1.5f)
            })
        })

        host.addView(ctx.cardView().apply {
            setPadding(ctx.dp(16), ctx.dp(14), ctx.dp(16), ctx.dp(14))
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT,
            ).apply { topMargin = ctx.dp(12) }
            addView(ctx.captionText("新建双绑").apply {
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
            })
            fun pickField(placeholder: String): TextView {
                return TextView(context).apply {
                    text = placeholder
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                    setTextColor(ctx.themeColor(R.color.text_primary))
                    setPadding(ctx.dp(12), ctx.dp(12), ctx.dp(12), ctx.dp(12))
                    background = GradientDrawable().apply {
                        setColor(ctx.themeColor(R.color.chip_bg))
                        cornerRadius = ctx.dp(10).toFloat()
                    }
                    layoutParams = LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.MATCH_PARENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT,
                    ).apply { topMargin = ctx.dp(10) }
                }
            }
            protocolBindWxPickBtn = pickField("选择微信 wxid").also {
                it.setOnClickListener { showProtocolBindWxPicker() }
                addView(it)
            }
            protocolBindYybPickBtn = pickField("选择应用宝 openid").also {
                it.setOnClickListener { showProtocolBindYybPicker() }
                addView(it)
            }
            addView(ctx.primaryButton("建立双绑").apply {
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                ).apply { topMargin = ctx.dp(14) }
                setOnClickListener { submitProtocolBind() }
            })
        })

        host.addView(ctx.captionText("已绑定配对").apply {
            setPadding(ctx.dp(4), ctx.dp(16), 0, ctx.dp(8))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTypeface(typeface, Typeface.BOLD)
            setTextColor(ctx.themeColor(R.color.text_secondary))
        })
        protocolBindListHost = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            tag = "protocol_bind_list"
        }
        host.addView(protocolBindListHost)
        renderProtocolBindList()
    }

    private fun renderProtocolBindQuota() {
        // 名额信息改在扫码时展示，此处不再显示
    }

    private fun renderProtocolBindList() {
        val host = protocolBindListHost ?: contentRoot.findViewWithTag("protocol_bind_list") as? LinearLayout ?: return
        host.removeAllViews()
        val ctx = requireContext()
        if (protocolBindings.isEmpty()) {
            host.addView(ctx.cardView().apply {
                addView(ctx.bodyText("暂无绑定，可在上方选择微信与应用宝账号建立双绑。").apply {
                    gravity = Gravity.CENTER
                    setTextColor(ctx.themeColor(R.color.text_muted))
                })
            })
            return
        }
        protocolBindings.forEach { b ->
            val wx = wxDevices.firstOrNull { it.wxid == b.wxWxid }
            val yyb = yybAccounts.firstOrNull { it.openid == b.yybOpenId }
            host.addView(ctx.cardView().apply {
                setPadding(ctx.dp(14), ctx.dp(12), ctx.dp(14), ctx.dp(12))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                ).apply { bottomMargin = ctx.dp(8) }
                addView(TextView(context).apply {
                    text = "微信 · ${wx?.nickname ?: b.nickname ?: "设备"}"
                    setTypeface(typeface, Typeface.BOLD)
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                })
                addView(ctx.captionText(shortenProtocolId(b.wxWxid)).apply {
                    setPadding(0, ctx.dp(4), 0, 0)
                })
                addView(TextView(context).apply {
                    text = "↕"
                    gravity = Gravity.CENTER
                    setTextColor(ctx.themeColor(R.color.text_muted))
                    setPadding(0, ctx.dp(4), 0, ctx.dp(4))
                })
                addView(TextView(context).apply {
                    text = "应用宝 · ${yyb?.nickname ?: "账号"}"
                    setTypeface(typeface, Typeface.BOLD)
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                })
                addView(ctx.captionText(shortenProtocolId(b.yybOpenId)).apply {
                    setPadding(0, ctx.dp(4), 0, ctx.dp(8))
                })
                addView(TextView(context).apply {
                    text = "解除绑定"
                    setTextColor(Color.parseColor("#DC2626"))
                    setTypeface(typeface, Typeface.BOLD)
                    gravity = Gravity.CENTER
                    setOnClickListener {
                        confirmProtocolUnbind(b.wxWxid.orEmpty(), b.yybOpenId.orEmpty())
                    }
                })
            })
        }
    }

    private fun unboundWxForBind() = wxDevices.filter { !boundWxSet().contains(it.wxid) }
    private fun unboundYybForBind() = yybAccounts.filter { !boundOpenIdSet().contains(it.openid) }

    private fun showProtocolBindWxPicker() {
        val items = unboundWxForBind()
        if (items.isEmpty()) {
            toast("暂无可绑定的微信设备")
            return
        }
        val labels = items.map {
            val offline = if (it.online == true) "" else "（离线）"
            "${it.nickname ?: "微信设备"} · ${shortenProtocolId(it.wxid)}$offline"
        }.toTypedArray()
        AlertDialog.Builder(requireContext())
            .setTitle("选择微信 wxid")
            .setItems(labels) { _, which ->
                protocolBindSelectedWx = items[which]
                protocolBindWxPickBtn?.text = labels[which]
            }
            .show()
    }

    private fun showProtocolBindYybPicker() {
        val items = unboundYybForBind()
        if (items.isEmpty()) {
            toast("暂无可绑定的应用宝账号")
            return
        }
        val labels = items.map { acc ->
            val alive = (acc.status ?: "").lowercase() in listOf("alive", "online")
            val name = acc.nickname?.takeIf { it.isNotBlank() } ?: "应用宝账号"
            "$name · ${shortenProtocolId(acc.openid)}${if (alive) "" else "（失效）"}"
        }.toTypedArray()
        AlertDialog.Builder(requireContext())
            .setTitle("选择应用宝 openid")
            .setItems(labels) { _, which ->
                protocolBindSelectedYyb = items[which]
                protocolBindYybPickBtn?.text = labels[which]
            }
            .show()
    }

    private fun submitProtocolBind() {
        val wx = protocolBindSelectedWx
        val yyb = protocolBindSelectedYyb
        if (wx == null || yyb == null) {
            toast("请先选择微信和应用宝账号")
            return
        }
        lifecycleScope.launch {
            runCatching {
                AppServices.portalRepository.protocolBind(
                    wxWxid = wx.wxid.orEmpty(),
                    yybOpenId = yyb.openid.orEmpty(),
                    nickname = yyb.nickname.orEmpty(),
                )
            }.onSuccess {
                toast("绑定成功")
                protocolBindSelectedWx = null
                protocolBindSelectedYyb = null
                protocolBindWxPickBtn?.text = "选择微信 wxid"
                protocolBindYybPickBtn?.text = "选择应用宝 openid"
                loadProtocolBindings()
            }.onFailure {
                toast(it.message ?: "绑定失败")
                handlePortalError(it)
            }
        }
    }

    private fun renderWxProtocol() {
        contentRoot.addView(requireContext().cardView().apply {
            val titleRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT)
            }
            val titleText = TextView(context).apply {
                text = "📖 什么是微信协议？"
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            }
            val toggleIcon = TextView(context).apply {
                text = "▶"
                setTextColor(requireContext().themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            }
            titleRow.addView(titleText)
            titleRow.addView(toggleIcon)
            addView(titleRow)

            val detailContent = requireContext().bodyText("""
                微信协议是一种自动化工具，主要用于获取微信小程序的登录凭证（CK）。
                
                【主要作用】
                • 自动获取小程序CK，无需手动抓包
                • CK通常有有效期，但配合微信协议可保持永不过期
                • 支持微信协议的项目，只需提交微信ID即可自动上车
                
                【使用场景】
                如果某个项目标注"支持微信协议"，你只需要：
                1. 在此页面扫码或提交微信ID绑定设备
                2. 在项目中心选择支持微信协议的项目上车
                3. 系统会自动通过你的微信协议获取CK
                
                【注意事项】
                • 需要保持微信在线状态
                • 定期检查设备是否掉线
                • 掉线后需要重新扫码登录
            """.trimIndent()).apply {
                setLineSpacing(0f, 1.5f)
                setPadding(0, requireContext().dp(6), 0, 0)
                visibility = View.GONE
            }
            addView(detailContent)

            titleRow.setOnClickListener {
                if (detailContent.visibility == View.GONE) {
                    detailContent.visibility = View.VISIBLE
                    toggleIcon.text = "▼"
                } else {
                    detailContent.visibility = View.GONE
                    toggleIcon.text = "▶"
                }
            }
        })

        contentRoot.addView(TextView(requireContext()).apply {
            text = "💡 操作提示：点击设备卡片上的按钮可直接操作｜设备离线时可点击「唤醒」或「重登」恢复｜首次挂主设备请使用下方「📱 微信扫码登录」"
            setTextColor(Color.parseColor("#0891B2"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            setLineSpacing(0f, 1.4f)
            background = GradientDrawable().apply {
                setColor(Color.parseColor("#F0F9FF"))
                cornerRadius = requireContext().dp(8).toFloat()
                setStroke(requireContext().dp(1), Color.parseColor("#BAE6FD"))
            }
            setPadding(requireContext().dp(10), requireContext().dp(8), requireContext().dp(10), requireContext().dp(8))
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                bottomMargin = requireContext().dp(8)
            }
        })

        contentRoot.addView(requireContext().cardView().apply {
            addView(requireContext().captionText("🔄 设备监控").apply {
                textSize = 14f
                setTypeface(typeface, Typeface.BOLD)
            })
            addView(requireContext().captionText("设备已自动识别，登录微信后将自动添加到监控列表").apply {
                setPadding(0, requireContext().dp(4), 0, requireContext().dp(4))
            })

            val refreshRow = LinearLayout(requireContext()).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
            }

            deviceLoadingIndicator = ProgressBar(requireContext()).apply {
                visibility = View.GONE
                layoutParams = LinearLayout.LayoutParams(requireContext().dp(20), requireContext().dp(20)).apply {
                    marginEnd = requireContext().dp(8)
                }
            }
            refreshRow.addView(deviceLoadingIndicator)

            refreshDeviceButton = requireContext().primaryButton("🔄 刷新设备列表", ContextCompat.getColor(requireContext(), R.color.brand_secondary)).apply {
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                setOnClickListener { loadWxDevices() }
            }
            refreshRow.addView(refreshDeviceButton)
            addView(refreshRow)
        })

        val deviceContainer = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.VERTICAL
            tag = "wx_device_container"
        }
        contentRoot.addView(deviceContainer)

        contentRoot.addView(requireContext().cardView().apply {
            addView(requireContext().captionText("⚡ 快捷操作（主设备）"))
            addView(requireContext().actionGridTile(listOf(
                Triple("扫码登录", "微信扫码授权登录") { confirmWxAction("扫码登录", "确认开始微信扫码登录吗？\n\n作用：通过微信扫码绑定主设备，扫码成功后将扣除积分。\n⚠️ 有概率封号，请谨慎考虑。", "/api/portal/wx/scan-login", true) },
                Triple("唤醒登录", "唤醒已登录设备") { confirmWxAction("唤醒登录", "确认执行「唤醒登录」？\n\n作用：手机端微信在线但服务器显示掉线时，唤醒设备重新扫码确认恢复上线。\n不扣积分。", "/api/portal/wx/wake-login", false) },
                Triple("重新登录", "重新获取登录凭证") { confirmWxAction("重新登录", "确认执行「重新登录」？\n\n作用：设备掉线时重新上线，需要用手机微信扫码确认。\n不扣积分。", "/api/portal/wx/relogin", false) },
                Triple("登出设备", "安全退出当前设备") { confirmWxAction("登出设备", "确认执行「登出」？\n\n作用：退出当前登录的主设备，设备将变为离线状态。\n不扣积分。", "/api/portal/wx/logout", false) },
                Triple("删除设备", "清除数据重新扫码") { confirmWxAction("删除设备", "⚠️ 确认「删除」主设备？\n\n作用：删除主设备数据，设备将完全移除。\n⚠️ 再次上线需要重新扫码登录并扣除积分！\n\n此操作不可撤销，请确认！", "/api/portal/wx/delete", false) }
            ), ContextCompat.getColor(requireContext(), R.color.brand_primary)))
        })

        loadWxDevices()
    }

    private fun loadWxDevices() {
        lifecycleScope.launch {
            refreshDeviceButton?.apply {
                isEnabled = false
                text = "正在刷新..."
            }
            deviceLoadingIndicator?.visibility = View.VISIBLE

            runCatching { AppServices.portalRepository.fetchWxDevices() }
                .onSuccess { devices ->
                    wxDevices = devices
                    renderWxDeviceList(devices)
                }
                .onFailure {
                    renderWxDeviceList(emptyList())
                    handlePortalError(it)
                }

            refreshDeviceButton?.apply {
                isEnabled = true
                text = "✅ 已刷新"
                postDelayed({ text = "🔄 刷新设备列表" }, 1500)
            }
            deviceLoadingIndicator?.visibility = View.GONE
        }
    }

    private fun renderWxDeviceList(devices: List<com.goudong.jd.data.model.PortalWxDevice>) {
        val container = contentRoot.findViewWithTag<LinearLayout>("wx_device_container")
            ?: return
        container.removeAllViews()

        if (devices.isEmpty()) {
            container.addView(requireContext().cardView().apply {
                gravity = Gravity.CENTER
                setPadding(0, requireContext().dp(24), 0, requireContext().dp(24))
                addView(TextView(context).apply {
                    text = "📱 暂无微信协议设备"
                    setTextColor(requireContext().themeColor(R.color.text_muted))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                    gravity = Gravity.CENTER
                })
                addView(TextView(context).apply {
                    text = "首次使用请点击下方「微信扫码登录」添加主设备"
                    setTextColor(requireContext().themeColor(R.color.text_hint))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    gravity = Gravity.CENTER
                    setPadding(0, context.dp(6), 0, 0)
                })
            })
            return
        }

        devices.forEach { device ->
            container.addView(wxDeviceCard(device))
        }
    }

    private fun wxDeviceCard(device: com.goudong.jd.data.model.PortalWxDevice): View {
        val isOnline = device.online == true
        val isPrimary = device.isPrimary == true
        return requireContext().cardView().apply {
            setPadding(context.dp(16), context.dp(14), context.dp(16), context.dp(14))

            if (isPrimary) {
                background = GradientDrawable().apply {
                    setColor(requireContext().themeColor(R.color.surface_card))
                    cornerRadius = context.dp(20).toFloat()
                    setStroke(context.dp(1), Color.parseColor("#BAE6FD"))
                }
            }

            val headerRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
            }

            if (isPrimary) {
                headerRow.addView(TextView(context).apply {
                    text = "⭐ 主设备"
                    setTextColor(Color.parseColor("#0891B2"))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    setTypeface(typeface, Typeface.BOLD)
                    background = GradientDrawable().apply {
                        setColor(Color.parseColor("#E0F2FE"))
                        cornerRadius = context.dp(6).toFloat()
                    }
                    setPadding(context.dp(7), context.dp(2), context.dp(7), context.dp(2))
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        marginEnd = context.dp(6)
                    }
                })
            } else {
                headerRow.addView(TextView(context).apply {
                    text = "📋 监控"
                    setTextColor(requireContext().themeColor(R.color.text_muted))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    setTypeface(typeface, Typeface.BOLD)
                    background = GradientDrawable().apply {
                        setColor(requireContext().themeColor(R.color.chip_bg))
                        cornerRadius = context.dp(6).toFloat()
                    }
                    setPadding(context.dp(7), context.dp(2), context.dp(7), context.dp(2))
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        marginEnd = context.dp(6)
                    }
                })
            }

            headerRow.addView(TextView(context).apply {
                text = if (isOnline) "🟢 在线" else "🔴 离线"
                setTextColor(if (isOnline) requireContext().themeColor(R.color.positive) else requireContext().themeColor(R.color.negative))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setTypeface(typeface, Typeface.BOLD)
                background = GradientDrawable().apply {
                    setColor(if (isOnline) Color.parseColor("#DCFCE7") else Color.parseColor("#FEF2F2"))
                    cornerRadius = context.dp(6).toFloat()
                }
                setPadding(context.dp(7), context.dp(2), context.dp(7), context.dp(2))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    marginEnd = context.dp(6)
                }
            })

            headerRow.addView(TextView(context).apply {
                text = device.nickname ?: device.wxid ?: "未知"
                setTextColor(requireContext().themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                setTextIsSelectable(true)
            })

            if (!isPrimary) {
                headerRow.addView(TextView(context).apply {
                    text = "✕"
                    setTextColor(requireContext().themeColor(R.color.text_hint))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
                    gravity = Gravity.CENTER
                    setOnClickListener {
                        androidx.appcompat.app.AlertDialog.Builder(requireContext())
                            .setTitle("确认移除")
                            .setMessage("确认移除该微信协议设备监控吗？")
                            .setNegativeButton("取消", null)
                            .setPositiveButton("确认") { _, _ ->
                                lifecycleScope.launch {
                                    runCatching { AppServices.portalRepository.removeWxDevice(device.id) }
                                        .onSuccess {
                                            toast(it)
                                            loadWxDevices()
                                        }
                                        .onFailure { handlePortalError(it) }
                                }
                            }
                            .show()
                    }
                })
            }

            addView(headerRow)

            val detailText = buildString {
                appendLine("微信ID：${device.wxid ?: "-"}")
                appendLine("设备：${device.device ?: "-"}")
                appendLine("登录时间：${device.loginTime ?: "-"}")
                appendLine("刷新时间：${device.refreshTime ?: "-"}")
            }.trim()
            addView(requireContext().captionText(detailText).apply {
                setPadding(0, context.dp(8), 0, context.dp(4))
                setTextIsSelectable(true)
            })

            bindingByWx(device.wxid)?.let { binding ->
                val yybAcc = yybAccounts.firstOrNull { it.openid == binding.yybOpenId }
                val peerName = yybAcc?.nickname?.takeIf { it.isNotBlank() } ?: "应用宝账号"
                addView(LinearLayout(context).apply {
                    orientation = LinearLayout.HORIZONTAL
                    gravity = Gravity.CENTER_VERTICAL
                    setPadding(0, context.dp(4), 0, 0)
                    addView(requireContext().captionText("已绑定应用宝 · $peerName · ${shortenProtocolId(binding.yybOpenId)}").apply {
                        layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                    })
                    addView(TextView(context).apply {
                        text = "解绑"
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                        setTextColor(Color.parseColor("#DC2626"))
                        setOnClickListener {
                            confirmProtocolUnbind(binding.wxWxid.orEmpty(), binding.yybOpenId.orEmpty())
                        }
                    })
                })
            }

            if (!isOnline) {
                val tipLabel = if (isPrimary) "主设备" else "监控设备"
                addView(TextView(context).apply {
                    text = "⚠️ $tipLabel 已离线，可点击「唤醒」或「重登」尝试恢复"
                    setTextColor(Color.parseColor("#D97706"))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    setLineSpacing(0f, 1.3f)
                    background = GradientDrawable().apply {
                        setColor(Color.parseColor("#FEF3C7"))
                        cornerRadius = context.dp(8).toFloat()
                        setStroke(context.dp(1), Color.parseColor("#FDE68A"))
                    }
                    setPadding(context.dp(10), context.dp(6), context.dp(10), context.dp(6))
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        topMargin = context.dp(4)
                    }
                })
            }

            val btnRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(0, context.dp(10), 0, 0)
            }

            fun mkBtn(label: String, bgColor: String, textColor: Int, onClick: () -> Unit): TextView {
                return TextView(context).apply {
                    text = label
                    setTextColor(textColor)
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                    setTypeface(typeface, Typeface.BOLD)
                    background = GradientDrawable().apply {
                        setColor(Color.parseColor(bgColor))
                        cornerRadius = context.dp(8).toFloat()
                    }
                    setPadding(context.dp(10), context.dp(6), context.dp(10), context.dp(6))
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        marginEnd = context.dp(6)
                    }
                    setOnClickListener { onClick() }
                }
            }

            btnRow.addView(mkBtn("⚡ 唤醒", "#E0F2FE", Color.parseColor("#0369A1")) {
                confirmWxActionForDevice("唤醒登录", "确认对${if (isPrimary) "主设备" else "监控设备"}执行「唤醒登录」？\n\n作用：手机端微信在线但服务器显示掉线时，唤醒设备重新扫码确认恢复上线。\n不扣积分。", "/api/portal/wx/wake-login", device.wxid)
            })
            btnRow.addView(mkBtn("🔄 重登", "#F3E8FF", Color.parseColor("#7C3AED")) {
                confirmWxActionForDevice("重新登录", "确认对${if (isPrimary) "主设备" else "监控设备"}执行「重新登录」？\n\n作用：设备掉线时重新上线，需要用手机微信扫码确认。\n不扣积分。", "/api/portal/wx/relogin", device.wxid)
            })
            btnRow.addView(mkBtn("🚪 登出", "#F1F5F9", requireContext().themeColor(R.color.text_secondary)) {
                confirmWxActionForDevice("登出", "确认对${if (isPrimary) "主设备" else "监控设备"}执行「登出」？\n\n作用：退出当前登录的设备，设备将变为离线状态。\n不扣积分。", "/api/portal/wx/logout", device.wxid)
            })
            btnRow.addView(mkBtn("🗑️ 删除", "#FEF2F2", ContextCompat.getColor(context, R.color.brand_red)) {
                if (isPrimary) {
                    androidx.appcompat.app.AlertDialog.Builder(requireContext())
                        .setTitle("⚠️ 确认删除主设备")
                        .setMessage("作用：删除主设备数据，设备将完全移除。\n⚠️ 再次上线需要重新扫码登录并扣除积分！\n\n此操作不可撤销，请确认！")
                        .setNegativeButton("取消", null)
                        .setPositiveButton("确认删除") { _, _ ->
                            performWxActionForDevice("/api/portal/wx/delete", device.wxid)
                        }
                        .show()
                } else {
                    androidx.appcompat.app.AlertDialog.Builder(requireContext())
                        .setTitle("确认删除监控设备")
                        .setMessage("确认删除该监控设备？删除后需重新添加。")
                        .setNegativeButton("取消", null)
                        .setPositiveButton("确认删除") { _, _ ->
                            lifecycleScope.launch {
                                runCatching { AppServices.portalRepository.removeWxDevice(device.id) }
                                    .onSuccess {
                                        toast(it)
                                        loadWxDevices()
                                    }
                                    .onFailure { handlePortalError(it) }
                            }
                        }
                        .show()
                }
            })

            addView(btnRow)
        }
    }

    private fun confirmWxActionForDevice(title: String, message: String, path: String, wxid: String?) {
        androidx.appcompat.app.AlertDialog.Builder(requireContext())
            .setTitle(title)
            .setMessage(message)
            .setNegativeButton("取消", null)
            .setPositiveButton("确认") { _, _ ->
                if (wxid != null) {
                    performWxActionForDevice(path, wxid)
                } else {
                    performWxAction(path, false)
                }
            }
            .show()
    }

    private fun performWxActionForDevice(path: String, wxid: String?) {
        if (wxid.isNullOrBlank()) {
            toast("设备微信ID为空，无法操作")
            return
        }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.performWechatAction(path, wxid!!) }
                .onSuccess { data ->
                    if (!data.qrBase64.isNullOrBlank() && !data.uuid.isNullOrBlank()) {
                        showQrCodeDialog(data.qrBase64, data.uuid, data.message ?: "请扫码", false)
                    } else {
                        toast(data.message ?: "操作成功")
                        loadWxDevices()
                    }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun confirmWxAction(title: String, message: String, path: String, deductCoin: Boolean) {
        androidx.appcompat.app.AlertDialog.Builder(requireContext())
            .setTitle(title)
            .setMessage(message)
            .setNegativeButton("取消", null)
            .setPositiveButton("确认") { _, _ -> performWxAction(path, deductCoin) }
            .show()
    }

    private fun performWxAction(path: String, deductCoin: Boolean) {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.performWechatAction(path) }
                .onSuccess { data ->
                    if (!data.qrBase64.isNullOrBlank() && !data.uuid.isNullOrBlank()) {
                        showQrCodeDialog(data.qrBase64, data.uuid, data.message ?: "请扫码", deductCoin)
                    } else {
                        toast(data.message ?: "操作成功")
                        loadWxDevices()
                    }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun showQrCodeDialog(base64: String, uuid: String, message: String, deductCoin: Boolean) {
        wxPolling = true
        val ctx = requireContext()
        val dialogView = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            gravity = android.view.Gravity.CENTER_HORIZONTAL
            setPadding(ctx.dp(24), ctx.dp(24), ctx.dp(24), ctx.dp(24))
        }
        val qrImage = android.widget.ImageView(ctx).apply {
            layoutParams = LinearLayout.LayoutParams(ctx.dp(260), ctx.dp(260))
            scaleType = android.widget.ImageView.ScaleType.FIT_CENTER
        }
        decodeQrImage(base64)?.let { qrImage.setImageBitmap(it) }
        val stateText = ctx.captionText(message).apply {
            gravity = android.view.Gravity.CENTER
            setPadding(0, ctx.dp(16), 0, 0)
        }
        dialogView.addView(qrImage)
        dialogView.addView(stateText)
        val dialog = androidx.appcompat.app.AlertDialog.Builder(ctx)
            .setTitle("扫码登录")
            .setView(dialogView)
            .setNegativeButton("关闭") { _, _ -> wxPolling = false }
            .create()
        dialog.show()
        lifecycleScope.launch {
            var failures = 0
            while (wxPolling) {
                kotlinx.coroutines.delay(3000)
                if (!wxPolling) break
                runCatching { AppServices.portalRepository.pollWechatLogin(uuid, deductCoin) }
                    .onSuccess { result ->
                        failures = 0
                        if (result.needPoll == true) {
                            stateText.text = result.message ?: "等待扫码中"
                        } else {
                            wxPolling = false
                            dialog.dismiss()
                            toast(result.message ?: "登录成功")
                            loadWxDevices()
                        }
                    }
                    .onFailure {
                        failures++
                        if (failures >= 4) {
                            wxPolling = false
                            dialog.dismiss()
                            handlePortalError(it)
                        } else {
                            stateText.text = "服务连接中断，正在重试（$failures/3）"
                        }
                    }
            }
        }
    }

    private fun decodeQrImage(raw: String): android.graphics.Bitmap? {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return null
        val payload = when {
            trimmed.startsWith("http") -> return null
            trimmed.contains("base64,") -> trimmed.substringAfter("base64,")
            trimmed.contains(",") && trimmed.contains("data:image") -> trimmed.substringAfter(",")
            else -> trimmed
        }
        val normalized = payload.replace("\n", "").replace("\r", "").replace(" ", "")
        val padding = normalized.length % 4
        val padded = if (padding == 0) normalized else normalized + "=".repeat(4 - padding)
        return runCatching {
            val bytes = android.util.Base64.decode(padded, android.util.Base64.DEFAULT)
            android.graphics.BitmapFactory.decodeByteArray(bytes, 0, bytes.size)
        }.getOrNull()
    }

    override val innerTabCount: Int
        get() = if (::tabs.isInitialized) tabs.tabCount else 0

    override val innerTabIndex: Int
        get() = if (::tabs.isInitialized) tabs.selectedTabPosition else currentTab

    override fun selectInnerTab(index: Int) {
        if (!::tabs.isInitialized || index !in 0 until tabs.tabCount) return
        if (tabs.selectedTabPosition == index) return
        tabs.getTabAt(index)?.select()
    }

    fun consumeBackPress(): Boolean {
        if (currentTab == 2 && rushFragment.isAdded &&
            rushFragment.childFragmentManager.backStackEntryCount > 0
        ) {
            rushFragment.childFragmentManager.popBackStack()
            return true
        }
        return false
    }

    override fun onInnerSwipeBoundary(direction: Int): Boolean {
        if (currentTab == 2 && rushFragment.isAdded) {
            val backStack = rushFragment.childFragmentManager.backStackEntryCount
            if (direction < 0 && backStack > 0) {
                rushFragment.childFragmentManager.popBackStack()
                return true
            }
        }
        return false
    }

    override fun resetToInitialState() {
        if (!::tabs.isInitialized) return
        if (rushFragment.isAdded) rushFragment.popToList()
        searchQuery = ""
        searchBox?.setText("")
        selectedCategory = ""
        renderCategoryChips()
        expandedKeys.clear()
        currentTab = 0
        if (tabs.selectedTabPosition != 0) {
            tabs.getTabAt(0)?.select()
        } else {
            renderCurrentTab(forceRefresh = false)
        }
        contentScroll?.scrollTo(0, 0)
        view?.findFirstScrollView()?.scrollTo(0, 0)
        if (::swipeRefreshLayout.isInitialized) {
            swipeRefreshLayout.isRefreshing = false
        }
    }
}
