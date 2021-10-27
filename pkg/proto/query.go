package proto

import (
	"bytes"
	"reflect"
)

type (
	// QueryResponder represents an interface to a concrete type which responds
	// to query requests.
	QueryResponder interface {
		Respond(clientAddress string, buf []byte) ([]byte, error)
	}

	// WireEncoder is an interface which allows for different query implementations
	// to write data to a byte buffer in a specific format.
	WireEncoder interface {
		WriteString(resp *bytes.Buffer, s string) error
		Write(resp *bytes.Buffer, v interface{}) error
	}

	// QueryState represents the state of a currently running game.
	QueryState struct {
		CurrentPlayers int32
		MaxPlayers     int32
		ServerName     string
		GameType       string
		Map            string
		Port           uint16
	}
)

// WireWrite writes the provided data to resp with the provided WireEncoder w.
func WireWrite(resp *bytes.Buffer, w WireEncoder, data interface{}) error {
	t := reflect.TypeOf(data)
	vs := reflect.Indirect(reflect.ValueOf(data))
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		v := vs.FieldByName(f.Name)

		// Dereference pointer
		if f.Type.Kind() == reflect.Ptr {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}

		switch v.Kind() {
		case reflect.Struct:
			if err := WireWrite(resp, w, v.Interface()); err != nil {
				return err
			}

		case reflect.String:
			if err := w.WriteString(resp, v.String()); err != nil {
				return err
			}

		default:
			if err := w.Write(resp, v.Interface()); err != nil {
				return err
			}
		}
	}

	return nil
}
