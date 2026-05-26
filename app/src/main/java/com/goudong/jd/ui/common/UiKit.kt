package com.goudong.jd.ui.common

import android.app.AlertDialog
import android.content.Context
import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.net.Uri
import android.text.InputType
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.EditText
import android.widget.ImageButton
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import android.widget.Toast
import androidx.annotation.ColorInt
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.core.view.setPadding
import androidx.fragment.app.Fragment
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.ApiError
import com.goudong.jd.ui.auth.AuthActivity

fun Context.dp(value: Int): Int = (value * resources.displayMetrics.density).toInt()

fun AppCompatActivity.applySafeStatusBar(root: View? = null) {
    window.statusBarColor = Color.parseColor("#F4F7FB")
    window.decorView.systemUiVisibility = View.SYSTEM_UI_FLAG_LAYOUT_STABLE or View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR
    val statusBarHeight = run {
        val resId = resources.getIdentifier("status_bar_height", "dimen", "android")
        if (resId > 0) resources.getDimensionPixelSize(resId) else 0
    }
    root?.setPadding(root.paddingLeft, root.paddingTop + statusBarHeight, root.paddingRight, root.paddingBottom)
}

fun Context.pageBackground(view: View) {
    view.setBackgroundColor(Color.parseColor("#F4F7FB"))
}

fun Context.heroCard(
    title: String,
    subtitle: String,
    @ColorInt tint: Int,
    iconRes: Int = android.R.drawable.ic_menu_info_details,
): LinearLayout {
    return LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        val p = dp(18)
        setPadding(p, p, p, p)
        background = GradientDrawable(
            GradientDrawable.Orientation.TL_BR,
            intArrayOf(adjustAlpha(tint, 0.10f), Color.WHITE)
        ).apply {
            cornerRadius = dp(22).toFloat()
            setStroke(dp(1), adjustAlpha(tint, 0.14f))
        }
        elevation = dp(2).toFloat()
        layoutParams = ViewGroup.MarginLayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT).apply {
            bottomMargin = dp(12)
        }

        val topRow = LinearLayout(context).apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
        }
        val iconWrap = LinearLayout(context).apply {
            gravity = Gravity.CENTER
            background = GradientDrawable().apply {
                setColor(adjustAlpha(tint, 0.14f))
                cornerRadius = dp(12).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(dp(36), dp(36))
        }
        iconWrap.addView(ImageView(context).apply {
            setImageResource(iconRes)
            setColorFilter(tint)
            layoutParams = LinearLayout.LayoutParams(dp(18), dp(18))
        })
        val textWrap = LinearLayout(context).apply {
            orientation = LinearLayout.VERTICAL
            layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f).apply {
                marginStart = dp(10)
            }
        }
        textWrap.addView(TextView(context).apply {
            text = title
            setTextColor(Color.parseColor("#0F172A"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 18f)
            setTypeface(typeface, Typeface.BOLD)
            setTextIsSelectable(true)
        })
        textWrap.addView(TextView(context).apply {
            text = subtitle
            setTextColor(Color.parseColor("#475569"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11.5f)
            setLineSpacing(0f, 1.2f)
            setTextIsSelectable(true)
        })
        topRow.addView(iconWrap)
        topRow.addView(textWrap)
        addView(topRow)
    }
}

fun Context.cardView(): LinearLayout {
    return LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        val p = dp(14)
        setPadding(p, p, p, p)
        background = GradientDrawable().apply {
            setColor(Color.WHITE)
            cornerRadius = dp(20).toFloat()
            setStroke(dp(1), Color.parseColor("#E7EDF5"))
        }
        elevation = dp(2).toFloat()
        layoutParams = ViewGroup.MarginLayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT).apply {
            bottomMargin = dp(12)
        }
    }
}

fun Context.softCard(@ColorInt tint: Int): LinearLayout {
    return cardView().apply {
        background = GradientDrawable().apply {
            setColor(adjustAlpha(tint, 0.06f))
            cornerRadius = dp(20).toFloat()
            setStroke(dp(1), adjustAlpha(tint, 0.12f))
        }
    }
}

