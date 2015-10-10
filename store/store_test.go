package store

import (
	"io/ioutil"
	"os"
	"testing"
)

func Test_StoreOpen(t *testing.T) {
	s := New()
	tmpDir, _ := ioutil.TempDir("", "store_test")
	defer os.RemoveAll(tmpDir)

	s.RaftBind = "127.0.0.1:8088"
	s.RaftDir = tmpDir
	if s == nil {
		t.Fatalf("failed to create store")
	}

	if err := s.Open(false); err != nil {
		t.Fatalf("failed to open store: %s", err)
	}
}
