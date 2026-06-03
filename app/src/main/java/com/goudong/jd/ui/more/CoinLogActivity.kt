package com.goudong.jd.ui.more

import android.graphics.Color
import android.graphics.Typeface
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
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.makeScrollContainer
import kotlinx.coroutines.launch

class CoinLogActivity : AppCompatActivity() {
    private lateinit var contentRoot: LinearLayout
    private lateinit var swipeRefreshLayout: SwipeRefreshLayout
    private lateinit var filterBar: LinearLayout
    private var currentSource: String = ""
    private val filterButtons = mutableListOf<TextView>()
    private val filters = listOf(
        "" to "全部",
        "Web端" to "Web端",
        "App端" to "App端",
        "微信" to "微信",
        "后台及其他" to "后台及其他",
    )
    private var touchStartX = 0f
    private var touchStartY = 0f

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        title = "积分变动记录"

        val wrapper = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL }

        // 筛选栏
        filterBar = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            setPadding(16.dp, 12.dp, 16.dp, 8.dp)
            gravity = Gravity.CENTER_VERTICAL
        }
        filters.forEach { (source, label) ->
            val btn = TextView(this).apply {
                text = label
                textSize = 12f
                setPadding(20.dp, 8.dp, 20.dp, 8.dp)
                val params = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply { marginEnd = 8.dp }
                layoutParams = params
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
            setColorSchemeColors(ContextCompat.getColor(this@CoinLogActivity, R.color.brand_primary))
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
                if (Math.abs(dx) > Math.abs(dy) && Math.abs(dx) > 100) {
                    val currentIdx = filters.indexOfFirst { it.first == currentSource }
                    if (dx < 0 && currentIdx < filters.size - 1) {
                        selectFilter(filters[currentIdx + 1].first, filterButtons[currentIdx + 1])
                    } else if (dx > 0 && currentIdx > 0) {
                        selectFilter(filters[currentIdx - 1].first, filterButtons[currentIdx - 1])
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
            b.setTextColor(Color.parseColor("#999999"))
            b.setBackgroundColor(Color.TRANSPARENT)
        }
        btn.setTextColor(Color.WHITE)
        btn.setBackgroundColor(Color.parseColor("#FF6B35"))
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
                        logs.forEach { log ->
                            contentRoot.addView(createLogItem(log))
                        }
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
            setPadding(16.dp, 12.dp, 16.dp, 12.dp)
            val params = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { setMargins(16.dp, 0.dp, 16.dp, 8.dp) }
            layoutParams = params
            setBackgroundColor(Color.WHITE)
            elevation = 2.dp.toFloat()
        }

        // 顶部：类型 + 时间
        val topRow = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }

        val typeText = TextView(this).apply {
            text = log.type ?: "其他"
            textSize = 14f
            setTextColor(Color.parseColor("#333333"))
            setTypeface(null, Typeface.BOLD)
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
        }
        topRow.addView(typeText)

        val timeText = TextView(this).apply {
            text = log.createdAt ?: ""
            textSize = 11f
            setTextColor(Color.parseColor("#999999"))
        }
        topRow.addView(timeText)
        card.addView(topRow)

        // 详情
        if (!log.detail.isNullOrEmpty()) {
            val detailText = TextView(this).apply {
                text = log.detail
                textSize = 12f
                setTextColor(Color.parseColor("#666666"))
                setPadding(0, 4.dp, 0, 0)
            }
            card.addView(detailText)
        }

        // 底部：金额 + 余额 + 来源
        val bottomRow = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, 8.dp, 0, 0)
        }

        val amountText = TextView(this).apply {
            val amount = log.amount
            text = if (amount > 0) "+$amount" else "$amount"
            textSize = 16f
            setTextColor(if (amount > 0) Color.parseColor("#4CAF50") else Color.parseColor("#F44336"))
            setTypeface(null, Typeface.BOLD)
        }
        bottomRow.addView(amountText)

        val balanceText = TextView(this).apply {
            text = "  余额: ${log.balanceAfter}"
            textSize = 12f
            setTextColor(Color.parseColor("#999999"))
        }
        bottomRow.addView(balanceText)

        val sourceText = TextView(this).apply {
            text = log.source ?: ""
            textSize = 11f
            setPadding(12.dp, 4.dp, 12.dp, 4.dp)
            setTextColor(Color.parseColor("#666666"))
            setBackgroundColor(Color.parseColor("#F0F0F0"))
        }
        bottomRow.addView(sourceText)

        card.addView(bottomRow)
        return card
    }

    private fun showLoading() {
        val loading = TextView(this).apply {
            text = "加载中..."
            textSize = 14f
            setTextColor(Color.parseColor("#999999"))
            gravity = Gravity.CENTER
            setPadding(0, 48.dp, 0, 0)
        }
        contentRoot.addView(loading)
    }

    private fun emptyCard(msg: String): View {
        return TextView(this).apply {
            text = msg
            textSize = 14f
            setTextColor(Color.parseColor("#999999"))
            gravity = Gravity.CENTER
            setPadding(0, 48.dp, 0, 0)
        }
    }

    private fun errorCard(msg: String): View {
        return TextView(this).apply {
            text = msg
            textSize = 14f
            setTextColor(Color.parseColor("#F44336"))
            gravity = Gravity.CENTER
            setPadding(0, 48.dp, 0, 0)
        }
    }

    private val Int.dp: Int
        get() = TypedValue.applyDimension(
            TypedValue.COMPLEX_UNIT_DIP, this.toFloat(), resources.displayMetrics
        ).toInt()
}
