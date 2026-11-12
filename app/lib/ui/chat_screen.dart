import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../application/chat_controller.dart';
import 'entry_view.dart';
import 'tokens.dart';
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
      if (c.cursor != _lastCursor) {
        _lastCursor = c.cursor;
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (_scroll.hasClients) {
            _scroll.animateTo(
              _scroll.position.maxScrollExtent,
              duration: const Duration(milliseconds: 180),
              curve: Curves.easeOut,
            );
          }
        });
      }
      return Scaffold(
        body: OrbitalBackground(
          scene: OrbitScene.chat,
          child: SafeArea(
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 760),
                child: Column(
                  children: [
                    Container(
                      padding: const EdgeInsets.fromLTRB(
                        Tokens.gutter,
                        18,
                        Tokens.gutter,
                        0,
                      ),
                      child: Row(
                        children: [
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                const Text(
                                  'CHAT',
                                  style: TextStyle(
                                    fontFamily: 'Michroma',
                                    fontSize: 22,
                                    letterSpacing: 4.4,
                                  ),
                                ),
                                const SizedBox(height: 8),
                                Row(
                                  children: [
                                    Container(
                                      width: 7,
                                      height: 7,
                                      color: c.connected
                                          ? Tokens.text
                                          : Tokens.dim,
                                    ),
                                    const SizedBox(width: 8),
                                    Flexible(
                                      child: Text(
                                        c.connected
                                            ? 'MINDER IS HERE'
                                            : 'CONNECTING',
                                        style: const TextStyle(
                                          fontSize: 12,
                                          color: Tokens.muted,
                                          letterSpacing: 1.2,
                                        ),
                                      ),
                                    ),
                                  ],
                                ),
                              ],
                            ),
                          ),
                          if (widget.onCalendar != null)
                            IconButton(
                              tooltip: 'Calendar',
                              onPressed: widget.onCalendar,
                              icon: const Icon(Icons.calendar_month),
                            ),
                          if (c.loading)
                            const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            ),
                        ],
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
                      child: SelectionArea(
                        child: ListView(
                          controller: _scroll,
                          padding: const EdgeInsets.fromLTRB(
                            Tokens.gutter,
                            22,
                            Tokens.gutter,
                            16,
                          ),
                          children: [
                            if (c.entries.isEmpty && !c.loading)
                              const Padding(
                                padding: EdgeInsets.symmetric(vertical: 30),
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
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
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(c.error!),
                                    if (c.needsSignIn && widget.signIn != null)
                                      Padding(
                                        padding: const EdgeInsets.only(top: 12),
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
                                      )
                                    else
                                      Padding(
                                        padding: const EdgeInsets.only(top: 12),
                                        child: OutlinedButton(
                                          onPressed: c.acting || c.waiting
                                              ? null
                                              : () => c.retry(),
                                          child: Text('Retry'),
                                        ),
                                      ),
                                  ],
                                ),
                              ),
                          ],
                        ),
                      ),
                    ),
                    Container(
                      key: const ValueKey('composer'),
                      constraints: const BoxConstraints(minHeight: 56),
                      padding: const EdgeInsets.symmetric(
                        horizontal: 14,
                        vertical: 5.5,
                      ),
                      decoration: const BoxDecoration(
                        border: Border(top: BorderSide(color: Tokens.line)),
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
