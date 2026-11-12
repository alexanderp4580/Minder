import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../application/chat_controller.dart';
import '../domain/chat_repository.dart';
import '../domain/entry.dart';
import 'format.dart';
import 'tokens.dart';

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

const _meta = TextStyle(fontSize: 12, color: Tokens.muted, letterSpacing: 1.2);

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
                          IconButton(tooltip: 'Calendar', onPressed: widget.onCalendar, icon: const Icon(Icons.calendar_month)),
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
                          OutlinedButton(onPressed: widget.notificationsBusy
                                ? null
                                : widget.enableNotifications, child: Text('Enable notifications')),
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
                                    Padding(
                                      padding: const EdgeInsets.only(top: 12),
                                      child: Align(
                                        alignment: Alignment.centerRight,
                                        child: FilledButton(onPressed: _signingIn ? null : _login, child: Text(_signingIn
                                            ? 'Signing in…'
                                            : 'Sign in')),
                                      ),
                                    )
                                  else
                                    Padding(
                                      padding: const EdgeInsets.only(top: 12),
                                      child: OutlinedButton(onPressed: c.acting || c.waiting
                                            ? null
                                            : () => c.retry(), child: Text('Retry')),
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
                          child: FilledButton(onPressed: c.waiting || c.needsSignIn
                                ? null
                                : _send, child: Text('Send')),
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
    final time =
        clock(e.createdAt.toLocal()) +
        (e.costEur > 0 ? ' · ${cost(e.costEur)}' : '');
    final body = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (ch != null && e.kind == 'preview') ...[
          Text(
            'PREVIEW · ${_action(ch['action'] as String)}',
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
              padding: const EdgeInsets.only(top: 12),
              child: Row(
                children: [
                  Expanded(
                    flex: 10,
                    child: OutlinedButton(onPressed: c.acting
                          ? null
                          : () => c.discard(ch['id'] as String), child: Text('Discard')),
                  ),
                  const SizedBox(width: Tokens.gap),
                  Expanded(
                    flex: 13,
                    child: FilledButton(onPressed: c.acting
                          ? null
                          : () => c.confirm(ch['id'] as String), child: Text('Confirm')),
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
            padding: const EdgeInsets.only(top: 12),
            child: Row(
              children: [
                Expanded(
                  flex: 10,
                  child: MenuAnchor(
                    menuChildren: [
                      for (final (action, label) in const [
                        (ReminderAction.snooze15, '15 minutes'),
                        (ReminderAction.snooze60, '1 hour'),
                        (ReminderAction.tonight, 'Tonight'),
                        (ReminderAction.tomorrow, 'Tomorrow'),
                      ])
                        MenuItemButton(
                          onPressed: () =>
                              c.remind(reminder['id'] as String, action),
                          child: Text(label),
                        ),
                    ],
                    builder: (context, menu, _) => OutlinedButton(
                      onPressed: c.acting
                          ? null
                          : () => menu.isOpen ? menu.close() : menu.open(),
                      child: const Text('Snooze'),
                    ),
                  ),
                ),
                const SizedBox(width: Tokens.gap),
                Expanded(
                  flex: 10,
                  child: OutlinedButton(onPressed: c.acting
                        ? null
                        : () => c.remind(
                            reminder['id'] as String,
                            ReminderAction.acknowledge,
                          ), child: Text('Open')),
                ),
                const SizedBox(width: Tokens.gap),
                Expanded(
                  flex: 11,
                  child: FilledButton(onPressed: c.acting
                        ? null
                        : () => c.remind(
                            reminder['id'] as String,
                            ReminderAction.done,
                          ), child: Text('Done')),
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
            padding: const EdgeInsets.only(top: 12),
            child: Align(
              alignment: Alignment.centerLeft,
              child: OutlinedButton(onPressed: c.acting
                    ? null
                    : () => c.undo(ch!['undo_id'] as String), child: Text('Undo')),
            ),
          ),
        if (e.kind == 'error')
          Padding(
            padding: const EdgeInsets.only(top: 12),
            child: Align(
              alignment: Alignment.centerLeft,
              child: OutlinedButton(onPressed: c.waiting ? null : () => c.retry(text: c.retryText(e)), child: Text('Retry')),
            ),
          ),
      ],
    );
    final boxed = mine || e.kind == 'preview';
    return Container(
      key: ValueKey(e.id),
      margin: const EdgeInsets.only(top: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Align(
            alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
            child: Text('${mine ? 'YOU' : 'MINDER'} · $time', style: _meta),
          ),
          const SizedBox(height: 6),
          Container(
            margin: EdgeInsets.only(left: mine ? 44 : 0),
            padding: !boxed
                ? EdgeInsets.zero
                : mine
                ? const EdgeInsets.symmetric(horizontal: 12, vertical: 10)
                : const EdgeInsets.all(12),
            decoration: boxed
                ? BoxDecoration(
                    color: mine ? Tokens.panel : Tokens.voidColor,
                    border: Border.all(color: Tokens.edge),
                  )
                : null,
            child: body,
          ),
        ],
      ),
    );
  }

  String _action(String name) => switch (name) {
    'create_item' => 'CREATE ITEM',
    'complete_item' => 'COMPLETE ITEM',
    'create_profile' => 'ADD PROFILE',
    'update_profile' => 'CHANGE PROFILE',
    _ => 'CHANGE ITEM',
  };

  String _field(String name) => switch (name) {
    'start_at' => 'Starts',
    'duration_minutes' => 'Duration (minutes)',
    'priority' => 'Priority',
    'location' => 'Location',
    'list' => 'List',
    'profile' => 'Profile',
    'recurrence' => 'Repeats',
    'name' => 'Name',
    'remind_before_minutes' => 'Remind before (minutes)',
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
    return dateTime(date.toLocal());
  }
}
