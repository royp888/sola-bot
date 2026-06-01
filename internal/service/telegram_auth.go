package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TelegramLoginData struct {
	ID        int64
	Username  string
	FirstName string
	LastName  string
	PhotoURL  string
	AuthDate  int64
	Hash      string
}

func VerifyTelegramLogin(data map[string]string, botToken string) (*TelegramLoginData, error) {
	botToken = strings.TrimSpace(botToken)
	if botToken == "" {
		return nil, errors.New("bot token is not configured")
	}
	receivedHash := strings.TrimSpace(data["hash"])
	if receivedHash == "" {
		return nil, errors.New("missing hash")
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		if key != "hash" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(data[key])
	}

	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	_, _ = mac.Write([]byte(builder.String()))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(receivedHash)) {
		return nil, errors.New("invalid login hash")
	}

	authDate, err := strconv.ParseInt(data["auth_date"], 10, 64)
	if err != nil || authDate <= 0 {
		return nil, errors.New("invalid auth_date")
	}
	if time.Since(time.Unix(authDate, 0)) > 24*time.Hour {
		return nil, errors.New("login data expired")
	}

	id, err := strconv.ParseInt(data["id"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}
	return &TelegramLoginData{
		ID:        id,
		Username:  data["username"],
		FirstName: data["first_name"],
		LastName:  data["last_name"],
		PhotoURL:  data["photo_url"],
		AuthDate:  authDate,
		Hash:      receivedHash,
	}, nil
}
