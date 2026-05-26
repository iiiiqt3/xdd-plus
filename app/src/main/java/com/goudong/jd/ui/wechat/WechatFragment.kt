package com.goudong.jd.ui.wechat

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.os.Bundle
import android.util.Base64
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import androidx.appcompat.app.AlertDialog
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.goudong.jd.AppServices
import com.goudong.jd.R
import com.goudong.jd.ui.common.actionGridTile
import com.goudong.jd.ui.common.captionText
import com.goudong.jd.ui.common.cardView
import com.goudong.jd.ui.common.dp
import com.goudong.jd.ui.common.handlePortalError
import com.goudong.jd.ui.common.heroCard
import com.goudong.jd.ui.common.makeScrollContainer
import com.goudong.jd.ui.common.toast
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

class WechatFragment : Fragment() {
    private lateinit var statusLabel: TextView
    private lateinit var detailLabel: TextView
    private var polling = false

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        val (scroll, root) = requireContext().makeScrollContainer()

        val statusCard = requireContext().cardView()
        statusCard.addView(requireContext().captionText("微信协议 · 当前状态"))
        statusLabel = requireContext().captionText("加载中...").apply { textSize = 16f ; setTextIsSelectable(true) }
        detailLabel = requireContext().captionText("正在获取微信状态...").apply { setTextIsSelectable(true) }
        statusCard.addView(statusLabel)
        statusCard.addView(detailLabel)
        root.addView(statusCard)

        root.addView(requireContext().cardView().apply {
            addView(requireContext().captionText("快捷操作"))
            addView(requireContext().actionGridTile(listOf(
                Triple("扫码登录", "微信扫码授权登录") { confirmAction("扫码登录", "确认开始微信扫码登录吗？扫码成功后将进入自动轮询状态。", "/api/portal/wx/scan-login", true) },
                Triple("重新登录", "重新获取登录凭证") { confirmAction("重新登录", "确认重新获取微信登录二维码吗？", "/api/portal/wx/relogin", false) },
                Triple("唤醒登录", "唤醒已登录设备") { confirmAction("唤醒登录", "确认执行微信唤醒登录吗？", "/api/portal/wx/wake-login", false) },
                Triple("登出设备", "安全退出当前设备") { confirmAction("登出设备", "确认登出当前微信设备吗？", "/api/portal/wx/logout", false) },
                Triple("删除设备", "清除设备数据后重新扫码") { confirmAction("删除设备", "确认删除当前微信设备数据吗？删除后需要重新扫码登录。", "/api/portal/wx/delete", false) }
            ), ContextCompat.getColor(requireContext(), R.color.brand_primary)))
        })

        return scroll
    }

    override fun onResume() {
        super.onResume()
        loadStatus()
    }

    override fun onPause() {
        super.onPause()
        polling = false
    }

    private fun loadStatus() {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.fetchWechatStatus() }
                .onSuccess { status ->
                    statusLabel.text = status.status ?: "未知状态"
                    detailLabel.text = "微信ID：${status.wxid ?: "-"}\n昵称：${status.nickname ?: "-"}\n设备：${status.device ?: "-"}\n登录时间：${status.loginTime ?: "-"}\n刷新时间：${status.refreshTime ?: "-"}"
                }
                .onFailure {
                    statusLabel.text = "未绑定或未在线"
                    detailLabel.text = com.goudong.jd.ui.common.sanitizeErrorMessage(it.message)
                    handlePortalError(it)
                }
        }
    }

    private fun confirmAction(title: String, message: String, path: String, deductCoin: Boolean) {
        AlertDialog.Builder(requireContext())
            .setTitle(title)
            .setMessage(message)
            .setNegativeButton("取消", null)
            .setPositiveButton("确认") { _, _ -> performAction(path, deductCoin) }
            .show()
    }

    private fun performAction(path: String, deductCoin: Boolean) {
        lifecycleScope.launch {
            runCatching { AppServices.portalRepository.performWechatAction(path) }
                .onSuccess { data ->
                    if (!data.qrBase64.isNullOrBlank() && !data.uuid.isNullOrBlank()) {
                        showQrCodeDialog(data.qrBase64, data.uuid, data.message ?: "请扫码", deductCoin)
                    } else {
                        toast(data.message ?: "操作成功")
                        loadStatus()
                    }
                }
                .onFailure { handlePortalError(it) }
        }
    }

    private fun showQrCodeDialog(base64: String, uuid: String, message: String, deductCoin: Boolean) {
        polling = true
        val ctx = requireContext()
        val dialogView = LinearLayout(ctx).apply {
            orientation = LinearLayout.VERTICAL
            gravity = android.view.Gravity.CENTER_HORIZONTAL
            setPadding(ctx.dp(24), ctx.dp(24), ctx.dp(24), ctx.dp(24))
        }
        val qrImage = ImageView(ctx).apply {
            layoutParams = LinearLayout.LayoutParams(ctx.dp(260), ctx.dp(260))
            scaleType = ImageView.ScaleType.FIT_CENTER
        }
        decodeQrImage(base64)?.let { qrImage.setImageBitmap(it) }
        val stateText = ctx.captionText(message).apply {
            gravity = android.view.Gravity.CENTER
            setPadding(0, ctx.dp(16), 0, 0)
        }
        dialogView.addView(qrImage)
        dialogView.addView(stateText)
        val dialog = AlertDialog.Builder(ctx)
            .setTitle("扫码登录")
            .setView(dialogView)
            .setNegativeButton("关闭") { _, _ -> polling = false }
            .create()
        dialog.show()
        lifecycleScope.launch {
            var failures = 0
            while (polling) {
                delay(3000)
                if (!polling) break
                runCatching { AppServices.portalRepository.pollWechatLogin(uuid, deductCoin) }
                    .onSuccess { result ->
                        failures = 0
                        if (result.needPoll == true) {
                            stateText.text = result.message ?: "等待扫码中"
                        } else {
                            polling = false
                            dialog.dismiss()
                            toast(result.message ?: "登录成功")
                            loadStatus()
                        }
                    }
                    .onFailure {
                        failures++
                        if (failures >= 4) {
                            polling = false
                            dialog.dismiss()
                            handlePortalError(it)
                        } else {
                            stateText.text = "服务连接中断，正在重试（$failures/3）"
                        }
                    }
            }
        }
    }

    private fun decodeQrImage(raw: String): Bitmap? {
        val trimmed = raw.trim()
        if (trimmed.isEmpty()) return null
        val payload = when {
            trimmed.startsWith("http") -> return null
            trimmed.contains("base64,") -> trimmed.substringAfter("base64,")
            trimmed.contains(",") && trimmed.contains("data:image") -> trimmed.substringAfter(",")
            else -> trimmed
        }
        val normalized = payload.replace("\n", "").replace("\r", "").replace(" ", "")
        val padding = normalized.length % 4
        val padded = if (padding == 0) normalized else normalized + "=".repeat(4 - padding)
        return runCatching {
            val bytes = Base64.decode(padded, Base64.DEFAULT)
            BitmapFactory.decodeByteArray(bytes, 0, bytes.size)
        }.getOrNull()
    }
}
