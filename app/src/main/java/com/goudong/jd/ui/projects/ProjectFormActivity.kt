package com.goudong.jd.ui.projects

import android.content.Context
import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
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
                    addView(TextView(context).apply {
                        text = guideText
                        setTextColor(Color.parseColor("#475569"))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
                        setLineSpacing(0f, 1.35f)
                        setPadding(0, dp(8), 0, 0)
                        setTextIsSelectable(true)
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
