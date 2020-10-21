// Copyright 2020 Nym Technologies SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package presence

import (
	"github.com/nymtech/nym/validator/nym/directory/models"
	"github.com/nymtech/nym/validator/nym/directory/presence/fixtures"
	"github.com/nymtech/nym/validator/nym/directory/presence/mocks"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("presence.Service", func() {
	var mockDb *mocks.IDb
	var serv *Service

	BeforeEach(func() {
		mockDb = &mocks.IDb{}
		serv = NewService(mockDb)
	})

	Describe("Adding mix registration info", func() {
		It("creates new registered mix with empty reputation and zero timestamp", func() {
			info := fixtures.GoodMixRegistrationInfo()
			registeredMix := models.RegisteredMix{
				MixRegistrationInfo: info,
			}

			mockDb.On("AddMix", registeredMix)
			serv.RegisterMix(info)
			mockDb.AssertCalled(GinkgoT(), "AddMix", registeredMix)
		})
	})

	Describe("Adding gateway registration info", func() {
		It("creates new registered gateway with empty reputation and zero timestamp", func() {
			info := fixtures.GoodGatewayRegistrationInfo()
			registeredGateway := models.RegisteredGateway{
				GatewayRegistrationInfo: info,
			}

			mockDb.On("AddGateway", registeredGateway)
			serv.RegisterGateway(info)
			mockDb.AssertCalled(GinkgoT(), "AddGateway", registeredGateway)
		})
	})

	Describe("Unregistering node", func() {
		Context("With given identity when it exists", func() {
			It("Calls internal database with correct arguments", func() {
				nodeID := "foomp"
				mockDb.On("RemoveNode", nodeID).Return(true)

				assert.True(GinkgoT(), serv.UnregisterNode(nodeID))
				mockDb.AssertCalled(GinkgoT(), "RemoveNode", nodeID)
			})
		})

		Context("With given identity when it doesn't exists", func() {
			It("Calls internal database with correct arguments", func() {
				nodeID := "foomp"
				mockDb.On("RemoveNode", nodeID).Return(false)

				assert.False(GinkgoT(), serv.UnregisterNode(nodeID))
				mockDb.AssertCalled(GinkgoT(), "RemoveNode", nodeID)
			})
		})
	})

	Describe("Setting reputation of a node", func() {
		Context("With given identity when it exists", func() {
			It("Calls internal database with correct arguments", func() {
				nodeID := "foomp"
				newRep := int64(42)
				mockDb.On("SetReputation", nodeID, newRep).Return(true)

				assert.True(GinkgoT(), serv.SetReputation(nodeID, newRep))
				mockDb.AssertCalled(GinkgoT(), "SetReputation", nodeID, newRep)
			})
		})

		Context("With given identity when it doesn't exists", func() {
			It("Calls internal database with correct arguments", func() {
				nodeID := "foomp"
				newRep := int64(42)
				mockDb.On("SetReputation", nodeID, newRep).Return(false)

				assert.False(GinkgoT(), serv.SetReputation(nodeID, newRep))
				mockDb.AssertCalled(GinkgoT(), "SetReputation", nodeID, newRep)
			})
		})
	})

	Describe("Getting topology", func() {
		It("Returns all mixnodes and gateways stored in database", func() {
			mix1 := fixtures.GoodRegisteredMix()
			mix2 := fixtures.GoodRegisteredMix()
			mix2.IdentityKey = "aaa"

			gate1 := fixtures.GoodRegisteredGateway()
			gate2 := fixtures.GoodRegisteredGateway()
			gate2.IdentityKey = "bbb"

			expectedTopology := models.Topology{
				MixNodes: []models.RegisteredMix{mix1, mix2},
				Gateways: []models.RegisteredGateway{gate1, gate2},
			}

			mockDb.On("Topology").Return(expectedTopology)
			assert.Equal(GinkgoT(), expectedTopology, serv.GetTopology())
			mockDb.AssertCalled(GinkgoT(), "Topology")
		})
	})

	Describe("Getting active topology", func() {
		It("Returns all mixnodes and gateways stored in database above reputation threshold", func() {
			mix1 := fixtures.GoodRegisteredMix()
			mix1.Reputation = ReputationThreshold

			gate1 := fixtures.GoodRegisteredGateway()
			gate1.Reputation = ReputationThreshold

			expectedTopology := models.Topology{
				MixNodes: []models.RegisteredMix{mix1},
				Gateways: []models.RegisteredGateway{gate1},
			}

			mockDb.On("ActiveTopology", ReputationThreshold).Return(expectedTopology)
			assert.Equal(GinkgoT(), expectedTopology, serv.GetActiveTopology())
			mockDb.AssertCalled(GinkgoT(), "ActiveTopology", ReputationThreshold)
		})
	})
})
