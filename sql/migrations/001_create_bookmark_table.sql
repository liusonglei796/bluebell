CREATE TABLE IF NOT EXISTS bluebell_bookmark (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL COMMENT 'User ID who bookmarked',
    post_id BIGINT NOT NULL COMMENT 'Bookmarked post ID',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT 'Bookmark creation time',
    UNIQUE KEY idx_bookmark_user_post (user_id, post_id),
    INDEX idx_user_id (user_id),
    INDEX idx_post_id (post_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='User bookmarks for posts';