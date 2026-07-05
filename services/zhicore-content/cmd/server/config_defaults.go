package main

import "time"

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
		RabbitMQ: ContentRabbitMQConfig{
			Exchange:              "zhicore.events",
			PublishConfirmTimeout: 3 * time.Second,
		},
	}
}
