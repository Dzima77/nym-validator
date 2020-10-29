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

package mixmining

import (
	"github.com/BorisBorshevsky/timemock"
	"github.com/nymtech/nym/validator/nym/directory/mixmining/fixtures"
	"github.com/nymtech/nym/validator/nym/directory/mixmining/mocks"
	"github.com/nymtech/nym/validator/nym/directory/models"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

// Some fixtures data to dry up tests a bit

// A slice of IPv4 mix statuses with 2 ups and 1 down during the past day
func twoUpOneDown() []models.PersistedMixStatus {
	db := []models.PersistedMixStatus{}
	var status = persistedStatus()

	booltrue := true
	status.PubKey = "key1"
	status.IPVersion = "4"
	status.Up = &booltrue

	status.Timestamp = minutesAgo(5)
	db = append(db, status)

	status.Timestamp = minutesAgo(10)
	db = append(db, status)

	boolfalse := false
	status.Timestamp = minutesAgo(15)
	status.Up = &boolfalse
	db = append(db, status)

	return db
}

func persistedStatus() models.PersistedMixStatus {
	mixStatus := status()
	persisted := models.PersistedMixStatus{
		MixStatus: mixStatus,
		Timestamp: Now(),
	}
	return persisted
}

func status() models.MixStatus {
	boolfalse := false
	return models.MixStatus{
		PubKey:    "key1",
		IPVersion: "4",
		Up:        &boolfalse,
	}
}

func statusUp(key string, ipversion string) models.MixStatus {
	booltrue := true
	return models.MixStatus{
		PubKey:    key,
		IPVersion: ipversion,
		Up:        &booltrue,
	}
}

func statusDown(key string, ipversion string) models.MixStatus {
	boolfalse := false
	return models.MixStatus{
		PubKey:    key,
		IPVersion: ipversion,
		Up:        &boolfalse,
	}
}

func persistedStatusFrom(mixStatus models.MixStatus) models.PersistedMixStatus {
	persisted := models.PersistedMixStatus{
		MixStatus: mixStatus,
		Timestamp: Now(),
	}
	return persisted
}

// A version of now with a frozen shared clock so we can have determinate time-based tests
func Now() int64 {
	now := timemock.Now()
	timemock.Freeze(now) //time is frozen
	nanos := now.UnixNano()
	return nanos
}

var _ = Describe("mixmining.Service", func() {
	var mockDb mocks.IDb
	var status1 models.MixStatus
	var status2 models.MixStatus
	var persisted1 models.PersistedMixStatus
	var persisted2 models.PersistedMixStatus

	var serv Service

	boolfalse := false
	booltrue := true

	status1 = models.MixStatus{
		PubKey:    "key1",
		IPVersion: "4",
		Up:        &boolfalse,
	}

	persisted1 = models.PersistedMixStatus{
		MixStatus: status1,
		Timestamp: Now(),
	}

	status2 = models.MixStatus{
		PubKey:    "key2",
		IPVersion: "6",
		Up:        &booltrue,
	}

	persisted2 = models.PersistedMixStatus{
		MixStatus: status2,
		Timestamp: Now(),
	}

	downer := persisted1
	downer.MixStatus.Up = &boolfalse

	upper := persisted1
	upper.MixStatus.Up = &booltrue

	persistedList := []models.PersistedMixStatus{persisted1, persisted2}
	emptyList := []models.PersistedMixStatus{}

	BeforeEach(func() {
		mockDb = *new(mocks.IDb)
		serv = *NewService(&mockDb)
	})

	Describe("Adding a mix status and creating a new summary report for a node", func() {
		Context("when no statuses have yet been saved", func() {
			It("should add a PersistedMixStatus to the db and save the new report", func() {

				mockDb.On("AddMixStatus", persisted1)

				serv.CreateMixStatus(status1)
				mockDb.AssertCalled(GinkgoT(), "AddMixStatus", persisted1)
			})
		})
	})
	Describe("Listing mix statuses", func() {
		Context("when receiving a list request", func() {
			It("should call to the Db", func() {
				mockDb.On("ListMixStatus", persisted1.PubKey, 1000).Return(persistedList)

				result := serv.ListMixStatus(persisted1.PubKey)

				mockDb.AssertCalled(GinkgoT(), "ListMixStatus", persisted1.PubKey, 1000)
				assert.Equal(GinkgoT(), persistedList[0].MixStatus.PubKey, result[0].MixStatus.PubKey)
				assert.Equal(GinkgoT(), persistedList[1].MixStatus.PubKey, result[1].MixStatus.PubKey)
			})
		})
	})

	Describe("Calculating uptime", func() {
		Context("when no statuses exist yet", func() {
			It("should return 0", func() {
				mockDb.On("ListMixStatusDateRange", "key1", "4", daysAgo(30), now()).Return(emptyList)

				uptime := serv.CalculateUptime(persisted1.PubKey, persisted1.IPVersion, daysAgo(30))
				assert.Equal(GinkgoT(), 0, uptime)
			})

		})
		Context("when 2 ups and 1 down exist in the given time period", func() {
			It("should return 66", func() {
				mockDb.On("ListMixStatusDateRange", "key1", "4", daysAgo(1), now()).Return(twoUpOneDown())

				uptime := serv.CalculateUptime("key1", "4", daysAgo(1))
				expected := 66 // percent
				assert.Equal(GinkgoT(), expected, uptime)
			})
		})
	})

	Describe("Saving a mix status report", func() {
		BeforeEach(func() {
			mockDb = *new(mocks.IDb)
			serv = *NewService(&mockDb)
		})
		Context("when 1 down status exists", func() {
			BeforeEach(func() {
				oneDown := []models.PersistedMixStatus{downer}
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, minutesAgo(5), now()).Return(oneDown)
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, minutesAgo(60), now()).Return(oneDown)
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(1), now()).Return(oneDown)
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(7), now()).Return(oneDown)
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(30), now()).Return(oneDown)
			})
			Context("this one *must be* a downer, so calculate using it", func() {
				BeforeEach(func() {
					mockDb.On("LoadReport", downer.PubKey).Return(models.MixStatusReport{}) // TODO: Mockery isn't happy returning an untyped nil, so I've had to sub in a blank `models.MixStatusReport{}`. It will actually return a nil.
					expectedSave := models.MixStatusReport{
						PubKey:           downer.PubKey,
						MostRecentIPV4:   false,
						Last5MinutesIPV4: 0,
						LastHourIPV4:     0,
						LastDayIPV4:      0,
						LastWeekIPV4:     0,
						LastMonthIPV4:    0,
						MostRecentIPV6:   false,
						Last5MinutesIPV6: 0,
						LastHourIPV6:     0,
						LastDayIPV6:      0,
						LastWeekIPV6:     0,
						LastMonthIPV6:    0,
					}
					mockDb.On("UpdateReputation", downer.PubKey, ReportFailureReputationDecrease).Return(true)
					mockDb.On("SaveMixStatusReport", expectedSave)
				})
				It("should save the initial report, all statuses will be set to down", func() {
					result := serv.SaveStatusReport(downer)
					assert.Equal(GinkgoT(), 0, result.Last5MinutesIPV4)
					assert.Equal(GinkgoT(), 0, result.LastHourIPV4)
					assert.Equal(GinkgoT(), 0, result.LastDayIPV4)
					assert.Equal(GinkgoT(), 0, result.LastWeekIPV4)
					assert.Equal(GinkgoT(), 0, result.LastMonthIPV4)
					mockDb.AssertExpectations(GinkgoT())
				})
			})

		})
		Context("when 1 up status exists", func() {
			BeforeEach(func() {
				oneUp := []models.PersistedMixStatus{upper}
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, minutesAgo(5), now()).Return(oneUp)
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, minutesAgo(60), now()).Return(oneUp)
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(1), now()).Return(oneUp)
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(7), now()).Return(oneUp)
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(30), now()).Return(oneUp)
			})
			Context("this one *must be* an upper, so calculate using it", func() {
				BeforeEach(func() {
					oneDown := []models.PersistedMixStatus{downer}
					mockDb.On("ListMixStatusDateRange", upper.PubKey, upper.IPVersion, minutesAgo(5), now()).Return(oneDown)
					mockDb.On("ListMixStatusDateRange", upper.PubKey, upper.IPVersion, minutesAgo(60), now()).Return(oneDown)
					mockDb.On("ListMixStatusDateRange", upper.PubKey, upper.IPVersion, daysAgo(1), now()).Return(oneDown)
					mockDb.On("ListMixStatusDateRange", upper.PubKey, upper.IPVersion, daysAgo(7), now()).Return(oneDown)
					mockDb.On("ListMixStatusDateRange", upper.PubKey, upper.IPVersion, daysAgo(30), now()).Return(oneDown)
					mockDb.On("LoadReport", upper.PubKey).Return(models.MixStatusReport{}) // TODO: Mockery isn't happy returning an untyped nil, so I've had to sub in a blank `models.MixStatusReport{}`. It will actually return a nil.
					expectedSave := models.MixStatusReport{
						PubKey:           upper.PubKey,
						MostRecentIPV4:   true,
						Last5MinutesIPV4: 100,
						LastHourIPV4:     100,
						LastDayIPV4:      100,
						LastWeekIPV4:     100,
						LastMonthIPV4:    100,
						MostRecentIPV6:   false,
						Last5MinutesIPV6: 0,
						LastHourIPV6:     0,
						LastDayIPV6:      0,
						LastWeekIPV6:     0,
						LastMonthIPV6:    0,
					}
					mockDb.On("UpdateReputation", upper.PubKey, ReportSuccessReputationIncrease).Return(true)
					mockDb.On("SaveMixStatusReport", expectedSave)
				})
				It("should save the initial report, all statuses will be set to up", func() {
					result := serv.SaveStatusReport(upper)
					assert.Equal(GinkgoT(), true, result.MostRecentIPV4)
					assert.Equal(GinkgoT(), 100, result.Last5MinutesIPV4)
					assert.Equal(GinkgoT(), 100, result.LastHourIPV4)
					assert.Equal(GinkgoT(), 100, result.LastDayIPV4)
					assert.Equal(GinkgoT(), 100, result.LastWeekIPV4)
					assert.Equal(GinkgoT(), 100, result.LastMonthIPV4)
					mockDb.AssertExpectations(GinkgoT())
				})
			})
		})

		Context("when 2 up statuses exist for the last 5 minutes already and we just added a down", func() {
			BeforeEach(func() {
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, minutesAgo(5), now()).Return(twoUpOneDown())
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, minutesAgo(60), now()).Return(twoUpOneDown())
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(1), now()).Return(twoUpOneDown())
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(7), now()).Return(twoUpOneDown())
				mockDb.On("ListMixStatusDateRange", downer.PubKey, downer.IPVersion, daysAgo(30), now()).Return(twoUpOneDown())
			})
			It("should save the report", func() {
				initialState := models.MixStatusReport{
					PubKey:           downer.PubKey,
					MostRecentIPV4:   true,
					Last5MinutesIPV4: 100,
					LastHourIPV4:     100,
					LastDayIPV4:      100,
					LastWeekIPV4:     100,
					LastMonthIPV4:    100,
					MostRecentIPV6:   false,
					Last5MinutesIPV6: 0,
					LastHourIPV6:     0,
					LastDayIPV6:      0,
					LastWeekIPV6:     0,
					LastMonthIPV6:    0,
				}

				expectedAfterUpdate := models.MixStatusReport{
					PubKey:           downer.PubKey,
					MostRecentIPV4:   false,
					Last5MinutesIPV4: 66,
					LastHourIPV4:     66,
					LastDayIPV4:      66,
					LastWeekIPV4:     66,
					LastMonthIPV4:    66,
					MostRecentIPV6:   false,
					Last5MinutesIPV6: 0,
					LastHourIPV6:     0,
					LastDayIPV6:      0,
					LastWeekIPV6:     0,
					LastMonthIPV6:    0,
				}
				mockDb.On("LoadReport", downer.PubKey).Return(initialState)
				mockDb.On("SaveMixStatusReport", expectedAfterUpdate)
				mockDb.On("UpdateReputation", downer.PubKey, ReportFailureReputationDecrease).Return(true)

				updatedStatus := serv.SaveStatusReport(downer)
				assert.Equal(GinkgoT(), expectedAfterUpdate, updatedStatus)
				mockDb.AssertCalled(GinkgoT(), "UpdateReputation", downer.PubKey, ReportFailureReputationDecrease)

				mockDb.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Saving batch status report", func() {
		Context("if it contains v4 and v6 up status for same node", func() {
			It("should combine them into single entry", func() {
				upv4 := persistedStatusFrom(statusDown("key1", "4"))
				upv6 := persistedStatusFrom(statusDown("key1", "6"))
				batchReport := []models.PersistedMixStatus{upv4, upv6}

				expected := models.BatchMixStatusReport{
					Report: []models.MixStatusReport{{
						PubKey:           "key1",
						MostRecentIPV4:   false,
						Last5MinutesIPV4: 0,
						LastHourIPV4:     0,
						LastDayIPV4:      0,
						LastWeekIPV4:     0,
						LastMonthIPV4:    0,
						MostRecentIPV6:   false,
						Last5MinutesIPV6: 0,
						LastHourIPV6:     0,
						LastDayIPV6:      0,
						LastWeekIPV6:     0,
						LastMonthIPV6:    0,
					}},
				}

				mockDb.On("ListMixStatusDateRange", "key1", "4", minutesAgo(5), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "4", minutesAgo(60), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "4", daysAgo(1), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "4", daysAgo(7), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "4", daysAgo(30), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "6", minutesAgo(5), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "6", minutesAgo(60), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "6", daysAgo(1), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "6", daysAgo(7), now()).Return([]models.PersistedMixStatus{})
				mockDb.On("ListMixStatusDateRange", "key1", "6", daysAgo(30), now()).Return([]models.PersistedMixStatus{})

				mockDb.On("BatchLoadReports", []string{"key1", "key1"}).Return(models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)})
				mockDb.On("SaveBatchMixStatusReport", expected)
				mockDb.On("BatchUpdateReputation", map[string]int64{"key1": 2 * ReportFailureReputationDecrease})

				updatedStatus := serv.SaveBatchStatusReport(batchReport)
				assert.Equal(GinkgoT(), 1, len(updatedStatus.Report))
				mockDb.AssertCalled(GinkgoT(), "BatchUpdateReputation", map[string]int64{"key1": 2 * ReportFailureReputationDecrease})
				mockDb.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("Getting a mix status report", func() {
		Context("When no saved report exists for a pubkey", func() {
			It("should return an empty report", func() {
				mockDb = *new(mocks.IDb)
				serv = *NewService(&mockDb)

				blank := models.MixStatusReport{}
				mockDb.On("LoadReport", "superkey").Return(blank)

				report := serv.GetStatusReport("superkey")
				assert.Equal(GinkgoT(), blank, report)
			})
		})
		Context("When a saved report exists for a pubkey", func() {
			It("should return the report", func() {
				mockDb = *new(mocks.IDb)
				serv = *NewService(&mockDb)

				perfect := models.MixStatusReport{
					PubKey:           "superkey",
					MostRecentIPV4:   true,
					Last5MinutesIPV4: 100,
					LastHourIPV4:     100,
					LastDayIPV4:      100,
					LastWeekIPV4:     100,
					LastMonthIPV4:    100,
					MostRecentIPV6:   true,
					Last5MinutesIPV6: 100,
					LastHourIPV6:     100,
					LastDayIPV6:      100,
					LastWeekIPV6:     100,
					LastMonthIPV6:    100,
				}
				mockDb.On("LoadReport", "superkey").Return(perfect)

				report := serv.GetStatusReport("superkey")
				assert.Equal(GinkgoT(), perfect, report)
			})
		})
	})
})

var _ = Describe("mixmining.registration.Service", func() {
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

			mockDb.On("RegisterMix", registeredMix)
			serv.RegisterMix(info)
			mockDb.AssertCalled(GinkgoT(), "RegisterMix", registeredMix)
		})
	})

	Describe("Adding gateway registration info", func() {
		It("creates new registered gateway with empty reputation and zero timestamp", func() {
			info := fixtures.GoodGatewayRegistrationInfo()
			registeredGateway := models.RegisteredGateway{
				GatewayRegistrationInfo: info,
			}

			mockDb.On("RegisterGateway", registeredGateway)
			serv.RegisterGateway(info)
			mockDb.AssertCalled(GinkgoT(), "RegisterGateway", registeredGateway)
		})
	})

	Describe("Unregistering node", func() {
		Context("With given identity when it exists", func() {
			It("Calls internal database with correct arguments", func() {
				nodeID := "foomp"
				mockDb.On("UnregisterNode", nodeID).Return(true)

				assert.True(GinkgoT(), serv.UnregisterNode(nodeID))
				mockDb.AssertCalled(GinkgoT(), "UnregisterNode", nodeID)
			})
		})

		Context("With given identity when it doesn't exists", func() {
			It("Calls internal database with correct arguments", func() {
				nodeID := "foomp"
				mockDb.On("UnregisterNode", nodeID).Return(false)

				assert.False(GinkgoT(), serv.UnregisterNode(nodeID))
				mockDb.AssertCalled(GinkgoT(), "UnregisterNode", nodeID)
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
