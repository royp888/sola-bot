package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	loginRateLimitWindow = 15 * time.Minute
	loginRateLimitMax    = 5
)

func VerifyConfiguredPassword(password string, plain string, hash string) bool {
	hash = strings.TrimSpace(hash)
	if hash != "" {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return false
	}
	return password == plain
}

func allowLoginAttempt(ctx context.Context, limiter LoginRateLimiter, key string) (bool, time.Duration, error) {
	if limiter == nil {
		return true, 0, nil
	}
	count, err := limiter.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if count == 1 {
		if err := limiter.Expire(ctx, key, loginRateLimitWindow).Err(); err != nil {
			return false, 0, err
		}
	}
	if count > loginRateLimitMax {
		ttl, ttlErr := limiter.TTL(ctx, key).Result()
		if ttlErr != nil {
			return false, 0, ttlErr
		}
		if ttl < 0 {
			ttl = loginRateLimitWindow
		}
		return false, ttl, nil
	}
	return true, 0, nil
}

func clearLoginAttempts(ctx context.Context, limiter LoginRateLimiter, key string) error {
	if limiter == nil {
		return nil
	}
	return limiter.Del(ctx, key).Err()
}

func loginRateLimitKey(username string, ip string) string {
	if username == "" {
		username = "anonymous"
	}
	if ip == "" {
		ip = "unknown"
	}
	return fmt.Sprintf("auth:login:%s:%s", username, ip)
}

func loginAttemptIdentity(username string, forwardedFor string, remoteAddr string) (string, string) {
	username = strings.ToLower(strings.TrimSpace(username))
	ip := strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	if ip == "" {
		host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
		if err == nil {
			ip = host
		} else {
			ip = strings.TrimSpace(remoteAddr)
		}
	}
	return username, ip
}

func formatRetryAfter(ttl time.Duration) string {
	seconds := int(ttl.Round(time.Second) / time.Second)
	if seconds <= 0 {
		seconds = int(loginRateLimitWindow / time.Second)
	}
	return strconv.Itoa(seconds)
}

func unauthorizedRateLimitError(ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("too many login attempts")
	}
	return fmt.Errorf("too many login attempts, retry after %s", ttl.Round(time.Second))
}
