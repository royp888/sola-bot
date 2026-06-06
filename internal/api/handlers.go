package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Server struct {
	deps Dependencies
}

func NewServer(deps Dependencies) *Server {
	return &Server{deps: deps}
}

func (s *Server) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		OK:        true,
		Service:   "tgbot-api",
		Timestamp: time.Now().UTC(),
	})
}

// Login godoc
// @Summary Admin login
// @Tags admin
// @Accept json
// @Produce json
// @Param request body AdminLoginRequest true "admin credentials"
// @Success 200 {object} AdminLoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/login [post]
func (s *Server) Login(c *gin.Context) {
	if s.deps.Auth == nil {
		writeError(c, http.StatusInternalServerError, "auth service is not configured")
		return
	}

	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	username := req.Username
	if strings.TrimSpace(username) == "" {
		username = req.Email
	}
	rateUsername, rateIP := loginAttemptIdentity(username, c.GetHeader("X-Forwarded-For"), c.ClientIP())
	rateKey := loginRateLimitKey(rateUsername, rateIP)
	allowed, retryAfter, err := allowLoginAttempt(c.Request.Context(), s.deps.Redis, rateKey)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if !allowed {
		c.Header("Retry-After", formatRetryAfter(retryAfter))
		writeError(c, http.StatusTooManyRequests, unauthorizedRateLimitError(retryAfter).Error())
		return
	}

	admin, err := s.deps.Auth.Authenticate(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err.Error())
		return
	}
	_ = clearLoginAttempts(c.Request.Context(), s.deps.Redis, rateKey)

	token, expiresAt, err := s.issueToken(admin)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, AdminLoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        admin,
		Admin:       admin,
	})
}

func (s *Server) TelegramLogin(c *gin.Context) {
	if s.deps.Auth == nil {
		writeError(c, http.StatusInternalServerError, "auth service is not configured")
		return
	}

	var req TelegramLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	admin, err := s.deps.Auth.AuthenticateTelegram(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusUnauthorized, err.Error())
		return
	}

	token, expiresAt, err := s.issueToken(admin)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, AdminLoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        admin,
		Admin:       admin,
	})
}

// GetBotConfig godoc
// @Summary Get bot config
// @Tags bot
// @Security BearerAuth
// @Produce json
// @Success 200 {object} BotConfig
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bot/config [get]
func (s *Server) GetBotConfig(c *gin.Context) {
	if s.deps.BotConfig == nil {
		writeError(c, http.StatusInternalServerError, "bot config service is not configured")
		return
	}

	config, err := s.deps.BotConfig.Get(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateBotConfig godoc
// @Summary Update bot config
// @Tags bot
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body BotConfigUpdateRequest true "bot config patch"
// @Success 200 {object} BotConfig
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /bot/config [put]
func (s *Server) UpdateBotConfig(c *gin.Context) {
	if s.deps.BotConfig == nil {
		writeError(c, http.StatusInternalServerError, "bot config service is not configured")
		return
	}

	var req BotConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	config, err := s.deps.BotConfig.Update(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, config)
}

// ListChats godoc
// @Summary List bound chats
// @Tags chats
// @Security BearerAuth
// @Produce json
// @Param limit query int false "page size"
// @Param offset query int false "page offset"
// @Success 200 {array} ChatBinding
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /chats [get]
func (s *Server) ListChats(c *gin.Context) {
	if s.deps.Chats == nil {
		writeError(c, http.StatusInternalServerError, "chat binding service is not configured")
		return
	}

	var query CommonListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if claims, ok := CurrentAdminClaims(c); ok && !claims.IsSuperAdmin() {
		query.OwnerUserID = claims.UserID
	}

	items, err := s.deps.Chats.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

// BindChat godoc
// @Summary Bind a chat or channel
// @Tags chats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ChatBindingRequest true "binding payload"
// @Success 200 {object} ChatBinding
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /chats/bind [post]
func (s *Server) BindChat(c *gin.Context) {
	if s.deps.Chats == nil {
		writeError(c, http.StatusInternalServerError, "chat binding service is not configured")
		return
	}

	var req ChatBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}

	binding, err := s.deps.Chats.Bind(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, binding)
}

// UnbindChat godoc
// @Summary Unbind a chat or channel
// @Tags chats
// @Security BearerAuth
// @Produce json
// @Param chat_id path int true "chat id"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /chats/{chat_id}/bind [delete]
func (s *Server) UnbindChat(c *gin.Context) {
	if s.deps.Chats == nil {
		writeError(c, http.StatusInternalServerError, "chat binding service is not configured")
		return
	}

	chatID, err := parseChatID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chat_id")
		return
	}

	if err := s.deps.Chats.Unbind(c.Request.Context(), chatID); err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// GetChatPointConfig godoc
// @Summary Get chat points config
// @Tags chats
// @Security BearerAuth
// @Produce json
// @Param chat_id path int true "chat id"
// @Success 200 {object} ChatPointConfig
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /chats/{chat_id}/points-config [get]
func (s *Server) GetChatPointConfig(c *gin.Context) {
	if s.deps.ChatPointConfigs == nil {
		writeError(c, http.StatusInternalServerError, "chat point config service is not configured")
		return
	}

	chatID, err := parseChatID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chat_id")
		return
	}

	config, err := s.deps.ChatPointConfigs.Get(c.Request.Context(), chatID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateChatPointConfig godoc
// @Summary Update chat points config
// @Tags chats
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param chat_id path int true "chat id"
// @Param request body ChatPointConfigUpdateRequest true "points config patch"
// @Success 200 {object} ChatPointConfig
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /chats/{chat_id}/points-config [put]
func (s *Server) UpdateChatPointConfig(c *gin.Context) {
	if s.deps.ChatPointConfigs == nil {
		writeError(c, http.StatusInternalServerError, "chat point config service is not configured")
		return
	}

	chatID, err := parseChatID(c)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chat_id")
		return
	}

	var req ChatPointConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, chatID) {
		return
	}

	config, err := s.deps.ChatPointConfigs.Update(c.Request.Context(), chatID, req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, config)
}

func (s *Server) GetPointConfig(c *gin.Context) {
	if s.deps.ChatPointConfigs == nil {
		writeError(c, http.StatusInternalServerError, "chat point config service is not configured")
		return
	}
	chatID, err := parseInt64Path(c, "chatID")
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chatID")
		return
	}
	if !s.ensureChatAllowed(c, chatID) {
		return
	}
	config, err := s.deps.ChatPointConfigs.Get(c.Request.Context(), chatID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, config)
}

