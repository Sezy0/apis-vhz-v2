package model

import "time"

// TokenData contains the data stored with a session token.
type TokenData struct {
	KeyAccountID   int64     `json:"key_account_id"`
	KeyID          int64     `json:"key_id"`
	RobloxUserID   string    `json:"roblox_user_id"`
	RobloxUsername string    `json:"roblox_username"`
	HWID           string    `json:"hwid"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}
