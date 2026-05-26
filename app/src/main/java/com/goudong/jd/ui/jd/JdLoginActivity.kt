package com.goudong.jd.ui.jd

import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.ViewGroup
import android.webkit.CookieManager
import android.webkit.WebChromeClient
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Button
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.applySafeStatusBar
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.primaryButton
import kotlinx.coroutines.launch

class JdLoginActivity : AppCompatActivity() {
    private lateinit var webView: WebView
    private lateinit var statusText: TextView
    private lateinit var submitBtn: Button
    private var jdCookie: String? = null
    private var isSubmitting = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        title = "京东登录"
        supportActionBar?.setDisplayHomeAsUpEnabled(true)

        val root = FrameLayout(this)
        applySafeStatusBar(root)
        val progress = ProgressBar(this, null, android.R.attr.progressBarStyleHorizontal).apply {
            layoutParams = FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT, Gravity.TOP)
            max = 100
        }
        webView = WebView(this).apply {
            layoutParams = FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT)
            settings.javaScriptEnabled = true
            settings.domStorageEnabled = true
            CookieManager.getInstance().setAcceptCookie(true)
            webChromeClient = object : WebChromeClient() {
                override fun onProgressChanged(view: WebView?, newProgress: Int) {
                    progress.progress = newProgress
                }
            }
            webViewClient = object : WebViewClient() {}
        }

        // 精简底部操作栏
        val bottomBar = LinearLayout(this).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(dp(14), dp(10), dp(14), dp(10))
            background = GradientDrawable().apply {
                setColor(Color.WHITE)
                cornerRadius = dp(16).toFloat()
            }
            elevation = dp(6).toFloat()
            layoutParams = FrameLayout.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT, Gravity.BOTTOM).apply {
                marginStart = dp(14)
                marginEnd = dp(14)
                bottomMargin = dp(14)
            }
        }
        // 状态文字（左）
        statusText = TextView(this).apply {
            text = "加载中..."
            setTextColor(Color.parseColor("#94A3B8"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
            layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                marginEnd = dp(6)
            }
        }
        // 刷新按钮（轻量文字风格）
        val refreshBtn = TextView(this).apply {
            text = "刷新Cookie"
            setTextColor(Color.parseColor("#0066FF"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setPadding(dp(10), 0, dp(10), 0)
            setOnClickListener { refreshCookies() }
        }
        // 提交按钮（填充圆角，与app主色一致）
        submitBtn = Button(this).apply {
            text = "提交"
            setAllCaps(false)
            setTextColor(Color.WHITE)
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setPadding(dp(16), dp(2), dp(16), dp(2))
            background = GradientDrawable().apply {
                setColor(Color.parseColor("#0066FF"))
                cornerRadius = dp(12).toFloat()
            }
            isEnabled = false
            alpha = 0.5f
            setOnClickListener { submitCookie() }
        }
        bottomBar.addView(statusText)
        bottomBar.addView(refreshBtn)
        bottomBar.addView(submitBtn)

        root.addView(webView)
        root.addView(progress)
        root.addView(bottomBar)
        setContentView(root)

        loadLoginPage()
    }

    private fun loadLoginPage() {
        statusText.text = "加载中..."
        webView.loadUrl(com.goudong.jd.data.model.AppEnvironment.JD_LOGIN_URL)
    }

    private fun refreshCookies() {
        val cm = CookieManager.getInstance()
        val cookies = cm.getCookie("https://m.jd.com").orEmpty() + "; " + cm.getCookie("https://plogin.m.jd.com").orEmpty()
        jdCookie = cookies
        val hasKey = cookies.contains("pt_key") && cookies.contains("pt_pin")
        submitBtn.isEnabled = hasKey
        submitBtn.alpha = if (hasKey) 1f else 0.5f
        statusText.text = if (hasKey) "Cookie 已获取" else "请先登录"
    }

    private fun submitCookie() {
        if (isSubmitting) return
        val qq = AppServices.sessionManager.loadQq()
        val cookie = jdCookie.orEmpty()
        if (qq.isBlank()) return alert("请先在京东页面输入QQ号")
        if (!cookie.contains("pt_key") || !cookie.contains("pt_pin")) return alert("Cookie 不完整，请先登录并刷新")

        isSubmitting = true
        submitBtn.isEnabled = false
        submitBtn.alpha = 0.5f
        statusText.text = "提交中..."

        lifecycleScope.launch {
            runCatching { AppServices.jdRepository.submitCookie(qq, cookie) }
                .onSuccess {
                    alert(it) {
                        clearWebViewCookies()
                        jdCookie = null
                        statusText.text = "已提交，请登录下一个账号"
                        isSubmitting = false
                        submitBtn.isEnabled = false
                        submitBtn.alpha = 0.5f
                        webView.loadUrl(com.goudong.jd.data.model.AppEnvironment.JD_LOGIN_URL)
                    }
                }
                .onFailure {
                    isSubmitting = false
                    submitBtn.isEnabled = true
                    submitBtn.alpha = 1f
                    statusText.text = "提交失败，请重试"
                    handlePortalError(it)
                }
        }
    }

    private fun clearWebViewCookies() {
        val cm = CookieManager.getInstance()
        cm.removeAllCookies(null)
        cm.flush()
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }
}
