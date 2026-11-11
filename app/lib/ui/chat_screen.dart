import 'package:flutter/material.dart';

import '../application/chat_controller.dart';
import '../domain/chat_repository.dart';
import '../domain/entry.dart';
import 'tokens.dart';

class ChatScreen extends StatefulWidget {
  const ChatScreen({
    super.key,
    required this.controller,
    this.signIn,
    this.enableNotifications,
    this.notificationStatus,
    this.notificationsBusy = false,
  });
  final Future<void> Function()? enableNotifications;
  final String? notificationStatus;
  final bool notificationsBusy;
  final ChatController controller;
  final Future<void> Function()? signIn;
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
        body: SafeArea(
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 760),
              child: Column(
                children: [
                  Container(
                    padding: const EdgeInsets.fromLTRB(
                      Tokens.gutter,
                      22,
                      Tokens.gutter,
                      18,
                    ),
                    decoration: const BoxDecoration(
                      border: Border(bottom: BorderSide(color: Tokens.line)),
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
                              const SizedBox(height: 10),
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
                          TextButton(
                            onPressed: widget.notificationsBusy
                                ? null
                                : widget.enableNotifications,
                            child: const Text('Enable notifications'),
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
                        padding: const EdgeInsets.symmetric(
                          horizontal: Tokens.gutter,
                          vertical: 16,
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
                          for (final e in c.entries) _entry(e),
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
                                    FilledButton(
                                      onPressed: _signingIn ? null : _login,
                                      child: Text(
                                        _signingIn ? 'Signing in…' : 'Sign in',
                                      ),
                                    )
                                  else
                                    TextButton(
                                      onPressed: c.acting || c.waiting
                                          ? null
                                          : () => c.retry(),
                                      child: const Text('Retry'),
                                    ),
                                ],
                              ),
                            ),
                        ],
                      ),
                    ),
                  ),
                  Container(
                    padding: const EdgeInsets.fromLTRB(
                      Tokens.gutter,
                      14,
                      Tokens.gutter,
                      14,
                    ),
                    decoration: const BoxDecoration(
                      border: Border(top: BorderSide(color: Tokens.line)),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Expanded(
                          child: TextField(
                            controller: _text,
                            minLines: 1,
                            maxLines: 4,
                            maxLength: 4000,
                            textInputAction: TextInputAction.send,
                            onSubmitted: (_) => _send(),
                            decoration: const InputDecoration(
                              hintText: 'Tell Minder…',
                              counterText: '',
                              contentPadding: EdgeInsets.symmetric(
                                horizontal: 12,
                                vertical: 15,
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: Tokens.gap),
                        SizedBox(
                          height: 54,
                          child: Tooltip(
                            message: 'Send',
                            child: FilledButton(
                              onPressed: c.waiting || c.needsSignIn
                                  ? null
                                  : _send,
                              child: const Text('Send'),
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
      );
    },
  );
  Widget _entry(Entry e) {
    final c = widget.controller;
    final mine = e.kind == 'user';
    final ch = e.change;
    final reminder = e.reminder;
    final pending = e.kind == 'preview' && ch?['state'] == 'pending';
    final time = TimeOfDay.fromDateTime(e.createdAt.toLocal()).format(context);
    final body = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          '${mine ? 'YOU' : 'MINDER'} · $time',
          style: const TextStyle(
            fontSize: 12,
            color: Tokens.muted,
            letterSpacing: 1.0,
          ),
        ),
        const SizedBox(height: 7),
        if (ch != null && e.kind == 'preview') ...[
          Text(
            'PREVIEW · ${ch['action'] == 'create_item'
                ? 'CREATE ITEM'
                : ch['action'] == 'complete_item'
                ? 'COMPLETE ITEM'
                : 'CHANGE ITEM'}',
            style: const TextStyle(
              fontSize: 12,
              color: Tokens.muted,
              letterSpacing: 1,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            ch['item_title'] as String? ?? '',
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          for (final f in (ch['fields'] as List? ?? []))
            Padding(
              padding: const EdgeInsets.only(bottom: 5),
              child: Wrap(
                spacing: 6,
                children: [
                  Text(
                    '${_field(f['name'] as String)}:',
                    style: const TextStyle(color: Tokens.muted),
                  ),
                  if ((f['old'] as String? ?? '').isNotEmpty)
                    Text(
                      _value(f['old'] as String),
                      style: const TextStyle(
                        color: Tokens.muted,
                        decoration: TextDecoration.lineThrough,
                      ),
                    ),
                  Text(_value(f['new'] as String? ?? '')),
                ],
              ),
            ),
          if (pending)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Wrap(
                spacing: Tokens.gap,
                children: [
                  FilledButton(
                    onPressed: c.acting
                        ? null
                        : () => c.confirm(ch['id'] as String),
                    child: const Text('Confirm'),
                  ),
                  TextButton(
                    onPressed: c.acting
                        ? null
                        : () => c.discard(ch['id'] as String),
                    child: const Text('Discard'),
                  ),
                ],
              ),
            )
          else
            Text(
              ch['state'] == 'confirmed'
                  ? 'Confirmed'
                  : ch['state'] == 'discarded'
                  ? 'Discarded'
                  : 'Item changed. See the new preview.',
              style: const TextStyle(color: Tokens.muted),
            ),
        ] else
          Text(
            e.text,
            style: TextStyle(
              color: e.kind == 'error' ? Tokens.alert : Tokens.text,
            ),
          ),
        if (reminder != null && reminder['state'] == 'active')
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Wrap(
              spacing: 8,
              children: [
                FilledButton(
                  onPressed: c.acting
                      ? null
                      : () => c.remind(
                          reminder['id'] as String,
                          ReminderAction.done,
                        ),
                  child: const Text('Done'),
                ),
                PopupMenuButton<ReminderAction>(
                  enabled: !c.acting,
                  onSelected: (action) =>
                      c.remind(reminder['id'] as String, action),
                  itemBuilder: (_) => [
                    const PopupMenuItem(
                      value: ReminderAction.snooze15,
                      child: Text('15 minutes'),
                    ),
                    const PopupMenuItem(
                      value: ReminderAction.snooze60,
                      child: Text('1 hour'),
                    ),
                    const PopupMenuItem(
                      value: ReminderAction.tonight,
                      child: Text('Tonight'),
                    ),
                    const PopupMenuItem(
                      value: ReminderAction.tomorrow,
                      child: Text('Tomorrow'),
                    ),
                  ],
                  child: const Padding(
                    padding: EdgeInsets.all(14),
                    child: Text('Snooze'),
                  ),
                ),
                TextButton(
                  onPressed: c.acting
                      ? null
                      : () => c.remind(
                          reminder['id'] as String,
                          ReminderAction.acknowledge,
                        ),
                  child: const Text('Open'),
                ),
              ],
            ),
          ),
        if (reminder != null && reminder['state'] != 'active')
          Text(
            reminder['state'] == 'superseded'
                ? 'See the latest reminder.'
                : reminder['state'] == 'cancelled'
                ? 'Reminder cancelled.'
                : 'Reminder handled.',
            style: const TextStyle(color: Tokens.muted),
          ),
        if (e.kind == 'result' && ch?['undo_id'] != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: OutlinedButton(
              style: OutlinedButton.styleFrom(
                foregroundColor: Tokens.text,
                side: const BorderSide(color: Tokens.edge),
                shape: const RoundedRectangleBorder(),
              ),
              onPressed: c.acting
                  ? null
                  : () => c.undo(ch!['undo_id'] as String),
              child: const Text('Undo'),
            ),
          ),
        if (e.kind == 'error')
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: OutlinedButton(
              style: OutlinedButton.styleFrom(
                foregroundColor: Tokens.text,
                side: const BorderSide(color: Tokens.edge),
                shape: const RoundedRectangleBorder(),
              ),
              onPressed: c.waiting ? null : () => c.retry(text: c.retryText(e)),
              child: const Text('Retry'),
            ),
          ),
      ],
    );
    return Container(
      key: ValueKey(e.id),
      margin: EdgeInsets.only(left: mine ? 44 : 0, bottom: 18),
      padding: mine || e.kind == 'preview'
          ? const EdgeInsets.all(12)
          : EdgeInsets.zero,
      decoration: mine || e.kind == 'preview'
          ? BoxDecoration(
              color: mine ? Tokens.panel : Tokens.voidColor,
              border: Border.all(color: Tokens.edge),
            )
          : null,
      child: body,
    );
  }

  String _field(String name) => switch (name) {
    'due_at' => 'Due',
    'remind_at' => 'Remind at',
    'nag_minutes' => 'Nag interval (minutes)',
    'title' => 'Title',
    'notes' => 'Notes',
    'status' => 'Status',
    'timezone' => 'Timezone',
    _ => name,
  };
  String _value(String value) {
    if (value.isEmpty) return 'None';
    final date = DateTime.tryParse(value);
    if (date == null || !value.contains('T')) return value;
    final d = date.toLocal();
    return '${d.day}/${d.month}/${d.year} ${d.hour.toString().padLeft(2, '0')}:${d.minute.toString().padLeft(2, '0')}';
  }
}
