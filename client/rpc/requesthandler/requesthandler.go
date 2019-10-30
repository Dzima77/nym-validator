// Copyright (C) 2019  Nym Authors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package requesthandler

import (
	"errors"
	ethcommon "github.com/ethereum/go-ethereum/common"
	Curve "github.com/nymtech/amcl/version3/go/amcl/BLS381"
	"github.com/nymtech/nym-validator/client"
	types "github.com/nymtech/nym-validator/client/rpc/clienttypes"
	"github.com/nymtech/nym-validator/constants"
	coconut "github.com/nymtech/nym-validator/crypto/coconut/scheme"
	"github.com/nymtech/nym-validator/nym/token"
)

func getErrorResponse(err error) *types.Response {
	return &types.Response{
		Value: &types.Response_Exception{
			Exception: &types.ResponseException{
				Error: err.Error(),
			},
		},
	}
}

func HandleGetCredential(req *types.Request_GetCredential, c *client.Client) *types.Response {
	s := c.RandomBIG()
	k := c.RandomBIG()
	tok, err := token.New(s, k, req.GetCredential.Value)
	if err != nil {
		return getErrorResponse(err)
	}
	cred, err := c.GetCredential(tok)
	if err != nil {
		return getErrorResponse(err)
	}
	protoCred, err := cred.ToProto()
	if err != nil {
		return getErrorResponse(err)
	}

	sb := make([]byte, constants.BIGLen)
	kb := make([]byte, constants.BIGLen)

	s.ToBytes(sb)
	k.ToBytes(kb)

	return &types.Response{
		Value: &types.Response_GetCredential{
			GetCredential: &types.ResponseGetCredential{
				Credential: protoCred,
				Materials: &types.CredentialMaterials{
					Value:    req.GetCredential.Value,
					Secret:   kb,
					Sequence: sb,
				},
			},
		},
	}
}

func HandleSpendCredential(req *types.Request_SpendCredential, c *client.Client) *types.Response {
	s := Curve.FromBytes(req.SpendCredential.Materials.Sequence)
	k := Curve.FromBytes(req.SpendCredential.Materials.Secret)

	tok, err := token.New(s, k, req.SpendCredential.Materials.Value)
	if err != nil {
		return getErrorResponse(err)
	}

	cred := &coconut.Signature{}
	if err := cred.FromProto(req.SpendCredential.Credential); err != nil {
		return getErrorResponse(err)
	}
	spAddress := ethcommon.HexToAddress(req.SpendCredential.Provider.Address)

	success, err := c.SpendCredential(tok, cred, req.SpendCredential.Provider.Ip, spAddress, nil)
	if err != nil {
		return getErrorResponse(err)
	}
	// shouldn't happen as no success implies error (or at least should)
	if !success {
		return getErrorResponse(errors.New("failed to spend the credential. reason: unknown"))
	}

	return &types.Response{
		Value: &types.Response_SpendCredential{
			SpendCredential: &types.ResponseSpendCredential{},
		},
	}

}

func HandleGetServiceProviders(req *types.Request_GetProviders, c *client.Client) *types.Response {
	sps := c.GetServiceProviders()
	return &types.Response{
		Value: &types.Response_GetProviders{
			GetProviders: &types.ResponseGetServiceProviders{
				Providers: sps,
			},
		},
	}

}

func HandleRerandomize(req *types.Request_Rerandomize, c *client.Client) *types.Response {
	protoCred := req.Rerandomize.GetCredential()
	cred := &coconut.Signature{}
	if err := cred.FromProto(protoCred); err != nil {
		return getErrorResponse(err)
	}
	rCred := c.ForceReRandomizeCredential(cred)
	rCredProto, err := rCred.ToProto()
	if err != nil {
		return getErrorResponse(err)
	}
	return &types.Response{
		Value: &types.Response_Rerandomize{
			Rerandomize: &types.ResponseRerandomize{
				Credential: rCredProto,
			},
		},
	}
}

func HandleFlush(req *types.Request_Flush) *types.Response {
	return &types.Response{
		Value: &types.Response_Flush{
			Flush: &types.ResponseFlush{},
		},
	}
}

func HandleInvalidRequest() *types.Response {
	return &types.Response{
		Value: &types.Response_Exception{
			Exception: &types.ResponseException{
				Error: "Invalid server request",
			},
		},
	}
}
