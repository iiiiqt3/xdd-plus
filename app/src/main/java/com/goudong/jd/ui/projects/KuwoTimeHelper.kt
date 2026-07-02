package com.goudong.jd.ui.projects

import java.util.Calendar
import java.util.TimeZone

object KuwoTimeHelper {
    val withdrawHours = listOf(0, 9, 13, 17, 20)

    data class BeijingTime(val hour: Int, val min: Int, val sec: Int, val totalMs: Long)
    data class NextWithdrawInfo(val hour: Int, val inWindow: Boolean, val diffMin: Int)

    fun getBeijingTime(): BeijingTime {
        val cal = Calendar.getInstance(TimeZone.getTimeZone("Asia/Shanghai"))
        val h = cal.get(Calendar.HOUR_OF_DAY)
        val m = cal.get(Calendar.MINUTE)
        val s = cal.get(Calendar.SECOND)
        val ms = cal.get(Calendar.MILLISECOND)
        return BeijingTime(h, m, s, (h * 3600L + m * 60L + s) * 1000L + ms)
    }

    /** 到目标整点的分钟差（支持跨午夜，如 23:57 → 00:00） */
    fun minutesUntilHour(targetHour: Int, hour: Int, min: Int): Int {
        var diffMin = targetHour * 60 - (hour * 60 + min)
        if (diffMin < 0) diffMin += 24 * 60
        return diffMin
    }

    /**
     * 到目标整点剩余毫秒；整点后 graceAfterMin 分钟内返回 0（执行窗口），避免误判为明天。
     */
    fun remainingMsUntilHour(targetHour: Int, graceAfterMin: Int = 30): Long {
        val bj = getBeijingTime()
        val targetMs = targetHour * 3600L * 1000L
        var remaining = targetMs - bj.totalMs
        if (remaining <= 0) {
            val pastMs = -remaining
            if (pastMs <= graceAfterMin * 60L * 1000L) {
                return 0L
            }
            remaining += 24L * 3600L * 1000L
        }
        return remaining.coerceAtLeast(0L)
    }

    fun getNextWithdrawInfo(): NextWithdrawInfo {
        val bj = getBeijingTime()
        for (h in withdrawHours) {
            val diffMin = minutesUntilHour(h, bj.hour, bj.min)
            if (diffMin > 0 && diffMin <= 4) {
                return NextWithdrawInfo(h, inWindow = true, diffMin = diffMin)
            }
            if (diffMin == 0) {
                return NextWithdrawInfo(h, inWindow = true, diffMin = 0)
            }
        }
        for (h in withdrawHours) {
            val diffMin = minutesUntilHour(h, bj.hour, bj.min)
            if (diffMin > 4) {
                return NextWithdrawInfo(h, inWindow = false, diffMin = diffMin)
            }
        }
        val nextDayMin = minutesUntilHour(withdrawHours.first(), bj.hour, bj.min)
        return NextWithdrawInfo(withdrawHours.first(), inWindow = false, diffMin = nextDayMin)
    }

    fun formatClock(h: Int, m: Int, s: Int): String =
        "%02d:%02d:%02d".format(h, m, s)

    fun formatHour(h: Int): String = "%02d:00".format(h)
}
