# Nym

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://github.com/nymtech/nym-validator/blob/master/LICENSE)
<!-- [![Build Status](https://travis-ci.com/jstuczyn/CoconutGo.svg?branch=master)](https://travis-ci.com/jstuczyn/CoconutGo)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/0xacab.org/jstuczyn/CoconutGo)
[![Coverage Status](http://codecov.io/github/jstuczyn/CoconutGo/coverage.svg?branch=master)](http://codecov.io/github/jstuczyn/CoconutGo?branch=master) -->

This is the Nym validator. It includes a Go implementation of the [Coconut](https://arxiv.org/pdf/1802.07344.pdf) selective disclosure credentials scheme. Coconut supports threshold issuance on multiple public and private attributes, re-randomization and multiple unlinkable selective attribute revelations.

Nym validators use a [Tendermint](https://tendermint.com/) blockchain to keep track of credential issuance and prevent double-spending of credentials, and contains server-side to support querying of credentials. There is client-side code to re-randomize credentials.

Currently, there exist two ways of running the demo code. One relies on local setup, which is more stable and more likely to work correctly. The other, uses external machines hosting the appropriate services. However, due to its experimental nature and being under heavy development, the latter option is more unstable and the machines might be at an unexpected state at any given moment of time, including being turned off, hence running the code locally is recommended.

The local setup requires `docker` and `docker-compose` to be installed on the machine. In order to startup the system, one needs to run the `runLocalNym_v$VERSION.sh` script, where `$VERSION` is the current release version of the code. This will pull the appropriate release binary for the gui and build and start all docker containers according to the `docker-compose.yml` file. When the application is started, it will ask for a configuration file in order to know which network it's supposed to connect to. In that case choose file located at `build/nymapp/local_config.toml`.

For the less stable remote version, you need to get the latest binary from the release page of the repository and when asked for configuration file use `aws_config.toml` located in the zip file. Note that the machines you would be trying to connect to may be switched off at any time without notice.

For more information, see the [documentation](https://github.com/nymtech/docs/).
