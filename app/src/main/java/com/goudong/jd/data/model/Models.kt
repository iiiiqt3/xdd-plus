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
    val canCheckIn: Boolean = false,
    val canCheckInMessage: String? = null,
    val todayCheckInCount: Int = 0,
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
    val category: String? = null,
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
    @com.google.gson.annotations.SerializedName("isProtocolActivity")
    val isProtocolActivity: Boolean? = null,
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
    val grantExpireDate: String? = null,
    val bizStatus: String? = null,
    val bizStatusText: String? = null,
    val daysLeft: Int? = null,
    val priceText: String? = null,
    val inputFields: List<PortalActivityField>? = null,
    val ckTemplate: String? = null,
    @com.google.gson.annotations.SerializedName("isProtocolActivity")
    val isProtocolActivity: Boolean? = null,
)

data class ProtocolAccountOption(
    val id: String? = null,
    val label: String? = null,
    val nickname: String? = null,
    val mode: String? = null,
    val wxid: String? = null,
    val openid: String? = null,
    @com.google.gson.annotations.SerializedName("fillRef")
    val fillRef: String? = null,
    @com.google.gson.annotations.SerializedName("usedInActivity")
    val usedInActivity: Boolean? = null,
    val selectable: Boolean? = null,
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

data class CoinLog(
    val id: Int = 0,
    val amount: Int = 0,
    val balanceAfter: Int = 0,
    val type: String? = null,
    val detail: String? = null,
    val source: String? = null,
    val sourceLabel: String? = null,
    val sourceTagCls: String? = null,
    val clientSource: String? = null,
    val clientPlatform: String? = null,
    val createdAt: String? = null,
)

data class ResetInfoPayload(
    val username: String? = null,
)

data class PortalHomePayload(
    val dashboard: PortalDashboard,
    val profile: PortalProfile,
)

data class PortalHomeSnapshot(
    val dashboard: PortalDashboard,
    val profile: PortalProfile,
    val wechatStatus: PortalWechatStatus? = null,
    val notifications: PortalNotificationPage? = null,
    val topNotifications: List<PortalNotification> = emptyList(),
    val wxDevices: List<PortalWxDevice> = emptyList(),
    val yybStatus: PortalYybStatus? = null,
    val protocolBindings: List<PortalProtocolBinding> = emptyList(),
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

data class PortalJdAccount(
    val index: Int = 0,
    val pin: String? = null,
    val nickname: String? = null,
    val statusText: String? = null,
    val valid: Boolean = false,
)

data class PortalJdSmsVerifyResult(
    val message: String? = null,
    val queryResult: String? = null,
    val needIdVerify: Boolean = false,
)

data class PortalJdWxDevice(
    val index: Int = 0,
    val wxid: String? = null,
    val nickname: String? = null,
    val device: String? = null,
    val serverType: String? = null,
    val jdNickname: String? = null,
)

data class PortalJdWxRefreshResult(
    val success: Int = 0,
    val fail: Int = 0,
    val details: List<String>? = null,
    val needRiskVerify: Boolean = false,
    val riskUrl: String? = null,
    val riskMsg: String? = null,
)

data class PortalYybAccount(
    val bindingId: Long = 0,
    val yybAccountId: Long = 0,
    val openid: String? = null,
    val uin: Long? = null,
    val nickname: String? = null,
    val avatarUrl: String? = null,
    val status: String? = null,
    val lastCheckedAt: Long? = null,
    val createdAt: Long = 0,
    val loginAt: Long? = null,
    val expiresAt: Long? = null,
    val proxyRegionCode: String? = null,
    val proxyRegionName: String? = null,
)

data class PortalProtocolBinding(
    val id: Long = 0,
    val userNumber: Int = 0,
    val wxWxid: String? = null,
    val yybOpenId: String? = null,
    val nickname: String? = null,
)

data class PortalProtocolBindQuota(
    val onlineWxSlots: Int = 0,
    val yybAccounts: Int = 0,
    val boundPairs: Int = 0,
    val freeSlots: Int = 0,
    val scanLoginCost: Int? = null,
    val scanCostHint: String? = null,
)

data class PortalProxyConfig(
    val proxyEnabled: Boolean = false,
    val proxyAccountConfigured: Boolean = true,
    val proxyDefaultPackid: String? = null,
    val proxyBypassRegionName: String? = null,
)

data class PortalYybCheckSummary(
    val total: Int = 0,
    val alive: Int = 0,
    val dead: Int = 0,
    val failed: Int = 0,
    val cooldown: Int = 0,
    val message: String? = null,
)

data class PortalYybStatus(
    val enabled: Boolean = false,
    val ready: Boolean = false,
    val message: String? = null,
    val coin: Int? = null,
    val scanLoginCost: Int? = null,
    val maxAccounts: Int? = null,
    val accounts: List<PortalYybAccount>? = null,
    val checkSummary: PortalYybCheckSummary? = null,
)

data class PortalYybQrCreateResult(
    val sessionId: String? = null,
    val status: String? = null,
    val imageBase64: String? = null,
    val scanLoginCost: Int? = null,
    val scanCostHint: String? = null,
)

data class PortalYybQrPollResult(
    val status: String? = null,
    val message: String? = null,
)

data class PortalYybConfirmResult(
    val account: PortalYybAccount? = null,
    val cost: Int = 0,
    val alreadyBound: Boolean = false,
)

data class PortalJdYybAccount(
    val index: Int = 0,
    val openid: String? = null,
    val nickname: String? = null,
    val status: String? = null,
    val jdNickname: String? = null,
    val proxyRegionCode: String? = null,
    val proxyRegionName: String? = null,
)

data class PortalJdTaskExecuteResult(
    val taskId: String? = null,
)

data class PortalJdTaskItem(
    val id: String? = null,
    val name: String? = null,
    val coin: Int = 0,
    val order: Int = 0,
)

data class PortalJdProxyStatus(
    val active: Boolean = false,
    val expireAt: String? = null,
    val monthlyCoin: Int = 0,
    val userCoin: Int = 0,
    val proxyReady: Boolean = false,
)

data class KuwoAccountInfo(
    val phone: String? = null,
    val password: String? = null,
)

data class KuwoCredentials(
    val phone: String? = null,
    val password: String? = null,
    val accounts: List<KuwoAccountInfo>? = null,
)

data class KuwoScheduleResult(
    val taskId: String? = null,
    val targetHour: Int? = null,
    val executeAt: String? = null,
    val status: String? = null,
    val reused: Boolean = false,
)

data class KuwoTaskLog(
    val time: String? = null,
    val level: String? = null,
    val message: String? = null,
    val proxyHost: String? = null,
)

data class KuwoWithdrawTask(
    val id: String? = null,
    val phone: String? = null,
    val quotaID: String? = null,
    val targetHour: Int? = null,
    val executeAt: String? = null,
    val status: String? = null,
    val immediate: Boolean = false,
    val smsFatal: Boolean? = null,
    val smsEditable: Boolean? = null,
    val logs: List<KuwoTaskLog>? = null,
)

data class UserInfoEnvelope(
    val code: Int = -1,
    val data: JsonElement? = null,
    val message: String? = null,
)
