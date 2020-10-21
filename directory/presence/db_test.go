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
	"github.com/nymtech/nym/validator/nym/directory/presence/fixtures"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"time"
)

var _ = Describe("The presence db", func() {
	Describe("Constructing a NewDb", func() {
		Context("a new db", func() {
			It("should have no mixmining statuses", func() {
				db := NewDb(true)
				allNodes := db.Topology()

				assert.Len(GinkgoT(), allNodes.MixNodes, 0)
				assert.Len(GinkgoT(), allNodes.Gateways, 0)
			})
		})
	})

	Describe("Registering mix node", func() {
		Context("For the first time", func() {
			It("should add the entry, with timestamp and initial reputation, to database", func() {
				db := NewDb(true)
				all := db.allMixes()
				assert.Len(GinkgoT(), all, 0)

				mix := fixtures.GoodRegisteredMix()
				startTime := time.Now()
				db.AddMix(mix)
				endTime := time.Now()

				all = db.allMixes()
				assert.Len(GinkgoT(), all, 1)
				assert.True(GinkgoT(), all[0].RegistrationTime >= startTime.UnixNano())
				assert.True(GinkgoT(), all[0].RegistrationTime <= endTime.UnixNano())

				// this is just so the comparison is easier
				all[0].RegistrationTime = 0
				assert.Equal(GinkgoT(), mix, all[0])
			})
		})
		Context("For second time", func() {
			It("should overwrite the existing entry without making a new one", func() {
				db := NewDb(true)
				all := db.allMixes()
				assert.Len(GinkgoT(), all, 0)

				initialMix := fixtures.GoodRegisteredMix()
				updatedInitialMix := fixtures.GoodRegisteredMix()
				updatedInitialMix.Location = "New Foomplandia"
				updatedInitialMix.MixHost = "100.100.100.100:1789"

				db.AddMix(initialMix)
				all = db.allMixes()
				initRegTime := all[0].RegistrationTime
				assert.Len(GinkgoT(), all, 1)

				db.AddMix(updatedInitialMix)
				all = db.allMixes()

				assert.Len(GinkgoT(), all, 1)
				// since we 'registered' again we should get new registration time
				assert.True(GinkgoT(), all[0].RegistrationTime > initRegTime)

				// this is just so the comparison is easier
				all[0].RegistrationTime = 0
				assert.Equal(GinkgoT(), updatedInitialMix, all[0])
			})
		})

		Context("Multiple with different identity", func() {
			It("Should not overwrite each other", func() {
				db := NewDb(true)
				all := db.allMixes()
				assert.Len(GinkgoT(), all, 0)

				initialMix1 := fixtures.GoodRegisteredMix()
				initialMix2 := fixtures.GoodRegisteredMix()
				initialMix2.IdentityKey = "NewID"

				db.AddMix(initialMix1)
				db.AddMix(initialMix2)
				all = db.allMixes()
				assert.Len(GinkgoT(), all, 2)
			})
		})
	})

	Describe("Removing mix node", func() {
		Context("If it exists", func() {
			It("Should get rid of it", func() {
				db := NewDb(true)
				all := db.allMixes()
				assert.Len(GinkgoT(), all, 0)

				mix := fixtures.GoodRegisteredMix()
				db.AddMix(mix)
				wasRemoved := db.RemoveNode(mix.IdentityKey)
				assert.True(GinkgoT(), wasRemoved)

				all = db.allMixes()
				assert.Len(GinkgoT(), all, 0)
			})
		})

		Context("If it doesn't exist", func() {
			It("Shouldn't do anything", func() {
				db := NewDb(true)
				all := db.allMixes()
				assert.Len(GinkgoT(), all, 0)

				wasRemoved := db.RemoveNode("foomp")
				assert.False(GinkgoT(), wasRemoved)

				all = db.allMixes()
				assert.Len(GinkgoT(), all, 0)
			})
		})
	})

	Describe("Registering gateway", func() {
		Context("For the first time", func() {
			It("should add the entry, with timestamp and initial reputation, to database", func() {
				db := NewDb(true)
				all := db.allGateways()
				assert.Len(GinkgoT(), all, 0)

				gateway := fixtures.GoodRegisteredGateway()
				startTime := time.Now()
				db.AddGateway(gateway)
				endTime := time.Now()

				all = db.allGateways()
				assert.Len(GinkgoT(), all, 1)
				assert.True(GinkgoT(), all[0].RegistrationTime >= startTime.UnixNano())
				assert.True(GinkgoT(), all[0].RegistrationTime <= endTime.UnixNano())

				// this is just so the comparison is easier
				all[0].RegistrationTime = 0
				assert.Equal(GinkgoT(), gateway, all[0])
			})
		})
		Context("For second time", func() {
			It("should overwrite the existing entry without making a new one", func() {
				db := NewDb(true)
				all := db.allGateways()
				assert.Len(GinkgoT(), all, 0)

				initialGateway := fixtures.GoodRegisteredGateway()
				updatedInitialGateway := fixtures.GoodRegisteredGateway()
				updatedInitialGateway.Location = "New Foomplandia"
				updatedInitialGateway.MixHost = "100.100.100.100:1789"

				db.AddGateway(initialGateway)
				all = db.allGateways()
				initRegTime := all[0].RegistrationTime
				assert.Len(GinkgoT(), all, 1)

				db.AddGateway(updatedInitialGateway)
				all = db.allGateways()

				assert.Len(GinkgoT(), all, 1)
				// since we 'registered' again we should get new registration time
				assert.True(GinkgoT(), all[0].RegistrationTime > initRegTime)

				// this is just so the comparison is easier
				all[0].RegistrationTime = 0
				assert.Equal(GinkgoT(), updatedInitialGateway, all[0])
			})
		})

		Context("Multiple with different identity", func() {
			It("Should not overwrite each other", func() {
				db := NewDb(true)
				all := db.allGateways()
				assert.Len(GinkgoT(), all, 0)

				initialGateway1 := fixtures.GoodRegisteredGateway()
				initialGateway2 := fixtures.GoodRegisteredGateway()
				initialGateway2.IdentityKey = "NewID"

				db.AddGateway(initialGateway1)
				db.AddGateway(initialGateway2)
				all = db.allGateways()
				assert.Len(GinkgoT(), all, 2)
			})
		})
	})

	Describe("Removing gateway node", func() {
		Context("If it exists", func() {
			It("Should get rid of it", func() {
				db := NewDb(true)
				all := db.allGateways()
				assert.Len(GinkgoT(), all, 0)

				gateway := fixtures.GoodRegisteredGateway()
				db.AddGateway(gateway)
				wasRemoved := db.RemoveNode(gateway.IdentityKey)
				assert.True(GinkgoT(), wasRemoved)

				all = db.allGateways()
				assert.Len(GinkgoT(), all, 0)
			})
		})
	})

	Describe("Setting reputation", func() {
		Context("For existing node", func() {
			It("Sets it to defined value", func() {
				db := NewDb(true)
				all := db.allMixes()
				assert.Len(GinkgoT(), all, 0)

				mix := fixtures.GoodRegisteredMix()
				db.AddMix(mix)
				all = db.allMixes()
				assert.Equal(GinkgoT(), all[0].Reputation, int64(0))

				wasChanged := db.SetReputation(mix.IdentityKey, 42)
				assert.True(GinkgoT(), wasChanged)

				all = db.allMixes()
				assert.Equal(GinkgoT(), all[0].Reputation, int64(42))
			})
		})

		Context("For non-existent node", func() {
			It("Does nothing", func() {
				db := NewDb(true)
				all := db.allMixes()
				assert.Len(GinkgoT(), all, 0)

				wasChanged := db.SetReputation("foomp", 42)
				assert.False(GinkgoT(), wasChanged)
			})
		})
	})

	Describe("Getting topology", func() {
		Context("With no registered nodes", func() {
			It("Returns empty slices", func() {
				db := NewDb(true)
				allMix := db.allMixes()
				assert.Len(GinkgoT(), allMix, 0)

				allGate := db.allGateways()
				assert.Len(GinkgoT(), allGate, 0)

				topology := db.Topology()
				assert.Len(GinkgoT(), topology.MixNodes, 0)
				assert.Len(GinkgoT(), topology.Gateways, 0)
			})
		})
		Context("With registered nodes", func() {
			It("Returns all registered mixnodes and gateways", func() {
				db := NewDb(true)
				allMix := db.allMixes()
				assert.Len(GinkgoT(), allMix, 0)

				allGate := db.allGateways()
				assert.Len(GinkgoT(), allGate, 0)

				mix1 := fixtures.GoodRegisteredMix()
				mix2 := fixtures.GoodRegisteredMix()
				mix2.IdentityKey = "aaa"

				gate1 := fixtures.GoodRegisteredGateway()
				gate2 := fixtures.GoodRegisteredGateway()
				gate2.IdentityKey = "bbb"

				db.AddMix(mix1)
				db.AddMix(mix2)

				db.AddGateway(gate1)
				db.AddGateway(gate2)

				topology := db.Topology()
				assert.Len(GinkgoT(), topology.MixNodes, 2)
				assert.Len(GinkgoT(), topology.Gateways, 2)
			})
		})
	})

	Describe("Getting active topology", func () {
		Context("With registered nodes but below reputation threshold", func() {
			It("Returns empty slices", func() {
				db := NewDb(true)
				allMix := db.allMixes()
				assert.Len(GinkgoT(), allMix, 0)

				allGate := db.allGateways()
				assert.Len(GinkgoT(), allGate, 0)

				mix1 := fixtures.GoodRegisteredMix()
				gate1 := fixtures.GoodRegisteredGateway()

				db.AddMix(mix1)
				db.AddGateway(gate1)

				db.SetReputation(mix1.IdentityKey, ReputationThreshold - 1)
				db.SetReputation(gate1.IdentityKey, ReputationThreshold - 1)

				topology := db.ActiveTopology(ReputationThreshold)
				assert.Len(GinkgoT(), topology.MixNodes, 0)
				assert.Len(GinkgoT(), topology.Gateways, 0)
			})
		})

		Context("With registered nodes, some above reputation threshold", func() {
			It("Returns only the nodes above the reputation threshold", func() {
				db := NewDb(true)
				allMix := db.allMixes()
				assert.Len(GinkgoT(), allMix, 0)

				allGate := db.allGateways()
				assert.Len(GinkgoT(), allGate, 0)

				mix1 := fixtures.GoodRegisteredMix()
				mix2 := fixtures.GoodRegisteredMix()
				mix2.IdentityKey = "aaa"

				gate1 := fixtures.GoodRegisteredGateway()
				gate2 := fixtures.GoodRegisteredGateway()
				gate2.IdentityKey = "bbb"

				db.AddMix(mix1)
				db.AddMix(mix2)

				db.AddGateway(gate1)
				db.AddGateway(gate2)

				db.SetReputation(mix1.IdentityKey, ReputationThreshold - 1)
				db.SetReputation(gate1.IdentityKey, ReputationThreshold - 1)
				db.SetReputation(mix2.IdentityKey, ReputationThreshold)
				db.SetReputation(gate2.IdentityKey, ReputationThreshold)

				topology := db.ActiveTopology(ReputationThreshold)
				// this is just so the comparison is easier
				topology.MixNodes[0].RegistrationTime = 0
				topology.Gateways[0].RegistrationTime = 0

				assert.Equal(GinkgoT(), topology.MixNodes[0].Reputation, ReputationThreshold)
				assert.Equal(GinkgoT(), topology.Gateways[0].Reputation, ReputationThreshold)

				topology.MixNodes[0].Reputation = int64(0)
				topology.Gateways[0].Reputation = int64(0)

				assert.Equal(GinkgoT(), topology.MixNodes[0], mix2)
				assert.Equal(GinkgoT(), topology.Gateways[0], gate2)
			})
		})
	})
})
