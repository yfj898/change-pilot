package handler

import (
	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// ChangeHandler uses the same authenticated context and error envelope as knowledge APIs.
type ChangeHandler struct{ service *service.ChangeService }

func (h *ChangeHandler) PrepareDraft(c *gin.Context) {
	var req struct {
		ModelID          string  `json:"model_id"`
		Content          *string `json:"content"`
		ExpectedRevision int64   `json:"expected_revision"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid context draft").WithDetails(err.Error()))
		return
	}
	out, err := h.service.PrepareDraft(c.Request.Context(), c.Param("id"), req.ModelID, req.Content, req.ExpectedRevision)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": out})
}

// NewChangeHandler wires the additive change workflow into the existing server.
func NewChangeHandler(s *service.ChangeService) *ChangeHandler { return &ChangeHandler{service: s} }

func (h *ChangeHandler) Projects(c *gin.Context) {
	data, err := h.service.Projects(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
func (h *ChangeHandler) Create(c *gin.Context) {
	var req service.CreateChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid change body").WithDetails(err.Error()))
		return
	}
	data, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": data})
}
func (h *ChangeHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.Error(apperrors.NewBadRequestError("page must be positive"))
		return
	}
	size, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || size < 1 || size > 100 {
		c.Error(apperrors.NewBadRequestError("page_size must be between 1 and 100"))
		return
	}
	data, total, err := h.service.List(c.Request.Context(), page, size)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "total": total, "page": page, "page_size": size})
}
func (h *ChangeHandler) Get(c *gin.Context) {
	data, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
func (h *ChangeHandler) Patch(c *gin.Context) {
	var req service.PatchChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid change patch").WithDetails(err.Error()))
		return
	}
	data, err := h.service.Patch(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
func (h *ChangeHandler) Context(c *gin.Context) {
	data, err := h.service.Context(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
func (h *ChangeHandler) Confirm(c *gin.Context) {
	var req struct {
		ExpectedRevision int64 `json:"expected_revision"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid confirmation").WithDetails(err.Error()))
		return
	}
	data, err := h.service.Confirm(c.Request.Context(), c.Param("id"), req.ExpectedRevision)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": data})
}
