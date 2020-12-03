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
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/nymtech/nym/validator/nym/directory/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// IDb holds status information
type IDb interface {
	AddMixStatus(models.PersistedMixStatus)
	BatchAddMixStatus(status []models.PersistedMixStatus)
	ListMixStatus(pubkey string, limit int) []models.PersistedMixStatus
	ListMixStatusDateRange(pubkey string, ipVersion string, start int64, end int64) []models.PersistedMixStatus
	LoadReport(pubkey string) models.MixStatusReport
	LoadNonStaleReports() models.BatchMixStatusReport
	BatchLoadReports(pubkeys []string) models.BatchMixStatusReport
	SaveMixStatusReport(models.MixStatusReport)
	SaveBatchMixStatusReport(models.BatchMixStatusReport)

	// moved from 'presence'
	RegisterMix(mix models.RegisteredMix)
	RegisterGateway(gateway models.RegisteredGateway)
	UnregisterNode(id string) bool
	UpdateReputation(id string, repIncrease int64) bool
	BatchUpdateReputation(reputationChangeMap map[string]int64)
	SetReputation(id string, newRep int64) bool
	Topology() models.Topology
	ActiveTopology(reputationThreshold int64) models.Topology

	IpExists(ip string) bool
	RemovedTopology() models.Topology
	MoveToRemovedSet(pubkey string)
	BatchMoveToRemovedSet(pubkeys []string)
}

// Db is a hashtable that holds mixnode uptime mixmining
type Db struct {
	orm *gorm.DB
}

// NewDb constructor
func NewDb(isTest bool) *Db {
	database, err := gorm.Open(sqlite.Open(dbPath(isTest)), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to orm!")
	}

	// mix status migration
	if err := database.AutoMigrate(&models.PersistedMixStatus{}); err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(&models.MixStatusReport{}); err != nil {
		log.Fatal(err)
	}

	// registered nodes migration
	if err := database.AutoMigrate(&models.RegisteredMix{}); err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(&models.RegisteredGateway{}); err != nil {
		log.Fatal(err)
	}

	// removed nodes migration
	if err := database.AutoMigrate(&models.RemovedMix{}); err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(&models.RemovedGateway{}); err != nil {
		log.Fatal(err)
	}

	d := Db{
		database,
	}
	return &d
}

func dbPath(isTest bool) string {
	if isTest {
		db, err := ioutil.TempFile("", "test_mixmining.db")
		if err != nil {
			panic(err)
		}
		return db.Name()
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := path.Join(usr.HomeDir, ".nym")
	if err := os.MkdirAll(dbPath, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	db := path.Join(dbPath, "mixmining.db")
	fmt.Printf("db is: %s\n", db)
	return db
}

// Add saves a PersistedMixStatus
func (db *Db) AddMixStatus(status models.PersistedMixStatus) {
	db.orm.Create(status)
}

// BatchAdd saves multiple PersistedMixStatus
func (db *Db) BatchAddMixStatus(status []models.PersistedMixStatus) {
	db.orm.Create(status)
}

// List returns all models.PersistedMixStatus in the orm
func (db *Db) ListMixStatus(pubkey string, limit int) []models.PersistedMixStatus {
	var statuses []models.PersistedMixStatus
	if err := db.orm.Order("timestamp desc").Limit(limit).Where("pub_key = ?", pubkey).Find(&statuses).Error; err != nil {
		return make([]models.PersistedMixStatus, 0)
	}
	return statuses
}

// ListDateRange lists all persisted mix statuses for a node for either IPv4 or IPv6 within the specified date range
func (db *Db) ListMixStatusDateRange(pubkey string, ipVersion string, start int64, end int64) []models.PersistedMixStatus {
	var statuses []models.PersistedMixStatus
	if err := db.orm.Order("timestamp desc").Where("pub_key = ?", pubkey).Where("ip_version = ?", ipVersion).Where("timestamp >= ?", start).Where("timestamp <= ?", end).Find(&statuses).Error; err != nil {
		return make([]models.PersistedMixStatus, 0)
	}
	return statuses
}

// SaveMixStatusReport creates or updates a status summary report for a given mixnode in the database
func (db *Db) SaveMixStatusReport(report models.MixStatusReport) {
	create := db.orm.Save(report)
	if create.Error != nil {
		fmt.Printf("Mix status report creation error: %+v", create.Error)
	}
}

// SaveBatchMixStatusReport creates or updates a status summary report for multiple mixnodex in the database
func (db *Db) SaveBatchMixStatusReport(report models.BatchMixStatusReport) {
	if result := db.orm.Save(report.Report); result.Error != nil {
		fmt.Printf("Batch Mix status report save error: %+v", result.Error)
	}
}

// LoadReport retrieves a models.MixStatusReport.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) LoadReport(pubkey string) models.MixStatusReport {
	var report models.MixStatusReport

	if retrieve := db.orm.First(&report, "pub_key  = ?", pubkey); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving mix status report %+v", retrieve.Error)
		return models.MixStatusReport{}
	}
	return report
}

