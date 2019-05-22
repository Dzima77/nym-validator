// check.go - CheckTx logic for Tendermint ABCI for Nym
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

package nymapplication

import (
	"bytes"
	"encoding/binary"

	"0xacab.org/jstuczyn/CoconutGo/tendermint/nymabci/code"
	tmconst "0xacab.org/jstuczyn/CoconutGo/tendermint/nymabci/constants"
	"0xacab.org/jstuczyn/CoconutGo/tendermint/nymabci/transaction"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
)

// implementation will be IP-specific
func (app *NymApplication) verifyCredential(cred []byte) bool {
	return true
}

func (app *NymApplication) validateTransfer(inAddr, outAddr []byte, amount uint64) (uint32, []byte) {
	if len(inAddr) != ethcommon.AddressLength {
		return code.MALFORMED_ADDRESS, []byte("SOURCE")
	}
	if len(outAddr) != ethcommon.AddressLength {
		return code.MALFORMED_ADDRESS, []byte("TARGET")
	}
	// don't allow transfer when addresses are identical because nothing would happen anyway...
	if bytes.Equal(inAddr, outAddr) {
		return code.SELF_TRANSFER, nil
	}

	sourceBalanceB, retCode := app.queryBalance(inAddr)
	if retCode != code.OK {
		return code.ACCOUNT_DOES_NOT_EXIST, []byte("SOURCE")
	}

	sourceBalance := binary.BigEndian.Uint64(sourceBalanceB)
	if sourceBalance < amount { // + some gas?
		return code.INSUFFICIENT_BALANCE, nil
	}

	if _, retCodeT := app.queryBalance(outAddr); retCodeT != code.OK {
		return code.ACCOUNT_DOES_NOT_EXIST, []byte("TARGET")
	}

	return code.OK, nil
}

// the tx prefix was removed
func (app *NymApplication) checkNewAccountTx(tx []byte) uint32 {
	req := &transaction.NewAccountRequest{}

	if err := proto.Unmarshal(tx, req); err != nil {
		app.log.Info("Failed to unmarshal request")
		return code.INVALID_TX_PARAMS
	}

	if len(req.Address) != ethcommon.AddressLength {
		return code.INVALID_TX_PARAMS
	}

	if !app.verifyCredential(req.Credential) {
		app.log.Info("Failed to verify IP credential")
		return code.INVALID_CREDENTIAL
	}

	msg := make([]byte, len(req.Address)+len(req.Credential))
	copy(msg, req.Address)
	copy(msg[len(req.Address):], req.Credential)

	recPub, err := ethcrypto.SigToPub(tmconst.HashFunction(msg), req.Sig)
	if err != nil {
		app.log.Info("Error while trying to recover public key associated with the signature")
		return code.INVALID_SIGNATURE
	}

	recAddr := ethcrypto.PubkeyToAddress(*recPub)
	if !bytes.Equal(recAddr[:], req.Address) {
		app.log.Info("Failed to verify signature on request")
		return code.INVALID_SIGNATURE
	}

	return code.OK

}

func (app *NymApplication) checkTransferBetweenAccountsTx(tx []byte) uint32 {
	req := &transaction.AccountTransferRequest{}

	if err := proto.Unmarshal(tx, req); err != nil {
		app.log.Info("Failed to unmarshal request")
		return code.INVALID_TX_PARAMS
	}

	if app.checkNonce(req.Nonce, req.SourceAddress) {
		return code.REPLAY_ATTACK_ATTEMPT
	}

	if retCode, _ := app.validateTransfer(req.SourceAddress, req.TargetAddress, req.Amount); retCode != code.OK {
		return retCode
	}

	msg := make([]byte, 2*ethcommon.AddressLength+tmconst.NonceLength+8)
	copy(msg, req.SourceAddress)
	copy(msg[ethcommon.AddressLength:], req.TargetAddress)
	binary.BigEndian.PutUint64(msg[2*ethcommon.AddressLength:], req.Amount)
	copy(msg[2*ethcommon.AddressLength+8:], req.Nonce)

	recPub, err := ethcrypto.SigToPub(tmconst.HashFunction(msg), req.Sig)
	if err != nil {
		app.log.Info("Error while trying to recover public key associated with the signature")
		return code.INVALID_SIGNATURE
	}

	recAddr := ethcrypto.PubkeyToAddress(*recPub)
	if !bytes.Equal(recAddr[:], req.SourceAddress) {
		app.log.Info("Failed to verify signature on request")
		return code.INVALID_SIGNATURE
	}

	return code.OK
}

