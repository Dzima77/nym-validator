// faucet.go - a temporary ERC20 Nym Faucet
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

// Package issuer defines structure for ERC20 Nym Faucet.
package faucet

import (
	"errors"
	"fmt"
	"sync"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/nymtech/nym-validator/logger"
	"github.com/nymtech/nym-validator/server"
	"github.com/nymtech/nym-validator/server/config"
	"gopkg.in/op/go-logging.v1"
)

// Faucet defines all the required attributes for a ERC20 Nym Faucet.
type Faucet struct {
	*server.BaseServer
	log *logging.Logger

	haltOnce sync.Once
}

// New returns a new Faucet instance parameterized with the specified configuration.
// nolint: gocyclo
func New(cfg *config.Config) (*Faucet, error) {
	// there is no need to further validate it, as if it's not nil, it was already done
	if cfg == nil {
		return nil, errors.New("nil config provided")
	}

	log, err := logger.New(cfg.Logging.File, cfg.Logging.Level, cfg.Logging.Disable)
	if err != nil {
		return nil, fmt.Errorf("failed to create a logger: %v", err)
	}

	faucetLog := log.GetLogger("Faucet - " + cfg.Server.Identifier)
	faucetLog.Noticef("Logging level set to %v", cfg.Logging.Level)

	baseServer, err := server.New(cfg, log)
	if err != nil {
		return nil, err
	}

	privateKey, err := ethcrypto.LoadECDSA(cfg.Faucet.BlockchainKeyFile)
	if err != nil {
		errStr := fmt.Sprintf("Failed to load Nym keys: %v", err)
		faucetLog.Error(errStr)
		return nil, errors.New(errStr)
	}

	faucetLog.Notice("Loaded Nym Blochain keys from the file.")

	faucet := &Faucet{
		BaseServer: baseServer,
		log:        faucetLog,
	}

	for i, l := range faucet.Listeners() {
		faucetLog.Debugf("Registering faucet handlers for listener %v", i)
		l.RegisterDefaultFaucetHandlers()
	}

	nodeAddress := cfg.Faucet.EthereumNodeAddress
	nymContract := cfg.Faucet.NymContract
	pipeContract := cfg.Faucet.PipeAccount
	etherAmount := cfg.Faucet.EtherAmount

	errCount := 0
	for i, sw := range faucet.ServerWorkers() {
		faucetLog.Debugf("Registering faucet handlers for serverworker %v", i)
		if err := sw.RegisterAsFaucet(privateKey, nodeAddress, nymContract, pipeContract, etherAmount, log); err != nil {
			errCount++
			faucetLog.Warningf("Could not register worker %v as faucet", i)
		}
	}

	if errCount == len(faucet.ServerWorkers()) {
		errMsg := "could not register any serverworker as faucet"
		faucetLog.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	return faucet, nil
}

// Shutdown cleanly shuts down a given Issuer instance.
func (f *Faucet) Shutdown() {
	f.haltOnce.Do(func() { f.halt() })
}

func (f *Faucet) halt() {
	f.log.Notice("Starting graceful shutdown.")

	f.BaseServer.Shutdown()
}
