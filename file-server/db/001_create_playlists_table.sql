
CREATE TABLE playlists (
    id          VARCHAR(36)  PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL,
    created_at  TIMESTAMP    DEFAULT NOW(),
    updated_at  TIMESTAMP    DEFAULT NOW()
);

CREATE TABLE playlist_videos (
    playlist_id VARCHAR(36) REFERENCES playlists(id) ON DELETE CASCADE,
    video_id    VARCHAR(36) REFERENCES videos(id)    ON DELETE CASCADE,
    position    INT,

    PRIMARY KEY (playlist_id, video_id)
);
