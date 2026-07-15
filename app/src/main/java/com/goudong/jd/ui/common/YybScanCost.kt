package com.goudong.jd.ui.common

data class ScanCostPreview(
    val text: String,
    val isFree: Boolean,
)

/** 按 scanLoginCost 展示：免费或扣除积分数，不展示库内名额等复杂文案。 */
fun formatScanCostPreview(cost: Int?, @Suppress("UNUSED_PARAMETER") hint: String?): ScanCostPreview {
    return when {
        cost != null && cost > 0 -> ScanCostPreview("本次扫码将扣除 $cost 积分", false)
        cost != null && cost == 0 -> ScanCostPreview("本次扫码免费，不扣除积分", true)
        else -> ScanCostPreview("", true)
    }
}

fun formatYybScanCostNote(cost: Int?, hint: String?): String? {
    val preview = formatScanCostPreview(cost, hint)
    return preview.text.ifBlank { null }
}

fun formatYybConfirmMessage(alreadyBound: Boolean, cost: Int): String = when {
    alreadyBound -> "续登录成功，未扣除积分"
    cost > 0 -> "扫码成功，已扣除 $cost 积分"
    else -> "扫码成功，账号已绑定"
}
