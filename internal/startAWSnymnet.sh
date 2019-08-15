#!/usr/bin/env bash

# also temporary. ignore errors.
NUM_NODES=4

for (( i = 1; i <= $NUM_NODES; i++ )); do
    ssh fullnode$i.nym "~/startFullNode.sh $i"
done
