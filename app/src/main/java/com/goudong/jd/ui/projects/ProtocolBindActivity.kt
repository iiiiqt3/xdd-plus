package com.goudong.jd.ui.projects

import android.graphics.Color
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.data.model.PortalProtocolBinding
import com.goudong.jd.data.model.PortalProtocolBindQuota
import com.goudong.jd.data.model.PortalWxDevice
import com.goudong.jd.data.model.PortalYybAccount
import com.goudong.jd.data.session.YybAccountStore
import com.goudong.jd.ui.common.bodyText
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.primaryButton
import com.goudong.jd.ui.common.themeColor
import com.goudong.jd.ui.common.toast
import com.goudong.jd.R
import kotlinx.coroutines.launch

class ProtocolBindActivity : AppCompatActivity() {

    private val root = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setBackgroundColor(Color.parseColor("#F8FAFC"))
    }

    private val contentHost = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        setPadding(dp(16), dp(12), dp(16), dp(24))
    }

    private val loadingBar = ProgressBar(this).apply {
        isIndeterminate = true
    }

    private var bindings: List<PortalProtocolBinding> = emptyList()
    private var quota: PortalProtocolBindQuota? = null
    private var wxDevices: List<PortalWxDevice> = emptyList()
    private var yybAccounts: List<PortalYybAccount> = emptyList()
    private var selectedWx: PortalWxDevice? = null
    private var selectedYyb: PortalYybAccount? = null

    private lateinit var wxPickBtn: TextView
    private lateinit var yybPickBtn: TextView
    private lateinit var boundListHost: LinearLayout

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        supportActionBar?.apply {
            title = "协议双绑"
            setDisplayHomeAsUpEnabled(true)
        }
        setContentView(root)
        root.addView(loadingBar)
        root.addView(android.widget.ScrollView(this).apply {
            addView(contentHost)
        })
        buildHeader()
        buildFormCard()
        buildBoundSection()
        loadAll()
    }

    override fun onSupportNavigateUp(): Boolean {
        finish()
        return true
    }

    private fun buildHeader() {
        contentHost.addView(cardView().apply {
            setPadding(dp(16), dp(14), dp(16), dp(14))
            addView(TextView(context).apply {
                text = "🔗 微信 wxid ↔ 应用宝 openid"
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 16f)
                setTypeface(typeface, Typeface.BOLD)
                setTextColor(themeColor(R.color.text_primary))
            })
            addView(bodyText(
                "绑定后青龙脚本原提交 CK 的微信 wxid，网关将自动路由微信协议 wxid 至应用宝请求协议 code；如果你之前没使用微信协议仅使用应用宝时可忽略本功能。绑定关系显示在下方账号卡片内。"
            ).apply {
                setPadding(0, dp(8), 0, 0)
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
                setLineSpacing(0f, 1.5f)
            })
        })
    }

    private fun buildFormCard() {
        contentHost.addView(cardView().apply {
            setPadding(dp(16), dp(14), dp(16), dp(14))
            layoutParams = LinearLayout.LayoutParams(
                LinearLayout.LayoutParams.MATCH_PARENT,
                LinearLayout.LayoutParams.WRAP_CONTENT,
            ).apply { topMargin = dp(12) }
            addView(captionText("新建双绑").apply {
                setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                setTypeface(typeface, Typeface.BOLD)
            })
            fun pickField(placeholder: String): TextView {
                return TextView(context).apply {
                    text = placeholder
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                    setTextColor(themeColor(R.color.text_primary))
                    setPadding(dp(12), dp(12), dp(12), dp(12))
                    background = GradientDrawable().apply {
                        setColor(themeColor(R.color.chip_bg))
                        cornerRadius = dp(10).toFloat()
                    }
                    layoutParams = LinearLayout.LayoutParams(
                        LinearLayout.LayoutParams.MATCH_PARENT,
                        LinearLayout.LayoutParams.WRAP_CONTENT,
                    ).apply { topMargin = dp(10) }
                }
            }
            wxPickBtn = pickField("选择微信 wxid")
            yybPickBtn = pickField("选择应用宝 openid")
            wxPickBtn.setOnClickListener { showWxPicker() }
            yybPickBtn.setOnClickListener { showYybPicker() }
            addView(wxPickBtn)
            addView(yybPickBtn)
            addView(primaryButton("建立双绑").apply {
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                ).apply { topMargin = dp(14) }
                setOnClickListener { submitBind() }
            })
        })
    }

    private fun buildBoundSection() {
        contentHost.addView(captionText("已绑定配对").apply {
            setPadding(dp(4), dp(16), 0, dp(8))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 13f)
            setTypeface(typeface, Typeface.BOLD)
            setTextColor(themeColor(R.color.text_secondary))
        })
        boundListHost = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL }
        contentHost.addView(boundListHost)
    }

    private fun loadAll() {
        loadingBar.visibility = View.VISIBLE
        lifecycleScope.launch {
            runCatching {
                bindings = AppServices.portalRepository.fetchProtocolBindings()
                quota = AppServices.portalRepository.fetchProtocolBindQuota()
                wxDevices = AppServices.portalRepository.fetchWxDevices()
                yybAccounts = YybAccountStore.accounts.ifEmpty {
                    AppServices.portalRepository.fetchYybStatus(false).accounts.orEmpty()
                }
            }.onFailure {
                toast(it.message ?: "加载失败")
                handlePortalError(it)
            }
            loadingBar.visibility = View.GONE
            renderQuota()
            renderBoundList()
            syncPickersEnabled()
        }
    }

    private fun renderQuota() {
        // 名额信息改在扫码时展示
    }

    private fun boundWxSet() = bindings.mapNotNull { it.wxWxid }.toSet()
    private fun boundOpenIdSet() = bindings.mapNotNull { it.yybOpenId }.toSet()

    private fun unboundWx() = wxDevices.filter { !boundWxSet().contains(it.wxid) }
    private fun unboundYyb() = yybAccounts.filter { !boundOpenIdSet().contains(it.openid) }

    private fun syncPickersEnabled() {
        val canBind = unboundWx().isNotEmpty() && unboundYyb().isNotEmpty()
        wxPickBtn.alpha = if (canBind) 1f else 0.5f
        yybPickBtn.alpha = if (canBind) 1f else 0.5f
    }

    private fun renderBoundList() {
        boundListHost.removeAllViews()
        if (bindings.isEmpty()) {
            boundListHost.addView(cardView().apply {
                addView(bodyText("暂无绑定，可在上方选择微信与应用宝账号建立双绑。").apply {
                    gravity = Gravity.CENTER
                    setTextColor(themeColor(R.color.text_muted))
                })
            })
            return
        }
        bindings.forEach { b ->
            val wx = wxDevices.firstOrNull { it.wxid == b.wxWxid }
            val yyb = yybAccounts.firstOrNull { it.openid == b.yybOpenId }
            boundListHost.addView(cardView().apply {
                setPadding(dp(14), dp(12), dp(14), dp(12))
                layoutParams = LinearLayout.LayoutParams(
                    LinearLayout.LayoutParams.MATCH_PARENT,
                    LinearLayout.LayoutParams.WRAP_CONTENT,
                ).apply { bottomMargin = dp(8) }
                addView(TextView(context).apply {
                    text = "微信 · ${wx?.nickname ?: b.nickname ?: "设备"}"
                    setTypeface(typeface, Typeface.BOLD)
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                })
                addView(captionText(shortenId(b.wxWxid)).apply {
                    setPadding(0, dp(4), 0, 0)
                })
                addView(TextView(context).apply {
                    text = "↕"
                    gravity = Gravity.CENTER
                    setTextColor(themeColor(R.color.text_muted))
                    setPadding(0, dp(4), 0, dp(4))
                })
                addView(TextView(context).apply {
                    text = "应用宝 · ${yyb?.nickname ?: "账号"}"
                    setTypeface(typeface, Typeface.BOLD)
                    setTextSize(TypedValue.COMPLEX_UNIT_SP, 14f)
                })
                addView(captionText(shortenId(b.yybOpenId)).apply {
                    setPadding(0, dp(4), 0, dp(8))
                })
                addView(TextView(context).apply {
                    text = "解除绑定"
                    setTextColor(Color.parseColor("#DC2626"))
                    setTypeface(typeface, Typeface.BOLD)
                    setOnClickListener {
                        androidx.appcompat.app.AlertDialog.Builder(this@ProtocolBindActivity)
                            .setTitle("解除双绑")
                            .setMessage("确定解除该配对？")
                            .setNegativeButton("取消", null)
                            .setPositiveButton("解绑") { _, _ ->
                                lifecycleScope.launch {
                                    runCatching {
                                        AppServices.portalRepository.protocolUnbind(
                                            b.wxWxid.orEmpty(),
                                            b.yybOpenId.orEmpty(),
                                        )
                                    }.onSuccess {
                                        toast(it)
                                        selectedWx = null
                                        selectedYyb = null
                                        loadAll()
                                    }.onFailure {
                                        toast(it.message ?: "解绑失败")
                                    }
                                }
                            }
                            .show()
                    }
                })
            })
        }
    }

    private fun showWxPicker() {
        val items = unboundWx()
        if (items.isEmpty()) {
            toast("暂无可绑定的微信设备")
            return
        }
        val labels = items.map {
            val offline = if (it.online == true) "" else "（离线）"
            "${it.nickname ?: "微信设备"} · ${shortenId(it.wxid)}$offline"
        }.toTypedArray()
        androidx.appcompat.app.AlertDialog.Builder(this)
            .setTitle("选择微信 wxid")
            .setItems(labels) { _, which ->
                selectedWx = items[which]
                wxPickBtn.text = labels[which]
            }
            .show()
    }

    private fun showYybPicker() {
        val items = unboundYyb()
        if (items.isEmpty()) {
            toast("暂无可绑定的应用宝账号")
            return
        }
        val labels = items.map { acc ->
            val alive = (acc.status ?: "").lowercase() in listOf("alive", "online")
            val name = acc.nickname?.takeIf { it.isNotBlank() } ?: "应用宝账号"
            "$name · ${shortenId(acc.openid)}${if (alive) "" else "（失效）"}"
        }.toTypedArray()
        androidx.appcompat.app.AlertDialog.Builder(this)
            .setTitle("选择应用宝 openid")
            .setItems(labels) { _, which ->
                selectedYyb = items[which]
                yybPickBtn.text = labels[which]
            }
            .show()
    }

    private fun submitBind() {
        val wx = selectedWx
        val yyb = selectedYyb
        if (wx == null || yyb == null) {
            toast("请先选择微信和应用宝账号")
            return
        }
        lifecycleScope.launch {
            runCatching {
                AppServices.portalRepository.protocolBind(
                    wxWxid = wx.wxid.orEmpty(),
                    yybOpenId = yyb.openid.orEmpty(),
                    nickname = yyb.nickname.orEmpty(),
                )
            }.onSuccess {
                toast("绑定成功")
                selectedWx = null
                selectedYyb = null
                wxPickBtn.text = "选择微信 wxid"
                yybPickBtn.text = "选择应用宝 openid"
                loadAll()
            }.onFailure {
                toast(it.message ?: "绑定失败")
                handlePortalError(it)
            }
        }
    }

    private fun shortenId(id: String?): String {
        val s = id?.trim().orEmpty()
        if (s.length <= 17) return s
        return s.take(8) + "…" + s.takeLast(6)
    }
}
