import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/application/use_cases.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/application/notifications.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('minder/notifications');
  tearDown(
    () => TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null),
  );
  test(
    'enable registers the granted device and preserves authentication',
    () async {
      var registered = false;
      final api = ChatApi(
        'https://test',
        authorization: () async => {'Authorization': 'Bearer owner'},
        client: MockClient((r) async {
          expect(r.headers['Authorization'], 'Bearer owner');
          if (r.url.path.endsWith('config')) {
            return http.Response(
              jsonEncode({
                'app_id': 'app',
                'api_key': 'public',
                'project_id': 'project',
                'sender_id': 'sender',
              }),
              200,
            );
          }
          registered = true;
          expect(jsonDecode(r.body)['token'], 'device-token');
          return http.Response('{}', 200);
        }),
      );
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
            expect(call.method, 'enable');
            return {
              'granted': true,
              'device_id': 'phone',
              'token': 'device-token',
            };
          });
      final c = NotificationController(
        api,
        ActOnReminder(api),
        onSync: () async {},
      );
      await c.enable();
      expect(registered, true);
      expect(c.enabled, true);
      c.dispose();
      api.close();
    },
  );
  test('denied permission stays visible and does not register', () async {
    final api = ChatApi(
      'https://test',
      client: MockClient((r) async {
        expect(r.url.path, '/v1/push/config');
        return http.Response('{}', 200);
      }),
    );
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async => {'granted': false});
    final c = NotificationController(
      api,
      ActOnReminder(api),
      onSync: () async {},
    );
    await c.enable();
    expect(c.enabled, false);
    expect(c.status, contains('permission'));
    c.dispose();
    api.close();
  });
  test(
    'failed notification action remains pending until authenticated retry',
    () async {
      var fail = true, completed = 0, syncs = 0;
      final api = ChatApi(
        'https://test',
        client: MockClient((r) async {
          expect(r.url.path, '/v1/reminders/reminder/done');
          if (fail) {
            return http.Response('{"message":"Sign in to continue."}', 401);
          }
          return http.Response(
            '{"id":"result","cursor":2,"kind":"result","text":"Done.","created_at":"2026-11-11T10:00:00Z"}',
            200,
          );
        }),
      );
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
            if (call.method == 'takeAction') {
              return {'id': 'reminder', 'action': 'done'};
            }
            if (call.method == 'completeAction') {
              completed++;
              return null;
            }
            if (call.method == 'token') return {'granted': false};
            return null;
          });
      final c = NotificationController(
        api,
        ActOnReminder(api),
        onSync: () async {
          syncs++;
        },
      );
      await c.resume();
      expect(completed, 0);
      expect(c.status, contains('Sign in'));
      fail = false;
      await c.resume();
      expect(completed, 1);
      expect(syncs, greaterThan(0));
      c.dispose();
      api.close();
    },
  );
}
