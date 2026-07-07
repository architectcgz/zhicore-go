SELECT notification_type, channel, enabled
FROM notification_user_preference
WHERE user_id = $1
ORDER BY notification_type ASC, channel ASC
