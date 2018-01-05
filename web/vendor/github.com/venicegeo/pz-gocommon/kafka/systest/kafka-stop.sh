#!/bin/sh
set -e

config=/usr/local/etc/kafka
tmp=/tmp
bin=/usr/local/bin

$bin/kafka-server-stop
$bin/zookeeper-server-stop
