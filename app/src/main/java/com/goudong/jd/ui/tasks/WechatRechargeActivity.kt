package com.goudong.jd.ui.tasks

import android.app.Dialog
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.graphics.BitmapFactory
import android.graphics.Typeface
import android.graphics.drawable.ColorDrawable
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.view.WindowManager
import android.widget.Button
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.ScrollView
import android.widget.TextView
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.AppEnvironment
import com.goudong.jd.data.model.WechatRechargeConfig
import com.goudong.jd.data.model.WechatRechargeOrder
import com.goudong.jd.data.model.WechatRechargeTier
import com.goudong.jd.ui.common.AppTheme
import com.goudong.jd.ui.common.WebBrowserActivity
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.themeColor
import com.goudong.jd.ui.common.toast
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

class WechatRechargeActivity : AppCompatActivity() {

    private lateinit var offlineBanner: TextView
    private lateinit var noticeBox: TextView
    private lateinit var tiersHost: LinearLayout
    private lateinit var disabledHint: TextView
    private lateinit var loadingBar: ProgressBar

    private var config: WechatRechargeConfig? = null
    private var creating = false
    private var pollJob: Job? = null
    private var countdownJob: Job? = null
    private var activeOrderNo: String? = null
    private var resultShown = false
    private var currentAmount = ""

    private var payDialog: Dialog? = null
    private var payAmountView: TextView? = null
    private var payMetaView: TextView? = null
    private var payStatusView: TextView? = null
    private var payTimerView: TextView? = null
    private var payOrderNoView: TextView? = null
    private var payQrView: ImageView? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AppTheme.applySystemBars(this)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        title = "微信赞赏充值"

