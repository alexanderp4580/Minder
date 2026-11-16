import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/board_controller.dart';
import 'package:minder/domain/failure.dart';
import 'package:minder/ui/board_screen.dart';
import 'package:minder/ui/bottom_nav.dart';
import 'package:minder/ui/tokens.dart';

import 'board_fakes.dart';

class Harness {
  Harness(this.repo) : controller = BoardController(repo);
  final FakeBoard repo;
  final BoardController controller;
  final navigated = <NavDestination>[];
}

Future<Harness> pump(
  WidgetTester tester, {
  Size size = const Size(390, 844),
  void Function(FakeBoard)? seed,
  bool load = true,
}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  tester.platformDispatcher.accessibilityFeaturesTestValue =
      const FakeAccessibilityFeatures(disableAnimations: true);
  addTearDown(tester.view.reset);
  addTearDown(tester.platformDispatcher.clearAccessibilityFeaturesTestValue);
  final repo = FakeBoard();
  if (seed != null) {
    seed(repo);
  } else {
    repo.items.addAll([
      task(
        'a',
        'open',
        priority: 2,
        list: 'House',
        notes: 'Ask for **two** quotes',
      ),
      task('b', 'open'),
      task('c', 'in_progress'),
      task('d', 'done'),
    ]);
  }
  final h = Harness(repo);
  addTearDown(h.controller.dispose);
  if (load) await h.controller.load();
  await tester.pumpWidget(
    MaterialApp(
      theme: Tokens.theme,
      home: BoardScreen(controller: h.controller, onSelect: h.navigated.add),
    ),
  );
  await tester.pump();
  return h;
}

