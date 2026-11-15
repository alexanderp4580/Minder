import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/domain/link_launcher.dart';
import 'package:minder/ui/entry_view.dart';
import 'package:minder/ui/tokens.dart';

class _Repository extends Fake implements ChatRepository {
  @override
  Future<List<Entry>> history(int after) async => [];
  @override
  Stream<Entry> watch(int after) => const Stream.empty();
}

class _Launcher implements LinkLauncher {
  final opened = <Uri>[];
  @override
  Future<bool> open(Uri uri) async {
    opened.add(uri);
    return true;
  }
}

Future<_Launcher> _show(
  WidgetTester tester,
  String text, {
  String kind = EntryKind.minder,
}) async {
  final launcher = _Launcher();
  final controller = ChatController(
    _Repository(),
    pollInterval: const Duration(days: 1),
  );
  addTearDown(controller.dispose);
  await tester.pumpWidget(
    MaterialApp(
      theme: Tokens.theme,
      home: Scaffold(
        body: SingleChildScrollView(
          child: EntryView(
            entry: Entry(
              id: 'e',
              cursor: 1,
              kind: kind,
              text: text,
              createdAt: DateTime.utc(2026, 11, 15, 9),
            ),
            controller: controller,
            linkLauncher: launcher,
          ),
        ),
      ),
    ),
  );
  return launcher;
}

void main() {
  testWidgets('Minder messages show bold, lists and code without the marks', (
    tester,
  ) async {
    await _show(tester, 'I found **two** slots:\n\n- Monday\n- `Friday`');
    expect(find.textContaining('**', findRichText: true), findsNothing);
    expect(find.textContaining('`', findRichText: true), findsNothing);
    expect(
      find.textContaining('I found two slots:', findRichText: true),
      findsOneWidget,
    );
    expect(find.text('Monday', findRichText: true), findsOneWidget);
    expect(find.text('Friday', findRichText: true), findsOneWidget);
  });

  testWidgets('your own messages stay literal', (tester) async {
    await _show(tester, 'call **me**', kind: EntryKind.user);
    expect(find.text('call **me**'), findsOneWidget);
  });

  testWidgets('only an HTTPS link opens, and only after a tap', (tester) async {
    final launcher = await _show(
      tester,
      '[safe](https://example.com/a)\n\n[bad](http://example.com)\n\n[worse](javascript:alert(1))',
    );
    expect(launcher.opened, isEmpty);
    for (final word in ['safe', 'bad', 'worse']) {
      final finder = find.text(word, findRichText: true);
      await tester.tapAt(tester.getCenter(finder));
      await tester.pump();
    }
    expect(launcher.opened, [Uri.parse('https://example.com/a')]);
  });

  testWidgets('an image is never fetched', (tester) async {
    await _show(tester, '![x](https://example.com/p.png)');
    expect(find.byType(Image), findsNothing);
    expect(tester.takeException(), isNull);
  });
}
