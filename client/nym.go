// nym.go - nym client API
// Copyright (C) 2018-2019  Jedrzej Stuczynski.
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

// Package client encapsulates all calls to issuers and providers.
package client

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"math/big"
	"net"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	"github.com/nymtech/nym/common/comm"
	"github.com/nymtech/nym/common/comm/commands"
	"github.com/nymtech/nym/common/comm/packet"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/crypto/elgamal"
	ethclient "github.com/nymtech/nym/ethereum/client"
	"github.com/nymtech/nym/nym/token"
	"github.com/nymtech/nym/tendermint/nymabci/code"
	"github.com/nymtech/nym/tendermint/nymabci/query"
	"github.com/nymtech/nym/tendermint/nymabci/transaction"
)

func (c *Client) WaitForEthereumTxToResolve(ctx context.Context, txHash ethcommon.Hash) (bool, error) {
	// TODO: variable ticket interval?
	ticker := time.NewTicker(1500 * time.Millisecond)

	for {
		select {
		case <-ctx.Done():
			// TODO: this log is not thread safe...
			c.log.Warning("Context timeout - we do no know if the tx suceeded or not")
			return false, errors.New("context timeout")
		case <-ticker.C:
			status := c.ethClient.GetTransactionStatus(ctx, txHash)
			switch status {
			case ethclient.TxStatusUnknown:
				c.log.Warningf("Tx %v is in unknown state...", txHash.Hex())
				return false, errors.New("transaction is in unknown state")
			case ethclient.TxStatusPending:
				c.log.Debugf("Tx %v is still pending", txHash.Hex())
			case ethclient.TxStatusRejected:
				c.log.Errorf("Tx %v was rejected!", txHash.Hex())
				// TODO: what to do now with that information?
				return false, nil
			case ethclient.TxStatusAccepted:
				c.log.Noticef("Tx %v was accepted", txHash.Hex())
				return true, nil
			}
		}
	}
}

func (c *Client) parseCredentialPairResponse(resp *commands.LookUpCredentialResponse,
	elGamalPrivateKey *elgamal.PrivateKey,
) (*coconut.Signature, int64, error) {
	if err := c.checkResponseStatus(resp); err != nil {
		return nil, -1, err
	}
	protoBlindSig := &coconut.ProtoBlindedSignature{}
	if err := proto.Unmarshal(resp.CredentialPair.Credential.Sig, protoBlindSig); err != nil {
		return nil, -1, c.logAndReturnError("parseCredentialPairResponse: failed to unmarshal received proto-credential")
	}
	blindSig := &coconut.BlindedSignature{}
	if err := blindSig.FromProto(protoBlindSig); err != nil {
		return nil, -1, c.logAndReturnError("parseCredentialPairResponse: failed to unmarshal received credential")
	}
	sig := c.cryptoworker.CoconutWorker().UnblindWrapper(blindSig, elGamalPrivateKey)
	return sig, resp.CredentialPair.Credential.IssuerID, nil
}

func (c *Client) parseLookUpCredentialServerResponses(responses []*comm.ServerResponse,
	elGamalPrivateKey *elgamal.PrivateKey,
) ([]*coconut.Signature, *coconut.PolynomialPoints) {
	if responses == nil || (len(responses) < c.cfg.Client.Threshold && c.cfg.Client.Threshold > 0) {
		return nil, nil
	}

	sigs := make([]*coconut.Signature, 0, len(responses))
	xs := make([]*Curve.BIG, 0, len(responses))
	for i := range responses {
		if responses[i] != nil && responses[i].ServerMetadata != nil {
			resp := &commands.LookUpCredentialResponse{}
			if err := proto.Unmarshal(responses[i].MarshaledData, resp); err != nil {
				c.log.Errorf("Failed to unmarshal response from: %v", responses[i].ServerMetadata.Address)
				continue
			}

			sig, id, err := c.parseCredentialPairResponse(resp, elGamalPrivateKey)
			if err != nil {
				continue
			}
			xs = append(xs, Curve.NewBIGint(int(id)))
			sigs = append(sigs, sig)
		}
	}
	return sigs, coconut.NewPP(xs)
}

