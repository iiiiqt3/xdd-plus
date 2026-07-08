package com.goudong.jd.push

import android.app.Activity
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Handler
import android.os.Looper
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.TextView
import com.goudong.jd.R
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.themeColor

object InAppPushBanner {
    private const val AUTO_DISMISS_MS = 6000L
    private const val POPUP_DISMISS_MS = 12000L

    private val handler = Handler(Looper.getMainLooper())
    private var hideRunnable: Runnable? = null
    private var bannerView: View? = null

    fun showSummary(activity: Activity, items: List<ForegroundPushNotifier.PushPayload>) {
        if (items.isEmpty()) {
            dismiss()
            return
        }

        val root = activity.findViewById<ViewGroup>(android.R.id.content) ?: return
        val ctx = activity
        val latest = items.last()
        val isPopup = items.any { it.displayType.equals("popup", ignoreCase = true) }

        val displayTitle: String
        val displayBody: String
        if (items.size == 1) {
            displayTitle = latest.title
            displayBody = latest.body
        } else {
            displayTitle = "您有 ${items.size} 条新消息"
            displayBody = "最新：${latest.title}"
        }

        val existing = bannerView
        if (existing != null && existing.parent === root) {
            updateBannerText(existing, displayTitle, displayBody, isPopup)
            resetAutoDismiss(isPopup)
            return
        }

        dismissImmediate()
        val banner = buildBanner(ctx, displayTitle, displayBody, isPopup) {
            onBannerClicked(ctx, items)
        }

        val topInset = run {
            val resId = ctx.resources.getIdentifier("status_bar_height", "dimen", "android")
            if (resId > 0) ctx.resources.getDimensionPixelSize(resId) else ctx.dp(24)
        }

        val params = FrameLayout.LayoutParams(
            FrameLayout.LayoutParams.MATCH_PARENT,
            FrameLayout.LayoutParams.WRAP_CONTENT,
        ).apply {
            gravity = Gravity.TOP
            topMargin = topInset + ctx.dp(8)
            marginStart = ctx.dp(12)
            marginEnd = ctx.dp(12)
        }

        root.addView(banner, params)
        bannerView = banner

        banner.alpha = 0f
        banner.translationY = -ctx.dp(40).toFloat()
        banner.animate()
            .translationY(0f)
            .alpha(1f)
            .setDuration(220)
            .start()

        vibrateOnce(ctx)
        resetAutoDismiss(isPopup)
    }

    private fun buildBanner(
        ctx: Activity,
        title: String,
        body: String,
        isPopup: Boolean,
        onClick: () -> Unit,
    ): LinearLayout {
        return LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            elevation = ctx.dp(if (isPopup) 16 else 12).toFloat()
            background = GradientDrawable().apply {
                setColor(ctx.themeColor(R.color.surface_card))
                cornerRadius = ctx.dp(14).toFloat()
                setStroke(ctx.dp(if (isPopup) 2 else 1), ctx.themeColor(if (isPopup) R.color.brand_primary else R.color.border_light))
            }
            setPadding(ctx.dp(14), ctx.dp(12), ctx.dp(14), ctx.dp(12))
            setOnClickListener {
                dismiss()
                onClick()
            }
            tag = BannerTag(title, body)
            addTitleView(ctx, title)
            addBodyView(ctx, body)
        }
    }

    private fun LinearLayout.addTitleView(ctx: Activity, text: String) {
        addView(TextView(ctx).apply {
            this.text = text
            tag = "banner_title"
            setTextColor(ctx.themeColor(R.color.text_primary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
            setTypeface(typeface, Typeface.BOLD)
            maxLines = 1
            ellipsize = android.text.TextUtils.TruncateAt.END
        })
    }

    private fun LinearLayout.addBodyView(ctx: Activity, text: String) {
        addView(TextView(ctx).apply {
            this.text = text
            tag = "banner_body"
            setTextColor(ctx.themeColor(R.color.text_secondary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            maxLines = 2
            ellipsize = android.text.TextUtils.TruncateAt.END
            setPadding(0, ctx.dp(4), 0, 0)
        })
    }

    private fun updateBannerText(banner: View, title: String, body: String, isPopup: Boolean = false) {
        val container = banner as? LinearLayout ?: return
        (container.findViewWithTag<TextView>("banner_title"))?.text = title
        (container.findViewWithTag<TextView>("banner_body"))?.text = body
        container.tag = BannerTag(title, body)
        vibrateOnce(banner.context)
    }

    private fun onBannerClicked(ctx: Activity, items: List<ForegroundPushNotifier.PushPayload>) {
        if (items.size == 1) {
            val item = items.first()
            ForegroundPushNotifier.removePending(item.key)
            if (item.notificationId > 0 && item.action.isBlank()) {
                PushNavigationHelper.openNotification(ctx, item.notificationId, openList = false)
            } else {
                PushNavigationHelper.openEphemeral(
                    ctx,
                    item.action.ifBlank { PushNavigationHelper.ACTION_SHOW_BODY },
                    item.title,
                    item.fullBody.ifBlank { item.body },
                )
            }
            return
        }
        ForegroundPushNotifier.clearPending()
        PushNavigationHelper.openNotification(ctx, notificationId = 0, openList = true)
    }

    private fun resetAutoDismiss(isPopup: Boolean = false) {
        hideRunnable?.let { handler.removeCallbacks(it) }
        hideRunnable = Runnable { dismiss() }
        handler.postDelayed(hideRunnable!!, if (isPopup) POPUP_DISMISS_MS else AUTO_DISMISS_MS)
    }

    fun dismiss() {
        hideRunnable?.let { handler.removeCallbacks(it) }
        hideRunnable = null
        val view = bannerView ?: return
        bannerView = null
        view.animate()
            .translationY(-view.context.dp(24).toFloat())
            .alpha(0f)
            .setDuration(180)
            .withEndAction {
                (view.parent as? ViewGroup)?.removeView(view)
            }
            .start()
    }

    private fun dismissImmediate() {
        hideRunnable?.let { handler.removeCallbacks(it) }
        hideRunnable = null
        val view = bannerView ?: return
        bannerView = null
        (view.parent as? ViewGroup)?.removeView(view)
    }

    private fun vibrateOnce(ctx: android.content.Context) {
        runCatching {
            val vibrator = ctx.getSystemService(android.os.Vibrator::class.java)
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.O) {
                vibrator?.vibrate(
                    android.os.VibrationEffect.createOneShot(40, android.os.VibrationEffect.DEFAULT_AMPLITUDE),
                )
            } else {
                @Suppress("DEPRECATION")
                vibrator?.vibrate(40)
            }
        }
    }

    private data class BannerTag(val title: String, val body: String)
}
