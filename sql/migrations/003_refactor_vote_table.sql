DELETE FROM vote WHERE direction = 0;
ALTER TABLE vote MODIFY COLUMN direction TINYINT NOT NULL CHECK (direction IN (-1, 1));
ALTER TABLE vote ADD CONSTRAINT uk_user_post_vote UNIQUE KEY (user_id, post_id);