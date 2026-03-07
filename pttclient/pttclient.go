package pttclient

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"ptt-live/pttcrawler"
	"ptt-live/ptterror"
	"regexp"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type PTTClient struct {
	Ctx        context.Context
	Account    string
	Password   string
	connection *ssh.Client
	session    *ssh.Session
	sessionIn  io.WriteCloser
	sessionOut *customOut
	lock       sync.Locker
	reconnect  chan struct{}
}

func NewPttClient() *PTTClient {
	return &PTTClient{
		lock:      &sync.Mutex{},
		reconnect: make(chan struct{}, 1),
	}
}

func (c *PTTClient) StartUp(ctx context.Context) {
	c.Ctx = ctx
}

func (c *PTTClient) Connect() {
	// SSH 連接設置
	config := &ssh.ClientConfig{
		User:            "bbsu",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var err error
	// 建立 SSH 連接
	c.connection, err = ssh.Dial("tcp", "ptt.cc:22", config)
	if err != nil {
		log.Fatalf("unable to connect: %s", err)
	}

	// 創建一個新的會話
	c.session, err = c.connection.NewSession()
	if err != nil {
		log.Fatalf("unable to create c.session: %s", err)
	}

	//// 視窗大小改變時發送 SIGWINCH 信號
	//sigwinch := make(chan os.Signal, 1)
	//signal.Notify(sigwinch, syscall.SIGWINCH)
	//go func() {
	//	for {
	//		<-sigwinch
	//		width, height, err := terminalSize()
	//		if err == nil {
	//			c.session.WindowChange(height, width)
	//		}
	//	}
	//}()

	c.sessionIn, _ = c.session.StdinPipe()

	reader := make(chan []byte, 10)

	out := new(customOut)
	out.reader = reader
	c.sessionOut = out

	c.session.Stdout = out
	c.session.Stderr = os.Stderr

	// 獲取本地終端的屬性
	terminalModes := ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用本地輸入的回顯
		ssh.TTY_OP_ISPEED: 14400, // 輸入速度
		ssh.TTY_OP_OSPEED: 14400, // 輸出速度
	}

	// 設置終端模式
	fileDescriptor := int(os.Stdin.Fd())
	if terminal.IsTerminal(fileDescriptor) {
		originalState, err := terminal.MakeRaw(fileDescriptor)
		if err != nil {
			panic(err)
		}
		defer terminal.Restore(fileDescriptor, originalState)

		termWidth, termHeight, err := terminal.GetSize(fileDescriptor)
		if err != nil {
			panic(err)
		}

		err = c.session.RequestPty("xterm-256color", termHeight, termWidth, terminalModes)
		if err != nil {
			panic(err)
		}
	}

	// 啟動一個 shell
	err = c.session.Shell()
	if err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}

	go func(c *PTTClient) {
		c.session.Wait()
		to := time.NewTicker(time.Second)
		select {
		case <-to.C:
			runtime.MessageDialog(c.Ctx, runtime.MessageDialogOptions{
				Type:          runtime.ErrorDialog,
				Title:         "PTT live",
				Message:       "偵測到重複登入，即將關閉程式",
				Buttons:       nil,
				DefaultButton: "",
				CancelButton:  "",
				Icon:          nil,
			})
			os.Exit(1)
		case <-c.reconnect:
			c = NewPttClient()
			return
		}
	}(c)
}

func (c *PTTClient) Reconnect() {
	c.reconnect <- struct{}{}
	//c.Close()
	c.Connect()
}

func (c *PTTClient) write(p []byte) (int, error) {
	return c.sessionIn.Write(p)
}

func (c *PTTClient) read(t time.Duration) ([]byte, error) {
	return c.sessionOut.Read(t)
}

func (c *PTTClient) Lock() {
	c.lock.Lock()
}

func (c *PTTClient) Unlock() {
	c.lock.Unlock()
}

func (c *PTTClient) Wait() {
	c.session.Wait()
	c.Close()
}

func (c *PTTClient) Close() {
	c.session.Close()
	c.connection.Close()
}

type customOut struct {
	reader chan []byte
}

func (w *customOut) Read(t time.Duration) ([]byte, error) {
	select {
	case p := <-w.reader:
		return p, nil
	case <-time.After(t):
		return nil, os.ErrDeadlineExceeded
	}
}

func (w *customOut) Write(p []byte) (n int, err error) {
	newP := cleanData(p)
	w.reader <- newP
	return os.Stdout.Write(p)
}

