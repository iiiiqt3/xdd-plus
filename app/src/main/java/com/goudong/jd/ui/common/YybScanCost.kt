package com.goudong.jd.ui.common

data class ScanCostPreview(
    val text: String,
    val isFree: Boolean,
)

/** 与后端 CalcYybScanLoginCost / scanCostHint 对齐，优先展示服务端文案（含库内 x/y）。 */
fun formatScanCostPreview(cost: Int?, hint: String?): ScanCostPreview {
    val trimmedHint = hint?.trim().orEmpty()
    if (trimmedHint.isNotEmpty()) {
        val isFree = cost == 0 ||
            trimmedHint.contains("免费") ||
            trimmedHint.contains("0积分") ||
            trimmedHint.contains("0 积分")
        return ScanCostPreview(trimmedHint, isFree)
    }
    return when {
        cost != null && cost > 0 -> ScanCostPreview("本次扫码将扣除 $cost 积分", false)
        cost != null && cost == 0 -> ScanCostPreview("本次扫码免费，不扣除积分", true)
        else -> ScanCostPreview("", true)
    }
}

fun formatYybScanCostNote(cost: Int?, hint: String?): String? {
    val preview = formatScanCostPreview(cost, hint)
    if (preview.text.isBlank()) return null
    return if (cost != null && cost > 0 && !preview.text.contains("确认登录后")) {
        "${preview.text}（确认登录后扣除）"
    } else {
        preview.text
    }
}

fun formatYybConfirmMessage(alreadyBound: Boolean, cost: Int): String = when {
    alreadyBound -> "续登录成功，未扣除积分"
    cost > 0 -> "扫码成功，已扣除 $cost 积分"
    else -> "扫码成功，账号已绑定（免费）"
}
