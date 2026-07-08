package com.goudong.jd.ui.projects

import android.content.Context
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.util.TypedValue
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.AdapterView
import android.widget.ArrayAdapter
import android.widget.Spinner
import android.widget.RadioButton
import android.widget.RadioGroup
import android.widget.TextView
import androidx.activity.addCallback
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.KuwoTaskLog
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.toast
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import org.json.JSONObject
import com.goudong.jd.ui.common.themeColor

class KuwoRushFragment : Fragment() {
    private lateinit var contentRoot: LinearLayout
    private lateinit var kuwoPanel: LinearLayout

    private var authHint: TextView? = null
    private var accountSelectWrap: LinearLayout? = null
    private var accountSpinner: Spinner? = null
    private var phoneInput: EditText? = null
    private var passwordInput: EditText? = null
    private var smsInput: EditText? = null
    private var smsStatus: TextView? = null
    private var nowTimeText: TextView? = null
    private var nextTimeText: TextView? = null
    private var timeHintText: TextView? = null
    private var withdrawBtn: TextView? = null
    private var updateSmsBtn: TextView? = null
    private var smsEditHint: TextView? = null
    private var countdownText: TextView? = null
    private var logText: TextView? = null
    private var quotaGroup: RadioGroup? = null

    private var kuwoAuthorized = false
    private var kuwoAccounts: List<Pair<String, String>> = emptyList()
    private var selectedAccountIndex = 0
    private var suppressAccountSwitch = false
    private var selectedQuotaId = "30002"
    private var activeTaskId: String? = null
    private var withdrawSubmitting = false
    private var taskLogIndex = 0
    private var monitorJob: Job? = null
    private var countdownJob: Job? = null
    private val clockHandler = Handler(Looper.getMainLooper())
    private val clockRunnable = object : Runnable {
        override fun run() {
            updateTimeDisplay()
            clockHandler.postDelayed(this, 1000)
        }
    }

