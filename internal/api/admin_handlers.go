package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) GetChatAdminConfig(c *gin.Context) {
	if s.deps.Admin == nil {
		writeError(c, http.StatusInternalServerError, "chat admin service is not configured")
		return
	}
	chatID, err := parseAnyChatID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chatID")
		return
	}
	config, err := s.deps.Admin.GetConfig(c.Request.Context(), chatID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, config)
}

func (s *Server) UpdateChatAdminConfig(c *gin.Context) {
	if s.deps.Admin == nil {
		writeError(c, http.StatusInternalServerError, "chat admin service is not configured")
		return
	}
	chatID, err := parseAnyChatID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chatID")
		return
	}
	var req ChatAdminConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	config, err := s.deps.Admin.UpdateConfig(c.Request.Context(), chatID, req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, config)
}

func (s *Server) ListBanLogs(c *gin.Context) {
	if s.deps.Admin == nil {
		writeError(c, http.StatusInternalServerError, "chat admin service is not configured")
		return
	}
	chatID, err := parseAnyChatID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chatID")
		return
	}
	var query CommonListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := s.deps.Admin.ListBans(c.Request.Context(), chatID, query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) AdminBan(c *gin.Context) {
	if s.deps.Admin == nil {
		writeError(c, http.StatusInternalServerError, "chat admin service is not configured")
		return
	}
	var req AdminBanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}
	req.OwnerUserID = s.ownerUserID(c)
	if err := s.deps.Admin.Ban(c.Request.Context(), req); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) AdminUnban(c *gin.Context) {
	if s.deps.Admin == nil {
		writeError(c, http.StatusInternalServerError, "chat admin service is not configured")
		return
	}
	chatID, err := parseAnyChatID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chatID")
		return
	}
	userID, err := parseUserID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid userID")
		return
	}
	if err := s.deps.Admin.Unban(c.Request.Context(), chatID, userID); err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) ListWarnRecords(c *gin.Context) {
	if s.deps.Admin == nil {
		writeError(c, http.StatusInternalServerError, "chat admin service is not configured")
		return
	}
	chatID, err := parseAnyChatID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chatID")
		return
	}
	userID, err := parseUserID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid userID")
		return
	}
	items, err := s.deps.Admin.ListWarns(c.Request.Context(), chatID, userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func parseAnyChatID(c *gin.Context) (int64, error) {
	if value := c.Param("chatID"); value != "" {
		return strconv.ParseInt(value, 10, 64)
	}
	return parseChatID(c)
}

func parseUserID(c *gin.Context) (int64, error) {
	if value := c.Param("userID"); value != "" {
		return strconv.ParseInt(value, 10, 64)
	}
	return strconv.ParseInt(c.Param("user_id"), 10, 64)
}
