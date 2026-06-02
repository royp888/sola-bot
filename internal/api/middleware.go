package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(cfg JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(cfg.SigningKey) == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "jwt signing key is not configured"})
			return
		}

		tokenString := bearerToken(c.GetHeader("Authorization"))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "missing bearer token"})
			return
		}

		claims := &AdminClaims{}
		parseOpts := []jwt.ParserOption{jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})}
		if cfg.Issuer != "" {
			parseOpts = append(parseOpts, jwt.WithIssuer(cfg.Issuer))
		}

		_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(cfg.SigningKey), nil
		}, parseOpts...)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid or expired token"})
			return
		}

		c.Set(ContextKeyAuthClaims, claims)
		c.Set("user_id", claims.UserID)
		c.Set("telegram_user_id", claims.TelegramUserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func (s *Server) RequireChatOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := CurrentAdminClaims(c)
		if !ok || claims.IsSuperAdmin() || s.deps.Chats == nil {
			c.Next()
			return
		}
		chatID := firstNonEmpty(c.Param("chatID"), c.Param("chat_id"))
		if chatID == "" {
			c.Next()
			return
		}
		owned, err := s.deps.Chats.UserOwnsChat(c.Request.Context(), claims.UserID, chatID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
			return
		}
		if !owned {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "无权访问该群组"})
			return
		}
		c.Next()
	}
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}

	if len(header) < 7 {
		return ""
	}

	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}

	return strings.TrimSpace(header[7:])
}

func CurrentAdminClaims(c *gin.Context) (*AdminClaims, bool) {
	value, ok := c.Get(ContextKeyAuthClaims)
	if !ok {
		return nil, false
	}

	claims, ok := value.(*AdminClaims)
	return claims, ok
}

func (c *AdminClaims) IsSuperAdmin() bool {
	if c == nil {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(c.Role))
	return role == "admin" || role == "super_admin" || role == "owner_admin"
}
