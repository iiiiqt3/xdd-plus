package com.goudong.jd.ui.jd

import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.MotionEvent
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.text.Editable
import android.text.TextWatcher
import android.widget.HorizontalScrollView
import android.widget.Spinner
import android.widget.ArrayAdapter
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.PortalJdAccount
import com.goudong.jd.data.model.PortalJdWxDevice
import com.goudong.jd.ui.common.ResultTextActivity
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.disallowAncestorsIntercept
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.applyCompactTabs
import com.goudong.jd.ui.common.InnerTabSwipeHost
import com.goudong.jd.ui.common.MainTabResettable
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.wrapMainTabSwipe
import com.goudong.jd.ui.common.primaryButton
import com.goudong.jd.ui.common.sectionTitle
import com.goudong.jd.ui.common.softCard
import com.goudong.jd.ui.common.toast
import com.goudong.jd.ui.common.WebBrowserActivity
import com.goudong.jd.ui.more.CoinLogActivity
import com.goudong.jd.data.model.AppEnvironment
import com.google.android.material.tabs.TabLayout
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch
import com.goudong.jd.ui.common.themeColor

private data class JdTaskDef(val id: String, val name: String, val icon: String, val desc: String)

class JdPortalFragment : Fragment(), InnerTabSwipeHost, MainTabResettable {
    // 主Tab: 0=查询, 1=登录, 2=京东任务
    private var mainTabIndex = 0
    // 登录子Tab: 0=短信登录, 1=应用宝刷新, 2=微信协议刷新
    private var loginSubIndex = 0

    private lateinit var mainTabs: TabLayout
    private lateinit var subTabRow: LinearLayout
    private lateinit var toolbarRow: LinearLayout
    private lateinit var contentHost: LinearLayout
    private lateinit var contentScroll: android.widget.ScrollView
    private lateinit var taskStickyBar: LinearLayout

    private var accounts: List<PortalJdAccount> = emptyList()
    private val globalTaskAccountSelection = mutableSetOf<Int>()
    private var jdProxyStatus: com.goudong.jd.data.model.PortalJdProxyStatus? = null
    private var proxyMetaView: TextView? = null
    private var proxyBadgeView: TextView? = null
    private var proxyBuyBtn: TextView? = null
    private var proxyStatsRow: LinearLayout? = null
    private var accountChipsHost: LinearLayout? = null
    private val runningTasks = mutableMapOf<String, String>()
    private val logLines = mutableListOf<String>()
    private var logJob: Job? = null
    private var initError: String? = null
    private val executeBtns = mutableMapOf<String, View>()

    private var smsPhoneInput: EditText? = null
    private var smsCodeInput: EditText? = null
    private var smsIdCardInput: EditText? = null
    private var smsIdCardBox: LinearLayout? = null
    private var smsResultText: TextView? = null
    private var wxResultText: TextView? = null
    private var wxRiskBox: LinearLayout? = null
    private var wxRiskMsg: TextView? = null
    private var wxRiskLink: TextView? = null
    private var wxRiskUrl: String? = null
    private var yybResultText: TextView? = null
    private var yybRiskBox: LinearLayout? = null
    private var yybRiskMsg: TextView? = null
    private var yybRiskLink: TextView? = null
    private var yybRiskUrl: String? = null
    private var logText: TextView? = null

