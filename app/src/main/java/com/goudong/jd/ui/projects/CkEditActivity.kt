package com.goudong.jd.ui.projects

import android.content.Intent
import android.graphics.Color
import android.graphics.Typeface
import android.os.Bundle
import android.text.InputType
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.data.model.PortalActivityField
import kotlinx.coroutines.launch
import com.goudong.jd.ui.common.alert
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.primaryButton
import com.goudong.jd.ui.common.sectionTitle
import com.goudong.jd.ui.common.themeColor
import com.goudong.jd.ui.common.AppTheme

class CkEditActivity : AppCompatActivity() {
    private lateinit var projectRemark: String
    private lateinit var activityId: String
    private var originalCk: String = ""
    private var ckTemplate: String = ""
    private var isProtocolActivity: Boolean = false
    private var inputFields: List<PortalActivityField> = emptyList()
    private val fieldInputs = mutableMapOf<String, EditText>()
    private val fieldNames = mutableListOf<String>()
    private lateinit var previewText: TextView
    private var submitting = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AppTheme.applySystemBars(this)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)
        title = "修改 CK"

        projectRemark = intent.getStringExtra(EXTRA_REMARK) ?: ""
        activityId = intent.getStringExtra(EXTRA_ACTIVITY_ID) ?: ""
        originalCk = intent.getStringExtra(EXTRA_CK_VALUE) ?: ""
        ckTemplate = intent.getStringExtra(EXTRA_CK_TEMPLATE) ?: ""
        isProtocolActivity = intent.getBooleanExtra(EXTRA_IS_PROTOCOL, false)
        @Suppress("UNCHECKED_CAST")
        inputFields = intent.getSerializableExtra(EXTRA_INPUT_FIELDS) as? List<PortalActivityField> ?: emptyList()

        val scroll = ScrollView(this).apply {
            setBackgroundColor(themeColor(R.color.surface_soft))
        }
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(18), dp(18), dp(18), dp(24))
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            )
        }

        root.addView(cardView().apply {
            addView(sectionTitle("项目信息"))
            addView(bodyText("备注：$projectRemark").apply { setPadding(0, dp(4), 0, dp(10)) })
            addView(captionText("活动ID：$activityId"))
        })

        val templateFields = getCkTemplateFields(ckTemplate)
        val parsedFields = if (templateFields.isNotEmpty()) splitCkValueByTemplate(ckTemplate, originalCk) else null
        val visibleFields = inputFields.filter { field -> templateFields.contains(field.key) }
        val fieldsCard = cardView()
        fieldsCard.addView(sectionTitle("CK 字段编辑"))

        root.addView(fieldsCard.apply {
            if (parsedFields != null && visibleFields.isNotEmpty()) {
                fieldNames.clear()
                fieldNames.addAll(visibleFields.mapNotNull { it.key })

                visibleFields.forEach { field ->
                    val key = field.key ?: return@forEach
                    val prompt = field.prompt?.takeIf { it.isNotBlank() } ?: key

                    addView(captionText(key).apply {
                        setPadding(0, dp(10), 0, dp(4))
                        setTypeface(typeface, Typeface.BOLD)
                        setTextColor(ContextCompat.getColor(context, R.color.brand_secondary))
                    })

                    if (prompt != key) {
                        addView(bodyText(prompt).apply {
                            setTextColor(themeColor(R.color.text_hint))
                            setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                            setPadding(0, 0, 0, dp(6))
                        })
                    }

                    val inputValue = parsedFields[key] ?: ""
                    val input = EditText(context).apply {
                        hint = "请输入 $key 的值"
                        setText(inputValue)
                        setTextColor(themeColor(R.color.text_primary))
                        setHintTextColor(themeColor(R.color.text_hint))
                        setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                        background = android.graphics.drawable.GradientDrawable().apply {
                            setColor(themeColor(R.color.input_bg))
                            cornerRadius = dp(12).toFloat()
                            setStroke(dp(1), themeColor(R.color.input_border))
                        }
                        setPadding(dp(14), dp(10), dp(14), dp(10))
                        inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_MULTI_LINE
                        minLines = 1
                        maxLines = 5
                        isSingleLine = false
                        minHeight = dp(44)
                        layoutParams = LinearLayout.LayoutParams(
                            LinearLayout.LayoutParams.MATCH_PARENT,
                            LinearLayout.LayoutParams.WRAP_CONTENT
                        ).apply { bottomMargin = dp(14) }
                        id = View.generateViewId()
                    }
                    fieldInputs[key] = input
                    addView(input)
                }

                addView(captionText("提示：字段值将按模板自动拼接，无需手动输入连接符").apply {
                    setPadding(0, dp(6), 0, 0)
                    gravity = Gravity.END
                    setTextColor(themeColor(R.color.text_hint))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                })
            } else {
                addView(bodyText("当前 CK 格式无法自动拆分，请直接编辑原始值").apply { setPadding(0, dp(8), 0, 0) })
                fieldInputs["full"] = EditText(context).apply {
                    hint = "请输入完整的 CK 值"
                    setText(originalCk)
                    setTextColor(themeColor(R.color.text_primary))
                    setHintTextColor(themeColor(R.color.text_hint))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                    background = android.graphics.drawable.GradientDrawable().apply {
                        setColor(themeColor(R.color.input_bg))
                        cornerRadius = dp(12).toFloat()
                        setStroke(dp(1), themeColor(R.color.input_border))
                    }
                    setPadding(dp(14), dp(10), dp(14), dp(10))
                    inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_MULTI_LINE
                    minLines = 4
                    maxLines = 6
                    isSingleLine = false
                    layoutParams = LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.MATCH_PARENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT
                    ).apply { bottomMargin = dp(10) }
                    id = View.generateViewId()
                }.also { addView(it) }

                addView(captionText("注意：请保留字段名和连接符格式").apply {
                    setTextColor(themeColor(R.color.text_hint))
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 11f)
                })
            }
        })

        root.addView(cardView().apply {
            addView(sectionTitle("保存预览"))
            previewText = TextView(this@CkEditActivity).apply {
                text = buildPreview()
                setTextColor(themeColor(R.color.text_secondary))
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 12f)
                setLineSpacing(0f, 1.5f)
                background = android.graphics.drawable.GradientDrawable().apply {
                    setColor(themeColor(R.color.chip_bg))
                    cornerRadius = dp(10).toFloat()
                }
                setPadding(dp(14), dp(14), dp(14), dp(14))
                setTypeface(Typeface.MONOSPACE)
            }
            addView(previewText)

            fieldInputs.values.forEach { input ->
                input.addTextChangedListener(object : android.text.TextWatcher {
                    override fun beforeTextChanged(s: CharSequence?, start: Int, count: Int, after: Int) {}
                    override fun onTextChanged(s: CharSequence?, start: Int, before: Int, count: Int) {}
                    override fun afterTextChanged(s: android.text.Editable?) {
                        previewText.text = buildPreview()
                    }
                })
            }
        })

        root.addView(primaryButton("确认修改") {
            submitUpdate()
        }.apply {
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT
            ).apply { topMargin = dp(6); bottomMargin = dp(16) }
        })

        scroll.addView(root)
        setContentView(scroll)

        if (isProtocolActivity && parsedFields != null && visibleFields.isNotEmpty()) {
            val protoKey = ProtocolActivityPicker.firstTemplateFieldKey(ckTemplate)
            val selectedRef = protoKey?.let { parsedFields[it] }.orEmpty()
            lifecycleScope.launch {
                val options = runCatching {
                    AppServices.portalRepository.fetchProtocolAccountOptions(activityId, projectRemark)
                }.getOrElse { emptyList() }
                ProtocolActivityPicker.mount(
                    context = this@CkEditActivity,
                    container = fieldsCard,
                    insertIndex = 1,
                    options = options,
                    template = ckTemplate,
                    selectedFillRef = selectedRef,
                ) { fillRef ->
                    protoKey?.let { fieldInputs[it]?.setText(fillRef) }
                    previewText.text = buildPreview()
                }
                protoKey?.let { key ->
                    fieldInputs[key]?.visibility = android.view.View.GONE
                }
            }
        }
    }

    private fun getCkTemplateFields(template: String): List<String> {
        val fields = mutableListOf<String>()
        val seen = mutableSetOf<String>()
        val regex = Regex("\\{\\{\\.([^}]+)\\}\\}")
        regex.findAll(template).forEach { match ->
            val key = match.groupValues[1]
            if (!seen.contains(key)) {
                seen.add(key)
                fields.add(key)
            }
        }
        return fields
    }

    private fun splitCkValueByTemplate(template: String, value: String): Map<String, String>? {
        val fieldKeys = getCkTemplateFields(template)
        if (fieldKeys.isEmpty()) return null

        val source = value
        val placeholders = mutableListOf<Triple<String, String, Int>>()
        var cursor = 0

        val regex = Regex("\\{\\{\\.([^}]+)\\}\\}")
        regex.findAll(template).forEach { match ->
            val key = match.groupValues[1]
            val startIdx = match.range.first
            val endIdx = match.range.last + 1
            val prefix = template.substring(cursor, startIdx)
            placeholders.add(Triple(key, prefix, endIdx))
            cursor = endIdx
        }

        val suffix = template.substring(cursor)
        var pos = 0
        val result = mutableMapOf<String, String>()

        for (i in placeholders.indices) {
            val (key, prefixStr, _) = placeholders[i]

            if (prefixStr.isNotEmpty()) {
                if (!source.startsWith(prefixStr, pos)) return null
                pos += prefixStr.length
            }

            val nextPrefix = if (i + 1 < placeholders.size) {
                placeholders[i + 1].second
            } else {
                suffix
            }

            var end = source.length
            if (nextPrefix.isNotEmpty()) {
                end = source.indexOf(nextPrefix, pos)
                if (end < 0) return null
            }

            result[key] = source.substring(pos, end)
            pos = end
        }

        if (suffix.isNotEmpty()) {
            if (!source.startsWith(suffix, pos)) return null
            pos += suffix.length
        }

        if (pos != source.length) return null
        return result
    }

    private fun buildPreview(): String {
        if (fieldInputs.containsKey("full")) {
            return fieldInputs["full"]?.text?.toString()?.trim() ?: "(空)"
        }

        if (ckTemplate.isBlank() || fieldNames.isEmpty()) return "(空)"

        var result = ckTemplate
        for ((key, input) in fieldInputs) {
            result = result.replace("{{.$key}}", input.text.toString().trim())
        }
        return result.trim().ifEmpty { "(空)" }
    }

    private fun submitUpdate() {
        if (submitting) return

        val newCk = buildPreview()
        if (newCk.isBlank() || newCk == "(空)") {
            alert("CK 内容不能为空")
            return
        }

        if (!fieldInputs.containsKey("full")) {
            val emptyField = fieldInputs.entries.find { it.value.text.isNullOrBlank() }
            if (emptyField != null) {
                alert("${emptyField.key} 不能为空")
                emptyField.value.requestFocus()
                return
            }
        }

        submitting = true
        lifecycleScope.launch {
            runCatching {
                AppServices.portalRepository.updateProject(activityId, projectRemark, newCk)
            }
                .onSuccess { msg ->
                    alert(msg, "修改成功") { finish() }
                }
                .onFailure { error ->
                    alert(com.goudong.jd.ui.common.sanitizeErrorMessage(error.message), "错误")
                }
            submitting = false
        }
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    companion object {
        private const val EXTRA_REMARK = "remark"
        private const val EXTRA_ACTIVITY_ID = "activityId"
        private const val EXTRA_CK_VALUE = "ckValue"
        private const val EXTRA_CK_TEMPLATE = "ckTemplate"
        private const val EXTRA_INPUT_FIELDS = "inputFields"
        private const val EXTRA_IS_PROTOCOL = "isProtocolActivity"

        fun intent(
            context: android.content.Context,
            remark: String,
            activityId: String,
            ckValue: String,
            ckTemplate: String = "",
            inputFields: List<PortalActivityField> = emptyList(),
            isProtocolActivity: Boolean = false,
        ): Intent {
            return Intent(context, CkEditActivity::class.java).apply {
                putExtra(EXTRA_REMARK, remark)
                putExtra(EXTRA_ACTIVITY_ID, activityId)
                putExtra(EXTRA_CK_VALUE, ckValue)
                putExtra(EXTRA_CK_TEMPLATE, ckTemplate)
                putExtra(EXTRA_INPUT_FIELDS, ArrayList(inputFields))
                putExtra(EXTRA_IS_PROTOCOL, isProtocolActivity)
            }
        }
    }
}