func (s *Server) UpdatePointConfig(c *gin.Context) {
	if s.deps.ChatPointConfigs == nil {
		writeError(c, http.StatusInternalServerError, "chat point config service is not configured")
		return
	}
	chatID, err := parseInt64Path(c, "chatID")
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chatID")
		return
	}
	var req ChatPointConfigUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, chatID) {
		return
	}
	config, err := s.deps.ChatPointConfigs.Update(c.Request.Context(), chatID, req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, config)
}

func (s *Server) GetPointRank(c *gin.Context) {
	if s.deps.Points == nil {
		writeError(c, http.StatusInternalServerError, "points service is not configured")
		return
	}
	chatID, err := parseInt64Path(c, "chatID")
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid chatID")
		return
	}
	if !s.ensureChatAllowed(c, chatID) {
		return
	}
	var query PointRankQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := s.deps.Points.GetRank(c.Request.Context(), chatID, query.Period, query.Limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) GetPointUser(c *gin.Context) {
	if s.deps.Points == nil {
		writeError(c, http.StatusInternalServerError, "points service is not configured")
		return
	}
	chatID, userID, ok := parseChatAndUserID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid chatID or userID")
		return
	}
	if !s.ensureChatAllowed(c, chatID) {
		return
	}
	item, err := s.deps.Points.GetUser(c.Request.Context(), chatID, userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) AdjustPointUser(c *gin.Context) {
	if s.deps.Points == nil {
		writeError(c, http.StatusInternalServerError, "points service is not configured")
		return
	}
	chatID, userID, ok := parseChatAndUserID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid chatID or userID")
		return
	}
	var req PointUserAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, chatID) {
		return
	}
	item, err := s.deps.Points.AdjustUser(c.Request.Context(), chatID, userID, req.Delta, req.Reason)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) ListPointLogs(c *gin.Context) {
	if s.deps.Points == nil {
		writeError(c, http.StatusInternalServerError, "points service is not configured")
		return
	}
	chatID, userID, ok := parseChatAndUserID(c)
	if !ok {
		writeError(c, http.StatusBadRequest, "invalid chatID or userID")
		return
	}
	if !s.ensureChatAllowed(c, chatID) {
		return
	}
	var query PointLogListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)
	items, err := s.deps.Points.ListLogs(c.Request.Context(), chatID, userID, query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) ListLotteries(c *gin.Context) {
	if s.deps.Lotteries == nil {
		writeError(c, http.StatusInternalServerError, "lottery service is not configured")
		return
	}

	var query LotteryListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Lotteries.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

func (s *Server) CreateLottery(c *gin.Context) {
	if s.deps.Lotteries == nil {
		writeError(c, http.StatusInternalServerError, "lottery service is not configured")
		return
	}

	var req LotteryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}

	item, err := s.deps.Lotteries.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, item)
}

