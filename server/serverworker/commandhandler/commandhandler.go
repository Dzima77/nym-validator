// commandhandler.go - handlers for coconut requests.
// Copyright (C) 2019  Jedrzej Stuczynski.
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

// Package commandhandler contains functions that are used to resolve commands issued to issuers and providers.
package commandhandler

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	"github.com/nymtech/nym/common/comm/commands"
	"github.com/nymtech/nym/crypto/coconut/concurrency/coconutworker"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/crypto/elgamal"
	ethclient "github.com/nymtech/nym/ethereum/client"
	"github.com/nymtech/nym/server/issuer/utils"
	"github.com/nymtech/nym/server/storage"
	nymclient "github.com/nymtech/nym/tendermint/client"
	"github.com/nymtech/nym/tendermint/nymabci/code"
	tmconst "github.com/nymtech/nym/tendermint/nymabci/constants"
	"github.com/nymtech/nym/tendermint/nymabci/query"
	"github.com/nymtech/nym/tendermint/nymabci/transaction"
	"gopkg.in/op/go-logging.v1"
)

// TODO: perhaps if it's too expensive, replace reflect.Type with some string or even a byte?
type HandlerRegistry map[reflect.Type]HandlerRegistryEntry

type HandlerRegistryEntry struct {
	DataFn func(cmd commands.Command) HandlerData
	Fn     HandlerFunc
}

// context is really useful for the most time consuming functions like blindverify
// it is not very useful for say "getVerificatonKey", but nevertheless, it is there for both,
// completion sake and future proofness
type HandlerFunc func(context.Context, HandlerData) *commands.Response

// command - request to resolve
// logger - possibly to remove later?
// pointer to coconut worker - that deals with actual crypto (passes it down to workers etc)
// request specific piece of data - for sign it's the secret key, for verify it's the verification key, etc.
type HandlerData interface {
	Command() commands.Command
	CoconutWorker() *coconutworker.CoconutWorker
	Log() *logging.Logger
	Data() interface{}
}

func DefaultResponse() *commands.Response {
	return &commands.Response{
		Data:         nil,
		ErrorStatus:  commands.DefaultResponseErrorStatusCode,
		ErrorMessage: commands.DefaultResponseErrorMessage,
	}
}

func setErrorResponse(log *logging.Logger, response *commands.Response, errMsg string, errCode commands.StatusCode) {
	log.Error(errMsg)
	// response.Data = nil
	response.ErrorMessage = errMsg
	response.ErrorStatus = errCode
}

type SignRequestHandlerData struct {
	Cmd       *commands.SignRequest
	Worker    *coconutworker.CoconutWorker
	Logger    *logging.Logger
	SecretKey *coconut.ThresholdSecretKey
}

func (handlerData *SignRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *SignRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *SignRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *SignRequestHandlerData) Data() interface{} {
	return handlerData.SecretKey
}

func SignRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.SignRequest)
	log := reqData.Log()
	tsk := reqData.Data().(*coconut.ThresholdSecretKey)
	response := DefaultResponse()

	log.Debug("SignRequestHandler")
	if len(req.PubM) > len(tsk.Y()) {
		errMsg := fmt.Sprintf("Received more attributes to sign than what the server supports."+
			" Got: %v, expected at most: %v", len(req.PubM), len(tsk.Y()))
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}
	bigs, err := coconut.BigSliceFromByteSlices(req.PubM)
	if err != nil {
		errMsg := fmt.Sprintf("Error while recovering big numbers from the slice: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}
	sig, err := reqData.CoconutWorker().SignWrapper(tsk.SecretKey, bigs)
	if err != nil {
		// TODO: should client really know those details?
		errMsg := fmt.Sprintf("Error while signing message: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}
	log.Debugf("Writing back signature %v", sig)
	response.Data = utils.IssuedSignature{
		Sig:      sig,
		IssuerID: tsk.ID(),
	}
	return response
}

type VerificationKeyRequestHandlerData struct {
	Cmd             *commands.VerificationKeyRequest
	Worker          *coconutworker.CoconutWorker
	Logger          *logging.Logger
	VerificationKey *coconut.ThresholdVerificationKey
}

func (handlerData *VerificationKeyRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *VerificationKeyRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *VerificationKeyRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *VerificationKeyRequestHandlerData) Data() interface{} {
	return handlerData.VerificationKey
}

func VerificationKeyRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	response := DefaultResponse()
	log := reqData.Log()
	log.Debug("VerificationKeyRequestHandler")
	response.Data = reqData.Data()
	return response
}

type BlindSignRequestHandlerData struct {
	Cmd       *commands.BlindSignRequest
	Worker    *coconutworker.CoconutWorker
	Logger    *logging.Logger
	SecretKey *coconut.ThresholdSecretKey
}

func (handlerData *BlindSignRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *BlindSignRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *BlindSignRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *BlindSignRequestHandlerData) Data() interface{} {
	return handlerData.SecretKey
}

func BlindSignRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.BlindSignRequest)
	log := reqData.Log()
	tsk := reqData.Data().(*coconut.ThresholdSecretKey)
	response := DefaultResponse()

	log.Debug("BlindSignRequestHandler")
	lambda := &coconut.Lambda{}
	if err := lambda.FromProto(req.Lambda); err != nil {
		errMsg := "Could not recover received lambda."
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}
	if len(req.PubM)+len(lambda.Enc()) > len(tsk.Y()) {
		errMsg := fmt.Sprintf("Received more attributes to sign than what the server supports."+
			" Got: %v, expected at most: %v", len(req.PubM)+len(lambda.Enc()), len(tsk.Y()))
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}
	egPub := &elgamal.PublicKey{}
	if err := egPub.FromProto(req.EgPub); err != nil {
		errMsg := "Could not recover received ElGamal Public Key."
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}
	bigs, err := coconut.BigSliceFromByteSlices(req.PubM)
	if err != nil {
		errMsg := fmt.Sprintf("Error while recovering big numbers from the slice: %v", err)
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}
	sig, err := reqData.CoconutWorker().BlindSignWrapper(tsk.SecretKey, lambda, egPub, bigs)
	if err != nil {
		// TODO: should client really know those details?
		errMsg := fmt.Sprintf("Error while signing message: %v", err)
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}
	log.Debugf("Writing back blinded signature")
	response.Data = utils.IssuedSignature{
		Sig:      sig,
		IssuerID: tsk.ID(),
	}
	return response
}

type LookUpCredentialRequestHandlerData struct {
	Cmd    *commands.LookUpCredentialRequest
	Worker *coconutworker.CoconutWorker
	Logger *logging.Logger
	Store  *storage.Database
}

func (handlerData *LookUpCredentialRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *LookUpCredentialRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *LookUpCredentialRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *LookUpCredentialRequestHandlerData) Data() interface{} {
	return handlerData.Store
}

func LookUpCredentialRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.LookUpCredentialRequest)
	log := reqData.Log()
	store := reqData.Data().(*storage.Database)

	log.Debug("LookUpCredentialRequestHandler")
	response := DefaultResponse()
	current := store.GetHighest()
	if current < req.Height {
		errMsg := fmt.Sprintf("Target height hasn't been processed yet. Target: %v, current: %v", req.Height, current)
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_NOT_PROCESSED_YET)
		return response
	}

	credPair := store.GetCredential(req.Height, req.Gamma)
	if len(credPair.Credential.Sig) == 0 {
		errMsg := "Could not lookup the credential using provided arguments"
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}
	log.Debugf("Writing back credentials for height %v with gamma %v", req.Height, req.Gamma)
	response.Data = credPair
	return response
}

type LookUpBlockCredentialsRequestHandlerData struct {
	Cmd    *commands.LookUpBlockCredentialsRequest
	Worker *coconutworker.CoconutWorker
	Logger *logging.Logger
	Store  *storage.Database
}

func (handlerData *LookUpBlockCredentialsRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *LookUpBlockCredentialsRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *LookUpBlockCredentialsRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *LookUpBlockCredentialsRequestHandlerData) Data() interface{} {
	return handlerData.Store
}

func LookUpBlockCredentialsRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.LookUpBlockCredentialsRequest)
	log := reqData.Log()
	store := reqData.Data().(*storage.Database)
	log.Debug("LookUpBlockCredentialsRequestHandler")

	response := DefaultResponse()
	current := store.GetHighest()
	if current < req.Height {
		errMsg := fmt.Sprintf("Target height hasn't been processed yet. Target: %v, current: %v", req.Height, current)
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_NOT_PROCESSED_YET)
		return response
	}

	credPairs := store.GetBlockCredentials(req.Height)
	if len(credPairs) == 0 {
		errMsg := "Could not lookup the credential using provided arguments. " +
			"Either there were no valid txs in this block or it wasn't processed yet."
		setErrorResponse(reqData.Log(), response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}

	log.Debugf("Writing back all credentials for height %v", req.Height)
	response.Data = credPairs
	return response
}

type VerifyRequestHandlerData struct {
	Cmd             *commands.VerifyRequest
	Worker          *coconutworker.CoconutWorker
	Logger          *logging.Logger
	VerificationKey *coconut.VerificationKey
}

func (handlerData *VerifyRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *VerifyRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *VerifyRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *VerifyRequestHandlerData) Data() interface{} {
	return handlerData.VerificationKey
}

func VerifyRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.VerifyRequest)
	log := reqData.Log()
	response := DefaultResponse()
	avk := reqData.Data().(*coconut.VerificationKey)

	log.Debug("VerifyRequestHandler")
	sig := &coconut.Signature{}
	if err := sig.FromProto(req.Sig); err != nil {
		errMsg := "Could not recover received signature."
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}
	bigs, err := coconut.BigSliceFromByteSlices(req.PubM)
	if err != nil {
		errMsg := fmt.Sprintf("Error while recovering big numbers from the slice: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}
	response.Data = reqData.CoconutWorker().VerifyWrapper(avk, bigs, sig)
	return response
}

type BlindVerifyRequestHandlerData struct {
	Cmd             *commands.BlindVerifyRequest
	Worker          *coconutworker.CoconutWorker
	Logger          *logging.Logger
	VerificationKey *coconut.VerificationKey
}

func (handlerData *BlindVerifyRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *BlindVerifyRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *BlindVerifyRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *BlindVerifyRequestHandlerData) Data() interface{} {
	return handlerData.VerificationKey
}

func BlindVerifyRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.BlindVerifyRequest)
	log := reqData.Log()
	response := DefaultResponse()
	avk := reqData.Data().(*coconut.VerificationKey)

	log.Debug("BlindVerifyRequestHandler")
	sig := &coconut.Signature{}
	if err := sig.FromProto(req.Sig); err != nil {
		errMsg := "Could not recover received signature."
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}
	theta := &coconut.Theta{}
	if err := theta.FromProto(req.Theta); err != nil {
		errMsg := "Could not recover received theta."
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}
	pubM, err := coconut.BigSliceFromByteSlices(req.PubM)
	if err != nil {
		errMsg := fmt.Sprintf("Error while recovering big numbers from the slice: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}
	response.Data = reqData.CoconutWorker().BlindVerifyWrapper(avk, sig, theta, pubM)
	return response
}

type SpendCredentialVerificationData struct {
	Avk       *coconut.VerificationKey
	Address   ethcommon.Address
	NymClient *nymclient.Client // in theory it should be safe to use the same instance for multiple requests
}

type SpendCredentialRequestHandlerData struct {
	Cmd              *commands.SpendCredentialRequest
	Worker           *coconutworker.CoconutWorker
	Logger           *logging.Logger
	VerificationData SpendCredentialVerificationData
}

func (handlerData *SpendCredentialRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *SpendCredentialRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *SpendCredentialRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *SpendCredentialRequestHandlerData) Data() interface{} {
	return handlerData.VerificationData
}

func depositCredential(ctx context.Context, originalReqData HandlerData) error {
	req := originalReqData.Command().(*commands.SpendCredentialRequest)
	verificationData := originalReqData.Data().(SpendCredentialVerificationData)
	log := originalReqData.Log()

	log.Debug("Going to deposit the received credential")
	blockchainRequest, err := transaction.CreateNewDepositCoconutCredentialRequest(
		req.Sig,
		req.PubM,
		req.Theta,
		req.Value,
		verificationData.Address,
	)
	if err != nil {
		return fmt.Errorf("Failed to create blockchain request: %v", err)
	}

	blockchainResponse, err := verificationData.NymClient.Broadcast(blockchainRequest)
	if err != nil {
		return fmt.Errorf("Failed to send transaction to the blockchain: %v", err)
	}

	log.Debugf("Received response from the blockchain.\n Return Deliver code: %v; Additional Deliver Data: %v\n"+
		"Return Check code: %v; Additional Check Data: %v",
		code.ToString(blockchainResponse.DeliverTx.Code), string(blockchainResponse.DeliverTx.Data),
		code.ToString(blockchainResponse.CheckTx.Code), string(blockchainResponse.CheckTx.Data),
	)

	if blockchainResponse.CheckTx.Code != code.OK {
		return fmt.Errorf("The transaction failed to be included on the blockchain (checkTx). Errorcode: %v - %v",
			blockchainResponse.CheckTx.Code, code.ToString(blockchainResponse.CheckTx.Code))
	}

	if blockchainResponse.DeliverTx.Code != code.OK {
		return fmt.Errorf("The transaction failed to be included on the blockchain (deliverTx). Errorcode: %v - %v",
			blockchainResponse.DeliverTx.Code, code.ToString(blockchainResponse.DeliverTx.Code))
	}

	// TODO: put value of this ticker in config
	tickerInterval := time.Second
	retryTicker := time.NewTicker(tickerInterval)
	log.Info("Waiting for the credential to get verified by the Nym network")
outerLoop:
	for {
		select {
		case <-ctx.Done():
			// TODO: perhaps recheck at later time?
			return fmt.Errorf("Timed out while waiting for credential verificaton")
		case <-retryTicker.C:
			zetaStatusRes, err := verificationData.NymClient.Query(query.FullZetaStatus, req.Theta.Zeta)
			if err != nil {
				return fmt.Errorf("Failed to check status of zeta")
			}
			zetaStatus := zetaStatusRes.Response.Value
			if bytes.HasPrefix(zetaStatus, tmconst.ZetaStatusUnspent.DbEntry()) {
				log.Critical("Zeta has invalid state - unspent") // TODO: what to actually do?
				// since the transaction to blockchain succeeded, it means our request to deposit the credential
				// was executed and zeta by default should have status of 'being verified'
			} else if bytes.HasPrefix(zetaStatus, tmconst.ZetaStatusBeingVerified.DbEntry()) {
				log.Debug("We are still waiting on consensus on credential validity")
			} else if bytes.HasPrefix(zetaStatus, tmconst.ZetaStatusSpent.DbEntry()) {
				// BytesToAddress is cropping address from the left, so it's perfect for us to remove status prefix
				creditedProviderAddress := ethcommon.BytesToAddress(zetaStatus)
				// make sure this is our address
				log.Info("The credential was successfully verified")
				if bytes.Equal(creditedProviderAddress[:], verificationData.Address[:]) {
					log.Notice("The credential was deposited to our account")
					break outerLoop
				} else {
					return fmt.Errorf("The credential was deposited to an unkown account (%v). Our address: %v",
						creditedProviderAddress.Hex(),
						verificationData.Address.Hex(),
					)
				}
			} else {
				log.Critical("Unknown state?")
				return fmt.Errorf("Zeta is in unknown state")
			}
			log.Debugf("Waiting for %v before retrying", tickerInterval)
		}
	}
	return nil
}

// TODO: split this function into multiple functions since clearly this is a procedure taking multiple steps
// even division into "spend" and "deposit" would make everything way more readable
func SpendCredentialRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.SpendCredentialRequest)
	verificationData := reqData.Data().(SpendCredentialVerificationData)
	log := reqData.Log()
	response := DefaultResponse()
	response.Data = false

	log.Debug("SpendCredentialRequestHandler")

	if !bytes.Equal(req.MerchantAddress, verificationData.Address[:]) {
		errMsg := "Invalid merchant address"
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_BINDING)
		return response
	}

	sig := &coconut.Signature{}
	if err := sig.FromProto(req.Sig); err != nil {
		errMsg := fmt.Sprintf("Failed to unmarshal signature: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_SIGNATURE)
		return response
	}

	thetaTumbler := &coconut.ThetaTumbler{}
	if err := thetaTumbler.FromProto(req.Theta); err != nil {
		errMsg := fmt.Sprintf("Failed to unmarshal theta: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}

	pubM, err := coconut.BigSliceFromByteSlices(req.PubM)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to unmarshal public attributes: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}

	// Depends on provider settings, if we are to verify credential, we do just that (it includes checking the binding)
	// otherwise we only verify the binding
	// When the provider is going to redeem the credential for itself, it will be verified by the nym system anyway.
	if verificationData.Avk != nil {
		isValid := reqData.CoconutWorker().BlindVerifyTumblerWrapper(
			verificationData.Avk,
			sig,
			thetaTumbler,
			pubM,
			verificationData.Address[:],
		)
		if !isValid {
			setErrorResponse(log, response, "Failed to verify the data", commands.StatusCode_INVALID_SIGNATURE)
			return response
		}
		log.Info("The received credential is valid (checked locally)")
	} else {
		bind, bindErr := coconut.CreateBinding(verificationData.Address[:])
		if bindErr != nil {
			log.Critical("Failed to create binding out of our own address")
			setErrorResponse(log, response, "Critical failure when generating own binding", commands.StatusCode_PROCESSING_ERROR)
			return response
		}
		if !bind.Equals(thetaTumbler.Zeta()) {
			setErrorResponse(log, response, "Invalid binding provided", commands.StatusCode_INVALID_BINDING)
			return response
		}
	}

	// this is not by any means a reliable check as this request is not properly ordered, etc.
	// All it does is check against credentials spent in the past (so say it would fail if client sent same request
	// to two SPs now)
	wasSpentRes, err := verificationData.NymClient.Query(query.FullZetaStatus, req.Theta.Zeta)
	log.Critical(fmt.Sprintf("spent: %v", wasSpentRes.Response.Value))

	if err != nil {
		errMsg := "Failed to preliminarily check status of zeta"
		setErrorResponse(log, response, errMsg, commands.StatusCode_UNAVAILABLE)
		return response
	}

	if !bytes.Equal(wasSpentRes.Response.Value, tmconst.ZetaStatusUnspent.DbEntry()) {
		errMsg := "Received zeta was already spent before"
		setErrorResponse(log, response, errMsg, commands.StatusCode_DOUBLE_SPENDING_ATTEMPT)
		return response
	}

	log.Info("The received credential seems to not have been spent before (THIS IS NOT A GUARANTEE)")

	// TODO: now it's a question of whether we want to immediately try to deposit our credential or wait and do it later
	// and possibly in bulk. In the former case: store the data in the database
	// However, for the demo sake (since it's easier), deposit immediately
	// TODO: in future we could just store that marshalled request (as below) rather than all attributes separately

	// however, do it in new goroutine so that we could reply to client immediately. in actual system the deposits
	// would most likely be batched anyway so the client-perceived delay would be similar
	// TODO: somehow catch the error?
	go depositCredential(ctx, reqData)

	// the response data in future might be provider dependent, to include say some authorization token
	response.ErrorStatus = commands.StatusCode_OK
	response.Data = true
	return response
}

type CredentialVerifierData struct {
	Avk        *coconut.VerificationKey
	PrivateKey *ecdsa.PrivateKey
	NymClient  *nymclient.Client // in theory it should be safe to use the same instance for multiple requests
}

type CredentialVerificationRequestHandlerData struct {
	Cmd              *commands.CredentialVerificationRequest
	Worker           *coconutworker.CoconutWorker
	Logger           *logging.Logger
	VerificationData CredentialVerifierData
}

func (handlerData *CredentialVerificationRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *CredentialVerificationRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *CredentialVerificationRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *CredentialVerificationRequestHandlerData) Data() interface{} {
	return handlerData.VerificationData
}

// TODO: This handler doesn't really fit in here...
func CredentialVerificationRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.CredentialVerificationRequest)
	log := reqData.Log()
	response := DefaultResponse()
	verificationData := reqData.Data().(CredentialVerifierData)
	response.Data = false // TODO: do we even need to return anything?

	log.Debug("CredentialVerificationRequestHandler")
	cryptoMaterials := &coconut.TumblerBlindVerifyMaterials{}
	if err := cryptoMaterials.FromProto(req.CryptoMaterials); err != nil {
		errMsg := "Could not recover crypto materials."
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}

	// make sure the actual value sent matches what is encoded
	if Curve.Comp(cryptoMaterials.PubM()[0], Curve.NewBIGint(int(req.Value))) != 0 {
		errMsg := "Values does not match up."
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_ARGUMENTS)
		return response
	}

	isValid := reqData.CoconutWorker().BlindVerifyTumblerWrapper(verificationData.Avk,
		cryptoMaterials.Sig(),
		cryptoMaterials.Theta(),
		cryptoMaterials.PubM(),
		req.BoundAddress,
	)

	tx, err := transaction.CreateNewCredentialVerificationNotification(verificationData.PrivateKey,
		ethcommon.BytesToAddress(req.BoundAddress),
		req.Value,
		req.CryptoMaterials.Theta.Zeta,
		isValid,
	)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create notification transaction: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}

	// TODO: later change to send sync or even async as technically we don't need to know
	// the resolution on this. We just need to send it.
	res, err := verificationData.NymClient.Broadcast(tx)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send notification transaction: %v", err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}

	log.Infof("Received tendermint Response.\nCheckCode: %v, Check additional data: %v\nDeliverCode: %v Deliver Aditional data: %v",
		code.ToString(res.CheckTx.Code),
		string(res.CheckTx.Data),
		code.ToString(res.DeliverTx.Code),
		string(res.DeliverTx.Data),
	)

	if res.CheckTx.Code == code.OK && res.DeliverTx.Code == code.OK {
		response.Data = true // our notification was accepted
	}

	return response
}

