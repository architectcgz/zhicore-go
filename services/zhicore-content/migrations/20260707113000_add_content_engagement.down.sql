BEGIN;

-- Drops user engagement relationship facts. Down migration is destructive by
-- nature because relationship rows cannot be reconstructed from post_stats.
DROP TABLE IF EXISTS post_favorites;
DROP TABLE IF EXISTS post_likes;

COMMIT;