    private var taskDefs: List<JdTaskDef> = emptyList()
    private var taskSearchQuery = ""
    private var taskGridHost: LinearLayout? = null
    private var taskSearchInput: EditText? = null

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        try { return buildFullView() } catch (e: Exception) {
            android.util.Log.e("JdPortal", "CRASH", e)
            initError = "${e.javaClass.simpleName}: ${e.message}"
            return TextView(requireContext()).apply { text = "出错: $initError"; setTextColor(Color.RED); setPadding(48, 48, 48, 48) }
        }
    }

    private fun buildFullView(): View {
        val wrapper = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        val (scroll, root) = requireContext().makeScrollContainer()
        contentScroll = scroll
        val dp14 = requireContext().dp(14)

        // 点击任意位置收起键盘
        scroll.setOnTouchListener { v, _ ->
            hideKeyboard(); false
        }

        // ====== 三Tab主菜单 ======
        mainTabs = TabLayout(requireContext()).apply {
            setPadding(dp14, 0, dp14, requireContext().dp(8))
            addTab(newTab().setText("查询"))
            addTab(newTab().setText("登录"))
            addTab(newTab().setText("京东任务"))
            applyCompactTabs()
            addOnTabSelectedListener(object : TabLayout.OnTabSelectedListener {
                override fun onTabSelected(tab: TabLayout.Tab) {
                    if (mainTabIndex == tab.position) return
                    mainTabIndex = tab.position
                    renderContent()
                    if (mainTabIndex == 2) loadTaskTabData()
                }
                override fun onTabUnselected(tab: TabLayout.Tab) = Unit
                override fun onTabReselected(tab: TabLayout.Tab) {
                    renderContent()
                    if (mainTabIndex == 2) loadTaskTabData()
                }
            })
        }
        wrapper.addView(mainTabs)

        // ====== 二级子菜单 + 工具栏 ======
        val subToolbarRow = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL
            setPadding(dp14, requireContext().dp(8), dp14, requireContext().dp(8))
        }
        subTabRow = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
        }
        toolbarRow = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL
        }
        subToolbarRow.addView(subTabRow)
        subToolbarRow.addView(toolbarRow)
        wrapper.addView(subToolbarRow)

        taskStickyBar = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
            setPadding(dp14, requireContext().dp(6), dp14, requireContext().dp(6))
            setBackgroundColor(requireContext().themeColor(R.color.surface_card))
        }
        setupTaskStickyBar()
        wrapper.addView(taskStickyBar)

        // ====== 内容区 ======
        contentHost = root
        scroll.layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, 0, 1f)
        wrapper.addView(scroll)

        // 布局完成后再渲染，并同步 Tab 选中态（避免 construction 阶段重入）
        wrapper.post {
            renderContent()
            mainTabs.getTabAt(mainTabIndex)?.select()
        }
        return wrapMainTabSwipe(wrapper)
    }

    override fun onResume() {
        super.onResume()
        if (initError != null) return
        try {
            if (mainTabIndex == 0) loadAccounts()
            else if (mainTabIndex == 2) loadTaskTabData()
        } catch (e: Exception) { android.util.Log.e("JdPortal", "onResume", e) }
    }

    // ==================== 内容渲染 ====================

    private fun renderContent() {
        if (!::contentHost.isInitialized || !::subTabRow.isInitialized || !::toolbarRow.isInitialized) return
        subTabRow.removeAllViews()
        toolbarRow.removeAllViews()
        contentHost.removeAllViews()
        taskStickyBar.visibility = if (mainTabIndex == 2) View.VISIBLE else View.GONE
        try {
            when (mainTabIndex) {
                0 -> renderQueryTab()
                1 -> renderLoginTab()
                2 -> renderTaskTab()
            }
        } catch (e: Exception) {
            android.util.Log.e("JdPortal", "render error", e)
            contentHost.addView(TextView(requireContext()).apply { text = "渲染出错: ${e.message}"; setTextColor(Color.RED); setPadding(requireContext().dp(14), requireContext().dp(14), requireContext().dp(14), requireContext().dp(14)) })
        }
    }

    // ==================== 查询Tab ====================

    private fun renderQueryTab() {
        toolbarRow.addView(makeSmallBtn("全部查询") { queryAllAccounts() }.apply { tag = "全部查询" })
        toolbarRow.addView(makeSmallBtn("刷新") { loadAccounts() }.apply { tag = "刷新" })
        contentHost.addView(LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL; tag = "account_list" })
        loadAccounts()
    }

    private fun defaultValidTaskSelection(): MutableSet<Int> {
        val validCount = accounts.count { it.valid }
        if (validCount == 0) return mutableSetOf()
        return mutableSetOf(0)
    }

    private fun ensureDefaultAccountSelection() {
        if (globalTaskAccountSelection.isEmpty()) {
            globalTaskAccountSelection.addAll(defaultValidTaskSelection())
        }
    }

    private fun renderAccountList(list: List<PortalJdAccount>, error: String? = null) {
        accounts = list
        ensureDefaultAccountSelection()
        renderAccountChips()
        val host = contentHost.findViewWithTag<LinearLayout>("account_list") ?: return
        host.removeAllViews()
        val ctx = requireContext()
        when {
            error != null -> host.addView(makeEmptyCard(error))
            list.isEmpty() -> host.addView(makeEmptyCard("暂无绑定的京东账号"))
            else -> {
                // 统计栏
                val validCount = list.count { it.valid }
                host.addView(LinearLayout(ctx).apply {
                    orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL
                    setPadding(0, 0, 0, ctx.dp(8))
                    addView(TextView(ctx).apply {
                        text = "有效 $validCount / 共 ${list.size} 个账号"
                        setTextColor(requireContext().themeColor(R.color.text_primary))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                        setTypeface(typeface, Typeface.BOLD)
                        layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                    })
                })
                // 两列网格
                val row = LinearLayout(ctx).apply { orientation = LinearLayout.HORIZONTAL }
                val left = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f); setPadding(0, 0, ctx.dp(4), 0) }
                val right = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f); setPadding(ctx.dp(4), 0, 0, 0) }
                row.addView(left); row.addView(right)
                list.sortedByDescending { it.valid }.forEachIndexed { i, acc ->
                    val card = buildAccountCard(acc)
                    if (i % 2 == 0) left.addView(card) else right.addView(card)
                }
                host.addView(row)
            }
        }
    }

    private fun buildAccountCard(acc: PortalJdAccount): View {
        val ctx = requireContext()
        val isPrimary = acc.valid
        return ctx.cardView().apply {
            setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
            // 头像 + 信息行
            val infoRow = LinearLayout(ctx).apply { orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL }
            // 头像圆圈
            val initial = (acc.nickname?.takeIf { it.isNotBlank() }
                ?: acc.pin?.takeIf { it.isNotBlank() }
                ?: "?").first().toString()
            infoRow.addView(TextView(ctx).apply {
                text = initial
                setTextColor(requireContext().themeColor(R.color.chip_active_text))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                layoutParams = LinearLayout.LayoutParams(ctx.dp(36), ctx.dp(36)).apply { marginEnd = ctx.dp(8) }
                background = GradientDrawable().apply { shape = GradientDrawable.OVAL; setColor(if (isPrimary) Color.parseColor("#3B82F6") else requireContext().themeColor(R.color.text_hint)) }
            })
            val textCol = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f) }
            textCol.addView(TextView(ctx).apply { text = acc.nickname ?: acc.pin ?: "未知"; setTextColor(requireContext().themeColor(R.color.text_primary)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f); setTypeface(typeface, Typeface.BOLD) })
            textCol.addView(TextView(ctx).apply { text = acc.pin ?: ""; setTextColor(requireContext().themeColor(R.color.text_hint)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f) })
            infoRow.addView(textCol)
            addView(infoRow)
            // 状态标签
            val tagText = if (isPrimary) "有效" else "失效"
            val tagColor = if (isPrimary) requireContext().themeColor(R.color.positive) else requireContext().themeColor(R.color.negative)
            val tagBg = if (isPrimary) Color.parseColor("#DCFCE7") else Color.parseColor("#FEE2E2")
            addView(TextView(ctx).apply {
                text = tagText; setTextColor(tagColor)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f); setTypeface(typeface, Typeface.BOLD)
                background = GradientDrawable().apply { setColor(tagBg); cornerRadius = ctx.dp(4).toFloat() }
                setPadding(ctx.dp(6), ctx.dp(2), ctx.dp(6), ctx.dp(2))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply { topMargin = ctx.dp(6) }
            })
            // 查询按钮
            addView(Button(ctx).apply {
                text = "查询"; setTextColor(requireContext().themeColor(R.color.chip_active_text)); setAllCaps(false)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply { setColor(Color.parseColor("#3B82F6")); cornerRadius = ctx.dp(8).toFloat() }
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, ctx.dp(36)).apply { topMargin = ctx.dp(8) }
                minimumHeight = 0; minimumWidth = 0; isAllCaps = false
                setOnClickListener { queryAccount(acc.index, this) }
            })
        }
    }

    private fun queryAllAccounts() {
        val btn = toolbarRow.findViewWithTag<TextView>("全部查询") ?: return
        showBtnLoading(btn, "全部查询")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.queryJdAccount(0) }
                .onSuccess { startActivity(ResultTextActivity.intent(requireContext(), "查询结果", it)) }
                .onFailure { handlePortalError(it) }
            hideBtnLoading(btn)
        }
    }

    private fun queryAccount(index: Int, btn: View) {
        showBtnLoading(btn, "查询")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.queryJdAccount(index) }
                .onSuccess { startActivity(ResultTextActivity.intent(requireContext(), "查询结果", it)) }
                .onFailure { handlePortalError(it) }
            hideBtnLoading(btn)
        }
    }

    private fun loadAccounts() {
        val btn = toolbarRow.findViewWithTag<TextView>("刷新")
        btn?.let { showBtnLoading(it, "刷新") }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdAccounts() }
                .onSuccess { renderAccountList(it) }
                .onFailure { renderAccountList(emptyList(), it.message); handlePortalError(it) }
            btn?.let { hideBtnLoading(it) }
        }
    }

    // ==================== 登录Tab ====================

    private fun renderLoginTab() {
        addPillTab("短信登录", loginSubIndex == 0) { loginSubIndex = 0; renderContent() }
        addPillTab("应用宝刷新", loginSubIndex == 1) { loginSubIndex = 1; renderContent() }
        addPillTab("微信协议", loginSubIndex == 2) { loginSubIndex = 2; renderContent() }
        when (loginSubIndex) {
            0 -> renderSmsPanel()
            1 -> renderYybPanel()
            2 -> renderWxPanel()
        }
    }

    private fun renderSmsPanel() {
        val ctx = requireContext()
        contentHost.addView(ctx.cardView().apply {
            setPadding(ctx.dp(16), ctx.dp(16), ctx.dp(16), ctx.dp(16))
            addView(TextView(ctx).apply { text = "短信验证码登录"; setTextColor(requireContext().themeColor(R.color.text_primary)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f); setTypeface(typeface, Typeface.BOLD) })
            addView(ctx.captionText("输入京东绑定的手机号，验证码登录后自动绑定到账号").apply { setPadding(0, ctx.dp(6), 0, ctx.dp(14)) })
            smsPhoneInput = ctx.inputField("请输入11位手机号", number = true).also { addView(it) }
            addView(spacer(12))
            addView(ctx.primaryButton("发送验证码").apply { setOnClickListener { sendSms(this) } })
            addView(spacer(14))
            smsCodeInput = ctx.inputField("请输入短信验证码", number = true).also { addView(it) }
            smsIdCardBox = LinearLayout(ctx).apply {
                orientation = LinearLayout.VERTICAL; visibility = View.GONE
                addView(spacer(14)); addView(ctx.captionText("身份证验证"))
                smsIdCardInput = ctx.inputField("身份证前两位+后四位").also { addView(it) }
            }.also { addView(it) }
            addView(spacer(14))
            addView(ctx.primaryButton("提交登录", ContextCompat.getColor(ctx, R.color.brand_secondary)).apply { tag = "提交登录"; setOnClickListener { verifySms() } })
            smsResultText = ctx.bodyText("").apply { visibility = View.GONE }.also { addView(it) }
        })
    }

    private fun sendSms(btn: View) {
        val phone = smsPhoneInput?.text?.toString().orEmpty().trim()
        if (phone.length != 11) return alert("请输入11位手机号")
        showBtnLoading(btn, "发送验证码")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.sendJdSms(phone) }
                .onSuccess { toast(it); smsIdCardBox?.visibility = View.GONE }
                .onFailure { handlePortalError(it) }
            hideBtnLoading(btn)
        }
    }

    private fun verifySms() {
        val phone = smsPhoneInput?.text?.toString().orEmpty().trim()
        val code = smsCodeInput?.text?.toString().orEmpty().trim()
        val idCard = smsIdCardInput?.text?.toString().orEmpty().trim()
        if (phone.isBlank()) return alert("请输入手机号")
        val submitBtn = contentHost.findViewWithTag<Button>("提交登录")
        submitBtn?.let { showBtnLoading(it, "提交登录") }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.verifyJdSms(phone, code, idCard) }
                .onSuccess { result ->
                    if (result.needIdVerify) { smsIdCardBox?.visibility = View.VISIBLE; toast(result.message ?: "需要身份证验证"); return@onSuccess }
                    smsIdCardBox?.visibility = View.GONE
                    smsResultText?.apply { text = result.queryResult ?: result.message ?: "登录成功"; visibility = View.VISIBLE }
                    toast(result.message ?: "登录成功"); loadAccounts()
                }
                .onFailure { handlePortalError(it) }
            submitBtn?.let { hideBtnLoading(it) }
        }
    }

    private fun renderWxPanel() {
        val ctx = requireContext()
        contentHost.addView(ctx.softCard(ContextCompat.getColor(ctx, R.color.brand_secondary)).apply {
            setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
            addView(ctx.bodyText("💡 需先在「项目-协议接入-微信协议」扫码绑定在线设备，再选择设备刷新京东 CK。").apply { setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f) })
        })
        contentHost.addView(LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.END or Gravity.CENTER_VERTICAL
            setPadding(0, ctx.dp(8), 0, ctx.dp(4))
            addView(makeSmallBtn("刷新设备") { loadWxDevices() }.apply {
                tag = "刷新设备"
                (layoutParams as LinearLayout.LayoutParams).marginStart = 0
            })
        })
        contentHost.addView(LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; tag = "wx_device_list" })
        // 风险验证区域
        wxRiskBox = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
            setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
            background = GradientDrawable().apply { setColor(Color.parseColor("#FEF3C7")); cornerRadius = ctx.dp(10).toFloat(); setStroke(ctx.dp(1), Color.parseColor("#FDE68A")) }
            wxRiskMsg = TextView(ctx).apply { setTextColor(Color.parseColor("#D97706")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f); text = "账号需要短信验证" }.also { addView(it) }
            wxRiskLink = TextView(ctx).apply { setTextColor(Color.parseColor("#2563EB")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f); paint?.isUnderlineText = true; setPadding(0, ctx.dp(6), 0, 0); setOnClickListener { wxRiskUrl?.let { url -> try { startActivity(Intent(Intent.ACTION_VIEW, android.net.Uri.parse(url))) } catch (_: Exception) {} } } }.also { addView(it) }
            val continueBtn = TextView(ctx).apply { text = "验证完成，继续刷新"; setTextColor(requireContext().themeColor(R.color.chip_active_text)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f); background = GradientDrawable().apply { setColor(Color.parseColor("#F59E0B")); cornerRadius = ctx.dp(8).toFloat() }; setPadding(ctx.dp(14), ctx.dp(8), ctx.dp(14), ctx.dp(8)); gravity = Gravity.CENTER; layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply { topMargin = ctx.dp(10) }; setOnClickListener { continueWxRisk() } }
            addView(continueBtn)
        }.also { contentHost.addView(it) }
        wxResultText = ctx.bodyText("").apply { visibility = View.GONE; setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f) }.also { contentHost.addView(it) }
        loadWxDevices()
    }

    private fun renderYybPanel() {
        val ctx = requireContext()
        contentHost.addView(ctx.softCard(ContextCompat.getColor(ctx, R.color.brand_secondary)).apply {
            setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
            addView(ctx.bodyText("💡 需先在「项目-协议接入-应用宝协议」扫码绑定账号，再选择账号刷新京东 CK。").apply { setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f) })
        })
        contentHost.addView(LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.END or Gravity.CENTER_VERTICAL
            setPadding(0, ctx.dp(8), 0, ctx.dp(4))
            addView(makeSmallBtn("刷新全部") { refreshYybAll() }.apply {
                tag = "刷新全部"
                (layoutParams as LinearLayout.LayoutParams).marginStart = 0
            })
            addView(makeSmallBtn("刷新账号") { loadYybAccounts() }.apply { tag = "刷新账号" })
        })
        contentHost.addView(LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; tag = "yyb_account_list" })
        yybRiskBox = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            visibility = View.GONE
            setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
            background = GradientDrawable().apply {
                setColor(Color.parseColor("#FEF3C7"))
                cornerRadius = ctx.dp(10).toFloat()
                setStroke(ctx.dp(1), Color.parseColor("#FDE68A"))
            }
            yybRiskMsg = TextView(ctx).apply {
                setTextColor(Color.parseColor("#D97706"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                text = "账号需要短信验证"
            }.also { addView(it) }
            yybRiskLink = TextView(ctx).apply {
                setTextColor(Color.parseColor("#2563EB"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                paint?.isUnderlineText = true
                setPadding(0, ctx.dp(6), 0, 0)
                setOnClickListener {
                    yybRiskUrl?.let { url ->
                        try {
                            startActivity(Intent(Intent.ACTION_VIEW, android.net.Uri.parse(url)))
                        } catch (_: Exception) {
                        }
                    }
                }
            }.also { addView(it) }
            addView(TextView(ctx).apply {
                text = "验证完成，继续刷新"
                setTextColor(requireContext().themeColor(R.color.chip_active_text))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#F59E0B"))
                    cornerRadius = ctx.dp(8).toFloat()
                }
                setPadding(ctx.dp(14), ctx.dp(8), ctx.dp(14), ctx.dp(8))
                gravity = Gravity.CENTER
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply { topMargin = ctx.dp(10) }
                setOnClickListener { continueYybRisk() }
            })
        }.also { contentHost.addView(it) }
        yybResultText = ctx.bodyText("").apply {
            visibility = View.GONE
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
        }.also { contentHost.addView(it) }
        loadYybAccounts()
    }

    private fun loadYybAccounts() {
        val btn = contentHost.findViewWithTag<TextView>("刷新账号")
        btn?.let { showBtnLoading(it, "刷新账号") }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdYybAccounts() }
                .onSuccess { renderYybAccounts(it) }
                .onFailure { renderYybAccounts(emptyList(), it.message); handlePortalError(it) }
            btn?.let { hideBtnLoading(it) }
        }
    }

    private fun renderYybAccounts(list: List<com.goudong.jd.data.model.PortalJdYybAccount>, error: String? = null) {
        val host = contentHost.findViewWithTag<LinearLayout>("yyb_account_list") ?: return
        host.removeAllViews()
        val ctx = requireContext()
        when {
            error != null -> host.addView(makeEmptyCard(error))
            list.isEmpty() -> host.addView(makeEmptyCard("暂无可用应用宝账号，请先到「协议接入-应用宝协议」扫码绑定"))
            else -> {
                var idx = 0
                while (idx < list.size) {
                    val pair = LinearLayout(ctx).apply {
                        orientation = LinearLayout.HORIZONTAL
                        layoutParams = LinearLayout.LayoutParams(
                            LinearLayout.LayoutParams.MATCH_PARENT,
                            LinearLayout.LayoutParams.WRAP_CONTENT
                        ).apply { bottomMargin = ctx.dp(10) }
                    }
                    val left = buildJdYybCard(list[idx])
                    left.layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.MATCH_PARENT, 1f).apply {
                        marginEnd = ctx.dp(5)
                    }
                    pair.addView(left)
                    if (idx + 1 < list.size) {
                        val right = buildJdYybCard(list[idx + 1])
                        right.layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.MATCH_PARENT, 1f).apply {
                            marginStart = ctx.dp(5)
                        }
                        pair.addView(right)
                    } else {
                        pair.addView(View(ctx).apply {
                            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.MATCH_PARENT, 1f).apply {
                                marginStart = ctx.dp(5)
                            }
                        })
                    }
                    host.addView(pair)
                    equalizeRowHeights(pair)
                    idx += 2
                }
            }
        }
    }

    private fun buildJdYybCard(acc: com.goudong.jd.data.model.PortalJdYybAccount): View {
        val ctx = requireContext()
        val st = (acc.status ?: "").lowercase()
        val alive = st == "alive" || st == "online" || st.isEmpty()
        val displayName = acc.nickname?.trim()?.takeIf { it.isNotEmpty() }
            ?: acc.openid?.trim()?.takeIf { it.isNotEmpty() }
            ?: "账号"
        val jdNick = acc.jdNickname?.trim()?.takeIf { it.isNotEmpty() }
        return ctx.cardView().apply {
            setPadding(ctx.dp(12), ctx.dp(12), ctx.dp(12), ctx.dp(12))
            addView(LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                addView(TextView(ctx).apply {
                    text = displayName
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
                    background = GradientDrawable().apply {
                        setColor(if (alive) Color.parseColor("#DCFCE7") else Color.parseColor("#FEE2E2"))
                        cornerRadius = ctx.dp(6).toFloat()
                    }
                    setPadding(ctx.dp(8), ctx.dp(3), ctx.dp(8), ctx.dp(3))
                })
            })
            addView(TextView(ctx).apply {
                text = acc.openid ?: ""
                setTextColor(ctx.themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                maxLines = 2
                minLines = 2
                ellipsize = android.text.TextUtils.TruncateAt.MIDDLE
                setPadding(0, ctx.dp(4), 0, 0)
            })
            // 始终占位，保证左右卡片高度一致
            addView(TextView(ctx).apply {
                text = if (jdNick != null) "京东 $jdNick" else " "
                visibility = if (jdNick != null) View.VISIBLE else View.INVISIBLE
                setTextColor(Color.parseColor("#EA580C"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setTypeface(typeface, Typeface.BOLD)
                maxLines = 1
                ellipsize = android.text.TextUtils.TruncateAt.END
                setPadding(0, ctx.dp(4), 0, 0)
            })
            addView(View(ctx).apply {
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    0,
                    1f
                )
            })
            addView(TextView(ctx).apply {
                text = "刷新 CK"
                setTextColor(Color.WHITE)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#14B8A6"))
                    cornerRadius = ctx.dp(8).toFloat()
                }
                setPadding(ctx.dp(10), ctx.dp(8), ctx.dp(10), ctx.dp(8))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    ctx.dp(32)
                ).apply { topMargin = ctx.dp(10) }
                setOnClickListener { refreshYyb(acc, this) }
            })
        }
    }

    private fun equalizeRowHeights(row: LinearLayout) {
        row.post {
            var maxH = 0
            for (i in 0 until row.childCount) {
                val child = row.getChildAt(i)
                if (child is LinearLayout) maxH = maxOf(maxH, child.height)
            }
            if (maxH <= 0) return@post
            for (i in 0 until row.childCount) {
                val child = row.getChildAt(i)
                if (child is LinearLayout) {
                    child.layoutParams = (child.layoutParams as LinearLayout.LayoutParams).apply {
                        height = maxH
                    }
                    child.requestLayout()
                }
            }
        }
    }

    private fun showYybRefreshResult(result: com.goudong.jd.data.model.PortalJdWxRefreshResult) {
        if (result.needRiskVerify) {
            yybRiskBox?.visibility = View.VISIBLE
            yybRiskMsg?.text = result.riskMsg ?: "账号需要短信验证"
            yybRiskUrl = result.riskUrl
            yybRiskLink?.apply {
                text = result.riskUrl ?: "验证链接"
                visibility = if (result.riskUrl.isNullOrBlank()) View.GONE else View.VISIBLE
            }
            yybResultText?.visibility = View.GONE
        } else {
            yybRiskBox?.visibility = View.GONE
            val details = result.details?.joinToString("\n").orEmpty()
            val summary = "成功 ${result.success}，失败 ${result.fail}"
            yybResultText?.apply {
                text = if (details.isBlank()) summary else "$summary\n$details"
                visibility = View.VISIBLE
            }
        }
    }

    private fun refreshYyb(account: com.goudong.jd.data.model.PortalJdYybAccount, btn: View) {
        val openid = account.openid ?: return alert("账号 OpenID 无效")
        showBtnLoading(btn, "刷新CK")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.refreshJdYyb(openid) }
                .onSuccess {
                    showYybRefreshResult(it)
                    if (it.success > 0) {
                        toast("京东 CK 刷新成功")
                        loadAccounts()
                    }
                }
                .onFailure { handlePortalError(it) }
            hideBtnLoading(btn)
        }
    }

    private fun refreshYybAll() {
        val btn = contentHost.findViewWithTag<TextView>("刷新全部") ?: return
        showBtnLoading(btn, "刷新全部")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.refreshJdYyb("all") }
                .onSuccess {
                    showYybRefreshResult(it)
                    if (it.success > 0) {
                        toast("批量刷新完成")
                        loadAccounts()
                    }
                }
                .onFailure { handlePortalError(it) }
            hideBtnLoading(btn)
        }
    }

    private fun continueYybRisk() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.continueJdYybRisk() }
                .onSuccess {
                    showYybRefreshResult(it)
                    if (!it.needRiskVerify) {
                        toast("刷新完成")
                        loadAccounts()
                    }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun loadWxDevices() {
        val btn = contentHost.findViewWithTag<TextView>("刷新设备")
        btn?.let { showBtnLoading(it, "刷新设备") }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdWxDevices() }
                .onSuccess { renderWxDevices(it) }
                .onFailure { renderWxDevices(emptyList(), it.message); handlePortalError(it) }
            btn?.let { hideBtnLoading(it) }
        }
    }

    private fun renderWxDevices(list: List<PortalJdWxDevice>, error: String? = null) {
        val host = contentHost.findViewWithTag<LinearLayout>("wx_device_list") ?: return
        host.removeAllViews()
        val ctx = requireContext()
        when {
            error != null -> host.addView(makeEmptyCard(error))
            list.isEmpty() -> host.addView(makeEmptyCard("暂无在线微信协议设备"))
            else -> {
                var idx = 0
                while (idx < list.size) {
                    val pair = LinearLayout(ctx).apply {
                        orientation = LinearLayout.HORIZONTAL
                        layoutParams = LinearLayout.LayoutParams(
                            LinearLayout.LayoutParams.MATCH_PARENT,
                            LinearLayout.LayoutParams.WRAP_CONTENT
                        ).apply { bottomMargin = ctx.dp(10) }
                    }
                    val left = buildJdWxCard(list[idx])
                    left.layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.MATCH_PARENT, 1f).apply {
                        marginEnd = ctx.dp(5)
                    }
                    pair.addView(left)
                    if (idx + 1 < list.size) {
                        val right = buildJdWxCard(list[idx + 1])
                        right.layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.MATCH_PARENT, 1f).apply {
                            marginStart = ctx.dp(5)
                        }
                        pair.addView(right)
                    } else {
                        pair.addView(View(ctx).apply {
                            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.MATCH_PARENT, 1f).apply {
                                marginStart = ctx.dp(5)
                            }
                        })
                    }
                    host.addView(pair)
                    equalizeRowHeights(pair)
                    idx += 2
                }
            }
        }
    }

    private fun buildJdWxCard(device: PortalJdWxDevice): View {
        val ctx = requireContext()
        val displayName = device.nickname?.trim()?.takeIf { it.isNotEmpty() }
            ?: device.wxid?.trim()?.takeIf { it.isNotEmpty() }
            ?: "未知设备"
        val jdNick = device.jdNickname?.trim()?.takeIf { it.isNotEmpty() }
        return ctx.cardView().apply {
            setPadding(ctx.dp(12), ctx.dp(12), ctx.dp(12), ctx.dp(12))
            addView(TextView(ctx).apply {
                text = displayName
                setTextColor(ctx.themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
                maxLines = 1
                ellipsize = android.text.TextUtils.TruncateAt.END
            })
            addView(TextView(ctx).apply {
                text = device.wxid ?: ""
                setTextColor(ctx.themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                maxLines = 2
                minLines = 2
                setPadding(0, ctx.dp(4), 0, 0)
            })
            addView(TextView(ctx).apply {
                text = if (jdNick != null) "京东 $jdNick" else " "
                visibility = if (jdNick != null) View.VISIBLE else View.INVISIBLE
                setTextColor(Color.parseColor("#EA580C"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setTypeface(typeface, Typeface.BOLD)
                maxLines = 1
                ellipsize = android.text.TextUtils.TruncateAt.END
                setPadding(0, ctx.dp(4), 0, 0)
            })
            addView(TextView(ctx).apply {
                text = device.device?.takeIf { it.isNotBlank() } ?: device.serverType ?: "微信设备"
                setTextColor(ctx.themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, ctx.dp(4), 0, 0)
            })
            addView(View(ctx).apply {
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    0,
                    1f
                )
            })
            addView(TextView(ctx).apply {
                text = "刷新 CK"
                setTextColor(Color.WHITE)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#14B8A6"))
                    cornerRadius = ctx.dp(8).toFloat()
                }
                setPadding(ctx.dp(10), ctx.dp(8), ctx.dp(10), ctx.dp(8))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    ctx.dp(32)
                ).apply { topMargin = ctx.dp(10) }
                setOnClickListener { refreshWx(device, this) }
            })
        }
    }

    private fun refreshWx(device: PortalJdWxDevice, btn: View) {
        val wxid = device.wxid ?: return alert("设备ID无效")
        showBtnLoading(btn, "刷新CK")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.refreshJdWx(wxid) }
                .onSuccess { result ->
                    if (result.needRiskVerify) {
                        wxRiskBox?.visibility = View.VISIBLE
                        wxRiskMsg?.text = result.riskMsg ?: "账号需要短信验证"
                        wxRiskUrl = result.riskUrl
                        wxRiskLink?.apply {
                            text = result.riskUrl ?: "验证链接"
                            visibility = if (result.riskUrl.isNullOrBlank()) View.GONE else View.VISIBLE
                        }
                    } else {
                        wxRiskBox?.visibility = View.GONE
                    }
                    wxResultText?.apply { text = result.details?.joinToString("\n") ?: "刷新完成"; visibility = View.VISIBLE }
                }
                .onFailure { handlePortalError(it) }
            hideBtnLoading(btn)
        }
    }

    private fun continueWxRisk() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.continueJdWxRisk() }
                .onSuccess { result ->
                    if (result.needRiskVerify) {
                        wxRiskBox?.visibility = View.VISIBLE
                        wxRiskMsg?.text = result.riskMsg ?: "账号需要短信验证"
                        wxRiskUrl = result.riskUrl
                        wxRiskLink?.apply {
                            text = result.riskUrl ?: "验证链接"
                            visibility = if (result.riskUrl.isNullOrBlank()) View.GONE else View.VISIBLE
                        }
                    } else {
                        wxRiskBox?.visibility = View.GONE
                        alert("刷新完成")
                    }
                    wxResultText?.apply { text = result.details?.joinToString("\n") ?: "刷新完成"; visibility = View.VISIBLE }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    // ==================== 京东任务Tab ====================

    private fun renderTaskTab() {
        val ctx = requireContext()
        if (mainTabIndex == 2) {
            renderAccountChips()
            renderProxyCard()
        }

        val searchRow = LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            background = GradientDrawable().apply {
                setColor(requireContext().themeColor(R.color.input_bg))
                cornerRadius = ctx.dp(10).toFloat()
                setStroke(ctx.dp(1), requireContext().themeColor(R.color.border_default))
            }
            setPadding(ctx.dp(12), ctx.dp(4), ctx.dp(8), ctx.dp(4))
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                bottomMargin = ctx.dp(10)
            }
        }
        taskSearchInput = ctx.inputField("搜索任务名称、ID…").apply {
            background = null
            setPadding(0, ctx.dp(8), 0, ctx.dp(8))
            setText(taskSearchQuery)
            addTextChangedListener(object : TextWatcher {
                override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) = Unit
                override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) = Unit
                override fun afterTextChanged(s: Editable?) { onTaskSearch(s?.toString().orEmpty()) }
            })
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
        }
        searchRow.addView(taskSearchInput)
        val clearBtn = TextView(ctx).apply {
            text = "✕"
            visibility = if (taskSearchQuery.isNotBlank()) View.VISIBLE else View.GONE
            setTextColor(requireContext().themeColor(R.color.text_muted))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
            setPadding(ctx.dp(8), ctx.dp(6), ctx.dp(4), ctx.dp(6))
            setOnClickListener {
                taskSearchInput?.setText("")
                onTaskSearch("")
            }
            tag = "task_search_clear"
        }
        searchRow.addView(clearBtn)
        contentHost.addView(searchRow)

        taskGridHost = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            tag = "task_grid_host"
        }
        contentHost.addView(taskGridHost)

        contentHost.addView(ctx.cardView().apply {
            setPadding(ctx.dp(10), ctx.dp(8), ctx.dp(10), ctx.dp(8))
            val hdr = LinearLayout(ctx).apply { orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL }
            hdr.addView(TextView(ctx).apply { text = "📋 执行日志"; setTextColor(requireContext().themeColor(R.color.text_secondary)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f); setTypeface(typeface, Typeface.BOLD); layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f) })
            hdr.addView(makeSmallBtn("清空") { logLines.clear(); refreshLogView() })
            addView(hdr)
            logText = ctx.bodyText("暂无日志").apply { setPadding(0, ctx.dp(4), 0, 0); setBackgroundColor(requireContext().themeColor(R.color.input_bg)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f) }.also { addView(it) }
        })

        loadTaskTabData()
    }

    private fun setupTaskStickyBar() {
        val ctx = requireContext()
        val accountRow = LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        accountRow.addView(TextView(ctx).apply {
            text = "执行账号"
            setTextColor(requireContext().themeColor(R.color.text_secondary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            setTypeface(typeface, Typeface.BOLD)
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginEnd = ctx.dp(8)
            }
        })
        accountRow.addView(HorizontalScrollView(ctx).apply {
            tag = "account_chips_scroll"
            isHorizontalScrollBarEnabled = false
            overScrollMode = View.OVER_SCROLL_ALWAYS
            isFillViewport = false
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            accountChipsHost = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                tag = "account_chips_host"
                layoutParams = ViewGroup.LayoutParams(
                    ViewGroup.LayoutParams.WRAP_CONTENT,
                    ViewGroup.LayoutParams.WRAP_CONTENT,
                )
            }
            addView(accountChipsHost)
            setOnTouchListener { v, event ->
                when (event.actionMasked) {
                    MotionEvent.ACTION_DOWN, MotionEvent.ACTION_MOVE ->
                        v.disallowAncestorsIntercept(true)
                    MotionEvent.ACTION_UP, MotionEvent.ACTION_CANCEL ->
                        v.disallowAncestorsIntercept(false)
                }
                false
            }
        })
        accountRow.addView(TextView(ctx).apply {
            text = "滑动"
            setTextColor(requireContext().themeColor(R.color.text_muted))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 9f)
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginStart = ctx.dp(4)
            }
        })
        taskStickyBar.addView(accountRow)

        val divider = View(ctx).apply {
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, ctx.dp(1)).apply {
                topMargin = ctx.dp(6)
                bottomMargin = ctx.dp(6)
            }
            setBackgroundColor(requireContext().themeColor(R.color.border_default))
        }
        taskStickyBar.addView(divider)
        taskStickyBar.addView(buildCompactProxyCard())
    }

    private fun buildCompactProxyCard(): View {
        val ctx = requireContext()
        return LinearLayout(ctx).apply {
            tag = "proxy_card"
            orientation = LinearLayout.VERTICAL

            val infoRow = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
            }
            infoRow.addView(TextView(ctx).apply {
                text = "任务代理"
                setTextColor(requireContext().themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setTypeface(typeface, Typeface.BOLD)
            })
            proxyBadgeView = TextView(ctx).apply {
                text = "未开通"
                setTextColor(Color.parseColor("#6B7280"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 9f)
                setPadding(ctx.dp(6), ctx.dp(2), ctx.dp(6), ctx.dp(2))
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#F3F4F6"))
                    cornerRadius = ctx.dp(8).toFloat()
                }
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    marginStart = ctx.dp(6)
                }
            }
            infoRow.addView(proxyBadgeView)

            proxyStatsRow = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                visibility = View.GONE
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    marginStart = ctx.dp(8)
                }
            }
            proxyStatsRow?.addView(makeProxyStat(ctx, "到期", "-", "proxy_expire"))
            proxyStatsRow?.addView(makeProxyStat(ctx, "积分", "-", "proxy_coin"))
            infoRow.addView(proxyStatsRow)
            addView(infoRow)

            val actionRow = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(0, ctx.dp(6), 0, 0)
            }
            proxyMetaView = TextView(ctx).apply {
                text = "加载中..."
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f)
                maxLines = 1
                ellipsize = android.text.TextUtils.TruncateAt.END
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            }
            actionRow.addView(proxyMetaView)

            val buyRow = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                tag = "proxy_buy_row"
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    marginStart = ctx.dp(8)
                }
            }
            val monthLabels = listOf("1个月", "3个月", "6个月", "12个月")
            val months = Spinner(ctx).apply {
                adapter = object : ArrayAdapter<String>(
                    ctx,
                    android.R.layout.simple_spinner_item,
                    monthLabels,
                ) {
                    override fun getView(position: Int, convertView: View?, parent: ViewGroup): View {
                        val view = super.getView(position, convertView, parent) as TextView
                        view.setTextColor(requireContext().themeColor(R.color.text_primary))
                        view.setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                        view.gravity = Gravity.CENTER_VERTICAL or Gravity.START
                        return view
                    }

                    override fun getDropDownView(position: Int, convertView: View?, parent: ViewGroup): View {
                        val view = super.getDropDownView(position, convertView, parent) as TextView
                        view.setTextColor(requireContext().themeColor(R.color.text_primary))
                        view.setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                        view.setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
                        return view
                    }
                }
                layoutParams = LinearLayout.LayoutParams(ctx.dp(88), LinearLayout.LayoutParams.WRAP_CONTENT)
                tag = "proxy_months"
            }
            buyRow.addView(months)
            proxyBuyBtn = TextView(ctx).apply {
                text = "购买"
                setTextColor(Color.WHITE)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                gravity = Gravity.CENTER
                minWidth = ctx.dp(52)
                setPadding(ctx.dp(12), ctx.dp(7), ctx.dp(12), ctx.dp(7))
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#3B82F6"))
                    cornerRadius = ctx.dp(6).toFloat()
                }
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    marginStart = ctx.dp(6)
                }
                setOnClickListener { buyJdProxy(this) }
            }
            buyRow.addView(proxyBuyBtn)
            actionRow.addView(buyRow)
            addView(actionRow)
        }
    }

    private fun makeProxyStat(ctx: android.content.Context, label: String, value: String, viewTag: String): View {
        return LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            tag = viewTag
            gravity = Gravity.CENTER_VERTICAL
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginEnd = ctx.dp(8)
            }
            addView(TextView(ctx).apply {
                text = "$label "
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 9f)
            })
            addView(TextView(ctx).apply {
                text = value
                setTextColor(requireContext().themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f)
                setTypeface(typeface, Typeface.BOLD)
            })
        }
    }

    private fun renderAccountChips() {
        val host = accountChipsHost ?: taskStickyBar.findViewWithTag("account_chips_host") as? LinearLayout ?: return
        host.removeAllViews()
        val ctx = requireContext()
        val prev = globalTaskAccountSelection.toSet()
        if (prev.isEmpty()) globalTaskAccountSelection.add(0)

        fun addChip(label: String, value: Int) {
            val selected = globalTaskAccountSelection.contains(value)
            host.addView(TextView(ctx).apply {
                text = label
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setTextColor(if (selected) Color.WHITE else requireContext().themeColor(R.color.text_secondary))
                setPadding(ctx.dp(10), ctx.dp(6), ctx.dp(10), ctx.dp(6))
                background = GradientDrawable().apply {
                    setColor(if (selected) Color.parseColor("#3B82F6") else Color.parseColor("#F3F4F6"))
                    cornerRadius = ctx.dp(14).toFloat()
                }
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    marginEnd = ctx.dp(6)
                }
                setOnClickListener {
                    if (value == 0) {
                        globalTaskAccountSelection.clear()
                        if (!selected) globalTaskAccountSelection.add(0)
                    } else {
                        globalTaskAccountSelection.remove(0)
                        if (selected) globalTaskAccountSelection.remove(value) else globalTaskAccountSelection.add(value)
                        if (globalTaskAccountSelection.isEmpty()) globalTaskAccountSelection.add(0)
                    }
                    renderAccountChips()
                }
            })
        }

        addChip("所有有效账号", 0)
        var validIdx = 1
        accounts.filter { it.valid }.forEach { acc ->
            addChip(acc.nickname ?: acc.pin ?: "账号$validIdx", validIdx)
            validIdx++
        }
        val accountScroll = taskStickyBar.findViewWithTag<HorizontalScrollView>("account_chips_scroll")
        accountScroll?.post { accountScroll.scrollTo(0, 0) }
    }

    private fun renderProxyCard() {
        val st = jdProxyStatus
        val card = taskStickyBar.findViewWithTag<View>("proxy_card")
        val monthly = st?.monthlyCoin ?: 0
        val active = st?.active == true
        val ready = st?.proxyReady == true

        proxyBadgeView?.apply {
            text = if (active) "已开通" else "未开通"
            setTextColor(if (active) Color.parseColor("#047857") else Color.parseColor("#6B7280"))
            background = GradientDrawable().apply {
                setColor(if (active) Color.parseColor("#D1FAE5") else Color.parseColor("#F3F4F6"))
                cornerRadius = requireContext().dp(10).toFloat()
            }
        }

        if (!ready || monthly <= 0) {
            proxyStatsRow?.visibility = View.GONE
            proxyMetaView?.text = if (monthly <= 0) "管理员尚未开放代理购买" else "管理员尚未配置任务代理，暂不可购买"
            card?.findViewWithTag<View>("proxy_buy_row")?.visibility = View.GONE
            return
        }

        proxyStatsRow?.visibility = View.VISIBLE
        (proxyStatsRow?.findViewWithTag<LinearLayout>("proxy_expire")?.getChildAt(1) as? TextView)?.text =
            if (active) st?.expireAt ?: "-" else "未开通"
        (proxyStatsRow?.findViewWithTag<LinearLayout>("proxy_coin")?.getChildAt(1) as? TextView)?.text =
            (st?.userCoin ?: 0).toString()
        proxyMetaView?.text = if (active) "走代理" else "直连 · $monthly 积分/月"
        card?.findViewWithTag<View>("proxy_buy_row")?.visibility = View.VISIBLE
        proxyBuyBtn?.text = if (active) "续费" else "购买"
    }

    private fun buyJdProxy(btn: View) {
        val card = taskStickyBar.findViewWithTag<View>("proxy_card") ?: return
        val spinner = card.findViewWithTag<Spinner>("proxy_months") ?: return
        val months = when (spinner.selectedItemPosition) {
            1 -> 3
            2 -> 6
            3 -> 12
            else -> 1
        }
        val st = jdProxyStatus
        val monthly = st?.monthlyCoin ?: 0
        val need = monthly * months
        if (monthly <= 0) return toast("代理订阅暂未开放")
        if ((st?.userCoin ?: 0) < need) return toast("积分不足，需要 $need 积分")
        android.app.AlertDialog.Builder(requireContext())
            .setTitle(if (st?.active == true) "续费任务代理" else "购买任务代理")
            .setMessage("确认购买 $months 个月任务代理？将扣除 $need 积分。")
            .setPositiveButton("确认") { _, _ ->
                val isRenew = st?.active == true
                showBtnLoading(btn, proxyBuyBtn?.text?.toString() ?: "购买")
                lifecycleScope.launch {
                    runCatching { AppServices.portalRepository.buyJdProxy(months) }
                        .onSuccess { status ->
                            jdProxyStatus = status
                            renderProxyCard()
                            showProxyPurchaseSuccess(status, months, need, isRenew)
                        }
                        .onFailure { handlePortalError(it) }
                    hideBtnLoading(btn)
                }
            }
            .setNegativeButton("取消", null)
            .show()
    }

    private fun showProxyPurchaseSuccess(
        status: com.goudong.jd.data.model.PortalJdProxyStatus,
        months: Int,
        coinSpent: Int,
        isRenew: Boolean,
    ) {
        val expire = status.expireAt?.take(10) ?: "-"
        val action = if (isRenew) "续费" else "购买"
        alert(
            "已扣除 $coinSpent 积分，${action} $months 个月任务代理。\n到期时间：$expire\n执行任务将自动使用代理线路。",
            "购买成功",
        )
    }

    private fun onTaskSearch(value: String) {
        taskSearchQuery = value.trim().lowercase()
        contentHost.findViewWithTag<TextView>("task_search_clear")?.visibility =
            if (taskSearchQuery.isNotBlank()) View.VISIBLE else View.GONE
        renderTaskGrid()
    }

    private fun getFilteredTaskDefs(): List<JdTaskDef> {
        if (taskSearchQuery.isBlank()) return taskDefs
        return taskDefs.filter { task ->
            task.name.lowercase().contains(taskSearchQuery)
                || task.id.lowercase().contains(taskSearchQuery)
                || task.desc.lowercase().contains(taskSearchQuery)
        }
    }

    private fun loadTaskTabData() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdAccounts() }
                .onSuccess {
                    accounts = it
                    ensureDefaultAccountSelection()
                    renderAccountChips()
                }
                .onFailure { accounts = emptyList() }
            runCatching { AppServices.portalRepository.fetchJdTasks() }
                .onSuccess { list ->
                    taskDefs = list.mapNotNull { item ->
                        val id = item.id?.trim().orEmpty()
                        if (id.isEmpty()) null
                        else JdTaskDef(
                            id = id,
                            name = item.name?.ifBlank { id } ?: id,
                            icon = "⚡",
                            desc = if (item.coin > 0) "每账号扣 ${item.coin} 积分" else "免费",
                        )
                    }
                }
                .onFailure { taskDefs = emptyList() }
            runCatching { AppServices.portalRepository.fetchJdProxyStatus() }
                .onSuccess {
                    jdProxyStatus = it
                    renderProxyCard()
                }
                .onFailure {
                    jdProxyStatus = null
                    renderProxyCard()
                }
            renderTaskGrid()
        }
    }

    private fun renderTaskGrid() {
        val host = taskGridHost ?: return
        val ctx = requireContext()
        host.removeAllViews()

        val tasks = getFilteredTaskDefs()
        when {
            tasks.isEmpty() && taskSearchQuery.isNotBlank() -> {
                host.addView(ctx.bodyText("未找到匹配「$taskSearchQuery」的任务").apply {
                    setTextColor(requireContext().themeColor(R.color.text_muted))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                    gravity = Gravity.CENTER
                    setPadding(0, ctx.dp(16), 0, ctx.dp(16))
                })
            }
            tasks.isEmpty() -> {
                host.addView(ctx.bodyText("暂无可用任务，请联系管理员在后台启用").apply {
                    setTextColor(requireContext().themeColor(R.color.text_muted))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                    setPadding(0, ctx.dp(8), 0, 0)
                })
            }
            else -> {
                val row = LinearLayout(ctx).apply { orientation = LinearLayout.HORIZONTAL }
                val left = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f); setPadding(0, 0, ctx.dp(4), 0) }
                val right = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f); setPadding(ctx.dp(4), 0, 0, 0) }
                row.addView(left)
                row.addView(right)
                host.addView(row)
                tasks.forEachIndexed { i, task ->
                    val card = buildTaskCard(task)
                    if (i % 2 == 0) left.addView(card) else right.addView(card)
                }
            }
        }
        restoreRunningTaskBtns()
    }

    private fun restoreRunningTaskBtns() {
        runningTasks.keys.forEach { taskId ->
            val btn = executeBtns[taskId] as? TextView ?: return@forEach
            btn.text = "停止"
            (btn.background as? GradientDrawable)?.setColor(requireContext().themeColor(R.color.brand_red))
        }
    }

    private fun buildTaskCard(task: JdTaskDef): View {
        val ctx = requireContext()
        val isRunning = runningTasks.containsKey(task.id)
        return ctx.cardView().apply {
            setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
            addView(TextView(ctx).apply { text = "${task.icon} ${task.name}"; setTextColor(requireContext().themeColor(R.color.text_primary)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f); setTypeface(typeface, Typeface.BOLD) })
            addView(ctx.captionText(task.desc).apply { setPadding(0, ctx.dp(2), 0, ctx.dp(8)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f) })
            addView(TextView(ctx).apply {
                text = "ID: ${task.id}"
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f)
                setPadding(0, 0, 0, ctx.dp(8))
            })
            val btnColor = if (isRunning) requireContext().themeColor(R.color.brand_red) else Color.parseColor("#3B82F6")
            val execBtn = TextView(ctx).apply {
                text = if (isRunning) "停止" else "执行"; setTextColor(requireContext().themeColor(R.color.chip_active_text)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply { setColor(btnColor); cornerRadius = ctx.dp(8).toFloat() }
                gravity = Gravity.CENTER; setPadding(0, ctx.dp(6), 0, ctx.dp(6))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, ctx.dp(32))
                setOnClickListener { if (isRunning) stopTask(task) else executeTask(task) }
            }
            executeBtns[task.id] = execBtn
            addView(execBtn)
        }
    }

    private fun globalAccountIndexes(): List<Int> {
        val selection = globalTaskAccountSelection.toSet()
        if (selection.isEmpty() || selection.contains(0)) return listOf(0)
        return selection.sorted()
    }

    private fun executeTask(task: JdTaskDef) {
        val indices = globalAccountIndexes()
        if (indices.isEmpty()) return alert("请至少选择一个账号")
        if (jdProxyStatus?.active == true) {
            appendLog("[${task.name}] 已开通任务代理，本次执行将使用代理")
        }
        appendLog("[${task.name}] 开始执行...")
        val btn = executeBtns[task.id]
        btn?.let { showBtnLoading(it, "执行") }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.executeJdTask(task.id, task.name, indices) }
                .onSuccess { result ->
                    val logId = result.taskId
                    if (logId == null) { appendLog("[${task.name}] 启动失败"); btn?.let { hideBtnLoading(it) }; return@onSuccess }
                    appendLog("[${task.name}] 已启动，连接日志流...")
                    runningTasks[task.id] = logId
                    btn?.let { hideBtnLoading(it) }
                    (btn as? TextView)?.text = "停止"
                    (btn?.background as? GradientDrawable)?.setColor(requireContext().themeColor(R.color.brand_red))
                    logJob?.cancel()
                    logJob = lifecycleScope.launch {
                        AppServices.portalRepository.streamJdTaskLogs(
                            taskId = logId,
                            onLine = { line -> appendLog("[${task.name}] $line") },
                            onDone = {
                                appendLog("[${task.name}] ✅ 完成")
                                runningTasks.remove(task.id)
                                (executeBtns[task.id] as? TextView)?.text = "执行"
                                (executeBtns[task.id]?.background as? GradientDrawable)?.setColor(Color.parseColor("#3B82F6"))
                            },
                            onError = { err ->
                                appendLog("[${task.name}] 错误: ${err.message}", true)
                                runningTasks.remove(task.id)
                                (executeBtns[task.id] as? TextView)?.text = "执行"
                                (executeBtns[task.id]?.background as? GradientDrawable)?.setColor(Color.parseColor("#3B82F6"))
                            },
                        )
                    }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun stopTask(task: JdTaskDef) {
        val logId = runningTasks[task.id] ?: return
        val btn = executeBtns[task.id]
        btn?.let { showBtnLoading(it, "停止") }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.stopJdTask(logId) }
            logJob?.cancel(); runningTasks.remove(task.id); appendLog("[${task.name}] ⚠️ 已停止")
            btn?.let { hideBtnLoading(it) }
        }
    }

    // ==================== 工具方法 ====================

    private fun addPillTab(label: String, active: Boolean, onClick: () -> Unit) {
        subTabRow.addView(TextView(requireContext()).apply {
            text = label
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            setTypeface(typeface, if (active) Typeface.BOLD else Typeface.NORMAL)
            setTextColor(
                if (active) ContextCompat.getColor(context, R.color.brand_primary)
                else requireContext().themeColor(R.color.text_muted)
            )
            gravity = Gravity.CENTER
            maxLines = 1
            ellipsize = android.text.TextUtils.TruncateAt.END
            setPadding(context.dp(8), context.dp(8), context.dp(8), context.dp(8))
            background = GradientDrawable().apply {
                setColor(if (active) Color.parseColor("#EFF6FF") else Color.TRANSPARENT)
                cornerRadius = context.dp(8).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                marginEnd = context.dp(4)
            }
            setOnClickListener { onClick() }
        })
    }

    private fun makeSmallBtn(label: String, onClick: () -> Unit): TextView {
        return TextView(requireContext()).apply {
            text = label; setTextColor(requireContext().themeColor(R.color.text_muted)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            background = GradientDrawable().apply { setColor(requireContext().themeColor(R.color.chip_bg)); cornerRadius = requireContext().dp(6).toFloat() }
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, requireContext().dp(26)).apply { marginStart = requireContext().dp(6) }
            gravity = Gravity.CENTER
            setPadding(requireContext().dp(10), 0, requireContext().dp(10), 0)
            setOnClickListener { onClick() }
        }
    }

    private fun makeEmptyCard(message: String): View {
        val ctx = requireContext()
        return ctx.cardView().apply { setPadding(0, ctx.dp(20), 0, ctx.dp(20)); gravity = Gravity.CENTER
            addView(ctx.bodyText(message).apply { textSize = 13f; gravity = Gravity.CENTER; setTextColor(requireContext().themeColor(R.color.text_muted)) })
        }
    }

    private fun spacer(dp: Int): View = View(requireContext()).apply { layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, requireContext().dp(dp)) }

    private fun appendLog(line: String, @Suppress("UNUSED_PARAMETER") isError: Boolean = false) {
        logLines.add(line); if (logLines.size > 200) logLines.removeAt(0); refreshLogView()
    }

    private fun refreshLogView() { logText?.text = if (logLines.isEmpty()) "暂无日志" else logLines.joinToString("\n") }

    private fun hideKeyboard() {
        val imm = requireContext().getSystemService(android.content.Context.INPUT_METHOD_SERVICE) as? android.view.inputmethod.InputMethodManager
        imm?.hideSoftInputFromWindow(requireView().windowToken, 0)
    }

    private fun showBtnLoading(v: View, originalText: String) {
        v.tag = v.tag ?: originalText  // 只在第一次保存原始文字
        when (v) {
            is TextView -> { v.text = "加载中..."; v.isEnabled = false; v.alpha = 0.6f }
            is Button -> { v.text = "加载中..."; v.isEnabled = false; v.alpha = 0.6f }
        }
    }

    private fun hideBtnLoading(v: View) {
        val orig = v.tag?.toString() ?: ""
        when (v) {
            is TextView -> { v.text = orig; v.isEnabled = true; v.alpha = 1f }
            is Button -> { v.text = orig; v.isEnabled = true; v.alpha = 1f }
        }
    }

    override val innerTabCount: Int
        get() = if (::mainTabs.isInitialized) mainTabs.tabCount else 0

    override val innerTabIndex: Int
        get() = if (::mainTabs.isInitialized) mainTabs.selectedTabPosition else mainTabIndex

    override fun selectInnerTab(index: Int) {
        if (!::mainTabs.isInitialized || index !in 0 until mainTabs.tabCount) return
        if (mainTabs.selectedTabPosition == index) return
        mainTabs.getTabAt(index)?.select()
    }

    override fun resetToInitialState() {
        if (!::mainTabs.isInitialized) return
        hideKeyboard()
        mainTabIndex = 0
        loginSubIndex = 0
        mainTabs.getTabAt(0)?.select()
        renderContent()
        if (::contentScroll.isInitialized) {
            contentScroll.scrollTo(0, 0)
        }
    }
}
