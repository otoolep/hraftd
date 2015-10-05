package main

import "flag"

// Program parameters
var httpAddr string
var raftAddr string
var joinAddr string
var raftDir string

const (
	DefaultHTTPAddr = ":11000"
	DefaultRaftAddr = ":11001"
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

	n := hraftd.New(httpAddr, raftAddr, joinAddr)
}