    private val quotaOptions = listOf(
        "30002" to "2元",
        "60004" to "1元",
        "60001" to "10元",
    )

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val wrapper = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        val (scroll, root) = requireContext().makeScrollContainer()
        contentRoot = root
        wrapper.addView(scroll)
        buildPage()
        return wrapper
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        requireActivity().onBackPressedDispatcher.addCallback(viewLifecycleOwner) {
            parentFragmentManager.popBackStack()
        }
    }

    override fun onResume() {
        super.onResume()
        clockHandler.post(clockRunnable)
        loadKuwoPage()
    }

    override fun onPause() {
        super.onPause()
        clockHandler.removeCallbacks(clockRunnable)
        monitorJob?.cancel()
        countdownJob?.cancel()
    }

    private fun buildPage() {
        val ctx = requireContext()
        contentRoot.removeAllViews()

        contentRoot.addView(TextView(ctx).apply {
            text = "酷我提现"
            setTextColor(requireContext().themeColor(R.color.text_primary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 17f)
            setTypeface(typeface, Typeface.BOLD)
            setPadding(ctx.dp(14), ctx.dp(4), ctx.dp(14), ctx.dp(8))
        })

        contentRoot.addView(ctx.captionText("定时抢兑，与网页端功能一致").apply {
            setPadding(ctx.dp(14), 0, ctx.dp(14), ctx.dp(10))
        })

        kuwoPanel = LinearLayout(ctx).apply { orientation = LinearLayout.VERTICAL }
        buildKuwoPanel(kuwoPanel)
        contentRoot.addView(kuwoPanel)
    }

    private fun buildKuwoPanel(host: LinearLayout) {
        val ctx = requireContext()
        val pad = ctx.dp(14)

        authHint = TextView(ctx).apply {
            visibility = View.GONE
            setTextColor(requireContext().themeColor(R.color.negative))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
            setTypeface(typeface, Typeface.BOLD)
            background = GradientDrawable().apply {
                setColor(Color.parseColor("#FEF2F2"))
                cornerRadius = ctx.dp(10).toFloat()
                setStroke(ctx.dp(1), Color.parseColor("#FECACA"))
            }
            setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginStart = pad; marginEnd = pad; bottomMargin = ctx.dp(10)
            }
        }
        host.addView(authHint)

        host.addView(ctx.cardView().apply {
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginStart = pad; marginEnd = pad; bottomMargin = ctx.dp(12)
            }
            addView(TextView(ctx).apply {
                text = "账号配置"
                setTextColor(requireContext().themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
            })
            addView(ctx.captionText("账号密码自动读取自「酷我音乐」上车信息，修改请前往「我的项目」。").apply {
                setPadding(0, ctx.dp(6), 0, ctx.dp(10))
                setLineSpacing(0f, 1.35f)
            })
            accountSelectWrap = LinearLayout(ctx).apply {
                orientation = LinearLayout.VERTICAL
                visibility = View.GONE
                setPadding(0, 0, 0, ctx.dp(10))
            }
            accountSelectWrap?.addView(fieldLabel("选择抢兑账号"))
            accountSpinner = Spinner(ctx).apply {
                background = GradientDrawable().apply {
                    setColor(requireContext().themeColor(R.color.surface_card))
                    cornerRadius = ctx.dp(10).toFloat()
                    setStroke(ctx.dp(1), requireContext().themeColor(R.color.border_light))
                }
                setPadding(ctx.dp(8), ctx.dp(4), ctx.dp(8), ctx.dp(4))
                onItemSelectedListener = object : AdapterView.OnItemSelectedListener {
                    override fun onItemSelected(parent: AdapterView<*>?, view: View?, position: Int, id: Long) {
                        if (suppressAccountSwitch) return
                        if (position != selectedAccountIndex) {
                            switchAccount(position)
                        }
                    }
                    override fun onNothingSelected(parent: AdapterView<*>?) {}
                }
            }
            accountSelectWrap?.addView(accountSpinner)
            addView(accountSelectWrap)
            addView(fieldLabel("手机号"))
            phoneInput = readonlyField("自动读取中...")
            addView(phoneInput)
            addView(fieldLabel("密码").apply { setPadding(0, ctx.dp(10), 0, ctx.dp(4)) })
            passwordInput = readonlyField("自动读取中...")
            addView(passwordInput)
            addView(fieldLabel("短信验证码").apply { setPadding(0, ctx.dp(10), 0, ctx.dp(4)) })
            smsInput = editableField("输入短信验证码")
            addView(smsInput)
            val smsRow = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(0, ctx.dp(8), 0, 0)
            }
            smsRow.addView(TextView(ctx).apply {
                text = "发送验证码"
                setTextColor(requireContext().themeColor(R.color.chip_active_text))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#3B82F6"))
                    cornerRadius = ctx.dp(8).toFloat()
                }
                setPadding(ctx.dp(14), ctx.dp(8), ctx.dp(14), ctx.dp(8))
                setOnClickListener { sendSms() }
            })
            smsStatus = TextView(ctx).apply {
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                    marginStart = ctx.dp(10)
                }
            }
            smsRow.addView(smsStatus)
            addView(smsRow)
        })

        host.addView(ctx.cardView().apply {
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginStart = pad; marginEnd = pad; bottomMargin = ctx.dp(12)
            }
            addView(TextView(ctx).apply {
                text = "提现设置"
                setTextColor(requireContext().themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
            })
            addView(fieldLabel("抢兑档位").apply { setPadding(0, ctx.dp(10), 0, ctx.dp(6)) })
            quotaGroup = RadioGroup(ctx).apply {
                orientation = RadioGroup.HORIZONTAL
                quotaOptions.forEachIndexed { i, (quotaId, label) ->
                    val rb = RadioButton(ctx).apply {
                        text = label
                        this.id = View.generateViewId()
                        tag = quotaId
                        isChecked = i == 0
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                        setPadding(ctx.dp(4), 0, ctx.dp(12), 0)
                    }
                    addView(rb)
                }
                setOnCheckedChangeListener { group, checkedId ->
                    val rb = group.findViewById<RadioButton>(checkedId)
                    selectedQuotaId = rb?.tag as? String ?: "30002"
                }
            }
            addView(quotaGroup)

            addView(LinearLayout(ctx).apply {
                orientation = LinearLayout.VERTICAL
                setPadding(ctx.dp(12), ctx.dp(12), ctx.dp(12), ctx.dp(12))
                background = GradientDrawable().apply {
                    setColor(requireContext().themeColor(R.color.input_bg))
                    cornerRadius = ctx.dp(10).toFloat()
                    setStroke(ctx.dp(1), requireContext().themeColor(R.color.border_light))
                }
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    topMargin = ctx.dp(12)
                }
                nowTimeText = TextView(ctx).apply { setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f); setTextColor(requireContext().themeColor(R.color.text_primary)) }
                nextTimeText = TextView(ctx).apply { setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f); setTextColor(requireContext().themeColor(R.color.text_primary)) }
                timeHintText = TextView(ctx).apply {
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                    setTypeface(typeface, Typeface.BOLD)
                    setPadding(0, ctx.dp(4), 0, 0)
                }
                addView(nowTimeText)
                addView(nextTimeText)
                addView(timeHintText)
            })

            addView(TextView(ctx).apply {
                text = "💡 抢兑说明\n• 抢兑时段：00:00、09:00、13:00、17:00、20:00\n• 抢兑时段前4分钟内开始，将倒计时到点抢兑\n• 非抢兑时段点击开始，将立即提交抢兑"
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11.5f)
                setLineSpacing(0f, 1.4f)
                background = GradientDrawable().apply {
                    setColor(Color.parseColor("#EEF2FF"))
                    cornerRadius = ctx.dp(10).toFloat()
                    setStroke(ctx.dp(1), Color.parseColor("#C7D2FE"))
                }
                setPadding(ctx.dp(12), ctx.dp(10), ctx.dp(12), ctx.dp(10))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    topMargin = ctx.dp(12)
                }
            })

            val actionRow = LinearLayout(ctx).apply {
                orientation = LinearLayout.VERTICAL
                setPadding(0, ctx.dp(12), 0, 0)
            }
            withdrawBtn = TextView(ctx).apply {
                text = "开始抢兑"
                gravity = Gravity.CENTER
                setTextColor(requireContext().themeColor(R.color.chip_active_text))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
                background = GradientDrawable(
                    GradientDrawable.Orientation.TL_BR,
                    intArrayOf(Color.parseColor("#6366F1"), Color.parseColor("#8B5CF6")),
                ).apply { cornerRadius = ctx.dp(10).toFloat() }
                setPadding(0, ctx.dp(12), 0, ctx.dp(12))
                setOnClickListener { startWithdraw() }
            }
            updateSmsBtn = TextView(ctx).apply {
                visibility = View.GONE
                text = "更新验证码"
                gravity = Gravity.CENTER
                setTextColor(Color.parseColor("#6366F1"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTypeface(typeface, Typeface.BOLD)
                background = GradientDrawable().apply {
                    setColor(requireContext().themeColor(R.color.surface_card))
                    cornerRadius = ctx.dp(10).toFloat()
                    setStroke(ctx.dp(1), requireContext().themeColor(R.color.border_light))
                }
                setPadding(ctx.dp(16), ctx.dp(10), ctx.dp(16), ctx.dp(10))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    topMargin = ctx.dp(8)
                }
                setOnClickListener { updateSmsCode() }
            }
            smsEditHint = TextView(ctx).apply {
                visibility = View.GONE
                text = "倒计时中可修改上方验证码，改完后点击「更新验证码」同步到后端"
                setTextColor(Color.parseColor("#6366F1"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11.5f)
                setTypeface(typeface, Typeface.BOLD)
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    topMargin = ctx.dp(6)
                }
            }
            countdownText = TextView(ctx).apply {
                visibility = View.GONE
                gravity = Gravity.CENTER
                setTextColor(requireContext().themeColor(R.color.chip_active_text))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
                background = GradientDrawable(
                    GradientDrawable.Orientation.TL_BR,
                    intArrayOf(Color.parseColor("#F59E0B"), requireContext().themeColor(R.color.brand_red)),
                ).apply { cornerRadius = ctx.dp(10).toFloat() }
                setPadding(0, ctx.dp(12), 0, ctx.dp(12))
            }
            actionRow.addView(withdrawBtn)
            actionRow.addView(updateSmsBtn)
            actionRow.addView(smsEditHint)
            actionRow.addView(countdownText)
            addView(actionRow)
        })

        host.addView(ctx.cardView().apply {
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginStart = pad; marginEnd = pad; bottomMargin = ctx.dp(20)
            }
            val hdr = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
            }
            hdr.addView(TextView(ctx).apply {
                text = "执行日志"
                setTextColor(requireContext().themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            })
            hdr.addView(TextView(ctx).apply {
                text = "清空"
                setTextColor(requireContext().themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setPadding(ctx.dp(8), ctx.dp(4), ctx.dp(8), ctx.dp(4))
                setOnClickListener { clearLogs() }
            })
            addView(hdr)
            addView(ctx.captionText("倒计时抢兑会实时同步后端每一步").apply { setPadding(0, ctx.dp(2), 0, ctx.dp(8)) })
            logText = ctx.bodyText("").apply {
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                typeface = Typeface.MONOSPACE
                setTextColor(requireContext().themeColor(R.color.text_secondary))
                setLineSpacing(0f, 1.5f)
                setBackgroundColor(requireContext().themeColor(R.color.input_bg))
                setPadding(ctx.dp(10), ctx.dp(10), ctx.dp(10), ctx.dp(10))
            }
            addView(logText)
        })

        restoreLogs()
        updateTimeDisplay()
    }

    private fun fieldLabel(text: String) = TextView(requireContext()).apply {
        this.text = text
        setTextColor(requireContext().themeColor(R.color.text_muted))
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 11.5f)
    }

    private fun readonlyField(hint: String) = EditText(requireContext()).apply {
        isEnabled = false
        isFocusable = false
        setHint(hint)
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
        setTextColor(requireContext().themeColor(R.color.text_muted))
        setPadding(requireContext().dp(12), requireContext().dp(10), requireContext().dp(12), requireContext().dp(10))
        background = GradientDrawable().apply {
            setColor(requireContext().themeColor(R.color.chip_bg))
            cornerRadius = requireContext().dp(10).toFloat()
            setStroke(requireContext().dp(1), requireContext().themeColor(R.color.border_light))
        }
        layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT)
    }

    private fun editableField(hint: String) = EditText(requireContext()).apply {
        this.hint = hint
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
        setPadding(requireContext().dp(12), requireContext().dp(10), requireContext().dp(12), requireContext().dp(10))
        background = GradientDrawable().apply {
            setColor(requireContext().themeColor(R.color.surface_card))
            cornerRadius = requireContext().dp(10).toFloat()
            setStroke(requireContext().dp(1), requireContext().themeColor(R.color.border_light))
        }
        layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT)
    }

    private fun prefs() = requireContext().getSharedPreferences("kuwo_rush", Context.MODE_PRIVATE)

    private fun appendLog(msg: String) {
        val current = logText?.text?.toString().orEmpty()
        val next = if (current.isEmpty()) msg else "$current\n$msg"
        logText?.text = next
        prefs().edit().putString("log", next).apply()
    }

    private fun restoreLogs() {
        logText?.text = prefs().getString("log", "") ?: ""
    }

    private fun clearLogs() {
        logText?.text = ""
        taskLogIndex = 0
        prefs().edit().remove("log").apply()
    }

    private fun renderTaskLog(entry: KuwoTaskLog) {
        val icons = mapOf("info" to "ℹ️", "success" to "✅", "warn" to "⚠️", "error" to "❌")
        val icon = icons[entry.level] ?: "•"
        var line = "[${entry.time ?: "--"}] $icon ${entry.message ?: ""}"
        if (!entry.proxyHost.isNullOrBlank()) {
            line += " [代理:${entry.proxyHost}]"
        }
        appendLog(line)
    }

    private fun flushTaskLogs(logs: List<KuwoTaskLog>?) {
        if (logs.isNullOrEmpty() || logs.size <= taskLogIndex) return
        for (i in taskLogIndex until logs.size) renderTaskLog(logs[i])
        taskLogIndex = logs.size
    }

    private fun loadKuwoPage() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.checkKuwoAuth() }
                .onSuccess { (authorized, msg) ->
                    kuwoAuthorized = authorized
                    authHint?.apply {
                        visibility = if (authorized) View.GONE else View.VISIBLE
                        text = msg.ifBlank { "酷我活动授权已到期，请前往我的项目续费后再使用" }
                    }
                }
                .onFailure { handlePortalError(it) }

            runCatching { AppServices.portalRepository.fetchKuwoCredentials() }
                .onSuccess { applyKuwoCredentials(it) }
                .onFailure {
                    kuwoAccounts = emptyList()
                    accountSelectWrap?.visibility = View.GONE
                    phoneInput?.setText("")
                    passwordInput?.setText("")
                }
        }
    }

    private fun maskPhone(phone: String): String {
        return if (phone.length >= 11) phone.substring(0, 3) + "****" + phone.substring(7) else phone
    }

    private fun resolveKuwoAccounts(cred: com.goudong.jd.data.model.KuwoCredentials): List<Pair<String, String>> {
        val fromList = cred.accounts
            ?.mapNotNull { acc ->
                val phone = acc.phone?.trim().orEmpty()
                if (phone.isEmpty()) null else phone to acc.password.orEmpty()
            }
            .orEmpty()
        if (fromList.isNotEmpty()) return fromList
        val phone = cred.phone?.trim().orEmpty()
        return if (phone.isNotEmpty()) listOf(phone to cred.password.orEmpty()) else emptyList()
    }

    private fun applyKuwoCredentials(cred: com.goudong.jd.data.model.KuwoCredentials) {
        val previousPhone = phoneInput?.text?.toString()?.trim().orEmpty()
        kuwoAccounts = resolveKuwoAccounts(cred)
        if (previousPhone.isNotEmpty()) {
            val matched = kuwoAccounts.indexOfFirst { it.first == previousPhone }
            if (matched >= 0) selectedAccountIndex = matched
        }
        if (kuwoAccounts.size > 1) {
            accountSelectWrap?.visibility = View.VISIBLE
            val labels = kuwoAccounts.map { (phone, _) -> maskPhone(phone) }
            val adapter = ArrayAdapter(requireContext(), android.R.layout.simple_spinner_item, labels).apply {
                setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item)
            }
            suppressAccountSwitch = true
            accountSpinner?.adapter = adapter
            accountSpinner?.setSelection(selectedAccountIndex.coerceIn(0, kuwoAccounts.lastIndex))
            suppressAccountSwitch = false
        } else {
            accountSelectWrap?.visibility = View.GONE
            selectedAccountIndex = 0
        }
        val index = selectedAccountIndex.coerceIn(0, (kuwoAccounts.size - 1).coerceAtLeast(0))
        if (kuwoAccounts.isNotEmpty()) {
            val (phone, password) = kuwoAccounts[index]
            phoneInput?.setText(phone)
            passwordInput?.setText(password)
            restoreKuwoState(phone)
        } else {
            phoneInput?.setText(cred.phone ?: "")
            passwordInput?.setText(cred.password ?: "")
            restoreKuwoState(cred.phone ?: "")
        }
    }

    private fun switchAccount(index: Int) {
        if (index < 0 || index >= kuwoAccounts.size) return
        selectedAccountIndex = index
        clearActiveState()
        restoreWithdrawUi()
        val (phone, password) = kuwoAccounts[index]
        phoneInput?.setText(phone)
        passwordInput?.setText(password)
        smsInput?.setText("")
        smsStatus?.text = ""
        smsStatus?.setTextColor(requireContext().themeColor(R.color.text_muted))
    }

    private fun updateTimeDisplay() {
        val bj = KuwoTimeHelper.getBeijingTime()
        val info = KuwoTimeHelper.getNextWithdrawInfo()
        val nextLabel = KuwoTimeHelper.formatHour(info.hour)
        nowTimeText?.text = "🕐 当前北京时间：${KuwoTimeHelper.formatClock(bj.hour, bj.min, bj.sec)}"
        when {
            info.inWindow && info.diffMin > 0 -> {
                nextTimeText?.text = "⏰ 下次抢兑：$nextLabel（${info.diffMin}分钟后）"
                timeHintText?.text = "✅ 可点击【开始抢兑】自动倒计时到点提交"
                timeHintText?.setTextColor(requireContext().themeColor(R.color.positive))
            }
            info.inWindow && info.diffMin == 0 -> {
                nextTimeText?.text = "⏰ 下次抢兑：$nextLabel（当前时段）"
                timeHintText?.text = "🚀 当前为抢兑时段，点击可直接提交"
                timeHintText?.setTextColor(requireContext().themeColor(R.color.negative))
            }
            else -> {
                nextTimeText?.text = "⏰ 下次抢兑：$nextLabel（还有${info.diffMin}分钟）"
                timeHintText?.text = "💡 点击【开始抢兑】将立即提交（非倒计时时段）"
                timeHintText?.setTextColor(ContextCompat.getColor(requireContext(), R.color.brand_primary))
            }
        }
    }

    private fun clearActiveState() {
        monitorJob?.cancel()
        countdownJob?.cancel()
        activeTaskId = null
        prefs().edit().remove("state").apply()
    }

    private fun sendSms() {
        if (!kuwoAuthorized) {
            toast("酷我活动授权已到期，请前往我的项目续费")
            return
        }
        val phone = phoneInput?.text?.toString()?.trim().orEmpty()
        val password = passwordInput?.text?.toString()?.trim().orEmpty()
        if (phone.isEmpty() || password.isEmpty()) {
            toast("未读取到酷我账号，请先在项目中上车")
            return
        }
        clearActiveState()
        restoreWithdrawUi()
        smsInput?.setText("")
        smsStatus?.text = "正在登录并发送验证码..."
        appendLog("正在登录酷我账号...")
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.sendKuwoSms(phone, password) }
                .onSuccess {
                    val masked = if (phone.length >= 11) phone.substring(0, 3) + "****" + phone.substring(7) else phone
                    smsStatus?.text = "✅ 已发送至 $masked"
                    smsStatus?.setTextColor(requireContext().themeColor(R.color.positive))
                    appendLog("验证码已发送，请输入验证码")
                    toast("验证码已发送")
                }
                .onFailure {
                    smsStatus?.text = "❌ ${it.message}"
                    smsStatus?.setTextColor(requireContext().themeColor(R.color.negative))
                    appendLog("发送失败: ${it.message}")
                    handlePortalError(it)
                }
        }
    }

    private fun startWithdraw() {
        if (!kuwoAuthorized) {
            toast("酷我活动授权已到期，请前往我的项目续费")
            return
        }
        if (withdrawSubmitting) {
            toast("任务提交中，请稍候")
            return
        }
        val phone = phoneInput?.text?.toString()?.trim().orEmpty()
        val password = passwordInput?.text?.toString()?.trim().orEmpty()
        val smsCode = smsInput?.text?.toString()?.trim().orEmpty()
        if (phone.isEmpty() || password.isEmpty()) {
            toast("未读取到酷我账号")
            return
        }
        if (smsCode.isEmpty()) {
            toast("请输入验证码")
            return
        }
        val info = KuwoTimeHelper.getNextWithdrawInfo()
        if (info.inWindow && info.diffMin > 0 && info.diffMin <= 4) {
            appendLog("🎯 提交定时抢兑任务，后端将在 ${KuwoTimeHelper.formatHour(info.hour)} 自动执行（3轮错峰）")
            setWithdrawUiLocked(true, allowSmsEdit = true)
            saveKuwoState(phone, password, smsCode, selectedQuotaId, info.hour, null, false)
            scheduleOnBackend(phone, password, smsCode, selectedQuotaId, info.hour, info, immediate = false)
            return
        }
        appendLog("⚡ 立即提交抢兑...")
        setWithdrawUiLocked(true)
        scheduleOnBackend(phone, password, smsCode, selectedQuotaId, null, null, immediate = true)
    }

    private fun scheduleOnBackend(
        phone: String,
        password: String,
        smsCode: String,
        quotaId: String,
        targetHour: Int?,
        info: KuwoTimeHelper.NextWithdrawInfo?,
        immediate: Boolean,
    ) {
        if (withdrawSubmitting) return
        withdrawSubmitting = true
        lifecycleScope.launch {
            runCatching {
                AppServices.portalRepository.scheduleKuwoWithdraw(phone, password, quotaId, smsCode, targetHour, immediate)
            }.onSuccess { result ->
                val taskId = result.taskId
                if (taskId.isNullOrBlank()) {
                    appendLog("❌ 提交失败：未返回任务ID")
                    restoreWithdrawUi()
                    return@onSuccess
                }
                activeTaskId = taskId
                appendLog((if (result.reused) "♻️ 复用已有任务" else "✅ 任务已提交") + "，ID: $taskId")
                taskLogIndex = 0
                updateSavedTaskId(taskId, immediate, targetHour)
                if (!immediate) setWithdrawUiLocked(true, allowSmsEdit = true)
                monitorTask(taskId, immediate)
                if (!immediate && info != null) {
                    runCountdown(info, taskId)
                }
            }.onFailure {
                appendLog("❌ 提交失败: ${it.message}")
                handlePortalError(it)
                restoreWithdrawUi()
            }
            withdrawSubmitting = false
        }
    }

    private fun setWithdrawUiLocked(locked: Boolean, allowSmsEdit: Boolean = false) {
        withdrawBtn?.visibility = if (locked) View.GONE else View.VISIBLE
        countdownText?.visibility = if (locked) View.VISIBLE else View.GONE
        updateSmsBtn?.visibility = if (locked && allowSmsEdit) View.VISIBLE else View.GONE
        smsEditHint?.visibility = if (locked && allowSmsEdit) View.VISIBLE else View.GONE
        phoneInput?.isEnabled = !locked
        passwordInput?.isEnabled = !locked
        smsInput?.isEnabled = !locked || allowSmsEdit
    }

    private fun restoreWithdrawUi() {
        withdrawSubmitting = false
        countdownJob?.cancel()
        countdownJob = null
        activeTaskId = null
        setWithdrawUiLocked(false)
        countdownText?.text = ""
    }

    private fun updateSmsCode() {
        val taskId = activeTaskId
        val smsCode = smsInput?.text?.toString()?.trim().orEmpty()
        if (taskId.isNullOrBlank()) {
            toast("暂无进行中的任务")
            return
        }
        if (smsCode.isEmpty()) {
            toast("请输入验证码")
            return
        }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.updateKuwoSmsCode(taskId, smsCode) }
                .onSuccess {
                    appendLog("✅ 验证码已同步到后端")
                    toast("验证码已更新")
                    runCatching {
                        val raw = prefs().getString("state", null) ?: return@runCatching
                        val json = JSONObject(raw)
                        json.put("smsCode", smsCode)
                        prefs().edit().putString("state", json.toString()).apply()
                    }
                }
                .onFailure {
                    toast(it.message ?: "更新失败")
                    handlePortalError(it)
                }
        }
    }

    private fun monitorTask(taskId: String, immediate: Boolean) {
        monitorJob?.cancel()
        val pollMs = if (immediate) 250L else 800L
        val maxPoll = if (immediate) 120 else 600
        monitorJob = lifecycleScope.launch {
            var pollCount = 0
            var errorCount = 0
            while (isActive) {
                pollCount++
                var shouldStop = false
                runCatching { AppServices.portalRepository.fetchKuwoWithdrawStatus(taskId = taskId) }
                    .onSuccess { task ->
                        errorCount = 0
                        if (task != null) {
                            if (task.status == "pending" || task.status == "running") {
                                applyTaskResult(task, immediate)
                            } else if (applyTaskResult(task, immediate)) {
                                shouldStop = true
                            }
                        }
                    }
                    .onFailure {
                        errorCount++
                        if (errorCount > 20) {
                            countdownText?.text = "⏰ 网络异常，请返回页面查看结果"
                            shouldStop = true
                        }
                    }
                if (shouldStop) return@launch
                if (pollCount > maxPoll) {
                    countdownText?.text = "⏰ 仍在等待后端，继续同步…"
                }
                delay(pollMs)
            }
        }
    }

    private fun applyTaskResult(task: com.goudong.jd.data.model.KuwoWithdrawTask, immediate: Boolean): Boolean {
        flushTaskLogs(task.logs)
        when (task.status) {
            "pending" -> {
                if (!immediate) {
                    setWithdrawUiLocked(true, allowSmsEdit = true)
                    if (task.smsFatal == true) {
                        appendLog("⚠️ 后端反馈疑似验证码错误，请修改后点击「更新验证码」")
                    }
                }
                return false
            }
            "running" -> {
                if (immediate) countdownText?.text = "🚀 后端正在执行..."
                return false
            }
            "completed" -> {
                monitorJob?.cancel()
                countdownJob?.cancel()
                countdownText?.text = "✅ 抢兑已完成"
                appendLog("--- 任务结束 ---")
                restoreWithdrawUi()
                clearSavedState()
                return true
            }
            "failed" -> {
                monitorJob?.cancel()
                countdownJob?.cancel()
                countdownText?.text = "❌ 抢兑失败"
                appendLog("--- 任务失败 ---")
                restoreWithdrawUi()
                clearSavedState()
                return true
            }
        }
        return false
    }

    private fun runCountdown(info: KuwoTimeHelper.NextWithdrawInfo, taskId: String) {
        countdownJob?.cancel()
        val targetHour = info.hour
        val targetLabel = KuwoTimeHelper.formatHour(targetHour)
        var fired = false
        countdownJob = lifecycleScope.launch {
            while (isActive) {
                val remaining = KuwoTimeHelper.remainingMsUntilHour(targetHour)
                if (remaining > 20L * 3600L * 1000L) {
                    if (!fired) {
                        fired = true
                        countdownText?.text = "🚀 后端正在执行 $targetLabel 抢兑..."
                        monitorTask(taskId, false)
                    }
                    break
                }
                if (remaining > 0) {
                    val rMin = remaining / 60000
                    val rSec = (remaining % 60000) / 1000
                    countdownText?.text = "⏳ $targetLabel 自动抢兑 — 剩余 ${rMin}分${rSec}秒"
                } else if (!fired) {
                    fired = true
                    countdownText?.text = "🚀 后端正在执行 $targetLabel 抢兑..."
                    appendLog("⏰ 到达抢兑时间 $targetLabel，等待后端日志同步...")
                    monitorTask(taskId, false)
                    break
                }
                delay(100)
            }
        }
    }

    private fun saveKuwoState(
        phone: String,
        password: String,
        smsCode: String,
        quotaId: String,
        targetHour: Int?,
        taskId: String?,
        immediate: Boolean,
    ) {
        val json = JSONObject().apply {
            put("phone", phone)
            put("password", password)
            put("smsCode", smsCode)
            put("quotaId", quotaId)
            if (targetHour != null) put("targetHour", targetHour)
            if (taskId != null) put("taskId", taskId)
            put("immediate", immediate)
            put("savedAt", System.currentTimeMillis())
        }
        prefs().edit().putString("state", json.toString()).apply()
    }

    private fun updateSavedTaskId(taskId: String, immediate: Boolean, targetHour: Int?) {
        val raw = prefs().getString("state", null) ?: return
        runCatching {
            val json = JSONObject(raw)
            json.put("taskId", taskId)
            json.put("immediate", immediate)
            if (targetHour != null) json.put("targetHour", targetHour)
            prefs().edit().putString("state", json.toString()).apply()
        }
    }

    private fun clearSavedState() {
        prefs().edit().remove("state").apply()
    }

    private fun restoreKuwoState(phone: String) {
        val raw = prefs().getString("state", null) ?: return
        lifecycleScope.launch {
            runCatching {
                val json = JSONObject(raw)
                val savedPhone = json.optString("phone")
                if (savedPhone.isNotEmpty() && phone.isNotEmpty() && savedPhone != phone) {
                    clearActiveState()
                    return@runCatching
                }
                val taskIdFromState = json.optString("taskId").takeIf { it.isNotBlank() }
                if (taskIdFromState.isNullOrBlank()) {
                    clearActiveState()
                    return@runCatching
                }
                val task = runCatching {
                    AppServices.portalRepository.fetchKuwoWithdrawStatus(taskId = taskIdFromState)
                }.getOrNull()
                if (task != null && (task.status == "completed" || task.status == "failed")) {
                    taskLogIndex = task.logs?.size ?: 0
                    applyTaskResult(task, json.optBoolean("immediate"))
                    return@runCatching
                }
                if (task == null || (task.status != "pending" && task.status != "running")) {
                    clearActiveState()
                    restoreWithdrawUi()
                    return@runCatching
                }
                activeTaskId = task.id ?: taskIdFromState
                phoneInput?.setText(json.optString("phone").ifBlank { phone })
                passwordInput?.setText(json.optString("password"))
                if (task.status == "pending") {
                    smsInput?.setText(json.optString("smsCode"))
                }
                val quotaId = json.optString("quotaId", "30002")
                selectedQuotaId = quotaId
                quotaGroup?.let { group ->
                    for (i in 0 until group.childCount) {
                        val rb = group.getChildAt(i) as? RadioButton ?: continue
                        if (rb.tag == quotaId) {
                            group.check(rb.id)
                            break
                        }
                    }
                }
                appendLog("🔄 检测到进行中的抢兑任务，已恢复监控")
                taskLogIndex = task.logs?.size ?: 0
                val immediate = json.optBoolean("immediate") || task.immediate
                val hour = if (json.has("targetHour")) json.optInt("targetHour") else task.targetHour ?: 0
                setWithdrawUiLocked(true, allowSmsEdit = !immediate && task.status == "pending")
                if (!immediate) {
                    val remaining = KuwoTimeHelper.remainingMsUntilHour(hour)
                    if (remaining > 0) {
                        runCountdown(KuwoTimeHelper.NextWithdrawInfo(hour, inWindow = true, diffMin = 1), activeTaskId!!)
                    } else {
                        countdownText?.text = "🚀 后端正在执行 ${KuwoTimeHelper.formatHour(hour)} 抢兑..."
                        appendLog("🔄 倒计时已结束，等待后端执行结果…")
                    }
                }
                monitorTask(activeTaskId!!, immediate)
            }
        }
    }
}
