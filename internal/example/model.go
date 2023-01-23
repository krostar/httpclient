package example

import (
	"strconv"
)

type UserID uint64

func (id UserID) String() string { return strconv.FormatUint(uint64(id), 10) }

type User struct {
	ID   UserID
	Name string
}

const (
	ErrUserNotFound sentinelError = "user not found"
	ErrUnauthorized sentinelError = "unauthorized"
)

type sentinelError string

func (err sentinelError) Error() string { return string(err) }
