# Installation

The below instructions are outdated. Please refer to <https://nymtech.net/docs/>

## OUTDATED DOCS

The installation process of the Nym validator for development purposes takes multiple steps.

0. Ensure you have correctly installed and configured docker and docker-compose.

1. Firstly get the copy of the repo with either `git clone git@github.com:nymtech/nym.git` or `go get github.com/nymtech/nym-validator`. Using the first command is recommended in case there were any issue with go tools.

2. Build the entire system by invoking `make localnet-build`.

3. If you wish to modify keys used by issuers or their configuration, modify files inside `localnetdata/` directory. Currently those files are being copied into Docker volumes.

4. Run the system with `make localnet-start`
