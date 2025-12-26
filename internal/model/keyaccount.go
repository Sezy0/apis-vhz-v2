package model

// KeyAccountValidation contains the result of key+hwid validation.
type KeyAccountValidation struct {
	KeyAccountID   int64
	KeyID          int64
	RobloxUserID   string
	RobloxUsername string
	HWID           string
	KeyStatus      string
}
