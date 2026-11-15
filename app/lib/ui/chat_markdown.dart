import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:markdown/markdown.dart' as md;

import '../domain/links.dart';
import '../domain/link_launcher.dart';
import 'tokens.dart';

/// Minder's message text with Markdown shown as formatting. Headings keep body size.
/// Phone numbers, emails and web addresses are tappable as in [LinkedText]. Only an
/// HTTPS, tel or mailto link opens, after a tap; images are never fetched.
class ChatMarkdown extends StatelessWidget {
  const ChatMarkdown({super.key, required this.text, this.launcher});
  final String text;
  final LinkLauncher? launcher;

  static const _opens = {'https', 'tel', 'mailto'};
  static const _body = TextStyle(color: Tokens.text);
  static const _strong = TextStyle(
    color: Tokens.text,
    fontWeight: FontWeight.w600,
  );

  Future<void> _open(BuildContext context, String? href) async {
    final uri = href == null ? null : Uri.tryParse(href);
    if (uri == null || !_opens.contains(uri.scheme) || launcher == null) return;
    var opened = false;
    try {
      opened = await launcher!.open(uri);
    } catch (_) {}
    if (!opened && context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Could not open this link.')),
      );
    }
  }

  @override
  Widget build(BuildContext context) => MarkdownBody(
    data: text,
    inlineSyntaxes: [_ContactSyntax()],
    onTapLink: (_, href, _) => _open(context, href),
    imageBuilder: (_, _, _) => const SizedBox.shrink(),
    styleSheet: MarkdownStyleSheet(
      p: _body,
      strong: _strong,
      em: _body.copyWith(fontStyle: FontStyle.italic),
      del: _body.copyWith(decoration: TextDecoration.lineThrough),
      a: _body.copyWith(decoration: TextDecoration.underline),
      h1: _strong,
      h2: _strong,
      h3: _strong,
      h4: _strong,
      h5: _strong,
      h6: _strong,
      listBullet: _body,
      code: _body.copyWith(backgroundColor: Tokens.panel, fontSize: 13),
      codeblockDecoration: const BoxDecoration(
        color: Tokens.panel,
        border: Border.fromBorderSide(BorderSide(color: Tokens.edge)),
      ),
      blockquoteDecoration: const BoxDecoration(
        border: Border(left: BorderSide(color: Tokens.edge, width: 3)),
      ),
      tableBorder: TableBorder.all(color: Tokens.edge),
      horizontalRuleDecoration: const BoxDecoration(
        border: Border(top: BorderSide(color: Tokens.edge)),
      ),
      blockSpacing: 8,
    ),
  );
}

/// Links plain phone numbers, emails and web addresses by the same rules as
/// [findLinks], which judges each candidate inside the whole line.
class _ContactSyntax extends md.InlineSyntax {
  _ContactSyntax() : super(linkCandidate.pattern, caseSensitive: false);

  @override
  bool onMatch(md.InlineParser parser, Match match) {
    final matched = match[0]!;
    var offset = 0;
    LinkSpan? found;
    for (final span in findLinks(parser.source)) {
      if (offset == match.start) found = span;
      offset += span.text.length;
      if (offset > match.start) break;
    }
    if (found == null || found.target == null) {
      parser.addNode(md.Text(matched));
      return true;
    }
    parser.addNode(
      md.Element('a', [md.Text(found.text)])
        ..attributes['href'] = found.target.toString(),
    );
    final rest = matched.substring(found.text.length);
    if (rest.isNotEmpty) parser.addNode(md.Text(rest));
    return true;
  }
}
