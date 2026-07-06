INSERT INTO notification_user_dnd (user_id, enabled, start_time, end_time, timezone, categories, channels, created_at, updated_at)
VALUES ($1, $2, $3::time, $4::time, $5, $6, $7, $8, $8)
ON CONFLICT (user_id) DO UPDATE
SET enabled = EXCLUDED.enabled,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time,
    timezone = EXCLUDED.timezone,
    categories = EXCLUDED.categories,
    channels = EXCLUDED.channels,
    updated_at = EXCLUDED.updated_at
RETURNING enabled,
          to_char(start_time, 'HH24:MI') AS start_time,
          to_char(end_time, 'HH24:MI') AS end_time,
          timezone,
          categories,
          channels
