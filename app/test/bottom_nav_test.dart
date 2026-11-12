import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/ui/bottom_nav.dart';
import 'package:minder/ui/tokens.dart';

Widget host(
  NavDestination current,
  void Function(NavDestination) onSelect, {
  double keyboard = 0,
}) => MaterialApp(
  theme: Tokens.theme,
  home: MediaQuery(
    data: MediaQueryData(viewInsets: EdgeInsets.only(bottom: keyboard)),
    child: Scaffold(
      bottomNavigationBar: BottomNav(current: current, onSelect: onSelect),
    ),
  ),
);

void main() {
  testWidgets('shows Chat and Calendar and marks the current one', (
    tester,
  ) async {
    await tester.pumpWidget(host(NavDestination.calendar, (_) {}));
    expect(find.text('Chat'), findsOneWidget);
    expect(find.text('Calendar'), findsOneWidget);
    final selected = tester.getSemantics(
      find.byKey(const ValueKey('nav-calendar')),
    );
    final other = tester.getSemantics(find.byKey(const ValueKey('nav-chat')));
    expect(
      selected,
      isSemantics(label: 'Calendar', isButton: true, isSelected: true),
    );
    expect(other, isNot(isSemantics(isSelected: true)));
    expect(other, isSemantics(label: 'Chat', isButton: true));
    expect(
      tester.widget<Icon>(find.byIcon(Icons.calendar_month_outlined)).color,
      Tokens.signal,
    );
    expect(
      tester.widget<Icon>(find.byIcon(Icons.chat_bubble_outline)).color,
      Tokens.muted,
    );
    expect(
      tester.widget<Text>(find.text('Calendar')).style?.color,
      Tokens.text,
    );
    expect(tester.widget<Text>(find.text('Chat')).style?.color, Tokens.muted);
    expect(find.byKey(const ValueKey('nav-active-bar')), findsOneWidget);
    expect(
      tester.getTopLeft(find.byKey(const ValueKey('nav-active-bar'))).dx,
      tester.getTopLeft(find.byKey(const ValueKey('nav-calendar'))).dx,
    );
  });

  testWidgets(
    'tapping the other item selects it; the current one does nothing',
    (tester) async {
      final picked = <NavDestination>[];
      await tester.pumpWidget(host(NavDestination.chat, picked.add));
      await tester.tap(find.text('Chat'));
      expect(picked, isEmpty);
      await tester.tap(find.text('Calendar'));
      expect(picked, [NavDestination.calendar]);
    },
  );

  testWidgets('items are 56 px tall with full-width targets', (tester) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(host(NavDestination.chat, (_) {}));
    final chat = tester.getRect(find.byKey(const ValueKey('nav-chat')));
    final calendar = tester.getRect(find.byKey(const ValueKey('nav-calendar')));
    expect(chat.height, 56);
    expect(calendar.height, 56);
    expect(chat.width, 206);
    expect(tester.getCenter(find.text('Chat')).dx, chat.center.dx);
    expect(tester.getCenter(find.text('Calendar')).dx, calendar.center.dx);
    expect(
      tester.getCenter(find.byIcon(Icons.chat_bubble_outline)).dx,
      chat.center.dx,
    );
    expect(calendar.width, 206);
    expect(tester.takeException(), isNull);
  });

  testWidgets('the bar hides while the keyboard is open', (tester) async {
    await tester.pumpWidget(host(NavDestination.chat, (_) {}, keyboard: 300));
    expect(find.text('Chat'), findsNothing);
    expect(find.text('Calendar'), findsNothing);
  });

  testWidgets('icon and label sit 8 px from the top and bottom of the item', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(host(NavDestination.chat, (_) {}));
    final item = tester.getRect(find.byKey(const ValueKey('nav-chat')));
    final icon = tester.getRect(find.byIcon(Icons.chat_bubble_outline));
    final label = tester.getRect(find.text('Chat'));
    expect(icon.size, const Size(24, 24));
    expect(label.height, 12);
    expect(icon.top - item.top, 8);
    expect(item.bottom - label.bottom, 8);
    expect(label.top - icon.bottom, 4);
  });
}