// LoadNonStaleReports retrieves a models.BatchMixStatusReport, such that each mixnode
// in the retrieved report must have been online for over 50% of time in the last day.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) LoadNonStaleReports() models.BatchMixStatusReport {
	var reports []models.MixStatusReport

	if retrieve := db.orm.Where("last_day_ip_v4 >= 50").Or("last_day_ip_v6 >= 50").Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving multiple mix status report %+v", retrieve.Error)
		return models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)}
	}
	return models.BatchMixStatusReport{Report: reports}
}

// BatchLoadReports retrieves a models.BatchMixStatusReport based on provided set of public keys.
// If a report isn't found, it crudely generates a new instance and returns that instead.
func (db *Db) BatchLoadReports(pubkeys []string) models.BatchMixStatusReport {
	var reports []models.MixStatusReport

	if retrieve := db.orm.Where("pub_key IN ?", pubkeys).Find(&reports); retrieve.Error != nil {
		fmt.Printf("ERROR while retrieving multiple mix status report %+v", retrieve.Error)
		return models.BatchMixStatusReport{Report: make([]models.MixStatusReport, 0)}
	}
	return models.BatchMixStatusReport{Report: reports}
}

func (db *Db) RegisterMix(mix models.RegisteredMix) {
	db.orm.Unscoped().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identity_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"mix_host", "sphinx_key", "version", "location", "layer", "registration_time", "deleted", "incentives_address"}),
	}).Create(&mix)

	// if it was ever in "removed" set, delete it
	db.orm.Unscoped().Where("identity_key = ?", mix.IdentityKey).Delete(&models.RemovedMix{})
}

func (db *Db) RegisterGateway(gateway models.RegisteredGateway) {
	db.orm.Unscoped().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identity_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"mix_host", "sphinx_key", "version", "location", "clients_host", "registration_time", "deleted", "incentives_address"}),
	}).Create(&gateway)

	// if it was ever in "removed" set, delete it
	db.orm.Unscoped().Where("identity_key = ?", gateway.IdentityKey).Delete(&models.RemovedGateway{})
}

func (db *Db) allRegisteredMixes() []models.RegisteredMix {
	var mixes []models.RegisteredMix
	if err := db.orm.Find(&mixes).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read mixes from the database - %v\n", err)
	}
	return mixes
}

func (db *Db) activeRegisteredMixes(reputationThreshold int64) []models.RegisteredMix {
	var mixes []models.RegisteredMix
	if err := db.orm.Where("reputation >= ?", reputationThreshold).Find(&mixes).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read mixes from the database - %v\n", err)
	}
	return mixes
}

func (db *Db) allRegisteredGateways() []models.RegisteredGateway {
	var gateways []models.RegisteredGateway
	if err := db.orm.Find(&gateways).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read gateways from the database - %v\n", err)
	}
	return gateways
}

func (db *Db) activeRegisteredGateways(reputationThreshold int64) []models.RegisteredGateway {
	var gateways []models.RegisteredGateway
	if err := db.orm.Where("reputation >= ?", reputationThreshold).Find(&gateways).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read gateways from the database - %v\n", err)
	}
	return gateways
}

func (db *Db) UnregisterNode(id string) bool {
	tx := db.orm.Begin()

	res := tx.Where("identity_key = ?", id).Delete(&models.RegisteredMix{})
	if res.Error != nil {
		tx.Rollback()
		return false
	}
	if res.RowsAffected > 0 {
		// now try the same for 'removed mix' - remember, all we do are soft deletes, and a removed mix
		// can only exist if there used to be an entry for 'registered mix' (don't blame me, blame gorm + sql :) )
		res = tx.Where("identity_key = ?", id).Delete(&models.RemovedMix{})
		if res.Error != nil {
			tx.Rollback()
			return false
		}

		tx.Commit()
		return true
	}

	res = tx.Where("identity_key = ?", id).Delete(&models.RegisteredGateway{})
	if res.Error != nil {
		tx.Rollback()
		return false
	}
	if res.RowsAffected > 0 {
		res = tx.Where("identity_key = ?", id).Delete(&models.RemovedGateway{})
		if res.Error != nil {
			tx.Rollback()
			return false
		}
		
		tx.Commit()
		return true
	}

	return false
}

