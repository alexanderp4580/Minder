import 'package:flutter/material.dart';

import 'tokens.dart';

class ScreenHeader extends StatelessWidget {
  const ScreenHeader({super.key, required this.title, this.trailing});

  final String title;

  /// A control at the right end of the title row, such as the connection status.
  final Widget? trailing;

  @override
  Widget build(BuildContext context) => Center(
    child: ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: Tokens.pageWidth),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          LayoutBuilder(
            builder: (context, constraints) {
              final narrow = constraints.maxWidth < 360;
              final size = narrow ? 20.0 : 22.0;
              return Container(
                key: const ValueKey('screen-header'),
                padding: EdgeInsets.fromLTRB(
                  narrow ? Tokens.relatedGap : Tokens.sectionGap,
                  Tokens.sectionGap,
                  narrow ? Tokens.relatedGap : Tokens.sectionGap,
                  Tokens.relatedGap,
                ),
                child: SizedBox(
                  height: Tokens.minTarget,
                  child: Row(
                    children: [
                      Expanded(
                        child: FittedBox(
                          fit: BoxFit.scaleDown,
                          alignment: Alignment.centerLeft,
                          child: Semantics(
                            header: true,
                            child: Text(
                              title,
                              style: TextStyle(
                                fontFamily: 'Michroma',
                                fontSize: size,
                                letterSpacing: size * .2,
                              ),
                            ),
                          ),
                        ),
                      ),
                      ?trailing,
                    ],
                  ),
                ),
              );
            },
          ),
          const DecoratedBox(
            key: ValueKey('header-rule'),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.centerLeft,
                end: Alignment.centerRight,
                colors: [Color(0x80f2f2f2), Color(0x00f2f2f2)],
              ),
            ),
            child: SizedBox(height: 1, width: double.infinity),
          ),
        ],
      ),
    ),
  );
}
