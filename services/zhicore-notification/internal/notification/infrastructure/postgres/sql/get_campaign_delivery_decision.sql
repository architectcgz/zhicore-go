WITH subscription AS (
    SELECT level, in_app_enabled, websocket_enabled, digest_enabled
    FROM notification_author_subscription
    WHERE user_id = $1 AND author_id = $2
),
in_app_preference AS (
    SELECT enabled
    FROM notification_user_preference
    WHERE user_id = $1 AND notification_type = $3 AND channel = 'IN_APP'
),
websocket_preference AS (
    SELECT enabled
    FROM notification_user_preference
    WHERE user_id = $1 AND notification_type = $3 AND channel = 'WEBSOCKET'
),
email_preference AS (
    SELECT enabled
    FROM notification_user_preference
    WHERE user_id = $1 AND notification_type = $3 AND channel = 'EMAIL'
),
dnd AS (
    SELECT enabled,
           to_char(start_time, 'HH24:MI') AS start_time,
           to_char(end_time, 'HH24:MI') AS end_time,
           timezone,
           categories,
           channels
    FROM notification_user_dnd
    WHERE user_id = $1
)
SELECT COALESCE((SELECT level FROM subscription), 'ALL') AS level,
       COALESCE((SELECT enabled FROM in_app_preference), (SELECT in_app_enabled FROM subscription), TRUE) AS in_app_enabled,
       COALESCE((SELECT enabled FROM websocket_preference), (SELECT websocket_enabled FROM subscription), TRUE) AS websocket_enabled,
       COALESCE((SELECT enabled FROM email_preference), FALSE) AS email_preference_enabled,
       COALESCE((SELECT digest_enabled FROM subscription), TRUE) AS digest_enabled,
       COALESCE((SELECT enabled FROM dnd), FALSE) AS dnd_enabled,
       COALESCE((SELECT start_time FROM dnd), '') AS start_time,
       COALESCE((SELECT end_time FROM dnd), '') AS end_time,
       COALESCE((SELECT timezone FROM dnd), 'UTC') AS timezone,
       COALESCE((SELECT categories FROM dnd), '{}'::varchar[]) AS categories,
       COALESCE((SELECT channels FROM dnd), '{}'::varchar[]) AS channels;
