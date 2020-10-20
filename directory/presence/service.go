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
)

type IService interface {
	RegisterMix(info models.MixRegistrationInfo)
	RegisterGateway(info models.GatewayRegistrationInfo)
	UnregisterNode(id string) bool
	SetReputation(id string, newRep int64) bool
	GetTopology() models.Topology
}

type Service struct {
	db IDb
}


func (service *Service) RegisterMix(info models.MixRegistrationInfo) {
	registeredMix := models.RegisteredMix{
		MixRegistrationInfo: info,
	}

	service.db.AddMix(registeredMix)
}

func (service *Service) RegisterGateway(info models.GatewayRegistrationInfo) {
	registeredGateway := models.RegisteredGateway{
		GatewayRegistrationInfo: info,
	}

	service.db.AddGateway(registeredGateway)
}

func (service *Service) UnregisterNode(id string) bool {
	return service.db.RemoveNode(id)
}

func (service *Service) SetReputation(id string, newRep int64) bool {
	return service.db.SetReputation(id, newRep)
}

func (service *Service) GetTopology() models.Topology {
	return service.db.Topology()
}

// NewService constructor
func NewService(db IDb) *Service {
	return &Service{
		db: db,
	}
}
