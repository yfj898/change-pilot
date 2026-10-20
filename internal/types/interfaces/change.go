package interfaces

import (
	"context"
	"github.com/Tencent/WeKnora/internal/types"
)

// ChangeRepository scopes every lookup by the authenticated workspace.
type ChangeRepository interface {
	Create(context.Context, *types.ChangeRequest) error
	Get(context.Context, uint64, string) (*types.ChangeRequest, error)
	List(context.Context, uint64, int, int) ([]*types.ChangeRequest, int64, error)
	UpdateCAS(context.Context, *types.ChangeRequest, int64) error
	Confirm(context.Context, *types.ChangeSnapshot, int64) error
	Snapshots(context.Context, uint64, string) ([]*types.ChangeSnapshot, error)
	Snapshot(context.Context, uint64, string, string) (*types.ChangeSnapshot, error)
	CreateRunIntent(context.Context, *types.ChangeRunLink, int64) error
	RunLink(context.Context, uint64, string) (*types.ChangeRunLink, error)
	RunLinks(context.Context, uint64, string) ([]*types.ChangeRunLink, error)
	UpdateRunSubmission(context.Context, uint64, string, string, string) error
	Handoff(context.Context, *types.ChangeHandoff, int64) error
	Handoffs(context.Context, uint64, string) ([]*types.ChangeHandoff, error)
}