func cleanData(data []byte) []byte {
	// First strip out general control characters except for \x1B, \n, \r, \t, and \x08 (which are used by terminal/PTT)
	data = regexp.MustCompile(`[\x00-\x07\x0B\x0C\x0E-\x1A\x1C-\x1F\x7F]+`).ReplaceAll(data, nil)

	// Replace ANSI escape sequences with =ESC=.
	data = regexp.MustCompile(`\x1B`).ReplaceAll(data, nil)

	// Remove any remaining ANSI escape codes.
	data = regexp.MustCompile(`\[[\d+;]*m`).ReplaceAll(data, nil)

	// Replace any [21;2H, [1;3H to change line - matching cursor positioning commands up to column 4
	data = regexp.MustCompile(`\[\d+;[0-4]H`).ReplaceAll(data, []byte("\n"))
	
	// Remove leftover positional H/r/J/K ANSI commands
	data = regexp.MustCompile(`\[[\d;]*[HrJK]`).ReplaceAll(data, nil)

	// Remove carriage returns.
	data = bytes.ReplaceAll(data, []byte{'\r'}, nil)

	// Remove backspaces.
	data = bytes.ReplaceAll(data, []byte{' ', '\x08'}, nil)

	return data
}

//func terminalSize() (int, int, error) {
//	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
//	if err != nil {
//		return 0, 0, err
//	}
//	return int(ws.Col), int(ws.Row), nil
//}

func (c *PTTClient) Login(account, password string) error {
	c.Lock()
	defer c.Unlock()
	c.write([]byte(account + "\r"))
	c.write([]byte(password + "\r"))

	timer := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-timer.C:
			c.Reconnect()
			return ptterror.Timeout
		default:
			screen, _ := c.read(5 * time.Second)
			if bytes.Contains(screen, []byte("系統過載, 請稍後再來")) {
				return ptterror.PTTOverloadError
			}
			if bytes.Contains(screen, []byte("密碼不對或無此帳號")) {
				return ptterror.AuthError
			}
			if bytes.Contains(screen, []byte("請仔細回憶您的密碼")) {
				c.Reconnect()
				runtime.MessageDialog(c.Ctx, runtime.MessageDialogOptions{
					Type:  runtime.ErrorDialog,
					Title: "失敗次數超過上限",
					Message: fmt.Sprintf("請仔細回憶您的密碼，\n" +
						"如果真的忘記密碼了，\n" +
						"可參考以下連結 : https://reurl.cc/282Mp9\n" +
						"亂踹密碼會留下記錄喔。"),
					Buttons:       nil,
					DefaultButton: "",
					CancelButton:  "",
					Icon:          nil,
				})
				return ptterror.AuthErrorMax
			}
			if bytes.Contains(screen, []byte("您想刪除其他重複登入的連線嗎？[Y/n]")) {
				c.write([]byte("y\r"))
				timer = time.NewTimer(5 * time.Second)
			}
			if bytes.Contains(screen, []byte("您要刪除以上錯誤嘗試的記錄嗎?")) {
				c.write([]byte("n\r"))
				timer = time.NewTimer(5 * time.Second)
			}
			if bytes.Contains(screen, []byte("您有一篇文章尚未完成")) {
				return ptterror.NotFinishArticleError
			}
			if bytes.Contains(screen, []byte("按任意鍵繼續")) {
				c.write([]byte(" "))
			}
			if bytes.Contains(screen, []byte("精華公佈欄")) {
				_, err := runtime.MessageDialog(c.Ctx, runtime.MessageDialogOptions{
					Type:          runtime.InfoDialog,
					Title:         "登入成功",
					Message:       "",
					Buttons:       nil,
					DefaultButton: "",
					CancelButton:  "",
					Icon:          nil,
				})
				return err
			}
		}
	}
}

//type Post struct {
//	SearchId string `json:"search_id"`
//	Author   string `json:"author"`
//	Title    string `json:"title"`
//}

var postReg = regexp.MustCompile(`(?i)\s*(\d+)\s+([~+]?爆\d*|\d+)\s*(\d{1,2}/\d{1,2})?\s+(\S+)\s+(\S+)\s+\[live\]\s+(.*)`)

func (c *PTTClient) GotoBoard(board string) (*[]pttcrawler.Post, error) {
	c.Lock()
	defer c.Unlock()
	c.write([]byte("s"))
	c.read(5 * time.Second)
	for _, b := range board {
		c.write([]byte(string(b)))
	}
	c.write([]byte("\r"))
	timer := time.NewTimer(1 * time.Second)
	for {
		select {
		case <-timer.C:
			return nil, ptterror.BoardNameError
		default:
			screen, _ := c.read(5 * time.Second)
			if bytes.Contains(screen, []byte("按任意鍵繼續")) || bytes.Contains(screen, []byte("動畫播放中...")) {
				c.write([]byte(" "))
				break
			}
			if bytes.Contains(screen, []byte("【板主:")) && bytes.Contains(screen, []byte("看板《")) {
				return pttcrawler.FetchLivePosts(board)
			}
		}
	}
}

