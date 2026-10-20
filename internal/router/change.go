package router

import (
	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

// RegisterChangeRoutes declares read_changes for reads; writes stay API-key denied.
// The unrelated retrieve capability grants no access to this workflow.
func RegisterChangeRoutes(v1 *gin.RouterGroup, h *handler.ChangeHandler, g *rbacGuards) {
	if h == nil {
		return
	}
	group := v1.Group("/changes")
	group.GET("/projects", g.Viewer(), h.Projects)
	reads := g.apiKeyGroup(group, apiKeyAny().WithCapability(types.APIKeyCapabilityReadChanges))
	reads.GET("", g.Viewer(), h.List)
	group.POST("", g.Contributor(), h.Create)
	reads.GET("/:id", g.Viewer(), h.Get)
	group.PATCH("/:id", g.Contributor(), h.Patch)
	reads.GET("/:id/context", g.Viewer(), h.Context)
	group.POST("/:id/snapshots", g.Contributor(), h.Confirm)
	group.POST("/:id/context-draft", g.Contributor(), h.PrepareDraft)
	group.POST("/:id/runs", g.Contributor(), h.SubmitRun)
	reads.GET("/:id/runs", g.Viewer(), h.Runs)
	reads.GET("/:id/runs/:run_id", g.Viewer(), h.Run)
	reads.GET("/:id/runs/:run_id/artifacts/:artifact_id", g.Viewer(), h.Artifact)
	group.POST("/:id/handoffs", g.Contributor(), h.Handoff)
	reads.GET("/:id/handoffs", g.Viewer(), h.Handoffs)
}
