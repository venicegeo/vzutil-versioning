#!/bin/sh
set -e

config=/usr/local/etc/kafka
tmp=/tmp
bin=/usr/local/bin

#$bin/zookeeper-server-start $config/zookeeper.properties > $tmp/zookeeper.log &
#sleep 2
$bin/kafka-server-start $config/server.properties > $tmp/kafka.log &