func (db *Db) SetReputation(id string, newRep int64) bool {
	tx := db.orm.Begin()
	res := tx.Model(&models.RegisteredMix{}).Where("identity_key = ?", id).Update("reputation", newRep)

	if res.Error != nil {
		tx.Rollback()
		return false
	}
	if res.RowsAffected > 0 {
		tx.Commit()
		return true
	}

	res = tx.Model(&models.RegisteredGateway{}).Where("identity_key = ?", id).Update("reputation", newRep)
	if res.Error != nil {
		tx.Rollback()
		return false
	}

	tx.Commit()

	if res.RowsAffected > 0 {
		return true
	} else {
		return false
	}
}

func (db *Db) BatchUpdateReputation(reputationChangeMap map[string]int64) {
	// this is based on the idea (which might be totally wrong) that doing
	// it this way will only result in single io action on db
	// I don't know enough SQL to do it as a single query
	tx := db.orm.Begin()
	for id, repChange := range reputationChangeMap {
		// ensuring reputation will not go negative (haha, this can probably be solved in a simpler way inside SQL, but hey, it works)
		if repChange < 0 {
			res := tx.Model(&models.RegisteredMix{}).Where("identity_key = ? AND reputation >= ?", id, -repChange).Update("reputation", gorm.Expr("reputation + ?", repChange))
			// TODO: rollback on fail here??
			if res.RowsAffected == 0 {
				tx.Model(&models.RegisteredGateway{}).Where("identity_key = ? AND reputation >= ?", id, -repChange).Update("reputation", gorm.Expr("reputation + ?", repChange))
			}
		} else {
			res := tx.Model(&models.RegisteredMix{}).Where("identity_key = ?", id).Update("reputation", gorm.Expr("reputation + ?", repChange))
			// TODO: rollback on fail here??
			if res.RowsAffected == 0 {
				tx.Model(&models.RegisteredGateway{}).Where("identity_key = ?", id).Update("reputation", gorm.Expr("reputation + ?", repChange))
			}
		}
	}

	tx.Commit()
}

func (db *Db) UpdateReputation(id string, repIncrease int64) bool {
	tx := db.orm.Begin()
	// ensuring reputation will not go negative (haha, this can probably be solved in a simpler way inside SQL, but hey, it works)
	if repIncrease < 0 {
		res := tx.Model(&models.RegisteredMix{}).Where("identity_key = ? AND reputation >= ?", id, -repIncrease).Update("reputation", gorm.Expr("reputation + ?", repIncrease))

		if res.Error != nil {
			tx.Rollback()
			return false
		}
		if res.RowsAffected > 0 {
			tx.Commit()
			return true
		}

		res = tx.Model(&models.RegisteredGateway{}).Where("identity_key = ? AND reputation >= ?", id, -repIncrease).Update("reputation", gorm.Expr("reputation + ?", repIncrease))
		if res.Error != nil {
			tx.Rollback()
			return false
		}

		tx.Commit()

		if res.RowsAffected > 0 {
			return true
		} else {
			return false
		}
	} else {
		res := tx.Model(&models.RegisteredMix{}).Where("identity_key = ?", id).Update("reputation", gorm.Expr("reputation + ?", repIncrease))

		if res.Error != nil {
			tx.Rollback()
			return false
		}
		if res.RowsAffected > 0 {
			tx.Commit()
			return true
		}

		res = tx.Model(&models.RegisteredGateway{}).Where("identity_key = ?", id).Update("reputation", gorm.Expr("reputation + ?", repIncrease))
		if res.Error != nil {
			tx.Rollback()
			return false
		}

		tx.Commit()

		if res.RowsAffected > 0 {
			return true
		} else {
			return false
		}
	}

}

func (db *Db) Topology() models.Topology {
	// TODO: if we keep it (and I doubt it, because it will get moved onto blockchain), this
	// should be done as a single query rather than as two separate ones.
	mixes := db.allRegisteredMixes()
	gateways := db.allRegisteredGateways()

	return models.Topology{
		MixNodes: mixes,
		Gateways: gateways,
	}
}

