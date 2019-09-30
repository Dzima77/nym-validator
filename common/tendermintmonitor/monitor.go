// monitor.go - Tendermint Blockchain monitor.
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

// Package monitor implements the support for monitoring the state of the Tendermint Blockchain
package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/sortkeys"
	"github.com/nymtech/nym-validator/logger"
	"github.com/nymtech/nym-validator/server/storage"
	tmclient "github.com/nymtech/nym-validator/tendermint/client"
	"github.com/nymtech/nym-validator/worker"
	atypes "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"gopkg.in/op/go-logging.v1"
)

const (
	maxInterval = time.Second * 30
)

// Monitor represents the Blockchain monitor
type Monitor struct {
	sync.Mutex
	worker.Worker
	store                      *storage.Database
	tmClient                   *tmclient.Client
	latestConsecutiveProcessed int64              // everything up to that point (including it) is already stored on disk
	processedBlocks            map[int64]struct{} // think of it as a set rather than a hashmap
	unprocessedBlocks          map[int64]*block
	log                        *logging.Logger
}

type block struct {
	height         int64
	beingProcessed bool
	Txs            []*tx
}

type tx struct {
	DeliverResult atypes.ResponseDeliverTx
}

func (m *Monitor) PrintProcessingState() {
	m.log.Info("##################\nUNPROCESSED BLOCKS:")

	// for debug purposes, sort them
	heights := make([]int64, 0, len(m.unprocessedBlocks))
	for h := range m.unprocessedBlocks {
		heights = append(heights, h)
	}
	sortkeys.Int64s(heights)
	m.log.Debugf("Height: %d", heights)

	m.log.Info("PROCESSED, BUT NOT COMMITTED BLOCKS:")
	heights = make([]int64, 0, len(m.processedBlocks))
	for h := range m.processedBlocks {
		heights = append(heights, h)
	}
	sortkeys.Int64s(heights)
	m.log.Debugf("Height: %d", heights)
	m.log.Debug("##################")
}

// FinalizeHeight gets called when all txs from a particular block are processed.
// processing is done by outside package; it's implementation specific, for example validator is looking
// for issuance transactions, verifier for verification requests, etc.
func (m *Monitor) FinalizeHeight(height int64) {
	m.log.Debugf("Finalizing height %v", height)
	if m.log.IsEnabledFor(logging.DEBUG) {
		m.PrintProcessingState()
	}
	m.Lock()
	defer m.Unlock()
	if height == m.latestConsecutiveProcessed+1 {
		m.latestConsecutiveProcessed = height
		for i := height + 1; ; i++ {
			if _, ok := m.processedBlocks[i]; ok {
				m.log.Debugf("Also finalizing %d", i)
				m.latestConsecutiveProcessed = i
				delete(m.processedBlocks, i)
			} else {
				m.log.Debugf("%d is not in processed blocks", i)
				break
			}
		}
		m.store.FinalizeHeight(m.latestConsecutiveProcessed)
	} else {
		m.processedBlocks[height] = struct{}{}
	}
	delete(m.unprocessedBlocks, height)
}

func (m *Monitor) getFinalisedBlock(height *int64) (*ctypes.ResultBlockResults, error) {
	res, err := m.tmClient.BlockResults(height)
	if err != nil {
		if height == nil {
			return nil, fmt.Errorf("could not obtain results for the most recent height")
		}
		return nil, fmt.Errorf("could not obtain results for height: %v", height)
	}
	return res, nil
}

// GetLowestFullUnprocessedBlock returns block on lowest height that is currently not being processed.
//nolint: golint
func (m *Monitor) GetLowestUnprocessedBlock() (int64, *block) {
	m.Lock()
	defer m.Unlock()

	i := m.latestConsecutiveProcessed + 1
	for {
		// in the current design there shouldn't be any gaps, so if we get to non-existent block
		// it means Tendermint hasn't gotten there itself
		b, ok := m.unprocessedBlocks[i]
		if !ok {
			return -1, nil
		}
		if !b.beingProcessed {
			b.beingProcessed = true
			return b.height, b
		}
		i++
	}
}

