package io.liveleds.minder

import android.app.Activity
import android.content.Intent
import androidx.core.content.FileProvider
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import java.io.File
import java.util.TimeZone
import java.util.UUID

object AttachmentDownloads {
    fun handle(activity: Activity, call: MethodCall, result: MethodChannel.Result) {
        when (call.method) {
            "timezone" -> result.success(TimeZone.getDefault().id)
            "save" -> {
                val bytes = call.argument<ByteArray>("bytes")
                val name = call.argument<String>("name")?.substringAfterLast('/')?.substringAfterLast('\\')
                    ?.replace(Regex("[^a-zA-Z0-9._ -]"), "_")?.take(160)
                val mediaType = call.argument<String>("media_type") ?: "application/octet-stream"
                if (bytes == null || bytes.size > 20 * 1024 * 1024 || name.isNullOrBlank() || name == "." || name == "..") {
                    result.error("attachment", "The attachment is invalid.", null)
                    return
                }
                try {
                    val root = File(activity.cacheDir, "email-attachments").apply { mkdirs() }
                    root.listFiles()?.filter { it.lastModified() < System.currentTimeMillis() - 86400000 }?.forEach { it.deleteRecursively() }
                    val directory = File(root, UUID.randomUUID().toString()).apply { mkdirs() }
                    val file = File(directory, name)
                    file.writeBytes(bytes)
                    val uri = FileProvider.getUriForFile(activity, "${activity.packageName}.attachments", file)
                    val intent = Intent(Intent.ACTION_VIEW).apply {
                        setDataAndType(uri, mediaType)
                        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                    }
                    activity.startActivity(Intent.createChooser(intent, "Open attachment"))
                    result.success(null)
                } catch (_: Exception) {
                    result.error("attachment", "No app could open this attachment.", null)
                }
            }
            else -> result.notImplemented()
        }
    }
}
