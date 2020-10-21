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
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/nymtech/nym/validator/nym/directory/models"
	"github.com/nymtech/nym/validator/nym/directory/presence/fixtures"
	"github.com/nymtech/nym/validator/nym/directory/presence/mocks"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strconv"
)

var _ = Describe("Controller", func() {
	Describe("Registering mixnode", func() {
		It("Should save the information", func() {
			info := fixtures.GoodMixRegistrationInfo()
			router, mockService, mockSanitizer := SetupRouter()

			mockSanitizer.On("Sanitize", &info)
			mockService.On("RegisterMix", info)

			JSONReq, _ := json.Marshal(info)

			resp := performRequest(router, "POST", "/api/presence/mix", JSONReq)
			assert.Equal(GinkgoT(), http.StatusOK, resp.Code)
			// make sure sanitize is actually called on our request
			mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", &info)
			mockService.AssertCalled(GinkgoT(), "RegisterMix", info)
		})
	})

	Describe("Registering gateway", func() {
		It("Should save the information", func() {
			info := fixtures.GoodGatewayRegistrationInfo()
			router, mockService, mockSanitizer := SetupRouter()

			mockSanitizer.On("Sanitize", &info)
			mockService.On("RegisterGateway", info)

			JSONReq, _ := json.Marshal(info)

			resp := performRequest(router, "POST", "/api/presence/gateway", JSONReq)
			assert.Equal(GinkgoT(), http.StatusOK, resp.Code)
			// make sure sanitize is actually called on our request
			mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", &info)
			mockService.AssertCalled(GinkgoT(), "RegisterGateway", info)
		})
	})

	Describe("Unregistering node", func() {
		Context("If node exists", func() {
			It("Should return success", func() {
				nodeIdentity := "foomp"
				router, mockService, mockSanitizer := SetupRouter()

				mockSanitizer.On("Sanitize", &nodeIdentity)
				mockService.On("UnregisterNode", nodeIdentity).Return(true)

				resp := performRequest(router, "DELETE", "/api/presence/"+nodeIdentity, nil)
				assert.Equal(GinkgoT(), http.StatusOK, resp.Code)

				mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", &nodeIdentity)
				mockService.AssertCalled(GinkgoT(), "UnregisterNode", nodeIdentity)
			})
		})

		Context("If node does not exist", func() {
			It("Should return a 404", func() {
				nodeIdentity := "foomp"
				router, mockService, mockSanitizer := SetupRouter()

				mockSanitizer.On("Sanitize", &nodeIdentity)
				mockService.On("UnregisterNode", nodeIdentity).Return(false)

				resp := performRequest(router, "DELETE", "/api/presence/"+nodeIdentity, nil)
				assert.Equal(GinkgoT(), http.StatusNotFound, resp.Code)

				mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", &nodeIdentity)
				mockService.AssertCalled(GinkgoT(), "UnregisterNode", nodeIdentity)
			})
		})
	})

	Describe("Changing reputation", func() {
		Context("If node exists", func() {
			It("Should return success", func() {
				nodeIdentity := "foomp"
				newRep := int64(42)
				repStr := strconv.FormatInt(newRep, 10)
				router, mockService, mockSanitizer := SetupRouter()

				mockSanitizer.On("Sanitize", &nodeIdentity)
				mockSanitizer.On("Sanitize", &repStr)

				mockService.On("SetReputation", nodeIdentity, newRep).Return(true)

				resp := performRequest(router, "PATCH", "/api/presence/reputation/"+nodeIdentity+"?reputation="+repStr, nil)
				assert.Equal(GinkgoT(), http.StatusOK, resp.Code)

				mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", &nodeIdentity)
				mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", &repStr)
				mockService.AssertCalled(GinkgoT(), "SetReputation", nodeIdentity, newRep)
			})
		})

		Context("If node does not exist", func() {
			It("Should return a 404", func() {
				nodeIdentity := "foomp"
				newRep := int64(42)
				repStr := strconv.FormatInt(newRep, 10)
				router, mockService, mockSanitizer := SetupRouter()

				mockSanitizer.On("Sanitize", &nodeIdentity)
				mockSanitizer.On("Sanitize", &repStr)

				mockService.On("SetReputation", nodeIdentity, newRep).Return(false)

				resp := performRequest(router, "PATCH", "/api/presence/reputation/"+nodeIdentity+"?reputation="+repStr, nil)
				assert.Equal(GinkgoT(), http.StatusNotFound, resp.Code)

				mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", &nodeIdentity)
				mockSanitizer.AssertCalled(GinkgoT(), "Sanitize", &repStr)
				mockService.AssertCalled(GinkgoT(), "SetReputation", nodeIdentity, newRep)
			})
		})
	})

	Describe("Getting topology", func() {
		It("Delegates the call to the service", func() {
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

			router, mockService, _ := SetupRouter()

			mockService.On("GetTopology").Return(expectedTopology)

			resp := performRequest(router, "GET", "/api/presence/topology", nil)
			var response models.Topology
			if err := json.Unmarshal([]byte(resp.Body.String()), &response); err != nil {
				panic(err)
			}

			assert.Equal(GinkgoT(), http.StatusOK, resp.Code)
			assert.Equal(GinkgoT(), expectedTopology, response)
			mockService.AssertCalled(GinkgoT(), "GetTopology")
		})
	})

	Describe("Getting active topology", func() {
		It("Delegates the call to the service", func() {
			mix1 := fixtures.GoodRegisteredMix()
			mix1.Reputation = ReputationThreshold

			gate1 := fixtures.GoodRegisteredGateway()
			gate1.Reputation = ReputationThreshold

			expectedTopology := models.Topology{
				MixNodes: []models.RegisteredMix{mix1},
				Gateways: []models.RegisteredGateway{gate1},
			}

			router, mockService, _ := SetupRouter()

			mockService.On("GetActiveTopology").Return(expectedTopology)

			resp := performRequest(router, "GET", "/api/presence/topology/active", nil)
			var response models.Topology
			if err := json.Unmarshal([]byte(resp.Body.String()), &response); err != nil {
				panic(err)
			}

			assert.Equal(GinkgoT(), http.StatusOK, resp.Code)
			assert.Equal(GinkgoT(), expectedTopology, response)
			mockService.AssertCalled(GinkgoT(), "GetActiveTopology")
		})
	})
})

func SetupRouter() (*gin.Engine, *mocks.IService, *mocks.Sanitizer) {
	mockSanitizer := new(mocks.Sanitizer)
	mockService := new(mocks.IService)
	cfg := Config{
		Sanitizer: mockSanitizer,
		Service:   mockService,
	}
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	controller := New(cfg)
	controller.RegisterRoutes(router)
	return router, mockService, mockSanitizer
}

func performRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	buf := bytes.NewBuffer(body)
	req, _ := http.NewRequest(method, path, buf)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
