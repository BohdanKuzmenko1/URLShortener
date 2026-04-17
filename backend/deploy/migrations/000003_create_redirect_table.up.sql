CREATE TABLE IF NOT EXISTS redirects (
    id BIGSERIAL PRIMARY KEY,

    url_id INT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,

    client_ip TEXT,
    country CHAR(25),
    user_agent VARCHAR(300),
    referer TEXT,
    language VARCHAR(100),
    is_bot BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_redirects_url_id ON redirects(url_id);
