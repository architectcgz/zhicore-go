UPDATE scheduled_publish_events
SET status = 'CANCELED',
    canceled_at = $2,
    updated_at = $2
WHERE post_id = $1
  AND status = 'PENDING'
