package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dabowin/sola/internal/config"
)

// AiFilterService calls an LLM API to judge whether a message is spam/advertisement.
type AiFilterService struct {
	cfg    config.AiFilterConfig
	client *http.Client
}

// NewAiFilterService creates a new AI filter service. Returns nil when disabled.
func NewAiFilterService(cfg config.AiFilterConfig) *AiFilterService {
	if !cfg.Enabled || cfg.APIKey == "" {
		return nil
	}
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		endpoint = "https://api.deepseek.com/v1"
	}
	return &AiFilterService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// chatMessage is a single message in the OpenAI chat format.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// IsSpam sends the message text to the AI and returns true if it is classified as spam/ad.
// Returns (isSpam, reason, error). On API errors, isSpam is false (fail-open).
func (s *AiFilterService) IsSpam(ctx context.Context, text string, userName string) (bool, string, error) {
	if s == nil {
		return false, "", nil
	}

	prompt := fmt.Sprintf(
		`你是一个 Telegram 群组的内容审核助手。请判断以下消息是否是广告、垃圾信息或违规内容。

判断标准：
- 包含推广链接、邀请码、拉人进群的是广告
- 加密货币/博彩/色情相关内容是违规
- 无意义的刷屏、符号灌水是垃圾信息
- 正常的聊天、提问、回答不是广告

请只回复一个 JSON：{"spam": true/false, "reason": "简短中文理由"}

用户 %s 发送的消息：
%s`, userName, text,
	)

	reqBody := chatRequest{
		Model: s.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: "You are a content moderation assistant. Always respond with valid JSON."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   100,
		Temperature: 0.1,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false, "", err
	}

	url := strings.TrimRight(s.cfg.Endpoint, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("ai filter request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return false, "", fmt.Errorf("ai filter read: %w", err)
	}

	var result chatResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return false, "", fmt.Errorf("ai filter parse: %w", err)
	}
	if result.Error != nil {
		return false, "", fmt.Errorf("ai filter api error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return false, "", nil
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)

	// Try to parse JSON response
	var aiResult struct {
		Spam   bool   `json:"spam"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &aiResult); err != nil {
		// Fallback: check if the response contains "spam" or "true"
		lower := strings.ToLower(content)
		if strings.Contains(lower, `"spam": true`) || strings.Contains(lower, `"spam":true`) {
			return true, content, nil
		}
		return false, content, nil
	}

	return aiResult.Spam, aiResult.Reason, nil
}
