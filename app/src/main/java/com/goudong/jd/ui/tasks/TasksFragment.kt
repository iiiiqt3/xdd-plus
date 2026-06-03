package com.goudong.jd.ui.tasks

import android.graphics.Color
import android.graphics.Typeface
import android.os.Bundle
import android.util.TypedValue
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
import com.goudong.jd.data.model.AppEnvironment
import com.goudong.jd.data.model.PortalDashboard
import com.goudong.jd.ui.common.WebBrowserActivity
import com.goudong.jd.ui.common.actionGridTile
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.heroCard
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.primaryButton
import com.goudong.jd.ui.common.toast
import kotlinx.coroutines.launch

class TasksFragment : Fragment() {
    private var isRedeeming = false
    private var checkinBtn: Button? = null
    private var prayBtn: Button? = null
    private var authHintView: TextView? = null
    private var statusView: TextView? = null

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val (scroll, root) = requireContext().makeScrollContainer()

        root.addView(requireContext().heroCard("积分任务", "每日打卡、祈福与积分补充入口", ContextCompat.getColor(requireContext(), R.color.brand_orange)))

        root.addView(requireContext().heroCard("每日任务", "完成任务即可获得积分奖励", ContextCompat.getColor(requireContext(), R.color.brand_primary)).apply {
            authHintView = TextView(requireContext()).apply {
                text = "⚠️ 暂无有效授权项目，打卡和祈福需要至少一个有效授权的活动账号才能使用。请先前往「项目中心」上车活动。"
                setTextColor(Color.parseColor("#DC2626"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setLineSpacing(0f, 1.4f)
                setPadding(requireContext().dp(8), requireContext().dp(8), requireContext().dp(8), requireContext().dp(8))
                setBackgroundColor(Color.parseColor("#FEF2F2"))
                visibility = View.GONE
            }
            addView(authHintView)

            statusView = TextView(requireContext()).apply {
                text = ""
                setTextColor(ContextCompat.getColor(requireContext(), R.color.brand_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setPadding(0, requireContext().dp(6), 0, requireContext().dp(6))
                visibility = View.GONE
            }
            addView(statusView)

            val grid = LinearLayout(requireContext()).apply {
                orientation = LinearLayout.HORIZONTAL
            }
            checkinBtn = requireContext().primaryButton("每日打卡", ContextCompat.getColor(requireContext(), R.color.brand_primary)).apply {
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                    marginEnd = requireContext().dp(6)
                }
                setOnClickListener { checkin() }
            }
            prayBtn = requireContext().primaryButton("每日祈福", ContextCompat.getColor(requireContext(), R.color.brand_secondary)).apply {
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
                setOnClickListener { pray() }
            }
            grid.addView(checkinBtn)
            grid.addView(prayBtn)
            addView(grid)
        })

        root.addView(requireContext().heroCard("积分购买", "积分不足时可通过购买页面补充积分", ContextCompat.getColor(requireContext(), R.color.brand_secondary)).apply {
            addView(requireContext().primaryButton("前往购买") {
                startActivity(WebBrowserActivity.intent(requireContext(), AppEnvironment.COIN_PURCHASE_URL, "积分购买"))
            })
        })

        val redeemInput = requireContext().inputField("请输入卡密（XDD... 或 ZSKM...）")
        root.addView(requireContext().heroCard("卡密兑换", "购买卡密后可直接兑换积分，支持普通卡密和赠送卡密", ContextCompat.getColor(requireContext(), R.color.brand_green)).apply {
            addView(redeemInput)
            addView(requireContext().captionText("卡密格式以 XDD 或 ZSKM 开头，每批赠送卡密通常限用一张").apply {
                setPadding(0, 0, 0, requireContext().dp(10))
            })
            val redeemBtn = requireContext().primaryButton("兑换积分", ContextCompat.getColor(requireContext(), R.color.brand_green))
            redeemBtn.setOnClickListener {
                if (isRedeeming) return@setOnClickListener
                val token = redeemInput.text?.toString().orEmpty().trim()
                if (token.isBlank()) {
                    redeemInput.requestFocus()
                    toast("请输入卡密")
                    return@setOnClickListener
                }
                isRedeeming = true
                redeemBtn.isEnabled = false
                redeemBtn.text = "兑换中..."
                lifecycleScope.launch {
                    runCatching { AppServices.portalRepository.redeemKey(token) }
                        .onSuccess {
                            redeemInput.setText("")
                            toast(it)
                        }
                        .onFailure { handlePortalError(it) }
                    isRedeeming = false
                    redeemBtn.isEnabled = true
                    redeemBtn.text = "兑换积分"
                }
            }
            addView(redeemBtn)
        })

        root.addView(requireContext().heroCard("积分变动记录", "查看积分收支明细，了解积分来源与去向", ContextCompat.getColor(requireContext(), R.color.brand_orange)).apply {
            addView(requireContext().primaryButton("查看记录") {
                startActivity(android.content.Intent(requireContext(), com.goudong.jd.ui.more.CoinLogActivity::class.java))
            })
        })

        return scroll
    }

    override fun onResume() {
        super.onResume()
        loadDashboard()
    }

    fun refreshDashboard() {
        loadDashboard()
    }

    private fun loadDashboard() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchHomeSnapshot() }
                .onSuccess { snapshot ->
                    val d = snapshot.dashboard
                    val hasValidAccount = d.activeCount > 0
                    val hasActiveProjects = d.joinedCount > 0
                    val canCheckin = hasValidAccount || hasActiveProjects

                    if (!canCheckin) {
                        authHintView?.visibility = View.VISIBLE
                        setButtonsEnabled(false)
                    } else {
                        authHintView?.visibility = View.GONE
                        setButtonsEnabled(true)
                    }

                    val statusParts = mutableListOf<String>()
                    if (d.checkedInToday) statusParts.add("✅ 今日已打卡（连续${d.continuousDays}天）")
                    if (d.prayedToday) statusParts.add("✅ 今日已祈福")
                    if (statusParts.isNotEmpty()) {
                        statusView?.text = statusParts.joinToString("  |  ")
                        statusView?.visibility = View.VISIBLE
                    } else {
                        statusView?.visibility = View.GONE
                    }
                }
                .onFailure {
                    authHintView?.visibility = View.GONE
                    statusView?.visibility = View.GONE
                }
        }
    }

    private fun setButtonsEnabled(enabled: Boolean) {
        checkinBtn?.let { btn ->
            btn.isEnabled = enabled
            btn.alpha = if (enabled) 1f else 0.5f
        }
        prayBtn?.let { btn ->
            btn.isEnabled = enabled
            btn.alpha = if (enabled) 1f else 0.5f
        }
    }

    private fun checkin() {
        checkinBtn?.isEnabled = false
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.checkin() }
                .onSuccess {
                    toast(it)
                    loadDashboard()
                }
                .onFailure { handlePortalError(it) }
            checkinBtn?.isEnabled = true
        }
    }

    private fun pray() {
        prayBtn?.isEnabled = false
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.pray() }
                .onSuccess {
                    toast(it)
                    loadDashboard()
                }
                .onFailure { handlePortalError(it) }
            prayBtn?.isEnabled = true
        }
    }
}
