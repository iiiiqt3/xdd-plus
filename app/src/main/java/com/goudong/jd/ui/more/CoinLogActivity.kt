package com.goudong.jd.ui.more

import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.MotionEvent
import android.view.View
import android.view.ViewGroup
import android.widget.HorizontalScrollView
import android.widget.LinearLayout
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.CoinLog
import com.goudong.jd.ui.common.AppTheme
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.themeColor
import kotlinx.coroutines.launch

class CoinLogActivity : AppCompatActivity() {
    private lateinit var contentRoot: LinearLayout
    private lateinit var swipeRefreshLayout: SwipeRefreshLayout
    private lateinit var filterBar: LinearLayout
    private var currentSource: String = ""
    private val filterButtons = mutableListOf<TextView>()
    private var touchStartX = 0f
    private var touchStartY = 0f

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AppTheme.applySystemBars(this)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        title = "积分变动记录"

        val wrapper = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setBackgroundColor(themeColor(R.color.surface_soft))
        }

        filterBar = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            setPadding(dp(16), dp(12), dp(16), dp(8))
            gravity = Gravity.CENTER_VERTICAL
        }
        AppTheme.coinLogFilters.forEach { (source, label) ->
            val btn = TextView(this).apply {
                text = label
                textSize = 12f
                setPadding(dp(20), dp(8), dp(20), dp(8))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply { marginEnd = dp(8) }
                setOnClickListener { selectFilter(source, this) }
            }
            filterButtons.add(btn)
            filterBar.addView(btn)
        }
        wrapper.addView(HorizontalScrollView(this).apply {
            isHorizontalScrollBarEnabled = false
            addView(filterBar)
        })

        val (scroll, root) = makeScrollContainer()
        contentRoot = root
        swipeRefreshLayout = SwipeRefreshLayout(this).apply {
            setColorSchemeColors(themeColor(R.color.brand_primary))
            setProgressBackgroundColorSchemeColor(themeColor(R.color.surface_card))
            setOnRefreshListener { loadCoinLogs() }
            addView(scroll)
        }
        wrapper.addView(swipeRefreshLayout)
        setContentView(wrapper)

        selectFilter("", filterButtons[0])
    }

    override fun dispatchTouchEvent(ev: MotionEvent): Boolean {
        when (ev.action) {
            MotionEvent.ACTION_DOWN -> {
                touchStartX = ev.x
                touchStartY = ev.y
            }
            MotionEvent.ACTION_UP -> {
                val dx = ev.x - touchStartX
                val dy = ev.y - touchStartY
                if (kotlin.math.abs(dx) > kotlin.math.abs(dy) && kotlin.math.abs(dx) > 100) {
                    val currentIdx = AppTheme.coinLogFilters.indexOfFirst { it.first == currentSource }
                    if (dx < 0 && currentIdx < AppTheme.coinLogFilters.size - 1) {
                        selectFilter(AppTheme.coinLogFilters[currentIdx + 1].first, filterButtons[currentIdx + 1])
                    } else if (dx > 0 && currentIdx > 0) {
                        selectFilter(AppTheme.coinLogFilters[currentIdx - 1].first, filterButtons[currentIdx - 1])
                    }
                }
            }
        }
        return super.dispatchTouchEvent(ev)
    }

    override fun onResume() {
        super.onResume()
        loadCoinLogs()
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    private fun selectFilter(source: String, btn: TextView) {
        currentSource = source
        filterButtons.forEach { b ->
            b.setTextColor(themeColor(R.color.text_muted))
            b.background = null
        }
        btn.setTextColor(themeColor(R.color.chip_active_text))
        btn.background = GradientDrawable().apply {
            setColor(themeColor(R.color.brand_primary))
            cornerRadius = dp(18).toFloat()
        }
        loadCoinLogs()
    }

    private fun loadCoinLogs() {
        if (!swipeRefreshLayout.isRefreshing) swipeRefreshLayout.isRefreshing = true
        contentRoot.removeAllViews()
        showLoading()

        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchCoinLogs(currentSource.ifEmpty { null }) }
                .onSuccess { logs ->
                    contentRoot.removeAllViews()
                    swipeRefreshLayout.isRefreshing = false
                    if (logs.isEmpty()) {
                        contentRoot.addView(emptyCard("暂无积分变动记录"))
                    } else {
                        logs.forEach { log -> contentRoot.addView(createLogItem(log)) }
                    }
                }
                .onFailure { e ->
                    contentRoot.removeAllViews()
                    swipeRefreshLayout.isRefreshing = false
                    contentRoot.addView(errorCard("加载失败: ${e.message}"))
                }
        }
    }

    private fun createLogItem(log: CoinLog): View {
        val card = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(16), dp(12), dp(16), dp(12))
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { setMargins(dp(16), dp(0), dp(16), dp(8)) }
            background = GradientDrawable().apply {
                setColor(themeColor(R.color.surface_card))
                cornerRadius = dp(14).toFloat()
                setStroke(dp(1), themeColor(R.color.border_default))
            }
            elevation = dp(2).toFloat()
        }

        val topRow = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        topRow.addView(TextView(this).apply {
            text = log.type ?: "其他"
            textSize = 14f
            setTextColor(themeColor(R.color.text_primary))
            setTypeface(null, Typeface.BOLD)
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
        })
        topRow.addView(TextView(this).apply {
            text = log.createdAt ?: ""
            textSize = 11f
            setTextColor(themeColor(R.color.text_hint))
        })
        card.addView(topRow)

        if (!log.detail.isNullOrEmpty()) {
            card.addView(TextView(this).apply {
                text = log.detail
                textSize = 12f
                setTextColor(themeColor(R.color.text_secondary))
                setPadding(0, dp(4), 0, 0)
            })
        }

        val bottomRow = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(8), 0, 0)
        }
        bottomRow.addView(TextView(this).apply {
            val amount = log.amount
            text = if (amount > 0) "+$amount" else "$amount"
            textSize = 16f
            setTextColor(if (amount > 0) themeColor(R.color.positive) else themeColor(R.color.negative))
            setTypeface(null, Typeface.BOLD)
        })
        bottomRow.addView(TextView(this).apply {
            text = "  余额: ${log.balanceAfter}"
            textSize = 12f
            setTextColor(themeColor(R.color.text_hint))
        })

        val tag = AppTheme.sourceTagStyle(this, log.sourceTagCls, log.source)
        val sourceText = TextView(this).apply {
            text = AppTheme.sourceLabel(log)
            textSize = 11f
            setPadding(dp(12), dp(4), dp(12), dp(4))
            setTextColor(tag.text)
            background = GradientDrawable().apply {
                setColor(tag.background)
                cornerRadius = dp(10).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.WRAP_CONTENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { marginStart = dp(8) }
        }
        bottomRow.addView(sourceText)
        card.addView(bottomRow)
        return card
    }

    private fun showLoading() {
        contentRoot.addView(TextView(this).apply {
            text = "加载中..."
            textSize = 14f
            setTextColor(themeColor(R.color.text_hint))
            gravity = Gravity.CENTER
            setPadding(0, dp(48), 0, 0)
        })
    }

    private fun emptyCard(msg: String): View {
        return TextView(this).apply {
            text = msg
            textSize = 14f
            setTextColor(themeColor(R.color.text_hint))
            gravity = Gravity.CENTER
            setPadding(0, dp(48), 0, 0)
        }
    }

    private fun errorCard(msg: String): View {
        return TextView(this).apply {
            text = msg
            textSize = 14f
            setTextColor(themeColor(R.color.negative))
            gravity = Gravity.CENTER
            setPadding(0, dp(48), 0, 0)
        }
    }
}
