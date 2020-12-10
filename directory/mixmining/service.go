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
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"time"

	"github.com/BorisBorshevsky/timemock"
	"github.com/nymtech/nym/validator/nym/directory/models"
)

// so if you can mix ipv4 but not ipv6, your reputation will go down but not as fast as if you didn't mix at all
const ReportSuccessReputationIncrease = int64(3)
const ReportFailureReputationDecrease = int64(-2)
const ReputationThreshold = int64(100)
const TopologyCacheTTL = time.Second * 30

// Service struct
type Service struct {
	db         IDb
	cliCtx     context.CLIContext
	validators *rpc.ResultValidatorsOutput

	topology                 models.Topology
	topologyRefreshed        time.Time
	activeTopology           models.Topology
	activeTopologyRefreshed  time.Time
	removedTopology          models.Topology
	removedTopologyRefreshed time.Time
}

// IService defines the REST service interface for mixmining.
type IService interface {
	CreateMixStatus(mixStatus models.MixStatus) models.PersistedMixStatus
	ListMixStatus(pubkey string) []models.PersistedMixStatus
	SaveStatusReport(status models.PersistedMixStatus) models.MixStatusReport
	GetStatusReport(pubkey string) models.MixStatusReport

	SaveBatchStatusReport(status []models.PersistedMixStatus) models.BatchMixStatusReport
	BatchCreateMixStatus(batchMixStatus models.BatchMixStatus) []models.PersistedMixStatus
	BatchGetMixStatusReport() models.BatchMixStatusReport

	RegisterMix(info models.MixRegistrationInfo)
	RegisterGateway(info models.GatewayRegistrationInfo)
	UnregisterNode(id string) bool
	SetReputation(id string, newRep int64) bool
	GetTopology() models.Topology
	GetActiveTopology() models.Topology

	CheckForDuplicateIP(host string) bool
	MixCount() int
	GatewayCount() int
	GetRemovedTopology() models.Topology
	StartupPurge()
}

// NewService constructor
func NewService(db IDb, cliCtx context.CLIContext) *Service {
	emptyValidators := emptyValidators()
	service := &Service{
		db:                       db,
		cliCtx:                   cliCtx,
		validators:               &emptyValidators,
		topology:                 db.Topology(),
		topologyRefreshed:        timemock.Now(),
		activeTopology:           db.ActiveTopology(ReputationThreshold),
		activeTopologyRefreshed:  timemock.Now(),
		removedTopology:          db.RemovedTopology(),
		removedTopologyRefreshed: timemock.Now(),
	}

	// start validator updater in background
	go updateValidators(service)
	go lastDayReportsUpdater(service)

	return service
}

func updateValidators(service *Service) {
	ticker := time.NewTicker(time.Second * 30)

	for {
		validators, err := rpc.GetValidators(service.cliCtx, nil, 1, 100)
		if err != nil {
			fmt.Printf("failed to grab validators - %v\n", err)
		} else {
			*service.validators = validators
		}
		<-ticker.C
	}
}

func lastDayReportsUpdater(service *Service) {
	ticker := time.NewTicker(time.Minute * 10)

	for {
		<-ticker.C
		batchReport := service.updateLastDayReports()
		service.removeBrokenNodes(&batchReport)
	}

}

func (service* Service) updateLastDayReports() models.BatchMixStatusReport {
	topology := service.GetTopology()

	// right there are no reports for gateways so ignore them.
	reportKeys := make([]string, 0, len(topology.MixNodes))
	for _, mix := range topology.MixNodes {
		reportKeys = append(reportKeys, mix.IdentityKey)
	}

	batchReport := service.db.BatchLoadReports(reportKeys)
	for idx, _ := range batchReport.Report {
		report := &batchReport.Report[idx]
		lastDayUptime := service.CalculateUptime(report.PubKey, "4", daysAgo(1))
		if lastDayUptime == -1 {
			// there were no reports to calculate uptime with
			continue
		}

		report.LastDayIPV4 = lastDayUptime
		report.LastDayIPV6 = service.CalculateUptime(report.PubKey, "6", daysAgo(1))
	}

	service.db.SaveBatchMixStatusReport(batchReport)
	return batchReport
}

