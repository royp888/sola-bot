package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func broadcastStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, ErrForbidden) {
		return http.StatusForbidden
	}
	msg := err.Error()
	switch {
	case containsAny(msg, "not found"):
		return http.StatusNotFound
	case containsAny(msg, "invalid", "must be", "cannot", "needs", "no recipients", "already"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func containsAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if len(needle) == 0 {
			continue
		}
		if indexOfFold(haystack, needle) >= 0 {
			return true
		}
	}
	return false
}

func indexOfFold(haystack string, needle string) int {
	if len(needle) == 0 {
		return 0
	}
	h := toLowerASCII(haystack)
	n := toLowerASCII(needle)
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}

func toLowerASCII(value string) string {
	out := []byte(value)
	for i, b := range out {
		if b >= 'A' && b <= 'Z' {
			out[i] = b + ('a' - 'A')
		}
	}
	return string(out)
}

// ListBroadcasts handles GET /api/v1/broadcasts.
func (s *Server) ListBroadcasts(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	var query CommonListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Broadcasts.List(c.Request.Context(), query, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

// CreateBroadcast handles POST /api/v1/broadcasts.
func (s *Server) CreateBroadcast(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	var req BroadcastCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.deps.Broadcasts.Create(c.Request.Context(), req, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusCreated, created)
}

// GetBroadcastStats handles GET /api/v1/broadcasts/stats.
func (s *Server) GetBroadcastStats(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	stats, err := s.deps.Broadcasts.Stats(c.Request.Context(), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, stats)
}

// PreviewBroadcastTargets handles POST /api/v1/broadcasts/preview.
func (s *Server) PreviewBroadcastTargets(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	var req BroadcastCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	preview, err := s.deps.Broadcasts.PreviewTargets(c.Request.Context(), req, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, preview)
}

// GetBroadcast handles GET /api/v1/broadcasts/:broadcast_id.
func (s *Server) GetBroadcast(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	item, err := s.deps.Broadcasts.Get(c.Request.Context(), c.Param("broadcast_id"), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// UpdateBroadcast handles PUT /api/v1/broadcasts/:broadcast_id.
func (s *Server) UpdateBroadcast(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	var req BroadcastUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := s.deps.Broadcasts.Update(c.Request.Context(), c.Param("broadcast_id"), req, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// DeleteBroadcast handles DELETE /api/v1/broadcasts/:broadcast_id.
func (s *Server) DeleteBroadcast(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	if err := s.deps.Broadcasts.Delete(c.Request.Context(), c.Param("broadcast_id"), s.ownerUserID(c)); err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// SendBroadcast handles POST /api/v1/broadcasts/:broadcast_id/send.
func (s *Server) SendBroadcast(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	item, err := s.deps.Broadcasts.Send(c.Request.Context(), c.Param("broadcast_id"), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// CancelBroadcast handles POST /api/v1/broadcasts/:broadcast_id/cancel.
func (s *Server) CancelBroadcast(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	item, err := s.deps.Broadcasts.Cancel(c.Request.Context(), c.Param("broadcast_id"), s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// ListBroadcastDeliveries handles GET /api/v1/broadcasts/:broadcast_id/deliveries.
func (s *Server) ListBroadcastDeliveries(c *gin.Context) {
	if s.deps.Broadcasts == nil {
		writeError(c, http.StatusServiceUnavailable, "broadcast service is not configured")
		return
	}

	var query CommonListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	items, err := s.deps.Broadcasts.Deliveries(c.Request.Context(), c.Param("broadcast_id"), query, s.ownerUserID(c))
	if err != nil {
		writeError(c, broadcastStatusCode(err), err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}
