# nym-validator Changelog

## 0.13.0

* Introduced ability for the client to listen on a socket (TCP or a Websocket) for requests for getting credential, re-randomizing credential, spending credential and getting available service providers.

## 0.12.16

* Defined constant specifying length of FP12 (Gt element)

## 0.12.15

* Fixed possibly source of slice bounds out of range error when reading packet payload

## 0.12.14

* Increased the default blockchain (both Tendermint and Ethereum) polling rate

## 0.12.13

* Validators sending hardcoded port in their presence data

## 0.12.12

* Improved reliability of Tendermint chain monitor

## 0.12.11

* Validator reporting presence to directory server
* Improve Ethereum watcher reliability

## 0.12.10

* Changed references of imported BLS381 curve implementation from `github.com/nymtech/jstuczyn/version3/go/amcl/BLS381` to `github.com/nymtech/amcl/version3/go/amcl/BLS381`

## 0.12.9

* Renamed the repository from 'nym' into 'nym-validator'
* Updated server daemons behaviour
* Fixed possibly buffer overflow when reading packet from connection
* Removed references to the qt client application. It is being maintained now in separate repository

## 0.12.8

* Updated daemons of all entities to provide more information regarding required flags; similar to current design of loopix
* Made naming scheme slightly more consistent
* Restored travis-ci integration

## 0.12.7

* Provider depositing received credential in background, i.e. new gouroutine, after receiving it, hence reintroducing proper double-spending checks

## 0.12.6

* Hidden unnecessary parts of the UI
* Included configs in the release package
* Temporarily disabled sample service providers from depositing received credentials immediately as in the actual system they might be batching them

## 0.12.5

* More detailed error on server replies
* Increased default connection timeouts on both sides
* Possibly decreased maximum memory usage by token redeemer
* Increased number of displayed bytes of credential in the GUI

## 0.12.4

* Added option to specify Service Providers in client's config

## 0.12.3

* Added timeout to tendermint monitor startup. It is required for fresh chains that have not gone past genesis block
* Button to explicitly re-randomize credential
* Updating balance on load
* Other minor changes and fixes
* Configuration for nymnet hosted on AWS + scripts for automatically copying data over
* Makefile additions

## 0.12.2

* Fixed Dockerfile for the issuer
* Fixed Dockerfile for the provider
* Fixed Dockerfile for the verifier
* Added start-up script for running everything locally
* Adjusted width of the field displaying value of currently chosen credential

## 0.12.1

* Faucet checking and printing its own ERC20 and Ether balances before each request

## 0.12.0

* Added ERC20 Nym Faucet transfering the tokens (+ some Ether) to specified address
* Ability to 'register' new account from the GUI
* Some code refactoring

## 0.11.1

* Updated dependencies
* Updated Dockerfiles
* Updated install instructions

## 0.11.0

* Created GUI appliction for demonstrating the capabilities of the current system
* Adjustments to default timeout values
* Changed docker-compose to auto-restart Ethereum watchers
* Possibly fixed the issue of reconnection loop for the Tendermint client

## 0.10.4

* Added token redemption to sample client

## 0.10.3

* Changed `Tags` field in Tx struct inside Tendermint monitor to `Events` to simplify code and to be more consistent with the actual Tendermint
* Added a recover call GetServerResponses()

## 0.10.2

* Updated all used dependencies to most recent versions
* Fixed code using Tendermint due to breaking API changes present in version 0.32.0 of Tendermint

## 0.10.1

* Included Redeemers in docker-compose file

## 0.10.0

* Introduced Redeemer entity that monitors Tendermint chain for any requests to move tokens back into ERC20. When threshold number of them agrees on a request, only one of them calls the ERC20 smartcontract
* Changed all local import paths due to repository switch
* Decreased levels of logging in multiple locations to make outputs more readable
* Changed default address of the pipe account
* Increased default polling rate of Ethereum watchers
* Additional minor fixes and changes

## 0.9.1

* Sample client cleanup + description of what is actually happening
* Made Client's WaitForBalanceIncrease() function public and used in the demo code

## 0.9.0

* Credential verifier entity - they monitor the tendermint chain for any deposit requests and verify the written credentials (and cryptographic materials)
* Tendermint nodes waiting for threshold number of verifiers to validate the credential before increasing provider's balance
* Move of repository to github.com and related import path changes
* Moved tendermint monitor code to make it useable by different entities (issuer and verifier)
* Minor changes and fixes

## 0.8.2

* Restored provider's ability to redeem received credentials
* Tendermint-side handling of the above request (currently credentials are verified by the tendermint nodes, ON CHAIN)
* Ability to send a query to check if given zeta was spent (it does not indicate that it was NOT spent)
* Fixed checkIfAccountExists method

## 0.8.1

* Restored client's ability to query issued credentials
* Modified the way issuers are storing issued credentials
* Moved IssuedSignature struct to new issuer utils

## 0.8.0

* Separated "server" into separate provider and issuer
* Ability to register handlers for different types of requests for listener
* Ability to register handlers for different types of commands for serverworker
* Further inclusion of context argument to different processing methods
* Separate type for Threshold Coconut Keys - they include the ID used during generation
* Removed ServerID from ServerMetadata from all server responses - it's now included in relevant attached key
* Created shared daemon service code making it easier to create any future daemons
* Bug fix in PolyEval function causing possibly invalid results

