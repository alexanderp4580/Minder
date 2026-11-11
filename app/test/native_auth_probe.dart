import 'package:flutter/material.dart';
import 'package:flutter_web_auth_2/flutter_web_auth_2.dart';

// Emulator entrypoint. This checks browser return only and creates no session.
void main() => runApp(const MaterialApp(home: NativeAuthProbe()));

class NativeAuthProbe extends StatefulWidget {
  const NativeAuthProbe({super.key});
  @override
  State<NativeAuthProbe> createState() => _NativeAuthProbeState();
}

class _NativeAuthProbeState extends State<NativeAuthProbe> {
  String status = 'Ready';
  Future<void> open() async {
    setState(() => status = 'Waiting for browser');
    try {
      final result = Uri.parse(
        await FlutterWebAuth2.authenticate(
          url: 'http://10.0.2.2:8184/',
          callbackUrlScheme: 'io.liveleds.minder',
          options: const FlutterWebAuth2Options(
            customTabsPackageOrder: [
              'org.mozilla.firefox',
              'com.android.chrome',
            ],
          ),
        ),
      );
      setState(
        () => status = result.queryParameters['state'] == 'emulator-probe'
            ? 'Callback received'
            : 'Wrong callback',
      );
    } catch (_) {
      setState(() => status = 'Callback failed');
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    body: Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(status),
          ElevatedButton(
            onPressed: open,
            child: const Text('Open sign-in probe'),
          ),
        ],
      ),
    ),
  );
}
