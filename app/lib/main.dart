import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'application/calendar_controller.dart';
import 'application/chat_controller.dart';
import 'application/notifications.dart';
import 'application/use_cases.dart';
import 'data/chat_api.dart';
import 'data/notification_device.dart';
import 'data/access_auth.dart';
import 'ui/calendar_screen.dart';
import 'ui/chat_screen.dart';
import 'ui/tokens.dart';

const configuredBase = String.fromEnvironment(
  'API_BASE',
  defaultValue: 'https://minder.liveleds.io',
);
const localAuth = bool.fromEnvironment('LOCAL_AUTH', defaultValue: false);
final authProvider = Provider<AccessAuth>((ref) => AccessAuth(configuredBase));
final apiProvider = Provider<ChatApi>((ref) {
  final auth = ref.watch(authProvider);
  final api = ChatApi(
    kIsWeb ? '' : configuredBase,
    authorization: kIsWeb || localAuth ? null : auth.headers,
  );
  return api;
});
final chatProvider = Provider<ChatController>((ref) {
  final controller = ChatController(ref.watch(apiProvider));
  ref.onDispose(controller.dispose);
  unawaited(controller.start());
  return controller;
});
void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const ProviderScope(child: MinderApp()));
}

final calendarProvider = Provider<CalendarController>((ref) {
  final controller = CalendarController(LoadItems(ref.watch(apiProvider)));
  ref.onDispose(controller.dispose);
  return controller;
});
final notificationProvider = Provider<NotificationController>((ref) {
  final api = ref.watch(apiProvider);
  final controller = NotificationController(
    api,
    ActOnReminder(api),
    device: AndroidNotificationDevice(),
    onSync: () => ref.read(chatProvider).sync(),
  );
  ref.onDispose(controller.dispose);
  return controller;
});

class MinderApp extends ConsumerStatefulWidget {
  const MinderApp({super.key});
  @override
  ConsumerState<MinderApp> createState() => _MinderAppState();
}

class _MinderAppState extends ConsumerState<MinderApp>
    with WidgetsBindingObserver {
  bool get nativeAndroid =>
      !kIsWeb && defaultTargetPlatform == TargetPlatform.android;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (nativeAndroid) unawaited(ref.read(notificationProvider).resume());
    });
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed && nativeAndroid) {
      unawaited(ref.read(notificationProvider).resume());
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final notifications = nativeAndroid
        ? ref.watch(notificationProvider)
        : null;
    return ListenableBuilder(
      listenable: notifications ?? const AlwaysStoppedAnimation(0),
      builder: (context, _) => MaterialApp(
        title: 'Minder',
        debugShowCheckedModeBanner: false,
        theme: Tokens.theme,
        home: Builder(
          builder: (context) => ChatScreen(
            controller: ref.watch(chatProvider),
            onCalendar: () {
              final calendar = ref.read(calendarProvider);
              unawaited(calendar.goToday());
              Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => CalendarScreen(controller: calendar),
                ),
              );
            },
            signIn: () async {
              if (kIsWeb) {
                await ref.read(authProvider).webSignIn();
              } else {
                await ref.read(authProvider).signIn();
                if (nativeAndroid) await notifications!.resume();
              }
            },
            enableNotifications: notifications?.enabled == true
                ? null
                : notifications?.enable,
            notificationStatus: notifications?.enabled == true
                ? null
                : notifications?.status,
            notificationsBusy: notifications?.busy ?? false,
          ),
        ),
      ),
    );
  }
}
