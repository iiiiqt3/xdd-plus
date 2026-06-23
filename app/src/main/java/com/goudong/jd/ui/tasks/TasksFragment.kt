package com.goudong.jd.ui.tasks

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
import android.widget.LinearLayout
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.PortalJdAccount
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.toast
import com.goudong.jd.ui.common.WebBrowserActivity
import com.goudong.jd.ui.more.CoinLogActivity
import com.goudong.jd.data.model.AppEnvironment
import com.google.android.material.tabs.TabLayout
import kotlinx.coroutines.launch

class TasksFragment : Fragment() {

    private lateinit var tabs: TabLayout
    private lateinit var contentHost: LinearLayout
    private var currentTab = 0

    private var accounts: List<PortalJdAccount> = emptyList()
    private var authHintView: TextView? = null
    private var statusView: TextView? = null
    private var checkinBtn: Button? = null
    private var prayBtn: Button? = null
    private var isRedeeming = false

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        // 固定的外层容器（TabLayout在外面，不随内容滚动）
        val wrapper = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        val (scroll, root) = requireContext().makeScrollContainer()

        // ====== TabLayout（固定在顶部） ======
        tabs = TabLayout(requireContext()).apply {
            setPadding(requireContext().dp(14), 0, requireContext().dp(14), requireContext().dp(8))
            addTab(newTab().setText("积分任务"))
            addTab(newTab().setText("积分变动"))
            addOnTabSelectedListener(object : TabLayout.OnTabSelectedListener {
                override fun onTabSelected(tab: TabLayout.Tab) {
                    currentTab = tab.position
                    renderCurrentTab()
                }
                override fun onTabUnselected(tab: TabLayout.Tab) = Unit
                override fun onTabReselected(tab: TabLayout.Tab) {
                    if (tab.position == currentTab) renderCurrentTab()
                }
            })
        }
        wrapper.addView(tabs)

        // ====== 内容区（可滚动） ======
        contentHost = root
        wrapper.addView(scroll)

