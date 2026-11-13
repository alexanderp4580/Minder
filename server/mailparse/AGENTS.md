# Purpose

Bounded MIME and document conversion adapter.

# Ownership

MIME traversal, safe HTML-to-text, ICS parsing, PDF conversion and per-part coverage.

# Local Contracts

HTML conversion preserves safe HTTPS, tel and mailto anchor targets as plain source text, and separates table cells and block headings. It ignores active content and does not fetch links. Preserve original bytes. Never execute attachments or load remote assets. Enforce part, byte, page and process limits.
Poppler tools run with fixed argument arrays and deadlines. Unreadable/unsupported input remains retained and visible.

# Work Guidance

Use synthetic MIME/calendar/document fixtures. No production email belongs in tests.

# Verification

`toolchain/run go go test -race ./mailparse`.

# Child DOX Index

None.
