package io.liveleds.minder

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent

/** Receives the Done and Snooze buttons. It runs the action without opening the app. */
class ReminderActionReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        MinderNotifications.capture(context, intent)
        val pending = goAsync()
        BackgroundAction.run(context.applicationContext) { ok ->
            if (!ok) MinderNotifications.failed(context)
            pending.finish()
        }
    }
}