// GetCurrentERC20Balance gets the current balance of ERC20 tokens associated with the client's address
func (c *Client) GetCurrentERC20Balance() (uint64, error) {
	ctx := context.TODO()
	address := ethcrypto.PubkeyToAddress(*c.privateKey.Public().(*ecdsa.PublicKey))
	balance, err := c.ethClient.QueryERC20Balance(ctx, address, false)
	if err != nil {
		return 0, c.logAndReturnError("GetCurrentERC20Balance: failed to query balance: %v", err)
	}
	t := new(big.Int)
	fullTokens := t.Div(balance, big.NewInt(1000000000000000000))

	return fullTokens.Uint64(), nil
}

// GetCurrentERC20PendingBalance gets the current pending balance of ERC20 tokens associated with the client's address
func (c *Client) GetCurrentERC20PendingBalance() (uint64, error) {
	ctx := context.TODO()
	address := ethcrypto.PubkeyToAddress(*c.privateKey.Public().(*ecdsa.PublicKey))
	balance, err := c.ethClient.QueryERC20Balance(ctx, address, true)
	if err != nil {
		return 0, c.logAndReturnError("GetCurrentERC20PendingBalance: failed to query balance: %v", err)
	}
	t := new(big.Int)
	fullTokens := t.Div(balance, big.NewInt(1000000000000000000))

	return fullTokens.Uint64(), nil
}

// GetCurrentNymBalance gets the current (might be slightly stale due to request being
// sent as a query and not transaction) balance associated with the client's address.
func (c *Client) GetCurrentNymBalance() (uint64, error) {
	address := ethcrypto.PubkeyToAddress(*c.privateKey.Public().(*ecdsa.PublicKey))
	res, err := c.nymClient.Query(query.QueryCheckBalancePath, address[:])
	if err != nil {
		return 0, c.logAndReturnError("GetCurrentNymBalance: failed to send getBalance Query: %v", err)
	}
	if res.Response.Code != code.OK {
		return 0, c.logAndReturnError("GetCurrentNymBalance: the query failed with code %v (%v)",
			res.Response.Code,
			code.ToString(res.Response.Code),
		)
	}
	balance := binary.BigEndian.Uint64(res.Response.Value)
	c.log.Debugf("Queried balance is : %v", balance)
	return balance, nil
}

func (c *Client) RegisterAccount(credential []byte) error {
	exists, err := c.CheckAccountExistence()
	if err != nil {
		return c.logAndReturnError("RegisterAccount: could not check account status")
	}
	if exists {
		return c.logAndReturnError("RegisterAccount: account already exists")
	}
	tx, err := transaction.CreateNewAccountRequest(c.privateKey, credential)
	if err != nil {
		return c.logAndReturnError("RegisterAccount: could not create register transaction")
	}

	res, err := c.nymClient.Broadcast(tx)
	if err != nil {
		return c.logAndReturnError("RegisterAccount: Failed to send new account request: %v", err)
	}
	if res.CheckTx.Code != code.OK || res.DeliverTx.Code != code.OK {
		// TODO: once we include Logs field, return those
		return c.logAndReturnError("RegisterAccount: Failed to send new account request: checkTx code: %v (%v) deliverTx code: %v (%v)",
			res.CheckTx.Code,
			code.ToString(res.CheckTx.Code),
			res.DeliverTx.Code,
			code.ToString(res.DeliverTx.Code),
		)
	}

	c.log.Notice("Created new account")
	return nil
}

func (c *Client) CheckAccountExistence() (bool, error) {
	address := ethcrypto.PubkeyToAddress(*c.privateKey.Public().(*ecdsa.PublicKey))
	res, err := c.nymClient.Query(query.AccountExistence, address[:])
	if err != nil {
		return false, c.logAndReturnError("CheckAccountExistence: failed to send getBalance Query: %v", err)
	}
	if res.Response.Code != code.OK {
		return false, c.logAndReturnError("CheckAccountExistence: the query failed with code %v (%v)",
			res.Response.Code,
			code.ToString(res.Response.Code),
		)
	}
	if bytes.Equal(res.Response.Value, query.AccountStatusExists) {
		return true, nil
	}
	return false, nil
}

