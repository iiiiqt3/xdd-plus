package com.goudong.jd.ui.jd

import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
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
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.makeScrollContainer
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

private data class JdTaskDef(val id: String, val name: String, val icon: String, val desc: String)

class JdPortalFragment : Fragment() {
    // 主Tab: 0=查询, 1=登录, 2=京东任务
    private var mainTabIndex = 0
    // 登录子Tab: 0=短信登录, 1=协议刷新
    private var loginSubIndex = 0

    private lateinit var mainTabs: TabLayout
    private lateinit var subTabRow: LinearLayout
    private lateinit var toolbarRow: LinearLayout
    private lateinit var contentHost: LinearLayout

    private var accounts: List<PortalJdAccount> = emptyList()
    private val taskSelections = mutableMapOf<String, MutableSet<Int>>()
    private val runningTasks = mutableMapOf<String, String>()
    private val logLines = mutableListOf<String>()
    private var logJob: Job? = null
    private var initError: String? = null
    private val selectedLabels = mutableMapOf<String, TextView>()
    private val executeBtns = mutableMapOf<String, View>()

    private var smsPhoneInput: EditText? = null
    private var smsCodeInput: EditText? = null
    private var smsIdCardInput: EditText? = null
    private var smsIdCardBox: LinearLayout? = null
    private var smsResultText: TextView? = null
    private var wxResultText: TextView? = null
    private var logText: TextView? = null

    private val taskDefs = listOf(
        JdTaskDef("plantBean", "种豆得豆", "🫘", "种豆得豆任务"),
        JdTaskDef("dwapp", "话费积分", "📱", "话费积分签到"),
        JdTaskDef("price", "一键保价", "💰", "自动保价退款"),
        JdTaskDef("autoEval", "一键评价", "⭐", "自动评价订单"),
        JdTaskDef("insight", "问卷调查", "📝", "问卷调查得豆"),
    )

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
            addOnTabSelectedListener(object : TabLayout.OnTabSelectedListener {
                override fun onTabSelected(tab: TabLayout.Tab) {
                    mainTabIndex = tab.position
                    renderContent()
                }
                override fun onTabUnselected(tab: TabLayout.Tab) = Unit
                override fun onTabReselected(tab: TabLayout.Tab) { renderContent() }
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

        // ====== 内容区 ======
        contentHost = root
        scroll.layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, 0, 1f)
        wrapper.addView(scroll)

        // 所有字段初始化完成后才触发tab选中
        mainTabs.getTabAt(mainTabIndex)?.select()
        return wrapper
    }

    override fun onResume() {
        super.onResume()
        if (initError != null) return
        try { if (mainTabIndex == 0) loadAccounts() } catch (e: Exception) { android.util.Log.e("JdPortal", "onResume", e) }
    }

    // ==================== 内容渲染 ====================