func (m *Monitor) startNewBlock(blockResults *ctypes.ResultBlockResults) error {
	m.log.Infof("Starting block at height %d", blockResults.Height)
	if len(blockResults.Results.DeliverTx) > 0 {
		m.log.Notice("Actual transactions present")
		for _, tx := range blockResults.Results.DeliverTx {
			if tx != nil && tx.Events != nil && len(tx.Events) > 0 {
				m.log.Notice("This block should have useful transactions!")
			}
		}
	}

	b := &block{
		height:         blockResults.Height,
		beingProcessed: false,
		Txs:            make([]*tx, 0, len(blockResults.Results.DeliverTx)),
	}

	for _, txRes := range blockResults.Results.DeliverTx {
		if txRes != nil {
			b.Txs = append(b.Txs, &tx{
				DeliverResult: *txRes,
			})
		}

	}

	m.Lock()
	defer m.Unlock()
	m.unprocessedBlocks[blockResults.Height] = b

	return nil
}

func (m *Monitor) catchupBlocks(startingHeight int64, latestBlock *ctypes.ResultBlockResults) (int64, error) {
	latestBlockNumber := latestBlock.Height
	if latestBlockNumber > startingHeight {
		// if latest block is starting + 1, reuse what we already have
		if latestBlockNumber == startingHeight+1 {
			if err := m.startNewBlock(latestBlock); err != nil {
				return startingHeight, fmt.Errorf("failed to start block %d: %s", latestBlockNumber, err)
			}
			return latestBlockNumber, nil
		} else {
			// get everything from starting+1 to latest
			for i := startingHeight + 1; i < latestBlockNumber; i++ {
				nextBlockRes, err := m.getFinalisedBlock(&i)
				if err != nil {
					return startingHeight, fmt.Errorf("could not obtain block %d: %v", i, err)
				}
				if err := m.startNewBlock(nextBlockRes); err != nil {
					return startingHeight, fmt.Errorf("failed to start block %d: %s", i, err)
				}
			}
			if err := m.startNewBlock(latestBlock); err != nil {
				return startingHeight, fmt.Errorf("failed to start block %d: %s", latestBlockNumber, err)
			}
			return latestBlockNumber, nil
		}
	}
	return startingHeight, nil
}

// for now assume we receive all subscription events and nodes never go down
func (m *Monitor) worker() {
	// TODO: make ticker value configurable in config file
	heartbeat := time.NewTicker(2000 * time.Millisecond)

	// our current block is what we already have seen and stored
	currentBlockNum := m.store.GetHighest()

	for {
		select {
		case <-m.HaltCh():
			return
		case <-heartbeat.C:
			latestBlock, err := m.getFinalisedBlock(nil)
			if err != nil {
				m.log.Errorf("could not get the latest block: %v", err)
				continue
			}
			latestBlockNumber := latestBlock.Height

			m.log.Debugf("most recent: %d", latestBlockNumber)
			if latestBlockNumber <= 0 {
				m.log.Warning("Failed to obtain latest block number")
				continue
			}

			updatedCurrentBlock, err := m.catchupBlocks(currentBlockNum, latestBlock)
			if err != nil {
				m.log.Errorf("failed to catchup: %v", err)
				continue
			}

			if updatedCurrentBlock <= currentBlockNum {
				m.log.Debugf("Latest block seems to be identical or smaller than current: %d / %d",
					latestBlockNumber,
					currentBlockNum,
				)
			} else {
				m.PrintProcessingState()
			}
			currentBlockNum = updatedCurrentBlock
		}
	}
}

// Halt stops the monitor
func (m *Monitor) Halt() {
	m.log.Debugf("Halting the monitor")
	m.Worker.Halt()
}

// New creates a new monitor.
func New(l *logger.Logger, tmClient *tmclient.Client, store *storage.Database, id string,
) (*Monitor, error) {
	// read db with current state etc
	log := l.GetLogger("Monitor " + id)

	monitor := &Monitor{
		tmClient:                   tmClient,
		store:                      store,
		log:                        log,
		unprocessedBlocks:          make(map[int64]*block),
		processedBlocks:            make(map[int64]struct{}),
		latestConsecutiveProcessed: store.GetHighest(),
	}

	monitor.Go(monitor.worker)
	return monitor, nil
}
