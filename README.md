# Nym validator

The Nym validator is a blockchain application. It maintains a token called `nym`, which can be used for staking on mixnodes in the [Nym network](https://nymtech.net/).

## Installing and running

See the [Nym validator quick-start docs](https://nymtech.net/docs/quickstart/run-a-validator/).


## Developing

If you are a Nym developer, you can run a development setup as follows.

First, start up all the Starport components (including the user interface) like so:

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


