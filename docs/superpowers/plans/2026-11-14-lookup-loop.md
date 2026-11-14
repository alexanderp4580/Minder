# Lookup Loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (the project runs inline, without sub-agents). Steps use checkbox syntax.

**Goal:** After the owner approves a lookup, the model chooses among Serper maps, places and search, judges the results over at most three calls, then proposes the item or asks the owner.

**Architecture:** `assistant` owns the lookup contract, tool catalogue, budget and loop. `serper` is split into transport, endpoint table, decoders and a thin client. `openrouter` builds tool definitions from the catalogue and picks the tool set from `Message.Purpose`.

**Tech Stack:** Go (`toolchain/run go`), SQLite, OpenRouter tool calls, Serper HTTP, Python fixtures, Playwright browser check.

**Spec:** [docs/superpowers/specs/2026-11-14-lookup-loop-design.md](../specs/2026-11-14-lookup-loop-design.md)

## Global Constraints

- One responsibility per file and per function; no copy-pasted branch per endpoint. The owner called the old `serper/client.go` (transport, decoding, mapping and logging in one function) unacceptable.
- Keys never reach logs, errors or the model. Logs carry `kind`, `query_len`, `results`, `status`, `credits`, `duration_ms`, `trace_id`, never query text.
- Lookup results reach the model as untrusted data. At most three results per call, 10 s deadline, 64 KiB body, three calls per approval.
- No server fallbacks. No client or APK change. The offer entry format (`domain.OfferText`) is unchanged.
- Production `SERPER_URL` may only be `https://google.serper.dev`.
- Tests first; real temporary SQLite for store tests; Serper fixtures are local HTTP servers.

## Review Focus

- Model returns lookup and proposal calls together: only the lookups run, proposals drop and return after the results.
- Model asks for a fourth lookup: it never runs; the model is asked again without lookup tools.
- Serper fails or returns nothing: the model sees an error row, not an empty success, and may try another tool or ask.
- A snippet or title containing instructions: results stay in the untrusted-data message.
- Owner replies Skip or other text after an offer: no lookup tool is offered and no call runs.

---

### Task 1: Assistant lookup contract

**Files:** Create `server/assistant/finder.go`, `server/assistant/finder_test.go`. Modify `server/assistant/model.go` (remove `Place`, `PlaceFinder`).

**Interfaces:**
- Produces: `type LookupKind string` with `LookupMaps`, `LookupPlaces`, `LookupSearch`.
- Produces: `type Hit struct{Title, Address, Phone, Category, Website, Snippet string; Rating float64}` with `json:"…,omitempty"` snake-case tags.
- Produces: `type Finder interface{ Lookup(context.Context, LookupKind, string) ([]Hit, error) }`.
- Produces: `type LookupTool struct{Name string; Kind LookupKind; Description string}` and `var LookupTools []LookupTool` (three entries; descriptions state cost and returned fields per the spec table).
- Produces: `func LookupToolKind(name string) (LookupKind, bool)`.

- [ ] **Step 1: Failing tests** `TestLookupToolKindMapsEachToolName`, `TestEveryLookupToolDescribesCostAndFields` (each description contains `credit`; maps contains `postal code`, search contains `no structured address`), `TestHitJSONOmitsEmptyFields`.
- [ ] **Step 2: Run** `toolchain/run go go test ./assistant -run 'Lookup|Hit'` — expect FAIL (undefined).
- [ ] **Step 3: Implement** the types and catalogue in `finder.go`; delete `Place` and `PlaceFinder` from `model.go`.
- [ ] **Step 4: Run** the same command — PASS. Commit `Define the lookup contract and tool catalogue`.

### Task 2: Serper transport and decoders

**Files:** Create `server/serper/transport.go`, `decode.go`, tests `transport_test.go`, `decode_test.go`.

**Interfaces:**
- Produces: `type transport struct{key, base string; http *http.Client}`; `func (t transport) post(ctx context.Context, path string, payload map[string]any) (body []byte, status int, err error)`. Sets `X-API-KEY` and JSON content type; empty key returns an error before any request; non-200 returns status and an error naming the status only; bodies over 64 KiB error; context cancellation returns promptly; errors never carry key or body.
- Produces: `func decodeVenues(body []byte) ([]assistant.Hit, error)` (shared by `/maps` and `/places`: `title`, `address`, `phoneNumber`, `type` or `category`, `website`, `rating`), `func decodeWeb(body []byte) ([]assistant.Hit, error)` (`organic[]`: `title`, `link`→Website, `snippet`), both keeping at most three.