func (c *Client) parseFaucetTransferResponse(packetResponse *packet.Packet) (ethcommon.Hash, ethcommon.Hash, error) {
	faucetTransferResponse := &commands.FaucetTransferResponse{}
	if err := proto.Unmarshal(packetResponse.Payload(), faucetTransferResponse); err != nil {
		return ethcommon.Hash{}, ethcommon.Hash{}, c.logAndReturnError("parseFaucetTransferResponse: Failed to recover the result: %v", err)
	} else if faucetTransferResponse.GetStatus().Code != int32(commands.StatusCode_OK) {
		return ethcommon.Hash{}, ethcommon.Hash{}, c.logAndReturnError(
			"parseFaucetTransferResponse: Received invalid response with status: %v. Error: %v",
			faucetTransferResponse.GetStatus().Code,
			faucetTransferResponse.GetStatus().Message,
		)
	}

	erc20Hash := ethcommon.BytesToHash(faucetTransferResponse.Erc20TxHash)
	etherHash := ethcommon.BytesToHash(faucetTransferResponse.EtherTxHash)

	c.log.Warningf("hash1: %v, hash2: %v", erc20Hash.Hex(), etherHash.Hex())

	return erc20Hash, etherHash, nil
}

func (c *Client) MakeFaucetRequest(ctx context.Context, amount int64) (ethcommon.Hash, ethcommon.Hash, error) {
	cmd, err := commands.NewFaucetTransferRequest(c.privateKey, uint64(amount))
	if err != nil {
		return ethcommon.Hash{}, ethcommon.Hash{}, c.logAndReturnError("MakeFaucetRequest: Failed to create faucet transfer request: %v", err)
	}

	packetBytes, err := commands.CommandToMarshalledPacket(cmd)
	if err != nil {
		return ethcommon.Hash{}, ethcommon.Hash{}, c.logAndReturnError("MakeFaucetRequest: Could not create data packet for faucet transfer command: %v", err)
	}

	c.log.Debugf("Dialing %v", c.cfg.Nym.FaucetAddress)
	conn, err := net.Dial("tcp", c.cfg.Nym.FaucetAddress)
	if err != nil {
		return ethcommon.Hash{}, ethcommon.Hash{}, c.logAndReturnError("MakeFaucetRequest: Could not dial %v (%v)", c.cfg.Nym.FaucetAddress, err)
	}

	// currently will never be thrown since there is no writedeadline
	if _, werr := conn.Write(packetBytes); werr != nil {
		return ethcommon.Hash{}, ethcommon.Hash{},
			c.logAndReturnError("MakeFaucetRequest: Failed to write to connection: %v", werr)
	}

	sderr := conn.SetReadDeadline(time.Now().Add(time.Duration(c.cfg.Debug.FaucetRequestTimeout) * time.Millisecond))
	if sderr != nil {
		return ethcommon.Hash{}, ethcommon.Hash{},
			c.logAndReturnError("MakeFaucetRequest: Failed to set read deadline for connection: %v",
				sderr)
	}

	resp, err := comm.ReadPacketFromConn(conn)
	if err != nil {
		return ethcommon.Hash{}, ethcommon.Hash{},
			c.logAndReturnError("MakeFaucetRequest: Received invalid response from %v: %v", c.cfg.Nym.FaucetAddress, err)
	}

	return c.parseFaucetTransferResponse(resp)
}

func (c *Client) SendToPipeAccount(ctx context.Context, amount int64) error {
	if _, err := c.ethClient.TransferERC20Tokens(ctx, amount, c.cfg.Nym.PipeAccount); err != nil {
		return err
	}
	return nil
}

func (c *Client) WaitForERC20BalanceChangeWrapper(ctx context.Context, expectedBalance uint64) error {
	return c.waitForERC20BalanceChange(ctx, expectedBalance)
}

// TODO: perhaps wait for N blocks to be more certain of it?
// TODO: only works under assumption given client is ONLY communicating with us
func (c *Client) waitForERC20BalanceChange(ctx context.Context, expectedBalance uint64) error {
	c.log.Info("Waiting for our transaction to reach the Ethereum chain")
	// TODO: make ticker interval configurable in config.toml file?
	retryTicker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-retryTicker.C:
			currentBalance, err := c.GetCurrentERC20Balance()
			if err != nil {
				// TODO: should we cancel instead?
				c.log.Warningf("Error while querying for balance: %v", err)
			}
			pendingBalance, err := c.GetCurrentERC20PendingBalance()
			if err != nil {
				c.log.Warningf("Error while querying for pending balance: %v", err)
			}
			c.log.Debugf("Waiting for balance to change to: %v\nCurrent: %v, Pending: %v",
				expectedBalance,
				currentBalance,
				pendingBalance,
			)
			if currentBalance == expectedBalance {
				c.log.Debug("Target balance was reached")
				return nil
			}
		case <-ctx.Done():
			c.log.Warning("Context timeout was reached")
			return errors.New("operation was cancelled")
		}
	}
}

