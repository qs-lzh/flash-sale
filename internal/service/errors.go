package service

import (
	"errors"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrInvalidCredential = errors.New("invalid credential")
)

// error for reservation service
var (
	ErrNoTicketsAvailable = errors.New("no tickets available")
	ErrShowtimeNotExist   = errors.New("the showtime doesn't not exist")
	ErrAlreadyReserved    = errors.New("the user have already have the same reservation")
)
