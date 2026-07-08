package com.goudong.jd.ui.common

import android.content.Context
import android.content.res.Configuration
import android.os.Build
import android.view.View
import androidx.annotation.ColorInt
import androidx.annotation.ColorRes
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import com.goudong.jd.R

object AppTheme {
    val coinLogFilters = listOf(
        "" to "全部",
        "web" to "网页",
        "app" to "App",
        "wx" to "微信",
        "bot" to "机器人",
        "admin" to "后台及其他",
    )

    fun isDark(context: Context): Boolean {
        val mode = context.resources.configuration.uiMode and Configuration.UI_MODE_NIGHT_MASK
        return mode == Configuration.UI_MODE_NIGHT_YES
    }

    @ColorInt
    fun color(context: Context, @ColorRes resId: Int): Int = ContextCompat.getColor(context, resId)

    @ColorInt fun pageBackground(context: Context) = color(context, R.color.surface_soft)

    @ColorInt fun cardBackground(context: Context) = color(context, R.color.surface_card)

    @ColorInt fun textPrimary(context: Context) = color(context, R.color.text_primary)

    @ColorInt fun textSecondary(context: Context) = color(context, R.color.text_secondary)

    @ColorInt fun textMuted(context: Context) = color(context, R.color.text_muted)

    @ColorInt fun textHint(context: Context) = color(context, R.color.text_hint)

    @ColorInt fun borderDefault(context: Context) = color(context, R.color.border_default)

    @ColorInt fun borderLight(context: Context) = color(context, R.color.border_light)

    @ColorInt fun inputBackground(context: Context) = color(context, R.color.input_bg)

    @ColorInt fun inputBorder(context: Context) = color(context, R.color.input_border)

    @ColorInt fun chipBackground(context: Context) = color(context, R.color.chip_bg)

    @ColorInt fun positive(context: Context) = color(context, R.color.positive)

    @ColorInt fun negative(context: Context) = color(context, R.color.negative)

    fun applySystemBars(activity: AppCompatActivity) {
        val dark = isDark(activity)
        activity.window.statusBarColor = pageBackground(activity)
        activity.window.navigationBarColor = pageBackground(activity)
        var flags = View.SYSTEM_UI_FLAG_LAYOUT_STABLE
        if (!dark) {
            flags = flags or View.SYSTEM_UI_FLAG_LIGHT_STATUS_BAR
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                flags = flags or View.SYSTEM_UI_FLAG_LIGHT_NAVIGATION_BAR
            }
        }
        activity.window.decorView.systemUiVisibility = flags
    }

    data class TagStyle(@ColorInt val background: Int, @ColorInt val text: Int)

    fun sourceTagStyle(context: Context, tagCls: String?, sourceKey: String?): TagStyle {
        val cls = tagCls?.takeIf { it.isNotBlank() } ?: when (sourceKey?.lowercase()) {
            "app" -> "tag-info"
            "bot" -> "tag-warning"
            "wx", "web" -> "tag-success"
            else -> "tag-admin"
        }
        return when (cls) {
            "tag-info" -> TagStyle(color(context, R.color.tag_info_bg), color(context, R.color.tag_info_text))
            "tag-warning" -> TagStyle(color(context, R.color.tag_warning_bg), color(context, R.color.tag_warning_text))
            "tag-success" -> TagStyle(color(context, R.color.tag_success_bg), color(context, R.color.tag_success_text))
            else -> TagStyle(color(context, R.color.tag_admin_bg), color(context, R.color.tag_admin_text))
        }
    }

    fun sourceLabel(log: com.goudong.jd.data.model.CoinLog): String {
        if (!log.sourceLabel.isNullOrBlank()) return log.sourceLabel
        return when (log.source?.lowercase()) {
            "web" -> "网页"
            "app" -> if (log.clientPlatform == "android") "App(安卓)" else if (log.clientPlatform == "ios") "App(iOS)" else "App"
            "wx" -> "微信"
            "bot" -> "机器人"
            "admin" -> "后台及其他"
            else -> log.source.orEmpty()
        }
    }
}

fun Context.themeColor(@ColorRes resId: Int): Int = AppTheme.color(this, resId)
