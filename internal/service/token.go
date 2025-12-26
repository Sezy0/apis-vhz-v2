package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"vinzhub-rest-api-v2/internal/model"

	"github.com/redis/go-redis/v9"
)

const (
	// TokenPrefix is the prefix for all session tokens
	TokenPrefix = "vht_"

	// TokenTTL is the default token lifetime (1 hour)
	TokenTTL = 1 * time.Hour

	// TokenRedisKeyPrefix is the Redis key prefix for tokens
	TokenRedisKeyPrefix = "vinzhub:token:"
)

// TokenService handles session token generation and validation.
type TokenService struct {
	redis *redis.Client
}

// NewTokenService creates a new token service.
func NewTokenService(redisClient *redis.Client) *TokenService {
	return &TokenService{
		redis: redisClient,
	}
}

// GenerateToken creates a new session token and stores it in Redis.
func (s *TokenService) GenerateToken(ctx context.Context, data model.TokenData) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := TokenPrefix + hex.EncodeToString(tokenBytes)

	data.CreatedAt = time.Now()
	data.ExpiresAt = data.CreatedAt.Add(TokenTTL)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to serialize token data: %w", err)
	}

	key := TokenRedisKeyPrefix + token
	err = s.redis.Set(ctx, key, jsonData, TokenTTL).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store token: %w", err)
	}

	log.Printf("[TokenService] Generated token for key_account_id=%d, roblox_id=%s, expires=%v",
		data.KeyAccountID, data.RobloxUserID, data.ExpiresAt)

	return token, nil
}

// ValidateToken checks if a token is valid and returns its data.
func (s *TokenService) ValidateToken(ctx context.Context, token string) (*model.TokenData, error) {
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}

	if len(token) < len(TokenPrefix) || token[:len(TokenPrefix)] != TokenPrefix {
		return nil, fmt.Errorf("invalid token format")
	}

	key := TokenRedisKeyPrefix + token
	jsonData, err := s.redis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("token not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var data model.TokenData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse token data: %w", err)
	}

	if time.Now().After(data.ExpiresAt) {
		s.redis.Del(ctx, key)
		return nil, fmt.Errorf("token expired")
	}

	return &data, nil
}

// RevokeToken deletes a token from Redis.
func (s *TokenService) RevokeToken(ctx context.Context, token string) error {
	key := TokenRedisKeyPrefix + token
	return s.redis.Del(ctx, key).Err()
}

// RefreshToken extends the TTL of an existing token.
func (s *TokenService) RefreshToken(ctx context.Context, token string) error {
	key := TokenRedisKeyPrefix + token

	jsonData, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return fmt.Errorf("token not found: %w", err)
	}

	var data model.TokenData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	data.ExpiresAt = time.Now().Add(TokenTTL)

	newJSON, _ := json.Marshal(data)
	return s.redis.Set(ctx, key, newJSON, TokenTTL).Err()
}
