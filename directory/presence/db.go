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
	"fmt"
	"github.com/nymtech/nym/validator/nym/directory/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"log"
	"os"
	"os/user"
	"path"
)

type IDb interface {
	AddMix(mix models.RegisteredMix)
	AddGateway(gateway models.RegisteredGateway)
	RemoveNode(id string) bool
	SetReputation(id string, newRep int64) bool
	Topology() models.Topology
}

type Db struct {
	// I guess this should be combined with our mixmining DB, but for time being that's fine
	orm *gorm.DB
}

// NewDb constructor
func NewDb() *Db {
	database, err := gorm.Open(sqlite.Open(dbPath()), &gorm.Config{})
	if err != nil {
		panic("Failed to connect to orm!")
	}

	if err := database.AutoMigrate(&models.RegisteredMix{}); err != nil {
		log.Fatal(err)
	}
	if err := database.AutoMigrate(&models.RegisteredGateway{}); err != nil {
		log.Fatal(err)
	}

	d := Db{
		database,
	}
	return &d
}

func dbPath() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := path.Join(usr.HomeDir, ".nym")
	if err := os.MkdirAll(dbPath, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	db := path.Join(dbPath, "presence.db")
	fmt.Printf("db is: %s\n", db)
	return db
}

func (db *Db) AddMix(mix models.RegisteredMix) {
	db.orm.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "identity_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"mix_host", "sphinx_key", "version", "location", "layer", "registration_time"}), // TOOD: registration_time
	}).Create(&mix)
}

func (db *Db) AddGateway(gateway models.RegisteredGateway) {
	panic("implement me")
}

func (db *Db) allMixes() []models.RegisteredMix {
	var mixes []models.RegisteredMix
	if err := db.orm.Find(&mixes).Error; err != nil {
		fmt.Fprintf(os.Stderr, "failed to read mixes from the database - %v\n", err)
	}
	return mixes
}

func (db *Db) allGateways() []models.RegisteredGateway {
	var gateways []models.RegisteredGateway
	if err := db.orm.Find(&gateways).Error; err != nil {
		fmt.Fprintf(os.Stderr, "failed to read gateways from the database - %v\n", err)
	}
	return gateways
}

func (db *Db) RemoveNode(id string) bool {
	tx := db.orm.Begin()
	res := tx.Where("identity_key = ?", id).Delete(&models.RegisteredMix{})

	if res.Error != nil {
		tx.Rollback()
		return false
	}
	if res.RowsAffected > 0 {
		tx.Commit()
		return true
	}

	res = tx.Where("identity_key = ?", id).Delete(&models.RegisteredGateway{})
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

func (db *Db) Topology() models.Topology {
	// TODO: if we keep it (and I doubt it, because it will get moved onto blockchain), this
	// should be done as a single query rather than as two separate ones.
	mixes := db.allMixes()
	gateways := db.allGateways()

	return models.Topology{
		MixNodes: mixes,
		Gateways: gateways,
	}
}
