package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ListForwards handles GET /api/v1/forwards.
func (s *Server) ListForwards(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	var query CommonListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Forwards.List(c.Request.Context(), query, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

// CreateForward handles POST /api/v1/forwards.
func (s *Server) CreateForward(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	var req AutoForwardCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.deps.Forwards.Create(c.Request.Context(), req, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusCreated, created)
}

// GetForwardStats handles GET /api/v1/forwards/stats.
func (s *Server) GetForwardStats(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	stats, err := s.deps.Forwards.Stats(c.Request.Context(), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetForward handles GET /api/v1/forwards/:forward_id.
func (s *Server) GetForward(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	item, err := s.deps.Forwards.Get(c.Request.Context(), c.Param("forward_id"), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// UpdateForward handles PUT/PATCH /api/v1/forwards/:forward_id.
func (s *Server) UpdateForward(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	var req AutoForwardUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := s.deps.Forwards.Update(c.Request.Context(), c.Param("forward_id"), req, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// DeleteForward handles DELETE /api/v1/forwards/:forward_id.
func (s *Server) DeleteForward(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	if err := s.deps.Forwards.Delete(c.Request.Context(), c.Param("forward_id"), s.ownerUserID(c)); err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// StartForwardHistory handles POST /api/v1/forwards/:forward_id/history/start.
func (s *Server) StartForwardHistory(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	item, err := s.deps.Forwards.StartHistory(c.Request.Context(), c.Param("forward_id"), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// PauseForwardHistory handles POST /api/v1/forwards/:forward_id/history/pause.
func (s *Server) PauseForwardHistory(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	item, err := s.deps.Forwards.PauseHistory(c.Request.Context(), c.Param("forward_id"), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// ResetForwardHistory handles POST /api/v1/forwards/:forward_id/history/reset.
func (s *Server) ResetForwardHistory(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	item, err := s.deps.Forwards.ResetHistory(c.Request.Context(), c.Param("forward_id"), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// ListForwardLogs handles GET /api/v1/forwards/:forward_id/logs.
func (s *Server) ListForwardLogs(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	var query CommonListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	items, err := s.deps.Forwards.Logs(c.Request.Context(), c.Param("forward_id"), query, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

// PreviewForward handles POST /api/v1/forwards/:forward_id/preview.
func (s *Server) PreviewForward(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	var req AutoForwardPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	preview, err := s.deps.Forwards.Preview(c.Request.Context(), c.Param("forward_id"), req, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, preview)
}

// TestSendForward handles POST /api/v1/forwards/:forward_id/test-send.
func (s *Server) TestSendForward(c *gin.Context) {
	if s.deps.Forwards == nil {
		writeError(c, http.StatusServiceUnavailable, "forward service is not configured")
		return
	}

	var req AutoForwardPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	log, err := s.deps.Forwards.TestSend(c.Request.Context(), c.Param("forward_id"), req, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, log)
}
