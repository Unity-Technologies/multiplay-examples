package sqp

import (
	"errors"
	"fmt"
)

type (
	// ErrUnsupportedSQPVersion is an error which represents an invalid SQP version provided to the reader.
	ErrUnsupportedSQPVersion struct {
		version int8
	}
)

var (
	ErrChallengeMismatch   = errors.New("challenge mismatch")
	ErrInvalidPacketLength = errors.New("invalid packet length")
	ErrNoChallenge         = errors.New("no challenge")
	ErrUnsupportedQuery    = errors.New("unsupported query")
)

// NewErrUnsupportedSQPVersion returns a new instance of ErrUnsupportedSQPVersion.
func NewErrUnsupportedSQPVersion(version int8) error {
	return &ErrUnsupportedSQPVersion{
		version: version,
	}
}

// Error returns the error string.
func (e *ErrUnsupportedSQPVersion) Error() string {
	return fmt.Sprintf("unsupported sqp version: %d", e.version)
}
