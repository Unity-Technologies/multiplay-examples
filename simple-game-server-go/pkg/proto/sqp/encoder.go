package sqp

import (
	"bytes"
	"encoding/binary"
)

type (
	// encoder is a struct which implements proto.WireEncoder.
	encoder struct{}
)

// WriteString writes a string to the provided buffer.
func (e *encoder) WriteString(resp *bytes.Buffer, s string) error {
	if err := binary.Write(resp, binary.BigEndian, byte(len(s))); err != nil {
		return err
	}

	return binary.Write(resp, binary.BigEndian, []byte(s))
}

// Write writes arbitrary data to the provided buffer.
func (e *encoder) Write(resp *bytes.Buffer, v interface{}) error {
	return binary.Write(resp, binary.BigEndian, v)
}
