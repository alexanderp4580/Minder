# Calendar item details and links Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (project rule: inline, no sub-agents) to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Show appointments with correct labels, expandable details, a Google Maps link and tappable phone numbers, emails and web links.

**Architecture:** A pure link finder in the domain layer, a `LinkLauncher` interface with a `url_launcher` adapter, three new `Item` fields, and an expandable calendar card.

**Tech Stack:** Flutter, `url_launcher`.

**Spec:** [calendar details design](../specs/2026-11-13-calendar-details-design.md)

## Global Constraints

- Test first. `toolchain/check` passes. DOX updated for each touched folder. No server change.
- Only `https`, `tel` and `mailto` targets open. Notes stay plain text, 600 characters at most.
- Targets 44 px. Existing calendar spacing is unchanged.

## Review Focus

- Dates (`2026-11-20`), times and booking references are not phone numbers. Pinned in Task 1.
- A trailing full stop or bracket stays out of a link. Pinned in Task 1.
- Hostile text (`javascript:` and similar) never becomes a link. Pinned in Task 1.
- An address with spaces, `&` or non-ASCII letters is encoded in the maps link. Pinned in Task 1.
- An item with no available detail rows shows no Details button. Pinned in Task 3.

---

### Task 1: Link finder

**Files:** Create `app/lib/domain/links.dart`; Test `app/test/links_test.dart`.

**Interfaces:** Produces `enum LinkKind { text, phone, email, url }`, `class LinkSpan { final LinkKind kind; final String text; final Uri? target; }`, `List<LinkSpan> findLinks(String text)`, `Uri mapsUri(String address)`.

- [x] Failing tests: `+31 6 1234 5678`, `020-1234567`, `(020) 123 4567` and `0031201234567` are phones with `tel:` targets; `2026-11-20`, `ABC123456789`, `14:30` and `12345` are plain; `a@b.nl` is an email with `mailto:`; `https://x.nl/a.` and `www.x.nl,` keep the punctuation outside; `javascript:alert(1)` is plain; spans rebuild the original text when joined; `mapsUri('Damrak 1 & Co, Zürich')` encodes the query.
- [x] Implement with regular expressions and an overlap rule (web link over email over phone).
- [x] `toolchain/run flutter flutter test test/links_test.dart` passes. Commit.

### Task 2: Item fields

**Files:** Modify `app/lib/domain/item.dart`, `app/lib/data/wire.dart`; Test `app/test/calendar_test.dart`.

**Interfaces:** Produces `Item.notes` (`String`), `Item.kind` (`String`), `Item.reminderOffsets` (`List<int>`), read from `notes`, `kind`, `reminder_offsets_minutes`.

- [x] Failing test: `itemFromJson` keeps all three; missing keys give `''`, `''` and `[]`.
- [x] Implement. Commit.

### Task 3: Expandable card

**Files:** Modify `app/lib/ui/calendar_screen.dart`; Create `app/lib/ui/linked_text.dart`, `app/lib/domain/link_launcher.dart`; Test `app/test/calendar_test.dart`.

**Interfaces:** Consumes Tasks 1 and 2. Produces `abstract interface class LinkLauncher { Future<bool> open(Uri uri); }`, `CalendarScreen({LinkLauncher? linkLauncher, ...})`, `LinkedText({required String text, required LinkLauncher? launcher})`.

- [x] Failing tests: an appointment with equal zones shows no "Departs" or "Arrives"; with different zones it does; an item with notes shows a `Details` button, and one without shows none; tapping it shows Starts, Ends, Type, Address, Notes and Reminders and `Hide details` hides them; tapping the address calls `open` with the maps URL; tapping a phone number in the notes calls `open` with `tel:`; the Details button is 44 px high.
- [x] Implement. View the new render, then update `goldens/calendar.png` if it changes. Commit.

### Task 4: Adapter, dependency, Android, composition

**Files:** Modify `app/pubspec.yaml`, `app/android/app/src/main/AndroidManifest.xml`, `app/lib/main.dart`; Create `app/lib/data/url_link_launcher.dart`; Test `app/test/link_launcher_test.dart`.

- [x] Failing test: the adapter refuses a URI whose scheme is not `https`, `tel` or `mailto` without calling the platform, and returns false.
- [x] Add `url_launcher` as a direct dependency, `<queries>` intents for `tel`, `mailto`, `https`, and pass the launcher to `CalendarScreen` in `main.dart`. Commit.

### Task 5: Docs, checks, delivery

- [x] Update DOX (`app/lib/domain`, `app/lib/data`, `app/lib/ui`, `app/android`, `app/test`), `docs/calendar-spec.md`, the mockup note and `HANDOFF.md`. Write `docs/evidence/calendar-details.md`.
- [x] Bump to 0.8.0+24. `toolchain/check` exits 0. Build the arm64 APK, verify signature and version, send once through TokenCrypt. No deploy.
