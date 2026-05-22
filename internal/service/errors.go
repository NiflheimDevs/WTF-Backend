package service

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveUser       = errors.New("user is inactive")
	ErrNotFound           = errors.New("resource not found")
	ErrInvalidTransition  = errors.New("invalid status transition")
)
