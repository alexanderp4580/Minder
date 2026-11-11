# Handoff: Minder chat and M0

State on 2026-11-11. Branch `main`. Nothing is committed. This file is temporary. Delete it when the M0 slice starts.

## Where the project stands

- The app is one chat screen in Flutter (Android and web). The backend is Go and SQLite on Coolify. No code exists yet.
- The docs describe the design and the first build slice. The owner has not approved the slice.
- Cloudflare Access is set up for `minder.liveleds.io`.

## Read in this order

1. `docs/m0-spec.md`: scope, API, task plan, acceptance criteria, approval packet.
2. `docs/architecture.md`: Go and Flutter layers, Cloudflare setup, checks with sources, spikes.
3. `docs/foundation.md`: colour, type, space and chat components.
4. `docs/screens.md`: the chat screen.
5. `docs/requirements.md` §17: all decisions.
6. `docs/mockups/chat.html` and `chat.png`: the screen draft.

## Decisions made by the owner

- Chat is the whole app. No other screen is needed now.
- Public HTTPS behind Cloudflare Access. Web signs in through the browser. Android signs in through managed OAuth.
- A reminder is a notification and a chat message. Done and Snooze act at once with Undo.
- The model is Claude Haiku 5.5 through OpenRouter. Voice uses a speech model through OpenRouter. Input is English only.
- The monthly limit is €5 for model and speech.
- The Go service serves the web build. The open app uses server-sent events. Polling every 5 seconds is the fallback.
- A preview card never expires. Confirm fails if the item changed.
- Coolify backups only, on the same server. Riverpod for state.
- Edit is not in M0. Old mockups stay as unmaintained drafts.
- No sub-agents. I write the code myself.
- Application ID is `io.liveleds.minder`, the reverse of `liveleds.io`.

## Cloudflare (done)

- Hostname `minder.liveleds.io`, proxied. Team domain `liveleds.cloudflareaccess.com`. Access app id `5eaa7708-2f25-43e6-b570-af5422bd5941`.
- The policy allows one email. Managed OAuth is on with the redirect `io.liveleds.minder:/oauth/callback`.
- Full table and rules: `docs/architecture.md`, "Cloudflare setup".
- Not tested: a native Android sign-in. The first M0 spike proves it. The fallback is a service token.

## Secrets

- `cloud_flare_access_token` in TokenCrypt is a working API token. It can edit Access applications and DNS on `liveleds.io`. Revoke it when the work ends if the owner does not want it kept.
- `openrouter_api_key` does not exist yet. The owner adds it in the TokenCrypt website. Request it with `tokencrypt request` for real-call checks.
- A TokenCrypt grant lasts 30 minutes. The vault locks again, so the owner may need to unlock it.
- Telegram delivery reads the bot token from `~/.config/tokencrypt/telegram.env`.

## Next

1. The owner approves the slice. The spec proposes a git worktree `m0-chat` from `main`.
2. Task 0: pinned podman images for Go and Flutter, and the `server/` and `app/` skeletons with their `AGENTS.md` files. The Steam Deck has Podman and no Go, Flutter or Docker.
3. Tasks 1 to 7 are Part A and run on the local machine. Tasks 8 to 10 are Part B and need the Coolify application for `minder.liveleds.io`.
4. Part A phone check: `tailscale serve` for HTTPS. To check that HTTPS is enabled on the tailnet.

## Open and unverified

- Flutter OAuth library support for RFC 8707 and dynamic client registration.
- Cloudflare Access with a held event stream for 5 minutes.
- `tailscale serve` HTTPS on the tailnet and the phone.
- The speech model and its real cost. The prices I read look like USD per second.
- The JSON model package, the icon package and web push are not researched.

## Docs state

- Updated for the chat-only direction: `PRODUCT.md`, `README.md`, `docs/requirements.md`, `docs/master-plan.md`, `docs/screens.md`, `docs/AGENTS.md`, `docs/mockups/AGENTS.md`.
- `stop-slop` did not run over the new files `architecture.md`, `foundation.md` and `m0-spec.md`. Run it before the owner reads them.
- `brand/index.html` and `docs/mockups/common.css` carry the new colour and component rules. Nobody checked the brand page at 1280 px and 390 px.
- `chat.png` shows a nag frame and a countdown. M0 omits both.
- Every other `.png` in `docs/mockups/` is an earlier draft. `docs/mockups/AGENTS.md` says so.
- Contrast ratios for `foundation.md` are not checked.

## Checks run

- `git diff --check` passes.
- `impeccable detect` on `chat.html` gave two warnings on shared components (`.ph`, `.cap`). Not fixed.
- No code exists, so no tests ran.
