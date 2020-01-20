#!/bin/bash

cd ../ && pwd

go build -o /tmp/hornet *.go

rm -rf /tmp/home/

mkdir -p /tmp/home/log/
mkdir -p /tmp/home/hdd/hornet/
mkdir -p /tmp/home/ssd/hornet/
mkdir -p /tmp/home/dev/shm/hornet/

cp -f hornet.yaml /tmp/
cp -f test/hornet.yaml /tmp/local_hornet.yaml

cd /tmp/ && ./hornet