// processor.go - Blockchain monitor processor.
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

// Package processor processes data obtained by the monitor.
package processor

import (
	"bytes"
	"fmt"
	"time"

	proto "github.com/golang/protobuf/proto"
	"github.com/nymtech/nym-validator/common/comm/commands"
	monitor "github.com/nymtech/nym-validator/common/tendermintmonitor"
	coconut "github.com/nymtech/nym-validator/crypto/coconut/scheme"
	"github.com/nymtech/nym-validator/logger"
	"github.com/nymtech/nym-validator/server/issuer/utils"
	"github.com/nymtech/nym-validator/server/storage"
	"github.com/nymtech/nym-validator/tendermint/nymabci/code"
	tmconst "github.com/nymtech/nym-validator/tendermint/nymabci/constants"
	"github.com/nymtech/nym-validator/worker"
	"gopkg.in/op/go-logging.v1"
)

const (
	backoffDuration = time.Second * 10
)

// Processor defines struct containing all data required to sign requests committed on the blockchain.
type Processor struct {
	worker.Worker
	monitor    *monitor.Monitor
	store      *storage.Database
	incomingCh chan<- *commands.CommandRequest

	log *logging.Logger
	id  int
}

func (p *Processor) worker() {
	for {
		// first check if haltCh was closed to halt if needed

		select {
		case <-p.HaltCh():
			p.log.Debug("Returning from initial select")
			return
		default:
			p.log.Debug("Default")
		}

		height, nextBlock := p.monitor.GetLowestUnprocessedBlock()
		if nextBlock == nil || height <= 0 {
			p.log.Debugf("No blocks to process at height: %v", height)
			select {
			case <-p.HaltCh():
				p.log.Debug("Returning from backoff select")
				return
			case <-time.After(backoffDuration):
				p.log.Debug("time after")
			}
			continue
		}

		p.log.Debugf("Processing block at height: %v", height)

		for i, txRes := range nextBlock.Txs {
			tx := txRes.DeliverResult
			if tx.Code != code.OK || len(tx.Events) == 0 ||
				!bytes.HasPrefix(tx.Events[0].Attributes[0].Key, tmconst.CredentialRequestKeyPrefix) {
				p.log.Infof("Tx %v at height %v is not sign request", i, height)
				continue
			}

			blindSignMaterials := &coconut.ProtoBlindSignMaterials{}

			err := proto.Unmarshal(tx.Events[0].Attributes[0].Value, blindSignMaterials)
			if err != nil {
				p.log.Errorf("Error while unmarshalling tags: %v", err)
				continue
			}

			cmd := &commands.BlindSignRequest{
				Lambda: blindSignMaterials.Lambda,
				EgPub:  blindSignMaterials.EgPub,
				PubM:   blindSignMaterials.PubM,
			}

			// just reuse existing processing pipeline
			resCh := make(chan *commands.Response, 1)
			cmdReq := commands.NewCommandRequest(cmd, resCh)

			p.incomingCh <- cmdReq
			res := <-resCh

			if res == nil || res.Data == nil {
				p.log.Errorf("Failed to sign request at index: %v on height %v", i, height)
				continue
			}
			p.log.Debugf("Signed tx %v on height %v", i, height)

			issuedCred := res.Data.(utils.IssuedSignature)

			p.store.StoreIssuedSignature(height, blindSignMaterials.EgPub.Gamma, issuedCred)
			p.log.Debugf("Stored sig for tx %v on height %v", i, height)
		}
		p.monitor.FinalizeHeight(height)
	}
}

func (p *Processor) Halt() {
	p.log.Noticef("Halting Processor %v", p.id)
	p.Worker.Halt()
}

func New(inCh chan<- *commands.CommandRequest,
	monitor *monitor.Monitor,
	l *logger.Logger,
	id int,
	store *storage.Database,
) (*Processor, error) {

	p := &Processor{
		monitor:    monitor,
		store:      store,
		incomingCh: inCh,
		log:        l.GetLogger(fmt.Sprintf("Processor:%v", id)),
		id:         id,
	}

	p.Go(p.worker)
	return p, nil
}
