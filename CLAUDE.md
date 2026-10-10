# CLAUDE.md

Guidance for Claude Code (and the author) when working in this repo.

## What this project is

A **covert communication channel between two Discord bots**, built for a class.
The idea: a *message feature* is mapped to a *bit*. Each message carries a hidden
byte in the presence/absence of features (does it mention a user? is it a reply?
was it edited? etc.). A sequence of messages spells out a secret string.

- **Alice** (`alice-sender/`) is the sender: she converts text to characters and
  sends one Discord message per character, applying the features that encode it.
- **Bob** (`bob-receiver/`) is the receiver: he reads the messages back and
  reconstructs the string.

The channel should **look like normal user chatter** — the visible text of each
message is innocuous filler ("lol", "ok", "yeah"); the payload lives entirely in
the metadata features. This is why we do NOT use rich embeds (normal users can't
create them).

## Design decisions (settled)

**7 bits per character.** Standard ASCII is `0x00`–`0x7F`, so 7 bits is exactly
enough. This let us drop the original 8th feature ("link embed"), which was also
unreliable (Discord auto-unfurls links asynchronously).

**Bit → feature mapping** (MSB first, matching the worked example
`0100 0001 = 0x41 = 'A'`):

| bit | position | feature        | read from settled message |
|-----|----------|----------------|---------------------------|
| 0   | MSB      | user mention?  | `.Mentions`               |
| 1   |          | role mention?  | `.MentionRoles`           |
| 2   |          | attachment?    | `.Attachments`            |
| 3   |          | is reply?      | `.MessageReference`       |
| 4   |          | edited?        | `.EditedTimestamp`        |
| 5   |          | reaction?      | `.Reactions`              |
| 6   | LSB      | thread?        | `.Thread` (or list threads) |

The mapping is the **single source of truth**, defined once as an ordered list in
`common/features.go`. Both bots iterate the SAME list in the SAME order, so the
bit order can never silently disagree.

**Decode model: batch / "send now, decode later" (primary).**
Bob does NOT decode live. He waits until the transmission has fully settled on
Discord, then fetches each message's FINAL state over the REST API
(`ChannelMessages`) and decodes the whole frame in one pass. Because the state is
settled, all 7 features (including the async ones — edited/reaction/thread) are
readable straight off the fetched `*discordgo.Message`. This eliminates the live
settling-window / event-correlation machinery entirely and gives an
asynchronous, deniable channel (Bob can decode a transmission he was never online
for). A live mode is possible later using the same `Detect` functions, but is out
of scope for now.

**Framing** (each box = one Discord message carrying 7 hidden bits):

```
[ 0x01 ][ 0x02 ]   [ payload chars... ]   [ CRC-8 of payload ]   [ 0x03 ]
  SYNC    SYNC                              checksum               END
```

- **SYNC magic** = 2 bytes `{0x01, 0x02}` (SOH, STX). Decided: 2 bytes for now;
  add a `SYNC2` later if the channel turns out noisy. A 2-byte magic makes a
  false start from random chatter vanishingly unlikely.
- **Payload** = the N characters, printable ASCII (`0x20`–`0x7E`).
- **CRC-8/SMBUS** (poly `0x07`, init `0x00`, no reflection), over the **payload
  bytes ONLY** (not SYNC, not END). Appended as the single byte before END.
- **END** = 1 byte `0x03` (ETX).

Markers are C0 control codes (`0x01`–`0x03`) which never appear in printable
payload text, so markers can't collide with payload — no escaping needed.
We have NO spare per-character parity bit (all 7 bits carry the ASCII value), so
integrity is a whole-frame CRC. Its real job is catching a dropped async event.

**Bob's batch decode** = a pure function over chronological settled messages:
fetch history (REST is newest-first, paginated 100/page → reverse it) → scan for
SYNC → read bytes until END → recompute CRC-8 over the decoded payload and
compare → emit the string or flag a bad frame.

### Still open / to decide
- **Decode trigger**: manual "decode now" command to Bob vs. a light activity
  listener that auto-sweeps. (Manual is simplest and bulletproof for a demo.)
