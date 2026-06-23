package com.goudong.jd.ui.jd

import android.graphics.Color
import android.graphics.Typeface
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import androidx.appcompat.app.AlertDialog
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
import com.goudong.jd.ui.common.heroCard
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.openExternalUrl
import com.goudong.jd.ui.common.primaryButton
import com.goudong.jd.ui.common.secondaryButton
import com.goudong.jd.ui.common.sectionTitle
import com.goudong.jd.ui.common.softCard
import com.goudong.jd.ui.common.toast
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

private data class JdTaskDef(val id: String, val name: String, val icon: String, val desc: String)

class JdPortalFragment : Fragment() {
    private enum class MainTab { LOGIN, TASK }
    private enum class LoginTab { QUERY, SMS, WX }

    private var mainTab = MainTab.LOGIN
    private var loginTab = LoginTab.QUERY

    private lateinit var mainTabRow: LinearLayout
    private lateinit var loginTabRow: LinearLayout
    private lateinit var toolbarRow: LinearLayout
    private lateinit var contentHost: LinearLayout

    private var accounts: List<PortalJdAccount> = emptyList()
    private val taskSelections = mutableMapOf<String, MutableSet<Int>>()
    private val runningTasks = mutableMapOf<String, String>()
    private val logLines = mutableListOf<String>()
    private var logJob: Job? = null

