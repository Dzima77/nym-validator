# nym

**nym** is a blockchain application built using Cosmos SDK and Tendermint and generated with [Starport](https://github.com/tendermint/starport).

## Get started developing

```
starport serve
```

`serve` command installs dependencies, initializes and runs the application.

## Configure

Initialization parameters of your app are stored in `config.yml`.

### `accounts`

A list of user accounts created during genesis of your application.

| Key   | Required | Type            | Description                                       |
| ----- | -------- | --------------- | ------------------------------------------------- |
| name  | Y        | String          | Local name of the key pair                        |
| coins | Y        | List of Strings | Initial coins with denominations (e.g. "100coin") |

## Learn more

- [Starport](https://github.com/tendermint/starport)
- [Cosmos SDK documentation](https://docs.cosmos.network)
- [Cosmos Tutorials](https://tutorials.cosmos.network)
- [Channel on Discord](https://discord.gg/W8trcGV)


## Run it as infrastructure

First, build the node and cli:

```
mkdir build
go build -mod=mod -o build/nymd ./cmd/nymd
go build -mod=mod -o build/nymcli ./cmd/nymcli
```

SCP them to wherever you'd like to run them. 

Then generate the initial chain setup:

```
./nymd init qa-validator-moniker --chain-id qa-chain
./nymcli keys add qa-validator-key
./nymcli keys show qa-validator-key
./nymd add-genesis-account $(./nymcli keys show qa-validator-key -a) 1000000000nym,1000000000stake
./nymd gentx --name qa-validator-key --amount 10000000stake
./nymd  collect-gentxs
```

Lastly, start the chain and REST API:

```
./nymd start
./nymcli rest-server --chain-id qa-chain
```

At this point, you'll be running a single-node validator with a REST API on http://localhost:8081/swagger/index.html, a different (Cosmos) REST API on http://localhost:1317. These two will be merged in the next release. First we want to get the validators, currency, and mixnet all running, then we'll join them together once all the node types are working :).

The testnet setup is as above, with a few more `gentx` back and forth before starting the chain. 