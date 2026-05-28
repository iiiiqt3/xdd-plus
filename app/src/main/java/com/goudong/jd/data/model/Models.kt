package com.goudong.jd.data.model

import com.google.gson.JsonElement

object AppEnvironment {
    const val BASE_URL = "http://180.152.5.230:5701/"
    const val COIN_PURCHASE_URL = "http://180.152.5.230:8005/#/"
    const val GROUP_URL = "https://qm.qq.com/q/4gYwV6YzPW"
    const val MORE_WOOL_URL = "https://h5.lot-ml.com/ProductEn/Index/7ee6c54f2d550fab"
    const val NOTICE_URL = "https://gitee.com/feiniao520/notice/raw/master/notice"
    const val JD_LOGIN_URL = "https://plogin.m.jd.com/login/login?appid=300&returnurl=https%3A%2F%2Fwq.jd.com%2Fpassport%2FLoginRedirect%3Fstate%3D1101806886554%26returnurl%3Dhttps%253A%252F%252Fhome.m.jd.com%252FmyJd%252Fnewhome.action%253Fsceneval%253D2%2526ufc%253D%2526&source=wq_passport"
    const val UPDATE_CHECK_URL = "http://180.152.5.230:5701/static/update_apk/version.json"
}

data class ApiEnvelope<T>(
    val code: Int = -1,
    val msg: String? = null,
    val data: T? = null,
)

data class ApiError(
    override val message: String,
    val unauthorized: Boolean = false,
) : Exception(message)

data class PortalDashboard(
    val number: Long = 0L,
    val qq: String? = null,
    val wxid: String? = null,
    val coin: Int = 0,
    val nickname: String? = null,
    val accountId: Int? = null,
    val username: String? = null,
    val boundAt: String? = null,
    val lastLoginAt: String? = null,
    val availableCount: Int = 0,
    val projectCount: Int = 0,
    val joinedCount: Int = 0,
    val activeCount: Int = 0,
    val validCkCount: Int = 0,
    val expiringCount: Int = 0,
    val expiredCount: Int = 0,
    val notificationTotal: Long = 0L,
    val notificationUnread: Long = 0L,
    val checkedInToday: Boolean = false,
    val continuousDays: Int = 0,
    val nextCheckInBonus: Int = 0,
    val daysUntilNextCheckInBonus: Int = 0,
    val prayedToday: Boolean = false,
)

data class PortalProfile(
    val user: PortalUser? = null,
    val account: PortalAccount? = null,
)

data class PortalUser(
    val Number: Long? = null,
    val QQ: String? = null,
    val Wxid: String? = null,
    val Coin: Int? = null,
    val Class: String? = null,
    val Nickname: String? = null,
)

data class PortalAccount(
    val ID: Int? = null,
    val Username: String? = null,
    val UserNumber: Long? = null,
    val BoundAt: String? = null,
    val LastLoginAt: String? = null,
)

data class PortalActivityField(
    val key: String? = null,
    val prompt: String? = null,
    val required: Boolean? = null,
    val trimSpace: Boolean? = null,
    val timeoutSec: Int? = null,
    val errorMsg: String? = null,
) : java.io.Serializable

data class PortalActivity(
    val id: String? = null,
    val name: String? = null,
    val envKey: String? = null,
    val needCoin: Int? = null,
    val isMonthlyDeduct: Boolean? = null,
    val monthlyCoin: Int? = null,
    val isDailyDeduct: Boolean? = null,
    val dailyCoin: Int? = null,
    val minDays: Int? = null,
    val qingLongConfig: String? = null,
    val guide: String? = null,
    val inputFields: List<PortalActivityField>? = null,
    val ckTemplate: String? = null,
    val enabled: Boolean? = null,
) : java.io.Serializable

data class PortalProject(
    val activityId: String? = null,
    val activityName: String? = null,
    val envKey: String? = null,
    val envId: Int? = null,
    val envValue: String? = null,
    val qingLongConfig: String? = null,
    val remark: String? = null,
    val displayName: String? = null,
    val expireDate: String? = null,
    val status: Int? = null,
    val statusText: String? = null,
    val updatedAt: String? = null,
    val createdAt: String? = null,
    val isMonthlyDeduct: Boolean? = null,
    val monthlyCoin: Int? = null,
    val isDailyDeduct: Boolean? = null,
    val dailyCoin: Int? = null,
    val needCoin: Int? = null,
    val bizStatus: String? = null,
    val bizStatusText: String? = null,
    val daysLeft: Int? = null,
    val priceText: String? = null,
    val inputFields: List<PortalActivityField>? = null,
    val ckTemplate: String? = null,
)

data class PortalWechatStatus(
    val nickname: String? = null,
    val wxid: String? = null,
    val device: String? = null,
    val status: String? = null,
    val online: Boolean? = null,
    val loginTime: String? = null,
    val refreshTime: String? = null,
)

data class PortalWechatActionResult(
    val message: String? = null,
    val status: PortalWechatStatus? = null,
    val qrBase64: String? = null,
    val uuid: String? = null,
    val cost: Int? = null,
    val needPoll: Boolean? = null,
)

data class PortalWxDevice(
    val id: Int = 0,
    val wxid: String? = null,
    val nickname: String? = null,
    val device: String? = null,
    val status: String? = null,
    val online: Boolean? = null,
    val isPrimary: Boolean? = null,
    val loginTime: String? = null,
    val refreshTime: String? = null,
)

data class ResetInfoPayload(
    val username: String? = null,
)

data class PortalHomeSnapshot(
    val dashboard: PortalDashboard,
    val profile: PortalProfile,
    val wechatStatus: PortalWechatStatus? = null,
    val notifications: PortalNotificationPage? = null,
    val topNotifications: List<PortalNotification> = emptyList(),
)

data class PortalNotificationPage(
    val list: List<PortalNotification> = emptyList(),
    val total: Int = 0,
    val unread: Int = 0,
    val unreadStats: JsonElement? = null,
)

data class PortalNotification(
    val id: Int = 0,
    val title: String? = null,
    val content: String? = null,
    val category: String? = null,
    val source: String? = null,
    val channels: String? = null,
    val isRead: Boolean = false,
    val clickCount: Int = 0,
    val displayType: String? = null,
    val isTop: Boolean = false,
    val createdAt: String? = null,
    val readAt: String? = null,
) : java.io.Serializable

data class SubmitFeedbackPayload(
    val type: String,
    val title: String,
    val content: String,
    val contact: String = "",
)

data class UserInfoEnvelope(
    val code: Int = -1,
    val data: JsonElement? = null,
    val message: String? = null,
)
