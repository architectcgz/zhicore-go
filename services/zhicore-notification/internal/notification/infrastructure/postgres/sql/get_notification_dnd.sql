SELECT enabled,
       to_char(start_time, 'HH24:MI') AS start_time,
       to_char(end_time, 'HH24:MI') AS end_time,
       timezone,
       categories,
       channels
FROM notification_user_dnd
WHERE user_id = $1
