import 'dart:js_interop';

import 'package:web/web.dart' as web;

import '../domain/capture.dart';

Future<void> saveAttachment(AttachmentDownload attachment) async {
  final blob = web.Blob(
    [attachment.bytes.toJS].toJS,
    web.BlobPropertyBag(type: attachment.mediaType),
  );
  final url = web.URL.createObjectURL(blob);
  final anchor = web.HTMLAnchorElement()
    ..href = url
    ..download = attachment.name;
  web.document.body?.append(anchor);
  anchor.click();
  anchor.remove();
  await Future<void>.delayed(const Duration(seconds: 1));
  web.URL.revokeObjectURL(url);
}

@JS('Intl.DateTimeFormat')
extension type _DateTimeFormat._(JSObject _) implements JSObject {
  external factory _DateTimeFormat();
  external _Options resolvedOptions();
}

extension type _Options._(JSObject _) implements JSObject {
  external String get timeZone;
}

Future<String> deviceTimezone() async =>
    _DateTimeFormat().resolvedOptions().timeZone;
