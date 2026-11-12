import 'package:flutter/material.dart';

import 'tokens.dart';

enum NavDestination { chat, calendar }

class BottomNav extends StatelessWidget {
  const BottomNav({super.key, required this.current, required this.onSelect});

  final NavDestination current;
  final ValueChanged<NavDestination> onSelect;

  @override
  Widget build(BuildContext context) {
    if (MediaQuery.viewInsetsOf(context).bottom > 0) {
      return const SizedBox.shrink();
    }
    return Container(
      color: Tokens.voidColor,
      child: Center(
        heightFactor: 1,
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: Tokens.pageWidth),
          child: DecoratedBox(
            decoration: const BoxDecoration(
              border: Border(top: BorderSide(color: Tokens.line)),
            ),
            child: SafeArea(
              top: false,
              child: Row(
                children: [
                  _item(NavDestination.chat, 'Chat', Icons.chat_bubble_outline),
                  _item(
                    NavDestination.calendar,
                    'Calendar',
                    Icons.calendar_month_outlined,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _item(NavDestination destination, String label, IconData icon) {
    final selected = destination == current;
    return Expanded(
      child: Semantics(
        key: ValueKey('nav-${destination.name}'),
        container: true,
        button: true,
        selected: selected,
        label: label,
        excludeSemantics: true,
        child: InkWell(
          onTap: selected ? null : () => onSelect(destination),
          child: SizedBox(
            height: 56,
            child: Stack(
              fit: StackFit.expand,
              children: [
                if (selected)
                  const Positioned(
                    key: ValueKey('nav-active-bar'),
                    left: 0,
                    right: 0,
                    top: 0,
                    height: 2,
                    child: ColoredBox(color: Tokens.signal),
                  ),
                Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Icon(
                      icon,
                      size: 24,
                      color: selected ? Tokens.signal : Tokens.muted,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      label,
                      style: TextStyle(
                        fontSize: 12,
                        height: 1,
                        letterSpacing: 1,
                        fontWeight: FontWeight.w600,
                        color: selected ? Tokens.text : Tokens.muted,
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
