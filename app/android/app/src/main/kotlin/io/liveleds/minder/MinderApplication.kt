package io.liveleds.minder

import android.app.Application

class MinderApplication : Application() {
    override fun onCreate() {
        super.onCreate()
        MinderNotifications.initialize(this)
    }
}
