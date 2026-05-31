package com.goudong.jd.data.network

import android.annotation.SuppressLint
import android.content.Context
import android.os.Build
import android.provider.Settings
import java.security.MessageDigest
import java.util.UUID
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

object SignatureHelper {
    private const val SECRET = "G0uD0ng@2024#S1gn@ture!K3y"
    private const val SALT = "x9D\$kL2mN#pQ7rT5"
    private const val VERSION = "v2"
    private const val APP_VERSION = "2.0.0"

    fun getHeaders(context: Context, path: String): Map<String, String> {
        val timestamp = (System.currentTimeMillis() / 1000).toString()
        val nonce = generateNonce(16)
        val deviceId = getDeviceId(context)
        val signature = generateSignature(timestamp, nonce, deviceId, APP_VERSION, path)

        return mapOf(
            "X-Sign-Timestamp" to timestamp,
            "X-Sign-Nonce" to nonce,
            "X-Sign-DeviceID" to deviceId,
            "X-Sign-Value" to signature,
            "X-Sign-Version" to VERSION,
            "X-App-Version" to APP_VERSION
        )
    }

    private fun generateSignature(
        timestamp: String,
        nonce: String,
        deviceId: String,
        appVersion: String,
        path: String
    ): String {
        val round1Input = "$timestamp|$nonce|$deviceId|$appVersion|$path|$SALT"
        val round1 = sha256(round1Input)

        val round2 = hmacSha256(round1, SECRET)

        val reversedTimestamp = timestamp.reversed()
        val reversedNonce = nonce.reversed()
        val round3Input = "$round2$reversedTimestamp$reversedNonce"
        return sha256(round3Input)
    }

    private fun sha256(input: String): String {
        val digest = MessageDigest.getInstance("SHA-256")
        val hash = digest.digest(input.toByteArray(Charsets.UTF_8))
        return hash.joinToString("") { "%02x".format(it) }
    }

    private fun hmacSha256(message: String, secret: String): String {
        val mac = Mac.getInstance("HmacSHA256")
        val secretKey = SecretKeySpec(secret.toByteArray(Charsets.UTF_8), "HmacSHA256")
        mac.init(secretKey)
        val hash = mac.doFinal(message.toByteArray(Charsets.UTF_8))
        return hash.joinToString("") { "%02x".format(it) }
    }

    private fun generateNonce(length: Int): String {
        val charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
        return (1..length).map { charset.random() }.joinToString("")
    }

    @SuppressLint("HardwareIds")
    private fun getDeviceId(context: Context): String {
        return try {
            Settings.Secure.getString(context.contentResolver, Settings.Secure.ANDROID_ID)
                ?: UUID.randomUUID().toString()
        } catch (e: Exception) {
            UUID.randomUUID().toString()
        }
    }
}
