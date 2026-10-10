package common

// codec.go — the character <-> bits conversion. Pure logic, no Discord here,
// so it can be unit-tested in isolation (round-trip: char -> bits -> char).
//
// Design decisions (agreed):
//   - 7 bits per character. Standard ASCII is 0x00..0x7F, so 7 bits is exactly
//     enough and lets us drop the "link embed" feature.
//   - Bit order is MSB-first, matching the worked example:
//         user mention = bit 0 (most significant) ... thread = bit 6 (least).
//         0100 0001 = 0x41 = 'A'
//     (With "link embed" removed we now have 7 features for 7 bits; keep the
//      same high-to-low ordering.)
//
// Planned contents (no code yet — placeholders):
//   - ByteToBits(b byte) [7]bool   // MSB-first; panics/masks above 0x7F
//   - BitsToByte(bits [7]bool) byte
//   - A couple of helpers to turn a whole string into a slice of [7]bool
//     frames and back, used by Alice (encode) and Bob (decode).
