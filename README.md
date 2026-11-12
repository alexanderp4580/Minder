# Minder

Personal task assistant for Android and web. Flutter provides chat and a calendar. Go and SQLite store items and chat history.

The app supports item and profile previews, Confirm, Discard, Undo, lists, recurrence, reminders and calendar queries. Cloudflare Access protects the deployed app.
The assistant uses Claude Haiku 5.5 through OpenRouter. Its key stays on the server.

## Local work

Podman runners pin Go 1.27.2 and Flutter 3.47.6. No host SDK is needed.

```sh
toolchain/check
toolchain/run flutter flutter build web --release --no-wasm-dry-run
```

`toolchain/check` enforces Go package boundaries, Flutter layers and the Agent0ai DOX tree, then runs the local suites.

Run the server with `ENV=development AUTH_MODE=dev`. Use `ENV=test AUTH_MODE=dev` for test fixtures. Production requires authentication.
Set `DATABASE_PATH`, `WEB_DIR` and `LISTEN_ADDR` to the local database, web build and listener.
Provide `OPENROUTER_API_KEY` through the environment. `MONTHLY_LIMIT_EUR` defaults to 5; `USD_TO_EUR` defaults to 1.
Production requires `CF_ISSUER`, `CF_AUDIENCE` and `OWNER_EMAIL`.
Set them in the deployment environment before starting this build. The server has no identity defaults.
`CF_ISSUER` must be an HTTPS origin. Unknown `ENV` values fail startup.
Budget and conversion settings must be finite positive numbers.
Set `LOG_LEVEL` (debug, info, warn, error; default info) and `LOG_FORMAT` (text or json; default text). The server logs requests, state changes, model calls, push attempts and reminder events. Debug adds transactions and worker ticks. Logs never hold secrets, tokens or message text.
The meter reserves €0.02 times the conversion rate before each call. Unknown provider costs keep that reservation.

For controlled browser checks, run `python3 toolchain/model_fixture.py`, then `toolchain/local-check`.
The fixture listens on loopback port 8181. It never runs in production.
`local-check` starts a temporary SQLite-backed server and drives the compiled Flutter web app with Playwright.

Build an installable arm64 test APK only when the owner authorizes the stopped build:

```sh
toolchain/run flutter flutter build apk --release --target-platform android-arm64 --split-per-abi
```

The build uses the debug signing key. Phone approval gates merging to `main`.

Requirements: [docs/requirements.md](docs/requirements.md).
Scope: [docs/m0-spec.md](docs/m0-spec.md).
Execution: [docs/m0-execution.md](docs/m0-execution.md).
Evidence: [docs/evidence/m0.md](docs/evidence/m0.md).
Architecture: [docs/architecture.md](docs/architecture.md).
Architecture checks: [docs/evidence/architecture-cleanup.md](docs/evidence/architecture-cleanup.md).
Brand: [brand/index.html](brand/index.html).

## Forwarded email capture

Email capture is opt-in. It uses direct Cloudflare Worker delivery, a private source directory and the existing model budget.
Configure the environment privately and follow [email capture operations](docs/email-capture-operations.md).
Personal sender values belong only in deployment environment/secrets; they are absent from this repository's examples.
