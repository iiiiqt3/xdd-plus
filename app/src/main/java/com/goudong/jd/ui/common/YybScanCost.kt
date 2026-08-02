package com.goudong.jd.ui.common

data class ScanCostPreview(
    val text: String,
    val isFree: Boolean,
)

/** 优先使用后端 scanCostHint（含续登免费/新增扣费等说明）。 */
fun formatScanCostPreview(cost: Int?, hint: String?): ScanCostPreview {
    val trimmed = hint?.trim().orEmpty()
    if (trimmed.isNotEmpty()) {
        val isFree = when {
            cost != null && cost == 0 -> true
            cost != null && cost > 0 -> false
            else -> trimmed.contains("免费") && !trimmed.contains("扣除")
        }
        return ScanCostPreview(trimmed, isFree)
    }
    return when {
        cost != null && cost > 0 -> ScanCostPreview("本次扫码将扣除 $cost 积分", false)
        cost != null && cost == 0 -> ScanCostPreview("本次扫码免费，不扣除积分", true)
        else -> ScanCostPreview("", true)
    }
}

/** 积分说明 + 地区短提醒，用于积分预览区与扫码页。 */
fun formatYybScanNotes(cost: Int?, hint: String?, regionQrHint: String?): String? {
    val costText = formatScanCostPreview(cost, hint).text.trim()
    val regionText = regionQrHint?.trim().orEmpty().ifEmpty {
        "若微信提示「异地登录」，说明地区不匹配，有效期可能仅 1 天，请重新选择与你所在地一致的省/市。"
    }
    return when {
        costText.isNotEmpty() && regionText.isNotEmpty() -> "$costText\n\n$regionText"
        costText.isNotEmpty() -> costText
        regionText.isNotEmpty() -> regionText
        else -> null
    }
}

fun formatYybScanCostNote(cost: Int?, hint: String?): String? {
    return formatScanCostPreview(cost, hint).text.ifBlank { null }
}

fun formatYybConfirmMessage(alreadyBound: Boolean, cost: Int): String = when {
    alreadyBound -> "续登录成功，未扣除积分"
    cost > 0 -> "扫码成功，已扣除 $cost 积分"
    else -> "扫码成功，账号已绑定"
}

/** 免代理地区前缀 + 后端地区提示 */
fun formatYybScanRegionHint(bypassRegionName: String?, regionHint: String?): String {
    val base = regionHint?.trim().takeUnless { it.isNullOrEmpty() }
        ?: "请务必选择与你实际所在地一致的省/市。地区正确时有效期约 30 天；若微信提示「异地登录」，有效期可能仅 1 天。"
    val bypass = bypassRegionName?.trim().orEmpty()
    return if (bypass.isNotEmpty()) {
        "「$bypass」等地区可免代理直连；$base"
    } else {
        base
    }
}
