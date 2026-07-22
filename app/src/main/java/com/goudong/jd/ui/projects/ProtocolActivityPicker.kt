package com.goudong.jd.ui.projects

import android.content.Context
import android.graphics.Color
import android.text.TextUtils
import android.util.TypedValue
import android.view.View
import android.view.ViewGroup
import android.widget.AdapterView
import android.widget.ArrayAdapter
import android.widget.LinearLayout
import android.widget.Spinner
import android.widget.TextView
import com.goudong.jd.data.model.ProtocolAccountOption
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.themeColor
import com.goudong.jd.R

object ProtocolActivityPicker {
    private const val SPINNER_TEXT_SP = 12.5f
    private const val HINT_TEXT_SP = 10.5f
    private const val LABEL_TEXT_SP = 12f

    data class MountedPicker(
        val spinner: Spinner,
        val options: List<ProtocolAccountOption>,
        val protoFieldKey: String?,
    )

    fun firstTemplateFieldKey(template: String?): String? {
        val regex = Regex("""\{\{\.([^}]+)\}\}""")
        return regex.find(template.orEmpty())?.groupValues?.getOrNull(1)?.trim()?.takeIf { it.isNotEmpty() }
    }

    private fun styleSpinnerText(
        context: Context,
        textView: TextView,
        position: Int,
        options: List<ProtocolAccountOption>,
        dropdown: Boolean,
    ) {
        textView.setTextSize(TypedValue.COMPLEX_UNIT_SP, SPINNER_TEXT_SP)
        textView.setLineSpacing(0f, 0.95f)
        textView.ellipsize = TextUtils.TruncateAt.END
        textView.maxLines = if (dropdown) 3 else 2
        val padV = context.dp(if (dropdown) 8 else 2)
        val padH = context.dp(2)
        textView.setPadding(padH, padV, padH, padV)
        if (position > 0) {
            val opt = options.getOrNull(position - 1)
            val disabled = opt?.selectable == false
            textView.setTextColor(if (disabled) Color.parseColor("#9CA3AF") else context.themeColor(R.color.text_primary))
        } else {
            textView.setTextColor(context.themeColor(R.color.text_hint))
        }
    }

    fun mount(
        context: Context,
        container: LinearLayout,
        insertIndex: Int,
        options: List<ProtocolAccountOption>,
        template: String?,
        selectedFillRef: String? = null,
        onSelected: (String) -> Unit,
    ): MountedPicker {
        val protoFieldKey = firstTemplateFieldKey(template)
        container.addView(TextView(context).apply {
            text = "协议账号"
            setTextColor(context.themeColor(R.color.text_secondary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, LABEL_TEXT_SP)
            setPadding(0, context.dp(10), 0, context.dp(4))
        }, insertIndex)
        val labels = mutableListOf("请选择协议账号")
        labels.addAll(options.map { it.label ?: it.nickname ?: it.fillRef.orEmpty() })
        val spinner = Spinner(context).apply {
            minimumHeight = context.dp(38)
            setPadding(context.dp(2), context.dp(0), context.dp(2), context.dp(0))
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT,
            ).apply { bottomMargin = context.dp(4) }
            adapter = object : ArrayAdapter<String>(context, android.R.layout.simple_spinner_item, labels) {
                override fun getView(position: Int, convertView: View?, parent: ViewGroup): View {
                    val view = super.getView(position, convertView, parent)
                    styleSpinnerText(context, view.findViewById(android.R.id.text1), position, options, dropdown = false)
                    return view
                }

                override fun getDropDownView(position: Int, convertView: View?, parent: ViewGroup): View {
                    val view = super.getDropDownView(position, convertView, parent)
                    val textView = view.findViewById<TextView>(android.R.id.text1)
                    styleSpinnerText(context, textView, position, options, dropdown = true)
                    if (position > 0) {
                        val disabled = options.getOrNull(position - 1)?.selectable == false
                        textView.isEnabled = !disabled
                        view.isEnabled = !disabled
                    }
                    return view
                }

                override fun isEnabled(position: Int): Boolean {
                    if (position == 0) return true
                    return options.getOrNull(position - 1)?.selectable != false
                }
            }.also { it.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item) }
            onItemSelectedListener = object : AdapterView.OnItemSelectedListener {
                override fun onItemSelected(parent: AdapterView<*>?, view: View?, position: Int, id: Long) {
                    if (position <= 0) return
                    val fillRef = options.getOrNull(position - 1)?.fillRef.orEmpty()
                    if (fillRef.isNotBlank()) onSelected(fillRef)
                }
                override fun onNothingSelected(parent: AdapterView<*>?) {}
            }
        }
        container.addView(spinner, insertIndex + 1)
        val selected = selectedFillRef?.trim().orEmpty()
        if (selected.isNotEmpty()) {
            val idx = options.indexOfFirst {
                it.fillRef == selected || it.wxid == selected || it.openid == selected
            }
            if (idx >= 0) spinner.setSelection(idx + 1, false)
        }
        if (options.isEmpty()) {
            container.addView(TextView(context).apply {
                text = "暂无在线协议账号，请先到协议接入扫码登录"
                setTextColor(context.themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, HINT_TEXT_SP)
                setPadding(0, context.dp(4), 0, context.dp(6))
            }, insertIndex + 2)
        } else if (options.none { it.selectable != false }) {
            container.addView(TextView(context).apply {
                text = "本活动在线协议账号已全部上车"
                setTextColor(context.themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, HINT_TEXT_SP)
                setPadding(0, context.dp(4), 0, context.dp(6))
            }, insertIndex + 2)
        }
        return MountedPicker(spinner, options, protoFieldKey)
    }
}
