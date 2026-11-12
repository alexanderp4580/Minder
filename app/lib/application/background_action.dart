import '../domain/chat_repository.dart';
import '../domain/push_repository.dart';

/// Runs the notification action that Android stored, without any screen.
/// Returns true only when the server accepted it and the device marked it handled.
/// A failed action stays stored for the app to retry.
Future<bool> performNotificationAction(
  NotificationDevice device,
  ChatRepository chat,
) async {
  try {
    final pending = await device.takeAction();
    if (pending == null) return false;
    await chat.actOnReminder(pending.reminderId, pending.action);
    await device.completeAction(pending);
    return true;
  } catch (_) {
    return false;
  }
}