## 0.7.1

* Additional method to wait for balance change for an ERC20 Nym
* Adjustments in watcher heartbeat interval

## 0.7.0

* Working conversion of ERC20 Nym tokens into coconut credentials
* Using Ethereum addresses for accounts on the Nym-Tendermint side
* Ability for watchers to send notification transactions to Tendermint chain
* Ability for client to query its Ethereum (ERC20 Nym) and Tendermint balances
* Changes to Tendermint app state and the genesis state
* More ERC20-Nym specific Ethereum-client methods
* Checks for whether binary were compiled in 64bit mode
* Moved all localnet related keys and configs to a dedicated directory
* Other minor changes and fixes

## 0.6.6

* Updated Nym Node genesis state to include Ethereum watchers
* Modified the nymnode dockerfile to allow include gcc required by Ethereum build process
* Updates all dependencies

## 0.6.5

* Introduced constants file with method signatures for ERC20 token functions
* Generalised Ethereum's client transfer function so rather than being hardcoded to transferring to the holding account using Nym contract, both of those attributes can be specified
* Introduced ECDSA keypair to Ethereum watcher
* Protobuf definitions for notifications watcher sends to Tendermint chain

## 0.6.4

* A lot of linter-related fixes

## 0.6.3

* Replaced all function calls in watcher file with methods on watcher object. Config object is no longer passed to them
* Ability to cleanly shutdown the watcher
* Fixed watcher logger

## v0.6.2

* Dedicated configuration file for the Ethereum watcher

## v0.6.1

* "Daemon" for Ethereum watcher
* Semi-split the watcher files

## v0.6.0

* Copied the Ethereum watcher codebase to the repository
* A very initial take on Ethereum client - ability to send Nym tokens to Holding Account
* Fixed remaining old tests

## v0.5.1

* Fixed monitor/processor deadlock when there are no blocks to be processed.

## v0.5.0

* Combined tendermint node and nym abci into a single binary to significantly simplify deployment and testing
* Minor bug fixes

## v0.4.0

* All entities in the system working - full ability to issue and spend credentials.
* Fixed provider-side handling of Spend Credential
* Reintroduced blockchain keys for providers
* Fixed infinite catchup look for issuers

## v0.3.2

* Client retrying to look up credentials with specified backoff interval
* Client correctly parsing look up credential responses from the issuers
* Minor refactoring and bug fixes

## v0.3.1

* Fixed issuers not storing issued credentials

## v0.3.0

* Issuers monitoring the blockchain state
* Issuers keeping persistent state with credentials for given txs
* Issuers resyncing with the blockchain upon startup or periodically after not receiving any data during a specified interval

## v0.2.2

* Docker-compose for the entire environment
* Issuers monitoring state of the Tendermint blockchain
* Bunch of Work in Progress files related to Issuers having internal state of signed requests

## v0.2.1

* Updated transfer to the holding account to return hash of the block including the tx
* IAs verifying that transfer to holding actually happened
* Finished logic for Provider to accept 'spend credential' requests

## v0.2.0

* Tumbler-related Coconut logic for sequential and concurrent computation
* Tendermint ABCI used to keep track of clients' tokens and preventing double spending of credentials
* IAs having extra set of keys used to authorise requests on the blockchain
* Provider accepting 'spend credential' request; Interaction with the blockchain is not implemented
* Ability of a client to request transfer of some of its tokens to "Holding Account"
* Work on clients' ability to spend credentials
* Bug fixes and refactor work
* Additional tests and updates to docstrings

## v0.1.5

* More shared code between client and server
* Fixed a bug where provider server would fail to aggregate received verification keys of IAs if it received more than threshold of them (even if they all were valid)

## v0.1.4

* Made ElGamal Public and Private key fields private and only accessible via method receivers

## v0.1.3

* Refactored repository structure
* Renamed BlindSignMats and BlindShowMats to Lambda and Theta respectively
* Refactored server/CryptoWorker and simplified main processing loop
* Fixed crash on GetVerificationKey[grpc] if any server was down

## v0.1.2

* Updated milagro library to the current version as of 10.01.2019

## v0.1.1

* Reimplemented commandsQueue using the created template

## v0.1.0

* Created template to generate infinite channel behaviour for any type

## v0.0.4

* Reimplemented JobQueue with different queue implementation to introduce thread safety

## v0.0.3

* Refactored server/comm/utils/utils.go
* Introduced ServerMetadata struct used in ServerRequests/Responses + associated changes
* Renamed crypto/coconut/concurrency/coconutworker/coconut_worker.go Worker to CoconutWorker + associated changes
* Renamed client/cryptoworker/cryptoworker.go Worker to CryptoWorker + associated changes
* Refactored /home/jedrzej/go/src/github.com/nymtech/nym-validator/server/cryptoworker/cryptoworker.go + associated changes

## v0.0.2

* Fixes jstuczyn/CoconutGo#4

## v0.0.1 - Initial Release

* Coconut Signature Scheme
* Initial Coconut Issuing Authority Server
* Initial Coconut Provider Server
* Initial Coconut Client interacting with the above
* TTP for generating keys for the IAs
