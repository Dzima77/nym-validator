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

package rest

import (
	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client/context"
)

// RegisterRoutes registers nym-related REST handlers to a router
func RegisterRoutes(cliCtx context.CLIContext, r *mux.Router) {
	// this line is used by starport scaffolding # 1
	r.HandleFunc("/nym/gateway", createGatewayHandler(cliCtx)).Methods("POST")
	r.HandleFunc("/nym/gateway", listGatewayHandler(cliCtx, "nym")).Methods("GET")
	r.HandleFunc("/nym/gateway/{key}", getGatewayHandler(cliCtx, "nym")).Methods("GET")
	r.HandleFunc("/nym/gateway", setGatewayHandler(cliCtx)).Methods("PUT")
	r.HandleFunc("/nym/gateway", deleteGatewayHandler(cliCtx)).Methods("DELETE")

	r.HandleFunc("/nym/mixnode", createMixnodeHandler(cliCtx)).Methods("POST")
	r.HandleFunc("/nym/mixnode", listMixnodeHandler(cliCtx, "nym")).Methods("GET")
	r.HandleFunc("/nym/mixnode/{key}", getMixnodeHandler(cliCtx, "nym")).Methods("GET")
	r.HandleFunc("/nym/mixnode", setMixnodeHandler(cliCtx)).Methods("PUT")
	r.HandleFunc("/nym/mixnode", deleteMixnodeHandler(cliCtx)).Methods("DELETE")

}