- Whether reliability (SYN/ACK handshake) is needed — currently out of scope
  (fire-and-forget); adding it would make the channel bidirectional.

## Repo layout

```
discord-covert-channel/
├── go.work                 # Go workspace tying the 3 modules together (committed)
├── CLAUDE.md               # this file
├── common/                 # shared single source of truth (module: common)
│   ├── go.mod
│   ├── codec.go            # char <-> 7 bits, MSB-first   [PLACEHOLDER COMMENTS]
│   ├── features.go         # ordered bit->feature registry [PLACEHOLDER COMMENTS]
│   └── protocol.go         # framing + CRC + decode plan   [PLACEHOLDER COMMENTS]
├── alice-sender/           # sender bot (module: alice-sender), uses ALICE_TOKEN
│   ├── main.go  bot/bot.go  go.mod
└── bob-receiver/           # receiver bot (module: bob-receiver), uses BOB_TOKEN
    └── main.go  bot/bot.go  go.mod
```

The three `common/*.go` files currently compile as an empty `package common` and
hold the agreed design as comments. Fill them in incrementally.

## Current state

- Workspace + `common` module scaffolding is set up; `go.work` is committed
  (removed from `.gitignore` on purpose so relative paths tie the modules).
- Both bots are still the starter echo bots from `discordgo`; no covert-channel
  logic implemented yet.
- `alice-sender`/`bob-receiver` `go.mod` have a comment noting how to add
  `require common v0.0.0` once they first `import "common"` (then run
  `go work sync`).

## Recommended next steps (in order)

1. **`common/codec.go`** — `ByteToBits(b byte) [7]bool` (MSB-first) and
   `BitsToByte([7]bool) byte`, plus string<->frames helpers. Pure logic; write a
   round-trip unit test (`'A'` → `0100001` → `'A'`) to lock the bit order.
2. **`common/protocol.go`** — the constants + `CRC8` function + frame
   encode/decode helpers.
3. **`common/features.go`** — the `Feature` registry with `Encode`/`Detect`.
4. **Alice** — text → frames → send one message per char, applying features
   (note: Alice still applies async features per message at send time — edit it,
   react to it, make a thread — and must pace for Discord rate limits).
5. **Bob** — REST history sweep → batch decode → verify CRC → print result.

## Gotchas / environment notes

- **Discord intents**: Bob needs the **MessageContent** privileged intent
  (toggle in the Discord Developer Portal) plus GuildMessages, and
  GuildMessageReactions if any live listening is added. Bob also needs
  **Read Message History** permission for the REST sweep. The current starter
  code does NOT set intents — that must be added.
- **Starter-code bugs to fix when touching the bots**: `alice-sender/bot/bot.go`
  uses `discord` before checking the error from `discordgo.New` and ignores the
  `Open()` error; `checkNilErr` logs a fixed `"Error message"` string instead of
  the real error `e`.
- **Tokens** come from env vars `ALICE_TOKEN` / `BOB_TOKEN`. Keep them in a
  `.env` (already gitignored) or the shell environment — never commit them.
- **CRC-8**: not in the Go stdlib (`hash/crc32`/`crc64` only). Use the ~10-line
  bitwise implementation. Compute cost is negligible vs. Discord rate-limit
  latency.
- **Rate limits**: Alice sends many API calls per character (send + maybe
  edit/react/thread). Thread creation is especially throttled. Pace the sender;
  throughput will be low by design.
- **Build/test**: this is a Go workspace — run `go build ./...` / `go test ./...`
  from the repo root so all three modules resolve via `go.work`.

## Working-style notes for Claude

- Match shell commands to whatever OS the session is running on (the scaffolding
  above was created on Windows, but the author expects to continue on a Mac).
- Commit attribution the author uses: commits end with a
  `Co-Authored-By: Claude ...` line; PR descriptions end with the
  "Generated with Claude Code" line. (Only add these when actually creating
  commits/PRs, and follow whatever the active session's attribution reminder
  says.)
