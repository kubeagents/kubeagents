package store

import "errors"

// ErrNotFound represents a not found error
var ErrNotFound = errors.New("not found")

// ErrDuplicateEmail represents a duplicate email error
var ErrDuplicateEmail = errors.New("email already exists")
