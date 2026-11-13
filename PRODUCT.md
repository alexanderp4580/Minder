# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

The app has one screen: chat. The owner has not agreed the design yet. The product ships as an Android app and a web client.

## Stack

Current stage: a website. The final client is Flutter, one codebase for Android and web. Backend: Go and SQLite in one container on Coolify.

## Users

One person: the owner. Single user, self-hosted. The owner ignores reminders and avoids painful tasks. Timezone Europe/Amsterdam. English UI and English input.

## Product Purpose

A personal planner for tasks, lists, reminders, calendar events and schedules. Capture takes almost no effort. Reminders are hard to ignore. The product helps with painful tasks. Running cost stays at or below €10 per month.

## Positioning

An AI agent manages everything in Minder through one command interface that validates every change. The data keeps its structure. The assistant confirms before it creates or changes anything. Reminders nag until the owner acknowledges them. The server never skips a reminder.

## Operating Context

The owner captures items by assistant chat (text, voice), email forwarding and manual quick-add. Android notifications carry the reminders, with Done, Snooze and Open actions. The web build is the debug target on the Steam Deck. The app needs a connection to the server.

## Capabilities and Constraints

- Items are Task, Event or Reminder. Lists, projects, labels (six fixed colours), inbox, profiles.
- The interface is one chat screen. Lists and summaries appear as chat messages. Other views are not designed.
- MVP (P0): items, lists, text assistant with confirm, Android push with sticky nagging and Done/Snooze, recurring items, backup.
- Out of v1: team sharing, iOS, desktop apps, a Telegram bot, billing.
- All user-facing text follows about 80% of ASD-STE100.
- Full detail and open questions: `docs/requirements.md`, `docs/screens.md`, `docs/foundation.md`, `docs/architecture.md`, `docs/m0-spec.md`. Undecided items stay in the open questions there.

## Brand Commitments

The brand guide `brand/index.html` is binding: Void/Signal palette, Michroma titles, Chakra Petch text, edge column per section, focus frame, square progress row. The primary logo is the word Minder. The app icon is the Michroma M with a lit rim over the Orbital background. Colour values, sizes and contrast ratios live in that file. Items marked "to check" or "Design pending" there stay open.

## Evidence on Hand

- `brand/index.html`: brand guide.
- `docs/mockups/chat.html` and `chat.png`: the chat screen draft. The user has not agreed it.
- No user data, testimonials or usage numbers exist. Do not invent them.

## Product Principles

- A reminder is a critical message. Swiping it away is not acknowledgement.
- The data stays valid, whoever writes it.
- Capture takes almost no effort.
- The product helps with painful tasks.