func (service *Service) removeBrokenNodes(batchReport *models.BatchMixStatusReport) {
	// figure out which nodes should get removed
	toRemove := service.batchShouldGetRemoved(batchReport)
	if len(toRemove) > 0 {
		service.db.BatchMoveToRemovedSet(toRemove)
	}
}

// CreateMixStatus adds a new PersistedMixStatus in the orm.
func (service *Service) CreateMixStatus(mixStatus models.MixStatus) models.PersistedMixStatus {
	persistedMixStatus := models.PersistedMixStatus{
		MixStatus: mixStatus,
		Timestamp: timemock.Now().UnixNano(),
	}
	service.db.AddMixStatus(persistedMixStatus)

	return persistedMixStatus
}

// List lists the given number mix metrics
func (service *Service) ListMixStatus(pubkey string) []models.PersistedMixStatus {
	return service.db.ListMixStatus(pubkey, 1000)
}

// GetStatusReport gets a single MixStatusReport by node public key
func (service *Service) GetStatusReport(pubkey string) models.MixStatusReport {
	return service.db.LoadReport(pubkey)
}

// BatchCreateMixStatus batch adds new multiple PersistedMixStatus in the orm.
func (service *Service) BatchCreateMixStatus(batchMixStatus models.BatchMixStatus) []models.PersistedMixStatus {
	statusList := make([]models.PersistedMixStatus, len(batchMixStatus.Status))
	for i, mixStatus := range batchMixStatus.Status {
		persistedMixStatus := models.PersistedMixStatus{
			MixStatus: mixStatus,
			Timestamp: timemock.Now().UnixNano(),
		}
		statusList[i] = persistedMixStatus
	}
	service.db.BatchAddMixStatus(statusList)

	return statusList
}

// BatchGetMixStatusReport gets BatchMixStatusReport which contain multiple MixStatusReport.
func (service *Service) BatchGetMixStatusReport() models.BatchMixStatusReport {
	return service.db.LoadNonStaleReports()
}

// SaveBatchStatusReport builds and saves a status report for multiple mixnodes simultaneously.
// Those reports can be updated once whenever we receive a new status,
// and the saved results can then be queried. This keeps us from having to build the report dynamically
// on every request at runtime.
func (service *Service) SaveBatchStatusReport(status []models.PersistedMixStatus) models.BatchMixStatusReport {
	pubkeys := make([]string, len(status))
	for i := range status {
		pubkeys[i] = status[i].PubKey
	}
	batchReport := service.db.BatchLoadReports(pubkeys)

	// that's super crude but I don't think db results are guaranteed to come in order, plus some entries might
	// not exist
	reportMap := make(map[string]int)
	reputationChangeMap := make(map[string]int64)
	for i, report := range batchReport.Report {
		reportMap[report.PubKey] = i
	}

	for _, mixStatus := range status {
		if reportIdx, ok := reportMap[mixStatus.PubKey]; ok {
			service.updateReportUpToLastHour(&batchReport.Report[reportIdx], &mixStatus)
			if *mixStatus.Up {
				reputationChangeMap[mixStatus.PubKey] += ReportSuccessReputationIncrease
			} else {
				reputationChangeMap[mixStatus.PubKey] += ReportFailureReputationDecrease
			}
		} else {
			var freshReport models.MixStatusReport
			service.updateReportUpToLastHour(&freshReport, &mixStatus)
			batchReport.Report = append(batchReport.Report, freshReport)
			reportMap[freshReport.PubKey] = len(batchReport.Report) - 1
			if *mixStatus.Up {
				reputationChangeMap[mixStatus.PubKey] = ReportSuccessReputationIncrease
			} else {
				reputationChangeMap[mixStatus.PubKey] = ReportFailureReputationDecrease
			}
		}
	}

	service.db.SaveBatchMixStatusReport(batchReport)
	service.db.BatchUpdateReputation(reputationChangeMap)

	return batchReport
}

