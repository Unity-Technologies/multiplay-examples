package a2s

import "fmt"

type (
	// UnsupportedQueryError is an error which represents an invalid SQP query header.
	UnsupportedQueryError struct {
		header []byte
	}
)

// NewUnsupportedQueryError returns a new instance of UnsupportedQueryError.
func NewUnsupportedQueryError(header []byte) error {
	return &UnsupportedQueryError{
		header: header,
	}
}

// Error returns the error string.
func (e *UnsupportedQueryError) Error() string {
	return fmt.Sprintf("unsupported query: %x", e.header)
}
