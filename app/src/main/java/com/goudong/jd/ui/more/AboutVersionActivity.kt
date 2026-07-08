package com.goudong.jd.ui.more

import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.BuildConfig
import com.goudong.jd.R
import com.goudong.jd.ui.common.AppTheme
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.themeColor
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.sanitizeErrorMessage
import com.goudong.jd.update.UpdateChecker
import com.goudong.jd.update.UpdateInfo
import kotlinx.coroutines.launch

class AboutVersionActivity : AppCompatActivity() {
    private lateinit var changelogText: TextView
    private lateinit var changelogLoading: ProgressBar

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AppTheme.applySystemBars(this)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        title = "关于版本"

        val scroll = android.widget.ScrollView(this).apply {
            setBackgroundColor(ContextCompat.getColor(this@AboutVersionActivity, R.color.surface_soft))
        }
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            val p = dp(18)
            setPadding(p, p, p, p)
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            )
        }
        scroll.addView(root)
        setContentView(scroll)

        val iconWrap = LinearLayout(this).apply {
            gravity = Gravity.CENTER_HORIZONTAL
            orientation = LinearLayout.VERTICAL
            setPadding(0, dp(20), 0, dp(16))
        }
        iconWrap.addView(TextView(this).apply {
            text = "狗东"
            setTextColor(ContextCompat.getColor(this@AboutVersionActivity, R.color.brand_primary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 32f)
            setTypeface(typeface, android.graphics.Typeface.BOLD)
        })
        iconWrap.addView(captionText("GouDong v${BuildConfig.VERSION_NAME}").apply {
            setPadding(0, dp(6), 0, 0)
            textSize = 14f
        })
        root.addView(iconWrap)

        root.addView(cardView().apply {
            addView(infoRow("当前版本", BuildConfig.VERSION_NAME))
            addView(infoRow("版本号", "${BuildConfig.VERSION_CODE}"))
            addView(infoRow("SDK版本", "${android.os.Build.VERSION.SDK_INT} (${android.os.Build.VERSION.RELEASE})"))
            addView(infoRow("系统", "${android.os.Build.MANUFACTURER} ${android.os.Build.MODEL}"))
        })

        root.addView(cardView().apply {
            addView(TextView(this@AboutVersionActivity).apply {
                text = "更新内容"
                setTextColor(ContextCompat.getColor(context, R.color.brand_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
                setPadding(0, 0, 0, dp(8))
            })

            changelogLoading = ProgressBar(this@AboutVersionActivity).apply {
                layoutParams = LinearLayout.LayoutParams(dp(32), dp(32)).apply {
                    gravity = Gravity.CENTER
                }
            }
            addView(changelogLoading)

            changelogText = bodyText("正在获取最新更新内容...").apply {
                setLineSpacing(0f, 1.55f)
                setPadding(0, dp(4), 0, dp(4))
                visibility = android.view.View.GONE
            }
            addView(changelogText)
        })

        root.addView(cardView().apply {
            addView(TextView(this@AboutVersionActivity).apply {
                text = "开发者信息"
                setTextColor(ContextCompat.getColor(context, R.color.brand_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
                setPadding(0, 0, 0, dp(8))
            })
            addView(infoRow("作者", "大师"))
            addView(infoRow("联系方式", "QQ: 694738267"))
            addView(LinearLayout(this@AboutVersionActivity).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                setPadding(0, dp(8), 0, 0)
                addView(bodyText("点击QQ号联系作者"))
                foreground = context.obtainStyledAttributes(intArrayOf(android.R.attr.selectableItemBackground)).getDrawable(0)
                setOnClickListener {
                    try {
                        startActivity(Intent(Intent.ACTION_VIEW, Uri.parse("mqqwpa://im/chat?chat_type=wpa&uin=694738267")))
                    } catch (_: Exception) {}
                }
            })
        })

        root.addView(captionText("© 2026 狗东 App · All Rights Reserved").apply {
            gravity = Gravity.CENTER
            setPadding(0, dp(24), 0, dp(12))
        })

        loadChangelog()
    }

    private fun loadChangelog() {
        val updateChecker = UpdateChecker(AppServices.apiClient, this)
        lifecycleScope.launch {
            runCatching { updateChecker.check() }
                .onSuccess { updateInfo ->
                    changelogLoading.visibility = android.view.View.GONE
                    changelogText.visibility = android.view.View.VISIBLE
                    if (updateInfo != null && updateInfo.changelog.isNotBlank()) {
                        changelogText.text = updateInfo.changelog
                    } else {
                        changelogText.text = "当前已是最新版本 v${BuildConfig.VERSION_NAME}"
                    }
                }
                .onFailure {
                    changelogLoading.visibility = android.view.View.GONE
                    changelogText.visibility = android.view.View.VISIBLE
                    changelogText.text = sanitizeErrorMessage(it.message)
                }
        }
    }

    private fun infoRow(label: String, value: String): LinearLayout {
        return LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(0, dp(10), 0, dp(10))

            addView(TextView(this@AboutVersionActivity).apply {
                text = label
                setTextColor(themeColor(R.color.text_muted))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 0.35f)
            })

            addView(TextView(this@AboutVersionActivity).apply {
                text = value
                setTextColor(themeColor(R.color.text_primary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setTypeface(typeface, android.graphics.Typeface.BOLD)
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 0.65f)
                gravity = Gravity.END
            })

            background = android.graphics.drawable.GradientDrawable().apply {
                setColor(android.graphics.Color.TRANSPARENT)
            }
        }
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }
}
