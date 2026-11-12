# Minder

Personal task assistant for Android and web. Flutter provides chat and a calendar. Go and SQLite store items and chat history.

The app supports item and profile previews, Confirm, Discard, Undo, lists, recurrence, reminders and calendar queries. Cloudflare Access protects the deployed app.
The assistant uses Claude Haiku 5.5 through OpenRouter. Its key stays on the server.

## Local work

Podman runners pin Go 1.27.2 and Flutter 3.47.6. No host SDK is needed.

```sh
toolchain/run go go test -race ./...
toolchain/run go go vet ./...
toolchain/run flutter flutter test
toolchain/run flutter flutter analyze
toolchain/run flutter flutter build web --release --no-wasm-dry-run
```

Run the server with `ENV=development AUTH_MODE=dev`. Use `ENV=test AUTH_MODE=dev` for test fixtures. Production requires authentication.
Set `DATABASE_PATH`, `WEB_DIR` and `LISTEN_ADDR` to the local database, web build and listener.
Provide `OPENROUTER_API_KEY` through the environment. `MONTHLY_LIMIT_EUR` defaults to 5; `USD_TO_EUR` defaults to 1.
Production requires `CF_ISSUER`, `CF_AUDIENCE` and `OWNER_EMAIL`.
Set them in the deployment environment before starting this build. The server has no identity defaults.
`CF_ISSUER` must be an HTTPS origin. Unknown `ENV` values fail startup.
Budget and conversion settings must be finite positive numbers.
The meter reserves €0.02 times the conversion rate before each call. Unknown provider costs keep that reservation.

For controlled browser checks, run `python3 toolchain/model_fixture.py`, then `toolchain/local-check`.
The fixture listens on loopback port 8181. It never runs in production.
`local-check` starts a temporary SQLite-backed server and drives the compiled Flutter web app with Playwright.

Build an installable arm64 test APK with:

```sh
toolchain/run flutter flutter build apk --release --target-platform android-arm64 --split-per-abi
```

The build uses the debug signing key. Phone approval gates merging to `main`.

Requirements: [docs/requirements.md](docs/requirements.md).
Scope: [docs/m0-spec.md](docs/m0-spec.md).
Execution: [docs/m0-execution.md](docs/m0-execution.md).
Evidence: [docs/evidence/m0.md](docs/evidence/m0.md).
Architecture: [docs/architecture.md](docs/architecture.md).
Review: [docs/evidence/review-refactor.md](docs/evidence/review-refactor.md).
Brand: [brand/index.html](brand/index.html).
