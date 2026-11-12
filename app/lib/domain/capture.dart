import 'dart:typed_data';

class Capture {
  const Capture({
    required this.id,
    required this.status,
    required this.revision,
    this.sender = '',
    this.subject = '',
    this.error = '',
    this.deleted = false,
    this.coverageAcknowledged = false,
    this.questions = const [],
    this.coverage = const [],
    this.attachments = const [],
    this.candidates = const [],
  });
  final String id, status, sender, subject, error;
  final int revision;
  final bool deleted, coverageAcknowledged;
  final List<String> questions, coverage;
  final List<CaptureAttachment> attachments;
  final List<CaptureCandidate> candidates;
  bool get processing => status == 'received' || status == 'processing';
  bool get needsCoverageReview => coverage.isNotEmpty && !coverageAcknowledged;
}

class CaptureAttachment {
  const CaptureAttachment({
    required this.id,
    required this.name,
    required this.mediaType,
    required this.size,
  });
  final String id, name, mediaType;
  final int size;
}

class CaptureCandidate {
  const CaptureCandidate({
    required this.id,
    required this.state,
    required this.title,
    this.changeId = '',
    this.question = '',
    this.evidence = const [],
  });
  final String id, state, title, changeId, question;
  final List<String> evidence;
}

class AttachmentDownload {
  const AttachmentDownload(this.name, this.mediaType, this.bytes);
  final String name, mediaType;
  final Uint8List bytes;
}
