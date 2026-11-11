import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

import '../domain/chat_repository.dart';
import 'use_cases.dart';

class NotificationController extends ChangeNotifier {
  NotificationController(
    this.push,
    this.actOnReminder, {
    required this.onSync,
    MethodChannel? channel,
  }) : channel = channel ?? const MethodChannel('minder/notifications') {
    this.channel.setMethodCallHandler((call) async {
      if (call.method == 'actionAvailable') await resume();
    });
  }
  final PushRepository push;
  final ActOnReminder actOnReminder;
  final Future<void> Function() onSync;
  final MethodChannel channel;
  bool enabled = false, busy = false, _disposed = false;
  String? status;
  void _notify() {
    if (!_disposed) notifyListeners();
  }

  Future<void> _register(Map<dynamic, dynamic>? value) async {
    enabled = value?['granted'] == true;
    if (!enabled) {
      status = 'Allow notification permission in Android settings.';
      return;
    }
    final token = value?['token'] as String?;
    if (token == null || token.isEmpty) {
      enabled = false;
      status = 'The push token is not ready. Try again.';
      return;
    }
    await push.registerDevice(value!['device_id'] as String, token);
    status = 'Notifications are on.';
  }

  Future<void> enable() async {
    if (busy) return;
    busy = true;
    _notify();
    try {
      final config = await push.pushConfig();
      final result = await channel.invokeMapMethod<String, dynamic>(
        'enable',
        config,
      );
      await _register(result);
    } catch (e) {
      enabled = false;
      status = e is ApiException
          ? e.message
          : 'Notifications could not start. Try again.';
    } finally {
      busy = false;
      _notify();
    }
  }

  Future<void> resume() async {
    if (busy) return;
    busy = true;
    try {
      final pending = await channel.invokeMapMethod<String, dynamic>(
        'takeAction',
      );
      if (pending != null) {
        final id = pending['id'] as String,
            action = pending['action'] as String;
        final parsed = const {
          'done': ReminderAction.done,
          'ack': ReminderAction.acknowledge,
          'snooze15': ReminderAction.snooze15,
        }[action];
        if (!RegExp(r'^[a-zA-Z0-9]+$').hasMatch(id) || parsed == null) {
          throw const ApiException('This notification action is invalid.');
        }
        await actOnReminder(id, parsed);
        await channel.invokeMethod<void>('completeAction', pending);
        await onSync();
        status = null;
      }
      final token = await channel.invokeMapMethod<String, dynamic>('token');
      if (token?['granted'] == true) {
        await _register(token);
      }
    } catch (e) {
      status = e is ApiException
          ? e.message
          : 'Notification action did not finish. Open Minder and try again.';
      await onSync();
    } finally {
      busy = false;
      _notify();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}
