package com.goudong.jd.ui.common

import android.animation.ObjectAnimator
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.view.animation.DecelerateInterpolator
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.goudong.jd.R
import com.goudong.jd.ui.common.AppTheme
import com.goudong.jd.ui.common.themeColor

class ResultTextActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AppTheme.applySystemBars(this)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        title = intent.getStringExtra(EXTRA_TITLE) ?: "结果"

        val p = dp(16)
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setBackgroundColor(themeColor(R.color.surface_soft))
            layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
        }
        applySafeStatusBar(root)

        // 内容卡片（入场动画从下方滑入）
        val card = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(p, p, p, dp(12))
            background = GradientDrawable().apply {
                setColor(themeColor(R.color.surface_card))
                cornerRadius = dp(16).toFloat()
                setStroke(dp(1), themeColor(R.color.border_default))
            }
            elevation = dp(2).toFloat()
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                setMargins(p, p, p, p)
            }
            // 初始位置：下方偏移
            translationY = dp(30).toFloat()
            alpha = 0f
        }

        val textView = TextView(this).apply {
            text = intent.getStringExtra(EXTRA_TEXT).orEmpty()
            setTextColor(themeColor(R.color.text_secondary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13.5f)
            setLineSpacing(0f, 1.5f)
            setTextIsSelectable(true)
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                bottomMargin = dp(12)
            }
        }
        card.addView(textView)

        // 复制按钮
        val copyBtn = TextView(this).apply {
            text = "复制全部内容"
            setTextColor(themeColor(R.color.text_secondary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
            gravity = Gravity.CENTER
            setPadding(dp(12), dp(8), dp(12), dp(8))
            background = GradientDrawable().apply {
                setColor(themeColor(R.color.chip_bg))
                cornerRadius = dp(10).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                gravity = Gravity.END
            }
            setOnClickListener {
                val clip = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                clip.setPrimaryClip(ClipData.newPlainText("result", textView.text))
                Toast.makeText(this@ResultTextActivity, "已复制到剪贴板", Toast.LENGTH_SHORT).show()
            }
        }
        card.addView(copyBtn)

        val scrollView = ScrollView(this).apply {
            layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
        }
        scrollView.addView(card)
        root.addView(scrollView)
        setContentView(root)

        // 入场动画
        ObjectAnimator.ofFloat(card, "translationY", dp(30).toFloat(), 0f).apply {
            duration = 280
            interpolator = DecelerateInterpolator()
            start()
        }
        ObjectAnimator.ofFloat(card, "alpha", 0f, 1f).apply {
            duration = 280
            start()
        }
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    companion object {
        private const val EXTRA_TITLE = "title"
        private const val EXTRA_TEXT = "text"

        fun intent(context: Context, title: String, text: String): Intent {
            return Intent(context, ResultTextActivity::class.java)
                .putExtra(EXTRA_TITLE, title)
                .putExtra(EXTRA_TEXT, text)
        }
    }
}
