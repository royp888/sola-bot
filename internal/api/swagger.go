package api

import (
	"encoding/json"
	"sync"

	"github.com/swaggo/swag"
)

type SwaggerMeta struct {
	Title       string
	Version     string
	Description string
	Host        string
	BasePath    string
	Schemes     []string
}

var (
	swaggerMeta = SwaggerMeta{
		Title:       "DaBoWin Sola Admin API",
		Version:     "0.1.0",
		Description: "Telegram bot operations backend skeleton.",
		BasePath:    "/api/v1",
		Schemes:     []string{"http", "https"},
	}
	swaggerMu sync.RWMutex
)

func init() {
	swag.Register(swag.Name, swaggerDoc{})
}

func ConfigureSwagger(meta SwaggerMeta) {
	swaggerMu.Lock()
	defer swaggerMu.Unlock()

	if meta.Title != "" {
		swaggerMeta.Title = meta.Title
	}
	if meta.Version != "" {
		swaggerMeta.Version = meta.Version
	}
	if meta.Description != "" {
		swaggerMeta.Description = meta.Description
	}
	if meta.Host != "" {
		swaggerMeta.Host = meta.Host
	}
	if meta.BasePath != "" {
		swaggerMeta.BasePath = meta.BasePath
	}
	if len(meta.Schemes) > 0 {
		swaggerMeta.Schemes = append([]string(nil), meta.Schemes...)
	}
}

type swaggerDoc struct{}

func (swaggerDoc) ReadDoc() string {
	swaggerMu.RLock()
	meta := swaggerMeta
	swaggerMu.RUnlock()

	doc := map[string]any{
		"swagger": "2.0",
		"info": map[string]any{
			"title":       meta.Title,
			"version":     meta.Version,
			"description": meta.Description,
		},
		"host":     meta.Host,
		"basePath": meta.BasePath,
		"schemes":  meta.Schemes,
		"securityDefinitions": map[string]any{
			"BearerAuth": map[string]any{
				"type":        "apiKey",
				"name":        "Authorization",
				"in":          "header",
				"description": "Bearer access token",
			},
		},
		"paths": map[string]any{
			"/health": map[string]any{
				"get": operation("Health", "system", false, "Health probe", false),
			},
			"/admin/login": map[string]any{
				"post": operation("Admin login", "admin", false, "Login with admin credentials", false),
			},
			"/bot/config": map[string]any{
				"get": operation("Get bot config", "bot", true, "Fetch bot configuration", false),
				"put": operation("Update bot config", "bot", true, "Patch bot configuration", false),
			},
			"/chats": map[string]any{
				"get": operation("List bound chats", "chats", true, "List bound chats", true),
			},
			"/chats/bind": map[string]any{
				"post": operation("Bind a chat", "chats", true, "Bind a chat or channel", false),
			},
			"/chats/{chat_id}/bind": map[string]any{
				"delete": operation("Unbind a chat", "chats", true, "Unbind a chat or channel", false),
			},
			"/posts": map[string]any{
				"get":  operation("List posts", "posts", true, "List posts", true),
				"post": operation("Create post", "posts", true, "Create a post", false),
			},
			"/posts/{post_id}": map[string]any{
				"get":    operation("Get post", "posts", true, "Get post details", false),
				"patch":  operation("Update post", "posts", true, "Update a post", false),
				"delete": operation("Delete post", "posts", true, "Delete a post", false),
			},
			"/schedules": map[string]any{
				"get":  operation("List schedules", "schedules", true, "List schedules", true),
				"post": operation("Create schedule", "schedules", true, "Create a schedule", false),
			},
			"/schedules/{schedule_id}": map[string]any{
				"get":    operation("Get schedule", "schedules", true, "Get schedule details", false),
				"patch":  operation("Update schedule", "schedules", true, "Update a schedule", false),
				"delete": operation("Delete schedule", "schedules", true, "Delete a schedule", false),
			},
			"/stats/overview": map[string]any{
				"get": operation("Stats overview", "stats", true, "Aggregated statistics", false),
			},
			"/stats/activity": map[string]any{
				"get": operation("Activity stats", "stats", true, "Daily activity metrics", true),
			},
			"/stats/points": map[string]any{
				"get": operation("Points leaderboard", "stats", true, "Points ranking", true),
			},
		},
	}

	raw, _ := json.Marshal(doc)
	return string(raw)
}

func operation(summary, tag string, secured bool, description string, arrayResult bool) map[string]any {
	op := map[string]any{
		"summary":     summary,
		"description": description,
		"tags":        []string{tag},
		"produces":    []string{"application/json"},
		"responses": map[string]any{
			"200": map[string]any{
				"description": "OK",
			},
			"400": map[string]any{
				"description": "Bad Request",
			},
			"401": map[string]any{
				"description": "Unauthorized",
			},
			"500": map[string]any{
				"description": "Internal Server Error",
			},
		},
	}

	if secured {
		op["security"] = []map[string][]string{
			{"BearerAuth": []string{}},
		}
	}

	if arrayResult {
		op["responses"].(map[string]any)["200"] = map[string]any{
			"description": "OK",
			"schema": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "object"},
			},
		}
		return op
	}

	op["responses"].(map[string]any)["200"] = map[string]any{
		"description": "OK",
		"schema": map[string]any{
			"type": "object",
		},
	}

	return op
}
