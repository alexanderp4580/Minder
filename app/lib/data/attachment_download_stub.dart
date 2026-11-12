import 'package:flutter/services.dart';

import '../domain/capture.dart';

const _channel = MethodChannel('minder/attachments');
Future<void> saveAttachment(AttachmentDownload attachment) async {
  await _channel.invokeMethod<void>('save', {
    'name': attachment.name,
    'media_type': attachment.mediaType,
    'bytes': attachment.bytes,
  });
}

Future<String> deviceTimezone() async {
  try {
    return await _channel.invokeMethod<String>('timezone') ?? 'UTC';
  } catch (_) {
    return 'UTC';
  }
}