// func (app *NymApplication) checkDepositCoconutCredentialTx(tx []byte) uint32 {
// 	req := &transaction.DepositCoconutCredentialRequest{}

// 	if err := proto.Unmarshal(tx, req); err != nil {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	var merchantAddress account.ECPublicKey = req.MerchantAddress

// 	// start with checking for double spending -
// 	// if credential was already spent, there is no point in any further checks
// 	dbZetaEntry := prefixKey(tmconst.SpentZetaPrefix, req.Theta.Zeta)
// 	_, zetaStatus := app.state.db.Get(dbZetaEntry)
// 	if zetaStatus != nil {
// 		return code.DOUBLE_SPENDING_ATTEMPT
// 	}

// 	cred := &coconut.Signature{}
// 	if err := cred.FromProto(req.Sig); err != nil {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	theta := &coconut.ThetaTumbler{}
// 	if err := theta.FromProto(req.Theta); err != nil {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	pubM := coconut.BigSliceFromByteSlices(req.PubM)
// 	if !coconut.ValidateBigSlice(pubM) {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	// check if the merchant address is correctly formed
// 	if err := merchantAddress.Compress(); err != nil {
// 		return code.INVALID_MERCHANT_ADDRESS
// 	}

// 	if !app.checkIfAccountExists(merchantAddress) {
// 		if !createAccountOnDepositIfDoesntExist {
// 			app.log.Error("Merchant's account doesnt exist")
// 			return code.MERCHANT_DOES_NOT_EXIST
// 		}

// 		// checkTx will not try creating the account for obvious reasons
// 	}

// 	// don't verify the credential itself as it's rather expensive operation; it will only be done during deliverTx

// 	return code.OK
// }

// func (app *NymApplication) checkTxTransferToHolding(tx []byte) uint32 {
// 	// verify sigs and check if all structs can be unmarshalled
// 	req := &transaction.TransferToHoldingRequest{}
// 	if err := proto.Unmarshal(tx, req); err != nil {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	if len(req.PubM) < 1 ||
// 		len(req.PubM[0]) != constants.BIGLen ||
// 		Curve.Comp(Curve.FromBytes(req.PubM[0]), Curve.NewBIGint(int(req.Amount))) != 0 {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	// only recovered to see if an error is thrown
// 	lambda := &coconut.Lambda{}
// 	if err := lambda.FromProto(req.Lambda); err != nil {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	lambdab, err := proto.Marshal(req.Lambda)
// 	if err != nil {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	// only recovered to see if an error is thrown
// 	egPub := &elgamal.PublicKey{}
// 	if rerr := egPub.FromProto(req.EgPub); rerr != nil {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	egPubb, err := proto.Marshal(req.EgPub)
// 	if err != nil {
// 		return code.INVALID_TX_PARAMS
// 	}

// 	var sourcePublicKey account.ECPublicKey = req.SourcePublicKey
// 	recoveredHoldingAddress := req.TargetAddress

// 	// TODO: update once epochs, etc. are introduced
// 	if !bytes.Equal(recoveredHoldingAddress, tmconst.HoldingAccountAddress) {
// 		return code.MALFORMED_ADDRESS
// 	}

// 	if retCode, _ := app.validateTransfer(
// 		sourcePublicKey,
// 		recoveredHoldingAddress,
// 		uint64(req.Amount),
// 	); retCode != code.OK {
// 		return retCode
// 	}

// 	msg := make([]byte,
// 		len(sourcePublicKey)+len(recoveredHoldingAddress)+4+len(egPubb)+len(lambdab)+constants.BIGLen*len(req.PubM),
// 	)
// 	copy(msg, sourcePublicKey)
// 	copy(msg[len(sourcePublicKey):], recoveredHoldingAddress)
// 	binary.BigEndian.PutUint32(msg[len(sourcePublicKey)+len(recoveredHoldingAddress):], uint32(req.Amount))
// 	copy(msg[len(sourcePublicKey)+len(recoveredHoldingAddress)+4:], egPubb)
// 	copy(msg[len(sourcePublicKey)+len(recoveredHoldingAddress)+4+len(egPubb):], lambdab)
// 	for i := range req.PubM {
// 		copy(msg[len(sourcePublicKey)+len(recoveredHoldingAddress)+4+len(egPubb)+len(lambdab)+constants.BIGLen*i:],
// 			req.PubM[i],
// 		)
// 	}

// 	if len(req.Sig) != account.SignatureSize || !sourcePublicKey.VerifyBytes(msg, req.Sig) {
// 		app.log.Info("Failed to verify signature on request")
// 		return code.INVALID_SIGNATURE
// 	}

// 	return code.OK
// }
