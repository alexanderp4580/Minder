import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../domain/chat_repository.dart';
import '../domain/entry.dart';
import '../domain/failure.dart';
import '../domain/item.dart';
import '../domain/item_repository.dart';
import '../domain/push_repository.dart';
import 'wire.dart';
import '../domain/capture.dart';
import '../domain/capture_repository.dart';
import 'attachment_download.dart' as downloads;

class ChatApi
    implements
        ChatRepository,
        PushRepository,
        ItemRepository,
        CaptureRepository {
  ChatApi(this.base, {http.Client? client, this.authorization})
    : _client = client ?? http.Client();
  final String base;
  final http.Client _client;
  final Future<Map<String, String>> Function()? authorization;
  final _streamClients = <http.Client>{};
  Uri _uri(String path) => Uri.parse('$base$path');
  Future<Map<String, String>> _headers() async => {
    'Content-Type': 'application/json',
    ...?(await authorization?.call()),
  };
  Future<dynamic> _request(String method, String path, {Object? body}) async {
    try {
      final req = http.Request(method, _uri(path));
      req.headers.addAll(await _headers());
      if (body != null) req.body = jsonEncode(body);
      final stream = await _client
          .send(req)
          .timeout(const Duration(seconds: 20));
      final res = await http.Response.fromStream(stream)
          .timeout(const Duration(seconds: 20));
      dynamic data;
      try {
        data = jsonDecode(res.body);
      } catch (_) {
        if (res.statusCode == 401 || res.statusCode == 403) {
          throw const RepositoryFailure(
            'Sign in to continue.',
            unauthorized: true,
          );
        }
        throw const RepositoryFailure(
          'The server reply was unreadable. Try again.',
        );
      }
      if (res.statusCode >= 400) {
        final j = data as Map<String, dynamic>;
        throw RepositoryFailure(
          j['message'] as String? ?? 'Try again.',
          unauthorized: res.statusCode == 401,
          replacement: j['entry'] is Map<String, dynamic>
              ? entryFromJson(j['entry'] as Map<String, dynamic>)
              : null,
        );
      }
      return data;
    } on RepositoryFailure {
      rethrow;
    } catch (_) {
      throw const RepositoryFailure(
        'Check your connection. Then try again.',
        transport: true,
      );
    }
  }

  @override
  Future<PushConfiguration> pushConfig() async => pushConfigurationFromJson(
    (await _request('GET', '/v1/push/config')) as Map<String, dynamic>,
  );
  @override
  Future<void> registerDevice(String id, String token) async {
    await _request(
      'POST',
      '/v1/push/devices',
      body: {'device_id': id, 'token': token},
    );
  }

  @override
  Future<List<Entry>> history(int after) async =>
      ((await _request('GET', '/v1/chat?after=$after&limit=200')) as List)
          .map((j) => entryFromJson(j as Map<String, dynamic>))
          .toList();
  @override
  Future<Entry> send(String text) async {
    final j = await _request(
      'POST',
      '/v1/chat/messages',
      body: {'text': text},
    ) as Map<String, dynamic>;
    return entryFromJson(j['entry'] as Map<String, dynamic>);
  }

  @override
  Future<List<Item>> itemsBetween(
    DateTime from,
    DateTime to,
  ) async => ((await _request(
    'GET',
    '/v1/items?from=${from.toUtc().toIso8601String()}'
        '&to=${to.toUtc().toIso8601String()}'
        '&timezone=${Uri.encodeQueryComponent(await downloads.deviceTimezone())}',
  )) as List).map((j) => itemFromJson(j as Map<String, dynamic>)).toList();
  Future<Entry> _post(String route) async =>
      entryFromJson(await _request('POST', route) as Map<String, dynamic>);
  @override
  Future<Entry> confirm(String changeId) =>
      _post('/v1/changes/$changeId/confirm');
  @override
  Future<Entry> discard(String changeId) =>
      _post('/v1/changes/$changeId/discard');
  @override
  Future<Entry> setChangeReminders(String changeId, List<int> offsets) async =>
      entryFromJson(
        await _request(
          'POST',
          '/v1/changes/$changeId/reminders',
          body: {'offsets': offsets},
        ) as Map<String, dynamic>,
      );
  @override
  Future<Entry> undo(String undoId) => _post('/v1/undo/$undoId');
  @override
  Future<Entry> actOnReminder(String reminderId, ReminderAction action) =>
      _post('/v1/reminders/$reminderId/${_wire(action)}');
  static String _wire(ReminderAction a) => switch (a) {
    ReminderAction.done => 'done',
    ReminderAction.acknowledge => 'ack',
    ReminderAction.snooze15 => 'snooze15',
    ReminderAction.snooze60 => 'snooze60',
    ReminderAction.tonight => 'tonight',
    ReminderAction.tomorrow => 'tomorrow',
  };
  @override
  Stream<Entry> watch(int after) async* {
    final client = http.Client();
    _streamClients.add(client);
    try {
      final req = http.Request('GET', _uri('/v1/events?after=$after'));
      req.headers.addAll(await _headers());
      req.headers['Accept'] = 'text/event-stream';
      final res = await client.send(req).timeout(const Duration(seconds: 20));
      if (res.statusCode != 200) {
        throw RepositoryFailure(
          'Sign in to continue.',
          unauthorized: res.statusCode == 401,
        );
      }
      final data = <String>[];
      await for (final line
          in res.stream
              .timeout(const Duration(seconds: 45))
              .transform(utf8.decoder)
              .transform(const LineSplitter())) {
        if (line.startsWith('data: ')) {
          data.add(line.substring(6));
        }
        if (line.isEmpty && data.isNotEmpty) {
          final j = jsonDecode(data.join('\n'));
          data.clear();
          if (j is Map<String, dynamic> && j['id'] != null) {
            yield entryFromJson(j);
          }
        }
      }
    } finally {
      _streamClients.remove(client);
      client.close();
    }
  }

  @override
  Future<Capture> getCapture(String id) async => captureFromJson(
    await _request('GET', '/v1/captures/${Uri.encodeComponent(id)}')
        as Map<String, dynamic>,
  );
  @override
  Future<Entry> answerCapture(
    String id,
    int revision,
    String text, {
    bool acknowledgeCoverage = false,
    String? candidateId,
  }) async => entryFromJson(
    await _request(
      'POST',
      '/v1/captures/${Uri.encodeComponent(id)}/answers',
      body: {
        'revision': revision,
        'text': text,
        'acknowledge_coverage': acknowledgeCoverage,
        'candidate_id': ?candidateId,
      },
    ) as Map<String, dynamic>,
  );
  @override
  Future<Entry> dismissCapture(String id, int revision) async => entryFromJson(
    await _request(
      'POST',
      '/v1/captures/${Uri.encodeComponent(id)}/dismiss',
      body: {'revision': revision},
    ) as Map<String, dynamic>,
  );
  @override
  Future<Entry> retryCapture(String id) =>
      _post('/v1/captures/${Uri.encodeComponent(id)}/retry');
  @override
  Future<Entry> deleteCapture(String id, int revision) async => entryFromJson(
    await _request(
      'POST',
      '/v1/captures/${Uri.encodeComponent(id)}/delete',
      body: {'revision': revision},
    ) as Map<String, dynamic>,
  );
  @override
  Future<AttachmentDownload> downloadAttachment(
    String captureId,
    CaptureAttachment attachment,
  ) async {
    try {
      final response = await _client
          .get(
            _uri(
              '/v1/captures/${Uri.encodeComponent(captureId)}/attachments/${Uri.encodeComponent(attachment.id)}',
            ),
            headers: await _headers(),
          )
          .timeout(const Duration(seconds: 30));
      if (response.statusCode != 200) {
        throw const RepositoryFailure(
          'The attachment could not be downloaded.',
        );
      }
      return AttachmentDownload(
        attachment.name,
        attachment.mediaType,
        response.bodyBytes,
      );
    } on RepositoryFailure {
      rethrow;
    } catch (_) {
      throw const RepositoryFailure(
        'Check your connection. Then try again.',
        transport: true,
      );
    }
  }

  @override
  Future<void> saveAttachment(AttachmentDownload attachment) =>
      downloads.saveAttachment(attachment);
  void close() {
    for (final client in _streamClients.toList()) {
      client.close();
    }
    _streamClients.clear();
    _client.close();
  }
}