func (service *Service) updateReportUpToLastHour(report *models.MixStatusReport, status *models.PersistedMixStatus) {
	report.PubKey = status.PubKey // crude, we do this in case it's a fresh struct returned from the db

	if status.IPVersion == "4" {
		report.MostRecentIPV4 = *status.Up
		report.Last5MinutesIPV4 = service.CalculateUptime(status.PubKey, "4", minutesAgo(5))
		report.LastHourIPV4 = service.CalculateUptime(status.PubKey, "4", minutesAgo(60))
	} else if status.IPVersion == "6" {
		report.MostRecentIPV6 = *status.Up
		report.Last5MinutesIPV6 = service.CalculateUptime(status.PubKey, "6", minutesAgo(5))
		report.LastHourIPV6 = service.CalculateUptime(status.PubKey, "6", minutesAgo(60))
	}
}

// SaveStatusReport builds and saves a status report for a mixnode. The report can be updated once
// whenever we receive a new status, and the saved result can then be queried. This keeps us from
// having to build the report dynamically on every request at runtime.
func (service *Service) SaveStatusReport(status models.PersistedMixStatus) models.MixStatusReport {
	report := service.db.LoadReport(status.PubKey)

	service.updateReportUpToLastHour(&report, &status)
	service.db.SaveMixStatusReport(report)

	if *status.Up {
		service.db.UpdateReputation(status.PubKey, ReportSuccessReputationIncrease)
		// if the status was up, there's no way the quality has decreased
	} else {
		service.db.UpdateReputation(status.PubKey, ReportFailureReputationDecrease)
		if service.shouldGetRemoved(&report) {
			service.db.MoveToRemovedSet(report.PubKey)
		}
	}

	return report
}

// shouldGetRemoved is called upon receiving mix status for this particular node. It determines whether the node is still
// eligible to be part of the main topology or should moved into 'removed set'
func (service *Service) shouldGetRemoved(report *models.MixStatusReport) bool {
	// check if last 24h ipv4 uptime is > 50%
	if report.LastDayIPV4 < 50 {
		return true
	}

	// if it ever mixed any ipv6 packet, do the same check for ipv6 uptime
	if report.LastDayIPV6 > 0 && report.LastDayIPV6 < 50 {
		return true
	}

	// TODO: does it make sense to also check reputation here? But if we do it, then each new node would get
	// removed immediately before they even get a chance to build it up

	return false
}

// batchShouldGetRemoved is called upon receiving batch mix status for the set of those particular nodes.
// It determines whether the nodes are still eligible to be part of the main topology or should moved into 'removed set'
func (service *Service) batchShouldGetRemoved(batchReport *models.BatchMixStatusReport) []string {
	broken := make([]string, 0)

	for _, report := range batchReport.Report {
		// check if last 24h ipv4 uptime is > 50%
		if report.LastDayIPV4 < 50 {
			broken = append(broken, report.PubKey)
			continue
		}

		// if it ever mixed any ipv6 packet, do the same check for ipv6 uptime
		if report.LastDayIPV6 > 0 && report.LastDayIPV6 < 50 {
			broken = append(broken, report.PubKey)
			continue
		}

		// TODO: does it make sense to also check reputation here? But if we do it, then each new node would get
		// removed immediately before they even get a chance to build it up
	}

	return broken
}

// CalculateUptime calculates percentage uptime for a given node, protocol since a specific time
func (service *Service) CalculateUptime(pubkey string, ipVersion string, since int64) int {
	statuses := service.db.ListMixStatusDateRange(pubkey, ipVersion, since, now())
	numStatuses := len(statuses)
	if numStatuses == 0 {
		return -1
	}
	up := 0
	for _, status := range statuses {
		if *status.Up {
			up = up + 1
		}
	}

	return service.calculatePercent(up, numStatuses)
}

