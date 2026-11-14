# Calendar colours implementation plan

> **For agentic workers:** Use executing-plans inline. The owner approved the mock and explicitly requested implementation. No sub-agents.

**Goal:** Ship named coloured calendar spans, editable item colours in details and chat, and accurate event/location wording.

**Architecture:** Extend existing JSON item/proposal records and preview transactions. Reuse Confirm/Discard, version checks and event streams. Keep semantic address and date-boundary decisions in the agent; UI renders typed fields.

**Tech Stack:** Go, SQLite JSON records, Flutter/Dart.

**Spec:** [Approved calendar draft](../../screens.md#calendar-event-bars--approval-draft), owner approval on 2026-11-14.

## Global constraints

- Six colours: blue, teal, green, amber, lilac, slate. Legacy items display Slate; new omitted colours default Slate if the agent supplies none.
- Named bars continue across week boundaries, with stable colour and full titles in the selected-day agenda.
- Header, 44 px selection/controls, 56 px navigation and 8/16/24/32 spacing follow the approved guide. Occupied week rows grow for bars.
- `location_kind` is address or label; missing classifications render as plain Location. No production address keyword parser.
- `date_end` is inclusive or checkout. Empty retains the old exclusive boundary; timed ends remain exclusive. Ordinary multi-day events use inclusive; only explicit checkout uses Checkout wording.
- Existing JSON, reminder schedules, source previews, Undo and stale-version protection remain compatible. No destructive data rewrite or migration.
- User-approved colour edit in details uses the existing versioned preview/Confirm transaction. Chat picker changes pending proposals only.
- Build an APK for phone review; no merge to main.

## Review focus

- Midnight and daylight-saving boundaries: instant ends exclude the next day; date ends use calendar arithmetic.
- Overlapping long events and month edges: allocate lanes without hidden events or mismatched continuation colours.
- Old unclassified locations and ranges: preserve stored boundaries, never silently turn a label into a Maps link.
- Stale versions and failed requests: colours stay unchanged, errors remain visible, pending proposals can be discarded.
- Enlarged text and narrow phones: names may ellipsize in bars; agenda preserves full names; controls remain 44 px.

## Tasks

### 1. Item presentation values and versioned colour commands

Files: domain/model.go, domain/event.go, command/attributes.go, command/colour.go; domain/command tests.

- [ ] Write failing tests for palette validation; inclusive dates versus checkout; colour create/edit, pending picker edits, Discard, Undo, unrelated field retention and conflicts on real temporary SQLite.
- [ ] Add `color`, `location_kind`, `date_end` to Item/Proposal. Validate enums. Clear stale location classification when the location changes without one. Publish `Change.color` on create/update previews.
- [ ] Implement `SetColor(changeID, color)` and versioned item-colour preview via the existing Preview API. Preserve capture candidate colour choices in review state.
- [ ] Run domain and command tests; record results.

### 2. Wire and agent behaviour

Files: api/api.go, api/item_colour.go, openrouter/client.go, assistant/prompt.go and boundary tests.

- [ ] Write failing HTTP tests for item colour previews and pending colour edits, rejecting unknown values, stale versions and resolved changes.
- [ ] Add POST /v1/items/{id}/color {expected_version,color}; POST /v1/changes/{id}/color {color}.
- [ ] Expose the three typed fields in chat/source tool schemas. Instruct the agent to choose/display a colour, honour explicit choices, preserve unrelated colour edits, distinguish labels from addresses, clarify ambiguous hour/date ranges and use inclusive ordinary event dates versus checkout.
- [ ] Run Go boundary tests and a synthetic real-model check of creation/colour editing where credentials are available.

### 3. Flutter colour editing and spanning calendar

Files: domain/item.dart, domain/entry.dart, repositories, data/wire.dart, data/chat_api.dart, controllers; ui/item_colours.dart, ui/calendar_week.dart, ui/calendar_screen.dart, ui/calendar_item_details.dart, ui/entry_view.dart, ui/format.dart; Flutter tests/goldens.

- [ ] Write failing tests for item/wire colours, date boundaries, continued spans, overlapping lanes, selected-day agreement and detail/chat picker changes.
- [ ] Add typed fields, palette values and repository methods, using existing controller error and lifecycle handling.
- [ ] Build a reusable named six-colour picker. Details Apply creates/confirms a versioned colour preview; Cancel changes nothing. Chat choices call SetColor while pending and disable while acting/resolved.
- [ ] Replace dots with named spanning bars, grow rows with lane count and text scale, retain 44 px day frames and full agenda titles. Use palette accents in cards and details.
- [ ] Change generic event label to Event; use Starts/Ends for date boundaries, Checkout only with date_end checkout; link only location_kind address.
- [ ] Run focused Flutter tests, update only approved calendar/preview goldens, then analyze/format.

### 4. Guide, verification and review build

Files: canonical brand/foundation/calendar docs, owning AGENTS files, HANDOFF; app/pubspec.yaml; browser fixtures.

- [ ] Promote approved calendar colour extension into canonical guide/contracts without changing unrelated brand rules.
- [ ] Run toolchain/check, release web, browser scenarios and inline requirements/code review. Fix material failures with regressions.
- [ ] Bump app version, build/signature-check an APK and make it available through the existing LAN artifact server.
- [ ] Record exact validation/build state and remaining phone approval in HANDOFF. Preserve branch unmerged.

## Execution rulings

- User's “Good enough. Implement” authorizes execution of the approved mock inline; no additional plan approval is requested.
- Empty `date_end` preserves legacy exclusive boundaries because their meaning cannot be inferred safely. New ordinary ranges include their stated final date; ambiguous 17–20 still needs the agent's question.
