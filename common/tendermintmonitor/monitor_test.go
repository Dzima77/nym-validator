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

package monitor

// TODO: MAKE THIS TEST WORK IN REMOTE ENVIRONMENTS
//
//import (
//	"io/ioutil"
//	"os"
//	"testing"
//	"time"
//
//	"github.com/nymtech/nym-validator/logger"
//	"github.com/nymtech/nym-validator/server/storage"
//	nymclient "github.com/nymtech/nym-validator/tendermint/client"
//	"github.com/stretchr/testify/assert"
//	cmn "github.com/tendermint/tendermint/libs/common"
//)
//
//const (
//	nodeAddress = "localhost:26657" // this particular chain currently is at block >10
//)
//
//func createValidMonitor() *Monitor {
//	tmpDir, err := ioutil.TempDir("", "monitor_test"+cmn.RandStr(6))
//	if err != nil {
//		panic(err)
//	}
//	defer os.RemoveAll(tmpDir)
//
//	disabledLog, err := logger.New("", "DEBUG", true)
//	if err != nil {
//		panic(err)
//	}
//
//	nymClient, err := nymclient.New([]string{nodeAddress}, disabledLog)
//	if err != nil {
//		panic(err)
//	}
//
//	store, err := storage.New("tmp_store", tmpDir)
//	if err != nil {
//		panic(err)
//	}
//
//	mon, err := New(disabledLog, nymClient, store, "foo")
//	if err != nil {
//		panic(err)
//	}
//	return mon
//}
//
//func TestNew(t *testing.T) {
//	validMon := createValidMonitor()
//
//	assert.NotNil(t, validMon)
//}
//
//func TestFinalizeFirstBlock(t *testing.T) {
//	// initial wait for starting blocks to appear
//	validMon := createValidMonitor()
//
//	for {
//		time.Sleep(time.Millisecond * 500)
//		if len(validMon.unprocessedBlocks) > 0 {
//			break
//		}
//	}
//
//	assert.Equal(t, int64(0), validMon.latestConsecutiveProcessed)
//
//	// pretest checks
//	_, ok := validMon.unprocessedBlocks[1]
//	assert.True(t, ok)
//	_, ok = validMon.processedBlocks[1]
//	assert.False(t, ok)
//
//	validMon.FinalizeHeight(1)
//	_, ok = validMon.unprocessedBlocks[1]
//	assert.False(t, ok)
//
//	// it's not added because if it's first, its deleted immediately
//	_, ok = validMon.processedBlocks[1]
//	assert.False(t, ok)
//
//	assert.Equal(t, int64(1), validMon.latestConsecutiveProcessed)
//
//}
//
//func TestFinalizeSingleBlock(t *testing.T) {
//	// initial wait for starting blocks to appear
//	validMon := createValidMonitor()
//
//	for {
//		time.Sleep(time.Millisecond * 500)
//		if len(validMon.unprocessedBlocks) > 2 {
//			break
//		}
//	}
//
//	assert.Equal(t, int64(0), validMon.latestConsecutiveProcessed)
//
//	// pretest checks
//	_, ok := validMon.unprocessedBlocks[1]
//	assert.True(t, ok)
//	_, ok = validMon.unprocessedBlocks[2]
//	assert.True(t, ok)
//	_, ok = validMon.unprocessedBlocks[3]
//	assert.True(t, ok)
//
//	_, ok = validMon.processedBlocks[1]
//	assert.False(t, ok)
//	_, ok = validMon.processedBlocks[2]
//	assert.False(t, ok)
//	_, ok = validMon.processedBlocks[3]
//	assert.False(t, ok)
//
//	validMon.FinalizeHeight(2)
//
//	_, ok = validMon.unprocessedBlocks[1]
//	assert.True(t, ok)
//	_, ok = validMon.unprocessedBlocks[2]
//	assert.False(t, ok)
//	_, ok = validMon.unprocessedBlocks[3]
//	assert.True(t, ok)
//
//	_, ok = validMon.processedBlocks[1]
//	assert.False(t, ok)
//	_, ok = validMon.processedBlocks[2]
//	assert.True(t, ok)
//	_, ok = validMon.processedBlocks[3]
//	assert.False(t, ok)
//
//	assert.Equal(t, int64(0), validMon.latestConsecutiveProcessed)
//
//}
//
//func TestFinalizeBlocksNotInOrder(t *testing.T) {
//	// initial wait for starting blocks to appear
//	validMon := createValidMonitor()
//
//	for {
//		time.Sleep(time.Millisecond * 500)
//		if len(validMon.unprocessedBlocks) > 2 {
//			break
//		}
//	}
//
//	assert.Equal(t, int64(0), validMon.latestConsecutiveProcessed)
//
//	// pretest checks
//	_, ok := validMon.unprocessedBlocks[1]
//	assert.True(t, ok)
//	_, ok = validMon.unprocessedBlocks[2]
//	assert.True(t, ok)
//	_, ok = validMon.unprocessedBlocks[3]
//	assert.True(t, ok)
//
//	_, ok = validMon.processedBlocks[1]
//	assert.False(t, ok)
//	_, ok = validMon.processedBlocks[2]
//	assert.False(t, ok)
//	_, ok = validMon.processedBlocks[3]
//	assert.False(t, ok)
//
//	validMon.FinalizeHeight(2)
//
//	_, ok = validMon.unprocessedBlocks[1]
//	assert.True(t, ok)
//	_, ok = validMon.unprocessedBlocks[2]
//	assert.False(t, ok)
//	_, ok = validMon.unprocessedBlocks[3]
//	assert.True(t, ok)
//
//	_, ok = validMon.processedBlocks[1]
//	assert.False(t, ok)
//	_, ok = validMon.processedBlocks[2]
//	assert.True(t, ok)
//	_, ok = validMon.processedBlocks[3]
//	assert.False(t, ok)
//
//	validMon.FinalizeHeight(1)
//
//	_, ok = validMon.unprocessedBlocks[1]
//	assert.False(t, ok)
//	_, ok = validMon.unprocessedBlocks[2]
//	assert.False(t, ok)
//	_, ok = validMon.unprocessedBlocks[3]
//	assert.True(t, ok)
//
//	_, ok = validMon.processedBlocks[1]
//	assert.False(t, ok)
//	_, ok = validMon.processedBlocks[2]
//	assert.False(t, ok)
//	_, ok = validMon.processedBlocks[3]
//	assert.False(t, ok)
//
//	assert.Equal(t, int64(2), validMon.latestConsecutiveProcessed)
//}
//
//func TestGetLowestUnprocessed(t *testing.T) {
//	// initial wait for starting blocks to appear
//	validMon := createValidMonitor()
//
//	for {
//		time.Sleep(time.Millisecond * 500)
//		if len(validMon.unprocessedBlocks) > 4 {
//			break
//		}
//	}
//
//	i, b := validMon.GetLowestUnprocessedBlock()
//
//	assert.Equal(t, int64(1), i)
//	assert.Equal(t, i, b.height)
//
//	i, b = validMon.GetLowestUnprocessedBlock()
//	assert.Equal(t, int64(2), i)
//	assert.Equal(t, i, b.height)
//
//	validMon.FinalizeHeight(1)
//	validMon.FinalizeHeight(2)
//	validMon.FinalizeHeight(3)
//
//	i, b = validMon.GetLowestUnprocessedBlock()
//	assert.Equal(t, int64(4), i)
//	assert.Equal(t, i, b.height)
//}
