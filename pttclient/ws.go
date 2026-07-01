package pttclient

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
	"golang.org/x/text/width"
)

const wsURL = "wss://ws.ptt.cc/bbs"

// WSConnection implements Connection over WebSocket with a VT100 screen buffer.
type WSConnection struct {
	ctx    context.Context
	conn   *websocket.Conn
	reader chan []byte
	screen *wsScreen
}

func NewWSConnection() *WSConnection {
	return &WSConnection{
		reader: make(chan []byte, 10),
		screen: newWSScreen(),
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
		decoded, _ := decodeBig5(buf)
		buf = nil

		// Apply terminal bytes to the persistent screen buffer,
		// then render the full 24×80 grid as plain text.
		w.screen.apply(decoded)
		data := []byte(w.screen.render())

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
			// No new frames for 100ms — apply and render the complete screen.
			flush()
		}
	}
}

func (w *WSConnection) Reconnect() {
	w.Close()
	w.reader = make(chan []byte, 10)
	w.screen = newWSScreen()
	w.Connect(w.ctx)
}

func (w *WSConnection) Write(p []byte) (int, error) {
	p, _ = encodeBig5(string(p))
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

func decodeBig5(b []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(b), traditionalchinese.Big5.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func encodeBig5(s string) ([]byte, error) {
	var buf bytes.Buffer
	// 建立 Big5 編碼器（Encoder）
	writer := transform.NewWriter(&buf, traditionalchinese.Big5.NewEncoder())

	_, err := writer.Write([]byte(s))
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ── Virtual terminal screen buffer ────────────────────────────────────────────

// wsScreen is a minimal VT100 screen buffer (24 rows × 80 cols).
// It persists across flushes so that partial PTT screen updates are applied
// onto the correct state rather than a blank canvas each time.
type wsScreen struct {
	cells [24][80]rune
	row   int // cursor row, 0-based
	col   int // cursor col, 0-based
}

func newWSScreen() *wsScreen {
	s := &wsScreen{}
	for i := range s.cells {
		for j := range s.cells[i] {
			s.cells[i][j] = ' '
		}
	}
	return s
}

// apply processes terminal bytes and updates the screen buffer in place.
func (s *wsScreen) apply(data []byte) {
	runes := []rune(string(data))
	i := 0
	for i < len(runes) {
		r := runes[i]
		switch {
		case r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[':
			// CSI sequence: ESC [ <params> <cmd>
			i += 2
			start := i
			for i < len(runes) && (runes[i] == ';' || runes[i] == '?' ||
				(runes[i] >= '0' && runes[i] <= '9')) {
				i++
			}
			if i >= len(runes) {
				return
			}
			params, cmd := string(runes[start:i]), runes[i]
			i++
			s.csi(params, cmd)

		case r == '\x1b' && i+1 < len(runes):
			i += 2 // ESC + single char (ESC=, ESC>, …) — ignore

		case r == '\r':
			s.col = 0
			i++

		case r == '\n':
			s.row = wsClamp(s.row+1, 0, 23)
			i++

		case r == '\x08': // backspace
			s.col = wsClamp(s.col-1, 0, 79)
			i++

		case r >= ' ':
			if s.row < 24 && s.col < 80 {
				s.cells[s.row][s.col] = r
				w := wideRune(r)
				if w == 2 && s.col+1 < 80 {
					// '\x00' marks the trailing cell of a wide char.
					// render() skips it so CJK chars appear without gaps.
					// ESC[K overwrites it with ' ' when clearing the line.
					s.cells[s.row][s.col+1] = '\x00'
				}
				s.col += w
				if s.col >= 80 {
					s.col = 0
					s.row = wsClamp(s.row+1, 0, 23)
				}
			}
			i++

		default:
			i++ // skip other control chars
		}
	}
}

// csi handles CSI (Control Sequence Introducer) escape sequences.
func (s *wsScreen) csi(params string, cmd rune) {
	if strings.HasPrefix(params, "?") {
		return // DEC private modes (ESC[?25h etc.) — ignore
	}

	parts := strings.Split(params, ";")
	// p returns the n-th param (0-based) as int, defaulting to def if absent.
	p := func(idx, def int) int {
		if idx >= len(parts) || parts[idx] == "" {
			return def
		}
		n, err := strconv.Atoi(parts[idx])
		if err != nil {
			return def
		}
		return n
	}

	switch cmd {
	case 'H', 'f': // cursor position: ESC[row;colH  (1-based)
		s.row = wsClamp(p(0, 1)-1, 0, 23)
		s.col = wsClamp(p(1, 1)-1, 0, 79)

	case 'A': // cursor up
		s.row = wsClamp(s.row-p(0, 1), 0, 23)
	case 'B': // cursor down
		s.row = wsClamp(s.row+p(0, 1), 0, 23)
	case 'C': // cursor right
		s.col = wsClamp(s.col+p(0, 1), 0, 79)
	case 'D': // cursor left
		s.col = wsClamp(s.col-p(0, 1), 0, 79)
	case 'G': // cursor character absolute (column only)
		s.col = wsClamp(p(0, 1)-1, 0, 79)

	case 'K': // erase in line
		switch p(0, 0) {
		case 0: // erase to end of line
			for j := s.col; j < 80; j++ {
				s.cells[s.row][j] = ' '
			}
		case 1: // erase to beginning of line
			for j := 0; j <= s.col; j++ {
				s.cells[s.row][j] = ' '
			}
		case 2: // erase entire line
			for j := 0; j < 80; j++ {
				s.cells[s.row][j] = ' '
			}
		}

	case 'J': // erase in display
		switch p(0, 0) {
		case 0: // erase to end of screen
			for j := s.col; j < 80; j++ {
				s.cells[s.row][j] = ' '
			}
			for r := s.row + 1; r < 24; r++ {
				for j := 0; j < 80; j++ {
					s.cells[r][j] = ' '
				}
			}
		case 2: // erase entire screen
			for r := range s.cells {
				for j := range s.cells[r] {
					s.cells[r][j] = ' '
				}
			}
			s.row, s.col = 0, 0
		}

	case 'm': // SGR — ignore colors/attributes
	case 'r': // DECSTBM — ignore scroll-region setting
	}
}

// render converts the screen buffer to plain text.
// '\x00' cells (trailing halves of wide CJK chars) are skipped so that
// Chinese text is contiguous and bytes.Contains / regex matching works correctly.
func (s *wsScreen) render() string {
	var sb strings.Builder
	for _, row := range s.cells {
		for _, r := range row {
			if r != '\x00' {
				sb.WriteRune(r)
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// wideRune returns 2 for fullwidth / wide / ambiguous runes, 1 otherwise.
// Ambiguous-width chars (e.g. □ U+25A1) are rendered as 2 cols on CJK terminals like PTT.
func wideRune(r rune) int {
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth, width.EastAsianAmbiguous:
		return 2
	default:
		return 1
	}
}

func wsClamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
