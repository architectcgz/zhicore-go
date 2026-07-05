package domain

import "time"

type DomainEvent interface {
	EventName() string
}

type PostCreated struct {
	PublicID PublicPostID
	OwnerID  OwnerID
}

func (PostCreated) EventName() string { return "PostCreated" }

type PostPublished struct {
	PublicID    PublicPostID
	OwnerID     OwnerID
	PublishedAt time.Time
}

func (PostPublished) EventName() string { return "PostPublished" }
