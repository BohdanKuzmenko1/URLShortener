package internal

type Redirect struct {
	Slug      string `db:"slug"`
	ClientIP  string `db:"client_ip"`
	Language  string `db:"language"`
	UserAgent string `db:"user_agent"`
	Country   string `db:"country"`
	Referer   string `db:"referer"`
}
