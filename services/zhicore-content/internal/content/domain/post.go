package domain

import (
	"strings"
	"time"
)

const MaxPostTitleRunes = 200

type PostID int64
type PublicPostID string
type OwnerID int64
type PostTitle string
type PostSummary string

type PostStatus string

const (
	PostStatusDraft     PostStatus = "DRAFT"
	PostStatusPublished PostStatus = "PUBLISHED"
	PostStatusScheduled PostStatus = "SCHEDULED"
	PostStatusDeleted   PostStatus = "DELETED"
)

type BodyPointer struct {
	ID              string
	Hash            string
	PlainTextLength int
	SizeBytes       int
}

func (p BodyPointer) IsZero() bool {
	return p.ID == "" || p.Hash == ""
}

type OwnerSnapshot struct {
	DisplayName    string
	AvatarFileID   string
	ProfileVersion int64
}

type CreateDraftInput struct {
	PublicID  PublicPostID
	OwnerID   OwnerID
	Title     string
	Summary   string
	Owner     OwnerSnapshot
	DraftBody *BodyPointer
}

type HydratePostInput struct {
	ID            PostID
	PublicID      PublicPostID
	OwnerID       OwnerID
	Title         string
	Summary       string
	Owner         OwnerSnapshot
	Status        PostStatus
	DraftBody     *BodyPointer
	PublishedBody *BodyPointer
}

type PublishInput struct {
	DraftBody   BodyPointer
	PublishedAt time.Time
}

func HydratePost(input HydratePostInput) (*Post, error) {
	title, err := NewPostTitle(input.Title)
	if err != nil {
		return nil, err
	}
	post := &Post{
		id:       input.ID,
		publicID: input.PublicID,
		ownerID:  input.OwnerID,
		title:    title,
		summary:  PostSummary(strings.TrimSpace(input.Summary)),
		owner:    input.Owner,
		status:   input.Status,
	}
	if input.DraftBody != nil {
		body := *input.DraftBody
		post.draftBody = &body
	}
	if input.PublishedBody != nil {
		body := *input.PublishedBody
		post.publishedBody = &body
	}
	return post, nil
}

type PostPublishPolicy struct {
	minPlainTextRunes int
}

func NewPostPublishPolicy(minPlainTextRunes int) PostPublishPolicy {
	if minPlainTextRunes < 1 {
		minPlainTextRunes = 1
	}
	return PostPublishPolicy{minPlainTextRunes: minPlainTextRunes}
}

type PostFactory struct{}

func (PostFactory) CreateDraft(input CreateDraftInput) (*Post, error) {
	title, err := NewPostTitle(input.Title)
	if err != nil {
		return nil, err
	}

	post := &Post{
		publicID: input.PublicID,
		ownerID:  input.OwnerID,
		title:    title,
		summary:  PostSummary(strings.TrimSpace(input.Summary)),
		owner:    input.Owner,
		status:   PostStatusDraft,
		events: []DomainEvent{
			PostCreated{PublicID: input.PublicID, OwnerID: input.OwnerID},
		},
	}
	if input.DraftBody != nil {
		body := *input.DraftBody
		post.draftBody = &body
	}
	return post, nil
}

func NewPostTitle(raw string) (PostTitle, error) {
	title := strings.TrimSpace(raw)
	if len([]rune(title)) > MaxPostTitleRunes {
		return "", ErrTitleTooLong
	}
	return PostTitle(title), nil
}

type Post struct {
	id            PostID
	publicID      PublicPostID
	ownerID       OwnerID
	title         PostTitle
	summary       PostSummary
	owner         OwnerSnapshot
	status        PostStatus
	draftBody     *BodyPointer
	publishedBody *BodyPointer
	publishedAt   *time.Time
	deletedAt     *time.Time
	events        []DomainEvent
}

func (p *Post) ID() PostID                   { return p.id }
func (p *Post) PublicID() PublicPostID       { return p.publicID }
func (p *Post) OwnerID() OwnerID             { return p.ownerID }
func (p *Post) Title() PostTitle             { return p.title }
func (p *Post) Summary() PostSummary         { return p.summary }
func (p *Post) OwnerSnapshot() OwnerSnapshot { return p.owner }
func (p *Post) Status() PostStatus           { return p.status }

func (p *Post) DraftBody() BodyPointer {
	if p.draftBody == nil {
		return BodyPointer{}
	}
	return *p.draftBody
}

func (p *Post) PublishedBody() BodyPointer {
	if p.publishedBody == nil {
		return BodyPointer{}
	}
	return *p.publishedBody
}

func (p *Post) SaveDraftBody(pointer BodyPointer) error {
	if p.status == PostStatusDeleted {
		// Deleted posts are intentionally frozen so cleanup / repair workers do
		// not race with authors creating new draft pointers on removed content.
		return ErrPostDeleted
	}
	if pointer.IsZero() {
		return ErrBodyRequired
	}
	body := pointer
	p.draftBody = &body
	return nil
}

func (p *Post) Publish(policy PostPublishPolicy, input PublishInput) error {
	if p.status == PostStatusDeleted {
		return ErrPostDeleted
	}
	if p.status == PostStatusPublished {
		return ErrPostAlreadyPublished
	}
	if p.status != PostStatusDraft {
		return ErrDraftConflict
	}
	if strings.TrimSpace(string(p.title)) == "" {
		return ErrTitleRequired
	}
	if input.DraftBody.IsZero() {
		return ErrBodyRequired
	}
	if input.DraftBody.PlainTextLength == 0 {
		return ErrBodyRequired
	}
	if input.DraftBody.PlainTextLength < policy.minPlainTextRunes {
		// The minimum effective text guard keeps media-only or whitespace-only
		// drafts from becoming public posts even when a body document exists.
		return ErrBodyTooShort
	}

	body := input.DraftBody
	p.publishedBody = &body
	p.draftBody = nil
	p.status = PostStatusPublished
	publishedAt := input.PublishedAt
	p.publishedAt = &publishedAt
	p.events = append(p.events, PostPublished{
		PublicID:    p.publicID,
		OwnerID:     p.ownerID,
		PublishedAt: input.PublishedAt,
	})
	return nil
}

func (p *Post) Delete(deletedAt time.Time) {
	p.status = PostStatusDeleted
	p.deletedAt = &deletedAt
	p.events = append(p.events, PostDeleted{
		PublicID:  p.publicID,
		OwnerID:   p.ownerID,
		DeletedAt: deletedAt,
	})
}

func (p *Post) Unpublish(unpublishedAt time.Time) error {
	if p.status == PostStatusDeleted {
		return ErrPostDeleted
	}
	if p.status != PostStatusPublished {
		return ErrPostNotPublished
	}
	p.status = PostStatusDraft
	p.events = append(p.events, PostUnpublished{
		PublicID:      p.publicID,
		OwnerID:       p.ownerID,
		UnpublishedAt: unpublishedAt,
	})
	return nil
}

func (p *Post) Restore(restoredAt time.Time) error {
	if p.status != PostStatusDeleted {
		return ErrPostNotFound
	}
	p.status = PostStatusDraft
	p.deletedAt = nil
	p.events = append(p.events, PostRestored{
		PublicID:   p.publicID,
		OwnerID:    p.ownerID,
		RestoredAt: restoredAt,
	})
	return nil
}

func (p *Post) PullEvents() []DomainEvent {
	events := append([]DomainEvent(nil), p.events...)
	p.events = nil
	return events
}
