-- migrate:up

CREATE TABLE IF NOT EXISTS robots (
    uuid         VARCHAR(255) PRIMARY KEY,
    public_key   TEXT         NOT NULL,
    device_type  VARCHAR(100) NOT NULL,
    is_blacklisted BOOLEAN    NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_robots_device_type ON robots(device_type);
CREATE INDEX idx_robots_blacklisted ON robots(is_blacklisted) WHERE is_blacklisted = TRUE;

CREATE TABLE IF NOT EXISTS users (
    id           SERIAL PRIMARY KEY,
    username     VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT         NOT NULL,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- migrate:down

DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS robots;
