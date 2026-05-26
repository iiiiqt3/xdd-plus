package com.goudong.jd.ui.auth

import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.text.InputType
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.widget.CheckBox
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import android.widget.ViewFlipper
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.inputField
import com.goudong.jd.ui.common.primaryButton
import com.goudong.jd.ui.common.secondaryButton
import com.goudong.jd.ui.common.sectionTitle
import kotlinx.coroutines.launch

class AuthActivity : AppCompatActivity() {
    private lateinit var flipper: ViewFlipper
    private val PREFS_NAME = "auth_prefs"
    private val KEY_SAVED_USER = "saved_username"
    private val KEY_SAVED_PASS = "saved_password"
    private val KEY_REMEMBER = "remember_me"

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        title = "登录"
        supportActionBar?.setDisplayHomeAsUpEnabled(true)

        val scroll = ScrollView(this).apply {
            setBackgroundColor(Color.parseColor("#F4F7FB"))
        }
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
            setPadding(dp(24), dp(48), dp(24), dp(32))
        }
        scroll.addView(root)

        // 用户头像 + 欢迎区域
        root.addView(LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                bottomMargin = dp(32)
            }
            addView(LinearLayout(context).apply {
                gravity = Gravity.CENTER
                background = GradientDrawable().apply {
                    setColor(ContextCompat.getColor(context, R.color.brand_primary))
                    cornerRadius = dp(28).toFloat()
                }
                layoutParams = LinearLayout.LayoutParams(dp(80), dp(80))
                addView(TextView(context).apply {
                    text = "👤"
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 36f)
                    gravity = Gravity.CENTER
                    setPadding(0, 0, 0, dp(4))
                })
            })
            addView(TextView(context).apply {
                text = "欢迎回来"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 24f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                setPadding(0, dp(16), 0, dp(4))
            })
            addView(TextView(context).apply {
                text = "登录后即可访问全部功能"
                setTextColor(Color.parseColor("#64748B"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                gravity = Gravity.CENTER
            })
        })

        flipper = ViewFlipper(this).apply { layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT) }
        flipper.addView(buildLoginPage())
        flipper.addView(buildRegisterPage())
        flipper.addView(buildResetPage())
        root.addView(flipper)
        setContentView(scroll)
    }

    private fun styledInput(hint: String, multiline: Boolean = false, number: Boolean = false): EditText {
        return EditText(this).apply {
            this.hint = hint
            setTextColor(Color.parseColor("#0F172A"))
            setHintTextColor(Color.parseColor("#94A3B8"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
            background = GradientDrawable().apply {
                setColor(Color.parseColor("#F8FAFC"))
                cornerRadius = dp(14).toFloat()
                setStroke(dp(1), Color.parseColor("#D6E0EA"))
            }
            setPadding(dp(16), dp(14), dp(16), dp(14))
            inputType = when {
                multiline -> InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_MULTI_LINE
                number -> InputType.TYPE_CLASS_NUMBER
                else -> InputType.TYPE_CLASS_TEXT
            }
            minHeight = dp(48)
            layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                bottomMargin = dp(14)
            }
        }
    }

    private fun buildLoginPage(): View {
        val prefs = getSharedPreferences(PREFS_NAME, MODE_PRIVATE)
        val remember = prefs.getBoolean(KEY_REMEMBER, false)
        val savedUser = prefs.getString(KEY_SAVED_USER, "").orEmpty()
        val savedPass = prefs.getString(KEY_SAVED_PASS, "").orEmpty()

        val loginUser = styledInput("网页账号").apply { if (remember) setText(savedUser) }
        val loginPass = styledInput("登录密码").apply {
            inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD
            if (remember) setText(savedPass)
        }
        val rememberCheck = CheckBox(this).apply {
            text = "记住账号密码"
            isChecked = remember
            setTextColor(Color.parseColor("#475569"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setPadding(dp(4), 0, 0, dp(12))
            buttonTintList = android.content.res.ColorStateList.valueOf(ContextCompat.getColor(this@AuthActivity, R.color.brand_primary))
        }
        val loginBtn = primaryButton("登录用户中心")
        val registerBtn = secondaryButton("注册新账号")
        val resetBtn = secondaryButton("忘记密码")

        loginBtn.setOnClickListener {
            val username = loginUser.text?.toString().orEmpty().trim()
            val password = loginPass.text?.toString().orEmpty()
            if (username.isBlank() || password.isBlank()) return@setOnClickListener alert("请填写账号和密码")
            // 记住密码
            prefs.edit().apply {
                putBoolean(KEY_REMEMBER, rememberCheck.isChecked)
                if (rememberCheck.isChecked) {
                    putString(KEY_SAVED_USER, username)
                    putString(KEY_SAVED_PASS, password)
                } else {
                    remove(KEY_SAVED_USER)
                    remove(KEY_SAVED_PASS)
                }
                apply()
            }
            lifecycleScope.launch {
                runCatching { AppServices.authRepository.login(username, password) }
                    .onSuccess { finishWithSuccess("登录成功") }
                    .onFailure { alert(it.message ?: "登录失败") }
            }
        }
        registerBtn.setOnClickListener { flipper.displayedChild = 1; title = "注册" }
        resetBtn.setOnClickListener { flipper.displayedChild = 2; title = "重置密码" }

        return cardView().apply {
            setPadding(dp(20), dp(24), dp(20), dp(20))
            addView(TextView(context).apply {
                text = "登录"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 20f)
                setTypeface(typeface, Typeface.BOLD)
                setPadding(0, 0, 0, dp(20))
            })
            addView(loginUser)
            addView(loginPass)
            addView(rememberCheck)
            addView(loginBtn)
            // 分割线 + 注册/忘记
            addView(android.view.View(context).apply {
                setBackgroundColor(Color.parseColor("#E7EDF5"))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(1)).apply {
                    topMargin = dp(8)
                    bottomMargin = dp(12)
                }
            })
            val linkRow = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER
            }
            linkRow.addView(registerBtn.apply {
                layoutParams = LinearLayout.LayoutParams(0, dp(40), 1f).apply { marginEnd = dp(8) }
            })
            linkRow.addView(resetBtn.apply {
                layoutParams = LinearLayout.LayoutParams(0, dp(40), 1f)
            })
            addView(linkRow)
        }
    }

    private fun buildRegisterPage(): View {
        val username = styledInput("新账号")
        val password = styledInput("登录密码").apply {
            inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD
        }
        val bindCode = styledInput("绑定码 / UserID")
        val submit = primaryButton("注册并进入用户中心")
        val back = secondaryButton("返回登录")

        submit.setOnClickListener {
            val u = username.text?.toString().orEmpty().trim()
            val p = password.text?.toString().orEmpty()
            val b = bindCode.text?.toString().orEmpty().trim()
            if (u.isBlank() || p.isBlank() || b.isBlank()) return@setOnClickListener alert("请完整填写注册信息")
            lifecycleScope.launch {
                runCatching { AppServices.authRepository.register(u, p, b) }
                    .onSuccess { finishWithSuccess(it) }
                    .onFailure { alert(it.message ?: "注册失败") }
            }
        }
        back.setOnClickListener { flipper.displayedChild = 0; title = "登录" }

        return cardView().apply {
            setPadding(dp(20), dp(24), dp(20), dp(20))
            addView(TextView(context).apply {
                text = "注册新账号"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 20f)
                setTypeface(typeface, Typeface.BOLD)
                setPadding(0, 0, 0, dp(6))
            })
            addView(TextView(context).apply {
                text = "请先给机器人发送「账号注册」获取绑定码"
                setTextColor(Color.parseColor("#64748B"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
                setPadding(0, 0, 0, dp(20))
            })
            addView(username)
            addView(password)
            addView(bindCode)
            addView(submit)
            addView(android.view.View(context).apply {
                setBackgroundColor(Color.parseColor("#E7EDF5"))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(1)).apply {
                    topMargin = dp(8)
                    bottomMargin = dp(12)
                }
            })
            addView(back)
        }
    }

    private fun buildResetPage(): View {
        val code = styledInput("6 位重置验证码", number = true)
        val username = styledInput("网页账号")
        val password = styledInput("新密码").apply {
            inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_VARIATION_PASSWORD
        }
        val verify = secondaryButton("验证验证码")
        val submit = primaryButton("重置密码", getColor(R.color.brand_red))
        val back = secondaryButton("返回登录")

        verify.setOnClickListener {
            val c = code.text?.toString().orEmpty().trim()
            if (c.isBlank()) return@setOnClickListener alert("请输入验证码")
            lifecycleScope.launch {
                runCatching { AppServices.authRepository.fetchResetInfo(c) }
                    .onSuccess { username.setText(it) }
                    .onFailure { alert(it.message ?: "验证失败") }
            }
        }
        submit.setOnClickListener {
            val c = code.text?.toString().orEmpty().trim()
            val u = username.text?.toString().orEmpty().trim()
            val p = password.text?.toString().orEmpty()
            if (c.isBlank() || u.isBlank() || p.isBlank()) return@setOnClickListener alert("请完整填写重置信息")
            lifecycleScope.launch {
                runCatching { AppServices.authRepository.resetPassword(c, u, p) }
                    .onSuccess { alert(it) { finishWithSuccess(it) } }
                    .onFailure { alert(it.message ?: "重置失败") }
            }
        }
        back.setOnClickListener { flipper.displayedChild = 0; title = "登录" }

        return cardView().apply {
            setPadding(dp(20), dp(24), dp(20), dp(20))
            addView(TextView(context).apply {
                text = "重置密码"
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 20f)
                setTypeface(typeface, Typeface.BOLD)
                setPadding(0, 0, 0, dp(6))
            })
            addView(TextView(context).apply {
                text = "请先给机器人发送「忘记密码」获取 6 位重置验证码"
                setTextColor(Color.parseColor("#64748B"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
                setPadding(0, 0, 0, dp(20))
            })
            addView(code)
            addView(verify)
            addView(username)
            addView(password)
            addView(submit)
            addView(android.view.View(context).apply {
                setBackgroundColor(Color.parseColor("#E7EDF5"))
                layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, dp(1)).apply {
                    topMargin = dp(8)
                    bottomMargin = dp(12)
                }
            })
            addView(back)
        }
    }

    override fun onSupportNavigateUp(): Boolean {
        if (::flipper.isInitialized && flipper.displayedChild != 0) {
            flipper.displayedChild = 0
            title = "登录"
            return true
        }
        finish()
        return true
    }

    private fun finishWithSuccess(message: String) {
        setResult(RESULT_OK)
        android.widget.Toast.makeText(this, message, android.widget.Toast.LENGTH_SHORT).show()
        finish()
    }
}
