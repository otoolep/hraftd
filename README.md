hraftd [![Circle CI](https://circleci.com/gh/otoolep/hraftd/tree/master.svg?style=svg)](https://circleci.com/gh/otoolep/hraftd/tree/master) [![GoDoc](https://godoc.org/github.com/otoolep/hraftd?status.png)](https://godoc.org/github.com/otoolep/hraftd)
======

hraftd is a reference use of the [Hashicorp Raft implementation](https://github.com/hashicorp/raft), inspired by [raftd](https://github.com/goraft/raftd). Raft is a _distributed consensus protocol_, meaning its purpose is to ensure that a set of nodes -- a cluster -- agree on the state of some abitrary system, even when nodes are vulnerable to failure and network partitions. It is a fundamental part of building fault-tolerant systems.

Like raftd, the implementation is a very simple key-value store. You can set a key like so:

`curl -XPOST localhost:11000/key -d '{"foo": "bar"}'`

You can read the value for a key like so:

`curl -XGET localhost:11002/key/foo`

## Running hraftd
Starting and running a hraftd cluster is easy. Download hraftd like so:

```
mkdir hraftd
cd hraftd/
export GOPATH=$PWD
go get github.com/otoolep/hraftd
```

Run your first hraftd node like so:

`$GOPATH/hraftd ~/node0`

You can now set a key and read its value back:

```
curl -XPOST localhost:11000/key -d '{"user1": "batman"}'
curl -XGET localhost:11000/key/user1
```

### Bring up a cluster
Let's bring up 2 more nodes, so we have a 3-node cluster. That way we can tolerate the failure of 1 node:

```
$GOPATH/hraftd -haddr :11001 -raddr :12001 -join :11000 ~/node1
$GOPATH/hraftd -haddr :11002 -raddr :12002 -join :11000 ~/node2
```

This tells each new node to join the existing node. Once joined, each node now knows about the key:

```
curl -XGET localhost:11000/key/user1
curl -XGET localhost:11001/key/user1
curl -XGET localhost:11002/key/user1
```

Furthermore you add a new key:

`curl -XPOST localhost:11000/key -d '{"user2": "robin"}'`

Confirm that the new key has been set like so:

```
curl -XGET localhost:11000/key/user2
curl -XGET localhost:11001/key/user2
curl -XGET localhost:11002/key/user2
```

### Tolerating failure
Kill the leader process and watch one of the other nodes be elected leader. The keys are still available for query on the other nodes, and you can set keys on the new leader. Furthermore when the first node is restarted, it will rejoin the cluster and learn about any updates that occurred while it was down.
