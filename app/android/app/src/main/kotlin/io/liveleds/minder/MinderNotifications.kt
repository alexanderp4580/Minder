package io.liveleds.minder

import android.Manifest
import android.app.*
import android.content.*
import android.content.pm.PackageManager
import android.graphics.Color
import android.net.Uri
import android.os.Build
import com.google.firebase.FirebaseApp
import com.google.firebase.FirebaseOptions
import com.google.firebase.messaging.FirebaseMessaging
import org.json.JSONObject
import java.util.UUID

object MinderNotifications {
    const val CHANNEL = "minder_reminders"
    fun prefs(context: Context) = context.getSharedPreferences("minder_push", Context.MODE_PRIVATE)
    fun initialize(context: Context): Boolean {
        val raw = prefs(context).getString("config", null) ?: return false
        return try {
            val config = JSONObject(raw)
            if (FirebaseApp.getApps(context).isEmpty()) {
                FirebaseApp.initializeApp(context, FirebaseOptions.Builder()
                    .setApplicationId(config.getString("app_id"))
                    .setApiKey(config.getString("api_key"))
                    .setProjectId(config.getString("project_id"))
                    .setGcmSenderId(config.getString("sender_id")).build())
            }
            true
        } catch (_: Exception) { false }
    }
    fun channel(context: Context) {
        if (Build.VERSION.SDK_INT >= 26) {
            val manager = context.getSystemService(NotificationManager::class.java)
            val channel = NotificationChannel(CHANNEL, "Reminders", NotificationManager.IMPORTANCE_HIGH)
            channel.description = "Task reminders repeat until you act."
            channel.enableVibration(true)
            channel.lockscreenVisibility = Notification.VISIBILITY_PRIVATE
            manager.createNotificationChannel(channel)
        }
    }
    fun granted(context: Context): Boolean {
        val manager = context.getSystemService(NotificationManager::class.java)
        if (!manager.areNotificationsEnabled()) return false
        if (Build.VERSION.SDK_INT >= 26 && manager.getNotificationChannel(CHANNEL)?.importance == NotificationManager.IMPORTANCE_NONE) return false
        return Build.VERSION.SDK_INT < 33 || context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED
    }
    fun deviceID(context: Context): String {
        val prefs = prefs(context)
        prefs.getString("device_id", null)?.let { return it }
        val id = UUID.randomUUID().toString()
        prefs.edit().putString("device_id", id).apply()
        return id
    }
    private fun action(context: Context, id: String, name: String): PendingIntent {
        val intent = Intent(context, MainActivity::class.java)
            .setAction("io.liveleds.minder.NOTIFICATION")
            .setData(Uri.parse("minder-notification://$id/$name"))
            .putExtra("reminder_id", id).putExtra("reminder_action", name)
            .addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP or Intent.FLAG_ACTIVITY_CLEAR_TOP)
        return PendingIntent.getActivity(context, 0, intent, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
    }
    fun show(context: Context, data: Map<String, String>) {
        val id = data["reminder_id"] ?: return
        if (!id.matches(Regex("^[a-zA-Z0-9]+$"))) return
        if (prefs(context).getBoolean("handled_$id", false)) return
        channel(context)
        if (!granted(context)) return
        val title = data["title"]?.take(200) ?: "Minder reminder"
        val content = if (data["late"] == "true") "Late reminder. Choose Done, Snooze or Open." else "Choose Done, Snooze or Open."
        val builder = if (Build.VERSION.SDK_INT >= 26) Notification.Builder(context, CHANNEL) else Notification.Builder(context)
        val notification = builder.setSmallIcon(R.drawable.ic_minder_notification)
            .setColor(Color.rgb(224, 18, 26)).setContentTitle(title).setContentText(content)
            .setStyle(Notification.BigTextStyle().bigText(content))
            .setVisibility(Notification.VISIBILITY_PRIVATE).setCategory(Notification.CATEGORY_REMINDER)
            .setPriority(Notification.PRIORITY_HIGH).setAutoCancel(false).setOnlyAlertOnce(false)
            .addAction(Notification.Action.Builder(null, "Done", action(context, id, "done")).build())
            .addAction(Notification.Action.Builder(null, "Snooze 15m", action(context, id, "snooze15")).build())
            .addAction(Notification.Action.Builder(null, "Open", action(context, id, "ack")).build())
            .build()
        context.getSystemService(NotificationManager::class.java).notify(id, 1, notification)
    }
    fun capture(context: Context, intent: Intent?) {
        if (intent?.action != "io.liveleds.minder.NOTIFICATION") return
        val id = intent.getStringExtra("reminder_id") ?: return
        val action = intent.getStringExtra("reminder_action") ?: return
        if (!id.matches(Regex("^[a-zA-Z0-9]+$")) || action !in listOf("done", "ack", "snooze15")) return
        prefs(context).edit().putString("pending", JSONObject(mapOf("id" to id, "action" to action)).toString()).apply()
    }
    fun token(context: Context, complete: (Map<String, Any>) -> Unit) {
        if (!granted(context) || !initialize(context)) {
            complete(mapOf("granted" to false)); return
        }
        FirebaseMessaging.getInstance().token.addOnCompleteListener { task ->
            val result = mutableMapOf<String, Any>("granted" to true, "device_id" to deviceID(context))
            if (task.isSuccessful) result["token"] = task.result
            complete(result)
        }
    }
}
