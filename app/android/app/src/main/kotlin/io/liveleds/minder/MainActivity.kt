package io.liveleds.minder

import android.Manifest
import android.app.NotificationManager
import android.content.Intent
import android.os.Build
import android.os.Bundle
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import org.json.JSONObject

class MainActivity : FlutterActivity() {
    private var permissionResult: MethodChannel.Result? = null
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        MinderNotifications.capture(this, intent)
    }
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        MinderNotifications.capture(this, intent)
        flutterEngine?.dartExecutor?.binaryMessenger?.let { MethodChannel(it, "minder/notifications").invokeMethod("actionAvailable", null) }
    }
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "minder/notifications").setMethodCallHandler { call, result ->
            when (call.method) {
                "enable" -> {
                    val config = call.arguments as? Map<*, *>
                    if (config == null || listOf("app_id", "api_key", "project_id", "sender_id").any { (config[it] as? String).isNullOrBlank() }) {
                        result.error("config", "Firebase configuration is incomplete.", null)
                    } else {
                        MinderNotifications.prefs(this).edit().putString("config", JSONObject(config).toString()).apply()
                        if (!MinderNotifications.initialize(this)) {
                            result.error("config", "Firebase configuration is invalid.", null)
                        } else {
                            MinderNotifications.channel(this)
                            if (Build.VERSION.SDK_INT >= 33 && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != android.content.pm.PackageManager.PERMISSION_GRANTED) {
                                if (permissionResult != null) result.error("busy", "Permission request is open.", null)
                                else { permissionResult = result; requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 413) }
                            } else MinderNotifications.token(this) { result.success(it) }
                        }
                    }
                }
                "token" -> MinderNotifications.token(this) { result.success(it) }
                "takeAction", "completeAction" -> MinderNotifications.handleAction(this, call, result)
                "probe" -> {
                    if ((applicationInfo.flags and android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) == 0) result.notImplemented()
                    else {
                        MinderNotifications.channel(this)
                        if (Build.VERSION.SDK_INT >= 33 && checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != android.content.pm.PackageManager.PERMISSION_GRANTED) requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 414)
                        MinderNotifications.show(this, mapOf("reminder_id" to "probe123", "title" to "Minder notification check"))
                        result.success(null)
                    }
                }
                else -> result.notImplemented()
            }
        }
    }
    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, grantResults: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == 413) {
            val result = permissionResult; permissionResult = null
            if (result != null) MinderNotifications.token(this) { result.success(it) }
        }
    }
}
