#!/usr/bin/env bash

# node1 is also a faucet
# node2 is also a provider
# node3 is also a provider

NUM_NODES=4
APP_STATE_ORIGINAL='\"app_hash\": \"\"'
APP_STATE_REPLACEMENT=$(<awsnetdata/temp_escaped_genesis_app_state)

if [ ! -z "$1" ]; then
    if [ "$1" = "-rm" ]; then 
        for (( i = 1; i <= $NUM_NODES; i++ )); do
            echo "removing remote data on node $i..."
            ssh fullnode$i.nym "rm -rf ~/nymnet"
        done
    fi
fi

rm -rf `pwd`/build/aws_tmp_nodes
mkdir -p build/aws_tmp_nodes
docker run --user $(id -u ${USER}):$(id -g ${USER}) --rm -v `pwd`/build/aws_tmp_nodes:/tendermint:Z tendermint/tendermint testnet --v $NUM_NODES --o . --populate-persistent-peers --starting-ip-address 1.2.3.1 

# TODO: just have a json and run a script to automatically escape it. This is going much easier to manage and maintain

for (( i = 1; i <= $NUM_NODES; i++ )); do
    echo "Setting up node $i"

    echo "Setting and copying tendermint node data..."

    let "di = i - 1"
    # replace app state
    sed -i -e "s/$APP_STATE_ORIGINAL/$APP_STATE_REPLACEMENT/g" build/aws_tmp_nodes/node$di/config/genesis.json

    # for now let's just be lazy about it and replace the addresses with the target value
    # TODO: in future just set a proper persistent_peers list or a better solution, because how is it going to work if we decide to add new node while system is running?
    for (( j = 1; j <= $NUM_NODES; j++ )); do
        addr="1.2.3."$j
        target=$(awk '/^Host fullnode'$j'.nym$/{x=1}x&&/Hostname/{print $2;exit}' ~/.ssh/config)
        sed -i -e "s/$addr/$target/g" build/aws_tmp_nodes/node$di/config/config.toml
    done

    ssh fullnode$i.nym "mkdir -p ~/nymnet/node/ && mkdir -p ~/nymnet/issuer/coconutkeys/ && mkdir -p ~/nymnet/ethereum_watcher/ && mkdir -p ~/nymnet/verifier/issuerKeys/ && mkdir -p ~/nymnet/redeemer/"

    scp -r build/aws_tmp_nodes/node$di/* fullnode$i.nym:~/nymnet/node

    echo "Copying nym validator data..."
    scp awsnetdata/issuers/configs/config$i.toml fullnode$i.nym:~/nymnet/issuer/config.toml
    scp awsnetdata/issuers/keys/coconutkeys/threshold-secretKey-id=$i-attrs=5-n=5-t=3.pem fullnode$i.nym:~/nymnet/issuer/coconutkeys/
    scp awsnetdata/issuers/keys/coconutkeys/threshold-verificationKey-id=$i-attrs=5-n=5-t=3.pem fullnode$i.nym:~/nymnet/issuer/coconutkeys/

    echo "Copying ethereum watcher data..."
    scp awsnetdata/ethereum-watchers/configs/config$i.toml fullnode$i.nym:~/nymnet/ethereum_watcher/config.toml
    scp awsnetdata/ethereum-watchers/keys/watcher$i.key fullnode$i.nym:~/nymnet/ethereum_watcher/watcher.key

    echo "Copying credential verifier data..."
    scp awsnetdata/verifiers/configs/config$i.toml fullnode$i.nym:~/nymnet/verifier/config.toml
    scp awsnetdata/verifiers/keys/verifier$i.key fullnode$i.nym:~/nymnet/verifier/verifier.key
    scp awsnetdata/issuers/keys/coconutkeys/threshold-verificationKey-* fullnode$i.nym:~/nymnet/verifier/issuerKeys/

    echo "Copying token redeemer data..."
    scp awsnetdata/redeemers/configs/config$i.toml fullnode$i.nym:~/nymnet/redeemer/config.toml
	scp awsnetdata/redeemers/keys/redeemer$i.key fullnode$i.nym:~/nymnet/redeemer/redeemer.key
	scp awsnetdata/redeemers/keys/pipeaccount.key fullnode$i.nym:~/nymnet/redeemer/pipeAccount.key

    if (($i == 1 )); then
        echo "Copying faucet data..."
        ssh fullnode$i.nym "mkdir -p ~/nymnet/faucet/"
        scp awsnetdata/faucet/config.toml fullnode$i.nym:~/nymnet/faucet/config.toml
		scp awsnetdata/faucet/faucet.key fullnode$i.nym:~/nymnet/faucet/faucet.key
    fi
    if (($i == 2 || $i == 3)); then
        echo "Copying provider data..."
        ssh fullnode$i.nym "mkdir -p ~/nymnet/provider/issuerKeys && mkdir -p ~/nymnet/provider/accountKey"

        # naive workaround but temporarily works
        let "pi = i - 1"

        scp awsnetdata/providers/configs/config$pi.toml fullnode$i.nym:~/nymnet/provider/config.toml
		scp awsnetdata/providers/keys/accountKeys/provider$pi.key fullnode$i.nym:~/nymnet/provider/accountKey/provider.key
		scp awsnetdata/issuers/keys/coconutkeys/threshold-verificationKey-* fullnode$i.nym:~/nymnet/provider/issuerKeys/
    fi   
done

