#!/usr/bin/env bash

VERSION=v0.12.6

command -v unzip >/dev/null 2>&1 || { echo >&2 "unzip is not installed. Aborting."; exit 1; }
command -v docker >/dev/null 2>&1 || { echo >&2 "docker is not installed. Aborting."; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo >&2 "docker-compose is not installed. Aborting."; exit 1; }

# make sure we start fresh
rm -rf build 

echo "Downloading release $VERSION..."
curl -s https://api.github.com/repos/nymtech/nym/releases/tags/$VERSION \
  | grep browser_download_url \
  | grep $VERSION/linux_amd64.zip \
  | cut -d '"' -f 4 \
  | wget -qi -

echo "Unzipping the application ..."
mkdir -p build/nymapp
unzip linux_amd64.zip -d build/nymapp &> /dev/null

echo "Cleaning up the downloaded file..."
rm linux_amd64.zip

# Rather than building everything in one 'make' call, do it separately to have indication of progress and not be overwhelmed with docker messages
echo "Building Docker containers for the NYM system..."
echo "Building Tendermint node container..."
make build_nym_nodes &> /dev/null

echo "Building Nym Validator node container..."
make build_issuers &> /dev/null

echo "Building Ethereum Watcher container..."
make build_ethereum_watchers &> /dev/null

echo "Building Dummy Service Provider container..."
make build_providers &> /dev/null

echo "Building Credential Verifier container..."
make build_verifiers &> /dev/null

echo "Building Token Redeemer container..."
make build_redeemers &> /dev/null

echo "Building Faucet container..."
make build_faucet &> /dev/null

echo "Starting up all local containers..."
docker-compose up -d

# find call is included in case I messed up directory structure when packaging release
echo "#################"
echo "Starting up the GUI interface... When asked for config file, the one in localnetdata/localclient/config.toml will work for this local setup"
find build -type f -iname "clientapp" -exec {} \;
