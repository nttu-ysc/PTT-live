package pttclient

type PTTError string

func (e PTTError) Error() string {
	return string(e)
}

const (
	Timeout               PTTError = "連線逾時"
	PTTOverloadError      PTTError = "PTT 伺服器繁忙請稍後再試"
	AuthError             PTTError = "帳號或密碼錯誤"
	AuthErrorMax          PTTError = "帳號或密碼錯誤超過上限"
	NotFinishArticleError PTTError = "文章尚未完成"
	BoardNameError        PTTError = "看板名稱錯誤"
)