        renderCurrentTab()
        return wrapper
    }

    override fun onResume() {
        super.onResume()
        loadAccounts()
    }

    private fun renderCurrentTab() {
        contentHost.removeAllViews()
        when (currentTab) {
            0 -> renderPointsTab()
            1 -> renderCoinLogTab()
        }
    }

    // ==================== 积分任务 Tab ====================

    private fun renderPointsTab() {
        // ====== 每日任务卡片 ======
        val ctx = requireContext()
        contentHost.addView(ctx.cardView().apply {
            setPadding(ctx.dp(14), ctx.dp(14), ctx.dp(14), ctx.dp(14))
            addView(TextView(ctx).apply {
                text = "每日任务"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
            })
            // 授权提示
            authHintView = TextView(ctx).apply {
                text = "暂无有效授权项目，请先前往「项目中心」上车活动"
                setTextColor(Color.parseColor("#DC2626"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, ctx.dp(10), 0, 0)
                visibility = View.GONE
            }.also { addView(it) }
            // 状态
            statusView = TextView(ctx).apply {
                text = ""
                setTextColor(Color.parseColor("#64748B"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, ctx.dp(10), 0, 0)
                visibility = View.GONE
            }.also { addView(it) }
            // 按钮行（缩短宽度，居中）
            val btnRow = LinearLayout(ctx).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_HORIZONTAL
                setPadding(0, ctx.dp(14), 0, 0)
            }
            checkinBtn = Button(ctx).apply {
                text = "每日打卡"
                setTextColor(Color.WHITE)
                setAllCaps(false)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply { setColor(ContextCompat.getColor(ctx, R.color.brand_primary)); cornerRadius = ctx.dp(8).toFloat() }
                layoutParams = LinearLayout.LayoutParams(ctx.dp(110), ctx.dp(36)).apply { marginEnd = ctx.dp(10) }
                setOnClickListener { checkin() }
            }
            prayBtn = Button(ctx).apply {
                text = "每日祈福"
                setTextColor(Color.WHITE)
                setAllCaps(false)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply { setColor(ContextCompat.getColor(ctx, R.color.brand_secondary)); cornerRadius = ctx.dp(8).toFloat() }
                layoutParams = LinearLayout.LayoutParams(ctx.dp(110), ctx.dp(36))
                setOnClickListener { pray() }
            }
            btnRow.addView(checkinBtn); btnRow.addView(prayBtn)
            addView(btnRow)
        })

        // ====== 卡密兑换卡片 ======
        val redeemInput = requireContext().inputField("请输入卡密（XDD... 或 ZSKM...）").apply {
            minHeight = requireContext().dp(38)
        }
        contentHost.addView(requireContext().cardView().apply {
            setPadding(context.dp(14), context.dp(14), context.dp(14), context.dp(14))
            addView(TextView(context).apply {
                text = "卡密兑换"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
            })
            addView(ctx.captionText("支持 XDD / ZSKM 开头的卡密").apply { setPadding(0, ctx.dp(4), 0, ctx.dp(8)) })
            addView(redeemInput)
            addView(Button(context).apply {
                text = "兑换"
                setTextColor(Color.WHITE)
                setAllCaps(false)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply { setColor(ContextCompat.getColor(context, R.color.brand_green)); cornerRadius = context.dp(8).toFloat() }
                layoutParams = LinearLayout.LayoutParams(requireContext().dp(90), requireContext().dp(36)).apply { topMargin = context.dp(8) }
                setOnClickListener {
                    if (isRedeeming) return@setOnClickListener
                    val token = redeemInput.text?.toString().orEmpty().trim()
                    if (token.isBlank()) { redeemInput.requestFocus(); toast("请输入卡密"); return@setOnClickListener }
                    isRedeeming = true; isEnabled = false; text = "兑换中..."
                    lifecycleScope.launch {
                        runCatching { AppServices.portalRepository.redeemKey(token) }
                            .onSuccess { redeemInput.setText(""); toast(it) }
                            .onFailure { handlePortalError(it) }
                        isRedeeming = false; isEnabled = true; text = "兑换"
                    }
                }
            })
        })

        // ====== 积分购买 + 手机卡业务（两列） ======
        val gridRow = LinearLayout(context).apply { orientation = LinearLayout.HORIZONTAL }
        gridRow.addView(makeQuickTile("积分购买", "购买积分", ContextCompat.getColor(requireContext(), R.color.brand_secondary)) {
            startActivity(WebBrowserActivity.intent(requireContext(), AppEnvironment.COIN_PURCHASE_URL, "积分购买"))
        })
        gridRow.addView(makeQuickTile("手机卡", "办理大流量卡", ContextCompat.getColor(requireContext(), R.color.brand_primary)) {
            startActivity(WebBrowserActivity.intent(requireContext(), AppEnvironment.MORE_WOOL_URL, "手机卡业务"))
        })
        contentHost.addView(gridRow)
    }

    private fun renderCoinLogTab() {
        val ctx = requireContext()
        contentHost.addView(ctx.cardView().apply {
            setPadding(ctx.dp(14), ctx.dp(14), ctx.dp(14), ctx.dp(14))
            addView(TextView(ctx).apply {
                text = "积分变动记录"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
            })
            addView(ctx.captionText("查看积分收支明细，了解积分来源与去向").apply { setPadding(0, ctx.dp(4), 0, ctx.dp(12)) })
            addView(Button(ctx).apply {
                text = "查看记录"
                setTextColor(Color.WHITE)
                setAllCaps(false)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply { setColor(ContextCompat.getColor(ctx, R.color.brand_orange)); cornerRadius = ctx.dp(8).toFloat() }
                layoutParams = LinearLayout.LayoutParams(ctx.dp(100), ctx.dp(38))
                setOnClickListener { startActivity(android.content.Intent(requireContext(), CoinLogActivity::class.java)) }
            })
        })
    }

    private fun makeQuickTile(title: String, desc: String, tint: Int, onClick: () -> Unit): LinearLayout {
        val card = requireContext().cardView()
        card.layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
            marginStart = requireContext().dp(4); marginEnd = requireContext().dp(4)
        }
        card.setPadding(requireContext().dp(12), requireContext().dp(12), requireContext().dp(12), requireContext().dp(12))
        card.addView(TextView(requireContext()).apply {
            text = title
            setTextColor(tint)
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTypeface(typeface, Typeface.BOLD)
        })
        card.addView(TextView(requireContext()).apply {
            text = desc
            setTextColor(Color.parseColor("#94A3B8"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f)
            setPadding(0, requireContext().dp(2), 0, 0)
        })
        card.setOnClickListener { onClick() }
        return card
    }

    // ==================== 每日打卡/祈福 ====================

    private fun loadAccounts() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchJdAccounts() }
                .onSuccess { accounts = it; updateAuthHint() }
                .onFailure { accounts = emptyList() }
        }
    }

    private fun updateAuthHint() {
        val hasValid = accounts.any { it.valid }
        authHintView?.visibility = if (hasValid) View.GONE else View.VISIBLE
        statusView?.visibility = View.GONE
    }

    private fun checkin() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.checkin() }
                .onSuccess { toast(it) }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun pray() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.pray() }
                .onSuccess { toast(it) }
                .onFailure { handlePortalError(it) }
        }
    }

    fun refreshDashboard() {
        renderCurrentTab()
        loadAccounts()
    }
}
