# Purpose

Bounded MIME and document conversion adapter.

# Ownership

MIME traversal, safe HTML-to-text, ICS parsing, PDF conversion and per-part coverage.

# Local Contracts

Preserve original bytes. Never execute attachments or load remote assets. Enforce part, byte, page and process limits.
Poppler tools run with fixed argument arrays and deadlines. Unreadable/unsupported input remains retained and visible.

# Work Guidance

Use synthetic MIME/calendar/document fixtures. No production email belongs in tests.

# Verification

`toolchain/run go go test -race ./mailparse`.

# Child DOX Index

None.