    private var smsPhoneInput: EditText? = null
    private var smsCodeInput: EditText? = null
    private var smsIdCardInput: EditText? = null
    private var smsIdCardBox: LinearLayout? = null
    private var smsResultText: TextView? = null
    private var wxRiskBox: LinearLayout? = null
    private var wxRiskMsg: TextView? = null
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
        val (scroll, root) = requireContext().makeScrollContainer()
        root.addView(requireContext().heroCard("京东工作台", "查询资产、登录与京东任务", ContextCompat.getColor(requireContext(), R.color.brand_primary)))
        mainTabRow = LinearLayout(requireContext()).apply { orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER }
        loginTabRow = LinearLayout(requireContext()).apply { orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER }
        toolbarRow = LinearLayout(requireContext()).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.END
            setPadding(0, requireContext().dp(6), 0, requireContext().dp(6))
        }
        contentHost = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        root.addView(mainTabRow)
        root.addView(View(requireContext()).apply {
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, requireContext().dp(1)).apply {
                topMargin = requireContext().dp(8); bottomMargin = requireContext().dp(8)
            }
            setBackgroundColor(Color.parseColor("#E2E8F0"))
        })
        root.addView(loginTabRow)
        root.addView(toolbarRow)
        root.addView(contentHost)
        renderMainTabs()
        renderContent()
        return scroll
    }

    override fun onResume() {
        super.onResume()
        if (mainTab == MainTab.LOGIN && loginTab == LoginTab.QUERY) loadAccounts()
    }

    private fun renderMainTabs() {
        mainTabRow.removeAllViews()
        addTabButton(mainTabRow, "🔍 查询与登录", mainTab == MainTab.LOGIN) {
            mainTab = MainTab.LOGIN; renderMainTabs(); renderContent()
        }
        addTabButton(mainTabRow, "⚡ 京东任务", mainTab == MainTab.TASK) {
            mainTab = MainTab.TASK; renderMainTabs(); renderContent()
        }
    }

    private fun addTabButton(parent: LinearLayout, label: String, active: Boolean, onClick: () -> Unit) {
        parent.addView(TextView(requireContext()).apply {
            text = label
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTypeface(typeface, if (active) Typeface.BOLD else Typeface.NORMAL)
            setTextColor(if (active) ContextCompat.getColor(context, R.color.brand_primary) else Color.parseColor("#64748B"))
            setPadding(context.dp(12), context.dp(8), context.dp(12), context.dp(8))
            background = android.graphics.drawable.GradientDrawable().apply {
                setColor(if (active) Color.parseColor("#EFF6FF") else Color.TRANSPARENT)
                cornerRadius = context.dp(10).toFloat()
            }
            setOnClickListener { onClick() }
        })
    }

    private fun renderContent() {
        loginTabRow.removeAllViews()
        toolbarRow.removeAllViews()
        contentHost.removeAllViews()
        when (mainTab) {
            MainTab.LOGIN -> renderLoginPanel()
            MainTab.TASK -> renderTaskPanel()
        }
    }

    private fun renderLoginPanel() {
        loginTabRow.visibility = View.VISIBLE
        addTabButton(loginTabRow, "查询", loginTab == LoginTab.QUERY) {
            loginTab = LoginTab.QUERY; renderContent(); loadAccounts()
        }
        addTabButton(loginTabRow, "短信登录", loginTab == LoginTab.SMS) {
            loginTab = LoginTab.SMS; renderContent()
        }
        addTabButton(loginTabRow, "微信协议", loginTab == LoginTab.WX) {
            loginTab = LoginTab.WX; renderContent(); loadWxDevices()
        }
        when (loginTab) {
            LoginTab.QUERY -> {
                toolbarRow.addView(requireContext().primaryButton("全部查询").apply {
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        marginEnd = requireContext().dp(8)
                    }
                    setOnClickListener { queryAccount(0) }
                })
                toolbarRow.addView(requireContext().secondaryButton("刷新").apply { setOnClickListener { loadAccounts() } })
                contentHost.addView(LinearLayout(requireContext()).apply {
                    orientation = LinearLayout.VERTICAL
                    tag = "account_list"
                    addView(requireContext().bodyText("加载中..."))
                })
                loadAccounts()
            }
            LoginTab.SMS -> renderSmsPanel()
            LoginTab.WX -> renderWxPanel()
        }
    }

    private fun renderSmsPanel() {
        contentHost.addView(requireContext().cardView().apply {
            addView(requireContext().sectionTitle("短信验证码登录"))
            addView(requireContext().captionText("输入京东绑定的手机号，验证码登录后自动绑定").apply { setPadding(0, 0, 0, requireContext().dp(10)) })
            smsPhoneInput = requireContext().inputField("请输入11位手机号", number = true).also { addView(it) }
            addView(requireContext().primaryButton("发送验证码").apply { setOnClickListener { sendSms(this) } })
            smsCodeInput = requireContext().inputField("请输入短信验证码", number = true).also { addView(it) }
            smsIdCardBox = LinearLayout(requireContext()).apply {
                orientation = LinearLayout.VERTICAL
                visibility = View.GONE
                addView(requireContext().captionText("身份证验证"))
                smsIdCardInput = requireContext().inputField("身份证前两位+后四位").also { addView(it) }
            }.also { addView(it) }
            addView(requireContext().primaryButton("提交登录", ContextCompat.getColor(requireContext(), R.color.brand_secondary)).apply {
                setOnClickListener { verifySms() }
            })
            smsResultText = requireContext().bodyText("").apply { visibility = View.GONE }.also { addView(it) }
        })
    }

    private fun renderWxPanel() {
        contentHost.addView(requireContext().softCard(ContextCompat.getColor(requireContext(), R.color.brand_secondary)).apply {
            addView(requireContext().bodyText("💡 需先在「更多-微信协议」扫码绑定在线设备，再选择设备刷新京东 CK。"))
        })
        toolbarRow.addView(requireContext().secondaryButton("刷新设备").apply { setOnClickListener { loadWxDevices() } })
        contentHost.addView(LinearLayout(requireContext()).apply {
            orientation = LinearLayout.VERTICAL
            tag = "wx_device_list"
            addView(requireContext().bodyText("加载中..."))
        })
        wxRiskBox = requireContext().softCard(Color.parseColor("#F59E0B")).apply {
            visibility = View.GONE
            wxRiskMsg = requireContext().bodyText("").also { addView(it) }
            addView(requireContext().primaryButton("验证完成，继续刷新") { continueWxRisk() })
        }.also { contentHost.addView(it) }
        wxResultText = requireContext().bodyText("").apply { visibility = View.GONE }.also { contentHost.addView(it) }
        loadWxDevices()
    }

    private fun renderTaskPanel() {
        loginTabRow.visibility = View.GONE
        contentHost.addView(requireContext().captionText("选择任务和账号，点击执行按钮开始任务").apply { setPadding(0, 0, 0, requireContext().dp(8)) })
        val taskHost = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL; tag = "task_cards" }
        contentHost.addView(taskHost)
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdAccounts() }
                .onSuccess { accounts = it }
                .onFailure { accounts = emptyList() }
            taskHost.removeAllViews()
            taskDefs.forEach { task -> taskHost.addView(buildTaskCard(task)) }
        }
        contentHost.addView(requireContext().cardView().apply {
            val header = LinearLayout(requireContext()).apply { orientation = LinearLayout.HORIZONTAL; gravity = Gravity.CENTER_VERTICAL }
            header.addView(requireContext().sectionTitle("执行日志").apply { layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f) })
            header.addView(requireContext().secondaryButton("清空").apply { setOnClickListener { logLines.clear(); refreshLogView() } })
            addView(header)
            logText = requireContext().bodyText("暂无日志，请执行任务").apply {
                setPadding(requireContext().dp(8), requireContext().dp(8), requireContext().dp(8), requireContext().dp(8))
                setBackgroundColor(Color.parseColor("#F8FAFC"))
            }.also { addView(ScrollView(requireContext()).apply {
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, requireContext().dp(220))
                addView(it)
            }) }
        })
    }

    private fun buildTaskCard(task: JdTaskDef): LinearLayout {
        val selection = taskSelections.getOrPut(task.id) { mutableSetOf(0) }
        val accountLabel = TextView(requireContext()).apply {
            text = accountSelectionLabel(selection)
            setTextColor(Color.parseColor("#475569"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
        }
        val executeBtn = requireContext().primaryButton(if (runningTasks.containsKey(task.id)) "停止任务" else "执行任务")
        if (runningTasks.containsKey(task.id)) executeBtn.setBackgroundColor(Color.parseColor("#EF4444"))
        executeBtn.setOnClickListener {
            if (runningTasks.containsKey(task.id)) stopTask(task, executeBtn) else executeTask(task, selection, executeBtn)
        }
        return requireContext().cardView().apply {
            addView(TextView(requireContext()).apply {
                text = "${task.icon} ${task.name}"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
            })
            addView(requireContext().captionText(task.desc).apply { setPadding(0, 0, 0, requireContext().dp(8)) })
            addView(requireContext().secondaryButton("选择账号").apply { setOnClickListener { showAccountPicker(selection, accountLabel) } })
            addView(accountLabel)
            addView(executeBtn)
        }
    }

    private fun accountSelectionLabel(selection: Set<Int>): String {
        if (selection.isEmpty()) return "未选择账号"
        if (selection.contains(0)) return "已选：所有有效账号"
        return "已选：${buildAccountOptions().filter { selection.contains(it.first) }.map { it.second }.joinToString("、")}"
    }

    private fun buildAccountOptions(): List<Pair<Int, String>> {
        val options = mutableListOf(0 to "所有有效账号")
        var validIdx = 1
        accounts.filter { it.valid }.forEach { acc ->
            options.add(validIdx to (acc.nickname ?: acc.pin ?: "账号$validIdx"))
            validIdx++
        }
        return options
    }

    private fun showAccountPicker(selection: MutableSet<Int>, labelView: TextView) {
        val options = buildAccountOptions()
        val checked = BooleanArray(options.size) { selection.contains(options[it].first) }
        AlertDialog.Builder(requireContext())
            .setTitle("选择账号")
            .setMultiChoiceItems(options.map { it.second }.toTypedArray(), checked) { _, which, isChecked ->
                val idx = options[which].first
                if (idx == 0 && isChecked) { selection.clear(); selection.add(0) }
                else if (isChecked) { selection.remove(0); selection.add(idx) }
                else selection.remove(idx)
            }
            .setPositiveButton("确定") { _, _ ->
                if (selection.isEmpty()) selection.add(0)
                labelView.text = accountSelectionLabel(selection)
            }
            .show()
    }

    private fun loadAccounts() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdAccounts() }
                .onSuccess { renderAccountList(it) }
                .onFailure { renderAccountList(emptyList(), it.message); handlePortalError(it) }
        }
    }

    private fun renderAccountList(list: List<PortalJdAccount>, error: String? = null) {
        accounts = list
        val host = contentHost.findViewWithTag<LinearLayout>("account_list") ?: return
        host.removeAllViews()
        when {
            error != null -> host.addView(requireContext().bodyText(error))
            list.isEmpty() -> host.addView(requireContext().bodyText("暂无绑定的京东账号，请先使用短信或微信协议登录"))
            else -> {
                host.addView(requireContext().captionText("有效账号 ${list.count { it.valid }} / ${list.size}").apply { setPadding(0, 0, 0, requireContext().dp(8)) })
                list.sortedByDescending { it.valid }.forEach { acc ->
                    host.addView(requireContext().cardView().apply {
                        addView(TextView(requireContext()).apply {
                            text = acc.nickname ?: acc.pin ?: "未知账号"
                            setTextColor(Color.parseColor("#0F172A"))
                            setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                            setTypeface(typeface, Typeface.BOLD)
                        })
                        addView(requireContext().captionText(acc.pin ?: ""))
                        addView(requireContext().captionText(if (acc.valid) "✅ 有效" else "❌ 失效"))
                        addView(requireContext().primaryButton("查询") { queryAccount(acc.index) })
                    })
                }
            }
        }
    }

    private fun queryAccount(index: Int) {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.queryJdAccount(index) }
                .onSuccess { startActivity(ResultTextActivity.intent(requireContext(), "查询结果", it)) }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun sendSms(btn: Button) {
        val phone = smsPhoneInput?.text?.toString().orEmpty().trim()
        if (phone.length != 11) return alert("请输入11位手机号")
        btn.isEnabled = false
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.sendJdSms(phone) }
                .onSuccess { toast(it); smsIdCardBox?.visibility = View.GONE }
                .onFailure { handlePortalError(it) }
            btn.isEnabled = true
        }
    }

    private fun verifySms() {
        val phone = smsPhoneInput?.text?.toString().orEmpty().trim()
        val code = smsCodeInput?.text?.toString().orEmpty().trim()
        val idCard = smsIdCardInput?.text?.toString().orEmpty().trim()
        if (phone.isBlank()) return alert("请输入手机号")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.verifyJdSms(phone, code, idCard) }
                .onSuccess { result ->
                    if (result.needIdVerify) {
                        smsIdCardBox?.visibility = View.VISIBLE
                        toast(result.message ?: "需要身份证验证")
                        return@onSuccess
                    }
                    smsIdCardBox?.visibility = View.GONE
                    smsResultText?.apply { text = result.queryResult ?: result.message ?: "登录成功"; visibility = View.VISIBLE }
                    toast(result.message ?: "登录成功")
                    loadAccounts()
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun loadWxDevices() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdWxDevices() }
                .onSuccess { renderWxDevices(it) }
                .onFailure { renderWxDevices(emptyList(), it.message); handlePortalError(it) }
        }
    }

    private fun renderWxDevices(list: List<PortalJdWxDevice>, error: String? = null) {
        val host = contentHost.findViewWithTag<LinearLayout>("wx_device_list") ?: return
        host.removeAllViews()
        when {
            error != null -> host.addView(requireContext().bodyText(error))
            list.isEmpty() -> host.addView(requireContext().bodyText("暂无在线微信协议设备，请先到「更多-微信协议」扫码登录"))
            else -> list.forEach { device ->
                host.addView(requireContext().cardView().apply {
                    addView(TextView(requireContext()).apply {
                        text = device.nickname ?: device.wxid ?: "未知设备"
                        setTextColor(Color.parseColor("#0F172A"))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                        setTypeface(typeface, Typeface.BOLD)
                    })
                    addView(requireContext().captionText(device.wxid ?: ""))
                    addView(requireContext().primaryButton("刷新 CK") { refreshWx(device.wxid.orEmpty()) })
                })
            }
        }
    }

    private fun refreshWx(wxid: String) {
        if (wxid.isBlank()) return
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.refreshJdWx(wxid) }
                .onSuccess { result ->
                    wxRiskBox?.visibility = View.GONE
                    if (result.needRiskVerify) {
                        wxRiskBox?.visibility = View.VISIBLE
                        wxRiskMsg?.text = result.riskMsg ?: "账号需要短信验证"
                        result.riskUrl?.let { url ->
                            wxRiskBox?.addView(requireContext().primaryButton("打开验证链接") { openExternalUrl(url) })
                        }
                    }
                    wxResultText?.apply {
                        text = "成功 ${result.success} / 失败 ${result.fail}\n${result.details?.joinToString("\n").orEmpty()}"
                        visibility = View.VISIBLE
                    }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun continueWxRisk() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.continueJdWxRisk() }
                .onSuccess { result ->
                    wxRiskBox?.visibility = View.GONE
                    wxResultText?.apply { text = result.details?.joinToString("\n") ?: "刷新完成"; visibility = View.VISIBLE }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun executeTask(task: JdTaskDef, selection: Set<Int>, btn: Button) {
        if (selection.isEmpty()) return alert("请至少选择一个账号")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.executeJdTask(task.id, task.name, selection.toList()) }
                .onSuccess { result ->
                    val logId = result.taskId ?: return@onSuccess alert("任务启动失败")
                    runningTasks[task.id] = logId
                    btn.text = "停止任务"
                    btn.setBackgroundColor(Color.parseColor("#EF4444"))
                    appendLog("[${task.name}] 任务已启动...")
                    logJob?.cancel()
                    logJob = lifecycleScope.launch {
                        AppServices.portalRepository.streamJdTaskLogs(
                            taskId = logId,
                            onLine = { line -> appendLog("[${task.name}] $line") },
                            onDone = {
                                appendLog("[${task.name}] ✅ 任务执行完成")
                                runningTasks.remove(task.id)
                                btn.text = "执行任务"
                                btn.setBackgroundColor(ContextCompat.getColor(requireContext(), R.color.brand_primary))
                            },
                            onError = { err ->
                                appendLog("[${task.name}] 错误: ${err.message}", true)
                                runningTasks.remove(task.id)
                                btn.text = "执行任务"
                            },
                        )
                    }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun stopTask(task: JdTaskDef, btn: Button) {
        val logId = runningTasks[task.id] ?: return
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.stopJdTask(logId) }
            logJob?.cancel()
            runningTasks.remove(task.id)
            appendLog("[${task.name}] ⚠️ 任务已手动停止")
            btn.text = "执行任务"
            btn.setBackgroundColor(ContextCompat.getColor(requireContext(), R.color.brand_primary))
        }
    }

    private fun appendLog(line: String, @Suppress("UNUSED_PARAMETER") isError: Boolean = false) {
        logLines.add(line)
        if (logLines.size > 200) logLines.removeAt(0)
        refreshLogView()
    }

    private fun refreshLogView() {
        logText?.text = if (logLines.isEmpty()) "暂无日志，请执行任务" else logLines.joinToString("\n")
    }
}
