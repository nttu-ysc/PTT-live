package consts

type Post struct {
	Url       string `json:"url"`
	AID       string `json:"aid"`
	SN        string `json:"sn"` // serial number
	Title     string `json:"title"`
	Author    string `json:"author"`
	PushCount string `json:"push_count"`
	Date      string `json:"date"`
}
