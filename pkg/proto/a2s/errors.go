package a2s

import "fmt"

type (
	// ErrUnsupportedQuery is an error which represents an invalid SQP query header.
	ErrUnsupportedQuery struct {
		header []byte
	}
)

// NewErrUnsupportedQuery returns a new instance of ErrUnsupportedQuery.
func NewErrUnsupportedQuery(header []byte) error {
	return &ErrUnsupportedQuery{
		header: header,
	}
}

// Error returns the error string.
func (e *ErrUnsupportedQuery) Error() string {
	return fmt.Sprintf("unsupported query: %x", e.header)
}
