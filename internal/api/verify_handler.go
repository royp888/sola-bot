package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var cfHTTPClient = &http.Client{Timeout: 10 * time.Second}

// VerifyTurnstile handles POST /api/verify/turnstile.
// Called by the Mini App after the user passes the CF Turnstile challenge.
func (s *Server) VerifyTurnstile(c *gin.Context) {
	var req TurnstileVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ChatID == 0 || req.UserID == 0 || req.Sig == "" || req.CFToken == "" {
		writeError(c, http.StatusBadRequest, "missing required fields")
		return
	}

	if s.deps.TurnstileVerifySecret == "" {
		writeError(c, http.StatusServiceUnavailable, "turnstile not configured")
		return
	}

	if time.Now().Unix() > req.Exp {
		_ = telegramJoinAction(s.deps.BotToken, req.ChatID, req.UserID, false)
		writeError(c, http.StatusUnauthorized, "link expired")
		return
	}

	expected := turnstileHMAC(s.deps.TurnstileVerifySecret, req.ChatID, req.UserID, req.Exp)
	if !hmac.Equal([]byte(expected), []byte(req.Sig)) {
		_ = telegramJoinAction(s.deps.BotToken, req.ChatID, req.UserID, false)
		writeError(c, http.StatusUnauthorized, "invalid signature")
		return
	}

	ok, err := verifyCFToken(s.deps.TurnstileSecretKey, req.CFToken, c.ClientIP())
	if err != nil || !ok {
		_ = telegramJoinAction(s.deps.BotToken, req.ChatID, req.UserID, false)
		writeError(c, http.StatusUnauthorized, "turnstile verification failed")
		return
	}

	if err := telegramJoinAction(s.deps.BotToken, req.ChatID, req.UserID, true); err != nil {
		writeError(c, http.StatusInternalServerError, "failed to approve join request")
		return
	}

	c.JSON(http.StatusOK, TurnstileVerifyResponse{OK: true, Message: "已批准入群申请"})
}

// TurnstileConfig handles GET /api/verify/turnstile/config.
// Returns the site key so the Mini App can render the widget without hardcoding.
func (s *Server) TurnstileConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"site_key": s.deps.TurnstileSiteKey})
}

// turnstileHMAC signs "chatID|userID|exp" with the verify secret using HMAC-SHA256.
func turnstileHMAC(secret string, chatID, userID, exp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d|%d|%d", chatID, userID, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

type cfSiteverifyResponse struct {
	Success bool `json:"success"`
}

// verifyCFToken calls the Cloudflare Turnstile siteverify endpoint.
func verifyCFToken(secretKey, token, remoteIP string) (bool, error) {
	vals := url.Values{
		"secret":   {secretKey},
		"response": {token},
	}
	if remoteIP != "" {
		vals.Set("remoteip", remoteIP)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(vals.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := cfHTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result cfSiteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Success, nil
}

// telegramJoinAction approves or declines a chat join request via the Bot API.
func telegramJoinAction(botToken string, chatID, userID int64, approve bool) error {
	if botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	method := "declineChatJoinRequest"
	if approve {
		method = "approveChatJoinRequest"
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", botToken, method)
	body := fmt.Sprintf(`{"chat_id":%d,"user_id":%d}`, chatID, userID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cfHTTPClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
