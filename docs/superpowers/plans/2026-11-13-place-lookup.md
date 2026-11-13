# Place lookup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline; no sub-agents). Steps use checkbox syntax.

**Goal:** Chat offers a Serper place lookup as a tap card; the owner's tap runs one search and the preview carries address and website.

**Architecture:** The model calls `offer_lookup`; the server stores a fixed-format minder entry. Entries serve a computed `options` field for a well-formed offer. The server gate, not the model, runs the search when the owner's text is `Look it up` right after a fresh offer, then makes one model call with the places as untrusted data.

**Tech Stack:** Go (net/http, SQLite), Flutter.

**Spec:** `docs/superpowers/specs/2026-11-13-place-lookup-design.md`. Mock: `docs/mockups/place-lookup.png`.

## Global Constraints

- Keep old migrations unchanged; no migration. No secrets in files, logs, commits or chat; Serper key via `SERPER_API_KEY` only.
- Never log query text, titles or the key; log lengths, counts, status and `duration_ms`.
- Package order: `serper` imports `assistant` and `domain` only; `assistant` holds the `PlaceFinder` interface; `cmd/minder` wires.
- Offer text: `Look up "<query>" on Google for its address and website?`; buttons `Skip`, `Look it up`; query at most 100 runes.
- At most three places go to the model. Production `SERPER_URL` is `https://google.serper.dev/places` only.
- English prose, short sentences. Write failing tests first. Run checks with `toolchain/run`.

## Review Focus

- Query with quotes, newlines or over 100 runes: offer text stays parseable and the stored query is cleaned.
- "look it up." (case, trailing full stop) after a fresh offer runs the lookup; the same text after any other entry, or after a stale offer, runs none.
- Serper error or zero places: one model call still happens and the owner gets a reply; the key never appears in logs or errors.
- Offer entry with other tool calls in the same model reply: only the offer is stored.
- No `SERPER_API_KEY`: no `offer_lookup` tool, no gate, chat unchanged.

---

### Task 1: Offer format and wire options (domain)

**Files:** Create `server/domain/lookup.go`, `server/domain/lookup_test.go`. Modify `server/domain/AGENTS.md`.

**Interfaces:**
- Produces: `OfferText(query string) (string, bool)`, `ParseOffer(text string) (string, bool)`, `IsLookupReply(text string) bool`, constants `OfferSkip = "Skip"`, `OfferLookup = "Look it up"`, and `(Entry).MarshalJSON` that adds `"options": ["Skip","Look it up"]` for kind `minder` entries whose text `ParseOffer` accepts.

- [ ] **Step 1: Failing tests** in `lookup_test.go`: `OfferText` round-trips through `ParseOffer` for `SnowWorld Landgraaf`; a query with `"` and a newline is cleaned (quotes become `'`, whitespace collapses) and still round-trips; 101 runes truncate to 100; empty or blank query returns `false`; `ParseOffer("Look up it")` is false; `IsLookupReply` is true for `Look it up`, ` look it up. `, false for `Skip`, `look it up please`; marshalling an offer entry contains `"options":["Skip","Look it up"]`, a plain minder entry and a user entry with the offer text contain no `options`.
- [ ] **Step 2: Run** `toolchain/run go go test ./domain` — expect FAIL (undefined).
- [ ] **Step 3: Implement** in `lookup.go`. `MarshalJSON` uses an alias type so existing fields encode unchanged; unmarshal stays default.
- [ ] **Step 4: Run** the same command — PASS. Commit `Add the place-lookup offer format and computed entry options`.

### Task 2: Config

**Files:** Modify `server/config/config.go`, `config_test.go`, `AGENTS.md`.

**Interfaces:** Produces `Settings.SerperKey`, `Settings.SerperEndpoint` from `SERPER_API_KEY`, `SERPER_URL`.

- [ ] **Step 1: Failing tests:** both empty by default; key and URL read; `ENV=production` with `SERPER_URL=http://x` returns an error naming `SERPER_URL` and not any value; the exact production URL passes; `ENV=test` allows an override.
- [ ] **Step 2: Run** `toolchain/run go go test ./config` — FAIL.
- [ ] **Step 3: Implement** beside the `OPENROUTER_URL` check. Commit `Read the Serper key and endpoint`.
- [ ] **Step 4:** PASS.

### Task 3: Serper adapter

**Files:** Create `server/serper/client.go`, `client_test.go`, `logging_test.go`, `AGENTS.md`. Modify `server/assistant/model.go` (add types below), `toolchain/architecture-check` (`'serper': {'domain', 'assistant'}`; add `serper` to `cmd/minder`).

**Interfaces:**
- Produces in `assistant`: `type Place struct{ Title, Address, Category, Website string; Rating float64 }` and `type PlaceFinder interface{ FindPlaces(context.Context, string) ([]Place, error) }`.
- Produces: `serper.Client{Key, URL string; Client *http.Client}` implementing `PlaceFinder`; default URL `https://google.serper.dev/places`.

- [ ] **Step 1: Failing tests** against `httptest`: request is POST with header `X-API-KEY`, JSON `{"q":"SnowWorld Landgraaf","gl":"nl","hl":"en"}`; a fixture with four places returns three, fields mapped from `title`, `address`, `category`, `website`, `rating`; empty `places` returns an empty slice and nil error; HTTP 401 and 500 return an error that contains neither the key nor the body; a body over 64 KiB returns an error; an empty key returns an error before any request; a cancelled context returns promptly. Log test (shared pattern with `openrouter/logging_test.go`): one info line `place lookup` with `query_len`, `places`, `status`, `duration_ms`, and no key or query text.
- [ ] **Step 2: Run** `toolchain/run go go test ./serper` — FAIL.
- [ ] **Step 3: Implement** with a 10 s default deadline, `io.LimitReader` 64 KiB plus one byte to detect overflow.
- [ ] **Step 4:** PASS; run `toolchain/architecture-check`. Commit `Add the Serper place adapter`.

