#!/bin/bash

DIR=$PWD
CMD=../../cmd

# Kill all logistics-* process
function cleanup {
  pkill ones-demo
}

cd $CMD/subscriber
exec -a dcf-subscriber ./subscriber-go -cfg=./res/config.json &
cd $DIR
sleep 3

cd $CMD/calculator
exec -a dcf-calculator ./calculator-go -cfg=./res/config-mqtt.json -mode=default &
cd $DIR
sleep 1

cd $CMD/populator
exec -a dcf-populator ./populator-go -cfg ./res/config.json &
cd $DIR
sleep 1

cd $CMD/populator-api
exec -a dcf-populator-api ./populator-api-go -cfg ./res/config.json &
cd $DIR

trap cleanup EXIT

while : ; do sleep 1 ; done