// redeemer.go - A Nym redeemer
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

// Package redeemer defines a Nym redeemer responsible for 'confirming' requests to redeem Nym tokens back to ERC20
// accounts.
package redeemer

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	monitor "github.com/nymtech/nym-validator/common/tendermintmonitor"
	ethclient "github.com/nymtech/nym-validator/ethereum/client"
	"github.com/nymtech/nym-validator/logger"
	"github.com/nymtech/nym-validator/redeemer/config"
	"github.com/nymtech/nym-validator/server/storage"
	nymclient "github.com/nymtech/nym-validator/tendermint/client"
	"github.com/nymtech/nym-validator/tendermint/nymabci/code"
	tmconst "github.com/nymtech/nym-validator/tendermint/nymabci/constants"
	"github.com/nymtech/nym-validator/tendermint/nymabci/transaction"
	"github.com/nymtech/nym-validator/worker"
	"gopkg.in/op/go-logging.v1"
)

const (
	dbName          = "redeemerStore"
	backoffDuration = time.Second * 10
)

type Redeemer struct {
	privateKey *ecdsa.PrivateKey // TODO: move it elsewhere?
	cfg        *config.Config
	monitor    *monitor.Monitor
	store      *storage.Database
	nymClient  *nymclient.Client
	ethClient  *ethclient.Client
	log        *logging.Logger
	worker.Worker
	haltedCh chan struct{}
	haltOnce sync.Once
}

func (r *Redeemer) notifyOfTxStatus(txHash ethcommon.Hash) {
	ticker := time.NewTicker(time.Duration(r.cfg.Debug.TransactionStatusQueryInterval) * time.Millisecond)
	ctxQuery, cancelQuery := context.WithTimeout(context.Background(),
		time.Duration(r.cfg.Debug.TransactionStatusQueryTimeout)*time.Millisecond,
	)
	defer cancelQuery()
	for {
		select {
		case <-ctxQuery.Done():
			// TODO: this log is not thread safe...
			r.log.Warning("Context timeout - we do no know if the tx suceeded or not")
			return
		case <-ticker.C:
			status := r.ethClient.GetTransactionStatus(ctxQuery, txHash)
			switch status {
			case ethclient.TxStatusUnknown:
				r.log.Warningf("Tx %v is in unknown state...", txHash.Hex())
			case ethclient.TxStatusPending:
				r.log.Infof("Tx %v is still pending", txHash.Hex())
			case ethclient.TxStatusRejected:
				r.log.Errorf("Tx %v was rejected!", txHash.Hex())
				// TODO: what to do now with that information?
				return
			case ethclient.TxStatusAccepted:
				r.log.Noticef("Tx %v was accepted", txHash.Hex())
				return
			}
		}
	}
}

func (r *Redeemer) worker() {
	for {
		select {
		case <-r.HaltCh():
			r.log.Debug("Returning from initial select")
			return
		default:
			r.log.Debug("Default")
		}

		height, nextBlock := r.monitor.GetLowestUnprocessedBlock()
		if nextBlock == nil {
			r.log.Debugf("No blocks to process at height: %v", height)
			select {
			case <-r.HaltCh():
				r.log.Debug("Returning from backoff select")
				return
			case <-time.After(backoffDuration):
				r.log.Debug("time after")
			}
			continue
		}

		r.log.Debugf("Processing block at height: %v", height)

		for i, txRes := range nextBlock.Txs {
			tx := txRes.DeliverResult
			if tx.Code != code.OK || len(tx.Events) == 0 ||
				!bytes.HasPrefix(tx.Events[0].Attributes[0].Key, tmconst.RedeemTokensRequestKeyPrefix) {
				r.log.Infof("Tx %v at height %v is not a redeem token request", i, height)
				continue
			}

			kvpair := tx.Events[0].Attributes[0]
			// remember that the key field is: [ Prefix || User || amount || nonce ]
			// and all of them have constants lengths
			plen := len(tmconst.RedeemTokensRequestKeyPrefix)
			alen := ethcommon.AddressLength

			addressBytes := kvpair.Key[plen : plen+alen]
			address := ethcommon.BytesToAddress(addressBytes)
			amount := binary.BigEndian.Uint64(kvpair.Key[plen+alen:])
			nonce := kvpair.Key[plen+alen+8:]

			r.log.Debugf("Received data. Address: %v, amount: %v, nonce: %v", address, amount, nonce)

			// TODO: perhaps do similarly to what 'verifier' does as in delegate all work
			// (even though it's literally just to send the transaction) to serverworker?

			// crete notification
			notification, err := transaction.CreateNewTokenRedemptionConfirmationNotification(r.privateKey,
				address,
				amount,
				nonce,
			)
			if err != nil {
				r.log.Warningf("failed to create notification tx for: %v: %v", address, err)
			}

			res, err := r.nymClient.Broadcast(notification)
			if err != nil {
				r.log.Warningf("failed to send notification tx for %v: %v", address, err)
			}

			if res.CheckTx.Code == code.ALREADY_COMMITTED || res.DeliverTx.Code == code.ALREADY_COMMITTED {
				r.log.Notice("The threshold was already reached before and another redeemer should have sent Ethereum transaction")
				continue
			}

			if res.CheckTx.Code != code.OK || res.DeliverTx.Code != code.OK {
				r.log.Warningf("Notification transaction failed to be successfully executed on the chain"+
					"checkCode: %v (%v), deliverCode: %v (%v)",
					res.CheckTx.Code,
					code.ToString(res.CheckTx.Code),
					res.DeliverTx.Code,
					code.ToString(res.DeliverTx.Code),
				)
				continue
			}

			// at this point all should be fine
			if len(res.DeliverTx.Data) != 8 {
				r.log.Warningf("Data field has unexpected length (%v), expecting 8 (threshold || count)", len(res.DeliverTx.Data))
				continue
			}

			threshold := binary.BigEndian.Uint32(res.DeliverTx.Data)
			count := binary.BigEndian.Uint32(res.DeliverTx.Data[4:])

			r.log.Noticef("Threshold: %v, our count: %v", threshold, count)
			if threshold == count {
				r.log.Notice("Our notification was the thresholdth one. Going to call the Ethereum contract")
				ctx, cancel := context.WithTimeout(context.Background(),
					time.Duration(r.cfg.Debug.EthereumCallTimeout)*time.Millisecond,
				)
				// we can't defer cancel as we usually do as this function might posssibly never terminate

				txHash, err := r.ethClient.TransferERC20Tokens(ctx, int64(amount), address)
				if err != nil {
					r.log.Errorf("Failed to send ERC20 tokens from the pipe account back to %v: %v", address, err)
					// TODO: if this entity was to stay, this error would need to be somehow resolved,
					// but in production we won't have to deal with redeemers, etc. so it's not much of an issue

					cancel()
					continue
				}

				// start a background goroutine to just monitor the tx status
				// TODO: perhaps down the line do something more?
				go r.notifyOfTxStatus(txHash)
				// call cancel explicitly rather than defer due to previously described reason
				cancel()
			} else {
				r.log.Notice("We haven't reached the threshold - another redeemer will need to call Ethereum contract")
			}
		}

		r.monitor.FinalizeHeight(height)
	}
}

