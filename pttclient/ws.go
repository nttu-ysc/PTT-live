package pttclient

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

const wsURL = "wss://ws.ptt.cc/bbs" // TODO: confirm endpoint

// WSConnection implements Connection over WebSocket.
type WSConnection struct {
	ctx    context.Context
	conn   *websocket.Conn
	reader chan []byte
}

func NewWSConnection() *WSConnection {
	return &WSConnection{
		reader: make(chan []byte, 10),
	}
}

func (w *WSConnection) Connect(ctx context.Context) {
	w.ctx = ctx

	header := http.Header{}
	header.Set("Origin", "https://term.ptt.cc")

	var err error
	w.conn, _, err = websocket.DefaultDialer.DialContext(ctx, wsURL, header)
	if err != nil {
		log.Fatalf("websocket: unable to connect: %s", err)
	}

	go w.readLoop()
}

func (w *WSConnection) readLoop() {
	// rawCh decouples the blocking WebSocket read from the batching logic.
	rawCh := make(chan []byte, 32)
	go func() {
		for {
			_, msg, err := w.conn.ReadMessage()
			if err != nil {
				log.Println("websocket: read error:", err)
				close(rawCh)
				return
			}
			rawCh <- msg
		}
	}()

	const idleWindow = 100 * time.Millisecond
	var buf []byte
	idleTimer := time.NewTimer(idleWindow)
	idleTimer.Stop()

	flush := func() {
		if len(buf) == 0 {
			return
		}
		buf, _ = decodeBig5(buf)
		data := cleanData(buf)
		buf = nil
		select {
		case w.reader <- data:
		default:
			select {
			case <-w.reader:
			default:
			}
			w.reader <- data
		}
	}

	for {
		select {
		case msg, ok := <-rawCh:
			if !ok {
				// Connection closed; flush whatever is buffered.
				flush()
				return
			}
			buf = append(buf, msg...)
			// Reset idle window: every new frame delays the flush by 100ms.
			if !idleTimer.Stop() {
				select {
				case <-idleTimer.C:
				default:
				}
			}
			idleTimer.Reset(idleWindow)

		case <-idleTimer.C:
			// No new frames for 100ms — this burst is complete.
			flush()
		}
	}
}

func decodeBig5(b []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(b), traditionalchinese.Big5.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (w *WSConnection) Reconnect() {
	w.Close()
	w.reader = make(chan []byte, 10)
	w.Connect(w.ctx)
}

func (w *WSConnection) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *WSConnection) Read(t time.Duration) ([]byte, error) {
	select {
	case msg := <-w.reader:
		return msg, nil
	case <-time.After(t):
		return nil, os.ErrDeadlineExceeded
	}
}

func (w *WSConnection) Close() {
	if w.conn != nil {
		w.conn.Close()
		w.conn = nil
	}
}
