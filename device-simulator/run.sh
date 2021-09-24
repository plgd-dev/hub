#!/usr/bin/env bash
set -e

logbt --setup
echo Spawning $NUM_DEVICES devices


umask 0000
mkdir -p /tmp/logbt-coredumps
pids=()
for ((i=0;i<$NUM_DEVICES;i++)); do
    logbt -- /iotivity-lite/port/linux/cloud_server $@ > /tmp/$i.log &
    pids+=($!)
done

# Naive check runs checks once a minute to see if either of the processes exited.
# This illustrates part of the heavy lifting you need to do if you want to run
# more than one service in a container. The container exits with an error
# if it detects that either of the processes has exited.
# Otherwise it loops forever, waking up every 60 seconds
while sleep 10; do 
for (( i=0; i<${#pids[@]}; i++ ));
do
    if ! kill -0 ${pids[$i]} 2>/dev/null; then
        echo "cloud_server[$i] with pid=${pids[$i]} is dead"
        exit 1
    fi
done
echo sleeping
done