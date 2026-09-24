#!/bin/bash
# start.sh

cd ~/Documents/code/go/skip-go/
killport 2000
go run . &

cd ./client/
killport 1500
pnpm run dev --host &
PID2=$!

trap "kill $PID1 $PID2" EXIT
wait