        val wrapper = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setBackgroundColor(themeColor(R.color.surface_soft))
        }
        loadingBar = ProgressBar(this).apply {
            isIndeterminate = true
            visibility = View.GONE
        }
        wrapper.addView(loadingBar)

        val (scroll, root) = makeScrollContainer()
        wrapper.addView(scroll)
        setContentView(wrapper)

        offlineBanner = TextView(this).apply {
            setTextColor(themeColor(R.color.negative))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            setPadding(dp(12), dp(10), dp(12), dp(10))
            visibility = View.GONE
            background = GradientDrawable().apply {
                setColor(0x14EF4444)
                cornerRadius = dp(10).toFloat()
            }
        }
        noticeBox = TextView(this).apply {
            setTextColor(themeColor(R.color.text_secondary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            setPadding(dp(12), dp(10), dp(12), dp(10))
            visibility = View.GONE
            setLineSpacing(dp(2).toFloat(), 1f)
            background = GradientDrawable().apply {
                setColor(0x14F59E0B)
                cornerRadius = dp(10).toFloat()
            }
        }
        tiersHost = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL }
        disabledHint = TextView(this).apply {
            text = "微信赞赏充值暂不可用，请使用积分购买或卡密兑换。"
            setTextColor(themeColor(R.color.text_muted))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            visibility = View.GONE
        }

        root.addView(cardView().apply {
            setPadding(dp(16), dp(16), dp(16), dp(16))
            addView(captionText("选择充值档位，按弹窗精确金额支付后自动到账").apply {
                setPadding(0, 0, 0, dp(12))
            })
            addView(offlineBanner)
            addView(noticeBox, LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                topMargin = dp(10)
            })
            addView(tiersHost, LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                topMargin = dp(14)
            })
            addView(disabledHint, LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                topMargin = dp(10)
            })
            addView(LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                setPadding(0, dp(16), 0, 0)
                addView(makeOutlineButton("充值记录", weight = 1f) { showHistory() })
                addView(makeOutlineButton("外链购买", weight = 1f) {
                    val url = config?.externalPurchaseUrl?.takeIf { it.isNotBlank() } ?: AppEnvironment.COIN_PURCHASE_URL
                    startActivity(WebBrowserActivity.intent(this@WechatRechargeActivity, url, "积分购买"))
                }.apply { layoutParams = (layoutParams as LinearLayout.LayoutParams).apply { marginStart = dp(10) } })
            })
        })

        loadConfig()
    }

    override fun onDestroy() {
        stopPolling()
        payDialog?.dismiss()
        super.onDestroy()
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    private fun loadConfig() {
        loadingBar.visibility = View.VISIBLE
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchWechatRechargeConfig() }
                .onSuccess { applyConfig(it) }
                .onFailure { handlePortalError(it) }
            loadingBar.visibility = View.GONE
        }
    }

    private fun applyConfig(cfg: WechatRechargeConfig) {
        config = cfg
        when {
            cfg.enabled && !cfg.billAccountOnline -> {
                offlineBanner.visibility = View.VISIBLE
                offlineBanner.text = cfg.billAccountMessage ?: cfg.blockReason ?: "查账账号离线，暂无法发起赞赏充值"
            }
            cfg.enabled && !cfg.canRecharge && !cfg.blockReason.isNullOrBlank() -> {
                offlineBanner.visibility = View.VISIBLE
                offlineBanner.text = cfg.blockReason
            }
            else -> offlineBanner.visibility = View.GONE
        }

        val notices = cfg.paymentNotice.orEmpty().take(2)
        if (cfg.enabled && notices.isNotEmpty()) {
            noticeBox.visibility = View.VISIBLE
            noticeBox.text = notices.joinToString("\n")
        } else {
            noticeBox.visibility = View.GONE
        }

        tiersHost.removeAllViews()
        if (!cfg.enabled) {
            disabledHint.visibility = View.VISIBLE
            return
        }
        disabledHint.visibility = if (cfg.canRecharge) View.GONE else View.VISIBLE

        val tiers = cfg.tiers.orEmpty()
        var row: LinearLayout? = null
        tiers.forEachIndexed { index, tier ->
            if (index % 2 == 0) {
                row = LinearLayout(this).apply {
                    orientation = LinearLayout.HORIZONTAL
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        if (index > 0) topMargin = dp(10)
                    }
                }
                tiersHost.addView(row)
            }
            row?.addView(makeTierCard(tier, cfg.canRecharge))
        }
    }

    private fun makeTierCard(tier: WechatRechargeTier, canRecharge: Boolean): LinearLayout {
        val card = cardView().apply {
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                marginStart = dp(4)
                marginEnd = dp(4)
            }
            setPadding(dp(12), dp(14), dp(12), dp(14))
            isEnabled = canRecharge && !creating
            alpha = if (canRecharge && !creating) 1f else 0.55f
            setOnClickListener { startRecharge(tier.fen) }
        }
        card.addView(TextView(this).apply {
            text = "${tier.yuan} 元"
            setTextColor(themeColor(R.color.brand_primary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 18f)
            setTypeface(typeface, Typeface.BOLD)
            gravity = Gravity.CENTER
        })
        card.addView(TextView(this).apply {
            text = "${tier.points} 积分"
            setTextColor(themeColor(R.color.text_secondary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            gravity = Gravity.CENTER
            setPadding(0, dp(4), 0, 0)
        })
        return card
    }

    private fun startRecharge(fen: Int) {
        if (creating) return
        val cfg = config
        if (cfg != null && !cfg.canRecharge) {
            toast(cfg.blockReason ?: "微信赞赏充值暂不可用")
            return
        }
        creating = true
        setTiersLoading(true)
        openPayDialogLoading(fen)
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.createWechatRechargeOrder(fen) }
                .onSuccess { result ->
                    if (result.replacedPrevious) {
                        toast("上一笔未支付订单已自动取消，请按本页新金额支付")
                    }
                    result.message?.takeIf { it.isNotBlank() }?.let { toast(it) }
                    openPayDialog(result.order)
                }
                .onFailure {
                    dismissPayDialog()
                    handlePortalError(it)
                }
            creating = false
            setTiersLoading(false)
        }
    }

    private fun setTiersLoading(loading: Boolean) {
        for (i in 0 until tiersHost.childCount) {
            val row = tiersHost.getChildAt(i) as? LinearLayout ?: continue
            for (j in 0 until row.childCount) {
                val card = row.getChildAt(j)
                card.isEnabled = !loading && (config?.canRecharge == true)
                card.alpha = if (loading) 0.55f else 1f
            }
        }
    }

    private fun openPayDialogLoading(fen: Int) {
        resultShown = false
        currentAmount = ""
        stopPolling(keepCountdown = false)
        val tier = config?.tiers?.find { it.fen == fen }
        ensurePayDialog()
        payAmountView?.text = "计算中…"
        payMetaView?.text = tier?.let { "档位 ${it.yuan} 元 · 到账 ${it.points} 积分" } ?: "正在生成订单"
        payOrderNoView?.text = "订单号：创建中…"
        payStatusView?.text = "正在创建订单，请稍候…"
        payTimerView?.text = "--:--"
        payQrView?.setImageDrawable(null)
        payDialog?.show()
    }

    private fun openPayDialog(order: WechatRechargeOrder) {
        resultShown = false
        activeOrderNo = order.orderNo
        currentAmount = formatYuan(order.paymentFen)
        ensurePayDialog()
        payAmountView?.text = "¥ $currentAmount"
        payMetaView?.text = "档位 ${order.requestedFen / 100} 元 · 到账 ${order.points.takeIf { it > 0 } ?: order.requestedFen} 积分"
        payOrderNoView?.text = "订单号：${order.orderNo}"
        payStatusView?.text = "请使用微信扫一扫，务必支付 ¥$currentAmount"
        payDialog?.show()
        loadQrCode(order)
        startCountdown(order.remainingSec)
        startPolling(order.orderNo)
    }

    private fun ensurePayDialog() {
        if (payDialog != null) return

        val panel = cardView().apply {
            setPadding(dp(20), dp(20), dp(20), dp(16))
        }
        payStatusView = TextView(this).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTextColor(themeColor(R.color.text_secondary))
            gravity = Gravity.CENTER
        }
        payAmountView = TextView(this).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 32f)
            setTypeface(typeface, Typeface.BOLD)
            setTextColor(themeColor(R.color.brand_primary))
            gravity = Gravity.CENTER
            setPadding(0, dp(6), 0, dp(4))
        }
        payMetaView = TextView(this).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            setTextColor(themeColor(R.color.text_muted))
            gravity = Gravity.CENTER
        }
        payQrView = ImageView(this).apply {
            adjustViewBounds = true
            layoutParams = LinearLayout.LayoutParams(dp(220), dp(220)).apply {
                gravity = Gravity.CENTER_HORIZONTAL
                topMargin = dp(14)
                bottomMargin = dp(10)
            }
        }
        payTimerView = TextView(this).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTextColor(ContextCompat.getColor(this@WechatRechargeActivity, R.color.brand_orange))
            gravity = Gravity.CENTER
        }
        payOrderNoView = TextView(this).apply {
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            setTextColor(themeColor(R.color.text_muted))
            gravity = Gravity.CENTER
            setPadding(0, dp(4), 0, dp(12))
        }

        val copyBtn = Button(this).apply {
            text = "复制金额"
            isAllCaps = false
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTextColor(themeColor(R.color.chip_active_text))
            background = GradientDrawable().apply {
                setColor(ContextCompat.getColor(context, R.color.brand_primary))
                cornerRadius = dp(10).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(0, dp(42), 1f)
            setOnClickListener { copyAmount() }
        }
        val closeBtn = Button(this).apply {
            text = "关闭"
            isAllCaps = false
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTextColor(themeColor(R.color.text_primary))
            background = GradientDrawable().apply {
                setColor(themeColor(R.color.surface_card))
                setStroke(dp(1), themeColor(R.color.border_default))
                cornerRadius = dp(10).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(0, dp(42), 1f).apply { marginStart = dp(10) }
            setOnClickListener {
                if (!resultShown && !activeOrderNo.isNullOrBlank()) {
                    toast("支付窗口已关闭，系统仍会自动检测到账；重新点档位将作废本笔并生成新订单")
                }
                payDialog?.hide()
            }
        }

        panel.addView(TextView(this).apply {
            text = "微信赞赏支付"
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 17f)
            setTypeface(typeface, Typeface.BOLD)
            setTextColor(themeColor(R.color.text_primary))
            gravity = Gravity.CENTER
            setPadding(0, 0, 0, dp(10))
        })
        panel.addView(payStatusView)
        panel.addView(payAmountView)
        panel.addView(payMetaView)
        panel.addView(TextView(this).apply {
            text = "请使用微信扫一扫"
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            setTextColor(themeColor(R.color.text_muted))
            gravity = Gravity.CENTER
            setPadding(0, dp(10), 0, 0)
        })
        panel.addView(payQrView)
        panel.addView(TextView(this).apply {
            text = "剩余支付时间"
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            setTextColor(themeColor(R.color.text_muted))
            gravity = Gravity.CENTER
        })
        panel.addView(payTimerView)
        panel.addView(payOrderNoView)
        panel.addView(LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            addView(copyBtn)
            addView(closeBtn)
        })

        val scroll = ScrollView(this).apply {
            addView(panel, ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT))
        }

        payDialog = Dialog(this).apply {
            setContentView(scroll)
            window?.setBackgroundDrawable(ColorDrawable(android.graphics.Color.TRANSPARENT))
            window?.setLayout(WindowManager.LayoutParams.MATCH_PARENT, WindowManager.LayoutParams.WRAP_CONTENT)
            setCanceledOnTouchOutside(false)
        }
    }

    private fun dismissPayDialog() {
        payDialog?.hide()
    }

    private fun copyAmount() {
        if (currentAmount.isBlank()) {
            toast("金额尚未生成")
            return
        }
        val clip = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        clip.setPrimaryClip(ClipData.newPlainText("amount", currentAmount))
        toast("金额已复制：$currentAmount")
    }

    private fun loadQrCode(order: WechatRechargeOrder) {
        val path = order.qrcodeUrl ?: return
        val url = resolveQrPath(path)
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.downloadWechatRechargeQR(url) }
                .onSuccess { bytes ->
                    val bitmap = BitmapFactory.decodeByteArray(bytes, 0, bytes.size)
                    payQrView?.setImageBitmap(bitmap)
                }
                .onFailure { payStatusView?.text = "收款码加载失败，请稍后重试" }
        }
    }

    private fun resolveQrPath(path: String): String {
        var resolved = if (path.startsWith("http")) path else AppEnvironment.BASE_URL.trimEnd('/') + path
        if (resolved.contains("/api/portal/wechat-recharge/qrcode/")) {
            resolved += if (resolved.contains("?")) "&" else "?"
            resolved += "_t=${System.currentTimeMillis()}"
        }
        return resolved
    }

    private fun startCountdown(sec: Long) {
        countdownJob?.cancel()
        countdownJob = lifecycleScope.launch {
            var left = sec
            while (isActive && left >= 0) {
                val m = left / 60
                val s = left % 60
                payTimerView?.text = "$m:${s.toString().padStart(2, '0')}"
                if (left == 0L) break
                delay(1000)
                left--
            }
            if (left < 0) payTimerView?.text = "已超时"
        }
    }

    private fun startPolling(orderNo: String) {
        stopPolling(keepCountdown = true)
        activeOrderNo = orderNo
        pollJob = lifecycleScope.launch {
            while (isActive && activeOrderNo == orderNo) {
                runCatching { AppServices.portalRepository.fetchWechatRechargeOrder(orderNo) }
                    .onSuccess { handleOrderUpdate(it) }
                delay(2000)
            }
        }
    }

    private fun handleOrderUpdate(order: WechatRechargeOrder) {
        when (order.status) {
            "paid" -> {
                payStatusView?.text = "充值成功，正在展示到账详情…"
                stopPolling()
                showSuccessDialog(order)
            }
            "crediting" -> payStatusView?.text = "支付已确认，正在入账…"
            "expired", "failed", "wrong_amount" -> {
                stopPolling()
                if (payDialog?.isShowing == true) {
                    payStatusView?.text = if (order.status == "expired") "订单已超时，未检测到入账" else "订单已结束"
                }
            }
            else -> {
                if (order.remainingSec > 0) startCountdown(order.remainingSec)
            }
        }
    }

    private fun showSuccessDialog(order: WechatRechargeOrder) {
        resultShown = true
        dismissPayDialog()
        val paidYuan = if (order.paidFen > 0) formatYuan(order.paidFen) else currentAmount
        val message = buildString {
            appendLine("实付金额：¥ $paidYuan")
            appendLine("到账积分：+${order.points} 积分")
            appendLine("当前余额：${order.coin ?: "-"} 积分")
            appendLine("充值时间：${order.paidAt ?: order.createdAt ?: "-"}")
            append("订单号：${order.orderNo}")
        }
        AlertDialog.Builder(this)
            .setTitle("充值成功")
            .setMessage(message)
            .setPositiveButton("我知道了", null)
            .show()
        toast("充值成功，到账 ${order.points} 积分")
    }

    private fun showHistory() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchWechatRechargeHistory() }
                .onSuccess { page ->
                    if (page.list.isEmpty()) {
                        toast("暂无充值记录")
                        return@onSuccess
                    }
                    val lines = page.list.take(20).joinToString("\n\n") { order ->
                        val yuan = if (order.paidFen > 0) formatYuan(order.paidFen) else formatYuan(order.paymentFen)
                        "${statusLabel(order.status)} · ¥$yuan · +${order.points}积分\n${order.orderNo}\n${order.paidAt ?: order.createdAt ?: ""}"
                    }
                    AlertDialog.Builder(this@WechatRechargeActivity)
                        .setTitle("充值记录")
                        .setMessage(lines)
                        .setPositiveButton("关闭", null)
                        .show()
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun stopPolling(keepCountdown: Boolean = false) {
        pollJob?.cancel()
        pollJob = null
        if (!keepCountdown) {
            countdownJob?.cancel()
            countdownJob = null
        }
        if (!keepCountdown) {
            activeOrderNo = null
        }
    }

    private fun makeOutlineButton(text: String, weight: Float = 0f, onClick: () -> Unit): Button {
        return Button(this).apply {
            this.text = text
            isAllCaps = false
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTextColor(themeColor(R.color.brand_primary))
            background = GradientDrawable().apply {
                setColor(themeColor(R.color.surface_card))
                setStroke(dp(1), themeColor(R.color.border_default))
                cornerRadius = dp(10).toFloat()
            }
            layoutParams = if (weight > 0f) {
                LinearLayout.LayoutParams(0, dp(40), weight)
            } else {
                LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, dp(40))
            }
            setOnClickListener { onClick() }
        }
    }

    private fun formatYuan(fen: Int): String = String.format("%.2f", fen / 100.0)

    private fun statusLabel(status: String): String = when (status) {
        "paid" -> "已到账"
        "pending", "preparing" -> "待支付"
        "crediting" -> "入账中"
        "expired" -> "已超时"
        "failed" -> "已取消"
        else -> status
    }

    companion object {
        fun intent(context: Context) = Intent(context, WechatRechargeActivity::class.java)
    }
}
