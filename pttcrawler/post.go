package pttcrawler

import (
	"ptt-live/ptterror"
	"ptt-live/utils"

	"github.com/gocolly/colly"
)

const PttUrl = "https://www.ptt.cc/bbs/"

type Post struct {
	Url       string `json:"url"`
	AID       string `json:"aid"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	PushCount string `json:"push_count"`
	Date      string `json:"date"`
}

func FetchLivePosts(board string) (posts *[]Post, err error) {
	url := PttUrl + board + "/search?q=live"
	posts = new([]Post)

	c := colly.NewCollector()
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 1})
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("over18", "1")
	})

	c.OnHTML(`.r-list-container > .r-ent`, func(e *colly.HTMLElement) {
		post := new(Post)
		// 推文數
		post.PushCount = e.DOM.Find(`div.nrec > span`).Text()
		// 標題
		t := e.DOM.Find(`div.title > a`)
		if t.Size() == 0 {
			return
		}
		post.Title = t.Text()
		post.Url, _ = t.Attr("href")
		// 作者
		post.Author = e.DOM.Find(`div.meta > div.author`).Text()
		// 日期
		post.Date = e.DOM.Find(`div.meta > div.date`).Text()
		post.AID = utils.Url2Aid(post.Url)
		*posts = append(*posts, *post)
	})

	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode == 404 {
			err = ptterror.BoardNameError
		}
	})

	err = c.Visit(url)
	return posts, err
}

// HotBoard represents a board entry from the PTT hot boards page.
type HotBoard struct {
	Name      string `json:"name"`
	UserCount string `json:"user_count"`
	Category  string `json:"category"`
}

// FetchHotBoards scrapes the PTT hot boards page and returns the top boards.
func FetchHotBoards() (boards []*HotBoard, err error) {
	c := colly.NewCollector()
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 1})

	// PTT hotboards requires the over18 cookie
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Cookie", "over18=1")
	})

	// Actual HTML structure on ptt.cc/bbs/hotboards.html:
	//   <div class="b-ent">
	//     <a class="board" href="...">
	//       <div class="board-name">BoardName</div>
	//       <div class="board-nuser"><span ...>1234</span></div>
	//       <div class="board-class">Category</div>
	//     </a>
	//   </div>
	c.OnHTML(`.b-list-container .b-ent`, func(e *colly.HTMLElement) {
		board := &HotBoard{
			Name:      e.ChildText(".board-name"),
			UserCount: e.ChildText(".board-nuser"),
			Category:  e.ChildText(".board-class"),
		}
		if board.Name != "" {
			boards = append(boards, board)
		}
	})

	err = c.Visit("https://www.ptt.cc/bbs/hotboards.html")
	return boards, err
}
