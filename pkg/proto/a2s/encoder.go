package a2s

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
	return binary.Write(resp, binary.LittleEndian, []byte(s+"\x00"))
}

// Write writes arbitrary data to the provided buffer.
func (e *encoder) Write(resp *bytes.Buffer, v interface{}) error {
	return binary.Write(resp, binary.LittleEndian, v)
}
