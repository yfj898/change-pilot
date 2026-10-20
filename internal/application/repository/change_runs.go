package repository

import (
	"context"
	"errors"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func (r *changeRepository) Snapshot(ctx context.Context, tenant uint64, changeID, id string) (*types.ChangeSnapshot, error) {
	var out types.ChangeSnapshot
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND change_id = ? AND id = ?", tenant, changeID, id).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFoundError("snapshot not found")
	}
	return &out, err
}
func (r *changeRepository) CreateRunIntent(ctx context.Context, link *types.ChangeRunLink, expected int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&types.ChangeRequest{}).Where("tenant_id = ? AND id = ? AND revision = ? AND stage <> ?", link.TenantID, link.ChangeID, expected, "closed").UpdateColumn("revision", gorm.Expr("revision"))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return apperrors.NewConflictError("change was modified or closed; reload before submitting")
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(link).Error
	})
}
func (r *changeRepository) RunLink(ctx context.Context, tenant uint64, id string) (*types.ChangeRunLink, error) {
	var out types.ChangeRunLink
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND run_id = ?", tenant, id).First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.NewNotFoundError("run link not found")
	}
	return &out, err
}
func (r *changeRepository) RunLinks(ctx context.Context, tenant uint64, id string) ([]*types.ChangeRunLink, error) {
	out := []*types.ChangeRunLink{}
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND change_id = ?", tenant, id).Order("created_at DESC, run_id DESC").Find(&out).Error
	return out, err
}
func (r *changeRepository) UpdateRunSubmission(ctx context.Context, tenant uint64, id, state, reason string) error {
	return r.db.WithContext(ctx).Model(&types.ChangeRunLink{}).Where("tenant_id = ? AND run_id = ?", tenant, id).Updates(map[string]any{"submission_state": state, "submission_error": reason, "updated_at": time.Now()}).Error
}
func (r *changeRepository) Handoff(ctx context.Context, h *types.ChangeHandoff, expected int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		update := map[string]any{"revision": expected + 1, "updated_at": time.Now()}
		if h.Conclusion == "accepted" {
			update["stage"] = "closed"
		}
		res := tx.Model(&types.ChangeRequest{}).Where("tenant_id = ? AND id = ? AND revision = ? AND stage = ?", h.TenantID, h.ChangeID, expected, "active").Updates(update)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return apperrors.NewConflictError("change was modified or closed; reload before handoff")
		}
		return tx.Create(h).Error
	})
}
func (r *changeRepository) Handoffs(ctx context.Context, tenant uint64, id string) ([]*types.ChangeHandoff, error) {
	out := []*types.ChangeHandoff{}
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND change_id = ?", tenant, id).Order("created_at DESC, id DESC").Find(&out).Error
	return out, err
}
