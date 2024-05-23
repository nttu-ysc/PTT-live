package pttcrawler

import (
	"github.com/gocolly/colly"
	"ptt-live/ptterror"
	"ptt-live/utils"
)

const PTT_URL = "https://www.ptt.cc/bbs/"

type Post struct {
	Url       string `json:"url"`
	AID       string `json:"aid"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	PushCount string `json:"push_count"`
	Date      string `json:"date"`
}

func FetchLivePosts(board string) (posts *[]Post, err error) {
	url := PTT_URL + board + "/search?q=%5Blive%5D"
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
