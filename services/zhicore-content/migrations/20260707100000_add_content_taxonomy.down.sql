-- Destructive rollback: this drops Content taxonomy tables, post tag
-- relations, tag counters, and category / topic references from posts.
-- Run only against disposable data or after an explicit backup/restore decision.
BEGIN;

DROP TABLE IF EXISTS post_tags;
DROP TABLE IF EXISTS tag_stats;
DROP TABLE IF EXISTS tags;

ALTER TABLE posts DROP CONSTRAINT IF EXISTS fk_posts_topic_id;
ALTER TABLE posts DROP CONSTRAINT IF EXISTS fk_posts_category_id;
DROP TABLE IF EXISTS categories;

ALTER TABLE posts DROP COLUMN IF EXISTS topic_id;
ALTER TABLE posts DROP COLUMN IF EXISTS category_id;

COMMIT;
