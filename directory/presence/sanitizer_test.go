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
	"github.com/microcosm-cc/bluemonday"
	"github.com/nymtech/nym/validator/nym/directory/presence/fixtures"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
)

var _ = Describe("Sanitizer", func() {
	Describe("sanitizing inputs", func() {
		Context("for CocoHostInfo", func() {
			Context("when XSS is present", func() {
				It("sanitizes input", func() {
					policy := bluemonday.UGCPolicy()
					sanitizer := NewCoconodeSanitizer(policy)

					result := sanitizer.Sanitize(fixtures.XssCocoHost())
					assert.Equal(GinkgoT(), fixtures.GoodCocoHost(), result)
				})
			})
			Context("when XSS is not present", func() {
				It("doesn't change input", func() {
					policy := bluemonday.UGCPolicy()
					sanitizer := NewCoconodeSanitizer(policy)
					result := sanitizer.Sanitize(fixtures.GoodCocoHost())
					assert.Equal(GinkgoT(), fixtures.GoodCocoHost(), result)
				})
			})
		})
	})
	Context("for MixHostInfo", func() {
		Context("when XSS is present", func() {
			It("sanitizes input", func() {
				policy := bluemonday.UGCPolicy()
				sanitizer := NewMixnodeSanitizer(policy)

				result := sanitizer.Sanitize(fixtures.XssMixHost())
				assert.Equal(GinkgoT(), fixtures.GoodMixHost(), result)
			})
		})
		Context("when XSS is not present", func() {
			It("doesn't change input", func() {
				policy := bluemonday.UGCPolicy()
				sanitizer := NewMixnodeSanitizer(policy)
				result := sanitizer.Sanitize(fixtures.GoodMixHost())
				assert.Equal(GinkgoT(), fixtures.GoodMixHost(), result)
			})
		})
	})
	Context("for GatewayHostInfo", func() {
		Context("when XSS is present", func() {
			It("sanitizes input", func() {
				policy := bluemonday.UGCPolicy()
				sanitizer := NewGatewaySanitizer(policy)

				result := sanitizer.Sanitize(fixtures.XssGatewayHost())
				assert.Equal(GinkgoT(), fixtures.GoodGatewayHost(), result)
			})
		})
		Context("when XSS is not present", func() {
			It("doesn't change input", func() {
				policy := bluemonday.UGCPolicy()
				sanitizer := NewGatewaySanitizer(policy)
				result := sanitizer.Sanitize(fixtures.GoodGatewayHost())
				assert.Equal(GinkgoT(), fixtures.GoodGatewayHost(), result)
			})
		})
	})
})
