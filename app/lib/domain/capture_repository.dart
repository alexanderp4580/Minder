import 'capture.dart';
import 'entry.dart';

abstract interface class CaptureRepository {
  Future<Capture> getCapture(String id);
  Future<Entry> answerCapture(
    String id,
    int revision,
    String text, {
    bool acknowledgeCoverage = false,
    String? candidateId,
  });
  Future<Entry> dismissCapture(String id, int revision);
  Future<Entry> retryCapture(String id);
  Future<Entry> deleteCapture(String id, int revision);
  Future<AttachmentDownload> downloadAttachment(
    String captureId,
    CaptureAttachment attachment,
  );
  Future<void> saveAttachment(AttachmentDownload attachment);
}
