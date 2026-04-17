CREATE TABLE IF NOT EXISTS url_stats (
    url_id        integer       NOT NULL,
    date          date          NOT NULL,
    country       varchar(2)    NOT NULL DEFAULT 'XX',
    device        varchar(10)   NOT NULL DEFAULT 'unknown',
    human_clicks  integer       NOT NULL DEFAULT 0,
    bot_clicks    integer       NOT NULL DEFAULT 0,

    PRIMARY KEY (url_id, date, country, device),
    FOREIGN KEY (url_id) REFERENCES urls(id) ON DELETE CASCADE
    );

CREATE INDEX IF NOT EXISTS idx_url_stats_url_id ON url_stats(url_id, date DESC);