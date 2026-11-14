import 'item.dart';

import 'dart:typed_data';

class Capture {
  const Capture({
    required this.id,
    required this.status,
    required this.revision,
    this.kind = '',
    this.sender = '',
    this.subject = '',
    this.error = '',
    this.unverified = 0,
    this.deleted = false,
    this.coverageAcknowledged = false,
    this.questions = const [],
    this.coverage = const [],
    this.attachments = const [],
    this.candidates = const [],
  });
  final String id, status, kind, sender, subject, error;
  final int revision;

  /// Appointments the assistant dropped because the email did not back them up.
  final int unverified;
  final bool deleted, coverageAcknowledged;
  final List<CaptureQuestion> questions;
  final List<String> coverage;
  final List<CaptureAttachment> attachments;
  final List<CaptureCandidate> candidates;

  /// A review of text the owner pasted into chat. Cards stored before the field existed
  /// have no kind and are forwarded emails.
  bool get pasted => kind == 'chat';
  bool get processing => status == 'received' || status == 'processing';
  bool get needsCoverageReview => coverage.isNotEmpty && !coverageAcknowledged;
}

/// Something the assistant needs to know, with optional quick replies.
class CaptureQuestion {
  const CaptureQuestion({
    this.id = '',
    required this.text,
    this.options = const [],
  });
  final String id, text;
  final List<String> options;
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
    this.options = const [],
    this.startAt = '',
    this.startDate = '',
    this.dueAt = '',
    this.dueDate = '',
    this.location = '',
    this.evidence = const [],
    this.details,
  });
  final String id, state, title, changeId, question;
  final String startAt, startDate, dueAt, dueDate, location;
  final List<String> options, evidence;
  final Item? details;
}

class AttachmentDownload {
  const AttachmentDownload(this.name, this.mediaType, this.bytes);
  final String name, mediaType;
  final Uint8List bytes;
}
