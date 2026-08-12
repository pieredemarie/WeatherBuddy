CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    contact_type  TEXT NOT NULL,              -- 'Telegram' | 'Email'
    contact_value TEXT NOT NULL,
    city          TEXT NOT NULL,
    latitude      DOUBLE PRECISION NOT NULL,
    longitude     DOUBLE PRECISION NOT NULL,
    timezone      TEXT NOT NULL,               -- IANA, напр. 'Europe/Moscow'
    notify_time   TIME NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- повторная регистрация того же контакта обновляет его же строку (см. ON CONFLICT в репозитории),
    -- а не плодит дубликаты
    UNIQUE (contact_type, contact_value)
);

-- пригодится Scheduler-сервису: "у кого сейчас locally notify_time и он активен"
CREATE INDEX IF NOT EXISTS idx_users_notify_time ON users (notify_time) WHERE is_active;
