#!/bin/bash


rm -rf /tmp/home/

mkdir -p /tmp/home/log/
mkdir -p /tmp/home/hdd/hornet/
mkdir -p /tmp/home/ssd/hornet/
mkdir -p /tmp/home/dev/shm/hornet/

cp hornet.yaml ../local_hornet.yaml
