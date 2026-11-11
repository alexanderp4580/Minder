package io.liveleds.minder

import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage

class MinderMessagingService : FirebaseMessagingService() {
    override fun onMessageReceived(message: RemoteMessage) {
        MinderNotifications.show(this, message.data)
    }
    override fun onNewToken(token: String) {
        // The authenticated app registers the current token whenever it resumes.
        MinderNotifications.prefs(this).edit().putBoolean("token_changed", true).apply()
    }
}
