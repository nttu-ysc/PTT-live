package pttclient

import (
	"context"
	"time"
)

// Connection abstracts the low-level PTT transport (SSH, WebSocket, etc.).
type Connection interface {
	// Connect establishes the transport session.
	// ctx is stored for use in native dialogs triggered by unexpected disconnects.
	Connect(ctx context.Context)
	// Reconnect tears down the current session and opens a new one.
	Reconnect()
	// Write sends raw bytes to the PTT server.
	Write(p []byte) (int, error)
	// Read blocks up to duration t waiting for bytes from the PTT server.
	Read(t time.Duration) ([]byte, error)
	// Close terminates the connection.
	Close()
}
