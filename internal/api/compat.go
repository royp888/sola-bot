package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) Me(c *gin.Context) {
	claims, ok := CurrentAdminClaims(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "missing admin claims")
		return
	}
	c.JSON(http.StatusOK, AdminIdentity{
		ID:             firstNonEmpty(claims.UserID, claims.AdminID),
		UserID:         claims.UserID,
		TelegramUserID: claims.TelegramUserID,
		Username:       claims.Username,
		Email:          claims.Username,
		Role:           claims.Role,
		Name:           claims.Username,
		DisplayName:    claims.Username,
		Language:       "zh-CN",
	})
}

func (s *Server) ListBots(c *gin.Context) {
	c.JSON(http.StatusOK, []gin.H{
		{
			"id":            "primary",
			"name":          "Lumanman Bot",
			"username":      "@lumanmanbot",
			"status":        "online",
			"boundChats":    0,
			"lastHeartbeat": time.Now().UTC().Format(time.RFC3339),
			"language":      "zh-CN",
		},
	})
}

func (s *Server) DashboardSummary(c *gin.Context) {
	var overview *StatsOverview
	if s.deps.Stats != nil {
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
		overview = item
	}
	if overview == nil {
		overview = &StatsOverview{}
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": []gin.H{
			{"label": "绑定频道/群", "value": fmt.Sprint(overview.TotalChats), "delta": "实时", "tone": "primary"},
			{"label": "定时任务", "value": fmt.Sprint(overview.TotalSchedules), "delta": "调度器扫描", "tone": "warning"},
			{"label": "活跃用户", "value": fmt.Sprint(overview.ActiveUsers), "delta": "近 7 天", "tone": "success"},
			{"label": "积分发放", "value": fmt.Sprint(overview.PointsIssued), "delta": "近 7 天", "tone": "primary"},
		},
		"activity": []gin.H{
			{"title": "后台已启动", "detail": "真实 API、Bot polling 和 worker 可独立运行", "time": time.Now().Format("15:04"), "status": "success"},
		},
		"jobs": []gin.H{
			{"title": "定时发布调度器", "schedule": "每分钟扫描", "nextRun": "运行中", "status": "live"},
		},
		"health": []gin.H{
			{"label": "API", "value": 100, "note": "OK"},
			{"label": "Bot", "value": 100, "note": "polling"},
			{"label": "Worker", "value": 100, "note": "已配置"},
		},
	})
}