// // actually we don't need this method at all - when we broadcast the data we wait for it to be included
// TODO: only works under assumption given client is ONLY communicating with us
func (c *Client) WaitForBalanceChange(ctx context.Context, expectedBalance uint64) error {
	c.log.Info("Waiting for our transaction to reach Tendermint chain")
	retryTicker := time.NewTicker(2 * time.Second)

	for {
		select {
		case <-retryTicker.C:
			currentBalance, err := c.GetCurrentNymBalance()
			if err != nil {
				// TODO: should we cancel instead?
				c.log.Warningf("Error while querying for balance: %v", err)
			}
			if currentBalance == expectedBalance {
				return nil
			}
		case <-ctx.Done():
			return errors.New("operation was cancelled")
		}
	}
}

// RedeemTokens allows to move specified number of Nym tokens back into ERC20
func (c *Client) RedeemTokens(ctx context.Context, amount uint64) error {
	tx, err := transaction.CreateNewTokenRedemptionRequest(c.privateKey, amount)
	if err != nil {
		return c.logAndReturnError("RedeemTokens: Failed to create redemption request: %v", err)
	}
	res, err := c.nymClient.Broadcast(tx)
	if err != nil {
		return c.logAndReturnError("RedeemTokens: Failed to send redemption request: %v", err)
	}
	if res.CheckTx.Code != code.OK || res.DeliverTx.Code != code.OK {
		// TODO: once we include Logs field, return those
		return c.logAndReturnError("RedeemTokens: Failed to send redemption request: checkTx code: %v (%v) deliverTx code: %v (%v)",
			res.CheckTx.Code,
			code.ToString(res.CheckTx.Code),
			res.DeliverTx.Code,
			code.ToString(res.DeliverTx.Code),
		)
	}
	return nil
}

// LookUpIssuedCredential allows to recover a previously issued credential given knowledge of height on which we
// sent the materials and the elGamal keypair associated with the request.
func (c *Client) LookUpIssuedCredential(height int64,
	elGamalPrivateKey *elgamal.PrivateKey,
	elGamalPublicKey *elgamal.PublicKey,
) (*coconut.Signature, error) {
	cmd, err := commands.NewLookUpCredentialRequest(height, elGamalPublicKey)
	if err != nil {
		return nil, c.logAndReturnError("LookUpIssuedCredential: Failed to create LookUpCredential request: %v", err)
	}
	packetBytes, err := commands.CommandToMarshalledPacket(cmd)
	if err != nil {
		return nil,
			c.logAndReturnError("LookUpIssuedCredential: Could not create data packet for look up credential command: %v",
				err,
			)
	}

	retryTicker := time.NewTicker(time.Duration(c.cfg.Debug.LookUpBackoff) * time.Millisecond)
	defer retryTicker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.cfg.Debug.RequestTimeout)*time.Millisecond)
	defer cancel()

	var responses []*comm.ServerResponse
	retryCount := 0

	c.log.Infof("Waiting for %vms before trying to contact the issuers", c.cfg.Debug.LookUpBackoff)

	// we actually don't want to enter tickerCase immediately to give issuers some time to actually handle the request
outerFor:
	for {
		if retryCount == c.cfg.Debug.NumberOfLookUpRetries {
			break
		}
		select {
		case <-ctx.Done():
			c.log.Warning("Exceeded context timeout for the request")
			break outerFor
		case <-retryTicker.C:
			retryCount++

			c.log.Notice("Going to send look up credential request to %v IAs", len(c.cfg.Client.IAAddresses))
			responses = comm.GetServerResponses(
				ctx,
				&comm.RequestParams{
					MarshaledPacket:   packetBytes,
					MaxRequests:       c.cfg.Client.MaxRequests,
					ConnectionTimeout: time.Duration(c.cfg.Debug.ConnectTimeout) * time.Millisecond,
					ServerAddresses:   c.cfg.Client.IAAddresses,
				},
				c.log,
			)

			// try to parse signature with data we received
			sig, err := c.handleReceivedSignatures(c.parseLookUpCredentialServerResponses(responses, elGamalPrivateKey))
			if err != nil {
				c.log.Warningf("LookUpIssuedCredential: Failed to parse received credentials: %v", err)
				continue outerFor
			}
			return sig, nil
		}
	}

	// TODO: somehow return gamma and height in response rather than in error message
	return nil, c.logAndReturnError(`LookUpIssuedCredential: Could not communicate with enough IAs to obtain credentials.
				Token was spent in block: %v and gamma used was: %v`, cmd.Height, cmd.Gamma)
}

