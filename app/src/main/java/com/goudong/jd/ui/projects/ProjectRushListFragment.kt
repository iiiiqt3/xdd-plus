package com.goudong.jd.ui.projects

import android.graphics.Color
import android.graphics.Typeface
import android.util.TypedValue
import android.view.Gravity
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.LinearLayout
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import com.goudong.jd.R
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.makeScrollContainer

class ProjectRushListFragment : Fragment() {

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: android.os.Bundle?): View {
        val ctx = requireContext()
        val (scroll, root) = ctx.makeScrollContainer()

        root.addView(TextView(ctx).apply {
            text = "⚡ 项目抢兑"
            setTextColor(Color.parseColor("#0F172A"))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 17f)
            setTypeface(typeface, Typeface.BOLD)
            setPadding(ctx.dp(14), ctx.dp(4), ctx.dp(14), ctx.dp(8))
        })
        root.addView(ctx.captionText("选择要使用的抢兑项目").apply {
            setPadding(ctx.dp(14), 0, ctx.dp(14), ctx.dp(12))
        })
        root.addView(buildItemCard(
            icon = "🎵",
            title = "酷我提现",
            subtitle = "定时抢兑 · 00:00 / 09:00 / 13:00 / 17:00 / 20:00",
        ) {
            (parentFragment as? ProjectRushFragment)?.openKuwo()
        })
        return scroll
    }

    private fun buildItemCard(icon: String, title: String, subtitle: String, onClick: () -> Unit): View {
        val ctx = requireContext()
        val brandBlue = ContextCompat.getColor(ctx, R.color.brand_secondary)
        return ctx.cardView().apply {
            orientation = LinearLayout.HORIZONTAL
            gravity = Gravity.CENTER_VERTICAL
            setPadding(ctx.dp(16), ctx.dp(14), ctx.dp(14), ctx.dp(14))
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT,
            ).apply {
                marginStart = ctx.dp(14)
                marginEnd = ctx.dp(14)
                bottomMargin = ctx.dp(10)
            }
            addView(TextView(ctx).apply {
                text = icon
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 22f)
                gravity = Gravity.CENTER
                layoutParams = LinearLayout.LayoutParams(ctx.dp(40), ctx.dp(40))
            })
            val infoWrap = LinearLayout(ctx).apply {
                orientation = LinearLayout.VERTICAL
                layoutParams = LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                    marginStart = ctx.dp(10)
                }
            }
            infoWrap.addView(TextView(ctx).apply {
                text = title
                setTextColor(Color.parseColor("#0F172A"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
                setTypeface(typeface, Typeface.BOLD)
            })
            infoWrap.addView(TextView(ctx).apply {
                text = subtitle
                setTextColor(Color.parseColor("#64748B"))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11.5f)
                setPadding(0, ctx.dp(4), 0, 0)
            })
            addView(infoWrap)
            addView(TextView(ctx).apply {
                text = "›"
                setTextColor(brandBlue)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 24f)
                setTypeface(typeface, Typeface.BOLD)
                gravity = Gravity.CENTER
                layoutParams = LinearLayout.LayoutParams(ctx.dp(28), ctx.dp(28)).apply { marginStart = ctx.dp(6) }
            })
            setOnClickListener { onClick() }
            foreground = ctx.obtainStyledAttributes(intArrayOf(android.R.attr.selectableItemBackground)).getDrawable(0)
        }
    }
}
