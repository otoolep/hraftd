package store

import "testing"

func Test_StoreOpen(t *testing.T) {
	s := New()
	s.RaftBind = "127.0.0.1:8088"
	s.RaftDir = "/tmp/raft"
	if s == nil {
		t.Fatalf("failed to create store")
	}

	if err := s.Open(); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}
}
