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
import com.goudong.jd.data.model.PortalDashboard
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.InnerTabSwipeHost
import com.goudong.jd.ui.common.MainTabResettable
import com.goudong.jd.ui.common.findFirstScrollView
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.toast
import com.goudong.jd.ui.common.WebBrowserActivity
import com.goudong.jd.ui.more.CoinLogActivity
import com.goudong.jd.data.model.AppEnvironment
import com.google.android.material.tabs.TabLayout
import kotlinx.coroutines.launch

class TasksFragment : Fragment(), MainTabResettable, InnerTabSwipeHost {

    private lateinit var tabs: TabLayout
    private lateinit var contentHost: LinearLayout
    private var currentTab = 0

    private var dashboard: PortalDashboard? = null
    private var authHintView: TextView? = null
    private var statsRow: LinearLayout? = null
    private var checkinBtn: Button? = null
    private var prayBtn: Button? = null
    private var isRedeeming = false

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val wrapper = LinearLayout(requireContext()).apply { orientation = LinearLayout.VERTICAL }
        val (scroll, root) = requireContext().makeScrollContainer()

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

        contentHost = root
        wrapper.addView(scroll)

        renderCurrentTab()
        return wrapper
    }

    override fun onResume() {
        super.onResume()
        loadDashboard()
    }

    private fun renderCurrentTab() {
        contentHost.removeAllViews()
        when (currentTab) {
            0 -> renderPointsTab()
            1 -> renderCoinLogTab()
        }
    }

    private fun renderPointsTab() {
        val ctx = requireContext()
        contentHost.addView(ctx.cardView().apply {
            setPadding(ctx.dp(14), ctx.dp(14), ctx.dp(14), ctx.dp(14))
            addView(TextView(ctx).apply {
                text = "每日任务"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
            })
            addView(ctx.captionText("打卡与祈福需有效的按月/按天项目").apply {
                setPadding(0, ctx.dp(4), 0, 0)
            })
            statsRow = LinearLayout(ctx).apply {
                orientation = LinearLayout.VERTICAL
                setPadding(0, ctx.dp(10), 0, 0)
            }.also { addView(it) }
            authHintView = TextView(ctx).apply {
                setTextColor(Color.parseColor("#DC2626"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, ctx.dp(10), 0, 0)
                visibility = View.GONE
            }.also { addView(it) }
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
                background = GradientDrawable().apply {
                    setColor(ContextCompat.getColor(ctx, R.color.brand_primary))
                    cornerRadius = ctx.dp(8).toFloat()
                }
                layoutParams = LinearLayout.LayoutParams(ctx.dp(110), ctx.dp(36)).apply { marginEnd = ctx.dp(10) }
                setOnClickListener { checkin() }
            }
            prayBtn = Button(ctx).apply {
                text = "每日祈福"
                setTextColor(Color.WHITE)
                setAllCaps(false)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply {
                    setColor(ContextCompat.getColor(ctx, R.color.brand_secondary))
                    cornerRadius = ctx.dp(8).toFloat()
                }
                layoutParams = LinearLayout.LayoutParams(ctx.dp(110), ctx.dp(36))
                setOnClickListener { pray() }
            }
            btnRow.addView(checkinBtn)
            btnRow.addView(prayBtn)
            addView(btnRow)
        })

        dashboard?.let { applyDashboard(it) }

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
                background = GradientDrawable().apply {
                    setColor(ContextCompat.getColor(context, R.color.brand_green))
                    cornerRadius = context.dp(8).toFloat()
                }
                layoutParams = LinearLayout.LayoutParams(requireContext().dp(90), requireContext().dp(36)).apply {
                    topMargin = context.dp(8)
                }
                setOnClickListener {
                    if (isRedeeming) return@setOnClickListener
                    val token = redeemInput.text?.toString().orEmpty().trim()
                    if (token.isBlank()) {
                        redeemInput.requestFocus()
                        toast("请输入卡密")
                        return@setOnClickListener
                    }
                    isRedeeming = true
                    isEnabled = false
                    text = "兑换中..."
                    lifecycleScope.launch {
                        runCatching { AppServices.portalRepository.redeemKey(token) }
                            .onSuccess { redeemInput.setText(""); toast(it) }
                            .onFailure { handlePortalError(it) }
                        isRedeeming = false
                        isEnabled = true
                        text = "兑换"
                    }
                }
            })
        })

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
            addView(ctx.captionText("查看积分收支明细，了解积分来源与去向").apply {
                setPadding(0, ctx.dp(4), 0, ctx.dp(12))
            })
            addView(Button(ctx).apply {
                text = "查看记录"
                setTextColor(Color.WHITE)
                setAllCaps(false)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                background = GradientDrawable().apply {
                    setColor(ContextCompat.getColor(ctx, R.color.brand_orange))
                    cornerRadius = ctx.dp(8).toFloat()
                }
                layoutParams = LinearLayout.LayoutParams(ctx.dp(100), ctx.dp(38))
                setOnClickListener {
                    startActivity(android.content.Intent(requireContext(), CoinLogActivity::class.java))
                }
            })
        })
    }

    private fun makeQuickTile(title: String, desc: String, tint: Int, onClick: () -> Unit): LinearLayout {
        val card = requireContext().cardView()
        card.layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
            marginStart = requireContext().dp(4)
            marginEnd = requireContext().dp(4)
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

    private fun loadDashboard() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchDashboard() }
                .onSuccess {
                    dashboard = it
                    applyDashboard(it)
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun applyDashboard(d: PortalDashboard) {
        statsRow?.let { row ->
            row.removeAllViews()
            row.addView(makeStatsGrid(d))
        }

        if (d.canCheckIn) {
            authHintView?.visibility = View.GONE
        } else {
            authHintView?.apply {
                text = d.canCheckInMessage?.takeIf { it.isNotBlank() }
                    ?: "请先前往「项目中心」上车有效的按月/按天活动"
                visibility = View.VISIBLE
            }
        }

        val checkinEnabled = d.canCheckIn && !d.checkedInToday
        checkinBtn?.apply {
            isEnabled = checkinEnabled
            alpha = if (checkinEnabled) 1f else 0.5f
            text = if (d.checkedInToday) "今日已打卡" else "每日打卡"
        }

        val prayEnabled = d.canCheckIn && !d.prayedToday
        prayBtn?.apply {
            isEnabled = prayEnabled
            alpha = if (prayEnabled) 1f else 0.5f
            text = if (d.prayedToday) "今日已祈福" else "每日祈福"
        }
    }

    private fun makeStatsGrid(d: PortalDashboard): LinearLayout {
        val ctx = requireContext()
        val bonusText = if (d.nextCheckInBonus > 0) {
            "+${d.nextCheckInBonus}（还差${d.daysUntilNextCheckInBonus}天）"
        } else {
            "已达最高档"
        }
        val topRow = LinearLayout(ctx).apply { orientation = LinearLayout.HORIZONTAL }
        topRow.addView(makeStatTile("连续打卡", "${d.continuousDays} 天", Color.parseColor("#2563EB")))
        topRow.addView(makeStatTile("今日状态", if (d.checkedInToday) "已打卡" else "未打卡",
            if (d.checkedInToday) Color.parseColor("#059669") else Color.parseColor("#64748B")))
        val bottomRow = LinearLayout(ctx).apply {
            orientation = LinearLayout.HORIZONTAL
            setPadding(0, ctx.dp(8), 0, 0)
        }
        bottomRow.addView(makeStatTile("下次奖励", bonusText, Color.parseColor("#D97706")))
        bottomRow.addView(makeStatTile("今日打卡", "${d.todayCheckInCount} 人", Color.parseColor("#7C3AED")))
        return LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            addView(topRow)
            addView(bottomRow)
        }
    }

    private fun makeStatTile(label: String, value: String, tint: Int): LinearLayout {
        val ctx = requireContext()
        return LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                marginEnd = ctx.dp(4)
            }
            background = GradientDrawable().apply {
                setColor(Color.parseColor("#F8FAFC"))
                cornerRadius = ctx.dp(8).toFloat()
            }
            setPadding(ctx.dp(10), ctx.dp(10), ctx.dp(10), ctx.dp(10))
            addView(TextView(ctx).apply {
                text = label
                setTextColor(Color.parseColor("#94A3B8"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f)
            })
            addView(TextView(ctx).apply {
                text = value
                setTextColor(tint)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTypeface(typeface, Typeface.BOLD)
                setPadding(0, ctx.dp(2), 0, 0)
            })
        }
    }

    private fun checkin() {
        val d = dashboard
        if (d != null && !d.canCheckIn) {
            toast(d.canCheckInMessage ?: "暂无法打卡")
            return
        }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.checkin() }
                .onSuccess {
                    toast(it)
                    loadDashboard()
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun pray() {
        val d = dashboard
        if (d != null && !d.canCheckIn) {
            toast(d.canCheckInMessage ?: "暂无法祈福")
            return
        }
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.pray() }
                .onSuccess {
                    toast(it)
                    loadDashboard()
                }
                .onFailure { handlePortalError(it) }
        }
    }

    fun refreshDashboard() {
        renderCurrentTab()
        loadDashboard()
    }

    override fun resetToInitialState() {
        if (!::tabs.isInitialized) return
        currentTab = 0
        tabs.getTabAt(0)?.select()
        view?.findFirstScrollView()?.scrollTo(0, 0)
        renderCurrentTab()
        loadDashboard()
    }

    override val innerTabCount: Int
        get() = if (::tabs.isInitialized) tabs.tabCount else 0

    override val innerTabIndex: Int
        get() = currentTab

    override fun selectInnerTab(index: Int) {
        tabs.getTabAt(index)?.select()
    }
}
