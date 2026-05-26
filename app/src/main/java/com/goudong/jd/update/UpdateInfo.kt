package com.goudong.jd.update

import com.google.gson.annotations.SerializedName

data class UpdateInfo(
    @SerializedName("versionCode") val versionCode: Int,
    @SerializedName("versionName") val versionName: String,
    @SerializedName("forceUpdate") val forceUpdate: Boolean = true,
    @SerializedName("apkUrl") val apkUrl: String,
    @SerializedName("changelog") val changelog: String = "",
)
