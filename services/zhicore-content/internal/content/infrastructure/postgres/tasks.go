package postgres

import (
	"context"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type CleanupTaskStore struct {
	store *Store
}

func NewCleanupTaskStore(store *Store) *CleanupTaskStore {
	return &CleanupTaskStore{store: store}
}

func (s *CleanupTaskStore) Append(ctx context.Context, tx ports.Tx, task ports.BodyCleanupTask) error {
	execer, err := s.store.execer(tx)
	if err != nil {
		return err
	}
	return appendCleanupTask(ctx, execer, task)
}

func (s *CleanupTaskStore) AppendOutsideTx(ctx context.Context, task ports.BodyCleanupTask) error {
	return appendCleanupTask(ctx, s.store.db, task)
}

type RepairTaskStore struct {
	store *Store
}

func NewRepairTaskStore(store *Store) *RepairTaskStore {
	return &RepairTaskStore{store: store}
}

func (s *RepairTaskStore) Append(ctx context.Context, tx ports.Tx, task ports.BodyRepairTask) error {
	execer, err := s.store.execer(tx)
	if err != nil {
		return err
	}
	return appendRepairTask(ctx, execer, task)
}

func (s *RepairTaskStore) AppendOutsideTx(ctx context.Context, task ports.BodyRepairTask) error {
	return appendRepairTask(ctx, s.store.db, task)
}

var _ ports.BodyCleanupTaskStore = (*CleanupTaskStore)(nil)
var _ ports.BodyRepairTaskStore = (*RepairTaskStore)(nil)
