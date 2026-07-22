package com.goudong.jd.ui.projects

import android.content.Context
import android.graphics.Color
import android.util.TypedValue
import android.view.View
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
    data class MountedPicker(
        val spinner: Spinner,
        val options: List<ProtocolAccountOption>,
        val protoFieldKey: String?,
    )

    fun firstTemplateFieldKey(template: String?): String? {
        val regex = Regex("""\{\{\.([^}]+)\}\}""")
        return regex.find(template.orEmpty())?.groupValues?.getOrNull(1)?.trim()?.takeIf { it.isNotEmpty() }
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
            setTextColor(themeColor(R.color.text_secondary))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 12.5f)
            setPadding(0, dp(14), 0, dp(6))
        }, insertIndex)
        val labels = mutableListOf("请选择协议账号")
        labels.addAll(options.map { it.label ?: it.nickname ?: it.fillRef.orEmpty() })
        val spinner = Spinner(context).apply {
            adapter = object : ArrayAdapter<String>(context, android.R.layout.simple_spinner_item, labels) {
                override fun getDropDownView(position: Int, convertView: View?, parent: android.view.ViewGroup): View {
                    val view = super.getDropDownView(position, convertView, parent)
                    val textView = view.findViewById<TextView>(android.R.id.text1)
                    if (position > 0) {
                        val opt = options.getOrNull(position - 1)
                        val disabled = opt?.selectable == false
                        textView.setTextColor(if (disabled) Color.parseColor("#9CA3AF") else themeColor(R.color.text_primary))
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
                setTextColor(themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, dp(6), 0, dp(8))
            }, insertIndex + 2)
        } else if (options.none { it.selectable != false }) {
            container.addView(TextView(context).apply {
                text = "本活动在线协议账号已全部上车"
                setTextColor(themeColor(R.color.text_hint))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                setPadding(0, dp(6), 0, dp(8))
            }, insertIndex + 2)
        }
        return MountedPicker(spinner, options, protoFieldKey)
    }
}