func (service *Service) calculatePercent(num int, outOf int) int {
	return int(float32(num) / float32(outOf) * 100)
}

func (service *Service) CheckForDuplicateIP(host string) bool {
	return service.db.IpExists(host)
}

func (service *Service) RegisterMix(info models.MixRegistrationInfo) {
	registeredMix := models.RegisteredMix{
		MixRegistrationInfo: info,
	}

	service.db.RegisterMix(registeredMix)
}

func (service *Service) RegisterGateway(info models.GatewayRegistrationInfo) {
	registeredGateway := models.RegisteredGateway{
		GatewayRegistrationInfo: info,
	}

	service.db.RegisterGateway(registeredGateway)
}

func (service *Service) UnregisterNode(id string) bool {
	return service.db.UnregisterNode(id)
}

func (service *Service) SetReputation(id string, newRep int64) bool {
	return service.db.SetReputation(id, newRep)
}

func emptyValidators() rpc.ResultValidatorsOutput {
	return rpc.ResultValidatorsOutput{
		BlockHeight: 0,
		Validators:  []rpc.ValidatorOutput{},
	}
}

func (service *Service) GetTopology() models.Topology {
	now := timemock.Now()

	if now.Sub(service.topologyRefreshed) > TopologyCacheTTL {
		newTopology := service.db.Topology()
		newTopology.Validators = *service.validators
		service.topology = newTopology
		service.topologyRefreshed = now
	}

	return service.topology
}

func (service *Service) GetActiveTopology() models.Topology {
	now := timemock.Now()
	if now.Sub(service.activeTopologyRefreshed) > TopologyCacheTTL {
		newTopology := service.db.ActiveTopology(ReputationThreshold)
		newTopology.Validators = *service.validators
		service.activeTopology = newTopology
		service.activeTopologyRefreshed = now
	}

	return service.activeTopology
}

func (service *Service) MixCount() int {
	topology := service.db.Topology()
	return len(topology.MixNodes)
}

func (service *Service) GatewayCount() int {
	topology := service.db.Topology()
	return len(topology.Gateways)
}

func (service *Service) GetRemovedTopology() models.Topology {
	now := timemock.Now()
	if now.Sub(service.removedTopologyRefreshed) > TopologyCacheTTL {
		newTopology := service.db.RemovedTopology()
		service.removedTopology = newTopology
		service.removedTopologyRefreshed = now
	}

	return service.removedTopology
}

// StartupPurge moves any mixnode from the main topology into 'removed' if it is not running
// version 0.9.2. The "50%" uptime requirement does not need to be checked here as if it's
// not fulfilled, the node will be automatically moved to "removed set" on the first
// run of the network monitor
func (service *Service) StartupPurge() {
	nodesToRemove := make([]string, 0)
	topology := service.db.Topology()
	for _, mix := range topology.MixNodes {
		if mix.Version != SystemVersion {
			nodesToRemove = append(nodesToRemove, mix.IdentityKey)
		}
	}
	for _, gateway := range topology.Gateways {
		if gateway.Version != SystemVersion {
			nodesToRemove = append(nodesToRemove, gateway.IdentityKey)
		}
	}
	service.db.BatchMoveToRemovedSet(nodesToRemove)
}

func now() int64 {
	return timemock.Now().UnixNano()
}

func daysAgo(days int) int64 {
	now := timemock.Now()
	return now.Add(time.Duration(-days) * time.Hour * 24).UnixNano()
}

func minutesAgo(minutes int) int64 {
	now := timemock.Now()
	return now.Add(time.Duration(-minutes) * time.Minute).UnixNano()
}

func secondsAgo(seconds int) int64 {
	now := timemock.Now()
	return now.Add(time.Duration(-seconds) * time.Second).UnixNano()
}
