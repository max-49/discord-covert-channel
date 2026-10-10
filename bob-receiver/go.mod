module bob-receiver

go 1.27.2

// When this bot imports the shared module (import "common"), add:
//     require common v0.0.0
// It resolves locally via ../go.work (no publishing needed).
// Run `go work sync` / `go mod tidy` after adding the first import.

require (
	github.com/bwmarrin/discordgo v0.29.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
	golang.org/x/sys v0.0.0-20201119102817-f84b799fce68 // indirect
)
