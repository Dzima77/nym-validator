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

package models

// MixMetric is a report from each mixnode detailing recent traffic.
// Useful for creating visualisations.
type MixMetric struct {
	PubKey   string          `json:"pubKey" binding:"required"`
	Sent     map[string]uint `json:"sent" binding:"required"`
	Received *uint           `json:"received" binding:"required"`
}

// PersistedMixMetric is a saved MixMetric with a timestamp recording when it
// was seen by the directory server. It can be used to build visualizations of
// mixnet traffic.
type PersistedMixMetric struct {
	MixMetric
	Timestamp int64 `json:"timestamp" binding:"required"`
}