func (s *Server) CancelLottery(c *gin.Context) {
	if s.deps.Lotteries == nil {
		writeError(c, http.StatusInternalServerError, "lottery service is not configured")
		return
	}

	id, err := parseInt64Path(c, "id")
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid lottery id")
		return
	}

	if err := s.deps.Lotteries.Cancel(c.Request.Context(), id, s.ownerUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) ListLotteryEntries(c *gin.Context) {
	if s.deps.Lotteries == nil {
		writeError(c, http.StatusInternalServerError, "lottery service is not configured")
		return
	}

	id, err := parseInt64Path(c, "id")
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid lottery id")
		return
	}

	items, err := s.deps.Lotteries.Entries(c.Request.Context(), id, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, items)
}

func (s *Server) ListLotteryWinners(c *gin.Context) {
	if s.deps.Lotteries == nil {
		writeError(c, http.StatusInternalServerError, "lottery service is not configured")
		return
	}

	id, err := parseInt64Path(c, "id")
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid lottery id")
		return
	}

	items, err := s.deps.Lotteries.Winners(c.Request.Context(), id, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, items)
}

// ListLevels godoc
// @Summary List levels
// @Tags levels
// @Security BearerAuth
// @Produce json
// @Param chat_id query int false "chat id"
// @Param limit query int false "page size"
// @Param offset query int false "page offset"
// @Success 200 {array} Level
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /levels [get]
func (s *Server) ListLevels(c *gin.Context) {
	if s.deps.Levels == nil {
		writeError(c, http.StatusInternalServerError, "level service is not configured")
		return
	}

	var query LevelListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Levels.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

// CreateLevel godoc
// @Summary Create a level
// @Tags levels
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body LevelCreateRequest true "level payload"
// @Success 200 {object} Level
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /levels [post]
func (s *Server) CreateLevel(c *gin.Context) {
	if s.deps.Levels == nil {
		writeError(c, http.StatusInternalServerError, "level service is not configured")
		return
	}

	var req LevelCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}

	item, err := s.deps.Levels.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// UpdateLevel godoc
// @Summary Update a level
// @Tags levels
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param level_id path int true "level id"
// @Param request body LevelUpdateRequest true "level patch"
// @Success 200 {object} Level
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /levels/{level_id} [patch]
func (s *Server) UpdateLevel(c *gin.Context) {
	if s.deps.Levels == nil {
		writeError(c, http.StatusInternalServerError, "level service is not configured")
		return
	}

	levelID := c.Param("level_id")
	if strings.TrimSpace(levelID) == "" {
		writeError(c, http.StatusBadRequest, "invalid level_id")
		return
	}

	var req LevelUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.ChatID == 0 {
		if value := strings.TrimSpace(c.Query("chat_id")); value != "" {
			chatID, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				writeError(c, http.StatusBadRequest, "invalid chat_id")
				return
			}
			req.ChatID = chatID
		}
	}
	if req.ChatID != 0 && !s.ensureChatAllowed(c, req.ChatID) {
		return
	}

	item, err := s.deps.Levels.Update(c.Request.Context(), levelID, req, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// DeleteLevel godoc
// @Summary Delete a level
// @Tags levels
// @Security BearerAuth
// @Produce json
// @Param level_id path int true "level id"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /levels/{level_id} [delete]
func (s *Server) DeleteLevel(c *gin.Context) {
	if s.deps.Levels == nil {
		writeError(c, http.StatusInternalServerError, "level service is not configured")
		return
	}

	levelID := c.Param("level_id")
	if strings.TrimSpace(levelID) == "" {
		writeError(c, http.StatusBadRequest, "invalid level_id")
		return
	}

	if err := s.deps.Levels.Delete(c.Request.Context(), levelID, s.ownerUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListAdminViolations godoc
// @Summary List admin violations
// @Tags admin
// @Security BearerAuth
// @Produce json
// @Param chat_id query int false "chat id"
// @Param user_id query int false "user id"
// @Param type query string false "violation type"
// @Param status query string false "status"
// @Param limit query int false "page size"
// @Param offset query int false "page offset"
// @Param cursor query string false "page cursor"
// @Success 200 {object} CursorListResponse[AdminViolation]
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/violations [get]
func (s *Server) ListAdminViolations(c *gin.Context) {
	if s.deps.AdminViolations == nil {
		writeError(c, http.StatusInternalServerError, "admin violation service is not configured")
		return
	}

	var query AdminViolationListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if query.ChatID != 0 && !s.ensureChatAllowed(c, query.ChatID) {
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.AdminViolations.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

// UpdateAdminViolation godoc
// @Summary Update an admin violation
// @Tags admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param violation_id path int true "violation id"
// @Param request body AdminViolationUpdateRequest true "violation patch"
// @Success 200 {object} AdminViolation
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /admin/violations/{violation_id} [patch]
func (s *Server) UpdateAdminViolation(c *gin.Context) {
	if s.deps.AdminViolations == nil {
		writeError(c, http.StatusInternalServerError, "admin violation service is not configured")
		return
	}

	violationID := c.Param("violation_id")
	if strings.TrimSpace(violationID) == "" {
		writeError(c, http.StatusBadRequest, "invalid violation_id")
		return
	}

	var req AdminViolationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := s.deps.AdminViolations.Update(c.Request.Context(), violationID, req, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// ListKeywords godoc
// @Summary List keywords
// @Tags keywords
// @Security BearerAuth
// @Produce json
// @Param chat_id query int false "chat id"
// @Param scope query string false "scope"
// @Param action query string false "action"
// @Param enabled query bool false "enabled"
// @Param limit query int false "page size"
// @Param offset query int false "page offset"
// @Success 200 {array} Keyword
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /keywords [get]
func (s *Server) ListAuditLogs(c *gin.Context) {
	if s.deps.AuditLogs == nil {
		writeError(c, http.StatusInternalServerError, "audit log service is not configured")
		return
	}
	var query AuditLogListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if query.ChatID != 0 && !s.ensureChatAllowed(c, query.ChatID) {
		return
	}
	query.OwnerUserID = s.ownerUserID(c)
	items, err := s.deps.AuditLogs.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) ListKeywords(c *gin.Context) {
	if s.deps.Keywords == nil {
		writeError(c, http.StatusInternalServerError, "keyword service is not configured")
		return
	}

	var query KeywordListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Keywords.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

// CreateKeyword godoc
// @Summary Create a keyword
// @Tags keywords
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body KeywordCreateRequest true "keyword payload"
// @Success 200 {object} Keyword
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /keywords [post]
func (s *Server) CreateKeyword(c *gin.Context) {
	if s.deps.Keywords == nil {
		writeError(c, http.StatusInternalServerError, "keyword service is not configured")
		return
	}

	var req KeywordCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}

	item, err := s.deps.Keywords.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// UpdateKeyword godoc
// @Summary Update a keyword
// @Tags keywords
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param keyword_id path int true "keyword id"
// @Param request body KeywordUpdateRequest true "keyword patch"
// @Success 200 {object} Keyword
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /keywords/{keyword_id} [patch]
func (s *Server) UpdateKeyword(c *gin.Context) {
	if s.deps.Keywords == nil {
		writeError(c, http.StatusInternalServerError, "keyword service is not configured")
		return
	}

	keywordID := c.Param("keyword_id")
	if strings.TrimSpace(keywordID) == "" {
		writeError(c, http.StatusBadRequest, "invalid keyword_id")
		return
	}

	var req KeywordUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := s.deps.Keywords.Update(c.Request.Context(), keywordID, req, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// DeleteKeyword godoc
// @Summary Delete a keyword
// @Tags keywords
// @Security BearerAuth
// @Produce json
// @Param keyword_id path int true "keyword id"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /keywords/{keyword_id} [delete]
func (s *Server) DeleteKeyword(c *gin.Context) {
	if s.deps.Keywords == nil {
		writeError(c, http.StatusInternalServerError, "keyword service is not configured")
		return
	}

	keywordID := c.Param("keyword_id")
	if strings.TrimSpace(keywordID) == "" {
		writeError(c, http.StatusBadRequest, "invalid keyword_id")
		return
	}

	if err := s.deps.Keywords.Delete(c.Request.Context(), keywordID, s.ownerUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListAutoReplies godoc
// @Summary List auto replies
// @Tags auto-replies
// @Security BearerAuth
// @Produce json
// @Param chat_id query int false "chat id"
// @Param enabled query bool false "enabled"
// @Param limit query int false "page size"
// @Param offset query int false "page offset"
// @Success 200 {array} AutoReply
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auto-replies [get]
func (s *Server) ListAutoReplies(c *gin.Context) {
	if s.deps.AutoReplies == nil {
		writeError(c, http.StatusInternalServerError, "auto reply service is not configured")
		return
	}
	var query AutoReplyListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)
	items, err := s.deps.AutoReplies.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

// CreateAutoReply godoc
// @Summary Create an auto reply
// @Tags auto-replies
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body AutoReplyCreateRequest true "auto reply payload"
// @Success 200 {object} AutoReply
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auto-replies [post]
func (s *Server) CreateAutoReply(c *gin.Context) {
	if s.deps.AutoReplies == nil {
		writeError(c, http.StatusInternalServerError, "auto reply service is not configured")
		return
	}
	var req AutoReplyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}
	item, err := s.deps.AutoReplies.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

// UpdateAutoReply godoc
// @Summary Update an auto reply
// @Tags auto-replies
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param auto_reply_id path int true "auto reply id"
// @Param request body AutoReplyUpdateRequest true "auto reply patch"
// @Success 200 {object} AutoReply
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auto-replies/{auto_reply_id} [patch]
func (s *Server) UpdateAutoReply(c *gin.Context) {
	if s.deps.AutoReplies == nil {
		writeError(c, http.StatusInternalServerError, "auto reply service is not configured")
		return
	}
	autoReplyID := c.Param("auto_reply_id")
	if strings.TrimSpace(autoReplyID) == "" {
		writeError(c, http.StatusBadRequest, "invalid auto_reply_id")
		return
	}
	var req AutoReplyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := s.deps.AutoReplies.Update(c.Request.Context(), autoReplyID, req, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// DeleteAutoReply godoc
// @Summary Delete an auto reply
// @Tags auto-replies
// @Security BearerAuth
// @Produce json
// @Param auto_reply_id path int true "auto reply id"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /auto-replies/{auto_reply_id} [delete]
func (s *Server) DeleteAutoReply(c *gin.Context) {
	if s.deps.AutoReplies == nil {
		writeError(c, http.StatusInternalServerError, "auto reply service is not configured")
		return
	}
	autoReplyID := c.Param("auto_reply_id")
	if strings.TrimSpace(autoReplyID) == "" {
		writeError(c, http.StatusBadRequest, "invalid auto_reply_id")
		return
	}
	if err := s.deps.AutoReplies.Delete(c.Request.Context(), autoReplyID, s.ownerUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreatePost godoc
// @Summary Create a post
// @Tags posts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body PostCreateRequest true "post payload"
// @Success 200 {object} Post
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /posts [post]
func (s *Server) CreatePost(c *gin.Context) {
	if s.deps.Posts == nil {
		writeError(c, http.StatusInternalServerError, "post service is not configured")
		return
	}

	var req PostCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}

	post, err := s.deps.Posts.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, post)
}

// ListPosts godoc
// @Summary List posts
// @Tags posts
// @Security BearerAuth
// @Produce json
// @Param limit query int false "page size"
// @Param offset query int false "page offset"
// @Success 200 {array} Post
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /posts [get]
func (s *Server) ListPosts(c *gin.Context) {
	if s.deps.Posts == nil {
		writeError(c, http.StatusInternalServerError, "post service is not configured")
		return
	}

	var query CommonListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Posts.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetPost godoc
// @Summary Get post details
// @Tags posts
// @Security BearerAuth
// @Produce json
// @Param post_id path string true "post id"
// @Success 200 {object} Post
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /posts/{post_id} [get]
func (s *Server) GetPost(c *gin.Context) {
	if s.deps.Posts == nil {
		writeError(c, http.StatusInternalServerError, "post service is not configured")
		return
	}

	post, err := s.deps.Posts.Get(c.Request.Context(), c.Param("post_id"), s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, post)
}

// UpdatePost godoc
// @Summary Update a post
// @Tags posts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param post_id path string true "post id"
// @Param request body PostUpdateRequest true "post patch"
// @Success 200 {object} Post
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /posts/{post_id} [patch]
func (s *Server) UpdatePost(c *gin.Context) {
	if s.deps.Posts == nil {
		writeError(c, http.StatusInternalServerError, "post service is not configured")
		return
	}

	var req PostUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	post, err := s.deps.Posts.Update(c.Request.Context(), c.Param("post_id"), req, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, post)
}

// TogglePost godoc
// @Summary Toggle a scheduled post
// @Tags posts
// @Security BearerAuth
// @Produce json
// @Param post_id path string true "post id"
// @Success 200 {object} Post
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /posts/{post_id}/toggle [put]
func (s *Server) TogglePost(c *gin.Context) {
	if s.deps.Posts == nil {
		writeError(c, http.StatusInternalServerError, "post service is not configured")
		return
	}

	post, err := s.deps.Posts.Toggle(c.Request.Context(), c.Param("post_id"), s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, post)
}

// DeletePost godoc
// @Summary Delete a post
// @Tags posts
// @Security BearerAuth
// @Produce json
// @Param post_id path string true "post id"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /posts/{post_id} [delete]
func (s *Server) DeletePost(c *gin.Context) {
	if s.deps.Posts == nil {
		writeError(c, http.StatusInternalServerError, "post service is not configured")
		return
	}

	if err := s.deps.Posts.Delete(c.Request.Context(), c.Param("post_id"), s.ownerUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateSchedule godoc
// @Summary Create a schedule
// @Tags schedules
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body ScheduleCreateRequest true "schedule payload"
// @Success 200 {object} Schedule
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /schedules [post]
func (s *Server) CreateSchedule(c *gin.Context) {
	if s.deps.Schedules == nil {
		writeError(c, http.StatusInternalServerError, "schedule service is not configured")
		return
	}

	var req ScheduleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}

	item, err := s.deps.Schedules.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, item)
}

// ListSchedules godoc
// @Summary List schedules
// @Tags schedules
// @Security BearerAuth
// @Produce json
// @Param limit query int false "page size"
// @Param offset query int false "page offset"
// @Success 200 {array} Schedule
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /schedules [get]
func (s *Server) ListSchedules(c *gin.Context) {
	if s.deps.Schedules == nil {
		writeError(c, http.StatusInternalServerError, "schedule service is not configured")
		return
	}

	var query CommonListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Schedules.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetSchedule godoc
// @Summary Get schedule details
// @Tags schedules
// @Security BearerAuth
// @Produce json
// @Param schedule_id path string true "schedule id"
// @Success 200 {object} Schedule
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /schedules/{schedule_id} [get]
func (s *Server) GetSchedule(c *gin.Context) {
	if s.deps.Schedules == nil {
		writeError(c, http.StatusInternalServerError, "schedule service is not configured")
		return
	}

	item, err := s.deps.Schedules.Get(c.Request.Context(), c.Param("schedule_id"), s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

// UpdateSchedule godoc
// @Summary Update a schedule
// @Tags schedules
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param schedule_id path string true "schedule id"
// @Param request body ScheduleUpdateRequest true "schedule patch"
// @Success 200 {object} Schedule
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /schedules/{schedule_id} [patch]
func (s *Server) UpdateSchedule(c *gin.Context) {
	if s.deps.Schedules == nil {
		writeError(c, http.StatusInternalServerError, "schedule service is not configured")
		return
	}

	var req ScheduleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	item, err := s.deps.Schedules.Update(c.Request.Context(), c.Param("schedule_id"), req, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteSchedule godoc
// @Summary Delete a schedule
// @Tags schedules
// @Security BearerAuth
// @Produce json
// @Param schedule_id path string true "schedule id"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /schedules/{schedule_id} [delete]
func (s *Server) DeleteSchedule(c *gin.Context) {
	if s.deps.Schedules == nil {
		writeError(c, http.StatusInternalServerError, "schedule service is not configured")
		return
	}

	if err := s.deps.Schedules.Delete(c.Request.Context(), c.Param("schedule_id"), s.ownerUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// StatsOverview godoc
// @Summary Stats overview
// @Tags stats
// @Security BearerAuth
// @Produce json
// @Param chat_id query int false "chat id"
// @Param from query string false "from date YYYY-MM-DD"
// @Param to query string false "to date YYYY-MM-DD"
// @Success 200 {object} StatsOverview
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /stats/overview [get]
func (s *Server) StatsOverview(c *gin.Context) {
	if s.deps.Stats == nil {
		writeError(c, http.StatusInternalServerError, "stats service is not configured")
		return
	}

	var query StatsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	item, err := s.deps.Stats.Overview(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, item)
}

// StatsActivity godoc
// @Summary Daily activity stats
// @Tags stats
// @Security BearerAuth
// @Produce json
// @Param chat_id query int false "chat id"
// @Param from query string false "from date YYYY-MM-DD"
// @Param to query string false "to date YYYY-MM-DD"
// @Success 200 {array} ActivityStats
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /stats/activity [get]
func (s *Server) StatsActivity(c *gin.Context) {
	if s.deps.Stats == nil {
		writeError(c, http.StatusInternalServerError, "stats service is not configured")
		return
	}

	var query StatsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Stats.Activity(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

// StatsPoints godoc
// @Summary Points leaderboard
// @Tags stats
// @Security BearerAuth
// @Produce json
// @Param chat_id query int false "chat id"
// @Param from query string false "from date YYYY-MM-DD"
// @Param to query string false "to date YYYY-MM-DD"
// @Success 200 {array} PointsStats
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /stats/points [get]
func (s *Server) StatsPoints(c *gin.Context) {
	if s.deps.Stats == nil {
		writeError(c, http.StatusInternalServerError, "stats service is not configured")
		return
	}

	var query StatsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Stats.Points(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

func (s *Server) ListUsers(c *gin.Context) {
	if s.deps.Users == nil {
		writeError(c, http.StatusInternalServerError, "user service is not configured")
		return
	}

	var query UserListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	items, err := s.deps.Users.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, items)
}

func (s *Server) ExportUsers(c *gin.Context) {
	if s.deps.Admin == nil {
		writeError(c, http.StatusInternalServerError, "admin service is not configured")
		return
	}

	var query ExportUserQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if query.ChatID == 0 {
		if value := strings.TrimSpace(c.Query("chat_id")); value != "" {
			chatID, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				writeError(c, http.StatusBadRequest, "invalid chat_id")
				return
			}
			query.ChatID = chatID
		}
	}
	if query.ChatID != 0 && !s.ensureChatAllowed(c, query.ChatID) {
		return
	}
	query.OwnerUserID = s.ownerUserID(c)

	rows, err := s.deps.Admin.ExportUserRows(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("sola-users-%s.csv", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(c.Writer)
	_ = writer.Write([]string{"user_id", "username", "display_name", "chat_id", "total_points", "level", "status", "warn_count", "joined_at", "last_seen_at"})
	for _, row := range rows {
		_ = writer.Write([]string{
			strconv.FormatInt(row.UserID, 10),
			row.Username,
			row.DisplayName,
			strconv.FormatInt(row.ChatID, 10),
			strconv.FormatInt(row.TotalPoints, 10),
			row.Level,
			row.Status,
			strconv.Itoa(row.WarnCount),
			row.JoinedAt.Format(time.RFC3339),
			row.LastSeenAt.Format(time.RFC3339),
		})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
}

func (s *Server) BatchUsers(c *gin.Context) {
	if s.deps.Admin == nil {
		writeError(c, http.StatusInternalServerError, "admin service is not configured")
		return
	}

	var req BatchUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}
	req.OwnerUserID = s.ownerUserID(c)

	result, err := s.deps.Admin.BatchUserAction(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrForbidden) || errors.Is(err, gorm.ErrRecordNotFound) {
			writeServiceError(c, err)
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) ExportBackup(c *gin.Context) {
	if s.deps.Backups == nil {
		writeError(c, http.StatusInternalServerError, "backup service is not configured")
		return
	}
	if !s.isAdminAllowed(c) {
		writeError(c, http.StatusForbidden, "admin role is required")
		return
	}

	scope := strings.TrimSpace(c.DefaultQuery("scope", "business"))
	data, err := s.deps.Backups.Export(c.Request.Context(), scope)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("sola-backup-%s-%s.json", data.Scope, time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.JSON(http.StatusOK, data)
}

func (s *Server) ImportBackup(c *gin.Context) {
	if s.deps.Backups == nil {
		writeError(c, http.StatusInternalServerError, "backup service is not configured")
		return
	}
	if !s.isAdminAllowed(c) {
		writeError(c, http.StatusForbidden, "admin role is required")
		return
	}

	mode := strings.TrimSpace(c.DefaultQuery("mode", "merge"))
	var data BackupData
	file, err := c.FormFile("file")
	if err == nil {
		opened, err := file.Open()
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		defer opened.Close()
		if err := json.NewDecoder(opened).Decode(&data); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
	} else if err := c.ShouldBindJSON(&data); err != nil {
		writeError(c, http.StatusBadRequest, "backup file or json body is required")
		return
	}

	if err := s.deps.Backups.Import(c.Request.Context(), &data, mode); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, BackupImportResponse{Message: "backup imported"})
}

func (s *Server) ListTemplates(c *gin.Context) {
	if s.deps.Templates == nil {
		writeError(c, http.StatusInternalServerError, "template service is not configured")
		return
	}
	var query TemplateListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)
	items, err := s.deps.Templates.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) CreateTemplate(c *gin.Context) {
	if s.deps.Templates == nil {
		writeError(c, http.StatusInternalServerError, "template service is not configured")
		return
	}
	var req TemplateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureOptionalChatAllowed(c, req.ChatID) {
		return
	}
	item, err := s.deps.Templates.Create(c.Request.Context(), req, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) UpdateTemplate(c *gin.Context) {
	if s.deps.Templates == nil {
		writeError(c, http.StatusInternalServerError, "template service is not configured")
		return
	}
	var req TemplateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.ChatID != nil && !s.ensureOptionalChatAllowed(c, *req.ChatID) {
		return
	}
	item, err := s.deps.Templates.Update(c.Request.Context(), c.Param("template_id"), req, s.ownerUserID(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) DeleteTemplate(c *gin.Context) {
	if s.deps.Templates == nil {
		writeError(c, http.StatusInternalServerError, "template service is not configured")
		return
	}
	if err := s.deps.Templates.Delete(c.Request.Context(), c.Param("template_id"), s.ownerUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) ListInviteLinks(c *gin.Context) {
	if s.deps.InviteLinks == nil {
		writeError(c, http.StatusInternalServerError, "invite link service is not configured")
		return
	}
	var query InviteLinkListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	query.OwnerUserID = s.ownerUserID(c)
	items, err := s.deps.InviteLinks.List(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) CreateInviteLink(c *gin.Context) {
	if s.deps.InviteLinks == nil {
		writeError(c, http.StatusInternalServerError, "invite link service is not configured")
		return
	}
	var req InviteLinkCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !s.ensureChatAllowed(c, req.ChatID) {
		return
	}
	item, err := s.deps.InviteLinks.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func (s *Server) DeleteInviteLink(c *gin.Context) {
	if s.deps.InviteLinks == nil {
		writeError(c, http.StatusInternalServerError, "invite link service is not configured")
		return
	}
	if err := s.deps.InviteLinks.Delete(c.Request.Context(), c.Param("invite_link_id"), s.ownerUserID(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) isAdminAllowed(c *gin.Context) bool {
	claims, ok := CurrentAdminClaims(c)
	if !ok {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(claims.Role))
	return role == "admin" || role == "super_admin"
}

func (s *Server) ownerUserID(c *gin.Context) string {
	claims, ok := CurrentAdminClaims(c)
	if !ok || claims.IsSuperAdmin() {
		return ""
	}
	return claims.UserID
}

func (s *Server) ensureChatAllowed(c *gin.Context, chatID int64) bool {
	claims, ok := CurrentAdminClaims(c)
	if !ok || claims.IsSuperAdmin() || s.deps.Chats == nil || chatID == 0 {
		return true
	}
	owned, err := s.deps.Chats.UserOwnsChat(c.Request.Context(), claims.UserID, strconv.FormatInt(chatID, 10))
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return false
	}
	if !owned {
		writeError(c, http.StatusForbidden, "无权访问该群组")
		return false
	}
	return true
}

func (s *Server) ensureOptionalChatAllowed(c *gin.Context, chatID *int64) bool {
	if chatID == nil {
		return true
	}
	return s.ensureChatAllowed(c, *chatID)
}

func (s *Server) issueToken(admin AdminIdentity) (string, time.Time, error) {
	if strings.TrimSpace(s.deps.JWT.SigningKey) == "" {
		return "", time.Time{}, errors.New("jwt signing key is not configured")
	}

	expiresAt := time.Now().UTC().Add(s.deps.JWT.TTL)
	if s.deps.JWT.TTL <= 0 {
		expiresAt = time.Now().UTC().Add(24 * time.Hour)
	}

	claims := AdminClaims{
		AdminID:        admin.ID,
		UserID:         firstNonEmpty(admin.UserID, admin.ID),
		TelegramUserID: admin.TelegramUserID,
		Username:       admin.Username,
		Role:           admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.deps.JWT.Issuer,
			Subject:   firstNonEmpty(admin.UserID, admin.ID),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.deps.JWT.SigningKey))
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func writeError(c *gin.Context, status int, msg string) {
	if status >= http.StatusInternalServerError {
		msg = "internal server error"
	}
	c.JSON(status, ErrorResponse{Error: msg})
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(c, http.StatusForbidden, "无权访问该资源")
	case errors.Is(err, gorm.ErrRecordNotFound):
		writeError(c, http.StatusNotFound, "resource not found")
	default:
		writeError(c, http.StatusInternalServerError, err.Error())
	}
}

func parseChatID(c *gin.Context) (int64, error) {
	if value := c.Param("chat_id"); value != "" {
		return strconv.ParseInt(value, 10, 64)
	}
	return strconv.ParseInt(c.Param("chatID"), 10, 64)
}

func parseInt64Path(c *gin.Context, key string) (int64, error) {
	return strconv.ParseInt(c.Param(key), 10, 64)
}

func parseChatAndUserID(c *gin.Context) (int64, int64, bool) {
	chatID, err := parseChatID(c)
	if err != nil {
		return 0, 0, false
	}
	userID, err := parseInt64Path(c, "userID")
	if err != nil {
		return 0, 0, false
	}
	return chatID, userID, true
}