//func (c *PTTClient) searchLivePost() (*[]Post, error) {
//	posts := new([]Post)
//	c.read(3 * time.Second)
//	c.write([]byte("/[live]\r"))
//	screen, _ := c.read(2 * time.Second)
//	screen2, _ := c.read(1 * time.Second)
//	screen = append(screen, screen2...)
//
//	matches := postReg.FindAllStringSubmatch(string(screen), -1)
//	for _, match := range matches {
//		*posts = append(*posts, Post{
//			SearchId: match[1],
//			Author:   match[4],
//			Title:    match[6],
//		})
//	}
//
//	pttcrawler.FetchLivePosts()
//
//	return posts, nil
//}

var msgReg = regexp.MustCompile(`(推|噓|→)?\s+(\S+)\s*:\s+(.*)\s+(\d{2}/\d{2}\s+\d{2}:\d{2})`)

type Message struct {
	Time    time.Time `json:"time"`
	Content string    `json:"content"`
	Author  string    `json:"author"`
	Hash    string    `json:"hash"`
}

func (c *PTTClient) FetchPostMessages(aid string, msgHash string) (*[]Message, error) {
	c.Lock()
	defer c.Unlock()

	// Step 1: Navigate to the post
	c.write([]byte(fmt.Sprintf("#%s\r\r", aid)))
	
	// Drain ALL pending output to ensure clean state before `$`
	// This prevents the first page's content from bleeding into the end page buffer
	for {
		_, err := c.read(500 * time.Millisecond)
		if err != nil {
			break
		}
	}

	// Step 2: Jump to the end of the post
	c.write([]byte("$"))

	// Step 3: Read frames until we see "100%" (= end of post).
	// We must accumulate all chunks (TCP packets) received during this poll
	// to ensure we capture the full screen text without missing the `msgHash`.
	var screen []byte
	for i := 0; i < 30; i++ {
		tmp, err := c.read(1 * time.Second)
		if err != nil {
			break // timeout: no more data
		}
		screen = append(screen, tmp...)
		if bytes.Contains(tmp, []byte("100%")) {
			break
		}
	}

	// Run cleanData again on the fully accumulated screen to catch split ANSI sequences
	screen = cleanData(screen)

	matches := msgReg.FindAllStringSubmatch(string(screen), -1)
	messages := new([]Message)
	seen := make(map[string]bool)
	
	// Precompile regex to strip out garbage characters
	garbageReg := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F\x{FFFD}\x{200B}-\x{200F}]+`)
	spaceTrimReg := regexp.MustCompile(`^\s+|\s+$`)

	for i := len(matches) - 1; i >= 0; i-- {
		content := matches[i][3]
		// Aggressively clean up invisible/garbage chars
		content = garbageReg.ReplaceAllString(content, "")
		// Trim edge spaces so varying terminal spaces don't change the hash
		content = spaceTrimReg.ReplaceAllString(content, "")

		h := md5.Sum([]byte(matches[i][2] + content + matches[i][4]))
		hash := fmt.Sprintf("%x", h)
		
		if hash == msgHash {
			break
		}

		if seen[hash] {
			continue // Skip duplicate lines rendered on the same screen
		}
		seen[hash] = true

		t, err := time.Parse("01/02 15:04", matches[i][4])
		if err != nil {
			t = time.Now()
		}

		*messages = append(*messages, Message{
			Time:    t,
			Content: content,
			Author:  matches[i][2],
			Hash:    hash,
		})
	}
	// reverse messages
	for i, j := 0, len(*messages)-1; i < j; i, j = i+1, j-1 {
		(*messages)[i], (*messages)[j] = (*messages)[j], (*messages)[i]
	}

	return messages, nil
}

type MessageType int

const (
	_ MessageType = iota
	Like
	Dislike
	Comment
)

func (c *PTTClient) SendMessage(t MessageType, message string) error {
	c.Lock()
	defer c.Unlock()
	c.write([]byte("X"))
	screen, _ := c.read(5 * time.Second)
	if bytes.Contains(screen, []byte("時間太近, 使用")) {
		c.write([]byte(message + "\r"))
		c.read(5 * time.Second)
		c.write([]byte("y\r"))
		c.read(5 * time.Second)
		log.Println("done1")
		return nil
	}
	if bytes.Contains(screen, []byte("您覺得這篇文章 1.值得推薦 2.給它噓聲")) {
		c.write([]byte(fmt.Sprintf("%d", t)))
		c.read(5 * time.Second)
		c.write([]byte(message + "\r"))
		c.read(5 * time.Second)
		c.write([]byte("y\r"))
		c.read(5 * time.Second)
		log.Println("done2")
	}
	return nil
}

// GetHotBoards fetches the PTT hot boards list via Go (bypasses frontend CORS).
func (c *PTTClient) GetHotBoards() ([]*pttcrawler.HotBoard, error) {
	return pttcrawler.FetchHotBoards()
}