func (db *Db) ActiveTopology(reputationThreshold int64) models.Topology {
	// TODO: if we keep it (and I doubt it, because it will get moved onto blockchain), this
	// should be done as a single query rather than as two separate ones.
	mixes := db.activeRegisteredMixes(reputationThreshold)
	gateways := db.activeRegisteredGateways(reputationThreshold)

	return models.Topology{
		MixNodes: mixes,
		Gateways: gateways,
	}
}

func (db *Db) IpExists(ip string) bool {
	ip, _, err := net.SplitHostPort(ip)
	if err != nil {
		// I guess we got a domain name?
		split := strings.Split(ip, ":")
		chunks := len(split)
		if chunks != 2 {
			// no idea what we got here
			// return true to disallow registration for this address
			return true
		}

		ip = split[0]
	}

	if db.orm.Where("mix_host LIKE ?", "%"+ip+"%").Find(&models.RegisteredMix{}).RowsAffected > 0 {
		return true
	} else if db.orm.Where("mix_host LIKE ? OR clients_host LIKE ?", "%"+ip+"%", "%"+ip+"%").Find(&models.RegisteredGateway{}).RowsAffected > 0 {
		return true
	} else {
		return false
	}
}

func (db *Db) addRemovedMix(mix models.RemovedMix) {
	db.orm.Unscoped().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identity_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"mix_host", "sphinx_key", "version", "location", "layer", "registration_time", "deleted", "incentives_address"}),
	}).Create(&mix)
}

func (db *Db) addRemovedGateway(gateway models.RemovedGateway) {
	db.orm.Unscoped().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identity_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"mix_host", "sphinx_key", "version", "location", "clients_host", "registration_time", "deleted", "incentives_address"}),
	}).Create(&gateway)
}

func (db *Db) allRemovedMixes() []models.RemovedMix {
	var mixes []models.RemovedMix
	if err := db.orm.Find(&mixes).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read mixes from the database - %v\n", err)
	}
	return mixes
}

func (db *Db) allRemovedGateways() []models.RemovedGateway {
	var gateways []models.RemovedGateway
	if err := db.orm.Find(&gateways).Error; err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to read gateways from the database - %v\n", err)
	}
	return gateways
}

func (db *Db) MoveToRemovedSet(pubkey string) {
	mix := models.RegisteredMix{}
	res := db.orm.Where("identity_key = ?", pubkey).Find(&mix)
	if res.Error != nil {
		return
	}
	if res.RowsAffected > 0 {
		// add to removed set
		db.addRemovedMix(models.RemovedMix{RegisteredMix: mix})
		// and remove/unregister it from the 'good' set
		db.orm.Where("identity_key = ?", pubkey).Delete(&models.RegisteredMix{})
		return
	}

	gateway := models.RegisteredGateway{}
	res = db.orm.Where("identity_key = ?", pubkey).Find(&gateway)
	if res.Error != nil {
		return
	}
	if res.RowsAffected > 0 {
		// add to removed set
		db.addRemovedGateway(models.RemovedGateway{RegisteredGateway: gateway})
		// and remove/unregister it from the 'good' set
		db.orm.Where("identity_key = ?", pubkey).Delete(&models.RemovedGateway{})
		return
	}
}

func (db *Db) BatchMoveToRemovedSet(pubkeys []string) {
	// I honestly doubt we will ever remove a lot of nodes in a single batch report, so I think
	// not doing it tx way is fine
	for _, pubkey := range pubkeys {
		db.MoveToRemovedSet(pubkey)
	}

}

// DeletedTopology returns lists of all gateways and mixnodes that are now in the 'removed' set.
// It is retrieved as `Topology` so there would be less coding on the API client side of things.
func (db *Db) RemovedTopology() models.Topology {
	// TODO: if we keep it (and I doubt it, because it will get moved onto blockchain), this
	// should be done as a single query rather than as two separate ones.
	removedMixes := db.allRemovedMixes()
	removedGateways := db.allRemovedGateways()

	mixes := make([]models.RegisteredMix, len(removedMixes))
	for i, mix := range removedMixes {
		mixes[i] = mix.RegisteredMix
	}

	gateways := make([]models.RegisteredGateway, len(removedGateways))
	for i, gateway := range removedGateways {
		gateways[i] = gateway.RegisteredGateway
	}

	return models.Topology{
		MixNodes: mixes,
		Gateways: gateways,
	}
}
