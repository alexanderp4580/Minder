import 'package:flutter/material.dart';

import '../application/capture_controller.dart';
import '../application/chat_controller.dart';
import '../domain/capture.dart';
import '../domain/capture_repository.dart';
import 'tokens.dart';

class CaptureView extends StatefulWidget {
  const CaptureView({super.key, required this.capture, required this.chat});
  final Capture capture;
  final ChatController chat;
  @override
  State<CaptureView> createState() => _CaptureViewState();
}

class _CaptureViewState extends State<CaptureView> {
  final _answer = TextEditingController();
  String? _candidateId;
  CaptureController? _controller;
  @override
  void initState() {
    super.initState();
    final repository = widget.chat.repository;
    if (repository is CaptureRepository) {
      _controller = CaptureController(
        repository as CaptureRepository,
        widget.chat.receiveCaptureEntry,
      );
    }
  }

  @override
  void dispose() {
    _controller?.dispose();
    _answer.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;
    if (controller == null) {
      return Text(
        widget.capture.deleted
            ? 'Email source deleted.'
            : widget.capture.subject,
      );
    }
    return AnimatedBuilder(
      animation: controller,
      builder: (context, _) => _content(controller),
    );
  }

  Widget _content(CaptureController controller) {
    final capture = widget.capture;
    if (capture.deleted) {
      return const Text('Email source deleted. Confirmed items remain.');
    }
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Tokens.panel,
        border: Border.all(color: Tokens.edge),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'EMAIL',
            style: TextStyle(
              fontSize: 12,
              color: Tokens.muted,
              letterSpacing: 1,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            capture.subject.isEmpty ? 'Forwarded email' : capture.subject,
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          if (capture.sender.isNotEmpty)
            Text(capture.sender, style: const TextStyle(color: Tokens.muted)),
          const SizedBox(height: 8),
          Text(switch (capture.status) {
            'received' => 'Received. Waiting to process.',
            'processing' => 'Reading email and attachments…',
            'needs-answer' => 'Some details need your answer.',
            'failed' =>
              capture.error.isEmpty
                  ? 'Processing failed. The original is available.'
                  : capture.error,
            'reviewed' => 'Review complete.',
            _ => 'Review the proposals in chat.',
          }),
          if (!capture.processing &&
              capture.candidates.isEmpty &&
              capture.status != 'failed')
            const Padding(
              padding: EdgeInsets.only(top: 8),
              child: Text(
                'No actionable items found. Tell Minder what to capture.',
              ),
            ),
          for (final candidate in capture.candidates)
            ExpansionTile(
              tilePadding: EdgeInsets.zero,
              childrenPadding: const EdgeInsets.only(bottom: 8),
              title: Text(
                candidate.title.isEmpty ? 'Proposed item' : candidate.title,
              ),
              subtitle: Text(
                candidate.question.isEmpty
                    ? candidate.state
                    : candidate.question,
              ),
              children: [
                if (candidate.state == 'pending' ||
                    candidate.state == 'needs-answer')
                  TextButton(
                    onPressed: () =>
                        setState(() => _candidateId = candidate.id),
                    child: const Text('Clarify this item'),
                  ),
                for (final evidence in candidate.evidence)
                  Align(
                    alignment: Alignment.centerLeft,
                    child: Text(
                      evidence,
                      style: const TextStyle(color: Tokens.muted),
                    ),
                  ),
              ],
            ),
          for (final question in capture.questions)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(question),
            ),
          for (final warning in capture.coverage)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(warning, style: const TextStyle(color: Tokens.muted)),
            ),
          if (!capture.processing && capture.status != 'reviewed') ...[
            const SizedBox(height: 12),
            TextField(
              controller: _answer,
              minLines: 1,
              maxLines: 4,
              decoration: InputDecoration(
                labelText: _candidateId == null
                    ? 'Answer or clarify this email'
                    : 'Clarify the selected item',
              ),
            ),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: controller.busy
                  ? null
                  : () async {
                      if (_answer.text.trim().isEmpty) return;
                      await controller.answer(
                        capture,
                        _answer.text,
                        candidateId: _candidateId,
                      );
                      if (mounted && controller.error == null) _answer.clear();
                    },
              child: const Text('Send answer'),
            ),
          ],
          if (capture.needsCoverageReview && !capture.processing)
            OutlinedButton(
              onPressed: controller.busy
                  ? null
                  : () => controller.answer(
                      capture,
                      'Continue with readable content.',
                      acknowledgeCoverage: true,
                    ),
              child: const Text('Continue with readable content'),
            ),
          if (capture.status == 'failed')
            OutlinedButton(
              onPressed: controller.busy
                  ? null
                  : () => controller.retry(capture),
              child: const Text('Retry email'),
            ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final attachment in capture.attachments)
                OutlinedButton(
                  onPressed: controller.busy
                      ? null
                      : () => controller.download(capture, attachment),
                  child: Text(attachment.name),
                ),
            ],
          ),
          TextButton(
            onPressed: controller.busy
                ? null
                : () => controller.delete(capture),
            child: const Text('Delete email source…'),
          ),
          if (controller.error != null)
            Text(
              controller.error!,
              style: const TextStyle(color: Tokens.alert),
            ),
        ],
      ),
    );
  }
}