    private fun renderContent() {
        subTabRow.removeAllViews()
        toolbarRow.removeAllViews()
        contentHost.removeAllViews()
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

    private fun renderAccountList(list: List<PortalJdAccount>, error: String? = null) {
        accounts = list
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
                        setTextColor(Color.parseColor("#0F172A"))
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
            val initial = (acc.nickname ?: acc.pin ?: "?").first().toString()
            infoRow.addView(TextView(ctx).apply {
                text = initial
                setTextColor(Color.WHITE)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                layoutParams = LinearLayout.LayoutParams(ctx.dp(36), ctx.dp(36)).apply { marginEnd = ctx.dp(8) }
                background = GradientDrawable().apply { shape = GradientDrawable.OVAL; setColor(if (isPrimary) Color.parseColor("#3B82F6") else Color.parseColor("#94A3B8")) }
            })
            val textCol = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f) }
            textCol.addView(TextView(ctx).apply { text = acc.nickname ?: acc.pin ?: "未知"; setTextColor(Color.parseColor("#0F172A")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f); setTypeface(typeface, Typeface.BOLD) })
            textCol.addView(TextView(ctx).apply { text = acc.pin ?: ""; setTextColor(Color.parseColor("#94A3B8")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f) })
            infoRow.addView(textCol)
            addView(infoRow)
            // 状态标签
            val tagText = if (isPrimary) "有效" else "失效"
            val tagColor = if (isPrimary) Color.parseColor("#16A34A") else Color.parseColor("#DC2626")
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
                text = "查询"; setTextColor(Color.WHITE); setAllCaps(false)
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
        addPillTab("协议刷新", loginSubIndex == 1) { loginSubIndex = 1; renderContent() }
        when (loginSubIndex) {
            0 -> renderSmsPanel()
            1 -> renderWxPanel()
        }
    }

    private fun renderSmsPanel() {
        val ctx = requireContext()
        contentHost.addView(ctx.cardView().apply {
            setPadding(ctx.dp(16), ctx.dp(16), ctx.dp(16), ctx.dp(16))
            addView(TextView(ctx).apply { text = "短信验证码登录"; setTextColor(Color.parseColor("#0F172A")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f); setTypeface(typeface, Typeface.BOLD) })
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
            addView(ctx.bodyText("💡 需先在「更多-微信协议」扫码绑定在线设备，再选择设备刷新京东 CK。").apply { setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f) })
        })
        toolbarRow.addView(makeSmallBtn("刷新设备") { loadWxDevices() })
        contentHost.addView(LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; tag = "wx_device_list" })
        wxResultText = ctx.bodyText("").apply { visibility = View.GONE; setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f) }.also { contentHost.addView(it) }
        loadWxDevices()
    }

    private fun loadWxDevices() {
        val btn = toolbarRow.findViewWithTag<TextView>("刷新设备")
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
                val row = LinearLayout(ctx).apply { orientation = LinearLayout.HORIZONTAL }
                val left = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f); setPadding(0, 0, ctx.dp(4), 0) }
                val right = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f); setPadding(ctx.dp(4), 0, 0, 0) }
                row.addView(left); row.addView(right)
                list.forEachIndexed { i, d ->
                    val card = ctx.cardView().apply {
                        setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
                        addView(TextView(ctx).apply { text = "🟢 在线"; setTextColor(Color.parseColor("#16A34A")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f); setTypeface(typeface, Typeface.BOLD); background = GradientDrawable().apply { setColor(Color.parseColor("#DCFCE7")); cornerRadius = ctx.dp(4).toFloat() }; setPadding(ctx.dp(6), ctx.dp(2), ctx.dp(6), ctx.dp(2)) })
                        addView(TextView(ctx).apply { text = d.nickname ?: d.wxid ?: "设备"; setTextColor(Color.parseColor("#0F172A")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f); setTypeface(typeface, Typeface.BOLD); setPadding(0, ctx.dp(6), 0, 0) })
                        addView(TextView(ctx).apply { text = d.wxid ?: ""; setTextColor(Color.parseColor("#94A3B8")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f) })
                        addView(TextView(ctx).apply { text = "刷新CK"; setTextColor(Color.parseColor("#3B82F6")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f); background = GradientDrawable().apply { setColor(Color.parseColor("#EFF6FF")); cornerRadius = ctx.dp(8).toFloat() }; setPadding(ctx.dp(10), ctx.dp(4), ctx.dp(10), ctx.dp(4)); gravity = Gravity.CENTER; layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, ctx.dp(28)).apply { topMargin = ctx.dp(6) }; setOnClickListener { refreshWx(d, this) } })
                    }
                    if (i % 2 == 0) left.addView(card) else right.addView(card)
                }
                host.addView(row)
            }
        }
    }

    private fun refreshWx(device: PortalJdWxDevice, btn: View) {
        val wxid = device.wxid ?: return alert("设备ID无效")
        showBtnLoading(btn, "刷新CK")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.refreshJdWx(wxid) }
                .onSuccess { wxResultText?.apply { text = it.details?.joinToString("\n") ?: "刷新完成"; visibility = View.VISIBLE } }
                .onFailure { handlePortalError(it) }
            hideBtnLoading(btn)
        }
    }

    // ==================== 京东任务Tab ====================

    private fun renderTaskTab() {
        val ctx = requireContext()
        contentHost.addView(ctx.bodyText("选择任务和账号，点击执行开始").apply {
            setTextColor(Color.parseColor("#64748B"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            setPadding(0, 0, 0, ctx.dp(8))
        })
        val row = LinearLayout(ctx).apply { orientation = LinearLayout.HORIZONTAL }
        val left = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f); setPadding(0, 0, ctx.dp(4), 0) }
        val right = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL; layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f); setPadding(ctx.dp(4), 0, 0, 0) }
        row.addView(left); row.addView(right)
        contentHost.addView(row)

        // 日志
        contentHost.addView(ctx.cardView().apply {
            setPadding(ctx.dp(10), ctx.dp(8), ctx.dp(10), ctx.dp(8))
            val hdr = LinearLayout(ctx).apply { orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL }
            hdr.addView(TextView(ctx).apply { text = "📋 执行日志"; setTextColor(Color.parseColor("#475569")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f); setTypeface(typeface, Typeface.BOLD); layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f) })
            hdr.addView(makeSmallBtn("清空") { logLines.clear(); refreshLogView() })
            addView(hdr)
            logText = ctx.bodyText("暂无日志").apply { setPadding(0, ctx.dp(4), 0, 0); setBackgroundColor(Color.parseColor("#F8FAFC")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f) }.also { addView(it) }
        })

        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdAccounts() }.onSuccess { accounts = it }.onFailure { accounts = emptyList() }
            taskDefs.forEachIndexed { i, task ->
                val card = buildTaskCard(task)
                if (i % 2 == 0) left.addView(card) else right.addView(card)
            }
        }
    }

    private fun buildTaskCard(task: JdTaskDef): View {
        val ctx = requireContext()
        val selection = taskSelections.getOrPut(task.id) { mutableSetOf() }
        val isRunning = runningTasks.containsKey(task.id)
        return ctx.cardView().apply {
            setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
            // 任务名
            addView(TextView(ctx).apply { text = "${task.icon} ${task.name}"; setTextColor(Color.parseColor("#0F172A")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f); setTypeface(typeface, Typeface.BOLD) })
            addView(ctx.captionText(task.desc).apply { setPadding(0, ctx.dp(2), 0, ctx.dp(4)); setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f) })
            // 已选账号（动态更新）
            val label = TextView(ctx).apply { text = "已选：未选择账号"; setTextColor(Color.parseColor("#64748B")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f); setPadding(0, 0, 0, ctx.dp(6)) }
            selectedLabels[task.id] = label
            addView(label)
            // 选账号按钮（紧凑）
            addView(TextView(ctx).apply {
                text = "选账号"; setTextColor(Color.parseColor("#374151")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                background = GradientDrawable().apply { setColor(Color.parseColor("#F3F4F6")); cornerRadius = ctx.dp(6).toFloat(); setStroke(ctx.dp(1), Color.parseColor("#D1D5DB")) }
                gravity = Gravity.CENTER; setPadding(0, ctx.dp(4), 0, ctx.dp(4))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, ctx.dp(28)).apply { bottomMargin = ctx.dp(6) }
                setOnClickListener { showAccountPicker(task, selection, this) }
            })
            // 执行按钮
            val btnColor = if (isRunning) Color.parseColor("#EF4444") else Color.parseColor("#3B82F6")
            val execBtn = TextView(ctx).apply {
                text = if (isRunning) "停止" else "执行"; setTextColor(Color.WHITE); setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply { setColor(btnColor); cornerRadius = ctx.dp(8).toFloat() }
                gravity = Gravity.CENTER; setPadding(0, ctx.dp(6), 0, ctx.dp(6))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, ctx.dp(32))
                setOnClickListener { if (isRunning) stopTask(task) else executeTask(task, selection) }
            }
            executeBtns[task.id] = execBtn
            addView(execBtn)
        }
    }

    private fun showAccountPicker(task: JdTaskDef, selection: MutableSet<Int>, btn: View) {
        val validAccounts = accounts.filter { it.valid }
        if (validAccounts.isEmpty()) return alert("暂无有效账号")
        val names = mutableListOf("所有账号")
        validAccounts.forEach { names.add(it.nickname ?: it.pin ?: "账号") }
        // selection 存储的是1-based位置（和portal.html一致），-1表示所有
        val checked = BooleanArray(names.size) { i ->
            if (i == 0) selection.contains(-1)
            else selection.contains(i)
        }
        android.app.AlertDialog.Builder(requireContext())
            .setTitle("选择账号")
            .setMultiChoiceItems(names.toTypedArray(), checked) { _, which, isChecked ->
                if (which == 0) {
                    if (isChecked) { selection.clear(); selection.add(-1) }
                    else selection.remove(-1)
                } else {
                    // which 就是1-based位置
                    if (isChecked) { selection.remove(-1); selection.add(which) }
                    else selection.remove(which)
                }
            }
            .setPositiveButton("确定") { _, _ ->
                if (selection.isEmpty() || selection.contains(-1)) {
                    selection.clear(); selection.add(-1)
                }
                updateSelectedLabel(task)
            }
            .show()
    }

    private fun updateSelectedLabel(task: JdTaskDef) {
        val label = selectedLabels[task.id] ?: return
        val sel = taskSelections[task.id]
        val validAccounts = accounts.filter { it.valid }
        label.text = when {
            sel == null || sel.isEmpty() || sel.contains(-1) -> "已选：所有有效账号"
            else -> {
                val names = sel.mapNotNull { pos -> validAccounts.getOrNull(pos - 1)?.nickname ?: validAccounts.getOrNull(pos - 1)?.pin }
                "已选：${names.joinToString(", ")}"
            }
        }
    }

    // ==================== 任务执行 ====================

    private fun executeTask(task: JdTaskDef, selection: Set<Int>) {
        if (selection.isEmpty()) return alert("请至少选择一个账号")
        // selection 存储1-based位置，-1表示所有账号（和portal.html一致）
        val indices = if (selection.contains(-1)) {
            listOf(0)  // 后端0=所有账号
        } else {
            selection.sorted().toList()  // 直接传1-based位置
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
                    // 启动成功，不显示loading了（任务正在运行中，按钮应显示"停止"）
                    btn?.let { hideBtnLoading(it) }
                    logJob?.cancel()
                    logJob = lifecycleScope.launch {
                        AppServices.portalRepository.streamJdTaskLogs(
                            taskId = logId,
                            onLine = { line -> appendLog("[${task.name}] $line") },
                            onDone = { appendLog("[${task.name}] ✅ 完成"); runningTasks.remove(task.id) },
                            onError = { err -> appendLog("[${task.name}] 错误: ${err.message}", true); runningTasks.remove(task.id) },
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
            text = label; setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            setTypeface(typeface, if (active) Typeface.BOLD else Typeface.NORMAL)
            setTextColor(if (active) ContextCompat.getColor(context, R.color.brand_primary) else Color.parseColor("#64748B"))
            setPadding(context.dp(12), context.dp(6), context.dp(12), context.dp(6))
            background = GradientDrawable().apply { setColor(if (active) Color.parseColor("#EFF6FF") else Color.TRANSPARENT); cornerRadius = context.dp(8).toFloat() }
            setOnClickListener { onClick() }
        })
    }

    private fun makeSmallBtn(label: String, onClick: () -> Unit): TextView {
        return TextView(requireContext()).apply {
            text = label; setTextColor(Color.parseColor("#64748B")); setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            background = GradientDrawable().apply { setColor(Color.parseColor("#F1F5F9")); cornerRadius = requireContext().dp(6).toFloat() }
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, requireContext().dp(26)).apply { marginStart = requireContext().dp(6) }
            gravity = Gravity.CENTER
            setPadding(requireContext().dp(10), 0, requireContext().dp(10), 0)
            setOnClickListener { onClick() }
        }
    }

    private fun makeEmptyCard(message: String): View {
        val ctx = requireContext()
        return ctx.cardView().apply { setPadding(0, ctx.dp(20), 0, ctx.dp(20)); gravity = Gravity.CENTER
            addView(ctx.bodyText(message).apply { textSize = 13f; gravity = Gravity.CENTER; setTextColor(Color.parseColor("#64748B")) })
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
}
