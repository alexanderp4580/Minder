# Email capture verification

Date: 2026-11-12. Branch: `notification-body-tap`. App: 0.5.0+18.
Scope: [approved spec](../superpowers/specs/2026-11-12-email-capture-design.md).
Activation: [operations runbook](../email-capture-operations.md).

## Local checks

The final local run used synthetic mail and isolated temporary SQLite/source directories. No real forwarded email or private sender value appears in fixtures.

| Command | Exit | Observed result |
|---|---|---|
| `toolchain/check` | 0 | Go race tests/vet/format; 61 Flutter tests, analyzer and format; 5 Worker tests; toolchain/hook, architecture, DOX and orbital checks |
| `toolchain/run flutter flutter build web --release --no-wasm-dry-run` | 0 | Release web bundle |
| `toolchain/run flutter flutter build apk --release --target-platform android-arm64` | 0 | Release arm64 APK |
| `toolchain/local-check` | 0 | Preview, confirm, discard, updates, completion, Undo, reload and offline retry on compiled web |
| `toolchain/reminder-local-check` | 0 | Reminder entry and Done action |
| `toolchain/calendar-local-check` | 0 | Confirmed item on its calendar day and navigation back to chat |
| `toolchain/email-local-check` | 0 | Signed ingress, duplicate receipt, targeted answer, authenticated attachment download, individual confirmations, travel calendar and reviewed cancellation |

The email browser check also inspects the temporary database: arrival creates no items, Confirm persists provenance and date ranges, and a cancellation leaves the unrelated flight intact.
Poppler ran separately in the Go runner with `poppler-utils` installed: `go test -v ./mailparse -run TestPDFTextAndPageLimit` exited 0 for text extraction and a 51-page coverage limit. The ordinary Go runner skips that runtime-dependent test when Poppler is absent.

## Review and regressions

The inline review covered admission/auth separation, source ownership, model budgets, concurrent deletion, stale previews, shutdown, source downloads and unrelated chat state. Project instructions require inline work without sub-agents.
Regression tests reproduced and then passed for deletion during attachment writes, stale revision notifications, lease release on shutdown, declared MIME charsets, existing/pending duplicate bookings, answerable zero-item captures and dropped candidates during clarification. Event edits recompute derived duration; item reads retain source links.

The existing keyboard-following implementation and tests remain separate working-tree changes. The full suite includes those regressions.

## Limits

No live Cloudflare, DNS, Access or production configuration changed. Verify actual trusted authentication headers, service-token policy, real delivery failure behaviour and the selected model's document capability/retention settings during activation.
The APK compiles; no emulator was available to verify native attachment opening. Physical-phone email review, download and notification checks remain pending. No APK was delivered for this slice. Phone approval gates merging.
Local model fixtures verify integration contracts, not real-world extraction accuracy. Cloudflare and the model provider receive readable content as specified; this pipeline is not end-to-end encrypted.
