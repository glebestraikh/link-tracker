CREATE TABLE users (
    id         BIGINT      PRIMARY KEY,
    username   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE chats (
    tg_chat_id BIGINT      PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