fun Context.sectionTitle(text: String): TextView {
    return TextView(this).apply {
        this.text = text
        setTextColor(Color.parseColor("#0F172A"))
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
        setTypeface(typeface, Typeface.BOLD)
        setTextIsSelectable(true)
    }
}

fun Context.bodyText(text: String = ""): TextView {
    return TextView(this).apply {
        this.text = text
        setTextColor(Color.parseColor("#475569"))
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.2f)
        setLineSpacing(0f, 1.28f)
        setTextIsSelectable(true)
    }
}

fun Context.captionText(text: String = ""): TextView {
    return TextView(this).apply {
        this.text = text
        setTextColor(Color.parseColor("#64748B"))
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 10.8f)
        setLineSpacing(0f, 1.2f)
        setTextIsSelectable(true)
    }
}

fun Context.statCard(title: String, value: String, @ColorInt tint: Int): LinearLayout {
    return softCard(tint).apply {
        addView(captionText(title))
        addView(TextView(context).apply {
            text = value
            setTextColor(Color.parseColor("#0F172A"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 19f)
            setTypeface(typeface, Typeface.BOLD)
        })
    }
}

fun Context.inputField(hint: String, multiline: Boolean = false, number: Boolean = false): EditText {
    return EditText(this).apply {
        this.hint = hint
        setTextColor(Color.parseColor("#0F172A"))
        setHintTextColor(Color.parseColor("#94A3B8"))
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 13.5f)
        background = GradientDrawable().apply {
            setColor(Color.parseColor("#F8FAFC"))
            cornerRadius = dp(15).toFloat()
            setStroke(dp(1), Color.parseColor("#D6E0EA"))
        }
        setPadding(dp(13))
        inputType = when {
            multiline -> InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_MULTI_LINE
            number -> InputType.TYPE_CLASS_NUMBER
            else -> InputType.TYPE_CLASS_TEXT
        }
        minHeight = dp(46)
        layoutParams = ViewGroup.MarginLayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT).apply {
            bottomMargin = dp(8)
        }
    }
}

fun Context.primaryButton(text: String, @ColorInt color: Int = ContextCompat.getColor(this, R.color.brand_primary)): Button {
    return Button(this).apply {
        this.text = text
        setTextColor(Color.WHITE)
        setAllCaps(false)
        textSize = 13.5f
        background = GradientDrawable(
            GradientDrawable.Orientation.LEFT_RIGHT,
            intArrayOf(color, blendColor(color, Color.WHITE, 0.16f))
        ).apply { cornerRadius = dp(16).toFloat() }
        layoutParams = ViewGroup.MarginLayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT).apply {
            bottomMargin = dp(8)
        }
        minHeight = dp(46)
        elevation = dp(1).toFloat()
    }
}

fun Context.primaryButton(text: String, onClick: () -> Unit): Button {
    return primaryButton(text).apply { setOnClickListener { onClick() } }
}

fun Context.secondaryButton(text: String): Button {
    return Button(this).apply {
        this.text = text
        setAllCaps(false)
        textSize = 13.5f
        setTextColor(Color.parseColor("#0F172A"))
        background = GradientDrawable().apply {
            setColor(Color.WHITE)
            cornerRadius = dp(16).toFloat()
            setStroke(dp(1), Color.parseColor("#D6E0EA"))
        }
        layoutParams = ViewGroup.MarginLayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT).apply {
            bottomMargin = dp(8)
        }
        minHeight = dp(46)
    }
}

fun Context.badge(text: String, @ColorInt tint: Int): TextView {
    return TextView(this).apply {
        this.text = "  $text  "
        setTextColor(tint)
        setTextSize(TypedValue.COMPLEX_UNIT_SP, 10.5f)
        setTypeface(typeface, Typeface.BOLD)
        background = GradientDrawable().apply {
            setColor(adjustAlpha(tint, 0.12f))
            cornerRadius = dp(999).toFloat()
        }
        setPadding(dp(2), dp(2), dp(2), dp(2))
    }
}

