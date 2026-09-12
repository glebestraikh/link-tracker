CREATE TABLE chats (
    tg_chat_id BIGINT      PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE links (
    id           BIGSERIAL   PRIMARY KEY,
    url          TEXT        NOT NULL UNIQUE,
    last_updated TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE link_chat (
    link_id BIGINT NOT NULL REFERENCES links(id)        ON DELETE CASCADE,
    chat_id BIGINT NOT NULL REFERENCES chats(tg_chat_id) ON DELETE CASCADE,
    PRIMARY KEY (link_id, chat_id)
);

CREATE INDEX idx_link_chat_chat_id ON link_chat (chat_id);

CREATE TABLE tags (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT      NOT NULL UNIQUE
);

CREATE TABLE link_chat_tag (
    link_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    tag_id  BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (link_id, chat_id, tag_id),
    FOREIGN KEY (link_id, chat_id) REFERENCES link_chat(link_id, chat_id) ON DELETE CASCADE
);

CREATE INDEX idx_link_chat_tag_tag_id  ON link_chat_tag (tag_id);
CREATE INDEX idx_link_chat_tag_chat_id ON link_chat_tag (chat_id);
