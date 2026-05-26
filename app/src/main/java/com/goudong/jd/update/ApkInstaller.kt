package com.goudong.jd.update

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import android.widget.Toast
import androidx.core.content.FileProvider
import okhttp3.Call
import okhttp3.Callback
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import java.io.File
import java.io.IOException
import java.util.concurrent.TimeUnit

class ApkInstaller(private val context: Context) {

    private val mainHandler = Handler(Looper.getMainLooper())

    fun download(
        apkUrl: String,
        onProgress: (Int) -> Unit,
        onComplete: (File) -> Unit,
        onError: (String) -> Unit,
    ) {
        val client = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .build()

        val request = Request.Builder().url(apkUrl).build()

        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                mainHandler.post { onError(e.message ?: "下载失败") }
            }

            override fun onResponse(call: Call, response: Response) {
                if (!response.isSuccessful) {
                    mainHandler.post { onError("下载失败 (${response.code})") }
                    return
                }
                val body = response.body ?: run {
                    mainHandler.post { onError("下载失败：响应为空") }
                    return
                }
                val total = body.contentLength()
                val apkFile = File(context.cacheDir, "update.apk")
                try {
                    body.byteStream().use { input ->
                        apkFile.outputStream().use { output ->
                            val buf = ByteArray(8192)
                            var downloaded = 0L
                            var len: Int
                            while (input.read(buf).also { len = it } != -1) {
                                output.write(buf, 0, len)
                                downloaded += len
                                if (total > 0) {
                                    val pct = (downloaded * 100 / total).toInt().coerceIn(0, 100)
                                    mainHandler.post { onProgress(pct) }
                                }
                            }
                        }
                    }
                    mainHandler.post { onComplete(apkFile) }
                } catch (e: Exception) {
                    mainHandler.post { onError(e.message ?: "保存失败") }
                }
            }
        })
    }

    fun install(apkFile: File) {
        if (!apkFile.exists() || apkFile.length() <= 0L) {
            Toast.makeText(context, "安装包不存在或下载不完整", Toast.LENGTH_LONG).show()
            return
        }

        // Android 8.0+ 需要用户允许“安装未知应用”
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O && !context.packageManager.canRequestPackageInstalls()) {
            Toast.makeText(context, "请先允许本应用安装未知应用，授权后重新点击更新", Toast.LENGTH_LONG).show()
            val settingsIntent = Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES).apply {
                data = Uri.parse("package:${context.packageName}")
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            context.startActivity(settingsIntent)
            return
        }

        val intent = Intent(Intent.ACTION_VIEW)
        intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)

        val uri: Uri = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            FileProvider.getUriForFile(context, "${context.packageName}.fileProvider", apkFile)
        } else {
            Uri.fromFile(apkFile)
        }

        intent.setDataAndType(uri, "application/vnd.android.package-archive")
        context.startActivity(intent)
    }
}
