package store

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

// Test_StoreOpen tests that the store can be opened.
func Test_StoreOpen(t *testing.T) {
	s := New(false)
	tmpDir, _ := ioutil.TempDir("", "store_test")
	defer os.RemoveAll(tmpDir)

	s.RaftBind = "127.0.0.1:0"
	s.RaftDir = tmpDir
	if s == nil {
		t.Fatalf("failed to create store")
	}

	if err := s.Open(false, "node0"); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}
}

// Test_StoreOpenSingleNode tests that a command can be applied to the log
func Test_StoreOpenSingleNode(t *testing.T) {
	s := New(false)
	tmpDir, _ := ioutil.TempDir("", "store_test")
	defer os.RemoveAll(tmpDir)

	s.RaftBind = "127.0.0.1:0"
	s.RaftDir = tmpDir
	if s == nil {
		t.Fatalf("failed to create store")
	}

	if err := s.Open(true, "node0"); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)

	if err := s.Set("foo", "bar"); err != nil {
		t.Fatalf("failed to set key: %s", err.Error())
	}

	// Wait for committed log entry to be applied.
	time.Sleep(500 * time.Millisecond)
	value, err := s.Get("foo")
	if err != nil {
		t.Fatalf("failed to get key: %s", err.Error())
	}
	if value != "bar" {
		t.Fatalf("key has wrong value: %s", value)
	}

	if err := s.Delete("foo"); err != nil {
		t.Fatalf("failed to delete key: %s", err.Error())
	}

	// Wait for committed log entry to be applied.
	time.Sleep(500 * time.Millisecond)
	value, err = s.Get("foo")
	if err != nil {
		t.Fatalf("failed to get key: %s", err.Error())
	}
	if value != "" {
		t.Fatalf("key has wrong value: %s", value)
	}
}

// Test_StoreInMemOpenSingleNode tests that a command can be applied to the log
// stored in RAM.
func Test_StoreInMemOpenSingleNode(t *testing.T) {
	s := New(true)
	tmpDir, _ := ioutil.TempDir("", "store_test")
	defer os.RemoveAll(tmpDir)

	s.RaftBind = "127.0.0.1:0"
	s.RaftDir = tmpDir
	if s == nil {
		t.Fatalf("failed to create store")
	}

	if err := s.Open(true, "node0"); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)

	if err := s.Set("foo", "bar"); err != nil {
		t.Fatalf("failed to set key: %s", err.Error())
	}

	// Wait for committed log entry to be applied.
	time.Sleep(500 * time.Millisecond)
	value, err := s.Get("foo")
	if err != nil {
		t.Fatalf("failed to get key: %s", err.Error())
	}
	if value != "bar" {
		t.Fatalf("key has wrong value: %s", value)
	}

	if err := s.Delete("foo"); err != nil {
		t.Fatalf("failed to delete key: %s", err.Error())
	}

	// Wait for committed log entry to be applied.
	time.Sleep(500 * time.Millisecond)
	value, err = s.Get("foo")
	if err != nil {
		t.Fatalf("failed to get key: %s", err.Error())
	}
	if value != "" {
		t.Fatalf("key has wrong value: %s", value)
	}
}

func Test_StoreStatus(t *testing.T) {
	s := New(false)
	tmpDir, _ := ioutil.TempDir("", "store_test")
	defer os.RemoveAll(tmpDir)

	s.RaftBind = "127.0.0.1:0"
	s.RaftDir = tmpDir
	if s == nil {
		t.Fatalf("failed to create store")
	}

	if err := s.Open(true, "node0"); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}

	// assuming 3 seconds enough for raft to initialized
	time.Sleep(3 * time.Second)

	status, err := s.Status()
	if err != nil {
		t.Errorf("failed to get store status: %s", err)
	}

	if status.Me.ID != "node0" {
		t.Errorf("status `me.id` has invalid value")
	}
	if status.Me.Address != s.RaftBind {
		t.Errorf("status `me.address` has invalid value")
	}

	for _, follower := range status.Followers {
		if (follower.ID == status.Leader.ID) || (follower.Address == status.Leader.Address) {
			t.Fatalf("a node cannot be leader and follow at the same time")
		}
	}

	isMeInFollowersOrLeader := false
	for _, node := range append(status.Followers, status.Leader) {
		if node.ID == status.Me.ID && node.Address == status.Me.Address {
			isMeInFollowersOrLeader = true
			break
		}
	}
	if !isMeInFollowersOrLeader {
		t.Errorf("me must be exist exclusively as a leader or as a follower")
	}
}
