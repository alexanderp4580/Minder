import 'package:flutter/material.dart';

import '../application/capture_controller.dart';
import '../domain/capture.dart';
import '../domain/capture_repository.dart';

/// Read-only source access from a calendar item.
class ItemSources extends StatefulWidget {
  const ItemSources({
    super.key,
    required this.repository,
    required this.sourceIds,
  });
  final CaptureRepository repository;
  final List<String> sourceIds;
  @override
  State<ItemSources> createState() => _ItemSourcesState();
}

class _ItemSourcesState extends State<ItemSources> {
  late Future<List<Capture>> _sources;
  late final CaptureController _controller;
  @override
  void initState() {
    super.initState();
    _controller = CaptureController(widget.repository, (_) {});
    _load();
  }

  void _load() {
    _sources = Future.wait(widget.sourceIds.map(widget.repository.getCapture));
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
    title: const Text('Email and attachments'),
    content: SizedBox(
      width: 420,
      child: FutureBuilder<List<Capture>>(
        future: _sources,
        builder: (context, snapshot) {
          if (snapshot.hasError) {
            return Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Text('The sources could not load.'),
                OutlinedButton(
                  onPressed: () => setState(_load),
                  child: const Text('Retry'),
                ),
              ],
            );
          }
          if (!snapshot.hasData) {
            return const SizedBox(
              height: 60,
              child: Center(child: CircularProgressIndicator()),
            );
          }
          return AnimatedBuilder(
            animation: _controller,
            builder: (context, _) => SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  for (final source in snapshot.data!)
                    if (!source.deleted) ...[
                      Text(
                        source.subject.isEmpty
                            ? 'Forwarded email'
                            : source.subject,
                      ),
                      for (final attachment in source.attachments)
                        OutlinedButton(
                          onPressed: _controller.busy
                              ? null
                              : () => _controller.download(source, attachment),
                          child: Text(attachment.name),
                        ),
                    ],
                  if (_controller.error != null) Text(_controller.error!),
                ],
              ),
            ),
          );
        },
      ),
    ),
    actions: [
      TextButton(
        onPressed: () => Navigator.of(context).pop(),
        child: const Text('Close'),
      ),
    ],
  );
}
