package watcher

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nymtech/nym-validator/ethereum/token"
	"github.com/nymtech/nym-validator/ethereum/watcher/config"
	"github.com/nymtech/nym-validator/logger"
	"github.com/nymtech/nym-validator/tendermint/nymabci/code"
	"github.com/nymtech/nym-validator/tendermint/nymabci/transaction"
	"github.com/nymtech/nym-validator/worker"
	tmclient "github.com/tendermint/tendermint/rpc/client"
	"gopkg.in/op/go-logging.v1"
)

type Watcher struct {
	cfg *config.Config
	// TODO: change both to the 'bigger' clients that auto-reconnect, etc?
	ethClient *ethclient.Client
	tmClient  *tmclient.HTTP

	privateKey *ecdsa.PrivateKey

	log *logging.Logger
	worker.Worker
	haltedCh chan struct{}
	haltOnce sync.Once
}

// Wait waits till the Watcher is terminated for any reason.
func (w *Watcher) Wait() {
	<-w.haltedCh
}

// Shutdown cleanly shuts down a given Watcher instance.
func (w *Watcher) Shutdown() {
	w.haltOnce.Do(func() { w.halt() })
}

// right now it's only using a single worker so all of this is redundant,
// but more future proof if we decided to include more workers
func (w *Watcher) halt() {
	w.log.Notice("Starting graceful shutdown.")

	w.Worker.Halt()

	w.log.Notice("Shutdown complete.")

	close(w.haltedCh)
}

func (w *Watcher) processBlock(num *big.Int) error {
	w.log.Infof("Processing block at height %s", num.String())
	block := w.getFinalisedBlock(num)
	if block == nil {
		return fmt.Errorf("finalised block for height: %d is nil", num)
	}

	for i, tx := range block.Transactions() {
		w.log.Debugf("Processing Tx %d for block %s", i, num.String())
		if tx.To() == nil {
			w.log.Debugf("Nil Tx.To() result - probably a contract creation tx")
			continue
		}
		if tx.To().Hex() != w.cfg.Watcher.NymContract.Hex() { // transaction used the Nym ERC20 contract
			w.log.Debugf("The tx was not sent to the Nym ERC20 contract")
			continue
		}

		txHash := tx.Hash()
		tr := w.getTransactionReceipt(txHash)
		if len(tr.Logs) == 0 {
			w.log.Warning("Transaction logs struct is empty")
			continue
		}

		from, to := erc20decode(*tr.Logs[0])

		if to.Hex() != w.cfg.Watcher.PipeAccount.Hex() { // transaction didn't go to the pipeAccount
			w.log.Debugf("The tx did not go to the pipe account")
			continue
		}

		value := getValue(*tr.Logs[0])
		transferStr := fmt.Sprintf("Block %s [tx: %d]: %d Nyms were moved from %s to pipe account at %s",
			num.String(),
			i,
			value,
			from.Hex(),
			to.Hex(),
		)
		w.log.Notice(transferStr)
		tmtx, err := transaction.CreateNewTransferToPipeAccountNotification(w.privateKey,
			from,
			to,
			value.Uint64(),
			txHash,
		)
		if err != nil {
			w.log.Errorf("Failed to create notification transaction: %v", err)
			continue
		}

		res, err := w.tmClient.BroadcastTxCommit(tmtx)
		if err != nil {
			w.log.Errorf("Failed to send notification transaction: %v", err)
			return fmt.Errorf("failed to notify tendermint about: %s", transferStr)
		}
		w.log.Infof("Received tendermint Response.\nCheckCode: %v, "+
			"Check additional data: %v\nDeliverCode: %v Deliver Additional data: %v",
			code.ToString(res.CheckTx.Code),
			string(res.CheckTx.Data),
			code.ToString(res.DeliverTx.Code),
			string(res.DeliverTx.Data),
		)
	}

	return nil
}

func (w *Watcher) worker() {
	w.log.Noticef("Watching Ethereum blockchain at: %s", w.cfg.Watcher.EthereumNodeAddress)
	heartbeat := time.NewTicker(2500 * time.Millisecond)

	// Just to the simplest thing of incrementing the block number by one...
	var currentBlockNum *big.Int
	bigOne := big.NewInt(1)
	for {
		// make sure our starting block is not nil
		currentBlockNum = w.getLatestBlockNumber()
		if currentBlockNum == nil {
			time.Sleep(time.Second)
		}
		if err := w.processBlock(currentBlockNum); err != nil {
			w.log.Errorf("Failed to process block %s: %s", currentBlockNum.String(), err)
		} else {
			w.log.Debugf("Processed our first block and starting from: %s", currentBlockNum.String())
			break
		}
	}

	for {
		select {
		case <-w.HaltCh():
			return
		case <-heartbeat.C:
			latestBlockNumber := w.getLatestBlockNumber()
			w.log.Infof("most recent: %s", latestBlockNumber.String())
			if latestBlockNumber == nil {
				w.log.Warning("Failed to obtain latest block number - nil result")
				continue
			}
			if latestBlockNumber.Cmp(currentBlockNum) == 1 { // latest is bigger than current
				newTarget := big.NewInt(0)
				newTarget.Add(currentBlockNum, bigOne)

				if err := w.processBlock(newTarget); err != nil {
					w.log.Errorf("Failed to process block %s: %s", newTarget.String(), err)
				} else {
					// if we're done, increment
					currentBlockNum = newTarget
				}
			} else {
				w.log.Debugf("Latest block seems to be identical or smaller than current: %s / %s",
					latestBlockNumber.String(),
					currentBlockNum.String(),
				)
			}
		}
	}
}