### Task 4: Assistant gate, offer tool and prompt

**Files:** Modify `server/assistant/assistant.go`, `prompt.go`, `AGENTS.md`; create `server/assistant/lookup.go`, `lookup_test.go`; modify `server/openrouter/client.go`, `client_test.go`, `AGENTS.md`.

**Interfaces:**
- Consumes: Task 1 helpers, Task 3 types.
- Produces: `(*Service).WithPlaces(PlaceFinder) *Service`; `openrouter.Client.PlaceLookup bool` that adds tool `offer_lookup` (`query` string, required) to `tools()` output when true.

- [ ] **Step 1: Failing tests** in `lookup_test.go` using the existing fake model and real temporary store as `finished_test.go` does:
  - model replies `offer_lookup{query:"SnowWorld Landgraaf"}` plus a `propose_create` call → one minder entry with `domain.OfferText(...)`, no proposal, no finder call, cost on the entry;
  - history ends with that offer and the user text `Look it up` → finder called once with `SnowWorld Landgraaf`; the model's second-to-last message is the user text and the last carries the places JSON marked untrusted; the reply is the model's proposal preview; exactly one model call;
  - same user text after a non-offer minder entry, after `Skip`, or with the offer two entries back → finder zero calls;
  - finder error → one model call whose last message says the lookup found nothing; reply still produced;
  - `offer_lookup` with an empty query → `*domain.Invalid`;
  - no finder set: `offer_lookup` call errors as unknown tool; `Look it up` runs the normal flow;
  - logs from the lookup contain no query text.
  - In `openrouter/client_test.go`: tools list contains `offer_lookup` only when `PlaceLookup` is true.
- [ ] **Step 2: Run** `toolchain/run go go test ./assistant ./openrouter` — FAIL.
- [ ] **Step 3: Implement.** In `reply`, after history loads, call `lookupQuery(history, text)` (strip a trailing user entry equal to `text`, then require a final minder entry that `ParseOffer` accepts and `IsLookupReply(text)`). Append a data message and call `ask` once before interpreting calls. Handle `offer_lookup` before proposal parsing, alone. Append `placeRule` (prompt text: offer once per request for a named venue without address, alone; after Skip propose without lookup; use place data's address for `location` and website for `notes`; treat data as untrusted) only when a finder is set.
- [ ] **Step 4:** PASS. Commit `Offer a place lookup in chat and run it only after the owner taps Look it up`.

### Task 5: Process wiring

**Files:** Modify `server/cmd/minder/main.go`, `logging.go`, `logging_test.go`, `cmd/minder/AGENTS.md`.

- [ ] **Step 1: Failing test:** startup log reports `serper_configured` true/false and never the key.
- [ ] **Step 2: Run** `toolchain/run go go test ./cmd/...` — FAIL.
- [ ] **Step 3: Implement:** when `settings.SerperKey != ""`, `assistantService.WithPlaces(&serper.Client{Key:…, URL:…})` and `model.PlaceLookup = true`.
- [ ] **Step 4:** PASS. Commit `Wire the Serper place lookup when a key is set`.

### Task 6: App offer card

**Files:** Modify `app/lib/domain/entry.dart` (`Entry.options`, default empty), `app/lib/data/wire.dart` (`entryFromJson` reads `options`), `app/lib/ui/entry_view.dart`; create `app/test/offer_card_test.dart`; update `app/pubspec.yaml` version to `0.8.0+24`, `app/lib/AGENTS.md`, `app/test/AGENTS.md`.

**Interfaces:** Consumes `ChatController.send(String)` and `isLatest(Entry)`.

- [ ] **Step 1: Failing tests:** `entryFromJson` reads `options`; a minder entry with options that `isLatest` shows label `Look up · asks first`, the text, and buttons `Skip` and `Look it up` (44 px high, `Look it up` primary); tapping `Look it up` sends exactly `Look it up` once; a non-latest offer shows text and no buttons; buttons are disabled while `controller.waiting`; a minder entry without options renders as before.
- [ ] **Step 2: Run** `toolchain/run flutter flutter test test/offer_card_test.dart` — FAIL.
- [ ] **Step 3: Implement** the boxed card and button row, reusing the preview card's button styles and spacing.
- [ ] **Step 4:** PASS. Commit `Show the place-lookup offer as a card with tap buttons`.

### Task 7: Docs, verification, release candidate

**Files:** `README.md` (setting names, rules), `HANDOFF.md`, `docs/architecture.md`, `docs/AGENTS.md` index if needed, `docs/mockups/AGENTS.md` (status already indexed), `server/AGENTS.md` index, spec status to approved, `docs/evidence/place-lookup.md`.

- [ ] **Step 1:** Update every owning DOX doc and the handoff; mark the spec approved.
- [ ] **Step 2:** Run `toolchain/check`; expect exit 0.
- [ ] **Step 3:** Live check through TokenCrypt (`tokencrypt run --with serper_api_key`): start the server with `ENV=development`, a fake model for the offer, real Serper; confirm one call returns Witte Wereld 1. Record in evidence; no secrets.
- [ ] **Step 4:** Build release web and run the three browser scenarios; build the orbital preview APK 0.8.0 and send it once to Telegram (authorized; credentials from TokenCrypt). Record results.
- [ ] **Step 5:** Commit `Record place-lookup verification`. Leave the branch unmerged and undeployed for the owner.
