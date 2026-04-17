package broker

type RedirectEvent struct {
	URLId     int    `json:"url_id"`
	ClientIP  string `json:"client_ip"`
	Referer   string `json:"referer"`
	Country   string `json:"country"`
	Language  string `json:"language"`
	UserAgent string `json:"user_agent"`
	CreatedAt int64  `json:"created_at"`
}

type RedirectProducer interface {
	SendRedirect(event RedirectEvent, slug string) error
}
