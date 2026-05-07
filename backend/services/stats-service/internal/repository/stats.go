package repository

import (
	"context"
	"fmt"
	"github.com/BohdanKuzmenko1/URLShortener/services/stats-service/internal"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

type StatsRepository interface {
	RecordClick(ctx context.Context, click internal.Click) error
	RecordClickBatch(ctx context.Context, clicks []internal.Click) error
	GetURLStats(ctx context.Context, URLId int32, date string) ([]internal.URLStats, error)
}

type statsRepository struct {
	db *sqlx.DB
}

// GetURLStats returns per-day click statistics for a given URL filtered by date.
// Each row in the result represents a unique (country, device) combination for that day.
func (s statsRepository) GetURLStats(ctx context.Context, URLId int32, date string) ([]internal.URLStats, error) {
	query := `
	SELECT url_id, date, country, device, human_clicks, bot_clicks 
	FROM url_stats 
	WHERE url_id = $1 AND date = $2;`

	rows, err := s.db.QueryContext(ctx, query, URLId, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []internal.URLStats

	for rows.Next() {
		var stat internal.URLStats
		err = rows.Scan(
			&stat.URLId,
			&stat.Date,
			&stat.Country,
			&stat.Device,
			&stat.Clicks,
			&stat.BotClicks,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// RecordClickBatch writes a batch of clicks to url_stats in a single INSERT.
// Before writing, clicks are aggregated in memory so that multiple clicks
// sharing the same (url_id, date, country, device) are collapsed into one row,
// reducing the number of VALUES clauses sent to the database.
//
// ON CONFLICT increments the existing counters instead of failing,
// making the operation safe to retry on replay (at-least-once delivery from Kafka).
func (s statsRepository) RecordClickBatch(ctx context.Context, clicks []internal.Click) error {
	if len(clicks) == 0 {
		return nil
	}

	// key identifies a unique dimension combination in url_stats.
	// Clicks that share the same key are counted together before the INSERT.
	type key struct {
		URLId   int
		Date    time.Time
		Country string
		Device  string
	}

	// aggregated maps each unique key to [humanClicks, botClicks] counters.
	aggregated := make(map[key][2]int) // [humanClicks, botClicks]
	for _, c := range clicks {
		k := key{c.URLId, c.Date, c.Country, c.Device}
		counts := aggregated[k]
		if c.IsBot {
			counts[1]++
		} else {
			counts[0]++
		}
		aggregated[k] = counts
	}

	// placeholders holds one ($i, $i+1, ...) entry per aggregated row.
	// args holds the corresponding values in the same order.
	placeholders := make([]string, 0, len(aggregated))
	args := make([]interface{}, 0, len(aggregated)*6)
	i := 1

	for k, counts := range aggregated {
		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d)",
			i, i+1, i+2, i+3, i+4, i+5,
		))
		args = append(args, k.URLId, k.Date, k.Country, k.Device, counts[0], counts[1])
		i += 6
	}

	// The final query looks like:
	//   INSERT INTO url_stats (...) VALUES ($1,...,$6), ($7,...,$12), ...
	//   ON CONFLICT (...) DO UPDATE SET human_clicks = ... + EXCLUDED.human_clicks, ...
	// EXCLUDED refers to the values we tried to insert, so existing counters
	// are incremented rather than overwritten.
	query := `
		INSERT INTO url_stats (url_id, date, country, device, human_clicks, bot_clicks)
		VALUES ` + strings.Join(placeholders, ", ") + `
		ON CONFLICT (url_id, date, country, device)
		DO UPDATE SET
			human_clicks = url_stats.human_clicks + EXCLUDED.human_clicks,
			bot_clicks   = url_stats.bot_clicks + EXCLUDED.bot_clicks
	`

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// RecordClick inserts or increments stats for a single click.
//
// Deprecated: use RecordClickBatch for all new code.
// This method performs one database round-trip per click, which is inefficient at scale.
func (s statsRepository) RecordClick(ctx context.Context, click internal.Click) error {
	query := `
        INSERT INTO url_stats (url_id, date, country, device, human_clicks, bot_clicks)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (url_id, date, country, device)
        DO UPDATE SET
            human_clicks = url_stats.human_clicks + EXCLUDED.human_clicks,
            bot_clicks   = url_stats.bot_clicks + EXCLUDED.bot_clicks
    `

	var humanClicks, botClicks int
	if click.IsBot {
		botClicks = 1
	} else {
		humanClicks = 1
	}

	_, err := s.db.ExecContext(ctx, query,
		click.URLId,
		click.Date,
		click.Country,
		click.Device,
		humanClicks,
		botClicks,
	)
	return err
}

func NewStatsRepository(db *sqlx.DB) StatsRepository {
	return &statsRepository{db: db}
}
