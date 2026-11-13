import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter/rendering.dart';

import '../application/chat_controller.dart';
import 'bottom_nav.dart';
import 'entry_view.dart';
import 'tokens.dart';
import 'screen_header.dart';
import 'orbital_background.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({
    super.key,
    required this.controller,
    this.signIn,
    this.onCalendar,
    this.enableNotifications,
    this.notificationStatus,
    this.notificationsBusy = false,
  });
  final Future<void> Function()? enableNotifications;
  final String? notificationStatus;
  final bool notificationsBusy;
  final ChatController controller;
  final Future<void> Function()? signIn;
  final VoidCallback? onCalendar;
  @override
  State<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends State<ChatScreen> {
  final _text = TextEditingController();
  final _scroll = ScrollController();
  int _lastCursor = -1;
  bool _signingIn = false;
  bool _stickToBottom = true;
  bool _bottomScrollScheduled = false;

  void _followBottom() {
    if (!_stickToBottom || _bottomScrollScheduled) return;
    _bottomScrollScheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _bottomScrollScheduled = false;
      if (!mounted || !_stickToBottom || !_scroll.hasClients) return;
      _scroll.jumpTo(_scroll.position.maxScrollExtent);
    });
  }

  @override
  void dispose() {
    _text.dispose();
    _scroll.dispose();
    super.dispose();
  }

  void _send() {
    final text = _text.text.trim();
    if (text.isEmpty || widget.controller.waiting) return;
    FocusScope.of(context).unfocus();
    _text.clear();
    widget.controller.send(text);
  }

  Future<void> _login() async {
    setState(() => _signingIn = true);
    try {
      await widget.signIn?.call();
      await widget.controller.signedIn();
    } catch (_) {
      widget.controller.error = 'Sign-in did not finish. Try again.';
    } finally {
      if (mounted) setState(() => _signingIn = false);
    }
  }

  @override
  Widget build(BuildContext context) => AnimatedBuilder(
    animation: widget.controller,
    builder: (context, _) {
      final c = widget.controller;
      if (c.hasUnsentText) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          final unsent = c.takeUnsentText();
          if (unsent == null || !mounted) return;
          final typed = _text.text;
          final restored = typed.isEmpty ? unsent : '$unsent $typed';
          _text.value = TextEditingValue(
            text: restored,
            selection: TextSelection.collapsed(offset: restored.length),
          );
        });
      }
      if (c.cursor != _lastCursor) {
        _lastCursor = c.cursor;
        _stickToBottom = true;
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && _stickToBottom && _scroll.hasClients) {
            _scroll.animateTo(
              _scroll.position.maxScrollExtent,
              duration: const Duration(milliseconds: 180),
              curve: Curves.easeOut,
            );
          }
        });
      }
      return Scaffold(
        bottomNavigationBar: BottomNav(
          current: NavDestination.chat,
          onSelect: (_) => widget.onCalendar?.call(),
        ),
        body: OrbitalBackground(
          scene: OrbitScene.chat,
          child: SafeArea(
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: Tokens.pageWidth),
                child: Column(
                  children: [
                    ScreenHeader(
                      title: 'CHAT',
                      trailing: c.showOffline
                          ? Semantics(
                              button: true,
                              label: 'Offline. Tap to retry',
                              excludeSemantics: true,
                              child: IconButton(
                                onPressed: c.sync,
                                constraints: const BoxConstraints.tightFor(
                                  width: Tokens.minTarget,
                                  height: Tokens.minTarget,
                                ),
                                icon: const Icon(
                                  Icons.cloud_off_outlined,
                                  color: Tokens.alert,
                                ),
                              ),
                            )
                          : null,
                    ),
                    if (c.loading)
                      const Padding(
                        padding: EdgeInsets.only(top: Tokens.relatedGap),
                        child: SizedBox.square(
                          dimension: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        ),
                      ),
                    if (widget.enableNotifications != null)
                      Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: Tokens.gutter,
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const SizedBox(height: 12),
                            OutlinedButton(
                              onPressed: widget.notificationsBusy
                                  ? null
                                  : widget.enableNotifications,
                              child: Text('Enable notifications'),
                            ),
                            if (widget.notificationStatus != null)
                              Text(
                                widget.notificationStatus!,
                                style: const TextStyle(color: Tokens.muted),
                              ),
                          ],
                        ),
                      ),
                    Expanded(
                      child: NotificationListener<ScrollMetricsNotification>(
                        onNotification: (notification) {
                          if (notification.depth == 0) _followBottom();
                          return false;
                        },
                        child: NotificationListener<UserScrollNotification>(
                          onNotification: (notification) {
                            if (notification.depth == 0) {
                              _stickToBottom =
                                  notification.direction ==
                                      ScrollDirection.idle &&
                                  notification.metrics.extentAfter <= 1;
                            }
                            return false;
                          },
                          child: SelectionArea(
                            child: ListView(
                              controller: _scroll,
                              padding: const EdgeInsets.fromLTRB(
                                Tokens.gutter,
                                Tokens.relatedGap,
                                Tokens.gutter,
                                16,
                              ),
                              children: [
                                if (c.entries.isEmpty && !c.loading)
                                  const Padding(
                                    padding: EdgeInsets.symmetric(vertical: 30),
                                    child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        Text(
                                          'MINDER',
                                          style: TextStyle(
                                            fontSize: 12,
                                            color: Tokens.muted,
                                            letterSpacing: 1.2,
                                          ),
                                        ),
                                        SizedBox(height: 10),
                                        Text(
                                          'What do you need to do?',
                                          style: TextStyle(
                                            fontSize: 20,
                                            fontWeight: FontWeight.w600,
                                          ),
                                        ),
                                        SizedBox(height: 8),
                                        Text(
                                          'Tell me a task, or ask what is open.',
                                          style: TextStyle(color: Tokens.muted),
                                        ),
                                      ],
                                    ),
                                  ),
                                for (final e in c.entries)
                                  EntryView(entry: e, controller: c),
                                if (c.waiting)
                                  const Padding(
                                    padding: EdgeInsets.symmetric(vertical: 16),
                                    child: Row(
                                      children: [
                                        SizedBox(
                                          width: 12,
                                          height: 12,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                          ),
                                        ),
                                        SizedBox(width: 12),
                                        Text(
                                          'Working',
                                          style: TextStyle(color: Tokens.muted),
                                        ),
                                      ],
                                    ),
                                  ),
                                if (c.error != null)
                                  Container(
                                    margin: const EdgeInsets.only(top: 16),
                                    padding: const EdgeInsets.all(12),
                                    decoration: BoxDecoration(
                                      border: Border.all(color: Tokens.edge),
                                    ),
                                    child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        Text(c.error!),
                                        if (c.needsSignIn &&
                                            widget.signIn != null)
                                          Padding(
                                            padding: const EdgeInsets.only(
                                              top: 12,
                                            ),
                                            child: Align(
                                              alignment: Alignment.centerRight,
                                              child: FilledButton(
                                                onPressed: _signingIn
                                                    ? null
                                                    : _login,
                                                child: Text(
                                                  _signingIn
                                                      ? 'Signing in…'
                                                      : 'Sign in',
                                                ),
                                              ),
                                            ),
                                          ),
                                      ],
                                    ),
                                  ),
                              ],
                            ),
                          ),
                        ),
                      ),
                    ),
                    Container(
                      key: const ValueKey('composer'),
                      constraints: const BoxConstraints(minHeight: 68),
                      padding: const EdgeInsets.fromLTRB(
                        14,
                        Tokens.gap,
                        14,
                        Tokens.relatedGap,
                      ),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.center,
                        children: [
                          Expanded(
                            child: TextField(
                              controller: _text,
                              minLines: 1,
                              maxLines: 4,
                              inputFormatters: [
                                LengthLimitingTextInputFormatter(4000),
                              ],
                              textInputAction: TextInputAction.send,
                              onSubmitted: (_) => _send(),
                              decoration: const InputDecoration(
                                hintText: 'Tell Minder…',
                              ),
                            ),
                          ),
                          const SizedBox(width: Tokens.gap),
                          Tooltip(
                            message: 'Send',
                            child: FilledButton(
                              onPressed: c.waiting || c.needsSignIn
                                  ? null
                                  : _send,
                              child: Text('Send'),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      );
    },
  );
}
