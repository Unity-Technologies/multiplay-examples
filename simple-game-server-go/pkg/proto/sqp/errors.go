package sqp

import (
	"errors"
	"fmt"
)

type (
	// UnsupportedSQPVersionError is an error which represents an invalid SQP version provided to the reader.
	UnsupportedSQPVersionError struct {
		version int8
	}
)

var (
	ErrChallengeMalformed  = errors.New("challenge malformed")
	ErrChallengeMismatch   = errors.New("challenge mismatch")
	ErrInvalidPacketLength = errors.New("invalid packet length")
	ErrNoChallenge         = errors.New("no challenge")
	ErrUnsupportedQuery    = errors.New("unsupported query")
)

// NewUnsupportedSQPVersionError returns a new instance of UnsupportedSQPVersionError.
func NewUnsupportedSQPVersionError(version int8) error {
	return &UnsupportedSQPVersionError{
		version: version,
	}
}

// Error returns the error string.
func (e *UnsupportedSQPVersionError) Error() string {
	return fmt.Sprintf("unsupported sqp version: %d", e.version)
}
