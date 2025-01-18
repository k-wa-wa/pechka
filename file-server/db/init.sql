
CREATE TABLE videos (
    id          VARCHAR(36)  PRIMARY KEY,
    fullpath    VARCHAR(255) NOT NULL UNIQUE,
    title       VARCHAR(255) NOT NULL,
    description TEXT         NOT NULL,
    url         VARCHAR(255) NOT NULL,
    created_at  TIMESTAMP    DEFAULT NOW(),
    updated_at  TIMESTAMP    DEFAULT NOW()
);

CREATE TABLE video_timestamps (
    timestamp_id VARCHAR(36)  PRIMARY KEY,
    video_id     VARCHAR(36)  NOT NULL,
    timestamp    CHAR(8)      NOT NULL,
    description  VARCHAR(255)
);
