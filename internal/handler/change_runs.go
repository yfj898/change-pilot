package handler

import (
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (h *ChangeHandler) SubmitRun(c *gin.Context) {
	var req struct {
		SnapshotID       string `json:"snapshot_id"`
		RunID            string `json:"run_id"`
		ExpectedRevision int64  `json:"expected_revision"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid run submission").WithDetails(err.Error()))
		return
	}
	out, err := h.service.SubmitRun(c.Request.Context(), c.Param("id"), req.SnapshotID, req.RunID, req.ExpectedRevision)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": out})
}
func (h *ChangeHandler) Runs(c *gin.Context) {
	out, err := h.service.Runs(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}
func (h *ChangeHandler) Run(c *gin.Context) {
	out, err := h.service.Run(c.Request.Context(), c.Param("id"), c.Param("run_id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}
func (h *ChangeHandler) Artifact(c *gin.Context) {
	generation, err := strconv.ParseUint(c.Query("generation"), 10, 64)
	if err != nil || generation < 1 {
		c.Error(apperrors.NewBadRequestError("generation is required and must be positive"))
		return
	}
	out, err := h.service.Artifact(c.Request.Context(), c.Param("id"), c.Param("run_id"), generation, c.Param("artifact_id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", out)
}
func (h *ChangeHandler) Handoff(c *gin.Context) {
	var req struct {
		RunID            string `json:"run_id"`
		Conclusion       string `json:"conclusion"`
		Notes            string `json:"notes"`
		ExpectedRevision int64  `json:"expected_revision"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid handoff").WithDetails(err.Error()))
		return
	}
	out, err := h.service.Handoff(c.Request.Context(), c.Param("id"), req.RunID, req.Conclusion, req.Notes, req.ExpectedRevision)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": out})
}
func (h *ChangeHandler) Handoffs(c *gin.Context) {
	out, err := h.service.Handoffs(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}
