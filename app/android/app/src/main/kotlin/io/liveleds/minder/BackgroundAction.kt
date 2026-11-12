package io.liveleds.minder

import android.content.Context
import android.os.Handler
import android.os.Looper
import io.flutter.FlutterInjector
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.embedding.engine.dart.DartExecutor
import io.flutter.plugin.common.MethodChannel

/** Runs the Dart `notificationAction` entry point in a headless engine, with a hard deadline. */
object BackgroundAction {
    private const val DEADLINE_MS = 9000L

    fun run(context: Context, finished: (Boolean) -> Unit) {
        val handler = Handler(Looper.getMainLooper())
        handler.post {
            var engine: FlutterEngine? = null
            var done = false
            val finish = { ok: Boolean ->
                if (!done) {
                    done = true
                    handler.removeCallbacksAndMessages(null)
                    engine?.destroy()
                    finished(ok)
                }
            }
            try {
                val loader = FlutterInjector.instance().flutterLoader()
                loader.startInitialization(context)
                loader.ensureInitializationComplete(context, null)
                val created = FlutterEngine(context)
                engine = created
                MethodChannel(created.dartExecutor.binaryMessenger, "minder/notifications").setMethodCallHandler { call, result ->
                    if (call.method == "backgroundFinished") {
                        result.success(null)
                        finish(call.argument<Boolean>("ok") == true)
                    } else if (!MinderNotifications.handleAction(context, call, result)) {
                        result.notImplemented()
                    }
                }
                created.dartExecutor.executeDartEntrypoint(DartExecutor.DartEntrypoint(loader.findAppBundlePath(), "notificationAction"))
                handler.postDelayed({ finish(false) }, DEADLINE_MS)
            } catch (e: Exception) {
                finish(false)
            }
        }
    }
}
