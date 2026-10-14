# 0030 — Answers to the digest come through outbound connections, as commands

- **Status:** Accepted
- **Date:** 2026-10-14
- **Epic:** EP-71

## Context

The owner asked to answer the digest directly in the chat app, for example to say in Slack during the day that an item of the to-do list is done, with Axon in step (2026-10-14). Axon is reachable on the tailnet only (ADR 0004): Slack, Discord, Telegram, and Microsoft cannot call it. Agents propose and people approve (ADR 0008).

## Decision

- **Listen by connecting out.** Telegram by long polling `getUpdates`; Slack through Socket Mode (an app-level token opens a WebSocket); Discord through its gateway with the direct-messages intent only (message content in DMs needs no privileged intent). A small RFC 6455 client (`internal/wsclient`) avoids a dependency. One listener runs per bot, reconnecting with back-off; the manager starts and stops them as channels change and waits for them on shutdown. Teams has no such mode, so Teams channels only send.
- **One person per bot channel.** A bot obeys only the chat linked to the channel, by sending the one-time code from Preferences (`link CODE`, or `/start CODE` in Telegram; hashed, 30 minutes, used once). Group and server messages are ignored; at most 20 messages a minute per sender are handled.
- **Commands, not guesses.** Messages are parsed deterministically, in English and French: `done 2` (numbers of the latest digest or list, `digest_items`), `done passport` or a sentence such as "I finished the passport" (matched to open tasks, today's habits, and open milestones by Snowball stems; one clear match is done, close ones are listed to pick from), `add … tomorrow at 3pm`, `note …`, `snooze 2 monday`, `list`, `undo`, `help`. A longer message that is none of these becomes a fleeting note. The person who writes is the actor, as when they click in the app; no AI interprets the message, so ADR 0008 is respected without an approval step. Every answer says exactly what changed, and `undo` reverts the last command (`last_action`).
- Changes go through the same domain functions as the app (`tasks.Update` plans a repeating task's next occurrence, `habits.CheckIn`, `goals.UpdateMilestone`, `tasks.Delete` into the trash), so the vault mirror, the sync, and every page follow.

## Consequences

Answering works on a private instance with no inbound port. A misread sentence is visible and one word away from undone. The chat id and display name of the linked person are stored with the channel. Free-form understanding by an AI model could be added later as a proposal to approve, not as an action.
