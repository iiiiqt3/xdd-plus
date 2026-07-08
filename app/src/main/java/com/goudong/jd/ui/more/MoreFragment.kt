package com.goudong.jd.ui.more

import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.TextView
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.R
import com.goudong.jd.AppServices
import com.goudong.jd.ui.common.actionTile
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.heroCard
import com.goudong.jd.ui.common.MainTabResettable
import com.goudong.jd.ui.common.findFirstScrollView
import com.goudong.jd.ui.common.wrapMainTabSwipe
import com.goudong.jd.ui.common.makeScrollContainer
import kotlinx.coroutines.launch
import com.goudong.jd.ui.common.themeColor

class MoreFragment : Fragment(), MainTabResettable {
    private var notificationBadge: TextView? = null
    
    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val (scroll, root) = requireContext().makeScrollContainer()

        // 移除顶部卡片栏

        root.addView(requireContext().cardView().apply {
            addView(requireContext().captionText("常用功能"))
            
            val notificationTile = createNotificationTile()
            addView(notificationTile)
            
            addView(requireContext().actionTile(
                title = "投稿与反馈",
                desc = "提交Bug、活动投稿或建议",
                tint = requireContext().getColor(R.color.brand_secondary),
            ) {
                startActivity(Intent(requireContext(), FeedbackActivity::class.java))
            })
        })

        root.addView(requireContext().cardView().apply {
            addView(requireContext().captionText("关于"))
            addView(requireContext().actionTile(
                title = "关于版本",
                desc = "当前版本号、SDK信息与更新内容",
                tint = requireContext().getColor(R.color.brand_green),
            ) {
                startActivity(Intent(requireContext(), AboutVersionActivity::class.java))
            })
            addView(requireContext().actionTile(
                title = "关于作者",
                desc = "大师 · QQ: 694738267",
                tint = requireContext().getColor(R.color.brand_orange),
            ) {
                val intent = Intent(Intent.ACTION_VIEW, android.net.Uri.parse("mqqwpa://im/chat?chat_type=wpa&uin=694738267"))
                try { startActivity(intent) } catch (_: Exception) {}
            })
        })

        loadUnreadCount()

        return wrapMainTabSwipe(scroll)
    }
    
    override fun onResume() {
        super.onResume()
        loadUnreadCount()
    }
    
    private fun createNotificationTile(): View {
        return android.widget.FrameLayout(requireContext()).apply {
            
            val tileContent = requireContext().actionTile(
                title = "消息通知",
                desc = "查看管理员推送的所有公告与通知",
                tint = requireContext().getColor(R.color.brand_primary),
            ) {
                startActivity(Intent(requireContext(), NotificationListActivity::class.java))
            }
            
            addView(tileContent)
            
            notificationBadge = TextView(requireContext()).apply {
                text = ""
                setTextColor(requireContext().themeColor(R.color.chip_active_text))
                setTextSize(android.util.TypedValue.COMPLEX_UNIT_SP, 11f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
                gravity = android.view.Gravity.CENTER
                
                val badgeSize = requireContext().dp(20)
                
                background = android.graphics.drawable.GradientDrawable().apply {
                    shape = android.graphics.drawable.GradientDrawable.RECTANGLE
                    setColor(requireContext().themeColor(R.color.brand_red))
                    cornerRadius = (badgeSize / 2).toFloat()
                }
                
                setPadding(requireContext().dp(6), requireContext().dp(2), requireContext().dp(6), requireContext().dp(2))
                
                layoutParams = android.widget.FrameLayout.LayoutParams(
                    android.widget.FrameLayout.LayoutParams.WRAP_CONTENT,
                    badgeSize,
                    android.view.Gravity.TOP or android.view.Gravity.END
                ).apply {
                    topMargin = requireContext().dp(12)
                    marginEnd = requireContext().dp(12)
                }
                
                visibility = View.GONE
            }
            
            addView(notificationBadge)
        }
    }
    
    private fun loadUnreadCount() {
        if (!AppServices.sessionManager.isAuthenticated()) return
        
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchNotifications(includeContent = false) }
                .onSuccess { response ->
                    updateNotificationBadge(response.unread)
                    
                    (activity as? com.goudong.jd.MainActivity)?.updateMoreBadge(response.unread)
                }
        }
    }
    
    private fun updateNotificationBadge(unreadCount: Int) {
        notificationBadge?.let { badge ->
            if (unreadCount > 0) {
                badge.text = if (unreadCount > 99) "99+" else unreadCount.toString()
                badge.visibility = View.VISIBLE
            } else {
                badge.visibility = View.GONE
            }
        }
    }

    override fun resetToInitialState() {
        view?.findFirstScrollView()?.scrollTo(0, 0)
        loadUnreadCount()
    }
}