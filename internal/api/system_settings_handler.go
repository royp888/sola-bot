package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetSystemSettings handles GET /api/v1/system/settings.
func (s *Server) GetSystemSettings(c *gin.Context) {
	if s.deps.SystemSettings == nil {
		writeError(c, http.StatusServiceUnavailable, "settings service unavailable")
		return
	}
	data, err := s.deps.SystemSettings.Get(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateSystemSettings handles PUT /api/v1/system/settings.
func (s *Server) UpdateSystemSettings(c *gin.Context) {
	if s.deps.SystemSettings == nil {
		writeError(c, http.StatusServiceUnavailable, "settings service unavailable")
		return
	}
	var req SystemSettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := s.deps.SystemSettings.Update(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, data)
}

// resolveSystemKey returns the live value of a config key, checking DB first then falling back.
func (s *Server) resolveSystemKey(ctx context.Context, key, fallback string) string {
	if s.deps.SystemSettings != nil {
		if v := s.deps.SystemSettings.ResolveKey(ctx, key); v != "" {
			return v
		}
	}
	return fallback
}
