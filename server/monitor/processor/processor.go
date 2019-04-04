// processor.go - Blockchain monitor processor.
// Copyright (C) 2018  Jedrzej Stuczynski.
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

	"0xacab.org/jstuczyn/CoconutGo/common/comm/commands"
	"0xacab.org/jstuczyn/CoconutGo/crypto/coconut/scheme"
	"0xacab.org/jstuczyn/CoconutGo/logger"
	"0xacab.org/jstuczyn/CoconutGo/server/monitor"
	"0xacab.org/jstuczyn/CoconutGo/tendermint/nymabci/code"
	tmconst "0xacab.org/jstuczyn/CoconutGo/tendermint/nymabci/constants"
	"0xacab.org/jstuczyn/CoconutGo/worker"
	proto "github.com/golang/protobuf/proto"
	"gopkg.in/op/go-logging.v1"
)

type Processor struct {
	worker.Worker
	monitor    *monitor.Monitor
	incomingCh chan<- *commands.CommandRequest

	haltCh chan struct{}
	log    *logging.Logger
	id     int
}

const (
	backoffDuration = time.Second * 5
)

func (p *Processor) worker() {
	for {
		// first check if haltCh was closed to halt if needed

		select {
		case <-p.haltCh:
			return
		default:
			p.log.Debug("Default")
		}

		height, nextBlock := p.monitor.GetLowestFullUnprocessedBlock()
		if nextBlock == nil {
			p.log.Info("No blocks to process")
			time.Sleep(backoffDuration)
			continue
		}

		p.log.Warningf("Processing block at height: %v", height)
		continue

		// In principle there should be no need to use the lock here because the block shouldn't be touched anymore,
		// but better safe than sorry
		nextBlock.Lock()

		if nextBlock.NumTxs == 0 {
			p.monitor.FinalizeHeight(height)
		}
		for i, tx := range nextBlock.Txs {
			if tx.Code != code.OK || len(tx.Tags) <= 0 ||
				!bytes.HasPrefix(tx.Tags[0].Key, tmconst.CredentialRequestKeyPrefix) {
				continue
			}

			blindSignMaterials := &coconut.BlindSignMaterials{}

			err := proto.Unmarshal(tx.Tags[0].Value, blindSignMaterials)
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

			if res.ErrorStatus != commands.StatusCode_OK {
				p.log.Errorf("Failed to sign request at index: %v on height %v, err: %v", i, height, res.ErrorMessage)
			}

			blindedSig := res.Data.(*coconut.BlindedSignature)
			// TODO: writing it to storage
			// _ = blindedSig

			p.log.Warningf("Signed tx %v on height %v : %v", i, height, blindedSig.Sig1().ToString())
		}

		p.monitor.FinalizeHeight(height)
		nextBlock.Unlock()
	}
}

func (p *Processor) Halt() {
	p.log.Noticef("Halting Processor %v", p.id)
	close(p.haltCh)
	p.Worker.Halt()
	// todo
}

func New(inCh chan<- *commands.CommandRequest, monitor *monitor.Monitor, l *logger.Logger, id int) (*Processor, error) {

	p := &Processor{
		monitor:    monitor,
		incomingCh: inCh,
		haltCh:     make(chan struct{}),
		log:        l.GetLogger(fmt.Sprintf("Processor:%v", id)),
		id:         id,
	}

	p.Go(p.worker)
	return p, nil
}
