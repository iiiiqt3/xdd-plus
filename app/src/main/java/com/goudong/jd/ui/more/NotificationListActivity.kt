package com.goudong.jd.ui.more

import android.graphics.Color
import android.graphics.Typeface
import android.os.Bundle
import android.content.Intent
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.ApiError
import com.goudong.jd.data.model.PortalNotification
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.makeScrollContainer
import kotlinx.coroutines.launch
import com.goudong.jd.ui.common.themeColor
import com.goudong.jd.ui.common.AppTheme

class NotificationListActivity : AppCompatActivity() {
    private lateinit var contentRoot: LinearLayout
    private lateinit var swipeRefreshLayout: SwipeRefreshLayout

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AppTheme.applySystemBars(this)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)

        val wrapper = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL }
        val (scroll, root) = makeScrollContainer()
        contentRoot = root
        swipeRefreshLayout = SwipeRefreshLayout(this).apply {
            setColorSchemeColors(ContextCompat.getColor(this@NotificationListActivity, R.color.brand_primary))
            setOnRefreshListener { loadNotifications(forceRefresh = true) }
            addView(scroll)
        }
        wrapper.addView(swipeRefreshLayout)
        setContentView(wrapper)
    }

    override fun onResume() {
        super.onResume()
        loadNotifications(forceRefresh = true)
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    private fun loadNotifications(forceRefresh: Boolean = false) {
        android.util.Log.d("NotificationList", "=== 开始加载通知列表 (forceRefresh=$forceRefresh) ===")
        
        if (!swipeRefreshLayout.isRefreshing && forceRefresh) swipeRefreshLayout.isRefreshing = true
        contentRoot.removeAllViews()
        showLoading()
        
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchNotifications(includeContent = true) }
                .onSuccess { page ->
                    contentRoot.removeAllViews()
                    title = if (page.unread > 0) "消息通知 ($page.unread条未读)" else "消息通知"
                    
                    android.util.Log.d("NotificationList", "✅ 通知列表加载成功")
                    android.util.Log.d("NotificationList", "总数量: ${page.list.size}, 未读数: ${page.unread}")
                    
                    if (page.list.isEmpty()) {
                        contentRoot.addView(emptyCard("暂无通知"))
                    } else {
                        page.list.forEachIndexed { index, item ->
                            android.util.Log.d("NotificationList", "通知[$index]: id=${item.id}, title=${item.title}, isRead=${item.isRead}")
                            contentRoot.addView(notificationItem(item))
                        }
                    }
                    
                    swipeRefreshLayout.isRefreshing = false
                }
                .onFailure { error ->
                    contentRoot.removeAllViews()
                    android.util.Log.e("NotificationList", "❌ 通知列表加载失败: ${error.message}", error)
                    
                    when (error) {
                        is ApiError -> {
                            if (error.unauthorized) handlePortalError(error)
                            else {
                                contentRoot.addView(emptyCard(com.goudong.jd.ui.common.sanitizeErrorMessage(error.message)))
                            }
                        }
                        else -> {
                            contentRoot.addView(emptyCard(com.goudong.jd.ui.common.sanitizeErrorMessage(error.message)))
                        }
                    }
                    swipeRefreshLayout.isRefreshing = false
                }
        }
    }

    private fun showLoading() {
        contentRoot.addView(LinearLayout(this).apply {
            gravity = Gravity.CENTER
            orientation = LinearLayout.VERTICAL
            setPadding(0, dp(48), 0, 0)
            addView(android.widget.ProgressBar(this@NotificationListActivity).apply {
                layoutParams = LinearLayout.LayoutParams(dp(32), dp(32))
            })
            addView(TextView(this@NotificationListActivity).apply {
                text = "加载中..."
                setTextColor(themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                gravity = Gravity.CENTER
                setPadding(0, dp(10), 0, 0)
            })
        })
    }

    private fun notificationItem(item: PortalNotification): View {
        return cardView().apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(16), dp(14), dp(16), dp(14))
            descendantFocusability = ViewGroup.FOCUS_BLOCK_DESCENDANTS

            if (!item.isRead) {
                background = android.graphics.drawable.GradientDrawable().apply {
                    setColor(Color.parseColor("#EFF6FF"))
                    cornerRadius = dp(12).toFloat()
                    setStroke(dp(1), Color.parseColor("#3B82F6"))
                }
            }

            val topRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
            }

            if (!item.isRead) {
                topRow.addView(View(context).apply {
                    layoutParams = LinearLayout.LayoutParams(dp(10), dp(10)).apply {
                        marginEnd = dp(8)
                        topMargin = dp(2)
                    }
                    background = android.graphics.drawable.GradientDrawable().apply {
                        shape = android.graphics.drawable.GradientDrawable.OVAL
                        setColor(themeColor(R.color.brand_red))
                    }
                })
            }

            val titleText = TextView(context).apply {
                text = item.title ?: "无标题"
                setTextColor(themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, if (item.isTop == true) Typeface.BOLD else Typeface.NORMAL)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f)
            }
            topRow.addView(titleText)

            if (item.isTop == true) {
                topRow.addView(TextView(context).apply {
                    text = "置顶"
                    setTextColor(themeColor(R.color.chip_active_text))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f)
                    setTypeface(typeface, Typeface.BOLD)
                    background = android.graphics.drawable.GradientDrawable().apply {
                        setColor(themeColor(R.color.brand_red))
                        cornerRadius = dp(4).toFloat()
                    }
                    setPadding(dp(6), dp(1), dp(6), dp(1))
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        marginStart = dp(8)
                    }
                })
            }

            if (!item.isRead) {
                topRow.addView(TextView(context).apply {
                    text = "未读"
                    setTextColor(Color.parseColor("#409EFF"))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 10f)
                    setTypeface(typeface, Typeface.BOLD)
                    background = android.graphics.drawable.GradientDrawable().apply {
                        setColor(Color.parseColor("#E6F7FF"))
                        cornerRadius = dp(4).toFloat()
                    }
                    setPadding(dp(6), dp(1), dp(6), dp(1))
                    layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                        marginStart = dp(6)
                    }
                })
            }

            addView(topRow)

            item.category?.let {
                addView(captionText("分类：$it").apply { setPadding(0, dp(4), 0, 0) })
            }

            item.content?.takeIf { it.isNotBlank() }?.let { content ->
                val preview = if (content.length > 120) content.take(120) + "..." else content
                addView(TextView(context).apply {
                    text = preview
                    setTextColor(themeColor(R.color.text_secondary))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.2f)
                    setLineSpacing(0f, 1.28f)
                    setPadding(0, dp(6), 0, 0)
                    maxLines = 3
                })
            }

            addView(captionText("${item.createdAt ?: "-"}").apply {
                setPadding(0, dp(8), 0, 0)
                gravity = Gravity.END
            })

            setOnClickListener {
                val intent = Intent(this@NotificationListActivity, NotificationDetailActivity::class.java)
                intent.putExtra(NotificationDetailActivity.EXTRA_NOTIFICATION, item)
                startActivity(intent)
                
                android.util.Log.d("Notification", "点击通知，跳转到详情页: id=${item.id}")
            }
            foreground = context.obtainStyledAttributes(intArrayOf(android.R.attr.selectableItemBackground)).getDrawable(0)
        }
    }

    private fun emptyCard(message: String): View {
        return cardView().apply {
            gravity = Gravity.CENTER
            setPadding(0, dp(48), 0, dp(48))
            addView(bodyText(message).apply {
                gravity = Gravity.CENTER
                setTextColor(themeColor(R.color.text_hint))
            })
        }
    }
}
