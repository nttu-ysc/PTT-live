package pttclient

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"ptt-live/pttcrawler"
	"ptt-live/ptterror"
	"regexp"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type PTTClient struct {
	Ctx      context.Context
	Account  string
	Password string
	conn     Connection
	lock     sync.Locker
}

func NewPttClient(conn Connection) *PTTClient {
	return &PTTClient{
		conn: conn,
		lock: &sync.Mutex{},
	}
}

func (c *PTTClient) StartUp(ctx context.Context) {
	c.Ctx = ctx
	c.conn.Connect(ctx)
}

func (c *PTTClient) Close() {
	c.conn.Close()
}

func (c *PTTClient) Lock() {
	c.lock.Lock()
}

func (c *PTTClient) Unlock() {
	c.lock.Unlock()
}

func (c *PTTClient) write(p []byte) (int, error) {
	return c.conn.Write(p)
}

func (c *PTTClient) read(t time.Duration) ([]byte, error) {
	return c.conn.Read(t)
}

func (c *PTTClient) Login(account, password string) error {
	c.Lock()
	defer c.Unlock()
	c.write([]byte(account + "\r"))
	c.write([]byte(password + "\r"))

	timer := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-timer.C:
			c.conn.Reconnect()
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
				c.conn.Reconnect()
				runtime.MessageDialog(c.Ctx, runtime.MessageDialogOptions{
					Type:  runtime.ErrorDialog,
					Title: "失敗次數超過上限",
					Message: fmt.Sprintf("請仔細回憶您的密碼，\n" +
						"如果真的忘記密碼了，\n" +
						"可參考以下連結 : https://reurl.cc/282Mp9\n" +
						"亂踹密碼會留下記錄喔。"),
				})
				return ptterror.AuthErrorMax
			}
			if bytes.Contains(screen, []byte("您想刪除其他重複登入的連線嗎？[Y/n]")) {
				btn, _ := runtime.MessageDialog(c.Ctx, runtime.MessageDialogOptions{
					Type:          runtime.QuestionDialog,
					Title:         "重複連線",
					Message:       "偵測到其他重複登入的連線，是否要刪除？",
					Buttons:       []string{"是", "否"},
					DefaultButton: "是",
					CancelButton:  "否",
				})
				if btn == "是" {
					c.write([]byte("y\r"))
				} else {
					c.write([]byte("n\r"))
				}
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
					Type:  runtime.InfoDialog,
					Title: "登入成功",
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

var (
	// (?s) enables dot-all mode so (.*?) can capture across \n,
	// which prevents long image URLs from being truncated at ANSI-injected newlines.
	msgReg        = regexp.MustCompile(`(推|噓| →)?\s+(\S+)\s*:\s+(.*?)(?:\s+\d+(?:\.\d+)+(?: \d+[KkMm]?)?)?\s+(\d{2}/\d{2}\s+\d{2}:\d{2})`)
	garbageReg    = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F\x{FFFD}\x{200B}-\x{200F}]+`)
	spaceTrimReg  = regexp.MustCompile(`^\s+|\s+$`)
	internalNLReg = regexp.MustCompile(`\n+`)
)

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
	c.write(fmt.Appendf(nil, "#%s\r\r", aid))

	// Drain ALL pending output to ensure clean state before `$`
	for {
		_, err := c.read(500 * time.Millisecond)
		if err != nil {
			break
		}
	}

	// Step 2: Jump to the end of the post
	c.write([]byte("$"))

	// Step 3: Read frames until we see "100%" (= end of post).
	var screen []byte
	for range 30 {
		tmp, err := c.read(1 * time.Second)
		if err != nil {
			break
		}
		screen = append(screen, tmp...)
		if bytes.Contains(tmp, []byte("100%")) {
			for {
				extra, errExtra := c.read(100 * time.Millisecond)
				if errExtra != nil {
					break
				}
				screen = append(screen, extra...)
			}
			break
		}
	}

	// Run cleanData again on the accumulated screen to catch split ANSI sequences
	screen = cleanData(screen)

	matches := msgReg.FindAllStringSubmatch(string(screen), -1)
	messages := new([]Message)
	seen := make(map[string]bool)

	for i := len(matches) - 1; i >= 0; i-- {
		content := matches[i][3]
		content = garbageReg.ReplaceAllString(content, "")
		content = internalNLReg.ReplaceAllString(content, " ")
		content = spaceTrimReg.ReplaceAllString(content, "")

		h := md5.Sum([]byte(matches[i][2] + content + matches[i][4]))
		hash := fmt.Sprintf("%x", h)

		if hash == msgHash {
			break
		}

		if seen[hash] {
			continue
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
		return nil
	}
	if bytes.Contains(screen, []byte("您覺得這篇文章 1.值得推薦 2.給它噓聲")) {
		c.write([]byte(fmt.Sprintf("%d", t)))
		c.read(5 * time.Second)
		c.write([]byte(message + "\r"))
		c.read(5 * time.Second)
		c.write([]byte("y\r"))
		c.read(5 * time.Second)
	}
	return nil
}

func (c *PTTClient) GetHotBoards() ([]*pttcrawler.HotBoard, error) {
	return pttcrawler.FetchHotBoards()
}

func (c *PTTClient) ReturnToBoard() {
	c.Lock()
	defer c.Unlock()
	c.write([]byte("q\r"))
	c.read(500 * time.Millisecond)
}
