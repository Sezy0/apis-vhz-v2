package uid

import "github.com/google/uuid"

// New generates a new unique identifier.
func New() string {
	return uuid.New().String()
}

// IsValid checks if a string is a valid UUID.
func IsValid(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
