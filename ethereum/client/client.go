// client.go - Ethereum client
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

// package client provides API for communication with an Ethereum blockchain.
package client

// TODO: transfer to pipe, redeem credential calls handling

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	token "github.com/nymtech/nym-validator/ethereum/token"
	"github.com/nymtech/nym-validator/logger"
	"gopkg.in/op/go-logging.v1"
)

// Client defines necessary attributes for establishing communication with an Ethereum blockchain
// and for performing functions required by the Nym system.
type Client struct {
	nodeAddress      string
	privateKey       *ecdsa.PrivateKey
	nymTokenInstance *token.Token
	ethClient        *ethclient.Client
	log              *logging.Logger

	pipeAccount common.Address // TODO: needed?
}

const (
	// Nym specific
	defaultDecimals = 18 // TODO: move this one to erc20.constants?
)

type TxStatus byte

const (
	TxStatusUnknown  TxStatus = 0
	TxStatusAccepted TxStatus = 1
	TxStatusRejected TxStatus = 2
	TxStatusPending  TxStatus = 3
)

// TODO: move to separate token-related package
func getTokenDenomination(decimals int64) *big.Int {
	// return big.NewInt(int64(10) * *18)
	t := new(big.Int)
	// look at: https://github.com/securego/gosec/issues/283
	//nolint: gosec
	t.Exp(big.NewInt(10), big.NewInt(decimals), nil)
	return t
}

// TODO: Since it's literally copied from the main client's code, should we just move it to common or something?
func (c *Client) logAndReturnError(fmtString string, a ...interface{}) error {
	errstr := fmtString
	if a != nil {
		errstr = fmt.Sprintf(fmtString, a...)
	}
	c.log.Error(errstr)
	return errors.New(errstr)
}

// used to get status of transaction, pending, accepted, rejected, etc
func (c *Client) GetTransactionStatus(ctx context.Context, txHash common.Hash) TxStatus {
	// TODO:
	_, isPending, err := c.ethClient.TransactionByHash(ctx, txHash)
	if err != nil {
		return TxStatusUnknown
	}
	if isPending {
		return TxStatusPending
	}
	receipt, err := c.ethClient.TransactionReceipt(ctx, txHash)
	// if tx is not pending we should be able to get a receipt
	if receipt == nil || err != nil {
		return TxStatusUnknown
	}
	if receipt.Status == types.ReceiptStatusSuccessful {
		return TxStatusAccepted
	}
	return TxStatusRejected
}

// pending is used to decide whether to query pending balance
func (c *Client) QueryERC20Balance(ctx context.Context, address common.Address, pending bool) (*big.Int, error) {
	balance, err := c.nymTokenInstance.BalanceOf(&bind.CallOpts{
		Pending: pending,
		Context: ctx,
	}, address)

	if err != nil {
		return nil, c.logAndReturnError("QueryERC20Balance: Failed to query balance of %v: %v", address.Hex(), err)
	}

	return balance, nil
}

func (c *Client) EtherToWei(ether *big.Float) *big.Int {
	t := new(big.Float).SetInt64(1000000000000000000)
	res, _ := t.Mul(t, ether).Int(nil)
	return res
}

// for nil blockNumber, latest known is used
func (c *Client) QueryEtherBalance(ctx context.Context, address common.Address, blockNumber *big.Int) (*big.Int, error) {
	return c.ethClient.BalanceAt(ctx, address, blockNumber)
}

