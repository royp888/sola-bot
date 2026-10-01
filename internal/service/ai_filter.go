package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/config"
	"github.com/dabowin/sola/internal/store"
)

// AI setting keys stored in the system_settings table (editable from the web admin).
const (
	aiSettingEnabled      = "ai_filter.enabled"
	aiSettingProvider     = "ai_filter.provider"
	aiSettingAPIKey       = "ai_filter.api_key"
	aiSettingModel        = "ai_filter.model"
	aiSettingEndpoint     = "ai_filter.endpoint"
	aiSettingSystemPrompt = "ai_filter.system_prompt"
	aiSettingTemperature  = "ai_filter.temperature"
)

// AiFilterService calls an LLM API to judge whether a message is spam/advertisement
// and to answer user questions. Its configuration is resolved at call time: values
// stored in the database (via the web admin) take precedence over config.yaml.
type AiFilterService struct {
	fallback config.AiFilterConfig
	store    *store.Store
	client   *http.Client
}

// NewAiFilterService creates a new AI filter service. The returned service resolves
// its live configuration at call time from the DB (when a store is provided),
// falling back to cfg. It is never nil.
func NewAiFilterService(st *store.Store, cfg config.AiFilterConfig) *AiFilterService {
	return &AiFilterService{
		fallback: cfg,
		store:    st,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// resolveConfig returns the effective AI config, preferring DB overrides.
func (s *AiFilterService) resolveConfig(ctx context.Context) config.AiFilterConfig {
	cfg := s.fallback
	if s.store == nil {
		return normalizeAiConfig(cfg)
	}
	if v, ok, _ := s.store.GetSystemSetting(ctx, aiSettingEnabled); ok {
		cfg.Enabled = strings.EqualFold(strings.TrimSpace(v), "true")
	}
	if v, ok, _ := s.store.GetSystemSetting(ctx, aiSettingProvider); ok && v != "" {
		cfg.Provider = v
	}
	if v, ok, _ := s.store.GetSystemSetting(ctx, aiSettingAPIKey); ok && v != "" {
		cfg.APIKey = v
	}
	if v, ok, _ := s.store.GetSystemSetting(ctx, aiSettingModel); ok && v != "" {
		cfg.Model = v
	}
	if v, ok, _ := s.store.GetSystemSetting(ctx, aiSettingEndpoint); ok && v != "" {
		cfg.Endpoint = v
	}
	if v, ok, _ := s.store.GetSystemSetting(ctx, aiSettingSystemPrompt); ok {
		cfg.SystemPrompt = v
	}
	if v, ok, _ := s.store.GetSystemSetting(ctx, aiSettingTemperature); ok && v != "" {
		if t, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Temperature = t
		}
	}
	return normalizeAiConfig(cfg)
}

// normalizeAiConfig applies provider defaults for endpoint/model.
func normalizeAiConfig(cfg config.AiFilterConfig) config.AiFilterConfig {
	cfg.Endpoint = strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.deepseek.com/v1"
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "deepseek-chat"
	}
	return cfg
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
// Returns (isSpam, reason, error). On API errors or when disabled, isSpam is false (fail-open).
func (s *AiFilterService) IsSpam(ctx context.Context, text string, userName string) (bool, string, error) {
	if s == nil {
		return false, "", nil
	}
	cfg := s.resolveConfig(ctx)
	if !cfg.Enabled || cfg.APIKey == "" {
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

	content, err := s.complete(ctx, cfg, []chatMessage{
		{Role: "system", Content: "You are a content moderation assistant. Always respond with valid JSON."},
		{Role: "user", Content: prompt},
	}, 100, 0.1)
	if err != nil {
		return false, "", err
	}
	if content == "" {
		return false, "", nil
	}

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

// complete sends a chat completion request to the LLM and returns the assistant reply.
func (s *AiFilterService) complete(ctx context.Context, cfg config.AiFilterConfig, messages []chatMessage, maxTokens int, temperature float64) (string, error) {
	reqBody := chatRequest{
		Model:       cfg.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := strings.TrimRight(cfg.Endpoint, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", fmt.Errorf("ai read: %w", err)
	}

	var result chatResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("ai parse: %w", err)
	}
	if result.Error != nil {
		return "", fmt.Errorf("ai api error: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("ai: empty response")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// chatSystemPrompt is the default system prompt for the /ai chat feature.
const chatSystemPrompt = "你是一个友好、有帮助的 Telegram 群组助手。请用简体中文回答用户的问题，回答要简洁、准确、条理清晰。"

// Chat sends a user prompt to the LLM and returns the assistant's reply.
// It returns bot.ErrAINotConfigured when the AI service is not configured.
func (s *AiFilterService) Chat(ctx context.Context, userPrompt string) (string, error) {
	if s == nil {
		return "", bot.ErrAINotConfigured
	}
	cfg := s.resolveConfig(ctx)
	if !cfg.Enabled || cfg.APIKey == "" {
		return "", bot.ErrAINotConfigured
	}

	systemPrompt := strings.TrimSpace(cfg.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = chatSystemPrompt
	}
	temperature := cfg.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}

	return s.complete(ctx, cfg, []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, 1024, temperature)
}
