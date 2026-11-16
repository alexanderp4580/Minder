import 'package:flutter/material.dart';

import '../application/board_controller.dart';
import '../domain/board.dart';
import '../domain/item.dart';
import '../domain/link_launcher.dart';
import 'board_task_card.dart';
import 'board_task_sheet.dart';
import 'bottom_nav.dart';
import 'composer_field.dart';
import 'item_delete_dialog.dart';
import 'orbital_background.dart';
import 'screen_header.dart';
import 'tokens.dart';

/// Width from which the three columns sit side by side.
const _wideBoard = 720.0;

String _emptyText(String status) => switch (status) {
  BoardStatus.open => 'Nothing to pick up. Add a task above or tell Minder.',
  BoardStatus.inProgress => 'Nothing in progress.',
  _ => 'Nothing done in the last 14 days.',
};

class BoardScreen extends StatefulWidget {
  const BoardScreen({
    super.key,
    required this.controller,
    required this.onSelect,
    this.linkLauncher,
  });
  final BoardController controller;
  final ValueChanged<NavDestination> onSelect;
  final LinkLauncher? linkLauncher;
  @override
  State<BoardScreen> createState() => _BoardScreenState();
}

class _BoardScreenState extends State<BoardScreen> {
  BoardController get c => widget.controller;
  var _pages = PageController();
  final _add = TextEditingController();
  int _index = 0;
  bool _wasWide = false;
  BoardMove? _announced;
  late ScaffoldMessengerState _messenger;

  @override
  void initState() {
    super.initState();
    // A move made before this screen opened was announced where it happened.
    _announced = c.lastMove;
    c.addListener(_changed);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _messenger = ScaffoldMessenger.of(context);
  }

  @override
  void dispose() {
    c.removeListener(_changed);
    c.dismissMove();
    _pages.dispose();
    _add.dispose();
    super.dispose();
  }

  void _changed() {
    if (!mounted) return;
    setState(() {});
    final move = c.lastMove;
    if (move != null && !identical(move, _announced)) {
      _announced = move;
      ScaffoldMessenger.of(context)
        ..hideCurrentSnackBar()
        ..showSnackBar(
          SnackBar(
            content: Text('Moved to ${BoardStatus.label(move.to)}'),
            duration: const Duration(seconds: 6),
            action: SnackBarAction(label: 'Undo', onPressed: c.undoMove),
          ),
        );
    }
  }

  /// Leaving the Board ends the chance to undo. The messenger belongs to the app, so without
  /// this the Undo bar would stay over the next tab.
  void _leave() {
    c.dismissMove();
    _messenger.clearSnackBars();
  }

  List<Item> _items(String status) => switch (status) {
    BoardStatus.open => c.board.open,
    BoardStatus.inProgress => c.board.inProgress,
    _ => c.board.done,
  };

  Item _latest(Item item) {
    for (final column in [c.board.open, c.board.inProgress, c.board.done]) {
      for (final candidate in column) {
        if (candidate.id == item.id) return candidate;
      }
    }
    return item;
  }

  Future<void> _submit(String text) async {
    if (await c.add(text) && mounted) _add.clear();
  }

