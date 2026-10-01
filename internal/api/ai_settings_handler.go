package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetAiSettings handles GET /api/v1/system/ai-settings.
func (s *Server) GetAiSettings(c *gin.Context) {
	if s.deps.AiSettings == nil {
		writeError(c, http.StatusServiceUnavailable, "ai settings service unavailable")
		return
	}
	data, err := s.deps.AiSettings.Get(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateAiSettings handles PUT /api/v1/system/ai-settings.
func (s *Server) UpdateAiSettings(c *gin.Context) {
	if s.deps.AiSettings == nil {
		writeError(c, http.StatusServiceUnavailable, "ai settings service unavailable")
		return
	}
	var req AiSettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := s.deps.AiSettings.Update(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, data)
}