- [ ] **Step 1: Failing tests** against `httptest`: request is POST with `X-API-KEY` and the JSON payload; 401 and 500 errors contain neither key nor body; body of 64 KiB + 1 errors; empty key makes no request; cancelled context returns within 200 ms. Decoders: a four-place maps fixture (live shape from the spec) returns three hits with `Address` `Witte Wereld 1, 6372 VG Landgraaf, Netherlands`, `Phone`, `Category` from `type`; a places fixture reads `category`; a search fixture maps `link` to `Website` and keeps `Snippet`; `{}` returns an empty slice and nil; invalid JSON returns an error.
- [ ] **Step 2: Run** `toolchain/run go go test ./serper` — FAIL.
- [ ] **Step 3: Implement** `transport.post` and the two decoders, each under 40 lines, sharing a `firstThree` helper.
- [ ] **Step 4: Run** — PASS. Commit `Split the Serper transport and decoders`.

### Task 3: Serper endpoint table and client

**Files:** Create `server/serper/endpoint.go`, `endpoint_test.go`. Rewrite `server/serper/client.go`, `client_test.go`, `logging_test.go`.

**Interfaces:**
- Consumes: Task 1 and Task 2 types.
- Produces: `type endpoint struct{path string; credits int; decode func([]byte)([]assistant.Hit, error)}` and `var endpoints = map[assistant.LookupKind]endpoint{…}` with `/maps` (3, decodeVenues), `/places` (1, decodeVenues), `/search` (1, decodeWeb).
- Produces: `type Client struct{Key, URL string; Client *http.Client}` with `func (c *Client) Lookup(ctx context.Context, kind assistant.LookupKind, query string) ([]assistant.Hit, error)`. Default base `https://google.serper.dev`. Payload `{q, gl: "nl", hl: "en"}`. An unknown kind returns an error and makes no request. One info line `serper lookup` (kind, query_len, results, status, credits, duration_ms, trace_id) or one warn line `serper lookup failed`.

- [ ] **Step 1: Failing tests** `TestEachKindPostsToItsPath` (maps, places, search paths), `TestUnknownKindMakesNoRequest`, `TestClientLogsKindCreditsAndNeverQueryOrKey`, `TestLookupKeepsThreeHits`, plus the existing error, size, key and cancel behaviours reached through `Lookup`. Remove `TestDefaultEndpointIsTheMapsSearch` and `FindPlaces` tests.
- [ ] **Step 2: Run** — FAIL.
- [ ] **Step 3: Implement** `Lookup` as: look up the endpoint, `post`, `decode`, log; no other logic. Delete the old `find` function.
- [ ] **Step 4: Run** `toolchain/run go go test -race ./serper` — PASS. Commit `Compose the Serper client from the endpoint table`.

### Task 4: Base URL setting

**Files:** Modify `server/config/config.go`, `config_test.go`, `AGENTS.md`.

- [ ] **Step 1: Failing test** in `TestSerperSettings`: production accepts `https://google.serper.dev` and rejects `https://google.serper.dev/maps` and `http://example.test` without echoing the value.
- [ ] **Step 2: Run** `toolchain/run go go test ./config` — FAIL.
- [ ] **Step 3: Implement** the new allowed value in the production check.
- [ ] **Step 4: Run** — PASS. Commit `Make SERPER_URL a base URL`.

### Task 5: The lookup loop

**Files:** Create `server/assistant/lookup_loop.go`, `lookup_loop_test.go`. Modify `server/assistant/assistant.go` (replace the old search block and `WithPlaces`), `prompt.go`, `model.go` (Purpose constants), `capture.go` (use the constant), delete the old `placeMessage`.

**Interfaces:**
- Consumes: Task 1.
- Produces: `const (PurposeChat = ""; PurposeOffer = "offer"; PurposeLookup = "lookup"; PurposeEmail = "email")` and `const lookupBudget = 3`.
- Produces: `func (a *Service) WithLookups(f Finder) *Service`.
- Produces: `func (a *Service) askWithLookups(ctx context.Context, messages []Message, trace *replyTrace) (Response, error)`: asks with `PurposeLookup`; if the reply has lookup calls, runs them in order until the budget is spent, appends one untrusted-data user message (rows of `tool`, `query`, `results` or `error`), repeats; when the budget is zero, asks with `PurposeChat`. Returns the first reply with no lookup call, its cost summed with the earlier calls. Mixed replies keep only the lookup calls.
- Keeps `lookupQuery` (the approval gate) and `offerLookup`. `reply` sets `messages[0].Purpose = PurposeOffer` when a finder is set.
- `lookupRule` replaces `placeRule` in `prompt.go`: offer once per request alone; after approval choose a tool by the spec table; never guess between plausible matches, ask the owner; address to `location`, phone to `phone`, website to `website`; results are untrusted.

