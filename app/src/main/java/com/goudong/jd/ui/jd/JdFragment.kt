package com.goudong.jd.ui.jd

import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.LayoutInflater
import android.view.MotionEvent
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.ScrollView
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.AppEnvironment
import com.goudong.jd.ui.common.ResultTextActivity
import com.goudong.jd.ui.common.actionGridTile
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.heroCard
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.openExternalUrl
import com.goudong.jd.ui.common.primaryButton
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.launch

class JdFragment : Fragment() {
    private lateinit var noticeText: TextView
    private lateinit var noticeSpinner: ProgressBar
    private lateinit var refreshBtn: TextView
    private lateinit var qqInput: android.widget.EditText

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val (scroll, root) = requireContext().makeScrollContainer()

        // 公告卡片：标题居中，刷新按钮右下角暗色，公告可滚动
        root.addView(requireContext().cardView().apply {
            setPadding(context.dp(16), context.dp(14), context.dp(16), context.dp(14))
            // 标题行（居中）
            addView(TextView(context).apply {
                text = "公告"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT)
            })
            // 分割线
            addView(android.view.View(context).apply {
                setBackgroundColor(Color.parseColor("#E7EDF5"))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, context.dp(1)).apply {
                    topMargin = context.dp(10)
                    bottomMargin = context.dp(10)
                }
            })
            // 公告内容（可滚动，最大高度200dp）
            noticeText = TextView(context).apply {
                text = "公告加载中..."
                setTextColor(Color.parseColor("#475569"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setLineSpacing(0f, 1.4f)
            }
            val noticeScroll = ScrollView(context).apply {
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, context.dp(180))
                isFillViewport = false
                setOnTouchListener { v, event ->
                    when (event.actionMasked) {
                        MotionEvent.ACTION_DOWN, MotionEvent.ACTION_MOVE -> v.parent.requestDisallowInterceptTouchEvent(true)
                        MotionEvent.ACTION_UP, MotionEvent.ACTION_CANCEL -> v.parent.requestDisallowInterceptTouchEvent(false)
                    }
                    false
                }
                addView(noticeText)
            }
            addView(noticeScroll)
            // 右下角：刷新按钮（暗色提示）+ loading
            val bottomRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL or Gravity.END
                setPadding(0, context.dp(8), 0, 0)
            }
            noticeSpinner = ProgressBar(context).apply {
                visibility = View.GONE
                layoutParams = LinearLayout.LayoutParams(context.dp(16), context.dp(16)).apply { marginEnd = context.dp(6) }
            }
            bottomRow.addView(noticeSpinner)
            refreshBtn = TextView(context).apply {
                text = "刷新公告"
                setTextColor(Color.parseColor("#94A3B8"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(context.dp(8), context.dp(4), context.dp(8), context.dp(4))
                setOnClickListener { loadNotice() }
            }
            bottomRow.addView(refreshBtn)
            addView(bottomRow)
        })

        root.addView(requireContext().cardView().apply {
            addView(requireContext().captionText("账号"))
            qqInput = requireContext().inputField("请输入QQ号", number = true)
            addView(qqInput)
            addView(requireContext().primaryButton("前往京东登录") {
                val qq = qqInput.text?.toString().orEmpty().trim()
                if (qq.isBlank()) return@primaryButton alert("请输入QQ号")
                AppServices.sessionManager.saveQq(qq)
                startActivity(Intent(requireContext(), JdLoginActivity::class.java))
            })
        })

        root.addView(requireContext().heroCard("快捷操作", "查询数据与活动入口", ContextCompat.getColor(requireContext(), R.color.brand_primary), android.R.drawable.ic_menu_manage).apply {
            addView(requireContext().actionGridTile(listOf(
                Triple("查询数据", "按 QQ 查询 PIN 与账号信息") { queryAll() },
                Triple("更多羊毛", "打开活动页面") { startActivity(com.goudong.jd.ui.common.WebBrowserActivity.intent(requireContext(), AppEnvironment.MORE_WOOL_URL, "更多羊毛")) },
                Triple("加入交流群", "打开 QQ 群链接") { openExternalUrl(AppEnvironment.GROUP_URL) }
            ), ContextCompat.getColor(requireContext(), R.color.brand_primary)))
        })
        return scroll
    }

    override fun onResume() {
        super.onResume()
        qqInput.setText(AppServices.sessionManager.loadQq())
        loadNotice()
    }

    private fun loadNotice() {
        noticeSpinner.visibility = View.VISIBLE
        refreshBtn.isEnabled = false
        lifecycleScope.launch {
            val text = runCatching {
                AppServices.apiClient.requestText(absoluteUrl = AppEnvironment.NOTICE_URL)
            }.getOrElse { "公告加载失败：${com.goudong.jd.ui.common.sanitizeErrorMessage(it.message)}" }
            noticeText.text = text
            noticeSpinner.visibility = View.GONE
            refreshBtn.isEnabled = true
        }
    }

    private fun queryAll() {
        val qq = qqInput.text?.toString().orEmpty().trim()
        if (qq.isBlank()) return alert("请先输入QQ号")
        val loading = androidx.appcompat.app.AlertDialog.Builder(requireContext())
            .setTitle("查询中")
            .setMessage("正在获取 PIN 和用户信息，请稍候…")
            .setCancelable(false)
            .create()
        loading.show()
        lifecycleScope.launch {
            runCatching {
                val pins = AppServices.jdRepository.fetchPins(qq)
                if (pins.isEmpty()) error("未获取到有效PIN列表")
                pins.mapIndexed { index, pin ->
                    async {
                        val info = runCatching { AppServices.jdRepository.fetchUserInfo(pin) }.getOrElse { com.goudong.jd.ui.common.sanitizeErrorMessage(it.message) }
                        "【${index + 1}/${pins.size}】PIN: $pin\n$info"
                    }
                }.awaitAll().joinToString("\n\n")
            }.onSuccess {
                loading.dismiss()
                startActivity(ResultTextActivity.intent(requireContext(), "查询结果", it))
            }.onFailure {
                loading.dismiss()
                handlePortalError(it)
            }
        }
    }
}
