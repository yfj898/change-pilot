package repository

import (
	"context"
	"errors"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"time"
)

type changeRepository struct{ db *gorm.DB }

// NewChangeRepository reuses the application's database and migration lifecycle.
func NewChangeRepository(db *gorm.DB) interfaces.ChangeRepository { return &changeRepository{db: db} }

func (r *changeRepository) Create(ctx context.Context, c *types.ChangeRequest) error {
	return r.db.WithContext(ctx).Create(c).Error
}
func (r *changeRepository) Get(ctx context.Context, tenantID uint64, id string) (*types.ChangeRequest, error) {
	var c types.ChangeRequest
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFoundError("change not found")
	}
	return &c, err
}
func (r *changeRepository) List(ctx context.Context, tenantID uint64, offset, limit int) ([]*types.ChangeRequest, int64, error) {
	var total int64
	q := r.db.WithContext(ctx).Model(&types.ChangeRequest{}).Where("tenant_id = ?", tenantID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := []*types.ChangeRequest{}
	err := q.Omit("draft").Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}
func (r *changeRepository) UpdateCAS(ctx context.Context, c *types.ChangeRequest, expected int64) error {
	res := r.db.WithContext(ctx).Model(c).Where("tenant_id = ? AND id = ? AND revision = ? AND stage <> ?", c.TenantID, c.ID, expected, "closed").
		Select("title", "assignee_id", "notes", "input", "draft", "revision", "input_revision", "updated_at").Updates(c)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return apperrors.NewConflictError("change was modified or closed; reload before saving")
	}
	return nil
}
func (r *changeRepository) Confirm(ctx context.Context, snapshot *types.ChangeSnapshot, expected int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&types.ChangeRequest{}).Where("tenant_id = ? AND id = ? AND revision = ? AND stage <> ?", snapshot.TenantID, snapshot.ChangeID, expected, "closed").
			Updates(map[string]any{"stage": "active", "revision": expected + 1, "updated_at": time.Now()})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return apperrors.NewConflictError("change was modified or closed; confirm the current revision")
		}
		var count int64
		if err := tx.Model(&types.ChangeSnapshot{}).Where("tenant_id = ? AND change_id = ? AND input_revision = ?", snapshot.TenantID, snapshot.ChangeID, snapshot.InputRevision).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return apperrors.NewConflictError("this input revision already has a confirmed snapshot")
		}
		return tx.Create(snapshot).Error
	})
}
func (r *changeRepository) Snapshots(ctx context.Context, tenantID uint64, id string) ([]*types.ChangeSnapshot, error) {
	items := []*types.ChangeSnapshot{}
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND change_id = ?", tenantID, id).Order("created_at DESC, id DESC").Find(&items).Error
	return items, err
}
