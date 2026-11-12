import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/application/background_action.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/data/notification_device.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('minder/notifications');
  final calls = <String>[];
  Map<String, String>? pending;
  setUp(() {
    calls.clear();
    pending = {'id': 'abc123', 'action': 'snooze15'};
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          calls.add(call.method);
          if (call.method == 'takeAction') return pending;
          return null;
        });
  });
  tearDown(
    () => TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null),
  );
  ChatApi api(Future<http.Response> Function(http.Request) handler) => ChatApi(
    'https://test',
    authorization: () async => {'Authorization': 'Bearer owner'},
    client: MockClient(handler),
  );

  test('a pending action is sent with the owner login and completed', () async {
    final requests = <http.Request>[];
    final chat = api((r) async {
      requests.add(r);
      return http.Response(
        '{"id":"e1","cursor":1,"kind":"result","text":"Snoozed.","created_at":"2026-11-12T09:00:00Z"}',
        200,
      );
    });
    final ok = await performNotificationAction(
      AndroidNotificationDevice(),
      chat,
    );
    expect(ok, true);
    expect(requests.single.method, 'POST');
    expect(requests.single.url.path, '/v1/reminders/abc123/snooze15');
    expect(requests.single.headers['Authorization'], 'Bearer owner');
    expect(calls, ['takeAction', 'completeAction']);
    chat.close();
  });

  test('a server failure leaves the action incomplete', () async {
    final chat = api((r) async => http.Response('{"code":"conflict"}', 409));
    expect(
      await performNotificationAction(AndroidNotificationDevice(), chat),
      false,
    );
    expect(calls, ['takeAction']);
    chat.close();
  });

  test('a network error leaves the action incomplete', () async {
    final chat = api((r) async => throw http.ClientException('offline'));
    expect(
      await performNotificationAction(AndroidNotificationDevice(), chat),
      false,
    );
    expect(calls, ['takeAction']);
    chat.close();
  });

  test('no pending action means no request', () async {
    pending = null;
    var requests = 0;
    final chat = api((r) async {
      requests++;
      return http.Response('{}', 200);
    });
    expect(
      await performNotificationAction(AndroidNotificationDevice(), chat),
      false,
    );
    expect(requests, 0);
    chat.close();
  });

  test('an invalid action is rejected without a request', () async {
    pending = {'id': 'abc123', 'action': 'delete'};
    var requests = 0;
    final chat = api((r) async {
      requests++;
      return http.Response('{}', 200);
    });
    expect(
      await performNotificationAction(AndroidNotificationDevice(), chat),
      false,
    );
    expect(requests, 0);
    chat.close();
  });
}
