#!/usr/bin/env bash

NUM_NODES=4

# node are strictly greater than 0
if (( $1 > 0 && $1 <= NUM_NODES )); then 
    echo "starting node $1"
    session="nymnet"

    tmux new-session -d -s $session -n "tmnode"
    tmux select-window -t $session:0
    tmux send-keys -t $session:0 "nym_nymnode -cfgFile ~/nymnet/node/config/config.toml -dataRoot ~/nymnet/node/" 
    tmux send-keys -t $session:0 C-m 

    # gives just enough time for the node to startup so that the other services would not need to go into timeout immediately
    sleep 2s 

    tmux new-window -t $session:1 -n "IA"
    tmux select-window -t $session:1
    tmux send-keys -t $session:1 "nym_issuer -f ~/nymnet/issuer/config.toml" 
    tmux send-keys -t $session:1 C-m 

    tmux new-window -t $session:2 -n "ethw."
    tmux select-window -t $session:2
    tmux send-keys -t $session:2 "nym_eth_watcher -f ~/nymnet/ethereum-watcher/config.toml" 
    tmux send-keys -t $session:2 C-m 

    tmux new-window -t $session:3 -n "ver."
    tmux select-window -t $session:3
    tmux send-keys -t $session:3 "nym_verifier -f ~/nymnet/verifier/config.toml" 
    tmux send-keys -t $session:3 C-m 

    tmux new-window -t $session:4 -n "red."
    tmux select-window -t $session:4
    tmux send-keys -t $session:4 "nym_redeemer -f ~/nymnet/redeemer/config.toml" 
    tmux send-keys -t $session:4 C-m 

    if (($1 == 1)); then 
        tmux new-window -t $session:5 -n "fauc."
        tmux select-window -t $session:5
        tmux send-keys -t $session:5 "nym_faucet -f ~/nymnet/faucet/config.toml" 
        tmux send-keys -t $session:5 C-m 
    fi

    if (($1 == 2 || $1 == 3)); then
        tmux new-window -t $session:5 -n "SP."
        tmux select-window -t $session:5
        tmux send-keys -t $session:5 "nym_provider -f ~/nymnet/provider/config.toml" 
        tmux send-keys -t $session:5 C-m 
    fi

    tmux select-window -t $session:0
    tmux attach-session -t $session
    
    exit
else 
    echo "invalid node number"
    exit
fi

