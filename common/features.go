package common

// features.go — the single source of truth for the bit <-> feature mapping.
//
// The mapping is defined ONCE, as an ordered list. Alice walks it in order to
// build an outgoing message; Bob walks the SAME list in the SAME order to read
// the bits back. Because both sides iterate one shared slice, the bit order can
// never silently disagree between the two bots.
//
// Bit order (MSB -> LSB), 7 features after dropping "link embed":
//   bit 0 (MSB)  user mention?
//   bit 1        role mention?
//   bit 2        attachment?
//   bit 3        is reply?
//   bit 4        edited?
//   bit 5        reaction?
//   bit 6 (LSB)  thread?
//
// Planned types (no code yet — placeholders):
//
//   type OutgoingSpec struct { ... }
//       What Alice needs to assemble before sending: whether to add a mention,
//       attach a file, send as a reply, edit afterwards, react afterwards,
//       create a thread, plus the innocuous visible text ("lol", "ok", ...).
//
//   type Feature struct {
//       Name   string
//       Encode func(spec *OutgoingSpec, bit bool)       // Alice: set the feature
//       Detect func(m *discordgo.Message) bool          // Bob: read the bit back
//   }
//
//   var Registry = []Feature{ ... 7 entries, in the bit order above ... }
//
// Detect operates on a SETTLED *discordgo.Message — the final state of the
// message after all edits/reactions/threads have been applied. In the primary
// "decode later" model (see protocol.go) Bob gets that settled object straight
// from a REST history fetch, so all 7 bits are readable from one object:
//     user mention -> .Mentions        role mention -> .MentionRoles
//     attachment   -> .Attachments     is reply     -> .MessageReference
//     edited       -> .EditedTimestamp reaction     -> .Reactions
//     thread       -> .Thread (or list threads)
// The same Detect functions also work for an optional live mode, where a
// settling window first lets a gateway message settle before decoding.
