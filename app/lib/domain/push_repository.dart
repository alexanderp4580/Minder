import 'chat_repository.dart';

class PushConfiguration {
  const PushConfiguration({
    required this.appId,
    required this.apiKey,
    required this.projectId,
    required this.senderId,
  });
  final String appId, apiKey, projectId, senderId;
}

class DeviceRegistration {
  const DeviceRegistration({required this.granted, this.deviceId, this.token});
  final bool granted;
  final String? deviceId, token;
}

class NotificationAction {
  const NotificationAction(this.reminderId, this.action);
  final String reminderId;
  final ReminderAction action;
}

abstract interface class PushRepository {
  Future<PushConfiguration> pushConfig();
  Future<void> registerDevice(String id, String token);
}

abstract interface class NotificationDevice {
  Future<DeviceRegistration> enable(PushConfiguration config);
  Future<DeviceRegistration> registration();
  Future<NotificationAction?> takeAction();
  Future<void> completeAction(NotificationAction action);
  void onActionAvailable(Future<void> Function()? callback);
}
