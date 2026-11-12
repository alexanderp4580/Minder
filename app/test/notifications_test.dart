import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/application/notifications.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/data/notification_device.dart';

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
            expect(call.arguments, {
              'app_id': 'app',
              'api_key': 'public',
              'project_id': 'project',
              'sender_id': 'sender',
            });
            return {
              'granted': true,
              'device_id': 'phone',
              'token': 'device-token',
            };
          });
      final c = NotificationController(
        api,
        api,
        device: AndroidNotificationDevice(),
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
      api,
      device: AndroidNotificationDevice(),
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
        api,
        device: AndroidNotificationDevice(),
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
  test('failed registration on resume keeps notifications disabled', () async {
    final api = ChatApi(
      'https://test',
      client: MockClient(
        (r) async => http.Response('{"message":"Sign in to continue."}', 401),
      ),
    );
    addTearDown(api.close);
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method == 'token') {
            return {
              'granted': true,
              'device_id': 'phone',
              'token': 'device-token',
            };
          }
          return null;
        });
    final c = NotificationController(
      api,
      api,
      device: AndroidNotificationDevice(),
      onSync: () async {},
    );
    addTearDown(c.dispose);
    await c.resume();
    expect(c.enabled, false);
    expect(c.status, contains('Sign in'));
  });

  test('resume detects revoked Android permission', () async {
    var granted = true;
    final api = ChatApi(
      'https://test',
      client: MockClient((r) async => http.Response('{}', 200)),
    );
    addTearDown(api.close);
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          if (call.method == 'enable' || call.method == 'token') {
            return {
              'granted': granted,
              'device_id': 'phone',
              'token': 'device-token',
            };
          }
          return null;
        });
    final c = NotificationController(
      api,
      api,
      device: AndroidNotificationDevice(),
      onSync: () async {},
    );
    addTearDown(c.dispose);
    await c.enable();
    expect(c.enabled, true);
    granted = false;
    await c.resume();
    expect(c.enabled, false);
  });
  test('dispose during native enable stops device registration', () async {
    final started = Completer<void>();
    final native = Completer<Map<String, Object?>>();
    var registrations = 0;
    final api = ChatApi(
      'https://test',
      client: MockClient((request) async {
        if (request.url.path.endsWith('/devices')) registrations++;
        return http.Response('{}', 200);
      }),
    );
    addTearDown(api.close);
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
          started.complete();
          return native.future;
        });
    final controller = NotificationController(
      api,
      api,
      device: AndroidNotificationDevice(),
      onSync: () async {},
    );
    final enabling = controller.enable();
    await started.future;
    controller.dispose();
    native.complete({
      'granted': true,
      'device_id': 'phone',
      'token': 'device-token',
    });
    await enabling;
    expect(registrations, 0);
  });

  test(
    'dispose during resume leaves the pending action for the next controller',
    () async {
      final started = Completer<void>();
      final pending = Completer<Map<String, Object?>>();
      var actions = 0, syncs = 0;
      final api = ChatApi(
        'https://test',
        client: MockClient((request) async {
          actions++;
          return http.Response(
            '{"id":"result","cursor":2,"kind":"result","created_at":"2026-11-12T10:00:00Z"}',
            200,
          );
        }),
      );
      addTearDown(api.close);
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
            if (call.method == 'takeAction') {
              started.complete();
              return pending.future;
            }
            return {'granted': false};
          });
      final controller = NotificationController(
        api,
        api,
        device: AndroidNotificationDevice(),
        onSync: () async {
          syncs++;
        },
      );
      final resuming = controller.resume();
      await started.future;
      controller.dispose();
      pending.complete({'id': 'reminder', 'action': 'done'});
      await resuming;
      expect(actions, 0);
      expect(syncs, 0);
    },
  );
}
