#!/usr/bin/env bash

go build

#启动leader节点
$GOPATH/bin/hraftd -id node0  ~/node0

#启动node1
$GOPATH/bin/hraftd -id node1 -haddr :11001 -raddr :12001 -join :11000 ~/node1

#只能通过leader的http端口来进行set
curl -XPOST localhost:11000/key -d '{foo: bar}'

#cluster中的节点都可以获取状态
curl -XGET localhost:11000/key/foo