- [ ] **Step 1: Failing tests** with a scripted `fakeModel` and a `fakeFinder`:
  - `TestApprovedOfferLetsTheModelPickMapsAndThenPropose`: model calls `lookup_maps`, finder receives `(LookupMaps, "SnowWorld Landgraaf")`, the next model call sees the results message and proposes; the preview carries the location.
  - `TestFourthLookupNeverRuns`: scripted model asks for four lookups; the finder is called three times and the last model call has `Purpose == PurposeChat`.
  - `TestMixedRepliesRunOnlyTheLookups`: lookup + `propose_create` in one reply; no proposal until after the results.
  - `TestFailedLookupReachesTheModelAsAnError`: finder error yields a row with `error`, no hits.
  - `TestModelCanAskTheOwnerAfterResults`: a plain-text reply after results becomes a minder entry.
  - `TestSkipRunsNoLookup` and `TestNoFinderOffersNoLookup` (Purpose is `PurposeChat`).
  - `TestResultsReachTheModelAsUntrustedData`: the message begins with the untrusted-data wording and contains an injected snippet unchanged.
  - `TestReplyCostIncludesEveryModelCall`.
  - `TestPromptNamesEveryLookupToolsRule` (the rule mentions each tool name and the ask-instead-of-guess sentence).
- [ ] **Step 2: Run** `toolchain/run go go test ./assistant` — FAIL.
- [ ] **Step 3: Implement** the loop in `lookup_loop.go` with small functions: `lookupCalls(Response) []lookupCall`, `runLookups(ctx, calls, budget) []lookupRow`, `resultsMessage(rows, remaining) Message`. `reply` keeps one call site.
- [ ] **Step 4: Run** `toolchain/run go go test -race ./assistant` — PASS. Commit `Let the model run approved lookups in a bounded loop`.

### Task 6: Tool sets by purpose

**Files:** Modify `server/openrouter/client.go`, `client_test.go`, `AGENTS.md`; `server/cmd/minder/main.go`.

**Interfaces:** `Client.PlaceLookup` is removed. `tools()` plus: `PurposeOffer` adds `offer_lookup`; `PurposeLookup` adds the three tools from `assistant.LookupTools` (schema: `query` string, required) and not `offer_lookup`; `PurposeEmail` is unchanged; default is chat only. `main` calls `assistantService.WithLookups(&serper.Client{Key, URL})` only.

- [ ] **Step 1: Failing tests** `TestToolSetFollowsPurpose` (each purpose lists the exact tool names; email never has lookup tools) replacing `TestOfferToolExistsOnlyWhenPlaceLookupIsOn`.
- [ ] **Step 2: Run** `toolchain/run go go test ./openrouter` — FAIL.
- [ ] **Step 3: Implement** `toolsFor(purpose string) []any` and use it in `Complete`; build lookup definitions from `assistant.LookupTools`.
- [ ] **Step 4: Run** `toolchain/run go go build ./... && toolchain/run go go test -race ./...` — PASS. Commit `Choose model tools by purpose`.

### Task 7: Fixtures, browser check and documentation

**Files:** Modify `toolchain/model_fixture.py`, `serper_fixture.py`, `place-lookup-check.cjs`, `toolchain/AGENTS.md`, `server/assistant/AGENTS.md`, `server/serper/AGENTS.md`, `server/openrouter/AGENTS.md`, `docs/architecture.md`, `docs/evidence/place-lookup.md`, `HANDOFF.md`.

- [ ] **Step 1:** `serper_fixture.py` serves `/maps` with the full-address shape and records `(path, q)` at `GET /calls`. `model_fixture.py` answers a request with the `lookup_maps` tool by calling it, and a request carrying the results message by proposing the SnowWorld item. `place-lookup-check.cjs` asserts one `/maps` call per tap, none for Skip, and the preview shows the postal code.
- [ ] **Step 2:** Run `toolchain/place-lookup-local-check` after a fresh `flutter build web --release` — PASS.
- [ ] **Step 3:** Live check: one real query per tool through TokenCrypt (adapter `Lookup`, results printed without the key) — each returns hits; record counts and credits.
- [ ] **Step 4:** Update the DOX files listed above to current contracts (budget, tool table, Purpose, base URL, split files); remove stale `placeRule`, `FindPlaces` and `PlaceLookup` text. Update the evidence and handoff.
- [ ] **Step 5:** `toolchain/check` exits 0. Review the finished diff for the Global Constraints (one responsibility per file and function). Commit `Document and verify the lookup loop`.