// TODO: we should be able to simply return the transferEvent, instead of decoding
// from and to separately. But for some reason transferEvent.From and transferEvent.To
// are not deserializing from TokenABI in the same way as transferEvent.Tokens.
//
// Once that works we can get rid of the separate erc20Decode function.
func getValue(logData types.Log) *big.Int {
	tokenAbi, err := abi.JSON(strings.NewReader(string(token.TokenABI)))
	if err != nil {
		log.Fatal(err)
	}

	var transferEvent struct {
		From   common.Address
		To     common.Address
		Tokens *big.Int
	}

	err = tokenAbi.Unpack(&transferEvent, "Transfer", logData.Data)
	if err != nil {
		log.Fatalf("Failed to unpack transfer data: %s", err)
	}

	// Uncomment to see what I mean about From and To not deserializing:
	// fmt.Printf("Decoded To as: %s\n", transferEvent.To.Hex())
	// fmt.Printf("Decoded From as: %s\n", transferEvent.From.Hex())
	// fmt.Printf("Decoded Tokens as: %s\n", transferEvent.Tokens)
	// "Tokens" works, To and From unexpectedly come out as zeros.

	// Let's use whole tokens for display purposes. Later we'll need to figure out
	// denominations to keep things anonymized.
	var tokens = transferEvent.Tokens.Div(transferEvent.Tokens, big.NewInt(1000000000000000000))

	return tokens
}

// see https://stackoverflow.com/questions/52222758/erc20-tokens-transferred-information-from-transaction-hash re ERC20 token transfer structure
//
func erc20decode(log types.Log) (common.Address, common.Address) {
	erc20FromHash := log.Topics[1]
	erc20ToHash := log.Topics[2]
	from := common.BytesToAddress(erc20FromHash.Bytes())
	to := common.BytesToAddress(erc20ToHash.Bytes())
	return from, to
}

func (w *Watcher) getTransactionReceipt(txHash common.Hash) types.Receipt {
	tr, err := w.ethClient.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		w.log.Critical(fmt.Sprintf("Failed getting TransactionReceipt: %s", err))
	}
	return *tr
}

func (w *Watcher) getFinalisedBlock(latestBlockNumber *big.Int) *types.Block {
	finalisedBlockNumber := new(big.Int).Sub(latestBlockNumber, big.NewInt(w.cfg.Debug.NumConfirmations))
	block, err := w.ethClient.BlockByNumber(context.Background(), finalisedBlockNumber)
	if err != nil {
		w.log.Critical(fmt.Sprintf("Failed getting block: %s", err))
	}

	return block
}

// getFinalisedBalance returns the balance of the given account (typically the pipe account)
// as it was 13 blocks ago.
//
// We use 13 blocks to approximate "finality" but PoW chains are not really "final" in any rigorous sense.
// TODO: make this configurable and recommend a number for node runners to roll the dice on in the docs.
//
// TODO: for some reason I can't find the discussion of forks which made me think that
// 13 confirmations should have a one-in-a-million chance of a fork. Dig this out as
// a reference.
// func (w *Watcher) getFinalisedBalance(addr common.Address, latestBlockNumber *big.Int) *big.Int {
// 	finalisedBlockNumber := new(big.Int).Sub(latestBlockNumber, big.NewInt(w.cfg.Debug.NumConfirmations))

// 	balance, err := w.ethClient.BalanceAt(context.Background(), addr, finalisedBlockNumber)
// 	if err != nil {
// 		// log.Fatalf("Error getting account balance: %s", err)
// 		w.log.Critical(fmt.Sprintf("Failed getting account balance: %s", err))
// 	}

// 	return balance
// }

func (w *Watcher) getLatestBlockNumber() *big.Int {
	latestHeader, err := w.ethClient.HeaderByNumber(context.Background(), nil)
	if err != nil {
		// log.Fatalf("Error getting latest block header: %s", err)
		w.log.Critical(fmt.Sprintf("Failed getting latest block header: %s", err))
		return nil
	}

	return latestHeader.Number
}

func (w *Watcher) connectToEthereum(ethHost string) error {
	client, err := ethclient.Dial(ethHost)
	if err != nil {
		return fmt.Errorf("error connecting to Infura: %s", err)
	}

	w.ethClient = client
	return nil
}

func (w *Watcher) connectToTendermint(tmHost string) error {
	client := tmclient.NewHTTP(tmHost, "/websocket")
	if err := client.Start(); err != nil {
		return fmt.Errorf("could not connect to: %v (%v)", tmHost, err)
	}

	w.tmClient = client
	return nil
}

func New(cfg *config.Config) (*Watcher, error) {
	log, err := logger.New(cfg.Logging.File, cfg.Logging.Level, cfg.Logging.Disable)
	if err != nil {
		return nil, fmt.Errorf("failed to create a logger: %v", err)
	}
	watcherLog := log.GetLogger("watcher")
	watcherLog.Noticef("Logging level set to %v", cfg.Logging.Level)

	privateKey, err := crypto.LoadECDSA(cfg.Watcher.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load watcher's key: %v", err)
	}

	w := &Watcher{
		cfg:        cfg,
		privateKey: privateKey,
		log:        watcherLog,
		haltedCh:   make(chan struct{}),
	}

	if err := w.connectToEthereum(w.cfg.Watcher.EthereumNodeAddress); err != nil {
		return nil, err
	}

	if err := w.connectToTendermint(w.cfg.Watcher.TendermintNodeAddress); err != nil {
		return nil, err
	}

	w.Go(w.worker)

	return w, nil
}
