# Place lookup with Serper

Date: 2026-11-13. Branch `place-lookup`, built on `fix-connection-message` at `5807cca`. App version 0.9.2+27.
Spec: [place lookup design](../superpowers/specs/2026-11-13-place-lookup-design.md). Plan: [place lookup plan](../superpowers/plans/2026-11-13-place-lookup.md). Mock: `../mockups/place-lookup.png`.

## Behaviour

- With `SERPER_API_KEY` set, the model may call `offer_lookup` for a request that names a venue without an address. The server stores a fixed-format offer entry and serves it with computed `options` (`Skip`, `Look it up`). The app draws it as a card with two buttons while it is the newest entry.
- The server runs the search, not the model: only when the owner's text is `Look it up` and the newest entry before it, skipping error entries and earlier `Look it up` messages, is an offer. It searches the offer's query and adds at most three places to the one normal model call as untrusted data. The preview then carries the address in location and the website in the website field.
- Skip, other text, an answered offer, or no key runs no search. A failed or empty search says so, and the item is still proposed without an address.
- Logs carry `query_len`, `places`, `status` and `duration_ms`; never the query, a place or the key.

## Checks

| Check | Result |
|---|---|
| Test-first | Domain, config, Serper adapter, assistant gate, openrouter tool exposure, app card and wire tests each failed before the code existed |
| Startup log | Without `serper_key_set` in `logStartup`, `TestStartupSummaryShowsSettingsButNeverSecrets` fails; with it, it passes. The summary never holds the key |
| Prompt order | `TestWithoutAFinderThereIsNoOfferToolOrRule` asserts the place rule comes before the untrusted item data and appears only when a finder is set |
| Retry | `TestRetryAfterAFailedReplyStillSearches` and `TestNoSearchOnceTheOfferWasAnswered` cover a failed reply and an answered offer |
| API wire | `TestChatServesOptionsOnlyForAPlaceLookupOffer` checks `/v1/chat` and `/v1/chat?after=1` (not mutation-checked) |
| Live Serper | One real query for `SnowWorld Landgraaf` through TokenCrypt returned `Witte Wereld 1` and `https://www.snowworld.com/landgraaf?utm_source=gmb_listing&utm_medium=organic`, 1,170 ms, status 200 |
| Flutter | Part of `toolchain/check`: Flutter tests, analyze and format |
| Browser, place lookup | `toolchain/place-lookup-local-check` passed against compiled web, the model fixture and a fake Serper: the offer ran no search, Look it up ran exactly one, the preview showed `Witte Wereld 1, Landgraaf`, Skip ran none. Screenshots: `place-lookup-offer.png`, `place-lookup-preview.png` |
| Browser, calendar and reminder | `toolchain/calendar-local-check` and `toolchain/reminder-local-check` passed |
| Website field | `TestLookupReplyAfterAFreshOfferSearchesOnceAndFoldsInThePlaces` asserts the prompt rule and result message send the website to its own field; the preview screenshot shows a Website row |
| Browser, chat | `toolchain/local-check` passed (offline icon, recovery and restored send) |
| `toolchain/check` | Exit 0 on the final tree: checker tests, hook tests, Worker tests, architecture, DOX, orbital exports, Go race tests, vet, gofmt, Flutter tests, analyze, format, `git diff --check` |

## Deployment

`main` at `adb1bdf` was deployed through Coolify (deployment `gn89izeozdgbno8bmucrxgxl`) with `SERPER_API_KEY` as a runtime variable. Coolify reports running:healthy; the startup log shows `serper_key_set=true`, schema version 5 and no migration; the first 60 log lines hold no warnings or errors. Rollback: branch `main`, commit `d04c3a8`.

Artifact: `.local/artifacts/Minder-0.9.2-arm64.apk` (ignored by Git), 19,289,894 bytes, SHA-256 `de359f7f3254a74dfbe2d536ed760181ef8f95f7bbb9c49daa07cf7fdb0328d8`. aapt2 reports `io.liveleds.minder`, version 0.9.2, code 27 (above 0.9.1's 26). The signer SHA-256 `7bbc3aba5e417ffe87efa00c53c183c6490dc61905ce7231aaf99f87666d2572` matches 0.7.0, 0.8.0, 0.9.0 and 0.9.1. `libapp.so` contains the offer card strings; 0.9.1's does not. Telegram confirmed one upload (message 2009).

## Limits

- Serper credits are counted in logs only; there is no euro meter for them.
- The offer's query comes from the model, shown to the owner before any search. A model that writes a misleading query is visible to the owner, not blocked.
- Typing `look it up` works like the tap. Phone behaviour is unverified until the APK is tried.
- No lookup has run against production or on a phone yet; the live Serper check used the adapter alone.
