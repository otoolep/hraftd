
#只能通过leader的http端口来进行set
curl -XPOST localhost:11000/key -d '{foo: bar}'

#cluster中的节点都可以获取状态
curl -XGET localhost:11000/key/foo
