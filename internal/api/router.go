package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title DaBoWin Sola Admin API
// @version 0.1.0
// @description Gin backend skeleton for Telegram bot operations, scheduling, and analytics.
// @BasePath /api/v1
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func NewRouter(deps Dependencies) *gin.Engine {
	r := gin.New()
	r.Use(RecoveryJSON(), CORSMiddleware(deps.AllowedOrigins()))

	server := NewServer(deps)

	r.GET("/healthz", server.Health)

	// Public Turnstile verification endpoints — no JWT required.
	r.POST("/api/verify/turnstile", server.VerifyTurnstile)
	r.GET("/api/verify/turnstile/config", server.TurnstileConfig)

	if deps.EnableSwagger {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	r.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "resource not found")
	})
	r.NoMethod(func(c *gin.Context) {
		writeError(c, http.StatusMethodNotAllowed, "method not allowed")
	})

	compat := r.Group("/api")
	{
		compat.POST("/auth/login", server.Login)
		compat.POST("/auth/telegram", server.TelegramLogin)
		compatSecured := compat.Group("")
		compatSecured.Use(JWTMiddleware(deps.JWT), server.RequireChatOwnership())
		{
			compatSecured.GET("/auth/me", server.Me)
			compatSecured.GET("/dashboard/summary", server.DashboardSummary)
			compatSecured.GET("/bots", server.ListBots)
			compatSecured.GET("/users", server.ListUsers)
			compatSecured.GET("/users/export", server.ExportUsers)
			compatSecured.POST("/users/batch", server.BatchUsers)
			compatSecured.GET("/chats", server.ListChats)
			compatSecured.GET("/chats/:chat_id/points-config", server.GetChatPointConfig)
			compatSecured.PUT("/chats/:chat_id/points-config", server.UpdateChatPointConfig)
			compatSecured.GET("/points/config/:chatID", server.GetPointConfig)
			compatSecured.PUT("/points/config/:chatID", server.UpdatePointConfig)
			compatSecured.GET("/points/rank/:chatID", server.GetPointRank)
			compatSecured.GET("/points/user/:chatID/:userID", server.GetPointUser)
			compatSecured.PUT("/points/user/:chatID/:userID", server.AdjustPointUser)
			compatSecured.GET("/points/logs/:chatID/:userID", server.ListPointLogs)
			compatSecured.GET("/admin/config/:chatID", server.GetChatAdminConfig)
			compatSecured.PUT("/admin/config/:chatID", server.UpdateChatAdminConfig)
			compatSecured.GET("/admin/bans/:chatID", server.ListBanLogs)
			compatSecured.POST("/admin/ban", server.AdminBan)
			compatSecured.POST("/admin/mute", server.AdminMute)
			compatSecured.POST("/admin/unmute", server.AdminUnmute)
			compatSecured.DELETE("/admin/ban/:chatID/:userID", server.AdminUnban)
			compatSecured.GET("/admin/warns/:chatID/:userID", server.ListWarnRecords)
			compatSecured.GET("/posts", server.ListPosts)
			compatSecured.POST("/posts", server.CreatePost)
			compatSecured.PUT("/posts/:post_id", server.UpdatePost)
			compatSecured.DELETE("/posts/:post_id", server.DeletePost)
			compatSecured.PUT("/posts/:post_id/toggle", server.TogglePost)
			compatSecured.GET("/lottery", server.ListLotteries)
			compatSecured.POST("/lottery", server.CreateLottery)
			compatSecured.DELETE("/lottery/:id", server.CancelLottery)
			compatSecured.GET("/lottery/:id/entries", server.ListLotteryEntries)
			compatSecured.GET("/lottery/:id/winners", server.ListLotteryWinners)
			compatSecured.GET("/levels", server.ListLevels)
			compatSecured.POST("/levels", server.CreateLevel)
			compatSecured.PUT("/levels/:level_id", server.UpdateLevel)
			compatSecured.PATCH("/levels/:level_id", server.UpdateLevel)
			compatSecured.DELETE("/levels/:level_id", server.DeleteLevel)
			compatSecured.GET("/audit-logs", server.ListAuditLogs)
			compatSecured.GET("/admin/violations", server.ListAdminViolations)
			compatSecured.PATCH("/admin/violations/:violation_id", server.UpdateAdminViolation)
			compatSecured.GET("/keywords", server.ListKeywords)
			compatSecured.POST("/keywords", server.CreateKeyword)
			compatSecured.PUT("/keywords/:keyword_id", server.UpdateKeyword)
			compatSecured.PATCH("/keywords/:keyword_id", server.UpdateKeyword)
			compatSecured.DELETE("/keywords/:keyword_id", server.DeleteKeyword)
			compatSecured.GET("/auto-replies", server.ListAutoReplies)
			compatSecured.POST("/auto-replies", server.CreateAutoReply)
			compatSecured.PUT("/auto-replies/:auto_reply_id", server.UpdateAutoReply)
			compatSecured.PATCH("/auto-replies/:auto_reply_id", server.UpdateAutoReply)
			compatSecured.DELETE("/auto-replies/:auto_reply_id", server.DeleteAutoReply)
			compatSecured.GET("/backup/export", server.ExportBackup)
			compatSecured.POST("/backup/import", server.ImportBackup)
			compatSecured.GET("/templates", server.ListTemplates)
			compatSecured.POST("/templates", server.CreateTemplate)
			compatSecured.PUT("/templates/:template_id", server.UpdateTemplate)
			compatSecured.DELETE("/templates/:template_id", server.DeleteTemplate)
			compatSecured.GET("/invite-links", server.ListInviteLinks)
			compatSecured.POST("/invite-links", server.CreateInviteLink)
			compatSecured.DELETE("/invite-links/:invite_link_id", server.DeleteInviteLink)
			compatSecured.GET("/stats/overview", server.StatsOverview)
			compatSecured.GET("/stats/activity", server.StatsActivity)
			compatSecured.GET("/stats/points", server.StatsPoints)
		}
	}

	api := r.Group("/api/v1")
	{
		api.GET("/health", server.Health)

		admin := api.Group("/admin")
		{
			admin.POST("/login", server.Login)
		}

		auth := api.Group("/auth")
		{
			auth.POST("/login", server.Login)
			auth.POST("/telegram", server.TelegramLogin)
		}

		secured := api.Group("")
		secured.Use(JWTMiddleware(deps.JWT), server.RequireChatOwnership())
		{
			secured.GET("/auth/me", server.Me)
			secured.GET("/dashboard/summary", server.DashboardSummary)
			secured.GET("/bots", server.ListBots)
			secured.GET("/users", server.ListUsers)
			secured.GET("/users/export", server.ExportUsers)
			secured.POST("/users/batch", server.BatchUsers)
			secured.GET("/bot/config", server.GetBotConfig)
			secured.PUT("/bot/config", server.UpdateBotConfig)

			chats := secured.Group("/chats")
			{
				chats.GET("", server.ListChats)
				chats.POST("/bind", server.BindChat)
				chats.DELETE("/:chat_id/bind", server.UnbindChat)
				chats.GET("/:chat_id/points-config", server.GetChatPointConfig)
				chats.PUT("/:chat_id/points-config", server.UpdateChatPointConfig)
			}

			adminPanel := secured.Group("/admin")
			{
				adminPanel.GET("/config/:chatID", server.GetChatAdminConfig)
				adminPanel.PUT("/config/:chatID", server.UpdateChatAdminConfig)
				adminPanel.GET("/bans/:chatID", server.ListBanLogs)
				adminPanel.POST("/ban", server.AdminBan)
				adminPanel.POST("/mute", server.AdminMute)
				adminPanel.POST("/unmute", server.AdminUnmute)
				adminPanel.DELETE("/ban/:chatID/:userID", server.AdminUnban)
				adminPanel.GET("/warns/:chatID/:userID", server.ListWarnRecords)
				adminPanel.GET("/violations", server.ListAdminViolations)
				adminPanel.PATCH("/violations/:violation_id", server.UpdateAdminViolation)
			}

			points := secured.Group("/points")
			{
				points.GET("/config/:chatID", server.GetPointConfig)
				points.PUT("/config/:chatID", server.UpdatePointConfig)
				points.GET("/rank/:chatID", server.GetPointRank)
				points.GET("/user/:chatID/:userID", server.GetPointUser)
				points.PUT("/user/:chatID/:userID", server.AdjustPointUser)
				points.GET("/logs/:chatID/:userID", server.ListPointLogs)
			}

			posts := secured.Group("/posts")
			{
				posts.POST("", server.CreatePost)
				posts.GET("", server.ListPosts)
				posts.GET("/:post_id", server.GetPost)
				posts.PUT("/:post_id", server.UpdatePost)
				posts.PATCH("/:post_id", server.UpdatePost)
				posts.DELETE("/:post_id", server.DeletePost)
				posts.PUT("/:post_id/toggle", server.TogglePost)
			}

			schedules := secured.Group("/schedules")
			{
				schedules.POST("", server.CreateSchedule)
				schedules.GET("", server.ListSchedules)
				schedules.GET("/:schedule_id", server.GetSchedule)
				schedules.PATCH("/:schedule_id", server.UpdateSchedule)
				schedules.DELETE("/:schedule_id", server.DeleteSchedule)
			}

			lottery := secured.Group("/lottery")
			{
				lottery.GET("", server.ListLotteries)
				lottery.POST("", server.CreateLottery)
				lottery.DELETE("/:id", server.CancelLottery)
				lottery.GET("/:id/entries", server.ListLotteryEntries)
				lottery.GET("/:id/winners", server.ListLotteryWinners)
			}

			levels := secured.Group("/levels")
			{
				levels.GET("", server.ListLevels)
				levels.POST("", server.CreateLevel)
				levels.PUT("/:level_id", server.UpdateLevel)
				levels.PATCH("/:level_id", server.UpdateLevel)
				levels.DELETE("/:level_id", server.DeleteLevel)
			}

			keywords := secured.Group("/keywords")
			{
				keywords.GET("", server.ListKeywords)
				keywords.POST("", server.CreateKeyword)
				keywords.PUT("/:keyword_id", server.UpdateKeyword)
				keywords.PATCH("/:keyword_id", server.UpdateKeyword)
				keywords.DELETE("/:keyword_id", server.DeleteKeyword)
			}

			autoReplies := secured.Group("/auto-replies")
			{
				autoReplies.GET("", server.ListAutoReplies)
				autoReplies.POST("", server.CreateAutoReply)
				autoReplies.PUT("/:auto_reply_id", server.UpdateAutoReply)
				autoReplies.PATCH("/:auto_reply_id", server.UpdateAutoReply)
				autoReplies.DELETE("/:auto_reply_id", server.DeleteAutoReply)
			}

			backup := secured.Group("/backup")
			{
				backup.GET("/export", server.ExportBackup)
				backup.POST("/import", server.ImportBackup)
			}

			templates := secured.Group("/templates")
			{
				templates.GET("", server.ListTemplates)
				templates.POST("", server.CreateTemplate)
				templates.PUT("/:template_id", server.UpdateTemplate)
				templates.PATCH("/:template_id", server.UpdateTemplate)
				templates.DELETE("/:template_id", server.DeleteTemplate)
			}

			inviteLinks := secured.Group("/invite-links")
			{
				inviteLinks.GET("", server.ListInviteLinks)
				inviteLinks.POST("", server.CreateInviteLink)
				inviteLinks.DELETE("/:invite_link_id", server.DeleteInviteLink)
			}

			auditLogs := secured.Group("/audit-logs")
			{
				auditLogs.GET("", server.ListAuditLogs)
			}

			stats := secured.Group("/stats")
			{
				stats.GET("/overview", server.StatsOverview)
				stats.GET("/activity", server.StatsActivity)
				stats.GET("/points", server.StatsPoints)
			}
		}
	}

	return r
}

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
			}
		}
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			if origin != "" {
				if _, ok := allowed[origin]; !ok {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func RecoveryJSON() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		_ = normalizeRecoveredError(recovered)
		writeError(c, http.StatusInternalServerError, "internal server error")
	})
}

func normalizeRecoveredError(recovered any) error {
	switch v := recovered.(type) {
	case error:
		return v
	case string:
		return errors.New(v)
	default:
		return errors.New("panic")
	}
}
