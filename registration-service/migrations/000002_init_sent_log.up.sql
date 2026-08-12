CREATE TABLE IF NOT EXISTS sent_log (
    id        BIGSERIAL PRIMARY KEY,
    user_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sent_date DATE NOT NULL,
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- защита от дубля отправки одному пользователю в один день —
    -- идемпотентность на уровне БД, а не только логики Notification service
    UNIQUE (user_id, sent_date)
);

-- Scheduler будет проверять "есть ли у юзера запись за сегодня" на каждой итерации cron —
-- этот индекс делает такую проверку дешёвой
CREATE INDEX IF NOT EXISTS idx_sent_log_user_date ON sent_log (user_id, sent_date);
