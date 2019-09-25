// genesis.go - genesis appstate for Nym ABCI
// Copyright (C) 2019  Nym Authors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package nymapplication

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type CoconutProperties struct {
	// Defines maximum number of attributes the coconut keys of the issuers can sign.
	MaximumAttributes int `json:"q"`
	// Defines the threshold parameter of the coconut system, i.e. minimum number of issuers required to successfully
	// issue a credential
	Threshold int `json:"threshold"`
}

type SystemProperties struct {
	WatcherThreshold  int               `json:"watcherThreshold"`
	VerifierThreshold int               `json:"verifierThreshold"`
	RedeemerThreshold int               `json:"redeemerThreshold"`
	PipeAccount       ethcommon.Address `json:"pipeAccount"`
	CoconutProperties CoconutProperties `json:"coconutProperties"`
}

type Issuer struct {
	// While currently Issuers do not need any additional keypair to interact with the blockchain, it might be useful
	// to just leave it in genesis app state would we ever need it down the line.
	PublicKey []byte `json:"pub_key"`
	// The coconut verification key of the particular issuer.
	VerificationKey []byte `json:"vk"`
}

type Watcher struct {
	// Public key associated with given watcher. Used to authenticate any notifications they send to the chain.
	PublicKey []byte `json:"pub_key"`
}

type Verifier struct {
	// Public key associated with given verifier. Used to authenticate any notifications they send to the chain.
	PublicKey []byte `json:"pub_key"`
}

type Redeemer struct {
	// Public key associated with given redeemer. Used to authenticate any notifications they send to the chain.
	PublicKey []byte `json:"pub_key"`
}

type GenesisAccount struct {
	Address ethcommon.Address `json:"address"`
	Balance uint64            `json:"balance"`
}

// GenesisAppState defines the json structure of the the AppState in the Genesis block. This allows parsing it
// and applying appropriate changes to the state upon InitChain.
type GenesisAppState struct {
	SystemProperties    SystemProperties `json:"systemProperties"`
	Accounts            []GenesisAccount `json:"accounts"`
	Issuers             []Issuer         `json:"issuingAuthorities"`
	EthereumWatchers    []Watcher        `json:"ethereumWatchers"`
	CredentialVerifiers []Verifier       `json:"verifiers"`
	TokenRedeemers      []Redeemer       `json:"redeemers"`
}