void main() {
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    final icons = FontLoader('MaterialIcons')
      ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
    await icons.load();
    final michroma = FontLoader('Michroma')
      ..addFont(rootBundle.load('assets/fonts/Michroma-Regular.ttf'));
    await michroma.load();
    final chakra = FontLoader('Chakra Petch');
    for (final file in ['Regular', 'Medium', 'SemiBold', 'Bold']) {
      chakra.addFont(rootBundle.load('assets/fonts/ChakraPetch-$file.ttf'));
    }
    await chakra.load();
  });

  testWidgets('a narrow screen shows one column with counts in the tabs', (
    tester,
  ) async {
    await pump(tester);
    expect(find.text('BOARD'), findsOneWidget);
    expect(find.byKey(const ValueKey('tab-open')), findsOneWidget);
    expect(find.text('To pick up'), findsOneWidget);
    expect(find.text('In progress'), findsOneWidget);
    expect(find.text('Done'), findsWidgets);
    expect(find.text('2'), findsOneWidget);
    expect(find.text('Task a'), findsOneWidget);
    expect(find.text('Task b'), findsOneWidget);
    expect(find.text('Task c'), findsNothing);
    expect(find.text('Task d'), findsNothing);
    await tester.tap(find.byKey(const ValueKey('tab-in_progress')));
    await tester.pumpAndSettle();
    expect(find.text('Task c'), findsOneWidget);
    expect(find.text('Task a'), findsNothing);
    await tester.tap(find.byKey(const ValueKey('tab-done')));
    await tester.pumpAndSettle();
    expect(find.text('Task d'), findsOneWidget);
  });

  testWidgets('a wide screen shows the three columns side by side', (
    tester,
  ) async {
    await pump(tester, size: const Size(1100, 800));
    for (final t in ['Task a', 'Task b', 'Task c', 'Task d']) {
      expect(find.text(t), findsOneWidget);
    }
    expect(find.byKey(const ValueKey('tab-open')), findsNothing);
    final open = tester.getTopLeft(find.byKey(const ValueKey('column-open')));
    final doing = tester.getTopLeft(
      find.byKey(const ValueKey('column-in_progress')),
    );
    final done = tester.getTopLeft(find.byKey(const ValueKey('column-done')));
    expect(open.dx < doing.dx && doing.dx < done.dx, isTrue);
    expect(open.dy, doing.dy);
  });

  testWidgets('a card shows priority, list and one 44 px move button', (
    tester,
  ) async {
    await pump(tester);
    expect(find.text('Priority 2'), findsOneWidget);
    expect(find.text('House'), findsOneWidget);
    final start = find.byKey(const ValueKey('move-a'));
    expect(
      find.descendant(of: start, matching: find.text('Start')),
      findsOneWidget,
    );
    expect(tester.getSize(start).height, greaterThanOrEqualTo(44));
    expect(find.byKey(const ValueKey('stripe-a')), findsOneWidget);
  });

  testWidgets('each column offers its own forward action', (tester) async {
    await pump(tester, size: const Size(1100, 800));
    expect(
      find.descendant(
        of: find.byKey(const ValueKey('move-a')),
        matching: find.text('Start'),
      ),
      findsOneWidget,
    );
    expect(
      find.descendant(
        of: find.byKey(const ValueKey('move-c')),
        matching: find.text('Done'),
      ),
      findsOneWidget,
    );
    expect(
      find.descendant(
        of: find.byKey(const ValueKey('move-d')),
        matching: find.text('Reopen'),
      ),
      findsOneWidget,
    );
  });

  testWidgets('Start moves the task and Undo puts it back', (tester) async {
    final h = await pump(tester);
    await tester.tap(find.byKey(const ValueKey('move-a')));
    await tester.pumpAndSettle();
    expect(h.repo.calls, contains('move a in_progress v1'));
    expect(find.text('Task a'), findsNothing);
    expect(find.text('Moved to In progress'), findsOneWidget);
    final bar = tester.widget<Material>(
      find
          .descendant(
            of: find.byType(SnackBar),
            matching: find.byType(Material),
          )
          .first,
    );
    expect(bar.color, Tokens.panel, reason: 'a dark bar keeps Undo readable');
    await tester.tap(find.text('Undo'));
    await tester.pumpAndSettle();
    expect(h.repo.calls, contains('move a open v2'));
    expect(find.text('Task a'), findsOneWidget);
  });

  testWidgets('the add field creates a task and clears itself', (tester) async {
    final h = await pump(tester);
    await tester.enterText(
      find.byKey(const ValueKey('add-task')),
      'Call plumber',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();
    expect(h.repo.calls, contains('add Call plumber'));
    expect(find.text('Call plumber'), findsOneWidget);
    expect(
      tester
          .widget<TextField>(find.byKey(const ValueKey('add-task')))
          .controller!
          .text,
      isEmpty,
    );
    expect(
      FocusManager.instance.primaryFocus?.context
          ?.findAncestorWidgetOfExactType<TextField>()
          ?.key,
      const ValueKey('add-task'),
      reason: 'the field keeps focus for the next task',
    );
  });

  testWidgets('a failed add keeps the text so the owner can retry', (
    tester,
  ) async {
    final h = await pump(tester);
    h.repo.failNext = const RepositoryFailure(
      'Check your connection. Then try again.',
      transport: true,
    );
    await tester.enterText(
      find.byKey(const ValueKey('add-task')),
      'Call plumber',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();
    expect(find.text('Check your connection. Then try again.'), findsOneWidget);
    expect(
      tester
          .widget<TextField>(find.byKey(const ValueKey('add-task')))
          .controller!
          .text,
      'Call plumber',
    );
  });

  testWidgets('empty columns say what belongs there', (tester) async {
    await pump(tester, seed: (_) {});
    expect(
      find.text('Nothing to pick up. Add a task above or tell Minder.'),
      findsOneWidget,
    );
    await tester.tap(find.byKey(const ValueKey('tab-in_progress')));
    await tester.pumpAndSettle();
    expect(find.text('Nothing in progress.'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('tab-done')));
    await tester.pumpAndSettle();
    expect(find.text('Nothing done in the last 14 days.'), findsOneWidget);
  });

  testWidgets('a load error shows a message and Retry reloads', (tester) async {
    final h = await pump(tester, load: false);
    h.repo.failNext = const RepositoryFailure(
      'Check your connection. Then try again.',
      transport: true,
    );
    await h.controller.load();
    await tester.pump();
    expect(find.text('Check your connection. Then try again.'), findsOneWidget);
    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();
    expect(find.text('Check your connection. Then try again.'), findsNothing);
    expect(find.text('Task a'), findsOneWidget);
  });

  testWidgets('the details sheet shows notes and moves to any column', (
    tester,
  ) async {
    final h = await pump(tester);
    await tester.tap(find.text('Task a'));
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('task-sheet')), findsOneWidget);
    expect(find.textContaining('Ask for', findRichText: true), findsOneWidget);
    expect(find.byKey(const ValueKey('sheet-move-open')), findsNothing);
    await tester.tap(find.byKey(const ValueKey('sheet-move-done')));
    await tester.pumpAndSettle();
    expect(h.repo.calls, contains('move a done v1'));
    expect(find.byKey(const ValueKey('task-sheet')), findsNothing);
  });

  testWidgets('Delete in the sheet opens the delete preview dialog', (
    tester,
  ) async {
    final h = await pump(tester);
    await tester.tap(find.text('Task a'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('sheet-delete')));
    await tester.pumpAndSettle();
    expect(find.text('Delete item'), findsOneWidget);
    expect(h.repo.calls, contains('previewDelete a'));
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();
    expect(h.repo.calls, contains('confirm c-a'));
  });

  testWidgets('the bar has Chat, Board and Calendar and reports taps', (
    tester,
  ) async {
    final h = await pump(tester);
    expect(find.text('Chat'), findsOneWidget);
    expect(find.text('Calendar'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('nav-calendar')));
    expect(h.navigated, [NavDestination.calendar]);
    await tester.tap(find.byKey(const ValueKey('nav-chat')));
    expect(h.navigated.last, NavDestination.chat);
    await tester.tap(find.byKey(const ValueKey('nav-board')));
    expect(h.navigated.length, 2, reason: 'the current tab does nothing');
  });

  testWidgets('320 px with large text does not overflow', (tester) async {
    tester.platformDispatcher.textScaleFactorTestValue = 1.4;
    addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
    await pump(tester, size: const Size(320, 640));
    expect(tester.takeException(), isNull);
    await tester.tap(find.byKey(const ValueKey('tab-in_progress')));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  testWidgets('golden: the narrow board', (tester) async {
    await pump(
      tester,
      seed: (repo) => repo.items.addAll([
        task('a', 'open', title: 'Call plumber', priority: 1, list: 'House'),
        task('b', 'open', title: 'Order a new bike light'),
        task(
          'c',
          'open',
          title: 'Ask the landlord about the broken window latch in the back bedroom and the damp patch',
          priority: 3,
        ),
        task('d', 'in_progress', title: 'Renew passport'),
        task('e', 'done', title: 'Book dentist'),
      ]),
    );
    await expectLater(
      find.byType(BoardScreen),
      matchesGoldenFile('goldens/board.png'),
    );
  });
}
