package application

import (
	"fmt"

	userevents "github.com/architectcgz/zhicore-go/libs/contracts/events/user"
	"github.com/architectcgz/zhicore-go/services/zhicore-user/internal/user/ports"
)

type Dependencies struct {
	Profiles      ports.ProfileRepository
	Queries       ports.ProfileQueryRepository
	Relationships ports.RelationshipRepository
	Files         ports.FileReferenceClient
	IDs           ports.PublicIDGenerator
	Outbox        ports.OutboxPublisher
	TxRunner      ports.TransactionRunner
	Clock         ports.Clock
	Cache         ports.CacheStore
	CacheFailures ports.CacheFailureRecorder
}

type Service struct {
	profiles      ports.ProfileRepository
	queries       ports.ProfileQueryRepository
	relationships ports.RelationshipRepository
	files         ports.FileReferenceClient
	ids           ports.PublicIDGenerator
	outbox        ports.OutboxPublisher
	txRunner      ports.TransactionRunner
	clock         ports.Clock
	cache         ports.CacheStore
	cacheFailures ports.CacheFailureRecorder
}

const (
	relationshipEventUserFollowed   = userevents.EventFollowed
	relationshipEventUserUnfollowed = userevents.EventUnfollowed
	relationshipEventUserBlocked    = userevents.EventBlocked
	relationshipEventUserUnblocked  = userevents.EventUnblocked
)

func NewService(deps Dependencies) (*Service, error) {
	if deps.Profiles == nil {
		return nil, fmt.Errorf("Profiles is required")
	}
	if deps.Queries == nil {
		return nil, fmt.Errorf("Queries is required")
	}
	if deps.Files == nil {
		return nil, fmt.Errorf("Files is required")
	}
	if deps.IDs == nil {
		return nil, fmt.Errorf("IDs is required")
	}
	if deps.Outbox == nil {
		return nil, fmt.Errorf("Outbox is required")
	}
	if deps.TxRunner == nil {
		return nil, fmt.Errorf("TxRunner is required")
	}
	if deps.Clock == nil {
		return nil, fmt.Errorf("Clock is required")
	}
	if deps.Cache == nil {
		return nil, fmt.Errorf("Cache is required")
	}
	if deps.CacheFailures == nil {
		return nil, fmt.Errorf("CacheFailures is required")
	}
	return &Service{
		profiles:      deps.Profiles,
		queries:       deps.Queries,
		relationships: deps.Relationships,
		files:         deps.Files,
		ids:           deps.IDs,
		outbox:        deps.Outbox,
		txRunner:      deps.TxRunner,
		clock:         deps.Clock,
		cache:         deps.Cache,
		cacheFailures: deps.CacheFailures,
	}, nil
}