// GetCredential is a multistep procedure. First it sends 'GetCredential' request to Tendermint blockchain.
// This is followed by query to all IA servers specified in the config to obtain partial credentials based on
// materials sent to the chain.
func (c *Client) GetCredential(token *token.Token) (*coconut.Signature, error) {
	if c.cfg.Client.UseGRPC {
		return nil, c.logAndReturnError(gRPCClientErr)
	}

	elGamalPrivateKey, elGamalPublicKey := c.cryptoworker.CoconutWorker().ElGamalKeygenWrapper()

	// first check if we have loaded the account information
	if c.privateKey == nil {
		return nil, c.logAndReturnError("GetCredential: Tried to obtain credential on undefined account")
	}

	// query our balance to make sure we have enough funds to get credential on specified value
	currentBalance, err := c.GetCurrentNymBalance()
	if err != nil {
		return nil, c.logAndReturnError("GetCredential: could not query for current balance: %v", err)
	}

	// FIXME: this seems like a dodgy comparison due to type conversion, we need to find a way to change it
	// However, even though token value is an int64, it must always be positive
	if currentBalance < uint64(token.Value()) {
		// TODO: flag to transfer remaining funds to pipe account if available on ethereum?
		return nil, c.logAndReturnError("GetCredential: current balance is lower than the value of desired credential")
	}

	// we send request to the chain
	height, err := c.sendCredentialRequest(token, elGamalPublicKey)
	if err != nil {
		return nil, c.logAndReturnError("GetCredential: could not send credential request: %v", err)
	}

	if height <= 1 {
		return nil, c.logAndReturnError("GetCredential: tx was included at invalid height: %v", height)
	}
	c.log.Debugf("Our tx was included in block: %v", height)

	// TODO: if there's a failure anywhere beyond this point, we must be able to return height and elgamal keypair
	// so that client could theoretically retry at later time
	return c.LookUpIssuedCredential(height, elGamalPrivateKey, elGamalPublicKey)
}

func (c *Client) sendCredentialRequest(token *token.Token, egPub *elgamal.PublicKey) (int64, error) {
	lambda, err := c.cryptoworker.CoconutWorker().PrepareBlindSignTokenWrapper(egPub, token)
	if err != nil {
		return -1, c.logAndReturnError("sendCredentialRequest: Could not create lambda: %v", err)
	}

	pubM, _ := token.GetPublicAndPrivateSlices()
	bsm := coconut.NewBlindSignMaterials(lambda, egPub, pubM)

	req, err := transaction.CreateNewCredentialRequest(c.privateKey, c.cfg.Nym.PipeAccount, bsm, token.Value())
	if err != nil {
		return -1, c.logAndReturnError("sendCredentialRequest: Failed to create request: %v", err)
	}

	res, err := c.nymClient.Broadcast(req)
	if err != nil {
		return -1, c.logAndReturnError("sendCredentialRequest: Failed to send request to the blockchain: %v", err)
	}
	if res.DeliverTx.Code != code.OK || res.CheckTx.Code != code.OK {
		return -1,
			c.logAndReturnError(`sendCredentialRequest: Our request failed to be processed by the blockchain:
CheckTx: %v - %v
DeliverTx: %v - %v`,
				res.CheckTx.Code,
				code.ToString(res.CheckTx.Code),
				res.DeliverTx.Code,
				code.ToString(res.DeliverTx.Code),
			)
	}

	return res.Height, nil
}

