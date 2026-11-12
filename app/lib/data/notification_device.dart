import 'package:flutter/services.dart';

import '../domain/chat_repository.dart';
import '../domain/push_repository.dart';

class AndroidNotificationDevice implements NotificationDevice {
  AndroidNotificationDevice({MethodChannel? channel})
    : _channel = channel ?? const MethodChannel('minder/notifications');

  final MethodChannel _channel;

  @override
  void onActionAvailable(Future<void> Function()? callback) {
    _channel.setMethodCallHandler(
      callback == null
          ? null
          : (call) async {
              if (call.method == 'actionAvailable') await callback();
            },
    );
  }

  DeviceRegistration _registration(Map<String, dynamic>? value) =>
      DeviceRegistration(
        granted: value?['granted'] == true,
        deviceId: value?['device_id'] as String?,
        token: value?['token'] as String?,
      );

  @override
  Future<DeviceRegistration> enable(PushConfiguration config) async =>
      _registration(
        await _channel.invokeMapMethod<String, dynamic>('enable', {
          'app_id': config.appId,
          'api_key': config.apiKey,
          'project_id': config.projectId,
          'sender_id': config.senderId,
        }),
      );

  @override
  Future<DeviceRegistration> registration() async =>
      _registration(await _channel.invokeMapMethod<String, dynamic>('token'));

  static const _actions = {
    'done': ReminderAction.done,
    'ack': ReminderAction.acknowledge,
    'snooze15': ReminderAction.snooze15,
  };

  @override
  Future<NotificationAction?> takeAction() async {
    final pending = await _channel.invokeMapMethod<String, dynamic>(
      'takeAction',
    );
    if (pending == null) return null;
    final id = pending['id'];
    final action = _actions[pending['action']];
    if (id is! String ||
        !RegExp(r'^[a-zA-Z0-9]+$').hasMatch(id) ||
        action == null) {
      throw const ApiException('This notification action is invalid.');
    }
    return NotificationAction(id, action);
  }

  @override
  Future<void> completeAction(NotificationAction action) =>
      _channel.invokeMethod<void>('completeAction', {
        'id': action.reminderId,
        'action': _actions.entries
            .singleWhere((e) => e.value == action.action)
            .key,
      });
}