fun Context.actionTile(title: String, desc: String, @ColorInt tint: Int, onClick: () -> Unit): View {
    return softCard(tint).apply {
        orientation = LinearLayout.HORIZONTAL
        gravity = Gravity.CENTER_VERTICAL
        val iconWrap = LinearLayout(context).apply {
            gravity = Gravity.CENTER
            background = GradientDrawable().apply {
                setColor(adjustAlpha(tint, 0.14f))
                cornerRadius = context.dp(12).toFloat()
            }
            layoutParams = LinearLayout.LayoutParams(context.dp(40), context.dp(40))
        }
        val icon = ImageView(context).apply {
            setImageResource(android.R.drawable.ic_menu_info_details)
            setColorFilter(tint)
        }
        iconWrap.addView(icon)
        val textWrap = LinearLayout(context).apply {
            orientation = LinearLayout.VERTICAL
            layoutParams = LinearLayout.LayoutParams(0, ViewGroup.LayoutParams.WRAP_CONTENT, 1f).apply {
                marginStart = context.dp(10)
            }
        }
        textWrap.addView(sectionTitle(title).apply { textSize = 14f ; setTextIsSelectable(false) ; isClickable = false })
        textWrap.addView(captionText(desc).apply { setTextIsSelectable(false) ; isClickable = false })
        addView(iconWrap)
        addView(textWrap)
        setOnClickListener { onClick() }
    }
}

/**
 * 2列布局的快捷入口（iOS风格，带图标+标题+描述）
 * 每个 item 为 Triple(title, desc, onClick)
 */
fun Context.actionGridTile(
    items: List<Triple<String, String, () -> Unit>>,
    @ColorInt tint: Int
): LinearLayout {
    return cardView().apply {
        orientation = LinearLayout.VERTICAL
        items.chunked(2).forEachIndexed { rowIndex, chunk ->
            val rowLayout = LinearLayout(context).apply {
                orientation = LinearLayout.HORIZONTAL
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT
                ).apply {
                    if (rowIndex > 0) topMargin = context.dp(8)
                }
            }
            chunk.forEachIndexed { colIndex, (title, desc, onClick) ->
                val tile = LinearLayout(context).apply {
                    orientation = LinearLayout.VERTICAL
                    gravity = Gravity.CENTER_HORIZONTAL
                    background = GradientDrawable().apply {
                        setColor(adjustAlpha(tint, 0.06f))
                        cornerRadius = context.dp(14).toFloat()
                        setStroke(context.dp(1), adjustAlpha(tint, 0.13f))
                    }
                    layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                        marginStart = if (colIndex == 0) 0 else context.dp(8)
                        setPadding(context.dp(10), context.dp(12), context.dp(10), context.dp(12))
                    }
                    setOnClickListener { onClick() }

                    // 图标容器（居中）
                    addView(LinearLayout(context).apply {
                        orientation = LinearLayout.HORIZONTAL
                        gravity = Gravity.CENTER_HORIZONTAL
                        val iconWrap = LinearLayout(context).apply {
                            gravity = Gravity.CENTER
                            background = GradientDrawable().apply {
                                setColor(adjustAlpha(tint, 0.15f))
                                cornerRadius = context.dp(10).toFloat()
                            }
                            layoutParams = LinearLayout.LayoutParams(context.dp(34), context.dp(34))
                        }
                        iconWrap.addView(ImageView(context).apply {
                            setImageResource(android.R.drawable.ic_menu_info_details)
                            setColorFilter(tint)
                            layoutParams = LinearLayout.LayoutParams(context.dp(16), context.dp(16))
                        })
                        addView(iconWrap)
                    })

                    // 标题
                    addView(TextView(context).apply {
                        text = title
                        setTextColor(Color.parseColor("#0F172A"))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
                        setTypeface(typeface, Typeface.BOLD)
                        gravity = Gravity.CENTER
                        setPadding(0, context.dp(7), 0, 0)
                    })

                    // 描述
                    addView(TextView(context).apply {
                        text = desc
                        setTextColor(Color.parseColor("#64748B"))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 9.5f)
                        gravity = Gravity.CENTER
                        setLineSpacing(0f, 1.15f)
                        setPadding(0, context.dp(2), 0, 0)
                    })
                }
                rowLayout.addView(tile)
            }
            // 奇数个时补空白占位
            if (chunk.size == 1) {
                rowLayout.addView(View(context).apply {
                    layoutParams = LinearLayout.LayoutParams(0, 0, 1f).apply {
                        marginStart = context.dp(8)
                    }
                })
            }
            addView(rowLayout)
        }
    }
}