func (c *Client) TransferEther(ctx context.Context, toAddress common.Address, amount float64) (common.Hash, error) {
	fromAddress := crypto.PubkeyToAddress(*c.privateKey.Public().(*ecdsa.PublicKey))
	ethereumClient := c.ethClient

	nonce, err := ethereumClient.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return common.Hash{}, c.logAndReturnError("TransferEther: can't determine pending nonce: %v", err)
	}

	value := c.EtherToWei(new(big.Float).SetFloat64(amount))

	// TODO: set it dynamically
	gasLimit := uint64(100000) // in units
	gasPrice, err := ethereumClient.SuggestGasPrice(context.Background())
	if err != nil {
		return common.Hash{}, c.logAndReturnError("TransferEther: can't eastimate gas price: %v", err)
	}

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, []byte{})

	chainID, err := ethereumClient.NetworkID(context.Background())
	if err != nil {
		return common.Hash{}, c.logAndReturnError("TransferEther: can't obtain network ID %v", err)
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.privateKey)
	if err != nil {
		return common.Hash{}, c.logAndReturnError("TransferEther: can't sign the tx: %v", err)
	}

	if err := ethereumClient.SendTransaction(context.Background(), signedTx); err != nil {
		return common.Hash{}, c.logAndReturnError("TransferEther: can't send the tx: %v", err)
	}

	return signedTx.Hash(), nil
}

// TransferERC20Tokens sends specified amount of ERC20 tokens to given account.
func (c *Client) TransferERC20Tokens(ctx context.Context,
	amount int64,
	targetAddress common.Address,
	tokenDecimals ...int,
) (common.Hash, error) {
	if amount <= 0 {
		return common.Hash{}, c.logAndReturnError("TransferERC20Tokens: trying to transfer negative number of tokens")
	}

	var decimals int64
	if len(tokenDecimals) != 1 {
		decimals = defaultDecimals
		c.log.Infof("Assuming target token is using %v decimals", decimals)
	} else {
		decimals = int64(tokenDecimals[0])
		c.log.Infof("Using %v decimals for the token", decimals)
	}

	tokenAmount := new(big.Int)
	tokenAmount.Mul(getTokenDenomination(decimals), big.NewInt(amount))

	auth := bind.NewKeyedTransactor(c.privateKey)
	auth.Context = ctx

	tx, err := c.nymTokenInstance.Transfer(auth, targetAddress, tokenAmount)
	if err != nil {
		return common.Hash{}, c.logAndReturnError("TransferERC20Tokens: Failed to send transaction: %v", err)
	}
	c.log.Noticef("Sent Transaction with hash: %v", tx.Hash().Hex())

	return tx.Hash(), nil
}

func (c *Client) connect(ctx context.Context, ethHost string) error {
	client, err := ethclient.Dial(ethHost)
	if err != nil {
		errMsg := fmt.Sprintf("Error connecting to Infura: %s", err)
		c.log.Error(errMsg)
		return errors.New(errMsg)
	}

	c.log.Debugf("Connected to %v", ethHost)
	c.ethClient = client
	return nil
}

// Config defines configuration for Ethereum Client.
// TODO: if expands too much, move it to a toml file? Or just include it in new section of existing Client toml.
type Config struct {
	privateKey       *ecdsa.PrivateKey
	nodeAddress      string
	erc20NymContract common.Address
	pipeAccount      common.Address

	logger *logger.Logger
}

// NewConfig creates new instance of Config struct.
func NewConfig(pk *ecdsa.PrivateKey, node string, erc20, pipeAccount common.Address, logger *logger.Logger) Config {
	cfg := Config{
		privateKey:       pk,
		nodeAddress:      node,
		erc20NymContract: erc20,
		pipeAccount:      pipeAccount,
		logger:           logger,
	}
	return cfg
}

func New(cfg Config) (*Client, error) {
	c := &Client{
		privateKey:  cfg.privateKey,
		pipeAccount: cfg.pipeAccount,
		nodeAddress: cfg.nodeAddress,
		log:         cfg.logger.GetLogger("Ethereum-Client"),
	}

	// TODO: reconnection, etc as with Tendermint client? Or just have a single node to which we connect and if it
	// fails, it fails (+ actually same consideration for the Tendermint client)
	if err := c.connect(context.TODO(), c.nodeAddress); err != nil {
		return nil, err
	}

	instance, err := token.NewToken(cfg.erc20NymContract, c.ethClient)
	if err != nil {
		return nil, err
	}
	c.nymTokenInstance = instance

	return c, nil
}
