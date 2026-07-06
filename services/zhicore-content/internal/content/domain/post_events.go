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

type PostUnpublished struct {
	PublicID      PublicPostID
	OwnerID       OwnerID
	UnpublishedAt time.Time
}

func (PostUnpublished) EventName() string { return "PostUnpublished" }

type PostDeleted struct {
	PublicID  PublicPostID
	OwnerID   OwnerID
	DeletedAt time.Time
}

func (PostDeleted) EventName() string { return "PostDeleted" }

type PostRestored struct {
	PublicID   PublicPostID
	OwnerID    OwnerID
	RestoredAt time.Time
}

func (PostRestored) EventName() string { return "PostRestored" }
