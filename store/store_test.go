package store

import "testing"

func Test_StoreOpen(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatalf("failed to create store")
	}

	if err := s.Open(); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}
}
