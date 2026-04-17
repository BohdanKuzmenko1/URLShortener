CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    target_url VARCHAR(2000) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);