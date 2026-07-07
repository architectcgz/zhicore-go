SELECT d.public_id,
       d.recipient_id,
       n.public_id AS notification_public_id,
       d.channel,
       d.notification_type,
       d.status,
       d.provider,
       d.attempt_count,
       d.last_error_code,
       d.next_retry_at,
       d.created_at,
       d.updated_at
FROM notification_delivery d
LEFT JOIN notifications n ON n.id = d.notification_id
WHERE d.recipient_id = $1
  AND ($2 = '' OR d.channel = $2)
  AND ($3 = '' OR d.status = $3)
  AND ($4::timestamptz IS NULL OR d.created_at < $4 OR (d.created_at = $4 AND d.public_id < $5))
ORDER BY d.created_at DESC, d.public_id DESC
LIMIT $6
