package example

import (
	"strconv"
)

// UserID represents a unique identifier for a user in the system.
// It's defined as a custom type based on uint64 to provide type safety
// and prevent accidental mixing of user IDs with other numeric values.
type UserID uint64

// String implements the fmt.Stringer interface for UserID, providing
// a string representation of the user ID for logging, debugging, and
// API parameter formatting.
func (id UserID) String() string { return strconv.FormatUint(uint64(id), 10) }

// User represents a user entity in the system, containing the essential
// information needed for user management operations through the API.
type User struct {
	ID   UserID `json:"id"`
	Name string `json:"name"`
}

// Predefined sentinel errors for common API error conditions.
// These errors provide type-safe, comparable error values that can be
// used with errors.Is() for error handling and flow control.
const (
	// ErrUserNotFound indicates that a requested user was not found.
	// Typically returned when a user ID doesn't exist in the system.
	ErrUserNotFound sentinelError = "user not found"

	// ErrUnauthorized indicates that the request lacks valid authentication.
	// Typically returned for 401 Unauthorized HTTP responses.
	ErrUnauthorized sentinelError = "unauthorized"
)

// sentinelError provides a simple way to create constant error values
// that can be compared using errors.Is().
type sentinelError string

// Error implements the error interface for sentinelError, returning
// the string representation of the error for display and logging.
func (err sentinelError) Error() string { return string(err) }
