package main

import (
	"time"

	contentruntime "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/runtime"
)

func DefaultContentServerConfig() ContentServerConfig {
	return ContentServerConfig{
		ServiceName: "zhicore-content",
		HTTP: ContentHTTPConfig{
			Addr:              ":8080",
			ReadHeaderTimeout: 2 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ShutdownTimeout:   20 * time.Second,
			MaxJSONBodyBytes:  1 << 20,
		},
		Mongo: ContentMongoConfig{
			Database:       "zhicore_content",
			BodyCollection: "post_bodies",
		},
		Redis: contentruntime.DefaultRedisConfig(),
		RabbitMQ: ContentRabbitMQConfig{
			Exchange:              "zhicore.events",
			PublishConfirmTimeout: 3 * time.Second,
		},
		RateLimit: contentruntime.DefaultRateLimitConfig(),
	}
}
