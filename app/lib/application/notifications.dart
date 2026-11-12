import 'package:flutter/foundation.dart';

import '../domain/chat_repository.dart';
import '../domain/push_repository.dart';
import 'use_cases.dart';

class NotificationController extends ChangeNotifier {
  NotificationController(
    this.push,
    this.actOnReminder, {
    required this.onSync,
    required this.device,
  }) {
    device.onActionAvailable(resume);
  }
  final PushRepository push;
  final ActOnReminder actOnReminder;
  final Future<void> Function() onSync;
  final NotificationDevice device;
  bool enabled = false, busy = false, _disposed = false;
  String? status;
  void _notify() {
    if (!_disposed) notifyListeners();
  }

  Future<void> _register(DeviceRegistration value) async {
    enabled = false;
    if (!value.granted) {
      status = 'Allow notification permission in Android settings.';
      return;
    }
    final token = value.token;
    if (token == null || token.isEmpty) {
      enabled = false;
      status = 'The push token is not ready. Try again.';
      return;
    }
    final deviceId = value.deviceId;
    if (deviceId == null || deviceId.isEmpty) {
      enabled = false;
      status = 'The device is not ready. Try again.';
      return;
    }
    await push.registerDevice(deviceId, token);
    enabled = true;
    status = 'Notifications are on.';
  }

  Future<void> enable() async {
    if (busy) return;
    busy = true;
    _notify();
    try {
      final config = await push.pushConfig();
      final result = await device.enable(config);
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
      final pending = await device.takeAction();
      if (pending != null) {
        await actOnReminder(pending.reminderId, pending.action);
        await device.completeAction(pending);
        await onSync();
        status = null;
      }
      final token = await device.registration();
      if (token.granted) {
        await _register(token);
      } else {
        enabled = false;
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
    device.onActionAvailable(null);
    super.dispose();
  }
}
