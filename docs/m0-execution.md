# M0 execution

Approved on 2026-11-11: commit the draft, complete M0 autonomously, test local Flutter web first, deliver the APK.
Spec: [m0-spec.md](m0-spec.md). Branch: `m0-chat`, isolated checkout `Minder-m0-chat`.
No sub-agents. Record unresolved guidance for later. Phone approval still gates merging to main.

## Tasks

- [ ] 0. Pin existing Go 1.27.2 and Flutter 3.47.6 Podman images; generate Android/web skeletons.
- [ ] 1. SQLite schema and domain validation; prove migrations repeat safely and rows survive reopen.
- [ ] 2. Preview/confirm/discard/undo transactions; prove idempotency, optimistic conflicts, audit and one-use undo.
- [ ] 3. Cloudflare JWT verifier and production guard; test signature, expiry, audience and owner rejection.
- [ ] 4. Chat history and SSE; prove ordered cursors, updates, reconnect replay and heartbeat.
- [ ] 5. OpenRouter tool client and assistant; fake boundary tests, clarify ambiguous requests, meter costs and block excess.
- [ ] 6. Flutter tokens and chat widgets; widget and golden tests for user, reply, preview, result, error and waiting.
- [ ] 7. Controller/API; prove send, confirm, discard, undo, retry and reconnect against controlled transport.
- [ ] 8. Android OAuth discovery, PKCE and refresh; web uses same-origin cookies. Test protocol boundaries.
- [ ] 9. Local web/server end-to-end first, then production container and Coolify deployment where credentials permit.
- [ ] 10. Build arm64 APK, deliver through configured Telegram chat, record evidence and outstanding external checks.

## Interfaces and checks

Use the API in m0-spec.md. A durable increasing event cursor carries new entries and changes to older entries.
SQLite transactions apply each proposal once. Undo checks the current version and never overwrites later changes.
Only the server holds the OpenRouter key. Dev mode refuses production. No fake model in production.

For every behavior: write its failing test, run it, implement it, run the full affected suite.
Go checks: `toolchain/run go go test -race ./...`, `toolchain/run go go vet ./...` (exit 0).
Flutter checks: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze` (exit 0).
Local browser: create, confirm, query, change, discard, complete, undo, reload and offline retry.
Build: Flutter web and Android APK. Validate HTTP health and stream through the real server.

## Decisions

- The user's start instruction approves the existing architectural spec and task plan, with review guidance deferred.
- Use hand-written JSON DTOs for M0. No code generator or icon package is needed for this slice.
- Model cost is returned in USD. Use an explicit configurable USD-to-EUR conversion and reserve request headroom.
- Preserve the branch until phone approval. No merge is part of this run.