// Wait waits till the Redeemer is terminated for any reason.
func (r *Redeemer) Wait() {
	<-r.haltedCh
}

// Shutdown cleanly shuts down a given Redeemer instance.
func (r *Redeemer) Shutdown() {
	r.haltOnce.Do(func() { r.halt() })
}

// right now it's only using a single worker so all of this is redundant,
// but more future proof if we decided to include more workers
func (r *Redeemer) halt() {
	r.log.Notice("Starting graceful shutdown.")
	r.Worker.Halt()

	if r.monitor != nil {
		r.log.Debugf("Stopping Tendermint monitor")
		r.monitor.Halt()
		r.monitor = nil
	}

	if r.store != nil {
		r.log.Debugf("Closing datastore")
		r.store.Close()
		r.store = nil
	}

	r.log.Notice("Shutdown complete.")

	close(r.haltedCh)
}

func New(cfg *config.Config) (*Redeemer, error) {
	log, err := logger.New(cfg.Logging.File, cfg.Logging.Level, cfg.Logging.Disable)
	if err != nil {
		return nil, fmt.Errorf("failed to create a logger: %v", err)
	}
	redeemerLog := log.GetLogger("redeemer")
	redeemerLog.Noticef("Logging level set to %v", cfg.Logging.Level)

	privateKey, err := crypto.LoadECDSA(cfg.Redeemer.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load watcher's key: %v", err)
	}

	pipeAccountKey, err := crypto.LoadECDSA(cfg.Redeemer.PipeAccountKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load pipe account key: %v", err)
	}

	nymClient, err := nymclient.New(cfg.Redeemer.BlockchainNodeAddresses, log)
	if err != nil {
		return nil, fmt.Errorf("failed to create a nymClient: %v", err)
	}

	pipeAccountAddress := ethcrypto.PubkeyToAddress(*pipeAccountKey.Public().(*ecdsa.PublicKey))
	ethClientCfg := ethclient.NewConfig(
		pipeAccountKey,
		cfg.Redeemer.EthereumNodeAddress,
		cfg.Redeemer.NymContract,
		pipeAccountAddress,
		log,
	)

	ethClient, err := ethclient.New(ethClientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create a ethClient: %v", err)
	}

	store, err := storage.New(dbName, cfg.Redeemer.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create a data store: %v", err)
	}

	mon, err := monitor.New(log, nymClient, store, fmt.Sprintf("Verifier%v", cfg.Redeemer.Identifier))
	if err != nil {
		return nil, fmt.Errorf("failed to spawn blockchain monitor")
	}

	r := &Redeemer{
		privateKey: privateKey,
		cfg:        cfg,
		monitor:    mon,
		store:      store,
		nymClient:  nymClient,
		ethClient:  ethClient,
		log:        redeemerLog,
		haltedCh:   make(chan struct{}),
	}

	r.Go(r.worker)
	return r, nil
}