  void _open(Item item) => showBoardTaskSheet(
    context,
    item: item,
    launcher: widget.linkLauncher,
    // The sheet holds the item as it was opened. Move the newest copy, so a reload
    // in the meantime does not turn a plain tap into a version conflict.
    onMove: (status) => c.move(_latest(item), status),
    onDelete: () => showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (_) => ItemDeleteDialog(controller: c, item: item),
    ),
  );

  Widget _addField() => Padding(
    padding: const EdgeInsets.only(bottom: Tokens.gap),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Expanded(
          child: ComposerField(
            key: const ValueKey('add-task'),
            controller: _add,
            hint: 'Add a task',
            maxLines: 1,
            maxLength: 200,
            readOnly: c.adding,
            keepFocusOnSubmit: true,
            onSubmitted: () => _submit(_add.text),
          ),
        ),
        const SizedBox(width: Tokens.gap),
        FilledButton(
          key: const ValueKey('add-task-button'),
          onPressed: c.adding ? null : () => _submit(_add.text),
          child: const Text('Add'),
        ),
      ],
    ),
  );

  Widget _column(String status, {required bool header}) {
    final items = _items(status);
    return Column(
      key: ValueKey('column-$status'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (header)
          Padding(
            padding: const EdgeInsets.only(bottom: Tokens.gap),
            child: Row(
              children: [
                Text(
                  BoardStatus.label(status).toUpperCase(),
                  style: const TextStyle(
                    fontSize: 12,
                    letterSpacing: 1,
                    fontWeight: FontWeight.w600,
                    color: Tokens.muted,
                  ),
                ),
                const SizedBox(width: 8),
                Text(
                  '${items.length}',
                  style: const TextStyle(fontSize: 12, color: Tokens.muted),
                ),
              ],
            ),
          ),
        Expanded(
          child: RefreshIndicator(
            onRefresh: c.load,
            child: ListView(
              physics: const AlwaysScrollableScrollPhysics(),
              padding: const EdgeInsets.only(bottom: Tokens.sectionGap),
              children: [
                if (status == BoardStatus.open) _addField(),
                if (items.isEmpty && !c.loading)
                  Padding(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    child: Text(
                      _emptyText(status),
                      style: const TextStyle(color: Tokens.muted),
                    ),
                  ),
                for (final item in items)
                  Padding(
                    padding: const EdgeInsets.only(bottom: Tokens.gap),
                    child: BoardTaskCard(
                      item: item,
                      busy: c.busy.contains(item.id),
                      onOpen: () => _open(item),
                      onMove: (to) => c.move(item, to),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _tabs() =>
      Row(children: [for (var n = 0; n < BoardStatus.all.length; n++) _tab(n)]);

  Widget _tab(int n) {
    final status = BoardStatus.all[n];
    final selected = n == _index;
    return Expanded(
      child: Semantics(
        container: true,
        button: true,
        selected: selected,
        label: '${BoardStatus.label(status)}, ${_items(status).length}',
        excludeSemantics: true,
        child: InkWell(
          key: ValueKey('tab-$status'),
          onTap: () => _pages.animateToPage(
            n,
            duration: const Duration(milliseconds: 200),
            curve: Curves.easeOut,
          ),
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 56),
            child: DecoratedBox(
              decoration: BoxDecoration(
                border: Border(
                  bottom: BorderSide(
                    color: selected ? Tokens.signal : Tokens.line,
                    width: selected ? 2 : 1,
                  ),
                ),
              ),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    BoardStatus.label(status),
                    textAlign: TextAlign.center,
                    style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                      color: selected ? Tokens.text : Tokens.muted,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    '${_items(status).length}',
                    style: TextStyle(
                      fontSize: 16,
                      color: selected ? Tokens.text : Tokens.muted,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _banner() => Container(
    margin: const EdgeInsets.only(bottom: Tokens.gap),
    padding: const EdgeInsets.fromLTRB(16, 4, 8, 4),
    decoration: BoxDecoration(
      color: Tokens.panel,
      border: Border.all(color: Tokens.alert),
    ),
    child: Row(
      children: [
        Expanded(
          child: Text(c.error!, style: const TextStyle(color: Tokens.alert)),
        ),
        TextButton(onPressed: c.load, child: const Text('Retry')),
      ],
    ),
  );

  @override
  Widget build(BuildContext context) => PopScope(
    onPopInvokedWithResult: (didPop, _) {
      if (didPop) _leave();
    },
    child: _scaffold(context),
  );

  Widget _scaffold(BuildContext context) => Scaffold(
    bottomNavigationBar: BottomNav(
      current: NavDestination.board,
      onSelect: (destination) {
        _leave();
        widget.onSelect(destination);
      },
    ),
    body: OrbitalBackground(
      scene: OrbitScene.calendar,
      child: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 1200),
            child: Column(
              children: [
                const ScreenHeader(title: 'BOARD'),
                if (c.loading)
                  const LinearProgressIndicator(minHeight: 2)
                else
                  const SizedBox(height: 2),
                Expanded(
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      final wide = constraints.maxWidth >= _wideBoard;
                      if (wide != _wasWide) {
                        // The pages leave the tree in the wide layout. A new controller
                        // opens the page the tabs show instead of page 0.
                        if (!wide) {
                          _pages.dispose();
                          _pages = PageController(initialPage: _index);
                        }
                        _wasWide = wide;
                      }
                      return Padding(
                        padding: const EdgeInsets.fromLTRB(
                          Tokens.relatedGap,
                          Tokens.relatedGap,
                          Tokens.relatedGap,
                          0,
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            if (c.error != null) _banner(),
                            if (!wide) _tabs(),
                            if (!wide) const SizedBox(height: Tokens.gap),
                            Expanded(
                              child: wide
                                  ? Row(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        for (final status in BoardStatus.all)
                                          Expanded(
                                            child: Padding(
                                              padding: const EdgeInsets.only(
                                                right: Tokens.relatedGap,
                                              ),
                                              child: _column(
                                                status,
                                                header: true,
                                              ),
                                            ),
                                          ),
                                      ],
                                    )
                                  : PageView(
                                      controller: _pages,
                                      onPageChanged: (n) =>
                                          setState(() => _index = n),
                                      children: [
                                        for (final status in BoardStatus.all)
                                          _column(status, header: false),
                                      ],
                                    ),
                            ),
                          ],
                        ),
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}
