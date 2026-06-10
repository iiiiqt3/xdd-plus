package com.goudong.jd.ui.projects

import android.content.Context
import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.ViewGroup
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.AppEnvironment
import com.goudong.jd.data.model.PortalActivity
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.applySafeStatusBar
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.primaryButton
import com.goudong.jd.ui.common.sectionTitle
import kotlinx.coroutines.launch

class ProjectFormActivity : AppCompatActivity() {
    private lateinit var activityItem: PortalActivity

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        @Suppress("DEPRECATION")
        activityItem = intent.getSerializableExtra(EXTRA_ACTIVITY) as? PortalActivity
            ?: return finish()
        title = activityItem.name ?: "活动详情"
        supportActionBar?.setDisplayHomeAsUpEnabled(true)

        val scroll = ScrollView(this).apply {
            setBackgroundColor(Color.parseColor("#F4F7FB"))
        }
        applySafeStatusBar(scroll)
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(16), dp(16), dp(16), dp(24))
            layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT)
        }

        val inputs = LinkedHashMap<String, android.widget.EditText>()

        // 项目信息卡（松弛排版）
        root.addView(cardView().apply {
            setPadding(dp(18), dp(18), dp(18), dp(18))
            // 项目名
            addView(TextView(context).apply {
                text = activityItem.name ?: "未命名项目"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 18f)
                setTypeface(typeface, Typeface.BOLD)
            })
            // 描述间隔
            val desc = when {
                activityItem.isDailyDeduct == true -> "每天扣 ${activityItem.dailyCoin ?: 0} 积分 · 青龙：${activityItem.qingLongConfig ?: "默认容器"}"
                activityItem.isMonthlyDeduct == true -> "每月扣 ${activityItem.monthlyCoin ?: 0} 积分 · 青龙：${activityItem.qingLongConfig ?: "默认容器"}"
                else -> "一次扣 ${activityItem.needCoin ?: 0} 积分 · 青龙：${activityItem.qingLongConfig ?: "默认容器"}"
            }
            addView(TextView(context).apply {
                text = desc
                setTextColor(ContextCompat.getColor(context, R.color.brand_secondary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setPadding(0, dp(6), 0, dp(20))
            })
            // 分割线
            addView(android.view.View(context).apply {
                setBackgroundColor(Color.parseColor("#E7EDF5"))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(1))
            })
            // 填写说明标题
            addView(TextView(context).apply {
                text = "填写说明"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
                setPadding(0, dp(18), 0, dp(8))
            })
            activityItem.guide?.trim()?.takeIf { it.isNotEmpty() }?.let { guideText ->
                addView(LinearLayout(context).apply {
                    orientation = LinearLayout.VERTICAL
                    background = android.graphics.drawable.GradientDrawable().apply {
                        setColor(Color.parseColor("#F8FAFC"))
                        cornerRadius = dp(14).toFloat()
                        setStroke(dp(1), Color.parseColor("#E2E8F0"))
                    }
                    setPadding(dp(14), dp(12), dp(14), dp(12))
                    layoutParams = LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.MATCH_PARENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT
                    ).apply {
                        bottomMargin = dp(6)
                    }
                    addView(TextView(context).apply {
                        text = "玩法和说明"
                        setTextColor(Color.parseColor("#0F172A"))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                        setTypeface(typeface, Typeface.BOLD)
                    })
                    addView(WebView(context).apply {
                        layoutParams = LinearLayout.LayoutParams(
                            LinearLayout.LayoutParams.MATCH_PARENT,
                            LinearLayout.LayoutParams.WRAP_CONTENT
                        ).apply { topMargin = dp(8) }
                        val html = markdownToHtml(guideText)
                        loadDataWithBaseURL(AppEnvironment.BASE_URL, html, "text/html", "UTF-8", null)
                        webViewClient = object : WebViewClient() {
                            override fun shouldOverrideUrlLoading(view: WebView?, request: android.webkit.WebResourceRequest?): Boolean {
                                val uri = request?.url?.toString() ?: return false
                                if (uri.startsWith("openvideo:")) {
                                    val videoUrl = uri.removePrefix("openvideo:")
                                    try {
                                        val intent = Intent(Intent.ACTION_VIEW)
                                        intent.setDataAndType(android.net.Uri.parse(videoUrl), "video/*")
                                        startActivity(intent)
                                    } catch (_: Exception) {}
                                    return true
                                }
                                try { startActivity(Intent(Intent.ACTION_VIEW, android.net.Uri.parse(uri))) } catch (_: Exception) {}
                                return true
                            }
                        }
                        webChromeClient = android.webkit.WebChromeClient()
                        settings.javaScriptEnabled = true
                        settings.mediaPlaybackRequiresUserGesture = false
                        settings.loadsImagesAutomatically = true
                        settings.builtInZoomControls = false
                        settings.displayZoomControls = false
                        settings.domStorageEnabled = true
                        isVerticalScrollBarEnabled = false
                        setLayerType(android.view.View.LAYER_TYPE_HARDWARE, null)
                        setBackgroundColor(Color.TRANSPARENT)
                    })
                })
            }
            // 动态字段
            activityItem.inputFields.orEmpty().forEach { field ->
                addView(TextView(context).apply {
                    text = field.prompt ?: field.key ?: "输入项"
                    setTextColor(Color.parseColor("#475569"))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
                    setPadding(0, dp(14), 0, dp(6))
                })
                val input = inputField(field.prompt ?: field.key ?: "请输入")
                inputs[field.key ?: ""] = input
                addView(input)
            }
            // 备注名
            addView(TextView(context).apply {
                text = "用户备注名"
                setTextColor(Color.parseColor("#475569"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
                setPadding(0, dp(14), 0, dp(6))
            })
            val remark = inputField("请输入唯一备注名")
            addView(remark)
            // 授权时长输入框
            val monthInput = when {
                activityItem.isDailyDeduct == true -> {
                    val minDays = activityItem.minDays ?: 1
                    addView(TextView(context).apply {
                        text = "授权天数（最少${minDays}天）"
                        setTextColor(Color.parseColor("#475569"))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
                        setPadding(0, dp(14), 0, dp(6))
                    })
                    inputField("最少${minDays}天，如 ${minDays}/7/30", number = true).also { addView(it) }
                }
                activityItem.isMonthlyDeduct == true -> {
                    addView(TextView(context).apply {
                        text = "授权月数"
                        setTextColor(Color.parseColor("#475569"))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
                        setPadding(0, dp(14), 0, dp(6))
                    })
                    inputField("请输入 1-12", number = true).also { addView(it) }
                }
                else -> null
            }
            // 间隔
            addView(android.view.View(context).apply {
                layoutParams = LinearLayout.LayoutParams(0, dp(16))
            })
            // 提交按钮
            val submit = primaryButton("确认上车")
            submit.setOnClickListener {
                val remarks = remark.text?.toString().orEmpty().trim()
                if (remarks.isBlank()) return@setOnClickListener alert("请输入备注名")
                val map = inputs.mapValues { it.value.text?.toString().orEmpty().trim() }
                val months = monthInput?.text?.toString()?.toIntOrNull() ?: 0

                // 验证输入
                if (activityItem.isDailyDeduct == true) {
                    val minDays = activityItem.minDays ?: 1
                    if (months < minDays) return@setOnClickListener alert("授权天数最少${minDays}天")
                    if (months > 365) return@setOnClickListener alert("授权天数不能超过365天")
                } else if (activityItem.isMonthlyDeduct == true) {
                    if (months <= 0) return@setOnClickListener alert("请输入正确的授权月数")
                }

                val totalCoin = when {
                    activityItem.isDailyDeduct == true -> (activityItem.dailyCoin ?: 0) * months
                    activityItem.isMonthlyDeduct == true -> (activityItem.monthlyCoin ?: 0) * months
                    else -> activityItem.needCoin ?: 0
                }
                val expireText = if ((activityItem.isDailyDeduct == true || activityItem.isMonthlyDeduct == true) && months > 0) {
                    val cal = java.util.Calendar.getInstance()
                    if (activityItem.isDailyDeduct == true) {
                        cal.add(java.util.Calendar.DAY_OF_MONTH, months)
                    } else {
                        cal.add(java.util.Calendar.MONTH, months)
                    }
                    val y = cal.get(java.util.Calendar.YEAR)
                    val m = cal.get(java.util.Calendar.MONTH) + 1
                    val d = cal.get(java.util.Calendar.DAY_OF_MONTH)
                    String.format("%04d-%02d-%02d", y, m, d)
                } else null
                val confirmMessage = buildString {
                    append("项目：${activityItem.name ?: "未命名"}\n")
                    append("备注名：$remarks\n")
                    append("将扣积分：$totalCoin\n")
                    when {
                        activityItem.isDailyDeduct == true -> {
                            append("授权天数：$months\n")
                            if (!expireText.isNullOrBlank()) append("预计有效期至：$expireText\n")
                        }
                        activityItem.isMonthlyDeduct == true -> {
                            append("授权月数：$months\n")
                            if (!expireText.isNullOrBlank()) append("预计有效期至：$expireText\n")
                        }
                        else -> {
                            append("生效方式：一次性上车\n")
                        }
                    }
                    append("确认后才会正式上车并扣除积分。")
                }
                androidx.appcompat.app.AlertDialog.Builder(this@ProjectFormActivity)
                    .setTitle("确认上车")
                    .setMessage(confirmMessage)
                    .setNegativeButton("取消", null)
                    .setPositiveButton("确认上车") { _, _ ->
                        lifecycleScope.launch {
                            runCatching { AppServices.portalRepository.createProject(activityItem.id ?: "", map, remarks, months) }
                                .onSuccess { alert(it) { finish() } }
                                .onFailure { alert(com.goudong.jd.ui.common.sanitizeErrorMessage(it.message)) }
                        }
                    }
                    .show()
            }
            addView(submit)
        })
        scroll.addView(root)
        setContentView(scroll)
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    companion object {
        private const val EXTRA_ACTIVITY = "activity"

        fun intent(context: Context, activity: PortalActivity): Intent {
            return Intent(context, ProjectFormActivity::class.java).putExtra(EXTRA_ACTIVITY, activity)
        }
    }
}

