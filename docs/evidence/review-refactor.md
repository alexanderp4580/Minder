# Review and refactor

Date: 2026-11-12. Branch: `m0-chat`. Workspace: `Minder-m0-chat`.
Owner request: review the handoff, remove slop, preserve architecture and keep Cloudflare out of main.
The owner requested autonomous work. The M0 execution instructions require inline work without sub-agents.

## Scope and acceptance

- Keep the current product behaviour and wire contracts.
- Isolate provider authentication from HTTP routes and process wiring.
- Remove hardcoded deployment identities and validate startup settings.
- Keep JSON and Android channel messages in Flutter's data layer.
- Reproduce and fix request-ordering and notification status errors.
- Verify Go, Flutter and compiled web scenarios. Preserve `main` and the existing worktree.

## Findings and changes

`access` owns authentication settings and Cloudflare JWT verification.
The API accepts a request authenticator interface. `cmd/minder` wires it without provider headers or identity values.
Missing issuer, audience or owner settings stop startup. Development authentication requires an explicit development or test environment.
JWT tests cover signature, issuer, audience, owner, expiry and the assertion header.

`config.Load` owns process settings. It rejects unknown environments and custom production model endpoints.
It also rejects zero, negative, NaN and infinite budget values.
Go module metadata now marks direct imports as direct dependencies. Dependency versions did not change.

Flutter domain models contain typed values. `data/wire.dart` decodes JSON at the API boundary.
Push repositories return typed configuration. Notification controllers use a typed device interface.
`data/notification_device.dart` owns Android channel calls, action validation and channel payloads.
Dispose removes the notification callback.

Two calendar tests reproduced request-ordering errors before the fix:
- An earlier month response replaced the current month's items.
- An earlier failure ended a newer load and showed the wrong error.
The controller now ignores responses and errors from superseded requests.

Two notification tests reproduced stale enabled state before the fix:
- A failed device registration on resume left notifications enabled.
- Revoking Android permission left notifications enabled after resume.
The controller now enables notifications after registration and checks revoked permission on resume.

The architecture document describes the current source and routes.
The handoff separates local changes from production and phone status.

## Verification

| Command or check | Result |
|---|---|
| `toolchain/run go go test -race ./...` | Passed, exit 0 |
| `toolchain/run go go vet ./...` | Passed, exit 0 |
| `toolchain/run go gofmt -l .` | Passed, no output |
| `toolchain/run flutter flutter test` | Passed, 29 tests, including goldens |
| `toolchain/run flutter flutter analyze` | Passed, no issues |
| `toolchain/run flutter dart format --set-exit-if-changed lib test` | Passed, zero changes |
| `toolchain/run flutter flutter build web --release --no-wasm-dry-run` | Passed, exit 0 |
| `toolchain/calendar-local-check` | Passed, calendar placement and back navigation |
| `toolchain/reminder-local-check` | Passed, reminder display and Done |
| `toolchain/local-check`, with the model fixture | Passed, preview, Confirm, query, Discard, change, Done, Undo, reload and offline retry |
| Production process without identity settings | Passed, nonzero exit before database creation |
| Provider references in `cmd/minder` and `api/api.go` | None |
| `git diff --check` | Passed |

The web build reports a missing Cupertino font family. The app has no Cupertino imports.
Flutter's container prints a root-user warning and notices for newer dependency versions.
These checks use the pinned SDKs and current dependency versions.

Final review ran inline. It covered authentication gates, typed-nil handling, startup errors, wire compatibility and controller request ordering.
The branch retains its concrete store transaction boundaries. This review adds no storage abstraction or dependencies.
Browser screenshots from this run remain in `.local/review-evidence/`; tracked screenshots retain their prior contents.

## Release requirements

On 2026-11-12, the owner authorized setting `CF_ISSUER`, `CF_AUDIENCE` and `OWNER_EMAIL` in Coolify.
All three now match the live Cloudflare application and its single-email owner policy.
Read-back verified exact values, runtime enabled, build-time disabled and preview disabled.
Coolify reported `running:healthy` after the update. The settings apply on the next deployment.
The API credentials stayed in process memory. No credential values entered files or output.
No deploy, APK build, phone check or merge forms part of this review.
The owner stopped the APK 0.4.0 build. The handoff retains that restriction.

## Remaining product checks

The handoff still requires phone checks of lists, profiles, recurrence, follow-up questions and the calendar.
Date-only interpretation needs a product rule and real-model checks.
A weaker model can omit a requested change. This review does not change the model prompt or tools.
The app uses Material icons; the design guide calls for Carbon icons.

## Architecture assessment

Flutter has inward dependencies: UI uses controllers, controllers use domain interfaces, and data adapters implement those interfaces.
The domain has no Flutter, HTTP, JSON or platform channel dependencies.
Controllers use Flutter's `ChangeNotifier` for state notifications.

Go has package separation but still couples `command`, `assistant`, `api` and the reminder worker to concrete store types.
Command operations also use concrete `store.Tx` transactions.
`store.ReminderAction` contains snooze policy and coordinates completion, audit and chat results alongside SQL.
A strict clean architecture would put that orchestration in application services and use persistence interfaces.
The refactor does not complete that separation.
