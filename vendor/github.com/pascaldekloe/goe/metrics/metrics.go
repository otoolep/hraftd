// Package metrics provides application metrics registration.
package metrics

import (
	"time"
)

type Register interface {
	// Seen increments the counter with n.
	Seen(key string, n int)

	// Took adds a timing from since to now.
	Took(key string, since time.Time)

	// KeyPrefix defines a prefix applied to all keys.
	KeyPrefix(string)
}

type dummy struct{}

// NewDummy returns a new Register which does nothing.
func NewDummy() Register {
	return dummy{}
}

func (d dummy) Seen(key string, n int)           {}
func (d dummy) Took(key string, since time.Time) {}
func (d dummy) KeyPrefix(s string)               {}