func (c *Client) parseSpendCredentialResponse(packetResponse *packet.Packet) (bool, error) {
	spendCredentialResponse := &commands.SpendCredentialResponse{}
	if err := proto.Unmarshal(packetResponse.Payload(), spendCredentialResponse); err != nil {
		return false, c.logAndReturnError("parseSpendCredentialResponse: Failed to recover spend credential result: %v", err)
	} else if spendCredentialResponse.GetStatus().Code != int32(commands.StatusCode_OK) {
		return false, c.logAndReturnError(
			"parseSpendCredentialResponse: Received invalid response with status: %v. Error: %v",
			spendCredentialResponse.GetStatus().Code,
			spendCredentialResponse.GetStatus().Message,
		)
	}
	return spendCredentialResponse.WasSuccessful, nil
}

func (c *Client) prepareSpendCredentialRequest(
	token *token.Token,
	sig *coconut.Signature,
	vk *coconut.VerificationKey,
	providerAddress ethcommon.Address,
) (*commands.SpendCredentialRequest, error) {
	var err error
	if vk == nil {
		if c.cfg.Client.UseGRPC {
			vk, err = c.GetAggregateVerificationKeyGrpc()
		} else {
			vk, err = c.GetAggregateVerificationKey()
		}
		if err != nil {
			return nil,
				c.logAndReturnError("prepareSpendCredentialRequest: "+
					"Could not obtain aggregate verification key required to create proofs for verification: %v",
					err,
				)
		}
	}

	pubM, privM := token.GetPublicAndPrivateSlices()
	theta, err := c.cryptoworker.CoconutWorker().ShowBlindSignatureTumblerWrapper(vk, sig, privM, providerAddress[:])
	if err != nil {
		return nil,
			c.logAndReturnError("prepareSpendCredentialRequest: Failed when creating proofs for verification: %v", err)
	}

	spendCredentialRequest, err := commands.NewSpendCredentialRequest(sig, pubM, theta, token.Value(), providerAddress)
	if err != nil {
		return nil,
			c.logAndReturnError("prepareSpendCredentialRequest: Failed to create SpendCredential request: %v", err)
	}
	return spendCredentialRequest, nil
}

// SpendCredential sends a TCP request to spend an issued credential at a particular provider.
//nolint: dupl
func (c *Client) SpendCredential(
	token *token.Token, // token on which the credential is issued; encapsulates required attributes
	credential *coconut.Signature, // the credential to be spent
	address string, // physical address of the merchant to which we send the request
	providerAccountAddress ethcommon.Address, // blockchain address of the merchant to which the proof will be bound
	vk *coconut.VerificationKey, // aggregate verification key of the issuers in the system
) (bool, error) {
	if c.cfg.Client.UseGRPC {
		return false, c.logAndReturnError(gRPCClientErr)
	}

	spendCredentialRequest, err := c.prepareSpendCredentialRequest(token, credential, vk, providerAccountAddress)
	if err != nil {
		return false, c.logAndReturnError("SpendCredential: Failed to prepare spendCredentialRequest: %v", err)
	}

	packetBytes, err := commands.CommandToMarshalledPacket(spendCredentialRequest)
	if err != nil {
		return false, c.logAndReturnError("Could not create data packet for spend credential command: %v", err)
	}

	c.log.Debugf("Dialing %v", address)
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return false, c.logAndReturnError("SpendCredential: Could not dial %v (%v)", address, err)
	}

	// currently will never be thrown since there is no writedeadline
	if _, werr := conn.Write(packetBytes); werr != nil {
		return false,
			c.logAndReturnError("SpendCredential: Failed to write to connection: %v", werr)
	}

	// TODO: is it the actual connect timeout or rather timeout to receive any response?
	sderr := conn.SetReadDeadline(time.Now().Add(time.Duration(c.cfg.Debug.ConnectTimeout+c.cfg.Debug.RequestTimeout) * time.Millisecond))
	if sderr != nil {
		return false,
			c.logAndReturnError("SpendCredential: Failed to set read deadline for connection: %v",
				sderr)
	}

	resp, err := comm.ReadPacketFromConn(conn)
	if err != nil {
		return false,
			c.logAndReturnError("SpendCredential: Received invalid response from %v: %v", address, err)
	}

	return c.parseSpendCredentialResponse(resp)
}

// SpendCredentialGrpc sends a gRPC request to spend an issued credential at a particular provider.
func (c *Client) SpendCredentialGrpc(token *token.Token, credential *coconut.Signature, providerAddress []byte) error {
	if !c.cfg.Client.UseGRPC {
		return c.logAndReturnError(nonGRPCClientErr)
	}

	return nil // not implemeneted
}
