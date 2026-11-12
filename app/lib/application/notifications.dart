import 'package:flutter/foundation.dart';

import '../domain/chat_repository.dart';
import '../domain/failure.dart';
import '../domain/push_repository.dart';

class NotificationController extends ChangeNotifier {
  NotificationController(
    this.push,
    this.chat, {
    required this.onSync,
    required this.device,
  }) {
    device.onActionAvailable(resume);
  }
  final PushRepository push;
  final ChatRepository chat;
  final Future<void> Function() onSync;
  final NotificationDevice device;
  bool enabled = false, busy = false, _disposed = false;
  String? status;
  void _notify() {
    if (!_disposed) notifyListeners();
  }

  Future<void> _register(DeviceRegistration value) async {
    if (_disposed) return;
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
    if (_disposed) return;
    enabled = true;
    status = 'Notifications are on.';
  }

  Future<void> enable() async {
    if (_disposed || busy) return;
    busy = true;
    _notify();
    try {
      final config = await push.pushConfig();
      if (_disposed) return;
      final result = await device.enable(config);
      await _register(result);
    } catch (e) {
      if (_disposed) return;
      enabled = false;
      status = e is RepositoryFailure
          ? e.message
          : 'Notifications could not start. Try again.';
    } finally {
      busy = false;
      _notify();
    }
  }

  Future<void> resume() async {
    if (_disposed || busy) return;
    busy = true;
    try {
      final pending = await device.takeAction();
      if (_disposed) return;
      if (pending != null) {
        await chat.actOnReminder(pending.reminderId, pending.action);
        if (_disposed) return;
        await device.completeAction(pending);
        if (_disposed) return;
        await onSync();
        if (_disposed) return;
        status = null;
      }
      final token = await device.registration();
      if (token.granted) {
        await _register(token);
      } else {
        enabled = false;
      }
    } catch (e) {
      if (_disposed) return;
      status = e is RepositoryFailure
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
