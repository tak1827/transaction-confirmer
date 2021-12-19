package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/tak1827/transaction-confirmer/confirm"
)

const (
	DefaultGasLimit = 21000
	DefaultGasPrice = 0 // 1 gwai
)

var _ confirm.Client = (*Client)(nil)

type Client struct {
	Endpoint string

	ethclient *ethclient.Client
	rpcclient *rpc.Client

	GasPrice *big.Int
}

func NewClient(ctx context.Context, endpoint string) (c Client, err error) {
	c.rpcclient, err = rpc.DialContext(ctx, endpoint)
	if err != nil {
		err = fmt.Errorf("failed to conecting endpoint(%s) err: %w", endpoint, err)
		return
	}

	c.ethclient = ethclient.NewClient(c.rpcclient)

	return
}

func (c *Client) Nonce(ctx context.Context, privKey string) (nonce uint64, err error) {
	priv, err := crypto.HexToECDSA(PrivKey)
	if err != nil {
		err = errors.Wrap(err, "failed to get nonce")
		return
	}

	account := crypto.PubkeyToAddress(priv.PublicKey)
	nonce, err = c.ethclient.NonceAt(ctx, account, nil)
	return
}

func (c *Client) BuildTx(priv *ecdsa.PrivateKey, nonce uint64, to common.Address, amount *big.Int) (*types.Transaction, error) {
	var (
		tx     = types.NewTransaction(nonce, to, amount, uint64(DefaultGasLimit), big.NewInt(int64(DefaultGasPrice)), nil)
		signer = types.HomesteadSigner{}
	)

	sig, err := crypto.Sign(signer.Hash(tx).Bytes(), priv)
	if err != nil {
		return nil, errors.Wrap(err, "err Sign")
	}

	return tx.WithSignature(signer, sig)
}

func (c *Client) SendTx(ctx context.Context, tx interface{}) (string, error) {
	signedTx := tx.(*types.Transaction)

	if err := c.ethclient.SendTransaction(ctx, signedTx); err != nil {
		return "", errors.Wrap(err, "err SendTransaction")
	}

	return signedTx.Hash().Hex(), nil
}

func (c *Client) Receipt(ctx context.Context, hash string) (*types.Receipt, error) {
	return c.ethclient.TransactionReceipt(ctx, common.HexToHash(hash))
}

func (c *Client) LatestBlockNumber(ctx context.Context) (uint64, error) {
	header, err := c.ethclient.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	return header.Number.Uint64(), nil
}

func (c *Client) ConfirmTx(ctx context.Context, hash string, confirmationBlocks uint64) error {
	recept, err := c.Receipt(ctx, hash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return confirm.ErrTxNotFound
		}
		return errors.Wrap(err, "err TransactionReceipt")
	}

	if recept.Status != 1 {
		return confirm.ErrTxFailed
	}

	block, err := c.LatestBlockNumber(ctx)
	if err != nil {
		return errors.Wrap(err, "err LatestBlockNumber")
	}

	if recept.BlockNumber.Uint64()+confirmationBlocks > block {
		return confirm.ErrTxConfirmPending
	}

	return nil
}

func GenerateAddr() (addr common.Address, err error) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		return
	}
	addr = crypto.PubkeyToAddress(priv.PublicKey)
	return
}

// ToWei decimals to wei
func ToWei(iamount interface{}, decimals int) *big.Int {
	amount := decimal.NewFromFloat(0)
	switch v := iamount.(type) {
	case string:
		amount, _ = decimal.NewFromString(v)
	case float64:
		amount = decimal.NewFromFloat(v)
	case int64:
		amount = decimal.NewFromFloat(float64(v))
	case decimal.Decimal:
		amount = v
	case *decimal.Decimal:
		amount = *v
	}

	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromFloat(float64(decimals)))
	result := amount.Mul(mul)

	wei := new(big.Int)
	wei.SetString(result.String(), 10)

	return wei
}
