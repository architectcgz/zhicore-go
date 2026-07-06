
INSERT INTO comment_counter_deltas (comment_id, post_id, counter_type, delta_value, status, created_at)
VALUES ($1, $2, $3, $4, 'PENDING', $5)