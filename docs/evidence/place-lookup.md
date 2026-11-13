# Place lookup with Serper

Date: 2026-11-13. Branch `place-lookup` (worktree `Minder-place-lookup`, from `main` at `127e4c6`). App version 0.8.0+24. Not merged, not deployed.
Spec: [place lookup design](../superpowers/specs/2026-11-13-place-lookup-design.md). Plan: [place lookup plan](../superpowers/plans/2026-11-13-place-lookup.md). Mock: `../mockups/place-lookup.png`.

## Behaviour

- With `SERPER_API_KEY` set, the model may call `offer_lookup` for a request that names a venue without an address. The server stores a fixed-format offer entry and serves it with computed `options` (`Skip`, `Look it up`). The app draws it as a card with two buttons while it is the newest entry.
- The server runs the search, not the model: only when the owner's text is `Look it up` and the newest entry before it, skipping error entries and earlier `Look it up` messages, is an offer. It searches the offer's query and adds at most three places to the one normal model call as untrusted data. The preview then carries the address in location and the website in notes.
- Skip, other text, an answered offer, or no key runs no search. A failed or empty search says so, and the item is still proposed without an address.
- Logs carry `query_len`, `places`, `status` and `duration_ms`; never the query, a place or the key.

## Checks

| Check | Result |
|---|---|
| Red tests before each change | Domain, config, Serper adapter, assistant gate, openrouter tool exposure, app card and wire tests each failed first (undefined symbols or missing behaviour) |
| Startup log mutation | Reverting `serper_key_set` in `logStartup` failed `TestStartupSummaryShowsSettingsButNeverSecrets`; restoring it passed |
| Prompt order | Found by the browser fixture: the place rule sat after the untrusted item data. `TestWithoutAFinderThereIsNoOfferToolOrRule` failed, then passed once the rule joined the other rules |
| Retry | `TestRetryAfterAFailedReplyStillSearches` failed first; `TestNoSearchOnceTheOfferWasAnswered` guards the stale case |
| API wire | `TestChatServesOptionsOnlyForAPlaceLookupOffer` checks `/v1/chat` and `/v1/chat?after=1`; it passed on first run because Task 1 already added the marshalling (a boundary test, not a mutation-checked one) |
| Live Serper | One real query for `SnowWorld Landgraaf` through TokenCrypt returned `Witte Wereld 1` and `https://www.snowworld.com/landgraaf?utm_source=gmb_listing&utm_medium=organic`, 1,170 ms, status 200. The throwaway test file was deleted |
| Flutter | `toolchain/run flutter flutter test`: 102 passed; `flutter analyze` and `dart format` clean |
| Browser, place lookup | `toolchain/place-lookup-local-check` passed against compiled web, the model fixture and a fake Serper: the offer ran no search, Look it up ran exactly one, the preview showed `Witte Wereld 1, Landgraaf`, Skip ran none. Screenshots: `place-lookup-offer.png`, `place-lookup-preview.png` |
| Browser, calendar and reminder | `toolchain/calendar-local-check` and `toolchain/reminder-local-check` passed |
| Browser, chat | `toolchain/local-check` fails at its offline step: it expects the "Check your connection." block and Retry that the delete-items slice removed (`app/lib/application/AGENTS.md`). The steps before it (preview, confirm, query, discard, change, complete, undo) passed. Not caused by this change; not fixed here. I did not run it on `main` to prove that |
| `toolchain/check` | Exit 0 on the final tree: checker tests, hook tests, Worker tests, architecture, DOX, orbital exports, Go race tests, vet, gofmt, Flutter tests, analyze, format, `git diff --check` |

Artifact: `.local/artifacts/Minder-0.8.0-arm64.apk` (ignored by Git), 19,224,066 bytes, SHA-256 `d69b7b590f803827a134ba8b142f48a7950153f08a958ec8195c4decb2063a51`. aapt2 reports `io.liveleds.minder`, version 0.8.0, code 24. Signer SHA-256 `7bbc3aba5e417ffe87efa00c53c183c6490dc61905ce7231aaf99f87666d2572`; I did not compare it with the 0.7.0 certificate. Telegram confirmed one upload (message 2006); an earlier attempt failed with 404 before upload because only one secret reached the command.

## Limits

- Serper credits are counted in logs only; there is no euro meter for them.
- The offer's query comes from the model, shown to the owner before any search. A model that writes a misleading query is visible to the owner, not blocked.
- Typing `look it up` works like the tap. Phone behaviour is unverified until the APK is tried.
- Production needs `SERPER_API_KEY` as a Coolify process variable. No deployment was made.
