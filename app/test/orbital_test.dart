import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/calendar_controller.dart';
import 'package:minder/domain/item.dart';
import 'package:minder/domain/item_repository.dart';
import 'package:minder/ui/calendar_screen.dart';
import 'package:minder/ui/orbital_artwork_data.dart';
import 'package:minder/ui/tokens.dart';
import 'package:minder/ui/orbital_background.dart';

class EmptyItems implements ItemRepository {
  @override
  Future<List<Item>> itemsBetween(DateTime from, DateTime to) async => [];
}

const captureKey = ValueKey('scene-capture');

Widget scene({
  OrbitScene scene = OrbitScene.calendar,
  bool reduced = false,
  bool visible = true,
  Widget child = const SizedBox(),
}) => MaterialApp(
  home: MediaQuery(
    data: MediaQueryData(disableAnimations: reduced),
    child: TickerMode(
      enabled: visible,
      child: RepaintBoundary(
        key: captureKey,
        child: ColoredBox(
          color: Tokens.voidColor,
          child: OrbitalBackground(scene: scene, child: child),
        ),
      ),
    ),
  ),
);

Future<Uint8List> pixels(WidgetTester tester, {Key key = captureKey}) async {
  final boundary = tester.renderObject<RenderRepaintBoundary>(find.byKey(key));
  return (await tester.runAsync(() async {
    final image = await boundary.toImage();
    final bytes = await image.toByteData(format: ui.ImageByteFormat.rawRgba);
    image.dispose();
    return Uint8List.fromList(bytes!.buffer.asUint8List());
  }))!;
}

void main() {
  testWidgets(
    'moving highlights preserve the authored background composition',
    (tester) async {
      await tester.pumpWidget(scene());
      const artwork = ValueKey('orbital-artwork');
      expect(find.byKey(artwork), findsOneWidget);
      final before = await pixels(tester, key: artwork);
      final fullBefore = await pixels(tester);
      await tester.pump(const Duration(seconds: 20));
      expect(await pixels(tester, key: artwork), orderedEquals(before));
      expect(await pixels(tester), isNot(orderedEquals(fullBefore)));
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets('highlights complete a loop in 68.571 s', (tester) async {
    await tester.pumpWidget(scene());
    final start = await pixels(tester);
    await tester.pump(const Duration(milliseconds: 34285));
    final half = await pixels(tester);
    expect(half, isNot(orderedEquals(start)));
    await tester.pump(const Duration(milliseconds: 34286));
    expect(await pixels(tester), orderedEquals(start));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('highlights ease in and out of each loop', (tester) async {
    int changed(Uint8List a, Uint8List b) {
      var count = 0;
      for (var i = 0; i < a.length; i++) {
        if (a[i] != b[i]) count++;
      }
      return count;
    }

    await tester.pumpWidget(scene());
    final start = await pixels(tester);
    await tester.pump(const Duration(milliseconds: 6857));
    final early = await pixels(tester);
    await tester.pump(const Duration(milliseconds: 20571));
    final before = await pixels(tester);
    await tester.pump(const Duration(milliseconds: 6857));
    final middle = await pixels(tester);
    expect(
      changed(start, early),
      lessThan(changed(before, middle) ~/ 2),
      reason:
          'the first tenth of a loop covers less ground than the middle tenth',
    );
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('highlights move without rebuilding foreground content', (
    tester,
  ) async {
    var builds = 0;
    await tester.pumpWidget(
      scene(
        child: Builder(
          builder: (_) {
            builds++;
            return const Text('Task');
          },
        ),
      ),
    );
    final before = await pixels(tester);
    await tester.pump(const Duration(seconds: 20));
    final after = await pixels(tester);
    expect(after, isNot(orderedEquals(before)));
    expect(builds, 1);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets(
    'reduced motion keeps the scene static and schedules no animation frames',
    (tester) async {
      await tester.pumpWidget(scene(reduced: true));
      final before = await pixels(tester);
      await tester.pump(const Duration(seconds: 20));
      expect(await pixels(tester), orderedEquals(before));
      expect(tester.binding.hasScheduledFrame, isFalse);
      await tester.pumpWidget(scene());
      await tester.pump();
      expect(tester.binding.hasScheduledFrame, isTrue);
      await tester.pumpWidget(scene(reduced: true));
      await tester.pump();
      expect(tester.binding.hasScheduledFrame, isFalse);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'hidden routes and backgrounded apps stop ticking and resume when visible',
    (tester) async {
      await tester.pumpWidget(scene());
      await tester.pump();
      expect(tester.binding.hasScheduledFrame, isTrue);
      await tester.pumpWidget(scene(visible: false));
      await tester.pump();
      expect(tester.binding.hasScheduledFrame, isFalse);
      await tester.pumpWidget(scene());
      await tester.pump();
      expect(tester.binding.hasScheduledFrame, isTrue);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pump();
      expect(tester.binding.hasScheduledFrame, isFalse);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pump();
      expect(tester.binding.hasScheduledFrame, isTrue);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets('chat and calendar have different orbit compositions', (
    tester,
  ) async {
    await tester.pumpWidget(scene(reduced: true));
    final calendar = await pixels(tester);
    await tester.pumpWidget(scene(scene: OrbitScene.chat, reduced: true));
    expect(await pixels(tester), isNot(orderedEquals(calendar)));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('calendar paints an orbital background behind its controls', (
    tester,
  ) async {
    final controller = CalendarController(EmptyItems());
    await controller.load();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: CalendarScreen(controller: controller),
      ),
    );
    await tester.pump();
    expect(find.byKey(const ValueKey('orbital-arcs')), findsOneWidget);
    await tester.tap(find.byTooltip('Next month'));
    await tester.pump();
    expect(controller.selected.day, 1);
    await tester.pumpWidget(const SizedBox());
    controller.dispose();
  });

  test('background lines peak at 24 percent opacity, 60 percent below the original 40', () {
    for (final data in [chatArtwork(), calendarArtwork()]) {
      final alphas = [
        for (final edge in data.edges)
          for (final stop in edge.stops) stop.$2,
      ];
      expect(alphas.reduce((a, b) => a > b ? a : b), closeTo(.24, 1e-9));
    }
  });
}
