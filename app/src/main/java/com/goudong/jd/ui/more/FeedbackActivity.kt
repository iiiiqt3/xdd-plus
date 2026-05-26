package com.goudong.jd.ui.more

import android.graphics.Color
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.TextView
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.SubmitFeedbackPayload
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.primaryButton
import kotlinx.coroutines.launch

class FeedbackActivity : AppCompatActivity() {
    private var selectedType = ""
    private lateinit var typeLabel: TextView
    private lateinit var titleInput: EditText
    private lateinit var contentInput: EditText
    private lateinit var contactInput: EditText
    private var submitting = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        title = "投稿与反馈"

        val scroll = android.widget.ScrollView(this).apply {
            setBackgroundColor(ContextCompat.getColor(this@FeedbackActivity, R.color.surface_soft))
        }
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            val p = dp(18)
            setPadding(p, p, p, p)
            layoutParams = android.widget.LinearLayout.LayoutParams(
                android.widget.LinearLayout.LayoutParams.MATCH_PARENT,
                android.widget.LinearLayout.LayoutParams.WRAP_CONTENT
            )
        }
        scroll.addView(root)
        setContentView(scroll)

        root.addView(LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            background = android.graphics.drawable.GradientDrawable().apply {
                setColor(Color.parseColor("#FFF7ED"))
                cornerRadius = dp(14).toFloat()
                setStroke(dp(1), Color.parseColor("#FDBA74"))
            }
            setPadding(dp(16), dp(14), dp(16), dp(14))
            layoutParams = android.widget.LinearLayout.LayoutParams(
                android.widget.LinearLayout.LayoutParams.MATCH_PARENT,
                android.widget.LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = dp(16) }

            addView(TextView(this@FeedbackActivity).apply {
                text = "📝 反馈类型（必选）"
                setTextColor(Color.parseColor("#C2410C"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
            })

            typeLabel = TextView(this@FeedbackActivity).apply {
                text = if (selectedType.isEmpty()) "请点击选择反馈类型 ›" else "已选：$selectedType"
                textSize = 15f
                setTextColor(if (selectedType.isEmpty()) Color.parseColor("#94A3B8") else Color.parseColor("#0F172A"))
                setTypeface(typeface, if (selectedType.isEmpty()) android.graphics.Typeface.NORMAL else android.graphics.Typeface.BOLD)
                setPadding(0, dp(8), 0, 0)
                foreground = context.obtainStyledAttributes(intArrayOf(android.R.attr.selectableItemBackground)).getDrawable(0)
                setOnClickListener { showTypeSelector() }
            }
            addView(typeLabel)

            addView(TextView(this@FeedbackActivity).apply {
                text = "💡 反馈被采纳将获得积分奖励！"
                setTextColor(Color.parseColor("#D97706"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11.5f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
                setPadding(0, dp(6), 0, 0)
            })
        })

        titleInput = inputField("请输入主题（必填）")
        root.addView(titleInput)

        contentInput = inputField("请输入详细内容（必填）", multiline = true).apply {
            minHeight = dp(140)
            layoutParams = android.widget.LinearLayout.LayoutParams(
                android.widget.LinearLayout.LayoutParams.MATCH_PARENT,
                android.widget.LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = dp(12) }
        }
        root.addView(contentInput)

        root.addView(captionText("联系方式（选填，方便我们联系你）").apply {
            setPadding(0, dp(4), 0, dp(4))
        })
        contactInput = inputField("QQ或微信等联系方式").apply {
            layoutParams = android.widget.LinearLayout.LayoutParams(
                android.widget.LinearLayout.LayoutParams.MATCH_PARENT,
                android.widget.LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = dp(20) }
        }
        root.addView(contactInput)

        root.addView(primaryButton("提交反馈") {
            submitFeedback()
        }.apply {
            layoutParams = android.widget.LinearLayout.LayoutParams(
                android.widget.LinearLayout.LayoutParams.MATCH_PARENT,
                android.widget.LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { bottomMargin = dp(30) }
        })

        root.addView(LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            background = android.graphics.drawable.GradientDrawable().apply {
                setColor(ContextCompat.getColor(this@FeedbackActivity, R.color.brand_primary).let { c ->
                    android.graphics.Color.argb(15, android.graphics.Color.red(c), android.graphics.Color.green(c), android.graphics.Color.blue(c))
                })
                cornerRadius = dp(14).toFloat()
            }
            setPadding(dp(14), dp(14), dp(14), dp(14))
            addView(TextView(this@FeedbackActivity).apply {
                text = "提示"
                setTextColor(ContextCompat.getColor(this@FeedbackActivity, R.color.brand_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
            })
            addView(TextView(this@FeedbackActivity).apply {
                text = "• Bug反馈：发现App问题请选择此项\n• 活动投稿：参与活动或分享经验\n• 建议：功能改进建议或其他想法\n\n提交后管理员会在后台处理，处理结果会在消息通知中推送给你。"
                setTextColor(Color.parseColor("#475569"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11.5f)
                setLineSpacing(0f, 1.55f)
                setPadding(0, dp(6), 0, 0)
            })
        })
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    private fun showTypeSelector() {
        val types = arrayOf("bug反馈", "活动投稿", "建议")
        AlertDialog.Builder(this)
            .setTitle("选择反馈类型")
            .setItems(types) { _, which ->
                selectedType = types[which]
                typeLabel.text = "已选：$selectedType"
                typeLabel.setTextColor(Color.parseColor("#0F172A"))
                typeLabel.setTypeface(typeLabel.typeface, android.graphics.Typeface.BOLD)
            }
            .show()
    }

    private fun submitFeedback() {
        if (submitting) return
        if (selectedType.isEmpty()) {
            alert("请先选择反馈类型")
            return
        }
        val title = titleInput.text.toString().trim()
        val content = contentInput.text.toString().trim()
        val contact = contactInput.text.toString().trim()

        if (title.isEmpty()) {
            alert("请输入主题")
            titleInput.requestFocus()
            return
        }
        if (content.isEmpty()) {
            alert("请输入详细内容")
            contentInput.requestFocus()
            return
        }

        submitting = true
        lifecycleScope.launch {
            runCatching {
                AppServices.portalRepository.submitFeedback(
                    SubmitFeedbackPayload(
                        type = selectedType,
                        title = title,
                        content = content,
                        contact = contact,
                    )
                )
            }
                .onSuccess { msg ->
                    alert(msg, "提交成功") { finish() }
                }
                .onFailure { error ->
                    if (error is com.goudong.jd.data.model.ApiError && error.unauthorized) {
                        handlePortalError(error)
                    } else {
                        alert(com.goudong.jd.ui.common.sanitizeErrorMessage(error.message), "错误")
                    }
                }
            submitting = false
        }
    }
}
