CREATE TABLE guestbook_entries (
    id         BIGSERIAL PRIMARY KEY,
    message    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