type FaucetData struct {
	PrivateKey  *ecdsa.PrivateKey
	EthClient   *ethclient.Client
	EtherAmount float64
}

type FaucetTransferRequestHandlerData struct {
	Cmd        *commands.FaucetTransferRequest
	Worker     *coconutworker.CoconutWorker
	Logger     *logging.Logger
	FaucetData FaucetData
}

func (handlerData *FaucetTransferRequestHandlerData) Command() commands.Command {
	return handlerData.Cmd
}

func (handlerData *FaucetTransferRequestHandlerData) CoconutWorker() *coconutworker.CoconutWorker {
	return handlerData.Worker
}

func (handlerData *FaucetTransferRequestHandlerData) Log() *logging.Logger {
	return handlerData.Logger
}

func (handlerData *FaucetTransferRequestHandlerData) Data() interface{} {
	return handlerData.FaucetData
}

func getTripleDigitRounding(balance *big.Int) float64 {
	// denomination has 18 decimal places but we want to have 3 decimal precision
	t := new(big.Int)
	denomination := t.Exp(big.NewInt(10), big.NewInt(15), nil)
	rounded := t.Div(balance, denomination)
	return float64(rounded.Int64()) / 1000.0
}

func FaucetTransferRequestHandler(ctx context.Context, reqData HandlerData) *commands.Response {
	req := reqData.Command().(*commands.FaucetTransferRequest)
	log := reqData.Log()
	response := DefaultResponse()
	faucetData := reqData.Data().(FaucetData)

	log.Debug("FaucetTransferRequestHandler")

	ourAddress := ethcrypto.PubkeyToAddress(*faucetData.PrivateKey.Public().(*ecdsa.PublicKey))
	ethClient := faucetData.EthClient
	erc20balance, err := ethClient.QueryERC20Balance(ctx, ourAddress, false)
	if err != nil {
		errMsg := "Error while quering for our own ERC20 balance"
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}

	roundedERC20Balance := getTripleDigitRounding(erc20balance)
	log.Noticef("We have %v (rounded) ERC20 Nym remaining. Full: %v (18 decimal places)", roundedERC20Balance, erc20balance)
	if roundedERC20Balance < float64(req.Amount) {
		errMsg := "Requested more ERC20 tokens than available"
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}

	etherBalance, err := ethClient.QueryEtherBalance(ctx, ourAddress, nil)
	if err != nil {
		errMsg := "Error while quering for our own Ether balance"
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}

	roundedEtherBalance := getTripleDigitRounding(etherBalance)
	log.Noticef("We have %v (rounded) Ether remaining. Full: %v (18 decimal places aka Wei)", roundedEtherBalance, etherBalance)
	if roundedEtherBalance < faucetData.EtherAmount {
		errMsg := "Requested more Ether than available"
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}

	msg := make([]byte, ethcommon.AddressLength+8)
	i := copy(msg, req.Address)
	binary.BigEndian.PutUint64(msg[i:], req.Amount)

	recPub, err := ethcrypto.SigToPub(tmconst.HashFunction(msg), req.Sig)
	if err != nil {
		errMsg := "Error while trying to recover public key associated with the signature"
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_SIGNATURE)
		return response
	}

	recAddr := ethcrypto.PubkeyToAddress(*recPub)
	if !bytes.Equal(recAddr[:], req.Address) {
		errMsg := "Failed to verify signature on request"
		setErrorResponse(log, response, errMsg, commands.StatusCode_INVALID_SIGNATURE)
		return response
	}

	erc20Hash, err := ethClient.TransferERC20Tokens(ctx, int64(req.Amount), recAddr)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send %v ERC20 Nyms to %v: %v", req.Amount, recAddr.Hex(), err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}

	etherHash, err := ethClient.TransferEther(ctx, recAddr, faucetData.EtherAmount)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to send %v Ether to %v: %v", faucetData.EtherAmount, recAddr.Hex(), err)
		setErrorResponse(log, response, errMsg, commands.StatusCode_PROCESSING_ERROR)
		return response
	}

	// // just wait server-side for resolvment of the request. it does not need to be efficient or concurrent etc.
	// // it just simplifies client-logic which is temporary anyway.
	// log.Debug("waiting for erc20 transfer to resolve")
	// waitForTxToResolve(ctx, erc20Hash, ethClient)
	// log.Debug("waiting for ether transfer to resolve")
	// waitForTxToResolve(ctx, etherHash, ethClient)
	// log.Debug("both transfers resolved")

	data := make([]byte, 2*ethcommon.HashLength)
	i = copy(data, erc20Hash[:])
	copy(data[i:], etherHash[:])

	log.Warningf("hash1: %v, hash2: %v", erc20Hash.Hex(), etherHash.Hex())
	response.Data = data

	return response
}
