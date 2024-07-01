#! /bin/bash

PEBBLE_DIR=~/code/pebble
MY_DIR=$(pwd)

export TERM=xterm
export PEBBLE=/tmp/pebble

tmux new-session -d -s simulator

tmux split-window -t simulator:0 -h
tmux split-window -t simulator:0 -v
tmux split-window -t simulator:0 -v
tmux select-layout -t simulator:0 tiled

tmux send-keys -t simulator:0.0 "cd $PEBBLE_DIR && go run ./cmd/pebble run" C-m
tmux send-keys -t simulator:0.1 "watch 'tail charm.log -n 10'" C-m
tmux send-keys -t simulator:0.2 'go run sim.go' C-m
tmux send-keys -t simulator:0.3 "sleep 0.1 && cd $PEBBLE_DIR && go run ./cmd/pebble add base $MY_DIR/base.yaml && cd $MY_DIR" C-m
tmux send-keys -t simulator:0.3 'python3 -m http.server 8080' C-m

tmux attach-session -t simulator

tmux kill-session -t simulator