fun adjustAlpha(@ColorInt color: Int, factor: Float): Int {
    val alpha = Math.round(Color.alpha(color) * factor)
    return Color.argb(alpha, Color.red(color), Color.green(color), Color.blue(color))
}

fun blendColor(@ColorInt color: Int, @ColorInt target: Int, ratio: Float): Int {
    val inverse = 1f - ratio
    return Color.argb(
        255,
        (Color.red(color) * inverse + Color.red(target) * ratio).toInt(),
        (Color.green(color) * inverse + Color.green(target) * ratio).toInt(),
        (Color.blue(color) * inverse + Color.blue(target) * ratio).toInt(),
    )
}

fun Fragment.toast(text: String) {
    Toast.makeText(requireContext(), text, Toast.LENGTH_SHORT).show()
}

fun AppCompatActivity.toast(text: String) {
    Toast.makeText(this, text, Toast.LENGTH_SHORT).show()
}

fun sanitizeErrorMessage(message: String?): String {
    if (message.isNullOrBlank()) return "操作失败"
    var msg = message
    msg = Regex("https?://\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}(:\\d+)?[^\\s]*").replace(msg!!, "服务器")
    msg = Regex("\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}(:\\d+)?").replace(msg, "服务器")
    msg = Regex("https?://[^\\s]+").replace(msg, "服务器地址")
    if (msg.contains("failed to connect") || msg.contains("ConnectException") || msg.contains("timeout") || msg.contains("Unable to resolve host") || msg.contains("ECONNREFUSED") || msg.contains("ENETUNREACH")) {
        return "无法连接服务器，请检查网络或稍后重试"
    }
    return msg
}

fun Fragment.handlePortalError(error: Throwable, title: String = "提示", onUnauthorized: (() -> Unit)? = null) {
    val apiError = error as? ApiError
    if (apiError?.unauthorized == true) {
        AppServices.sessionManager.setAuthenticated(false)
        AppServices.apiClient.clearCookies()
        startActivity(Intent(requireContext(), AuthActivity::class.java))
        onUnauthorized?.invoke()
        return
    }
    alert(sanitizeErrorMessage(error.message), title)
}

fun AppCompatActivity.handlePortalError(error: Throwable, title: String = "提示", onUnauthorized: (() -> Unit)? = null) {
    val apiError = error as? ApiError
    if (apiError?.unauthorized == true) {
        AppServices.sessionManager.setAuthenticated(false)
        AppServices.apiClient.clearCookies()
        startActivity(Intent(this, AuthActivity::class.java))
        onUnauthorized?.invoke()
        return
    }
    alert(sanitizeErrorMessage(error.message), title)
}

fun Fragment.alert(message: String, title: String = "提示", onOk: (() -> Unit)? = null) {
    AlertDialog.Builder(requireContext())
        .setTitle(title)
        .setMessage(message)
        .setPositiveButton("确定") { _, _ -> onOk?.invoke() }
        .show()
}

fun AppCompatActivity.alert(message: String, title: String = "提示", onOk: (() -> Unit)? = null) {
    AlertDialog.Builder(this)
        .setTitle(title)
        .setMessage(message)
        .setPositiveButton("确定") { _, _ -> onOk?.invoke() }
        .show()
}

fun Fragment.openExternalUrl(url: String) {
    startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
}

fun AppCompatActivity.openExternalUrl(url: String) {
    startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
}

fun Context.openUrl(url: String) {
    startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
}

fun Context.makeScrollContainer(): Pair<ScrollView, LinearLayout> {
    val scroll = ScrollView(this).apply {
        setBackgroundColor(Color.parseColor("#F4F7FB"))
    }
    val content = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        val p = dp(14)
        setPadding(p, p, p, p)
        layoutParams = ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.WRAP_CONTENT)
    }
    scroll.addView(content)
    return scroll to content
}

fun Context.closeButton(onClick: () -> Unit): ImageButton {
    return ImageButton(this).apply {
        setImageResource(android.R.drawable.ic_menu_close_clear_cancel)
        background = null
        setOnClickListener { onClick() }
    }
}
