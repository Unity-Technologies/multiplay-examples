//go:build generate
// +build generate

package sdkclient

import (

	// Import to force a dependency
	_ "golang.org/x/tools/cmd/stringer"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=EventType -output=event_string.go
