package internal

import "time"

type Click struct {
	URLId   int       `db:"url_id"`
	Country string    `db:"country"`
	IsBot   bool      `db:"is_bot"`
	Device  string    `db:"device"`
	Date    time.Time `db:"created_at"`
}
