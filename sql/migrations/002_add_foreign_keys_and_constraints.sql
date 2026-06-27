ALTER TABLE post
ADD CONSTRAINT fk_post_author
    FOREIGN KEY (author_id) REFERENCES user(user_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT fk_post_community
    FOREIGN KEY (community_id) REFERENCES community(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT chk_post_status
    CHECK (status IN (0, 1)),
MODIFY COLUMN status INT8 NOT NULL DEFAULT 1;
ALTER TABLE user
MODIFY COLUMN user_id BIGINT NOT NULL UNIQUE,
MODIFY COLUMN user_name VARCHAR(64) NOT NULL UNIQUE;
ALTER TABLE community
MODIFY COLUMN community_name VARCHAR(255) NOT NULL UNIQUE;
ALTER TABLE vote
ADD CONSTRAINT fk_vote_post
    FOREIGN KEY (post_id) REFERENCES post(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT fk_vote_user
    FOREIGN KEY (user_id) REFERENCES user(user_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT chk_vote_direction
    CHECK (direction IN (-1, 0, 1));
ALTER TABLE remark
ADD CONSTRAINT fk_remark_post
    FOREIGN KEY (post_id) REFERENCES post(id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT fk_remark_author
    FOREIGN KEY (author_id) REFERENCES user(user_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
MODIFY COLUMN author_id BIGINT NOT NULL;
ALTER TABLE remark
ADD CONSTRAINT fk_remark_reply_to
    FOREIGN KEY (reply_to) REFERENCES remark(id)
    ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE bluebell_bookmark
ADD CONSTRAINT fk_bookmark_user
    FOREIGN KEY (user_id) REFERENCES user(user_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT fk_bookmark_post
    FOREIGN KEY (post_id) REFERENCES post(id)
    ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE user_profile
ADD CONSTRAINT fk_user_profile_user
    FOREIGN KEY (user_id) REFERENCES user(user_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
MODIFY COLUMN avatar_url VARCHAR(500),
MODIFY COLUMN github_url VARCHAR(500);
ALTER TABLE follow
ADD CONSTRAINT fk_follow_follower
    FOREIGN KEY (follower_id) REFERENCES user(user_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT fk_follow_following
    FOREIGN KEY (following_id) REFERENCES user(user_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT chk_no_self_follow
    CHECK (follower_id != following_id);
ALTER TABLE activity
ADD CONSTRAINT fk_activity_user
    FOREIGN KEY (user_id) REFERENCES user(user_id)
    ON DELETE CASCADE ON UPDATE CASCADE,
ADD CONSTRAINT chk_activity_type
    CHECK (type IN (1, 2, 3, 4)),
MODIFY COLUMN type INT8 NOT NULL,
CHANGE COLUMN target_id target_post_id BIGINT,
ADD COLUMN target_user_id BIGINT AFTER target_post_id;
ALTER TABLE activity
ADD CONSTRAINT fk_activity_target_post
    FOREIGN KEY (target_post_id) REFERENCES post(id)
    ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE post
ADD INDEX idx_author_community (author_id, community_id);
ALTER TABLE remark
ADD INDEX idx_post_created (post_id, created_at);
ALTER TABLE activity
ADD INDEX idx_user_type (user_id, type);
ALTER TABLE bluebell_bookmark
ADD INDEX idx_created_at (created_at);