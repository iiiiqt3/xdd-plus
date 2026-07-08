package com.goudong.jd.ui.more

import android.graphics.Color
import android.graphics.Typeface
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.widget.LinearLayout
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.PortalNotification
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.makeScrollContainer
import kotlinx.coroutines.launch
import com.goudong.jd.ui.common.themeColor
import com.goudong.jd.ui.common.AppTheme

class NotificationDetailActivity : AppCompatActivity() {
    
    companion object {
        const val EXTRA_NOTIFICATION = "notification"
    }
    
    private lateinit var notification: PortalNotification
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        AppTheme.applySystemBars(this)
        notification = intent.getSerializableExtra(EXTRA_NOTIFICATION) as? PortalNotification 
            ?: run { finish(); return }
        
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        title = "通知详情"
        
        val wrapper = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL }
        val (scroll, root) = makeScrollContainer()
        root.addView(createDetailView())
        wrapper.addView(scroll)
        setContentView(wrapper)
        
        markAsRead()
    }
    
    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }
    
    override fun onResume() {
        super.onResume()
        markAsRead()
    }
    
    private fun createDetailView(): View {
        val ctx = this
        
        return cardView().apply {
            setPadding(dp(20), dp(24), dp(20), dp(24))
            
            addView(TextView(ctx).apply {
                text = notification.title ?: "无标题"
                setTextColor(themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 18f)
                setTypeface(typeface, Typeface.BOLD)
                setPadding(0, 0, 0, dp(8))
            })
            
            if (notification.isTop == true) {
                addView(LinearLayout(ctx).apply {
                    orientation = LinearLayout.HORIZONTAL
                    gravity = Gravity.CENTER_VERTICAL
                    setPadding(0, 0, 0, dp(12))
                    
                    addView(TextView(ctx).apply {
                        text = "📌 置顶通知"
                        setTextColor(themeColor(R.color.brand_red))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                        setTypeface(typeface, Typeface.BOLD)
                        background = android.graphics.drawable.GradientDrawable().apply {
                            setColor(Color.parseColor("#FEF2F2"))
                            cornerRadius = dp(4).toFloat()
                        }
                        setPadding(dp(8), dp(3), dp(8), dp(3))
                    })
                })
            }
            
            if (!notification.category.isNullOrBlank()) {
                addView(captionText("分类：${notification.category}").apply {
                    setTextColor(themeColor(R.color.text_muted))
                    setPadding(0, 0, 0, dp(16))
                })
            }
            
            addView(View(ctx).apply {
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    dp(1)
                ).apply {
                    topMargin = dp(12)
                    bottomMargin = dp(16)
                }
                setBackgroundColor(themeColor(R.color.border_light))
            })
            
            addView(bodyText(notification.content ?: "暂无内容").apply {
                setTextColor(themeColor(R.color.text_secondary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14.5f)
                setLineSpacing(0f, 1.7f)
                setTextIsSelectable(true)
                setPadding(0, 0, 0, dp(24))
            })
            
            addView(captionText("发布时间：${notification.createdAt ?: "-"}").apply {
                setTextColor(themeColor(R.color.text_hint))
                gravity = Gravity.END
            })
        }
    }
    
    private fun markAsRead() {
        lifecycleScope.launch {
            runCatching {
                android.util.Log.d("NotificationDetail", "=== 开始标记已读 ===")
                android.util.Log.d("NotificationDetail", "通知ID: ${notification.id}")
                android.util.Log.d("NotificationDetail", "调用API: /api/portal/notification?id=${notification.id}")
                
                val result = AppServices.portalRepository.markNotificationRead(notification.id)
                
                android.util.Log.d("NotificationDetail", "✅ API调用成功")
                android.util.Log.d("NotificationDetail", "返回的isRead状态: ${result.isRead}")
                android.util.Log.d("NotificationDetail", "返回的readAt时间: ${result.readAt}")
                
                result
            }
                .onSuccess { result ->
                    android.util.Log.d("NotificationDetail", "=== 标记已读完成 ===")
                    
                    runCatching {
                        val response = AppServices.portalRepository.fetchNotifications(includeContent = false)
                        android.util.Log.d("NotificationDetail", "📊 刷新未读数成功: ${response.unread}条未读")
                    }
                }
                .onFailure { error ->
                    android.util.Log.e("NotificationDetail", "❌ 已读标记失败!!!")
                    android.util.Log.e("NotificationDetail", "错误类型: ${error.javaClass.simpleName}")
                    android.util.Log.e("NotificationDetail", "错误信息: ${error.message}", error)
                    
                    if (error is com.goudong.jd.data.model.ApiError) {
                        android.util.Log.e("NotificationDetail", "是否未授权: ${error.unauthorized}")
                    }
                }
        }
    }
}
