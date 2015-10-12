package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/otoolep/hraftd/http"
	"github.com/otoolep/hraftd/store"
)

const (
	DefaultHTTPAddr = ":11000"
	DefaultRaftAddr = ":12000"
)

// Flag set
var fs *flag.FlagSet

func main() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	var (
		httpAddr = fs.String("haddr", DefaultHTTPAddr, "Set HTTP bind address")
		raftAddr = fs.String("raddr", DefaultRaftAddr, "Set Raft bind address")
		joinAddr = fs.String("join", "", "Set join address, if any")
		raftDir  = fs.String("rdir", "", "Set storage path for Raft")
	)
	_ = joinAddr
	fs.Parse(os.Args[1:])

	// Ensure Raft storage exists.
	if *raftDir == "" {
		fmt.Fprintf(os.Stderr, "No Raft storage directory specified\n")
		os.Exit(1)
	}
	os.MkdirAll(*raftDir, 0700)

	s := store.New()
	s.RaftDir = *raftDir
	s.RaftBind = *raftAddr
	if err := s.Open(*joinAddr == ""); err != nil {
		log.Fatalf("failed to open store: %s", err.Error())
	}

	h := httpd.New(*httpAddr, s)
	if err := h.Start(); err != nil {
		log.Fatalf("failed to start HTTP service: %s", err.Error())
	}

	// If join was specified, make the join request.
	if *joinAddr != "" {
		if err := join(*joinAddr, *raftAddr); err != nil {
			log.Fatalf("failed to join node at %s: %s", *joinAddr, err.Error())
		}
	}

	log.Println("hraft started successfully")

	select {}
}

func join(joinAddr, raftAddr string) error {
	b, err := json.Marshal(map[string]string{"addr": raftAddr})
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/join", joinAddr), "application-type/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
