UPDATE notification_delivery
SET status = 'WEBSOCKET_PENDING',
    attempt_count = attempt_count + 1,
    next_retry_at = NULL,
    updated_at = $4
WHERE id = $1
  AND ($3 = TRUE OR recipient_id = $2)
  AND status IN ('WEBSOCKET_PENDING', 'DIGEST_PENDING', 'FAILED')
RETURNING public_id, recipient_id, status
