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

package fixtures

import "github.com/nymtech/nym/validator/nym/directory/models"

func GoodMixRegistrationInfo() models.MixRegistrationInfo {
	return models.MixRegistrationInfo{
		NodeInfo: models.NodeInfo{
			MixHost:     "1.2.3.4:1789",
			IdentityKey: "D6YaMzLSY7mANtSQRKXsmMZpqgqiVkeiagKM4V4oFPFr",
			SphinxKey:   "51j2kyqE7iTYc8RBtn5FR5E9jp8BdqZamggSg4PYN6ie",
			Version:     "0.9.0",
			Location:    "London, UK",
		},
		Layer: 1,
	}
}

func GoodGatewayRegistrationInfo() models.GatewayRegistrationInfo {
	return models.GatewayRegistrationInfo{
		NodeInfo: models.NodeInfo{
			MixHost:     "5.6.7.8:1789",
			IdentityKey: "3ebjp1Fb9hdcS1AR6AZihgeJiMHkB5jjJUsvqNnfQwU7",
			SphinxKey:   "E7eRtfBY2TeaREfdk6Wua9pXMrbWsJNEQyNaboa9Yx26",
			Version:     "0.9.0",
			Location:    "Neuchatel, CH",
		},
		ClientsHost: "ws://5.6.7.8:9000",
	}
}

func GoodRegisteredMix() models.RegisteredMix {
	return models.RegisteredMix{
		MixRegistrationInfo: GoodMixRegistrationInfo(),
	}
}

func GoodRegisteredGateway() models.RegisteredGateway {
	return models.RegisteredGateway{
		GatewayRegistrationInfo: GoodGatewayRegistrationInfo(),
	}
}
