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
| Prompt order | `TestWithAFinderTheRuleComesBeforeTheUntrustedItems` and `TestWithoutAFinderThereIsNoOfferPurposeOrRule` assert the rule comes before the untrusted item data and appears only when a finder is set |
| Retry | `TestRetryAfterAFailedReplyStillLooksUp` and `TestNoLookupOnceTheOfferWasAnswered` cover a failed reply and an answered offer |
| API wire | `TestChatServesOptionsOnlyForAPlaceLookupOffer` checks `/v1/chat` and `/v1/chat?after=1` (not mutation-checked) |
| Live Serper | One real query for `SnowWorld Landgraaf` through TokenCrypt returned `Witte Wereld 1` and `https://www.snowworld.com/landgraaf?utm_source=gmb_listing&utm_medium=organic`, 1,170 ms, status 200 |
| Flutter | Part of `toolchain/check`: Flutter tests, analyze and format |
| Browser, place lookup | `toolchain/place-lookup-local-check` passed against compiled web, the model fixture and a fake Serper: the offer ran no search, Look it up ran exactly one, the preview showed `Witte Wereld 1, Landgraaf`, Skip ran none. Screenshots: `place-lookup-offer.png`, `place-lookup-preview.png` |
| Browser, calendar and reminder | `toolchain/calendar-local-check` and `toolchain/reminder-local-check` passed |
| Browser, chat | `toolchain/local-check` passed (offline icon, recovery and restored send) |
| `toolchain/check` | Exit 0 on the final tree: checker tests, hook tests, Worker tests, architecture, DOX, orbital exports, Go race tests, vet, gofmt, Flutter tests, analyze, format, `git diff --check` |

## Deployment

`main` at `adb1bdf` was deployed through Coolify (deployment `gn89izeozdgbno8bmucrxgxl`) with `SERPER_API_KEY` as a runtime variable. Coolify reports running:healthy; the startup log shows `serper_key_set=true`, schema version 5 and no migration; the first 60 log lines hold no warnings or errors. Rollback: branch `main`, commit `d04c3a8`.

Artifact: `.local/artifacts/Minder-0.9.2-arm64.apk` (ignored by Git), 19,289,894 bytes, SHA-256 `de359f7f3254a74dfbe2d536ed760181ef8f95f7bbb9c49daa07cf7fdb0328d8`. aapt2 reports `io.liveleds.minder`, version 0.9.2, code 27 (above 0.9.1's 26). The signer SHA-256 `7bbc3aba5e417ffe87efa00c53c183c6490dc61905ce7231aaf99f87666d2572` matches 0.7.0, 0.8.0, 0.9.0 and 0.9.1. `libapp.so` contains the offer card strings; 0.9.1's does not. Telegram confirmed one upload (message 2009).

## Postal code fix

The owner tried SnowWorld Landgraaf on a phone and got only `Witte Wereld 1`, with no postal code or town. A live comparison on 2026-11-14 found the cause: Serper `/places` returns the short address, `/maps` returns `Witte Wereld 1, 6372 VG Landgraaf, Netherlands`. Fixed by moving the lookup to the maps search (deployed at `323e5fb`); the kinds are now offered by the model, see Approved lookups. The first live check (earlier table) used `/places`, so it could not show the gap. Not yet verified on a phone.

## Approved lookups

Spec: [approved lookups design](../superpowers/specs/2026-11-14-approved-lookups-design.md). Plan: [approved lookups plan](../superpowers/plans/2026-11-14-approved-lookups.md). A first version let the model run up to three searches after one tap. On 2026-11-14 the owner asked "try to get full address" and the model replied that it had used all its searches, though no search had run in that turn: the prompt described the lookup tools and a budget in turns where the model had no such tools. The owner asked why any limit existed and chose per-call approval.
Now the model calls `offer_lookup` with a kind (`maps`, `places`, `search`) and a query. The card names the search. A tap runs exactly that search on the server; the model reads the hits as untrusted data and proposes, asks the owner, or offers one more search. No loop, no budget, no app change.

| Check | Result |
|---|---|
| Tests first | Domain offer format per kind and rejection of other text; the approved offer runs that kind and query once; results are untrusted data; failed and empty searches reach the model; a new offer after results runs nothing; gate cases; the prompt names no budget; tool sets by purpose; the offer tool requires `kind` and `query` |
| Browser | `toolchain/place-lookup-local-check` passed: the card read `Search Google Maps for "SnowWorld Landgraaf"?`, nothing ran before the tap, exactly one `/maps` request after it, the preview showed the postal code, Skip ran none |
| Live Serper, 2026-11-14 | Maps, places and search each returned results through the adapter (see the earlier run: maps 3 credits, places 1, search 1) |
| Real model, 2026-11-14 | Local server, OpenRouter and Serper through TokenCrypt. "Aviad Landgraaf SnowWorld 17 to 24 november" produced a Maps offer; the tap produced a preview with title, `Witte Wereld 1, 6372 VG Landgraaf, Netherlands`, phone and website in their own fields. "try to get full address" afterwards produced a new offer instead of a refusal. "Add Albert Heijn tomorrow at 10:00" made the model ask which branch; after "Look it up" typed as text it offered `Albert Heijn Amsterdam`, a guess the owner sees and can skip |
| Not verified | Phone behaviour; whether the model picks places or search when they fit better than maps (the real runs only offered maps) |

## Limits

- Serper credits are counted in logs only (`credits` on the `serper lookup` line); there is no euro meter for them.
- The offer's query comes from the model, shown to the owner before any search. A model that writes a misleading query is visible to the owner, not blocked.
- Typing `look it up` works like the tap. Phone behaviour is unverified until the APK is tried.
- No lookup has run against production or on a phone yet; the live Serper check used the adapter alone.
