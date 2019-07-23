# Multi-node Clustering
What follows is a detailed example of running a multi-node hraftd cluster.

Imagine you have 3 machines, with the IP addresses 192.168.0.1, 192.168.0.2, and 192.168.0.3 respectively. Let's also assume that each machine can reach the other two machines using these addresses.

## Walkthrough
You should start the first node like so:
```
$GOPATH/bin/hraftd -id node1 -haddr 192.168.0.1:11000 -raddr 192.168.0.1:12000 ~/node
```
This way the node is listening on an address reachable from the other nodes. This node will start up and become leader of a single-node cluster.

Next, start the second node as follows:
```
$GOPATH/bin/hraftd -id node2 -haddr 192.168.0.2:11000 -raddr 192.168.0.2:12000 -join 192.168.0.1:11000 ~/node
```

Finally, start the third node as follows:
```
$GOPATH/bin/hraftd -id node3 -haddr 192.168.0.3:11000 -raddr 192.168.0.3:12000 -join 192.168.0.2:11000 ~/node
```

_Specifically using ports 11000 and 12000 is not required. You can use other ports if you wish._

Note how each node listens on its own address, but joins to the address of the leader node. The second and third nodes will start, join the with leader at `192.168.0.2:11000`, and a 3-node cluster will be formed.
