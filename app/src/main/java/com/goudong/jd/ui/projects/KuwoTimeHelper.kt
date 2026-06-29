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

    fun getNextWithdrawInfo(): NextWithdrawInfo {
        val bj = getBeijingTime()
        val currentMin = bj.hour * 60 + bj.min
        for (h in withdrawHours) {
            val diffMin = h * 60 - currentMin
            if (diffMin > 0 && diffMin <= 4) {
                return NextWithdrawInfo(h, inWindow = true, diffMin = diffMin)
            }
            if (diffMin == 0) {
                return NextWithdrawInfo(h, inWindow = true, diffMin = 0)
            }
        }
        for (h in withdrawHours) {
            if (h * 60 > currentMin) {
                return NextWithdrawInfo(h, inWindow = false, diffMin = h * 60 - currentMin)
            }
        }
        val nextDayMin = (24 * 60 - currentMin) + withdrawHours.first() * 60
        return NextWithdrawInfo(withdrawHours.first(), inWindow = false, diffMin = nextDayMin)
    }

    fun formatClock(h: Int, m: Int, s: Int): String =
        "%02d:%02d:%02d".format(h, m, s)

    fun formatHour(h: Int): String = "%02d:00".format(h)
}
