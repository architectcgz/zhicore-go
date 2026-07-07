package main

import "time"

func DefaultNotificationServerConfig() NotificationServerConfig {
	return NotificationServerConfig{
		ServiceName: "zhicore-notification",
		HTTP: NotificationHTTPConfig{
			Addr:              ":8080",
			ReadHeaderTimeout: 2 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   20 * time.Second,
		},
		UserService: NotificationUserServiceConfig{
			Timeout: 2 * time.Second,
		},
		PublicID: NotificationPublicIDConfig{
			Prefix:        "ntf_",
			ActiveVersion: 1,
			Secrets:       map[uint8]string{},
		},
		Consumer: NotificationConsumerConfig{
			ConsumedEventsRetention: 7 * 24 * time.Hour,
		},
		RealtimeFanout: NotificationRealtimeFanoutConfig{
			Timeout: 500 * time.Millisecond,
		},
		Campaign: NotificationCampaignConfig{
			ClaimTimeout:           30 * time.Second,
			ShardBatchSize:         200,
			MaxConcurrentShardJobs: 4,
		},
	}
}
