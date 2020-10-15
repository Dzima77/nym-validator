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

func MixNodeID() models.MixNodeID {
	d := models.MixNodeID{
		PubKey: "abc123",
	}
	return d
}

func GoodCocoHost() models.CocoHostInfo {
	good := models.CocoHostInfo{
		HostInfo: models.HostInfo{
			Location: "foomplandia",
			Host:     ":1789",
			PubKey:   "pubkey",
			Version:  "0.1.0",
		},
		Type: "type",
	}
	return good
}

func GoodMixHost() models.MixHostInfo {
	good := models.MixHostInfo{
		HostInfo: models.HostInfo{
			Location: "foomplandia",
			Host:     ":1789",
			PubKey:   "pubkey",
			Version:  "0.7.0",
		},
		Layer: 1,
	}
	return good
}

func GoodGatewayHost() models.GatewayHostInfo {
	good := models.GatewayHostInfo{
		Location:       "foomplandia",
		ClientListener: ":1789",
		MixnetListener: ":1790",
		IdentityKey:    "pubkey",
		SphinxKey:      "sphinxkey",
		Version:        "0.1.0",
	}
	return good
}

func XssCocoHost() models.CocoHostInfo {
	xss := models.CocoHostInfo{
		HostInfo: models.HostInfo{
			Location: "foomplandia",
			Host:     ":1789",
			PubKey:   "pubkey<script>alert('gotcha')</script>",
			Version:  "0.1.0",
		},
		Type: "type<script>alert('gotcha')",
	}
	return xss
}

func XssMixHost() models.MixHostInfo {
	xss := models.MixHostInfo{
		HostInfo: models.HostInfo{
			Location: "foomplandia",
			Host:     ":1789",
			PubKey:   "pubkey<script>alert('gotcha')</script>",
			Version:  "0.7.0",
		},
		Layer: 1,
	}
	return xss
}

func XssGatewayHost() models.GatewayHostInfo {
	xss := models.GatewayHostInfo{
		Location:       "foomplandia",
		ClientListener: ":1789",
		MixnetListener: ":1790",
		IdentityKey:    "pubkey<script>alert('gotcha')</script>",
		SphinxKey:      "sphinxkey",
		Version:        "0.1.0",
	}
	return xss
}
