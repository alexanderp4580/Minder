import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'application/chat_controller.dart';
import 'data/chat_api.dart';
import 'data/access_auth.dart';
import 'ui/chat_screen.dart';
import 'ui/tokens.dart';

const configuredBase = String.fromEnvironment(
  'API_BASE',
  defaultValue: 'https://minder.liveleds.io',
);
const localAuth = bool.fromEnvironment('LOCAL_AUTH', defaultValue: false);
final authProvider = Provider<AccessAuth>((ref) => AccessAuth(configuredBase));
final chatProvider = Provider<ChatController>((ref) {
  final auth = ref.watch(authProvider);
  final api = ChatApi(
    kIsWeb ? '' : configuredBase,
    authorization: kIsWeb || localAuth ? null : auth.headers,
  );
  final controller = ChatController(api);
  ref.onDispose(controller.dispose);
  unawaited(controller.start());
  return controller;
});
void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const ProviderScope(child: MinderApp()));
}

class MinderApp extends ConsumerWidget {
  const MinderApp({super.key});
  @override
  Widget build(BuildContext context, WidgetRef ref) => MaterialApp(
    title: 'Minder',
    debugShowCheckedModeBanner: false,
    theme: Tokens.theme,
    home: ChatScreen(
      controller: ref.watch(chatProvider),
      signIn: kIsWeb
          ? () => ref.read(authProvider).webSignIn()
          : () => ref.read(authProvider).signIn(),
    ),
  );
}