/** 简易Markdown转HTML，支持标题、粗体、斜体、列表、引用、代码、图片、链接 */
fun markdownToHtml(md: String): String {
    var h = md
        .replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
    // 代码块
    h = Regex("```(\\w*)\\n([\\s\\S]*?)```").replace(h) { "<pre><code>${it.groupValues[2]}</code></pre>" }
    h = Regex("`([^`]+)`").replace(h) { "<code>${it.groupValues[1]}</code>" }
    // 图片/视频/音频
    h = Regex("!\\[([^\\]]*)]\\(([^)]+)\\)").replace(h) { m ->
        val url = m.groupValues[2]
        val ext = url.split('.').lastOrNull()?.split('?')?.firstOrNull()?.lowercase() ?: ""
        when {
            ext in listOf("mp4", "webm", "mov", "avi") ->
                "<div onclick=\"window.location='openvideo:$url'\" style=\"position:relative;cursor:pointer;text-align:center;margin:6px 0;border-radius:8px;overflow:hidden;background:#1e293b;\"><div style=\"width:80px;height:80px;margin:20px auto;background:rgba(255,255,255,0.15);border-radius:50%;display:flex;align-items:center;justify-content:center;\"><div style=\"width:0;height:0;border-left:30px solid white;border-top:18px solid transparent;border-bottom:18px solid transparent;margin-left:8px;\"></div></div><div style=\"padding:8px;color:white;font-size:12px;\">点击播放视频</div></div>"
            ext in listOf("mp3", "wav", "ogg", "m4a", "aac", "flac") ->
                "<audio src=\"$url\" controls preload=\"metadata\" style=\"width:100%;margin:6px 0;\"></audio>"
            else ->
                "<img src=\"$url\" alt=\"${m.groupValues[1]}\" style=\"max-width:100%;max-height:300px;object-fit:contain;border-radius:8px;margin:6px 0;\">"
        }
    }
    // 链接
    h = Regex("\\[([^\\]]+)]\\(([^)]+)\\)").replace(h) {
        "<a href=\"${it.groupValues[2]}\">${it.groupValues[1]}</a>"
    }
    // 标题
    h = Regex("^### (.+)$", RegexOption.MULTILINE).replace(h) { "<h3>${it.groupValues[1]}</h3>" }
    h = Regex("^## (.+)$", RegexOption.MULTILINE).replace(h) { "<h2>${it.groupValues[1]}</h2>" }
    h = Regex("^# (.+)$", RegexOption.MULTILINE).replace(h) { "<h1>${it.groupValues[1]}</h1>" }
    // 粗体、斜体、删除线
    h = Regex("\\*\\*(.+?)\\*\\*").replace(h) { "<strong>${it.groupValues[1]}</strong>" }
    h = Regex("\\*(.+?)\\*").replace(h) { "<em>${it.groupValues[1]}</em>" }
    h = Regex("~~(.+?)~~").replace(h) { "<del>${it.groupValues[1]}</del>" }
    // 引用
    h = Regex("^&gt; (.+)$", RegexOption.MULTILINE).replace(h) {
        "<blockquote style=\"border-left:3px solid #6366f1;padding-left:12px;color:#64748b;margin:8px 0;\">${it.groupValues[1]}</blockquote>"
    }
    // 分割线
    h = Regex("^---$", RegexOption.MULTILINE).replace(h) { "<hr style=\"border:none;border-top:1px solid #e2e8f0;margin:12px 0;\">" }
    // 列表
    h = Regex("^- (.+)$", RegexOption.MULTILINE).replace(h) { "<li>${it.groupValues[1]}</li>" }
    h = Regex("(<li>.*</li>\\n?)+").replace(h) { "<ul style=\"padding-left:20px;margin:6px 0;\">${it.value}</ul>" }
    h = Regex("^\\d+\\. (.+)$", RegexOption.MULTILINE).replace(h) { "<li>${it.groupValues[1]}</li>" }
    // 换行
    h = h.replace("\n\n", "</p><p>").replace("\n", "<br>")
    h = "<p>$h</p>"
    // 清理
    h = h.replace(Regex("<p>\\s*</p>"), "")
    return """<!DOCTYPE html><html><head><meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1">
<style>body{font-family:-apple-system,sans-serif;font-size:13px;color:#475569;line-height:1.7;margin:0;padding:0;}
img,video{max-width:100%;max-height:300px;object-fit:contain;border-radius:8px;margin:6px 0;}
audio{width:100%;margin:6px 0;}
pre{background:#1e293b;color:#e2e8f0;padding:12px;border-radius:6px;overflow-x:auto;}
code{background:#f1f5f9;padding:2px 5px;border-radius:3px;font-size:12px;}pre code{background:none;color:inherit;padding:0;}
a{color:#6366f1;}h1{font-size:17px;font-weight:700;margin:12px 0 6px;}h2{font-size:15px;font-weight:700;margin:10px 0 6px;}
h3{font-size:14px;font-weight:700;margin:8px 0 4px;}ul{padding-left:20px;}</style></head><body>$h</body></html>"""
}
