package common

// protocol.go — framing: how Bob tells where a transmission starts and ends
// amid unrelated chatter, and how the two bots stay in sync.
//
// A single frame on the wire looks like this (each box = one Discord message
// carrying 7 hidden bits; the visible text is innocuous filler):
//
//     [ SYNC ][ SYNC ] [ payload char ][ char ]...[ char ] [ CHECKSUM ] [ END ]
//      \__ 2-byte __/   \_________ N payload bytes _______/   1 byte      1 byte
//          magic
//
// Why this shape:
//   - Payload text is printable ASCII (0x20..0x7E). The framing markers are
//     C0 control codes (0x01..0x03), so a marker can NEVER collide with a
//     real payload character — no escaping needed for text payloads.
//   - A 2-byte SYNC magic (not a single byte) makes an accidental start from
//     random/noise messages astronomically unlikely.
//   - END is a terminator, so transmissions are variable-length; Bob doesn't
//     need to know the length in advance.
//   - A whole-frame CHECKSUM is how we catch a dropped async event. Note we
//     have NO spare per-character parity bit: all 7 bits are spent on the
//     ASCII value, so integrity has to live at the frame level instead.
//
// Planned constants (no code yet — placeholders):
//   SYNC0 = 0x01 (SOH), SYNC1 = 0x02 (STX)   // start-of-transmission magic
//   END   = 0x03 (ETX)                        // end-of-transmission
//   Checksum = XOR (or CRC-8) over the payload bytes
//   SettlingWindow = ~2-3s   // how long Bob waits per message for edits/
//                            // reactions/thread before decoding the character
//
// DECODE MODE: batch / "send now, decode later" (primary).
//   Bob does NOT decode live. He waits until the transmission has fully settled
//   on Discord (all edits/reactions/threads applied), then fetches the messages'
//   FINAL state over the REST API (ChannelMessages) and decodes the whole frame
//   in one pass. Because the state is settled, every one of the 7 features is
//   readable straight off the fetched *discordgo.Message — no gateway events, no
//   per-message settling window, no race. (This is also what makes the SYNC/END
//   markers reliably detectable: their byte values 0x01/0x02/0x03 map onto the
//   async bits, which would be the hardest to catch live but are trivial once
//   settled.)
//
//   Batch decode is a pure function over an ordered (chronological) slice of
//   settled messages:
//     1. Fetch channel history (REST, newest-first, paginated 100/page) and
//        reverse to chronological order.
//     2. Scan for the 2-byte SYNC magic -> marks frame start.
//     3. Read decoded bytes until END.
//     4. Last payload byte is the checksum; verify it; emit the recovered
//        string (or flag a bad frame). Continue scanning for the next frame.
//
//   Requirements/assumptions: messages must still exist at decode time (don't
//   delete them); Bob's bot needs Read Message History. Trigger the sweep
//   manually (a "decode now" command) or from a light activity listener.
//
// OPTIONAL live mode (if real-time delivery is ever wanted): the SAME Detect
// functions run on a settled gateway message, with a per-message settling
// window to collect async follow-ups before decoding each character. Out of
// scope unless we decide we need it.
//
// Optional (only if we decide the channel should be reliable rather than
// fire-and-forget): a SYN/ACK handshake before the payload. That makes the
// channel bidirectional (Bob has to transmit too), so it's out of scope unless
// you want it.
